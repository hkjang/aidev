package server

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type apiKey struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	Name        string     `json:"name"`
	Prefix      string     `json:"prefix"`
	Scopes      []string   `json:"scopes"`
	WorkspaceID string     `json:"workspace_id"`
	ExpiresAt   time.Time  `json:"expires_at"`
	LastUsedAt  *time.Time `json:"last_used_at"`
	CreatedAt   time.Time  `json:"created_at"`
	RevokedAt   *time.Time `json:"revoked_at"`
	IPAllowlist []string   `json:"ip_allowlist"`
	RateLimit   int        `json:"rate_limit"`
}

const keyColumns = `id::text,user_id::text,name,prefix,scopes,workspace_id::text,expires_at,last_used_at,created_at,revoked_at,ip_allowlist,rate_limit`

func scanAPIKey(row pgx.Row) (apiKey, error) {
	var k apiKey
	err := row.Scan(&k.ID, &k.UserID, &k.Name, &k.Prefix, &k.Scopes, &k.WorkspaceID, &k.ExpiresAt, &k.LastUsedAt, &k.CreatedAt, &k.RevokedAt, &k.IPAllowlist, &k.RateLimit)
	return k, err
}

func integrationSecret() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func integrationHash(value string) string {
	h := sha256.Sum256([]byte(value))
	return hex.EncodeToString(h[:])
}

type tokenAuthError struct {
	Status  int
	Message string
}

// This unexported context key is only set when MCP dispatches an already counted
// HTTP request into REST. Authorization and ACL checks still execute normally.
type integrationCountedTokenKey struct{}

func (e *tokenAuthError) Error() string { return e.Message }
func integrationAuthStatus(err error) int {
	var target *tokenAuthError
	if errors.As(err, &target) {
		return target.Status
	}
	return http.StatusUnauthorized
}

func integrationClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func integrationIPAllowed(ip string, allowlist []string) bool {
	if len(allowlist) == 0 {
		return true
	}
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return false
	}
	addr = addr.Unmap()
	for _, entry := range allowlist {
		if prefix, err := netip.ParsePrefix(entry); err == nil && prefix.Contains(addr) {
			return true
		}
		if exact, err := netip.ParseAddr(entry); err == nil && exact.Unmap() == addr {
			return true
		}
	}
	return false
}

func (s *Server) tokenPrincipal(r *http.Request) (*Principal, error) {
	parts := strings.Fields(r.Header.Get("Authorization"))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || !strings.HasPrefix(parts[1], "madi_") {
		return nil, errors.New("유효한 API 키가 필요합니다.")
	}
	var p Principal
	var ipAllowlist []string
	var expires time.Time
	var revoked *time.Time
	var disabled bool
	err := s.DB.QueryRow(r.Context(), `SELECT u.id::text,u.email,u.name,u.role,u.kind,u.disabled,k.id::text,k.scopes,k.workspace_id::text,k.expires_at,k.revoked_at,k.ip_allowlist FROM api_keys k JOIN users u ON u.id=k.user_id WHERE k.token_hash=$1`, integrationHash(parts[1])).Scan(&p.ID, &p.Email, &p.Name, &p.Role, &p.Kind, &disabled, &p.TokenID, &p.Scopes, &p.WorkspaceID, &expires, &revoked, &ipAllowlist)
	if err != nil || disabled || revoked != nil || !time.Now().Before(expires) {
		return nil, errors.New("만료되었거나 폐기된 API 키입니다.")
	}
	if !integrationIPAllowed(integrationClientIP(r), ipAllowlist) {
		return nil, &tokenAuthError{http.StatusForbidden, "허용되지 않은 IP 주소입니다."}
	}
	settings, err := s.settings(r.Context())
	if err != nil {
		return nil, err
	}
	allowed := settingStrings(settings, "allowed_key_scopes", keyScopes)
	p.Scopes = slices.DeleteFunc(p.Scopes, func(scope string) bool { return !slices.Contains(allowed, scope) })
	if !s.canWorkspace(r.Context(), &p, p.WorkspaceID, false) {
		return nil, &tokenAuthError{http.StatusForbidden, "키의 워크스페이스 접근 권한이 없습니다."}
	}
	if countedID, _ := r.Context().Value(integrationCountedTokenKey{}).(string); countedID == p.TokenID {
		return &p, nil
	}
	result, err := s.DB.Exec(r.Context(), `UPDATE api_keys SET last_used_at=now(),rate_count=CASE WHEN rate_window < date_trunc('minute',now()) THEN 1 ELSE rate_count+1 END,rate_window=date_trunc('minute',now()) WHERE id=$1 AND revoked_at IS NULL AND expires_at>now() AND (rate_window < date_trunc('minute',now()) OR rate_count<rate_limit)`, p.TokenID)
	if err != nil {
		return nil, err
	}
	if result.RowsAffected() != 1 {
		return nil, &tokenAuthError{http.StatusTooManyRequests, "API 키의 분당 호출 제한을 초과했습니다."}
	}
	return &p, nil
}

func keyOwner(r *http.Request) string {
	p := current(r)
	if p.Role == "admin" && r.URL.Query().Get("user_id") != "" {
		return r.URL.Query().Get("user_id")
	}
	return p.ID
}

func (s *Server) listKeys(w http.ResponseWriter, r *http.Request) {
	rows, err := s.DB.Query(r.Context(), `SELECT `+keyColumns+` FROM api_keys WHERE user_id=$1 ORDER BY created_at DESC`, keyOwner(r))
	if err != nil {
		apiError(w, 500, "API 키 목록을 불러올 수 없습니다.")
		return
	}
	defer rows.Close()
	out := []apiKey{}
	for rows.Next() {
		k, err := scanAPIKey(rows)
		if err != nil {
			apiError(w, 500, "API 키를 읽을 수 없습니다.")
			return
		}
		out = append(out, k)
	}
	if rows.Err() != nil {
		apiError(w, 500, "API 키를 읽을 수 없습니다.")
		return
	}
	jsonResponse(w, 200, out)
}

type keyInput struct {
	Name          string   `json:"name"`
	Scopes        []string `json:"scopes"`
	WorkspaceID   string   `json:"workspace_id"`
	ExpiresInDays int      `json:"expires_in_days"`
	IPAllowlist   []string `json:"ip_allowlist"`
	RateLimit     int      `json:"rate_limit"`
	UserID        string   `json:"user_id"`
}

func validateKeyInput(in *keyInput, settings map[string]any) error {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" || len(in.Name) > 120 {
		return errors.New("키 이름은 1~120자로 입력하세요.")
	}
	if in.WorkspaceID == "" {
		return errors.New("키가 사용할 워크스페이스를 선택하세요.")
	}
	if len(in.Scopes) == 0 {
		return errors.New("권한을 하나 이상 선택하세요.")
	}
	allowed := settingStrings(settings, "allowed_key_scopes", keyScopes)
	for _, scope := range in.Scopes {
		if (!slices.Contains(keyScopes, scope) && scope != scimProvisionScope) || !slices.Contains(allowed, scope) {
			return fmt.Errorf("허용되지 않은 권한입니다: %s", scope)
		}
	}
	slices.Sort(in.Scopes)
	in.Scopes = slices.Compact(in.Scopes)
	if in.ExpiresInDays == 0 {
		in.ExpiresInDays = settingInt(settings, "default_key_days", 90)
	}
	if in.ExpiresInDays < 1 || in.ExpiresInDays > 3650 {
		return errors.New("유효기간은 1~3650일로 입력하세요.")
	}
	if in.RateLimit == 0 {
		in.RateLimit = 120
	}
	if in.RateLimit < 1 || in.RateLimit > 10000 {
		return errors.New("분당 호출 제한은 1~10000으로 입력하세요.")
	}
	if len(in.IPAllowlist) > 100 {
		return errors.New("IP 허용 목록은 최대 100개입니다.")
	}
	for i, entry := range in.IPAllowlist {
		entry = strings.TrimSpace(entry)
		in.IPAllowlist[i] = entry
		if _, err := netip.ParseAddr(entry); err == nil {
			continue
		}
		if _, err := netip.ParsePrefix(entry); err != nil {
			return fmt.Errorf("IP 또는 CIDR 형식이 아닙니다: %s", entry)
		}
	}
	if in.IPAllowlist == nil {
		in.IPAllowlist = []string{}
	}
	return nil
}

func (s *Server) createKey(w http.ResponseWriter, r *http.Request) {
	var in keyInput
	if err := decode(r, &in); err != nil {
		apiError(w, 400, "API 키 설정을 확인하세요.")
		return
	}
	settings, err := s.settings(r.Context())
	if err != nil {
		apiError(w, 500, "설정을 불러올 수 없습니다.")
		return
	}
	if err := validateKeyInput(&in, settings); err != nil {
		apiError(w, 400, err.Error())
		return
	}
	p := current(r)
	owner := p.ID
	if in.UserID != "" && in.UserID != p.ID {
		if p.Role != "admin" {
			apiError(w, 403, "다른 사용자의 키를 발급할 수 없습니다.")
			return
		}
		owner = in.UserID
		var ownerP Principal
		var disabled bool
		err := s.DB.QueryRow(r.Context(), `SELECT id::text,role,disabled FROM users WHERE id=$1`, owner).Scan(&ownerP.ID, &ownerP.Role, &disabled)
		if err != nil || disabled || !s.canWorkspace(r.Context(), &ownerP, in.WorkspaceID, false) {
			apiError(w, 403, "사용자의 워크스페이스 접근 권한을 확인하세요.")
			return
		}
	}
	if !s.canWorkspace(r.Context(), p, in.WorkspaceID, false) {
		apiError(w, 403, "워크스페이스 접근 권한이 없습니다.")
		return
	}
	if !s.provisionKeyOwnerAllowed(r, owner, in.Scopes) {
		apiError(w, 403, "SCIM 키는 서비스 관리자가 서비스 계정에만 발급할 수 있습니다.")
		return
	}
	token := "madi_" + integrationSecret()
	k, err := scanAPIKey(s.DB.QueryRow(r.Context(), `INSERT INTO api_keys(id,user_id,name,prefix,token_hash,scopes,workspace_id,expires_at,ip_allowlist,rate_limit) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING `+keyColumns, newID(), owner, in.Name, token[:13], integrationHash(token), in.Scopes, in.WorkspaceID, time.Now().Add(time.Duration(in.ExpiresInDays)*24*time.Hour), in.IPAllowlist, in.RateLimit))
	if err != nil {
		apiError(w, 500, "API 키를 발급할 수 없습니다.")
		return
	}
	s.audit(r, "KEY_CREATE", k.ID, map[string]any{"scopes": k.Scopes, "workspace_id": k.WorkspaceID, "user_id": owner})
	jsonResponse(w, 201, map[string]any{"key": k, "token": token})
}

func (s *Server) ownedKey(r *http.Request) (apiKey, error) {
	k, err := scanAPIKey(s.DB.QueryRow(r.Context(), `SELECT `+keyColumns+` FROM api_keys WHERE id=$1 AND (user_id=$2 OR $3::boolean)`, r.PathValue("id"), current(r).ID, current(r).Role == "admin"))
	return k, err
}

func (s *Server) provisionKeyOwnerAllowed(r *http.Request, owner string, scopes []string) bool {
	if !slices.Contains(scopes, scimProvisionScope) {
		return true
	}
	p := current(r)
	if p.Role != "admin" || p.TokenID != "" || p.ScopeRestricted {
		return false
	}
	var allowed bool
	return s.DB.QueryRow(r.Context(), `SELECT kind='service' AND NOT disabled FROM users WHERE id=$1`, owner).Scan(&allowed) == nil && allowed
}

func (s *Server) updateKey(w http.ResponseWriter, r *http.Request) {
	k, err := s.ownedKey(r)
	if err != nil {
		apiError(w, 404, "API 키를 찾을 수 없습니다.")
		return
	}
	if k.RevokedAt != nil {
		apiError(w, 409, "폐기된 키는 수정할 수 없습니다.")
		return
	}
	in := keyInput{Name: k.Name, Scopes: k.Scopes, WorkspaceID: k.WorkspaceID, IPAllowlist: k.IPAllowlist, RateLimit: k.RateLimit, ExpiresInDays: 1}
	var patch map[string]any
	if err := decode(r, &patch); err != nil {
		apiError(w, 400, "설정을 확인하세요.")
		return
	}
	// Decode into initialized input to support genuine partial updates.
	encoded, err := json.Marshal(patch)
	if err != nil {
		apiError(w, 400, "설정을 확인하세요.")
		return
	}
	if err = json.Unmarshal(encoded, &in); err != nil {
		apiError(w, 400, "설정을 확인하세요.")
		return
	}
	settings, err := s.settings(r.Context())
	if err != nil {
		apiError(w, 500, "설정을 불러올 수 없습니다.")
		return
	}
	if err := validateKeyInput(&in, settings); err != nil {
		apiError(w, 400, err.Error())
		return
	}
	if in.WorkspaceID != k.WorkspaceID {
		apiError(w, 400, "워크스페이스 변경은 새 키를 발급하세요.")
		return
	}
	if !s.provisionKeyOwnerAllowed(r, k.UserID, append(append([]string{}, k.Scopes...), in.Scopes...)) {
		apiError(w, 403, "SCIM 키는 서비스 관리자만 관리할 수 있습니다.")
		return
	}
	expires := k.ExpiresAt
	if _, ok := patch["expires_in_days"]; ok {
		expires = time.Now().Add(time.Duration(in.ExpiresInDays) * 24 * time.Hour)
	}
	k, err = scanAPIKey(s.DB.QueryRow(r.Context(), `UPDATE api_keys SET name=$2,scopes=$3,expires_at=$4,ip_allowlist=$5,rate_limit=$6 WHERE id=$1 AND revoked_at IS NULL RETURNING `+keyColumns, k.ID, in.Name, in.Scopes, expires, in.IPAllowlist, in.RateLimit))
	if err != nil {
		apiError(w, 409, "API 키가 변경되었습니다. 다시 확인하세요.")
		return
	}
	s.audit(r, "KEY_UPDATE", k.ID, map[string]any{"scopes": k.Scopes, "expires_at": k.ExpiresAt})
	jsonResponse(w, 200, k)
}

func (s *Server) revokeKey(w http.ResponseWriter, r *http.Request) {
	k, err := s.ownedKey(r)
	if err != nil {
		apiError(w, 404, "API 키를 찾을 수 없습니다.")
		return
	}
	if _, err = s.DB.Exec(r.Context(), `UPDATE api_keys SET revoked_at=COALESCE(revoked_at,now()) WHERE id=$1`, k.ID); err != nil {
		apiError(w, 500, "키를 폐기할 수 없습니다.")
		return
	}
	s.audit(r, "KEY_REVOKE", k.ID, nil)
	jsonResponse(w, 200, map[string]bool{"ok": true})
}

func (s *Server) rotateKey(w http.ResponseWriter, r *http.Request) {
	k, err := s.ownedKey(r)
	if err != nil {
		apiError(w, 404, "API 키를 찾을 수 없습니다.")
		return
	}
	if k.RevokedAt != nil || !time.Now().Before(k.ExpiresAt) {
		apiError(w, 409, "만료 또는 폐기된 키는 새로 발급하세요.")
		return
	}
	if !s.provisionKeyOwnerAllowed(r, k.UserID, k.Scopes) {
		apiError(w, 403, "SCIM 키는 서비스 관리자만 회전할 수 있습니다.")
		return
	}
	token := "madi_" + integrationSecret()
	// One atomic UPDATE makes the previous secret immediately unusable.
	k, err = scanAPIKey(s.DB.QueryRow(r.Context(), `UPDATE api_keys SET token_hash=$2,prefix=$3,rate_count=0,rate_window=now() WHERE id=$1 AND revoked_at IS NULL AND expires_at>now() RETURNING `+keyColumns, k.ID, integrationHash(token), token[:13]))
	if err != nil {
		apiError(w, 409, "키를 회전할 수 없습니다. 현재 상태를 확인하세요.")
		return
	}
	s.audit(r, "KEY_ROTATE", k.ID, nil)
	jsonResponse(w, 200, map[string]any{"key": k, "token": token})
}
