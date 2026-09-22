package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
)

type aiSelectionTicket struct {
	Kind        string `json:"kind"`
	UserID      string `json:"user_id"`
	WorkspaceID string `json:"workspace_id"`
	DocumentID  string `json:"document_id"`
	Version     int    `json:"version"`
	Start       int    `json:"start"`
	End         int    `json:"end"`
	SourceHash  string `json:"source_hash"`
	AnswerHash  string `json:"answer_hash"`
	Provider    string `json:"provider"`
	Expires     int64  `json:"expires"`
}
type aiSelectionCreateKey struct{}

func (s *Server) sealAISelection(r *http.Request, source aiSelectionSource, start, end int, provider, output string) (string, error) {
	if !personalAIHistory(current(r)) {
		return "", nil
	}
	ticket := aiSelectionTicket{"madi-ai-selection-v1", current(r).ID, source.WorkspaceID, source.ID, source.Version, start, end, digest(source.Markdown[start:end]), digest(output), provider, time.Now().Add(30 * time.Minute).Unix()}
	return s.encrypt(string(jsonValue(ticket)))
}

func (s *Server) createAISelectionDraft(w http.ResponseWriter, r *http.Request) {
	p := current(r)
	if !personalAIHistory(p) || !hasIntegrationScope(p, "document:write") || !hasIntegrationScope(p, "document:read") || !hasIntegrationScope(p, "ai:execute") {
		apiError(w, 403, "개인 브라우저의 문서 작성·조회와 AI 실행 권한이 필요합니다")
		return
	}
	var in struct {
		Ticket   string `json:"ticket"`
		Title    string `json:"title"`
		Markdown string `json:"markdown"`
	}
	if decode(r, &in) != nil || len(in.Ticket) > 8192 || len(in.Markdown) > 65536 {
		apiError(w, 400, "완료된 AI 제안과 저장 요청을 확인하세요")
		return
	}
	plain, e := s.decrypt(in.Ticket)
	var ticket aiSelectionTicket
	if e != nil || json.Unmarshal([]byte(plain), &ticket) != nil || ticket.Kind != "madi-ai-selection-v1" || ticket.UserID != p.ID || ticket.Expires < time.Now().Unix() || !validID(ticket.DocumentID) || !validID(ticket.WorkspaceID) || ticket.Version < 1 || ticket.Start < 0 || ticket.End <= ticket.Start || ticket.End-ticket.Start > aiSelectionMaxBytes || ticket.AnswerHash != digest(in.Markdown) {
		apiError(w, 409, "완료된 제안의 저장 확인이 만료되었거나 변경되었습니다. 원문에서 다시 생성하세요")
		return
	}
	body := jsonValue(map[string]any{"workspace_id": ticket.WorkspaceID, "title": in.Title, "markdown": in.Markdown, "visibility": "private"})
	request := r.Clone(context.WithValue(r.Context(), aiSelectionCreateKey{}, ticket))
	request.Body = io.NopCloser(bytes.NewReader(body))
	request.ContentLength = int64(len(body))
	s.createDocument(w, request)
}

// Only the trusted handler can place this context value. HTTP create fields
// cannot forge it. Locks and validation share the document-create transaction.
func (s *Server) validateAISelectionCreateTx(r *http.Request, tx pgx.Tx, wid, markdown string) error {
	ticket, ok := r.Context().Value(aiSelectionCreateKey{}).(aiSelectionTicket)
	if !ok {
		return nil
	}
	p := current(r)
	changed := errors.New("참조 원문·현재 권한·AI 공급자가 변경되었습니다. 현재 원문에서 다시 생성하세요")
	if !personalAIHistory(p) || ticket.UserID != p.ID || ticket.WorkspaceID != wid || ticket.AnswerHash != digest(markdown) || ticket.Expires < time.Now().Unix() {
		return changed
	}
	var version int
	var fragment []byte
	if e := tx.QueryRow(r.Context(), `SELECT version,substring(convert_to(markdown,'UTF8') from $4 for $5) FROM documents WHERE id=$1 AND workspace_id=$2 AND deleted_at IS NULL AND madi_document_allowed($3,id,false) FOR SHARE`, ticket.DocumentID, wid, p.ID, ticket.Start+1, ticket.End-ticket.Start).Scan(&version, &fragment); e != nil || version != ticket.Version || digest(string(fragment)) != ticket.SourceHash {
		return changed
	}
	if ragActorTx(r.Context(), tx, p, ticket.DocumentID, wid, false) != nil {
		return changed
	}
	var writer bool
	if tx.QueryRow(r.Context(), `SELECT role IN ('owner','admin','editor') FROM workspace_members WHERE workspace_id=$1 AND user_id=$2 FOR SHARE`, wid, p.ID).Scan(&writer) != nil || !writer || p.Role == "viewer" {
		return changed
	}
	cookie, e := r.Cookie("madi_session")
	if e != nil {
		return changed
	}
	var sessionID string
	if tx.QueryRow(r.Context(), `SELECT user_id::text FROM sessions WHERE token_hash=$1 AND user_id=$2 AND expires_at>clock_timestamp() FOR SHARE`, digest(cookie.Value), p.ID).Scan(&sessionID) != nil {
		return changed
	}
	fresh, e := s.ragSettingsTx(r.Context(), tx, wid)
	if e != nil || aiHistoryProvider(fresh) != ticket.Provider {
		return changed
	}
	return nil
}
