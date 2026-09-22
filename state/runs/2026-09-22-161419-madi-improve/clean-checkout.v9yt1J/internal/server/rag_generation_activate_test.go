package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"sync"
	"testing"
)

func ragTestGenerationConsent(t *testing.T, s *Server, c *integrationTestClient, doc, generation string, version int) {
	t.Helper()
	path := "/api/v1/documents/" + doc + "/rag-index?generation_id=" + generation
	state := testJSONObject(t, c.request("GET", path, nil, 200))
	job := testJSONObject(t, c.request("POST", path, map[string]any{"expected_version": version, "provider_fingerprint": state["provider"].(map[string]any)["fingerprint"], "consent": true}, 202))
	ragTestRun(t, s, str(job, "job_id"))
	if ragTestStatus(t, s, str(job, "job_id")) != "succeeded" {
		t.Fatal("generation document index failed")
	}
}
func ragTestGenerationVerify(t *testing.T, c *integrationTestClient, generation, provider, doc, mode string, version int) map[string]any {
	t.Helper()
	return testJSONObject(t, c.request("POST", "/api/v1/rag-generations/"+generation+"/verify", map[string]any{"revision": 1, "provider_fingerprint": provider, "query": "GPU 운영", "mode": mode, "consent": true, "documents": []map[string]any{{"document_id": doc, "version": version}}}, 200))
}
func TestPostgresRAGGenerationBlueGreenCASRollback(t *testing.T) {
	s, c, wid, uid := ragTestSetup(t)
	provider, calls, _ := ragTestProvider(t, nil)
	ragTestConfigure(t, c, provider.URL)
	doc := str(testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "GPU 전환 근거", "markdown": "GPU 원문 보존", "visibility": "private"}, 200)), "id")
	legacy := ragTestConsent(t, c, doc, 1, false, false)
	ragTestRun(t, s, str(legacy, "job_id"))
	next := testJSONObject(t, c.request("POST", "/api/v1/workspaces/"+wid+"/rag-generations", map[string]any{"name": "새 모델", "dimensions": 3, "config": map[string]any{"rag_embedding_model": "blue-green-next"}, "confirm": true}, 201))
	generation := str(next, "id")
	list := testJSONObject(t, c.request("GET", "/api/v1/workspaces/"+wid+"/rag-generations", nil, 200))
	baseline := str(list, "active_id")
	oldConfig, e := s.effectiveSettings(t.Context(), wid)
	if e != nil {
		t.Fatal(e)
	}
	ragTestGenerationConsent(t, s, c, doc, generation, 1)
	verification := ragTestGenerationVerify(t, c, generation, str(next, "provider_fingerprint"), doc, "exact", 1)
	if !boolean(verification, "can_activate") {
		t.Fatal("exact cohort not ready", verification)
	}
	current := testJSONObject(t, c.request("GET", "/api/v1/workspaces/"+wid+"/rag-generations", nil, 200))
	if str(current, "active_id") != baseline {
		t.Fatal("verification activated automatically")
	}
	action := map[string]any{"validation_id": verification["validation_id"], "state_revision": verification["state_revision"], "mode": "exact", "confirm": true, "confirm_coverage": true}
	// Two tabs consume one receipt: exactly one transition and one conflict.
	statuses := make(chan int, 2)
	errors := make(chan error, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			body, _ := json.Marshal(action)
			r, _ := http.NewRequest("POST", c.base+"/api/v1/workspaces/"+wid+"/rag-generations/activate", bytes.NewReader(body))
			r.Header.Set("X-Madi-Request", "1")
			r.Header.Set("Content-Type", "application/json")
			response, e := c.client.Do(r)
			if e != nil {
				errors <- e
				return
			}
			defer response.Body.Close()
			io.Copy(io.Discard, response.Body)
			statuses <- response.StatusCode
		}()
	}
	wg.Wait()
	close(statuses)
	close(errors)
	for e := range errors {
		t.Fatal(e)
	}
	got := []int{}
	for code := range statuses {
		got = append(got, code)
	}
	sort.Ints(got)
	if len(got) != 2 || got[0] != 200 || got[1] != 409 {
		t.Fatal("non-atomic activation", got)
	}
	cfg, e := s.effectiveSettings(t.Context(), wid)
	if e != nil || ragProviderFingerprint(cfg) != str(next, "provider_fingerprint") {
		t.Fatal("settings and serving pointer diverged", e)
	}
	p, e := s.workerPrincipal(t.Context(), uid, "", wid)
	if e != nil {
		t.Fatal(e)
	}
	candidate, diag, e := s.ragVectorCandidates(t.Context(), p, wid, doc, cfg, []float32{1, 0, 0})
	if e != nil || len(candidate) != 1 || diag.GenerationID != generation {
		t.Fatal("new generation retrieval failed", e, diag)
	}
	// Rolling back is an explicit fresh validation of the previous provider and
	// its original (still current) per-document grants, not a stale receipt reuse.
	oldVerification := ragTestGenerationVerify(t, c, baseline, ragProviderFingerprint(oldConfig), doc, "exact", 1)
	c.request("POST", "/api/v1/workspaces/"+wid+"/rag-generations/activate", map[string]any{"validation_id": oldVerification["validation_id"], "state_revision": oldVerification["state_revision"], "mode": "exact", "confirm": true, "confirm_coverage": true}, 200)
	cfg, e = s.effectiveSettings(t.Context(), wid)
	if e != nil || ragProviderFingerprint(cfg) != ragProviderFingerprint(oldConfig) {
		t.Fatal("rollback provider not restored", e)
	}
	stale := ragTestGenerationVerify(t, c, generation, str(next, "provider_fingerprint"), doc, "exact", 1)
	c.request("PUT", "/api/v1/documents/"+doc, map[string]any{"version": 1, "markdown": "GPU 새 근거"}, 200)
	before := calls.Load()
	c.request("POST", "/api/v1/workspaces/"+wid+"/rag-generations/activate", map[string]any{"validation_id": stale["validation_id"], "state_revision": stale["state_revision"], "mode": "exact", "confirm": true, "confirm_coverage": true}, 409)
	if calls.Load() != before {
		t.Fatal("activation performed new external request")
	}
	var raw string
	s.DB.QueryRow(t.Context(), `SELECT cohort::text||report::text FROM rag_generation_validations WHERE id=$1`, str(stale, "validation_id")).Scan(&raw)
	if bytes.Contains([]byte(raw), []byte("GPU 원문")) {
		t.Fatal("source body persisted in receipt")
	}
}
