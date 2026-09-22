package server

import (
	"context"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestBrowserDocumentRaces(t *testing.T) {
	if os.Getenv("MADI_BROWSER_DOCUMENT_RACES") != "1" {
		t.Skip("set MADI_BROWSER_DOCUMENT_RACES=1 after web build")
	}
	seed, _ := integrationTestServer(t)
	app, err := New(t.Context(), seed.DB, seed.EncryptionKey, "0.2.0", "admin@example.test", "Integration-Test-Password-2026!", os.DirFS("../../web/dist"))
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(app)
	defer server.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 120*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "node", "../../tests/document-races.mjs")
	cmd.Env = append(os.Environ(), "MADI_BASE_URL="+server.URL)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("document races browser %v\n%s", err, out)
	}
	t.Log(string(out))
}
