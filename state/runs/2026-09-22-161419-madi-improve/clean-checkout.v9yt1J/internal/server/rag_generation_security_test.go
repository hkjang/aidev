package server

import (
	"sync/atomic"
	"testing"
)

func TestPostgresRAGGenerationVerificationRevocation(t *testing.T) {
	for _, change := range []string{"grant", "provider", "session"} {
		t.Run(change, func(t *testing.T) {
			s, c, wid, _ := ragTestSetup(t)
			var armed atomic.Bool
			var generation, doc string
			provider, calls, _ := ragTestProvider(t, func() {
				if !armed.Swap(false) {
					return
				}
				var e error
				switch change {
				case "grant":
					_, e = s.DB.Exec(t.Context(), `UPDATE rag_index_grants SET active=false,revision=revision+1 WHERE generation_id=$1 AND document_id=$2`, generation, doc)
				case "provider":
					_, e = s.DB.Exec(t.Context(), `UPDATE settings SET data=jsonb_set(data,'{rag_embedding_model}','"changed-during-query"'::jsonb) WHERE id=1`)
				case "session":
					_, e = s.DB.Exec(t.Context(), `UPDATE sessions SET expires_at=now()-interval '1 second'`)
				}
				if e != nil {
					t.Error(e)
				}
			})
			ragTestConfigure(t, c, provider.URL)
			next := testJSONObject(t, c.request("POST", "/api/v1/workspaces/"+wid+"/rag-generations", map[string]any{"name": "권한 경계 세대", "dimensions": 3, "config": map[string]any{"rag_embedding_model": "security-generation"}, "confirm": true}, 201))
			generation = str(next, "id")
			doc = str(testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "GPU 검증", "markdown": "GPU 보안 근거", "visibility": "private"}, 200)), "id")
			ragTestGenerationConsent(t, s, c, doc, generation, 1)
			before := calls.Load()
			armed.Store(true)
			c.request("POST", "/api/v1/rag-generations/"+generation+"/verify", map[string]any{"revision": 1, "provider_fingerprint": next["provider_fingerprint"], "query": "GPU 검증 질문", "mode": "exact", "consent": true, "documents": []map[string]any{{"document_id": doc, "version": 1}}}, 409)
			if calls.Load() != before+1 {
				t.Fatal("unexpected provider retry", calls.Load()-before)
			}
			var receipts int
			if e := s.DB.QueryRow(t.Context(), `SELECT count(*) FROM rag_generation_validations`).Scan(&receipts); e != nil || receipts != 0 {
				t.Fatal("revoked verification persisted a receipt", e, receipts)
			}
		})
	}
}

func TestPostgresRAGGenerationPrivateCohortNoAdminBypass(t *testing.T) {
	s, c, wid, _ := ragTestSetup(t)
	provider, calls, _ := ragTestProvider(t, nil)
	ragTestConfigure(t, c, provider.URL)
	g := testJSONObject(t, c.request("POST", "/api/v1/workspaces/"+wid+"/rag-generations", map[string]any{"name": "개인 영역 보존", "dimensions": 3, "config": map[string]any{"rag_embedding_model": "private-generation"}, "confirm": true}, 201))
	doc := str(testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "개인 자료", "markdown": "GPU 개인 자료", "visibility": "private"}, 200)), "id")
	ragTestGenerationConsent(t, s, c, doc, str(g, "id"), 1)
	u := testJSONObject(t, c.request("POST", "/api/v1/admin/users", map[string]any{"email": "generation-other@example.test", "name": "다른 관리 사용자", "password": "Generation-Other-Password-2026!", "role": "admin"}, 200))
	c.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": u["email"], "role": "admin"}, 200)
	other := newIntegrationTestClient(t, c.base)
	other.request("POST", "/api/v1/auth/login", map[string]any{"email": u["email"], "password": "Generation-Other-Password-2026!"}, 200)
	before := calls.Load()
	other.request("POST", "/api/v1/rag-generations/"+str(g, "id")+"/verify", map[string]any{"revision": 1, "provider_fingerprint": g["provider_fingerprint"], "query": "비공개 후보를 알아내려는 질문", "mode": "exact", "consent": true, "documents": []map[string]any{{"document_id": doc, "version": 1}}}, 409)
	if calls.Load() != before {
		t.Fatal("private cohort triggered outbound request")
	}
	list := testJSONObject(t, other.request("GET", "/api/v1/workspaces/"+wid+"/rag-generations", nil, 200))
	for _, entry := range list["generations"].([]any) {
		counts := entry.(map[string]any)["index"].(map[string]any)
		if number(counts, "consented_documents", -1) != 0 {
			t.Fatal("private cohort count leaked", counts)
		}
	}
}
