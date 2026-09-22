package server

import (
	"context"
	_ "embed"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"
)

//go:embed document_split.sql
var documentSplitSchema string

type documentSplitTicket struct {
	Kind       string `json:"kind"`
	Actor      string `json:"actor"`
	SourceID   string `json:"source_id"`
	ChildID    string `json:"child_id"`
	Title      string `json:"title"`
	Version    int    `json:"version"`
	Start      int    `json:"start"`
	End        int    `json:"end"`
	SourceHash string `json:"source_hash"`
	Expires    int64  `json:"expires"`
}
type documentSplitContextKey struct{}
type documentSplitIntent struct {
	Ticket      documentSplitTicket
	RequestID   string
	PayloadHash string
	Selected    string
	Replacement string
}

func (s *Server) migrateDocumentSplit(ctx context.Context) error {
	_, e := s.DB.Exec(ctx, documentSplitSchema)
	return e
}
func (s *Server) registerDocumentSplit() {
	s.handle("POST /api/v1/documents/{id}/split-preview", s.previewDocumentSplit)
	s.handle("POST /api/v1/documents/{id}/split", s.splitDocument)
}

func (s *Server) previewDocumentSplit(w http.ResponseWriter, r *http.Request) {
	p, id := current(r), r.PathValue("id")
	if !personalAccessRequest(p) || !s.canDocument(r.Context(), p, id, true) {
		apiError(w, 403, "개인 브라우저의 현재 문서 작성 권한이 필요합니다")
		return
	}
	var in struct {
		Version  int    `json:"expected_version"`
		Start    int    `json:"start_byte"`
		End      int    `json:"end_byte"`
		Selected string `json:"selected_text"`
		Title    string `json:"title"`
	}
	if decode(r, &in) != nil || in.Version < 1 || strings.TrimSpace(in.Title) == "" || len(in.Title) > 500 || len(in.Selected) > splitSelectionMaxBytes {
		apiError(w, 400, "제목·버전과 최대256KiB의 완전한 블록을 확인하세요")
		return
	}
	doc, e := s.document(r, id)
	if e != nil {
		respond(w, nil, e)
		return
	}
	if doc["deleted_at"] != nil || number(doc, "version", 0) != in.Version {
		apiError(w, 409, "문서 버전이 변경되었습니다. 저장된 원문에서 다시 선택하세요")
		return
	}
	if e = validateDocumentSplitRange(str(doc, "markdown"), in.Start, in.End, in.Selected); e != nil {
		apiError(w, 400, e.Error())
		return
	}
	for _, task := range indexMarkdown(str(doc, "markdown")).Tasks {
		if task.Start >= in.Start && task.Start < in.End && task.Ambiguous {
			apiError(w, 400, "중복된 할 일 ID는 분리 전에 정리하세요")
			return
		}
	}
	ticket := documentSplitTicket{"madi-document-split-v1", p.ID, id, newID(), strings.TrimSpace(in.Title), in.Version, in.Start, in.End, digest(str(doc, "markdown")), time.Now().Add(10 * time.Minute).Unix()}
	sealed, e := s.encrypt(string(jsonValue(ticket)))
	if e != nil {
		respond(w, nil, e)
		return
	}
	jsonResponse(w, 200, map[string]any{"ticket": sealed, "source_id": id, "source_version": in.Version, "child_id": ticket.ChildID, "child_title": ticket.Title, "selected_markdown": in.Selected, "replacement_markdown": documentSplitReplacement("", 0, 0, ticket.ChildID, ticket.Title), "notice": "새 문서는 원본의 하위 문서가 되고 원본·공간의 현재 권한을 상속합니다. 나만 보기 원본에서 분리한 내용이 공개되는 것은 아닙니다. 원본에는 새 문서 참조를 남기며 두 변경은 한 번에 저장됩니다.", "expires_in": 600})
}

func (s *Server) documentSplitReplay(w http.ResponseWriter, r *http.Request, requestID, hash string, replayed bool) bool {
	var source, child, savedHash string
	var sourceVersion, childVersion int
	e := s.DB.QueryRow(r.Context(), "SELECT source_id::text,child_id::text,source_version,child_version,payload_hash FROM document_split_receipts WHERE user_id=$1 AND request_id=$2", current(r).ID, requestID).Scan(&source, &child, &sourceVersion, &childVersion, &savedHash)
	if e != nil {
		return false
	}
	if source != r.PathValue("id") || savedHash != hash {
		apiError(w, 409, "같은 요청 ID의 분리 내용이 다릅니다")
		return true
	}
	if !s.canDocument(r.Context(), current(r), source, true) || !s.canDocument(r.Context(), current(r), child, false) {
		apiError(w, 403, "분리한 문서의 현재 권한이 변경되었습니다")
		return true
	}
	a, e := s.document(r, source)
	if e != nil {
		respond(w, nil, e)
		return true
	}
	b, e := s.document(r, child)
	if e != nil {
		respond(w, nil, e)
		return true
	}
	if a["deleted_at"] != nil || b["deleted_at"] != nil {
		apiError(w, 410, "분리한 문서가 휴지통에 있습니다. 같은 요청으로 다시 생성하지 않습니다")
		return true
	}
	jsonResponse(w, 200, map[string]any{"source": a, "child": b, "receipt": map[string]any{"request_id": requestID, "source_version": sourceVersion, "child_version": childVersion}, "replayed": replayed})
	return true
}

func (s *Server) splitDocument(w http.ResponseWriter, r *http.Request) {
	p, id := current(r), r.PathValue("id")
	if !personalAccessRequest(p) {
		apiError(w, 403, "개인 브라우저에서 분리할 블록을 확인하세요")
		return
	}
	var in struct {
		Ticket    string `json:"ticket"`
		RequestID string `json:"client_request_id"`
		Consent   bool   `json:"consent"`
	}
	if decode(r, &in) != nil || !validID(id) || !validID(in.RequestID) || !in.Consent || len(in.Ticket) > 8192 {
		apiError(w, 400, "분리 미리보기·요청 ID와 명시 동의를 확인하세요")
		return
	}
	hash := digest(in.Ticket)
	if s.documentSplitReplay(w, r, in.RequestID, hash, true) {
		return
	}
	raw, e := s.decrypt(in.Ticket)
	var ticket documentSplitTicket
	if e != nil || json.Unmarshal([]byte(raw), &ticket) != nil || ticket.Kind != "madi-document-split-v1" || ticket.Actor != p.ID || ticket.SourceID != id || !validID(ticket.ChildID) || ticket.Version < 1 || ticket.Expires <= time.Now().Unix() {
		apiError(w, 409, "문서 분리 확인이 만료되었거나 변경되었습니다")
		return
	}
	if !s.canDocument(r.Context(), p, id, true) {
		apiError(w, 403, "현재 문서 작성 권한이 필요합니다")
		return
	}
	doc, e := s.document(r, id)
	if e != nil {
		respond(w, nil, e)
		return
	}
	md := str(doc, "markdown")
	if doc["deleted_at"] != nil || number(doc, "version", 0) != ticket.Version || digest(md) != ticket.SourceHash || ticket.Start < 0 || ticket.End <= ticket.Start || ticket.End > len(md) {
		if !s.documentSplitReplay(w, r, in.RequestID, hash, true) {
			apiError(w, 409, "분리 기준 원문이 변경되었습니다")
		}
		return
	}
	selected := md[ticket.Start:ticket.End]
	if e = validateDocumentSplitRange(md, ticket.Start, ticket.End, selected); e != nil {
		apiError(w, 400, e.Error())
		return
	}
	replacement := documentSplitReplacement(md, ticket.Start, ticket.End, ticket.ChildID, ticket.Title)
	intent := documentSplitIntent{ticket, in.RequestID, hash, selected, replacement}
	request := r.Clone(context.WithValue(r.Context(), documentSplitContextKey{}, intent))
	recorder := httptest.NewRecorder()
	s.saveDocument(recorder, request, id, map[string]any{"version": ticket.Version, "markdown": replacement})
	if s.documentSplitReplay(w, r, in.RequestID, hash, recorder.Code != 200) {
		return
	}
	if recorder.Code >= 200 && recorder.Code < 300 {
		apiError(w, 503, "저장 결과를 다시 확인하지 못했습니다. 같은 요청 ID로 결과를 확인하세요")
		return
	}
	for name, values := range recorder.Header() {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}
	w.WriteHeader(recorder.Code)
	_, _ = w.Write(recorder.Body.Bytes())
}
