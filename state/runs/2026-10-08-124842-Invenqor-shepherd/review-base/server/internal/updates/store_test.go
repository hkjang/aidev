package updates

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

const testSignature = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=="

func TestPublishSelectAndReadArtifact(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	payload := []byte("signed-update")
	manifest, err := store.Publish(Manifest{
		Version: "1.2.3", Channel: "stable", OS: "linux",
		Architecture: "x86_64", Signature: testSignature,
		ManifestSignature: testSignature,
		SignatureScheme:   SignatureSchemeEd25519,
		SignatureVersion:  SignatureVersionV2, Rollout: 100,
	}, bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	expected := sha256.Sum256(payload)
	if manifest.SHA256 != hex.EncodeToString(expected[:]) {
		t.Fatalf("sha256 = %s", manifest.SHA256)
	}
	latest, err := store.Latest("stable", "linux", "x86_64", "agent-1")
	if err != nil || latest == nil || latest.Version != "1.2.3" {
		t.Fatalf("Latest() = %#v, %v", latest, err)
	}
	path, err := store.Artifact("1.2.3-linux-x86_64")
	if err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	if !bytes.Equal(got, payload) {
		t.Fatal("artifact content changed")
	}
}

func TestRolloutAndPathValidation(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.Publish(Manifest{
		Version: "1.0.0", Channel: "stable", OS: "linux",
		Architecture: "x86_64", Signature: testSignature,
		ManifestSignature: testSignature,
		SignatureScheme:   SignatureSchemeEd25519,
		SignatureVersion:  SignatureVersionV2, Rollout: 0,
	}, bytes.NewReader([]byte("x")))
	if err != nil {
		t.Fatal(err)
	}
	latest, err := store.Latest("stable", "linux", "x86_64", "agent")
	if err != nil || latest != nil {
		t.Fatalf("zero rollout returned %#v, %v", latest, err)
	}
	if _, err := store.Artifact("../master.key"); err == nil {
		t.Fatal("Artifact accepted path traversal")
	}
}

func TestPublishedVersionIsImmutable(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	manifest := Manifest{
		Version: "1.0.0", Channel: "stable", OS: "linux",
		Architecture: "x86_64", Signature: testSignature,
		ManifestSignature: testSignature,
		SignatureScheme:   SignatureSchemeEd25519,
		SignatureVersion:  SignatureVersionV2, Rollout: 100,
	}
	if _, err := store.Publish(manifest, bytes.NewReader([]byte("first"))); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Publish(manifest, bytes.NewReader([]byte("replacement"))); err == nil {
		t.Fatal("Publish replaced an immutable version")
	}
	path, err := store.Artifact("1.0.0-linux-x86_64")
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "first" {
		t.Fatalf("artifact = %q, %v", got, err)
	}
}

func TestLegacyReleaseIsServedWithAnExplicitSignatureContract(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	legacy := []byte(`{
  "version": "1.2.3",
  "channel": "stable",
  "os": "linux",
  "architecture": "x86_64",
  "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
  "signature": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
  "download_url": "/v1/agent/updates/1.2.3-linux-x86_64/artifact",
  "size": 1,
  "rollout_percent": 100
}`)
	if err := os.WriteFile(
		filepath.Join(store.root, "1.2.3-linux-x86_64.json"), legacy, 0o600,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(store.root, "1.2.3-linux-x86_64.bin"), []byte("x"), 0o600,
	); err != nil {
		t.Fatal(err)
	}

	latest, err := store.Latest("stable", "linux", "x86_64", "agent-1")
	if err != nil {
		t.Fatal(err)
	}
	if latest == nil || latest.SignatureScheme != SignatureSchemeEd25519 ||
		latest.SignatureVersion != SignatureVersionLegacy {
		t.Fatalf("legacy manifest contract = %#v", latest)
	}
	wire, err := json.Marshal(latest)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(wire, []byte(`"signature_scheme":"ed25519"`)) ||
		!bytes.Contains(wire, []byte(`"signature_version":1`)) {
		t.Fatalf("legacy manifest JSON did not make its contract explicit: %s", wire)
	}
}
