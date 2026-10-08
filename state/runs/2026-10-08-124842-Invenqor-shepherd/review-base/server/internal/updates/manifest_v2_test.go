package updates

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func TestSignatureMessageV2CanonicalContract(t *testing.T) {
	digest := sha256.Sum256([]byte("agent-binary"))
	manifest := Manifest{
		Version:          "1.2.3",
		Channel:          "stable",
		OS:               "linux",
		Architecture:     "x86_64",
		Size:             12,
		SHA256:           hex.EncodeToString(digest[:]),
		AllowDowngrade:   true,
		SignatureScheme:  SignatureSchemeEd25519,
		SignatureVersion: SignatureVersionV2,
	}

	message, err := SignatureMessageV2(manifest)
	if err != nil {
		t.Fatal(err)
	}
	want := "INVENQOR-AGENT-UPDATE-MANIFEST-V2\n" +
		"version=1.2.3\n" +
		"channel=stable\n" +
		"os=linux\n" +
		"architecture=x86_64\n" +
		"size=12\n" +
		"sha256=" + hex.EncodeToString(digest[:]) + "\n" +
		"allow_downgrade=true\n"
	if string(message) != want {
		t.Fatalf("canonical message = %q, want %q", message, want)
	}
}

func TestSignatureMessageV2BindsEveryProtectedField(t *testing.T) {
	seed := sha256.Sum256([]byte("invenqor manifest v2 test key"))
	private := ed25519.NewKeyFromSeed(seed[:])
	digest := sha256.Sum256([]byte("agent-binary"))
	base := Manifest{
		Version:          "1.2.3",
		Channel:          "stable",
		OS:               "linux",
		Architecture:     "x86_64",
		Size:             12,
		SHA256:           hex.EncodeToString(digest[:]),
		SignatureScheme:  SignatureSchemeEd25519,
		SignatureVersion: SignatureVersionV2,
	}
	message, err := SignatureMessageV2(base)
	if err != nil {
		t.Fatal(err)
	}
	signature := ed25519.Sign(private, message)
	otherDigest := sha256.Sum256([]byte("different-agent-binary"))

	tests := []struct {
		name   string
		mutate func(*Manifest)
	}{
		{"version", func(m *Manifest) { m.Version = "1.2.4" }},
		{"channel", func(m *Manifest) { m.Channel = "beta" }},
		{"os", func(m *Manifest) { m.OS = "windows" }},
		{"architecture", func(m *Manifest) { m.Architecture = "aarch64" }},
		{"size", func(m *Manifest) { m.Size++ }},
		{"sha256", func(m *Manifest) { m.SHA256 = hex.EncodeToString(otherDigest[:]) }},
		{"allow_downgrade", func(m *Manifest) { m.AllowDowngrade = true }},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			mutated := base
			testCase.mutate(&mutated)
			mutatedMessage, err := SignatureMessageV2(mutated)
			if err != nil {
				t.Fatal(err)
			}
			if ed25519.Verify(private.Public().(ed25519.PublicKey), mutatedMessage, signature) {
				t.Fatalf("signature still verified after mutating %s", testCase.name)
			}
		})
	}
}

func TestPublishRequiresManifestSignatureV2(t *testing.T) {
	store := newStore(t)
	candidate := manifest(testSignature)
	candidate.SignatureVersion = SignatureVersionLegacy
	_, err := store.Publish(candidate, strings.NewReader("agent-binary"))
	if err == nil || !strings.Contains(err.Error(), "signature_version") {
		t.Fatalf("Publish() error = %v, want a v2 requirement", err)
	}
}
