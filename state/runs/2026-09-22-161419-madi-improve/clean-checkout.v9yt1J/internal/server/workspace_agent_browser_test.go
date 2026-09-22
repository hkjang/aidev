package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestBrowserWorkspaceAgent(t *testing.T) {
	if os.Getenv("MADI_BROWSER_AGENT") != "1" {
		t.Skip("set MADI_BROWSER_AGENT=1 after web build")
	}
	s, c, wid, _ := agentTestSetup(t)
	doc := testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "지식 운영 원칙", "markdown": "# 지식 운영 원칙\n\n문서는 Markdown 원문으로 보존합니다.\n\n- 소유자를 지정합니다.\n- 변경 이력을 확인합니다.\n", "visibility": "private"}, 200))
	did := str(doc, "id")
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Messages []agentMessage `json:"messages"`
		}
		if json.NewDecoder(r.Body).Decode(&in) != nil || len(in.Messages) == 0 {
			http.Error(w, "bad", 400)
			return
		}
		last := in.Messages[len(in.Messages)-1]
		if last.Role != "tool" {
			agentTestSSE(w, "", "get_document", string(jsonValue(map[string]any{"document_id": did, "start_byte": 0})))
			return
		}
		if strings.Contains(last.ToolCallID, "get_document") {
			var value map[string]any
			json.Unmarshal([]byte(last.Content), &value)
			agentTestSSE(w, "", "update_document", string(jsonValue(map[string]any{"document_id": did, "expected_version": number(value, "version", 1), "title": "지식 운영 원칙", "markdown": "# 지식 운영 원칙\n\n문서는 Markdown 원문으로 보존합니다.\n\n- 소유자를 지정합니다.\n- 변경 이력을 확인합니다.\n- 분기별로 최신성을 점검합니다.\n"})))
			return
		}
		agentTestSSE(w, "확인한 지식 운영 원칙을 저장했습니다.\n\n분기별 최신성 점검 항목을 추가했습니다. 실제 문서의 버전과 변경 이력을 확인할 수 있습니다.", "", "")
	}))
	defer provider.Close()
	c.request("PUT", "/api/v1/admin/settings", map[string]any{"ai_enabled": true, "ai_base_url": provider.URL, "ai_model": "browser-tool-calling", "ai_max_tokens": 4096}, 200)
	a := agentTestConfig(t, c, wid, did, []string{"get_document", "update_document", "search_documents"})
	app, e := New(context.Background(), s.DB, s.EncryptionKey, "agent-test", "admin@example.test", "Integration-Test-Password-2026!", os.DirFS("../../web/dist"))
	if e != nil {
		t.Fatal(e)
	}
	server := httptest.NewServer(app)
	defer server.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 150*time.Second)
	defer cancel()
	app.StartJobs(ctx)
	command := exec.CommandContext(ctx, "node", "../../tests/workspace-agent.mjs")
	command.Env = append(os.Environ(), "MADI_BASE_URL="+server.URL, "MADI_AGENT_WORKSPACE="+wid, "MADI_AGENT_DOCUMENT="+did, "MADI_AGENT_ID="+str(a, "id"))
	output, e := command.CombinedOutput()
	if e != nil {
		t.Fatalf("Agent browser %v\n%s", e, output)
	}
	t.Log(string(output))
}
