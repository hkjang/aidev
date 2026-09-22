package server

import (
	"context"
	"crypto/subtle"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

const oidcCookieName = "madi_oidc_state"

func integrationHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{Timeout: timeout, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return errors.New("too many redirects")
		}
		if len(via) > 0 && req.URL.Host != via[0].URL.Host {
			return errors.New("cross-host redirect refused")
		}
		if len(via) > 0 && via[0].URL.Scheme == "https" && req.URL.Scheme != "https" {
			return errors.New("HTTPS downgrade refused")
		}
		return nil
	}}
}

func oidcConfiguration(ctx context.Context, settings map[string]any) (*oidc.Provider, *oauth2.Config, error) {
	if !settingBool(settings, "oidc_enabled") {
		return nil, nil, errors.New("SSO가 활성화되지 않았습니다.")
	}
	issuer := settingString(settings, "oidc_issuer")
	clientID := settingString(settings, "oidc_client_id")
	base := strings.TrimRight(settingString(settings, "site_url"), "/")
	for _, value := range []string{issuer, base} {
		u, err := url.Parse(value)
		if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
			return nil, nil, errors.New("관리자 설정의 서비스 URL과 OIDC issuer URL을 확인하세요.")
		}
	}
	if clientID == "" {
		return nil, nil, errors.New("OIDC client ID가 필요합니다.")
	}
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, nil, errors.New("SSO 제공자의 설정을 가져오지 못했습니다. issuer와 네트워크 연결을 확인하세요.")
	}
	config := &oauth2.Config{ClientID: clientID, ClientSecret: settingString(settings, "oidc_client_secret"), Endpoint: provider.Endpoint(), RedirectURL: base + "/api/v1/auth/oidc/callback", Scopes: []string{oidc.ScopeOpenID, "profile", "email"}}
	return provider, config, nil
}

func (s *Server) oidcStart(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	ctx = oidc.ClientContext(ctx, integrationHTTPClient(12*time.Second))
	settings, err := s.settings(ctx)
	if err != nil {
		apiError(w, 500, "SSO 설정을 불러올 수 없습니다.")
		return
	}
	_, config, err := oidcConfiguration(ctx, settings)
	if err != nil {
		apiError(w, 503, err.Error())
		return
	}
	// prompt=none asks the provider to answer from an existing session only and
	// never renders a login screen. It is honoured only when the administrator
	// enabled auto login, so a query parameter alone cannot change the flow.
	silent := r.URL.Query().Get("prompt") == "none" && settingBool(settings, "oidc_auto_login")
	returnTo := oidcReturnTo(r.URL.Query().Get("return_to"))
	state, browser, nonce := integrationSecret(), integrationSecret(), integrationSecret()
	verifier := oauth2.GenerateVerifier()
	sealedVerifier, err := s.encrypt(verifier)
	if err != nil {
		apiError(w, 500, "SSO 요청을 보호할 수 없습니다.")
		return
	}
	_, err = s.DB.Exec(ctx, `DELETE FROM oidc_attempts WHERE expires_at < now()`)
	if err != nil {
		apiError(w, 500, "SSO 요청을 준비할 수 없습니다.")
		return
	}
	_, err = s.DB.Exec(ctx, `INSERT INTO oidc_attempts(state_hash,browser_hash,nonce,verifier,issuer,client_id,redirect_uri,silent,return_to,expires_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,now()+interval '10 minutes')`, integrationHash(state), integrationHash(browser), nonce, sealedVerifier, settingString(settings, "oidc_issuer"), config.ClientID, config.RedirectURL, silent, returnTo)
	if err != nil {
		apiError(w, 500, "SSO 요청을 저장할 수 없습니다.")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: oidcCookieName, Value: browser, Path: "/api/v1/auth/oidc", HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: strings.HasPrefix(config.RedirectURL, "https://"), MaxAge: 600})
	options := []oauth2.AuthCodeOption{oidc.Nonce(nonce), oauth2.S256ChallengeOption(verifier)}
	if silent {
		options = append(options, oauth2.SetAuthURLParam("prompt", "none"))
	}
	http.Redirect(w, r, config.AuthCodeURL(state, options...), http.StatusFound)
}

// oidcSilentRefusalPath is where a refused prompt=none attempt lands. The query
// marker tells the browser not to try again even if its storage was cleared.
const oidcSilentRefusalPath = "/login?sso=none"

// oidcReturnTo accepts only a same-origin path so the login flow cannot be used
// as an open redirect: it must start with "/" and must not start with "//".
func oidcReturnTo(value string) string {
	if value == "" || !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") || strings.ContainsAny(value, "\\\r\n") || len(value) > 2048 {
		return "/app"
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.IsAbs() || parsed.Host != "" {
		return "/app"
	}
	return value
}

func (s *Server) oidcCallback(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
	defer cancel()
	ctx = oidc.ClientContext(ctx, integrationHTTPClient(20*time.Second))
	settings, err := s.settings(ctx)
	if err != nil {
		apiError(w, 500, "SSO 설정을 불러올 수 없습니다.")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: oidcCookieName, Value: "", Path: "/api/v1/auth/oidc", HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: strings.HasPrefix(settingString(settings, "site_url"), "https://"), MaxAge: -1})
	cookie, err := r.Cookie(oidcCookieName)
	state := r.URL.Query().Get("state")
	if err != nil || len(state) < 32 || cookie.Value == "" {
		apiError(w, 400, "SSO 로그인 요청이 유효하지 않습니다. 다시 로그인하세요.")
		return
	}
	var nonce, sealedVerifier, issuer, clientID, redirectURI, returnTo string
	var silent bool
	err = s.DB.QueryRow(ctx, `DELETE FROM oidc_attempts WHERE state_hash=$1 AND browser_hash=$2 AND expires_at>now() RETURNING nonce,verifier,issuer,client_id,redirect_uri,silent,return_to`, integrationHash(state), integrationHash(cookie.Value)).Scan(&nonce, &sealedVerifier, &issuer, &clientID, &redirectURI, &silent, &returnTo)
	if err != nil {
		apiError(w, 400, "SSO 요청이 만료되었거나 이미 사용되었습니다. 다시 로그인하세요.")
		return
	}
	if r.URL.Query().Get("error") != "" || r.URL.Query().Get("code") == "" {
		if silent {
			// Without a provider session prompt=none answers login_required.
			// That is an ordinary outcome, not a failure: show the login screen
			// with the marker that stops the browser from retrying in a loop.
			http.Redirect(w, r, oidcSilentRefusalPath, http.StatusFound)
			return
		}
		apiError(w, 401, "SSO 인증이 취소되었거나 실패했습니다.")
		return
	}
	provider, config, err := oidcConfiguration(ctx, settings)
	if err != nil {
		apiError(w, 503, err.Error())
		return
	}
	if issuer != settingString(settings, "oidc_issuer") || clientID != config.ClientID || redirectURI != config.RedirectURL {
		apiError(w, 409, "SSO 설정이 변경되었습니다. 다시 로그인하세요.")
		return
	}
	verifier, err := s.decrypt(sealedVerifier)
	if err != nil {
		apiError(w, 500, "SSO 요청을 검증할 수 없습니다.")
		return
	}
	token, err := config.Exchange(ctx, r.URL.Query().Get("code"), oauth2.VerifierOption(verifier))
	if err != nil {
		apiError(w, 401, "SSO 인증 코드를 검증하지 못했습니다.")
		return
	}
	rawToken, ok := token.Extra("id_token").(string)
	if !ok {
		apiError(w, 401, "SSO 응답에 ID 토큰이 없습니다.")
		return
	}
	idToken, err := provider.VerifierContext(ctx, &oidc.Config{ClientID: config.ClientID}).Verify(ctx, rawToken)
	if err != nil || subtle.ConstantTimeCompare([]byte(idTokenNonce(idToken)), []byte(nonce)) != 1 {
		apiError(w, 401, "SSO ID 토큰 검증에 실패했습니다.")
		return
	}
	if idToken.AccessTokenHash != "" && idToken.VerifyAccessToken(token.AccessToken) != nil {
		apiError(w, 401, "SSO 액세스 토큰 검증에 실패했습니다.")
		return
	}
	var claims struct {
		Email             string `json:"email"`
		EmailVerified     bool   `json:"email_verified"`
		Name              string `json:"name"`
		PreferredUsername string `json:"preferred_username"`
	}
	if err := idToken.Claims(&claims); err != nil || !strings.Contains(claims.Email, "@") || idToken.Subject == "" {
		apiError(w, 403, "SSO 계정의 이메일 주소와 고유 ID를 확인하세요.")
		return
	}
	if boolean(settings, "oidc_require_verified_email") && !claims.EmailVerified {
		apiError(w, 403, "관리자 SSO 설정에서 이메일 검증을 필수로 지정했습니다. Keycloak에서 이메일을 검증하거나 관리자에게 설정 변경을 요청하세요.")
		return
	}
	email := strings.ToLower(strings.TrimSpace(claims.Email))
	name := strings.TrimSpace(claims.Name)
	if name == "" {
		name = claims.PreferredUsername
	}
	if name == "" {
		name = email
	}
	var rawClaims map[string]any
	if err = idToken.Claims(&rawClaims); err != nil {
		apiError(w, 401, "SSO 사용자 속성을 읽지 못했습니다")
		return
	}
	groups, err := identityClaimGroups(rawClaims, str(settings, "oidc_groups_claim"))
	if err != nil {
		apiError(w, 403, err.Error())
		return
	}
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		apiError(w, 500, "SSO 계정을 확인할 수 없습니다.")
		return
	}
	defer tx.Rollback(ctx)
	userID, err := s.identityUser(ctx, tx, settings, externalIdentity{Provider: "oidc", Issuer: issuer, Subject: idToken.Subject, Email: email, Name: name, Groups: groups, VerifiedEmail: claims.EmailVerified, AutoRegister: boolean(settings, "oidc_auto_register"), ConfigurationHash: identityConfigurationHash(settings, "oidc")})
	if err != nil {
		identityLoginError(w, err)
		return
	}
	if err = tx.Commit(ctx); err != nil {
		apiError(w, 500, "SSO 계정을 저장하지 못했습니다.")
		return
	}
	if err = s.createSession(w, r, userID); err != nil {
		apiError(w, 500, "로그인 세션을 생성할 수 없습니다.")
		return
	}
	s.audit(r, "LOGIN", userID, map[string]any{"provider": "oidc", "issuer": issuer, "email_verified": claims.EmailVerified, "silent": silent})
	// A deep link that triggered a silent sign-in lands back where it started.
	http.Redirect(w, r, oidcReturnTo(returnTo), http.StatusFound)
}

func idTokenNonce(token *oidc.IDToken) string {
	if token == nil {
		return ""
	}
	return token.Nonce
}
