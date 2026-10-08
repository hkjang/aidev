package mcp_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/coreos/go-oidc/v3/oidc/oidctest"
	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/auth"
	"github.com/hkjang/umm/internal/cryptoutil"
	"github.com/hkjang/umm/internal/mcp"
	"github.com/hkjang/umm/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// An MCP client may sign the person in through Keycloak instead of carrying a
// key, once an administrator turns that on — and only then.
//
// The pieces pinned here are the ones that decide who gets in and with what:
// the door stays exactly as it was while the setting is off; when on, a
// refused client is told where to sign in and no more; a token is accepted
// only for this audience from this issuer; what it may do is its scope claim
// bounded by the administrator's key policy; a person seen for the first time
// is created with the same group mapping the login screen uses, and a person
// seen before is taken exactly as they are — a token neither promotes nor
// reactivates anyone; and a key and a browser session are treated as before.
func TestMCPSignsInThroughKeycloakOnlyWhenTurnedOnIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	ctx := context.Background()
	db := isolatedMCPStore(t, dsn)

	// A stand-in for Keycloak: discovery and a key set, nothing more. Tokens
	// are minted here directly, as the authorization server would.
	private, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	keycloak := &oidctest.Server{PublicKeys: []oidctest.PublicKey{{PublicKey: private.Public(), KeyID: "kc-key", Algorithm: oidc.RS256}}}
	issuer := httptest.NewServer(keycloak)
	t.Cleanup(issuer.Close)
	keycloak.SetIssuer(issuer.URL)
	type claims struct {
		Subject  string
		Audience string
		Scope    string
		Groups   []string
		Expires  time.Time
		Issuer   string
		Key      *rsa.PrivateKey
	}
	mint := func(c claims) string {
		t.Helper()
		if c.Audience == "" {
			c.Audience = "https://umm.example/mcp"
		}
		if c.Expires.IsZero() {
			c.Expires = time.Now().Add(time.Hour)
		}
		if c.Issuer == "" {
			c.Issuer = issuer.URL
		}
		if c.Key == nil {
			c.Key = private
		}
		groups, _ := json.Marshal(c.Groups)
		raw := `{"iss":` + strconv.Quote(c.Issuer) + `,"aud":` + strconv.Quote(c.Audience) + `,"sub":` + strconv.Quote(c.Subject) +
			`,"exp":` + strconv.FormatInt(c.Expires.Unix(), 10) + `,"scope":` + strconv.Quote(c.Scope) +
			`,"preferred_username":` + strconv.Quote("kc-"+c.Subject) + `,"name":"Someone","email":` + strconv.Quote(c.Subject+"@example.com") +
			`,"groups":` + string(groups) + `}`
		return oidctest.SignIDToken(c.Key, "kc-key", oidc.RS256, raw)
	}

	adminID := uuid.New()
	if _, err = db.Pool.Exec(ctx, `INSERT INTO users(id,username,display_name,role) VALUES($1,$2::citext,$2::text,'admin')`, adminID, "mcp_oauth_admin"); err != nil {
		t.Fatal(err)
	}
	if err = db.PutSetting(ctx, "general", map[string]any{"service_name": "umm", "public_url": "https://umm.example/", "session_hours": 24}, adminID); err != nil {
		t.Fatal(err)
	}
	// notes:write is deliberately not allowed on a key, so a token that claims
	// it shows whether the policy bounds tokens too.
	if err = db.PutSetting(ctx, "security", map[string]any{"api_key_scopes": []string{"notes:read", "spaces:read", "metrics:read"}, "default_key_days": 90, "rotation_overlap_hours": 24}, adminID); err != nil {
		t.Fatal(err)
	}
	configure := func(mcpOAuth bool) {
		t.Helper()
		err := db.PutSetting(ctx, "oidc", map[string]any{
			"enabled": true, "issuer_url": issuer.URL, "client_id": "umm", "client_secret": "plain",
			"admin_group": "umm-admins", "mcp_oauth": mcpOAuth,
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
	tokens := &auth.OIDCService{Store: db, Cipher: cipher, Sessions: authService}
	agents := &mcp.Handler{Store: db, Tokens: tokens, Version: "test"}
	handler := authService.Middleware(agents)

	short := func(body string) string {
		if len(body) > 220 {
			return body[:220] + "…"
		}
		return body
	}
	type reply struct {
		code      int
		body      string
		challenge string
	}
	rpc := func(payload map[string]any, apply func(*http.Request)) reply {
		t.Helper()
		body, _ := json.Marshal(payload)
		request := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		apply(request)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		return reply{response.Code, response.Body.String(), response.Header().Get("WWW-Authenticate")}
	}
	list := func(apply func(*http.Request)) reply {
		return rpc(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/list"}, apply)
	}
	call := func(tool string, args map[string]any, apply func(*http.Request)) reply {
		return rpc(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": tool, "arguments": args}}, apply)
	}
	bearer := func(token string) func(*http.Request) {
		return func(r *http.Request) { r.Header.Set("Authorization", "Bearer "+token) }
	}
	nothing := func(*http.Request) {}
	metadata := func(path string) *httptest.ResponseRecorder {
		t.Helper()
		response := httptest.NewRecorder()
		agents.ServeResourceMetadata(response, httptest.NewRequest(http.MethodGet, path, nil))
		return response
	}

	// Off — the default. A token is a stranger, the refusal is the one keys-only
	// clients have always read, with no challenge pointing anywhere, and the
	// metadata document does not exist.
	configure(false)
	if got := list(bearer(mint(claims{Subject: "early", Scope: "notes:read"}))); got.code != http.StatusUnauthorized || got.challenge != "" || !strings.Contains(got.body, "Bearer API key required") {
		t.Fatalf("with Keycloak sign-in off a token got %d %q: %s", got.code, got.challenge, short(got.body))
	}
	if got := metadata(auth.ProtectedResourceMetadataPath + "/mcp"); got.Code != http.StatusNotFound {
		t.Fatalf("metadata served while Keycloak sign-in is off: %d %s", got.Code, short(got.Body.String()))
	}
	var strangers int
	if err = db.Pool.QueryRow(ctx, `SELECT count(*) FROM users WHERE oidc_subject='early'`).Scan(&strangers); err != nil || strangers != 0 {
		t.Fatalf("a refused token left a user behind: %d %v", strangers, err)
	}

	// On. A client with nothing is told where to sign in — the metadata URL
	// for this resource — and nothing else; a client with a bad token is told
	// only that the token was bad.
	configure(true)
	got := list(nothing)
	if got.code != http.StatusUnauthorized {
		t.Fatalf("no credential got %d: %s", got.code, short(got.body))
	}
	if want := `resource_metadata="https://umm.example/.well-known/oauth-protected-resource/mcp"`; !strings.Contains(got.challenge, want) || strings.Contains(got.challenge, "error=") {
		t.Fatalf("challenge without a token = %q, want it to name %s and no error", got.challenge, want)
	}
	got = list(bearer("not.a.token"))
	if got.code != http.StatusUnauthorized || !strings.Contains(got.challenge, `error="invalid_token"`) || strings.Contains(got.challenge, "signature") {
		t.Fatalf("a bad token got %d %q", got.code, got.challenge)
	}

	// The metadata document: this resource, that issuer, and only the scopes
	// a tool asks for that the administrator also allows on a key.
	var document struct {
		Resource string   `json:"resource"`
		Servers  []string `json:"authorization_servers"`
		Scopes   []string `json:"scopes_supported"`
	}
	response := metadata(auth.ProtectedResourceMetadataPath + "/mcp")
	if response.Code != http.StatusOK {
		t.Fatalf("metadata = %d: %s", response.Code, short(response.Body.String()))
	}
	if err = json.Unmarshal(response.Body.Bytes(), &document); err != nil {
		t.Fatal(err)
	}
	if document.Resource != "https://umm.example/mcp" || len(document.Servers) != 1 || document.Servers[0] != issuer.URL {
		t.Fatalf("metadata names %q at %v, want https://umm.example/mcp at %s", document.Resource, document.Servers, issuer.URL)
	}
	if strings.Join(document.Scopes, " ") != "notes:read spaces:read" {
		t.Fatalf("scopes_supported = %v, want the tool scopes the policy allows", document.Scopes)
	}

	// A token for someone umm has never seen: they are created with the group
	// mapping the login screen would apply, may do what their scopes say, and
	// not what the policy withholds even when the token claims it.
	token := mint(claims{Subject: "alice", Scope: "openid spaces:read notes:write", Groups: []string{"umm-admins"}})
	if got = list(bearer(token)); got.code != http.StatusOK || !strings.Contains(got.body, "capture_thought") {
		t.Fatalf("a good token was refused: %d %q: %s", got.code, got.challenge, short(got.body))
	}
	if got = call("list_spaces", nil, bearer(token)); got.code != http.StatusOK || strings.Contains(got.body, `"isError":true`) {
		t.Fatalf("spaces:read from the token did not open list_spaces: %d %s", got.code, short(got.body))
	}
	got = call("create_note", map[string]any{"space_id": uuid.NewString(), "content": "x"}, bearer(token))
	if !strings.Contains(got.body, "lacks the notes:write scope") {
		t.Fatalf("notes:write is not allowed on a key, yet the token was not told it lacks it: %s", short(got.body))
	}
	got = call("search_notes", map[string]any{"space_id": uuid.NewString(), "query": "x"}, bearer(token))
	if !strings.Contains(got.body, "lacks the notes:read scope") {
		t.Fatalf("a scope the token never claimed was granted: %s", short(got.body))
	}
	var alice store.User
	if alice, err = db.UserByOIDCSubject(ctx, "alice"); err != nil {
		t.Fatalf("alice was not provisioned: %v", err)
	}
	if alice.Role != "admin" || alice.Username != "kc-alice" {
		t.Fatalf("alice provisioned as %s %q, want admin kc-alice from the group mapping", alice.Role, alice.Username)
	}
	var provisioned int
	if err = db.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_logs WHERE action='auth.oidc.mcp.provision' AND actor_id=$1`, alice.ID).Scan(&provisioned); err != nil || provisioned != 1 {
		t.Fatalf("provisioning was audited %d times (%v), want once", provisioned, err)
	}

	// Seen before: what an administrator set since is what stands. The same
	// token that made her an administrator does not make her one again, and
	// once deactivated she is refused, with no row reactivated on the way.
	if _, err = db.Pool.Exec(ctx, `UPDATE users SET role='user' WHERE id=$1`, alice.ID); err != nil {
		t.Fatal(err)
	}
	if got = list(bearer(token)); got.code != http.StatusOK {
		t.Fatalf("alice refused after a role change: %d %s", got.code, short(got.body))
	}
	if alice, err = db.UserByOIDCSubject(ctx, "alice"); err != nil || alice.Role != "user" {
		t.Fatalf("a token reset the role to %q (%v); the administrator's change must stand", alice.Role, err)
	}
	if _, err = db.Pool.Exec(ctx, `UPDATE users SET active=false WHERE id=$1`, alice.ID); err != nil {
		t.Fatal(err)
	}
	if got = list(bearer(token)); got.code != http.StatusUnauthorized || !strings.Contains(got.challenge, `error="invalid_token"`) {
		t.Fatalf("a deactivated person got in with a token: %d %q", got.code, got.challenge)
	}
	if alice, err = db.UserByOIDCSubject(ctx, "alice"); err != nil || alice.Active {
		t.Fatalf("a token reactivated a deactivated account (%v)", err)
	}

	// Only this audience, this issuer, this key set, and not after expiry.
	for name, bad := range map[string]claims{
		"another audience": {Subject: "bob", Scope: "notes:read", Audience: "account"},
		"another issuer":   {Subject: "bob", Scope: "notes:read", Issuer: "https://other.example"},
		"expired":          {Subject: "bob", Scope: "notes:read", Expires: time.Now().Add(-time.Minute)},
	} {
		if got = list(bearer(mint(bad))); got.code != http.StatusUnauthorized || !strings.Contains(got.challenge, `error="invalid_token"`) {
			t.Fatalf("%s: token got %d %q: %s", name, got.code, got.challenge, short(got.body))
		}
	}
	otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	if got = list(bearer(mint(claims{Subject: "bob", Scope: "notes:read", Key: otherKey}))); got.code != http.StatusUnauthorized {
		t.Fatalf("a token signed with an unknown key got %d", got.code)
	}
	if _, err = db.UserByOIDCSubject(ctx, "bob"); err != pgx.ErrNoRows {
		t.Fatalf("a refused token provisioned bob: %v", err)
	}

	// The two credentials that were there before behave as before: a key is
	// accepted, a browser session is not, and neither answer names Keycloak's
	// token as the reason.
	_, secret, err := authService.CreateKey(ctx, adminID, "mcp", []string{"notes:read"}, 30)
	if err != nil {
		t.Fatal(err)
	}
	if got = list(bearer(secret)); got.code != http.StatusOK || !strings.Contains(got.body, "capture_thought") {
		t.Fatalf("a key was refused with Keycloak sign-in on: %d %s", got.code, short(got.body))
	}
	session, err := authService.CreateSession(ctx, adminID, auth.SessionOrigin{UserAgent: "integration-test", ClientIP: "127.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	got = list(func(r *http.Request) { r.AddCookie(&http.Cookie{Name: auth.CookieName, Value: session}) })
	if got.code != http.StatusUnauthorized || strings.Contains(got.challenge, "error=") {
		t.Fatalf("a browser session got %d %q with Keycloak sign-in on", got.code, got.challenge)
	}
}

// isolatedMCPStore migrates umm into a schema of its own so settings written
// here — SSO on, a scope policy — reach no other test.
func isolatedMCPStore(t *testing.T, dsn string) *store.Store {
	t.Helper()
	ctx := context.Background()
	adminConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	adminPool, err := pgxpool.NewWithConfig(ctx, adminConfig)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(adminPool.Close)
	schema := "mcp_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	identifier := pgx.Identifier{schema}.Sanitize()
	if _, err = adminPool.Exec(ctx, "CREATE SCHEMA "+identifier); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = adminPool.Exec(context.Background(), "DROP SCHEMA "+identifier+" CASCADE") })
	testConfig := adminConfig.Copy()
	testConfig.ConnConfig.RuntimeParams["search_path"] = identifier + ", public"
	testPool, err := pgxpool.NewWithConfig(ctx, testConfig)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(testPool.Close)
	db := &store.Store{Pool: testPool}
	if err = db.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	return db
}
