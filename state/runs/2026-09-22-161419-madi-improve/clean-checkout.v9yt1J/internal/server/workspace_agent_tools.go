package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"sort"
	"strings"
	"unicode/utf8"
)

type agentDBQuery struct {
	DatabaseID  string           `json:"database_id"`
	PropertyIDs []string         `json:"property_ids"`
	Filters     []advancedFilter `json:"filters"`
	Limit       int              `json:"limit"`
}
type agentWritePlan struct {
	DocumentID      string `json:"document_id,omitempty"`
	ExpectedVersion int    `json:"expected_version,omitempty"`
	SpaceID         string `json:"space_id,omitempty"`
	Title           string `json:"title"`
	Markdown        string `json:"markdown"`
	Visibility      string `json:"visibility,omitempty"`
}

func agentDocSource(id string, version int, title string) agentSource {
	return agentSource{Key: "document:" + id, Kind: "document", ResourceID: id, Snapshot: map[string]any{"version": version, "title": title}, Fingerprint: fmt.Sprint(version)}
}

func (s *Server) agentReadTool(ctx context.Context, p *Principal, a workspaceAgent, call agentToolCall) (any, []agentSource, error) {
	if !s.canFeature(ctx, p, a.WorkspaceID, "workspace-agents") {
		return nil, nil, errAgentChanged
	}
	sources := []agentSource{}
	if !slices.Contains(a.Tools, call.Function.Name) {
		return nil, nil, errors.New("설정에서 허용하지 않은 도구입니다")
	}
	if call.Function.Name == "query_database" {
		var in agentDBQuery
		if e := agentStrictJSON([]byte(call.Function.Arguments), &in); e != nil {
			return nil, nil, e
		}
		result, fingerprint, e := s.agentQueryDatabase(ctx, p, a, in)
		if e != nil {
			return nil, nil, e
		}
		dependencies := result["_dependencies"]
		delete(result, "_dependencies")
		return result, []agentSource{{Key: "database:" + digest(string(jsonValue(in))), Kind: "database", ResourceID: in.DatabaseID, Snapshot: map[string]any{"query": in, "dependencies": dependencies}, Fingerprint: fingerprint}}, nil
	}
	if !hasIntegrationScope(p, "document:read") {
		return nil, nil, errors.New("문서 읽기 권한이 필요합니다")
	}
	switch call.Function.Name {
	case "get_document":
		var in struct {
			DocumentID string `json:"document_id"`
			StartByte  int    `json:"start_byte"`
		}
		if e := agentStrictJSON([]byte(call.Function.Arguments), &in); e != nil {
			return nil, nil, e
		}
		if !agentDocumentAllowed(ctx, s.DB, p, a, in.DocumentID, false) {
			return nil, nil, errors.New("허용된 지식 범위 안의 접근 가능한 문서를 선택하세요")
		}
		var md, title string
		var version int
		e := s.DB.QueryRow(ctx, `SELECT title,markdown,version FROM documents d WHERE id=$5 AND workspace_id=$2 AND deleted_at IS NULL AND madi_document_allowed($1,d.id,false) AND `+agentDocScopeSQL, p.ID, a.WorkspaceID, a.DocumentIDs, a.SpaceIDs, in.DocumentID).Scan(&title, &md, &version)
		if e != nil {
			return nil, nil, e
		}
		if in.StartByte < 0 || in.StartByte > len(md) || in.StartByte < len(md) && !utf8.RuneStart(md[in.StartByte]) {
			return nil, nil, errors.New("문서 읽기 시작 바이트가 올바르지 않습니다")
		}
		end := min(in.StartByte+16384, len(md))
		for end < len(md) && !utf8.RuneStart(md[end]) {
			end--
		}
		chunk := md[in.StartByte:end]
		sources = append(sources, agentDocSource(in.DocumentID, version, title))
		first := strings.Count(md[:in.StartByte], "\n") + 1
		last := first + strings.Count(strings.TrimSuffix(chunk, "\n"), "\n")
		citation := sourceFromChunk(in.DocumentID, title, version, ragChunk{Start: in.StartByte, End: end, StartLine: first, EndLine: last, Content: chunk, Hash: digest(chunk)})
		return map[string]any{"id": in.DocumentID, "title": title, "version": version, "markdown": chunk, "start_byte": in.StartByte, "end_byte": end, "total_bytes": len(md), "content_hash": digest(chunk), "url": "/app/documents/" + in.DocumentID, "source": citation}, sources, nil
	case "search_documents", "get_graph":
		var in struct {
			Query string `json:"query"`
		}
		if e := agentStrictJSON([]byte(call.Function.Arguments), &in); e != nil {
			return nil, nil, e
		}
		if len(in.Query) > 1024 {
			return nil, nil, errors.New("검색어는 1024바이트 이하여야 합니다")
		}
		limit := 6
		if call.Function.Name == "get_graph" {
			limit = 101
		}
		rows, e := s.DB.Query(ctx, `SELECT d.id::text,d.title,d.version,substring(d.markdown for 24000) FROM documents d WHERE d.workspace_id=$2 AND d.deleted_at IS NULL AND madi_document_allowed($1,d.id,false) AND `+agentDocScopeSQL+` AND ($5='' OR d.search_vector@@websearch_to_tsquery('simple',$5) OR d.title ILIKE '%'||$5||'%' OR d.markdown ILIKE '%'||$5||'%') ORDER BY CASE WHEN $5<>'' THEN ts_rank_cd(d.search_vector,websearch_to_tsquery('simple',$5)) ELSE 0 END DESC,d.id LIMIT $6`, p.ID, a.WorkspaceID, a.DocumentIDs, a.SpaceIDs, in.Query, limit)
		if e != nil {
			return nil, nil, e
		}
		items := []map[string]any{}
		for rows.Next() {
			var id, title, md string
			var version int
			if e = rows.Scan(&id, &title, &version, &md); e != nil {
				break
			}
			sources = append(sources, agentDocSource(id, version, title))
			item := map[string]any{"id": id, "title": title, "version": version, "url": "/app/documents/" + id}
			if call.Function.Name == "search_documents" {
				chunks, ex := chunkMarkdown(md, 6144, 0)
				if ex != nil {
					e = ex
					break
				}
				if len(chunks) > 0 {
					chosen := chunks[0]
					for _, c := range chunks {
						if strings.Contains(strings.ToLower(c.Content), strings.ToLower(in.Query)) {
							chosen = c
							break
						}
					}
					source := sourceFromChunk(id, title, version, chosen)
					item["source"] = source
					item["markdown"] = source.Markdown
				}
			}
			items = append(items, item)
		}
		if e == nil {
			e = rows.Err()
		}
		rows.Close()
		if e != nil {
			return nil, nil, e
		}
		if call.Function.Name == "search_documents" {
			return map[string]any{"documents": items, "limit": limit}, sources, nil
		}
		truncated := len(items) > 100
		if truncated {
			items, sources = items[:100], sources[:100]
		}
		// Links are resolved only among the already-authorized node set. A link
		// to a hidden node cannot disclose its title through graph output.
		ids := []string{}
		for _, src := range sources {
			ids = append(ids, src.ResourceID)
		}
		edges := []map[string]any{}
		edgesTruncated := false
		if len(ids) > 0 {
			docs, e := s.rows(ctx, `SELECT `+limitedGraphDocumentJSON+` FROM documents d WHERE d.id=ANY($5::uuid[]) AND d.workspace_id=$2 AND d.deleted_at IS NULL AND madi_document_allowed($1,d.id,false) AND `+agentDocScopeSQL, p.ID, a.WorkspaceID, a.DocumentIDs, a.SpaceIDs, ids)
			if e != nil {
				return nil, nil, e
			}
			if len(docs) != len(ids) {
				return nil, nil, errAgentChanged
			}
			lookup := map[string]string{}
			for _, d := range docs {
				truncated = truncated || boolean(d, "source_truncated") || boolean(d, "aliases_truncated")
				lookup[str(d, "title")] = str(d, "id")
				for _, alias := range listStrings(d["aliases"]) {
					lookup[alias] = str(d, "id")
				}
			}
			for _, d := range docs {
				for _, target := range wikiTargets(str(d, "markdown")) {
					if id := lookup[target]; id != "" {
						if len(edges) >= 500 {
							edgesTruncated = true
							break
						}
						edges = append(edges, map[string]any{"source": str(d, "id"), "target": id})
					}
				}
				if edgesTruncated {
					break
				}
			}
		}
		return map[string]any{"nodes": items, "edges": edges, "limit": 100, "source_character_limit": 32000, "alias_limit": 32, "truncated": truncated || edgesTruncated, "notice": "전체 그래프가 아닙니다. 현재 허용된 최대 100문서·500연결, 문서당 앞 32,000자와 제한된 alias 32개를 확인합니다."}, sources, nil
	}
	return nil, nil, errors.New("읽기 도구가 아닙니다")
}

func (s *Server) agentQueryDatabase(ctx context.Context, p *Principal, a workspaceAgent, in agentDBQuery) (map[string]any, string, error) {
	if !hasIntegrationScope(p, "database:read") || !slices.Contains(a.DatabaseIDs, in.DatabaseID) || len(in.PropertyIDs) < 1 || len(in.PropertyIDs) > 20 || len(in.Filters) > 10 || in.Limit < 1 || in.Limit > 100 {
		return nil, "", errors.New("허용 DB·속성(1~20)·행 수(1~100)·필터를 확인하세요")
	}
	r := automationRequest(ctx, p, http.MethodPost, nil)
	engine := newAdvancedEngine(s, r)
	engine.authorizeDatabase = func(id string) bool { return slices.Contains(a.DatabaseIDs, id) }
	db, e := engine.database(in.DatabaseID)
	if e != nil || db.workspaceID != a.WorkspaceID {
		return nil, "", errors.New("데이터베이스에 접근할 수 없습니다")
	}
	seen := map[string]bool{}
	for _, id := range in.PropertyIDs {
		if db.lookup[id] == nil || seen[id] {
			return nil, "", errors.New("속성 ID가 없거나 중복입니다")
		}
		seen[id] = true
	}
	for _, f := range in.Filters {
		if db.lookup[f.PropertyID] == nil || !oneOf(f.Operator, "eq", "contains", "gt", "gte", "lt", "lte") {
			return nil, "", errors.New("필터 속성 또는 연산자가 올바르지 않습니다")
		}
	}
	rows, e := s.rows(ctx, `SELECT to_jsonb(v) FROM database_rows v WHERE database_id=$1 ORDER BY id LIMIT 501`, in.DatabaseID)
	if e != nil {
		return nil, "", e
	}
	truncated := len(rows) > 500
	if truncated {
		rows = rows[:500]
	}
	out := []map[string]any{}
	for _, row := range rows {
		engine.rows[in.DatabaseID+":"+str(row, "id")] = row
		match := true
		for _, f := range in.Filters {
			value, ex := engine.value(db, row, f.PropertyID, 0)
			if ex != nil {
				return nil, "", ex
			}
			if !advancedFilterMatches(value, f) {
				match = false
				break
			}
		}
		if !match {
			continue
		}
		values := map[string]any{}
		for _, id := range in.PropertyIDs {
			v, ex := engine.value(db, row, id, 0)
			if ex != nil {
				return nil, "", ex
			}
			values[id] = v
		}
		out = append(out, map[string]any{"id": row["id"], "values": values})
		if len(out) >= in.Limit {
			break
		}
	}
	columns := []map[string]any{}
	for _, id := range in.PropertyIDs {
		columns = append(columns, db.lookup[id])
	}
	result := map[string]any{"database_id": in.DatabaseID, "columns": columns, "rows": out, "scan_limit": 500, "truncated": truncated}
	if len(jsonValue(result)) > 65536 {
		return nil, "", errors.New("DB 도구 결과가 64KB를 초과합니다. 속성이나 행 수를 줄이세요")
	}
	// The fingerprint covers every actual engine dependency, not merely rendered
	// values. Formula/schema/related-row changes and scope revocation invalidate
	// the source before the next model request or streamed output.
	dbs := map[string]any{}
	for id, d := range engine.databases {
		dbs[id] = map[string]any{"workspace_id": d.workspaceID, "space_id": d.spaceID, "properties": d.properties}
	}
	fingerprint := digest(string(jsonValue(map[string]any{"query": in, "result": result, "databases": dbs, "rows": engine.rows})))
	dependencies := []map[string]any{}
	for id, value := range dbs {
		dependencies = append(dependencies, map[string]any{"kind": "database", "id": id, "fingerprint": digest(string(jsonValue(value)))})
	}
	for key, value := range engine.rows {
		parts := strings.SplitN(key, ":", 2)
		dependencies = append(dependencies, map[string]any{"kind": "row", "id": parts[1], "database_id": parts[0], "fingerprint": digest(string(jsonValue(value)))})
	}
	result["_dependencies"] = dependencies
	return result, fingerprint, nil
}

func (s *Server) agentValidatePlan(ctx context.Context, p *Principal, a workspaceAgent, call agentToolCall) (agentWritePlan, error) {
	var in agentWritePlan
	if e := agentStrictJSON([]byte(call.Function.Arguments), &in); e != nil {
		return in, e
	}
	if !slices.Contains(a.Tools, call.Function.Name) || !hasIntegrationScope(p, "document:write") || strings.TrimSpace(in.Title) == "" || len(in.Title) > 500 || len(in.Markdown) > 60000 {
		return in, errors.New("허용된 작성 도구와 제목·본문(60KB 이하)을 확인하세요")
	}
	if call.Function.Name == "update_document" {
		if in.SpaceID != "" || in.Visibility != "" || in.ExpectedVersion < 1 || !agentDocumentAllowed(ctx, s.DB, p, a, in.DocumentID, true) {
			return in, errors.New("수정할 문서·버전·작성 권한을 확인하세요")
		}
		var version int
		if s.DB.QueryRow(ctx, `SELECT version FROM documents WHERE id=$1 AND deleted_at IS NULL`, in.DocumentID).Scan(&version) != nil || version != in.ExpectedVersion {
			return in, errAgentChanged
		}
	}
	if call.Function.Name == "create_document" {
		if in.DocumentID != "" || in.ExpectedVersion != 0 || !oneOf(in.Visibility, "private", "workspace") || !slices.Contains(a.SpaceIDs, in.SpaceID) || !s.canSpace(ctx, p, a.WorkspaceID, in.SpaceID, true) {
			return in, errors.New("새 문서는 설정에 명시된 공간에만 만들 수 있습니다")
		}
	}
	if _, e := parseFrontMatter(in.Markdown); e != nil {
		return in, e
	}
	return in, nil
}

func (s *Server) agentSources(ctx context.Context, runID string) ([]agentSource, error) {
	rows, e := s.DB.Query(ctx, `SELECT source_key,kind,resource_id::text,snapshot,fingerprint FROM agent_sources WHERE run_id=$1 ORDER BY source_key`, runID)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []agentSource{}
	for rows.Next() {
		var src agentSource
		var raw []byte
		if e = rows.Scan(&src.Key, &src.Kind, &src.ResourceID, &raw, &src.Fingerprint); e != nil {
			return nil, e
		}
		if e = json.Unmarshal(raw, &src.Snapshot); e != nil {
			return nil, e
		}
		out = append(out, src)
	}
	return out, rows.Err()
}
func (s *Server) validateAgentSources(ctx context.Context, p *Principal, a workspaceAgent, runID string, versions bool) error {
	sources, e := s.agentSources(ctx, runID)
	if e != nil {
		return e
	}
	for _, src := range sources {
		switch src.Kind {
		case "document":
			if !agentDocumentAllowed(ctx, s.DB, p, a, src.ResourceID, false) {
				return errAgentChanged
			}
			if versions {
				var v int
				if s.DB.QueryRow(ctx, `SELECT version FROM documents WHERE id=$1 AND deleted_at IS NULL`, src.ResourceID).Scan(&v) != nil || fmt.Sprint(v) != src.Fingerprint {
					return errAgentChanged
				}
			}
		case "database":
			var query agentDBQuery
			if json.Unmarshal(jsonValue(src.Snapshot["query"]), &query) != nil {
				return errAgentChanged
			}
			_, fingerprint, e := s.agentQueryDatabase(ctx, p, a, query)
			if e != nil || versions && fingerprint != src.Fingerprint {
				return errAgentChanged
			}
		}
	}
	return nil
}

func agentSortedTools(a workspaceAgent, p *Principal) workspaceAgent {
	a.Tools = slices.DeleteFunc(slices.Clone(a.Tools), func(tool string) bool {
		switch tool {
		case "query_database":
			return !hasIntegrationScope(p, "database:read")
		case "create_document", "update_document":
			return !hasIntegrationScope(p, "document:write")
		default:
			return !hasIntegrationScope(p, "document:read")
		}
	})
	sort.Strings(a.Tools)
	return a
}
