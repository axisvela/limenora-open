package network

import (
	"fmt"
	"net/netip"
	"net/url"
	"strings"
)

// cloudMetadataHosts are well-known cloud instance-metadata hostnames. They
// resolve to the link-local IMDS addresses below, so they are denied by name
// as well as by IP (deny by default, independent of the network mode).
var cloudMetadataHosts = []string{
	"metadata.google.internal", // GCP
	"metadata.azure.internal",  // Azure
}

// cloudMetadataIPs are well-known cloud instance-metadata addresses, including
// the IPv4 link-local IMDS range and its IPv6 variants. These hosts must never
// be reachable through the gateway, including under full/host network.
var cloudMetadataIPs = []netip.Addr{
	netip.MustParseAddr("169.254.169.254"), // AWS / GCP / Azure / Alibaba IMDS
	netip.MustParseAddr("168.63.129.16"),   // Azure WireServer (metadata proxy)
	netip.MustParseAddr("fe80::a9fe:a9fe"), // AWS IMDS (IPv6 link-local)
	netip.MustParseAddr("fd00:ec2::254"),   // AWS ECS (IPv6)
}

// cloudMetadataPrefixes cover whole metadata ranges rather than enumerating
// every host address.
var cloudMetadataPrefixes = []netip.Prefix{
	netip.MustParsePrefix("169.254.170.0/23"), // AWS ECS container + task metadata
}

// cloudMetadataDenied reports whether host names a cloud metadata endpoint.
// host may include an optional port or bracketed IPv6 literal.
func cloudMetadataDenied(host string) bool {
	h := strings.ToLower(strings.TrimSpace(host))
	if ip, err := netip.ParseAddr(h); err == nil {
		return inCloudMetadataIPs(ip)
	}
	// Strip a port and any IPv6 brackets.
	if u, err := url.Parse("//" + h); err == nil && u.Host != "" {
		h = u.Hostname()
	}
	for _, name := range cloudMetadataHosts {
		if h == name {
			return true
		}
	}
	if ip, err := netip.ParseAddr(h); err == nil {
		return inCloudMetadataIPs(ip)
	}
	return false
}

func inCloudMetadataIPs(ip netip.Addr) bool {
	for _, denied := range cloudMetadataIPs {
		if ip == denied {
			return true
		}
	}
	for _, prefix := range cloudMetadataPrefixes {
		if prefix.Contains(ip) {
			return true
		}
	}
	return false
}

// requestDeniedHost returns the host of rawURL when it names a cloud metadata
// endpoint, or "" when the destination is allowed. An unparsable URL yields ""
// so the caller reports the build error as usual.
func requestDeniedHost(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	if cloudMetadataDenied(u.Hostname()) {
		return u.Hostname()
	}
	return ""
}

// denyMetadataDestination rejects requests targeting cloud metadata endpoints
// before any outbound traffic is attempted (deny by default).
func denyMetadataDestination(req GatewayRequest) error {
	if host := requestDeniedHost(req.URL); host != "" {
		return fmt.Errorf("network: destination denied: cloud metadata endpoint %q", host)
	}
	return nil
}
