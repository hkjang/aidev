package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
)

type structuredDraft struct {
	Ticket                       structuredTicket
	Scope                        structuredScope
	ID, State, RowID, ValuesHash string
	Revision                     int
	Expires, Created             time.Time
	Committed                    *time.Time
	Review                       []structuredField
	Fresh                        bool
}

func (s *Server) saveStructuredDraft(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Ticket  string `json:"ticket"`
		Consent bool   `json:"consent"`
	}
	if !personalAIHistory(current(r)) {
		apiError(w, 403, "개인 로그인 화면에서 초안을 보관하세요")
		return
	}
	if decode(r, &in) != nil || !in.Consent || len(in.Ticket) > 1<<20 {
		apiError(w, 400, "완료된 제안의 개인 보관 동의가 필요합니다")
		return
	}
	plain, e := s.decrypt(in.Ticket)
	var t structuredTicket
	if e != nil || json.Unmarshal([]byte(plain), &t) != nil || !structuredTicketValid(t, current(r)) {
		apiError(w, 409, "완료된 제안이 만료됐거나 변경됐습니다. 다시 생성하세요")
		return
	}
	tx, e := s.structuredTx(r, true)
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	v, e := s.structuredScopeTx(r, tx, t.DocumentID, t.DatabaseID, true)
	if e != nil {
		approvalRespondError(w, e)
		return
	}
	provider, e := s.structuredProviderTx(r, tx, v)
	if e != nil || !structuredMatches(t, v) || str(provider, "fingerprint") != t.Provider {
		apiError(w, 409, errStructuredChanged.Error())
		return
	}
	if e = s.checkEvidenceProtection(r.Context(), tx, current(r), v.WorkspaceID, t.Payload); e != nil {
		apiError(w, 409, e.Error())
		return
	}
	var count int
	// Per-owner transaction lock makes the 100 live-draft quota concurrent-safe.
	if _, e = tx.Exec(r.Context(), `SELECT pg_advisory_xact_lock(hashtextextended($1,817))`, current(r).ID); e == nil {
		e = tx.QueryRow(r.Context(), `SELECT count(*) FROM knowledge_structured_drafts WHERE owner_id=$1 AND state='draft' AND expires_at>clock_timestamp() AND id<>$2`, current(r).ID, t.ID).Scan(&count)
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	if count >= 100 {
		apiError(w, 429, "개인 구조화 초안은100개까지 보관할 수 있습니다. 사용하지 않는 초안을 정리하세요")
		return
	}
	var existingState string
	e = tx.QueryRow(r.Context(), `SELECT state FROM knowledge_structured_drafts WHERE id=$1`, t.ID).Scan(&existingState)
	if e != nil && e != pgx.ErrNoRows {
		respond(w, nil, e)
		return
	}
	if e == nil && !oneOf(existingState, "draft", "committed") {
		apiError(w, 409, "이미 폐기·만료된 제안입니다. 새 제안에서 시작하세요")
		return
	}
	_, e = tx.Exec(r.Context(), `INSERT INTO knowledge_structured_drafts(id,owner_id,document_id,database_id,source_version,source_hash,start_byte,end_byte,schema_hash,destination_hash,provider_hash,ciphertext,expires_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,now()+interval '24 hours') ON CONFLICT(id) DO NOTHING`, t.ID, t.OwnerID, t.DocumentID, t.DatabaseID, t.Version, t.SourceHash, t.Start, t.End, t.Schema, t.Destination, t.Provider, in.Ticket)
	if e == nil {
		e = s.knowledgeActorTx(r, tx, v.WorkspaceID, "document:read", "database:read", "database:write", "ai:execute")
	}
	if e == nil && !structuredTicketValid(t, current(r)) {
		e = errStructuredChanged
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		approvalRespondError(w, e)
		return
	}
	s.audit(r, "STRUCTURED_DRAFT_CREATE", t.ID, map[string]any{"property_count": len(t.Payload.Fields), "automatic_apply": false})
	jsonResponse(w, 201, map[string]any{"id": t.ID, "automatic_apply": false})
}

func (s *Server) structuredDraftTx(r *http.Request, tx pgx.Tx, id string, write bool) (structuredDraft, error) {
	return s.structuredDraftLockTx(r, tx, id, write, write)
}
func (s *Server) structuredDraftLockTx(r *http.Request, tx pgx.Tx, id string, write, exclusive bool) (structuredDraft, error) {
	d := structuredDraft{ID: id}
	missing := approvalProblem(404, "현재 접근할 수 있는 본인의 구조화 초안을 확인하세요")
	if !validID(id) {
		return d, missing
	}
	var docID, dbID string
	if e := tx.QueryRow(r.Context(), `SELECT document_id::text,database_id::text FROM knowledge_structured_drafts WHERE id=$1 AND owner_id=$2 AND (state='committed' OR (state='draft' AND expires_at>clock_timestamp()))`, id, current(r).ID).Scan(&docID, &dbID); e != nil {
		return d, missing
	}
	v, e := s.structuredScopeTx(r, tx, docID, dbID, write)
	if e != nil {
		return d, e
	}
	d.Scope = v
	var cipher, review string
	query := `SELECT ciphertext,state,revision,expires_at,created_at,committed_at,coalesce(row_id::text,''),values_hash,review_ciphertext FROM knowledge_structured_drafts WHERE id=$1 AND owner_id=$2 AND (state='committed' OR (state='draft' AND expires_at>clock_timestamp())) FOR SHARE`
	if exclusive {
		query = query[:len(query)-len("FOR SHARE")] + "FOR UPDATE"
	}
	if e = tx.QueryRow(r.Context(), query, id, current(r).ID).Scan(&cipher, &d.State, &d.Revision, &d.Expires, &d.Created, &d.Committed, &d.RowID, &d.ValuesHash, &review); e != nil {
		return d, missing
	}
	if d.State != "committed" && !d.Expires.After(time.Now()) {
		return d, missing
	}
	plain, e := s.decrypt(cipher)
	if e != nil || json.Unmarshal([]byte(plain), &d.Ticket) != nil || d.Ticket.ID != id || d.Ticket.OwnerID != current(r).ID || d.Ticket.DocumentID != docID || d.Ticket.DatabaseID != dbID {
		return d, missing
	}
	if review != "" {
		plain, e = s.decrypt(review)
		if e != nil || json.Unmarshal([]byte(plain), &d.Review) != nil {
			return d, missing
		}
	}
	if e = s.checkEvidenceProtection(r.Context(), tx, current(r), v.WorkspaceID, map[string]any{"proposal": d.Ticket.Payload, "review": d.Review, "title": v.Title, "database": v.DatabaseName}); e != nil {
		return d, approvalProblem(403, e.Error())
	}
	d.Fresh = structuredMatches(d.Ticket, v)
	if write && !d.Fresh {
		return d, errStructuredChanged
	}
	return d, nil
}
func structuredView(d structuredDraft) map[string]any {
	props := []map[string]any{}
	for _, p := range d.Scope.Properties {
		if structuredPropertySupported(p) {
			clean, e := structuredProperties(d.Scope.Properties, []string{str(p, "id")})
			if e == nil {
				props = append(props, clean[0])
			}
		}
	}
	return map[string]any{"id": d.ID, "document_id": d.Scope.DocumentID, "database_id": d.Scope.DatabaseID, "workspace_id": d.Scope.WorkspaceID, "title": d.Scope.Title, "database_name": d.Scope.DatabaseName, "source_version": d.Ticket.Version, "current_version": d.Scope.Version, "source_hash": d.Ticket.SourceHash, "start_byte": d.Ticket.Start, "end_byte": d.Ticket.End, "fields": d.Ticket.Payload.Fields, "properties": props, "model": d.Ticket.Model, "state": d.State, "revision": d.Revision, "fresh": d.Fresh, "expires_at": d.Expires, "created_at": d.Created, "committed_at": d.Committed, "row_id": d.RowID, "reviewed_fields": d.Review, "automatic_apply": false, "integrity_notice": "원문 인용의 위치·해시는 값의 의미나 사실성을 보증하지 않습니다. 반영 전에 직접 확인하세요."}
}
func (s *Server) getStructuredDraft(w http.ResponseWriter, r *http.Request) {
	tx, e := s.structuredTx(r, false)
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	d, e := s.structuredDraftTx(r, tx, r.PathValue("id"), false)
	if e != nil {
		approvalRespondError(w, e)
		return
	}
	if e = s.knowledgeActorTx(r, tx, d.Scope.WorkspaceID, "document:read", "database:read"); e != nil {
		apiError(w, 403, e.Error())
		return
	}
	if e = structuredCurrentACLTx(r, tx, d.Scope); e != nil {
		approvalRespondError(w, e)
		return
	}
	if d.State != "committed" && !d.Expires.After(time.Now()) {
		apiError(w, 404, "개인 초안 보관 기간이 만료되었습니다")
		return
	}
	jsonResponse(w, 200, structuredView(d))
}
func (s *Server) listStructuredDrafts(w http.ResponseWriter, r *http.Request) {
	wid := r.URL.Query().Get("workspace_id")
	after := r.URL.Query().Get("after")
	if !validID(wid) || after != "" && !validID(after) {
		apiError(w, 400, "워크스페이스와 목록 위치를 확인하세요")
		return
	}
	tx, e := s.structuredTx(r, false)
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	if e = s.knowledgeActorTx(r, tx, wid, "document:read", "database:read"); e != nil {
		apiError(w, 403, e.Error())
		return
	}
	rows, e := tx.Query(r.Context(), `SELECT x.id::text FROM knowledge_structured_drafts x JOIN documents d ON d.id=x.document_id JOIN databases b ON b.id=x.database_id WHERE x.owner_id=$1 AND d.workspace_id=$2 AND d.deleted_at IS NULL AND madi_document_allowed($1,d.id,false) AND (b.space_id IS NULL OR madi_space_allowed($1,b.space_id,false)) AND (x.state='committed' OR (x.state='draft' AND x.expires_at>clock_timestamp())) AND ($3='' OR x.id>NULLIF($3,'')::uuid) ORDER BY x.id LIMIT 21`, current(r).ID, wid, after)
	if e != nil {
		respond(w, nil, e)
		return
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
		respond(w, nil, e)
		return
	}
	next := ""
	if len(ids) > 20 {
		ids = ids[:20]
		next = ids[19]
	}
	items := []map[string]any{}
	for _, id := range ids {
		d, e := s.structuredDraftTx(r, tx, id, false)
		if e != nil {
			continue
		}
		items = append(items, map[string]any{"id": id, "document_id": d.Scope.DocumentID, "title": d.Scope.Title, "database_id": d.Scope.DatabaseID, "database_name": d.Scope.DatabaseName, "state": d.State, "fresh": d.Fresh, "revision": d.Revision, "created_at": d.Created})
	}
	if e = s.knowledgeActorTx(r, tx, wid, "document:read", "database:read"); e != nil {
		apiError(w, 403, e.Error())
		return
	}
	visible := []map[string]any{}
	for _, item := range items {
		v := structuredScope{DocumentID: str(item, "document_id"), DatabaseID: str(item, "database_id")}
		if structuredCurrentACLTx(r, tx, v) == nil {
			visible = append(visible, item)
		} else if str(item, "id") == next {
			next = ""
		}
	}
	jsonResponse(w, 200, map[string]any{"items": visible, "next_after": next})
}
func (s *Server) deleteStructuredDraft(w http.ResponseWriter, r *http.Request) {
	if !personalAIHistory(current(r)) {
		apiError(w, 403, "개인 로그인 화면에서 삭제하세요")
		return
	}
	var in struct {
		Revision int  `json:"revision"`
		Consent  bool `json:"consent"`
	}
	if decode(r, &in) != nil || !in.Consent || in.Revision < 1 {
		apiError(w, 400, "현재 개정 번호와 근거 사본 삭제 동의를 확인하세요")
		return
	}
	tx, e := s.structuredTx(r, true)
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	d, e := s.structuredDraftLockTx(r, tx, r.PathValue("id"), false, true)
	if e != nil {
		approvalRespondError(w, e)
		return
	}
	var hold bool
	if e = tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM knowledge_document_meta WHERE document_id=$1 AND (legal_hold OR retain_until>now()))`, d.Scope.DocumentID).Scan(&hold); e != nil {
		respond(w, nil, e)
		return
	}
	if hold {
		apiError(w, 409, "원문의 법적 보존·보존기간 정책에 따라 근거 사본을 유지해야 합니다")
		return
	}
	// A content-free one-hour tombstone outlives every issued generation ticket
	// (30m), so deleting provenance cannot revive the same proposal into a new row.
	tag, e := tx.Exec(r.Context(), `UPDATE knowledge_structured_drafts SET state='discarded',revision=revision+1,ciphertext='',review_ciphertext='',row_id=NULL,values_hash='',committed_at=NULL,expires_at=now()+interval '1 hour' WHERE id=$1 AND owner_id=$2 AND revision=$3`, d.ID, current(r).ID, in.Revision)
	if e == nil && tag.RowsAffected() != 1 {
		e = approvalProblem(409, "초안 개정이 변경되었습니다")
	}
	if e == nil {
		e = s.knowledgeActorTx(r, tx, d.Scope.WorkspaceID, "document:read", "database:read")
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		approvalRespondError(w, e)
		return
	}
	s.audit(r, "STRUCTURED_DRAFT_DELETE", d.ID, map[string]any{"row_deleted": false})
	jsonResponse(w, 200, map[string]any{"deleted": true, "row_deleted": false})
}
