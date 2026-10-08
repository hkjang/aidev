package auth

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/hkjang/umm/internal/cryptoutil"
	"github.com/hkjang/umm/internal/store"
	"golang.org/x/oauth2"
)

type OIDCSettings struct {
	Enabled       bool     `json:"enabled"`
	IssuerURL     string   `json:"issuer_url"`
	ClientID      string   `json:"client_id"`
	ClientSecret  string   `json:"client_secret"`
	Scopes        []string `json:"scopes"`
	AdminGroup    string   `json:"admin_group"`
	TeamLeadGroup string   `json:"team_lead_group"`
	// AutoLogin lets the browser ask Keycloak for a session it already has
	// (prompt=none) before drawing the login screen, so someone signed in
	// there walks straight in here. Off by default: a silent attempt is a
	// redirect, and where a redirect can start from is the administrator's
	// decision — see Start, which ignores prompt=none while this is off.
	AutoLogin bool `json:"auto_login"`
	// MCPOAuth lets /mcp accept an access token Keycloak issued, so an MCP
	// client signs the person in through the provider instead of carrying a
	// key. Off by default; see access_token.go for what a token has to carry.
	MCPOAuth bool `json:"mcp_oauth"`
	// MCPAudience is the aud claim a token has to carry. Empty means the
	// resource identifier itself: the public URL with /mcp appended.
	MCPAudience string `json:"mcp_audience"`
}

// silentRefusals are the answers prompt=none gives when the provider has no
// session to answer from. None of them is a failure; each means "ask the
// person". Anything else the provider reports is a real error.
var silentRefusals = map[string]bool{"login_required": true, "interaction_required": true, "consent_required": true}

type GeneralSettings struct {
	ServiceName  string `json:"service_name"`
	PublicURL    string `json:"public_url"`
	SessionHours int    `json:"session_hours"`
}

type OIDCService struct {
	Store    *store.Store
	Cipher   *cryptoutil.Cipher
	Sessions *Service

	// The provider is discovery plus a key set, fetched from Keycloak. One
	// login could afford to fetch it; an access token on every MCP request
	// cannot, so it is kept for a while and refetched when the issuer changes.
	providerMu      sync.Mutex
	providerIssuer  string
	providerFetched time.Time
	provider        *oidc.Provider
}

// providerTTL bounds how long a fetched provider is reused. The key set inside
// it refetches on its own when a token names a key it has not seen, so this
// only has to catch an issuer that moved its endpoints.
const providerTTL = 10 * time.Minute

// discovery fetches or reuses the provider for an issuer. fresh forces a fetch,
// which is what a connection test is for. The provider is built on a background
// context with its own client on purpose: a request context would be cancelled
// with the request that created it, and the key set keeps the context.
func (s *OIDCService) discovery(issuer string, fresh bool) (*oidc.Provider, error) {
	s.providerMu.Lock()
	defer s.providerMu.Unlock()
	if !fresh && s.provider != nil && s.providerIssuer == issuer && time.Since(s.providerFetched) < providerTTL {
		return s.provider, nil
	}
	ctx := oidc.ClientContext(context.Background(), &http.Client{Timeout: 10 * time.Second})
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, err
	}
	s.provider, s.providerIssuer, s.providerFetched = provider, issuer, time.Now()
	return provider, nil
}

func (s *OIDCService) Enabled(ctx context.Context) bool {
	enabled, _ := s.Public(ctx)
	return enabled
}

// Public reports what the login screen needs to know before anyone is signed
// in: whether SSO is offered at all, and whether the browser should try it
// silently first. autoLogin is never true while SSO is off.
func (s *OIDCService) Public(ctx context.Context) (enabled, autoLogin bool) {
	var cfg OIDCSettings
	if s.Store.GetSetting(ctx, "oidc", &cfg) != nil || !cfg.Enabled {
		return false, false
	}
	return true, cfg.AutoLogin
}
func (s *OIDCService) Test(ctx context.Context) error {
	_, _, _, err := s.configure(ctx, true)
	return err
}

func (s *OIDCService) configuration(ctx context.Context) (OIDCSettings, *oidc.Provider, oauth2.Config, error) {
	return s.configure(ctx, false)
}

func (s *OIDCService) configure(ctx context.Context, fresh bool) (OIDCSettings, *oidc.Provider, oauth2.Config, error) {
	var cfg OIDCSettings
	var general GeneralSettings
	if err := s.Store.GetSetting(ctx, "oidc", &cfg); err != nil {
		return cfg, nil, oauth2.Config{}, err
	}
	if !cfg.Enabled {
		return cfg, nil, oauth2.Config{}, errors.New("OIDC is disabled")
	}
	if err := s.Store.GetSetting(ctx, "general", &general); err != nil {
		return cfg, nil, oauth2.Config{}, err
	}
	issuer, err := url.Parse(cfg.IssuerURL)
	if err != nil || !(issuer.Scheme == "http" || issuer.Scheme == "https") || issuer.Host == "" {
		return cfg, nil, oauth2.Config{}, errors.New("invalid OIDC issuer URL")
	}
	secret := cfg.ClientSecret
	if strings.HasPrefix(secret, "enc:") {
		secret, err = s.Cipher.Decrypt(strings.TrimPrefix(secret, "enc:"))
		if err != nil {
			return cfg, nil, oauth2.Config{}, err
		}
	}
	provider, err := s.discovery(strings.TrimRight(cfg.IssuerURL, "/"), fresh)
	if err != nil {
		return cfg, nil, oauth2.Config{}, err
	}
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{oidc.ScopeOpenID, "profile", "email"}
	}
	if !slices.Contains(scopes, oidc.ScopeOpenID) {
		scopes = append([]string{oidc.ScopeOpenID}, scopes...)
	}
	callback := strings.TrimRight(general.PublicURL, "/") + "/api/v1/auth/oidc/callback"
	oauthCfg := oauth2.Config{ClientID: cfg.ClientID, ClientSecret: secret, Endpoint: provider.Endpoint(), RedirectURL: callback, Scopes: scopes}
	return cfg, provider, oauthCfg, nil
}

func (s *OIDCService) Start(w http.ResponseWriter, r *http.Request) {
	returnTo := safeReturnTo(r.URL.Query().Get("return_to"))
	// Every way this can fail lands on the login screen, never on a page of
	// plain text. A silent attempt is a navigation the person did not ask for:
	// with Keycloak down it used to strand them on "503 dial tcp …" at an API
	// address, where a reload only fetched the same page again. The reason
	// goes to the log; the screen says SSO did not complete and offers the
	// password form.
	settings, _, cfg, err := s.configuration(r.Context())
	if err != nil {
		slog.Warn("OIDC login could not start", "error", err)
		http.Redirect(w, r, loginLanding("error", returnTo), http.StatusFound)
		return
	}
	state, err := randomToken(32)
	if err != nil {
		http.Redirect(w, r, loginLanding("error", returnTo), http.StatusFound)
		return
	}
	_, err = s.Store.Pool.Exec(r.Context(), `INSERT INTO oauth_states(state_hash,return_to,expires_at) VALUES($1,$2,now()+interval '10 minutes')`, digest(state), returnTo)
	if err != nil {
		slog.Warn("OIDC login state could not be stored", "error", err)
		http.Redirect(w, r, loginLanding("error", returnTo), http.StatusFound)
		return
	}
	// prompt=none asks the provider to answer from a session it already has
	// and never to draw a screen: either a code comes straight back or
	// login_required does. The browser asks for it only when auto_login is on;
	// if something asks anyway while it is off, the attempt quietly becomes an
	// ordinary login rather than a silent one, so appending ?prompt=none to a
	// URL cannot change the flow — only the administrator's setting can.
	options := []oauth2.AuthCodeOption{oauth2.AccessTypeOffline}
	if settings.AutoLogin && r.URL.Query().Get("prompt") == "none" {
		options = append(options, oauth2.SetAuthURLParam("prompt", "none"))
	}
	http.Redirect(w, r, cfg.AuthCodeURL(state, options...), http.StatusFound)
}

// safeReturnTo keeps a return_to inside this site. Only a path — starting with
// one slash, not two — is honoured; anything else, including an absolute URL
// or a scheme-relative one, becomes the front page, so the login flow cannot
// be used as a springboard to somewhere else.
//
// A backslash counts as a slash: browsers read "/\\evil.example" as
// "//evil.example", so it is refused the same way.
func safeReturnTo(value string) string {
	if !strings.HasPrefix(value, "/") || strings.ContainsAny(value, "\r\n\\") {
		return "/"
	}
	if strings.HasPrefix(value, "//") {
		return "/"
	}
	return value
}

// loginLanding is where the flow sends someone it could not sign in: the login
// screen, with the outcome in the address — sso=none for "no session at the
// provider", sso=error for anything else — and the address they were opening,
// so signing in from there still takes them to it. The marker is also what
// stops the browser trying silently again, even with storage cleared.
func loginLanding(outcome, returnTo string) string {
	landing := "/login?sso=" + outcome
	if returnTo = safeReturnTo(returnTo); returnTo != "/" {
		landing += "&return_to=" + url.QueryEscape(returnTo)
	}
	return landing
}

// fail ends a callback that cannot sign anyone in. The detail is logged and
// never shown: it can name the provider's internals.
func fail(w http.ResponseWriter, r *http.Request, returnTo, reason string, err error) {
	slog.Warn("OIDC callback failed", "reason", reason, "error", err)
	http.Redirect(w, r, loginLanding("error", returnTo), http.StatusFound)
}

func (s *OIDCService) Callback(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")
	// The state is spent first, whatever else the answer says: it is single
	// use, and it holds the address the person was opening, which every
	// outcome below — success, refusal, failure — takes them back towards.
	returnTo := "/"
	stateValid := false
	if state != "" {
		err := s.Store.Pool.QueryRow(r.Context(), `DELETE FROM oauth_states WHERE state_hash=$1 AND expires_at>now() RETURNING return_to`, digest(state)).Scan(&returnTo)
		stateValid = err == nil
		if !stateValid {
			returnTo = "/"
		}
	}
	// The provider reports a refusal as an error parameter rather than a code.
	// prompt=none answers login_required whenever there is no session — an
	// ordinary reply, not a failure — and the one thing that must not happen
	// next is another silent attempt, or the browser bounces between here and
	// the provider for as long as the person watches it flicker. So the
	// refusal lands on the login screen with a marker in the address: it is
	// the one guard that survives a cleared or unreadable browser storage.
	if providerError := r.URL.Query().Get("error"); providerError != "" {
		if silentRefusals[providerError] {
			http.Redirect(w, r, loginLanding("none", returnTo), http.StatusFound)
			return
		}
		slog.Warn("OIDC provider returned an error", "error", providerError)
		http.Redirect(w, r, loginLanding("error", returnTo), http.StatusFound)
		return
	}
	if code == "" {
		fail(w, r, returnTo, "callback without a code", nil)
		return
	}
	if !stateValid {
		// Expired, spent, or never issued: a back button pressed after a
		// login, or a link replayed from history. Not something to explain.
		fail(w, r, "/", "state expired or invalid", nil)
		return
	}
	settings, provider, cfg, err := s.configuration(r.Context())
	if err != nil {
		fail(w, r, returnTo, "configuration unavailable", err)
		return
	}
	token, err := cfg.Exchange(r.Context(), code)
	if err != nil {
		fail(w, r, returnTo, "token exchange failed", err)
		return
	}
	rawID, ok := token.Extra("id_token").(string)
	if !ok {
		fail(w, r, returnTo, "id_token missing", nil)
		return
	}
	idToken, err := provider.Verifier(&oidc.Config{ClientID: cfg.ClientID}).Verify(r.Context(), rawID)
	if err != nil {
		fail(w, r, returnTo, "id_token verification failed", err)
		return
	}
	var claims struct {
		Subject           string   `json:"sub"`
		PreferredUsername string   `json:"preferred_username"`
		Name              string   `json:"name"`
		Email             string   `json:"email"`
		Groups            []string `json:"groups"`
		RealmAccess       struct {
			Roles []string `json:"roles"`
		} `json:"realm_access"`
	}
	if err = idToken.Claims(&claims); err != nil {
		fail(w, r, returnTo, "claims invalid", err)
		return
	}
	role := settings.roleFor(append(claims.Groups, claims.RealmAccess.Roles...))
	u, err := s.Store.UpsertOIDCUser(r.Context(), claims.Subject, claims.PreferredUsername, claims.Name, claims.Email, role)
	if err != nil {
		fail(w, r, returnTo, "user provisioning failed", err)
		return
	}
	session, err := s.Sessions.CreateSession(r.Context(), u.ID, OriginOf(r))
	if err != nil {
		fail(w, r, returnTo, "session creation failed", err)
		return
	}
	SetSessionCookie(w, r, session)
	s.Store.Audit(r.Context(), &u.ID, "auth.oidc.login", "user", u.ID.String(), json.RawMessage(`{}`))
	http.Redirect(w, r, safeReturnTo(returnTo), http.StatusFound)
}

// roleFor maps the provider's groups and realm roles to umm's role. An empty
// administrator group never matches, so nobody is an administrator because the
// field was left blank.
func (cfg OIDCSettings) roleFor(groups []string) string {
	if cfg.AdminGroup != "" && slices.Contains(groups, cfg.AdminGroup) {
		return "admin"
	}
	if cfg.TeamLeadGroup != "" && slices.Contains(groups, cfg.TeamLeadGroup) {
		return "team_lead"
	}
	return "user"
}

func SetSessionCookie(w http.ResponseWriter, r *http.Request, token string) {
	secure := r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
	http.SetCookie(w, &http.Cookie{Name: CookieName, Value: token, Path: "/", HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode, Expires: time.Now().Add(30 * 24 * time.Hour), MaxAge: 30 * 86400})
}
func ClearSessionCookie(w http.ResponseWriter, r *http.Request) {
	secure := r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
	http.SetCookie(w, &http.Cookie{Name: CookieName, Value: "", Path: "/", HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode, Expires: time.Unix(0, 0), MaxAge: -1})
}
