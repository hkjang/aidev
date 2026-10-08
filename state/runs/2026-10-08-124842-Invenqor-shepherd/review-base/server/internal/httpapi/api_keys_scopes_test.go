package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// A key with no scopes still authenticates, so every request it makes is
// refused by the scope check rather than by the credential check. Creating
// such a key has always been rejected; these pin the paths that could reach
// the same state on a key that already exists.
func TestAPIKeyKeepsAtLeastOneScope(t *testing.T) {
	runtime := newRuntime(t)
	defer runtime.Close()
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)

	create := performAuthenticatedJSON(
		t, server, http.MethodPost, "/api/v1/admin/api-keys",
		map[string]any{"name": "reporting", "scopes": []string{"assets.read"}},
		cookie, csrf,
	)
	if create.Code != http.StatusCreated {
		t.Fatalf("create status = %d body = %s", create.Code, create.Body.String())
	}
	var created struct {
		Key struct {
			ID string `json:"id"`
		} `json:"api_key"`
	}
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created key: %v", err)
	}

	// The console offers one checkbox per scope, so clearing the last one is a
	// single click away, and the DELETE it sends used to answer 200.
	removed := performAuthenticatedJSON(
		t, server, http.MethodDelete,
		"/api/v1/admin/api-keys/"+created.Key.ID+"/scopes/assets.read",
		nil, cookie, csrf,
	)
	assertScopesRequired(t, "remove the last scope", removed)

	replaced := performAuthenticatedJSON(
		t, server, http.MethodPatch,
		"/api/v1/admin/api-keys/"+created.Key.ID,
		map[string]any{"scopes": []string{}},
		cookie, csrf,
	)
	assertScopesRequired(t, "replace every scope", replaced)

	rejected := performAuthenticatedJSON(
		t, server, http.MethodPost, "/api/v1/admin/api-keys",
		map[string]any{"name": "scopeless", "scopes": []string{}},
		cookie, csrf,
	)
	assertScopesRequired(t, "create without a scope", rejected)

	fetched := performAuthenticatedJSON(
		t, server, http.MethodGet,
		"/api/v1/admin/api-keys/"+created.Key.ID, nil, cookie, csrf,
	)
	if fetched.Code != http.StatusOK {
		t.Fatalf("get status = %d body = %s", fetched.Code, fetched.Body.String())
	}
	var fetchedKey struct {
		Key struct {
			Scopes []string `json:"scopes"`
		} `json:"api_key"`
	}
	if err := json.Unmarshal(fetched.Body.Bytes(), &fetchedKey); err != nil {
		t.Fatalf("decode key: %v", err)
	}
	if len(fetchedKey.Key.Scopes) != 1 || fetchedKey.Key.Scopes[0] != "assets.read" {
		t.Fatalf("scopes after rejected removals = %v", fetchedKey.Key.Scopes)
	}
}

// Every path that can empty a scope list reports the same code, so a console
// or script does not have to tell three answers to one rule apart.
func assertScopesRequired(
	t *testing.T,
	attempt string,
	response *httptest.ResponseRecorder,
) {
	t.Helper()
	if response.Code != http.StatusBadRequest {
		t.Fatalf(
			"%s status = %d body = %s",
			attempt, response.Code, response.Body.String(),
		)
	}
	var payload struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode %s error: %v", attempt, err)
	}
	if payload.Error.Code != "INVALID_SCOPES" {
		t.Fatalf("%s error code = %q", attempt, payload.Error.Code)
	}
}
