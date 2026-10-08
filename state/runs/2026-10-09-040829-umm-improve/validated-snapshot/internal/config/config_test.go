package config

import (
	"encoding/base64"
	"net/netip"
	"testing"
)

func TestParseKeyFormats(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	for _, input := range []string{string(key), base64.StdEncoding.EncodeToString(key)} {
		got, err := parseKey(input)
		if err != nil {
			t.Fatalf("parse key: %v", err)
		}
		if string(got) != string(key) {
			t.Fatal("parsed key mismatch")
		}
	}
}
func TestParseKeyRejectsWeakLength(t *testing.T) {
	if _, err := parseKey("too-short"); err == nil {
		t.Fatal("expected invalid key error")
	}
}

func TestLoadUsesOptionalHTTPAddress(t *testing.T) {
	t.Setenv("POSTGRES_DSN", "postgres://example/umm")
	t.Setenv("BOOTSTRAP_ADMIN", "admin")
	t.Setenv("BOOTSTRAP_ADMIN_PASSWORD", "test-password")
	t.Setenv("ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")
	t.Setenv("UMM_HTTP_ADDR", "127.0.0.1:18081")
	config, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if config.HTTPAddr != "127.0.0.1:18081" {
		t.Fatalf("HTTPAddr = %q", config.HTTPAddr)
	}
}

func TestLoadRejectsInvalidHTTPAddress(t *testing.T) {
	t.Setenv("POSTGRES_DSN", "postgres://example/umm")
	t.Setenv("BOOTSTRAP_ADMIN", "admin")
	t.Setenv("BOOTSTRAP_ADMIN_PASSWORD", "test-password")
	t.Setenv("ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")
	t.Setenv("UMM_HTTP_ADDR", "127.0.0.1")
	if _, err := Load(); err == nil {
		t.Fatal("invalid UMM_HTTP_ADDR was accepted")
	}
}

func TestLoadParsesTrustedProxyNetworks(t *testing.T) {
	t.Setenv("POSTGRES_DSN", "postgres://example/umm")
	t.Setenv("BOOTSTRAP_ADMIN", "admin")
	t.Setenv("BOOTSTRAP_ADMIN_PASSWORD", "test-password")
	t.Setenv("ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")
	t.Setenv("UMM_TRUSTED_PROXY_CIDRS", "127.0.0.1, 10.0.0.8/8, 2001:db8::/32")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	want := []netip.Prefix{
		netip.MustParsePrefix("127.0.0.1/32"),
		netip.MustParsePrefix("10.0.0.0/8"),
		netip.MustParsePrefix("2001:db8::/32"),
	}
	if len(cfg.TrustedProxyCIDRs) != len(want) {
		t.Fatalf("expected %d trusted proxy networks, got %v", len(want), cfg.TrustedProxyCIDRs)
	}
	for index := range want {
		if cfg.TrustedProxyCIDRs[index] != want[index] {
			t.Fatalf("network %d: want %s, got %s", index, want[index], cfg.TrustedProxyCIDRs[index])
		}
	}
}

func TestLoadRejectsInvalidTrustedProxyNetwork(t *testing.T) {
	t.Setenv("POSTGRES_DSN", "postgres://example/umm")
	t.Setenv("BOOTSTRAP_ADMIN", "admin")
	t.Setenv("BOOTSTRAP_ADMIN_PASSWORD", "test-password")
	t.Setenv("ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")
	for _, value := range []string{"not-a-network", "10.0.0.0/99", "127.0.0.1,", "0.0.0.0/0", "::/0"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("UMM_TRUSTED_PROXY_CIDRS", value)
			if _, err := Load(); err == nil {
				t.Fatalf("invalid trusted proxy value %q was accepted", value)
			}
		})
	}
}
