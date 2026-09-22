package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
)

func (s *Server) submitRunbookApproval(w http.ResponseWriter, r *http.Request) {
	if !runbookHuman(r) {
		apiError(w, 403, "실제 요청자의 로그인 세션으로 검토를 제출하세요")
		return
	}
	var in struct {
		Version int64  `json:"version"`
		Comment string `json:"comment"`
	}
	if decode(r, &in) != nil || in.Version < 1 {
		apiError(w, 400, "실행 계획 버전을 확인하세요")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	resource, e := s.runbookApprovalAdapter().Lock(r.Context(), tx, current(r), r.PathValue("id"), true)
	if e != nil {
		approvalRespondError(w, e)
		return
	}
	if resource.Version != in.Version {
		apiError(w, 409, "실행 계획 버전이 변경되었습니다")
		return
	}
	var required bool
	if e = tx.QueryRow(r.Context(), `SELECT approval_required FROM runbook_executions WHERE id=$1`, resource.ID).Scan(&required); e != nil {
		respond(w, nil, e)
		return
	}
	if !required {
		apiError(w, 409, "승인 사용 설정에 맞는 새 실행 계획을 준비하세요")
		return
	}
	request, e := s.StartApprovalTx(r.Context(), tx, current(r), "runbook", resource.ID, in.Comment)
	if e == nil {
		_, e = tx.Exec(r.Context(), `UPDATE runbook_executions SET approval_id=$2 WHERE id=$1`, resource.ID, request.ID)
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		approvalRespondError(w, e)
		return
	}
	s.audit(r, "RUNBOOK_APPROVAL_SUBMIT", resource.ID, map[string]any{"request_id": request.ID})
	jsonResponse(w, 200, approvalSummary(request))
}
func (s *Server) executeRunbook(w http.ResponseWriter, r *http.Request) {
	if !runbookHuman(r) {
		apiError(w, 403, "실행은 실제 요청자의 로그인 화면에서 확인해야 합니다")
		return
	}
	var in struct {
		Version         int64  `json:"version"`
		Confirmation    string `json:"confirmation"`
		ApprovalVersion int64  `json:"approval_version"`
	}
	id := r.PathValue("id")
	if decode(r, &in) != nil || !validID(id) || in.Version < 1 || in.Confirmation != "EXECUTE "+id {
		apiError(w, 400, "정확한 계획을 확인하고 EXECUTE 실행-ID를 입력하세요")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	plan, e := s.validateExecutionPlanTx(r.Context(), tx, current(r), id)
	if e != nil {
		approvalRespondError(w, e)
		return
	}
	var required, enabled bool
	var version int64
	var status, approvalID, wid string
	e = tx.QueryRow(r.Context(), `SELECT approval_required,version,status,COALESCE(approval_id::text,''),workspace_id::text FROM runbook_executions WHERE id=$1 AND owner_id=$2 AND job_id IS NULL`, id, current(r).ID).Scan(&required, &version, &status, &approvalID, &wid)
	if e != nil {
		apiError(w, 409, "이미 사용되었거나 현재 요청자의 실행 계획이 아닙니다")
		return
	}
	if version != in.Version {
		apiError(w, 409, "검토한 실행 계획 버전이 변경되었습니다")
		return
	}
	if e = tx.QueryRow(r.Context(), `SELECT COALESCE((data->>'approval_enabled')::boolean,false) FROM settings WHERE id=1 FOR SHARE`).Scan(&enabled); e != nil {
		respond(w, nil, e)
		return
	}
	if enabled != required {
		apiError(w, 409, "승인 사용 설정이 변경되었습니다. 새 실행 계획을 준비하세요")
		return
	}
	if required {
		if status != "approved" {
			apiError(w, 409, "정확한 실행 계획의 승인 후 실행할 수 있습니다")
			return
		}
		if _, e = s.ConsumeApprovalTx(r.Context(), tx, current(r), "runbook", id, approvalID, in.ApprovalVersion); e != nil {
			approvalRespondError(w, e)
			return
		}
	} else if status != "prepared" {
		apiError(w, 409, "실행할 수 있는 준비 상태가 아닙니다")
		return
	}
	// Global advisory gate bounds queued external executions and prevents
	// concurrent confirmation from bypassing the per-user capacity limit.
	if _, e = tx.Exec(r.Context(), `SELECT pg_advisory_xact_lock(729421596)`); e != nil {
		respond(w, nil, e)
		return
	}
	var count int
	if e = tx.QueryRow(r.Context(), `SELECT count(*) FROM runbook_executions WHERE owner_id=$1 AND status IN ('queued','running')`, current(r).ID).Scan(&count); e != nil {
		respond(w, nil, e)
		return
	}
	if count >= 10 {
		apiError(w, 429, "사용자당 대기·실행 중인 절차는 10개 이하입니다")
		return
	}
	var uncertain bool
	if e = tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM runbook_executions WHERE document_id=$1 AND status='unknown')`, plan.DocumentID).Scan(&uncertain); e != nil {
		respond(w, nil, e)
		return
	}
	if uncertain {
		apiError(w, 409, "같은 문서에 외부 실행 확인이 필요한 작업이 있습니다. 관리자가 먼저 기존 실행 상태를 확인하세요")
		return
	}
	jobID, e := s.EnqueueJob(r.Context(), tx, "runbook.execute", current(r).ID, wid, map[string]any{"execution_id": id})
	if e != nil {
		respond(w, nil, e)
		return
	}
	timeout := 120
	for _, step := range plan.Steps {
		timeout += step.Action.Timeout + 30
	}
	if _, e = tx.Exec(r.Context(), `UPDATE automation_jobs SET max_attempts=3,timeout_seconds=$2,resource_id=$3 WHERE id=$1`, jobID, timeout, plan.DocumentID); e == nil {
		_, e = tx.Exec(r.Context(), `UPDATE runbook_executions SET status='queued',job_id=$2,updated_at=now() WHERE id=$1`, id, jobID)
	}
	if e == nil {
		_, e = tx.Exec(r.Context(), `INSERT INTO runbook_events(execution_id,kind,message) VALUES($1,'status','사용자가 정확한 실행 계획을 확인하여 작업을 등록했습니다')`, id)
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "RUNBOOK_EXECUTE_CONFIRM", id, map[string]any{"job_id": jobID, "approval_id": approvalID, "phase": plan.Phase})
	jsonResponse(w, 202, map[string]any{"id": id, "job_id": jobID, "status": "queued"})
}

func (s *Server) runbookExecutionVisibleTx(ctx context.Context, tx pgx.Tx, p *Principal, id string) (map[string]any, error) {
	if !validID(id) {
		return nil, pgx.ErrNoRows
	}
	var raw []byte
	e := tx.QueryRow(ctx, `SELECT to_jsonb(e) FROM runbook_executions e WHERE id=$1`, id).Scan(&raw)
	if e != nil {
		return nil, e
	}
	out := map[string]any{}
	if e = json.Unmarshal(raw, &out); e != nil {
		return nil, e
	}
	doc, e := runbookDocumentTx(ctx, tx, p, str(out, "document_id"), false)
	if e != nil {
		return nil, e
	}
	if _, e = runbookEnabledTx(ctx, tx); e != nil {
		return nil, e
	}
	// The immutable plan has no credentials and is reviewable under document
	// ACL. Runtime stdout is narrower: current execution permission is required.
	snapshot, _ := out["snapshot"].(map[string]any)
	plan, e := runbookPlanFromSnapshot(snapshot)
	if e != nil {
		return nil, e
	}
	canLogs := true
	for _, step := range plan.Steps {
		runner, e := runbookRunnerTx(ctx, tx, step.RunnerID, 0)
		if e != nil {
			return nil, e
		}
		allowed := false
		if runner.Enabled {
			for _, a := range runner.Actions {
				if a.ID == step.Action.ID {
					allowed, e = runbookActionAllowedTx(ctx, tx, p.ID, str(doc, "workspace_id"), a)
					if e != nil {
						return nil, e
					}
					break
				}
			}
		}
		canLogs = canLogs && allowed
	}
	out["can_read_logs"] = canLogs
	out["title"] = doc["title"]
	out["can_confirm"] = p.ID == str(out, "owner_id") && p.Kind == "user" && p.TokenID == "" && !p.ScopeRestricted && oneOf(str(out, "status"), "prepared", "approved")
	if !canLogs {
		delete(out, "last_error")
	}
	return out, nil
}
func runbookRowsTx(ctx context.Context, tx pgx.Tx, query string, args ...any) ([]any, error) {
	rows, e := tx.Query(ctx, query, args...)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []any{}
	for rows.Next() {
		var raw []byte
		if e = rows.Scan(&raw); e != nil {
			return nil, e
		}
		var value any
		if e = json.Unmarshal(raw, &value); e != nil {
			return nil, e
		}
		out = append(out, value)
	}
	return out, rows.Err()
}
func (s *Server) getRunbookExecution(w http.ResponseWriter, r *http.Request) {
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	out, e := s.runbookExecutionVisibleTx(r.Context(), tx, current(r), r.PathValue("id"))
	if e != nil {
		approvalRespondError(w, e)
		return
	}
	steps, e := runbookRowsTx(r.Context(), tx, `SELECT to_jsonb(s)-'last_error'-'log_cursor' FROM runbook_execution_steps s WHERE execution_id=$1 ORDER BY step_index`, r.PathValue("id"))
	if e != nil {
		respond(w, nil, e)
		return
	}
	out["steps"] = steps
	if approvalID := str(out, "approval_id"); approvalID != "" {
		request, e := approvalReadRequestTx(r.Context(), tx, approvalID, false)
		if e == nil {
			out["approval"] = approvalSummary(request)
		} else if !errors.Is(e, pgx.ErrNoRows) {
			respond(w, nil, e)
			return
		}
	}
	if jobID := str(out, "job_id"); jobID != "" {
		var status string
		if e = tx.QueryRow(r.Context(), `SELECT status FROM automation_jobs WHERE id=$1`, jobID).Scan(&status); e == nil {
			out["job_status"] = status
		}
	}
	if e = tx.Commit(r.Context()); e != nil {
		respond(w, nil, e)
		return
	}
	jsonResponse(w, 200, out)
}
func (s *Server) listRunbookExecutions(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	if _, e = runbookDocumentTx(r.Context(), tx, current(r), id, false); e != nil {
		respond(w, nil, e)
		return
	}
	if _, e = runbookEnabledTx(r.Context(), tx); e != nil {
		approvalRespondError(w, e)
		return
	}
	out, e := runbookRowsTx(r.Context(), tx, `SELECT jsonb_build_object('id',id,'phase',phase,'status',status,'version',version,'approval_required',approval_required,'approval_id',approval_id,'job_id',job_id,'owner_id',owner_id,'created_at',created_at,'completed_at',completed_at) FROM runbook_executions WHERE document_id=$1 ORDER BY created_at DESC LIMIT 100`, id)
	if e == nil {
		e = tx.Commit(r.Context())
	}
	respond(w, out, e)
}
func (s *Server) cancelRunbookExecution(w http.ResponseWriter, r *http.Request) {
	if !runbookHuman(r) || !validID(r.PathValue("id")) {
		apiError(w, 403, "실제 요청자 또는 서비스 관리자가 실행을 중지할 수 있습니다")
		return
	}
	var in struct {
		Confirmation string `json:"confirmation"`
	}
	if decode(r, &in) != nil || in.Confirmation != "CANCEL "+r.PathValue("id") {
		apiError(w, 400, "CANCEL 실행-ID를 입력하여 중지를 확인하세요")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	var owner, status, jobID string
	e = tx.QueryRow(r.Context(), `SELECT owner_id::text,status,COALESCE(job_id::text,'') FROM runbook_executions WHERE id=$1 FOR UPDATE`, r.PathValue("id")).Scan(&owner, &status, &jobID)
	if e != nil {
		respond(w, nil, e)
		return
	}
	if current(r).ID != owner && current(r).Role != "admin" {
		apiError(w, 403, "실제 요청자 또는 서비스 관리자만 중지할 수 있습니다")
		return
	}
	if !oneOf(status, "queued", "running", "unknown") {
		apiError(w, 409, "현재 중지를 요청할 수 있는 실행 상태가 아닙니다")
		return
	}
	_, e = tx.Exec(r.Context(), `UPDATE runbook_executions SET cancel_requested=true,updated_at=now() WHERE id=$1`, r.PathValue("id"))
	if e == nil && jobID != "" {
		_, e = tx.Exec(r.Context(), `UPDATE automation_jobs SET cancel_requested=true,updated_at=now() WHERE id=$1`, jobID)
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "RUNBOOK_CANCEL_REQUEST", r.PathValue("id"), nil)
	jsonResponse(w, 202, map[string]any{"cancel_requested": true, "message": "추적 중인 외부 작업의 중지를 요청했습니다. 외부 실행 ID가 없는 불확실 작업은 운영자가 확인해야 합니다."})
}

func (s *Server) streamRunbookEvents(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cursor, _ := strconv.ParseInt(r.URL.Query().Get("after"), 10, 64)
	if cursor < 0 {
		cursor = 0
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()
	write := func(kind string, value any) error {
		if _, e := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", kind, jsonValue(value)); e != nil {
			return e
		}
		return http.NewResponseController(w).Flush()
	}
	started := false
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		// Re-authenticate the original cookie/key too; a connection must not
		// continue displaying output after session expiry or account disable.
		p, e := s.runbookStreamPrincipal(ctx, r)
		if e != nil {
			if started {
				_ = write("error", map[string]any{"error": "인증 또는 실행 출력 권한이 변경되었습니다"})
			} else {
				apiError(w, 401, "인증이 만료되었습니다")
			}
			return
		}
		tx, e := s.DB.Begin(ctx)
		if e != nil {
			return
		}
		out, e := s.runbookExecutionVisibleTx(ctx, tx, p, id)
		if e != nil || !boolean(out, "can_read_logs") {
			tx.Rollback(ctx)
			if started {
				_ = write("error", map[string]any{"error": "현재 실행 출력 권한이 없습니다"})
			} else {
				apiError(w, 403, "현재 실행 출력 권한이 없습니다")
			}
			return
		}
		events, e := runbookRowsTx(ctx, tx, `SELECT to_jsonb(e) FROM runbook_events e WHERE execution_id=$1 AND id>$2 ORDER BY id LIMIT 100`, id, cursor)
		tx.Rollback(ctx)
		if e != nil {
			return
		}
		if !started {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("X-Accel-Buffering", "no")
			started = true
		}
		for _, value := range events {
			event, _ := value.(map[string]any)
			cursor = int64(number(event, "id", 0))
			if write("event", event) != nil {
				return
			}
		}
		if write("state", map[string]any{"status": out["status"], "after": cursor, "cancel_requested": out["cancel_requested"]}) != nil {
			return
		}
		if oneOf(str(out, "status"), "succeeded", "failed", "cancelled", "unknown") && len(events) < 100 {
			_ = write("done", map[string]any{"status": out["status"]})
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Server) runbookStreamPrincipal(ctx context.Context, r *http.Request) (*Principal, error) {
	// Runtime logs intentionally require a human cookie, not long-lived keys or
	// plugin credentials. The current role and account status are reloaded.
	if !runbookHuman(r) {
		return nil, errors.New("로그인 세션이 필요합니다")
	}
	cookie, e := r.Cookie("madi_session")
	if e != nil {
		return nil, e
	}
	p := &Principal{}
	e = s.DB.QueryRow(ctx, `SELECT u.id::text,u.email,u.name,u.role,u.kind FROM sessions se JOIN users u ON u.id=se.user_id WHERE se.token_hash=$1 AND se.expires_at>now() AND NOT u.disabled AND u.kind='user'`, digest(cookie.Value)).Scan(&p.ID, &p.Email, &p.Name, &p.Role, &p.Kind)
	if e == nil && p.ID != current(r).ID {
		return nil, errors.New("로그인 사용자가 변경되었습니다")
	}
	return p, e
}
