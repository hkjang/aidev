package auth

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hkjang/invenqor/server/internal/bootstrap"
)

func TestOIDCAuthorizationCodePKCEProvisioningAndReplayProtection(t *testing.T) {
	runtime, admin := setupAuthUser(t)
	defer runtime.Close()
	root := filepath.Dir(runtime.SQLitePath())
	bootstrapStore, err := bootstrap.Open(root)
	if err != nil {
		t.Fatalf("bootstrap.Open() error = %v", err)
	}
	localAuth, err := NewService(runtime.DB(), DefaultServiceOptions())
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey() error = %v", err)
	}
	provider := newMockOIDCProvider(t, key)
	defer provider.Close()

	service := NewOIDCService(runtime.DB(), bootstrapStore, localAuth)
	settings := DefaultOIDCSettings()
	settings.Enabled = true
	settings.IssuerURL = provider.URL
	settings.ClientID = "invenqor-test"
	settings.RedirectURI = "https://invenqor.example.test/api/v1/auth/keycloak/callback"
	settings.LogoutRedirectURI = "https://invenqor.example.test/"
	settings.PrivateCAPEM = provider.CAPEM
	settings.RoleMappings = map[string]string{"inventory-view": "viewer"}
	settings.AllowedEmailDomains = []string{"example.test"}
	clientSecret := "oidc-client-secret"
	if err := service.SaveSettings(
		context.Background(),
		settings,
		&clientSecret,
		admin,
		"test configuration",
	); err != nil {
		t.Fatalf("SaveSettings() error = %v", err)
	}
	start, err := service.Start(
		context.Background(),
		"/assets",
		false,
		"192.0.2.20",
		"test-agent",
	)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	authorizationURL, err := url.Parse(start.AuthorizationURL)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}
	query := authorizationURL.Query()
	if query.Get("code_challenge_method") != "S256" ||
		query.Get("code_challenge") == "" ||
		query.Get("nonce") == "" ||
		query.Get("state") == "" {
		t.Fatalf("authorization URL omitted PKCE/state/nonce: %s", start.AuthorizationURL)
	}
	provider.SetAuthorization(
		query.Get("nonce"),
		query.Get("code_challenge"),
	)
	session, returnTo, err := service.Callback(
		context.Background(),
		query.Get("state"),
		"valid-code",
		"192.0.2.20",
		"test-agent",
		"request-oidc",
	)
	if err != nil {
		t.Fatalf("Callback() error = %v", err)
	}
	if returnTo != "/assets" {
		t.Fatalf("Callback() returnTo = %q, want /assets", returnTo)
	}
	if session.User.Username != "oidc.user" ||
		!containsString(session.User.Roles, "viewer") ||
		!containsString(session.User.Permissions, "assets.read") {
		t.Fatalf("provisioned OIDC user = %#v", session.User)
	}
	if session.Token == "" || session.CSRFToken == "" {
		t.Fatal("OIDC callback omitted session tokens")
	}
	logoutURL, err := service.LogoutURL(context.Background(), session.User.ID)
	if err != nil {
		t.Fatalf("LogoutURL() error = %v", err)
	}
	parsedLogoutURL, err := url.Parse(logoutURL)
	if err != nil {
		t.Fatalf("url.Parse(logoutURL) error = %v", err)
	}
	if parsedLogoutURL.Path != "/protocol/openid-connect/logout" ||
		parsedLogoutURL.Query().Get("client_id") != "invenqor-test" ||
		parsedLogoutURL.Query().Get("post_logout_redirect_uri") !=
			settings.LogoutRedirectURI {
		t.Fatalf("LogoutURL() = %q", logoutURL)
	}
	if _, _, err := service.Callback(
		context.Background(),
		query.Get("state"),
		"valid-code",
		"192.0.2.20",
		"test-agent",
		"request-replay",
	); err == nil {
		t.Fatal("Callback() accepted a replayed OIDC state")
	}
	settings.RoleMappings = map[string]string{"inventory-view": "auditor"}
	if err := service.SaveSettings(
		context.Background(),
		settings,
		nil,
		admin,
		"change mapped role",
	); err != nil {
		t.Fatalf("SaveSettings() role update error = %v", err)
	}
	secondStart, err := service.Start(
		context.Background(),
		"/audit",
		false,
		"192.0.2.20",
		"test-agent",
	)
	if err != nil {
		t.Fatalf("second Start() error = %v", err)
	}
	secondURL, err := url.Parse(secondStart.AuthorizationURL)
	if err != nil {
		t.Fatalf("url.Parse(secondStart) error = %v", err)
	}
	secondQuery := secondURL.Query()
	provider.SetAuthorization(
		secondQuery.Get("nonce"),
		secondQuery.Get("code_challenge"),
	)
	secondSession, _, err := service.Callback(
		context.Background(),
		secondQuery.Get("state"),
		"valid-code",
		"192.0.2.20",
		"test-agent",
		"request-role-sync",
	)
	if err != nil {
		t.Fatalf("second Callback() error = %v", err)
	}
	if !containsString(secondSession.User.Roles, "auditor") ||
		containsString(secondSession.User.Roles, "viewer") {
		t.Fatalf("synchronized OIDC roles = %v, want auditor only", secondSession.User.Roles)
	}
	if _, err := runtime.DB().Exec(
		"UPDATE users SET active=FALSE WHERE id=$1",
		secondSession.User.ID,
	); err != nil {
		t.Fatalf("deactivate OIDC user error = %v", err)
	}
	if _, err := service.provisionUser(
		context.Background(),
		settings,
		"subject-123",
		map[string]any{
			"preferred_username": "oidc.user",
			"email":              "oidc.user@example.test",
			"name":               "OIDC User",
			"roles":              []any{"inventory-view"},
		},
	); !errors.Is(err, ErrOIDCUserInactive) {
		t.Fatalf("inactive provisionUser() error = %v, want ErrOIDCUserInactive", err)
	}
	var linkedIdentities int
	if err := runtime.DB().QueryRow(
		"SELECT COUNT(*) FROM external_identities WHERE provider = 'keycloak'",
	).Scan(&linkedIdentities); err != nil {
		t.Fatalf("count external identities error = %v", err)
	}
	if linkedIdentities != 1 {
		t.Fatalf("external identity count = %d, want 1", linkedIdentities)
	}
	configured, err := service.ClientSecretConfigured(context.Background())
	if err != nil || !configured {
		t.Fatalf("ClientSecretConfigured() = %v, %v", configured, err)
	}
	var storedSecret any
	if err := runtime.DB().QueryRow(
		"SELECT value_json FROM settings WHERE key=$1 AND secret=TRUE",
		keycloakClientSecretSettingKey,
	).Scan(&storedSecret); err != nil {
		t.Fatalf("read encrypted client secret setting: %v", err)
	}
	storedSecretJSON, err := jsonBytes(storedSecret)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(storedSecretJSON), clientSecret) {
		t.Fatal("Keycloak client secret was stored as plaintext")
	}
}

func TestOIDCClientSecretIsSharedAcrossPods(t *testing.T) {
	runtime, admin := setupAuthUser(t)
	defer runtime.Close()
	primaryRoot := t.TempDir()
	primaryStore, err := bootstrap.Open(primaryRoot)
	if err != nil {
		t.Fatal(err)
	}
	secondaryStore, err := bootstrap.OpenWithKey(
		t.TempDir(),
		filepath.Join(primaryRoot, "master.key"),
	)
	if err != nil {
		t.Fatal(err)
	}
	localAuth, err := NewService(runtime.DB(), DefaultServiceOptions())
	if err != nil {
		t.Fatal(err)
	}
	primary := NewOIDCService(runtime.DB(), primaryStore, localAuth)
	secondary := NewOIDCService(runtime.DB(), secondaryStore, localAuth)
	secret := "shared-client-secret"
	if err := primary.SaveSettings(
		context.Background(),
		DefaultOIDCSettings(),
		&secret,
		admin,
		"multi-pod secret test",
	); err != nil {
		t.Fatal(err)
	}
	configured, err := secondary.ClientSecretConfigured(context.Background())
	if err != nil || !configured {
		t.Fatalf("secondary ClientSecretConfigured() = %v, %v", configured, err)
	}
	secondarySecret, err := secondary.clientSecret(context.Background())
	if err != nil || secondarySecret != secret {
		t.Fatalf("secondary clientSecret() = %q, %v", secondarySecret, err)
	}
	secondaryValues, err := secondaryStore.Load()
	if err != nil {
		t.Fatal(err)
	}
	if secondaryValues.KeycloakClientSecret != "" {
		t.Fatal("secondary Pod unexpectedly required a local client secret")
	}
}

func TestOIDCLegacyClientSecretMigratesToSharedSetting(t *testing.T) {
	runtime, _ := setupAuthUser(t)
	defer runtime.Close()
	primaryRoot := t.TempDir()
	primaryStore, err := bootstrap.Open(primaryRoot)
	if err != nil {
		t.Fatal(err)
	}
	legacySecret := "legacy-pod-local-secret"
	if err := primaryStore.Save(bootstrap.Values{
		KeycloakClientSecret: legacySecret,
	}); err != nil {
		t.Fatal(err)
	}
	secondaryStore, err := bootstrap.OpenWithKey(
		t.TempDir(),
		filepath.Join(primaryRoot, "master.key"),
	)
	if err != nil {
		t.Fatal(err)
	}
	localAuth, err := NewService(runtime.DB(), DefaultServiceOptions())
	if err != nil {
		t.Fatal(err)
	}
	primary := NewOIDCService(runtime.DB(), primaryStore, localAuth)
	secondary := NewOIDCService(runtime.DB(), secondaryStore, localAuth)
	configured, err := primary.ClientSecretConfigured(context.Background())
	if err != nil || !configured {
		t.Fatalf("legacy migration = %v, %v", configured, err)
	}
	migrated, err := secondary.clientSecret(context.Background())
	if err != nil || migrated != legacySecret {
		t.Fatalf("secondary migrated secret = %q, %v", migrated, err)
	}
}

func TestOIDCSettingsValidationAndRoleMapping(t *testing.T) {
	settings := DefaultOIDCSettings()
	settings.Enabled = true
	settings.IssuerURL = "http://insecure.example.test"
	settings.ClientID = "client"
	settings.RedirectURI = "https://invenqor.example.test/callback"
	if err := settings.Validate(); err == nil {
		t.Fatal("Validate() accepted an insecure issuer")
	}
	settings.IssuerURL = "https://keycloak.example.test"
	settings.Realm = "inventory"
	if issuer := settings.EffectiveIssuer(); issuer != "https://keycloak.example.test/realms/inventory" {
		t.Fatalf("EffectiveIssuer() = %q", issuer)
	}
	settings.RoleMappings = map[string]string{"kc-admin": "asset_manager"}
	settings.GroupMappings = map[string]string{"/audit": "auditor"}
	settings.RoleClaim = "realm_access.roles"
	roles := mappedRoles(settings, map[string]any{
		"realm_access": map[string]any{"roles": []any{"kc-admin"}},
		"groups":       []any{"/audit"},
	})
	if strings.Join(roles, ",") != "asset_manager,auditor" {
		t.Fatalf("mappedRoles() = %v", roles)
	}
	if !allowedEmail("user@example.test", []string{"example.test"}) ||
		allowedEmail("user@other.test", []string{"example.test"}) {
		t.Fatal("allowedEmail() domain policy mismatch")
	}
}

func TestOIDCURLPolicyRejectsAmbiguousIssuerAndExternalPlaintext(t *testing.T) {
	settings := DefaultOIDCSettings()
	settings.Enabled = true
	settings.ClientID = "invenqor"
	settings.RedirectURI = "https://invenqor.example.test/callback"
	for _, issuer := range []string{
		"https://operator@keycloak.example.test/realms/inventory",
		"https://keycloak.example.test/realms/inventory?tenant=other",
		"https://keycloak.example.test/realms/inventory#other",
	} {
		settings.IssuerURL = issuer
		if err := settings.Validate(); err == nil {
			t.Fatalf("Validate() accepted ambiguous issuer %q", issuer)
		}
	}

	settings.IssuerURL = "https://keycloak.example.test/realms/inventory"
	settings.RedirectURI = "http://invenqor.example.test/callback"
	if err := settings.Validate(); err == nil {
		t.Fatal("Validate() accepted a plaintext external redirect")
	}
	settings.RedirectURI = "https://invenqor.example.test/callback#fragment"
	if err := settings.Validate(); err == nil {
		t.Fatal("Validate() accepted a redirect URI fragment")
	}
	settings.RedirectURI = "http://127.0.0.1:7070/callback"
	settings.LogoutRedirectURI = "http://invenqor.example.test/"
	if err := settings.Validate(); err == nil {
		t.Fatal("Validate() accepted a plaintext external logout redirect")
	}
	settings.LogoutRedirectURI = "http://[::1]:7070/"
	if err := settings.Validate(); err != nil {
		t.Fatalf("Validate() rejected loopback HTTP endpoints: %v", err)
	}

	for raw, want := range map[string]bool{
		"https://inventory.example.test": true,
		"http://localhost:7070":          true,
		"http://127.0.0.1:7070":          true,
		"http://[::1]:7070":              true,
		"http://inventory.example.test":  false,
		"https://user@inventory.example": false,
	} {
		endpoint, err := url.Parse(raw)
		if err != nil {
			t.Fatal(err)
		}
		if got := secureBrowserEndpoint(endpoint); got != want {
			t.Fatalf("secureBrowserEndpoint(%q) = %v, want %v", raw, got, want)
		}
	}
}

func TestAutomaticOIDCSettingsDiscoverAndDeriveApplicationEndpoints(
	t *testing.T,
) {
	runtime, _ := setupAuthUser(t)
	defer runtime.Close()
	bootstrapStore, err := bootstrap.Open(filepath.Dir(runtime.SQLitePath()))
	if err != nil {
		t.Fatal(err)
	}
	localAuth, err := NewService(runtime.DB(), DefaultServiceOptions())
	if err != nil {
		t.Fatal(err)
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	provider := newMockOIDCProvider(t, key)
	defer provider.Close()
	service := NewOIDCService(runtime.DB(), bootstrapStore, localAuth)
	settings, err := service.AutomaticSettings(
		context.Background(),
		OIDCAutoConfig{
			KeycloakURL:    provider.URL,
			ClientID:       "invenqor",
			ApplicationURL: "https://inventory.example.test/",
			PrivateCAPEM:   provider.CAPEM,
		},
	)
	if err != nil {
		t.Fatalf("AutomaticSettings() error = %v", err)
	}
	if !settings.Enabled ||
		settings.EffectiveIssuer() != provider.URL ||
		settings.RedirectURI !=
			"https://inventory.example.test/api/v1/auth/keycloak/callback" ||
		settings.LogoutRedirectURI != "https://inventory.example.test/" ||
		!settings.LastConnectionOK ||
		settings.LastConnectionTestAt == nil {
		t.Fatalf("automatic settings = %#v", settings)
	}
}

func TestOIDCSettingsRequireSecretAndKnownMappedRoles(t *testing.T) {
	runtime, admin := setupAuthUser(t)
	defer runtime.Close()
	bootstrapStore, err := bootstrap.Open(filepath.Dir(runtime.SQLitePath()))
	if err != nil {
		t.Fatalf("bootstrap.Open() error = %v", err)
	}
	localAuth, err := NewService(runtime.DB(), DefaultServiceOptions())
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	service := NewOIDCService(runtime.DB(), bootstrapStore, localAuth)
	settings := DefaultOIDCSettings()
	settings.Enabled = true
	settings.IssuerURL = "https://keycloak.example.test"
	settings.ClientID = "invenqor"
	settings.RedirectURI = "https://invenqor.example.test/api/v1/auth/keycloak/callback"
	if err := service.SaveSettings(
		context.Background(),
		settings,
		nil,
		admin,
		"missing secret test",
	); !errors.Is(err, ErrOIDCSecret) {
		t.Fatalf("SaveSettings() error = %v, want ErrOIDCSecret", err)
	}
	secret := "client-secret"
	settings.RoleMappings = map[string]string{"realm-user": "not-a-role"}
	if err := service.SaveSettings(
		context.Background(),
		settings,
		&secret,
		admin,
		"invalid mapping test",
	); !errors.Is(err, ErrOIDCRole) {
		t.Fatalf("SaveSettings() error = %v, want ErrOIDCRole", err)
	}
}

type mockOIDCProvider struct {
	*httptest.Server
	URL           string
	CAPEM         string
	key           *rsa.PrivateKey
	mutex         sync.Mutex
	nonce         string
	codeChallenge string
}

func newMockOIDCProvider(t *testing.T, key *rsa.PrivateKey) *mockOIDCProvider {
	t.Helper()
	provider := &mockOIDCProvider{key: key}
	server := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		provider.ServeHTTP(response, request)
	}))
	provider.Server = server
	provider.URL = server.URL
	provider.CAPEM = string(pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: server.Certificate().Raw,
	}))
	return provider
}

func (provider *mockOIDCProvider) SetAuthorization(nonce, challenge string) {
	provider.mutex.Lock()
	defer provider.mutex.Unlock()
	provider.nonce = nonce
	provider.codeChallenge = challenge
}

func (provider *mockOIDCProvider) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	switch request.URL.Path {
	case "/.well-known/openid-configuration":
		writeMockJSON(response, map[string]any{
			"issuer":                                provider.URL,
			"authorization_endpoint":                provider.URL + "/auth",
			"token_endpoint":                        provider.URL + "/token",
			"jwks_uri":                              provider.URL + "/jwks",
			"response_types_supported":              []string{"code"},
			"subject_types_supported":               []string{"public"},
			"id_token_signing_alg_values_supported": []string{"RS256"},
			"code_challenge_methods_supported":      []string{"S256"},
		})
	case "/jwks":
		exponent := big.NewInt(int64(provider.key.PublicKey.E)).Bytes()
		writeMockJSON(response, map[string]any{
			"keys": []map[string]string{{
				"kty": "RSA",
				"kid": "test-key",
				"use": "sig",
				"alg": "RS256",
				"n":   base64.RawURLEncoding.EncodeToString(provider.key.PublicKey.N.Bytes()),
				"e":   base64.RawURLEncoding.EncodeToString(exponent),
			}},
		})
	case "/token":
		if err := request.ParseForm(); err != nil {
			http.Error(response, "invalid form", http.StatusBadRequest)
			return
		}
		provider.mutex.Lock()
		nonce := provider.nonce
		challenge := provider.codeChallenge
		provider.mutex.Unlock()
		actualChallenge := sha256.Sum256([]byte(request.Form.Get("code_verifier")))
		if request.Form.Get("code") != "valid-code" ||
			base64.RawURLEncoding.EncodeToString(actualChallenge[:]) != challenge {
			writeMockJSONStatus(response, http.StatusBadRequest, map[string]string{
				"error": "invalid_grant",
			})
			return
		}
		now := time.Now().UTC()
		idToken := signTestJWT(provider.key, map[string]any{
			"iss":                provider.URL,
			"sub":                "subject-123",
			"aud":                "invenqor-test",
			"iat":                now.Unix(),
			"exp":                now.Add(5 * time.Minute).Unix(),
			"nonce":              nonce,
			"preferred_username": "oidc.user",
			"email":              "oidc.user@example.test",
			"name":               "OIDC User",
			"roles":              []string{"inventory-view"},
		})
		writeMockJSON(response, map[string]any{
			"access_token": "access-token",
			"token_type":   "Bearer",
			"expires_in":   300,
			"id_token":     idToken,
		})
	default:
		http.NotFound(response, request)
	}
}

func signTestJWT(key *rsa.PrivateKey, claims map[string]any) string {
	header, _ := json.Marshal(map[string]string{
		"alg": "RS256",
		"kid": "test-key",
		"typ": "JWT",
	})
	payload, _ := json.Marshal(claims)
	signingInput := base64.RawURLEncoding.EncodeToString(header) + "." +
		base64.RawURLEncoding.EncodeToString(payload)
	digest := sha256.Sum256([]byte(signingInput))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		panic(fmt.Sprintf("sign test JWT: %v", err))
	}
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(signature)
}

func writeMockJSON(response http.ResponseWriter, payload any) {
	writeMockJSONStatus(response, http.StatusOK, payload)
}

func writeMockJSONStatus(response http.ResponseWriter, status int, payload any) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(payload)
}
