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

func TestBrowserDocumentAccessAndUndo(t *testing.T) {
	if os.Getenv("MADI_BROWSER_ACCESS") != "1" {
		t.Skip("set MADI_BROWSER_ACCESS=1 after web build")
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
	root, e := filepath.Abs("../../test-results/document-access")
	if e != nil {
		t.Fatal(e)
	}
	cmd := exec.CommandContext(ctx, "node", "../../tests/document-access.mjs")
	cmd.Env = append(os.Environ(), "MADI_BASE_URL="+server.URL, "MADI_SCREENSHOT_DIR="+root)
	out, e := cmd.CombinedOutput()
	if e != nil {
		t.Fatalf("document access browser %v\n%s", e, out)
	}
	t.Log(string(out))
}
