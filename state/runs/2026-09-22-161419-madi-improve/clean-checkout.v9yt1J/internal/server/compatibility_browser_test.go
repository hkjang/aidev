package server

import (
	"context"
	"net/http/httptest"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Opt-in representative matrix. This is not a claim about native OS Safari or
// an operating system IME: Node records the exact Playwright engine versions.
func TestBrowserCompatibility(t *testing.T) {
	if os.Getenv("MADI_BROWSER_COMPATIBILITY") != "1" {
		t.Skip("MADI_BROWSER_COMPATIBILITY=1 after web build and browser installation")
	}
	seed, _ := integrationTestServer(t)
	var version string
	var number int
	if e := seed.DB.QueryRow(t.Context(), `SELECT current_setting('server_version'),current_setting('server_version_num')::int`).Scan(&version, &number); e != nil {
		t.Fatal(e)
	}
	major := strconv.Itoa(number / 10000)
	if major != "17" && major != "18" {
		t.Fatalf("matrix target must be PostgreSQL 17 or 18, got %s", version)
	}
	if want := os.Getenv("MADI_COMPAT_PG_MAJOR"); want != "" && want != major {
		t.Fatalf("requested PostgreSQL %s but connected to %s", want, version)
	}
	rawVersion, e := os.ReadFile("../../VERSION")
	if e != nil {
		t.Fatal(e)
	}
	candidate := strings.TrimSpace(string(rawVersion))
	app, e := New(t.Context(), seed.DB, seed.EncryptionKey, candidate, "admin@example.test", "Integration-Test-Password-2026!", os.DirFS("../../web/dist"))
	if e != nil {
		t.Fatal(e)
	}
	endpoint := httptest.NewServer(app)
	defer endpoint.Close()
	defer app.CloseCollaboration()
	ctx, cancel := context.WithTimeout(t.Context(), 6*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "node", "../../tests/compatibility-browser.mjs")
	cmd.Env = append(os.Environ(), "MADI_BASE_URL="+endpoint.URL,
		"MADI_TEST_PASSWORD=Integration-Test-Password-2026!", "MADI_COMPAT_PG_MAJOR="+major,
		"MADI_COMPAT_APP_VERSION="+candidate,
		"MADI_COMPAT_PG_VERSION="+version, "MADI_COMPAT_GO="+runtime.Version(),
		"MADI_COMPAT_PLATFORM="+strings.Join([]string{runtime.GOOS, runtime.GOARCH}, "/"))
	output, e := cmd.CombinedOutput()
	if e != nil {
		t.Fatalf("compatibility browser: %v\n%s", e, output)
	}
	t.Log(string(output))
}
