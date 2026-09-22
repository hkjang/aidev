package server

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// This test-only process serves the real application and shares only its
// parent's disposable PostgreSQL schema. It is never a production endpoint.
func TestCollaborationSoakChild(t *testing.T) {
	if os.Getenv("MADI_COLLABORATION_CHILD") != "1" {
		t.Skip("private child of TestBrowserCollaborationSoak")
	}
	cfg, err := pgxpool.ParseConfig(os.Getenv("MADI_TEST_POSTGRES_DSN"))
	if err != nil {
		t.Fatal(err)
	}
	var params map[string]string
	if err = json.Unmarshal([]byte(os.Getenv("MADI_COLLABORATION_PARAMS")), &params); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(params["search_path"], "madi_test_") {
		t.Fatal("isolated schema required")
	}
	cfg.ConnConfig.RuntimeParams = params
	ctx, cancel := context.WithTimeout(t.Context(), 7*time.Minute)
	defer cancel()
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	key, err := base64.StdEncoding.DecodeString(os.Getenv("MADI_COLLABORATION_KEY"))
	if err != nil {
		t.Fatal(err)
	}
	app, err := New(ctx, pool, key, "soak-test", "admin@example.test", "Integration-Test-Password-2026!", os.DirFS("../../web/dist"))
	if err != nil {
		t.Fatal(err)
	}
	defer app.CloseCollaboration()
	listener, err := net.Listen("tcp", os.Getenv("MADI_COLLABORATION_ADDR"))
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: app, ReadHeaderTimeout: 5 * time.Second}
	defer server.Close()
	go func() { <-ctx.Done(); app.CloseCollaboration(); _ = server.Close() }()
	if err = server.Serve(listener); err != nil && err != http.ErrServerClosed {
		t.Fatal(err)
	}
}

func TestBrowserCollaborationSoak(t *testing.T) {
	if os.Getenv("MADI_BROWSER_COLLABORATION_SOAK") != "1" {
		t.Skip("opt-in real process restart and 2 minute Chromium IME soak")
	}
	s, _, _, _, _ := collaborationTestSetup(t)
	ctx, cancel := context.WithTimeout(t.Context(), 6*time.Minute)
	defer cancel()
	port, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := port.Addr().String()
	_ = port.Close()
	base := "http://" + addr
	params, _ := json.Marshal(s.DB.Config().ConnConfig.RuntimeParams)
	var mu sync.Mutex
	var child *exec.Cmd
	var output bytes.Buffer
	pids := []int{}
	stop := func() {
		if child != nil {
			_ = child.Process.Kill()
			_ = child.Wait()
			child = nil
		}
	}
	defer func() { mu.Lock(); stop(); mu.Unlock() }()
	start := func() error {
		output.Reset()
		child = exec.CommandContext(ctx, os.Args[0], "-test.run=^TestCollaborationSoakChild$", "-test.timeout=7m")
		child.Env = append(os.Environ(), "MADI_COLLABORATION_CHILD=1", "MADI_COLLABORATION_PARAMS="+string(params), "MADI_COLLABORATION_KEY="+base64.StdEncoding.EncodeToString(s.EncryptionKey), "MADI_COLLABORATION_ADDR="+addr)
		child.Stdout, child.Stderr = &output, &output
		if err := child.Start(); err != nil {
			return err
		}
		pids = append(pids, child.Process.Pid)
		client := &http.Client{Timeout: time.Second}
		deadline := time.Now().Add(20 * time.Second)
		for time.Now().Before(deadline) {
			response, err := client.Get(base + "/healthz")
			if err == nil {
				_ = response.Body.Close()
				if response.StatusCode == 200 {
					return nil
				}
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(100 * time.Millisecond):
			}
		}
		stop()
		return fmt.Errorf("child not healthy: %s", output.String())
	}
	if err = start(); err != nil {
		t.Fatal(err)
	}
	controlSecret := randomToken()
	control := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Test-Control") != controlSecret || r.Method != "POST" {
			http.NotFound(w, r)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		switch r.URL.Path {
		case "/restart":
			stop()
			if err := start(); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			jsonResponse(w, 200, map[string]any{"pids": pids, "actual_process_restart": true})
		default:
			http.NotFound(w, r)
		}
	}))
	defer control.Close()
	command := exec.CommandContext(ctx, "node", "../../web/src/collaboration/soak-test.mjs")
	command.Env = append(os.Environ(), "MADI_BASE_URL="+base, "MADI_COLLABORATION_CONTROL="+control.URL, "MADI_COLLABORATION_CONTROL_SECRET="+controlSecret)
	result, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("collaboration soak: %v\n%s", err, result)
	}
	t.Log(string(result))
	mu.Lock()
	stop()
	mu.Unlock()
	if len(pids) != 2 || pids[0] == pids[1] {
		t.Fatal("did not restart an actual distinct server process")
	}
	var journals, versions int
	if err = s.DB.QueryRow(t.Context(), "SELECT count(*) FROM collaboration_updates").Scan(&journals); err != nil {
		t.Fatal(err)
	}
	if err = s.DB.QueryRow(t.Context(), "SELECT count(*) FROM document_versions").Scan(&versions); err != nil {
		t.Fatal(err)
	}
	t.Logf("distinct child processes=%d immutable_versions=%d surviving_journal_entries=%d", len(pids), versions, journals)
}

func collaborationWriteEvidence(t *testing.T, name string, value any) {
	t.Helper()
	dir := filepath.Join("..", "..", "test-results", "collaboration-soak")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
		t.Fatal(err)
	}
}
