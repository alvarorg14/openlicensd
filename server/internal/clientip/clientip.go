package clientip

import (
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"
)

// Resolver determines the client IP from an HTTP request, optionally walking
// X-Forwarded-For when the direct peer is a trusted proxy.
type Resolver struct {
	trusted []netip.Prefix
}

// NewResolver builds a resolver from a list of trusted proxy IPs or CIDRs.
func NewResolver(entries []string) (*Resolver, error) {
	trusted := make([]netip.Prefix, 0, len(entries))
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if strings.Contains(entry, "/") {
			prefix, err := netip.ParsePrefix(entry)
			if err != nil {
				return nil, fmt.Errorf("invalid trusted proxy CIDR %q: %w", entry, err)
			}
			trusted = append(trusted, prefix)
			continue
		}
		addr, err := netip.ParseAddr(entry)
		if err != nil {
			return nil, fmt.Errorf("invalid trusted proxy IP %q: %w", entry, err)
		}
		bits := addr.BitLen()
		trusted = append(trusted, netip.PrefixFrom(addr, bits))
	}
	return &Resolver{trusted: trusted}, nil
}

// From returns the resolved client IP as a bare address string.
func (r *Resolver) From(req *http.Request) string {
	peerIP, ok := peerAddr(req.RemoteAddr)
	if !ok {
		return req.RemoteAddr
	}
	if r == nil || !r.isTrusted(peerIP) {
		return peerIP.String()
	}

	xff := req.Header.Get("X-Forwarded-For")
	if xff == "" {
		return peerIP.String()
	}

	parts := strings.Split(xff, ",")
	for i := len(parts) - 1; i >= 0; i-- {
		part := strings.TrimSpace(parts[i])
		if part == "" {
			continue
		}
		addr, err := netip.ParseAddr(part)
		if err != nil {
			continue
		}
		if !r.isTrusted(addr) {
			return addr.String()
		}
	}

	return peerIP.String()
}

func peerAddr(remoteAddr string) (netip.Addr, bool) {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	addr, err := netip.ParseAddr(host)
	if err != nil {
		return netip.Addr{}, false
	}
	return addr, true
}

func (r *Resolver) isTrusted(addr netip.Addr) bool {
	for _, prefix := range r.trusted {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}
