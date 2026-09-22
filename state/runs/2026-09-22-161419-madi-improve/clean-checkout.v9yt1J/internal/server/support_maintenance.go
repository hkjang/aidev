package server

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// Expiry is enforced on every request without waiting for this audit task.
// Keeping expiration audit in maintenance means idle sessions also receive a
// terminal record. Restores must mark all support sessions revoked.
func (s *Server) expireSupportSessions(ctx context.Context) error {
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	var exists bool
	if e = tx.QueryRow(ctx, `SELECT to_regclass('support_sessions') IS NOT NULL`).Scan(&exists); e != nil || !exists {
		return e
	}
	if e = supportExpireTx(ctx, tx); e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func supportExpireTx(ctx context.Context, tx pgx.Tx) error {
	rows, e := tx.Query(ctx, `UPDATE support_sessions SET status='expired',ended_at=expires_at WHERE id IN(SELECT id FROM support_sessions WHERE status='active' AND expires_at<=now() ORDER BY expires_at LIMIT 500 FOR UPDATE SKIP LOCKED) RETURNING id::text,operator_id::text,target_id::text,reason,expires_at`)
	if e != nil {
		return e
	}
	sessions := []supportSession{}
	for rows.Next() {
		var x supportSession
		if e = rows.Scan(&x.ID, &x.OperatorID, &x.TargetID, &x.Reason, &x.ExpiresAt); e != nil {
			break
		}
		sessions = append(sessions, x)
	}
	if e == nil {
		e = rows.Err()
	}
	rows.Close()
	if e != nil {
		return e
	}
	for _, x := range sessions {
		_, e = tx.Exec(ctx, `INSERT INTO audit_logs(id,user_id,action,resource,ip,details) VALUES($1,$2,'SUPPORT_EXPIRED',$3,'',$4)`, newID(), x.OperatorID, x.ID, jsonValue(map[string]any{"support_session_id": x.ID, "target_user_id": x.TargetID, "reason": x.Reason, "expires_at": x.ExpiresAt, "mode": "read_only_acl_intersection"}))
		if e != nil {
			return e
		}
	}
	return nil
}
