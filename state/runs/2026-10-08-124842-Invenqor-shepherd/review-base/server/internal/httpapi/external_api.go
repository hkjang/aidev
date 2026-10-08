package httpapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/hkjang/invenqor/server/internal/apikeys"
	"github.com/hkjang/invenqor/server/internal/auth"
)

func (s *Server) authenticateAPIKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		credential, err := s.apiKeyService.Authenticate(r.Context(), bearerToken(r))
		if errors.Is(err, apikeys.ErrUnauthorized) {
			w.Header().Set("WWW-Authenticate", `Bearer realm="invenqor-api"`)
			writeAPIError(w, r, 401, "INVALID_API_KEY", "The API key is invalid, expired, or revoked.")
			return
		}
		if err != nil {
			s.internalError(w, r, err)
			return
		}
		if !s.apiRateLimit.Allow(credential.KeyID) {
			w.Header().Set("Retry-After", "60")
			writeAPIError(w, r, 429, "API_RATE_LIMITED", "The API key rate limit was exceeded.")
			return
		}
		principal := auth.Principal{
			SessionID: "api_key:" + credential.KeyID,
			User: auth.User{
				ID: credential.UserID, Username: "api-key:" + credential.Name,
				DisplayName: credential.Name, Permissions: credential.Scopes,
			},
		}
		ctx := context.WithValue(r.Context(), principalContextKey{}, principal)
		ctx = context.WithValue(ctx, apiKeyContextKey{}, credential)
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type apiKeyContextKey struct{}

func apiKeyFromContext(ctx context.Context) apikeys.Credential {
	credential, _ := ctx.Value(apiKeyContextKey{}).(apikeys.Credential)
	return credential
}
