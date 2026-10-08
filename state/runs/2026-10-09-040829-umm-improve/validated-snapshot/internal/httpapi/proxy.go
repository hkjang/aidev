package httpapi

import (
	"net"
	"net/http"
	"net/netip"
	"strings"
)

var proxyControlledHeaders = []string{
	"Forwarded",
	"True-Client-IP",
	"X-Forwarded-For",
	"X-Forwarded-Host",
	"X-Forwarded-Port",
	"X-Forwarded-Proto",
	"X-Real-IP",
}

// trustedProxyHeaders accepts forwarding metadata only from an explicitly
// configured socket peer. It walks X-Forwarded-For from the nearest proxy back
// to the first untrusted address, so a client-supplied left edge cannot spoof
// the address even when a proxy appends instead of replacing the header.
func (s *Server) trustedProxyHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		peer, ok := remoteAddressIP(r.RemoteAddr)
		if !ok || !addressInPrefixes(peer, s.TrustedProxies) {
			stripProxyHeaders(r.Header)
			next.ServeHTTP(w, r)
			return
		}

		if client, resolved := forwardedClientIP(r.Header, peer, s.TrustedProxies); resolved {
			r.RemoteAddr = net.JoinHostPort(client.String(), "0")
		}
		for _, name := range []string{"Forwarded", "True-Client-IP", "X-Forwarded-For", "X-Forwarded-Host", "X-Forwarded-Port", "X-Real-IP"} {
			r.Header.Del(name)
		}
		sanitizeForwardedProto(r.Header)
		next.ServeHTTP(w, r)
	})
}

func stripProxyHeaders(header http.Header) {
	for _, name := range proxyControlledHeaders {
		header.Del(name)
	}
}

func addressInPrefixes(address netip.Addr, prefixes []netip.Prefix) bool {
	address = address.Unmap()
	for _, prefix := range prefixes {
		if prefix.Contains(address) {
			return true
		}
	}
	return false
}

func remoteAddressIP(remoteAddress string) (netip.Addr, bool) {
	raw := strings.TrimSpace(remoteAddress)
	if host, _, err := net.SplitHostPort(raw); err == nil {
		raw = host
	}
	raw = strings.TrimPrefix(strings.TrimSuffix(raw, "]"), "[")
	address, err := netip.ParseAddr(raw)
	if err != nil || address.Zone() != "" {
		return netip.Addr{}, false
	}
	return address.Unmap(), true
}

func forwardedClientIP(header http.Header, peer netip.Addr, trusted []netip.Prefix) (netip.Addr, bool) {
	values := header.Values("X-Forwarded-For")
	if len(values) > 0 {
		parts := strings.Split(strings.Join(values, ","), ",")
		candidate := peer.Unmap()
		for index := len(parts) - 1; index >= 0; index-- {
			if !addressInPrefixes(candidate, trusted) {
				return candidate, true
			}
			next, err := netip.ParseAddr(strings.TrimSpace(parts[index]))
			if err != nil || next.Zone() != "" {
				return netip.Addr{}, false
			}
			candidate = next.Unmap()
		}
		return candidate, true
	}

	if raw := strings.TrimSpace(header.Get("X-Real-IP")); raw != "" {
		address, err := netip.ParseAddr(raw)
		if err == nil && address.Zone() == "" {
			return address.Unmap(), true
		}
	}
	return netip.Addr{}, false
}

func sanitizeForwardedProto(header http.Header) {
	values := strings.Split(strings.Join(header.Values("X-Forwarded-Proto"), ","), ",")
	if len(values) == 0 {
		header.Del("X-Forwarded-Proto")
		return
	}
	scheme := strings.ToLower(strings.TrimSpace(values[len(values)-1]))
	if scheme != "http" && scheme != "https" {
		header.Del("X-Forwarded-Proto")
		return
	}
	header.Set("X-Forwarded-Proto", scheme)
}
