package server

import (
	"context"
	_ "embed"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

//go:embed knowledge_graph.sql
var knowledgeGraphSchema string

const graphNodesSQL = `WITH candidates AS (
 SELECT d.id,d.title,d.version,d.tags,d.aliases,d.icon,d.space_id,d.parent_id,d.owner_id,d.updated_at,
 madi_document_allowed($2,d.id,true) AS can_write,coalesce(x.document_version=d.version AND x.links_indexed,false) AS indexed,
 CASE WHEN x.document_version=d.version AND x.links_indexed THEN x.links ELSE '[]'::jsonb END AS links,
 CASE WHEN x.document_version=d.version AND x.links_indexed THEN x.source_tags ELSE '[]'::jsonb END AS source_tags
 FROM documents d LEFT JOIN search_index_documents x ON x.document_id=d.id
 WHERE d.workspace_id=$1 AND d.deleted_at IS NULL AND madi_document_allowed($2,d.id,false)
 ORDER BY d.updated_at DESC,d.id LIMIT 2001
), bounded AS (
 SELECT *,sum(octet_length(tags::text)+octet_length(aliases::text)+octet_length(links::text)+octet_length(source_tags::text)) OVER(ORDER BY updated_at DESC,id) AS metadata_bytes FROM candidates
) SELECT jsonb_build_object('id',id,'title',title,'version',version,'icon',icon,'space_id',space_id,'parent_id',parent_id,'owner_id',owner_id,'can_write',can_write,'indexed',indexed,'metadata_limited',metadata_bytes>16777216,
 'tags',CASE WHEN metadata_bytes<=16777216 THEN tags ELSE '[]'::jsonb END,
 'aliases',CASE WHEN metadata_bytes<=16777216 THEN aliases ELSE '[]'::jsonb END,
 'links',CASE WHEN metadata_bytes<=16777216 THEN links ELSE '[]'::jsonb END,
 'source_tags',CASE WHEN metadata_bytes<=16777216 THEN source_tags ELSE '[]'::jsonb END)
 FROM bounded ORDER BY updated_at DESC,id`

func (s *Server) migrateKnowledgeGraph(ctx context.Context) error {
	_, e := s.DB.Exec(ctx, knowledgeGraphSchema)
	return e
}
func (s *Server) registerKnowledgeGraph() {
	s.handle("POST /api/v1/documents/{id}/relations", s.mutateDocumentRelation)
	s.handle("DELETE /api/v1/documents/{id}/relations/{target}", s.mutateDocumentRelation)
}

func (s *Server) knowledgeGraph(w http.ResponseWriter, r *http.Request) {
	p := current(r)
	wid := r.URL.Query().Get("workspace_id")
	if wid == "" {
		wid = p.WorkspaceID
	}
	if !validID(wid) || !s.canWorkspace(r.Context(), p, wid, false) || (!hasIntegrationScope(p, "document:read") && !hasIntegrationScope(p, "search:read")) {
		apiError(w, 403, "접근 가능한 워크스페이스를 선택하세요")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	const limit = 2000
	nodes, e := s.rows(ctx, graphNodesSQL, wid, p.ID)
	if e != nil {
		respond(w, nil, e)
		return
	}
	truncated := len(nodes) > limit
	if truncated {
		nodes = nodes[:limit]
	}
	ids, pendingIDs := []string{}, []string{}
	byID := map[string]map[string]any{}
	for _, node := range nodes {
		truncated = truncated || boolean(node, "metadata_limited")
		id := str(node, "id")
		ids = append(ids, id)
		byID[id] = node
		if !boolean(node, "indexed") {
			pendingIDs = append(pendingIDs, id)
		}
		if !hasIntegrationScope(p, "document:write") {
			node["can_write"] = false
		}
	}
	// Newly saved documents remain useful while the local worker catches up.
	// Bound fresh parsing by both document count and cumulative source bytes.
	if len(pendingIDs) > 0 {
		fresh, err := s.rows(ctx, `WITH candidates AS (SELECT d.id,d.version,d.markdown,sum(octet_length(d.markdown)) OVER(ORDER BY d.updated_at DESC,d.id) AS bytes FROM documents d WHERE d.id=ANY($1::uuid[]) AND d.deleted_at IS NULL AND madi_document_allowed($2,d.id,false) ORDER BY d.updated_at DESC,d.id LIMIT 64) SELECT jsonb_build_object('id',id,'version',version,'markdown',markdown) FROM candidates WHERE bytes<=16777216`, pendingIDs, p.ID)
		if err != nil {
			respond(w, nil, err)
			return
		}
		for _, row := range fresh {
			node := byID[str(row, "id")]
			if node == nil || number(node, "version", 0) != number(row, "version", -1) {
				continue
			}
			links, tags := projectGraphReferences(str(row, "markdown"))
			node["links"], node["source_tags"] = links, tags
		}
	}
	edges, unresolved := resolveGraphReferences(nodes)
	if len(ids) > 0 {
		manual, err := s.rows(ctx, `SELECT jsonb_build_object('source',x.source_id,'target',x.target_id,'type',x.type,'origin','manual') FROM document_relations x WHERE x.source_id=ANY($1::uuid[]) AND x.target_id=ANY($1::uuid[]) AND madi_document_allowed($2,x.source_id,false) AND madi_document_allowed($2,x.target_id,false) ORDER BY x.source_id,x.target_id,x.type LIMIT 20001`, ids, p.ID)
		if err != nil {
			respond(w, nil, err)
			return
		}
		edges = append(edges, manual...)
	}
	truncated = truncated || len(edges) >= 20000 || len(unresolved) >= 5000
	if len(edges) > 20000 {
		edges = edges[:20000]
	}
	for _, node := range nodes {
		tagSet := map[string]bool{}
		for _, tag := range append(listStrings(node["tags"]), listStrings(node["source_tags"])...) {
			tagSet[tag] = true
		}
		tags := []string{}
		for tag := range tagSet {
			tags = append(tags, tag)
		}
		sort.Strings(tags)
		node["tags"] = tags
		delete(node, "links")
		delete(node, "source_tags")
		delete(node, "aliases")
	}
	jsonResponse(w, 200, map[string]any{"nodes": nodes, "edges": edges, "unresolved": unresolved, "diagnostics": map[string]any{"limit": limit, "truncated": truncated, "pending": len(pendingIDs), "notice": "현재 열람 가능한 최근 2,000개 문서·2만 연결·5천 미해결 링크 범위입니다. 미해결 링크는 삭제뿐 아니라 권한·동명 문서·표시 범위·색인 대기 때문에 나타날 수 있습니다. 메타데이터는 16MiB, 새 원문 검사는 최대 64개·16MiB입니다."}})
}

func resolveGraphReferences(nodes []map[string]any) ([]map[string]any, []map[string]any) {
	lookup := map[string]map[string]bool{}
	ids := map[string]bool{}
	for _, node := range nodes {
		id := str(node, "id")
		ids[id] = true
		for _, name := range documentNames(node) {
			key := strings.ToLower(strings.TrimSpace(name))
			if lookup[key] == nil {
				lookup[key] = map[string]bool{}
			}
			lookup[key][id] = true
		}
	}
	edges, unresolved := []map[string]any{}, []map[string]any{}
	seen := map[string]bool{}
	add := func(source, target, kind, origin string) {
		key := source + "/" + target + "/" + kind + "/" + origin
		if !seen[key] && len(edges) < 20000 {
			edges = append(edges, map[string]any{"source": source, "target": target, "type": kind, "origin": origin})
			seen[key] = true
		}
	}
	for _, node := range nodes {
		id := str(node, "id")
		for _, target := range listStrings(node["links"]) {
			key := strings.ToLower(strings.TrimSpace(target))
			if validID(key) && ids[key] {
				add(id, key, "reference", "wiki")
				continue
			}
			matches := lookup[key]
			if len(matches) == 1 {
				for found := range matches {
					add(id, found, "reference", "wiki")
				}
			} else {
				reason := "unresolved"
				if len(matches) > 1 {
					reason = "ambiguous"
				}
				if len(unresolved) < 5000 {
					unresolved = append(unresolved, map[string]any{"source": id, "target": target, "reason": reason})
				}
			}
		}
		if parent := str(node, "parent_id"); ids[parent] {
			add(id, parent, "parent", "tree")
		}
	}
	return edges, unresolved
}

func (s *Server) mutateDocumentRelation(w http.ResponseWriter, r *http.Request) {
	id, target, kind, expected := r.PathValue("id"), r.PathValue("target"), r.URL.Query().Get("type"), 0
	if r.Method == http.MethodPost {
		var in struct {
			Target  string `json:"target_id"`
			Type    string `json:"type"`
			Version int    `json:"expected_version"`
		}
		if decode(r, &in) != nil {
			apiError(w, 400, "관계 입력값을 확인하세요")
			return
		}
		target, kind, expected = in.Target, in.Type, in.Version
	} else {
		expected, _ = strconv.Atoi(r.URL.Query().Get("expected_version"))
	}
	if !validID(id) || !validID(target) || id == target || !oneOf(kind, "related", "reference", "policy", "execution", "data") || expected < 1 {
		apiError(w, 400, "같은 워크스페이스의 대상 문서·관계 종류·현재 문서 버전을 확인하세요")
		return
	}
	p := current(r)
	if !hasIntegrationScope(p, "document:write") || !hasIntegrationScope(p, "document:read") || !s.canDocument(r.Context(), p, id, true) || !s.canDocument(r.Context(), p, target, false) {
		apiError(w, 403, "원본 작성 권한과 대상 열람 권한이 필요합니다")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	rows, e := tx.Query(r.Context(), "SELECT id::text,workspace_id::text,version FROM documents WHERE id=ANY($1::uuid[]) AND deleted_at IS NULL AND madi_document_allowed($2,id,id=$3::uuid) ORDER BY id FOR UPDATE", []string{id, target}, p.ID, id)
	if e != nil {
		respond(w, nil, e)
		return
	}
	versions, wids := map[string]int{}, map[string]string{}
	for rows.Next() {
		var doc, ws string
		var version int
		if e = rows.Scan(&doc, &ws, &version); e != nil {
			break
		}
		versions[doc], wids[doc] = version, ws
	}
	rows.Close()
	if e == nil {
		e = rows.Err()
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	if len(versions) != 2 || wids[id] != wids[target] || (p.WorkspaceID != "" && p.WorkspaceID != wids[id]) {
		apiError(w, 403, "같은 워크스페이스에서 현재 접근 가능한 문서만 연결할 수 있습니다")
		return
	}
	if versions[id] != expected {
		apiError(w, 409, "원본 문서가 변경되었습니다. 다시 불러오세요")
		return
	}
	if r.Method == http.MethodPost {
		_, e = tx.Exec(r.Context(), "INSERT INTO document_relations(source_id,target_id,type,created_by) VALUES($1,$2,$3,$4) ON CONFLICT DO NOTHING", id, target, kind, p.ID)
	} else {
		_, e = tx.Exec(r.Context(), "DELETE FROM document_relations WHERE source_id=$1 AND target_id=$2 AND type=$3", id, target, kind)
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "DOCUMENT_RELATION_CHANGE", id, map[string]any{"target_id": target, "type": kind, "removed": r.Method == http.MethodDelete})
	jsonResponse(w, 200, map[string]any{"ok": true, "source": id, "target": target, "type": kind})
}

// Keep the existing array contract, with standard offset pagination. Resolve
// name collisions against current ACL before accepting a textual backlink.
func (s *Server) graphBacklinks(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	r = r.WithContext(ctx)
	id := r.PathValue("id")
	p := current(r)
	if !s.canDocument(r.Context(), p, id, false) {
		apiError(w, 404, "문서에 접근할 수 없습니다")
		return
	}
	target, e := s.document(r, id)
	if e != nil {
		respond(w, nil, e)
		return
	}
	names := documentNames(target)
	rows, e := s.rows(r.Context(), `SELECT jsonb_build_object('name',n.name,'count',(SELECT count(*) FROM (SELECT d.id FROM documents d WHERE d.workspace_id=$1 AND d.deleted_at IS NULL AND madi_document_allowed($2,d.id,false) AND (lower(d.title)=n.name OR EXISTS(SELECT 1 FROM jsonb_array_elements_text(d.aliases) a WHERE lower(a)=n.name)) LIMIT 2) matches)) FROM unnest($3::text[]) n(name)`, str(target, "workspace_id"), p.ID, lowerGraphNames(names))
	if e != nil {
		respond(w, nil, e)
		return
	}
	counts := map[string]int{}
	for _, row := range rows {
		counts[str(row, "name")] = number(row, "count", 0)
	}
	unique := []string{strings.ToLower(id)}
	for _, name := range names {
		name = strings.ToLower(name)
		if counts[name] == 1 {
			unique = append(unique, name)
		}
	}
	offset := 0
	if raw := r.URL.Query().Get("offset"); raw != "" {
		offset, e = strconv.Atoi(raw)
		if e != nil || offset < 0 || offset > 10000 {
			apiError(w, 400, "페이지 범위를 확인하세요")
			return
		}
	}
	// Fresh source candidates are bounded; indexed matches can cover the whole
	// workspace without transferring every Markdown body to the API process.
	candidates, e := s.rows(r.Context(), graphBacklinksSQL, str(target, "workspace_id"), p.ID, id, unique, offset)
	if e != nil {
		respond(w, nil, e)
		return
	}
	more := len(candidates) > 200
	if more {
		candidates = candidates[:200]
	}
	out := []map[string]any{}
	processed := 0
	for _, row := range candidates {
		if boolean(row, "deferred") {
			more = true
			break
		}
		processed++
		matched := boolean(row, "manual")
		links := listStrings(row["links"])
		md := str(row, "markdown")
		if md != "" {
			links, _ = projectGraphReferences(md)
		}
		for _, link := range links {
			if oneOf(strings.ToLower(link), unique...) {
				matched = true
				break
			}
		}
		delete(row, "links")
		delete(row, "markdown")
		delete(row, "manual")
		delete(row, "deferred")
		if matched {
			out = append(out, row)
		}
	}
	w.Header().Set("X-Madi-Has-More", strconv.FormatBool(more))
	w.Header().Set("X-Madi-Next-Offset", strconv.Itoa(offset+processed))
	jsonResponse(w, 200, out)
}

const graphBacklinksSQL = `WITH candidates AS (
 SELECT d.id,d.title,d.icon,d.version,d.updated_at,
 CASE WHEN x.document_version=d.version AND x.links_indexed THEN '' ELSE d.markdown END AS markdown,
 CASE WHEN x.document_version=d.version AND x.links_indexed THEN x.links ELSE '[]'::jsonb END AS links,
 EXISTS(SELECT 1 FROM document_relations rel WHERE rel.source_id=d.id AND rel.target_id=$3) AS manual
 FROM documents d LEFT JOIN search_index_documents x ON x.document_id=d.id
 WHERE d.workspace_id=$1 AND d.id<>$3 AND d.deleted_at IS NULL AND madi_document_allowed($2,d.id,false)
 AND (EXISTS(SELECT 1 FROM document_relations rel WHERE rel.source_id=d.id AND rel.target_id=$3)
 OR (x.document_version=d.version AND x.links_indexed AND EXISTS(SELECT 1 FROM jsonb_array_elements_text(x.links) link WHERE lower(link)=ANY($4::text[])))
 OR ((x.document_version IS DISTINCT FROM d.version OR NOT coalesce(x.links_indexed,false)) AND EXISTS(SELECT 1 FROM unnest($4::text[]) name WHERE strpos(lower(d.markdown),name)>0)))
 ORDER BY d.updated_at DESC,d.id LIMIT 201 OFFSET $5
), bounded AS (
 SELECT *,sum(octet_length(markdown)+octet_length(links::text)) OVER(ORDER BY updated_at DESC,id) AS bytes FROM candidates
) SELECT jsonb_build_object('id',id,'title',title,'icon',icon,'version',version,'updated_at',updated_at,'manual',manual,'deferred',bytes>16777216,
 'markdown',CASE WHEN bytes<=16777216 THEN markdown ELSE '' END,'links',CASE WHEN bytes<=16777216 THEN links ELSE '[]'::jsonb END)
 FROM bounded ORDER BY updated_at DESC,id`

func lowerGraphNames(names []string) []string {
	out := []string{}
	for _, name := range names {
		out = append(out, strings.ToLower(name))
	}
	return out
}
