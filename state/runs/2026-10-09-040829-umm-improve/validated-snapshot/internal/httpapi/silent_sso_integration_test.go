package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/auth"
	"github.com/hkjang/umm/internal/cryptoutil"
)

// Someone already signed in at Keycloak should walk straight into umm, and
// someone who is not should see the login screen once — not a browser that
// bounces between the two for as long as they watch it.
//
// prompt=none never draws a screen: the provider either hands a code straight
// back or answers login_required. The whole of this feature is what happens
// on that second answer, so the test pins the three server-side halves of the
// loop guard: the administrator's setting is the only thing that can start a
// silent attempt, a refusal lands on the login screen with a marker in the
// address, and a real error is told apart from an ordinary "no session".
func TestSilentSSOIsTheAdministratorsToStartAndLandsRefusalsOnTheLoginScreenIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	ctx := context.Background()
	db := isolatedHTTPStore(t, dsn)

	// A stand-in for Keycloak that answers discovery only; the test never
	// gets as far as an authorization screen because the redirect itself is
	// the thing under test.
	var issuer *httptest.Server
	issuer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/.well-known/openid-configuration" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                 issuer.URL,
			"authorization_endpoint": issuer.URL + "/auth",
			"token_endpoint":         issuer.URL + "/token",
			"jwks_uri":               issuer.URL + "/keys",
		})
	}))
	t.Cleanup(issuer.Close)

	adminID := uuid.New()
	username := "silent_sso_" + strings.ReplaceAll(adminID.String(), "-", "")
	if _, err := db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name,role) VALUES($1,$2::citext,$2::text,'admin')`, adminID, username); err != nil {
		t.Fatal(err)
	}
	if err := db.PutSetting(ctx, "general", map[string]any{"service_name": "umm", "public_url": "https://umm.example", "session_hours": 24}, adminID); err != nil {
		t.Fatal(err)
	}
	configure := func(autoLogin bool) {
		t.Helper()
		err := db.PutSetting(ctx, "oidc", map[string]any{
			"enabled": true, "issuer_url": issuer.URL, "client_id": "umm", "client_secret": "plain",
			"auto_login": autoLogin,
		}, adminID)
		if err != nil {
			t.Fatal(err)
		}
	}
	cipher, err := cryptoutil.New([]byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	authService := &auth.Service{Store: db}
	server := &Server{Store: db, Auth: authService, Cipher: cipher, OIDC: &auth.OIDCService{Store: db, Cipher: cipher, Sessions: authService}}
	router := chi.NewRouter()
	router.Get("/meta", server.meta)
	router.Get("/auth/oidc/start", server.OIDC.Start)
	router.Get("/auth/oidc/callback", server.OIDC.Callback)

	get := func(target string) *httptest.ResponseRecorder {
		t.Helper()
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, target, nil))
		return response
	}
	redirect := func(target string) *url.URL {
		t.Helper()
		response := get(target)
		if response.Code != http.StatusFound {
			t.Fatalf("GET %s = %d, want a redirect: %s", target, response.Code, response.Body.String())
		}
		location, err := url.Parse(response.Header().Get("Location"))
		if err != nil {
			t.Fatal(err)
		}
		return location
	}
	autoLoginAnnounced := func() bool {
		t.Helper()
		var meta struct {
			OIDCEnabled   bool `json:"oidcEnabled"`
			OIDCAutoLogin bool `json:"oidcAutoLogin"`
		}
		if err := json.Unmarshal(get("/meta").Body.Bytes(), &meta); err != nil {
			t.Fatal(err)
		}
		if !meta.OIDCEnabled {
			t.Fatal("meta says SSO is off while it is on")
		}
		return meta.OIDCAutoLogin
	}

	// Off — the default — and asking for prompt=none changes nothing: the
	// browser is sent on an ordinary login. Whoever appends it to a URL gets
	// the flow the administrator chose, not a different one.
	configure(false)
	if autoLoginAnnounced() {
		t.Fatal("meta announces auto login while the setting is off")
	}
	location := redirect("/auth/oidc/start?prompt=none&return_to=%2Fspace%2Fabc")
	if got := location.Query().Get("prompt"); got != "" {
		t.Fatalf("auto_login off, yet the provider is asked for prompt=%q", got)
	}

	// On: the same request is a silent one, and the deep link is kept so the
	// person lands where they were going rather than on the front page.
	configure(true)
	if !autoLoginAnnounced() {
		t.Fatal("meta does not announce auto login while the setting is on")
	}
	location = redirect("/auth/oidc/start?prompt=none&return_to=%2Fspace%2Fabc")
	if got := location.Query().Get("prompt"); got != "none" {
		t.Fatalf("auto_login on, yet the provider is asked for prompt=%q, want none", got)
	}
	if location.Host != strings.TrimPrefix(issuer.URL, "http://") || location.Path != "/auth" {
		t.Fatalf("silent attempt went to %s, want the provider's authorization endpoint", location)
	}
	state := location.Query().Get("state")
	var returnTo string
	if err := db.Pool.QueryRow(ctx, `SELECT return_to FROM oauth_states WHERE state_hash=$1`, digestOf(state)).Scan(&returnTo); err != nil {
		t.Fatalf("the state of the silent attempt was not recorded: %v", err)
	}
	if returnTo != "/space/abc" {
		t.Fatalf("return_to = %q, want the deep link kept", returnTo)
	}
	// Without prompt=none the switch is not a silent attempt on its own: a
	// person pressing the SSO button on the login screen still gets a screen.
	if got := redirect("/auth/oidc/start").Query().Get("prompt"); got != "" {
		t.Fatalf("an ordinary login asks the provider for prompt=%q", got)
	}

	// No session at the provider: login_required comes back in place of a
	// code. That is the ordinary answer, not a failure — the person is shown
	// the login screen, and the address carries the marker that stops the
	// browser from trying again even when its storage cannot be read.
	landing := redirect("/auth/oidc/callback?state=" + url.QueryEscape(state) + "&error=login_required")
	if landing.Path != "/login" || landing.Query().Get("sso") != "none" {
		t.Fatalf("a refused silent attempt landed on %s, want /login?sso=none", landing)
	}
	// And it keeps the address that was being opened, so signing in from the
	// login screen still lands there rather than on the front page.
	if got := landing.Query().Get("return_to"); got != "/space/abc" {
		t.Fatalf("a refused silent attempt dropped the deep link: return_to=%q", got)
	}
	var leftover int
	if err := db.Pool.QueryRow(ctx, `SELECT count(*) FROM oauth_states WHERE state_hash=$1`, digestOf(state)).Scan(&leftover); err != nil {
		t.Fatal(err)
	}
	if leftover != 0 {
		t.Fatal("the refused attempt's state was left behind")
	}
	// Any other answer from the provider is a real error and says so.
	landing = redirect("/auth/oidc/callback?state=other&error=access_denied")
	if landing.Path != "/login" || landing.Query().Get("sso") != "error" {
		t.Fatalf("a provider error landed on %s, want /login?sso=error", landing)
	}

	// A callback that cannot sign anyone in — an expired or replayed state,
	// or a code the provider will not exchange — lands on the login screen,
	// never on a page of plain text.
	landing = redirect("/auth/oidc/callback?state=never-issued&code=abc")
	if landing.Path != "/login" || landing.Query().Get("sso") != "error" || landing.Query().Get("return_to") != "" {
		t.Fatalf("an unknown state landed on %s, want /login?sso=error with no return_to", landing)
	}
	location = redirect("/auth/oidc/start?return_to=%2Fspace%2Fdef")
	landing = redirect("/auth/oidc/callback?state=" + url.QueryEscape(location.Query().Get("state")) + "&code=abc")
	if landing.Path != "/login" || landing.Query().Get("sso") != "error" || landing.Query().Get("return_to") != "/space/def" {
		t.Fatalf("a failed exchange landed on %s, want /login?sso=error&return_to=/space/def", landing)
	}

	// Keycloak unreachable. A silent attempt is a navigation nobody asked
	// for, and it used to end on "503" text at an API address that a reload
	// only fetched again. It lands on the login screen with the marker that
	// stops another attempt, and the deep link kept.
	issuer.Close()
	server.OIDC = &auth.OIDCService{Store: db, Cipher: cipher, Sessions: authService}
	router.Get("/auth/oidc/start-cold", server.OIDC.Start)
	landing = redirect("/auth/oidc/start-cold?prompt=none&return_to=%2Fspace%2Fabc")
	if landing.Path != "/login" || landing.Query().Get("sso") != "error" || landing.Query().Get("return_to") != "/space/abc" {
		t.Fatalf("with Keycloak down a silent attempt landed on %s, want /login?sso=error&return_to=/space/abc", landing)
	}

	// A return_to that would leave the site is not a place to return to. A
	// backslash is one: browsers read /\evil.example as //evil.example.
	for _, outside := range []string{"https://evil.example/", "//evil.example/x", "evil", "/\\evil.example/x"} {
		location = redirect("/auth/oidc/start?return_to=" + url.QueryEscape(outside))
		if err := db.Pool.QueryRow(ctx, `SELECT return_to FROM oauth_states WHERE state_hash=$1`, digestOf(location.Query().Get("state"))).Scan(&returnTo); err != nil {
			t.Fatal(err)
		}
		if returnTo != "/" {
			t.Fatalf("return_to %q was kept as %q, want /", outside, returnTo)
		}
	}
}

// digestOf is how the store keys an OAuth state: by its SHA-256, never the
// value itself.
func digestOf(state string) []byte {
	sum := sha256.Sum256([]byte(state))
	return sum[:]
}
