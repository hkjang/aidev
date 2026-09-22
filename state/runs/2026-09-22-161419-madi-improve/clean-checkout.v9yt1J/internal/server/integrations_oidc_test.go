package server

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

func signedOIDCTestToken(t *testing.T, key *rsa.PrivateKey, claims map[string]any) string {
	t.Helper()
	header, _ := json.Marshal(map[string]any{"alg": "RS256", "typ": "JWT", "kid": "madi-test-key"})
	payload, _ := json.Marshal(claims)
	unsigned := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
	digest := sha256.Sum256([]byte(unsigned))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(signature)
}

func TestPostgresOIDCCodePKCENonceAndVerifiedIdentity(t *testing.T) {
	s, server := integrationTestServer(t)
	admin := newIntegrationTestClient(t, server.URL)
	admin.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	wrongKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	var issuer string
	var mu sync.Mutex
	var nonce, challenge string
	verified := true
	omitVerified := false
	badNonce, badAudience, badSignature := false, false, false
	email := "sso@example.test"
	subject := "sso-subject-1"
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			_ = json.NewEncoder(w).Encode(map[string]any{"issuer": issuer, "authorization_endpoint": issuer + "/authorize", "token_endpoint": issuer + "/token", "jwks_uri": issuer + "/jwks", "response_types_supported": []string{"code"}, "subject_types_supported": []string{"public"}, "id_token_signing_alg_values_supported": []string{"RS256"}, "token_endpoint_auth_methods_supported": []string{"client_secret_basic"}})
		case "/jwks":
			_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]any{{"kty": "RSA", "kid": "madi-test-key", "use": "sig", "alg": "RS256", "n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()), "e": "AQAB"}}})
		case "/token":
			_ = r.ParseForm()
			id, secret, _ := r.BasicAuth()
			if id != "madi-client" || secret != "madi-client-secret" {
				t.Error("OIDC client authentication missing")
				http.Error(w, "invalid_client", 401)
				return
			}
			mu.Lock()
			defer mu.Unlock()
			sum := sha256.Sum256([]byte(r.Form.Get("code_verifier")))
			if r.Form.Get("grant_type") != "authorization_code" || r.Form.Get("code") != "test-code" || base64.RawURLEncoding.EncodeToString(sum[:]) != challenge || r.Form.Get("redirect_uri") != server.URL+"/api/v1/auth/oidc/callback" {
				t.Error("OIDC code / PKCE / redirect validation failed")
				http.Error(w, "invalid_grant", 400)
				return
			}
			tokenNonce := nonce
			if badNonce {
				tokenNonce = "wrong-nonce"
			}
			claims := map[string]any{"iss": issuer, "sub": subject, "aud": "madi-client", "exp": time.Now().Add(time.Minute).Unix(), "iat": time.Now().Unix(), "nonce": tokenNonce, "email": email, "name": "SSO 사용자"}
			if !omitVerified {
				claims["email_verified"] = verified
			}
			if badAudience {
				claims["aud"] = "another-client"
			}
			signingKey := key
			if badSignature {
				signingKey = wrongKey // Same kid, but not the key advertised by JWKS.
			}
			idToken := signedOIDCTestToken(t, signingKey, claims)
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "provider-access-token", "token_type": "Bearer", "expires_in": 60, "id_token": idToken})
		default:
			http.NotFound(w, r)
		}
	}))
	defer provider.Close()
	issuer = provider.URL
	admin.request("PUT", "/api/v1/admin/settings", map[string]any{"site_url": server.URL, "oidc_enabled": true, "oidc_issuer": issuer, "oidc_client_id": "madi-client", "oidc_client_secret": "madi-client-secret", "oidc_auto_register": true}, 200)
	cfg := testJSONObject(t, admin.request("GET", "/api/v1/admin/settings", nil, 200))
	if value, ok := cfg["oidc_require_verified_email"].(bool); !ok || value {
		t.Fatalf("email verification must default to boolean false: %v", cfg["oidc_require_verified_email"])
	}
	for _, invalid := range []string{"true", "false"} {
		admin.request("PUT", "/api/v1/admin/settings", map[string]any{"oidc_require_verified_email": invalid}, 400)
	}
	cfg = testJSONObject(t, admin.request("GET", "/api/v1/admin/settings", nil, 200))
	if boolean(cfg, "oidc_require_verified_email") {
		t.Fatal("rejected string setting changed the current policy")
	}
	browser := newIntegrationTestClient(t, server.URL)
	browser.client.CheckRedirect = func(r *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }
	start := func() string {
		t.Helper()
		response, err := browser.client.Get(server.URL + "/api/v1/auth/oidc/start")
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		if response.StatusCode != 302 {
			body, _ := io.ReadAll(response.Body)
			t.Fatalf("OIDC start: %d %s", response.StatusCode, body)
		}
		location, err := url.Parse(response.Header.Get("Location"))
		if err != nil {
			t.Fatal(err)
		}
		q := location.Query()
		if q.Get("code_challenge_method") != "S256" || q.Get("response_type") != "code" || !strings.Contains(q.Get("scope"), "openid") || len(q.Get("state")) < 32 || len(q.Get("nonce")) < 32 {
			t.Fatalf("OIDC authorization parameters: %v", q)
		}
		mu.Lock()
		nonce = q.Get("nonce")
		challenge = q.Get("code_challenge")
		mu.Unlock()
		return "/api/v1/auth/oidc/callback?" + url.Values{"state": {q.Get("state")}, "code": {"test-code"}}.Encode()
	}
	callback := start()
	otherBrowser := newIntegrationTestClient(t, server.URL)
	otherBrowser.request("GET", callback, nil, 400) // A stolen state cannot be used in another browser.
	browser.request("GET", callback, nil, 302)
	me := testJSONObject(t, browser.request("GET", "/api/v1/auth/me", nil, 200))
	if str(me, "email") != "sso@example.test" || str(me, "role") != "editor" {
		t.Fatalf("unexpected SSO user: %v", me)
	}
	browser.request("GET", callback, nil, 400) // Consumed state cannot be replayed.
	setIdentity := func(address, sub string, verifiedEmail, missingClaim bool) {
		mu.Lock()
		email, subject, verified, omitVerified = address, sub, verifiedEmail, missingClaim
		mu.Unlock()
	}
	logout := func() {
		browser.request("POST", "/api/v1/auth/logout", nil, 200)
		browser.request("GET", "/api/v1/auth/me", nil, 401)
	}
	login := func() map[string]any {
		browser.request("GET", start(), nil, 302)
		return testJSONObject(t, browser.request("GET", "/api/v1/auth/me", nil, 200))
	}
	reject := func(status int) {
		browser.request("GET", start(), nil, status)
		browser.request("GET", "/api/v1/auth/me", nil, 401)
	}
	logout()
	// New subjects work with false or absent email_verified under the default
	// policy. Subsequent logins resolve exactly the same provider/issuer/subject.
	for _, missing := range []bool{false, true} {
		address, sub := "unverified@example.test", "unverified-subject"
		if missing {
			address, sub = "missing-claim@example.test", "missing-claim-subject"
		}
		setIdentity(address, sub, false, missing)
		first := login()
		if str(first, "email") != address || str(first, "role") != "editor" {
			t.Fatalf("unexpected newly provisioned identity: %v", first)
		}
		logout()
		second := login()
		if str(second, "id") != str(first, "id") {
			t.Fatalf("same subject provisioned another user: first=%v second=%v", first, second)
		}
		logout()
		var accounts, links, verifiedAudits int
		if err = s.DB.QueryRow(t.Context(), `SELECT (SELECT count(*) FROM users WHERE email=$1),(SELECT count(*) FROM identity_links WHERE provider='oidc' AND issuer=$2 AND subject=$3),(SELECT count(*) FROM audit_logs WHERE action='LOGIN' AND resource=$4 AND details->>'provider'='oidc' AND details->>'email_verified'='false')`, address, issuer, sub, str(first, "id")).Scan(&accounts, &links, &verifiedAudits); err != nil || accounts != 1 || links != 1 || verifiedAudits != 2 {
			t.Fatalf("identity duplication or forged verified claim: accounts=%d links=%d actual-false-audits=%d err=%v", accounts, links, verifiedAudits, err)
		}
	}
	// Strict mode is an actual administrator policy: false and absent claims
	// fail even for an already bound subject, while the signed true claim works.
	admin.request("PUT", "/api/v1/admin/settings", map[string]any{"oidc_require_verified_email": true}, 200)
	for _, missing := range []bool{false, true} {
		setIdentity("sso@example.test", "sso-subject-1", false, missing)
		reject(403)
	}
	setIdentity("sso@example.test", "sso-subject-1", true, false)
	strictUser := login()
	if str(strictUser, "id") != str(me, "id") {
		t.Fatal("strict mode changed the existing identity", strictUser)
	}
	logout()
	admin.request("PUT", "/api/v1/admin/settings", map[string]any{"oidc_require_verified_email": false}, 200)
	// Relaxing the email policy must not relax any cryptographic token check.
	setIdentity("sso@example.test", "sso-subject-1", false, false)
	for _, fault := range []string{"nonce", "audience", "signature"} {
		mu.Lock()
		badNonce, badAudience, badSignature = fault == "nonce", fault == "audience", fault == "signature"
		mu.Unlock()
		reject(401)
	}
	mu.Lock()
	badNonce, badAudience, badSignature = false, false, false
	mu.Unlock()
	// Signature/nonce errors consume their state but do not poison a new login.
	if good := login(); str(good, "id") != str(me, "id") {
		t.Fatal("valid retry did not resolve the original subject", good)
	}
	logout()
	local := testJSONObject(t, admin.request("POST", "/api/v1/admin/users", map[string]any{"email": "local-user@example.test", "name": "로컬 사용자", "role": "editor", "password": "Local-user-Password-2026!"}, 200))
	// Actual false claims must reach identityUser unchanged. Neither role may
	// auto-link, including when verified-email auto-linking is explicitly enabled.
	for _, policy := range []string{"manual", "verified_email_non_admin"} {
		admin.request("PUT", "/api/v1/admin/settings", map[string]any{"identity_link_policy": policy}, 200)
		for _, address := range []string{"local-user@example.test", "admin@example.test"} {
			for _, missing := range []bool{false, true} {
				setIdentity(address, "untrusted-"+address, false, missing)
				reject(403)
			}
		}
	}
	var bound int
	if err = s.DB.QueryRow(t.Context(), `SELECT count(*) FROM identity_links i JOIN users u ON u.id=i.user_id WHERE i.provider='oidc' AND i.issuer=$1 AND u.email IN ('local-user@example.test','admin@example.test')`, issuer).Scan(&bound); err != nil || bound != 0 {
		t.Fatalf("unverified email took over a local account: %d %v", bound, err)
	}
	// Even a verified email may not automatically link a local administrator.
	setIdentity("admin@example.test", "verified-admin-attacker", true, false)
	reject(403)
	// A deliberate administrator-created subject binding remains authoritative;
	// this is not an email match and therefore works without a verified claim.
	admin.request("POST", "/api/v1/admin/identity/links", map[string]any{"user_id": local["id"], "provider": "oidc", "issuer": issuer, "subject": "explicit-local-subject"}, 201)
	setIdentity("local-user@example.test", "explicit-local-subject", false, true)
	if linked := login(); str(linked, "id") != str(local, "id") || str(linked, "role") != "editor" {
		t.Fatal("explicit trusted subject binding was not preserved", linked)
	}
	logout()
	mu.Lock()
	verified = true
	omitVerified = false
	email = "unregistered@example.test"
	subject = "unregistered-subject"
	mu.Unlock()
	admin.request("PUT", "/api/v1/admin/settings", map[string]any{"oidc_auto_register": false}, 200)
	reject(403)
}

func TestOIDCReturnTo(t *testing.T) {
	for value, want := range map[string]string{
		"": "/app", "/": "/", "/app/documents/abc?x=1": "/app/documents/abc?x=1", "/admin": "/admin",
		"//evil.example/x": "/app", "https://evil.example/x": "/app", "app": "/app", "/\\evil.example": "/app", "/x\r\nSet-Cookie: a=b": "/app",
	} {
		if got := oidcReturnTo(value); got != want {
			t.Errorf("oidcReturnTo(%q)=%q want %q", value, got, want)
		}
	}
}

// Silent SSO: prompt=none is forwarded only when auto login is enabled, a
// refusal lands on the login page with the no-retry marker, and a successful
// silent sign-in returns to the deep link it started from.
func TestPostgresOIDCSilentLogin(t *testing.T) {
	_, server := integrationTestServer(t)
	admin := newIntegrationTestClient(t, server.URL)
	admin.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	var issuer string
	var mu sync.Mutex
	var nonce string
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			_ = json.NewEncoder(w).Encode(map[string]any{"issuer": issuer, "authorization_endpoint": issuer + "/authorize", "token_endpoint": issuer + "/token", "jwks_uri": issuer + "/jwks", "response_types_supported": []string{"code"}, "subject_types_supported": []string{"public"}, "id_token_signing_alg_values_supported": []string{"RS256"}})
		case "/jwks":
			_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]any{{"kty": "RSA", "kid": "madi-test-key", "use": "sig", "alg": "RS256", "n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()), "e": "AQAB"}}})
		case "/token":
			mu.Lock()
			tokenNonce := nonce
			mu.Unlock()
			claims := map[string]any{"iss": issuer, "sub": "silent-subject", "aud": "madi-client", "exp": time.Now().Add(time.Minute).Unix(), "iat": time.Now().Unix(), "nonce": tokenNonce, "email": "silent@example.test", "email_verified": true, "name": "조용한 사용자"}
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "provider-access-token", "token_type": "Bearer", "expires_in": 60, "id_token": signedOIDCTestToken(t, key, claims)})
		default:
			http.NotFound(w, r)
		}
	}))
	defer provider.Close()
	issuer = provider.URL
	admin.request("PUT", "/api/v1/admin/settings", map[string]any{"site_url": server.URL, "oidc_enabled": true, "oidc_issuer": issuer, "oidc_client_id": "madi-client", "oidc_client_secret": "madi-client-secret", "oidc_auto_register": true}, 200)
	cfg := testJSONObject(t, admin.request("GET", "/api/v1/admin/settings", nil, 200))
	if value, ok := cfg["oidc_auto_login"].(bool); !ok || value {
		t.Fatalf("auto login must default to boolean false: %v", cfg["oidc_auto_login"])
	}
	admin.request("PUT", "/api/v1/admin/settings", map[string]any{"oidc_auto_login": "true"}, 400)
	public := testJSONObject(t, admin.request("GET", "/api/v1/public", nil, 200))
	if boolean(public, "oidc_auto_login") {
		t.Fatal("public metadata advertised auto login while it is off")
	}
	browser := newIntegrationTestClient(t, server.URL)
	browser.client.CheckRedirect = func(r *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }
	start := func(query string) (authorize url.Values, state string) {
		t.Helper()
		response, err := browser.client.Get(server.URL + "/api/v1/auth/oidc/start" + query)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		if response.StatusCode != 302 {
			body, _ := io.ReadAll(response.Body)
			t.Fatalf("OIDC start: %d %s", response.StatusCode, body)
		}
		location, err := url.Parse(response.Header.Get("Location"))
		if err != nil {
			t.Fatal(err)
		}
		q := location.Query()
		mu.Lock()
		nonce = q.Get("nonce")
		mu.Unlock()
		return q, q.Get("state")
	}
	callback := func(params url.Values) *http.Response {
		t.Helper()
		response, err := browser.client.Get(server.URL + "/api/v1/auth/oidc/callback?" + params.Encode())
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		return response
	}
	// Off (default): the query parameter alone cannot switch to a silent flow,
	// and a provider refusal is the ordinary visible error.
	q, state := start("?prompt=none&return_to=%2Fapp%2Fdocuments%2Fdeep")
	if q.Get("prompt") != "" {
		t.Fatalf("prompt=none forwarded while auto login is off: %v", q)
	}
	if response := callback(url.Values{"state": {state}, "error": {"login_required"}}); response.StatusCode != 401 {
		t.Fatalf("non-silent refusal: %d", response.StatusCode)
	}
	admin.request("PUT", "/api/v1/admin/settings", map[string]any{"oidc_auto_login": true}, 200)
	public = testJSONObject(t, admin.request("GET", "/api/v1/public", nil, 200))
	if !boolean(public, "oidc_auto_login") {
		t.Fatal("public metadata must advertise auto login once enabled")
	}
	// On, without a provider session: login_required sends the browser to the
	// login page carrying the marker that stops another silent attempt.
	q, state = start("?prompt=none&return_to=%2Fapp%2Fdocuments%2Fdeep")
	if q.Get("prompt") != "none" {
		t.Fatalf("prompt=none missing from authorization request: %v", q)
	}
	response := callback(url.Values{"state": {state}, "error": {"login_required"}})
	if response.StatusCode != 302 || response.Header.Get("Location") != "/login?sso=none" {
		t.Fatalf("silent refusal: %d %q", response.StatusCode, response.Header.Get("Location"))
	}
	browser.request("GET", "/api/v1/auth/me", nil, 401)
	// The consumed state cannot be replayed into another redirect.
	if response = callback(url.Values{"state": {state}, "error": {"login_required"}}); response.StatusCode != 400 {
		t.Fatalf("replayed refusal: %d", response.StatusCode)
	}
	// A normal (non-silent) start is unchanged even with auto login on.
	q, _ = start("")
	if q.Get("prompt") != "" {
		t.Fatalf("plain start must not request prompt=none: %v", q)
	}
	// On, with a provider session: the code comes back and the deep link is restored.
	_, state = start("?prompt=none&return_to=%2Fapp%2Fdocuments%2Fdeep%3Ftab%3Dhistory")
	response = callback(url.Values{"state": {state}, "code": {"test-code"}})
	if response.StatusCode != 302 || response.Header.Get("Location") != "/app/documents/deep?tab=history" {
		t.Fatalf("silent success: %d %q", response.StatusCode, response.Header.Get("Location"))
	}
	me := testJSONObject(t, browser.request("GET", "/api/v1/auth/me", nil, 200))
	if str(me, "email") != "silent@example.test" {
		t.Fatalf("unexpected silent SSO user: %v", me)
	}
	browser.request("POST", "/api/v1/auth/logout", nil, 200)
	// Off-origin return targets fall back to the application root.
	for _, unsafe := range []string{"//evil.example/x", "https://evil.example/x", "app"} {
		_, state = start("?return_to=" + url.QueryEscape(unsafe))
		response = callback(url.Values{"state": {state}, "code": {"test-code"}})
		if response.StatusCode != 302 || response.Header.Get("Location") != "/app" {
			t.Fatalf("return_to %q: %d %q", unsafe, response.StatusCode, response.Header.Get("Location"))
		}
		browser.request("POST", "/api/v1/auth/logout", nil, 200)
	}
}
