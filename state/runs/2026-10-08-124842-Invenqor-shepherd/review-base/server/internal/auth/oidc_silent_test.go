package auth

import (
	"context"
	"net/url"
	"testing"
)

func startQuery(t *testing.T, service *OIDCService, returnTo string, silent bool) (OIDCStart, url.Values) {
	t.Helper()
	start, err := service.Start(context.Background(), returnTo, silent, "192.0.2.20", "test-agent")
	if err != nil {
		t.Fatalf("Start(silent=%v) error = %v", silent, err)
	}
	parsed, err := url.Parse(start.AuthorizationURL)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}
	return start, parsed.Query()
}

// A silent attempt is a redirect that happens without anyone clicking. Where it
// can happen must be decided by the administrator, not by whoever appends
// ?prompt=none to the address, so the default configuration downgrades it.
func TestOIDCSilentStartIsIgnoredUntilAutoLoginIsEnabled(t *testing.T) {
	service, _, settings, admin, _ := newConfiguredOIDCService(t)
	if settings.AutoLogin {
		t.Fatal("auto_login defaulted to on")
	}

	start, query := startQuery(t, service, "/", true)
	if start.Silent || query.Has("prompt") {
		t.Fatalf("prompt=none was sent with auto_login off: %s", start.AuthorizationURL)
	}

	settings.AutoLogin = true
	if err := service.SaveSettings(
		context.Background(), settings, nil, admin, "enable silent SSO",
	); err != nil {
		t.Fatalf("SaveSettings() error = %v", err)
	}
	start, query = startQuery(t, service, "/", true)
	if !start.Silent || query.Get("prompt") != "none" {
		t.Fatalf("prompt=none missing with auto_login on: %s", start.AuthorizationURL)
	}
	// The button on the login screen still starts an ordinary login.
	start, query = startQuery(t, service, "/", false)
	if start.Silent || query.Has("prompt") {
		t.Fatalf("an ordinary login sent prompt=none: %s", start.AuthorizationURL)
	}
}

// login_required is the provider's ordinary answer to prompt=none when it has
// no session. It is recognised only for a flow this server started silently,
// and only once: the state is consumed so it cannot be replayed.
func TestOIDCRefusedSilentlyRecognisesOnlyASilentFlowsRefusal(t *testing.T) {
	service, _, settings, admin, _ := newConfiguredOIDCService(t)
	settings.AutoLogin = true
	if err := service.SaveSettings(
		context.Background(), settings, nil, admin, "enable silent SSO",
	); err != nil {
		t.Fatalf("SaveSettings() error = %v", err)
	}

	_, silentQuery := startQuery(t, service, "/", true)
	refused, err := service.RefusedSilently(
		context.Background(), silentQuery.Get("state"), "login_required",
	)
	if err != nil || !refused {
		t.Fatalf("RefusedSilently(silent, login_required) = %v, %v; want true", refused, err)
	}
	// Replaying the same state after it was consumed is not a refusal.
	refused, err = service.RefusedSilently(
		context.Background(), silentQuery.Get("state"), "login_required",
	)
	if err != nil || refused {
		t.Fatalf("RefusedSilently(replay) = %v, %v; want false", refused, err)
	}
	// The consumed flow cannot be finished with a code either.
	if _, _, err := service.Callback(
		context.Background(), silentQuery.Get("state"), "valid-code",
		"192.0.2.20", "test-agent", "request-replay",
	); err == nil {
		t.Fatal("Callback() accepted a state that was already refused")
	}

	// A real failure on a silent flow is still a failure.
	_, deniedQuery := startQuery(t, service, "/", true)
	refused, err = service.RefusedSilently(
		context.Background(), deniedQuery.Get("state"), "access_denied",
	)
	if err != nil || refused {
		t.Fatalf("RefusedSilently(silent, access_denied) = %v, %v; want false", refused, err)
	}

	// login_required on an ordinary login is not a silent refusal.
	_, ordinaryQuery := startQuery(t, service, "/", false)
	refused, err = service.RefusedSilently(
		context.Background(), ordinaryQuery.Get("state"), "login_required",
	)
	if err != nil || refused {
		t.Fatalf("RefusedSilently(ordinary, login_required) = %v, %v; want false", refused, err)
	}

	// An unknown or missing state is left to the usual failure reporting.
	for _, state := range []string{"", "never-issued"} {
		refused, err = service.RefusedSilently(context.Background(), state, "login_required")
		if err != nil || refused {
			t.Fatalf("RefusedSilently(%q) = %v, %v; want false", state, refused, err)
		}
	}
}

// Someone who opened a deep link and had a provider session must land on that
// deep link, not on the dashboard.
func TestOIDCSilentLoginReturnsToTheDeepLink(t *testing.T) {
	service, provider, settings, admin, _ := newConfiguredOIDCService(t)
	settings.AutoLogin = true
	if err := service.SaveSettings(
		context.Background(), settings, nil, admin, "enable silent SSO",
	); err != nil {
		t.Fatalf("SaveSettings() error = %v", err)
	}
	start, query := startQuery(t, service, "/#/assets?asset=42", true)
	if !start.Silent {
		t.Fatal("Start() did not send prompt=none")
	}
	provider.SetAuthorization(query.Get("nonce"), query.Get("code_challenge"))
	session, returnTo, err := service.Callback(
		context.Background(), query.Get("state"), "valid-code",
		"192.0.2.20", "test-agent", "request-silent",
	)
	if err != nil {
		t.Fatalf("Callback() error = %v", err)
	}
	if returnTo != "/#/assets?asset=42" {
		t.Fatalf("returnTo = %q, want the deep link", returnTo)
	}
	if session.Token == "" {
		t.Fatal("silent login issued no session")
	}

	// A return_to that leaves the origin is not carried along.
	for _, unsafe := range []string{"//evil.example/", "https://evil.example/", "\\evil"} {
		_, query := startQuery(t, service, unsafe, true)
		provider.SetAuthorization(query.Get("nonce"), query.Get("code_challenge"))
		_, returnTo, err := service.Callback(
			context.Background(), query.Get("state"), "valid-code",
			"192.0.2.20", "test-agent", "request-unsafe",
		)
		if err != nil {
			t.Fatalf("Callback(%q) error = %v", unsafe, err)
		}
		if returnTo != "/" {
			t.Fatalf("returnTo for %q = %q, want /", unsafe, returnTo)
		}
	}
}

func TestSilentLoginRefusedNamesOnlyTheNoSessionAnswers(t *testing.T) {
	for _, code := range []string{"login_required", "interaction_required", "consent_required"} {
		if !SilentLoginRefused(code) {
			t.Errorf("SilentLoginRefused(%q) = false", code)
		}
	}
	for _, code := range []string{"", "access_denied", "invalid_request", "server_error"} {
		if SilentLoginRefused(code) {
			t.Errorf("SilentLoginRefused(%q) = true", code)
		}
	}
}
