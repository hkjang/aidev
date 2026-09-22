package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDocumentAIActionsHaveSafeDistinctContract(t *testing.T) {
	seen := map[string]bool{}
	for _, a := range aiDocumentActions {
		if seen[a.ID] || a.ID == "" || a.Name == "" || a.Instruction == "" || a.Prompt == "" || a.Description == "" {
			t.Fatal("invalid/duplicate AI action", a.ID)
		}
		seen[a.ID] = true
		if got, err := documentAIAction(a.ID); err != nil || got != a {
			t.Fatal("action lookup mismatch")
		}
		b, _ := json.Marshal(a)
		if strings.Contains(string(b), "instruction") {
			t.Fatal("internal instructions need not be part of UI schema")
		}
	}
	if len(seen) < 8 {
		t.Fatal("at least eight writing/research actions required")
	}
	if a, err := documentAIAction(""); err != nil || a.ID != "ask" {
		t.Fatal("legacy requests must remain compatible")
	}
	if _, err := documentAIAction("execute_shell"); err == nil {
		t.Fatal("unknown action accepted")
	}
}

func TestPostgresDocumentAIActionsStreamWithoutAutomaticWrites(t *testing.T) {
	s, ts := integrationTestServer(t)
	client := newIntegrationTestClient(t, ts.URL)
	client.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200)
	requests := make(chan map[string]any, 20)
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var input map[string]any
		if json.NewDecoder(r.Body).Decode(&input) != nil {
			w.WriteHeader(400)
			return
		}
		requests <- input
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"검토할 제안입니다.\"}}]}\n\ndata: [DONE]\n\n")
	}))
	defer provider.Close()
	client.request("PUT", "/api/v1/admin/settings", map[string]any{"ai_enabled": true, "ai_base_url": provider.URL + "/v1", "ai_model": "actions", "ai_max_tokens": 262144}, 200)
	var before, after int
	if err := s.DB.QueryRow(t.Context(), "SELECT count(*) FROM document_versions").Scan(&before); err != nil {
		t.Fatal(err)
	}
	list := testJSONObject(t, client.request("GET", "/api/v1/ai/actions", nil, 200))
	if len(list["actions"].([]any)) != len(aiDocumentActions) || list["automatic_apply"] != false {
		t.Fatal("action catalogue inconsistent")
	}
	for _, action := range aiDocumentActions {
		body := client.request("POST", "/api/v1/ai/chat", map[string]any{"prompt": action.Prompt, "action": action.ID, "max_tokens": 262144}, 200)
		if !strings.Contains(string(body), "검토할 제안입니다.") || !strings.Contains(string(body), `"action":"`+action.ID+`"`) || strings.Contains(string(body), "retract") {
			t.Fatal(action.ID, string(body))
		}
		request := <-requests
		if request["stream"] != true || request["max_tokens"] != float64(262144) || request["model"] != "actions" {
			t.Fatal("invalid stream/model/token contract", request)
		}
		messages := request["messages"].([]any)
		if !strings.Contains(messages[0].(map[string]any)["content"].(string), action.Instruction) {
			t.Fatal("selected action missing from system instructions")
		}
	}
	client.request("POST", "/api/v1/ai/chat", map[string]any{"prompt": "unsafe", "action": "execute_shell"}, 400)
	if len(requests) != 0 {
		t.Fatal("unknown action called provider")
	}
	if err := s.DB.QueryRow(t.Context(), "SELECT count(*) FROM document_versions").Scan(&after); err != nil || before != after {
		t.Fatal("AI suggestion changed a document", before, after, err)
	}
}
