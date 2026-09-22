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
	"sync/atomic"
	"testing"
	"time"
)

func TestBrowserStructuredDrafts(t *testing.T) {
	if os.Getenv("MADI_BROWSER_STRUCTURED") != "1" {
		t.Skip("set MADI_BROWSER_STRUCTURED=1 after web build")
	}
	seed, _ := integrationTestServer(t)
	var invalid atomic.Bool
	var calls atomic.Int32
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if json.NewDecoder(r.Body).Decode(&body) != nil {
			w.WriteHeader(400)
			return
		}
		calls.Add(1)
		raw := string(jsonValue(body))
		for _, excluded := range []string{"선택하지 않은 앞쪽 설명", "선택하지 않은 뒤쪽 설명", "GPU 용량 현황", "인프라 점검 현황", "내부 식별자"} {
			if strings.Contains(raw, excluded) {
				invalid.Store(true)
			}
		}
		if body["stream"] != true || number(body, "max_tokens", 0) != 262144 {
			invalid.Store(true)
		}
		output := `{"fields":[{"property_id":"amount","value":2,"quote":"2개"},{"property_id":"state","value":"미등록","quote":"상태는 운영"},{"property_id":"tags","value":"장비","quote":"분류는 장비"}]}`
		w.Header().Set("Content-Type", "text/event-stream")
		// Two real upstream deltas, with a complete closed JSON proposal only at end.
		characters := []rune(output)
		for _, part := range []string{string(characters[:len(characters)/2]), string(characters[len(characters)/2:])} {
			fmt.Fprintf(w, "data: %s\n\n", jsonValue(map[string]any{"choices": []any{map[string]any{"delta": map[string]any{"content": part}}}}))
			w.(http.Flusher).Flush()
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer provider.Close()
	app, e := New(t.Context(), seed.DB, seed.EncryptionKey, "0.1.0", "admin@example.test", "Integration-Test-Password-2026!", os.DirFS("../../web/dist"))
	if e != nil {
		t.Fatal(e)
	}
	server := httptest.NewServer(app)
	defer server.Close()
	client := newIntegrationTestClient(t, server.URL)
	client.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200)
	client.request("PUT", "/api/v1/admin/settings", map[string]any{"ai_enabled": true, "ai_base_url": provider.URL + "/v1", "ai_model": "structured-browser", "ai_max_tokens": 262144}, 200)
	ctx, cancel := context.WithTimeout(t.Context(), 180*time.Second)
	defer cancel()
	outDir, e := filepath.Abs("../../test-results/structured-drafts")
	if e != nil {
		t.Fatal(e)
	}
	cmd := exec.CommandContext(ctx, "node", "../../tests/structured-drafts.mjs")
	cmd.Env = append(os.Environ(), "MADI_BASE_URL="+server.URL, "MADI_SCREENSHOT_DIR="+outDir)
	out, e := cmd.CombinedOutput()
	if e != nil {
		t.Fatalf("structured browser: %v\n%s", e, out)
	}
	if invalid.Load() || calls.Load() < 1 {
		t.Fatal("upstream scope or streaming contract failed", calls.Load())
	}
	t.Log(string(out))
}
