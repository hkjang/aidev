package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestBrowserKnowledgeOperations(t *testing.T) {
	if os.Getenv("MADI_BROWSER_KNOWLEDGE_OPERATIONS") != "1" {
		t.Skip("set MADI_BROWSER_KNOWLEDGE_OPERATIONS=1 after web build")
	}
	seed, _ := integrationTestServer(t)
	app, err := New(context.Background(), seed.DB, seed.EncryptionKey, "0.1.0", "admin@example.test", "Integration-Test-Password-2026!", os.DirFS("../../web/dist"))
	if err != nil {
		t.Fatal(err)
	}
	endpoint := httptest.NewServer(app)
	defer endpoint.Close()
	defer app.CloseCollaboration()
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var in map[string]any
		if json.NewDecoder(r.Body).Decode(&in) != nil || !boolean(in, "stream") {
			http.Error(w, "stream required", 400)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"GPU 정책의 동시 작업은 2개이며, 변경 전에 담당 검토가 필요합니다 [1].\"}}]}\n\ndata: [DONE]\n\n")
	}))
	defer provider.Close()
	c := newIntegrationTestClient(t, endpoint.URL)
	c.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200)
	c.request("PUT", "/api/v1/admin/settings", map[string]any{"ai_enabled": true, "ai_base_url": provider.URL + "/v1", "ai_model": "사내 검증 모델"}, 200)
	ctx, cancel := context.WithTimeout(t.Context(), 180*time.Second)
	defer cancel()
	app.StartJobs(ctx)
	command := exec.CommandContext(ctx, "node", "../../tests/knowledge-operations.mjs")
	command.Env = append(os.Environ(), "MADI_BASE_URL="+endpoint.URL, "MADI_ADMIN_PASSWORD=Integration-Test-Password-2026!")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("knowledge operations browser: %v\n%s", err, output)
	}
	t.Log(string(output))
}
