package server

import (
	"context"
	"net/http"

	"github.com/jackc/pgx/v5"
)

func (s *Server) registerSystemStatus() {
	s.admin("GET /api/v1/admin/system-status/policy", s.getSystemStatusPolicy)
	s.admin("PUT /api/v1/admin/system-status/policy", s.putSystemStatusPolicy)
	s.handle("GET /api/v1/documents/{id}/system-status", s.getSystemStatus)
	s.handle("PUT /api/v1/documents/{id}/system-status", s.putSystemStatus)
	s.handle("DELETE /api/v1/documents/{id}/system-status", s.deleteSystemStatus)
	s.handle("POST /api/v1/documents/{id}/system-status/reports", s.reportSystemStatus)
}
func (s *Server) getSystemStatusPolicy(w http.ResponseWriter, r *http.Request) {
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		systemStatusError(w, e)
		return
	}
	defer tx.Rollback(r.Context())
	p, e := systemStatusPolicyTx(r.Context(), tx, false)
	if e != nil {
		systemStatusError(w, e)
		return
	}
	if e = distributionAdminTx(r, tx); e != nil {
		apiError(w, 403, "현재 서비스 관리자 로그인 권한을 다시 확인하세요")
		return
	}
	history, e := runbookRowsTx(r.Context(), tx, `SELECT jsonb_build_object('revision',revision,'policy',policy,'created_at',created_at) FROM system_status_policy_history ORDER BY revision DESC LIMIT 50`)
	if e != nil {
		systemStatusError(w, e)
		return
	}
	if e = tx.Commit(r.Context()); e != nil {
		systemStatusError(w, e)
		return
	}
	jsonResponse(w, 200, map[string]any{"policy": p, "history": history, "notice": systemStatusNotice})
}
func (s *Server) putSystemStatusPolicy(w http.ResponseWriter, r *http.Request) {
	var in struct {
		systemStatusPolicy
		Confirm bool `json:"confirm"`
	}
	if decode(r, &in) != nil || !systemStatusRevision(in.Revision, false) || in.MaxTTL < 60 || in.MaxTTL > 604800 || in.MaxAge < 60 || in.MaxAge > 604800 || in.Retention < 8 || in.Retention > 3650 || !in.Confirm {
		apiError(w, 400, "현재 정책 revision, TTL·최대 과거 관측 60~604800초, 보존 8~3650일과 명시 확인이 필요합니다")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		systemStatusError(w, e)
		return
	}
	defer tx.Rollback(r.Context())
	p, e := systemStatusPolicyTx(r.Context(), tx, true)
	if e != nil {
		systemStatusError(w, e)
		return
	}
	if p.Revision != in.Revision {
		apiError(w, 409, "관리 정책이 변경되었습니다. 현재 설정을 다시 확인하세요")
		return
	}
	if e = distributionAdminTx(r, tx); e != nil {
		apiError(w, 403, "현재 서비스 관리자 로그인 권한을 다시 확인하세요")
		return
	}
	in.Revision++
	_, e = tx.Exec(r.Context(), `UPDATE system_status_policy SET revision=$1,enabled=$2,max_ttl_seconds=$3,max_observation_age_seconds=$4,retention_days=$5,updated_at=clock_timestamp() WHERE id=1`, in.Revision, in.Enabled, in.MaxTTL, in.MaxAge, in.Retention)
	if e == nil {
		_, e = tx.Exec(r.Context(), `INSERT INTO system_status_policy_history(revision,actor_id,policy) VALUES($1,$2,$3)`, in.Revision, current(r).ID, jsonValue(in.systemStatusPolicy))
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		systemStatusError(w, e)
		return
	}
	s.audit(r, "SYSTEM_STATUS_POLICY_UPDATE", "system-status", map[string]any{"revision": in.Revision, "enabled": in.Enabled, "retention_days": in.Retention})
	jsonResponse(w, 200, map[string]any{"policy": in.systemStatusPolicy, "notice": systemStatusNotice})
}

func (s *Server) systemStatusContextTx(r *http.Request, tx pgx.Tx, id, wid string) (map[string]any, error) {
	out := map[string]any{"runbook": nil, "connector_records": []any{}, "notice": "실행 성공은 기대 시스템 버전의 배포 확인이 아닙니다. 커넥터 last_seen_at은 원격 자료를 수집한 시각이지 서버 상태 관측 시각이 아닙니다."}
	var executionID string
	e := tx.QueryRow(r.Context(), `SELECT e.id::text FROM runbook_executions e WHERE e.document_id=$1 AND EXISTS(SELECT 1 FROM runbook_settings WHERE id=1 AND enabled) ORDER BY e.created_at DESC,e.id DESC LIMIT 1`, id).Scan(&executionID)
	if e != nil && e != pgx.ErrNoRows {
		return nil, e
	}
	if executionID != "" {
		execution, e := s.runbookExecutionVisibleTx(r.Context(), tx, current(r), executionID)
		if e == nil {
			out["runbook"] = map[string]any{"id": executionID, "phase": execution["phase"], "status": execution["status"], "created_at": execution["created_at"], "completed_at": execution["completed_at"], "independently_verified": false}
		}
	}
	// Connector context has exactly the existing management visibility boundary;
	// document read alone does not reveal integration configuration to viewers.
	if systemStatusHuman(current(r)) {
		rows, e := runbookRowsTx(r.Context(), tx, `SELECT jsonb_build_object('connector_id',c.id,'kind',c.kind,'document_version',x.document_version,'last_seen_at',x.last_seen_at,'content_checksum',left(x.checksum,128),'enabled',c.enabled,'independently_verified',false)
 FROM connector_records x JOIN connector_configs c ON c.id=x.connector_id
 JOIN workspace_members m ON m.workspace_id=c.workspace_id AND m.user_id=$2 AND m.role IN ('owner','admin')
 JOIN users owner ON owner.id=c.owner_id AND NOT owner.disabled
 JOIN users service ON service.id=c.service_account_id AND NOT service.disabled
 WHERE x.document_id=$1 AND c.workspace_id=$3 AND (c.space_id IS NULL OR madi_space_allowed($2,c.space_id,true))
 AND madi_document_allowed(c.owner_id,$1,false) AND madi_document_allowed(c.service_account_id,$1,true)
 ORDER BY x.last_seen_at DESC,c.id LIMIT 20`, id, current(r).ID, wid)
		if e != nil {
			return nil, e
		}
		out["connector_records"] = rows
	}
	return out, nil
}

func (s *Server) expireSystemStatusReports(ctx context.Context) (int64, error) {
	tag, e := s.DB.Exec(ctx, `DELETE FROM system_status_reports WHERE id IN (SELECT r.id FROM system_status_reports r CROSS JOIN system_status_policy p WHERE p.id=1 AND r.received_at<clock_timestamp()-make_interval(days=>p.retention_days) ORDER BY r.received_at LIMIT 5000)`)
	return tag.RowsAffected(), e
}
func (s *Server) invalidateSystemStatusRestoreTx(ctx context.Context, tx pgx.Tx) error {
	_, e := tx.Exec(ctx, `UPDATE system_status_policy SET enabled=false,revision=revision+1,updated_at=clock_timestamp() WHERE id=1;
 UPDATE system_status_cards SET verification_epoch=verification_epoch+1,revision=revision+1,updated_at=clock_timestamp();
 INSERT INTO system_status_policy_history(revision,policy) SELECT revision,to_jsonb(p)-'id'-'updated_at' FROM system_status_policy p WHERE id=1 ON CONFLICT(revision) DO NOTHING;`)
	return e
}
