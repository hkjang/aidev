package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type documentQueryTicket struct {
	Format                                                                  string
	ActorID, TokenID, Session, ParentID, WorkspaceID, ParentHash, QueryHash string
	ParentVersion                                                           int
	ProtectionRevision                                                      int64
	Body, Tasks                                                             bool
	Expires                                                                 time.Time
	Sources                                                                 []documentQueryReference
	Relations                                                               []documentQueryRelationRef
}

func documentQuerySession(r *http.Request) string {
	if current(r).TokenID != "" {
		return digest(r.Header.Get("Authorization"))
	}
	c, e := r.Cookie("madi_session")
	if e != nil {
		return ""
	}
	return digest(c.Value)
}
func documentQueryDecode(r *http.Request, out any, limit int64) error {
	raw, e := io.ReadAll(io.LimitReader(r.Body, limit+1))
	if e != nil || int64(len(raw)) > limit || documentQueryCheckJSON(raw) != nil {
		return errors.New("조회 요청의 JSON 형식과 크기를 확인하세요")
	}
	d := json.NewDecoder(strings.NewReader(string(raw)))
	d.DisallowUnknownFields()
	return d.Decode(out)
}
func documentQueryError(w http.ResponseWriter, e error) {
	var pgError *pgconn.PgError
	if errors.Is(e, context.DeadlineExceeded) || errors.Is(e, context.Canceled) || errors.As(e, &pgError) && pgError.Code == "57014" {
		apiError(w, 408, "조회 시간 예산을 초과했거나 취소되었습니다. 공간·문서 범위를 좁혀 다시 조회하세요")
		return
	}
	approvalRespondError(w, e)
}
func (s *Server) documentQueryBegin(w http.ResponseWriter, r *http.Request) (*http.Request, pgx.Tx, func(), bool) {
	select {
	case s.documentQuerySlots <- struct{}{}:
	default:
		w.Header().Set("Retry-After", "2")
		apiError(w, 429, "동시에 실행 중인 문서 조회가 많습니다. 잠시 후 다시 시도하세요")
		return r, nil, func() {}, false
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	r = r.WithContext(ctx)
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		cancel()
		<-s.documentQuerySlots
		documentQueryError(w, e)
		return r, nil, func() {}, false
	}
	cleanup := func() {
		rollbackCtx, stop := context.WithTimeout(context.Background(), time.Second)
		defer stop()
		tx.Rollback(rollbackCtx)
		cancel()
		<-s.documentQuerySlots
	}
	if _, e = tx.Exec(ctx, `SET LOCAL statement_timeout='1900ms'`); e != nil {
		cleanup()
		documentQueryError(w, e)
		return r, nil, func() {}, false
	}
	return r, tx, cleanup, true
}
func (s *Server) documentQueryActorTx(r *http.Request, tx pgx.Tx, parent documentQueryDocument) error {
	if e := s.knowledgeActorTx(r, tx, parent.WorkspaceID, "document:read"); e != nil {
		return approvalProblem(403, "현재 계정·세션·키 또는 문서 조회 권한이 변경되었습니다")
	}
	var allowed bool
	if tx.QueryRow(r.Context(), `SELECT madi_document_allowed($1,$2,false) AND madi_feature_allowed($1,$3,'document-queries') AND EXISTS(SELECT 1 FROM documents WHERE id=$2 AND deleted_at IS NULL AND version=$4 AND encode(sha256(convert_to(markdown,'UTF8')),'hex')=$5)`, current(r).ID, parent.ID, parent.WorkspaceID, parent.Version, parent.Hash).Scan(&allowed) != nil || !allowed {
		return approvalProblem(409, "조회 문서·권한·기능 정책이 변경되었습니다. 다시 불러오세요")
	}
	return nil
}
func (s *Server) getDocumentQueries(w http.ResponseWriter, r *http.Request) {
	r, tx, done, ok := s.documentQueryBegin(w, r)
	if !ok {
		return
	}
	defer done()
	parent, e := documentQueryParentTx(r, tx)
	if e != nil {
		documentQueryError(w, e)
		return
	}
	items, limited := documentQueryFences(parent.Markdown)
	if e = s.checkEvidenceProtection(r.Context(), tx, current(r), parent.WorkspaceID, items); e == nil {
		e = s.documentQueryActorTx(r, tx, parent)
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		documentQueryError(w, e)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	jsonResponse(w, 200, map[string]any{"document_id": parent.ID, "document_version": parent.Version, "queries": items, "truncated": limited, "notice": "원문에 저장된 조회 정의만 실행할 수 있습니다. 결과는 현재 사용자에게만 표시되며 원문·공개 링크·내보내기·승인 스냅샷에 저장하지 않습니다."})
}
func (s *Server) executeDocumentQuery(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Version    int                        `json:"document_version"`
		Hash       string                     `json:"query_hash"`
		Parameters map[string]json.RawMessage `json:"parameters"`
	}
	if documentQueryDecode(r, &in, 16<<10) != nil || in.Version < 1 || len(in.Hash) != 64 {
		apiError(w, 400, "저장된 문서 버전·조회 해시·매개변수를 확인하세요")
		return
	}
	r, tx, done, ok := s.documentQueryBegin(w, r)
	if !ok {
		return
	}
	defer done()
	started := time.Now()
	parent, e := documentQueryParentTx(r, tx)
	if e != nil {
		documentQueryError(w, e)
		return
	}
	if parent.Version != in.Version {
		apiError(w, 409, "조회 정의가 저장된 문서 버전이 변경되었습니다. 다시 불러오세요")
		return
	}
	fences, _ := documentQueryFences(parent.Markdown)
	var selected *documentQueryDefinition
	for _, f := range fences {
		if f.Hash == in.Hash && f.Definition != nil {
			selected = f.Definition
			break
		}
	}
	if selected == nil {
		apiError(w, 409, "현재 원문에 같은 유효한 조회 정의가 없습니다")
		return
	}
	params, e := documentQueryParameters(*selected, in.Parameters)
	if e != nil {
		apiError(w, 400, e.Error())
		return
	}
	if e = s.documentQueryActorTx(r, tx, parent); e != nil {
		documentQueryError(w, e)
		return
	}
	material, e := s.documentQueryMaterialTx(r, tx, parent, *selected, params)
	if e != nil {
		documentQueryError(w, e)
		return
	}
	ticket := documentQueryTicket{Format: "madi-query-result-v1", ActorID: current(r).ID, TokenID: current(r).TokenID, Session: documentQuerySession(r), ParentID: parent.ID, WorkspaceID: parent.WorkspaceID, ParentVersion: parent.Version, ParentHash: parent.Hash, QueryHash: in.Hash, Body: documentQueryNeedsBody(*selected), Tasks: selected.Source == "tasks", Expires: time.Now().Add(10 * time.Minute), Sources: []documentQueryReference{}, Relations: []documentQueryRelationRef{}}
	used := map[string]bool{}
	result := []documentQueryRow{}
	for _, row := range material.Rows {
		for _, id := range row.Sources {
			used[id] = true
		}
		values := map[string]any{}
		for _, c := range selected.Columns {
			values[c.Field] = row.Values[c.Field]
		}
		row.Values = values
		result = append(result, row)
	}
	for id := range used {
		d, exists := material.Documents[id]
		if !exists {
			apiError(w, 409, "조회 경로의 접근 상태가 변경되었습니다")
			return
		}
		ticket.Sources = append(ticket.Sources, documentQueryReference{ID: id, Hash: d.Hash})
	}
	if selected.Source == "relations" {
		for _, rel := range material.Relations {
			if used[rel.SourceID] && used[rel.TargetID] {
				ticket.Relations = append(ticket.Relations, rel)
			}
		}
	}
	raw, e := json.Marshal(result)
	if e != nil || len(raw) > documentQueryResultBytes {
		apiError(w, 413, "조회 결과 크기 한도를 초과했습니다. 열과 행 수를 줄이세요")
		return
	}
	if e = s.checkEvidenceProtection(r.Context(), tx, current(r), parent.WorkspaceID, map[string]any{"columns": selected.Columns, "rows": result}); e != nil {
		apiError(w, 422, "현재 보호 정책에서 조회 결과를 표시할 수 없습니다")
		return
	}
	if e = tx.QueryRow(r.Context(), `SELECT revision FROM protection_settings WHERE id=1`).Scan(&ticket.ProtectionRevision); e == nil {
		e = s.validateDocumentQueryTicketTx(r, tx, ticket)
	}
	sealed := ""
	if e == nil {
		raw, e = json.Marshal(ticket)
	}
	if e == nil {
		sealed, e = s.encrypt(string(raw))
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		documentQueryError(w, e)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	jsonResponse(w, 200, map[string]any{"document_version": parent.Version, "query_hash": in.Hash, "columns": selected.Columns, "rows": result, "diagnostics": material.Diagnostics, "validation_token": sealed, "expires_at": ticket.Expires, "executed_at": started.UTC(), "duration_ms": time.Since(started).Milliseconds()})
}
func (s *Server) validateDocumentQueryTicketTx(r *http.Request, tx pgx.Tx, ticket documentQueryTicket) error {
	changed := approvalProblem(409, "조회 결과의 권한·버전·정책 또는 유효기간이 변경되었습니다. 결과를 지우고 다시 조회하세요")
	if ticket.Format != "madi-query-result-v1" || ticket.ActorID != current(r).ID || ticket.TokenID != current(r).TokenID || ticket.Session == "" || ticket.Session != documentQuerySession(r) || !strings.EqualFold(ticket.ParentID, r.PathValue("id")) || !ticket.Expires.After(time.Now()) || len(ticket.Sources) > 400 || len(ticket.Relations) > 2000 {
		return changed
	}
	parent := documentQueryDocument{ID: ticket.ParentID, WorkspaceID: ticket.WorkspaceID, Version: ticket.ParentVersion, Hash: ticket.ParentHash}
	// Acquire current actor/credential locks before source reads: waiting for a
	// revoked session must never make an older ACL/hash check authoritative.
	if e := s.documentQueryActorTx(r, tx, parent); e != nil {
		return e
	}
	ids := []string{}
	for _, ref := range ticket.Sources {
		ids = append(ids, ref.ID)
	}
	hashes, e := documentQuerySourceHashes(r.Context(), tx, current(r), ticket.WorkspaceID, ids, ticket.Body, ticket.Tasks)
	if e != nil {
		return e
	}
	for _, ref := range ticket.Sources {
		if hashes[ref.ID] != ref.Hash {
			return changed
		}
	}
	if len(ticket.Relations) > 0 {
		var all bool
		raw, e := json.Marshal(ticket.Relations)
		if e != nil {
			return e
		}
		e = tx.QueryRow(r.Context(), `SELECT NOT EXISTS(SELECT 1 FROM jsonb_to_recordset($1::jsonb) AS expected("SourceID" text,"TargetID" text,"Type" text,"Created" text) WHERE NOT EXISTS(SELECT 1 FROM document_relations r WHERE r.source_id=expected."SourceID"::uuid AND r.target_id=expected."TargetID"::uuid AND r.type=expected."Type" AND r.created_at::text=expected."Created"))`, raw).Scan(&all)
		if e != nil {
			return e
		}
		if !all {
			return changed
		}
	}
	var revision int64
	if e = tx.QueryRow(r.Context(), `SELECT revision FROM protection_settings WHERE id=1`).Scan(&revision); e != nil {
		return e
	}
	if revision != ticket.ProtectionRevision {
		return changed
	}
	if e = s.documentQueryActorTx(r, tx, parent); e != nil {
		return e
	}
	if !ticket.Expires.After(time.Now()) {
		return changed
	}
	return nil
}
func (s *Server) checkDocumentQuery(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Token string `json:"validation_token"`
	}
	if documentQueryDecode(r, &in, 1<<20) != nil || len(in.Token) > 768<<10 {
		apiError(w, 400, "조회 결과 검증 토큰을 확인하세요")
		return
	}
	r, tx, done, ok := s.documentQueryBegin(w, r)
	if !ok {
		return
	}
	defer done()
	plain, e := s.decrypt(in.Token)
	var ticket documentQueryTicket
	if e != nil || json.Unmarshal([]byte(plain), &ticket) != nil {
		apiError(w, 409, "조회 결과의 검증 정보를 확인할 수 없습니다. 다시 조회하세요")
		return
	}
	if e = s.validateDocumentQueryTicketTx(r, tx, ticket); e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		documentQueryError(w, e)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	jsonResponse(w, 200, map[string]any{"current": true, "notice": "표시된 행과 경로의 현재 접근·버전만 확인했습니다. 새 행 포함 여부는 새로 조회해야 합니다."})
}
func (s *Server) registerDocumentQueries() {
	s.documentQuerySlots = make(chan struct{}, 4)
	s.handle("GET /api/v1/documents/{id}/queries", s.getDocumentQueries)
	s.handle("POST /api/v1/documents/{id}/queries/execute", s.executeDocumentQuery)
	s.handle("POST /api/v1/documents/{id}/queries/check", s.checkDocumentQuery)
}
