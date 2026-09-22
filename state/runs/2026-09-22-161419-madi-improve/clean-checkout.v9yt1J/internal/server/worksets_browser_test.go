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

func TestBrowserWorksets(t *testing.T) {
	if os.Getenv("MADI_BROWSER_WORKSETS") != "1" {
		t.Skip("set MADI_BROWSER_WORKSETS=1 after web build")
	}
	seed, _ := integrationTestServer(t)
	app, e := New(t.Context(), seed.DB, seed.EncryptionKey, "0.1.0", "admin@example.test", "Integration-Test-Password-2026!", os.DirFS("../../web/dist"))
	if e != nil {
		t.Fatal(e)
	}
	server := httptest.NewServer(app)
	defer server.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 180*time.Second)
	defer cancel()
	outDir, e := filepath.Abs("../../test-results/worksets")
	if e != nil {
		t.Fatal(e)
	}
	cmd := exec.CommandContext(ctx, "node", "../../tests/worksets.mjs")
	cmd.Env = append(os.Environ(), "MADI_BASE_URL="+server.URL, "MADI_SCREENSHOT_DIR="+outDir)
	out, e := cmd.CombinedOutput()
	if e != nil {
		t.Fatalf("worksets browser %v\n%s", e, out)
	}
	t.Log(string(out))
}
