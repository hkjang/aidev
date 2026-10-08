package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"net/url"
	"path/filepath"
	"testing"

	"github.com/hkjang/invenqor/server/internal/bootstrap"
	"github.com/hkjang/invenqor/server/internal/storage"
)

func newConfiguredOIDCService(
	t *testing.T,
) (*OIDCService, *mockOIDCProvider, OIDCSettings, User, *storage.Runtime) {
	t.Helper()
	runtime, admin := setupAuthUser(t)
	bootstrapStore, err := bootstrap.Open(filepath.Dir(runtime.SQLitePath()))
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
	t.Cleanup(provider.Close)

	service := NewOIDCService(runtime.DB(), bootstrapStore, localAuth)
	settings := DefaultOIDCSettings()
	settings.Enabled = true
	settings.IssuerURL = provider.URL
	settings.ClientID = "invenqor-test"
	settings.RedirectURI = "https://invenqor.example.test/api/v1/auth/keycloak/callback"
	settings.LogoutRedirectURI = "https://invenqor.example.test/"
	settings.PrivateCAPEM = provider.CAPEM
	secret := "oidc-client-secret"
	if err := service.SaveSettings(
		context.Background(), settings, &secret, admin, "failure path test",
	); err != nil {
		t.Fatalf("SaveSettings() error = %v", err)
	}
	return service, provider, settings, admin, runtime
}

func authorize(
	t *testing.T,
	service *OIDCService,
	provider *mockOIDCProvider,
) url.Values {
	t.Helper()
	start, err := service.Start(context.Background(), "/", false, "192.0.2.20", "test-agent")
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	parsed, err := url.Parse(start.AuthorizationURL)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}
	query := parsed.Query()
	provider.SetAuthorization(query.Get("nonce"), query.Get("code_challenge"))
	return query
}

// A directory account whose username already belongs to a local account cannot
// be linked automatically without risking a takeover, so the failure has to name
// the collision instead of surfacing as a generic login error.
func TestKeycloakLoginReportsAUsernameHeldByALocalAccount(t *testing.T) {
	service, provider, _, _, runtime := newConfiguredOIDCService(t)
	defer runtime.Close()

	if _, err := runtime.DB().Exec(
		`INSERT INTO users(id,username,normalized_username,display_name,email,
		                   active,super_admin)
		 VALUES('11111111-1111-1111-1111-111111111111','oidc.user','oidc.user',
		        'Local Holder','local@example.test',TRUE,FALSE)`,
	); err != nil {
		t.Fatalf("seed conflicting local user error = %v", err)
	}

	query := authorize(t, service, provider)
	_, _, err := service.Callback(
		context.Background(),
		query.Get("state"),
		"valid-code",
		"192.0.2.20",
		"test-agent",
		"request-conflict",
	)
	if !errors.Is(err, ErrOIDCUsernameTaken) {
		t.Fatalf("Callback() error = %v, want ErrOIDCUsernameTaken", err)
	}
	// No half-created account may remain behind the failure.
	var linked int
	if err := runtime.DB().QueryRow(
		"SELECT COUNT(*) FROM external_identities WHERE provider='keycloak'",
	).Scan(&linked); err != nil {
		t.Fatal(err)
	}
	if linked != 0 {
		t.Fatalf("external identity count = %d, want 0", linked)
	}
}

// The single-use flow has to be claimed before any account or session exists,
// otherwise a replayed callback leaves an unreferenced session behind.
func TestKeycloakReplayLeavesNoOrphanSession(t *testing.T) {
	service, provider, _, _, runtime := newConfiguredOIDCService(t)
	defer runtime.Close()

	query := authorize(t, service, provider)
	session, _, err := service.Callback(
		context.Background(),
		query.Get("state"),
		"valid-code",
		"192.0.2.20",
		"test-agent",
		"request-first",
	)
	if err != nil {
		t.Fatalf("Callback() error = %v", err)
	}
	var before int
	if err := runtime.DB().QueryRow(
		"SELECT COUNT(*) FROM sessions WHERE user_id=$1", session.User.ID,
	).Scan(&before); err != nil {
		t.Fatal(err)
	}
	if _, _, err := service.Callback(
		context.Background(),
		query.Get("state"),
		"valid-code",
		"192.0.2.20",
		"test-agent",
		"request-replay",
	); !errors.Is(err, ErrOIDCFlow) {
		t.Fatalf("replayed Callback() error = %v, want ErrOIDCFlow", err)
	}
	var after int
	if err := runtime.DB().QueryRow(
		"SELECT COUNT(*) FROM sessions WHERE user_id=$1", session.User.ID,
	).Scan(&after); err != nil {
		t.Fatal(err)
	}
	if after != before {
		t.Fatalf("session count grew from %d to %d on a replayed callback", before, after)
	}
}

// Abandoned logins are normal, so finished and expired flow rows must not
// accumulate forever in the shared database.
func TestKeycloakStartPrunesFinishedAndExpiredFlows(t *testing.T) {
	service, provider, _, _, runtime := newConfiguredOIDCService(t)
	defer runtime.Close()

	query := authorize(t, service, provider)
	if _, _, err := service.Callback(
		context.Background(),
		query.Get("state"),
		"valid-code",
		"192.0.2.20",
		"test-agent",
		"request-consumed",
	); err != nil {
		t.Fatalf("Callback() error = %v", err)
	}
	// An abandoned flow that has already expired.
	if _, err := runtime.DB().Exec(
		`INSERT INTO oidc_flows(
			id,state_hash,nonce_hash,pkce_verifier,redirect_uri,return_to,
			expires_at
		 ) VALUES('22222222-2222-2222-2222-222222222222','stale','stale','stale',
		          'https://invenqor.example.test/cb','/','2000-01-01T00:00:00Z')`,
	); err != nil {
		t.Fatalf("seed stale flow error = %v", err)
	}
	if _, err := service.Start(
		context.Background(), "/", false, "192.0.2.20", "test-agent",
	); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	var remaining int
	if err := runtime.DB().QueryRow(
		"SELECT COUNT(*) FROM oidc_flows",
	).Scan(&remaining); err != nil {
		t.Fatal(err)
	}
	if remaining != 1 {
		t.Fatalf("oidc_flows rows = %d, want only the pending flow", remaining)
	}
}

// The natural order is to test the connection before enabling the provider, so
// a disabled configuration must still be checked for the fields discovery needs.
func TestConnectionTestValidatesADisabledConfiguration(t *testing.T) {
	service, provider, settings, _, runtime := newConfiguredOIDCService(t)
	defer runtime.Close()

	settings.Enabled = false
	settings.IssuerURL = ""
	err := service.TestConnection(context.Background(), settings)
	if err == nil {
		t.Fatal("TestConnection() accepted an empty issuer")
	}
	if errors.Is(err, ErrOIDCUnreachable) {
		t.Fatalf("empty issuer reported as unreachable: %v", err)
	}

	settings.IssuerURL = provider.URL
	if err := service.TestConnection(context.Background(), settings); err != nil {
		t.Fatalf("TestConnection() on a disabled but complete configuration = %v", err)
	}

	settings.IssuerURL = "https://127.0.0.1:1"
	settings.PrivateCAPEM = ""
	if err := service.TestConnection(
		context.Background(), settings,
	); !errors.Is(err, ErrOIDCUnreachable) {
		t.Fatalf("TestConnection() error = %v, want ErrOIDCUnreachable", err)
	}
}
