package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/jackc/pgx/v5"
)

// An MCP client can sign a person in through Keycloak instead of carrying a
// key. umm is then the resource server of the MCP authorization flow: it says
// where the authorization server is, and it accepts the access token that
// comes back. Nothing here issues, refreshes or registers anything — Keycloak
// does all of that, and the MCP client talks to it directly.
//
// A token is accepted when it verifies against the issuer's keys, names this
// service as its audience, has not expired, and stands for a person umm knows
// or can create. Its scope claim, filtered by the administrator's allowed
// scopes, is what the token may do — the same words a key carries, so a tool
// asks the same question of both.

// AuthTypeOAuth marks a principal that arrived with a Keycloak access token.
// The other two, "session" and "api_key", predate the constant.
const AuthTypeOAuth = "oauth"

// MCPResourcePath is the resource an access token has to be issued for.
const MCPResourcePath = "/mcp"

// ProtectedResourceMetadataPath is where RFC 9728 has a resource server
// describe itself. The path-specific form, with the resource path appended, is
// the one the 401 challenge names.
const ProtectedResourceMetadataPath = "/.well-known/oauth-protected-resource"

// ErrTokenRejected is the whole of what a caller learns about a refused token.
// The reason goes to the log: an error description that reaches the client is
// a description an attacker reads too.
var ErrTokenRejected = errors.New("access token rejected")

// ProtectedResource is what RFC 9728 has a resource server publish, and what
// an MCP client reads after a 401 to find out where to sign in.
type ProtectedResource struct {
	Resource              string   `json:"resource"`
	AuthorizationServers  []string `json:"authorization_servers"`
	ScopesSupported       []string `json:"scopes_supported"`
	BearerMethods         []string `json:"bearer_methods_supported"`
	ResourceName          string   `json:"resource_name,omitempty"`
	ResourceDocumentation string   `json:"resource_documentation,omitempty"`
	// MetadataURL is where this document is served, for the challenge header.
	MetadataURL string `json:"-"`
}

// MCPResource describes /mcp as an OAuth protected resource, or reports false
// while the administrator has not turned Keycloak sign-in for MCP on. toolScopes
// are the scopes MCP tools ask for; only those the administrator also allows
// on a key are advertised, so a token cannot be asked for more than a key.
func (s *OIDCService) MCPResource(ctx context.Context, toolScopes []string) (ProtectedResource, bool) {
	cfg, general, ok := s.mcpSettings(ctx)
	if !ok {
		return ProtectedResource{}, false
	}
	base := strings.TrimRight(general.PublicURL, "/")
	allowed := s.allowedScopes(ctx)
	scopes := make([]string, 0, len(toolScopes))
	for _, scope := range toolScopes {
		if allowed[scope] {
			scopes = append(scopes, scope)
		}
	}
	slices.Sort(scopes)
	return ProtectedResource{
		Resource:              base + MCPResourcePath,
		AuthorizationServers:  []string{strings.TrimRight(cfg.IssuerURL, "/")},
		ScopesSupported:       scopes,
		BearerMethods:         []string{"header"},
		ResourceName:          general.ServiceName,
		ResourceDocumentation: base + "/docs/MCP.md",
		MetadataURL:           base + ProtectedResourceMetadataPath + MCPResourcePath,
	}, true
}

// AuthenticateAccessToken verifies a Keycloak access token presented at /mcp
// and returns the principal it stands for. Every refusal is ErrTokenRejected;
// the reason is logged.
func (s *OIDCService) AuthenticateAccessToken(ctx context.Context, raw string) (Principal, error) {
	cfg, general, ok := s.mcpSettings(ctx)
	if !ok {
		return Principal{}, ErrTokenRejected
	}
	issuer := strings.TrimRight(cfg.IssuerURL, "/")
	provider, err := s.discovery(issuer, false)
	if err != nil {
		slog.Warn("MCP access token: provider discovery failed", "issuer", issuer, "error", err)
		return Principal{}, ErrTokenRejected
	}
	audience := strings.TrimSpace(cfg.MCPAudience)
	if audience == "" {
		audience = strings.TrimRight(general.PublicURL, "/") + MCPResourcePath
	}
	token, err := provider.Verifier(&oidc.Config{ClientID: audience}).Verify(ctx, raw)
	if err != nil {
		slog.Info("MCP access token refused", "error", err)
		return Principal{}, ErrTokenRejected
	}
	var claims struct {
		Subject           string   `json:"sub"`
		Scope             string   `json:"scope"`
		PreferredUsername string   `json:"preferred_username"`
		Name              string   `json:"name"`
		Email             string   `json:"email"`
		Groups            []string `json:"groups"`
		RealmAccess       struct {
			Roles []string `json:"roles"`
		} `json:"realm_access"`
	}
	if err = token.Claims(&claims); err != nil || claims.Subject == "" {
		slog.Info("MCP access token refused: claims unreadable", "error", err)
		return Principal{}, ErrTokenRejected
	}
	// Someone who has signed in before is exactly who they were: a token
	// never changes a role or wakes a deactivated account. Someone new is
	// created the way the login callback would create them, group mapping
	// included, so the first door they come through does not decide their role.
	u, err := s.Store.UserByOIDCSubject(ctx, claims.Subject)
	if errors.Is(err, pgx.ErrNoRows) {
		role := cfg.roleFor(append(claims.Groups, claims.RealmAccess.Roles...))
		u, err = s.Store.ProvisionOIDCUser(ctx, claims.Subject, claims.PreferredUsername, claims.Name, claims.Email, role)
		if err == nil {
			s.Store.Audit(ctx, &u.ID, "auth.oidc.mcp.provision", "user", u.ID.String(), map[string]any{"role": role})
		}
	}
	if err != nil {
		slog.Warn("MCP access token refused: no user for subject", "error", err)
		return Principal{}, ErrTokenRejected
	}
	if !u.Active {
		slog.Info("MCP access token refused: user deactivated", "user", u.Username)
		return Principal{}, ErrTokenRejected
	}
	return Principal{User: u, Scopes: grantedScopes(claims.Scope, s.allowedScopes(ctx)), AuthType: AuthTypeOAuth}, nil
}

// grantedScopes reads the token's space-separated scope claim and keeps the
// scopes the administrator allows on a key. A token that names none of them
// stands for a person who may do nothing, which is a plain answer at each
// tool rather than a refusal at the door.
func grantedScopes(claim string, allowed map[string]bool) map[string]bool {
	granted := map[string]bool{}
	for _, scope := range strings.Fields(claim) {
		if allowed[scope] {
			granted[scope] = true
		}
	}
	return granted
}

// mcpSettings reads what accepting a token needs and reports false unless SSO
// and MCP sign-in are both on and a public URL is known: the public URL is
// the audience and the resource, and without one there is nothing to verify
// a token against.
func (s *OIDCService) mcpSettings(ctx context.Context) (OIDCSettings, GeneralSettings, bool) {
	var cfg OIDCSettings
	var general GeneralSettings
	if s.Store.GetSetting(ctx, "oidc", &cfg) != nil || !cfg.Enabled || !cfg.MCPOAuth {
		return cfg, general, false
	}
	if s.Store.GetSetting(ctx, "general", &general) != nil || strings.TrimSpace(general.PublicURL) == "" {
		return cfg, general, false
	}
	return cfg, general, true
}

// allowedScopes is the administrator's list of what a key may be given, read
// from the same setting the key screen reads. It bounds tokens too: turning
// Keycloak sign-in on for MCP must not be a way around that list.
func (s *OIDCService) allowedScopes(ctx context.Context) map[string]bool {
	var security struct {
		APIKeyScopes []string `json:"api_key_scopes"`
	}
	_ = s.Store.GetSetting(ctx, "security", &security)
	allowed := make(map[string]bool, len(security.APIKeyScopes))
	for _, scope := range security.APIKeyScopes {
		allowed[scope] = true
	}
	return allowed
}

// ValidateMCPSettings checks the MCP half of the oidc section before it is
// saved: sign-in for MCP needs SSO itself on, and an audience, if given, is
// one plain value rather than a list.
func ValidateMCPSettings(v map[string]any) error {
	mcpOAuth, _ := v["mcp_oauth"].(bool)
	enabled, _ := v["enabled"].(bool)
	if mcpOAuth && !enabled {
		return errors.New("MCP Keycloak 로그인은 Keycloak SSO가 켜져 있어야 합니다")
	}
	if raw, present := v["mcp_audience"]; present && raw != nil {
		audience, ok := raw.(string)
		if !ok || strings.ContainsAny(audience, " \t\r\n") {
			return fmt.Errorf("MCP 토큰 audience는 공백 없는 문자열 하나여야 합니다")
		}
	}
	return nil
}
