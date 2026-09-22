package server

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

//go:embed knowledge_conflicts.sql
var knowledgeConflictsSchema string

func (s *Server) migrateKnowledgeConflicts(ctx context.Context) error {
	_, e := s.DB.Exec(ctx, knowledgeConflictsSchema)
	return e
}
func (s *Server) registerKnowledgeConflicts() {
	s.handle("POST /api/v1/knowledge/conflicts", s.createKnowledgeConflicts)
	s.handle("GET /api/v1/knowledge/conflicts", s.listKnowledgeConflicts)
	s.handle("GET /api/v1/knowledge/conflicts/{id}", s.getKnowledgeConflicts)
	s.handle("DELETE /api/v1/knowledge/conflicts/{id}", s.deleteKnowledgeConflicts)
	s.handle("GET /api/v1/knowledge/conflict-candidates/{id}", s.getKnowledgeConflictCandidate)
	s.handle("POST /api/v1/knowledge/conflict-candidates/{id}/reviews", s.reviewKnowledgeConflict)
}

type conflictSourceRef struct {
	ID      string `json:"id"`
	Version int    `json:"version"`
	Hash    string `json:"hash"`
}
type conflictSourceRecord struct {
	conflictSource
	Hash  string
	Bytes int
}
type conflictPayload struct {
	Topic string          `json:"topic"`
	Rule  string          `json:"rule"`
	Left  conflictExcerpt `json:"left"`
	Right conflictExcerpt `json:"right"`
}
type conflictExcerpt struct {
	DocumentID   string `json:"document_id"`
	Title        string `json:"title"`
	Version      int    `json:"version"`
	DocumentHash string `json:"document_hash"`
	Start        int    `json:"start_byte"`
	End          int    `json:"end_byte"`
	Line         int    `json:"line"`
	Text         string `json:"text"`
	Hash         string `json:"excerpt_hash"`
}
type conflictCandidate struct {
	ID, RunID, WorkspaceID, OwnerID, LeftID, RightID, LeftHash, RightHash, Ciphertext, PayloadHash, Decision string
	Revision, LeftVersion, RightVersion                                                                      int
}

const conflictNotice = "유사한 문장 구조의 수치·허용/금지·필수/선택 차이를 찾는 제한된 규칙 검사입니다. 의미·사실·정책 충돌을 확정하지 않습니다. 판단 기록은 본인 보고서에만 저장되며 게시 승인이나 문서 수정이 아닙니다."

// Only the E10 read/analysis/judgment endpoints use this short fence. Canonical
// document writers (E4 merge) must not upgrade a documents SHARE table lock.
func conflictReadFenceTx(r *http.Request, tx pgx.Tx) error {
	if _, e := tx.Exec(r.Context(), `SET LOCAL lock_timeout='2s'; SET LOCAL statement_timeout='10s'`); e != nil {
		return e
	}
	_, e := tx.Exec(r.Context(), `LOCK TABLE documents,spaces,space_members,document_shares IN SHARE MODE`)
	return e
}

func (s *Server) conflictSourcesTx(r *http.Request, tx pgx.Tx, wid string, ids []string, memos ...map[string]conflictSourceRecord) (map[string]conflictSourceRecord, error) {
	ids = slices.Clone(ids)
	slices.Sort(ids)
	ids = slices.Compact(ids)
	if len(ids) > 32 || len(ids) == 0 {
		return nil, approvalProblem(400, "문서 선택 범위를 확인하세요")
	}
	// The caller owns the transaction: no nested pool queries while holding locks.
	rows, e := tx.Query(r.Context(), `SELECT id FROM documents WHERE id=ANY($1::uuid[]) ORDER BY id FOR SHARE`, ids)
	if e != nil {
		return nil, e
	}
	for rows.Next() {
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return nil, e
	}
	if e = s.knowledgeActorTx(r, tx, wid, "document:read"); e != nil {
		return nil, approvalProblem(403, "현재 사용자·워크스페이스·읽기 권한을 확인하세요")
	}
	result := map[string]conflictSourceRecord{}
	total := 0
	var memo map[string]conflictSourceRecord
	if len(memos) > 0 {
		memo = memos[0]
	}
	for _, id := range ids {
		if item, ok := memo[id]; ok {
			result[id] = item
			total += item.Bytes
			continue
		}
		var size int
		e = tx.QueryRow(r.Context(), `SELECT octet_length(jsonb_build_array(title,markdown,tags,aliases)::text) FROM documents WHERE id=$1 AND workspace_id=$2 AND deleted_at IS NULL AND madi_document_allowed($3,id,false)`, id, wid, current(r).ID).Scan(&size)
		if e != nil {
			return nil, approvalProblem(404, "현재 접근 가능한 비교 원문이 없습니다")
		}
		total += size
		if size > 1<<20 || total > 8<<20 {
			return nil, approvalProblem(413, "비교 원문과 속성은 문서당 1MiB, 합계 8MiB 이하여야 합니다")
		}
		item := conflictSourceRecord{}
		item.Bytes = size
		var aliases []string
		e = tx.QueryRow(r.Context(), `SELECT id::text,title,markdown,version,tags,aliases,encode(sha256(convert_to(jsonb_build_array(title,markdown,tags,aliases)::text,'UTF8')),'hex') FROM documents WHERE id=$1 AND workspace_id=$2 AND deleted_at IS NULL AND madi_document_allowed($3,id,false)`, id, wid, current(r).ID).Scan(&item.ID, &item.Title, &item.Markdown, &item.Version, &item.Tags, &aliases, &item.Hash)
		if e != nil {
			return nil, approvalProblem(404, "현재 접근 가능한 비교 원문이 없습니다")
		}
		if e = s.checkEvidenceProtection(r.Context(), tx, current(r), wid, map[string]any{"title": item.Title, "markdown": item.Markdown, "tags": item.Tags, "aliases": aliases}); e != nil {
			return nil, approvalProblem(422, "현재 정보 보호 정책으로 비교 원문을 처리할 수 없습니다")
		}
		result[id] = item
		if memo != nil {
			memoTotal := size
			for _, cached := range memo {
				memoTotal += cached.Bytes
			}
			if memoTotal > 8<<20 {
				return nil, approvalProblem(413, "현재 비교 원문의 합계가 8MiB를 초과했습니다. 더 작은 선택으로 다시 분석하세요")
			}
			memo[id] = item
		}
	}
	return result, nil
}
func (s *Server) createKnowledgeConflicts(w http.ResponseWriter, r *http.Request) {
	var in struct {
		WorkspaceID string              `json:"workspace_id"`
		Documents   []conflictSourceRef `json:"documents"`
		Consent     bool                `json:"consent"`
	}
	if decode(r, &in) != nil || !validID(in.WorkspaceID) || !personalAccessRequest(current(r)) || !in.Consent || len(in.Documents) < 2 || len(in.Documents) > 32 {
		apiError(w, 400, "본인 브라우저에서 현재 문서 2~32개와 개인 검토 보관 동의를 확인하세요")
		return
	}
	ids := []string{}
	seen := map[string]bool{}
	for _, ref := range in.Documents {
		if !validID(ref.ID) || ref.Version < 1 || seen[ref.ID] {
			apiError(w, 400, "중복 없는 문서와 현재 버전을 선택하세요")
			return
		}
		seen[ref.ID] = true
		ids = append(ids, ref.ID)
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	r = r.WithContext(ctx)
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(ctx)
	if e = conflictReadFenceTx(r, tx); e != nil {
		respond(w, nil, e)
		return
	}
	sources, e := s.conflictSourcesTx(r, tx, in.WorkspaceID, ids)
	if e != nil {
		approvalRespondError(w, e)
		return
	}
	ordered := []conflictSource{}
	refs := []conflictSourceRef{}
	for _, ref := range in.Documents {
		source := sources[ref.ID]
		if source.Version != ref.Version {
			apiError(w, 409, "선택 원문이 변경되었습니다. 최신 문서로 다시 분석하세요")
			return
		}
		ordered = append(ordered, source.conflictSource)
		refs = append(refs, conflictSourceRef{source.ID, source.Version, source.Hash})
	}
	candidates, stats, e := detectKnowledgeConflicts(ctx, ordered)
	if e != nil {
		apiError(w, 400, e.Error())
		return
	}
	if _, e = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,739))`, current(r).ID); e != nil {
		respond(w, nil, e)
		return
	}
	var count int
	var limited bool
	e = tx.QueryRow(ctx, `SELECT count(*),coalesce(max(created_at)>clock_timestamp()-interval '1 second',false) FROM knowledge_conflict_runs WHERE owner_id=$1`, current(r).ID).Scan(&count, &limited)
	if e != nil {
		respond(w, nil, e)
		return
	}
	if count >= 40 {
		apiError(w, 409, "개인 비교 보고서는 최대 40개입니다. 사용하지 않는 보고서를 정리하세요")
		return
	}
	if limited {
		apiError(w, 429, "분석 요청이 너무 빠릅니다. 잠시 후 다시 시도하세요")
		return
	}
	var id string
	e = tx.QueryRow(ctx, `INSERT INTO knowledge_conflict_runs(workspace_id,owner_id,rule_version,source_refs,diagnostics) VALUES($1,$2,$3,$4,$5) RETURNING id::text`, in.WorkspaceID, current(r).ID, conflictRuleVersion, jsonValue(refs), jsonValue(stats)).Scan(&id)
	if e != nil {
		respond(w, nil, e)
		return
	}
	for n, candidate := range candidates {
		excerpt := func(statement conflictStatement) conflictExcerpt {
			source := sources[statement.SourceID]
			return conflictExcerpt{source.ID, source.Title, source.Version, source.Hash, statement.Start, statement.End, statement.Line, statement.Excerpt, digest(statement.Excerpt)}
		}
		body := conflictPayload{candidate.Topic, candidate.Left.Rule, excerpt(candidate.Left), excerpt(candidate.Right)}
		plain := string(jsonValue(body))
		sealed, sealError := s.encrypt(plain)
		if sealError != nil {
			respond(w, nil, sealError)
			return
		}
		_, e = tx.Exec(ctx, `INSERT INTO knowledge_conflict_candidates(run_id,ordinal,left_id,right_id,left_version,right_version,left_hash,right_hash,ciphertext,payload_hash) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, id, n, body.Left.DocumentID, body.Right.DocumentID, body.Left.Version, body.Right.Version, body.Left.DocumentHash, body.Right.DocumentHash, sealed, digest(plain))
		if e != nil {
			respond(w, nil, e)
			return
		}
	}
	if e = s.conflictCurrentACLTx(r, tx, in.WorkspaceID, ids); e == nil {
		e = tx.Commit(ctx)
	}
	if e != nil {
		approvalRespondError(w, e)
		return
	}
	jsonResponse(w, 201, map[string]any{"id": id, "candidates": len(candidates), "diagnostics": stats, "rule_version": conflictRuleVersion, "notice": conflictNotice})
}
func (s *Server) conflictCurrentACLTx(r *http.Request, tx pgx.Tx, wid string, ids []string) error {
	if e := s.knowledgeActorTx(r, tx, wid, "document:read"); e != nil {
		return approvalProblem(403, "현재 사용자 권한이 변경되었습니다")
	}
	var allowed bool
	e := tx.QueryRow(r.Context(), `SELECT count(*)=$4 FROM documents WHERE id=ANY($1::uuid[]) AND workspace_id=$2 AND deleted_at IS NULL AND madi_document_allowed($3,id,false)`, ids, wid, current(r).ID, len(ids)).Scan(&allowed)
	if e != nil {
		return e
	}
	if !allowed {
		return approvalProblem(404, "현재 접근 가능한 비교 원문이 없습니다")
	}
	return nil
}
func (s *Server) listKnowledgeConflicts(w http.ResponseWriter, r *http.Request) {
	wid := r.URL.Query().Get("workspace_id")
	if !validID(wid) {
		apiError(w, 400, "워크스페이스를 선택하세요")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	if e = s.knowledgeActorTx(r, tx, wid, "document:read"); e != nil {
		apiError(w, 403, "현재 사용자와 읽기 권한을 확인하세요")
		return
	}
	rows, e := tx.Query(r.Context(), `SELECT jsonb_build_object('id',r.id,'created_at',r.created_at,'rule_version',r.rule_version,'source_count',jsonb_array_length(r.source_refs),'candidate_count',(SELECT count(*) FROM knowledge_conflict_candidates c WHERE c.run_id=r.id),'diagnostics',r.diagnostics) FROM knowledge_conflict_runs r WHERE r.owner_id=$1 AND r.workspace_id=$2 ORDER BY r.created_at DESC,r.id LIMIT 40`, current(r).ID, wid)
	if e != nil {
		respond(w, nil, e)
		return
	}
	out := []map[string]any{}
	for rows.Next() {
		var item map[string]any
		if e = rows.Scan(&item); e != nil {
			break
		}
		out = append(out, item)
	}
	if e == nil {
		e = rows.Err()
	}
	rows.Close()
	respond(w, map[string]any{"items": out, "private": true, "notice": conflictNotice}, e)
}
func conflictCandidateTx(r *http.Request, tx pgx.Tx, id string, ownerOnly, lock bool) (conflictCandidate, error) {
	var c conflictCandidate
	if !validID(id) {
		return c, approvalProblem(404, "현재 접근 가능한 비교 후보가 없습니다")
	}
	owner := ""
	if ownerOnly {
		owner = " AND r.owner_id=$2"
	}
	var run string
	var sources []string
	if e := tx.QueryRow(r.Context(), `SELECT c.run_id::text,ARRAY[c.left_id::text,c.right_id::text] FROM knowledge_conflict_candidates c JOIN knowledge_conflict_runs r ON r.id=c.run_id WHERE c.id=$1 AND $2::uuid IS NOT NULL`+owner, id, current(r).ID).Scan(&run, &sources); e != nil {
		return c, approvalProblem(404, "현재 접근 가능한 비교 후보가 없습니다")
	}
	// Document -> report -> candidate is shared by review, origin creation and
	// merge. Never take a candidate lock before either document lock.
	rows, e := tx.Query(r.Context(), `SELECT id FROM documents WHERE id=ANY($1::uuid[]) ORDER BY id FOR SHARE`, sources)
	if e != nil {
		return c, e
	}
	for rows.Next() {
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return c, e
	}
	if e = tx.QueryRow(r.Context(), `SELECT id::text FROM knowledge_conflict_runs WHERE id=$1 FOR SHARE`, run).Scan(&run); e != nil {
		return c, e
	}
	suffix := " FOR SHARE OF c"
	if lock {
		suffix = " FOR UPDATE OF c"
	}
	e = tx.QueryRow(r.Context(), `SELECT c.id::text,c.run_id::text,r.workspace_id::text,r.owner_id::text,c.left_id::text,c.right_id::text,c.left_version,c.right_version,c.left_hash,c.right_hash,c.ciphertext,c.payload_hash,c.revision,c.decision FROM knowledge_conflict_candidates c JOIN knowledge_conflict_runs r ON r.id=c.run_id WHERE c.id=$1 AND ($2::uuid IS NOT NULL)`+owner+suffix, id, current(r).ID).Scan(&c.ID, &c.RunID, &c.WorkspaceID, &c.OwnerID, &c.LeftID, &c.RightID, &c.LeftVersion, &c.RightVersion, &c.LeftHash, &c.RightHash, &c.Ciphertext, &c.PayloadHash, &c.Revision, &c.Decision)
	if errors.Is(e, pgx.ErrNoRows) {
		e = approvalProblem(404, "현재 접근 가능한 비교 후보가 없습니다")
	}
	return c, e
}
func (s *Server) conflictCandidateViewTx(r *http.Request, tx pgx.Tx, c conflictCandidate, history bool, memos ...map[string]conflictSourceRecord) (map[string]any, error) {
	// This guard is used even for report owners; service administrators get no bypass.
	sources, e := s.conflictSourcesTx(r, tx, c.WorkspaceID, []string{c.LeftID, c.RightID}, memos...)
	if e != nil {
		return nil, e
	}
	plain, e := s.decrypt(c.Ciphertext)
	var body conflictPayload
	if e != nil || digest(plain) != c.PayloadHash || json.Unmarshal([]byte(plain), &body) != nil {
		return nil, errors.New("비교 자료의 무결성을 확인할 수 없습니다")
	}
	if digest(body.Left.Text) != body.Left.Hash || digest(body.Right.Text) != body.Right.Hash {
		return nil, errors.New("비교 구간의 무결성을 확인할 수 없습니다")
	}
	if e = s.checkEvidenceProtection(r.Context(), tx, current(r), c.WorkspaceID, body); e != nil {
		return nil, approvalProblem(422, "현재 정보 보호 정책으로 비교 기록을 표시할 수 없습니다")
	}
	left, right := sources[c.LeftID], sources[c.RightID]
	stale := left.Version != c.LeftVersion || right.Version != c.RightVersion || left.Hash != c.LeftHash || right.Hash != c.RightHash
	out := map[string]any{"id": c.ID, "run_id": c.RunID, "workspace_id": c.WorkspaceID, "revision": c.Revision, "decision": c.Decision, "stale": stale, "left": body.Left, "right": body.Right, "topic": body.Topic, "rule": body.Rule, "left_current_version": left.Version, "right_current_version": right.Version}
	if history {
		rows, e := tx.Query(r.Context(), `SELECT revision,decision,note_ciphertext,created_at FROM knowledge_conflict_reviews WHERE candidate_id=$1 ORDER BY revision DESC LIMIT 100`, c.ID)
		if e != nil {
			return nil, e
		}
		events := []map[string]any{}
		for rows.Next() {
			var revision int
			var decision, sealed string
			var created time.Time
			if e = rows.Scan(&revision, &decision, &sealed, &created); e != nil {
				break
			}
			note, err := s.decrypt(sealed)
			if err != nil {
				e = err
				break
			}
			events = append(events, map[string]any{"revision": revision, "decision": decision, "note": note, "created_at": created})
		}
		if e == nil {
			e = rows.Err()
		}
		rows.Close()
		if e != nil {
			return nil, e
		}
		if e = s.checkEvidenceProtection(r.Context(), tx, current(r), c.WorkspaceID, events); e != nil {
			return nil, approvalProblem(422, "현재 정보 보호 정책으로 판단 기록을 표시할 수 없습니다")
		}
		out["history"] = events
	}
	if e = s.conflictCurrentACLTx(r, tx, c.WorkspaceID, []string{c.LeftID, c.RightID}); e != nil {
		return nil, e
	}
	return out, nil
}
func (s *Server) getKnowledgeConflictCandidate(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	r = r.WithContext(ctx)
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	if e = conflictReadFenceTx(r, tx); e != nil {
		respond(w, nil, e)
		return
	}
	c, e := conflictCandidateTx(r, tx, r.PathValue("id"), true, false)
	if e != nil {
		approvalRespondError(w, e)
		return
	}
	out, e := s.conflictCandidateViewTx(r, tx, c, true)
	if e != nil {
		approvalRespondError(w, e)
		return
	}
	jsonResponse(w, 200, out)
}
func (s *Server) getKnowledgeConflicts(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	r = r.WithContext(ctx)
	id := r.PathValue("id")
	if !validID(id) {
		apiError(w, 404, "개인 비교 보고서가 없습니다")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	if e = conflictReadFenceTx(r, tx); e != nil {
		respond(w, nil, e)
		return
	}
	var wid, rule string
	var created time.Time
	var diagnostics map[string]any
	e = tx.QueryRow(r.Context(), `SELECT workspace_id::text,rule_version,created_at,diagnostics FROM knowledge_conflict_runs WHERE id=$1 AND owner_id=$2 FOR SHARE`, id, current(r).ID).Scan(&wid, &rule, &created, &diagnostics)
	if e != nil {
		apiError(w, 404, "현재 접근 가능한 개인 비교 보고서가 없습니다")
		return
	}
	if e = s.knowledgeActorTx(r, tx, wid, "document:read"); e != nil {
		apiError(w, 403, "현재 읽기 권한을 확인하세요")
		return
	}
	rows, e := tx.Query(r.Context(), `SELECT id::text FROM knowledge_conflict_candidates WHERE run_id=$1 ORDER BY ordinal LIMIT 64`, id)
	if e != nil {
		respond(w, nil, e)
		return
	}
	ids := []string{}
	for rows.Next() {
		var cid string
		if e = rows.Scan(&cid); e != nil {
			break
		}
		ids = append(ids, cid)
	}
	if e == nil {
		e = rows.Err()
	}
	rows.Close()
	if e != nil {
		respond(w, nil, e)
		return
	}
	out := []map[string]any{}
	// Per-transaction only, protected by the read fence. Never shared across
	// requests/users and bounded by the same 8MiB combined input budget.
	memo := map[string]conflictSourceRecord{}
	for _, cid := range ids {
		c, err := conflictCandidateTx(r, tx, cid, true, false)
		if err != nil {
			approvalRespondError(w, err)
			return
		}
		item, err := s.conflictCandidateViewTx(r, tx, c, false, memo)
		if err != nil {
			var failure *approvalFailure
			if errors.As(err, &failure) && oneOfStatus(failure.Status, 403, 404, 413, 422) {
				out = append(out, map[string]any{"id": cid, "unavailable": true})
				continue
			}
			approvalRespondError(w, err)
			return
		}
		out = append(out, item)
	}
	jsonResponse(w, 200, map[string]any{"id": id, "workspace_id": wid, "rule_version": rule, "created_at": created, "diagnostics": diagnostics, "candidates": out, "notice": conflictNotice, "private": true})
}
func oneOfStatus(value int, options ...int) bool { return slices.Contains(options, value) }
func (s *Server) reviewKnowledgeConflict(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	r = r.WithContext(ctx)
	var in struct {
		Revision int    `json:"revision"`
		Decision string `json:"decision"`
		Note     string `json:"note"`
		Consent  bool   `json:"consent"`
	}
	if decode(r, &in) != nil || !personalAccessRequest(current(r)) || in.Revision < 1 || !oneOf(in.Decision, "conflict", "compatible", "needs_review") || !in.Consent || len(in.Note) > 4000 || strings.TrimSpace(in.Note) == "" {
		apiError(w, 400, "판단·근거·현재 버전과 개인 판단 기록 동의를 확인하세요")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	if e = conflictReadFenceTx(r, tx); e != nil {
		respond(w, nil, e)
		return
	}
	c, e := conflictCandidateTx(r, tx, r.PathValue("id"), true, true)
	if e != nil {
		approvalRespondError(w, e)
		return
	}
	out, e := s.conflictCandidateViewTx(r, tx, c, false)
	if e != nil {
		approvalRespondError(w, e)
		return
	}
	if boolean(out, "stale") || c.Revision != in.Revision || c.Revision >= 2147483647 {
		apiError(w, 409, "원문 또는 판단 기록이 변경되었습니다. 새 분석·최신 기록으로 다시 확인하세요")
		return
	}
	if e = s.checkEvidenceProtection(r.Context(), tx, current(r), c.WorkspaceID, in.Note); e != nil {
		apiError(w, 422, "현재 정보 보호 정책으로 판단 근거를 보관할 수 없습니다")
		return
	}
	sealed, e := s.encrypt(in.Note)
	if e == nil {
		_, e = tx.Exec(r.Context(), `UPDATE knowledge_conflict_candidates SET decision=$2,revision=revision+1,updated_at=now() WHERE id=$1 AND revision=$3`, c.ID, in.Decision, in.Revision)
	}
	if e == nil {
		_, e = tx.Exec(r.Context(), `INSERT INTO knowledge_conflict_reviews(candidate_id,actor_id,revision,decision,note_ciphertext) VALUES($1,$2,$3,$4,$5)`, c.ID, current(r).ID, c.Revision+1, in.Decision, sealed)
	}
	if e == nil {
		e = s.conflictCurrentACLTx(r, tx, c.WorkspaceID, []string{c.LeftID, c.RightID})
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		approvalRespondError(w, e)
		return
	}
	jsonResponse(w, 200, map[string]any{"revision": c.Revision + 1, "decision": in.Decision, "document_changed": false})
}
func (s *Server) deleteKnowledgeConflicts(w http.ResponseWriter, r *http.Request) {
	if !validID(r.PathValue("id")) || !personalAccessRequest(current(r)) {
		apiError(w, 404, "개인 비교 보고서가 없습니다")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	var wid string
	e = tx.QueryRow(r.Context(), `SELECT workspace_id::text FROM knowledge_conflict_runs WHERE id=$1 AND owner_id=$2 FOR UPDATE`, r.PathValue("id"), current(r).ID).Scan(&wid)
	if e != nil {
		apiError(w, 404, "개인 비교 보고서가 없습니다")
		return
	}
	if e = s.knowledgeActorTx(r, tx, wid, "document:read"); e != nil {
		apiError(w, 403, "현재 사용자 권한을 확인하세요")
		return
	}
	var linked bool
	e = tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM knowledge_proposal_conflict_origins o JOIN knowledge_conflict_candidates c ON c.id=o.candidate_id WHERE c.run_id=$1)`, r.PathValue("id")).Scan(&linked)
	if e != nil {
		respond(w, nil, e)
		return
	}
	if linked {
		apiError(w, 409, "변경안의 출처로 연결된 보고서는 출처 보존을 위해 삭제할 수 없습니다")
		return
	}
	_, e = tx.Exec(r.Context(), `DELETE FROM knowledge_conflict_runs WHERE id=$1 AND owner_id=$2`, r.PathValue("id"), current(r).ID)
	if e == nil {
		e = tx.Commit(r.Context())
	}
	respond(w, map[string]any{"deleted": true}, e)
}
