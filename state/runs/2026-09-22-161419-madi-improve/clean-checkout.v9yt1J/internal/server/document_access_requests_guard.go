package server

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
)

func documentAccessActorTx(r *http.Request, tx pgx.Tx, wid string, writing bool) error {
	p := current(r)
	denied := errors.New("로그인 세션 또는 워크스페이스 권한이 변경되었습니다")
	if !personalAccessRequest(p) {
		return denied
	}
	var allowed bool
	if tx.QueryRow(r.Context(), `SELECT NOT u.disabled AND u.kind='user' AND (NOT $3 OR (u.role<>'viewer' AND m.role IN ('owner','admin','editor'))) FROM users u JOIN workspace_members m ON m.user_id=u.id WHERE u.id=$1 AND m.workspace_id=$2 FOR SHARE OF u,m`, p.ID, wid, writing).Scan(&allowed) != nil || !allowed {
		return denied
	}
	cookie, e := r.Cookie("madi_session")
	if e != nil {
		return denied
	}
	var userID string
	if tx.QueryRow(r.Context(), "SELECT user_id::text FROM sessions WHERE token_hash=$1 AND user_id=$2 AND expires_at>clock_timestamp() FOR SHARE", digest(cookie.Value), p.ID).Scan(&userID) != nil {
		return denied
	}
	return nil
}
