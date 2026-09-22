package server

import (
	"context"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestBrowserRunbookIsolatedExecution(t *testing.T) {
	if os.Getenv("MADI_BROWSER_RUNBOOK") != "1" {
		t.Skip("set MADI_BROWSER_RUNBOOK=1 after web build")
	}
	s, _, _, _, _, doc, _ := runbookTestSetup(t)
	app, e := New(context.Background(), s.DB, s.EncryptionKey, "runbook-test", "admin@example.test", "Integration-Test-Password-2026!", os.DirFS("../../web/dist"))
	if e != nil {
		t.Fatal(e)
	}
	server := httptest.NewServer(app)
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()
	app.StartJobs(ctx)
	app.StartRunbookMaintenance(ctx)
	command := exec.CommandContext(ctx, "node", "../../web/src/runbook/browser-test.mjs")
	command.Env = append(os.Environ(), "MADI_BASE_URL="+server.URL, "MADI_RUNBOOK_DOCUMENT="+doc)
	output, e := command.CombinedOutput()
	if e != nil {
		t.Fatalf("runbook browser %v\n%s", e, output)
	}
	t.Log(string(output))
}
