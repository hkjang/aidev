package server

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/jackc/pgx/v5"
)

type questionDraftKey struct{}
type questionDraftIntent struct{ MessageID, Hash string }

func (s *Server) createPersonalQuestionDraft(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Hash    string `json:"preview_hash"`
		Consent bool   `json:"consent"`
	}
	if !personalAIHistory(current(r)) || !validID(r.PathValue("id")) {
		apiError(w, 403, "본인의 개인 질문만 비공개 초안으로 정리할 수 있습니다")
		return
	}
	if decode(r, &in) != nil || !in.Consent || len(in.Hash) != 64 {
		apiError(w, 400, "현재 미리보기 해시와 비공개 초안 생성 동의가 필요합니다")
		return
	}
	tx, e := s.questionTx(r)
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	preview, e := s.personalQuestionTx(r, tx, r.PathValue("id"))
	if e != nil || str(preview, "preview_hash") != in.Hash {
		apiError(w, 409, "개인 질문 또는 근거가 변경됐습니다. 현재 권한으로 다시 미리보세요")
		return
	}
	if e = s.knowledgeActorTx(r, tx, str(preview, "workspace_id"), "document:read", "document:write"); e != nil {
		apiError(w, 403, e.Error())
		return
	}
	var docID, hash string
	e = tx.QueryRow(r.Context(), `SELECT coalesce(document_id::text,''),preview_hash FROM knowledge_question_draft_receipts WHERE owner_id=$1 AND message_id=$2`, current(r).ID, r.PathValue("id")).Scan(&docID, &hash)
	if e == nil {
		_ = tx.Rollback(r.Context())
		if docID == "" || hash != in.Hash {
			apiError(w, 409, "이 질문의 이전 초안이 변경 또는 삭제됐습니다. 새 관리 질문에서 현재 답변을 직접 선택하세요")
			return
		}
		doc, e := s.document(r, docID)
		if e != nil || doc["deleted_at"] != nil {
			apiError(w, 404, "이전에 만든 답변 초안에 현재 접근할 수 없습니다")
			return
		}
		if str(doc, "visibility") != "private" || str(doc, "owner_id") != current(r).ID || digest(str(doc, "markdown")) != digest(str(preview, "answer")) {
			apiError(w, 409, "이전 답변 초안의 내용·공유 범위가 변경됐습니다. 기존 문서에서 변경을 확인한 뒤 관리 질문에 직접 연결하세요")
			return
		}
		jsonResponse(w, 200, doc)
		return
	}
	if !errors.Is(e, pgx.ErrNoRows) {
		respond(w, nil, e)
		return
	}
	_ = tx.Rollback(r.Context())
	title := []rune(str(preview, "question"))
	title = title[:min(100, len(title))]
	body := jsonValue(map[string]any{"workspace_id": preview["workspace_id"], "title": string(title), "markdown": preview["answer"], "visibility": "private"})
	request := r.Clone(context.WithValue(r.Context(), questionDraftKey{}, questionDraftIntent{r.PathValue("id"), in.Hash}))
	request.Body = io.NopCloser(bytes.NewReader(body))
	request.ContentLength = int64(len(body))
	s.createDocument(w, request)
}
func (s *Server) validateQuestionDraftTx(r *http.Request, tx pgx.Tx, wid, markdown string) error {
	intent, ok := r.Context().Value(questionDraftKey{}).(questionDraftIntent)
	if !ok {
		return nil
	}
	if _, e := tx.Exec(r.Context(), `SET LOCAL statement_timeout='8s'; SET LOCAL lock_timeout='3s'`); e != nil {
		return e
	}
	if _, e := tx.Exec(r.Context(), `SELECT pg_advisory_xact_lock(hashtextextended($1,816))`, current(r).ID+":"+intent.MessageID); e != nil {
		return e
	}
	var exists bool
	if e := tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM knowledge_question_draft_receipts WHERE owner_id=$1 AND message_id=$2)`, current(r).ID, intent.MessageID).Scan(&exists); e != nil {
		return e
	}
	if exists {
		return errors.New("같은 질문의 비공개 초안이 다른 요청에서 저장되었습니다. 다시 시도하면 기존 초안을 엽니다")
	}
	preview, e := s.personalQuestionTx(r, tx, intent.MessageID)
	if e != nil {
		return e
	}
	if str(preview, "workspace_id") != wid || str(preview, "preview_hash") != intent.Hash || str(preview, "answer") != markdown {
		return errors.New("개인 기록과 근거가 변경되었습니다. 현재 미리보기로 다시 정리하세요")
	}
	return s.knowledgeActorTx(r, tx, wid, "document:read", "document:write")
}
func (s *Server) finishQuestionDraftTx(r *http.Request, tx pgx.Tx, wid, id string) error {
	intent, ok := r.Context().Value(questionDraftKey{}).(questionDraftIntent)
	if !ok {
		return nil
	}
	if e := s.knowledgeActorTx(r, tx, wid, "document:read", "document:write"); e != nil {
		return e
	}
	var allowed bool
	if e := tx.QueryRow(r.Context(), `SELECT madi_ai_conversation_allowed($1,conversation_id) FROM ai_messages WHERE id=$2`, current(r).ID, intent.MessageID).Scan(&allowed); e != nil {
		return e
	}
	if !allowed {
		return errors.New("개인 기록과 근거 접근 권한이 변경되었습니다")
	}
	_, e := tx.Exec(r.Context(), `INSERT INTO knowledge_question_draft_receipts(owner_id,message_id,document_id,preview_hash) VALUES($1,$2,$3,$4)`, current(r).ID, intent.MessageID, id, intent.Hash)
	return e
}
