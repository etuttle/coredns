package kubernetes

import (
	"net"
)

func (k *Kubernetes) SourceAddrNamespace(source net.Addr) string {
	if !k.opts.initPodCache {
		log.Warning("nshorizon: cannot resolve because pod cache is not initialized")
		return ""
	}

	ip, _, err := net.SplitHostPort(source.String())
	if err != nil {
		ip = source.String()
	}

	pod := k.podWithIP(ip)
	if pod == nil {
		log.Debug(5, "nshorizon: source IP %s could not be mapped to a pod", ip)
		return ""
	}

	return pod.Namespace
}

func (k *Kubernetes) ClusterZone() string {
	return k.primaryZone()
}
