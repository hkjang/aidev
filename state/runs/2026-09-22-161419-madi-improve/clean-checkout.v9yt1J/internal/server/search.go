package server

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgconn"
)

var universalSearchKinds = []string{"document", "block", "code", "task", "file", "comment", "tag", "database", "row", "user", "ai_conversation"}

func searchAPIError(w http.ResponseWriter, status int, message string) {
	outcome := "invalid_query"
	if status == 403 {
		outcome = "scope_unavailable"
	}
	jsonResponse(w, status, map[string]any{"error": message, "outcome": outcome})
}
func searchFailure(w http.ResponseWriter, ctx context.Context, e error) {
	var pgerr *pgconn.PgError
	if errors.Is(e, context.DeadlineExceeded) || ctx.Err() != nil || (errors.As(e, &pgerr) && pgerr.Code == "57014") {
		jsonResponse(w, 504, map[string]any{"error": "검색 제한시간을 초과했습니다. 조건을 좁히거나 다시 시도하세요", "outcome": "timeout"})
		return
	}
	respond(w, nil, e)
}

// Search is a read model, not a new permission system. All document-backed
// branches join the current ancestor ACL; database branches independently need
// database:read. Projection versions are joined before snippets are returned.
func (s *Server) universalSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	p := current(r)
	wid := q.Get("workspace_id")
	term := strings.TrimSpace(q.Get("q"))
	kind := q.Get("type")
	if !s.canWorkspace(r.Context(), p, wid, false) {
		searchAPIError(w, 403, "검색할 워크스페이스에 접근할 수 없습니다")
		return
	}
	if !utf8.ValidString(term) || len(term) > 500 || (kind != "" && !oneOf(kind, universalSearchKinds...)) {
		searchAPIError(w, 400, "검색어는 500바이트 이내이며 올바른 검색 종류가 필요합니다")
		return
	}
	if !oneOf(q.Get("group"), "", "document") {
		searchAPIError(w, 400, "검색 결과 묶기 방식을 확인하세요")
		return
	}
	for _, name := range []string{"space_id", "author_id"} {
		if q.Get(name) != "" && !validID(q.Get(name)) {
			searchAPIError(w, 400, "검색 필터의 식별자를 확인하세요")
			return
		}
	}
	if len(q.Get("tag")) > 200 || !oneOf(q.Get("status"), "", "draft", "review", "published", "rejected", "stale", "archived") || !oneOf(q.Get("has_attachment"), "", "1") {
		searchAPIError(w, 400, "문서 상태·태그·첨부 필터를 확인하세요")
		return
	}
	var from, to *time.Time
	for _, name := range []string{"from", "to"} {
		if q.Get(name) != "" {
			t, e := time.Parse("2006-01-02", q.Get(name))
			if e != nil {
				searchAPIError(w, 400, "검색 날짜는 YYYY-MM-DD 형식입니다")
				return
			}
			if name == "from" {
				from = &t
			} else {
				t = t.AddDate(0, 0, 1)
				to = &t
			}
		}
	}
	if from != nil && to != nil && !from.Before(*to) {
		searchAPIError(w, 400, "검색 시작일은 종료일 이후일 수 없습니다")
		return
	}
	limit, offset := 40, 0
	for _, name := range []string{"limit", "offset"} {
		if q.Get(name) != "" {
			n, e := strconv.Atoi(q.Get(name))
			if e != nil || n < 0 || (name == "limit" && (n < 1 || n > 100)) || (name == "offset" && n > 10000) {
				searchAPIError(w, 400, "검색 페이지 범위를 확인하세요")
				return
			}
			if name == "limit" {
				limit = n
			} else {
				offset = n
			}
		}
	}
	sortSQL := "score DESC,updated_at DESC,kind,id"
	switch q.Get("sort") {
	case "", "relevance":
	case "newest":
		sortSQL = "updated_at DESC,score DESC,kind,id"
	case "oldest":
		sortSQL = "updated_at ASC,score DESC,kind,id"
	case "title":
		sortSQL = "title,kind,id"
	default:
		searchAPIError(w, 400, "검색 정렬을 확인하세요")
		return
	}
	canDocs := hasIntegrationScope(p, "document:read") || hasIntegrationScope(p, "search:read")
	canDB := hasIntegrationScope(p, "database:read")
	canUsers := p.TokenID == "" && !p.ScopeRestricted
	pattern := "%" + strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_").Replace(term) + "%"
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	revision, dictionary, e := searchDictionary(ctx, s.DB, wid)
	if e != nil {
		searchFailure(w, ctx, e)
		return
	}
	interpretation := interpretSearch(term, revision, dictionary)
	cursor, e := s.readSearchCursor(q.Get("cursor"), p, wid, q, revision)
	if e != nil {
		jsonResponse(w, 409, map[string]any{"error": e.Error(), "outcome": "search_changed"})
		return
	}
	if q.Get("cursor") != "" && offset != 0 {
		searchAPIError(w, 400, "커서와 offset 페이지를 동시에 지정할 수 없습니다")
		return
	}
	querySQL := interpretedSearchSQL(term, kind)
	if q.Get("group") == "document" {
		querySQL = groupedSearchSQL(querySQL)
	}
	querySQL = strings.ReplaceAll(querySQL, "now()", "$23::timestamptz") + searchCursorPredicate(q.Get("sort")) + " ORDER BY " + sortSQL + " LIMIT $17 OFFSET $18"
	arguments := []any{p.ID, wid, term, pattern, kind, q.Get("space_id"), q.Get("author_id"), q.Get("tag"), q.Get("status"), from, to, q.Get("has_attachment") == "1", canDocs, canDB, canUsers, q.Get("space_id") == "" && q.Get("author_id") == "" && q.Get("tag") == "" && q.Get("status") == "" && from == nil && to == nil && q.Get("has_attachment") == "", limit + 1, offset, jsonValue(interpretation.Parts), jsonValue(interpretation.FoldedParts), interpretation.GramQuery, jsonValue(cursor.After), cursor.AsOf}
	started := time.Now()
	rows, e := s.rows(ctx, querySQL, arguments...)
	observation, _ := r.Context().Value(searchObservationKey).(*searchObservation)
	if observation != nil {
		observation.SQL = querySQL
		observation.Arguments = arguments
		observation.Elapsed = time.Since(started)
	}
	if e != nil {
		searchFailure(w, ctx, e)
		return
	}
	more := len(rows) > limit
	if more {
		rows = rows[:limit]
	}
	nextCursor := ""
	if more && len(rows) > 0 {
		nextCursor, e = s.nextSearchCursor(cursor, rows[len(rows)-1])
		if e != nil {
			respond(w, nil, e)
			return
		}
	}
	for _, row := range rows {
		if str(row, "kind") == "tag" {
			row["url"] = "/app/search?type=document&tag=" + url.QueryEscape(str(row, "title"))
		}
	}
	available := []string{}
	for _, k := range universalSearchKinds {
		if (k == "database" || k == "row") && canDB || (k == "user" || k == "ai_conversation") && canUsers || k != "database" && k != "row" && k != "user" && k != "ai_conversation" && canDocs {
			available = append(available, k)
		}
	}
	// Do not write raw queries to the shared audit log: they can contain private
	// facts. Query-history consent and AI gap analysis use a separate owner store.
	historyStatus := "not_recorded"
	if observation == nil {
		s.audit(r, "SEARCH", wid, map[string]any{"result_count": len(rows), "empty": len(rows) == 0, "type": kind, "offset": offset})
		historyStatus = s.recordPersonalSearch(r, wid, term, len(rows))
	}
	outcome := "matched"
	if len(rows) == 0 {
		outcome = "no_match_in_current_scope"
		if canDocs && len(interpretation.FoldedParts) > 0 {
			var pending bool
			if e = s.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM documents d LEFT JOIN search_folded_documents z ON z.document_id=d.id WHERE d.workspace_id=$1 AND d.deleted_at IS NULL AND z.document_version IS DISTINCT FROM d.version AND madi_document_allowed($2,d.id,false))`, wid, p.ID).Scan(&pending); e != nil {
				searchFailure(w, ctx, e)
				return
			}
			if pending {
				outcome = "index_pending"
			}
		}
	}
	respond(w, map[string]any{"results": rows, "has_more": more, "next_offset": offset + len(rows), "next_cursor": nextCursor, "pagination": "current_acl_keyset", "outcome": outcome, "group": q.Get("group"), "available_types": available, "mode": "keyword", "history_status": historyStatus, "interpretation": interpretation, "ranking": "제목·태그·본문의 가중치 + 최근 수정 + 내 즐겨찾기 + 최근 30일 열람 인기도(최근 256회 내 서로 다른 열람자, 소폭 가중). 문서 조각과 공백 정규화 색인은 현재 원문 버전만 검색합니다."}, nil)
}

const universalSearchSQL = `WITH active_workspace AS MATERIALIZED (
 SELECT m.workspace_id FROM users u JOIN workspace_members m ON m.user_id=u.id
 WHERE u.id=$1 AND NOT u.disabled AND m.workspace_id=$2
), visible_documents AS NOT MATERIALIZED (
 SELECT d.* FROM documents d JOIN active_workspace membership ON membership.workspace_id=d.workspace_id
 WHERE $13 AND d.workspace_id=$2 AND d.deleted_at IS NULL AND
 CASE WHEN d.parent_id IS NULL AND d.space_id IS NULL THEN
  (d.visibility='workspace' OR d.owner_id=$1 OR (d.visibility='selected' AND EXISTS(SELECT 1 FROM document_shares sh WHERE sh.document_id=d.id AND sh.user_id=$1)))
 ELSE madi_document_allowed($1,d.id,false) END
 AND ($6='' OR d.space_id=NULLIF($6,'')::uuid) AND ($7='' OR d.owner_id=NULLIF($7,'')::uuid)
 AND ($8='' OR d.tags ? $8) AND ($9='' OR d.status=$9) AND ($10::timestamptz IS NULL OR d.updated_at>=$10) AND ($11::timestamptz IS NULL OR d.updated_at<$11)
 AND (NOT $12 OR EXISTS(SELECT 1 FROM attachments a WHERE a.document_id=d.id))
), visible_databases AS NOT MATERIALIZED (
 SELECT d.* FROM databases d JOIN active_workspace membership ON membership.workspace_id=d.workspace_id
 WHERE $14 AND d.workspace_id=$2 AND madi_space_allowed($1,d.space_id,false)
 AND ($6='' OR d.space_id=NULLIF($6,'')::uuid) AND $7='' AND $8='' AND $9='' AND NOT $12
 AND ($10::timestamptz IS NULL OR d.created_at>=$10) AND ($11::timestamptz IS NULL OR d.created_at<$11)
), hits AS (
 SELECT 'document'::text kind,d.id::text id,d.id::text document_id,d.title,
 ''::text snippet,
 '/app/documents/'||d.id url,d.updated_at,
 ((CASE WHEN lower(d.title)=lower($3) AND $3<>'' THEN 8 WHEN d.title ILIKE $4 THEN 4 ELSE 0 END)
 + CASE WHEN d.tags::text ILIKE $4 THEN 3 ELSE 0 END+ts_rank_cd(d.search_vector,websearch_to_tsquery('simple',$3))*2
 + 0.2/(1+greatest(0,extract(epoch FROM now()-d.updated_at)/86400))
 + CASE WHEN EXISTS(SELECT 1 FROM favorites f WHERE f.document_id=d.id AND f.user_id=$1) THEN 0.35 ELSE 0 END
 + (SELECT 0.25*least(1.0,count(DISTINCT seen.user_id)/20.0) FROM (SELECT a.user_id FROM audit_logs a WHERE a.action='DOCUMENT_READ' AND a.resource=d.id::text AND a.created_at>now()-interval '30 days' ORDER BY a.created_at DESC LIMIT 256) seen))::float8 score,
 '{}'::jsonb metadata
 FROM visible_documents d WHERE ($5='' OR $5='document') AND ($3='' OR d.search_vector@@websearch_to_tsquery('simple',$3) OR d.title ILIKE $4 OR d.markdown ILIKE $4 OR d.tags::text ILIKE $4 OR d.aliases::text ILIKE $4)
 UNION ALL
 SELECT f.kind,d.id||':'||f.ordinal,d.id::text,d.title,
 substring(f.content from greatest(1,strpos(lower(f.content),lower($3))-100) for 600),
 CASE WHEN f.kind='task' AND coalesce(f.metadata->>'task_id','')<>'' AND coalesce((f.metadata->>'ambiguous')::boolean,false)=false THEN '/app/tasks?task='||(f.metadata->>'task_id') ELSE '/app/documents/'||d.id||'?line='||f.start_line END,d.updated_at,
 (1+ts_rank_cd(f.search_vector,websearch_to_tsquery('simple',$3)))::float8,
 f.metadata||jsonb_build_object('version',d.version,'start_line',f.start_line,'start_byte',f.start_byte,'end_byte',f.end_byte)
 FROM search_fragments f JOIN visible_documents d ON d.id=f.document_id AND d.version=f.document_version
 WHERE ($5='' OR $5=f.kind) AND ($3='' OR f.search_vector@@websearch_to_tsquery('simple',$3) OR f.content ILIKE $4)
 UNION ALL
 SELECT 'file',a.id::text,d.id::text,a.name,d.title,'/app/documents/'||d.id,d.updated_at,2::float8,jsonb_build_object('size',a.size,'content_type',a.content_type,'document_title',d.title,'attachment_id',a.id)
 FROM attachments a JOIN visible_documents d ON d.id=a.document_id WHERE ($5='' OR $5='file') AND ($3='' OR a.name ILIKE $4 OR a.content_type ILIKE $4)
 UNION ALL
 SELECT 'comment',c.id::text,d.id::text,d.title,substring(c.body from greatest(1,strpos(lower(c.body),lower($3))-100) for 600),'/app/documents/'||d.id||'?comment='||c.id,coalesce(c.edited_at,c.created_at),1::float8,jsonb_build_object('user_id',c.user_id,'resolved',c.resolved_at IS NOT NULL)
 FROM comments c JOIN visible_documents d ON d.id=c.document_id WHERE c.deleted_at IS NULL AND ($5='' OR $5='comment') AND ($3='' OR to_tsvector('simple',c.body)@@websearch_to_tsquery('simple',$3) OR c.body ILIKE $4)
 UNION ALL
 SELECT 'tag',tag,'',tag,count(*)::text||'개 접근 가능한 문서','/app/search?type=document&tag='||tag,max(d.updated_at),3::float8,jsonb_build_object('document_count',count(*))
 FROM visible_documents d CROSS JOIN LATERAL jsonb_array_elements_text(d.tags) t(tag) WHERE octet_length(tag)<=200 AND ($5='' OR $5='tag') AND ($3='' OR tag ILIKE $4) GROUP BY tag
 UNION ALL
 SELECT 'database',d.id::text,'',d.name,'문서 데이터베이스','/app/databases/'||d.id,d.created_at,4::float8,jsonb_build_object('space_id',d.space_id)
 FROM visible_databases d WHERE ($5='' OR $5='database') AND ($3='' OR d.name ILIKE $4)
 UNION ALL
 SELECT 'row',r.id::text,'',d.name,substring(r.values::text from greatest(1,strpos(lower(r.values::text),lower($3))-100) for 600),'/app/databases/'||d.id||'?row='||r.id,r.updated_at,1::float8,jsonb_build_object('database_id',d.id)
 FROM database_rows r JOIN visible_databases d ON d.id=r.database_id WHERE ($5='' OR $5='row') AND ($3='' OR r.values::text ILIKE $4)
 UNION ALL
 SELECT 'user',u.id::text,'',u.name,'워크스페이스 구성원','/app/members',u.created_at,4::float8,'{}'::jsonb
 FROM users u JOIN workspace_members m ON m.user_id=u.id AND m.workspace_id=$2 WHERE $15 AND $16 AND u.disabled=false AND ($5='' OR $5='user') AND ($3='' OR u.name ILIKE $4)
 UNION ALL
 SELECT 'ai_conversation',m.id::text,'',c.title,
 substring(CASE WHEN m.question ILIKE $4 THEN m.question ELSE m.answer END from 1 for 600),
 '/app/ai-history?id='||c.id||'#'||m.id,m.created_at,
 (2+ts_rank_cd(to_tsvector('simple',m.question||' '||m.answer),websearch_to_tsquery('simple',$3)))::float8,
 jsonb_build_object('conversation_id',c.id,'action',m.action)
 FROM ai_messages m JOIN ai_conversations c ON c.id=m.conversation_id
 WHERE $15 AND c.owner_id=$1 AND c.workspace_id=$2 AND madi_ai_conversation_allowed($1,c.id)
 AND ($5='' OR $5='ai_conversation') AND $6='' AND $7='' AND $8='' AND $9='' AND NOT $12
 AND ($10::timestamptz IS NULL OR m.created_at>=$10) AND ($11::timestamptz IS NULL OR m.created_at<$11)
 AND ($3='' OR to_tsvector('simple',m.question||' '||m.answer)@@websearch_to_tsquery('simple',$3) OR m.question ILIKE $4 OR m.answer ILIKE $4)
)
SELECT CASE WHEN h.kind='document' THEN (
 SELECT to_jsonb(h)||jsonb_build_object(
  'snippet',substring(d.markdown from greatest(1,strpos(lower(d.markdown),lower($3))-100) for 600),
  'metadata',jsonb_build_object('version',d.version,'tags',` + documentSummaryTags + `,'tags_truncated',CASE WHEN jsonb_typeof(d.tags)='array' THEN jsonb_array_length(d.tags)>jsonb_array_length(` + documentSummaryTags + `) ELSE true END,'status',d.status,'owner_id',d.owner_id,'space_id',d.space_id))
 FROM documents d WHERE d.id=h.document_id::uuid
) ELSE to_jsonb(h) END FROM hits h`
