package server

import (
	"strings"
	"testing"
)

func TestPostgresRAGGenerationShadowConsentIsolation(t *testing.T) {
	s, c, wid, owner := ragTestSetup(t)
	provider, calls, _ := ragTestProvider(t, nil)
	ragTestConfigure(t, c, provider.URL)
	doc := testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "세대별 동의 문서", "markdown": "GPU 운영 지식", "visibility": "private"}, 200))
	id := str(doc, "id")
	oldJob := ragTestConsent(t, c, id, 1, false, false)
	ragTestRun(t, s, str(oldJob, "job_id"))
	before := calls.Load()
	created := testJSONObject(t, c.request("POST", "/api/v1/workspaces/"+wid+"/rag-generations", map[string]any{"name": "새 모델 준비", "dimensions": 3, "config": map[string]any{"rag_embedding_model": "next-generation"}, "confirm": true}, 201))
	generation := str(created, "id")
	if calls.Load() != before {
		t.Fatal("generation creation transmitted data without document consent")
	}
	list := testJSONObject(t, c.request("GET", "/api/v1/workspaces/"+wid+"/rag-generations", nil, 200))
	if len(list["generations"].([]any)) != 2 || str(list, "active_id") == generation {
		t.Fatal("shadow generation activated automatically", list)
	}
	active := testJSONObject(t, c.request("GET", "/api/v1/documents/"+id+"/rag-index", nil, 200))
	if str(active, "generation_id") != str(list, "active_id") {
		t.Fatal("legacy grant not safely adopted", active)
	}
	path := "/api/v1/documents/" + id + "/rag-index?generation_id=" + generation
	shadow := testJSONObject(t, c.request("GET", path, nil, 200))
	if shadow["grant"] != nil {
		t.Fatal("other generation reused consent", shadow)
	}
	fingerprint := shadow["provider"].(map[string]any)["fingerprint"]
	job := testJSONObject(t, c.request("POST", path, map[string]any{"expected_version": 1, "provider_fingerprint": fingerprint, "consent": true}, 202))
	ragTestRun(t, s, str(job, "job_id"))
	var indexes int
	if e := s.DB.QueryRow(t.Context(), `SELECT count(*) FROM rag_vector_indexes WHERE document_id=$1 AND status='ready'`, id).Scan(&indexes); e != nil || indexes != 2 {
		t.Fatal("shadow destroyed serving index", e, indexes)
	}
	cfg, e := s.effectiveSettings(t.Context(), wid)
	if e != nil {
		t.Fatal(e)
	}
	candidates, _, e := s.ragVectorCandidates(t.Context(), &Principal{ID: owner, Role: "admin", Kind: "user"}, wid, id, cfg, []float32{1, 0, 0})
	if e != nil || len(candidates) != 1 {
		t.Fatal("mixed shadow and active retrieval", e, len(candidates))
	}
	raw, _ := s.DB.Query(t.Context(), `SELECT config_cipher FROM rag_generations WHERE workspace_id=$1`, wid)
	for raw.Next() {
		var cipher string
		if e = raw.Scan(&cipher); e != nil || !strings.HasPrefix(cipher, "enc:") {
			t.Fatal("provider config plaintext", e)
		}
		plain, e := s.decrypt(cipher)
		if e != nil || strings.Contains(plain, "smtp_password") || strings.Contains(plain, "oidc_client_secret") {
			t.Fatal("unrelated secrets copied into generation", e)
		}
	}
	raw.Close()
	// Revoking one generation must not remove or disable the active one.
	state := testJSONObject(t, c.request("GET", path, nil, 200))
	grant := state["grant"].(map[string]any)
	c.request("DELETE", path, map[string]any{"grant_revision": grant["revision"]}, 200)
	if e = s.DB.QueryRow(t.Context(), `SELECT count(*) FROM rag_vector_indexes WHERE document_id=$1 AND status='ready'`, id).Scan(&indexes); e != nil || indexes != 1 {
		t.Fatal("generation revocation affected another generation", e, indexes)
	}
	c.request("POST", "/api/v1/workspaces/"+wid+"/rag-generations", map[string]any{"name": "잘못된 공급자", "dimensions": 3, "config": map[string]any{"rag_embedding_base_url": []any{"invalid"}}, "confirm": true}, 400)
}
