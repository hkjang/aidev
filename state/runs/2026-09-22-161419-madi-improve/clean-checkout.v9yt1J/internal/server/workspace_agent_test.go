package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func agentTestSetup(t *testing.T) (*Server, *integrationTestClient, string, string) {
	t.Helper()
	s, ts := integrationTestServer(t)
	if e := s.migrateWorkspaceAgent(t.Context()); e != nil {
		t.Fatal(e)
	}
	if !slices.Contains(s.apiRoutes, "GET /api/v1/workspaces/{id}/agents") {
		s.registerWorkspaceAgent()
	}
	c := newIntegrationTestClient(t, ts.URL)
	me := testJSONObject(t, c.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200))
	wid := str(testJSONObject(t, c.request("POST", "/api/v1/workspaces", map[string]any{"name": "실제 도구 Agent"}, 200)), "id")
	return s, c, wid, str(me, "id")
}
func agentTestSSE(w http.ResponseWriter, text, tool, args string) {
	w.Header().Set("Content-Type", "text/event-stream")
	send := func(delta any, finish any) {
		raw := jsonValue(map[string]any{"choices": []any{map[string]any{"index": 0, "delta": delta, "finish_reason": finish}}})
		fmt.Fprintf(w, "data: %s\n\n", raw)
		w.(http.Flusher).Flush()
	}
	if tool != "" {
		half := len(args) / 2
		for half > 0 && half < len(args) && args[half]&0xc0 == 0x80 {
			half--
		}
		send(map[string]any{"tool_calls": []any{map[string]any{"index": 0, "id": "call_" + tool, "type": "function", "function": map[string]any{"name": tool, "arguments": args[:half]}}}}, nil)
		send(map[string]any{"tool_calls": []any{map[string]any{"index": 0, "function": map[string]any{"arguments": args[half:]}}}}, nil)
		send(map[string]any{}, "tool_calls")
	} else {
		send(map[string]any{"content": text}, nil)
		send(map[string]any{}, "stop")
	}
	fmt.Fprint(w, "data: [DONE]\n\n")
	w.(http.Flusher).Flush()
}
func agentTestPump(t *testing.T, s *Server, id, want string) {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		var status, message string
		if e := s.DB.QueryRow(t.Context(), `SELECT status,error FROM agent_runs WHERE id=$1`, id).Scan(&status, &message); e != nil {
			t.Fatal(e)
		}
		if status == want {
			return
		}
		if status == "failed" && want != "failed" {
			t.Fatalf("agent failed: %s", message)
		}
		job, e := s.claimJob(t.Context())
		if e == nil {
			s.runJob(t.Context(), job)
		} else {
			time.Sleep(20 * time.Millisecond)
		}
	}
	t.Fatalf("agent did not reach %s", want)
}
func agentTestConfig(t *testing.T, c *integrationTestClient, wid, docID string, tools []string) map[string]any {
	t.Helper()
	return testJSONObject(t, c.request("POST", "/api/v1/workspaces/"+wid+"/agents", map[string]any{"name": "지식 운영 Agent", "enabled": true, "instructions": "문서를 확인하고 답변하세요", "document_ids": []string{docID}, "space_ids": []string{}, "database_ids": []string{}, "tools": tools, "max_steps": 8, "max_tokens": 4096}, 200))
}
func TestPostgresWorkspaceAgentRealToolLoopAndHumanConfirmation(t *testing.T) {
	s, c, wid, _ := agentTestSetup(t)
	doc := testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "Agent 원문", "markdown": "원문 그대로\n", "visibility": "private"}, 200))
	did := str(doc, "id")
	secret := testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "범위 밖", "markdown": "NEVER_SEND_OUTSIDE_AGENT_SCOPE"}, 200))
	_ = secret
	var calls atomic.Int64
	var leaked atomic.Bool
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Messages  []agentMessage `json:"messages"`
			Stream    bool           `json:"stream"`
			MaxTokens int            `json:"max_tokens"`
		}
		if json.NewDecoder(r.Body).Decode(&in) != nil || !in.Stream || in.MaxTokens != 4096 {
			http.Error(w, "bad", 400)
			return
		}
		raw := string(jsonValue(in.Messages))
		if strings.Contains(raw, "NEVER_SEND_OUTSIDE_AGENT_SCOPE") {
			leaked.Store(true)
		}
		n := calls.Add(1)
		switch n {
		case 1:
			agentTestSSE(w, "", "get_document", string(jsonValue(map[string]any{"document_id": did, "start_byte": 0})))
		case 2:
			if !strings.Contains(raw, "원문 그대로") {
				t.Error("tool result missing from real provider followup")
			}
			agentTestSSE(w, "", "update_document", string(jsonValue(map[string]any{"document_id": did, "expected_version": 1, "title": "Agent 원문", "markdown": "확인한 수정\n"})))
		default:
			agentTestSSE(w, "문서 변경이 확인되었습니다.", "", "")
		}
	}))
	defer provider.Close()
	c.request("PUT", "/api/v1/admin/settings", map[string]any{"ai_enabled": true, "ai_base_url": provider.URL + "/v1", "ai_model": "tool-model", "ai_max_tokens": 4096}, 200)
	a := agentTestConfig(t, c, wid, did, []string{"get_document", "update_document"})
	aid := str(a, "id")
	run := testJSONObject(t, c.request("POST", "/api/v1/agents/"+aid+"/runs", map[string]any{"prompt": "문서를 읽고 고쳐줘", "expected_agent_version": 1}, 202))
	rid := str(run, "id")
	agentTestPump(t, s, rid, "awaiting_confirmation")
	checkpoint, e := loadAgentRun(t.Context(), s.DB, rid, false)
	if e != nil {
		t.Fatal(e)
	}
	unchanged := testJSONObject(t, c.request("GET", "/api/v1/documents/"+did, nil, 200))
	if number(unchanged, "version", 0) != 1 || str(unchanged, "markdown") != "원문 그대로\n" {
		t.Fatal("document mutated before human confirmation")
	}
	view := testJSONObject(t, c.request("GET", "/api/v1/agent-runs/"+rid, nil, 200))
	actions := view["actions"].([]any)
	action := actions[0].(map[string]any)
	c.request("POST", "/api/v1/agent-runs/"+rid+"/actions/"+str(action, "id")+"/confirm", map[string]any{"action_hash": "changed", "confirm": true}, 409)
	c.request("POST", "/api/v1/agent-runs/"+rid+"/actions/"+str(action, "id")+"/confirm", map[string]any{"action_hash": action["action_hash"], "confirm": true}, 202)
	agentTestPump(t, s, rid, "succeeded")
	updated := testJSONObject(t, c.request("GET", "/api/v1/documents/"+did, nil, 200))
	if number(updated, "version", 0) != 2 || str(updated, "markdown") != "확인한 수정\n" {
		t.Fatalf("actual REST change missing %+v", updated)
	}
	var receipts int
	if e := s.DB.QueryRow(t.Context(), `SELECT count(*) FROM automation_effects e JOIN agent_actions a ON a.job_id=e.job_id WHERE a.run_id=$1`, rid).Scan(&receipts); e != nil || receipts != 1 {
		t.Fatalf("exactly-once receipt %d %v", receipts, e)
	}
	if leaked.Load() {
		t.Fatal("out-of-allowlist content sent to model")
	}
	if calls.Load() != 3 {
		t.Fatalf("loop calls %d", calls.Load())
	}
	c.request("POST", "/api/v1/agent-runs/"+rid+"/actions/"+str(action, "id")+"/confirm", map[string]any{"action_hash": action["action_hash"], "confirm": true}, 409)
	// Reproduce the durable state immediately after the document+receipt commit
	// but before action reconciliation (a process-crash boundary). A replay may
	// reconcile the receipt, never apply the original update a second time.
	var actionJobID string
	if e = s.DB.QueryRow(t.Context(), `SELECT job_id::text FROM agent_actions WHERE id=$1`, str(action, "id")).Scan(&actionJobID); e != nil {
		t.Fatal(e)
	}
	job, e := scanJob(s.DB.QueryRow(t.Context(), `SELECT `+jobSelect+` FROM automation_jobs WHERE id=$1`, actionJobID))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.DB.Exec(t.Context(), `UPDATE agent_runs SET status='pending',job_id=$2,messages=$3,step=$4 WHERE id=$1`, rid, actionJobID, jsonValue(checkpoint.Messages), checkpoint.Step); e != nil {
		t.Fatal(e)
	}
	if _, e = s.DB.Exec(t.Context(), `UPDATE agent_actions SET status='confirmed',result=NULL WHERE id=$1`, str(action, "id")); e != nil {
		t.Fatal(e)
	}
	replayCtx := context.WithValue(t.Context(), jobContextKey{}, jobContext{ActorID: job.ActorID, TokenID: job.TokenID, JobID: job.ID, LeaseID: job.LeaseID, Constraints: job.Constraints})
	if _, e = s.runAgentAction(replayCtx, job); e != nil {
		t.Fatalf("receipt replay: %v", e)
	}
	agentTestPump(t, s, rid, "succeeded")
	updated = testJSONObject(t, c.request("GET", "/api/v1/documents/"+did, nil, 200))
	if number(updated, "version", 0) != 2 {
		t.Fatal("crash replay performed a second mutation")
	}
}

func TestPostgresWorkspaceAgentProviderRevocationAndScope(t *testing.T) {
	s, c, wid, uid := agentTestSetup(t)
	doc := testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "접근 대상", "markdown": "내용"}, 200))
	did := str(doc, "id")
	var calls atomic.Int64
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		c.request("PUT", "/api/v1/admin/settings", map[string]any{"ai_model": "changed"}, 200)
		agentTestSSE(w, "MUST_NOT_ESCAPE_AFTER_CHANGE", "", "")
	}))
	defer provider.Close()
	c.request("PUT", "/api/v1/admin/settings", map[string]any{"ai_enabled": true, "ai_base_url": provider.URL, "ai_model": "tool-model"}, 200)
	a := agentTestConfig(t, c, wid, did, []string{"get_document"})
	rid := str(testJSONObject(t, c.request("POST", "/api/v1/agents/"+str(a, "id")+"/runs", map[string]any{"prompt": "검사", "expected_agent_version": 1}, 202)), "id")
	agentTestPump(t, s, rid, "failed")
	var events string
	s.DB.QueryRow(t.Context(), `SELECT coalesce(string_agg(data::text,''),'') FROM agent_events WHERE run_id=$1`, rid).Scan(&events)
	if strings.Contains(events, "MUST_NOT_ESCAPE") {
		t.Fatal("provider change text leaked")
	}
	p := &Principal{ID: uid, Role: "admin", ScopeRestricted: true, Scopes: []string{"ai:execute"}, WorkspaceID: wid}
	cfg, _ := agentConfig(t.Context(), s.DB, str(a, "id"), false)
	call := agentToolCall{Type: "function"}
	call.Function.Name = "get_document"
	call.Function.Arguments = string(jsonValue(map[string]any{"document_id": did, "start_byte": 0}))
	if _, _, e := s.agentReadTool(t.Context(), p, cfg, call); e == nil {
		t.Fatal("AI-only token read documents")
	}
	p.Scopes = append(p.Scopes, "document:read")
	outside := newID()
	call.Function.Arguments = string(jsonValue(map[string]any{"document_id": outside, "start_byte": 0}))
	if _, _, e := s.agentReadTool(t.Context(), p, cfg, call); e == nil {
		t.Fatal("out of scope document permitted")
	}
}

func TestWorkspaceAgentStreamingToolValidation(t *testing.T) {
	for _, tc := range []struct {
		name, stream string
		ok           bool
	}{{"partial", `data: {"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"x","function":{"name":"get_document","arguments":"{"}}]}}]}` + "\n\n", false}, {"unknown", `data: {"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"x","function":{"name":"run_shell","arguments":"{}"}}]},"finish_reason":"tool_calls"}]}` + "\n\ndata: [DONE]\n\n", false}, {"bounded", `data: {"choices":[{"index":0,"delta":{"tool_calls":[{"index":8,"id":"x","function":{"name":"get_document","arguments":"{}"}}]},"finish_reason":"tool_calls"}]}` + "\n\ndata: [DONE]\n\n", false}} {
		t.Run(tc.name, func(t *testing.T) {
			provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				fmt.Fprint(w, tc.stream)
			}))
			defer provider.Close()
			_, e := agentModelTurn(context.Background(), map[string]any{"ai_enabled": true, "ai_base_url": provider.URL, "ai_model": "test"}, workspaceAgent{Tools: []string{"get_document"}, MaxTokens: 4096}, []agentMessage{{Role: "user", Content: "test"}}, nil)
			if (e == nil) != tc.ok {
				t.Fatalf("validation error %v", e)
			}
		})
	}
}
