package server

import (
	"context"
	_ "embed"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

//go:embed knowledge_impact_exception.sql
var impactExceptionSchema string

type impactException struct {
	ID, ReviewID, RequesterID, ReasonCiphertext, ReasonHash, Status, ApprovalID string
	ReviewRevision, SourceVersion, TargetVersion                                int
	ValidUntil, CreatedAt                                                       time.Time
}
type impactExceptionReview struct {
	ID, SourceID, TargetID, WorkspaceID, SpaceID, OwnerID, Status, Relation            string
	SourceTitle, TargetTitle, RelationCreated                                          string
	SourceHash, TargetHash                                                             string
	Revision, SourceVersion, TargetVersion, CurrentSourceVersion, CurrentTargetVersion int
	Related                                                                            bool
}

// Locks follow document -> dependent review -> exception -> approval ordering.
// No callbacks acquire a connection from the pool inside these transactions.
func impactExceptionReviewTx(ctx context.Context, tx pgx.Tx, p *Principal, id string, write bool) (impactExceptionReview, error) {
	var out impactExceptionReview
	if p == nil || !validID(id) || !hasIntegrationScope(p, "document:read") || p.PluginID != "" {
		return out, approvalProblem(404, "접근 가능한 영향 검토가 없습니다")
	}
	id = strings.ToLower(id)
	out.ID = id
	if err := tx.QueryRow(ctx, `SELECT source_id::text,target_id::text FROM knowledge_impact_reviews WHERE id=$1 AND madi_impact_allowed($2,id)`, id, p.ID).Scan(&out.SourceID, &out.TargetID); err != nil {
		return out, approvalProblem(404, "접근 가능한 영향 검토가 없습니다")
	}
	rows, err := tx.Query(ctx, `SELECT id FROM documents WHERE id=ANY($1::uuid[]) ORDER BY id FOR SHARE`, []string{out.SourceID, out.TargetID})
	if err != nil {
		return out, err
	}
	for rows.Next() {
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, err
	}
	err = tx.QueryRow(ctx, `SELECT r.revision,r.source_version,r.target_version,r.owner_id::text,r.status,r.relation_type,s.workspace_id::text,coalesce(d.space_id::text,''),s.version,d.version,s.title,d.title,EXISTS(SELECT 1 FROM document_relations x WHERE x.source_id=d.id AND x.target_id=s.id AND x.type=r.relation_type) FROM knowledge_impact_reviews r JOIN documents s ON s.id=r.source_id JOIN documents d ON d.id=r.target_id WHERE r.id=$1 AND s.id=$3 AND d.id=$4 AND madi_impact_allowed($2,r.id) FOR UPDATE OF r`, id, p.ID, out.SourceID, out.TargetID).Scan(&out.Revision, &out.SourceVersion, &out.TargetVersion, &out.OwnerID, &out.Status, &out.Relation, &out.WorkspaceID, &out.SpaceID, &out.CurrentSourceVersion, &out.CurrentTargetVersion, &out.SourceTitle, &out.TargetTitle, &out.Related)
	if err != nil {
		return out, approvalProblem(404, "접근 가능한 영향 검토가 없습니다")
	}
	if p.WorkspaceID != "" && p.WorkspaceID != out.WorkspaceID {
		return out, approvalProblem(404, "접근 가능한 영향 검토가 없습니다")
	}
	// Hash the exact reviewed text/labels as a backstop to version CAS; do not
	// retain either canonical body in the generic approval snapshot.
	if err = tx.QueryRow(ctx, `SELECT encode(sha256(convert_to(jsonb_build_array(s.title,s.markdown,s.tags,s.aliases)::text,'UTF8')),'hex'),encode(sha256(convert_to(jsonb_build_array(d.title,d.markdown,d.tags,d.aliases)::text,'UTF8')),'hex') FROM documents s,documents d WHERE s.id=$1 AND d.id=$2`, out.SourceID, out.TargetID).Scan(&out.SourceHash, &out.TargetHash); err != nil {
		return out, err
	}
	if out.Related {
		if err = tx.QueryRow(ctx, `SELECT (created_at AT TIME ZONE 'UTC')::text FROM document_relations WHERE source_id=$1 AND target_id=$2 AND type=$3 FOR SHARE`, out.TargetID, out.SourceID, out.Relation).Scan(&out.RelationCreated); err != nil {
			return out, approvalProblem(409, "직접 의존 관계가 변경되었습니다")
		}
	}
	if write {
		var allowed bool
		err = tx.QueryRow(ctx, `SELECT madi_document_allowed($1,$2,true) AND ($1::uuid=$3 OR EXISTS(SELECT 1 FROM workspace_members WHERE user_id=$1 AND workspace_id=$4 AND role IN ('owner','admin')))`, p.ID, out.TargetID, out.OwnerID, out.WorkspaceID).Scan(&allowed)
		if err != nil {
			return out, err
		}
		if !allowed || !hasIntegrationScope(p, "document:write") || p.TokenID != "" || p.ScopeRestricted || p.Kind != "user" {
			return out, approvalProblem(403, "현재 문서 작성 권한이 있는 담당자·워크스페이스 관리자의 로그인 세션이 필요합니다")
		}
	}
	return out, nil
}
func impactExceptionReadTx(ctx context.Context, tx pgx.Tx, id string) (impactException, error) {
	var out impactException
	err := tx.QueryRow(ctx, `SELECT id::text,review_id::text,requester_id::text,review_revision,source_version,target_version,reason_ciphertext,reason_hash,valid_until,status,coalesce(approval_id::text,''),created_at FROM knowledge_impact_exceptions WHERE id=$1 FOR UPDATE`, id).Scan(&out.ID, &out.ReviewID, &out.RequesterID, &out.ReviewRevision, &out.SourceVersion, &out.TargetVersion, &out.ReasonCiphertext, &out.ReasonHash, &out.ValidUntil, &out.Status, &out.ApprovalID, &out.CreatedAt)
	return out, err
}
func impactExceptionResource(ex impactException, review impactExceptionReview) ApprovalResource {
	return ApprovalResource{Kind: "impact_exception", ID: ex.ID, WorkspaceID: review.WorkspaceID, SpaceID: review.SpaceID, OwnerID: review.OwnerID, DocumentID: review.TargetID, Title: "변경 영향 예외 검토", Version: 1, Snapshot: map[string]any{
		"exception_id": ex.ID, "review_id": review.ID, "review_revision": review.Revision, "review_status": review.Status,
		"source_id": review.SourceID, "source_version": review.CurrentSourceVersion, "target_id": review.TargetID, "target_version": review.CurrentTargetVersion,
		"source_hash": review.SourceHash, "target_hash": review.TargetHash,
		"relation_type": review.Relation, "relation_present": review.Related, "relation_created": review.RelationCreated, "owner_id": review.OwnerID, "reason_hash": ex.ReasonHash, "valid_until": ex.ValidUntil.UTC().Format(time.RFC3339Nano),
	}}
}
func impactExceptionCurrent(ex impactException, review impactExceptionReview) bool {
	return review.Status == "exception_requested" && review.Related && ex.ReviewRevision == review.Revision && ex.SourceVersion == review.CurrentSourceVersion && ex.SourceVersion == review.SourceVersion && ex.TargetVersion == review.CurrentTargetVersion && ex.TargetVersion == review.TargetVersion && ex.ValidUntil.After(time.Now())
}
func (s *Server) impactExceptionApprovalAdapter() ApprovalAdapter {
	return ApprovalAdapter{
		Lock: func(ctx context.Context, tx pgx.Tx, p *Principal, id string, write bool) (ApprovalResource, error) {
			var rid string
			var zero ApprovalResource
			if !validID(id) || tx.QueryRow(ctx, `SELECT review_id::text FROM knowledge_impact_exceptions WHERE id=$1 AND madi_impact_allowed($2,review_id)`, id, p.ID).Scan(&rid) != nil {
				return zero, approvalProblem(404, "접근 가능한 예외 요청이 없습니다")
			}
			review, err := impactExceptionReviewTx(ctx, tx, p, rid, write)
			if err != nil {
				return zero, err
			}
			ex, err := impactExceptionReadTx(ctx, tx, id)
			if err != nil {
				return zero, err
			}
			reason, err := s.decrypt(ex.ReasonCiphertext)
			if err != nil {
				return zero, err
			}
			if err = s.checkEvidenceProtection(ctx, tx, p, review.WorkspaceID, reason); err != nil {
				return zero, approvalProblem(422, "현재 보호 정책으로 예외 근거를 표시할 수 없습니다")
			}
			if write && (ex.RequesterID != p.ID || ex.Status != "pending" || !impactExceptionCurrent(ex, review)) {
				return zero, approvalProblem(409, "예외 요청의 원문·검토·기한이 변경되었습니다")
			}
			return impactExceptionResource(ex, review), nil
		},
		CanReview: func(ctx context.Context, tx pgx.Tx, uid string, res ApprovalResource) (bool, error) {
			var allowed bool
			err := tx.QueryRow(ctx, `SELECT madi_impact_allowed($1,$2) AND EXISTS(SELECT 1 FROM users WHERE id=$1 AND NOT disabled AND kind='user') AND $3::timestamptz>clock_timestamp()`, uid, str(res.Snapshot, "review_id"), str(res.Snapshot, "valid_until")).Scan(&allowed)
			return allowed, err
		},
		Complete: func(ctx context.Context, tx pgx.Tx, p *Principal, request ApprovalRequest, res ApprovalResource) error {
			// Expiry is a wall-clock condition, so it is checked again after decision
			// locks, not represented as an unstable bool in the immutable hash.
			if request.Status == "approved" {
				var fresh bool
				if err := tx.QueryRow(ctx, `SELECT valid_until>clock_timestamp() FROM knowledge_impact_exceptions WHERE id=$1`, res.ID).Scan(&fresh); err != nil {
					return err
				}
				if !fresh {
					return approvalProblem(409, "예외 유효기간이 지났습니다")
				}
			}
			if !oneOf(request.Status, "approved", "rejected", "cancelled") {
				return approvalProblem(409, "예외 결정 상태를 확인하세요")
			}
			tag, err := tx.Exec(ctx, `UPDATE knowledge_impact_exceptions SET status=$2,updated_at=clock_timestamp() WHERE id=$1 AND status='pending'`, res.ID, request.Status)
			if err == nil && tag.RowsAffected() != 1 {
				return approvalProblem(409, "이미 처리된 예외 요청입니다")
			}
			return err
		},
	}
}

func (s *Server) registerImpactExceptions() {
	s.RegisterApprovalAdapter("impact_exception", s.impactExceptionApprovalAdapter())
	s.handle("GET /api/v1/knowledge/impact-reviews/{id}/exceptions", s.listImpactExceptions)
	s.handle("POST /api/v1/knowledge/impact-reviews/{id}/exceptions", s.createImpactException)
	s.handle("GET /api/v1/knowledge/impact-exceptions/{id}", s.getImpactException)
}

// Restoration never revives an exception authorization from another system.
func invalidateImpactExceptionsRestoreTx(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `UPDATE knowledge_impact_exceptions SET status='cancelled',updated_at=clock_timestamp() WHERE status IN ('pending','approved')`)
	return err
}

func impactExceptionHuman(r *http.Request) bool {
	p := current(r)
	return p != nil && p.Kind == "user" && p.TokenID == "" && !p.ScopeRestricted && p.PluginID == ""
}

func (s *Server) impactExceptionActorTx(r *http.Request, tx pgx.Tx, review impactExceptionReview, write bool) error {
	scopes := []string{"document:read"}
	if write {
		scopes = append(scopes, "document:write")
	}
	if err := s.knowledgeActorTx(r, tx, review.WorkspaceID, scopes...); err != nil {
		return approvalProblem(403, err.Error())
	}
	var allowed bool
	err := tx.QueryRow(r.Context(), `SELECT madi_impact_allowed($1,$2) AND (NOT $3 OR madi_document_allowed($1,$4,true))`, current(r).ID, review.ID, write, review.TargetID).Scan(&allowed)
	if err != nil {
		return err
	}
	if !allowed {
		return approvalProblem(403, "현재 변경 원문·대상 접근 권한이 변경되었습니다")
	}
	return nil
}
