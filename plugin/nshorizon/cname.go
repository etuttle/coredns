package nshorizon

import (
	"strings"

	"github.com/miekg/dns"
)

// cnamer similar to autopath plugin, but the TTL of the CNAME and the RR are set to 0
func cnamerZeroTTL(m *dns.Msg, original string) {
	for _, a := range m.Answer {
		if strings.EqualFold(original, a.Header().Name) {
			continue
		}
		a.Header().Ttl = 0
		m.Answer = append(m.Answer, nil)
		copy(m.Answer[1:], m.Answer)
		m.Answer[0] = &dns.CNAME{
			Hdr:    dns.RR_Header{Name: original, Rrtype: dns.TypeCNAME, Class: dns.ClassINET, Ttl: 0},
			Target: a.Header().Name,
		}
		break
	}
	m.Question[0].Name = original
}
