package updates

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const (
	SignatureSchemeEd25519 = "ed25519"
	SignatureVersionLegacy = 1
	SignatureVersionV2     = 2
)

const signatureDomainV2 = "INVENQOR-AGENT-UPDATE-MANIFEST-V2"

// Publishing used to accept any 64-byte value as a signature. Nothing on the
// server could check it, because the server held no public key - so a mistyped
// or stale signature published successfully and then failed verification on
// every agent in the fleet, forever, with the only symptom a line in each
// agent's log. Publishing is the one moment where the mistake is cheap to
// catch, so the key belongs here.
//
// The key is optional: a deployment that signs elsewhere and trusts its release
// pipeline can leave it unset, and the API says so rather than pretending the
// signature was checked.

var (
	ErrSignatureUnverifiable = errors.New(
		"the update signing public key is not configured on this server",
	)
	ErrSignatureRejected = errors.New(
		"the signature does not verify against the configured signing key",
	)
)

// ParsePublicKey accepts the same base64 Ed25519 public key the agents pin.
func ParsePublicKey(value string) (ed25519.PublicKey, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}
	decoded, err := base64.StdEncoding.DecodeString(trimmed)
	if err != nil {
		decoded, err = base64.RawStdEncoding.DecodeString(trimmed)
	}
	if err != nil {
		return nil, fmt.Errorf("decode update signing key: %w", err)
	}
	if len(decoded) != ed25519.PublicKeySize {
		return nil, fmt.Errorf(
			"update signing key must be %d bytes, got %d",
			ed25519.PublicKeySize, len(decoded),
		)
	}
	return ed25519.PublicKey(decoded), nil
}

// DecodeSignature accepts the base64 detached signature an operator pastes or
// uploads, tolerating the line breaks `base64` inserts by default.
func DecodeSignature(value string) ([]byte, error) {
	cleaned := strings.Map(func(char rune) rune {
		if char == '\n' || char == '\r' || char == ' ' || char == '\t' {
			return -1
		}
		return char
	}, value)
	if cleaned == "" {
		return nil, errors.New("the signature is empty")
	}
	decoded, err := base64.StdEncoding.DecodeString(cleaned)
	if err != nil {
		decoded, err = base64.RawStdEncoding.DecodeString(cleaned)
	}
	if err != nil {
		return nil, fmt.Errorf("decode signature: %w", err)
	}
	if len(decoded) != ed25519.SignatureSize {
		return nil, fmt.Errorf(
			"an Ed25519 signature is %d bytes, got %d",
			ed25519.SignatureSize, len(decoded),
		)
	}
	return decoded, nil
}

// SigningKeyConfigured reports whether publish-time verification is possible.
func (s *Store) SigningKeyConfigured() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.publicKey) == ed25519.PublicKeySize
}

// SetSigningKey installs the key used to verify a publication.
func (s *Store) SetSigningKey(key ed25519.PublicKey) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.publicKey = key
}

// SignatureMessageV2 returns the complete byte contract signed by an offline
// Ed25519 key and verified independently by the server and Agent. The artifact
// itself is bound by both its byte length and lowercase SHA-256 digest. Values
// are UTF-8, separators are literal ASCII, booleans are lowercase, sizes are
// unsigned base-10 integers, and the final newline is mandatory.
//
// Rollout, notes, publication audit fields and download URL are intentionally
// absent: the server can change or derive those fields without authorizing a
// different executable or silently enabling rollback.
func SignatureMessageV2(manifest Manifest) ([]byte, error) {
	if manifest.SignatureScheme != SignatureSchemeEd25519 ||
		manifest.SignatureVersion != SignatureVersionV2 {
		return nil, fmt.Errorf(
			"manifest signature contract must be %s version %d",
			SignatureSchemeEd25519, SignatureVersionV2,
		)
	}
	if manifest.Size <= 0 {
		return nil, errors.New("manifest size must be positive")
	}
	if len(manifest.SHA256) != sha256HexSize ||
		manifest.SHA256 != strings.ToLower(manifest.SHA256) {
		return nil, errors.New("manifest sha256 must be 64 lowercase hexadecimal characters")
	}
	if _, err := hex.DecodeString(manifest.SHA256); err != nil {
		return nil, errors.New("manifest sha256 must be 64 lowercase hexadecimal characters")
	}
	for field, value := range map[string]string{
		"version": manifest.Version, "channel": manifest.Channel,
		"os": manifest.OS, "architecture": manifest.Architecture,
	} {
		if value == "" || strings.ContainsAny(value, "\r\n") {
			return nil, fmt.Errorf("manifest %s is empty or contains a line break", field)
		}
	}

	var message strings.Builder
	message.Grow(256)
	message.WriteString(signatureDomainV2)
	message.WriteString("\nversion=")
	message.WriteString(manifest.Version)
	message.WriteString("\nchannel=")
	message.WriteString(manifest.Channel)
	message.WriteString("\nos=")
	message.WriteString(manifest.OS)
	message.WriteString("\narchitecture=")
	message.WriteString(manifest.Architecture)
	message.WriteString("\nsize=")
	message.WriteString(strconv.FormatInt(manifest.Size, 10))
	message.WriteString("\nsha256=")
	message.WriteString(manifest.SHA256)
	message.WriteString("\nallow_downgrade=")
	message.WriteString(strconv.FormatBool(manifest.AllowDowngrade))
	message.WriteByte('\n')
	return []byte(message.String()), nil
}

const sha256HexSize = 64

func normalizeSignatureContract(manifest *Manifest) {
	if manifest.SignatureScheme == "" {
		manifest.SignatureScheme = SignatureSchemeEd25519
	}
	if manifest.SignatureVersion == 0 {
		manifest.SignatureVersion = SignatureVersionLegacy
	}
}

// verifySignature checks a detached signature over the supplied signature
// contract. New publications always supply SignatureMessageV2.
func (s *Store) verifySignature(signature []byte, message []byte) error {
	s.mu.RLock()
	key := s.publicKey
	s.mu.RUnlock()
	if len(key) != ed25519.PublicKeySize {
		return ErrSignatureUnverifiable
	}
	if !ed25519.Verify(key, message, signature) {
		return ErrSignatureRejected
	}
	return nil
}
