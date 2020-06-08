
package nshorizon

import (
	"context"
	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/kubernetes"
	"github.com/coredns/coredns/plugin/pkg/dnsutil"
	"github.com/coredns/coredns/plugin/pkg/nonwriter"
	"github.com/coredns/coredns/request"
	"github.com/miekg/dns"
	"net"
)

// NsHorizonBackend is the interface other plugins must implement to be a backend
// for the NsHorizon plugin.
type NsHorizonBackend interface {
	// If nil is returned, no split horizon records will be created.
	SourceAddrNamespace(net.Addr) string
	ClusterZone() string
}

// NsHorizon performs split-horizon DNS based on cluster namespace of source IP.
type NsHorizon struct {
	Next plugin.Handler
	Zones []string

	sourceAddrNamespace func (net.Addr) string
	clusterZone         string
}

func (nh *NsHorizon) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	state := request.Request{W: w, Req: r}
	zone := plugin.Zones(nh.Zones).Matches(state.Name())

	if zone == "" {
		return plugin.NextOrFailure(nh.Name(), nh.Next, ctx, w, r)
	}

	// capture the query result by wrapping w in a nonwriter and calling the plugin chain
	nw1 := nonwriter.New(w)
	rcode1, err := plugin.NextOrFailure(nh.Name(), nh.Next, ctx, nw1, r)

	if err != nil {
		// since the writer was wrapped, not sure what to do here?
		return rcode1, err
	}

	returnRCode1 := func () (int, error) {
		if plugin.ClientWrite(rcode1) {
			w.WriteMsg(nw1.Msg)
		}
		return rcode1, nil
	}

	if nw1.Msg.Rcode != dns.RcodeNameError &&
		!(nw1.Msg.Rcode == dns.RcodeSuccess && len(nw1.Msg.Answer) == 0) {
		return returnRCode1()
	}

	// here's our chance to find a namespaced service to resolve to
	source_ns := nh.sourceAddrNamespace(w.RemoteAddr())

	if source_ns == "" {
		return returnRCode1()
	}

	nh_r := r.Copy()
	base, err := dnsutil.TrimZone(state.QName(), zone)
	if err != nil {
		return dns.RcodeServerFailure, err
	}

	new_q := dnsutil.Join(base, source_ns, kubernetes.Svc, nh.clusterZone)
	nh_r.Question[0].Name = new_q

	nw2 := nonwriter.New(w)
	rcode2, err := plugin.NextOrFailure(nh.Name(), nh.Next, ctx, nw2, nh_r)
	if err != nil {
		// error from second request, return result of first request
		return returnRCode1()
	}

	if nw2.Msg != nil && nw2.Msg.Rcode == dns.RcodeSuccess && len(nw2.Msg.Answer) > 0 {
		msg := nw2.Msg
		cnamerZeroTTL(msg, state.QName())
		w.WriteMsg(msg)
		return rcode2, nil
	} else {
		return returnRCode1()
	}
}

// Name implements the Handler interface.
func (a *NsHorizon) Name() string { return "nshorizon" }
