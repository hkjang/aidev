package httpapi

import (
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/hkjang/invenqor/server/internal/auth"
	"github.com/hkjang/invenqor/server/internal/storagetest"
)

// configureDiscoverableKeycloak enables Keycloak against a provider that
// answers discovery only, which is all the start leg and a refused callback
// need. It returns a function that flips auto_login.
func configureDiscoverableKeycloak(
	t *testing.T,
	server *Server,
	cookie *http.Cookie,
	csrf string,
) func(autoLogin bool) {
	t.Helper()
	var providerURL string
	provider := httptest.NewTLSServer(http.HandlerFunc(
		func(response http.ResponseWriter, request *http.Request) {
			if request.URL.Path != "/realms/inventory/.well-known/openid-configuration" {
				http.NotFound(response, request)
				return
			}
			response.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(response).Encode(map[string]any{
				"issuer":                                providerURL + "/realms/inventory",
				"authorization_endpoint":                providerURL + "/auth",
				"token_endpoint":                        providerURL + "/token",
				"jwks_uri":                              providerURL + "/jwks",
				"response_types_supported":              []string{"code"},
				"subject_types_supported":               []string{"public"},
				"id_token_signing_alg_values_supported": []string{"RS256"},
			})
		},
	))
	t.Cleanup(provider.Close)
	providerURL = provider.URL
	caPEM := string(pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: provider.Certificate().Raw,
	}))
	response := performAuthenticatedJSON(
		t, server, http.MethodPost,
		"/api/v1/admin/settings/keycloak/auto-configure",
		map[string]any{
			"keycloak_url":    providerURL,
			"realm":           "inventory",
			"client_id":       "invenqor",
			"client_secret":   "minimum-secret",
			"application_url": "https://invenqor.example.test",
			"private_ca_pem":  caPEM,
			"reason":          "silent SSO test",
		},
		cookie, csrf,
	)
	if response.Code != http.StatusOK {
		t.Fatalf("auto configure status = %d body = %s", response.Code, response.Body.String())
	}
	return func(autoLogin bool) {
		t.Helper()
		current := performAuthenticatedJSON(
			t, server, http.MethodGet, "/api/v1/admin/settings/keycloak", nil, cookie, csrf,
		)
		var payload struct {
			Settings auth.OIDCSettings `json:"settings"`
		}
		if err := json.Unmarshal(current.Body.Bytes(), &payload); err != nil {
			t.Fatal(err)
		}
		payload.Settings.AutoLogin = autoLogin
		payload.Settings.PrivateCAPEM = caPEM
		saved := performAuthenticatedJSON(
			t, server, http.MethodPatch, "/api/v1/admin/settings/keycloak",
			map[string]any{"settings": payload.Settings, "reason": "toggle auto_login"},
			cookie, csrf,
		)
		if saved.Code != http.StatusOK {
			t.Fatalf("save auto_login=%v status = %d body = %s", autoLogin, saved.Code, saved.Body.String())
		}
	}
}

func authMethodsAutoLogin(t *testing.T, server *Server) bool {
	t.Helper()
	response := getJSON(t, server, "/api/v1/auth/methods", nil)
	if response.Code != http.StatusOK {
		t.Fatalf("auth methods status = %d body = %s", response.Code, response.Body.String())
	}
	var payload struct {
		Keycloak  bool `json:"keycloak"`
		AutoLogin bool `json:"keycloak_auto_login"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if !payload.Keycloak {
		t.Fatal("Keycloak was not advertised as ready")
	}
	return payload.AutoLogin
}

func startLocation(t *testing.T, server *Server, path string) *url.URL {
	t.Helper()
	response := getJSON(t, server, path, nil)
	if response.Code != http.StatusFound {
		t.Fatalf("start status = %d body = %s", response.Code, response.Body.String())
	}
	location, err := url.Parse(response.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	if location.Query().Has("auth_error") {
		t.Fatalf("start failed: %s", location)
	}
	return location
}

// The whole point of silent SSO is that it never loops. A refused prompt=none
// attempt must come back to the login screen carrying the marker the console
// reads as "do not try again", and nothing about it is a failure worth an
// administrator's attention.
func TestSilentKeycloakLoginIsGatedByAutoLoginAndRefusalDoesNotLoop(t *testing.T) {
	runtime := storagetest.Open(t)
	defer runtime.Close()
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)
	setAutoLogin := configureDiscoverableKeycloak(t, server, cookie, csrf)

	// Default installation: the setting is off, the console is told so, and a
	// visitor appending ?prompt=none gets an ordinary login.
	if authMethodsAutoLogin(t, server) {
		t.Fatal("auth methods advertised auto_login on a default installation")
	}
	location := startLocation(t, server,
		"/api/v1/auth/keycloak/start?prompt=none&return_to=%2F%23%2Fassets")
	if location.Query().Has("prompt") {
		t.Fatalf("prompt=none reached the provider with auto_login off: %s", location)
	}
	// That ordinary flow refused with login_required is a provider failure,
	// as before.
	refusedOrdinary := getJSON(t, server,
		"/api/v1/auth/keycloak/callback?error=login_required&state="+
			url.QueryEscape(location.Query().Get("state")), nil)
	if refusedOrdinary.Code != http.StatusFound ||
		!strings.Contains(refusedOrdinary.Header().Get("Location"), "auth_error=KEYCLOAK_PROVIDER_REJECTED") {
		t.Fatalf("ordinary refusal = %d %s", refusedOrdinary.Code, refusedOrdinary.Header().Get("Location"))
	}

	setAutoLogin(true)
	if !authMethodsAutoLogin(t, server) {
		t.Fatal("auth methods did not advertise auto_login after it was enabled")
	}
	location = startLocation(t, server,
		"/api/v1/auth/keycloak/start?prompt=none&return_to=%2F%23%2Fassets")
	if location.Query().Get("prompt") != "none" {
		t.Fatalf("prompt=none was not sent with auto_login on: %s", location)
	}
	// The login button still starts an ordinary login.
	if plain := startLocation(t, server, "/api/v1/auth/keycloak/start"); plain.Query().Has("prompt") {
		t.Fatalf("the login button sent prompt=none: %s", plain)
	}

	// No provider session: the callback returns login_required.
	state := location.Query().Get("state")
	refused := getJSON(t, server,
		"/api/v1/auth/keycloak/callback?error=login_required"+
			"&error_description=Login+required&state="+url.QueryEscape(state), nil)
	if refused.Code != http.StatusFound {
		t.Fatalf("refused callback status = %d body = %s", refused.Code, refused.Body.String())
	}
	if got := refused.Header().Get("Location"); got != silentLoginRefusedPath {
		t.Fatalf("refused callback redirected to %q, want %q", got, silentLoginRefusedPath)
	}
	if strings.Contains(refused.Header().Get("Set-Cookie"), "invenqor_session") {
		t.Fatal("a refused silent login issued a session cookie")
	}
	// Replaying the refusal is no longer a silent refusal: the state is gone.
	replay := getJSON(t, server,
		"/api/v1/auth/keycloak/callback?error=login_required&state="+url.QueryEscape(state), nil)
	if strings.Contains(replay.Header().Get("Location"), "sso=none") {
		t.Fatalf("a replayed state was accepted as a silent refusal: %s", replay.Header().Get("Location"))
	}

	// The refusal is an ordinary answer, so it is not in the Server log.
	logs := performAuthenticatedJSON(
		t, server, http.MethodGet,
		"/api/v1/admin/diagnostics/logs?component=keycloak", nil, cookie, "",
	)
	if logs.Code != http.StatusOK {
		t.Fatalf("diagnostics status = %d body = %s", logs.Code, logs.Body.String())
	}
	if strings.Contains(logs.Body.String(), "Login required") {
		t.Fatalf("a silent refusal was recorded as a failure: %s", logs.Body.String())
	}

	// A real provider error on a silent flow is still reported.
	location = startLocation(t, server, "/api/v1/auth/keycloak/start?prompt=none")
	denied := getJSON(t, server,
		"/api/v1/auth/keycloak/callback?error=access_denied&state="+
			url.QueryEscape(location.Query().Get("state")), nil)
	if !strings.Contains(denied.Header().Get("Location"), "auth_error=KEYCLOAK_PROVIDER_REJECTED") {
		t.Fatalf("access_denied on a silent flow = %s", denied.Header().Get("Location"))
	}

	// Turning the setting off again downgrades the next silent request.
	setAutoLogin(false)
	if authMethodsAutoLogin(t, server) {
		t.Fatal("auth methods still advertised auto_login after it was disabled")
	}
	if location := startLocation(t, server, "/api/v1/auth/keycloak/start?prompt=none"); location.Query().Has("prompt") {
		t.Fatalf("prompt=none reached the provider after auto_login was disabled: %s", location)
	}
}
