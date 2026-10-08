package httpapi

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
)

func TestTrustedProxyHeadersIgnoreUntrustedPeer(t *testing.T) {
	server := &Server{TrustedProxies: []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")}}
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.RemoteAddr = "203.0.113.7:4321"
	request.Header.Set("X-Forwarded-For", "198.51.100.9")
	request.Header.Set("X-Real-IP", "198.51.100.10")
	request.Header.Set("True-Client-IP", "198.51.100.11")
	request.Header.Set("X-Forwarded-Proto", "https")

	var gotIP, gotProto string
	handler := server.trustedProxyHeaders(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		gotIP = clientIP(r)
		gotProto = r.Header.Get("X-Forwarded-Proto")
		for _, name := range proxyControlledHeaders {
			if value := r.Header.Get(name); value != "" {
				t.Errorf("untrusted %s survived as %q", name, value)
			}
		}
	}))
	handler.ServeHTTP(httptest.NewRecorder(), request)

	if gotIP != "203.0.113.7" {
		t.Fatalf("untrusted forwarding header changed the client IP to %q", gotIP)
	}
	if gotProto != "" {
		t.Fatalf("untrusted forwarding proto survived as %q", gotProto)
	}
}

func TestTrustedProxyHeadersWalkTheTrustedChainFromTheRight(t *testing.T) {
	server := &Server{TrustedProxies: []netip.Prefix{
		netip.MustParsePrefix("10.0.0.0/8"),
		netip.MustParsePrefix("192.0.2.0/24"),
	}}
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.RemoteAddr = "10.1.2.3:4321"
	// The leftmost value is attacker supplied. The first untrusted address seen
	// from the right is the actual client appended by the trusted proxy chain.
	request.Header.Set("X-Forwarded-For", "203.0.113.66, 198.51.100.9, 192.0.2.20")
	request.Header.Set("X-Forwarded-Proto", "javascript, https")

	var gotIP, gotProto string
	handler := server.trustedProxyHeaders(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		gotIP = clientIP(r)
		gotProto = r.Header.Get("X-Forwarded-Proto")
	}))
	handler.ServeHTTP(httptest.NewRecorder(), request)

	if gotIP != "198.51.100.9" {
		t.Fatalf("expected the first untrusted address from the right, got %q", gotIP)
	}
	if gotProto != "https" {
		t.Fatalf("expected the nearest proxy's sanitized scheme, got %q", gotProto)
	}
}

func TestTrustedProxyHeadersSupportIPv6AndRealIPFallback(t *testing.T) {
	server := &Server{TrustedProxies: []netip.Prefix{netip.MustParsePrefix("2001:db8:ffff::/48")}}
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.RemoteAddr = "[2001:db8:ffff::10]:4321"
	request.Header.Set("X-Real-IP", "2001:db8:abcd::7")

	var got string
	server.trustedProxyHeaders(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = clientIP(r)
	})).ServeHTTP(httptest.NewRecorder(), request)

	if got != "2001:db8:abcd::7" {
		t.Fatalf("expected the forwarded IPv6 client, got %q", got)
	}
}

func TestTrustedProxyHeadersFailClosedOnMalformedForwardingChain(t *testing.T) {
	server := &Server{TrustedProxies: []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")}}
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.RemoteAddr = "10.1.2.3:4321"
	request.Header.Set("X-Forwarded-For", "198.51.100.9, malformed")

	var got string
	server.trustedProxyHeaders(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = clientIP(r)
	})).ServeHTTP(httptest.NewRecorder(), request)

	if got != "10.1.2.3" {
		t.Fatalf("a malformed chain must fall back to the socket peer, got %q", got)
	}
}
