package httpapi

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/hkjang/invenqor/server/internal/auth"
	"github.com/hkjang/invenqor/server/internal/diagnostics"
)

// Every Keycloak failure code an administrator can meet, with the action that
// resolves it. The console shows this next to the code.
var keycloakGuidance = map[string]string{
	"KEYCLOAK_DISABLED": "설정 > Keycloak 에서 Keycloak 로그인을 켜십시오.",
	"KEYCLOAK_SECRET_REQUIRED": "설정 > Keycloak 에서 Client Secret을 입력하고 " +
		"다시 저장하십시오.",
	"KEYCLOAK_FLOW_EXPIRED": "로그인이 10분을 넘겼거나 링크가 다시 사용되었습니다. " +
		"콘솔에서 로그인을 다시 시작하십시오.",
	"KEYCLOAK_NONCE_MISMATCH": "ID 토큰이 이 로그인 시도와 일치하지 않습니다. " +
		"하나의 Keycloak client를 두 서버가 공유하고 있지 않은지 확인하십시오.",
	"KEYCLOAK_EMAIL_DOMAIN_REJECTED": "해당 도메인을 허용 이메일 도메인 목록에 " +
		"추가하거나 제한을 해제하십시오.",
	"KEYCLOAK_PROVISIONING_DISABLED": "자동 사용자 생성을 켜거나, 사용자가 " +
		"로그인하기 전에 로컬 계정을 먼저 만드십시오.",
	"KEYCLOAK_USERNAME_UNUSABLE": "영문자·숫자·점·밑줄·하이픈으로 3~64자가 되는 " +
		"username claim을 매핑하십시오.",
	"KEYCLOAK_USERNAME_CONFLICT": "같은 이름의 로컬 계정이 이미 있습니다. 그 로컬 " +
		"계정의 이름을 바꾸거나 삭제하거나, 다른 username claim을 매핑하십시오.",
	"KEYCLOAK_USER_INACTIVE": "사용자 화면에서 해당 계정을 다시 활성화하십시오.",
	"KEYCLOAK_ROLE_MISSING": "역할 매핑이 더 이상 존재하지 않는 역할을 가리킵니다. " +
		"설정 > Keycloak 에서 매핑을 수정하십시오.",
	"KEYCLOAK_UNREACHABLE": "Keycloak URL, realm, DNS, TLS 신뢰를 확인하십시오. " +
		"설정 > Keycloak 의 연결 테스트를 사용하십시오.",
	"KEYCLOAK_PROVIDER_REJECTED": "Keycloak이 요청을 거부했습니다. client의 유효 " +
		"redirect URI와 인증 흐름 정책을 확인하십시오.",
	"KEYCLOAK_LOGIN_FAILED": "Server 진단 로그 를 열고 표시된 request ID로 " +
		"검색하십시오.",
}

func keycloakRemediation(code string) string {
	if guidance, found := keycloakGuidance[code]; found {
		return guidance
	}
	return keycloakGuidance["KEYCLOAK_LOGIN_FAILED"]
}

// silentLoginRefusedPath is where a refused prompt=none attempt lands. The
// console's login screen lives at the root of the SPA (there is no /login
// route), and it reads sso=none as "do not try silently again".
const silentLoginRefusedPath = "/?sso=none"

type keycloakSettingsUpdate struct {
	Settings     auth.OIDCSettings `json:"settings"`
	ClientSecret *string           `json:"client_secret,omitempty"`
	Reason       string            `json:"reason"`
}

func (s *Server) authMethods(response http.ResponseWriter, request *http.Request) {
	settings, err := s.oidcService.Settings(request.Context())
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	state, err := s.oidcService.ClientSecretState(request.Context())
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	secretConfigured := state == auth.ClientSecretReady
	unreadable := state == auth.ClientSecretUnreadable
	if unreadable {
		// This used to be a 500, which the login page's fetch swallowed: the
		// Keycloak button simply vanished after a restart and nothing anywhere
		// said why. It is recorded so it also appears in the Server log screen.
		s.recordDiagnostic(request, diagnostics.Event{
			Level:     "error",
			Component: "keycloak",
			EventCode: "KEYCLOAK_SECRET_UNREADABLE",
			Message: "a Keycloak client secret is stored but this instance's " +
				"master key cannot decrypt it, so SSO is unavailable",
			Details: map[string]any{
				// INVENQOR_MASTER_KEY_FILE names a file, not a key. Advice that
				// does not work is worse than none: an operator following it sees
				// no change and has no reason to doubt the instruction.
				"remediation": "Every replica must use the same master key. Either " +
					"mount the state directory holding master.key as a shared " +
					"read-write volume, or give every instance the same key file and " +
					"set INVENQOR_MASTER_KEY_FILE to its path - a Secret mounted into " +
					"each Pod does this. Then re-save the Keycloak client secret.",
			},
		})
	}
	// Offering the button without a client secret sends the user into a failed
	// redirect, so report readiness rather than the stored flag alone.
	ready := settings.Enabled && secretConfigured
	writeJSON(response, http.StatusOK, map[string]any{
		"local":                      true,
		"keycloak":                   ready,
		"keycloak_enabled":           settings.Enabled,
		"keycloak_client_secret":     secretConfigured,
		"keycloak_incomplete":        settings.Enabled && !secretConfigured,
		"keycloak_secret_unreadable": unreadable,
		"keycloak_provider_issuer":   settings.EffectiveIssuer(),
		// Published so the console knows whether to try a silent sign-in before
		// it draws the login screen. Tied to readiness: a silent attempt against
		// a provider that cannot complete the login only produces a redirect.
		"keycloak_auto_login": ready && settings.AutoLogin,
	})
}

func (s *Server) keycloakStart(response http.ResponseWriter, request *http.Request) {
	start, err := s.oidcService.Start(
		request.Context(),
		request.URL.Query().Get("return_to"),
		request.URL.Query().Get("prompt") == "none",
		clientIP(request),
		request.UserAgent(),
	)
	if err != nil {
		code, message, _ := keycloakFailure(err)
		s.recordKeycloakFailure(request, "keycloak_start", code, message, err)
		// The browser navigates here directly, so hand the console a code it can
		// display instead of a raw JSON body on a blank page.
		s.redirectKeycloakFailure(response, request, code)
		return
	}
	http.Redirect(response, request, start.AuthorizationURL, http.StatusFound)
}

func (s *Server) keycloakCallback(response http.ResponseWriter, request *http.Request) {
	if providerError := strings.TrimSpace(
		request.URL.Query().Get("error"),
	); providerError != "" {
		// A prompt=none attempt that finds no provider session comes back here
		// as error=login_required. That is the ordinary answer for a visitor who
		// is not signed in, not a failure: show the login screen, and leave the
		// marker in the address so the console does not try again even when its
		// sessionStorage was wiped in between. Retrying here is the loop.
		refused, err := s.oidcService.RefusedSilently(
			request.Context(),
			request.URL.Query().Get("state"),
			providerError,
		)
		if err != nil {
			s.internalError(response, request, err)
			return
		}
		if refused {
			http.Redirect(response, request, silentLoginRefusedPath, http.StatusFound)
			return
		}
		// Keycloak reports consent denial and policy failures this way; without
		// this branch the user only saw the generic flow-expired message.
		s.recordKeycloakFailure(
			request,
			"keycloak_callback",
			"KEYCLOAK_PROVIDER_REJECTED",
			"Keycloak rejected the login request.",
			errors.New(providerError+": "+request.URL.Query().Get("error_description")),
		)
		s.redirectKeycloakFailure(response, request, "KEYCLOAK_PROVIDER_REJECTED")
		return
	}
	session, returnTo, err := s.oidcService.Callback(
		request.Context(),
		request.URL.Query().Get("state"),
		request.URL.Query().Get("code"),
		clientIP(request),
		request.UserAgent(),
		middleware.GetReqID(request.Context()),
	)
	if err != nil {
		code, message, _ := keycloakFailure(err)
		s.recordKeycloakFailure(request, "keycloak_callback", code, message, err)
		s.redirectKeycloakFailure(response, request, code)
		return
	}
	setSessionCookie(response, request, session)
	http.Redirect(response, request, returnTo, http.StatusFound)
}

// keycloakFailure maps a login failure to a stable code an administrator can
// look up, so an SSO problem is diagnosable without server shell access.
func keycloakFailure(err error) (string, string, int) {
	switch {
	case errors.Is(err, auth.ErrOIDCDisabled):
		return "KEYCLOAK_DISABLED",
			"Keycloak login is not enabled.",
			http.StatusNotFound
	case errors.Is(err, auth.ErrOIDCSecret):
		return "KEYCLOAK_SECRET_REQUIRED",
			"The Keycloak client secret is not configured.",
			http.StatusServiceUnavailable
	case errors.Is(err, auth.ErrOIDCFlow):
		return "KEYCLOAK_FLOW_EXPIRED",
			"The login attempt expired or was already used.",
			http.StatusUnauthorized
	case errors.Is(err, auth.ErrOIDCNonce):
		return "KEYCLOAK_NONCE_MISMATCH",
			"The Keycloak ID token nonce did not match.",
			http.StatusUnauthorized
	case errors.Is(err, auth.ErrOIDCDomain):
		return "KEYCLOAK_EMAIL_DOMAIN_REJECTED",
			"The account email domain is not allowed.",
			http.StatusForbidden
	case errors.Is(err, auth.ErrOIDCProvisioning):
		return "KEYCLOAK_PROVISIONING_DISABLED",
			"Automatic user creation is disabled for Keycloak logins.",
			http.StatusForbidden
	case errors.Is(err, auth.ErrOIDCUsername):
		return "KEYCLOAK_USERNAME_UNUSABLE",
			"The Keycloak username claim is missing or unusable.",
			http.StatusForbidden
	case errors.Is(err, auth.ErrOIDCUsernameTaken):
		return "KEYCLOAK_USERNAME_CONFLICT",
			"A different local account already uses this Keycloak username.",
			http.StatusConflict
	case errors.Is(err, auth.ErrOIDCUserInactive):
		return "KEYCLOAK_USER_INACTIVE",
			"The linked account is deactivated.",
			http.StatusForbidden
	case errors.Is(err, auth.ErrOIDCRole):
		return "KEYCLOAK_ROLE_MISSING",
			"A Keycloak mapping references a role that no longer exists.",
			http.StatusConflict
	case errors.Is(err, auth.ErrOIDCUnreachable):
		return "KEYCLOAK_UNREACHABLE",
			"The Keycloak issuer could not be reached.",
			http.StatusBadGateway
	default:
		return "KEYCLOAK_LOGIN_FAILED",
			"Keycloak login could not be completed.",
			http.StatusUnauthorized
	}
}

func (s *Server) redirectKeycloakFailure(
	response http.ResponseWriter,
	request *http.Request,
	code string,
) {
	http.Redirect(
		response,
		request,
		"/?auth_error="+url.QueryEscape(code)+
			"&request_id="+url.QueryEscape(middleware.GetReqID(request.Context())),
		http.StatusFound,
	)
}

func (s *Server) recordKeycloakFailure(
	request *http.Request,
	component string,
	code string,
	message string,
	err error,
) {
	s.logger.Warn(
		component+"_failed",
		"request_id", middleware.GetReqID(request.Context()),
		"code", code,
		"error", err,
	)
	s.recordDiagnostic(request, diagnostics.Event{
		Level:     "warning",
		Component: "keycloak",
		EventCode: code,
		Message:   message,
		SourceIP:  clientIP(request),
		Details: map[string]any{
			"stage":       component,
			"error":       err.Error(),
			"remediation": keycloakRemediation(code),
		},
	})
}

func (s *Server) getKeycloakSettings(response http.ResponseWriter, request *http.Request) {
	settings, err := s.oidcService.Settings(request.Context())
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	secretConfigured, err := s.oidcService.ClientSecretConfigured(request.Context())
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	writeJSON(response, http.StatusOK, map[string]any{
		"settings":                 settings,
		"client_secret_configured": secretConfigured,
	})
}

func (s *Server) updateKeycloakSettings(response http.ResponseWriter, request *http.Request) {
	var input keycloakSettingsUpdate
	if err := decodeJSON(request, &input); err != nil {
		writeAPIError(response, request, http.StatusBadRequest, "INVALID_REQUEST", "The request body is invalid.")
		return
	}
	if err := s.oidcService.SaveSettings(
		request.Context(),
		input.Settings,
		input.ClientSecret,
		principalFromContext(request.Context()).User,
		input.Reason,
	); err != nil {
		s.logger.Warn(
			"keycloak_settings_rejected",
			"request_id", middleware.GetReqID(request.Context()),
			"error", err,
		)
		switch {
		case errors.Is(err, auth.ErrOIDCSecret):
			writeAPIError(response, request, http.StatusBadRequest, "KEYCLOAK_SECRET_REQUIRED", "A Keycloak client secret is required before enabling login.")
		case errors.Is(err, auth.ErrOIDCRole):
			writeAPIError(response, request, http.StatusBadRequest, "INVALID_KEYCLOAK_ROLE", "A Keycloak mapping references an unknown InvenQor role.")
		default:
			// The specific reason is what the administrator needs; the previous
			// fixed sentence made every field mistake look identical.
			writeAPIError(response, request, http.StatusBadRequest, "INVALID_KEYCLOAK_SETTINGS", err.Error())
		}
		return
	}
	writeJSON(response, http.StatusOK, map[string]any{"saved": true})
}

func (s *Server) testKeycloakSettings(response http.ResponseWriter, request *http.Request) {
	var input struct {
		Settings auth.OIDCSettings `json:"settings"`
	}
	if err := decodeJSON(request, &input); err != nil {
		writeAPIError(response, request, http.StatusBadRequest, "INVALID_REQUEST", "The request body is invalid.")
		return
	}
	if err := s.oidcService.TestConnection(request.Context(), input.Settings); err != nil {
		// A configuration mistake and an unreachable issuer need different
		// actions, and the previous single 502 hid which one occurred.
		if errors.Is(err, auth.ErrOIDCUnreachable) {
			writeAPIError(
				response, request, http.StatusBadGateway,
				"KEYCLOAK_CONNECTION_FAILED",
				"The Keycloak issuer could not be reached. "+
					keycloakRemediation("KEYCLOAK_UNREACHABLE"),
			)
			return
		}
		if errors.Is(err, auth.ErrOIDCRole) {
			writeAPIError(
				response, request, http.StatusBadRequest,
				"INVALID_KEYCLOAK_ROLE",
				"A Keycloak mapping references an unknown InvenQor role.",
			)
			return
		}
		writeAPIError(
			response, request, http.StatusBadRequest,
			"INVALID_KEYCLOAK_SETTINGS",
			err.Error(),
		)
		return
	}
	writeJSON(response, http.StatusOK, map[string]any{
		"connected": true,
		"issuer":    input.Settings.EffectiveIssuer(),
	})
}

func (s *Server) autoConfigureKeycloak(
	response http.ResponseWriter,
	request *http.Request,
) {
	var input struct {
		KeycloakURL    string  `json:"keycloak_url"`
		Realm          string  `json:"realm"`
		ClientID       string  `json:"client_id"`
		ClientSecret   *string `json:"client_secret,omitempty"`
		ApplicationURL string  `json:"application_url"`
		PrivateCAPEM   string  `json:"private_ca_pem"`
		Reason         string  `json:"reason"`
	}
	if err := decodeJSON(request, &input); err != nil {
		writeAPIError(
			response,
			request,
			http.StatusBadRequest,
			"INVALID_REQUEST",
			"The request body is invalid.",
		)
		return
	}
	settings, err := s.oidcService.AutomaticSettings(
		request.Context(),
		auth.OIDCAutoConfig{
			KeycloakURL:    input.KeycloakURL,
			Realm:          input.Realm,
			ClientID:       input.ClientID,
			ApplicationURL: input.ApplicationURL,
			PrivateCAPEM:   input.PrivateCAPEM,
		},
	)
	if err != nil {
		s.logger.Warn(
			"keycloak_auto_configuration_failed",
			"request_id", middleware.GetReqID(request.Context()),
			"error", err,
		)
		writeAPIError(
			response,
			request,
			http.StatusBadGateway,
			"KEYCLOAK_DISCOVERY_FAILED",
			"Keycloak discovery failed. Verify the URL, realm, and TLS trust.",
		)
		return
	}
	if err := s.oidcService.SaveSettings(
		request.Context(),
		settings,
		input.ClientSecret,
		principalFromContext(request.Context()).User,
		input.Reason,
	); err != nil {
		if errors.Is(err, auth.ErrOIDCSecret) {
			writeAPIError(
				response,
				request,
				http.StatusBadRequest,
				"KEYCLOAK_SECRET_REQUIRED",
				"A Keycloak client secret is required before enabling login.",
			)
			return
		}
		if errors.Is(err, auth.ErrOIDCRole) {
			writeAPIError(
				response,
				request,
				http.StatusBadRequest,
				"INVALID_KEYCLOAK_ROLE",
				"A Keycloak mapping references an unknown InvenQor role.",
			)
			return
		}
		s.internalError(response, request, err)
		return
	}
	// Report the stored state rather than assuming it: the save above succeeds
	// with a previously stored secret and no secret in this request.
	secretConfigured, err := s.oidcService.ClientSecretConfigured(request.Context())
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	writeJSON(response, http.StatusOK, map[string]any{
		"configured":               true,
		"settings":                 settings,
		"client_secret_configured": secretConfigured,
		"discovery_issuer":         settings.EffectiveIssuer(),
		"redirect_uri":             settings.RedirectURI,
	})
}
