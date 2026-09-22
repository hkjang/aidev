package server

import (
	"net/http"
	"strings"
)

// Resolution is an explicit operator attestation, NOT automatic evidence of a
// successful remote run. It can only close an unknown execution as cancelled;
// it cannot mark it successful, resume steps, consume a new approval or relaunch.
func (s *Server) adminRunbookUncertain(w http.ResponseWriter, r *http.Request) {
	out, e := s.rows(r.Context(), `SELECT jsonb_build_object('id',e.id,'workspace_id',e.workspace_id,'created_at',e.created_at,'phase',e.phase,'status',e.status,'steps',(SELECT COALESCE(jsonb_agg(jsonb_build_object('step_index',x.step_index,'runner_id',x.runner_id,'runner_revision',x.runner_revision,'external_id',x.external_id,'state',x.state) ORDER BY x.step_index),'[]'::jsonb) FROM runbook_execution_steps x WHERE x.execution_id=e.id)) FROM runbook_executions e WHERE e.status='unknown' ORDER BY e.created_at LIMIT 100`)
	respond(w, out, e)
}
func (s *Server) adminRunbookResolve(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var in struct {
		Confirmation    string `json:"confirmation"`
		Reason          string `json:"reason"`
		ExternalChecked bool   `json:"external_checked"`
	}
	if decode(r, &in) != nil || !validID(id) || in.Confirmation != "RESOLVE "+id || !in.ExternalChecked || len(strings.TrimSpace(in.Reason)) < 20 || len(in.Reason) > 2000 {
		apiError(w, 400, "외부 실행기에서 미실행 또는 중지를 직접 확인한 근거(20~2,000자)와 RESOLVE 실행-ID가 필요합니다")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	var status string
	if e = tx.QueryRow(r.Context(), `SELECT status FROM runbook_executions WHERE id=$1 FOR UPDATE`, id).Scan(&status); e != nil {
		respond(w, nil, e)
		return
	}
	if status != "unknown" {
		apiError(w, 409, "외부 실행 확인 필요 상태만 수동으로 종결할 수 있습니다")
		return
	}
	_, e = tx.Exec(r.Context(), `UPDATE runbook_executions SET status='cancelled',cancel_requested=true,last_error='운영자가 외부 미실행·중지를 직접 확인하여 수동 종결했습니다. 자동 검증 또는 실행 성공 기록이 아닙니다.',updated_at=now(),completed_at=now() WHERE id=$1`, id)
	if e == nil {
		_, e = tx.Exec(r.Context(), `UPDATE runbook_execution_steps SET state='cancelled',completed_at=now(),last_error='운영자 수동 종결: 자동 중지 검증이 아닙니다' WHERE execution_id=$1 AND state IN ('queued','launching','running','unknown')`, id)
	}
	if e == nil {
		_, e = tx.Exec(r.Context(), `INSERT INTO runbook_events(execution_id,kind,message) VALUES($1,'operator_resolution',$2)`, id, "운영자 수동 확인: "+in.Reason)
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "RUNBOOK_MANUAL_EXTERNAL_RESOLUTION", id, map[string]any{"reason": in.Reason, "not_automatically_verified": true})
	jsonResponse(w, 200, map[string]any{"status": "cancelled", "manually_resolved": true})
}
