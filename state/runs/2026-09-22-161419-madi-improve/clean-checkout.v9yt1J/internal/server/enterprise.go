package server

import (
	"net/http"
	"strings"
)

var enterpriseEntityKinds = []string{"person", "team", "project", "system", "server", "application", "database", "technology", "vendor", "model", "policy"}

func (s *Server) registerEnterprise() {
	s.handle("GET /api/v1/enterprise/search", s.searchEnterprise)
	s.handle("GET /api/v1/enterprise/entities", s.listEnterpriseEntities)
	s.handle("POST /api/v1/enterprise/entities", s.createEnterpriseEntity)
	s.handle("GET /api/v1/enterprise/entities/{id}", s.getEnterpriseEntity)
	s.handle("PUT /api/v1/enterprise/entities/{id}", s.updateEnterpriseEntity)
}
func (s *Server) searchEnterprise(w http.ResponseWriter, r *http.Request) {
	wid := r.URL.Query().Get("workspace_id")
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(q) > 500 || !s.canWorkspace(r.Context(), current(r), wid, false) {
		apiError(w, 403, "접근 가능한 워크스페이스와 검색어를 확인하세요")
		return
	}
	p := current(r)
	docs := []map[string]any{}
	tables := []map[string]any{}
	queries := []map[string]any{}
	var e error
	if hasIntegrationScope(p, "document:read") || hasIntegrationScope(p, "search:read") {
		docs, e = s.rows(r.Context(), `SELECT jsonb_build_object('id',d.id,'title',d.title,'kind',coalesce(m.kind,'page'),'entity_type',m.system_metadata->>'entity_type','connector',c.name,'connector_kind',c.kind,'source_url',cr.source_url,'updated_at',d.updated_at) FROM documents d LEFT JOIN knowledge_document_meta m ON m.document_id=d.id LEFT JOIN connector_records cr ON cr.document_id=d.id LEFT JOIN connector_configs c ON c.id=cr.connector_id WHERE d.workspace_id=$1 AND d.deleted_at IS NULL AND madi_document_allowed($2,d.id,false) AND (c.id IS NULL OR madi_space_allowed($2,c.space_id,false)) AND (d.title ILIKE '%'||$3||'%' OR d.markdown ILIKE '%'||$3||'%' OR c.name ILIKE '%'||$3||'%') ORDER BY d.updated_at DESC LIMIT 100`, wid, p.ID, q)
		if e != nil {
			respond(w, nil, e)
			return
		}
	}
	if hasIntegrationScope(p, "database:read") {
		tables, e = s.rows(r.Context(), `SELECT jsonb_build_object('source_id',c.id,'source_name',c.name,'engine',c.kind,'schema_name',t.schema_name,'table_name',t.table_name,'description',t.description,'column_count',jsonb_array_length(t.columns),'inspected_at',t.inspected_at) FROM sql_source_tables t JOIN sql_sources c ON c.id=t.source_id WHERE c.workspace_id=$1 AND madi_space_allowed($2,c.space_id,false) AND (c.config->'tables') ? (t.schema_name||'.'||t.table_name) AND (t.schema_name||'.'||t.table_name ILIKE '%'||$3||'%' OR c.name ILIKE '%'||$3||'%' OR t.description ILIKE '%'||$3||'%') ORDER BY c.name,t.schema_name,t.table_name LIMIT 100`, wid, p.ID, q)
		if e != nil {
			respond(w, nil, e)
			return
		}
		queries, e = s.rows(r.Context(), `SELECT jsonb_build_object('id',q.id,'name',q.name,'source_id',c.id,'source_name',c.name,'schema',q.plan->>'schema','table',q.plan->>'table','enabled',q.enabled) FROM sql_source_queries q JOIN sql_sources c ON c.id=q.source_id WHERE c.workspace_id=$1 AND madi_space_allowed($2,c.space_id,false) AND (c.config->'tables') ? ((q.plan->>'schema')||'.'||(q.plan->>'table')) AND (q.name ILIKE '%'||$3||'%' OR c.name ILIKE '%'||$3||'%') ORDER BY q.updated_at DESC LIMIT 100`, wid, p.ID, q)
		if e != nil {
			respond(w, nil, e)
			return
		}
	}
	respond(w, map[string]any{"documents": docs, "tables": tables, "queries": queries, "limit_per_kind": 100}, nil)
}
func (s *Server) listEnterpriseEntities(w http.ResponseWriter, r *http.Request) {
	wid := r.URL.Query().Get("workspace_id")
	if !s.canWorkspace(r.Context(), current(r), wid, false) || !hasIntegrationScope(current(r), "document:read") {
		apiError(w, 403, "엔터티 읽기 권한이 없습니다")
		return
	}
	v, e := s.rows(r.Context(), `SELECT jsonb_build_object('id',d.id,'title',d.title,'entity_type',coalesce(m.system_metadata->>'entity_type','technology'),'updated_at',d.updated_at,'space_id',d.space_id) FROM documents d JOIN knowledge_document_meta m ON m.document_id=d.id WHERE d.workspace_id=$1 AND m.kind='entity' AND d.deleted_at IS NULL AND madi_document_allowed($2,d.id,false) ORDER BY d.title LIMIT 1000`, wid, current(r).ID)
	respond(w, v, e)
}
func (s *Server) createEnterpriseEntity(w http.ResponseWriter, r *http.Request) {
	var in map[string]any
	if decode(r, &in) != nil {
		apiError(w, 400, "엔터티 정보를 확인하세요")
		return
	}
	wid, space, title, kind := str(in, "workspace_id"), str(in, "space_id"), strings.TrimSpace(str(in, "title")), str(in, "entity_type")
	if title == "" || len(title) > 300 || len(str(in, "description")) > 20000 || !oneOf(kind, enterpriseEntityKinds...) {
		apiError(w, 400, "이름·종류·설명을 확인하세요")
		return
	}
	if !hasIntegrationScope(current(r), "document:write") || !s.canSpace(r.Context(), current(r), wid, space, true) {
		apiError(w, 403, "엔터티를 만들 공간의 작성 권한이 없습니다")
		return
	}
	visibility := str(in, "visibility")
	if visibility == "" {
		visibility = "workspace"
	}
	if !oneOf(visibility, "workspace", "private") {
		apiError(w, 400, "공개 범위를 확인하세요")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	id := newID()
	markdown := "# " + title + "\n\n" + str(in, "description")
	protected, e := s.ProtectDocumentTx(r.Context(), tx, current(r), id, wid, title, markdown)
	if e != nil {
		if !WriteProtectionError(w, e) {
			respond(w, nil, e)
		}
		return
	}
	title, markdown = protected.Title, protected.Markdown
	_, e = tx.Exec(r.Context(), `INSERT INTO documents(id,workspace_id,space_id,title,markdown,owner_id,visibility,tags) VALUES($1,$2,NULLIF($3,'')::uuid,$4,$5,$6,$7,'["entity"]')`, id, wid, space, title, markdown, current(r).ID, visibility)
	if e == nil {
		_, e = tx.Exec(r.Context(), `INSERT INTO knowledge_document_meta(document_id,kind,system_metadata) VALUES($1,'entity',jsonb_build_object('entity_type',$2::text,'entity_source','manual'))`, id, kind)
	}
	if e == nil {
		_, e = tx.Exec(r.Context(), `INSERT INTO document_versions(document_id,version,title,markdown,tags,user_id,block_metadata) SELECT id,version,title,markdown,tags,$2,block_metadata FROM documents WHERE id=$1`, id, current(r).ID)
	}
	if e == nil {
		e = s.enqueueEvent(r.Context(), tx, Event{Type: "document.created", WorkspaceID: wid, ResourceID: id, After: map[string]any{"id": id, "title": title, "status": "draft", "version": 1}})
	}
	if e == nil {
		e = s.recordAutomationEffect(r.Context(), tx, map[string]any{"id": id, "version": 1})
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "ENTITY_CREATE", id, map[string]any{"entity_type": kind})
	respond(w, map[string]any{"id": id}, nil)
}
func (s *Server) getEnterpriseEntity(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !hasIntegrationScope(current(r), "document:read") || !s.canDocument(r.Context(), current(r), id, false) {
		apiError(w, 404, "엔터티를 찾을 수 없습니다")
		return
	}
	meta, e := s.documentKnowledge(r, id)
	if e != nil || str(meta, "kind") != "entity" {
		apiError(w, 404, "엔터티 문서가 아닙니다")
		return
	}
	doc, e := s.document(r, id)
	if e != nil {
		respond(w, nil, e)
		return
	}
	targets := append([]string{str(doc, "title"), id}, listStrings(doc["aliases"])...)
	incoming, e := s.rows(r.Context(), enterpriseRelatedDocumentsSQL, str(doc, "workspace_id"), id, current(r).ID, lowerGraphNames(targets))
	if e != nil {
		respond(w, nil, e)
		return
	}
	links := []map[string]any{}
	truncated := len(incoming) > 2000
	if truncated {
		incoming = incoming[:2000]
	}
	for _, candidate := range incoming {
		if boolean(candidate, "deferred") {
			truncated = true
			continue
		}
		matched := boolean(candidate, "confirmed_relation") || boolean(candidate, "indexed_match")
		for _, link := range indexMarkdown(str(candidate, "markdown")).Links {
			for _, target := range targets {
				if strings.EqualFold(link, target) {
					matched = true
					break
				}
			}
			if matched {
				break
			}
		}
		if matched {
			delete(candidate, "markdown")
			delete(candidate, "deferred")
			delete(candidate, "indexed_match")
			links = append(links, candidate)
		}
	}
	tables := []map[string]any{}
	if hasIntegrationScope(current(r), "database:read") {
		tables, e = s.rows(r.Context(), `SELECT jsonb_build_object('source_id',c.id,'source_name',c.name,'engine',c.kind,'schema_name',t.schema_name,'table_name',t.table_name,'description',t.description,'query_count',(SELECT count(*) FROM sql_source_queries q WHERE q.source_id=c.id AND q.plan->>'schema'=t.schema_name AND q.plan->>'table'=t.table_name AND q.enabled)) FROM sql_source_tables t JOIN sql_sources c ON c.id=t.source_id WHERE c.workspace_id=$1 AND madi_space_allowed($2,c.space_id,false) AND t.document_ids ? $3 AND (c.config->'tables') ? (t.schema_name||'.'||t.table_name) ORDER BY c.name,t.table_name LIMIT 100`, str(doc, "workspace_id"), current(r).ID, id)
		if e != nil {
			respond(w, nil, e)
			return
		}
	}
	respond(w, map[string]any{"document": doc, "metadata": meta, "related_documents": links, "related_tables": tables, "backlink_scan_limit": 2000, "backlink_source_bytes_limit": 16 << 20, "backlink_truncated": truncated}, nil)
}
func (s *Server) updateEnterpriseEntity(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !hasIntegrationScope(current(r), "document:write") || !s.canDocument(r.Context(), current(r), id, true) {
		apiError(w, 403, "엔터티 관리 권한이 없습니다")
		return
	}
	meta, e := s.documentKnowledge(r, id)
	if e != nil || str(meta, "kind") != "entity" || !boolean(meta, "can_manage") {
		apiError(w, 403, "소유자 또는 공간 관리 권한이 필요합니다")
		return
	}
	var in map[string]any
	if decode(r, &in) != nil || !oneOf(str(in, "entity_type"), enterpriseEntityKinds...) {
		apiError(w, 400, "엔터티 종류를 확인하세요")
		return
	}
	_, e = s.DB.Exec(r.Context(), "UPDATE knowledge_document_meta SET system_metadata=jsonb_set(system_metadata,'{entity_type}',to_jsonb($2::text),true) WHERE document_id=$1", id, str(in, "entity_type"))
	if e == nil {
		s.audit(r, "ENTITY_TYPE_CHANGE", id, map[string]any{"entity_type": str(in, "entity_type")})
	}
	respond(w, map[string]bool{"ok": e == nil}, e)
}
