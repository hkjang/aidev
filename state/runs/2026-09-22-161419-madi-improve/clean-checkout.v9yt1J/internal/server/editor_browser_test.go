package server

import (
	"context"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestBrowserAdvancedEditor(t *testing.T) {
	if os.Getenv("MADI_BROWSER_EDITOR") != "1" {
		t.Skip("set MADI_BROWSER_EDITOR=1 after web build")
	}
	s, _, _, _, _ := collaborationTestSetup(t)
	app, e := New(context.Background(), s.DB, s.EncryptionKey, "test", "admin@example.test", "Integration-Test-Password-2026!", os.DirFS("../../web/dist"))
	if e != nil {
		t.Fatal(e)
	}
	server := httptest.NewServer(app)
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "node", "../../web/src/editor/browser-test.mjs")
	command.Env = append(os.Environ(), "MADI_BASE_URL="+server.URL)
	output, e := command.CombinedOutput()
	if e != nil {
		t.Fatalf("advanced editor browser: %v\n%s", e, output)
	}
	t.Log(string(output))
}
