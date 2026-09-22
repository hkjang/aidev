package server

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func (s *Server) runbookEvent(ctx context.Context, id string, index int, kind, message string) error {
	if message == "" {
		return nil
	}
	message = strings.ToValidUTF8(message, "�")
	if len(message) > 65536 {
		message = strings.ToValidUTF8(message[:65536], "")
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	tag, e := tx.Exec(ctx, `UPDATE runbook_executions SET log_bytes=log_bytes+$2 WHERE id=$1 AND log_bytes+$2<=5242880`, id, len(message))
	if e != nil {
		return e
	}
	if tag.RowsAffected() == 0 {
		return nil
	}
	_, e = tx.Exec(ctx, `INSERT INTO runbook_events(execution_id,step_index,kind,message) VALUES($1,$2,$3,$4)`, id, index, kind, message)
	if e == nil {
		e = tx.Commit(ctx)
	}
	return e
}
func (s *Server) runbookFinish(id, status, message string, plan runbookPlan) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	if len(message) > 500 {
		message = strings.ToValidUTF8(message[:500], "")
	}
	var owner, doc string
	e = tx.QueryRow(ctx, `UPDATE runbook_executions SET status=$2,last_error=$3,updated_at=now(),completed_at=now() WHERE id=$1 AND (status IN ('queued','running') OR (status='unknown' AND $2='cancelled')) RETURNING owner_id::text,COALESCE(document_id::text,'')`, id, status, message).Scan(&owner, &doc)
	if errors.Is(e, pgx.ErrNoRows) { // Queue delivery can repeat after the terminal commit.
		return nil
	}
	if e != nil {
		return e
	}
	if status == "succeeded" && plan.Phase == "validate" {
		_, e = tx.Exec(ctx, `UPDATE runbook_documents SET last_tested_at=now(),last_tested_by=$2 WHERE document_id=$1 AND version=$3 AND EXISTS(SELECT 1 FROM documents d WHERE d.id=$1 AND d.version=$4 AND d.deleted_at IS NULL)`, plan.DocumentID, owner, plan.DefinitionVersion, plan.DocumentVersion)
		if e != nil {
			return e
		}
	}
	_, e = tx.Exec(ctx, `INSERT INTO notifications(id,user_id,title,document_id) VALUES($1,$2,$3,NULLIF($4,'')::uuid)`, newID(), owner, "격리 "+runbookPhaseName(plan.Phase)+" 작업 상태: "+map[string]string{"succeeded": "완료", "failed": "실패", "cancelled": "취소", "unknown": "외부 실행 확인 필요"}[status], doc)
	if e == nil {
		e = tx.Commit(ctx)
	}
	return e
}
func (s *Server) runbookCurrentForJob(ctx context.Context, j Job, id string) (runbookPlan, error) {
	var plan runbookPlan
	p, e := s.workerPrincipal(ctx, j.ActorID, j.TokenID, j.WorkspaceID)
	if e != nil {
		return plan, e
	}
	if p.Kind != "user" || p.TokenID != "" || p.ScopeRestricted || j.ActorID != j.OwnerID {
		return plan, jobPermanent("실행 작업의 원래 사용자가 유효하지 않습니다")
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return plan, e
	}
	defer tx.Rollback(ctx)
	var cancelled bool
	var jobID, owner, status, approvalID string
	var required bool
	e = tx.QueryRow(ctx, `SELECT cancel_requested,COALESCE(job_id::text,''),owner_id::text,status,approval_required,COALESCE(approval_id::text,'') FROM runbook_executions WHERE id=$1 FOR UPDATE`, id).Scan(&cancelled, &jobID, &owner, &status, &required, &approvalID)
	if e != nil {
		return plan, e
	}
	if cancelled {
		return plan, context.Canceled
	}
	if jobID != j.ID || owner != p.ID || !oneOf(status, "queued", "running") {
		return plan, jobPermanent("이미 종료되었거나 다른 실행 작업입니다")
	}
	plan, e = s.validateExecutionPlanTx(ctx, tx, p, id)
	if e != nil {
		return plan, e
	}
	if required {
		request, e := approvalReadRequestTx(ctx, tx, approvalID, false)
		if e != nil {
			return plan, e
		}
		if request.Status != "consumed" {
			return plan, jobPermanent("한 번 소비된 실행 승인이 필요합니다")
		}
		resource, e := s.runbookApprovalAdapter().Lock(ctx, tx, p, id, true)
		if e != nil {
			return plan, e
		}
		valid, reason, e := approvalCurrentTx(ctx, tx, request, resource)
		if e != nil {
			return plan, e
		}
		if !valid {
			return plan, jobPermanent(reason)
		}
	} else {
		var enabled bool
		if e = tx.QueryRow(ctx, `SELECT COALESCE((data->>'approval_enabled')::boolean,false) FROM settings WHERE id=1 FOR SHARE`).Scan(&enabled); e != nil {
			return plan, e
		}
		if enabled {
			return plan, jobPermanent("실행 승인 설정이 변경되었습니다")
		}
	}
	if _, e = tx.Exec(ctx, `UPDATE runbook_executions SET status='running',updated_at=now() WHERE id=$1`, id); e == nil {
		e = tx.Commit(ctx)
	}
	return plan, e
}
func (r *runbookRemote) cancelAndWait(ctx context.Context, external string, step runbookPlanStep) error {
	if external == "" {
		return fmt.Errorf("외부 실행 ID가 없어 중지를 확정할 수 없습니다")
	}
	initial := r.cancel(ctx, external)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		state, e := r.poll(ctx, external, step, false)
		if e == nil && oneOf(state.Status, "succeeded", "failed", "cancelled") {
			return nil
		}
		var response *runbookRemoteError
		if r.runner.Kind == "kubernetes" && errors.As(e, &response) && response.Status == 404 {
			return nil
		}
		if initial != nil && e != nil {
			return initial
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("외부 실행 중지의 완료를 확인하지 못했습니다")
		case <-ticker.C:
		}
	}
}
func (r *runbookRemote) existingKubernetesJob(ctx context.Context, id string, index int, step runbookPlanStep) (string, error) {
	job, e := r.json(ctx, "GET", "/apis/batch/v1/namespaces/"+url.PathEscape(str(r.runner.Config, "namespace"))+"/jobs/"+runbookJobName(id, index), nil)
	if e != nil {
		var response *runbookRemoteError
		if errors.As(e, &response) && response.Status == 404 {
			return "", nil
		}
		return "", e
	}
	if !r.verifyKubernetesJob(job, r.kubernetesJob(id, index, step)) {
		return "", fmt.Errorf("기존 Kubernetes 실행이 고정된 사양과 다릅니다")
	}
	uid := str(runbookObject(job, "metadata"), "uid")
	if !validID(uid) {
		return "", fmt.Errorf("기존 Kubernetes 실행 UID가 잘못되었습니다")
	}
	return runbookJobName(id, index) + "@" + uid, nil
}
func (s *Server) stopRunbookStep(id string, index int, step runbookPlanStep, remote *runbookRemote, external, reason string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	if external == "" && step.Kind == "kubernetes" {
		var e error
		external, e = remote.existingKubernetesJob(ctx, id, index, step)
		if e != nil {
			return "unknown", e
		}
		if external == "" {
			_, e = s.DB.Exec(ctx, `UPDATE runbook_execution_steps SET state='cancelled',completed_at=now(),last_error=$3 WHERE execution_id=$1 AND step_index=$2`, id, index, reason)
			return "cancelled", e
		}
	}
	status := "cancelled"
	e := remote.cancelAndWait(ctx, external, step)
	if e != nil {
		status = "unknown"
		reason += " · " + e.Error()
	}
	_, saveErr := s.DB.Exec(ctx, `UPDATE runbook_execution_steps SET state=$3,external_id=$4,completed_at=now(),last_error=$5 WHERE execution_id=$1 AND step_index=$2`, id, index, status, external, reason)
	_ = s.runbookEvent(ctx, id, index, "status", reason)
	if saveErr != nil {
		return status, saveErr
	}
	return status, e
}

func (s *Server) executeRunbookJob(ctx context.Context, j Job) (map[string]any, error) {
	id := str(j.Payload, "execution_id")
	if !validID(id) {
		return nil, jobPermanent("실행 계획 ID가 잘못되었습니다")
	}
	var previous string
	if e := s.DB.QueryRow(ctx, `SELECT status FROM runbook_executions WHERE id=$1 AND job_id=$2`, id, j.ID).Scan(&previous); e != nil {
		return nil, e
	}
	if previous == "succeeded" {
		return map[string]any{"execution_id": id, "status": previous}, nil
	}
	if oneOf(previous, "failed", "cancelled", "unknown") {
		return nil, jobPermanent("이미 종료되었거나 외부 확인이 필요한 실행입니다")
	}
	plan, e := s.runbookCurrentForJob(ctx, j, id)
	if e != nil { // No new launch on invalid credentials or a changed plan.
		_ = s.haltRunbookExecution(id, "원본·실행 권한 또는 작업 상태가 변경되어 실행을 중지합니다")
		return nil, jobPermanent("원본·실행 권한 또는 작업 상태가 변경되었습니다")
	}
	for index, step := range plan.Steps {
		runner, e := s.loadRunbookRunner(ctx, step.RunnerID, step.RunnerRevision)
		if e != nil {
			return nil, e
		}
		remote, e := s.newRunbookRemote(runner)
		if e != nil {
			return nil, e
		}
		state, err := s.executeRunbookStep(ctx, j, id, index, step, remote)
		remote.close()
		if err != nil {
			if oneOf(state, "failed", "cancelled", "unknown") {
				_, _ = s.DB.Exec(context.WithoutCancel(ctx), `UPDATE runbook_execution_steps SET state=$3,last_error=$4,completed_at=now() WHERE execution_id=$1 AND step_index=$2 AND state='queued'`, id, index, state, err.Error())
				_ = s.runbookFinish(id, state, err.Error(), plan)
				return nil, jobPermanent(err.Error())
			}
			return nil, err
		}
		if state != "succeeded" {
			_ = s.runbookFinish(id, "failed", "외부 단계가 완료되지 않았습니다", plan)
			return nil, jobPermanent("외부 단계가 완료되지 않았습니다")
		}
	}
	if _, e = s.runbookCurrentForJob(ctx, j, id); e != nil {
		_ = s.runbookFinish(id, "failed", "실행 종료 후 원본·권한 검증에 실패했습니다", plan)
		return nil, jobPermanent("실행 종료 후 원본·권한 검증에 실패했습니다")
	}
	if e = s.runbookFinish(id, "succeeded", "", plan); e != nil {
		return nil, e
	}
	return map[string]any{"execution_id": id, "status": "succeeded"}, nil
}
func (s *Server) executeRunbookStep(ctx context.Context, j Job, id string, index int, step runbookPlanStep, remote *runbookRemote) (string, error) {
	var state, external string
	var cursor int64
	var started *time.Time
	e := s.DB.QueryRow(ctx, `SELECT state,external_id,log_cursor,started_at FROM runbook_execution_steps WHERE execution_id=$1 AND step_index=$2`, id, index).Scan(&state, &external, &cursor, &started)
	if e != nil {
		return "", e
	}
	if state == "succeeded" {
		return state, nil
	}
	if oneOf(state, "failed", "cancelled", "unknown") {
		return state, jobPermanent("이미 종료되었거나 외부 확인이 필요한 실행 단계입니다")
	}
	if _, e = s.runbookCurrentForJob(ctx, j, id); e != nil {
		if state == "queued" {
			return "failed", jobPermanent("단계 시작 전 실행 권한이 변경되었습니다")
		}
		stopped, err := s.stopRunbookStep(id, index, step, remote, external, "현재 원본·실행 권한이 변경되어 중지합니다")
		if err != nil {
			return stopped, err
		}
		return stopped, jobPermanent("현재 원본·실행 권한이 변경되었습니다")
	}
	if state == "launching" && external == "" && step.Kind == "awx" {
		_ = s.runbookEvent(ctx, id, index, "status", "AWX launch 결과가 불확실합니다. 자동 재실행하지 않습니다.")
		_, e = s.DB.Exec(ctx, `UPDATE runbook_execution_steps SET state='unknown',last_error='AWX launch 결과 불확실' WHERE execution_id=$1 AND step_index=$2`, id, index)
		if e != nil {
			return "", e
		}
		return "unknown", jobPermanent("AWX 외부 실행 여부를 운영자가 확인해야 합니다")
	}
	if state == "queued" || state == "launching" && external == "" {
		report, e := remote.preflight(ctx, step.Action)
		if e != nil {
			return "failed", e
		}
		if len(step.Remote) == 0 || approvalSnapshotHash(report) != approvalSnapshotHash(step.Remote) {
			return "failed", jobPermanent("외부 실행기의 사전 검증 결과가 검토한 계획과 달라졌습니다")
		}
		if state == "queued" {
			tag, e := s.DB.Exec(ctx, `UPDATE runbook_execution_steps SET state='launching',started_at=now() WHERE execution_id=$1 AND step_index=$2 AND state='queued'`, id, index)
			if e != nil {
				return "", e
			}
			if tag.RowsAffected() != 1 {
				return "", fmt.Errorf("실행 단계가 다른 작업자에 의해 처리 중입니다")
			}
			now := time.Now()
			started = &now
		}
		external, e = remote.launch(ctx, id, index, step)
		if e != nil {
			if external != "" {
				stopped, err := s.stopRunbookStep(id, index, step, remote, external, e.Error())
				if err != nil {
					return stopped, err
				}
				return "failed", e
			}
			if step.Kind == "kubernetes" { // Deterministic name permits recovery, never a second distinct Job.
				return "", e
			}
			_, _ = s.DB.Exec(context.Background(), `UPDATE runbook_execution_steps SET state='unknown',last_error=$3 WHERE execution_id=$1 AND step_index=$2`, id, index, e.Error())
			return "unknown", e
		}
		tag, persistErr := s.DB.Exec(ctx, `UPDATE runbook_execution_steps SET state='running',external_id=$3 WHERE execution_id=$1 AND step_index=$2 AND state='launching'`, id, index, external)
		if persistErr != nil || tag.RowsAffected() != 1 {
			stopped, stopErr := s.stopRunbookStep(id, index, step, remote, external, "외부 실행 ID 저장을 확인할 수 없어 실행을 중지합니다")
			if stopErr != nil {
				return stopped, stopErr
			}
			return stopped, jobPermanent("외부 실행 ID를 안전하게 저장하지 못했습니다")
		}
		if e = s.runbookEvent(ctx, id, index, "status", step.Name+" · 격리 작업 시작 ("+external+")"); e != nil {
			return "", e
		}
	}
	if started == nil {
		return "unknown", jobPermanent("실행 단계 시작 시각을 확인할 수 없습니다")
	}
	deadline := started.Add(time.Duration(step.Action.Timeout+15) * time.Second)
	runCtx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	readOutput := cursor < 2<<20
	for {
		if _, e = s.runbookCurrentForJob(runCtx, j, id); e != nil {
			reason := "실행 취소·시간 제한 또는 현재 권한 변경으로 중지합니다"
			stopped, err := s.stopRunbookStep(id, index, step, remote, external, reason)
			if err != nil {
				return stopped, err
			}
			return stopped, jobPermanent(reason)
		}
		result, e := remote.poll(runCtx, external, step, readOutput)
		if e != nil {
			var unsafe *runbookSafetyError
			if errors.As(e, &unsafe) {
				stopped, err := s.stopRunbookStep(id, index, step, remote, external, unsafe.Error())
				if err != nil {
					return stopped, err
				}
				return stopped, jobPermanent(unsafe.Error())
			}
			if runCtx.Err() != nil {
				stopped, err := s.stopRunbookStep(id, index, step, remote, external, "실행이 취소되거나 제한 시간을 초과했습니다")
				if err != nil {
					return stopped, err
				}
				return stopped, jobPermanent("실행이 취소되거나 제한 시간을 초과했습니다")
			}
			return "", e
		}
		if readOutput && int64(len(result.Output)) > cursor {
			chunk := runbookCleanLog(result.Output[cursor:], remote.token)
			for len(chunk) > 0 {
				n := min(len(chunk), 60000)
				if e = s.runbookEvent(runCtx, id, index, "output", chunk[:n]); e != nil {
					return "", e
				}
				chunk = chunk[n:]
			}
			cursor = int64(len(result.Output))
			if _, e = s.DB.Exec(runCtx, `UPDATE runbook_execution_steps SET log_cursor=$3 WHERE execution_id=$1 AND step_index=$2`, id, index, cursor); e != nil {
				return "", e
			}
		}
		if result.Truncated {
			readOutput = false
			_ = s.runbookEvent(runCtx, id, index, "status", "이 단계 출력은 2MB까지만 저장합니다. 전체 출력은 격리 실행기에서 확인하세요.")
		}
		if oneOf(result.Status, "succeeded", "failed", "cancelled") {
			_, e = s.DB.Exec(runCtx, `UPDATE runbook_execution_steps SET state=$3,completed_at=now() WHERE execution_id=$1 AND step_index=$2`, id, index, result.Status)
			if e != nil {
				return "", e
			}
			_ = s.runbookEvent(runCtx, id, index, "status", step.Name+" · "+map[string]string{"succeeded": "완료", "failed": "실패", "cancelled": "취소 완료"}[result.Status])
			if result.Status != "succeeded" {
				return result.Status, jobPermanent("외부 실행 단계가 성공하지 않았습니다")
			}
			return result.Status, nil
		}
		select {
		case <-runCtx.Done():
		case <-ticker.C:
		}
	}
}
