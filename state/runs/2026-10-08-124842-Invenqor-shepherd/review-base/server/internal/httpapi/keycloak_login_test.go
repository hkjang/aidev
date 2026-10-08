package httpapi

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func enableKeycloakWithoutSecret(t *testing.T, server *Server) {
	t.Helper()
	// Written directly because SaveSettings correctly refuses to enable the
	// provider without a secret; this reproduces a configuration that reached
	// that state before the secret was rotated away.
	settings := map[string]any{
		"enabled":        true,
		"issuer_url":     "https://keycloak.example.test",
		"realm":          "inventory",
		"client_id":      "invenqor",
		"redirect_uri":   "https://invenqor.example.test/api/v1/auth/keycloak/callback",
		"scopes":         []string{"openid", "profile", "email"},
		"username_claim": "preferred_username",
	}
	encoded, err := json.Marshal(settings)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := server.database.DB().Exec(
		`INSERT INTO settings(key,value_json,secret,apply_mode,version,updated_by)
		 VALUES('auth.keycloak',$1,FALSE,'new_login',1,NULL)
		 ON CONFLICT(key) DO UPDATE SET value_json=excluded.value_json`,
		string(encoded),
	); err != nil {
		t.Fatalf("seed Keycloak settings error = %v", err)
	}
}

// The login page must not offer a button that cannot work: without a client
// secret the redirect always fails.
func TestAuthMethodsHidesKeycloakUntilTheSecretIsConfigured(t *testing.T) {
	server := testServer(t, newRuntime(t))
	enableKeycloakWithoutSecret(t, server)

	response := getJSON(t, server, "/api/v1/auth/methods", nil)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", response.Code, response.Body.String())
	}
	var payload struct {
		Local              bool `json:"local"`
		Keycloak           bool `json:"keycloak"`
		KeycloakEnabled    bool `json:"keycloak_enabled"`
		KeycloakIncomplete bool `json:"keycloak_incomplete"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if !payload.Local {
		t.Fatal("local login was not advertised")
	}
	if payload.Keycloak {
		t.Fatal("Keycloak was advertised without a client secret")
	}
	if !payload.KeycloakEnabled || !payload.KeycloakIncomplete {
		t.Fatalf("incomplete configuration was not reported: %+v", payload)
	}
}

// These endpoints are reached by a top-level browser navigation, so a JSON error
// body would leave the user on a blank page. They must redirect with a code.
func TestKeycloakBrowserFailuresRedirectWithADiagnosableCode(t *testing.T) {
	runtime := newRuntime(t)
	server := testServer(t, runtime)

	cases := []struct {
		name string
		path string
		code string
	}{
		{
			name: "disabled provider",
			path: "/api/v1/auth/keycloak/start",
			code: "KEYCLOAK_DISABLED",
		},
		{
			name: "provider rejected the request",
			path: "/api/v1/auth/keycloak/callback?error=access_denied" +
				"&error_description=Consent+denied",
			code: "KEYCLOAK_PROVIDER_REJECTED",
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			response := getJSON(t, server, testCase.path, nil)
			if response.Code != http.StatusFound {
				t.Fatalf(
					"status = %d body = %s, want a redirect",
					response.Code,
					response.Body.String(),
				)
			}
			location, err := url.Parse(response.Header().Get("Location"))
			if err != nil {
				t.Fatal(err)
			}
			if location.Query().Get("auth_error") != testCase.code {
				t.Fatalf(
					"redirect = %q, want auth_error=%s",
					response.Header().Get("Location"),
					testCase.code,
				)
			}
			if location.Query().Get("request_id") == "" {
				t.Fatal("redirect omitted the request identifier")
			}
		})
	}

	// An identity provider that is configured but down used to produce an opaque
	// HTTP 500. It has to name the unreachable issuer instead.
	enableKeycloakWithoutSecret(t, server)
	values, err := server.bootstrapStore.Load()
	if err != nil {
		t.Fatal(err)
	}
	values.KeycloakClientSecret = "configured-secret"
	if err := server.bootstrapStore.Save(values); err != nil {
		t.Fatal(err)
	}
	unreachable := getJSON(t, server, "/api/v1/auth/keycloak/start", nil)
	if unreachable.Code != http.StatusFound {
		t.Fatalf(
			"unreachable issuer status = %d body = %s, want a redirect",
			unreachable.Code,
			unreachable.Body.String(),
		)
	}
	if !strings.Contains(
		unreachable.Header().Get("Location"),
		"auth_error=KEYCLOAK_UNREACHABLE",
	) {
		t.Fatalf("redirect = %q", unreachable.Header().Get("Location"))
	}

	// Each failure has to be visible to an administrator with its remedy.
	cookie, _ := authenticateInitialAdmin(t, server, runtime)
	logs := performAuthenticatedJSON(
		t, server, http.MethodGet,
		"/api/v1/admin/diagnostics/logs?component=keycloak",
		nil, cookie, "",
	)
	if logs.Code != http.StatusOK {
		t.Fatalf("diagnostics status = %d body = %s", logs.Code, logs.Body.String())
	}
	body := logs.Body.String()
	for _, code := range []string{
		"KEYCLOAK_DISABLED",
		"KEYCLOAK_PROVIDER_REJECTED",
		"KEYCLOAK_UNREACHABLE",
	} {
		if !strings.Contains(body, code) {
			t.Fatalf("diagnostics omitted %s: %s", code, body)
		}
	}
	if !strings.Contains(body, "remediation") {
		t.Fatalf("diagnostics omitted the remediation text: %s", body)
	}
}

// Every code the login handlers can return must carry an action.
func TestKeycloakGuidanceCoversEveryLoginFailureCode(t *testing.T) {
	codes := []string{
		"KEYCLOAK_DISABLED",
		"KEYCLOAK_SECRET_REQUIRED",
		"KEYCLOAK_FLOW_EXPIRED",
		"KEYCLOAK_NONCE_MISMATCH",
		"KEYCLOAK_EMAIL_DOMAIN_REJECTED",
		"KEYCLOAK_PROVISIONING_DISABLED",
		"KEYCLOAK_USERNAME_UNUSABLE",
		"KEYCLOAK_USERNAME_CONFLICT",
		"KEYCLOAK_USER_INACTIVE",
		"KEYCLOAK_ROLE_MISSING",
		"KEYCLOAK_UNREACHABLE",
		"KEYCLOAK_PROVIDER_REJECTED",
		"KEYCLOAK_LOGIN_FAILED",
	}
	for _, code := range codes {
		if _, found := keycloakGuidance[code]; !found {
			t.Errorf("no operator guidance for %s", code)
		}
	}
}
