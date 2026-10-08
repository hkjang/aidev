package config

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os"
	"strings"
)

// Config contains the required deployment inputs and narrowly scoped transport
// settings. Mutable service policy remains in PostgreSQL and is managed in the
// UI.
type Config struct {
	PostgresDSN            string
	BootstrapAdmin         string
	BootstrapAdminPassword string
	EncryptionKey          []byte
	EncryptionPreviousKeys [][]byte
	HTTPAddr               string
	TrustedProxyCIDRs      []netip.Prefix
}

func Load() (Config, error) {
	cfg := Config{
		PostgresDSN:            strings.TrimSpace(os.Getenv("POSTGRES_DSN")),
		BootstrapAdmin:         strings.TrimSpace(os.Getenv("BOOTSTRAP_ADMIN")),
		BootstrapAdminPassword: os.Getenv("BOOTSTRAP_ADMIN_PASSWORD"),
		HTTPAddr:               strings.TrimSpace(os.Getenv("UMM_HTTP_ADDR")),
	}
	if cfg.HTTPAddr == "" {
		cfg.HTTPAddr = ":8080"
	}
	if _, _, err := net.SplitHostPort(cfg.HTTPAddr); err != nil {
		return Config{}, fmt.Errorf("UMM_HTTP_ADDR must be a valid host:port address: %w", err)
	}
	trustedProxies, err := parseTrustedProxyCIDRs(os.Getenv("UMM_TRUSTED_PROXY_CIDRS"))
	if err != nil {
		return Config{}, err
	}
	cfg.TrustedProxyCIDRs = trustedProxies
	if cfg.PostgresDSN == "" || cfg.BootstrapAdmin == "" || cfg.BootstrapAdminPassword == "" {
		return Config{}, errors.New("POSTGRES_DSN, BOOTSTRAP_ADMIN, BOOTSTRAP_ADMIN_PASSWORD and ENCRYPTION_KEY are required")
	}
	key, err := parseKey(os.Getenv("ENCRYPTION_KEY"))
	if err != nil {
		return Config{}, err
	}
	cfg.EncryptionKey = key
	previousRaw := strings.TrimSpace(os.Getenv("ENCRYPTION_KEY_PREVIOUS"))
	if previousRaw != "" {
		for _, value := range strings.Split(previousRaw, ",") {
			previous, parseErr := parseKey(value)
			if parseErr != nil {
				return Config{}, fmt.Errorf("ENCRYPTION_KEY_PREVIOUS: %w", parseErr)
			}
			cfg.EncryptionPreviousKeys = append(cfg.EncryptionPreviousKeys, previous)
		}
	}
	return cfg, nil
}

func parseTrustedProxyCIDRs(raw string) ([]netip.Prefix, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	prefixes := make([]netip.Prefix, 0, strings.Count(raw, ",")+1)
	for _, item := range strings.Split(raw, ",") {
		value := strings.TrimSpace(item)
		if value == "" {
			return nil, errors.New("UMM_TRUSTED_PROXY_CIDRS contains an empty entry")
		}
		prefix, err := netip.ParsePrefix(value)
		if err != nil {
			address, addressErr := netip.ParseAddr(value)
			if addressErr != nil || address.Zone() != "" {
				return nil, fmt.Errorf("UMM_TRUSTED_PROXY_CIDRS entry %q must be an IP address or CIDR", value)
			}
			address = address.Unmap()
			prefix = netip.PrefixFrom(address, address.BitLen())
		} else if prefix.Addr().Is4In6() && prefix.Bits() >= 96 {
			prefix = netip.PrefixFrom(prefix.Addr().Unmap(), prefix.Bits()-96)
		}
		if !prefix.IsValid() || prefix.Addr().Zone() != "" || prefix.Bits() == 0 {
			return nil, fmt.Errorf("UMM_TRUSTED_PROXY_CIDRS entry %q is not a usable network", value)
		}
		prefixes = append(prefixes, prefix.Masked())
	}
	return prefixes, nil
}

func parseKey(raw string) ([]byte, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, errors.New("ENCRYPTION_KEY is required")
	}
	for _, decode := range []func(string) ([]byte, error){base64.StdEncoding.DecodeString, hex.DecodeString} {
		if key, err := decode(raw); err == nil && len(key) == 32 {
			return key, nil
		}
	}
	if len([]byte(raw)) == 32 {
		return []byte(raw), nil
	}
	return nil, fmt.Errorf("ENCRYPTION_KEY must be exactly 32 bytes, 64 hex characters, or base64-encoded 32 bytes (sha256 hint: %x)", sha256.Sum256([]byte(raw)))
}
