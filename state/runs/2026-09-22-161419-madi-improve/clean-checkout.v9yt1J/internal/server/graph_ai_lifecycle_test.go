package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestPostgresGraphAIPreviewAndImmutableReview(t *testing.T) {
	s, c, wid, _ := graphAITestSetup(t)
	doc := testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "운영 기준", "markdown": "PostgreSQL 운영 기준"}, 200))
	did := str(doc, "id")
	var calls atomic.Int32
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		candidate := graphAITestCandidates(did, newID())[2]
		agentTestSSE(w, string(jsonValue(map[string]any{"candidates": []graphAICandidate{candidate}})), "", "")
	}))
	defer provider.Close()
	c.request("PUT", "/api/v1/admin/settings", map[string]any{"ai_enabled": true, "ai_base_url": provider.URL + "?api-key=MUST_NOT_EXPOSE", "ai_model": "graph-preview", "ai_max_tokens": 4096}, 200)
	previewRaw := c.request("GET", "/api/v1/workspaces/"+wid+"/graph-ai/context?document_ids="+did, nil, 200)
	if strings.Contains(string(previewRaw), "MUST_NOT_EXPOSE") {
		t.Fatal("endpoint query credential disclosed")
	}
	preview := testJSONObject(t, previewRaw)
	input := map[string]any{"request_id": newID(), "snapshots": preview["snapshots"], "provider_fingerprint": preview["provider"].(map[string]any)["fingerprint"], "kinds": []string{"topic"}, "consent": true}
	c.request("PUT", "/api/v1/admin/settings", map[string]any{"ai_model": "graph-preview-changed"}, 200)
	c.request("POST", "/api/v1/workspaces/"+wid+"/graph-ai/analyze", input, 409)
	if calls.Load() != 0 {
		t.Fatal("stale preview sent to changed provider")
	}
	preview = testJSONObject(t, c.request("GET", "/api/v1/workspaces/"+wid+"/graph-ai/context?document_ids="+did, nil, 200))
	input["provider_fingerprint"] = preview["provider"].(map[string]any)["fingerprint"]
	raw := c.request("POST", "/api/v1/workspaces/"+wid+"/graph-ai/analyze", input, 200)
	if !strings.Contains(string(raw), `"proposal":`) {
		t.Fatal(string(raw))
	}
	c.request("POST", "/api/v1/workspaces/"+wid+"/graph-ai/analyze", input, 409)
	if calls.Load() != 1 {
		t.Fatal("same request was automatically replayed")
	}
	runID := str(input, "request_id")
	view := testJSONObject(t, c.request("GET", "/api/v1/graph-ai/runs/"+runID, nil, 200))
	action := view["actions"].([]any)[0].(map[string]any)
	path := "/api/v1/graph-ai/actions/" + str(action, "id") + "/confirm"
	other := newIntegrationTestClient(t, c.base)
	other.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200)
	other.request("POST", path, map[string]any{"confirm": true, "action_hash": action["action_hash"]}, 404)
	c.request("POST", path, map[string]any{"confirm": true, "action_hash": digest("wrong")}, 404)
	c.request("PUT", "/api/v1/documents/"+did, map[string]any{"version": 1, "markdown": "PostgreSQL 운영 기준이 수정되었습니다."}, 200)
	c.request("POST", path, map[string]any{"confirm": true, "action_hash": action["action_hash"]}, 409)
	var status string
	if e := s.DB.QueryRow(t.Context(), `SELECT status FROM graph_ai_actions WHERE id=$1`, str(action, "id")).Scan(&status); e != nil || status != "proposed" {
		t.Fatalf("stale candidate mutated %s %v", status, e)
	}
}

func TestPostgresGraphAICandidateProtectionAndOrphanLease(t *testing.T) {
	for _, mode := range []string{"mask", "block"} {
		t.Run(mode, func(t *testing.T) {
			s, c, wid, _ := graphAITestSetup(t)
			doc := testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "운영 기준", "markdown": "PostgreSQL 운영 기준"}, 200))
			did := str(doc, "id")
			provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				candidate := graphAITestCandidates(did, newID())[2]
				candidate.Reason = "담당자는 sensitive-person@example.test 입니다."
				agentTestSSE(w, string(jsonValue(map[string]any{"candidates": []graphAICandidate{candidate}})), "", "")
			}))
			defer provider.Close()
			c.request("PUT", "/api/v1/admin/settings", map[string]any{"ai_enabled": true, "ai_base_url": provider.URL, "ai_model": "graph-policy", "ai_max_tokens": 4096}, 200)
			if _, e := s.DB.Exec(t.Context(), `UPDATE protection_settings SET data=data||jsonb_build_object('enabled',true,'mode',$1::text,'detectors',jsonb_build_array('email'),'custom_terms','[]'::jsonb) WHERE id=1`, mode); e != nil {
				t.Fatal(e)
			}
			preview := testJSONObject(t, c.request("GET", "/api/v1/workspaces/"+wid+"/graph-ai/context?document_ids="+did, nil, 200))
			id := newID()
			stream := c.request("POST", "/api/v1/workspaces/"+wid+"/graph-ai/analyze", map[string]any{"request_id": id, "snapshots": preview["snapshots"], "provider_fingerprint": preview["provider"].(map[string]any)["fingerprint"], "kinds": []string{"topic"}, "consent": true}, 200)
			var encoded []byte
			var count int
			if e := s.DB.QueryRow(t.Context(), `SELECT count(*),coalesce(jsonb_agg(payload),'[]'::jsonb) FROM graph_ai_actions WHERE run_id=$1`, id).Scan(&count, &encoded); e != nil {
				t.Fatal(e)
			}
			if strings.Contains(string(encoded), "sensitive-person@example.test") {
				t.Fatal("PII persisted in candidate")
			}
			if mode == "block" {
				if count != 0 || !strings.Contains(string(stream), `"retract":true`) {
					t.Fatalf("block candidate persisted %d %s", count, stream)
				}
			} else if count != 1 || !strings.Contains(string(stream), `"proposal":`) {
				t.Fatalf("masked proposal absent %d %s", count, stream)
			}
			// Simulate a killed process's durable foreground row without making any
			// new network request. Maintenance closes it; no automatic job exists.
			if _, e := s.DB.Exec(t.Context(), `UPDATE graph_ai_runs SET status='running',heartbeat_at=now()-interval '6 seconds',finished_at=NULL WHERE id=$1`, id); e != nil {
				t.Fatal(e)
			}
			if e := s.expireGraphAI(t.Context()); e != nil {
				t.Fatal(e)
			}
			var status, reason string
			if e := s.DB.QueryRow(t.Context(), `SELECT status,error FROM graph_ai_runs WHERE id=$1`, id).Scan(&status, &reason); e != nil || status != "cancelled" || strings.Contains(reason, "sensitive-person") {
				t.Fatalf("unsafe orphan result %s %s %v", status, reason, e)
			}
			if !json.Valid(encoded) {
				t.Fatal("invalid durable payload")
			}
		})
	}
}
