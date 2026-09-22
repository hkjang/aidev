package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBrowserAISelection(t *testing.T) {
	if os.Getenv("MADI_BROWSER_SELECTION") != "1" {
		t.Skip("set MADI_BROWSER_SELECTION=1 after web build")
	}
	seed, _ := integrationTestServer(t)
	app, err := New(t.Context(), seed.DB, seed.EncryptionKey, "0.1.0", "admin@example.test", "Integration-Test-Password-2026!", os.DirFS("../../web/dist"))
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(app)
	defer server.Close()
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		if json.NewDecoder(r.Body).Decode(&payload) != nil {
			w.WriteHeader(400)
			return
		}
		for _, msg := range payload["messages"].([]any) {
			text := msg.(map[string]any)["content"].(string)
			if strings.Contains(text, "NEVER_SEND_OUTSIDE_SELECTION") || strings.Contains(text, "제목과 태그 보존 검증") {
				t.Error("unselected source sent to provider")
			}
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"명료해진 제안 문장 😀\"}}]}\n\ndata: [DONE]\n\n")
	}))
	defer provider.Close()
	client := newIntegrationTestClient(t, server.URL)
	client.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200)
	client.request("PUT", "/api/v1/admin/settings", map[string]any{"ai_enabled": true, "ai_base_url": provider.URL + "/v1", "ai_model": "selection-browser", "ai_max_tokens": 262144}, 200)
	ctx, cancel := context.WithTimeout(t.Context(), 180*time.Second)
	defer cancel()
	root, err := filepath.Abs("../../test-results/ai-selection")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(ctx, "node", "../../tests/ai-selection.mjs")
	cmd.Env = append(os.Environ(), "MADI_BASE_URL="+server.URL, "MADI_ADMIN_PASSWORD=Integration-Test-Password-2026!", "MADI_SCREENSHOT_DIR="+root)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("AI selection browser %v\n%s", err, out)
	}
	t.Log(string(out))
}
