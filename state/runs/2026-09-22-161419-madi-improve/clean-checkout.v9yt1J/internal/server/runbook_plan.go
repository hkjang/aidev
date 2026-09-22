package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func runbookEnabledTx(ctx context.Context, tx pgx.Tx) (int64, error) {
	var enabled bool
	var revision int64
	e := tx.QueryRow(ctx, `SELECT enabled,revision FROM runbook_settings WHERE id=1 FOR SHARE`).Scan(&enabled, &revision)
	if e != nil {
		return 0, e
	}
	if !enabled {
		return 0, approvalProblem(404, "격리 실행 기능이 비활성화되어 있습니다")
	}
	return revision, nil
}
func runbookDocumentTx(ctx context.Context, tx pgx.Tx, p *Principal, id string, write bool) (map[string]any, error) {
	var raw []byte
	if !validID(id) {
		return nil, pgx.ErrNoRows
	}
	e := tx.QueryRow(ctx, `SELECT to_jsonb(d)-'search_vector' FROM documents d WHERE id=$1 AND deleted_at IS NULL AND madi_document_allowed($2,id,$3) AND ($4='' OR workspace_id::text=$4) FOR SHARE`, id, p.ID, write, p.WorkspaceID).Scan(&raw)
	if e != nil {
		return nil, e
	}
	out := map[string]any{}
	e = json.Unmarshal(raw, &out)
	return out, e
}
func runbookDefinitionTx(ctx context.Context, tx pgx.Tx, id string) (runbookDefinition, error) {
	var d runbookDefinition
	var steps, validation, rollback []byte
	e := tx.QueryRow(ctx, `SELECT document_id::text,version,purpose,prerequisites,validation,rollback,steps,validation_steps,rollback_steps FROM runbook_documents WHERE document_id=$1 FOR SHARE`, id).Scan(&d.DocumentID, &d.Version, &d.Purpose, &d.Prerequisites, &d.Validation, &d.Rollback, &steps, &validation, &rollback)
	if e != nil {
		return d, e
	}
	for _, pair := range []struct {
		raw  []byte
		dest any
	}{{steps, &d.Steps}, {validation, &d.ValidationSteps}, {rollback, &d.RollbackSteps}} {
		if e = json.Unmarshal(pair.raw, pair.dest); e != nil {
			return d, e
		}
	}
	return d, nil
}
func runbookPhaseSteps(d runbookDefinition, phase string) []runbookStep {
	switch phase {
	case "execute":
		return d.Steps
	case "validate":
		return d.ValidationSteps
	case "rollback":
		return d.RollbackSteps
	}
	return nil
}
func (s *Server) buildRunbookPlanTx(ctx context.Context, tx pgx.Tx, p *Principal, documentID, phase string) (runbookPlan, error) {
	var plan runbookPlan
	if !oneOf(phase, "execute", "validate", "rollback") {
		return plan, approvalProblem(400, "실행·검증·롤백 유형을 확인하세요")
	}
	doc, e := runbookDocumentTx(ctx, tx, p, documentID, true)
	if e != nil {
		return plan, e
	}
	d, e := runbookDefinitionTx(ctx, tx, documentID)
	if e != nil {
		return plan, e
	}
	steps := runbookPhaseSteps(d, phase)
	if len(steps) < 1 || len(steps) > 20 {
		return plan, approvalProblem(400, "해당 유형에 1~20개 허용 실행 단계를 설정하세요")
	}
	plan = runbookPlan{DocumentID: documentID, DocumentVersion: int64(number(doc, "version", 0)), DocumentHash: approvalSnapshotHash(approvalDocumentSnapshot(doc)), DefinitionVersion: d.Version, DefinitionHash: approvalSnapshotHash(map[string]any{"definition": d}), Phase: phase, Title: str(doc, "title"), Steps: []runbookPlanStep{}}
	timeout := 120
	for _, step := range steps {
		runner, e := runbookRunnerTx(ctx, tx, step.RunnerID, 0)
		if e != nil {
			return plan, e
		}
		if e = validateRunbookRunner(&runner); e != nil {
			return plan, e
		}
		if !runner.Enabled || runner.WorkspaceID != str(doc, "workspace_id") {
			return plan, approvalProblem(403, "실행기가 중지되었거나 워크스페이스 범위를 벗어났습니다")
		}
		var action *runbookAction
		for i := range runner.Actions {
			if runner.Actions[i].ID == step.ActionID {
				action = &runner.Actions[i]
				break
			}
		}
		if action == nil {
			return plan, approvalProblem(400, "허용 목록에서 실행 작업을 찾을 수 없습니다")
		}
		allowed, e := runbookActionAllowedTx(ctx, tx, p.ID, runner.WorkspaceID, *action)
		if e != nil {
			return plan, e
		}
		if !allowed {
			return plan, approvalProblem(403, "현재 사용자에게 명시적으로 허용된 실행 작업이 아닙니다")
		}
		parameters, argv, e := runbookParameters(*action, step.Parameters)
		if e != nil {
			return plan, e
		}
		timeout += action.Timeout + 30
		plan.Steps = append(plan.Steps, runbookPlanStep{Name: step.Name, RunnerID: runner.ID, RunnerRevision: runner.Revision, Kind: runner.Kind, Action: *action, Parameters: parameters, Argv: argv})
	}
	if timeout > 3600 {
		return plan, approvalProblem(400, "전체 실행 제한 시간은 시작 여유 시간을 포함해 1시간 이하입니다. 단계를 나누세요")
	}
	plan.SettingsRevision, e = runbookEnabledTx(ctx, tx)
	return plan, e
}

func planMap(plan runbookPlan) map[string]any {
	var out map[string]any
	_ = json.Unmarshal(jsonValue(plan), &out)
	return out
}
func runbookLocalPlanHash(plan runbookPlan) string {
	plan.Steps = append([]runbookPlanStep{}, plan.Steps...)
	for i := range plan.Steps {
		plan.Steps[i].Remote = nil
	}
	return approvalSnapshotHash(planMap(plan))
}
func runbookPlanFromSnapshot(snapshot map[string]any) (runbookPlan, error) {
	var p runbookPlan
	e := json.Unmarshal(jsonValue(snapshot), &p)
	if e == nil && (len(p.Steps) == 0 || len(p.Steps) > 20 || !validID(p.DocumentID)) {
		e = approvalProblem(409, "실행 계획 snapshot 형식이 잘못되었습니다")
	}
	return p, e
}

func (s *Server) runbookApprovalAdapter() ApprovalAdapter {
	return ApprovalAdapter{
		Lock: func(ctx context.Context, tx pgx.Tx, p *Principal, id string, write bool) (ApprovalResource, error) {
			var out ApprovalResource
			var raw []byte
			var owner string
			if !validID(id) {
				return out, pgx.ErrNoRows
			}
			if !hasIntegrationScope(p, "document:read") {
				return out, approvalProblem(403, "실행 계획 조회 권한이 없습니다")
			}
			e := tx.QueryRow(ctx, `SELECT workspace_id::text,COALESCE(document_id::text,''),owner_id::text,version,snapshot FROM runbook_executions WHERE id=$1 FOR UPDATE`, id).Scan(&out.WorkspaceID, &out.DocumentID, &owner, &out.Version, &raw)
			if e != nil {
				return out, e
			}
			if e = json.Unmarshal(raw, &out.Snapshot); e != nil {
				return out, e
			}
			plan, e := runbookPlanFromSnapshot(out.Snapshot)
			if e != nil {
				return out, e
			}
			doc, e := runbookDocumentTx(ctx, tx, p, plan.DocumentID, write)
			if e != nil {
				return out, e
			}
			if write && (p.ID != owner || p.Kind != "user" || p.TokenID != "" || p.ScopeRestricted) {
				return out, approvalProblem(403, "실제 계획 요청자만 실행 검토를 제출할 수 있습니다")
			}
			out.Kind = "runbook"
			out.ID = id
			out.OwnerID = str(doc, "owner_id")
			out.SpaceID = str(doc, "space_id")
			out.Title = plan.Title
			// Historical reads remain available under current document ACL, but
			// the synthesized current snapshot differs when launch authority drifts.
			actor := &Principal{ID: owner, Kind: "user"}
			current, e := s.buildRunbookPlanTx(ctx, tx, actor, plan.DocumentID, plan.Phase)
			if e != nil {
				var failure *approvalFailure
				if errors.As(e, &failure) || errors.Is(e, pgx.ErrNoRows) {
					out.Snapshot["current_invalid"] = "실행 원본 또는 권한·실행기 설정이 변경되었습니다"
					return out, nil
				}
				return out, e
			}
			if runbookLocalPlanHash(current) != runbookLocalPlanHash(plan) {
				out.Snapshot["current_invalid"] = "실행 원본 또는 권한·실행기 설정이 변경되었습니다"
			}
			return out, nil
		},
		CanReview: func(ctx context.Context, tx pgx.Tx, uid string, resource ApprovalResource) (bool, error) {
			var allowed bool
			e := tx.QueryRow(ctx, `SELECT madi_document_allowed($1,$2,false)`, uid, resource.DocumentID).Scan(&allowed)
			return allowed, e
		},
		Submit: func(ctx context.Context, tx pgx.Tx, p *Principal, resource ApprovalResource) (ApprovalResource, error) {
			if str(resource.Snapshot, "current_invalid") != "" {
				return resource, approvalProblem(409, "새로운 실행 계획을 만들어 검토하세요")
			}
			tag, e := tx.Exec(ctx, `UPDATE runbook_executions SET status='review',version=version+1,updated_at=now() WHERE id=$1 AND owner_id=$2 AND status IN ('prepared','rejected','cancelled') AND job_id IS NULL`, resource.ID, p.ID)
			if e != nil {
				return resource, e
			}
			if tag.RowsAffected() != 1 {
				return resource, approvalProblem(409, "이미 제출되거나 실행된 계획입니다")
			}
			resource.Version++
			return resource, nil
		},
		Complete: func(ctx context.Context, tx pgx.Tx, p *Principal, r ApprovalRequest, resource ApprovalResource) error {
			_, e := tx.Exec(ctx, `UPDATE runbook_executions SET status=$2,updated_at=now() WHERE id=$1 AND status='review' AND job_id IS NULL`, resource.ID, r.Status)
			return e
		},
	}
}

func (s *Server) validateExecutionPlanTx(ctx context.Context, tx pgx.Tx, p *Principal, id string) (runbookPlan, error) {
	var plan runbookPlan
	resource, e := s.runbookApprovalAdapter().Lock(ctx, tx, p, id, true)
	if e != nil {
		return plan, e
	}
	if str(resource.Snapshot, "current_invalid") != "" {
		return plan, approvalProblem(409, "원본·허용 명령·실행 권한이 변경되었습니다. 새 실행 계획이 필요합니다")
	}
	return runbookPlanFromSnapshot(resource.Snapshot)
}

func validateRunbookDefinition(d *runbookDefinition) error {
	if !validID(d.DocumentID) || d.Version < 0 {
		return approvalProblem(400, "문서 ID와 운영 절차 버전을 확인하세요")
	}
	for _, value := range []string{d.Purpose, d.Prerequisites, d.Validation, d.Rollback} {
		if len(value) > 30000 {
			return approvalProblem(400, "운영 절차 설명은 항목당 30,000자 이하입니다")
		}
	}
	for _, list := range [][]runbookStep{d.Steps, d.ValidationSteps, d.RollbackSteps} {
		if len(list) > 20 {
			return approvalProblem(400, "유형별 단계는 20개 이하입니다")
		}
		for i, step := range list {
			if !validID(step.RunnerID) || !runbookName.MatchString(step.ActionID) || len(step.Name) > 200 || len(jsonValue(step.Parameters)) > 30000 {
				return approvalProblem(400, fmt.Sprintf("%d단계 실행기·작업·매개변수를 확인하세요", i+1))
			}
		}
	}
	return nil
}
