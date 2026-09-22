package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestPostgresRAGIndexRetryResumesCommittedBatches(t *testing.T) {
	s, c, wid, _ := ragTestSetup(t)
	var calls atomic.Int64
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n == 2 {
			w.WriteHeader(503)
			return
		}
		var in struct {
			Input []string `json:"input"`
		}
		if json.NewDecoder(r.Body).Decode(&in) != nil {
			w.WriteHeader(400)
			return
		}
		rows := []map[string]any{}
		for i := range in.Input {
			rows = append(rows, map[string]any{"index": i, "embedding": []float32{1, 2, 3}})
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"data": rows})
	}))
	defer provider.Close()
	ragTestConfigure(t, c, provider.URL)
	md := "# 배치 재시도\n\n" + strings.Repeat("GPU 운영 작업의 체크포인트를 보존합니다.\n", 3300)
	chunks, e := chunkMarkdown(md, 6144, 512)
	if e != nil || len(chunks) <= 16 {
		t.Fatal(len(chunks), e)
	}
	id := str(testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "배치 재시도", "markdown": md}, 200)), "id")
	consent := ragTestConsent(t, c, id, 1, false, false)
	jobID := str(consent, "job_id")
	ragTestRun(t, s, jobID)
	var indexed int
	if e = s.DB.QueryRow(t.Context(), `SELECT indexed_chunks FROM rag_vector_indexes WHERE id=$1`, jobID).Scan(&indexed); e != nil || indexed != 16 || ragTestStatus(t, s, jobID) != "pending" {
		t.Fatal(indexed, e, ragTestStatus(t, s, jobID))
	}
	ragTestRun(t, s, jobID)
	if ragTestStatus(t, s, jobID) != "succeeded" {
		t.Fatal("retry failed")
	}
	if want := int64((len(chunks)+15)/16 + 1); calls.Load() != want {
		t.Fatal("committed batch resent", calls.Load(), want)
	}
	if e = s.DB.QueryRow(t.Context(), `SELECT indexed_chunks FROM rag_vector_indexes WHERE id=$1`, jobID).Scan(&indexed); e != nil || indexed != len(chunks) {
		t.Fatal(indexed, e)
	}
}

func TestPostgresRAGIndexReadOnlyStatusAndScopeBoundary(t *testing.T) {
	s, c, wid, _ := ragTestSetup(t)
	provider, calls, _ := ragTestProvider(t, nil)
	ragTestConfigure(t, c, provider.URL)
	id := str(testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "공유 색인", "markdown": "# GPU 공유 원문"}, 200)), "id")
	consent := ragTestConsent(t, c, id, 1, false, false)
	ragTestRun(t, s, str(consent, "job_id"))
	info := testJSONObject(t, c.request("POST", "/api/v1/admin/users", map[string]any{"email": "rag-status@example.test", "name": "읽기 전용", "password": "RAG-Status-Password-2026!", "role": "viewer"}, 200))
	c.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": info["email"], "role": "viewer"}, 200)
	viewer := newIntegrationTestClient(t, c.base)
	viewer.request("POST", "/api/v1/auth/login", map[string]any{"email": info["email"], "password": "RAG-Status-Password-2026!"}, 200)
	state := testJSONObject(t, viewer.request("GET", "/api/v1/documents/"+id+"/rag-index", nil, 200))
	if boolean(state, "can_index") || state["provider"] != nil || state["grant"] != nil || state["index"].(map[string]any)["job_id"] != nil {
		t.Fatal("reader received consent administration", state)
	}
	before := calls.Load()
	viewer.request("POST", "/api/v1/documents/"+id+"/rag-index", map[string]any{"consent": true}, 403)
	viewer.request("DELETE", "/api/v1/documents/"+id+"/rag-index", map[string]any{"grant_revision": 1}, 403)
	key := testJSONObject(t, c.request("POST", "/api/v1/keys", map[string]any{"name": "문서 쓰기만", "workspace_id": wid, "scopes": []string{"document:read", "document:write"}}, 201))
	limited := newIntegrationTestClient(t, c.base)
	limited.token = str(key, "token")
	limited.request("POST", "/api/v1/documents/"+id+"/rag-index", map[string]any{"consent": true}, 403)
	limited.request("POST", "/api/v1/documents/"+id+"/rag-index/cancel", map[string]any{"job_id": consent["job_id"]}, 403)
	if calls.Load() != before {
		t.Fatal("unauthorized actor contacted provider")
	}
	// Stale confirmation cannot revoke a newly changed grant.
	c.request("DELETE", "/api/v1/documents/"+id+"/rag-index", map[string]any{"grant_revision": 99}, 409)
}
