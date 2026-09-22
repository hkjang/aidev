package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
)

func (s *Server) personalQuestionTx(r *http.Request, tx pgx.Tx, id string) (map[string]any, error) {
	if !personalAIHistory(current(r)) || !validID(id) {
		return nil, pgx.ErrNoRows
	}
	query := `SELECT jsonb_build_object('workspace_id',c.workspace_id,'question',m.question,'answer',m.answer,'sources',m.sources,'model',m.model) FROM ai_messages m JOIN ai_conversations c ON c.id=m.conversation_id WHERE m.id=$1 AND c.owner_id=$2 AND madi_ai_conversation_allowed($2,c.id) AND ($3='' OR c.workspace_id::text=$3)`
	var raw []byte
	var row map[string]any
	if e := tx.QueryRow(r.Context(), query, id, current(r).ID, current(r).WorkspaceID).Scan(&raw); e != nil {
		return nil, e
	}
	if e := json.Unmarshal(raw, &row); e != nil {
		return nil, e
	}
	originalHash := digest(string(raw))
	var sources []aiSource
	if e := json.Unmarshal(jsonValue(row["sources"]), &sources); e != nil {
		return nil, e
	}
	if len(sources) == 0 || len(sources) > 32 || len(str(row, "question")) > 2000 || len(str(row, "answer")) > 4<<20 {
		return nil, errors.New("근거 1~32개·질문2000바이트·답변4MiB 범위의 기록만 직접 정리할 수 있습니다. 긴 질문은 새 관리 질문으로 요약하고 근거를 직접 선택하세요")
	}
	refs := []questionRef{}
	seen := map[string]bool{}
	for _, src := range sources {
		ref := questionRef{ID: src.ID, Version: src.Version}
		if src.AttachmentID != "" {
			copy := src
			ref.Attachment = &copy
		}
		if !questionRefValid(ref) {
			return nil, errors.New("개인 기록의 근거 형식을 다시 확인하세요")
		}
		if seen[questionRefKey(ref)] {
			continue
		}
		seen[questionRefKey(ref)] = true
		refs = append(refs, ref)
	}
	docs, e := s.questionDocumentsTx(r, tx, refs[0].ID, refs, false)
	if e != nil {
		return nil, e
	}
	refs, _, e = s.questionRefsTx(r, tx, docs, refs, true)
	if e != nil {
		return nil, e
	}
	// Never infer that private history has been shared by merely opening this view.
	if e = tx.QueryRow(r.Context(), query+` FOR SHARE OF m,c`, id, current(r).ID, current(r).WorkspaceID).Scan(&raw); e != nil {
		return nil, e
	}
	if digest(string(raw)) != originalHash {
		return nil, errors.New("개인 기록이 변경되었습니다")
	}
	if e = s.checkEvidenceProtection(r.Context(), tx, current(r), str(row, "workspace_id"), []any{row["question"], row["answer"]}); e != nil {
		return nil, e
	}
	for _, d := range docs {
		if e = s.checkEvidenceProtection(r.Context(), tx, current(r), str(row, "workspace_id"), []any{d["title"], d["markdown"]}); e != nil {
			return nil, e
		}
	}
	row["sources"] = refs
	row["preview_hash"] = digest(string(jsonValue(row)))
	row["notice"] = "이 화면은 개인 기록 미리보기입니다. 명시적으로 선택한 질문과 답변만 새 비공개 초안으로 복사한 뒤 관리 질문에 연결합니다. 기존 대화 전체·다른 메시지·실행 권한은 공유되지 않습니다. 팀 공유·게시·공식 확인은 별도 작업입니다."
	return row, nil
}
func (s *Server) personalQuestionPreview(w http.ResponseWriter, r *http.Request) {
	tx, e := s.questionTx(r)
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	out, e := s.personalQuestionTx(r, tx, r.PathValue("id"))
	if e != nil {
		apiError(w, 404, "현재 본인의 기록과 최신 근거로 정리할 수 있는 질문이 없습니다")
		return
	}
	if e = s.knowledgeActorTx(r, tx, str(out, "workspace_id"), "document:read"); e != nil {
		apiError(w, 403, e.Error())
		return
	}
	jsonResponse(w, 200, out)
}
