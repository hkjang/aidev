package server

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"io"
	"net/http"
	"time"
)

type structuredReviewTicket struct {
	Kind        string            `json:"kind"`
	OwnerID     string            `json:"owner_id"`
	DraftID     string            `json:"draft_id"`
	DatabaseID  string            `json:"database_id"`
	Revision    int               `json:"revision"`
	SourceHash  string            `json:"source_hash"`
	Schema      string            `json:"schema_hash"`
	Destination string            `json:"destination_hash"`
	Fields      []structuredField `json:"fields"`
	Values      map[string]any    `json:"values"`
	Hash        string            `json:"hash"`
	Expires     int64             `json:"expires"`
}
type structuredApplyKey struct{}

func structuredReviewFields(d structuredDraft, input []structuredProviderField) ([]structuredField, map[string]any, error) {
	if len(input) < 1 || len(input) > 32 {
		return nil, nil, approvalProblem(400, "반영할 속성1~32개를 선택하세요")
	}
	props, e := structuredProperties(d.Scope.Properties, d.Ticket.Payload.PropertyIDs)
	if e != nil {
		return nil, nil, e
	}
	allowed := map[string]map[string]any{}
	original := map[string]structuredField{}
	for _, p := range props {
		allowed[str(p, "id")] = p
	}
	for _, f := range d.Ticket.Payload.Fields {
		original[f.PropertyID] = f
	}
	values := map[string]any{}
	fields := []structuredField{}
	for _, f := range input {
		p := allowed[f.PropertyID]
		if _, exists := values[f.PropertyID]; exists || p == nil {
			return nil, nil, approvalProblem(400, "원래 선택된 속성을 중복 없이 검토하세요")
		}
		field := structuredValidateField(p, f, d.Scope.Markdown[d.Ticket.Start:d.Ticket.End], d.Ticket.Start)
		if !field.Valid {
			return nil, nil, approvalProblem(400, "속성 "+str(p, "name")+": "+field.Issue)
		}
		before := original[f.PropertyID]
		field.HumanEdited = !before.Valid || string(jsonValue(before.Value)) != string(jsonValue(field.Value)) || before.ContentHash != field.ContentHash || before.StartByte != field.StartByte || before.EndByte != field.EndByte
		fields = append(fields, field)
		values[f.PropertyID] = field.Value
	}
	if len(jsonValue(fields)) > 256<<10 {
		return nil, nil, approvalProblem(400, "검토한 값과 인용은256KiB 이하여야 합니다")
	}
	return fields, values, nil
}
func (s *Server) previewStructuredRow(w http.ResponseWriter, r *http.Request) {
	if !personalAIHistory(current(r)) {
		apiError(w, 403, "사람의 로그인 화면에서 값과 공유 대상을 확인하세요")
		return
	}
	var in struct {
		Revision int                       `json:"revision"`
		Fields   []structuredProviderField `json:"fields"`
	}
	if decode(r, &in) != nil || in.Revision < 1 {
		apiError(w, 400, "현재 초안과 검토할 속성을 확인하세요")
		return
	}
	tx, e := s.structuredTx(r, true)
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	d, e := s.structuredDraftTx(r, tx, r.PathValue("id"), true)
	if e != nil {
		approvalRespondError(w, e)
		return
	}
	if d.State != "draft" || d.Revision != in.Revision {
		apiError(w, 409, "이미 반영됐거나 초안 개정이 변경되었습니다")
		return
	}
	fields, values, e := structuredReviewFields(d, in.Fields)
	if e != nil {
		approvalRespondError(w, e)
		return
	}
	if e = s.checkEvidenceProtection(r.Context(), tx, current(r), d.Scope.WorkspaceID, fields); e != nil {
		apiError(w, 409, e.Error())
		return
	}
	t := structuredReviewTicket{Kind: "madi-structured-review-v1", OwnerID: current(r).ID, DraftID: d.ID, DatabaseID: d.Scope.DatabaseID, Revision: d.Revision, SourceHash: d.Ticket.SourceHash, Schema: d.Scope.Schema, Destination: d.Scope.Destination, Fields: fields, Values: values, Hash: digest(string(jsonValue(values))), Expires: time.Now().Add(10 * time.Minute).Unix()}
	sealed, e := s.encrypt(string(jsonValue(t)))
	if e != nil {
		respond(w, nil, e)
		return
	}
	if e = s.knowledgeActorTx(r, tx, d.Scope.WorkspaceID, "document:read", "database:read", "database:write"); e != nil {
		apiError(w, 403, e.Error())
		return
	}
	jsonResponse(w, 200, map[string]any{"review_ticket": sealed, "fields": fields, "values": values, "database_id": d.Scope.DatabaseID, "database_name": d.Scope.DatabaseName, "source_version": d.Ticket.Version, "expires_at": time.Unix(t.Expires, 0).UTC(), "automatic_apply": false, "destination_notice": "새 행은 대상 데이터베이스의 현재 열람자에게 공유됩니다. 원문 비공개 권한은 복사된 값에 상속되지 않으며 이미 공유한 사본은 원문 권한 변경으로 회수되지 않습니다."})
}
func structuredReviewValid(t structuredReviewTicket, p *Principal, id string) bool {
	return p != nil && t.Kind == "madi-structured-review-v1" && t.OwnerID == p.ID && t.DraftID == id && validID(id) && validID(t.DatabaseID) && t.Revision > 0 && t.Expires > time.Now().Unix() && len(t.Fields) > 0 && len(t.Fields) <= 32 && len(t.Values) > 0 && len(t.Values) <= 32 && t.Hash == digest(string(jsonValue(t.Values)))
}
func (s *Server) applyStructuredRow(w http.ResponseWriter, r *http.Request) {
	if !personalAIHistory(current(r)) {
		apiError(w, 403, "사람의 로그인 화면에서 새 행 생성을 확인하세요")
		return
	}
	var in struct {
		Ticket  string `json:"review_ticket"`
		Consent bool   `json:"consent"`
	}
	if decode(r, &in) != nil || !in.Consent || len(in.Ticket) > 1<<20 {
		apiError(w, 400, "최종 미리보기와 대상 데이터베이스 공유 동의가 필요합니다")
		return
	}
	plain, e := s.decrypt(in.Ticket)
	var t structuredReviewTicket
	if e != nil || json.Unmarshal([]byte(plain), &t) != nil || !structuredReviewValid(t, current(r), r.PathValue("id")) {
		apiError(w, 409, "최종 확인이 만료됐거나 변경되었습니다. 다시 비교하세요")
		return
	}
	// Replay never creates another row or overwrites a later user's edit.
	tx, e := s.structuredTx(r, true)
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	d, e := s.structuredDraftTx(r, tx, t.DraftID, true)
	if e != nil {
		approvalRespondError(w, e)
		return
	}
	if d.State == "committed" {
		var raw []byte
		var version int
		e = tx.QueryRow(r.Context(), `SELECT values,version FROM database_rows WHERE id=$1 AND database_id=$2 FOR SHARE`, d.RowID, t.DatabaseID).Scan(&raw, &version)
		var values map[string]any
		if e != nil || json.Unmarshal(raw, &values) != nil || version != 1 || d.ValuesHash != t.Hash || digest(string(jsonValue(values))) != t.Hash {
			apiError(w, 409, "이 초안으로 만든 행이 변경·삭제됐습니다. 기존 데이터베이스에서 확인하세요")
			return
		}
		if e = s.knowledgeActorTx(r, tx, d.Scope.WorkspaceID, "document:read", "database:read", "database:write"); e != nil {
			apiError(w, 403, e.Error())
			return
		}
		jsonResponse(w, 200, map[string]any{"id": d.RowID, "database_id": t.DatabaseID, "values": values, "version": version, "replayed": true})
		return
	}
	_ = tx.Rollback(r.Context())
	body := jsonValue(map[string]any{"values": t.Values})
	request := r.Clone(context.WithValue(r.Context(), structuredApplyKey{}, t))
	request.SetPathValue("id", t.DatabaseID)
	request.Body = io.NopCloser(bytes.NewReader(body))
	request.ContentLength = int64(len(body))
	s.createRow(w, request)
}
func (s *Server) prepareStructuredRowTx(r *http.Request, tx pgx.Tx, dbID string, values map[string]any) error {
	t, ok := r.Context().Value(structuredApplyKey{}).(structuredReviewTicket)
	if !ok {
		return nil
	}
	if !personalAIHistory(current(r)) || !structuredReviewValid(t, current(r), t.DraftID) || dbID != t.DatabaseID || digest(string(jsonValue(values))) != t.Hash {
		return errStructuredChanged
	}
	if _, e := tx.Exec(r.Context(), `SET LOCAL lock_timeout='3s'; SET LOCAL statement_timeout='8s'; LOCK TABLE documents,spaces,space_members,document_shares IN SHARE MODE`); e != nil {
		return e
	}
	d, e := s.structuredDraftTx(r, tx, t.DraftID, true)
	if e != nil {
		return e
	}
	if d.State != "draft" || d.Revision != t.Revision || d.Scope.Schema != t.Schema || d.Scope.Destination != t.Destination || d.Ticket.SourceHash != t.SourceHash {
		return errStructuredChanged
	}
	input := []structuredProviderField{}
	for _, f := range t.Fields {
		relative := f.StartByte - d.Ticket.Start
		input = append(input, structuredProviderField{PropertyID: f.PropertyID, Value: f.Value, Quote: f.Quote, StartByte: &relative})
	}
	fields, checked, e := structuredReviewFields(d, input)
	if e != nil || digest(string(jsonValue(checked))) != t.Hash || string(jsonValue(fields)) != string(jsonValue(t.Fields)) {
		return errStructuredChanged
	}
	if e = s.checkEvidenceProtection(r.Context(), tx, current(r), d.Scope.WorkspaceID, fields); e != nil {
		return e
	}
	return s.knowledgeActorTx(r, tx, d.Scope.WorkspaceID, "document:read", "database:read", "database:write")
}
func (s *Server) finishStructuredRowTx(r *http.Request, tx pgx.Tx, dbID, rowID string) error {
	t, ok := r.Context().Value(structuredApplyKey{}).(structuredReviewTicket)
	if !ok {
		return nil
	}
	if !structuredReviewValid(t, current(r), t.DraftID) || t.DatabaseID != dbID {
		return errStructuredChanged
	}
	var wid string
	if e := tx.QueryRow(r.Context(), `SELECT workspace_id::text FROM databases WHERE id=$1 AND (space_id IS NULL OR madi_space_allowed($2,space_id,true))`, dbID, current(r).ID).Scan(&wid); e != nil {
		return errStructuredChanged
	}
	if e := s.knowledgeActorTx(r, tx, wid, "document:read", "database:read", "database:write"); e != nil {
		return e
	}
	var docID string
	if e := tx.QueryRow(r.Context(), `SELECT document_id::text FROM knowledge_structured_drafts WHERE id=$1`, t.DraftID).Scan(&docID); e != nil {
		return errStructuredChanged
	}
	v, e := s.structuredScopeTx(r, tx, docID, dbID, true)
	if e != nil || v.Destination != t.Destination || v.Schema != t.Schema || digest(v.Markdown) != t.SourceHash || !structuredReviewValid(t, current(r), t.DraftID) {
		return errStructuredChanged
	}
	cipher, e := s.encrypt(string(jsonValue(t.Fields)))
	if e != nil {
		return e
	}
	tag, e := tx.Exec(r.Context(), `UPDATE knowledge_structured_drafts SET state='committed',revision=revision+1,row_id=$3,values_hash=$4,review_ciphertext=$5,committed_at=now() WHERE id=$1 AND owner_id=$2 AND state='draft' AND revision=$6 AND expires_at>clock_timestamp()`, t.DraftID, current(r).ID, rowID, t.Hash, cipher, t.Revision)
	if e != nil {
		return e
	}
	if tag.RowsAffected() != 1 {
		return errStructuredChanged
	}
	if e = s.knowledgeActorTx(r, tx, wid, "document:read", "database:read", "database:write"); e != nil {
		return e
	}
	if !structuredReviewValid(t, current(r), t.DraftID) {
		return errStructuredChanged
	}
	return nil
}
