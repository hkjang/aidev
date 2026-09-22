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

func TestBrowserSystemStatus(t *testing.T) {
	if os.Getenv("MADI_BROWSER_SYSTEM_STATUS") != "1" {
		t.Skip("set MADI_BROWSER_SYSTEM_STATUS=1 after web build")
	}
	seed, _, wid, uid, doc := systemStatusTestSetup(t)
	app, e := New(context.Background(), seed.DB, seed.EncryptionKey, "0.1.0", "admin@example.test", "Integration-Test-Password-2026!", os.DirFS("../../web/dist"))
	if e != nil {
		t.Fatal(e)
	}
	systemStatusInstallTest(t, app)
	server := httptest.NewServer(app)
	defer server.Close()
	defer app.CloseCollaboration()
	ctx, cancel := context.WithTimeout(t.Context(), 180*time.Second)
	defer cancel()
	out, _ := filepath.Abs("../../test-results/system-status")
	command := exec.CommandContext(ctx, "node", "../../tests/system-status.mjs")
	command.Env = append(os.Environ(), "MADI_BASE_URL="+server.URL, "MADI_WORKSPACE_ID="+wid, "MADI_DOCUMENT_ID="+doc, "MADI_ACTOR_ID="+uid, "MADI_SCREENSHOT_DIR="+out)
	raw, e := command.CombinedOutput()
	if e != nil {
		t.Fatalf("system status browser: %v\n%s", e, raw)
	}
	t.Log(string(raw))
}
