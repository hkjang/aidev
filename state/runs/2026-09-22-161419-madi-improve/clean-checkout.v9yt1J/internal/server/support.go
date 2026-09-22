package server

import (
	"context"
	_ "embed"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
)

//go:embed support.sql
var supportSchema string
var errSupportDenied = errors.New("읽기 전용 지원 진단의 원래 로그인·명시적 권한·대상 계정·만료 시간을 확인하세요")

type supportSession struct {
	ID               string     `json:"id"`
	OperatorID       string     `json:"operator_id"`
	TargetID         string     `json:"target_id"`
	TargetName       string     `json:"target_name"`
	Reason           string     `json:"reason"`
	Status           string     `json:"status"`
	CreatedAt        time.Time  `json:"created_at"`
	ExpiresAt        time.Time  `json:"expires_at"`
	SessionHash      string     `json:"-"`
	Operator, Target *Principal `json:"-"`
}

func (s *Server) migrateSupport(ctx context.Context) error {
	_, e := s.DB.Exec(ctx, supportSchema)
	return e
}

func supportCookie(r *http.Request) (string, error) {
	p := current(r)
	if p == nil || p.Role != "admin" || p.Kind != "user" || p.TokenID != "" || p.ScopeRestricted {
		return "", errSupportDenied
	}
	cookie, e := r.Cookie("madi_session")
	if e != nil || cookie.Value == "" {
		return "", errSupportDenied
	}
	return digest(cookie.Value), nil
}
func supportAuthority(ctx context.Context, q collaborationQuery, operator, session string) (int, error) {
	var maxMinutes int
	var enabled bool
	e := q.QueryRow(ctx, `SELECT coalesce((s.data->>'support_enabled')::boolean,false) AND coalesce(s.data->'support_operator_ids' ? $1,false),coalesce((s.data->>'support_max_minutes')::integer,30) FROM settings s,users u,sessions t WHERE s.id=1 AND u.id=$1::uuid AND NOT u.disabled AND u.role='admin' AND u.kind='user' AND t.user_id=u.id AND t.token_hash=$2 AND t.expires_at>now()`, operator, session).Scan(&enabled, &maxMinutes)
	if e != nil || !enabled || maxMinutes < 1 || maxMinutes > 30 {
		return 0, errSupportDenied
	}
	return maxMinutes, nil
}
func supportLoad(ctx context.Context, q collaborationQuery, id, operator, sessionHash string) (supportSession, error) {
	var session supportSession
	e := q.QueryRow(ctx, `SELECT x.id::text,x.operator_id::text,x.target_id::text,u.name,x.reason,x.status,x.created_at,x.expires_at,x.session_hash FROM support_sessions x JOIN users u ON u.id=x.target_id WHERE x.id=$1 AND x.operator_id=$2 AND x.session_hash=$3`, id, operator, sessionHash).Scan(&session.ID, &session.OperatorID, &session.TargetID, &session.TargetName, &session.Reason, &session.Status, &session.CreatedAt, &session.ExpiresAt, &session.SessionHash)
	return session, e
}
func supportAccess(ctx context.Context, q collaborationQuery, r *http.Request, id string) (supportSession, error) {
	var out supportSession
	cookie, e := supportCookie(r)
	if e != nil || !validID(id) {
		return out, errSupportDenied
	}
	out, e = supportLoad(ctx, q, id, current(r).ID, cookie)
	if e != nil || out.Status != "active" || !out.ExpiresAt.After(time.Now()) {
		return out, errSupportDenied
	}
	maxMinutes, e := supportAuthority(ctx, q, out.OperatorID, cookie)
	if e != nil {
		return out, e
	}
	policyExpiry := out.CreatedAt.Add(time.Duration(maxMinutes) * time.Minute)
	if policyExpiry.Before(out.ExpiresAt) {
		out.ExpiresAt = policyExpiry
	}
	if !out.ExpiresAt.After(time.Now()) {
		return out, errSupportDenied
	}
	operator, target := &Principal{}, &Principal{}
	for _, entry := range []struct {
		id string
		p  *Principal
	}{{out.OperatorID, operator}, {out.TargetID, target}} {
		if e = q.QueryRow(ctx, `SELECT id::text,name,role,kind FROM users WHERE id=$1 AND NOT disabled AND kind='user'`, entry.id).Scan(&entry.p.ID, &entry.p.Name, &entry.p.Role, &entry.p.Kind); e != nil {
			return out, errSupportDenied
		}
	}
	if operator.Role != "admin" || operator.ID == target.ID {
		return out, errSupportDenied
	}
	var common bool
	if q.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspace_members a JOIN workspace_members b ON a.workspace_id=b.workspace_id WHERE a.user_id=$1 AND b.user_id=$2)`, operator.ID, target.ID).Scan(&common) != nil || !common {
		return out, errSupportDenied
	}
	out.Operator, out.Target = operator, target
	return out, nil
}
func supportWorkspaceAllowed(ctx context.Context, q collaborationQuery, session supportSession, id string) bool {
	if !validID(id) {
		return false
	}
	var allowed bool
	e := q.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspace_members a JOIN workspace_members t ON a.workspace_id=t.workspace_id WHERE a.user_id=$1 AND t.user_id=$2 AND a.workspace_id=$3)`, session.OperatorID, session.TargetID, id).Scan(&allowed)
	return e == nil && allowed
}
func supportDocumentAllowed(ctx context.Context, q collaborationQuery, session supportSession, id string) bool {
	if !validID(id) {
		return false
	}
	var allowed bool
	e := q.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM documents d WHERE d.id=$3 AND d.deleted_at IS NULL AND madi_document_allowed($1,d.id,false) AND madi_document_allowed($2,d.id,false))`, session.OperatorID, session.TargetID, id).Scan(&allowed)
	return e == nil && allowed
}
func supportAuditTx(ctx context.Context, tx pgx.Tx, r *http.Request, session supportSession, action, resource, result string, extra map[string]any) error {
	if extra == nil {
		extra = map[string]any{}
	}
	extra["support_session_id"] = session.ID
	extra["target_user_id"] = session.TargetID
	extra["result"] = result
	extra["mode"] = "read_only_acl_intersection"
	extra["reason"] = session.Reason
	_, e := tx.Exec(ctx, `INSERT INTO audit_logs(id,user_id,action,resource,ip,details) VALUES($1,$2,$3,$4,$5,$6)`, newID(), session.OperatorID, action, resource, integrationClientIP(r), jsonValue(extra))
	return e
}
func (s *Server) supportAudit(r *http.Request, session supportSession, action, resource, result string, extra map[string]any) error {
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		return e
	}
	defer tx.Rollback(r.Context())
	if e = supportAuditTx(r.Context(), tx, r, session, action, resource, result, extra); e != nil {
		return e
	}
	return tx.Commit(r.Context())
}
func supportReasonTx(ctx context.Context, tx pgx.Tx, reason string) (string, []ProtectionFinding, string, error) {
	var raw []byte
	if e := tx.QueryRow(ctx, `SELECT data FROM protection_settings WHERE id=1 FOR SHARE`).Scan(&raw); e != nil {
		return "", nil, "", e
	}
	cfg, e := decodeProtectionSettings(raw)
	if e != nil {
		return "", nil, "", e
	}
	if !boolean(cfg, "enabled") {
		return reason, nil, "off", nil
	}
	masked, findings, e := protectionScanTx(ctx, tx, cfg, reason)
	if e != nil {
		return "", nil, "", e
	}
	mode := str(cfg, "mode")
	if len(findings) > 0 && mode == "block" {
		return "", findings, mode, ProtectionError{Code: "blocked", Findings: findings}
	}
	if mode == "mask" {
		reason = masked
	}
	return reason, findings, mode, nil
}
func (s *Server) supportDenied(w http.ResponseWriter, r *http.Request, session supportSession, resource string) {
	if validID(session.ID) && validID(session.OperatorID) {
		_ = s.supportAudit(r, session, "SUPPORT_ACCESS", resource, "denied", nil)
	} else {
		s.audit(r, "SUPPORT_ACCESS_DENIED", resource, map[string]any{"support_session_id": r.PathValue("id"), "mode": "read_only_acl_intersection"})
	}
	apiError(w, 403, errSupportDenied.Error())
}
