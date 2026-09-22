package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestBrowserDatabaseEditing(t *testing.T) {
	if os.Getenv("MADI_BROWSER_DATABASE_EDITING") != "1" {
		t.Skip("set MADI_BROWSER_DATABASE_EDITING=1 after web build")
	}
	seed, _ := integrationTestServer(t)
	app, e := New(t.Context(), seed.DB, seed.EncryptionKey, "0.1.0", "admin@example.test", "Integration-Test-Password-2026!", os.DirFS("../../web/dist"))
	if e != nil {
		t.Fatal(e)
	}
	// Hold a real successful row write AFTER commit, before the browser gets
	// its ACK. The browser releases this barrier explicitly; no timing guess
	// or mock row/version bypasses the ordinary CAS handler.
	committed, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	var intercepted atomic.Bool
	resume := func() { releaseOnce.Do(func() { close(release) }) }
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/__test/database-editing/ack" {
			if r.Method == http.MethodPost {
				resume()
			}
			select {
			case <-committed:
				jsonResponse(w, 200, map[string]bool{"committed": true})
			default:
				jsonResponse(w, 200, map[string]bool{"committed": false})
			}
			return
		}
		if r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/api/v1/databases/") && strings.Contains(r.URL.Path, "/rows/") {
			raw, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "test request read failed", 500)
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(raw))
			var in struct {
				Values map[string]any `json:"values"`
			}
			if json.Unmarshal(raw, &in) == nil && in.Values["state"] == "완료" && intercepted.CompareAndSwap(false, true) {
				response := httptest.NewRecorder()
				app.ServeHTTP(response, r)
				if response.Code == http.StatusOK {
					close(committed)
					select {
					case <-release:
					case <-r.Context().Done():
						return
					}
				}
				for key, values := range response.Header() {
					w.Header()[key] = values
				}
				w.WriteHeader(response.Code)
				_, _ = w.Write(response.Body.Bytes())
				return
			}
		}
		app.ServeHTTP(w, r)
	}))
	defer server.Close()
	defer resume()
	ctx, cancel := context.WithTimeout(t.Context(), 180*time.Second)
	defer cancel()
	outDir := os.Getenv("MADI_SCREENSHOT_DIR")
	if outDir == "" {
		outDir = "../../test-results/database-editing"
	}
	outDir, e = filepath.Abs(outDir)
	if e != nil {
		t.Fatal(e)
	}
	cmd := exec.CommandContext(ctx, "node", "../../tests/database-editing.mjs")
	cmd.Env = append(os.Environ(), "MADI_BASE_URL="+server.URL, "MADI_SCREENSHOT_DIR="+outDir, "MADI_DATABASE_EDITING_ACK_GATE=1")
	out, e := cmd.CombinedOutput()
	if e != nil {
		t.Fatalf("database editing browser %v\n%s", e, out)
	}
	t.Log(string(out))
	if os.Getenv("MADI_BROWSER_DATABASE_BASELINE") == "1" {
		baseline := exec.CommandContext(ctx, "node", "../../tests/database-advanced.mjs")
		baseline.Env = append(os.Environ(), "MADI_BASE_URL="+server.URL, "MADI_SCREENSHOT_DIR="+filepath.Join(outDir, "baseline"), "MADI_TEST_PASSWORD=Integration-Test-Password-2026!")
		output, err := baseline.CombinedOutput()
		if err != nil {
			t.Fatalf("database baseline browser %v\n%s", err, output)
		}
		t.Log(string(output))
	}
}
