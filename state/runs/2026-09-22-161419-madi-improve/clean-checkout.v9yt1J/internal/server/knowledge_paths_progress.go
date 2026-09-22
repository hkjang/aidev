package server

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
)

type learningProgress struct {
	ID              string    `json:"id"`
	StepID          string    `json:"step_id"`
	UserID          string    `json:"user_id"`
	PathRevision    int64     `json:"path_revision"`
	DocumentVersion int       `json:"document_version"`
	State           string    `json:"state"`
	Revision        int64     `json:"revision"`
	ApprovalID      string    `json:"approval_id"`
	Proof           string    `json:"-"`
	UpdatedAt       time.Time `json:"updated_at"`
}

const learningProgressSelect = `g.id::text,g.step_id::text,g.user_id::text,g.path_revision,g.document_version,g.state,g.revision,coalesce(g.approval_id::text,''),g.proof,g.updated_at`

func scanLearningProgress(row pgx.Row) (learningProgress, error) {
	var g learningProgress
	e := row.Scan(&g.ID, &g.StepID, &g.UserID, &g.PathRevision, &g.DocumentVersion, &g.State, &g.Revision, &g.ApprovalID, &g.Proof, &g.UpdatedAt)
	return g, e
}
func learningReviewPolicyTx(ctx context.Context, tx pgx.Tx, path knowledgePath) (bool, error) {
	cfg, _, e := approvalConfigurationTx(ctx, tx)
	if e == nil {
		_, e = approvalPolicyTx(ctx, tx, cfg, path.WorkspaceID, path.SpaceID, "learning_step")
	}
	var fail *approvalFailure
	if errors.As(e, &fail) && (fail.Status == 404 || fail.Status == 409) {
		return false, nil
	}
	return e == nil, e
}
func (s *Server) learningCurrentTx(ctx context.Context, tx pgx.Tx, p *Principal, path knowledgePath, step knowledgePathStep, g learningProgress, version int, policy bool) (bool, error) {
	if path.Archived || g.ID == "" || g.PathRevision != path.Revision || g.DocumentVersion != version || version < 1 {
		return false, nil
	}
	if step.Kind != "review" {
		return g.State == "confirmed", nil
	}
	if !policy || g.State != "approved" || g.ApprovalID == "" {
		return false, nil
	}
	res, e := s.learningResourceTx(ctx, tx, p, g.ID, false)
	if e != nil {
		var problem *approvalFailure
		if errors.Is(e, pgx.ErrNoRows) || errors.As(e, &problem) && (problem.Status == 404 || problem.Status == 409 || problem.Status == 422) {
			return false, nil
		}
		return false, e
	}
	a, e := approvalReadRequestTx(ctx, tx, g.ApprovalID, false)
	if e != nil {
		return false, e
	}
	if a.Status != "approved" {
		return false, nil
	}
	valid, _, e := approvalCurrentTx(ctx, tx, a, res)
	var failure *approvalFailure
	if errors.As(e, &failure) && (failure.Status == 404 || failure.Status == 409) {
		return false, nil
	}
	return valid, e
}
func (s *Server) learningDetailTx(r *http.Request, tx pgx.Tx, path knowledgePath) (map[string]any, error) {
	steps, e := knowledgePathStepsTx(r.Context(), tx, path.ID)
	if e != nil {
		return nil, e
	}
	policy, e := learningReviewPolicyTx(r.Context(), tx, path)
	if e != nil {
		return nil, e
	}
	out := []map[string]any{}
	done, total, omitted := 0, 0, 0
	prior := true
	for _, step := range steps {
		item := map[string]any{"id": step.ID, "ordinal": step.Ordinal, "kind": step.Kind, "available": false, "current": false, "can_confirm": false}
		isOmitted := step.Kind == "review" && !policy
		item["omitted"] = isOmitted
		if isOmitted {
			omitted++
		} else {
			total++
		}
		var version int
		var title string
		e = tx.QueryRow(r.Context(), `SELECT version,title FROM documents WHERE id=$1 AND deleted_at IS NULL AND madi_document_allowed($2,id,false)`, step.DocumentID, current(r).ID).Scan(&version, &title)
		if errors.Is(e, pgx.ErrNoRows) {
			if !isOmitted {
				prior = false
			}
			out = append(out, item)
			continue
		}
		if e != nil {
			return nil, e
		}
		item["available"], item["document_id"], item["document_version"], item["document_title"], item["title"], item["instruction"] = true, step.DocumentID, version, title, step.Title, step.Instruction
		g, e := scanLearningProgress(tx.QueryRow(r.Context(), "SELECT "+learningProgressSelect+" FROM knowledge_path_progress g WHERE step_id=$1 AND user_id=$2", step.ID, current(r).ID))
		if e != nil && !errors.Is(e, pgx.ErrNoRows) {
			return nil, e
		}
		valid, e := s.learningCurrentTx(r.Context(), tx, current(r), path, step, g, version, policy)
		if e != nil {
			return nil, e
		}
		item["progress"], item["current"], item["needs_recheck"], item["can_confirm"] = g, valid, g.ID != "" && !valid, prior && !isOmitted && !path.Archived && learningCookie(r)
		if g.Proof != "" && learningCookie(r) && (step.Kind != "review" || prior) {
			proof, e := s.decrypt(g.Proof)
			if e != nil {
				return nil, e
			}
			if e = s.checkEvidenceProtection(r.Context(), tx, current(r), path.WorkspaceID, map[string]any{"proof": proof}); e == nil {
				item["proof"] = proof
			} else {
				item["proof_hidden"] = true
			}
		}
		if !isOmitted {
			if valid && prior {
				done++
			} else {
				prior = false
			}
		}
		out = append(out, item)
	}
	var writable bool
	e = tx.QueryRow(r.Context(), `SELECT madi_knowledge_path_allowed($1,$2,true)`, current(r).ID, path.ID).Scan(&writable)
	if e != nil {
		return nil, e
	}
	return map[string]any{"path": path, "steps": out, "completed": done, "total": total, "omitted_reviews": omitted, "review_policy_configured": policy, "can_manage": writable && learningCookie(r), "notice": "문서 열람은 완료가 아닙니다. 읽기 확인·실습 결과·독립 검토를 각각 기록합니다. 승인 정책 없는 검토는 분모에서 제외되며, 경로 또는 원문 변경 뒤에는 재확인이 필요합니다. 실행·공유 권한은 부여하지 않습니다."}, nil
}
func (s *Server) getKnowledgePath(w http.ResponseWriter, r *http.Request) {
	tx, e := s.learningTx(r, true)
	if e != nil {
		learningRespondError(w, e)
		return
	}
	defer tx.Rollback(r.Context())
	path, e := knowledgePathReadTx(r.Context(), tx, current(r), r.PathValue("id"), false)
	if e != nil {
		learningRespondError(w, e)
		return
	}
	out, e := s.learningDetailTx(r, tx, path)
	if e == nil {
		e = s.learningActorTx(r, tx, path.WorkspaceID, "document:read")
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		learningRespondError(w, e)
		return
	}
	jsonResponse(w, 200, out)
}
func (s *Server) confirmLearningStep(w http.ResponseWriter, r *http.Request) {
	var in struct {
		PathRevision    int64  `json:"path_revision"`
		DocumentVersion int    `json:"document_version"`
		Revision        int64  `json:"revision"`
		Proof           string `json:"proof"`
		Confirmation    string `json:"confirmation"`
	}
	if !learningCookie(r) {
		apiError(w, 403, "개인 완료 확인은 사람의 로그인 화면에서만 가능합니다")
		return
	}
	if decode(r, &in) != nil || in.Confirmation != "CONFIRM" || in.PathRevision < 1 || in.DocumentVersion < 1 || in.Revision < 0 || len(in.Proof) > 16384 || !utf8.ValidString(in.Proof) || strings.ContainsRune(in.Proof, 0) {
		apiError(w, 400, "현재 버전과 완료 확인, 16KiB 이하 실습 기록을 확인하세요")
		return
	}
	tx, e := s.learningTx(r, true)
	if e != nil {
		learningRespondError(w, e)
		return
	}
	defer tx.Rollback(r.Context())
	path, e := knowledgePathReadTx(r.Context(), tx, current(r), r.PathValue("id"), false)
	if e != nil {
		learningRespondError(w, e)
		return
	}
	if path.Archived || path.Revision != in.PathRevision {
		apiError(w, 409, "경로 구성이 변경됐거나 보관되었습니다")
		return
	}
	steps, e := knowledgePathStepsTx(r.Context(), tx, path.ID)
	if e != nil {
		learningRespondError(w, e)
		return
	}
	var chosen knowledgePathStep
	prefix := []knowledgePathStep{}
	for _, step := range steps {
		prefix = append(prefix, step)
		if step.ID == r.PathValue("step") {
			chosen = step
			break
		}
	}
	if chosen.ID == "" {
		apiError(w, 404, "활성 단계가 없습니다")
		return
	}
	versions, e := learningDocumentsTx(r.Context(), tx, current(r), path.WorkspaceID, prefix)
	if e != nil {
		learningRespondError(w, e)
		return
	}
	if versions[chosen.DocumentID] != in.DocumentVersion {
		apiError(w, 409, "원문 버전이 변경됐습니다. 원문을 다시 확인하세요")
		return
	}
	policy, e := learningReviewPolicyTx(r.Context(), tx, path)
	if e != nil {
		learningRespondError(w, e)
		return
	}
	if chosen.Kind == "review" && !policy {
		apiError(w, 409, "관리자가 learning_step 승인 정책을 설정해야 검토를 요청할 수 있습니다")
		return
	}
	for _, step := range prefix {
		if step.ID == chosen.ID {
			break
		}
		if step.Kind == "review" && !policy {
			continue
		}
		g, err := scanLearningProgress(tx.QueryRow(r.Context(), "SELECT "+learningProgressSelect+" FROM knowledge_path_progress g WHERE step_id=$1 AND user_id=$2 FOR SHARE", step.ID, current(r).ID))
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			learningRespondError(w, err)
			return
		}
		ok, err := s.learningCurrentTx(r.Context(), tx, current(r), path, step, g, versions[step.DocumentID], policy)
		if err != nil {
			learningRespondError(w, err)
			return
		}
		if !ok {
			apiError(w, 409, "앞 단계를 현재 원문 버전으로 먼저 완료하세요")
			return
		}
	}
	if chosen.Kind == "read" {
		in.Proof = ""
	} else if strings.TrimSpace(in.Proof) == "" {
		apiError(w, 400, "실습 결과 또는 독립 검토 요청 사유를 직접 작성하세요")
		return
	}
	protected, e := s.ProtectDocumentMetadataTx(r.Context(), tx, current(r), chosen.DocumentID, path.WorkspaceID, map[string]any{"proof": in.Proof})
	if e != nil {
		learningRespondError(w, e)
		return
	}
	if len(protected.Findings) > 0 && oneOf(protected.Mode, "warn", "audit") {
		_, e = tx.Exec(r.Context(), `INSERT INTO protection_events(id,document_id,workspace_id,user_id,action,mode,findings) VALUES($1,$2,$3,$4,'learning.proof',$5,$6)`, newID(), chosen.DocumentID, path.WorkspaceID, current(r).ID, protected.Mode, jsonValue(protected.Findings))
		if e != nil {
			learningRespondError(w, e)
			return
		}
	}
	var cleaned map[string]any
	if e = json.Unmarshal(jsonValue(protected.Value), &cleaned); e != nil {
		learningRespondError(w, e)
		return
	}
	in.Proof = str(cleaned, "proof")
	if chosen.Kind != "read" && strings.TrimSpace(in.Proof) == "" {
		apiError(w, 400, "정제된 실습 기록을 다시 확인하세요")
		return
	}
	cipher := ""
	if in.Proof != "" {
		cipher, e = s.encrypt(in.Proof)
		if e != nil {
			learningRespondError(w, e)
			return
		}
	}
	// Serialize personal progress before CAS. The path share lock keeps the step active.
	if _, e = tx.Exec(r.Context(), `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, chosen.ID+":"+current(r).ID); e != nil {
		learningRespondError(w, e)
		return
	}
	old, e := scanLearningProgress(tx.QueryRow(r.Context(), "SELECT "+learningProgressSelect+" FROM knowledge_path_progress g WHERE step_id=$1 AND user_id=$2 FOR UPDATE", chosen.ID, current(r).ID))
	if e != nil && !errors.Is(e, pgx.ErrNoRows) {
		learningRespondError(w, e)
		return
	}
	if old.Revision != in.Revision {
		apiError(w, 409, "개인 진행 기록이 변경되었습니다. 최신 상태를 다시 확인하세요")
		return
	}
	id := old.ID
	if id == "" {
		id = newID()
	}
	state := "confirmed"
	if chosen.Kind == "review" {
		state = "pending_review"
	}
	if old.ID != "" {
		_, e = tx.Exec(r.Context(), `INSERT INTO knowledge_path_events(id,path_id,step_id,user_id,action,metadata,proof) VALUES($1,$2,$3,$4,'previous_progress',$5,$6)`, newID(), path.ID, chosen.ID, current(r).ID, jsonValue(old), old.Proof)
		if e != nil {
			learningRespondError(w, e)
			return
		}
	}
	_, e = tx.Exec(r.Context(), `INSERT INTO knowledge_path_progress(id,step_id,user_id,path_revision,document_version,state,revision,proof,confirmed_at) VALUES($1,$2,$3,$4,$5,$6,1,$7,clock_timestamp()) ON CONFLICT(step_id,user_id) DO UPDATE SET path_revision=excluded.path_revision,document_version=excluded.document_version,state=excluded.state,revision=knowledge_path_progress.revision+1,proof=excluded.proof,approval_id=NULL,confirmed_at=clock_timestamp(),updated_at=clock_timestamp()`, id, chosen.ID, current(r).ID, path.Revision, in.DocumentVersion, state, cipher)
	approvalID := ""
	if e == nil && chosen.Kind == "review" {
		var a ApprovalRequest
		a, e = s.StartApprovalTx(r.Context(), tx, current(r), "learning_step", id, "")
		if e == nil {
			approvalID = a.ID
			_, e = tx.Exec(r.Context(), `UPDATE knowledge_path_progress SET approval_id=$2 WHERE id=$1`, id, a.ID)
		}
	}
	if e == nil {
		_, e = tx.Exec(r.Context(), `INSERT INTO knowledge_path_events(id,path_id,step_id,user_id,action,metadata) VALUES($1,$2,$3,$4,$5,$6)`, newID(), path.ID, chosen.ID, current(r).ID, state, jsonValue(map[string]any{"progress_id": id, "revision": old.Revision + 1, "document_version": in.DocumentVersion, "path_revision": path.Revision, "approval_id": approvalID}))
	}
	if e == nil {
		e = s.learningActorTx(r, tx, path.WorkspaceID, "document:read")
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		learningRespondError(w, e)
		return
	}
	jsonResponse(w, 200, map[string]any{"id": id, "revision": old.Revision + 1, "state": state, "approval_id": approvalID, "proof": in.Proof, "protection": map[string]any{"changed": protected.Changed, "mode": protected.Mode, "findings": protected.Findings}})
}
