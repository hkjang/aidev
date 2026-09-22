package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (s *Server) searchKnowledgeAtDate(w http.ResponseWriter, r *http.Request) {
	wid, date, term := r.URL.Query().Get("workspace_id"), r.URL.Query().Get("date"), strings.TrimSpace(r.URL.Query().Get("q"))
	after := r.URL.Query().Get("after")
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		v, e := strconv.Atoi(raw)
		if e != nil || v < 1 || v > 50 {
			apiError(w, 400, "페이지 크기는 1~50입니다")
			return
		}
		limit = v
	}
	if !validID(wid) || !validKnowledgeDate(date) || len(term) > 500 || after != "" && !validID(after) {
		apiError(w, 400, "워크스페이스·기준일·500바이트 이하 검색어·다음 위치를 확인하세요")
		return
	}
	if _, err := s.packagePrincipal(r, wid); err != nil {
		apiError(w, 403, "현재 워크스페이스 문서 조회 권한이 필요합니다")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	r = r.WithContext(ctx)
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		respond(w, nil, err)
		return
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SET LOCAL statement_timeout='7s'`); err != nil {
		respond(w, nil, err)
		return
	}
	revision, entries, err := searchDictionary(ctx, tx, wid)
	if err != nil {
		respond(w, nil, err)
		return
	}
	interpretation := interpretSearch(term, revision, entries)
	if interpretation.Strategy == "websearch_syntax" {
		apiError(w, 400, "시점 검색은 일반 단어·조직 용어 검색을 지원합니다. 따옴표·OR·제외 문법은 현재 통합 검색에서 사용하세요")
		return
	}
	started := time.Now()
	rows, err := tx.Query(ctx, `SELECT jsonb_build_object('document_id',d.id,'title',v.title,'version',v.version,'current_version',d.version,'valid_from',p.valid_from,'valid_until',p.valid_until,'validity_revision',c.revision,'recorded_at',v.created_at,'excerpt',left(v.markdown,180)) FROM documents d JOIN knowledge_validity c ON c.document_id=d.id JOIN knowledge_validity_periods p ON p.document_id=d.id JOIN document_versions v ON v.document_id=d.id AND v.version=p.document_version WHERE d.workspace_id=$1 AND d.deleted_at IS NULL AND madi_document_allowed($2,d.id,false) AND daterange(p.valid_from,p.valid_until,'[)') @> $3::text::date AND ($4='' OR d.id>NULLIF($4,'')::uuid) AND ($5='' OR madi_search_folded_matches(regexp_replace(lower(normalize(v.title||E'\n'||v.markdown||E'\n'||v.tags::text,NFKC)),'[[:space:]]','','g'),$6::jsonb)) ORDER BY d.id LIMIT $7 FOR SHARE OF d,c`, wid, current(r).ID, date, after, term, jsonValue(interpretation.FoldedParts), limit+1)
	if err != nil {
		if ctx.Err() != nil || strings.Contains(err.Error(), "statement timeout") {
			apiError(w, 504, "시점 검색의 읽기 시간 한도를 초과했습니다. 검색 조건을 좁혀 다시 시도하세요")
		} else {
			respond(w, nil, err)
		}
		return
	}
	results := []map[string]any{}
	for rows.Next() {
		var raw []byte
		var item map[string]any
		if err = rows.Scan(&raw); err != nil {
			break
		}
		if err = json.Unmarshal(raw, &item); err != nil {
			break
		}
		results = append(results, item)
	}
	if err == nil {
		err = rows.Err()
	}
	rows.Close()
	if err != nil {
		respond(w, nil, err)
		return
	}
	more := len(results) > limit
	if more {
		results = results[:limit]
	}
	if err = s.knowledgeActorTx(r, tx, wid, "document:read"); err != nil {
		apiError(w, 403, "현재 조회 권한이 변경되었습니다")
		return
	}
	if err = s.checkEvidenceProtection(ctx, tx, current(r), wid, results); err != nil {
		apiError(w, 422, "현재 정보 보호 정책에서 과거 검색 조각을 표시할 수 없습니다")
		return
	}
	next := ""
	if more {
		next = str(results[len(results)-1], "document_id")
	}
	outcome := "matched"
	if len(results) == 0 {
		outcome = "no_match_in_declared_periods"
	}
	var protection int
	if err = tx.QueryRow(ctx, `SELECT revision FROM protection_settings WHERE id=1`).Scan(&protection); err != nil {
		respond(w, nil, err)
		return
	}
	jsonResponse(w, 200, map[string]any{"results": results, "has_more": more, "next_after": next, "date": date, "interpretation": interpretation, "outcome": outcome, "elapsed_ms": float64(time.Since(started).Microseconds()) / 1000, "protection_revision": protection, "notice": validityNotice + " 결과는 문서 ID 순서이며 고정 시점의 DB 스냅샷이나 과거 ACL 복원이 아닙니다. 현재 용어 사전으로 당시 원문을 검색합니다."})
}

// One bounded request for the page, never one poll per result. A false result
// intentionally does not identify which formerly-visible resource changed.
func (s *Server) checkKnowledgeTimeResults(w http.ResponseWriter, r *http.Request) {
	var in struct {
		WorkspaceID        string `json:"workspace_id"`
		ProtectionRevision int    `json:"protection_revision"`
		Documents          []struct {
			ID       string `json:"id"`
			Version  int    `json:"version"`
			Revision int    `json:"revision"`
		} `json:"documents"`
	}
	if decode(r, &in) != nil || !validID(in.WorkspaceID) || in.ProtectionRevision < 1 || in.ProtectionRevision > 2147483647 || in.Documents == nil || len(in.Documents) > 52 {
		apiError(w, 400, "현재 조회 범위와 보호 정책 revision을 확인하세요")
		return
	}
	seen := map[string]bool{}
	for _, d := range in.Documents {
		if !validID(d.ID) || d.Version < 1 || d.Version > 2147483647 || d.Revision < 0 || d.Revision > 2147483647 || seen[d.ID] {
			apiError(w, 400, "중복 없는 문서·버전·유효기간 revision이 필요합니다")
			return
		}
		seen[d.ID] = true
	}
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		respond(w, nil, err)
		return
	}
	defer tx.Rollback(r.Context())
	if err = s.knowledgeActorTx(r, tx, in.WorkspaceID, "document:read"); err != nil {
		apiError(w, 403, "현재 조회 권한이 변경되었습니다")
		return
	}
	var count, protection int
	err = tx.QueryRow(r.Context(), `SELECT count(*) FROM jsonb_to_recordset($1::jsonb) ref(id uuid,version integer,revision integer) JOIN documents d ON d.id=ref.id LEFT JOIN knowledge_validity c ON c.document_id=d.id WHERE d.workspace_id=$2 AND d.version=ref.version AND coalesce(c.revision,0)=ref.revision AND d.deleted_at IS NULL AND madi_document_allowed($3,d.id,false)`, jsonValue(in.Documents), in.WorkspaceID, current(r).ID).Scan(&count)
	if err == nil {
		err = tx.QueryRow(r.Context(), `SELECT revision FROM protection_settings WHERE id=1`).Scan(&protection)
	}
	respond(w, map[string]any{"valid": count == len(in.Documents) && protection == in.ProtectionRevision}, err)
}
