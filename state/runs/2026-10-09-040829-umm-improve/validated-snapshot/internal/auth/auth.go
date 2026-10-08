package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/store"
	"github.com/hkjang/umm/internal/textutil"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

const CookieName = "umm_session"

var ErrInvalidCredentials = errors.New("invalid credentials")

type contextKey string

const principalKey contextKey = "principal"

type Principal struct {
	User      store.User
	Scopes    map[string]bool
	AuthType  string
	SessionID uuid.UUID
}

type Service struct{ Store *store.Store }

type APIKey struct {
	ID           uuid.UUID  `json:"id"`
	Name         string     `json:"name"`
	Prefix       string     `json:"prefix"`
	Scopes       []string   `json:"scopes"`
	Status       string     `json:"status"`
	ExpiresAt    *time.Time `json:"expiresAt,omitempty"`
	OverlapUntil *time.Time `json:"overlapUntil,omitempty"`
	LastUsedAt   *time.Time `json:"lastUsedAt,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
}

func PrincipalFrom(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(principalKey).(Principal)
	return p, ok
}

func randomToken(bytes int) (string, error) {
	buf := make([]byte, bytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func newAPIKeyMaterial(generate func(int) (string, error)) (string, string, error) {
	secret, err := generate(32)
	if err != nil {
		return "", "", err
	}
	prefixRaw, err := generate(6)
	if err != nil {
		return "", "", err
	}
	if len(prefixRaw) < 8 {
		return "", "", errors.New("generated API key prefix is too short")
	}
	prefix := strings.ToLower(prefixRaw[:8])
	return prefix, "umm_key_" + prefix + "_" + secret, nil
}

func digest(v string) []byte { s := sha256.Sum256([]byte(v)); return s[:] }

// SessionOrigin records where a session was created so a user can recognise it
// in the active session list and revoke one they do not recognise.
type SessionOrigin struct {
	UserAgent string
	ClientIP  string
}

// OriginOf reads the origin of the request that is creating a session. The
// user agent is truncated because it is only ever shown to a human.
func OriginOf(r *http.Request) SessionOrigin {
	agent := boundedUserAgent(r.UserAgent())
	host := r.RemoteAddr
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	return SessionOrigin{UserAgent: agent, ClientIP: host}
}

func boundedUserAgent(agent string) string {
	return textutil.LimitUTF8Bytes(strings.TrimSpace(agent), 300)
}

func (s *Service) PasswordLogin(ctx context.Context, username, password string, origin SessionOrigin) (store.User, string, error) {
	u, hash, err := s.Store.UserByUsername(ctx, strings.TrimSpace(username))
	return s.finishPasswordLogin(ctx, nil, u, hash, password, origin, err)
}

// PasswordLoginTx verifies credentials and creates the session on the login
// throttle transaction's single connection. This keeps the throttle safe even
// when PostgreSQL pool_max_conns=1.
func (s *Service) PasswordLoginTx(ctx context.Context, tx pgx.Tx, username, password string, origin SessionOrigin) (store.User, string, error) {
	u, hash, err := s.Store.UserByUsernameTx(ctx, tx, strings.TrimSpace(username))
	return s.finishPasswordLogin(ctx, tx, u, hash, password, origin, err)
}

func (s *Service) finishPasswordLogin(ctx context.Context, tx pgx.Tx, u store.User, hash, password string, origin SessionOrigin, lookupErr error) (store.User, string, error) {
	if lookupErr != nil {
		return store.User{}, "", ErrInvalidCredentials
	}
	if !u.Active || hash == "" || bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return store.User{}, "", ErrInvalidCredentials
	}
	if tx != nil {
		token, err := s.CreateSessionTx(ctx, tx, u.ID, origin)
		return u, token, err
	}
	token, err := s.CreateSession(ctx, u.ID, origin)
	return u, token, err
}

func (s *Service) CreateSession(ctx context.Context, userID uuid.UUID, origin SessionOrigin) (string, error) {
	return s.createSession(ctx, nil, userID, origin)
}

// CreateSessionTx persists a session on an existing transaction connection.
func (s *Service) CreateSessionTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID, origin SessionOrigin) (string, error) {
	return s.createSession(ctx, tx, userID, origin)
}

func (s *Service) createSession(ctx context.Context, tx pgx.Tx, userID uuid.UUID, origin SessionOrigin) (string, error) {
	token, err := randomToken(32)
	if err != nil {
		return "", err
	}
	var general struct {
		SessionHours int `json:"session_hours"`
	}
	if tx == nil {
		_ = s.Store.GetSetting(ctx, "general", &general)
	} else {
		_ = s.Store.GetSettingTx(ctx, tx, "general", &general)
	}
	if general.SessionHours < 1 || general.SessionHours > 720 {
		general.SessionHours = 24
	}
	origin.UserAgent = boundedUserAgent(origin.UserAgent)
	if tx == nil {
		_, err = s.Store.Pool.Exec(ctx, `INSERT INTO sessions(user_id,token_hash,expires_at,user_agent,client_ip) VALUES($1,$2,now()+make_interval(hours=>$3),$4,$5)`, userID, digest(token), general.SessionHours, origin.UserAgent, origin.ClientIP)
	} else {
		_, err = tx.Exec(ctx, `INSERT INTO sessions(user_id,token_hash,expires_at,user_agent,client_ip) VALUES($1,$2,now()+make_interval(hours=>$3),$4,$5)`, userID, digest(token), general.SessionHours, origin.UserAgent, origin.ClientIP)
	}
	return token, err
}

func (s *Service) DeleteSession(ctx context.Context, token string) {
	if token != "" {
		_, _ = s.Store.Pool.Exec(ctx, `DELETE FROM sessions WHERE token_hash=$1`, digest(token))
	}
}

// Session is one active browser login, as shown on the personal settings page.
type Session struct {
	ID         uuid.UUID `json:"id"`
	UserAgent  string    `json:"userAgent"`
	ClientIP   string    `json:"clientIp"`
	Current    bool      `json:"current"`
	CreatedAt  time.Time `json:"createdAt"`
	LastSeenAt time.Time `json:"lastSeenAt"`
	ExpiresAt  time.Time `json:"expiresAt"`
}

func (s *Service) ListSessions(ctx context.Context, userID uuid.UUID, currentToken string) ([]Session, error) {
	rows, err := s.Store.Pool.Query(ctx, `SELECT id,user_agent,client_ip,created_at,last_seen_at,expires_at,token_hash=$2 FROM sessions WHERE user_id=$1 AND expires_at>now() ORDER BY last_seen_at DESC`, userID, digest(currentToken))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Session{}
	for rows.Next() {
		var item Session
		if err := rows.Scan(&item.ID, &item.UserAgent, &item.ClientIP, &item.CreatedAt, &item.LastSeenAt, &item.ExpiresAt, &item.Current); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

// RevokeSession ends one session. A user may only revoke their own.
func (s *Service) RevokeSession(ctx context.Context, userID, sessionID uuid.UUID) error {
	command, err := s.Store.Pool.Exec(ctx, `DELETE FROM sessions WHERE id=$1 AND user_id=$2`, sessionID, userID)
	if err == nil && command.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return err
}

// RevokeOtherSessions signs every other device out, which is what a user needs
// after a lost laptop or a shared-computer login.
func (s *Service) RevokeOtherSessions(ctx context.Context, userID uuid.UUID, currentToken string) (int64, error) {
	command, err := s.Store.Pool.Exec(ctx, `DELETE FROM sessions WHERE user_id=$1 AND token_hash<>$2`, userID, digest(currentToken))
	if err != nil {
		return 0, err
	}
	return command.RowsAffected(), nil
}

// sessionActivityInterval throttles the last_seen_at write so an active tab
// costs one UPDATE every few minutes instead of one per request.
const sessionActivityInterval = 5 * time.Minute

func (s *Service) Authenticate(r *http.Request) (Principal, error) {
	if raw := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")); raw != "" && raw != r.Header.Get("Authorization") {
		return s.authenticateAPIKey(r.Context(), raw)
	}
	cookie, err := r.Cookie(CookieName)
	if err != nil {
		return Principal{}, err
	}
	var u store.User
	var sessionID uuid.UUID
	var lastSeen time.Time
	err = s.Store.Pool.QueryRow(r.Context(), `SELECT u.id,u.username,u.display_name,COALESCE(u.email,''),u.role,u.team_id,u.active,s.id,s.last_seen_at FROM sessions s JOIN users u ON u.id=s.user_id WHERE s.token_hash=$1 AND s.expires_at>now() AND u.active`, digest(cookie.Value)).Scan(&u.ID, &u.Username, &u.DisplayName, &u.Email, &u.Role, &u.TeamID, &u.Active, &sessionID, &lastSeen)
	if err != nil {
		return Principal{}, err
	}
	if time.Since(lastSeen) > sessionActivityInterval {
		_, _ = s.Store.Pool.Exec(r.Context(), `UPDATE sessions SET last_seen_at=now() WHERE id=$1`, sessionID)
	}
	return Principal{User: u, Scopes: map[string]bool{"*": true}, AuthType: "session", SessionID: sessionID}, nil
}

func (s *Service) authenticateAPIKey(ctx context.Context, raw string) (Principal, error) {
	var u store.User
	var scopes []string
	var keyID uuid.UUID
	err := s.Store.Pool.QueryRow(ctx, `SELECT u.id,u.username,u.display_name,COALESCE(u.email,''),u.role,u.team_id,u.active,k.scopes,k.id FROM api_keys k JOIN users u ON u.id=k.user_id WHERE k.secret_hash=$1 AND k.status IN ('active','overlap') AND (k.expires_at IS NULL OR k.expires_at>now()) AND (k.status!='overlap' OR k.overlap_until>now()) AND u.active`, digest(raw)).Scan(&u.ID, &u.Username, &u.DisplayName, &u.Email, &u.Role, &u.TeamID, &u.Active, &scopes, &keyID)
	if err != nil {
		return Principal{}, err
	}
	_, _ = s.Store.Pool.Exec(ctx, `UPDATE api_keys SET last_used_at=now() WHERE id=$1`, keyID)
	scopeMap := map[string]bool{}
	for _, v := range scopes {
		scopeMap[v] = true
	}
	return Principal{User: u, Scopes: scopeMap, AuthType: "api_key"}, nil
}

func (s *Service) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, err := s.Authenticate(r)
		if err == nil {
			r = r.WithContext(context.WithValue(r.Context(), principalKey, p))
		}
		next.ServeHTTP(w, r)
	})
}

func Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := PrincipalFrom(r.Context()); !ok {
			http.Error(w, "authentication required", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, ok := PrincipalFrom(r.Context())
		if !ok || p.User.Role != "admin" {
			http.Error(w, "administrator required", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func RequireScope(scope string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, ok := PrincipalFrom(r.Context())
		if !ok {
			http.Error(w, "authentication required", 401)
			return
		}
		if !p.Scopes["*"] && !p.Scopes[scope] {
			http.Error(w, "missing scope: "+scope, 403)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Service) ListKeys(ctx context.Context, userID uuid.UUID) ([]APIKey, error) {
	rows, err := s.Store.Pool.Query(ctx, `SELECT id,name,prefix,scopes,status,expires_at,overlap_until,last_used_at,created_at FROM api_keys WHERE user_id=$1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []APIKey{}
	for rows.Next() {
		var k APIKey
		if err := rows.Scan(&k.ID, &k.Name, &k.Prefix, &k.Scopes, &k.Status, &k.ExpiresAt, &k.OverlapUntil, &k.LastUsedAt, &k.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

func (s *Service) CreateKey(ctx context.Context, userID uuid.UUID, name string, scopes []string, days int) (APIKey, string, error) {
	prefix, raw, err := newAPIKeyMaterial(randomToken)
	if err != nil {
		return APIKey{}, "", err
	}
	var expires *time.Time
	if days > 0 {
		t := time.Now().Add(time.Duration(days) * 24 * time.Hour)
		expires = &t
	}
	var k APIKey
	err = s.Store.Pool.QueryRow(ctx, `INSERT INTO api_keys(user_id,name,prefix,secret_hash,scopes,expires_at) VALUES($1,$2,$3,$4,$5,$6) RETURNING id,name,prefix,scopes,status,expires_at,overlap_until,last_used_at,created_at`, userID, name, prefix, digest(raw), scopes, expires).Scan(&k.ID, &k.Name, &k.Prefix, &k.Scopes, &k.Status, &k.ExpiresAt, &k.OverlapUntil, &k.LastUsedAt, &k.CreatedAt)
	return k, raw, err
}

func (s *Service) RotateKey(ctx context.Context, userID, keyID uuid.UUID, overlapHours int) (APIKey, string, error) {
	tx, err := s.Store.Pool.Begin(ctx)
	if err != nil {
		return APIKey{}, "", err
	}
	defer tx.Rollback(ctx)
	var name string
	var scopes []string
	var expires *time.Time
	if err = tx.QueryRow(ctx, `SELECT name,scopes,expires_at FROM api_keys WHERE id=$1 AND user_id=$2 AND status='active' FOR UPDATE`, keyID, userID).Scan(&name, &scopes, &expires); err != nil {
		return APIKey{}, "", err
	}
	prefix, raw, err := newAPIKeyMaterial(randomToken)
	if err != nil {
		return APIKey{}, "", err
	}
	var k APIKey
	err = tx.QueryRow(ctx, `INSERT INTO api_keys(user_id,name,prefix,secret_hash,scopes,expires_at) VALUES($1,$2,$3,$4,$5,$6) RETURNING id,name,prefix,scopes,status,expires_at,overlap_until,last_used_at,created_at`, userID, name+" (rotated)", prefix, digest(raw), scopes, expires).Scan(&k.ID, &k.Name, &k.Prefix, &k.Scopes, &k.Status, &k.ExpiresAt, &k.OverlapUntil, &k.LastUsedAt, &k.CreatedAt)
	if err != nil {
		return APIKey{}, "", err
	}
	_, err = tx.Exec(ctx, `UPDATE api_keys SET status=CASE WHEN $3>0 THEN 'overlap' ELSE 'revoked' END,overlap_until=CASE WHEN $3>0 THEN now()+make_interval(hours=>$3) END,replaced_by=$1,revoked_at=CASE WHEN $3<=0 THEN now() END WHERE id=$2 AND user_id=$4`, k.ID, keyID, overlapHours, userID)
	if err != nil {
		return APIKey{}, "", err
	}
	if err = tx.Commit(ctx); err != nil {
		return APIKey{}, "", err
	}
	return k, raw, nil
}

func (s *Service) RevokeKey(ctx context.Context, userID, keyID uuid.UUID) error {
	cmd, err := s.Store.Pool.Exec(ctx, `UPDATE api_keys SET status='revoked',revoked_at=now() WHERE id=$1 AND user_id=$2 AND status!='revoked'`, keyID, userID)
	if err == nil && cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return err
}

// ForgetRevokedKey removes a key that has already been revoked.
//
// Revoking and forgetting are two steps on purpose. Revoking is what stops the
// key working and must not be undoable by a slip; forgetting only tidies the
// list, and cannot happen to a key that still works.
//
// Without this a revoked key stayed on the screen for good. Someone pressing
// 폐기 saw the row remain and reasonably read that as deletion not working.
func (s *Service) ForgetRevokedKey(ctx context.Context, userID, keyID uuid.UUID) error {
	cmd, err := s.Store.Pool.Exec(ctx, `DELETE FROM api_keys WHERE id=$1 AND user_id=$2 AND status='revoked'`, keyID, userID)
	if err == nil && cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return err
}

func (s *Service) UpdateKeyScopes(ctx context.Context, userID, keyID uuid.UUID, scopes []string) error {
	cmd, err := s.Store.Pool.Exec(ctx, `UPDATE api_keys SET scopes=$3 WHERE id=$1 AND user_id=$2 AND status='active'`, keyID, userID, scopes)
	if err == nil && cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return err
}
