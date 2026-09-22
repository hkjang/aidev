package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"slices"
	"sync/atomic"
	"testing"
	"time"
)

func TestBrowserGraphAI(t *testing.T) {
	if os.Getenv("MADI_BROWSER_GRAPH_AI") != "1" {
		t.Skip("set MADI_BROWSER_GRAPH_AI=1 after web build")
	}
	s, c, wid, _ := graphAITestSetup(t)
	if _, e := s.DB.Exec(t.Context(), `UPDATE workspaces SET name='지식 운영' WHERE id=$1`, wid); e != nil {
		t.Fatal(e)
	}
	a := testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "PostgreSQL 운영 기준", "markdown": "# PostgreSQL 운영 기준\n\n서비스의 데이터 관리 원칙과 검증 기준을 기록합니다.\n", "visibility": "private"}, 200))
	b := testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "PostgreSQL 장애 복구", "markdown": "# PostgreSQL 장애 복구\n\n운영 기준을 바탕으로 복구 절차를 확인합니다.\n"}, 200))
	aid, bid := str(a, "id"), str(b, "id")
	if _, e := s.DB.Exec(t.Context(), `INSERT INTO knowledge_document_meta(document_id,classification) VALUES($1,'confidential')`, aid); e != nil {
		t.Fatal(e)
	}
	var calls atomic.Int32
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		if json.NewDecoder(r.Body).Decode(&payload) != nil || !boolean(payload, "stream") {
			http.Error(w, "invalid streaming request", 400)
			return
		}
		messages, ok := payload["messages"].([]any)
		if !ok || len(messages) == 0 {
			http.Error(w, "missing context", 400)
			return
		}
		var input struct {
			Kinds []string `json:"allowed_kinds"`
		}
		if json.Unmarshal([]byte(str(messages[len(messages)-1].(map[string]any), "content")), &input) != nil {
			http.Error(w, "invalid input", 400)
			return
		}
		calls.Add(1)
		if str(payload, "model") == "취소 검증 모델" {
			w.Header().Set("Content-Type", "text/event-stream")
			fmt.Fprintf(w, "data: %s\n\n", jsonValue(map[string]any{"choices": []any{map[string]any{"delta": map[string]any{"content": `{"candidates":[`}}}}))
			w.(http.Flusher).Flush()
			<-r.Context().Done()
			return
		}
		candidates := []graphAICandidate{}
		for _, candidate := range graphAITestCandidates(aid, bid) {
			if slices.Contains(input.Kinds, candidate.Kind) {
				candidates = append(candidates, candidate)
			}
		}
		agentTestSSE(w, string(jsonValue(map[string]any{"candidates": candidates})), "", "")
	}))
	defer provider.Close()
	c.request("PUT", "/api/v1/admin/settings", map[string]any{"ai_enabled": true, "ai_base_url": provider.URL, "ai_model": "지식 제안 검증 모델", "ai_max_tokens": 4096}, 200)
	app, e := New(context.Background(), s.DB, s.EncryptionKey, "graph-ai-test", "admin@example.test", "Integration-Test-Password-2026!", os.DirFS("../../web/dist"))
	if e != nil {
		t.Fatal(e)
	}
	server := httptest.NewServer(app)
	defer server.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 180*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "node", "../../tests/graph-ai.mjs")
	command.Env = append(os.Environ(), "MADI_BASE_URL="+server.URL, "MADI_GRAPH_AI_WORKSPACE="+wid, "MADI_GRAPH_AI_SOURCE_A="+aid, "MADI_GRAPH_AI_SOURCE_B="+bid)
	output, e := command.CombinedOutput()
	if e != nil {
		t.Fatalf("Graph AI browser %v\n%s", e, output)
	}
	if calls.Load() != 3 {
		t.Fatalf("browser must explicitly request exactly three provider analyses, got %d", calls.Load())
	}
	t.Log(string(output))
}
