package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/hkjang/invenqor/server/internal/auth"
	"github.com/hkjang/invenqor/server/internal/diagnostics"
)

type principalContextKey struct{}

func (s *Server) bootstrapStatus(response http.ResponseWriter, request *http.Request) {
	status, err := s.bootstrapManager.Ensure(request.Context())
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	writeJSON(response, http.StatusOK, status)
}

func (s *Server) createInitialAdmin(response http.ResponseWriter, request *http.Request) {
	var input auth.InitialAdminInput
	if err := decodeJSON(request, &input); err != nil {
		writeAPIError(response, request, http.StatusBadRequest, "INVALID_REQUEST", "The request body is invalid.")
		return
	}
	user, err := s.bootstrapManager.CreateInitialAdmin(
		request.Context(),
		request.Header.Get("X-Invenqor-Bootstrap-Token"),
		input,
		clientIP(request),
		request.UserAgent(),
		middleware.GetReqID(request.Context()),
	)
	switch {
	case err == nil:
		writeJSON(response, http.StatusCreated, map[string]any{"user": user})
	case errors.Is(err, auth.ErrBootstrapComplete):
		writeAPIError(response, request, http.StatusConflict, "BOOTSTRAP_COMPLETE", "Initial setup is already complete.")
	case errors.Is(err, auth.ErrBootstrapToken):
		writeAPIError(response, request, http.StatusUnauthorized, "INVALID_BOOTSTRAP_TOKEN", "The bootstrap token is invalid.")
	default:
		s.logger.Warn(
			"initial_admin_rejected",
			"request_id", middleware.GetReqID(request.Context()),
			"error", err,
		)
		writeAPIError(response, request, http.StatusBadRequest, "INVALID_ADMIN", "The initial administrator details do not satisfy policy.")
	}
}

func (s *Server) localLogin(response http.ResponseWriter, request *http.Request) {
	var input auth.LoginInput
	if err := decodeJSON(request, &input); err != nil {
		writeAPIError(response, request, http.StatusBadRequest, "INVALID_REQUEST", "The request body is invalid.")
		return
	}
	session, err := s.authService.Authenticate(
		request.Context(),
		input,
		clientIP(request),
		request.UserAgent(),
		middleware.GetReqID(request.Context()),
	)
	switch {
	case err == nil:
		setSessionCookie(response, request, session)
		writeJSON(response, http.StatusOK, map[string]any{
			"user":                session.User,
			"csrf_token":          session.CSRFToken,
			"idle_expires_at":     session.IdleExpiresAt,
			"absolute_expires_at": session.AbsoluteExpiresAt,
		})
	case errors.Is(err, auth.ErrInvalidCredentials):
		writeAPIError(response, request, http.StatusUnauthorized, "INVALID_CREDENTIALS", "The username or password is invalid.")
	case errors.Is(err, auth.ErrAccountLocked):
		writeAPIError(response, request, http.StatusLocked, "ACCOUNT_LOCKED", "The account is temporarily locked.")
	case errors.Is(err, auth.ErrAccountInactive):
		writeAPIError(response, request, http.StatusForbidden, "ACCOUNT_INACTIVE", "The account is inactive.")
	case errors.Is(err, auth.ErrMFARequired):
		writeAPIError(response, request, http.StatusUnauthorized, "MFA_REQUIRED", "A TOTP code is required.")
	case errors.Is(err, auth.ErrMFAInvalid):
		writeAPIError(response, request, http.StatusUnauthorized, "MFA_INVALID", "The TOTP code is invalid.")
	default:
		s.internalError(response, request, err)
	}
}

func (s *Server) setupTOTP(response http.ResponseWriter, request *http.Request) {
	setup, err := s.totpService.Setup(
		request.Context(),
		principalFromContext(request.Context()).User,
	)
	switch {
	case err == nil:
		writeJSON(response, http.StatusOK, setup)
	case errors.Is(err, auth.ErrMFAAlreadyEnabled):
		writeAPIError(response, request, http.StatusConflict, "MFA_ALREADY_ENABLED", "TOTP is already enabled.")
	default:
		s.internalError(response, request, err)
	}
}

func (s *Server) enableTOTP(response http.ResponseWriter, request *http.Request) {
	var input struct {
		Code string `json:"code"`
	}
	if err := decodeJSON(request, &input); err != nil {
		writeAPIError(response, request, http.StatusBadRequest, "INVALID_REQUEST", "The request body is invalid.")
		return
	}
	if err := s.totpService.Enable(
		request.Context(),
		principalFromContext(request.Context()).User.ID,
		input.Code,
	); err != nil {
		writeAPIError(response, request, http.StatusBadRequest, "MFA_INVALID", "The TOTP code is invalid.")
		return
	}
	writeJSON(response, http.StatusOK, map[string]any{"totp_enabled": true})
}

func (s *Server) regenerateRecoveryCodes(
	response http.ResponseWriter,
	request *http.Request,
) {
	var input struct {
		Code string `json:"code"`
	}
	if err := decodeJSON(request, &input); err != nil {
		writeAPIError(response, request, http.StatusBadRequest, "INVALID_REQUEST", "The request body is invalid.")
		return
	}
	codes, err := s.totpService.RegenerateRecoveryCodes(
		request.Context(),
		principalFromContext(request.Context()).User.ID,
		input.Code,
	)
	switch {
	case err == nil:
		writeJSON(response, http.StatusOK, map[string]any{
			"recovery_codes": codes,
		})
	case errors.Is(err, auth.ErrMFASetupRequired):
		writeAPIError(
			response, request, http.StatusConflict, "MFA_NOT_ENABLED",
			"Recovery codes exist only for an enabled second factor.",
		)
	default:
		writeAPIError(response, request, http.StatusBadRequest, "MFA_INVALID", "The TOTP code is invalid.")
	}
}

func (s *Server) disableTOTP(response http.ResponseWriter, request *http.Request) {
	var input struct {
		Code string `json:"code"`
	}
	if err := decodeJSON(request, &input); err != nil {
		writeAPIError(response, request, http.StatusBadRequest, "INVALID_REQUEST", "The request body is invalid.")
		return
	}
	if err := s.totpService.Disable(
		request.Context(),
		principalFromContext(request.Context()).User.ID,
		input.Code,
	); err != nil {
		writeAPIError(response, request, http.StatusBadRequest, "MFA_INVALID", "The TOTP code is invalid.")
		return
	}
	writeJSON(response, http.StatusOK, map[string]any{"totp_enabled": false})
}

func setSessionCookie(
	response http.ResponseWriter,
	request *http.Request,
	session auth.Session,
) {
	secure := requestIsHTTPS(request)
	http.SetCookie(response, &http.Cookie{
		Name:     auth.SessionCookie,
		Value:    session.Token,
		Path:     "/",
		Secure:   secure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  session.AbsoluteExpiresAt,
		MaxAge:   int(time.Until(session.AbsoluteExpiresAt).Seconds()),
	})
	http.SetCookie(response, &http.Cookie{
		Name:     auth.CSRFCookie,
		Value:    session.CSRFToken,
		Path:     "/",
		Secure:   secure,
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
		Expires:  session.AbsoluteExpiresAt,
		MaxAge:   int(time.Until(session.AbsoluteExpiresAt).Seconds()),
	})
}

// clearSessionCookies must mirror the attributes used when the cookies were
// issued: a browser matches a deletion cookie on name, path and domain, and an
// attribute mismatch leaves a stale cookie behind.
func clearSessionCookies(response http.ResponseWriter, request *http.Request) {
	secure := requestIsHTTPS(request)
	for _, name := range []string{auth.SessionCookie, auth.CSRFCookie} {
		http.SetCookie(response, &http.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/",
			Secure:   secure,
			HttpOnly: name == auth.SessionCookie,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   -1,
			Expires:  time.Unix(1, 0),
		})
	}
}

// requestIsHTTPS decides whether the session cookies may carry the Secure
// attribute. A browser silently discards a Secure cookie delivered over plain
// HTTP, which made login appear to succeed and then fail on the closed-network
// HTTP deployments this product documents. Marking the cookie Secure on an
// HTTP response protects nothing, so follow the actual transport instead.
func requestIsHTTPS(request *http.Request) bool {
	if request == nil {
		return true
	}
	if request.TLS != nil {
		return true
	}
	if forwarded := request.Header.Get("X-Forwarded-Proto"); forwarded != "" {
		first, _, _ := strings.Cut(forwarded, ",")
		return strings.EqualFold(strings.TrimSpace(first), "https")
	}
	return strings.EqualFold(request.Header.Get("X-Forwarded-Ssl"), "on")
}

func (s *Server) me(response http.ResponseWriter, request *http.Request) {
	principal := principalFromContext(request.Context())
	writeJSON(response, http.StatusOK, map[string]any{
		"user":                principal.User,
		"idle_expires_at":     principal.IdleExpiresAt,
		"absolute_expires_at": principal.AbsoluteExpiresAt,
		"security":            s.accountSecurity(request, principal.User.ID),
	})
}

// accountSecurity describes the signed-in account's own protections. Without it
// the console could not tell whether MFA was on, so it offered both "set up" and
// "disable" to everyone, and showed a password form to accounts that have no
// local password to change.
func (s *Server) accountSecurity(
	request *http.Request,
	userID string,
) map[string]any {
	security := map[string]any{
		"password_configured":      true,
		"totp_enabled":             false,
		"recovery_codes_remaining": 0,
	}
	if configured, err := s.authService.PasswordConfigured(
		request.Context(), userID,
	); err == nil {
		security["password_configured"] = configured
	} else {
		s.logger.Warn(
			"account_security_password_state_unavailable",
			"request_id", middleware.GetReqID(request.Context()),
			"error", err,
		)
	}
	if s.totpService == nil {
		return security
	}
	status, err := s.totpService.Status(request.Context(), userID)
	if err != nil {
		s.logger.Warn(
			"account_security_totp_state_unavailable",
			"request_id", middleware.GetReqID(request.Context()),
			"error", err,
		)
		return security
	}
	security["totp_enabled"] = status.Enabled
	security["recovery_codes_remaining"] = status.RecoveryCodesRemaining
	if status.VerifiedAt != nil {
		security["totp_verified_at"] = status.VerifiedAt.Format(time.RFC3339Nano)
	}
	return security
}

func (s *Server) logout(response http.ResponseWriter, request *http.Request) {
	principal := principalFromContext(request.Context())
	logoutURL, logoutURLError := s.oidcService.LogoutURL(
		request.Context(),
		principal.User.ID,
	)
	if err := s.authService.RevokeSession(
		request.Context(),
		principal,
		clientIP(request),
		request.UserAgent(),
		middleware.GetReqID(request.Context()),
	); err != nil {
		s.internalError(response, request, err)
		return
	}
	if logoutURLError != nil {
		s.logger.Warn(
			"keycloak_logout_url_failed",
			"request_id", middleware.GetReqID(request.Context()),
			"error", logoutURLError,
		)
	}
	clearSessionCookies(response, request)
	writeJSON(response, http.StatusOK, map[string]any{
		"logged_out": true,
		"logout_url": logoutURL,
	})
}

func (s *Server) changePassword(response http.ResponseWriter, request *http.Request) {
	var input auth.ChangePasswordInput
	if err := decodeJSON(request, &input); err != nil {
		writeAPIError(response, request, http.StatusBadRequest, "INVALID_REQUEST", "The request body is invalid.")
		return
	}
	err := s.authService.ChangePassword(
		request.Context(),
		principalFromContext(request.Context()),
		input,
		clientIP(request),
		request.UserAgent(),
		middleware.GetReqID(request.Context()),
	)
	switch {
	case err == nil:
		writeJSON(response, http.StatusOK, map[string]any{"password_changed": true})
	case errors.Is(err, auth.ErrInvalidCredentials):
		writeAPIError(response, request, http.StatusUnauthorized, "INVALID_CREDENTIALS", "The current password is invalid.")
	case errors.Is(err, auth.ErrPasswordUnchanged):
		writeAPIError(response, request, http.StatusBadRequest, "PASSWORD_UNCHANGED", "The new password must differ from the current password.")
	case errors.Is(err, auth.ErrPasswordUnavailable):
		// A federated account has no local password. This used to fail the
		// database scan and surface as an internal error.
		writeAPIError(
			response, request, http.StatusConflict, "PASSWORD_NOT_LOCAL",
			"This account signs in through the identity provider, so its password is managed there.",
		)
	default:
		s.logger.Warn(
			"password_change_rejected",
			"request_id", middleware.GetReqID(request.Context()),
			"error", err,
		)
		writeAPIError(response, request, http.StatusBadRequest, "PASSWORD_POLICY", "The new password does not satisfy policy.")
	}
}

func (s *Server) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		cookie, err := request.Cookie(auth.SessionCookie)
		if err != nil {
			writeAPIError(response, request, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication is required.")
			return
		}
		principal, err := s.authService.PrincipalByToken(request.Context(), cookie.Value)
		if err != nil {
			writeAPIError(response, request, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication is required.")
			return
		}
		context := context.WithValue(request.Context(), principalContextKey{}, principal)
		next.ServeHTTP(response, request.WithContext(context))
	})
}

func (s *Server) requireCSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		principal := principalFromContext(request.Context())
		if err := s.authService.VerifyCSRF(
			principal,
			request.Header.Get("X-CSRF-Token"),
		); err != nil {
			writeAPIError(response, request, http.StatusForbidden, "CSRF_VALIDATION_FAILED", "The CSRF token is invalid.")
			return
		}
		next.ServeHTTP(response, request)
	})
}

func (s *Server) requirePermission(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			if !principalFromContext(request.Context()).HasPermission(permission) {
				writeAPIError(response, request, http.StatusForbidden, "FORBIDDEN", "The current user lacks the required permission.")
				return
			}
			next.ServeHTTP(response, request)
		})
	}
}

func principalFromContext(ctx context.Context) auth.Principal {
	principal, _ := ctx.Value(principalContextKey{}).(auth.Principal)
	return principal
}

func decodeJSON(request *http.Request, destination any) error {
	request.Body = http.MaxBytesReader(nil, request.Body, 1024*1024)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain one JSON value")
	}
	return nil
}

func clientIP(request *http.Request) string {
	host, _, err := net.SplitHostPort(request.RemoteAddr)
	if err == nil {
		return host
	}
	return strings.TrimSpace(request.RemoteAddr)
}

func (s *Server) internalError(
	response http.ResponseWriter,
	request *http.Request,
	err error,
) {
	s.logger.Error(
		"http_internal_error",
		"request_id", middleware.GetReqID(request.Context()),
		"error", err,
	)
	s.recordDiagnostic(request, diagnostics.Event{
		Level:     "error",
		Component: "http",
		EventCode: "INTERNAL_ERROR",
		Message:   "The server could not complete the request.",
		Details: map[string]any{
			"method": request.Method,
			"path":   request.URL.Path,
			"error":  err.Error(),
		},
	})
	writeAPIError(response, request, http.StatusInternalServerError, "INTERNAL_ERROR", "The server could not complete the request.")
}

func writeAPIError(
	response http.ResponseWriter,
	request *http.Request,
	status int,
	code string,
	message string,
) {
	writeJSON(response, status, map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
		"request_id": middleware.GetReqID(request.Context()),
	})
}
