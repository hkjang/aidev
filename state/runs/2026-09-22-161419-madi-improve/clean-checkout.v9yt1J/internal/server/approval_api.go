package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func approvalRespondError(w http.ResponseWriter, e error) {
	var failure *approvalFailure
	if errors.As(e, &failure) {
		apiError(w, failure.Status, failure.Message)
		return
	}
	var pg *pgconn.PgError
	if errors.As(e, &pg) && pg.Code == "23505" {
		apiError(w, 409, "같은 대상의 승인 정책 또는 요청이 이미 있습니다")
		return
	}
	respond(w, nil, e)
}

func (s *Server) registerApproval() {
	s.RegisterApprovalAdapter("document", s.documentApprovalAdapter())
	s.handle("GET /api/v1/documents/{id}/approval", s.approvalDocumentStatus)
	s.handle("GET /api/v1/approvals/resources/{kind}/{id}", s.approvalResourceStatus)
	s.handle("GET /api/v1/approvals/requests/{id}", s.approvalRequestDetail)
	s.handle("POST /api/v1/approvals/requests/{id}/decisions", s.approvalDecision)
	s.handle("GET /api/v1/approvals/inbox", s.approvalInbox)
	s.admin("GET /api/v1/admin/approval/policies", func(w http.ResponseWriter, r *http.Request) {
		out, e := s.rows(r.Context(), `SELECT to_jsonb(p) FROM approval_policies p ORDER BY p.workspace_id,p.name`)
		respond(w, out, e)
	})
	s.admin("POST /api/v1/admin/approval/policies", s.approvalSavePolicy)
	s.admin("PUT /api/v1/admin/approval/policies/{id}", s.approvalSavePolicy)
	s.admin("DELETE /api/v1/admin/approval/policies/{id}", s.approvalDisablePolicy)
}

func (s *Server) documentApprovalAdapter() ApprovalAdapter {
	lock := func(ctx context.Context, tx pgx.Tx, p *Principal, id string, write bool) (ApprovalResource, error) {
		var raw []byte
		var out ApprovalResource
		if !validID(id) {
			return out, pgx.ErrNoRows
		}
		scope := "document:read"
		if write {
			scope = "document:write"
		}
		if !hasIntegrationScope(p, scope) {
			return out, approvalProblem(403, "문서 승인 대상의 API 권한이 없습니다")
		}
		if e := tx.QueryRow(ctx, `SELECT to_jsonb(d)-'search_vector' FROM documents d WHERE d.id=$1 AND d.deleted_at IS NULL AND ($3='' OR d.workspace_id::text=$3) AND madi_document_allowed($2,d.id,$4) FOR UPDATE`, id, p.ID, p.WorkspaceID, write).Scan(&raw); e != nil {
			return out, e
		}
		doc := map[string]any{}
		if e := json.Unmarshal(raw, &doc); e != nil {
			return out, e
		}
		out = ApprovalResource{Kind: "document", ID: id, WorkspaceID: str(doc, "workspace_id"), SpaceID: str(doc, "space_id"), OwnerID: str(doc, "owner_id"), DocumentID: id, Title: str(doc, "title"), Version: int64(number(doc, "version", 0)), Snapshot: approvalDocumentSnapshot(doc)}
		return out, nil
	}
	return ApprovalAdapter{
		Lock: lock,
		CanReview: func(ctx context.Context, tx pgx.Tx, uid string, resource ApprovalResource) (bool, error) {
			var allowed bool
			e := tx.QueryRow(ctx, `SELECT madi_document_allowed($1,$2,false)`, uid, resource.ID).Scan(&allowed)
			return allowed, e
		},
		Submit: func(ctx context.Context, tx pgx.Tx, p *Principal, resource ApprovalResource) (ApprovalResource, error) {
			var status string
			if e := tx.QueryRow(ctx, `SELECT status FROM documents WHERE id=$1`, resource.ID).Scan(&status); e != nil {
				return resource, e
			}
			if status == "archived" {
				return resource, approvalProblem(409, "보관된 문서는 검토할 수 없습니다")
			}
			if status == "review" {
				var id string
				e := tx.QueryRow(ctx, `SELECT id::text FROM approval_requests WHERE resource_kind='document' AND resource_id=$1 AND status='pending'`, resource.ID).Scan(&id)
				if e == nil {
					old, e := approvalReadRequestTx(ctx, tx, id, false)
					if e != nil {
						return resource, e
					}
					valid, _, e := approvalCurrentTx(ctx, tx, old, resource)
					if e != nil {
						return resource, e
					}
					if valid {
						return resource, approvalProblem(409, "이미 같은 버전의 검토 요청이 진행 중입니다")
					}
				} else if !errors.Is(e, pgx.ErrNoRows) {
					return resource, e
				}
			}
			if e := s.approvalDocumentStateTx(ctx, tx, p, resource.ID, "review"); e != nil {
				return resource, e
			}
			resource.Version++
			return resource, nil
		},
		Complete: func(ctx context.Context, tx pgx.Tx, p *Principal, request ApprovalRequest, resource ApprovalResource) error {
			status := "draft"
			if request.Status == "approved" {
				status = "published"
			} else if request.Status == "rejected" {
				status = "rejected"
			}
			return s.approvalDocumentStateTx(ctx, tx, p, resource.ID, status)
		},
	}
}

func (s *Server) approvalDocumentStateTx(ctx context.Context, tx pgx.Tx, p *Principal, id, status string) error {
	var raw []byte
	if e := tx.QueryRow(ctx, `SELECT to_jsonb(d)-'search_vector' FROM documents d WHERE d.id=$1 AND d.deleted_at IS NULL`, id).Scan(&raw); e != nil {
		return e
	}
	before := map[string]any{}
	if e := json.Unmarshal(raw, &before); e != nil {
		return e
	}
	if _, e := tx.Exec(ctx, `UPDATE documents SET status=$2,version=version+1,updated_at=now() WHERE id=$1`, id, status); e != nil {
		return e
	}
	if _, e := tx.Exec(ctx, `INSERT INTO document_versions(document_id,version,title,markdown,tags,user_id,block_metadata) SELECT id,version,title,markdown,tags,$2,block_metadata FROM documents WHERE id=$1`, id, p.ID); e != nil {
		return e
	}
	after := map[string]any{"id": id, "title": before["title"], "status": status, "version": number(before, "version", 0) + 1, "tags": before["tags"]}
	if e := s.enqueueEvent(ctx, tx, Event{Type: "document.updated", WorkspaceID: str(before, "workspace_id"), ResourceID: id, Before: before, After: after, ActorID: p.ID}); e != nil {
		return e
	}
	if str(before, "status") != status {
		return s.enqueueEvent(ctx, tx, Event{Type: "document.status_changed", WorkspaceID: str(before, "workspace_id"), ResourceID: id, Before: before, After: after, ActorID: p.ID})
	}
	return nil
}

type approvalCommand struct {
	Action         string `json:"action"`
	Comment        string `json:"comment"`
	RequestID      string `json:"request_id"`
	RequestVersion int64  `json:"request_version"`
	GateIndex      *int   `json:"gate_index"`
}

func (s *Server) approvalAdvanced(w http.ResponseWriter, r *http.Request) {
	var in approvalCommand
	if decode(r, &in) != nil {
		apiError(w, 400, "검토 요청 형식을 확인하세요")
		return
	}
	if in.Action != "submit" && (current(r).TokenID != "" || current(r).ScopeRestricted || current(r).Kind != "user") {
		apiError(w, 403, "승인·반려는 실제 사용자의 로그인 세션에서만 처리할 수 있습니다")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	var request ApprovalRequest
	if in.Action == "submit" {
		if !hasIntegrationScope(current(r), "document:write") {
			apiError(w, 403, "문서 작성 권한이 필요합니다")
			return
		}
		request, e = s.StartApprovalTx(r.Context(), tx, current(r), "document", r.PathValue("id"), in.Comment)
	} else if oneOf(in.Action, "approve", "reject", "cancel") {
		resource, err := s.approvalAdapters["document"].Lock(r.Context(), tx, current(r), r.PathValue("id"), false)
		e = err
		if e == nil {
			if in.Action == "cancel" {
				request, e = s.cancelApprovalTx(r.Context(), tx, current(r), resource, in.RequestID, in.RequestVersion)
			} else {
				request, e = s.decideApprovalTx(r.Context(), tx, current(r), resource, in.RequestID, in.RequestVersion, in.GateIndex, in.Action == "reject", in.Comment)
			}
		}
	} else {
		e = approvalProblem(400, "submit/approve/reject/cancel 작업을 선택하세요")
	}
	if e == nil {
		e = s.approvalActorTx(r, tx, request.WorkspaceID, "document")
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		approvalRespondError(w, e)
		return
	}
	s.audit(r, "APPROVAL_"+strings.ToUpper(in.Action), request.ID, map[string]any{"document_id": r.PathValue("id"), "request_version": request.Version, "status": request.Status})
	out, e := s.document(r, r.PathValue("id"))
	respond(w, out, e)
}

func (s *Server) cancelApprovalTx(ctx context.Context, tx pgx.Tx, p *Principal, resource ApprovalResource, id string, expected int64) (ApprovalRequest, error) {
	var out ApprovalRequest
	if !validID(id) || expected < 1 {
		return out, approvalProblem(400, "취소할 요청 ID와 버전이 필요합니다")
	}
	if _, _, e := approvalConfigurationTx(ctx, tx); e != nil {
		return out, e
	}
	out, e := approvalReadRequestTx(ctx, tx, id, true)
	if e != nil {
		return out, e
	}
	if out.ResourceKind != resource.Kind || out.ResourceID != resource.ID || out.Status != "pending" || out.Version != expected {
		return out, approvalProblem(409, "현재 취소할 수 없는 검토 요청입니다")
	}
	if p.ID != out.RequesterID && p.ID != out.OwnerID {
		return out, approvalProblem(403, "요청자나 소유자만 검토를 취소할 수 있습니다")
	}
	out.Status = "cancelled"
	out.Version++
	if _, e = tx.Exec(ctx, `UPDATE approval_requests SET status='cancelled',version=version+1,updated_at=now(),completed_at=now() WHERE id=$1`, id); e != nil {
		return out, e
	}
	if adapter := s.approvalAdapters[resource.Kind]; adapter.Complete != nil {
		e = adapter.Complete(ctx, tx, p, out, resource)
	}
	return out, e
}

func approvalSummary(request ApprovalRequest) map[string]any {
	return map[string]any{"id": request.ID, "resource_kind": request.ResourceKind, "resource_id": request.ResourceID, "resource_version": request.ResourceVersion, "requester_id": request.RequesterID, "owner_id": request.OwnerID, "policy": request.Policy, "status": request.Status, "current_stage": request.CurrentStage, "version": request.Version, "comment": request.Comment}
}

func (s *Server) approvalStatusTx(ctx context.Context, tx pgx.Tx, p *Principal, resource ApprovalResource, includeSnapshot bool) (map[string]any, error) {
	cfg, _, e := approvalConfigurationTx(ctx, tx)
	if e != nil {
		return nil, e
	}
	policy, e := approvalPolicyTx(ctx, tx, cfg, resource.WorkspaceID, resource.SpaceID, resource.Kind)
	if e != nil {
		return nil, e
	}
	rows, e := tx.Query(ctx, `SELECT id::text FROM approval_requests WHERE resource_kind=$1 AND resource_id=$2 ORDER BY created_at DESC,id DESC LIMIT 20`, resource.Kind, resource.ID)
	if e != nil {
		return nil, e
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if e = rows.Scan(&id); e != nil {
			break
		}
		ids = append(ids, id)
	}
	if e == nil {
		e = rows.Err()
	}
	rows.Close()
	if e != nil {
		return nil, e
	}
	history := []any{}
	out := map[string]any{"enabled": true, "policy": policy, "resource_version": resource.Version, "resource_title": resource.Title, "history": history, "request": nil, "eligible_gates": []int{}, "stale": false}
	for i, id := range ids {
		request, e := approvalReadRequestTx(ctx, tx, id, false)
		if e != nil {
			return nil, e
		}
		summary := approvalSummary(request)
		history = append(history, summary)
		if i == 0 {
			out["request"] = summary
			if includeSnapshot {
				out["snapshot"] = request.Snapshot
			}
			if request.Status == "pending" {
				valid, reason, e := approvalCurrentTx(ctx, tx, request, resource)
				if e != nil {
					return nil, e
				}
				out["stale"] = !valid
				out["reason"] = reason
				if valid {
					out["eligible_gates"], e = s.approvalEligibleGatesTx(ctx, tx, request, p, resource)
					if e != nil {
						return nil, e
					}
				}
			}
			details, e := approvalDetailsTx(ctx, tx, request.ID)
			if e != nil {
				return nil, e
			}
			out["assignments"] = details["assignments"]
			out["decisions"] = details["decisions"]
		}
	}
	out["history"] = history
	return out, nil
}

func approvalDetailsTx(ctx context.Context, tx pgx.Tx, id string) (map[string]any, error) {
	queryRows := func(query string) ([]any, error) {
		rows, e := tx.Query(ctx, query, id)
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
	assignments, e := queryRows(`SELECT jsonb_build_object('stage_index',a.stage_index,'gate_index',a.gate_index,'user_id',a.user_id,'name',u.name) FROM approval_assignments a JOIN users u ON u.id=a.user_id WHERE a.request_id=$1 ORDER BY a.stage_index,a.gate_index,u.name`)
	if e != nil {
		return nil, e
	}
	decisions, e := queryRows(`SELECT to_jsonb(d)||jsonb_build_object('name',u.name) FROM approval_decisions d JOIN users u ON u.id=d.user_id WHERE d.request_id=$1 ORDER BY d.created_at,d.id`)
	return map[string]any{"assignments": assignments, "decisions": decisions}, e
}

func (s *Server) approvalDocumentStatus(w http.ResponseWriter, r *http.Request) {
	r.SetPathValue("kind", "document")
	s.approvalResourceStatus(w, r)
}
func (s *Server) approvalResourceStatus(w http.ResponseWriter, r *http.Request) {
	adapter, ok := s.approvalAdapters[r.PathValue("kind")]
	if !ok {
		apiError(w, 404, "승인 대상 모듈이 없습니다")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	resource, e := adapter.Lock(r.Context(), tx, current(r), r.PathValue("id"), false)
	var out map[string]any
	if e == nil {
		out, e = s.approvalStatusTx(r.Context(), tx, current(r), resource, false)
	}
	if e == nil && resource.Kind == "impact_exception" {
		var review map[string]any
		review, e = s.impactExceptionApprovalContextTx(r.Context(), tx, current(r), resource.ID)
		if e == nil {
			out["review_context"] = review
			if !boolean(review, "current") {
				out["stale"] = true
				out["reason"] = review["notice"]
				out["eligible_gates"] = []int{}
			}
		}
	}
	if e == nil {
		e = s.approvalActorTx(r, tx, resource.WorkspaceID, resource.Kind)
		if e == nil && resource.Kind == "impact_exception" {
			var allowed bool
			if e = tx.QueryRow(r.Context(), `SELECT madi_impact_allowed($1,$2)`, current(r).ID, str(resource.Snapshot, "review_id")).Scan(&allowed); e == nil && !allowed {
				e = approvalProblem(403, "현재 변경 원문·대상 접근 권한이 변경되었습니다")
			}
		}
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		approvalRespondError(w, e)
		return
	}
	jsonResponse(w, 200, out)
}

func (s *Server) approvalRequestDetail(w http.ResponseWriter, r *http.Request) {
	s.approvalRequestAction(w, r, false)
}
func (s *Server) approvalDecision(w http.ResponseWriter, r *http.Request) {
	s.approvalRequestAction(w, r, true)
}
func (s *Server) approvalRequestAction(w http.ResponseWriter, r *http.Request, write bool) {
	id := r.PathValue("id")
	if !validID(id) {
		apiError(w, 404, "검토 요청이 없습니다")
		return
	}
	var kind, resourceID string
	if e := s.DB.QueryRow(r.Context(), `SELECT resource_kind,resource_id::text FROM approval_requests WHERE id=$1`, id).Scan(&kind, &resourceID); e != nil {
		respond(w, nil, e)
		return
	}
	adapter, ok := s.approvalAdapters[kind]
	if !ok {
		apiError(w, 404, "승인 대상 모듈이 없습니다")
		return
	}
	var in approvalCommand
	if write {
		if current(r).TokenID != "" || current(r).ScopeRestricted || current(r).Kind != "user" {
			apiError(w, 403, "실제 사용자의 로그인 세션만 결정할 수 있습니다")
			return
		}
		if decode(r, &in) != nil || !oneOf(in.Action, "approve", "reject", "cancel") {
			apiError(w, 400, "검토 결정 형식을 확인하세요")
			return
		}
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	resource, e := adapter.Lock(r.Context(), tx, current(r), resourceID, false)
	if e == nil && write && oneOf(kind, "impact_exception", "learning_step") {
		if err := s.checkEvidenceProtection(r.Context(), tx, current(r), resource.WorkspaceID, in.Comment); err != nil {
			e = approvalProblem(422, err.Error())
		}
	}
	var request ApprovalRequest
	var out map[string]any
	if e == nil {
		if write {
			if in.Action == "cancel" {
				request, e = s.cancelApprovalTx(r.Context(), tx, current(r), resource, id, in.RequestVersion)
			} else {
				request, e = s.decideApprovalTx(r.Context(), tx, current(r), resource, id, in.RequestVersion, in.GateIndex, in.Action == "reject", in.Comment)
			}
			out = approvalSummary(request)
		} else {
			if _, _, e = approvalConfigurationTx(r.Context(), tx); e == nil {
				request, e = approvalReadRequestTx(r.Context(), tx, id, false)
			}
			if e == nil {
				out = approvalSummary(request)
				out["snapshot"] = request.Snapshot
				if kind == "impact_exception" {
					out["review_context"], e = s.impactExceptionApprovalContextTx(r.Context(), tx, current(r), resourceID)
					if e != nil {
						approvalRespondError(w, e)
						return
					}
				}
				details, err := approvalDetailsTx(r.Context(), tx, id)
				e = err
				if e == nil && oneOf(kind, "impact_exception", "learning_step") {
					if err = s.checkEvidenceProtection(r.Context(), tx, current(r), resource.WorkspaceID, details); err != nil {
						e = approvalProblem(422, "현재 보호 정책으로 예외 결정 의견을 표시할 수 없습니다")
					}
				}
				out["assignments"] = details["assignments"]
				out["decisions"] = details["decisions"]
				if e == nil && request.Status == "pending" {
					valid, reason, err := approvalCurrentTx(r.Context(), tx, request, resource)
					if err != nil {
						e = err
					} else {
						out["stale"] = !valid
						out["reason"] = reason
						if valid {
							out["eligible_gates"], e = s.approvalEligibleGatesTx(r.Context(), tx, request, current(r), resource)
						}
					}
				}
			}
		}
	}
	if e == nil {
		e = s.approvalActorTx(r, tx, resource.WorkspaceID, resource.Kind)
		if e == nil && resource.Kind == "impact_exception" {
			var allowed bool
			if e = tx.QueryRow(r.Context(), `SELECT madi_impact_allowed($1,$2)`, current(r).ID, str(resource.Snapshot, "review_id")).Scan(&allowed); e == nil && !allowed {
				e = approvalProblem(403, "현재 변경 원문·대상 접근 권한이 변경되었습니다")
			}
		}
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		approvalRespondError(w, e)
		return
	}
	if write {
		s.audit(r, "APPROVAL_"+strings.ToUpper(in.Action), id, map[string]any{"resource_kind": kind, "resource_id": resourceID, "status": request.Status})
	} else if kind == "impact_exception" {
		if review, ok := out["review_context"].(map[string]any); ok && !boolean(review, "current") {
			out["stale"] = true
			out["reason"] = review["notice"]
			out["eligible_gates"] = []int{}
		}
	}
	jsonResponse(w, 200, out)
}

// Resource adapters authorize their own ACL; this final guard additionally
// rejects a session/key that expired while any resource or decision lock waited.
func (s *Server) approvalActorTx(r *http.Request, tx pgx.Tx, wid, kind string) error {
	scope := "document:read"
	if kind == "sql_query_plan" {
		scope = "database:read"
	}
	if err := s.knowledgeActorTx(r, tx, wid, scope); err != nil {
		return approvalProblem(403, err.Error())
	}
	return nil
}

func (s *Server) approvalInbox(w http.ResponseWriter, r *http.Request) {
	cfg, e := s.settings(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	if !boolean(cfg, "approval_enabled") {
		apiError(w, 404, "검토·승인 기능이 비활성화되어 있습니다")
		return
	}
	rows, e := s.DB.Query(r.Context(), `SELECT DISTINCT ar.id::text,ar.resource_kind,ar.resource_id::text,ar.created_at FROM approval_requests ar JOIN approval_assignments a ON a.request_id=ar.id AND a.stage_index=ar.current_stage WHERE a.user_id=$1 AND ar.status='pending' AND ($2='' OR ar.workspace_id::text=$2) ORDER BY ar.created_at DESC LIMIT 100`, current(r).ID, r.URL.Query().Get("workspace_id"))
	if e != nil {
		respond(w, nil, e)
		return
	}
	type item struct{ id, kind, resource string }
	all := []item{}
	for rows.Next() {
		var item item
		var created any
		if e = rows.Scan(&item.id, &item.kind, &item.resource, &created); e != nil {
			break
		}
		all = append(all, item)
	}
	if e == nil {
		e = rows.Err()
	}
	rows.Close()
	if e != nil {
		respond(w, nil, e)
		return
	}
	out := []any{}
	for _, item := range all {
		adapter, ok := s.approvalAdapters[item.kind]
		if !ok {
			continue
		}
		tx, e := s.DB.Begin(r.Context())
		if e != nil {
			respond(w, nil, e)
			return
		}
		resource, e := adapter.Lock(r.Context(), tx, current(r), item.resource, false)
		if e != nil {
			tx.Rollback(r.Context())
			if errors.Is(e, pgx.ErrNoRows) {
				continue
			}
			var denied *approvalFailure
			if errors.As(e, &denied) && (denied.Status == 403 || denied.Status == 404) {
				continue
			}
			approvalRespondError(w, e)
			return
		}
		request, e := approvalReadRequestTx(r.Context(), tx, item.id, false)
		if e == nil {
			var valid bool
			var reason string
			valid, reason, e = approvalCurrentTx(r.Context(), tx, request, resource)
			if e == nil {
				var gates []int
				gates, e = s.approvalEligibleGatesTx(r.Context(), tx, request, current(r), resource)
				if e == nil && len(gates) > 0 {
					summary := approvalSummary(request)
					summary["title"] = resource.Title
					summary["stale"] = !valid
					summary["reason"] = reason
					summary["eligible_gates"] = gates
					summary["document_id"] = resource.DocumentID
					out = append(out, summary)
				}
			}
		}
		tx.Rollback(r.Context())
		if e != nil {
			approvalRespondError(w, e)
			return
		}
	}
	jsonResponse(w, 200, out)
}

func (s *Server) approvalSavePolicy(w http.ResponseWriter, r *http.Request) {
	in := approvalPolicy{Enabled: true, ResourceKind: "document"}
	if decode(r, &in) != nil {
		apiError(w, 400, "정책 형식을 확인하세요")
		return
	}
	if e := approvalValidatePolicy(&in); e != nil {
		approvalRespondError(w, e)
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	// Serialize insertion of a new policy with decisions using the fallback
	// policy too: no policy row exists to lock in that case.
	var settingsID int
	if e = tx.QueryRow(r.Context(), `SELECT id FROM settings WHERE id=1 FOR UPDATE`).Scan(&settingsID); e != nil {
		respond(w, nil, e)
		return
	}
	if e = approvalValidatePolicyTargetsTx(r.Context(), tx, in); e != nil {
		approvalRespondError(w, e)
		return
	}
	if r.Method == "POST" {
		in.ID = newID()
		in.Version = 1
		_, e = tx.Exec(r.Context(), `INSERT INTO approval_policies(id,workspace_id,space_id,resource_kind,name,enabled,stages,created_by) VALUES($1,$2,NULLIF($3,'')::uuid,$4,$5,$6,$7,$8)`, in.ID, in.WorkspaceID, in.SpaceID, in.ResourceKind, in.Name, in.Enabled, jsonValue(in.Stages), current(r).ID)
	} else {
		if !validID(r.PathValue("id")) || in.Version < 1 {
			apiError(w, 400, "정책 ID와 버전이 필요합니다")
			return
		}
		in.ID = r.PathValue("id")
		tag, err := tx.Exec(r.Context(), `UPDATE approval_policies SET name=$2,enabled=$3,stages=$4,version=version+1,updated_at=now() WHERE id=$1 AND version=$5 AND workspace_id=$6 AND COALESCE(space_id::text,'')=$7 AND resource_kind=$8`, in.ID, in.Name, in.Enabled, jsonValue(in.Stages), in.Version, in.WorkspaceID, in.SpaceID, in.ResourceKind)
		e = err
		if e == nil && tag.RowsAffected() != 1 {
			apiError(w, 409, "정책 버전 또는 고정 대상 범위가 변경되었습니다")
			return
		}
		in.Version++
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		approvalRespondError(w, e)
		return
	}
	s.audit(r, "APPROVAL_POLICY_SAVE", in.ID, map[string]any{"workspace_id": in.WorkspaceID, "version": in.Version})
	jsonResponse(w, 200, in)
}
func (s *Server) approvalDisablePolicy(w http.ResponseWriter, r *http.Request) {
	if !validID(r.PathValue("id")) {
		apiError(w, 404, "정책이 없습니다")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	var settingsID int
	if e = tx.QueryRow(r.Context(), `SELECT id FROM settings WHERE id=1 FOR UPDATE`).Scan(&settingsID); e != nil {
		respond(w, nil, e)
		return
	}
	tag, e := tx.Exec(r.Context(), `UPDATE approval_policies SET enabled=false,version=version+1,updated_at=now() WHERE id=$1`, r.PathValue("id"))
	if e != nil {
		respond(w, nil, e)
		return
	}
	if tag.RowsAffected() == 0 {
		apiError(w, 404, "정책이 없습니다")
		return
	}
	if e = tx.Commit(r.Context()); e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "APPROVAL_POLICY_DISABLE", r.PathValue("id"), nil)
	jsonResponse(w, 200, map[string]bool{"ok": true})
}
