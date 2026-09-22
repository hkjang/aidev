package server

import (
	"context"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// Reuses the shared-browser suite against a fresh schema so document-scoped
// board/calendar regressions can be checked without changing a development DB.
func TestBrowserTasks(t *testing.T) {
	if os.Getenv("MADI_BROWSER_TASKS") != "1" {
		t.Skip("set MADI_BROWSER_TASKS=1 after web build")
	}
	seed, _ := integrationTestServer(t)
	app, e := New(t.Context(), seed.DB, seed.EncryptionKey, "0.1.0", "admin@example.test", "Integration-Test-Password-2026!", os.DirFS("../../web/dist"))
	if e != nil {
		t.Fatal(e)
	}
	server := httptest.NewServer(app)
	defer server.Close()
	outDir, e := filepath.Abs("../../test-results/tasks-final")
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 180*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "node", "../../tests/tasks.mjs")
	cmd.Env = append(os.Environ(), "MADI_BASE_URL="+server.URL, "MADI_TEST_PASSWORD=Integration-Test-Password-2026!", "MADI_SCREENSHOT_DIR="+outDir)
	out, e := cmd.CombinedOutput()
	if e != nil {
		t.Fatalf("tasks browser: %v\n%s", e, out)
	}
	t.Log(string(out))
}
