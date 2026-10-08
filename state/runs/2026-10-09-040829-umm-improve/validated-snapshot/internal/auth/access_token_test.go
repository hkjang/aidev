package auth

import (
	"testing"
)

// A token's scope claim is Keycloak's list of everything the client asked for
// and got: openid, profile, umm's scopes and anything else the realm hands
// out. Only umm's own scopes, and only those the administrator allows on a
// key, become permissions here — a token is not a way around the key policy.
func TestGrantedScopesAreTheTokensBoundedByThePolicy(t *testing.T) {
	allowed := map[string]bool{"notes:read": true, "spaces:read": true}
	granted := grantedScopes("openid profile  notes:read notes:write email", allowed)
	if !granted["notes:read"] {
		t.Fatal("an allowed scope the token carries was not granted")
	}
	if granted["notes:write"] {
		t.Fatal("a scope the policy does not allow was granted because the token claimed it")
	}
	if granted["spaces:read"] {
		t.Fatal("a scope the token never claimed was granted because the policy allows it")
	}
	for _, foreign := range []string{"openid", "profile", "email", "*"} {
		if granted[foreign] {
			t.Fatalf("%q became a permission", foreign)
		}
	}
	if len(grantedScopes("", allowed)) != 0 {
		t.Fatal("an empty scope claim granted something")
	}
}

// The group mapping is shared by the login callback and the MCP door, so a
// person is the same role whichever way they first arrive. A blank group name
// matches nothing: leaving the administrator field empty must not make an
// administrator of whoever carries an empty group.
func TestRoleForMapsGroupsAndIgnoresBlankGroupNames(t *testing.T) {
	cfg := OIDCSettings{AdminGroup: "umm-admins", TeamLeadGroup: "umm-leads"}
	for _, tc := range []struct {
		groups []string
		want   string
	}{
		{[]string{"umm-admins"}, "admin"},
		{[]string{"umm-leads"}, "team_lead"},
		{[]string{"umm-leads", "umm-admins"}, "admin"},
		{[]string{"staff"}, "user"},
		{nil, "user"},
	} {
		if got := cfg.roleFor(tc.groups); got != tc.want {
			t.Errorf("roleFor(%v) = %s, want %s", tc.groups, got, tc.want)
		}
	}
	blank := OIDCSettings{}
	if got := blank.roleFor([]string{""}); got != "user" {
		t.Fatalf("an empty group matched the empty administrator group: %s", got)
	}
}

// Keycloak sign-in for MCP rides on SSO: a token is verified against the SSO
// issuer, so it cannot be on while SSO is off. The audience is one value.
func TestValidateMCPSettings(t *testing.T) {
	if err := ValidateMCPSettings(map[string]any{"enabled": false, "mcp_oauth": true}); err == nil {
		t.Fatal("MCP sign-in was accepted with SSO off")
	}
	if err := ValidateMCPSettings(map[string]any{"enabled": true, "mcp_oauth": true, "mcp_audience": "https://umm.example/mcp"}); err != nil {
		t.Fatalf("a sound section was refused: %v", err)
	}
	if err := ValidateMCPSettings(map[string]any{"enabled": true, "mcp_oauth": true, "mcp_audience": "a b"}); err == nil {
		t.Fatal("an audience with a space in it was accepted")
	}
	if err := ValidateMCPSettings(map[string]any{"enabled": false}); err != nil {
		t.Fatalf("a section without the MCP fields was refused: %v", err)
	}
}
