package auth

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/google/uuid"
	"github.com/hkjang/invenqor/server/internal/audit"
	"github.com/hkjang/invenqor/server/internal/bootstrap"
	"golang.org/x/oauth2"
)

var (
	ErrOIDCDisabled     = errors.New("Keycloak login is disabled")
	ErrOIDCFlow         = errors.New("OIDC login flow is invalid or expired")
	ErrOIDCNonce        = errors.New("OIDC nonce validation failed")
	ErrOIDCDomain       = errors.New("OIDC email domain is not allowed")
	ErrOIDCProvisioning = errors.New("OIDC user provisioning is disabled")
	ErrOIDCUsername     = errors.New("OIDC response has no usable username")
	ErrOIDCUserInactive = errors.New("Keycloak-linked user is inactive")
	ErrOIDCSecret       = errors.New("Keycloak client secret is required")
	// ErrOIDCSecretUnreadable means a sealed secret exists but this instance's
	// master key cannot decrypt it.
	ErrOIDCSecretUnreadable = errors.New("the stored Keycloak client secret cannot be decrypted with this instance's master key")
	ErrOIDCRole             = errors.New("Keycloak role mapping references an unknown role")
	// ErrOIDCUsernameTaken separates "a local account owns this name" from a
	// generic database failure. Without it the administrator sees only a failed
	// login and has no way to learn that renaming the local account fixes it.
	ErrOIDCUsernameTaken = errors.New("a local account already uses this Keycloak username")
	ErrOIDCUnreachable   = errors.New("Keycloak issuer could not be reached")
)

const (
	keycloakSettingKey             = "auth.keycloak"
	keycloakClientSecretSettingKey = "auth.keycloak.client_secret"
	keycloakClientSecretPurpose    = "oidc.client_secret"
)

type oidcClientSecretEnvelope struct {
	Sealed string `json:"sealed"`
}

type OIDCSettings struct {
	Enabled           bool              `json:"enabled"`
	IssuerURL         string            `json:"issuer_url"`
	Realm             string            `json:"realm"`
	ClientID          string            `json:"client_id"`
	RedirectURI       string            `json:"redirect_uri"`
	LogoutRedirectURI string            `json:"logout_redirect_uri"`
	Scopes            []string          `json:"scopes"`
	UsernameClaim     string            `json:"username_claim"`
	EmailClaim        string            `json:"email_claim"`
	NameClaim         string            `json:"name_claim"`
	GroupClaim        string            `json:"group_claim"`
	RoleClaim         string            `json:"role_claim"`
	RoleMappings      map[string]string `json:"role_mappings"`
	GroupMappings     map[string]string `json:"group_mappings"`
	AutoCreateUsers   bool              `json:"auto_create_users"`
	// AutoLogin lets the console sign a visitor in silently (prompt=none) when
	// the Keycloak session is still valid. Off by default: a silent attempt is a
	// top-level redirect, and the place where one can happen has to be bound to
	// an administrator decision, not to a query parameter anyone can append.
	AutoLogin            bool       `json:"auto_login"`
	DefaultRole          string     `json:"default_role"`
	AllowedEmailDomains  []string   `json:"allowed_email_domains"`
	PrivateCAPEM         string     `json:"private_ca_pem,omitempty"`
	LastConnectionTestAt *time.Time `json:"last_connection_test_at,omitempty"`
	LastConnectionOK     bool       `json:"last_connection_ok"`
}

type OIDCAutoConfig struct {
	KeycloakURL    string
	Realm          string
	ClientID       string
	ApplicationURL string
	PrivateCAPEM   string
}

func DefaultOIDCSettings() OIDCSettings {
	return OIDCSettings{
		Scopes:              []string{oidc.ScopeOpenID, "profile", "email"},
		UsernameClaim:       "preferred_username",
		EmailClaim:          "email",
		NameClaim:           "name",
		GroupClaim:          "groups",
		RoleClaim:           "roles",
		RoleMappings:        map[string]string{},
		GroupMappings:       map[string]string{},
		AllowedEmailDomains: []string{},
		AutoCreateUsers:     true,
		DefaultRole:         "viewer",
	}
}

func (settings OIDCSettings) EffectiveIssuer() string {
	issuer := strings.TrimRight(strings.TrimSpace(settings.IssuerURL), "/")
	realm := strings.Trim(strings.TrimSpace(settings.Realm), "/")
	if realm != "" && !strings.Contains(issuer, "/realms/") {
		return issuer + "/realms/" + url.PathEscape(realm)
	}
	return issuer
}

// validateIssuer checks only what a discovery attempt needs, so the connection
// test can run against a configuration that is not enabled yet.
func (settings OIDCSettings) validateIssuer() error {
	if strings.TrimSpace(settings.IssuerURL) == "" {
		return errors.New("Keycloak URL is required")
	}
	issuer, err := url.Parse(settings.EffectiveIssuer())
	if err != nil || issuer.Scheme != "https" || issuer.Host == "" ||
		issuer.User != nil || issuer.RawQuery != "" || issuer.Fragment != "" {
		return errors.New("Keycloak issuer must be an HTTPS URL")
	}
	if strings.TrimSpace(settings.ClientID) == "" {
		return errors.New("Keycloak client ID is required")
	}
	if strings.TrimSpace(settings.PrivateCAPEM) != "" {
		roots := x509.NewCertPool()
		if !roots.AppendCertsFromPEM([]byte(settings.PrivateCAPEM)) {
			return errors.New("Keycloak private CA PEM is invalid")
		}
	}
	return nil
}

func (settings OIDCSettings) Validate() error {
	if !settings.Enabled {
		return nil
	}
	if err := settings.validateIssuer(); err != nil {
		return err
	}
	redirect, err := url.Parse(settings.RedirectURI)
	if err != nil || !secureBrowserEndpoint(redirect) || redirect.Fragment != "" {
		return errors.New("Keycloak redirect URI is invalid")
	}
	if strings.TrimSpace(settings.LogoutRedirectURI) != "" {
		logoutRedirect, logoutErr := url.Parse(settings.LogoutRedirectURI)
		if logoutErr != nil || !secureBrowserEndpoint(logoutRedirect) ||
			logoutRedirect.Fragment != "" {
			return errors.New("Keycloak logout redirect URI is invalid")
		}
	}
	if settings.UsernameClaim == "" {
		return errors.New("Keycloak username claim is required")
	}
	if len(settings.AllowedEmailDomains) > 0 && strings.TrimSpace(settings.EmailClaim) == "" {
		return errors.New("Keycloak email claim is required when domain filtering is enabled")
	}
	if len(settings.Scopes) == 0 || !containsString(settings.Scopes, oidc.ScopeOpenID) {
		return errors.New("Keycloak scopes must include openid")
	}
	for _, domain := range settings.AllowedEmailDomains {
		if strings.ContainsAny(domain, "@/ ") || strings.TrimSpace(domain) == "" {
			return errors.New("allowed email domain is invalid")
		}
	}
	if len(settings.RoleMappings) > 0 && strings.TrimSpace(settings.RoleClaim) == "" {
		return errors.New("Keycloak role claim is required when role mappings are configured")
	}
	if len(settings.GroupMappings) > 0 && strings.TrimSpace(settings.GroupClaim) == "" {
		return errors.New("Keycloak group claim is required when group mappings are configured")
	}
	for external, internal := range settings.RoleMappings {
		if strings.TrimSpace(external) == "" || strings.TrimSpace(internal) == "" {
			return errors.New("Keycloak role mapping contains an empty value")
		}
	}
	for external, internal := range settings.GroupMappings {
		if strings.TrimSpace(external) == "" || strings.TrimSpace(internal) == "" {
			return errors.New("Keycloak group mapping contains an empty value")
		}
	}
	if strings.TrimSpace(settings.PrivateCAPEM) != "" {
		roots := x509.NewCertPool()
		if !roots.AppendCertsFromPEM([]byte(settings.PrivateCAPEM)) {
			return errors.New("Keycloak private CA PEM is invalid")
		}
	}
	return nil
}

type OIDCStart struct {
	AuthorizationURL string `json:"authorization_url"`
	// Silent reports whether prompt=none was actually sent. A caller asking for
	// a silent attempt while auto_login is off gets an ordinary login instead.
	Silent bool `json:"silent"`
}

// Every error a provider returns to prompt=none when it has no usable session.
// These are ordinary answers, not failures: the visitor simply is not signed in
// at the provider and has to be shown the login screen.
var silentLoginRefusals = map[string]struct{}{
	"login_required":             {},
	"interaction_required":       {},
	"consent_required":           {},
	"account_selection_required": {},
}

// SilentLoginRefused reports whether providerError is a prompt=none refusal.
func SilentLoginRefused(providerError string) bool {
	_, refused := silentLoginRefusals[strings.TrimSpace(providerError)]
	return refused
}

type OIDCService struct {
	db             *sql.DB
	bootstrapStore *bootstrap.Store
	localAuth      *Service
	audit          audit.Recorder
}

func NewOIDCService(
	db *sql.DB,
	bootstrapStore *bootstrap.Store,
	localAuth *Service,
) *OIDCService {
	return &OIDCService{
		db:             db,
		bootstrapStore: bootstrapStore,
		localAuth:      localAuth,
	}
}

func (service *OIDCService) Settings(ctx context.Context) (OIDCSettings, error) {
	settings := DefaultOIDCSettings()
	var raw any
	if err := service.db.QueryRowContext(
		ctx,
		"SELECT value_json FROM settings WHERE key = $1",
		keycloakSettingKey,
	).Scan(&raw); errors.Is(err, sql.ErrNoRows) {
		return settings, nil
	} else if err != nil {
		return OIDCSettings{}, fmt.Errorf("read Keycloak settings: %w", err)
	}
	bytes, err := jsonBytes(raw)
	if err != nil {
		return OIDCSettings{}, err
	}
	if err := json.Unmarshal(bytes, &settings); err != nil {
		return OIDCSettings{}, fmt.Errorf("decode Keycloak settings: %w", err)
	}
	if settings.RoleMappings == nil {
		settings.RoleMappings = map[string]string{}
	}
	if settings.GroupMappings == nil {
		settings.GroupMappings = map[string]string{}
	}
	if settings.Scopes == nil {
		settings.Scopes = []string{oidc.ScopeOpenID, "profile", "email"}
	}
	if settings.AllowedEmailDomains == nil {
		settings.AllowedEmailDomains = []string{}
	}
	return settings, nil
}

// ClientSecretState distinguishes the three things that can be true of the
// stored client secret. They used to collapse into one boolean plus an error,
// and the console's fetch swallowed the error - so a secret that was present but
// undecryptable looked exactly like no secret at all: the Keycloak button simply
// disappeared after a restart with nothing said anywhere.
type ClientSecretState int

const (
	ClientSecretAbsent ClientSecretState = iota
	ClientSecretReady
	// ClientSecretUnreadable means a sealed secret is stored but this instance's
	// master key cannot open it. The usual cause is a replacement pod that
	// generated its own key because the state directory is not shared across
	// replicas, and every other sealed setting is in the same position.
	ClientSecretUnreadable
)

func (service *OIDCService) ClientSecretState(
	ctx context.Context,
) (ClientSecretState, error) {
	secret, err := service.clientSecret(ctx)
	if err != nil {
		if errors.Is(err, ErrOIDCSecretUnreadable) {
			return ClientSecretUnreadable, nil
		}
		return ClientSecretAbsent, err
	}
	if strings.TrimSpace(secret) == "" {
		return ClientSecretAbsent, nil
	}
	return ClientSecretReady, nil
}

func (service *OIDCService) ClientSecretConfigured(
	ctx context.Context,
) (bool, error) {
	state, err := service.ClientSecretState(ctx)
	if err != nil {
		return false, err
	}
	return state == ClientSecretReady, nil
}

func (service *OIDCService) SaveSettings(
	ctx context.Context,
	settings OIDCSettings,
	clientSecret *string,
	actor User,
	reason string,
) error {
	if err := settings.Validate(); err != nil {
		return err
	}
	if err := service.validateRoleMappings(ctx, settings); err != nil {
		return err
	}
	existingSecret, err := service.clientSecret(ctx)
	if err != nil {
		return err
	}
	if settings.Enabled {
		if clientSecret == nil && strings.TrimSpace(existingSecret) == "" {
			return ErrOIDCSecret
		}
		if clientSecret != nil && strings.TrimSpace(*clientSecret) == "" {
			return ErrOIDCSecret
		}
	}
	bytes, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("encode Keycloak settings: %w", err)
	}
	transaction, err := service.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin Keycloak settings update: %w", err)
	}
	defer transaction.Rollback()
	var previous any
	_ = transaction.QueryRowContext(
		ctx,
		"SELECT value_json FROM settings WHERE key = $1",
		keycloakSettingKey,
	).Scan(&previous)
	if _, err := transaction.ExecContext(
		ctx,
		`INSERT INTO settings(key, value_json, secret, apply_mode, version, updated_by)
		 VALUES ($1, $2, FALSE, 'new_login', 1, $3)
		 ON CONFLICT (key) DO UPDATE SET
		   value_json = excluded.value_json,
		   version = settings.version + 1,
		   updated_by = excluded.updated_by,
		   updated_at = CURRENT_TIMESTAMP`,
		keycloakSettingKey,
		string(bytes),
		actor.ID,
	); err != nil {
		return fmt.Errorf("save Keycloak settings: %w", err)
	}
	if clientSecret != nil {
		sealed, sealErr := service.bootstrapStore.SealString(
			keycloakClientSecretPurpose,
			*clientSecret,
		)
		if sealErr != nil {
			return fmt.Errorf("encrypt Keycloak client secret: %w", sealErr)
		}
		envelope, encodeErr := json.Marshal(oidcClientSecretEnvelope{
			Sealed: sealed,
		})
		if encodeErr != nil {
			return fmt.Errorf("encode Keycloak client secret: %w", encodeErr)
		}
		if _, secretErr := transaction.ExecContext(
			ctx,
			`INSERT INTO settings(
				key, value_json, secret, apply_mode, version, updated_by
			 ) VALUES ($1, $2, TRUE, 'immediate', 1, $3)
			 ON CONFLICT (key) DO UPDATE SET
				value_json = excluded.value_json,
				secret = TRUE,
				version = settings.version + 1,
				updated_by = excluded.updated_by,
				updated_at = CURRENT_TIMESTAMP`,
			keycloakClientSecretSettingKey,
			string(envelope),
			actor.ID,
		); secretErr != nil {
			return fmt.Errorf("save Keycloak client secret: %w", secretErr)
		}
	}
	if err := service.audit.Record(ctx, transaction, audit.Entry{
		ActorType:    "user",
		ActorID:      actor.ID,
		ActorName:    actor.Username,
		Action:       "settings.keycloak.update",
		ResourceType: "setting",
		ResourceID:   keycloakSettingKey,
		Result:       "success",
		Reason:       reason,
		Before:       maskKeycloakSettings(previous),
		After:        settings,
		Metadata: map[string]any{
			"client_secret_changed": clientSecret != nil,
		},
	}); err != nil {
		return err
	}
	return transaction.Commit()
}

func (service *OIDCService) TestConnection(
	ctx context.Context,
	settings OIDCSettings,
) error {
	// Validate() short-circuits for a disabled provider, but the natural order is
	// to test the connection before enabling it. Check the fields the test
	// actually needs so an empty form reports the missing field instead of a
	// discovery failure.
	if err := settings.validateIssuer(); err != nil {
		return err
	}
	if err := settings.Validate(); err != nil {
		return err
	}
	if err := service.validateRoleMappings(ctx, settings); err != nil {
		return err
	}
	oidcContext, err := oidcHTTPContext(ctx, settings.PrivateCAPEM)
	if err != nil {
		return err
	}
	if _, err := oidc.NewProvider(oidcContext, settings.EffectiveIssuer()); err != nil {
		return fmt.Errorf("%w: %s", ErrOIDCUnreachable, err.Error())
	}
	return nil
}

// AutomaticSettings builds a complete, enabled OIDC configuration from the
// four values an administrator normally knows. Discovery is completed before
// the caller persists either the settings or a client secret.
func (service *OIDCService) AutomaticSettings(
	ctx context.Context,
	input OIDCAutoConfig,
) (OIDCSettings, error) {
	settings, err := service.Settings(ctx)
	if err != nil {
		return OIDCSettings{}, err
	}
	applicationURL, err := url.Parse(strings.TrimSpace(input.ApplicationURL))
	if err != nil || !secureBrowserEndpoint(applicationURL) ||
		applicationURL.RawQuery != "" || applicationURL.Fragment != "" {
		return OIDCSettings{}, errors.New("InvenQor application URL is invalid")
	}
	applicationURL.Path = strings.TrimRight(applicationURL.Path, "/")
	base := strings.TrimRight(applicationURL.String(), "/")

	settings.Enabled = true
	settings.IssuerURL = strings.TrimRight(
		strings.TrimSpace(input.KeycloakURL),
		"/",
	)
	settings.Realm = strings.Trim(strings.TrimSpace(input.Realm), "/")
	settings.ClientID = strings.TrimSpace(input.ClientID)
	settings.RedirectURI = base + "/api/v1/auth/keycloak/callback"
	settings.LogoutRedirectURI = base + "/"
	if strings.TrimSpace(input.PrivateCAPEM) != "" {
		settings.PrivateCAPEM = strings.TrimSpace(input.PrivateCAPEM)
	}
	if settings.UsernameClaim == "" {
		settings.UsernameClaim = "preferred_username"
	}
	if settings.RoleClaim == "" || settings.RoleClaim == "roles" {
		settings.RoleClaim = "realm_access.roles"
	}
	if len(settings.Scopes) == 0 {
		settings.Scopes = []string{oidc.ScopeOpenID, "profile", "email"}
	}
	if err := service.TestConnection(ctx, settings); err != nil {
		return OIDCSettings{}, err
	}
	now := time.Now().UTC()
	settings.LastConnectionTestAt = &now
	settings.LastConnectionOK = true
	return settings, nil
}

// secureBrowserEndpoint permits plaintext HTTP only for a loopback deployment,
// where TLS is commonly terminated outside the developer machine. Redirects
// to any other HTTP host would leak authorization responses or logout state.
func secureBrowserEndpoint(endpoint *url.URL) bool {
	if endpoint == nil || endpoint.Host == "" || endpoint.User != nil {
		return false
	}
	switch strings.ToLower(endpoint.Scheme) {
	case "https":
		return true
	case "http":
		host := endpoint.Hostname()
		if strings.EqualFold(host, "localhost") {
			return true
		}
		address := net.ParseIP(host)
		return address != nil && address.IsLoopback()
	default:
		return false
	}
}

func (service *OIDCService) LogoutURL(
	ctx context.Context,
	userID string,
) (string, error) {
	settings, err := service.Settings(ctx)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(settings.LogoutRedirectURI) == "" ||
		strings.TrimSpace(settings.ClientID) == "" ||
		strings.TrimSpace(settings.EffectiveIssuer()) == "" {
		return "", nil
	}
	var linked int
	if err := service.db.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM external_identities
		  WHERE user_id=$1 AND provider='keycloak'`,
		userID,
	).Scan(&linked); err != nil {
		return "", fmt.Errorf("check Keycloak identity for logout: %w", err)
	}
	if linked == 0 {
		return "", nil
	}
	endpoint, err := url.Parse(
		strings.TrimRight(settings.EffectiveIssuer(), "/") +
			"/protocol/openid-connect/logout",
	)
	if err != nil {
		return "", fmt.Errorf("construct Keycloak logout URL: %w", err)
	}
	query := endpoint.Query()
	query.Set("client_id", settings.ClientID)
	query.Set("post_logout_redirect_uri", settings.LogoutRedirectURI)
	endpoint.RawQuery = query.Encode()
	return endpoint.String(), nil
}

func (service *OIDCService) validateRoleMappings(
	ctx context.Context,
	settings OIDCSettings,
) error {
	names := map[string]struct{}{}
	if role := strings.TrimSpace(settings.DefaultRole); role != "" {
		names[role] = struct{}{}
	}
	for _, role := range settings.RoleMappings {
		names[strings.TrimSpace(role)] = struct{}{}
	}
	for _, role := range settings.GroupMappings {
		names[strings.TrimSpace(role)] = struct{}{}
	}
	for name := range names {
		var count int
		if err := service.db.QueryRowContext(
			ctx,
			"SELECT COUNT(*) FROM roles WHERE name=$1",
			name,
		).Scan(&count); err != nil {
			return fmt.Errorf("validate Keycloak role mapping: %w", err)
		}
		if count != 1 {
			return fmt.Errorf("%w: %s", ErrOIDCRole, name)
		}
	}
	return nil
}

// Start begins an authorization-code flow. With silent set the provider is
// asked for prompt=none - answer from an existing session or refuse, never
// render a screen - but only when the administrator turned auto_login on;
// otherwise the request is quietly downgraded to an ordinary login so that a
// query parameter cannot change the flow.
func (service *OIDCService) Start(
	ctx context.Context,
	returnTo string,
	silent bool,
	sourceIP string,
	userAgent string,
) (OIDCStart, error) {
	settings, _, oauthConfig, err := service.provider(ctx)
	if err != nil {
		return OIDCStart{}, err
	}
	silent = silent && settings.AutoLogin
	state, _, err := newSecret()
	if err != nil {
		return OIDCStart{}, err
	}
	nonce, _, err := newSecret()
	if err != nil {
		return OIDCStart{}, err
	}
	verifier, _, err := newSecret()
	if err != nil {
		return OIDCStart{}, err
	}
	encryptedVerifier, err := service.bootstrapStore.SealString("oidc.pkce", verifier)
	if err != nil {
		return OIDCStart{}, err
	}
	if !safeReturnTo(returnTo) {
		returnTo = "/"
	}
	// Abandoned logins are the common case (a user closes the tab), so the table
	// only grows unless finished and expired rows are cleared here.
	if _, err := service.db.ExecContext(
		ctx,
		`DELETE FROM oidc_flows
		  WHERE expires_at < CURRENT_TIMESTAMP
		     OR consumed_at IS NOT NULL`,
	); err != nil {
		return OIDCStart{}, fmt.Errorf("prune expired OIDC flows: %w", err)
	}
	expiresAt := time.Now().UTC().Add(10 * time.Minute)
	if _, err := service.db.ExecContext(
		ctx,
		`INSERT INTO oidc_flows(
			id, state_hash, nonce_hash, pkce_verifier, redirect_uri,
			return_to, source_ip, user_agent, expires_at, silent
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		uuid.NewString(),
		hashSecret(state),
		hashSecret(nonce),
		encryptedVerifier,
		settings.RedirectURI,
		returnTo,
		sourceIP,
		userAgent,
		expiresAt,
		silent,
	); err != nil {
		return OIDCStart{}, fmt.Errorf("store OIDC flow: %w", err)
	}
	challengeHash := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(challengeHash[:])
	options := []oauth2.AuthCodeOption{
		oidc.Nonce(nonce),
		oauth2.SetAuthURLParam("code_challenge", challenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	}
	if silent {
		options = append(options, oauth2.SetAuthURLParam("prompt", "none"))
	}
	authorizationURL := oauthConfig.AuthCodeURL(state, options...)
	return OIDCStart{AuthorizationURL: authorizationURL, Silent: silent}, nil
}

// RefusedSilently answers the callback when the provider returned an error
// instead of a code. It reports true only for a flow this server started with
// prompt=none that the provider refused for lack of a session - the ordinary
// outcome for a visitor who is not signed in at Keycloak. The flow is consumed
// either way so the state cannot be replayed. Any other combination is a real
// failure and is left to the usual reporting.
func (service *OIDCService) RefusedSilently(
	ctx context.Context,
	state string,
	providerError string,
) (bool, error) {
	if state == "" {
		return false, nil
	}
	var silent bool
	var expiresAt, consumedAt flexibleTime
	err := service.db.QueryRowContext(
		ctx,
		`SELECT silent, expires_at, consumed_at FROM oidc_flows
		 WHERE state_hash = $1`,
		hashSecret(state),
	).Scan(&silent, &expiresAt, &consumedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("load OIDC flow for provider error: %w", err)
	}
	if !expiresAt.Valid || time.Now().UTC().After(expiresAt.Time) || consumedAt.Valid {
		return false, nil
	}
	if _, err := service.db.ExecContext(
		ctx,
		`UPDATE oidc_flows SET consumed_at = CURRENT_TIMESTAMP
		 WHERE state_hash = $1 AND consumed_at IS NULL`,
		hashSecret(state),
	); err != nil {
		return false, fmt.Errorf("consume refused OIDC flow: %w", err)
	}
	return silent && SilentLoginRefused(providerError), nil
}

func (service *OIDCService) Callback(
	ctx context.Context,
	state string,
	code string,
	sourceIP string,
	userAgent string,
	requestID string,
) (Session, string, error) {
	settings, provider, oauthConfig, err := service.provider(ctx)
	if err != nil {
		return Session{}, "", err
	}
	var flowID, nonceHash, encryptedVerifier, redirectURI, returnTo string
	var expiresAt, consumedAt flexibleTime
	err = service.db.QueryRowContext(
		ctx,
		`SELECT id, nonce_hash, pkce_verifier, redirect_uri, return_to,
		        expires_at, consumed_at
		 FROM oidc_flows
		 WHERE state_hash = $1`,
		hashSecret(state),
	).Scan(
		&flowID,
		&nonceHash,
		&encryptedVerifier,
		&redirectURI,
		&returnTo,
		&expiresAt,
		&consumedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Session{}, "", ErrOIDCFlow
	}
	if err != nil {
		return Session{}, "", fmt.Errorf("load OIDC flow: %w", err)
	}
	if !expiresAt.Valid || time.Now().UTC().After(expiresAt.Time) || consumedAt.Valid {
		return Session{}, "", ErrOIDCFlow
	}
	verifier, err := service.bootstrapStore.OpenString("oidc.pkce", encryptedVerifier)
	if err != nil {
		return Session{}, "", err
	}
	oidcContext, err := oidcHTTPContext(ctx, settings.PrivateCAPEM)
	if err != nil {
		return Session{}, "", err
	}
	token, err := oauthConfig.Exchange(
		oidcContext,
		code,
		oauth2.SetAuthURLParam("code_verifier", verifier),
		oauth2.SetAuthURLParam("redirect_uri", redirectURI),
	)
	if err != nil {
		return Session{}, "", fmt.Errorf("exchange Keycloak authorization code: %w", err)
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return Session{}, "", errors.New("Keycloak token response omitted id_token")
	}
	idToken, err := provider.Verifier(&oidc.Config{
		ClientID: settings.ClientID,
	}).Verify(oidcContext, rawIDToken)
	if err != nil {
		return Session{}, "", fmt.Errorf("verify Keycloak ID token: %w", err)
	}
	var claims map[string]any
	if err := idToken.Claims(&claims); err != nil {
		return Session{}, "", fmt.Errorf("decode Keycloak ID token claims: %w", err)
	}
	nonce, _ := claims["nonce"].(string)
	if nonce == "" || !subtleHashCompare(hashSecret(nonce), nonceHash) {
		return Session{}, "", ErrOIDCNonce
	}
	// Claim the single-use flow before anything is created. Consuming it after
	// the session was issued left an unreferenced session behind whenever two
	// callbacks raced on the same state value.
	result, err := service.db.ExecContext(
		ctx,
		`UPDATE oidc_flows
		 SET consumed_at = CURRENT_TIMESTAMP
		 WHERE id = $1 AND consumed_at IS NULL`,
		flowID,
	)
	if err != nil {
		return Session{}, "", fmt.Errorf("consume OIDC flow: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return Session{}, "", ErrOIDCFlow
	}
	user, err := service.provisionUser(ctx, settings, idToken.Subject, claims)
	if err != nil {
		return Session{}, "", err
	}
	user.Roles, user.Permissions, err = service.localAuth.rolesAndPermissions(ctx, user.ID)
	if err != nil {
		return Session{}, "", err
	}
	session, err := service.localAuth.createSession(
		ctx,
		user,
		sourceIP,
		userAgent,
		time.Now().UTC(),
	)
	if err != nil {
		return Session{}, "", err
	}
	if err := service.audit.Record(ctx, service.db, audit.Entry{
		ActorType:    "user",
		ActorID:      user.ID,
		ActorName:    user.Username,
		Action:       "auth.keycloak.login",
		ResourceType: "session",
		ResourceID:   session.ID,
		RequestID:    requestID,
		SourceIP:     sourceIP,
		UserAgent:    userAgent,
		Result:       "success",
		Metadata: map[string]any{
			"issuer":  settings.EffectiveIssuer(),
			"subject": idToken.Subject,
		},
	}); err != nil {
		return Session{}, "", err
	}
	return session, returnTo, nil
}

func (service *OIDCService) provider(
	ctx context.Context,
) (OIDCSettings, *oidc.Provider, *oauth2.Config, error) {
	settings, err := service.Settings(ctx)
	if err != nil {
		return OIDCSettings{}, nil, nil, err
	}
	if !settings.Enabled {
		return settings, nil, nil, ErrOIDCDisabled
	}
	if err := settings.Validate(); err != nil {
		return OIDCSettings{}, nil, nil, err
	}
	clientSecret, err := service.clientSecret(ctx)
	if err != nil {
		return OIDCSettings{}, nil, nil, err
	}
	if strings.TrimSpace(clientSecret) == "" {
		return OIDCSettings{}, nil, nil, ErrOIDCSecret
	}
	oidcContext, err := oidcHTTPContext(ctx, settings.PrivateCAPEM)
	if err != nil {
		return OIDCSettings{}, nil, nil, err
	}
	provider, err := oidc.NewProvider(oidcContext, settings.EffectiveIssuer())
	if err != nil {
		// Typed so the login handlers can report an unreachable identity
		// provider instead of an opaque internal error.
		return OIDCSettings{}, nil, nil, fmt.Errorf("%w: %s", ErrOIDCUnreachable, err.Error())
	}
	oauthConfig := &oauth2.Config{
		ClientID:     settings.ClientID,
		ClientSecret: clientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  settings.RedirectURI,
		Scopes:       settings.Scopes,
	}
	return settings, provider, oauthConfig, nil
}

// clientSecret reads the encrypted shared setting first. Releases before this
// storage existed kept the secret in bootstrap.enc, which is a Pod-local file
// in the StatefulSet. When such a value is found, migrate it with an
// insert-if-absent and then read the winner back. The shared master key mounted
// into every Pod makes the ciphertext usable without storing plaintext in
// PostgreSQL.
func (service *OIDCService) clientSecret(ctx context.Context) (string, error) {
	secret, found, err := service.sharedClientSecret(ctx)
	if err != nil || found {
		return secret, err
	}
	values, err := service.bootstrapStore.Load()
	if err != nil {
		return "", err
	}
	if values.KeycloakClientSecret == "" {
		return "", nil
	}
	sealed, err := service.bootstrapStore.SealString(
		keycloakClientSecretPurpose,
		values.KeycloakClientSecret,
	)
	if err != nil {
		return "", fmt.Errorf("encrypt legacy Keycloak client secret: %w", err)
	}
	envelope, err := json.Marshal(oidcClientSecretEnvelope{Sealed: sealed})
	if err != nil {
		return "", fmt.Errorf("encode legacy Keycloak client secret: %w", err)
	}
	if _, err := service.db.ExecContext(
		ctx,
		`INSERT INTO settings(key, value_json, secret, apply_mode, version)
		 VALUES ($1, $2, TRUE, 'immediate', 1)
		 ON CONFLICT (key) DO NOTHING`,
		keycloakClientSecretSettingKey,
		string(envelope),
	); err != nil {
		return "", fmt.Errorf("migrate legacy Keycloak client secret: %w", err)
	}
	secret, found, err = service.sharedClientSecret(ctx)
	if err != nil {
		return "", err
	}
	if !found {
		return "", errors.New("migrated Keycloak client secret is unavailable")
	}
	return secret, nil
}

func (service *OIDCService) sharedClientSecret(
	ctx context.Context,
) (string, bool, error) {
	var raw any
	err := service.db.QueryRowContext(
		ctx,
		`SELECT value_json FROM settings
		 WHERE key = $1 AND secret = TRUE`,
		keycloakClientSecretSettingKey,
	).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("read Keycloak client secret: %w", err)
	}
	encoded, err := jsonBytes(raw)
	if err != nil {
		return "", true, fmt.Errorf("decode Keycloak client secret setting: %w", err)
	}
	var envelope oidcClientSecretEnvelope
	if err := json.Unmarshal(encoded, &envelope); err != nil || envelope.Sealed == "" {
		return "", true, errors.New("decode Keycloak client secret setting")
	}
	secret, err := service.bootstrapStore.OpenString(
		keycloakClientSecretPurpose,
		envelope.Sealed,
	)
	if err != nil {
		// A stored secret this instance cannot open is a different situation
		// from no secret, and the operator has to be told which one it is.
		return "", true, fmt.Errorf("%w: %v", ErrOIDCSecretUnreadable, err)
	}
	return secret, true, nil
}

func (service *OIDCService) provisionUser(
	ctx context.Context,
	settings OIDCSettings,
	subject string,
	claims map[string]any,
) (User, error) {
	issuer := normalizedOIDCIssuer(settings.EffectiveIssuer())
	var user User
	var active, notDeleted bool
	lookup := func() error {
		return service.db.QueryRowContext(
			ctx,
			`SELECT u.id, u.username, u.display_name, u.email, u.super_admin,
			        u.active, CASE WHEN u.deleted_at IS NULL THEN TRUE ELSE FALSE END
			 FROM users u
			 JOIN external_identities e ON e.user_id = u.id
			 WHERE e.provider = 'keycloak' AND e.issuer = $1 AND e.subject = $2`,
			issuer,
			subject,
		).Scan(
			&user.ID,
			&user.Username,
			&user.DisplayName,
			&user.Email,
			&user.SuperAdmin,
			&active,
			&notDeleted,
		)
	}
	err := lookup()
	if errors.Is(err, sql.ErrNoRows) {
		migrated, migrationErr := service.migrateLegacyOIDCIdentity(
			ctx,
			issuer,
			subject,
		)
		if migrationErr != nil {
			return User{}, migrationErr
		}
		if migrated {
			err = lookup()
		}
	}
	if err == nil {
		if !active || !notDeleted {
			return User{}, ErrOIDCUserInactive
		}
		email := claimString(claims, settings.EmailClaim)
		if !allowedEmail(email, settings.AllowedEmailDomains) {
			return User{}, ErrOIDCDomain
		}
		if email == "" {
			email = user.Email
		}
		displayName := claimString(claims, settings.NameClaim)
		if displayName == "" {
			displayName = user.DisplayName
		}
		claimsJSON, _ := json.Marshal(claims)
		transaction, beginErr := service.db.BeginTx(ctx, nil)
		if beginErr != nil {
			return User{}, fmt.Errorf("begin Keycloak user synchronization: %w", beginErr)
		}
		defer transaction.Rollback()
		if _, updateErr := transaction.ExecContext(
			ctx,
			`UPDATE external_identities
			 SET claims_json = $1, last_login_at = CURRENT_TIMESTAMP
			 WHERE provider = 'keycloak' AND issuer = $2 AND subject = $3`,
			string(claimsJSON),
			issuer,
			subject,
		); updateErr != nil {
			return User{}, fmt.Errorf("update Keycloak identity: %w", updateErr)
		}
		superAdmin, syncErr := replaceKeycloakRoles(
			ctx,
			transaction,
			user.ID,
			mappedRoles(settings, claims),
		)
		if syncErr != nil {
			return User{}, syncErr
		}
		if _, updateErr := transaction.ExecContext(
			ctx,
			`UPDATE users
			    SET display_name=$1,email=$2,super_admin=$3,
			        updated_at=CURRENT_TIMESTAMP
			  WHERE id=$4`,
			displayName,
			email,
			superAdmin,
			user.ID,
		); updateErr != nil {
			return User{}, fmt.Errorf("synchronize Keycloak user profile: %w", updateErr)
		}
		if commitErr := transaction.Commit(); commitErr != nil {
			return User{}, fmt.Errorf("commit Keycloak user synchronization: %w", commitErr)
		}
		user.DisplayName = displayName
		user.Email = email
		user.SuperAdmin = superAdmin
		return user, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return User{}, fmt.Errorf("lookup Keycloak identity: %w", err)
	}
	if !settings.AutoCreateUsers {
		return User{}, ErrOIDCProvisioning
	}
	username := claimString(claims, settings.UsernameClaim)
	username, err = validateUsername(username)
	if err != nil {
		return User{}, ErrOIDCUsername
	}
	email := claimString(claims, settings.EmailClaim)
	if !allowedEmail(email, settings.AllowedEmailDomains) {
		return User{}, ErrOIDCDomain
	}
	displayName := claimString(claims, settings.NameClaim)
	transaction, err := service.db.BeginTx(ctx, nil)
	if err != nil {
		return User{}, fmt.Errorf("begin Keycloak user provisioning: %w", err)
	}
	defer transaction.Rollback()
	// Auto-linking by name would let a renamed directory account take over a
	// local one, so refuse instead - but say which name collided.
	var taken int
	if err := transaction.QueryRowContext(
		ctx,
		"SELECT COUNT(*) FROM users WHERE normalized_username = $1",
		normalizeUsername(username),
	).Scan(&taken); err != nil {
		return User{}, fmt.Errorf("check Keycloak username availability: %w", err)
	}
	if taken != 0 {
		return User{}, fmt.Errorf("%w: %s", ErrOIDCUsernameTaken, username)
	}
	user = User{
		ID:          uuid.NewString(),
		Username:    username,
		DisplayName: displayName,
		Email:       email,
	}
	if _, err := transaction.ExecContext(
		ctx,
		`INSERT INTO users(
			id, username, normalized_username, display_name, email,
			active, super_admin
		) VALUES ($1, $2, $3, $4, $5, TRUE, FALSE)`,
		user.ID,
		user.Username,
		normalizeUsername(user.Username),
		user.DisplayName,
		user.Email,
	); err != nil {
		return User{}, fmt.Errorf("create Keycloak user: %w", err)
	}
	claimsJSON, _ := json.Marshal(claims)
	if _, err := transaction.ExecContext(
		ctx,
		`INSERT INTO external_identities(
			id, user_id, provider, issuer, subject, claims_json, last_login_at
		) VALUES ($1, $2, 'keycloak', $3, $4, $5, CURRENT_TIMESTAMP)`,
		uuid.NewString(),
		user.ID,
		issuer,
		subject,
		string(claimsJSON),
	); err != nil {
		return User{}, fmt.Errorf("link Keycloak identity: %w", err)
	}
	superAdmin, err := replaceKeycloakRoles(
		ctx,
		transaction,
		user.ID,
		mappedRoles(settings, claims),
	)
	if err != nil {
		return User{}, err
	}
	if _, err := transaction.ExecContext(
		ctx,
		"UPDATE users SET super_admin=$1 WHERE id=$2",
		superAdmin,
		user.ID,
	); err != nil {
		return User{}, fmt.Errorf("synchronize Keycloak super administrator flag: %w", err)
	}
	user.SuperAdmin = superAdmin
	if err := transaction.Commit(); err != nil {
		return User{}, fmt.Errorf("commit Keycloak user provisioning: %w", err)
	}
	return user, nil
}

// migrateLegacyOIDCIdentity upgrades the v0.2.14 identity row, which did not
// have an issuer column, only when its persisted and cryptographically
// verified ID-token claims prove which issuer originally owned the subject.
// A row with missing or different issuer claims is intentionally left alone:
// after a realm/configuration change, trusting only a repeated `sub` would let
// the new realm take over the old local account.
func (service *OIDCService) migrateLegacyOIDCIdentity(
	ctx context.Context,
	issuer string,
	subject string,
) (bool, error) {
	transaction, err := service.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("begin legacy Keycloak identity migration: %w", err)
	}
	defer transaction.Rollback()
	var identityID string
	var rawClaims any
	err = transaction.QueryRowContext(
		ctx,
		`SELECT id, claims_json FROM external_identities
		 WHERE provider='keycloak' AND issuer='' AND subject=$1`,
		subject,
	).Scan(&identityID, &rawClaims)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read legacy Keycloak identity: %w", err)
	}
	claimsJSON, err := jsonBytes(rawClaims)
	if err != nil {
		return false, fmt.Errorf("decode legacy Keycloak identity claims: %w", err)
	}
	var storedClaims map[string]any
	if err := json.Unmarshal(claimsJSON, &storedClaims); err != nil {
		return false, fmt.Errorf("decode legacy Keycloak identity claims: %w", err)
	}
	storedIssuer := normalizedOIDCIssuer(claimString(storedClaims, "iss"))
	if storedIssuer == "" || storedIssuer != issuer {
		return false, nil
	}
	result, err := transaction.ExecContext(
		ctx,
		`UPDATE external_identities SET issuer=$1
		 WHERE id=$2 AND provider='keycloak' AND issuer='' AND subject=$3`,
		issuer,
		identityID,
		subject,
	)
	if err != nil {
		return false, fmt.Errorf("migrate legacy Keycloak identity: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("confirm legacy Keycloak identity migration: %w", err)
	}
	if rows != 1 {
		return false, nil
	}
	if err := transaction.Commit(); err != nil {
		return false, fmt.Errorf("commit legacy Keycloak identity migration: %w", err)
	}
	return true, nil
}

func normalizedOIDCIssuer(value string) string {
	return strings.TrimRight(strings.TrimSpace(value), "/")
}

func replaceKeycloakRoles(
	ctx context.Context,
	transaction *sql.Tx,
	userID string,
	roleNames []string,
) (bool, error) {
	if _, err := transaction.ExecContext(
		ctx,
		"DELETE FROM user_roles WHERE user_id=$1 AND source='keycloak'",
		userID,
	); err != nil {
		return false, fmt.Errorf("clear Keycloak role grants: %w", err)
	}
	for _, roleName := range roleNames {
		result, err := transaction.ExecContext(
			ctx,
			`INSERT INTO user_roles(user_id, role_id, source)
			 SELECT $1,id,'keycloak' FROM roles WHERE name=$2
			 ON CONFLICT (user_id, role_id, source) DO NOTHING`,
			userID,
			roleName,
		)
		if err != nil {
			return false, fmt.Errorf("grant mapped Keycloak role: %w", err)
		}
		if affected, _ := result.RowsAffected(); affected != 1 {
			return false, fmt.Errorf("%w: %s", ErrOIDCRole, roleName)
		}
	}
	var superAdmin bool
	if err := transaction.QueryRowContext(
		ctx,
		`SELECT CASE WHEN EXISTS(
		    SELECT 1 FROM user_roles ur JOIN roles r ON r.id=ur.role_id
		     WHERE ur.user_id=$1 AND r.name='super_admin'
		  ) THEN TRUE ELSE FALSE END`,
		userID,
	).Scan(&superAdmin); err != nil {
		return false, fmt.Errorf("resolve super administrator role: %w", err)
	}
	return superAdmin, nil
}

func oidcHTTPContext(ctx context.Context, privateCAPEM string) (context.Context, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if strings.TrimSpace(privateCAPEM) != "" {
		roots, err := x509.SystemCertPool()
		if err != nil {
			return nil, fmt.Errorf("load system CA pool: %w", err)
		}
		if !roots.AppendCertsFromPEM([]byte(privateCAPEM)) {
			return nil, errors.New("Keycloak private CA PEM is invalid")
		}
		if transport.TLSClientConfig == nil {
			transport.TLSClientConfig = new(tls.Config)
		} else {
			transport.TLSClientConfig = transport.TLSClientConfig.Clone()
		}
		transport.TLSClientConfig.RootCAs = roots
	}
	client := &http.Client{Transport: transport, Timeout: 15 * time.Second}
	return oidc.ClientContext(ctx, client), nil
}

func jsonBytes(value any) ([]byte, error) {
	switch typed := value.(type) {
	case []byte:
		return typed, nil
	case string:
		return []byte(typed), nil
	default:
		return nil, fmt.Errorf("unsupported JSON database type %T", value)
	}
}

func maskKeycloakSettings(value any) any {
	if value == nil {
		return nil
	}
	bytes, err := jsonBytes(value)
	if err != nil {
		return map[string]any{"masked": true}
	}
	var decoded map[string]any
	if json.Unmarshal(bytes, &decoded) != nil {
		return map[string]any{"masked": true}
	}
	delete(decoded, "private_ca_pem")
	return decoded
}

func claimString(claims map[string]any, name string) string {
	value, _ := claimValue(claims, name).(string)
	return strings.TrimSpace(value)
}

func claimStrings(claims map[string]any, name string) []string {
	switch value := claimValue(claims, name).(type) {
	case string:
		return []string{value}
	case []any:
		var result []string
		for _, item := range value {
			if text, ok := item.(string); ok {
				result = append(result, text)
			}
		}
		return result
	case []string:
		return value
	default:
		return nil
	}
}

func claimValue(claims map[string]any, name string) any {
	var value any = claims
	for _, component := range strings.Split(name, ".") {
		current, ok := value.(map[string]any)
		if !ok {
			return nil
		}
		value, ok = current[component]
		if !ok {
			return nil
		}
	}
	return value
}

func mappedRoles(settings OIDCSettings, claims map[string]any) []string {
	unique := map[string]struct{}{}
	for _, external := range claimStrings(claims, settings.RoleClaim) {
		if internal := settings.RoleMappings[external]; internal != "" {
			unique[internal] = struct{}{}
		}
	}
	for _, external := range claimStrings(claims, settings.GroupClaim) {
		if internal := settings.GroupMappings[external]; internal != "" {
			unique[internal] = struct{}{}
		}
	}
	if len(unique) == 0 && settings.DefaultRole != "" {
		unique[settings.DefaultRole] = struct{}{}
	}
	result := make([]string, 0, len(unique))
	for role := range unique {
		result = append(result, role)
	}
	sort.Strings(result)
	return result
}

func allowedEmail(email string, domains []string) bool {
	if len(domains) == 0 {
		return true
	}
	_, domain, found := strings.Cut(strings.ToLower(email), "@")
	if !found {
		return false
	}
	for _, allowed := range domains {
		if domain == strings.ToLower(strings.TrimSpace(allowed)) {
			return true
		}
	}
	return false
}

func safeReturnTo(value string) bool {
	if value == "" {
		return false
	}
	candidate := value
	// Decode repeatedly so a double-encoded backslash or authority prefix
	// cannot become dangerous only after another proxy/browser decoding pass.
	for {
		decoded, err := url.PathUnescape(candidate)
		if err != nil {
			return false
		}
		if decoded == candidate {
			break
		}
		candidate = decoded
	}
	for _, current := range []string{value, candidate} {
		if strings.Contains(current, `\`) ||
			!strings.HasPrefix(current, "/") ||
			strings.HasPrefix(current, "//") {
			return false
		}
		for _, character := range current {
			if character < 0x20 || character == 0x7f {
				return false
			}
		}
	}
	parsed, err := url.Parse(candidate)
	if err != nil || parsed.IsAbs() || parsed.Host != "" ||
		parsed.User != nil || parsed.Opaque != "" {
		return false
	}
	return strings.HasPrefix(parsed.Path, "/") &&
		!strings.HasPrefix(parsed.Path, "//") &&
		!strings.Contains(parsed.Path, `\`)
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func subtleHashCompare(left, right string) bool {
	if len(left) != len(right) {
		return false
	}
	var result byte
	for index := range left {
		result |= left[index] ^ right[index]
	}
	return result == 0
}
