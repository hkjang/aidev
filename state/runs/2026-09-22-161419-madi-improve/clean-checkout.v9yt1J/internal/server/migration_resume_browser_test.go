package server

import (
	"context"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestBrowserResumableMigration(t *testing.T) {
	if os.Getenv("MADI_BROWSER_MIGRATION_RESUME") != "1" {
		t.Skip("set MADI_BROWSER_MIGRATION_RESUME=1 after web build")
	}
	seed, _ := integrationTestServer(t)
	app, e := New(context.Background(), seed.DB, seed.EncryptionKey, "0.1.0", "admin@example.test", "Integration-Test-Password-2026!", os.DirFS("../../web/dist"))
	if e != nil {
		t.Fatal(e)
	}
	endpoint := httptest.NewServer(app)
	defer endpoint.Close()
	defer app.CloseCollaboration()
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()
	app.StartJobs(ctx)
	for _, script := range []string{"migration-resume-browser.mjs", "migration-browser.mjs"} {
		command := exec.CommandContext(ctx, "node", "../../tests/"+script)
		command.Env = append(os.Environ(), "MADI_BASE_URL="+endpoint.URL, "MADI_TEST_PASSWORD=Integration-Test-Password-2026!")
		output, e := command.CombinedOutput()
		if e != nil {
			t.Fatalf("%s: %v\n%s", script, e, output)
		}
		t.Log(string(output))
	}
}
