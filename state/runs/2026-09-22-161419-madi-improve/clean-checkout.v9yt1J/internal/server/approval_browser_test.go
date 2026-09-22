package server

import (
	"context"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestBrowserAdvancedApprovalWorkflow(t *testing.T) {
	if os.Getenv("MADI_BROWSER_APPROVAL") != "1" {
		t.Skip("set MADI_BROWSER_APPROVAL=1 after web build")
	}
	s, _ := integrationTestServer(t)
	app, e := New(context.Background(), s.DB, s.EncryptionKey, "approval-test", "admin@example.test", "Integration-Test-Password-2026!", os.DirFS("../../web/dist"))
	if e != nil {
		t.Fatal(e)
	}
	server := httptest.NewServer(app)
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "node", "../../web/src/approval/browser-test.mjs")
	command.Env = append(os.Environ(), "MADI_BASE_URL="+server.URL)
	output, e := command.CombinedOutput()
	if e != nil {
		t.Fatalf("approval browser: %v\n%s", e, output)
	}
	t.Log(string(output))
}
