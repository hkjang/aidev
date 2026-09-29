package platform

import (
	"bytes"
	"path/filepath"
	"testing"
)

func TestKeyringEncryptAndDigest(t *testing.T) {
	path := filepath.Join(t.TempDir(), "keys", "master.key")
	keys, err := NewKeyring(path)
	if err != nil {
		t.Fatal(err)
	}
	ciphertext, err := keys.Encrypt("client-secret")
	if err != nil {
		t.Fatal(err)
	}
	if ciphertext == "client-secret" {
		t.Fatal("secret was not encrypted")
	}
	plain, err := keys.Decrypt(ciphertext)
	if err != nil || plain != "client-secret" {
		t.Fatalf("decrypt: %q, %v", plain, err)
	}
	if !bytes.Equal(keys.Digest("same"), keys.Digest("same")) {
		t.Fatal("digest must be stable")
	}
	if bytes.Equal(keys.Digest("same"), keys.Digest("different")) {
		t.Fatal("different values must not share a digest")
	}

	reloaded, err := NewKeyring(path)
	if err != nil {
		t.Fatal(err)
	}
	if plain, err = reloaded.Decrypt(ciphertext); err != nil || plain != "client-secret" {
		t.Fatal("persisted key cannot decrypt")
	}
}
