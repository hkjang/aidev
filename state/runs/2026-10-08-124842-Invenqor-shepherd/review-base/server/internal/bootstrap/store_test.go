package bootstrap

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestStoreEncryptsAndLoadsBootstrapValues(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	store, err := Open(root)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	values := Values{
		PostgresDSN:          "postgres://user:secret@db.example/invenqor",
		SQLitePath:           filepath.Join(root, "invenqor.db"),
		KeycloakClientSecret: "oidc-secret",
	}
	if err := store.Save(values); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(root, bootstrapFileName))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	for _, secret := range []string{"secret", "oidc-secret", values.PostgresDSN} {
		if bytes.Contains(raw, []byte(secret)) {
			t.Fatalf("encrypted bootstrap file contains plaintext %q", secret)
		}
	}
	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded != values {
		t.Fatalf("Load() = %#v, want %#v", loaded, values)
	}
}

func TestStoreRejectsTamperedCiphertext(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	store, err := Open(root)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if err := store.Save(Values{PostgresDSN: "postgres://example"}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	path := filepath.Join(root, bootstrapFileName)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	raw[len(raw)/2] ^= 1
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if _, err := store.Load(); err == nil {
		t.Fatal("Load() accepted tampered ciphertext")
	}
}

func TestStoreUsesRestrictivePermissions(t *testing.T) {
	t.Parallel()
	root := filepath.Join(t.TempDir(), "state")
	store, err := Open(root)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if err := store.Save(Values{}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	for _, name := range []string{keyFileName, bootstrapFileName} {
		info, err := os.Stat(filepath.Join(root, name))
		if err != nil {
			t.Fatalf("Stat(%s) error = %v", name, err)
		}
		if permission := info.Mode().Perm(); permission != 0o600 {
			t.Errorf("%s permission = %o, want 600", name, permission)
		}
	}
	info, err := os.Stat(root)
	if err != nil {
		t.Fatalf("Stat(root) error = %v", err)
	}
	if permission := info.Mode().Perm(); permission != 0o700 {
		t.Errorf("state directory permission = %o, want 700", permission)
	}
}

func TestRuntimeSecretUsesPurposeBoundEncryption(t *testing.T) {
	t.Parallel()
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	encrypted, err := store.SealString("oidc.pkce", "verifier-secret")
	if err != nil {
		t.Fatalf("SealString() error = %v", err)
	}
	if bytes.Contains([]byte(encrypted), []byte("verifier-secret")) {
		t.Fatal("encrypted runtime secret contains plaintext")
	}
	plaintext, err := store.OpenString("oidc.pkce", encrypted)
	if err != nil {
		t.Fatalf("OpenString() error = %v", err)
	}
	if plaintext != "verifier-secret" {
		t.Fatalf("OpenString() = %q", plaintext)
	}
	if _, err := store.OpenString("oidc.other", encrypted); err == nil {
		t.Fatal("OpenString() accepted a different encryption purpose")
	}
}

func TestExternalMasterKeySupportsMultiplePodStores(t *testing.T) {
	root := t.TempDir()
	keyPath := filepath.Join(root, "master.key")
	if err := os.WriteFile(keyPath, bytes.Repeat([]byte{7}, 32), 0o400); err != nil {
		t.Fatal(err)
	}
	first, err := OpenWithKey(filepath.Join(root, "pod-a"), keyPath)
	if err != nil {
		t.Fatal(err)
	}
	second, err := OpenWithKey(filepath.Join(root, "pod-b"), keyPath)
	if err != nil {
		t.Fatal(err)
	}
	sealed, err := first.SealString("cluster-secret", "shared")
	if err != nil {
		t.Fatal(err)
	}
	plain, err := second.OpenString("cluster-secret", sealed)
	if err != nil || plain != "shared" {
		t.Fatalf("second pod OpenString() = %q, %v", plain, err)
	}
}
