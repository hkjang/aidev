package cryptoutil

import (
	"crypto/rand"
	"encoding/base64"
	"io"
	"strings"
	"testing"
)

func TestCipherRoundTripAndTamper(t *testing.T) {
	c, err := New([]byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	sealed, err := c.Encrypt("client-secret")
	if err != nil {
		t.Fatal(err)
	}
	plain, err := c.Decrypt(sealed)
	if err != nil || plain != "client-secret" {
		t.Fatalf("round trip failed: %q %v", plain, err)
	}
	parts := strings.SplitN(sealed, ".", 3)
	if len(parts) != 3 {
		t.Fatalf("unexpected ciphertext format: %q", sealed)
	}
	tampered, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		t.Fatal(err)
	}
	tampered[len(tampered)-1] ^= 1
	if _, err = c.Decrypt(parts[0] + "." + parts[1] + "." + base64.RawURLEncoding.EncodeToString(tampered)); err == nil {
		t.Fatal("tampered ciphertext must fail")
	}
}

func TestCipherKeyRotationAndLegacyCompatibility(t *testing.T) {
	oldKey := []byte("0123456789abcdef0123456789abcdef")
	newKey := []byte("abcdef0123456789abcdef0123456789")
	oldCipher, err := New(oldKey)
	if err != nil {
		t.Fatal(err)
	}
	oldValue, err := oldCipher.Encrypt("rotating-secret")
	if err != nil {
		t.Fatal(err)
	}
	keyring, err := NewWithPrevious(newKey, oldKey, oldKey)
	if err != nil {
		t.Fatal(err)
	}
	if keyring.FallbackKeyCount() != 1 {
		t.Fatalf("fallback count = %d, want 1", keyring.FallbackKeyCount())
	}
	plain, err := keyring.Decrypt(oldValue)
	if err != nil || plain != "rotating-secret" {
		t.Fatalf("fallback decrypt = %q, %v", plain, err)
	}
	if !keyring.NeedsRotation(oldValue) {
		t.Fatal("ciphertext made with the previous key must need rotation")
	}
	current, err := keyring.Encrypt("current-secret")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(current, "v2."+keyring.KeyID()+".") || keyring.NeedsRotation(current) {
		t.Fatal("new ciphertext must carry the primary key ID")
	}

	nonce := make([]byte, oldCipher.primary.aead.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		t.Fatal(err)
	}
	legacy := oldCipher.primary.aead.Seal(nonce, nonce, []byte("legacy-secret"), []byte("umm:v1"))
	legacyValue := base64.RawURLEncoding.EncodeToString(legacy)
	plain, err = keyring.Decrypt(legacyValue)
	if err != nil || plain != "legacy-secret" || !keyring.NeedsRotation(legacyValue) {
		t.Fatalf("legacy decrypt = %q, %v", plain, err)
	}
}

func TestCipherRejectsUnknownKeyID(t *testing.T) {
	cipher, err := New([]byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = cipher.Decrypt("v2.ffffffffffff.AAAA"); err == nil {
		t.Fatal("unknown key ID must be rejected")
	}
}
