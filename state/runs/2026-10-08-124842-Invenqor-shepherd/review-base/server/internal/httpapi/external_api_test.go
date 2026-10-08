package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Nothing exercised /api/v1/external/* over an API key at all, so the whole
// credential path a program actually uses - the 401, the throttle, the scope
// check and the name the audit log keeps - was only ever read, never run.
//
// The throttle matters most of the three. It is the only thing standing between
// one runaway integration and every other caller on the deployment, and it is
// the one answer a client is expected to act on rather than log: a 429 with
// Retry-After tells a script to wait, while a 429 without one tells it nothing
// and it retries immediately.
func TestTheExternalAPIThrottlesEachKeySeparately(t *testing.T) {
	runtime := newRuntime(t)
	defer runtime.Close()
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)

	noisy := createAPIKeySecret(t, server, cookie, csrf, "noisy", "assets.read")
	quiet := createAPIKeySecret(t, server, cookie, csrf, "quiet", "assets.read")

	// Two requests is the same rule 600 is, and a test that spent 600 requests
	// proving it would still not be checking anything more.
	clock := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	server.apiRateLimit = newAgentRateLimiter(2, time.Minute)
	server.apiRateLimit.now = func() time.Time { return clock }

	for attempt := range 2 {
		allowed := performWithAPIKey(t, server, http.MethodGet, "/api/v1/external/assets", nil, noisy)
		if allowed.Code != http.StatusOK {
			t.Fatalf("request %d status = %d body = %s", attempt+1, allowed.Code, allowed.Body.String())
		}
	}

	refused := performWithAPIKey(t, server, http.MethodGet, "/api/v1/external/assets", nil, noisy)
	if refused.Code != http.StatusTooManyRequests {
		t.Fatalf("status past the limit = %d body = %s", refused.Code, refused.Body.String())
	}
	if code := errorCode(t, refused); code != "API_RATE_LIMITED" {
		t.Fatalf("error code past the limit = %q", code)
	}
	// Without this a caller has no interval to wait and retries at once, which
	// is the behaviour the limit exists to stop.
	if retry := refused.Header().Get("Retry-After"); retry != "60" {
		t.Fatalf("Retry-After = %q, want the window in seconds", retry)
	}

	// The limiter is keyed by key ID. Were it keyed by anything shared, one
	// integration in a retry loop would answer 429 to every other program on
	// the deployment, and the operator would be looking at the wrong key.
	other := performWithAPIKey(t, server, http.MethodGet, "/api/v1/external/assets", nil, quiet)
	if other.Code != http.StatusOK {
		t.Fatalf("a second key was refused for the first key's traffic: %d %s", other.Code, other.Body.String())
	}

	clock = clock.Add(time.Minute)
	recovered := performWithAPIKey(t, server, http.MethodGet, "/api/v1/external/assets", nil, noisy)
	if recovered.Code != http.StatusOK {
		t.Fatalf("status in a new window = %d body = %s", recovered.Code, recovered.Body.String())
	}
}

// A credential that cannot work must say so as a credential problem, with the
// challenge header, rather than as an empty result a caller reads as "there is
// nothing there".
func TestTheExternalAPIRefusesACredentialItCannotAccept(t *testing.T) {
	runtime := newRuntime(t)
	defer runtime.Close()
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)

	for _, refusal := range []struct {
		name   string
		secret string
	}{
		{"no key at all", ""},
		{"a value that is not one of our keys", "not-an-invenqor-key"},
		{"our prefix with an unknown secret", "ivq_sk_" + "0123456789abcdef0123456789abcdef0123456789"},
	} {
		t.Run(refusal.name, func(t *testing.T) {
			response := performWithAPIKey(
				t, server, http.MethodGet, "/api/v1/external/assets", nil, refusal.secret,
			)
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d body = %s", response.Code, response.Body.String())
			}
			if code := errorCode(t, response); code != "INVALID_API_KEY" {
				t.Fatalf("error code = %q", code)
			}
			if challenge := response.Header().Get("WWW-Authenticate"); challenge == "" {
				t.Fatal("a 401 without WWW-Authenticate leaves the client guessing the scheme")
			}
		})
	}

	// A revoked key is the one a leaked secret becomes, so it has to stop
	// working on the very next request rather than at its expiry.
	secret, id := createAPIKey(t, server, cookie, csrf, "retired", "assets.read")
	if working := performWithAPIKey(
		t, server, http.MethodGet, "/api/v1/external/assets", nil, secret,
	); working.Code != http.StatusOK {
		t.Fatalf("status before revoking = %d body = %s", working.Code, working.Body.String())
	}
	revoke := performAuthenticatedJSON(
		t, server, http.MethodDelete, "/api/v1/admin/api-keys/"+id, nil, cookie, csrf,
	)
	if revoke.Code != http.StatusNoContent {
		t.Fatalf("revoke status = %d body = %s", revoke.Code, revoke.Body.String())
	}
	revoked := performWithAPIKey(t, server, http.MethodGet, "/api/v1/external/assets", nil, secret)
	if revoked.Code != http.StatusUnauthorized {
		t.Fatalf("status after revoking = %d body = %s", revoked.Code, revoked.Body.String())
	}
}

// The scopes chosen when a key is issued are the whole of what it may do. A
// read-only key reaching a write route must be refused by the scope check, and
// what it did do must be recorded against the key rather than against whoever
// happened to own it - otherwise an integration's writes are indistinguishable
// from its owner's own work in the console.
func TestTheExternalAPIHoldsAKeyToItsScopesAndRecordsItByName(t *testing.T) {
	runtime := newRuntime(t)
	defer runtime.Close()
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)

	readOnly := createAPIKeySecret(t, server, cookie, csrf, "read-only", "assets.read")
	refused := performWithAPIKey(
		t, server, http.MethodPost, "/api/v1/external/assets",
		map[string]any{"name": "db-01", "type": "host"}, readOnly,
	)
	if refused.Code != http.StatusForbidden {
		t.Fatalf("write with a read-only key = %d body = %s", refused.Code, refused.Body.String())
	}
	if code := errorCode(t, refused); code != "FORBIDDEN" {
		t.Fatalf("error code for a missing scope = %q", code)
	}

	writer := createAPIKeySecret(t, server, cookie, csrf, "importer", "assets.read", "assets.write")
	created := performWithAPIKey(
		t, server, http.MethodPost, "/api/v1/external/assets",
		map[string]any{"name": "db-01", "type": "host"}, writer,
	)
	if created.Code != http.StatusCreated {
		t.Fatalf("write with a writing key = %d body = %s", created.Code, created.Body.String())
	}

	var actorName string
	if err := runtime.DB().QueryRow(
		`SELECT actor_name FROM audit_logs WHERE action = 'asset.create'`,
	).Scan(&actorName); err != nil {
		t.Fatalf("read the audit entry for the write: %v", err)
	}
	if actorName != "api-key:importer" {
		t.Fatalf("audit actor_name = %q, want the key that made the call", actorName)
	}
}

func createAPIKeySecret(
	t *testing.T,
	server *Server,
	cookie *http.Cookie,
	csrf string,
	name string,
	scopes ...string,
) string {
	t.Helper()
	secret, _ := createAPIKey(t, server, cookie, csrf, name, scopes...)
	return secret
}

func createAPIKey(
	t *testing.T,
	server *Server,
	cookie *http.Cookie,
	csrf string,
	name string,
	scopes ...string,
) (string, string) {
	t.Helper()
	response := performAuthenticatedJSON(
		t, server, http.MethodPost, "/api/v1/admin/api-keys",
		map[string]any{"name": name, "scopes": scopes}, cookie, csrf,
	)
	if response.Code != http.StatusCreated {
		t.Fatalf("create %q status = %d body = %s", name, response.Code, response.Body.String())
	}
	var created struct {
		Key struct {
			ID string `json:"id"`
		} `json:"api_key"`
		Secret string `json:"secret"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created key %q: %v", name, err)
	}
	if created.Secret == "" {
		t.Fatalf("create %q returned no secret", name)
	}
	return created.Secret, created.Key.ID
}

func performWithAPIKey(
	t *testing.T,
	server *Server,
	method string,
	path string,
	body any,
	secret string,
) *httptest.ResponseRecorder {
	t.Helper()
	headers := map[string]string{}
	if secret != "" {
		headers["Authorization"] = "Bearer " + secret
	}
	return performJSON(t, server, method, path, body, headers)
}

func errorCode(t *testing.T, response *httptest.ResponseRecorder) string {
	t.Helper()
	var payload struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode error body %q: %v", response.Body.String(), err)
	}
	return payload.Error.Code
}
