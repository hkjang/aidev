package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func ragTestSetup(t *testing.T) (*Server, *integrationTestClient, string, string) {
	t.Helper()
	s, ts := integrationTestServer(t)
	if e := s.migrateRAGIndex(t.Context()); e != nil {
		t.Fatal(e)
	}
	if !slices.Contains(s.apiRoutes, "GET /api/v1/documents/{id}/rag-index") {
		s.registerRAGIndex()
	}
	c := newIntegrationTestClient(t, ts.URL)
	me := testJSONObject(t, c.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200))
	wid := str(testJSONObject(t, c.request("POST", "/api/v1/workspaces", map[string]any{"name": "동의 기반 검색 AI"}, 200)), "id")
	return s, c, wid, str(me, "id")
}

func ragTestProvider(t *testing.T, hook func()) (*httptest.Server, *atomic.Int64, *atomic.Int64) {
	t.Helper()
	embeddingCalls, rerankCalls := &atomic.Int64{}, &atomic.Int64{}
	p := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/rerank") {
			rerankCalls.Add(1)
			var in struct {
				Documents []string `json:"documents"`
				Top       int      `json:"top_n"`
			}
			if json.NewDecoder(r.Body).Decode(&in) != nil || in.Top != len(in.Documents) {
				http.Error(w, "bad request", 400)
				return
			}
			results := []map[string]any{}
			for i := range in.Documents {
				results = append(results, map[string]any{"index": len(in.Documents) - i - 1, "relevance_score": 1 - float64(i)/float64(len(in.Documents)+1)})
			}
			json.NewEncoder(w).Encode(map[string]any{"results": results})
			return
		}
		embeddingCalls.Add(1)
		var in struct {
			Input []string `json:"input"`
			Model string   `json:"model"`
		}
		if json.NewDecoder(r.Body).Decode(&in) != nil {
			http.Error(w, "bad request", 400)
			return
		}
		if hook != nil {
			hook()
		}
		if in.Model == "slow-browser-model" {
			timer := time.NewTimer(5 * time.Second)
			defer timer.Stop()
			select {
			case <-timer.C:
			case <-r.Context().Done():
				return
			}
		}
		data := []map[string]any{}
		for i, text := range in.Input {
			vec := []float32{0, 1, 0}
			if strings.Contains(text, "GPU") {
				vec = []float32{1, 0, 0}
			}
			data = append(data, map[string]any{"index": i, "embedding": vec})
		}
		json.NewEncoder(w).Encode(map[string]any{"data": data})
	}))
	t.Cleanup(p.Close)
	return p, embeddingCalls, rerankCalls
}

func ragTestConfigure(t *testing.T, c *integrationTestClient, url string) {
	t.Helper()
	c.request("PUT", "/api/v1/admin/settings", map[string]any{"rag_enabled": true, "rag_embedding_base_url": url + "/v1?secret=hidden-query", "rag_embedding_model": "internal", "rag_allow_http": true, "rag_embedding_dimensions": 3, "rag_search_mode": "hybrid", "rag_rerank_enabled": true, "rag_rerank_base_url": url + "/v1", "rag_rerank_model": "local-rerank"}, 200)
}
func ragTestConsent(t *testing.T, c *integrationTestClient, id string, version int, auto, rerank bool) map[string]any {
	t.Helper()
	state := testJSONObject(t, c.request("GET", "/api/v1/documents/"+id+"/rag-index", nil, 200))
	provider := state["provider"].(map[string]any)
	if strings.Contains(str(provider, "base_url"), "secret=") {
		t.Fatal("provider URL query leaked", provider)
	}
	in := map[string]any{"expected_version": version, "provider_fingerprint": provider["fingerprint"], "consent": true, "auto_reindex": auto, "allow_rerank": rerank}
	if rerank {
		in["rerank_fingerprint"] = provider["rerank"].(map[string]any)["fingerprint"]
	}
	return testJSONObject(t, c.request("POST", "/api/v1/documents/"+id+"/rag-index", in, 202))
}
func ragTestRun(t *testing.T, s *Server, id string) Job {
	t.Helper()
	ctx := t.Context()
	lease := newID()
	_, e := s.DB.Exec(ctx, `UPDATE automation_jobs SET status='running',lease_id=$2,lease_until=now()+interval '1 hour',attempts=attempts+1 WHERE id=$1`, id, lease)
	if e != nil {
		t.Fatal(e)
	}
	j, e := scanJob(s.DB.QueryRow(ctx, "SELECT "+jobSelect+" FROM automation_jobs WHERE id=$1", id))
	if e != nil {
		t.Fatal(e)
	}
	s.runJob(ctx, j)
	return j
}
func ragTestStatus(t *testing.T, s *Server, id string) string {
	t.Helper()
	var status string
	if e := s.DB.QueryRow(t.Context(), `SELECT status FROM automation_jobs WHERE id=$1`, id).Scan(&status); e != nil {
		t.Fatal(e)
	}
	return status
}

func TestPostgresRAGIndexConsentHybridCitationsAndRevoke(t *testing.T) {
	s, c, wid, uid := ragTestSetup(t)
	provider, calls, reranks := ragTestProvider(t, nil)
	ragTestConfigure(t, c, provider.URL)
	markdown := "# 운영 자료\n\n" + strings.Repeat("일반 기록을 자세하게 설명합니다.\n", 300) + "\n## GPU 메모리\n\nGPU OOM 해결 절차는 정확히 이 곳에 있습니다. 🚀\n"
	id := str(testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "운영 자료", "markdown": markdown, "visibility": "private"}, 200)), "id")
	consent := ragTestConsent(t, c, id, 1, false, true)
	jobID := str(consent, "job_id")
	ragTestRun(t, s, jobID)
	if status := ragTestStatus(t, s, jobID); status != "succeeded" {
		var err string
		s.DB.QueryRow(t.Context(), "SELECT last_error FROM automation_jobs WHERE id=$1", jobID).Scan(&err)
		t.Fatal(status, err)
	}
	p, e := s.workerPrincipal(t.Context(), uid, "", wid)
	if e != nil {
		t.Fatal(e)
	}
	r := httptest.NewRequest("GET", "/", nil).WithContext(context.WithValue(t.Context(), principalKey, p))
	sources, diagnostic, e := s.retrieveRAGAISources(r, p, id, wid, "GPU OOM")
	if e != nil || len(sources) == 0 || diagnostic.Backend != "array" || !diagnostic.Reranked || reranks.Load() != 1 {
		t.Fatal(e, diagnostic, sources)
	}
	late := false
	for _, source := range sources {
		if source.Markdown != markdown[source.StartByte:source.EndByte] || source.ContentHash != digest(source.Markdown) {
			t.Fatal("noncanonical source", source)
		}
		if strings.Contains(source.Markdown, "GPU OOM") {
			late = true
		}
		c.request("GET", "/api/v1"+source.CitationURL, nil, 200)
	}
	if !late {
		t.Fatal("late document chunk lost")
	}
	viewerInfo := testJSONObject(t, c.request("POST", "/api/v1/admin/users", map[string]any{"email": "rag-viewer@example.test", "name": "검색 사용자", "password": "RAG-Viewer-Password-2026!", "role": "viewer"}, 200))
	c.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": viewerInfo["email"], "role": "viewer"}, 200)
	viewer := newIntegrationTestClient(t, c.base)
	viewer.request("POST", "/api/v1/auth/login", map[string]any{"email": viewerInfo["email"], "password": "RAG-Viewer-Password-2026!"}, 200)
	viewer.request("GET", "/api/v1/documents/"+id+"/rag-index", nil, 404)
	vp, e := s.workerPrincipal(t.Context(), str(viewerInfo, "id"), "", wid)
	if e != nil {
		t.Fatal(e)
	}
	before := calls.Load()
	hidden, _, e := s.retrieveRAGAISources(r.WithContext(context.WithValue(t.Context(), principalKey, vp)), vp, "", wid, "GPU OOM")
	if e != nil || len(hidden) != 0 || calls.Load() != before {
		t.Fatal("private document leaked or triggered external query", e, hidden)
	}
	c.request("DELETE", "/api/v1/documents/"+id+"/rag-index", map[string]any{"grant_revision": consent["grant_revision"]}, 200)
	var count int
	s.DB.QueryRow(t.Context(), "SELECT count(*) FROM rag_vector_chunks").Scan(&count)
	if count != 0 {
		t.Fatal("revoked vectors retained", count)
	}
	before = calls.Load()
	_, diag, e := s.retrieveRAGAISources(r, p, id, wid, "GPU OOM")
	if e != nil || len(diag.Warnings) == 0 || calls.Load() != before {
		t.Fatal("revoked index still used", diag, e)
	}
}

func TestPostgresRAGIndexRechecksRevocationBeforePersist(t *testing.T) {
	for _, change := range []string{"document", "provider", "token", "grant"} {
		t.Run(change, func(t *testing.T) {
			s, c, wid, uid := ragTestSetup(t)
			entered, release := make(chan struct{}), make(chan struct{})
			var once sync.Once
			provider, _, _ := ragTestProvider(t, func() { once.Do(func() { close(entered); <-release }) })
			ragTestConfigure(t, c, provider.URL)
			id := str(testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "권한 회수", "markdown": "# GPU 비공개 운영 절차", "visibility": "private"}, 200)), "id")
			tokenID := ""
			issuer := c
			if change == "token" {
				key := testJSONObject(t, c.request("POST", "/api/v1/keys", map[string]any{"name": "색인 전용", "workspace_id": wid, "scopes": []string{"document:read", "document:write", "ai:execute"}}, 201))
				issuer = newIntegrationTestClient(t, c.base)
				issuer.token = str(key, "token")
				tokenID = str(key["key"].(map[string]any), "id")
				if tokenID == "" {
					t.Fatal(key)
				}
			}
			consent := ragTestConsent(t, issuer, id, 1, false, false)
			jobID := str(consent, "job_id")
			j, e := scanJob(s.DB.QueryRow(t.Context(), "SELECT "+jobSelect+" FROM automation_jobs WHERE id=$1", jobID))
			if e != nil {
				t.Fatal(e)
			}
			finished := make(chan error, 1)
			go func() { _, e := s.runRAGIndex(t.Context(), j); finished <- e }()
			<-entered
			switch change {
			case "document":
				c.request("PUT", "/api/v1/documents/"+id, map[string]any{"version": 1, "markdown": "현재 문서"}, 200)
			case "provider":
				c.request("PUT", "/api/v1/admin/settings", map[string]any{"rag_embedding_model": "different-model"}, 200)
			case "token":
				if _, e = s.DB.Exec(t.Context(), `UPDATE api_keys SET revoked_at=now() WHERE id=$1 AND user_id=$2`, tokenID, uid); e != nil {
					t.Fatal(e)
				}
			case "grant":
				c.request("DELETE", "/api/v1/documents/"+id+"/rag-index", map[string]any{"grant_revision": consent["grant_revision"]}, 200)
			}
			close(release)
			if e = <-finished; e == nil {
				t.Fatal("changed grant was persisted")
			}
			var count int
			s.DB.QueryRow(t.Context(), "SELECT count(*) FROM rag_vector_chunks").Scan(&count)
			if count != 0 {
				t.Fatal("stale vectors persisted", count)
			}
		})
	}
}

func TestPostgresRAGAutoReindexSameProviderOnly(t *testing.T) {
	s, c, wid, _ := ragTestSetup(t)
	provider, calls, _ := ragTestProvider(t, nil)
	ragTestConfigure(t, c, provider.URL)
	id := str(testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "자동 색인", "markdown": "# GPU 문서", "visibility": "private"}, 200)), "id")
	consent := ragTestConsent(t, c, id, 1, true, false)
	ragTestRun(t, s, str(consent, "job_id"))
	c.request("PUT", "/api/v1/documents/"+id, map[string]any{"version": 1, "markdown": "# GPU 문서\n\n새로운 내용"}, 200)
	s.DB.Exec(t.Context(), `UPDATE rag_reindex_queue SET enqueued_at=now()-interval '10 seconds'`)
	if e := s.dispatchRAGReindex(t.Context()); e != nil {
		t.Fatal(e)
	}
	g, e := ragGrantTx(t.Context(), s.DB, id, false)
	if e != nil || g.JobID == str(consent, "job_id") || g.Version != 2 {
		t.Fatal("auto reindex not scheduled", g, e)
	}
	ragTestRun(t, s, g.JobID)
	if ragTestStatus(t, s, g.JobID) != "succeeded" {
		t.Fatal("auto failed")
	}
	c.request("PUT", "/api/v1/admin/settings", map[string]any{"rag_embedding_model": "new-provider-model"}, 200)
	c.request("PUT", "/api/v1/documents/"+id, map[string]any{"version": 2, "markdown": "# GPU 문서\n\n변경된 원문"}, 200)
	s.DB.Exec(t.Context(), `UPDATE rag_reindex_queue SET enqueued_at=now()-interval '10 seconds'`)
	before := calls.Load()
	if e = s.dispatchRAGReindex(t.Context()); e != nil {
		t.Fatal(e)
	}
	next, e := ragGrantTx(t.Context(), s.DB, id, false)
	if e != nil || next.Auto || next.JobID != g.JobID || calls.Load() != before {
		t.Fatal("new provider silently consented", next, e)
	}
	state := testJSONObject(t, c.request("GET", "/api/v1/documents/"+id+"/rag-index", nil, 200))
	if !boolean(state["grant"].(map[string]any), "requires_reconsent") {
		t.Fatal(state)
	}
	// Deletion disables old grants, even if a document is later restored.
	c.request("DELETE", "/api/v1/documents/"+id, nil, 200)
	next, e = ragGrantTx(t.Context(), s.DB, id, false)
	if e != nil || next.Active || next.Auto {
		t.Fatal(next, e)
	}
}
