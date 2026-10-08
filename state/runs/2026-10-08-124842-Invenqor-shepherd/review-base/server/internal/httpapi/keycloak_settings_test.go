package httpapi

import (
	"context"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hkjang/invenqor/server/internal/storagetest"

	"github.com/hkjang/invenqor/server/internal/auth"
)

func TestKeycloakAutoConfigureDiscoversAndPersistsMinimumSettings(
	t *testing.T,
) {
	runtime := storagetest.Open(t)
	defer runtime.Close()
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)

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
	defer provider.Close()
	providerURL = provider.URL
	caPEM := string(pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: provider.Certificate().Raw,
	}))

	response := performAuthenticatedJSON(
		t,
		server,
		http.MethodPost,
		"/api/v1/admin/settings/keycloak/auto-configure",
		map[string]any{
			"keycloak_url":    providerURL,
			"realm":           "inventory",
			"client_id":       "invenqor",
			"client_secret":   "minimum-secret",
			"application_url": "https://invenqor.example.test",
			"private_ca_pem":  caPEM,
			"reason":          "minimum setup",
		},
		cookie,
		csrf,
	)
	if response.Code != http.StatusOK {
		t.Fatalf(
			"auto configure status = %d body = %s",
			response.Code,
			response.Body.String(),
		)
	}
	settings, err := server.oidcService.Settings(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !settings.Enabled ||
		settings.EffectiveIssuer() != providerURL+"/realms/inventory" ||
		settings.RedirectURI !=
			"https://invenqor.example.test/api/v1/auth/keycloak/callback" ||
		settings.RoleClaim != "realm_access.roles" ||
		!settings.LastConnectionOK {
		t.Fatalf("stored automatic settings = %#v", settings)
	}
	configured, err := server.oidcService.ClientSecretConfigured(context.Background())
	if err != nil || !configured {
		t.Fatalf("client secret configured = %v, error = %v", configured, err)
	}
}

func TestKeycloakSettingsNormalizeNullableCollections(t *testing.T) {
	runtime := storagetest.Open(t)
	defer runtime.Close()
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)
	if _, err := runtime.DB().Exec(
		`INSERT INTO settings(key,value_json,secret,apply_mode,version)
		 VALUES('auth.keycloak',$1,FALSE,'new_login',1)`,
		`{"scopes":null,"role_mappings":null,"group_mappings":null,
		  "allowed_email_domains":null}`,
	); err != nil {
		t.Fatalf("insert nullable Keycloak settings: %v", err)
	}
	response := performAuthenticatedJSON(
		t,
		server,
		http.MethodGet,
		"/api/v1/admin/settings/keycloak",
		nil,
		cookie,
		csrf,
	)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", response.Code, response.Body.String())
	}
	var payload struct {
		Settings struct {
			Scopes              []string          `json:"scopes"`
			RoleMappings        map[string]string `json:"role_mappings"`
			GroupMappings       map[string]string `json:"group_mappings"`
			AllowedEmailDomains []string          `json:"allowed_email_domains"`
		} `json:"settings"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Settings.Scopes == nil ||
		payload.Settings.RoleMappings == nil ||
		payload.Settings.GroupMappings == nil ||
		payload.Settings.AllowedEmailDomains == nil {
		t.Fatalf("nullable settings were not normalized: %#v", payload.Settings)
	}
	if strings.Join(payload.Settings.Scopes, ",") != "openid,profile,email" {
		t.Fatalf("nullable scopes did not recover safe defaults: %#v", payload.Settings.Scopes)
	}
}

func TestKeycloakSettingsRequireSecretBeforeEnable(t *testing.T) {
	runtime := storagetest.Open(t)
	defer runtime.Close()
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)
	settings := auth.DefaultOIDCSettings()
	settings.Enabled = true
	settings.IssuerURL = "https://keycloak.example.test"
	settings.ClientID = "invenqor"
	settings.RedirectURI = "https://invenqor.example.test/api/v1/auth/keycloak/callback"
	response := performAuthenticatedJSON(
		t,
		server,
		http.MethodPatch,
		"/api/v1/admin/settings/keycloak",
		map[string]any{
			"settings": settings,
			"reason":   "enable SSO",
		},
		cookie,
		csrf,
	)
	if response.Code != http.StatusBadRequest {
		t.Fatalf(
			"status = %d body = %s",
			response.Code,
			response.Body.String(),
		)
	}
	if !strings.Contains(response.Body.String(), "KEYCLOAK_SECRET_REQUIRED") {
		t.Fatalf("response = %s, want KEYCLOAK_SECRET_REQUIRED", response.Body.String())
	}
}

func TestKeycloakSettingsRejectUnknownMappedRole(t *testing.T) {
	runtime := storagetest.Open(t)
	defer runtime.Close()
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)
	settings := auth.DefaultOIDCSettings()
	settings.RoleMappings = map[string]string{"realm-user": "unknown-role"}
	response := performAuthenticatedJSON(
		t,
		server,
		http.MethodPatch,
		"/api/v1/admin/settings/keycloak",
		map[string]any{
			"settings": settings,
			"reason":   "configure mapping",
		},
		cookie,
		csrf,
	)
	if response.Code != http.StatusBadRequest ||
		!strings.Contains(response.Body.String(), "INVALID_KEYCLOAK_ROLE") {
		t.Fatalf(
			"status = %d body = %s, want INVALID_KEYCLOAK_ROLE",
			response.Code,
			response.Body.String(),
		)
	}
}
