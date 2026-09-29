package app

import (
	"context"
	"crypto/subtle"
	"fmt"
	"net/http"
	"net/mail"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-chi/chi/v5"
	"github.com/hkjang/seaton/internal/platform"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
)

const sessionCookie = "seaton_session"

func (s *Server) EnsureBootstrapAdmin(ctx context.Context, username, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return err
	}
	email := ""
	if strings.Contains(username, "@") {
		email = username
	}
	_, err = s.db.Exec(ctx, `INSERT INTO users(id, username, password_hash, display_name, email, role, source)
		VALUES($1,$2,$3,$2,NULLIF($4,''),'system_admin','local') ON CONFLICT (username) DO NOTHING`, newID(), username, string(hash), email)
	return err
}

func (s *Server) authConfig(w http.ResponseWriter, r *http.Request) {
	serviceName, _ := s.getSetting(r.Context(), "general.service_name")
	companyName, _ := s.getSetting(r.Context(), "general.company_name")
	local, _ := s.getSetting(r.Context(), "auth.local_enabled")
	oidcEnabled, _ := s.getSetting(r.Context(), "oidc.enabled")
	autoLogin, _ := s.getSetting(r.Context(), "oidc.auto_login")
	// oidcAutoLogin 은 브라우저가 로그인 화면을 그리기 전에 조용한 SSO 를
	// 시도할지 정하는 데 쓴다. 시도 여부 자체는 관리자 설정에만 묶인다.
	writeJSON(w, http.StatusOK, map[string]any{
		"serviceName": serviceName, "companyName": companyName,
		"localEnabled": local == "true", "oidcEnabled": oidcEnabled == "true",
		"oidcAutoLogin": oidcEnabled == "true" && autoLogin == "true",
		"version":       map[string]string{"version": s.version, "commit": s.commit},
	})
}

func (s *Server) localLogin(w http.ResponseWriter, r *http.Request) {
	local, _ := s.getSetting(r.Context(), "auth.local_enabled")
	if local != "true" {
		writeError(w, http.StatusForbidden, "local_login_disabled", "로컬 로그인이 비활성화되어 있습니다")
		return
	}
	var in struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	var hash string
	u, err := scanUser(s.db.QueryRow(r.Context(), `SELECT id,username,display_name,COALESCE(email,''),employee_id,role,source,last_login_at,active FROM users WHERE lower(username)=lower($1) AND active=true`, strings.TrimSpace(in.Username)))
	if err == nil {
		err = s.db.QueryRow(r.Context(), `SELECT COALESCE(password_hash,'') FROM users WHERE id=$1`, u.ID).Scan(&hash)
	}
	if err != nil || hash == "" || bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.Password)) != nil {
		time.Sleep(250 * time.Millisecond)
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "아이디 또는 비밀번호를 확인하세요")
		return
	}
	if err := s.issueSession(w, r, u); err != nil {
		writeError(w, http.StatusInternalServerError, "session_error", "로그인 세션을 만들지 못했습니다")
		return
	}
	s.audit(r.Context(), u.ID, "auth.login", "user", u.ID, r.RemoteAddr, map[string]string{"source": "local"})
	writeJSON(w, http.StatusOK, u)
}

func (s *Server) issueSession(w http.ResponseWriter, r *http.Request, u User) error {
	raw, err := platform.RandomToken(32)
	if err != nil {
		return err
	}
	csrf, err := platform.RandomToken(24)
	if err != nil {
		return err
	}
	hours := 8
	if v, _ := s.getSetting(r.Context(), "security.session_hours"); v != "" {
		if parsed, e := strconv.Atoi(v); e == nil && parsed >= 1 && parsed <= 720 {
			hours = parsed
		}
	}
	expires := time.Now().Add(time.Duration(hours) * time.Hour)
	if _, err := s.db.Exec(r.Context(), `INSERT INTO sessions(token_hash,user_id,csrf_token,expires_at,ip_address,user_agent) VALUES($1,$2,$3,$4,$5,$6)`, s.keys.Digest(raw), u.ID, csrf, expires, r.RemoteAddr, r.UserAgent()); err != nil {
		return err
	}
	_, _ = s.db.Exec(r.Context(), `UPDATE users SET last_login_at=now() WHERE id=$1`, u.ID)
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: raw, Path: "/", HttpOnly: true, Secure: requestIsHTTPS(r), SameSite: http.SameSiteLaxMode, Expires: expires, MaxAge: int(time.Until(expires).Seconds())})
	return nil
}

func requestIsHTTPS(r *http.Request) bool {
	return r.TLS != nil || strings.EqualFold(strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Proto"), ",")[0]), "https")
}

func (s *Server) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var u User
		var csrf string
		var apiScopes []string
		// bearerAuth 는 Authorization 헤더로 인증됐다는 뜻이다 — 개인 키든 SSO
		// 액세스 토큰이든. 둘 다 세션 쿠키와 CSRF 검사를 지나지 않는다.
		bearerAuth := false
		if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
			raw := strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
			switch {
			case strings.HasPrefix(raw, "seat_"):
				var keyID string
				var err error
				u, err = scanUser(s.db.QueryRow(r.Context(), `SELECT u.id,u.username,u.display_name,COALESCE(u.email,''),u.employee_id,u.role,u.source,u.last_login_at,u.active
					FROM api_keys k JOIN users u ON u.id=k.user_id
					WHERE k.secret_hash=$1 AND u.active=true AND (k.expires_at IS NULL OR k.expires_at>now()) AND (k.revoked_at IS NULL OR k.grace_until>now())`, s.keys.Digest(raw)))
				if err == nil {
					err = s.db.QueryRow(r.Context(), `SELECT id,scopes FROM api_keys WHERE secret_hash=$1`, s.keys.Digest(raw)).Scan(&keyID, &apiScopes)
					_, _ = s.db.Exec(r.Context(), `UPDATE api_keys SET last_used_at=now() WHERE id=$1`, keyID)
				}
				if err != nil {
					s.unauthorized(w, r, "invalid_api_key", "API 키가 유효하지 않습니다")
					return
				}
				bearerAuth = true
				if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions && r.URL.Path != "/mcp" && !containsString(apiScopes, "write") {
					writeError(w, http.StatusForbidden, "insufficient_scope", "write 범위가 있는 API 키가 필요합니다")
					return
				}
			case r.URL.Path == "/mcp" && looksLikeJWT(raw):
				// 키가 아닌 다른 하나는 Keycloak 액세스 토큰이다 — MCP 클라이언트가
				// OAuth 로 받아 온다(mcpoauth.go). /mcp 에서만, 켜져 있을 때만 받고,
				// 꺼져 있으면 아래 세션 검사로 떨어져 전과 똑같이 거절된다.
				if cfg := s.mcpOAuthConfig(r.Context()); cfg.active() {
					var err error
					u, apiScopes, err = s.oauthPrincipal(r.Context(), cfg, raw)
					if err != nil {
						s.refuseMCPToken(w, r, cfg, err)
						return
					}
					bearerAuth = true
				}
			}
		}
		if !bearerAuth {
			cookie, err := r.Cookie(sessionCookie)
			if err != nil || cookie.Value == "" {
				s.unauthorized(w, r, "authentication_required", "로그인이 필요합니다")
				return
			}
			u, err = scanUser(s.db.QueryRow(r.Context(), `SELECT u.id,u.username,u.display_name,COALESCE(u.email,''),u.employee_id,u.role,u.source,u.last_login_at,u.active
				FROM sessions s JOIN users u ON u.id=s.user_id WHERE s.token_hash=$1 AND s.expires_at>now() AND u.active=true`, s.keys.Digest(cookie.Value)))
			if err != nil {
				http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: "/", MaxAge: -1, HttpOnly: true})
				s.unauthorized(w, r, "session_expired", "세션이 만료되었습니다")
				return
			}
			_ = s.db.QueryRow(r.Context(), `SELECT csrf_token FROM sessions WHERE token_hash=$1`, s.keys.Digest(cookie.Value)).Scan(&csrf)
			if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions && r.URL.Path != "/api/v1/auth/logout" {
				provided := r.Header.Get("X-CSRF-Token")
				if provided == "" || subtle.ConstantTimeCompare([]byte(provided), []byte(csrf)) != 1 {
					writeError(w, http.StatusForbidden, "csrf_failed", "보안 토큰이 유효하지 않습니다")
					return
				}
			}
		}
		ctx := context.WithValue(r.Context(), userContextKey, u)
		ctx = context.WithValue(ctx, csrfContextKey, csrf)
		ctx = context.WithValue(ctx, apiScopesContextKey, apiScopes)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func containsString(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}

func (s *Server) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, _ := userFrom(r)
		if !u.IsAdmin() {
			writeError(w, http.StatusForbidden, "admin_required", "시스템 관리자 권한이 필요합니다")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) requireSeatManager(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, _ := userFrom(r)
		if !u.CanManageSeats() {
			writeError(w, http.StatusForbidden, "seat_manager_required", "좌석 관리자 권한이 필요합니다")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	u, _ := userFrom(r)
	csrf, _ := r.Context().Value(csrfContextKey).(string)
	writeJSON(w, http.StatusOK, map[string]any{"user": u, "csrfToken": csrf, "version": map[string]string{"version": s.version, "commit": s.commit, "builtAt": s.builtAt}})
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		_, _ = s.db.Exec(r.Context(), `DELETE FROM sessions WHERE token_hash=$1`, s.keys.Digest(c.Value))
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: requestIsHTTPS(r), SameSite: http.SameSiteLaxMode})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) changePassword(w http.ResponseWriter, r *http.Request) {
	if csrf, _ := r.Context().Value(csrfContextKey).(string); csrf == "" {
		writeError(w, http.StatusForbidden, "browser_session_required", "비밀번호 변경에는 브라우저 세션이 필요합니다")
		return
	}
	var in struct {
		CurrentPassword string `json:"currentPassword"`
		NewPassword     string `json:"newPassword"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	if len(in.NewPassword) < 12 {
		writeError(w, http.StatusBadRequest, "weak_password", "새 비밀번호는 12자 이상이어야 합니다")
		return
	}
	u, _ := userFrom(r)
	var hash, source string
	if err := s.db.QueryRow(r.Context(), `SELECT COALESCE(password_hash,''),source FROM users WHERE id=$1`, u.ID).Scan(&hash, &source); err != nil {
		notFoundOrServer(w, err)
		return
	}
	if source != "local" || hash == "" {
		writeError(w, http.StatusBadRequest, "oidc_user", "SSO 사용자의 비밀번호는 Keycloak에서 변경하세요")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.CurrentPassword)) != nil {
		writeError(w, http.StatusUnauthorized, "invalid_password", "현재 비밀번호가 일치하지 않습니다")
		return
	}
	newHash, err := bcrypt.GenerateFromPassword([]byte(in.NewPassword), 12)
	if err != nil {
		notFoundOrServer(w, err)
		return
	}
	_, err = s.db.Exec(r.Context(), `UPDATE users SET password_hash=$2,updated_at=now() WHERE id=$1`, u.ID, string(newHash))
	if err != nil {
		notFoundOrServer(w, err)
		return
	}
	s.audit(r.Context(), u.ID, "auth.password_change", "user", u.ID, r.RemoteAddr, nil)
	_, _ = s.db.Exec(r.Context(), `DELETE FROM sessions WHERE user_id=$1 AND token_hash<>$2`, u.ID, currentSessionDigest(r, s.keys))
	w.WriteHeader(http.StatusNoContent)
}

func currentSessionDigest(r *http.Request, keys *platform.Keyring) []byte {
	if c, err := r.Cookie(sessionCookie); err == nil {
		return keys.Digest(c.Value)
	}
	return nil
}

func (s *Server) oidcStart(w http.ResponseWriter, r *http.Request) {
	if enabled, _ := s.getSetting(r.Context(), "oidc.enabled"); enabled != "true" {
		writeError(w, http.StatusNotFound, "oidc_disabled", "SSO가 설정되지 않았습니다")
		return
	}
	issuer, _ := s.getSetting(r.Context(), "oidc.issuer_url")
	clientID, _ := s.getSetting(r.Context(), "oidc.client_id")
	secret, _ := s.getSetting(r.Context(), "oidc.client_secret")
	provider, err := oidc.NewProvider(r.Context(), issuer)
	if err != nil {
		writeError(w, http.StatusBadGateway, "oidc_discovery_failed", "Keycloak 연결을 확인하세요")
		return
	}
	state, _ := platform.RandomToken(32)
	nonce, _ := platform.RandomToken(24)
	verifier := oauth2.GenerateVerifier()
	returnTo := r.URL.Query().Get("returnTo")
	if !safeReturnTo(returnTo) {
		returnTo = "/"
	}
	// prompt=none 은 제공자에게 "있는 세션으로만 답하라"고 요구한다. 화면을
	// 절대 그리지 않으므로 인가 코드가 곧바로 오거나 login_required 가 온다.
	// 관리자가 auto_login 을 켜지 않았으면 조용히 평범한 로그인으로 바꾼다 —
	// 주소에 ?prompt=none 을 붙이는 것만으로 흐름이 달라져서는 안 된다.
	autoLogin, _ := s.getSetting(r.Context(), "oidc.auto_login")
	silent := silentLoginRequested(r, autoLogin == "true")
	_, err = s.db.Exec(r.Context(), `INSERT INTO oidc_states(state_hash,nonce,verifier,return_to,silent,expires_at) VALUES($1,$2,$3,$4,$5,now()+interval '10 minutes')`, s.keys.Digest(state), nonce, verifier, returnTo, silent)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "oidc_state_failed", "SSO 요청을 시작하지 못했습니다")
		return
	}
	cfg := oauth2.Config{ClientID: clientID, ClientSecret: secret, Endpoint: provider.Endpoint(), RedirectURL: requestBaseURL(r) + "/api/v1/auth/oidc/callback", Scopes: s.oidcScopes(r.Context())}
	http.Redirect(w, r, oidcAuthCodeURL(cfg, state, nonce, verifier, silent), http.StatusFound)
}

// silentLoginRequested 는 이 시작 요청을 prompt=none 으로 보낼지 정한다.
// 요청이 원해도 관리자 설정(auto_login)이 꺼져 있으면 거절한다.
func silentLoginRequested(r *http.Request, autoLogin bool) bool {
	return autoLogin && r.URL.Query().Get("prompt") == "none"
}

func oidcAuthCodeURL(cfg oauth2.Config, state, nonce, verifier string, silent bool) string {
	opts := []oauth2.AuthCodeOption{oidc.Nonce(nonce), oauth2.S256ChallengeOption(verifier)}
	if silent {
		opts = append(opts, oauth2.SetAuthURLParam("prompt", "none"))
	}
	return cfg.AuthCodeURL(state, opts...)
}

// safeReturnTo 는 로그인 뒤 돌아갈 자리를 같은 사이트 안으로 묶는다. '/' 로
// 시작하고 '//' 로 시작하지 않는 경로만 받는다 — 그러지 않으면 이 로그인
// 흐름이 밖으로 내보내는 발판이 된다. 브라우저는 '/\' 를 '//' 로 읽으므로
// 그것도 같이 막는다.
func safeReturnTo(value string) bool {
	if value == "" || !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") || strings.HasPrefix(value, `/\`) || strings.ContainsAny(value, "\r\n") {
		return false
	}
	parsed, err := url.Parse(value)
	return err == nil && !parsed.IsAbs() && parsed.Host == ""
}

// silentSSOErrors 는 prompt=none 에 세션이 없을 때 제공자가 돌려주는 대답이다.
// 실패가 아니라 "로그인 화면을 보여 달라"는 뜻이다.
var silentSSOErrors = map[string]bool{"login_required": true, "interaction_required": true, "consent_required": true}

// oidcErrorRedirect 는 제공자가 error 로 돌아왔을 때 브라우저를 보낼 자리다.
// 조용한 시도였으면 /login?sso=none 으로 보내 주소에 표시를 남긴다 —
// 브라우저 저장소가 지워졌더라도 이 표시가 있으면 다시 시도하지 않는다.
// 조용한 시도가 세션 없음 외의 이유로 거절되면 표시와 함께 오류도 알린다.
func oidcErrorRedirect(silent bool, providerError string) string {
	if !silent {
		return "/login?error=" + url.QueryEscape(providerError)
	}
	if silentSSOErrors[providerError] {
		return "/login?sso=none"
	}
	return "/login?sso=none&error=" + url.QueryEscape(providerError)
}

func (s *Server) oidcCallback(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	if e := r.URL.Query().Get("error"); e != "" {
		// 제공자는 거절을 코드 대신 error 로 알린다. prompt=none 은 세션이 없을
		// 때마다 login_required 를 보내는데, 그것은 평범한 대답이지 실패가 아니다.
		silent := false
		if state != "" {
			_ = s.db.QueryRow(r.Context(), `DELETE FROM oidc_states WHERE state_hash=$1 RETURNING silent`, s.keys.Digest(state)).Scan(&silent)
		}
		http.Redirect(w, r, oidcErrorRedirect(silent, e), http.StatusFound)
		return
	}
	var nonce, verifier, returnTo string
	err := s.db.QueryRow(r.Context(), `DELETE FROM oidc_states WHERE state_hash=$1 AND expires_at>now() RETURNING nonce,verifier,return_to`, s.keys.Digest(state)).Scan(&nonce, &verifier, &returnTo)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_oidc_state", "SSO 요청이 만료되었거나 유효하지 않습니다")
		return
	}
	issuer, _ := s.getSetting(r.Context(), "oidc.issuer_url")
	clientID, _ := s.getSetting(r.Context(), "oidc.client_id")
	secret, _ := s.getSetting(r.Context(), "oidc.client_secret")
	provider, err := oidc.NewProvider(r.Context(), issuer)
	if err != nil {
		writeError(w, http.StatusBadGateway, "oidc_discovery_failed", "Keycloak 연결을 확인하세요")
		return
	}
	cfg := oauth2.Config{ClientID: clientID, ClientSecret: secret, Endpoint: provider.Endpoint(), RedirectURL: requestBaseURL(r) + "/api/v1/auth/oidc/callback", Scopes: s.oidcScopes(r.Context())}
	token, err := cfg.Exchange(r.Context(), r.URL.Query().Get("code"), oauth2.VerifierOption(verifier))
	if err != nil {
		writeError(w, http.StatusUnauthorized, "oidc_exchange_failed", "SSO 인증 코드를 확인하지 못했습니다")
		return
	}
	rawID, ok := token.Extra("id_token").(string)
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing_id_token", "Keycloak ID 토큰이 없습니다")
		return
	}
	idToken, err := provider.Verifier(&oidc.Config{ClientID: clientID}).Verify(r.Context(), rawID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid_id_token", "Keycloak ID 토큰 검증에 실패했습니다")
		return
	}
	var claims struct {
		Subject           string   `json:"sub"`
		Nonce             string   `json:"nonce"`
		PreferredUsername string   `json:"preferred_username"`
		Name              string   `json:"name"`
		Email             string   `json:"email"`
		Groups            []string `json:"groups"`
	}
	if err := idToken.Claims(&claims); err != nil || subtle.ConstantTimeCompare([]byte(claims.Nonce), []byte(nonce)) != 1 {
		writeError(w, http.StatusUnauthorized, "invalid_oidc_claims", "Keycloak 토큰 요청값이 일치하지 않습니다")
		return
	}
	username := claims.PreferredUsername
	if username == "" {
		username = claims.Email
	}
	if username == "" {
		username = "oidc-" + claims.Subject
	}
	name := claims.Name
	if name == "" {
		name = username
	}
	role := s.roleForGroups(r.Context(), claims.Groups)
	id := newID()
	err = s.db.QueryRow(r.Context(), `INSERT INTO users(id,username,display_name,email,role,source,last_login_at) VALUES($1,$2,$3,$4,$5,'oidc',now())
		ON CONFLICT(username) DO UPDATE SET display_name=EXCLUDED.display_name,email=EXCLUDED.email,source='oidc',last_login_at=now(),
		role=CASE WHEN users.role='system_admin' THEN users.role ELSE EXCLUDED.role END RETURNING id`, id, username, name, claims.Email, role).Scan(&id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "user_provision_failed", "SSO 사용자를 등록하지 못했습니다")
		return
	}
	u, err := scanUser(s.db.QueryRow(r.Context(), `SELECT id,username,display_name,COALESCE(email,''),employee_id,role,source,last_login_at,active FROM users WHERE id=$1`, id))
	if err != nil || s.issueSession(w, r, u) != nil {
		writeError(w, http.StatusInternalServerError, "session_error", "로그인 세션을 만들지 못했습니다")
		return
	}
	s.audit(r.Context(), u.ID, "auth.login", "user", u.ID, r.RemoteAddr, map[string]string{"source": "oidc"})
	if !safeReturnTo(returnTo) {
		returnTo = "/"
	}
	http.Redirect(w, r, returnTo, http.StatusFound)
}

func requestBaseURL(r *http.Request) string {
	scheme := "http"
	if requestIsHTTPS(r) {
		scheme = "https"
	}
	host := r.Host
	if forwarded := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Host"), ",")[0]); forwarded != "" {
		host = forwarded
	}
	return scheme + "://" + host
}

func (s *Server) oidcScopes(ctx context.Context) []string {
	v, _ := s.getSetting(ctx, "oidc.scopes")
	parts := strings.Fields(v)
	if len(parts) == 0 {
		return []string{oidc.ScopeOpenID, "profile", "email"}
	}
	return parts
}

func (s *Server) roleForGroups(ctx context.Context, groups []string) string {
	admin, _ := s.getSetting(ctx, "oidc.admin_group")
	manager, _ := s.getSetting(ctx, "oidc.seat_manager_group")
	for _, g := range groups {
		if admin != "" && g == admin {
			return "system_admin"
		}
	}
	for _, g := range groups {
		if manager != "" && g == manager {
			return "seat_manager"
		}
	}
	return "employee"
}

func (s *Server) testOIDC(w http.ResponseWriter, r *http.Request) {
	issuer, _ := s.getSetting(r.Context(), "oidc.issuer_url")
	if issuer == "" {
		writeError(w, http.StatusBadRequest, "issuer_required", "Issuer URL을 입력하세요")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		writeError(w, http.StatusBadGateway, "oidc_discovery_failed", fmt.Sprintf("Discovery 실패: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "issuer": issuer, "authorizationEndpoint": provider.Endpoint().AuthURL, "tokenEndpoint": provider.Endpoint().TokenURL, "redirectUri": requestBaseURL(r) + "/api/v1/auth/oidc/callback"})
}

func (s *Server) listUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query(r.Context(), `SELECT id,username,display_name,COALESCE(email,''),employee_id,role,source,last_login_at,active FROM users ORDER BY display_name`)
	if err != nil {
		notFoundOrServer(w, err)
		return
	}
	defer rows.Close()
	users := []User{}
	for rows.Next() {
		u, err := scanUser(rows)
		if err == nil {
			users = append(users, u)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": users})
}

// userPatch 는 PATCH /users/{id} 가 받는 항목이다. 보내지 않은 항목은 그대로 둔다.
// email 은 빈 문자열이면 주소를 지운다(로컬 관리자 계정은 처음부터 주소가 없다).
type userPatch struct {
	Role   string  `json:"role"`
	Active *bool   `json:"active"`
	Email  *string `json:"email"`
}

// validateUserPatch 는 요청값을 다듬고 검사해 오류 코드와 안내문을 돌려준다.
// 자기 계정은 막을 수 없고 권한도 낮출 수 없다 — 마지막 관리자가 스스로를
// 잠그거나 직원으로 내리면 이 화면·이 API 를 더는 부를 수 없어 화면으로는
// 되돌릴 길이 없다. system_admin 을 다시 고르는 것(현상 유지)은 통과한다.
func validateUserPatch(in *userPatch, targetID, actorID string) (code, message string) {
	allowed := map[string]bool{"employee": true, "department_manager": true, "seat_manager": true, "system_admin": true}
	if in.Role != "" && !allowed[in.Role] {
		return "invalid_role", "권한 값이 올바르지 않습니다"
	}
	if in.Role != "" && in.Role != "system_admin" && targetID == actorID {
		return "self_demotion", "자기 계정의 권한은 낮출 수 없습니다"
	}
	if in.Active != nil && !*in.Active && targetID == actorID {
		return "self_deactivation", "자기 계정은 비활성화할 수 없습니다"
	}
	if in.Email != nil {
		email := strings.TrimSpace(*in.Email)
		if email != "" && !validEmailAddress(email) {
			return "invalid_email", "메일 주소 형식이 올바르지 않습니다"
		}
		in.Email = &email
	}
	return "", ""
}

// validEmailAddress 는 "이름 <주소>" 꼴이 아닌 주소 하나만 받는다. 알림 메일의
// 받는 사람 칸에 그대로 들어가므로 표시 이름이나 여러 주소는 거절한다.
func validEmailAddress(s string) bool {
	if len(s) > 254 || strings.ContainsAny(s, " \t\r\n,;<>") {
		return false
	}
	addr, err := mail.ParseAddress(s)
	return err == nil && addr.Address == s
}

func (s *Server) updateUser(w http.ResponseWriter, r *http.Request) {
	var in userPatch
	if !decodeJSON(w, r, &in) {
		return
	}
	actor, _ := userFrom(r)
	targetID := chiURLParam(r, "userID")
	if code, message := validateUserPatch(&in, targetID, actor.ID); code != "" {
		writeError(w, 400, code, message)
		return
	}
	if in.Email != nil {
		// SSO 사용자의 주소는 로그인할 때마다 Keycloak 프로필로 덮어쓰므로 여기서
		// 고쳐도 다음 로그인에 사라진다. 조용히 잃는 대신 거절한다.
		var source string
		if err := s.db.QueryRow(r.Context(), `SELECT source FROM users WHERE id=$1`, targetID).Scan(&source); err != nil {
			notFoundOrServer(w, err)
			return
		}
		if source != "local" {
			writeError(w, http.StatusConflict, "sso_managed_email", "SSO 사용자의 메일 주소는 Keycloak 프로필에서 가져옵니다")
			return
		}
	}
	_, err := s.db.Exec(r.Context(), `UPDATE users SET role=COALESCE(NULLIF($2,''),role),active=COALESCE($3,active),
		email=CASE WHEN $4 THEN NULLIF($5,'') ELSE email END,updated_at=now() WHERE id=$1`,
		targetID, in.Role, in.Active, in.Email != nil, in.Email)
	if err != nil {
		notFoundOrServer(w, err)
		return
	}
	s.audit(r.Context(), actor.ID, "user.update", "user", targetID, r.RemoteAddr, in)
	w.WriteHeader(http.StatusNoContent)
}

func chiURLParam(r *http.Request, key string) string {
	return chi.URLParam(r, key)
}
