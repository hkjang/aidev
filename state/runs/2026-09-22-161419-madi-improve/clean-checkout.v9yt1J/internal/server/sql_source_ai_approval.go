package server

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
)

//go:embed sql_source_ai.sql
var sqlSourceAISchema string

func (s *Server) migrateSQLSourceAI(ctx context.Context) error {
	_, e := s.DB.Exec(ctx, sqlSourceAISchema)
	return e
}
func (s *Server) registerSQLSourceApproval() {
	s.RegisterApprovalAdapter("sql_query_plan", ApprovalAdapter{Lock: s.lockSQLProposal, CanReview: s.canReviewSQLProposal, Submit: s.submitSQLProposal, Complete: s.completeSQLProposal})
}

func sqlSourceActorTx(ctx context.Context, tx pgx.Tx, uid, wid, space string) bool {
	var allowed bool
	e := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users u JOIN workspace_members m ON m.user_id=u.id WHERE u.id=$1 AND NOT u.disabled AND m.workspace_id=$2) AND madi_space_allowed($1,NULLIF($3,'')::uuid,false)`, uid, wid, space).Scan(&allowed)
	return e == nil && allowed
}
func (s *Server) lockSQLProposal(ctx context.Context, tx pgx.Tx, p *Principal, id string, write bool) (ApprovalResource, error) {
	scope := "database:read"
	if write {
		scope = "database:write"
	}
	if !hasIntegrationScope(p, scope) {
		return ApprovalResource{}, approvalProblem(403, "SQL 승인 대상의 API 권한이 없습니다")
	}
	var version int64
	var uid, cid, wid, space, name, status, serviceID, kind string
	var sourceRev, proposalRev int
	var enabled bool
	var configRaw, planRaw, columnsRaw []byte
	e := tx.QueryRow(ctx, `SELECT p.user_id::text,p.source_id::text,p.name,p.status,p.version,p.source_revision,p.plan,p.schema_snapshot,c.workspace_id::text,coalesce(c.space_id::text,''),c.service_account_id::text,c.enabled,c.revision,c.config,c.kind FROM sql_source_proposals p JOIN sql_sources c ON c.id=p.source_id WHERE p.id=$1 FOR UPDATE OF p FOR SHARE OF c`, id).Scan(&uid, &cid, &name, &status, &version, &proposalRev, &planRaw, &columnsRaw, &wid, &space, &serviceID, &enabled, &sourceRev, &configRaw, &kind)
	if e != nil {
		return ApprovalResource{}, approvalProblem(404, "SQL 계획 제안을 찾을 수 없습니다")
	}
	if !enabled || sourceRev != proposalRev || (p.WorkspaceID != "" && p.WorkspaceID != wid) || !hasIntegrationScope(p, "database:read") || !sqlSourceActorTx(ctx, tx, p.ID, wid, space) || !sqlSourceActorTx(ctx, tx, uid, wid, space) || !sqlSourceActorTx(ctx, tx, serviceID, wid, space) {
		return ApprovalResource{}, approvalProblem(403, "제안 이후 데이터 소스 또는 실행 권한이 변경되었습니다")
	}
	if write && p.ID != uid {
		return ApprovalResource{}, approvalProblem(403, "본인이 생성한 SQL 계획만 확인할 수 있습니다")
	}
	cfg := map[string]any{}
	if json.Unmarshal(configRaw, &cfg) != nil || !boolean(cfg, "allow_ai") {
		return ApprovalResource{}, approvalProblem(403, "데이터 소스 AI 제안 사용이 중지되었습니다")
	}
	var plan SQLSourcePlan
	if json.Unmarshal(planRaw, &plan) != nil {
		return ApprovalResource{}, approvalProblem(400, "SQL 계획 형식이 잘못되었습니다")
	}
	c := sqlSource{ID: cid, WorkspaceID: wid, SpaceID: space, Kind: kind, Config: cfg}
	if !sqlSourceTableAllowed(c, plan.Schema, plan.Table) {
		return ApprovalResource{}, approvalProblem(403, "허용 테이블에서 제외된 SQL 계획입니다")
	}
	var currentColumns []byte
	if e = tx.QueryRow(ctx, "SELECT columns FROM sql_source_tables WHERE source_id=$1 AND schema_name=$2 AND table_name=$3 AND columns=$4::jsonb FOR SHARE", cid, plan.Schema, plan.Table, columnsRaw).Scan(&currentColumns); e != nil {
		return ApprovalResource{}, approvalProblem(409, "테이블 메타데이터가 변경되었습니다. 새 계획을 생성하세요")
	}
	var cols []sqlColumn
	if json.Unmarshal(currentColumns, &cols) != nil {
		return ApprovalResource{}, errors.New("컬럼 메타데이터를 읽을 수 없습니다")
	}
	if _, _, e = buildSQLSourceQuery(kind, plan, cols, number(cfg, "max_rows", 500)); e != nil {
		return ApprovalResource{}, approvalProblem(400, e.Error())
	}
	var serviceKind bool
	if e = tx.QueryRow(ctx, "SELECT kind='service' AND NOT disabled FROM users WHERE id=$1", serviceID).Scan(&serviceKind); e != nil || !serviceKind {
		return ApprovalResource{}, approvalProblem(403, "SQL 연동 서비스 계정이 변경되었습니다")
	}
	snapshot := map[string]any{"proposal_id": id, "source_id": cid, "source_revision": sourceRev, "name": name, "plan": plan, "columns": cols}
	return ApprovalResource{Kind: "sql_query_plan", ID: id, WorkspaceID: wid, SpaceID: space, OwnerID: uid, Title: name, Version: version, Snapshot: snapshot}, nil
}
func (s *Server) canReviewSQLProposal(ctx context.Context, tx pgx.Tx, uid string, resource ApprovalResource) (bool, error) {
	var human bool
	e := tx.QueryRow(ctx, "SELECT kind='user' AND NOT disabled FROM users WHERE id=$1", uid).Scan(&human)
	if e != nil {
		return false, nil
	}
	return human && uid != resource.OwnerID && sqlSourceActorTx(ctx, tx, uid, resource.WorkspaceID, resource.SpaceID), nil
}
func (s *Server) submitSQLProposal(ctx context.Context, tx pgx.Tx, p *Principal, resource ApprovalResource) (ApprovalResource, error) {
	tag, e := tx.Exec(ctx, "UPDATE sql_source_proposals SET status='review',version=version+1,updated_at=now() WHERE id=$1 AND user_id=$2 AND status IN ('proposed','rejected','cancelled')", resource.ID, p.ID)
	if e != nil {
		return resource, e
	}
	if tag.RowsAffected() != 1 {
		return resource, approvalProblem(409, "이미 제출되었거나 활성화된 제안입니다")
	}
	resource.Version++
	return resource, nil
}
func (s *Server) completeSQLProposal(ctx context.Context, tx pgx.Tx, p *Principal, request ApprovalRequest, resource ApprovalResource) error {
	switch request.Status {
	case "approved":
		return activateSQLProposalTx(ctx, tx, resource)
	case "rejected", "cancelled":
		_, e := tx.Exec(ctx, "UPDATE sql_source_proposals SET status=$2,updated_at=now() WHERE id=$1 AND status='review'", resource.ID, request.Status)
		return e
	}
	return nil
}
func activateSQLProposalTx(ctx context.Context, tx pgx.Tx, resource ApprovalResource) error {
	var queryID string
	var status string
	e := tx.QueryRow(ctx, "SELECT coalesce(query_id::text,''),status FROM sql_source_proposals WHERE id=$1 FOR UPDATE", resource.ID).Scan(&queryID, &status)
	if e != nil {
		return e
	}
	if queryID != "" {
		return approvalProblem(409, "이미 저장된 쿼리입니다")
	}
	if !oneOf(status, "proposed", "rejected", "cancelled", "review") {
		return approvalProblem(409, "활성화할 수 없는 제안 상태입니다")
	}
	queryID = newID()
	_, e = tx.Exec(ctx, `INSERT INTO sql_source_queries(id,source_id,owner_id,name,plan,enabled,origin_proposal_id) VALUES($1,$2,$3,$4,$5,true,$6)`, queryID, str(resource.Snapshot, "source_id"), resource.OwnerID, resource.Title, jsonValue(resource.Snapshot["plan"]), resource.ID)
	if e != nil {
		return e
	}
	_, e = tx.Exec(ctx, "UPDATE sql_source_proposals SET status='approved',query_id=$2,updated_at=now() WHERE id=$1", resource.ID, queryID)
	return e
}
func sqlProposalError(w http.ResponseWriter, e error) {
	var approval *approvalFailure
	if errors.As(e, &approval) {
		apiError(w, approval.Status, approval.Message)
		return
	}
	respond(w, nil, e)
}
func (s *Server) confirmSQLProposal(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requestedSQLSource(w, r, false)
	if !ok {
		return
	}
	if current(r).TokenID != "" || current(r).ScopeRestricted || current(r).Kind != "user" {
		apiError(w, 403, "SQL 제안은 사용자 화면에서 확인해야 합니다")
		return
	}
	var in map[string]any
	if decode(r, &in) != nil || str(in, "confirmation") != "CREATE QUERY" {
		apiError(w, 400, "계획을 확인한 뒤 CREATE QUERY를 입력하세요")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	resource, e := s.lockSQLProposal(r.Context(), tx, current(r), r.PathValue("proposal"), true)
	if e != nil {
		sqlProposalError(w, e)
		return
	}
	if str(resource.Snapshot, "source_id") != c.ID || resource.Version != int64(number(in, "version", 0)) {
		apiError(w, 409, "확인 대상의 버전 또는 데이터 소스가 변경되었습니다")
		return
	}
	var approvalEnabled bool
	e = tx.QueryRow(r.Context(), "SELECT coalesce((data->>'approval_enabled')::boolean,false) FROM settings WHERE id=1 FOR SHARE").Scan(&approvalEnabled)
	if e != nil {
		respond(w, nil, e)
		return
	}
	result := map[string]any{"approval_required": approvalEnabled}
	if approvalEnabled {
		request, e := s.StartApprovalTx(r.Context(), tx, current(r), "sql_query_plan", resource.ID, "Text2SQL 계획을 확인했습니다")
		if e != nil {
			sqlProposalError(w, e)
			return
		}
		_, e = tx.Exec(r.Context(), "UPDATE sql_source_proposals SET approval_id=$2 WHERE id=$1", resource.ID, request.ID)
		if e != nil {
			respond(w, nil, e)
			return
		}
		result["approval_id"] = request.ID
		result["status"] = "review"
	} else {
		e = activateSQLProposalTx(r.Context(), tx, resource)
		if e != nil {
			sqlProposalError(w, e)
			return
		}
		result["status"] = "approved"
	}
	if e = tx.Commit(r.Context()); e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "SQL_PLAN_CONFIRM", resource.ID, map[string]any{"source_id": c.ID, "approval_required": approvalEnabled})
	respond(w, result, nil)
}
