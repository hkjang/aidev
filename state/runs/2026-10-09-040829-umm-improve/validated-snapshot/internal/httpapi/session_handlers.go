package httpapi

import (
	"errors"
	"net/http"

	"github.com/hkjang/umm/internal/auth"
	"github.com/jackc/pgx/v5"
)

// currentSessionToken reads the browser session cookie. Every handler in this
// file already runs behind requireSession, so the cookie is always present.
func currentSessionToken(r *http.Request) string {
	cookie, err := r.Cookie(auth.CookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func (s *Server) listSessions(w http.ResponseWriter, r *http.Request) {
	p := principal(r)
	sessions, err := s.Auth.ListSessions(r.Context(), p.User.ID, currentSessionToken(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "로그인 기기를 불러오지 못했습니다.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sessions": sessions})
}

func (s *Server) revokeSession(w http.ResponseWriter, r *http.Request) {
	sessionID, ok := parseID(w, r, "sessionID")
	if !ok {
		return
	}
	p := principal(r)
	if err := s.Auth.RevokeSession(r.Context(), p.User.ID, sessionID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "해당 로그인 기기를 찾을 수 없습니다.")
			return
		}
		writeError(w, http.StatusInternalServerError, "로그인을 종료하지 못했습니다.")
		return
	}
	// Revoking the session that is making the request is a deliberate sign out,
	// so the cookie goes with it.
	if sessionID == p.SessionID {
		auth.ClearSessionCookie(w, r)
	}
	s.Store.Audit(r.Context(), &p.User.ID, "session.revoke", "session", sessionID.String(), map[string]any{})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) revokeOtherSessions(w http.ResponseWriter, r *http.Request) {
	p := principal(r)
	revoked, err := s.Auth.RevokeOtherSessions(r.Context(), p.User.ID, currentSessionToken(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "다른 기기의 로그인을 종료하지 못했습니다.")
		return
	}
	s.Store.Audit(r.Context(), &p.User.ID, "session.revoke_others", "user", p.User.ID.String(), map[string]any{"revoked": revoked})
	writeJSON(w, http.StatusOK, map[string]any{"revoked": revoked})
}
