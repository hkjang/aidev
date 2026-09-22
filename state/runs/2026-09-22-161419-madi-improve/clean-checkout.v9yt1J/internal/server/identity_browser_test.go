package server

import (
	"context"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestBrowserIdentityManagement(t *testing.T) {
	if os.Getenv("MADI_BROWSER_IDENTITY") != "1" {
		t.Skip("set MADI_BROWSER_IDENTITY=1 after web build")
	}
	s, admin, _, _, _ := collaborationTestSetup(t)
	endpoint, ca, _ := identityTestLDAP(t)
	admin.request("PUT", "/api/v1/admin/settings", map[string]any{"ldap_enabled": true, "ldap_url": endpoint, "ldap_ca_pem": ca, "ldap_bind_dn": "cn=reader,dc=example,dc=test", "ldap_bind_password": "reader-secret", "ldap_base_dn": "dc=example,dc=test", "ldap_auto_register": true}, 200)
	app, e := New(context.Background(), s.DB, s.EncryptionKey, "identity-test", "admin@example.test", "Integration-Test-Password-2026!", os.DirFS("../../web/dist"))
	if e != nil {
		t.Fatal(e)
	}
	server := httptest.NewServer(app)
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "node", "../../web/src/identity-browser-test.mjs")
	command.Env = append(os.Environ(), "MADI_BASE_URL="+server.URL)
	output, e := command.CombinedOutput()
	if e != nil {
		t.Fatalf("identity browser: %v\n%s", e, output)
	}
	t.Log(string(output))
}
