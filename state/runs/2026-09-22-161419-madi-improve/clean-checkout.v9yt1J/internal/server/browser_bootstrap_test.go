package server

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestBrowserWorkflowNativeFetchBootstrap(t *testing.T) {
	s, server := integrationTestServer(t)
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("Node.js is required for native browser-workflow bootstrap verification")
	}
	// Match the disposable CI server's credentials while retaining an isolated
	// PostgreSQL schema and ephemeral HTTP port; never contact a running service.
	hash, err := bcrypt.GenerateFromPassword([]byte("Browser-Test-Password-2026!"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.DB.Exec(context.Background(), "UPDATE users SET password_hash=$1 WHERE email='admin@example.test'", string(hash)); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join("..", "..", "scripts", "verify-browser.sh"))
	if err != nil {
		t.Fatal(err)
	}
	_, script, ok := strings.Cut(string(data), "<<'NODE'\n")
	if !ok {
		t.Fatal("native Node bootstrap block missing")
	}
	script, _, ok = strings.Cut(script, "\nNODE\n")
	if !ok {
		t.Fatal("native Node bootstrap block terminator missing")
	}
	if strings.Count(script, "http://127.0.0.1:8080/api/v1") != 1 {
		t.Fatal("unexpected workflow bootstrap origin")
	}
	script = strings.Replace(script, "http://127.0.0.1:8080/api/v1", server.URL+"/api/v1", 1)
	storage := filepath.Join(t.TempDir(), "attachments")
	command := exec.Command("node", "--input-type=module", "-e", script)
	command.Env = append(os.Environ(), "MADI_BROWSER_STORAGE="+storage)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("exact workflow native-fetch bootstrap failed: %v\n%s", err, output)
	}
	var actual string
	if err := s.DB.QueryRow(context.Background(), "SELECT data->>'storage_path' FROM settings WHERE id=1").Scan(&actual); err != nil || actual != storage {
		t.Fatalf("real admin settings were not saved: %v", err)
	}
}
