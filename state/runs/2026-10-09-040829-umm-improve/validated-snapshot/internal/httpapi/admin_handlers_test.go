package httpapi

import (
	"testing"

	"github.com/hkjang/umm/internal/dream"
	"github.com/hkjang/umm/internal/store"
)

func TestValidateGeneralSettings(t *testing.T) {
	s := &Server{}
	valid := map[string]any{"service_name": "umm", "public_url": "https://umm.internal", "timezone": "Asia/Seoul"}
	if err := s.validateSetting("general", valid); err != nil {
		t.Fatalf("valid settings rejected: %v", err)
	}
	invalid := map[string]any{"service_name": "umm", "public_url": "javascript:alert(1)", "timezone": "Asia/Seoul"}
	if err := s.validateSetting("general", invalid); err == nil {
		t.Fatal("invalid public URL accepted")
	}
}

func TestValidateDreamTokenLimit(t *testing.T) {
	s := &Server{}
	valid := map[string]any{
		"quality_threshold": 0.7,
		"schedule":          "02:00",
		"frequency":         "daily",
		"token_limit":       float64(262144),
	}
	if err := s.validateSetting("dream", valid); err != nil {
		t.Fatalf("256K token limit rejected: %v", err)
	}
	for _, invalid := range []float64{63, 262145, 4096.5} {
		valid["token_limit"] = invalid
		if err := s.validateSetting("dream", valid); err == nil {
			t.Fatalf("invalid token limit accepted: %v", invalid)
		}
	}
}

func TestValidateAIGatewayTimeout(t *testing.T) {
	s := &Server{}
	valid := map[string]any{"base_url": "https://ai.internal", "log_retention_days": float64(90), "timeout_seconds": float64(dream.MaxGatewayTimeoutSeconds), "max_retries": float64(dream.MaxGatewayRetries)}
	if err := s.validateSetting("ai_gateway", valid); err != nil {
		t.Fatalf("30 minute timeout rejected: %v", err)
	}
	valid["timeout_seconds"] = float64(dream.MaxGatewayTimeoutSeconds + 1)
	if err := s.validateSetting("ai_gateway", valid); err == nil {
		t.Fatal("timeout over 30 minutes accepted")
	}
	valid["timeout_seconds"] = float64(dream.MaxGatewayTimeoutSeconds)
	valid["max_retries"] = float64(dream.MaxGatewayRetries + 1)
	if err := s.validateSetting("ai_gateway", valid); err == nil {
		t.Fatal("excessive gateway retries accepted")
	}
}

func TestValidateSecuritySettingsDistinguishesOmittedAndExplicitValues(t *testing.T) {
	s := &Server{}
	legacy := map[string]any{"api_key_scopes": []any{"notes:read"}}
	if err := s.validateSetting("security", legacy); err != nil {
		t.Fatalf("legacy payload with omitted abuse guards was rejected: %v", err)
	}

	withUnlimitedAI := map[string]any{
		"api_key_scopes":        []any{"notes:read"},
		"login_max_failures":    float64(8),
		"ai_daily_limit":        float64(0),
		"api_rate_per_minute":   float64(600),
		"ai_rate_per_minute":    float64(6),
		"login_lockout_minutes": float64(15),
	}
	if err := s.validateSetting("security", withUnlimitedAI); err != nil {
		t.Fatalf("explicit zero daily limit was rejected: %v", err)
	}

	withNull := map[string]any{"api_key_scopes": []any{"notes:read"}, "login_max_failures": nil}
	if err := s.validateSetting("security", withNull); err == nil {
		t.Fatal("an explicit null abuse guard was accepted as an omission")
	}
}

func TestValidatePtiumSettings(t *testing.T) {
	s := &Server{}
	base := func(over map[string]any) map[string]any {
		v := map[string]any{"base_url": "", "api_key": "", "template_id": "", "language": "ko", "timeout_seconds": 30.0}
		for k, val := range over {
			v[k] = val
		}
		return v
	}

	if err := s.validateSetting("ptium", base(nil)); err != nil {
		t.Fatalf("the seeded, unconnected default was rejected: %v", err)
	}
	if err := s.validateSetting("ptium", base(map[string]any{"base_url": "https://ptium.internal"})); err != nil {
		t.Fatalf("a valid address was rejected: %v", err)
	}
	// Refused rather than ignored: a saved address that does nothing looks
	// exactly like one that works until someone tries to make a deck.
	for _, bad := range []string{"ptium.internal", "javascript:alert(1)", "file:///etc/passwd", "not a url"} {
		if err := s.validateSetting("ptium", base(map[string]any{"base_url": bad})); err == nil {
			t.Fatalf("address %q accepted", bad)
		}
	}
	// A key with nowhere to send it is a credential stored for no reason.
	if err := s.validateSetting("ptium", base(map[string]any{"api_key": "ptium_x"})); err == nil {
		t.Fatal("an API key without an address was accepted")
	}
	for _, timeout := range []any{0.0, 4.0, 301.0, 30.5, "30", nil} {
		if err := s.validateSetting("ptium", base(map[string]any{"timeout_seconds": timeout})); err == nil {
			t.Fatalf("timeout %v accepted", timeout)
		}
	}
}

// The Ptium credential has to be masked on read and encrypted on write like
// every other secret, or connecting a second service quietly becomes the one
// place a key is served back in clear text.
func TestPtiumAPIKeyIsTreatedAsASecret(t *testing.T) {
	fields := secretFields("ptium")
	if len(fields) != 1 || fields[0] != "api_key" {
		t.Fatalf("ptium secrets are %v", fields)
	}
}

// An omitted address is unset, not invalid. fmt.Sprint renders a missing key as
// the literal "<nil>", which url.Parse accepts and the scheme check then
// rejects — so a payload with no base_url came back as "the URL is invalid"
// when there was no URL at all.
func TestOmittedGatewayAddressReadsAsUnset(t *testing.T) {
	s := &Server{}
	v := map[string]any{"timeout_seconds": 45.0, "max_retries": 2.0, "log_retention_days": 90.0}
	if err := s.validateSetting("ai_gateway", v); err != nil {
		t.Fatalf("settings with no address were rejected: %v", err)
	}

	ptium := map[string]any{"timeout_seconds": 30.0}
	if err := s.validateSetting("ptium", ptium); err != nil {
		t.Fatalf("ptium settings with no address were rejected: %v", err)
	}
}

// Every section that has validation has to be reachable, or the validation
// guards a page nobody can open.
//
// ptium shipped with a validator, a secret registration and a seeded row, and
// was still 404 through the admin API because it was missing from the
// allowlist — which made the whole feature unconfigurable. Checked as a class
// rather than one name, so the next section cannot repeat it.
func TestEverySectionWithValidationIsReachable(t *testing.T) {
	for _, section := range []string{"general", "oidc", "security", "workflow", "dream", "ai_gateway", "intelligence", "ptium"} {
		if !store.AllowedSetting(section) {
			t.Fatalf("section %q has validation but cannot be written through the admin API", section)
		}
	}
	// And the guard is a real one, not a function that says yes to everything.
	if store.AllowedSetting("not_a_section") {
		t.Fatal("the allowlist accepts anything")
	}
}

// A section whose secrets are registered must also be writable, or the
// credential is masked on a page that cannot be saved.
func TestEverySectionWithSecretsIsReachable(t *testing.T) {
	for _, section := range []string{"oidc", "ai_gateway", "ptium"} {
		if len(secretFields(section)) == 0 {
			t.Fatalf("section %q is expected to hold a secret", section)
		}
		if !store.AllowedSetting(section) {
			t.Fatalf("section %q holds a secret but cannot be written", section)
		}
	}
}
