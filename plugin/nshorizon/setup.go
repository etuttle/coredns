package nshorizon

import (
	"fmt"

	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"

	"github.com/caddyserver/caddy"
)

const NSHORIZON = "nshorizon"

func init() {
	caddy.RegisterPlugin(NSHORIZON, caddy.Plugin{
		ServerType: "dns",
		Action:     setup,
	})
}

func setup(c *caddy.Controller) error {
	nh, b, err := nsHorizonParse(c)
	if err != nil {
		return plugin.Error(NSHORIZON, err)
	}

	c.OnStartup(func() error {
		m := dnsserver.GetConfig(c).Handler(b)
		if m == nil {
			return nil
		}
		if x, ok := m.(NsHorizonBackend); ok {
			nh.clusterZone = x.ClusterZone()
			nh.sourceAddrNamespace = x.SourceAddrNamespace
		} else {
			return plugin.Error(NSHORIZON, fmt.Errorf("%s does not implement the AutoPather interface", b))
		}
		return nil
	})

	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		nh.Next = next
		return nh
	})

	return nil
}

func nsHorizonParse(c *caddy.Controller) (*NsHorizon, string, error) {
	nh := &NsHorizon{}
	c.Next()
	args := c.RemainingArgs()

	if len(args) < 2 {
		return nh, "", fmt.Errorf("nshorizon: incomplete config line")
	}

	at_b := args[len(args)-1]

	if at_b[0] != '@' {
		return nh, "", fmt.Errorf("nshorizon: backend field must begin with @")
	}

	b := at_b[1:]

	nh.Zones = args[:len(args)-1]

	for i, str := range nh.Zones {
		nh.Zones[i] = plugin.Host(str).Normalize()
	}

	return nh, b, nil
}
