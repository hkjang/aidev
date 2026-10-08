package httpapi

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hkjang/invenqor/server/internal/storagetest"

	"github.com/hkjang/invenqor/server/internal/updates"
)

func TestPublishAgentUpdateUsesManifestSignatureV2(t *testing.T) {
	runtime := storagetest.Open(t)
	defer runtime.Close()
	updateStore, err := updates.Open(filepath.Join(t.TempDir(), "updates"))
	if err != nil {
		t.Fatal(err)
	}
	public, private, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	updateStore.SetSigningKey(public)
	server := testServer(t, runtime)
	server.updateStore = updateStore
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)

	artifact := []byte("signed-agent-update")
	digest := sha256.Sum256(artifact)
	manifest := updates.Manifest{
		Version:          "1.2.3",
		Channel:          "stable",
		OS:               "linux",
		Architecture:     "x86_64",
		Size:             int64(len(artifact)),
		SHA256:           hex.EncodeToString(digest[:]),
		SignatureScheme:  updates.SignatureSchemeEd25519,
		SignatureVersion: updates.SignatureVersionV2,
	}
	message, err := updates.SignatureMessageV2(manifest)
	if err != nil {
		t.Fatal(err)
	}
	legacySignature := base64.StdEncoding.EncodeToString(ed25519.Sign(private, artifact))
	manifestSignature := base64.StdEncoding.EncodeToString(ed25519.Sign(private, message))
	bundle, err := json.Marshal(map[string]any{
		"version": "1.2.3", "channel": "stable", "os": "linux",
		"architecture": "x86_64", "size": len(artifact),
		"sha256": hex.EncodeToString(digest[:]), "allow_downgrade": false,
		"signature_scheme": "ed25519", "signature_version": 2,
		"signature": legacySignature, "manifest_signature": manifestSignature,
	})
	if err != nil {
		t.Fatal(err)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("rollout_percent", "25"); err != nil {
		t.Fatal(err)
	}
	part, err := writer.CreateFormFile("artifact", "invenqor-agent")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(artifact); err != nil {
		t.Fatal(err)
	}
	bundlePart, err := writer.CreateFormFile("signature_bundle_file", "signature-bundle.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bundlePart.Write(bundle); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(
		http.MethodPost, "/api/v1/admin/agent-updates", &body,
	)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("X-CSRF-Token", csrf)
	request.AddCookie(cookie)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("publish status = %d body = %s", response.Code, response.Body.String())
	}
	var published updates.Manifest
	if err := json.Unmarshal(response.Body.Bytes(), &published); err != nil {
		t.Fatal(err)
	}
	if published.SignatureScheme != updates.SignatureSchemeEd25519 ||
		published.SignatureVersion != updates.SignatureVersionV2 ||
		published.Signature != legacySignature ||
		published.ManifestSignature != manifestSignature ||
		!published.SignatureVerified || published.Rollout != 25 {
		t.Fatalf("published manifest = %+v", published)
	}
}

func TestAgentUpdateOfferGateProtectsPreV2AgentsFromRollbacks(t *testing.T) {
	normalBridge := &updates.Manifest{
		Version: "0.2.15", SignatureVersion: updates.SignatureVersionV2,
	}
	if !agentUpdateOfferAllowed(normalBridge, "0.2.14") {
		t.Fatal("v0.2.14 was not offered its normal dual-signature bridge upgrade")
	}
	v2Rollback := &updates.Manifest{
		Version: "0.2.13", AllowDowngrade: true,
		SignatureVersion: updates.SignatureVersionV2,
	}
	if agentUpdateOfferAllowed(v2Rollback, "0.2.14") {
		t.Fatal("pre-v2 Agent was offered a rollback whose flag it cannot authenticate")
	}
	v2MarkedNewer := &updates.Manifest{
		Version: "0.2.20", AllowDowngrade: true,
		SignatureVersion: updates.SignatureVersionV2,
	}
	if agentUpdateOfferAllowed(v2MarkedNewer, "0.2.14") {
		t.Fatal("pre-v2 Agent was offered a rollback-marked release through the upgrade path")
	}
	if !agentUpdateOfferAllowed(v2Rollback, "0.2.15") {
		t.Fatal("v2-capable Agent was not offered an authenticated rollback")
	}
	legacyRollback := &updates.Manifest{
		Version: "0.2.13", AllowDowngrade: true,
		SignatureVersion: updates.SignatureVersionLegacy,
	}
	if agentUpdateOfferAllowed(legacyRollback, "0.2.15") {
		t.Fatal("legacy artifact-only rollback was offered")
	}
	legacyRollback.Version = "0.2.20"
	if agentUpdateOfferAllowed(legacyRollback, "0.2.15") {
		t.Fatal("legacy rollback flag bypassed the gate through a numerically newer target")
	}
	if agentUpdateOfferAllowed(v2Rollback, "0.2.15-not-semver") {
		t.Fatal("malformed current_version bypassed the v2 capability gate")
	}
	if !agentUpdateOfferAllowed(v2Rollback, "v0.2.15") {
		t.Fatal("a canonical v-prefixed v2-capable Agent was rejected")
	}
	sameVersion := &updates.Manifest{
		Version: "0.2.15", AllowDowngrade: true,
		SignatureVersion: updates.SignatureVersionV2,
	}
	if agentUpdateOfferAllowed(sameVersion, "v0.2.15") {
		t.Fatal("equivalent version spellings bypassed the no-op update gate")
	}
}

func TestUnsafeRollbackDoesNotHideALegacyBridgeUpgrade(t *testing.T) {
	candidates := []updates.Manifest{
		{
			Version: "0.2.20", AllowDowngrade: true,
			SignatureVersion: updates.SignatureVersionV2,
		},
		{Version: "0.2.15", SignatureVersion: updates.SignatureVersionV2},
	}
	offered := selectAgentUpdateOffer(candidates, "0.2.14")
	if offered == nil || offered.Version != "0.2.15" {
		t.Fatalf("legacy bridge offer = %+v, want 0.2.15", offered)
	}
}

func TestPublishOptionsRejectsConflictingDirectAndFileSignatures(t *testing.T) {
	directBytes := bytes.Repeat([]byte{0x11}, ed25519.SignatureSize)
	fileBytes := bytes.Repeat([]byte{0x22}, ed25519.SignatureSize)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField(
		"signature", base64.StdEncoding.EncodeToString(directBytes),
	); err != nil {
		t.Fatal(err)
	}
	part, err := writer.CreateFormFile("signature_file", "artifact.sig")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(fileBytes); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	if err := request.ParseMultipartForm(1 << 20); err != nil {
		t.Fatal(err)
	}
	if _, err := publishOptions(request); err == nil ||
		!strings.Contains(err.Error(), "does not match") {
		t.Fatalf("publishOptions() error = %v, want a signature source mismatch", err)
	}
}

func TestPublishAgentUpdateRequiresAConfiguredVerificationKey(t *testing.T) {
	runtime := storagetest.Open(t)
	defer runtime.Close()
	updateStore, err := updates.Open(filepath.Join(t.TempDir(), "updates"))
	if err != nil {
		t.Fatal(err)
	}
	server := testServer(t, runtime)
	server.updateStore = updateStore
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)
	request := httptest.NewRequest(
		http.MethodPost, "/api/v1/admin/agent-updates", strings.NewReader(""),
	)
	request.Header.Set("X-CSRF-Token", csrf)
	request.AddCookie(cookie)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable ||
		!strings.Contains(response.Body.String(), "UPDATE_SIGNING_KEY_MISSING") {
		t.Fatalf("publish status = %d body = %s", response.Code, response.Body.String())
	}
}
