package server

import (
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestRAGSettingsTypesAndFingerprint(t *testing.T) {
	cfg := defaultSettings()
	if e := validateSettings(cfg); e != nil {
		t.Fatal(e)
	}
	for key, value := range map[string]any{"rag_embedding_dimensions": 1.5, "rag_scan_limit": 50001, "rag_top_k": "8", "rag_enabled": "true", "rag_backend": "shell", "rag_search_mode": "unknown", "rag_ca_pem": "bad certificate"} {
		next := defaultSettings()
		next[key] = value
		if validateSettings(next) == nil {
			t.Fatal("invalid setting accepted", key)
		}
	}
	cfg["rag_enabled"] = true
	cfg["rag_embedding_base_url"] = "http://model.test/v1"
	cfg["rag_embedding_model"] = "internal-embedding"
	if validateSettings(cfg) == nil {
		t.Fatal("HTTP enabled without explicit consent")
	}
	cfg["rag_allow_http"] = true
	if e := validateSettings(cfg); e != nil {
		t.Fatal(e)
	}
	before := ragProviderFingerprint(cfg)
	cfg["rag_embedding_api_key"] = "rotated"
	if before == ragProviderFingerprint(cfg) {
		t.Fatal("key rotation reused provider grant")
	}
}
func TestPostgresRAGSettingsEncryptionInheritanceAndDiagnostic(t *testing.T) {
	s, ts := integrationTestServer(t)
	ctx := t.Context()
	c := newIntegrationTestClient(t, ts.URL)
	c.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200)
	var gotAuth atomic.Value
	provider := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth.Store(r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":[{"index":0,"embedding":[1,2,3]}]}`))
	}))
	defer provider.Close()
	cert, e := x509.ParseCertificate(provider.TLS.Certificates[0].Certificate[0])
	if e != nil {
		t.Fatal(e)
	}
	ca := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}))
	cfg := map[string]any{"rag_enabled": true, "rag_embedding_base_url": provider.URL + "/v1", "rag_embedding_model": "internal", "rag_embedding_api_key": "GLOBAL_EMBEDDING_SECRET", "rag_ca_pem": ca}
	raw := c.request("PUT", "/api/v1/admin/settings", cfg, 200)
	if strings.Contains(string(raw), "GLOBAL_EMBEDDING_SECRET") {
		t.Fatal("global secret disclosed")
	}
	wid := str(testJSONObject(t, c.request("POST", "/api/v1/workspaces", map[string]any{"name": "검색 AI 설정"}, 200)), "id")
	wsPath := "/api/v1/workspaces/" + wid + "/settings"
	v := testJSONObject(t, c.request("PUT", wsPath, map[string]any{"version": 0, "data": map[string]any{"rag_embedding_api_key": "WORKSPACE_EMBEDDING_SECRET", "rag_rerank_api_key": "WORKSPACE_RERANK_SECRET"}}, 200))
	if strings.Contains(string(jsonValue(v)), "WORKSPACE_") {
		t.Fatal("workspace secret disclosed", v)
	}
	for _, key := range []string{"rag_embedding_api_key", "rag_rerank_api_key"} {
		var encrypted string
		if e = s.DB.QueryRow(ctx, "SELECT data->>$2 FROM workspace_settings WHERE workspace_id=$1", wid, key).Scan(&encrypted); e != nil || !strings.HasPrefix(encrypted, "enc:") {
			t.Fatal("unencrypted", key, e)
		}
	}
	effective, e := s.effectiveSettings(ctx, wid)
	if e != nil || str(effective, "rag_embedding_api_key") != "WORKSPACE_EMBEDDING_SECRET" {
		t.Fatal("override decryption", e)
	}
	test := testJSONObject(t, c.request("POST", "/api/v1/workspaces/"+wid+"/search-ai/test", map[string]any{}, 200))
	if number(test, "dimensions", 0) != 3 || gotAuth.Load() != "Bearer WORKSPACE_EMBEDDING_SECRET" {
		t.Fatal(test, gotAuth.Load())
	}
	inherited := testJSONObject(t, c.request("PUT", wsPath, map[string]any{"version": v["version"], "data": map[string]any{"rag_embedding_api_key": nil}}, 200))
	effective, e = s.effectiveSettings(ctx, wid)
	if e != nil || str(effective, "rag_embedding_api_key") != "GLOBAL_EMBEDDING_SECRET" {
		t.Fatal("inheritance", e)
	}
	c.request("POST", "/api/v1/admin/search-ai/test", map[string]any{}, 200)
	if gotAuth.Load() != "Bearer GLOBAL_EMBEDDING_SECRET" {
		t.Fatal(gotAuth.Load())
	}
	public := testJSONObject(t, c.request("GET", "/api/v1/workspaces/"+wid+"/search-ai", nil, 200))
	if strings.Contains(string(jsonValue(public)), "GLOBAL_EMBEDDING_SECRET") || number(public, "version", 0) != number(inherited, "version", -1) {
		t.Fatal(public)
	}
	// Choosing a different workspace endpoint never forwards the global secret.
	c.request("PUT", "/api/v1/admin/settings", map[string]any{"ai_base_url": provider.URL, "ai_api_key": "GLOBAL_CHAT_SECRET"}, 200)
	c.request("PUT", wsPath, map[string]any{"version": inherited["version"], "data": map[string]any{"rag_embedding_base_url": "https://different-provider.example/v1", "ai_base_url": "https://another-chat.example/v1"}}, 200)
	effective, e = s.effectiveSettings(ctx, wid)
	if e != nil || str(effective, "rag_embedding_api_key") != "" || str(effective, "ai_api_key") != "" {
		t.Fatal("global secret forwarded to a workspace endpoint", e)
	}
	// Cookie identity guard rejects queued preference patches from a former user.
	c.request("PUT", "/api/v1/profile", map[string]any{"expected_user_id": newID(), "preferences": map[string]any{"sidebar_width": 999}}, 409)
}
