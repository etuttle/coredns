package nshorizon

import (
	"context"
	"net"
	"testing"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/dnstest"
	"github.com/coredns/coredns/plugin/test"

	"github.com/miekg/dns"
)

func newTestNsHorizon(namespace string) *NsHorizon {
	nh := new(NsHorizon)
	nh.Zones = []string{"internal.com."}
	nh.Next = nextHandler(map[string]int{
		"service.internal.com.": dns.RcodeNameError,
		"service.ns.svc.cluster.local.": dns.RcodeSuccess,
		"no-such-service.internal.com.": dns.RcodeNameError,
		"no-such-service.ns.svc.cluster.local.": dns.RcodeNameError,
		"upstream-service.internal.com.": dns.RcodeSuccess,
		"www.external.com.": dns.RcodeSuccess,
	}, map[string]int{
		"empty-success.internal.com.": dns.RcodeSuccess,
	})
	nh.clusterZone = "cluster.local."
	nh.sourceAddrNamespace = func(a net.Addr) string {
		return namespace
	}
	return nh
}

type nsHorizonTestCase struct {
	test.Case
	sourcens string
}

var nsHorionTestCases = []nsHorizonTestCase{
	{
		// in horizon zone, record does not exist upstream, namespaced service exists
		sourcens: "ns",
		Case: test.Case{
			Qname: "service.internal.com.", Qtype: dns.TypeA,
			Rcode: dns.RcodeSuccess,
			Answer: []dns.RR{
				test.CNAME("service.internal.com. 0 IN CNAME service.ns.svc.cluster.local."),
				test.A("service.ns.svc.cluster.local." + zeroTTLA),
			},
		},
	},
	{
		// in horizon zone, record does not exist upstream, namespace record does not exist
		sourcens: "ns",
		Case: test.Case{
			Qname: "no-such-service.internal.com.", Qtype: dns.TypeA,
			Rcode: dns.RcodeNameError,
			Ns: []dns.RR{
				upstreamSoa,
			},
		},
	},
	{
		// in horizon zone, record does not exist upstream, source namespace unknown
		sourcens: "",
		Case: test.Case{
			Qname: "service.internal.com.", Qtype: dns.TypeA,
			Rcode: dns.RcodeNameError,
			Ns: []dns.RR{
				upstreamSoa,
			},
		},
	},
	{
		// in horizon zone, record does not exist upstream, namespace record does not exist
		// server returns NoError / empty Answer
		sourcens: "ns",
		Case: test.Case{
			Qname: "empty-success.internal.com.", Qtype: dns.TypeA,
			Rcode: dns.RcodeSuccess,
		},
	},
	{
		// in horizon zone, record does not exist upstream, source namespace unknown
		// server returns NoError / empty Answer
		sourcens: "",
		Case: test.Case{
			Qname: "empty-success.internal.com.", Qtype: dns.TypeA,
			Rcode: dns.RcodeSuccess,
		},
	},
	{
		// in horizon zone, record exists upstream
		sourcens: "ns",
		Case: test.Case{
			Qname: "upstream-service.internal.com.", Qtype: dns.TypeA,
			Rcode: dns.RcodeSuccess,
			Answer: []dns.RR{
				test.A("upstream-service.internal.com." + defaultA),
			},
		},
	},
	{
		// not in horizon zone
		sourcens: "ns",
		Case: test.Case{
			Qname: "www.external.com.", Qtype: dns.TypeA,
			Rcode: dns.RcodeSuccess,
			Answer: []dns.RR{
				test.A("www.external.com." + defaultA),
			},
		},
	},
}

func TestNSHorizon(t *testing.T) {
	ctx := context.TODO()

	for i, tc := range nsHorionTestCases {
		nh := newTestNsHorizon(tc.sourcens)
		r := tc.Msg()

		w := dnstest.NewRecorder(&test.ResponseWriter{})

		rcode, err := nh.ServeDNS(ctx, w, r)
		if err != tc.Error {
			t.Errorf("Test %d expected no error, got %v", i, err)
			return
		}
		if tc.Error != nil {
			continue
		}

		if rcode != tc.Rcode {
			t.Errorf("Returned rcode is %q, expected %q", dns.RcodeToString[rcode], dns.RcodeToString[tc.Rcode])
		}

		resp := w.Msg
		if resp == nil {
			t.Fatalf("Test %d, got nil message and no error for %q", i, r.Question[0].Name)
		}

		if err := test.CNAMEOrder(resp); err != nil {
			t.Error(err)
		}

		if err := test.SortAndCheck(resp, tc.Case); err != nil {
			t.Error(err)
		}
	}
}

var nsHorizonNoAnswerTestCases = []test.Case{
	{
		// search path expansion, no answer
		Qname: "a.example.org.", Qtype: dns.TypeA,
		Answer: []dns.RR{
			test.CNAME("a.example.org. 3600 IN CNAME a.com."),
			test.A("a.com." + defaultA),
		},
	},
}

func TestNsHorizonNoAnswer(t *testing.T) {
	ap := newTestNsHorizon("")
	ctx := context.TODO()

	for _, tc := range nsHorizonNoAnswerTestCases {
		m := tc.Msg()

		rec := dnstest.NewRecorder(&test.ResponseWriter{})
		rcode, err := ap.ServeDNS(ctx, rec, m)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
			continue
		}
		if plugin.ClientWrite(rcode) {
			t.Fatalf("Expected no client write, got one for rcode %d", rcode)
		}
	}
}

// nextHandler returns a Handler that returns an answer for the question in the
// request per the domain->answer map. On success an RR will be returned: "qname 3600 IN A 127.0.0.53"
//
// me is a second map with different behavior for RcodeSuccess responses: they include no
// records in the Answer section.  This is another way a server can send an empty response.
func nextHandler(mm map[string]int, me map[string]int) test.Handler {

	internalZone := plugin.Name("internal.com.")
	clusterZone := plugin.Name("cluster.local.")

	return test.HandlerFunc(func(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
		qname := r.Question[0].Name
		s_records := true
		var rcode int
		rcode, ok := mm[qname]
		if !ok {
			rcode2, ok2 := me[qname]
			if !ok2 {
				return dns.RcodeServerFailure, nil
			}
			s_records = false
			rcode = rcode2
		}

		m := new(dns.Msg)
		m.SetReply(r)

		switch rcode {
		case dns.RcodeNameError:
			m.Rcode = rcode
			var soa dns.RR
			switch {
			case internalZone.Matches(qname):
				soa = upstreamSoa
			case clusterZone.Matches(qname):
				soa = clusterSoa
			default:
				panic("NameError response needed for unknown zone")
			}
			m.Ns = []dns.RR{soa}
			w.WriteMsg(m)
			return m.Rcode, nil

		case dns.RcodeSuccess:
			m.Rcode = rcode
			if s_records {
				a, _ := dns.NewRR(qname + defaultA)
				m.Answer = []dns.RR{a}
			}
			w.WriteMsg(m)
			return m.Rcode, nil
		default:
			panic("nextHandler: unhandled rcode")
		}
	})
}

const defaultA = " 3600 IN A 127.0.0.53"
const zeroTTLA = " 0 IN A 127.0.0.53"

var upstreamSoa = func() dns.RR {
	s, _ := dns.NewRR("internal.com.		1800	IN	SOA	internal.com. internal.com. 1502165581 14400 3600 604800 14400")
	return s
}()

var clusterSoa = func() dns.RR {
	s, _ := dns.NewRR("cluster.local.		30	IN	SOA	ns.dns.cluster.local. hostmaster.cluster.local. 1573956451 7200 1800 86400 30")
	return s
}()
