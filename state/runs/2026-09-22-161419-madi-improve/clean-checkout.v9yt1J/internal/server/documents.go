package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"gopkg.in/yaml.v3"
)

const docJSON = "(to_jsonb(d)-'search_vector')||jsonb_build_object('is_favorite',EXISTS(SELECT 1 FROM favorites f WHERE f.document_id=d.id AND f.user_id=$1),'can_write',d.deleted_at IS NULL AND madi_document_allowed($1,d.id,true))"
const docACL = "madi_document_allowed($1,d.id,false)"

var wikiPattern = regexp.MustCompile(`\[\[([^\]\n]+)\]\]`)
var taskPattern = regexp.MustCompile(`^\s*(?:>\s*)*(?:[-*+]|\d+[.)])\s+\[([ xX])\](?:[\t ]+(.*))?$`)

func (s *Server) registerDocuments() {
	s.handle("GET /api/v1/documents", s.listDocuments)
	s.handle("POST /api/v1/documents", s.createDocument)
	s.handle("GET /api/v1/documents/{id}", s.getDocument)
	s.handle("PUT /api/v1/documents/{id}", s.updateDocument)
	s.handle("DELETE /api/v1/documents/{id}", s.deleteDocument)
	s.handle("POST /api/v1/documents/{id}/restore", s.restoreDocument)
	s.handle("POST /api/v1/documents/{id}/favorite", s.favoriteDocument)
	s.handle("GET /api/v1/documents/{id}/versions", s.documentVersions)
	s.handle("GET /api/v1/documents/{id}/versions/diff", s.documentVersionDiff)
	s.handle("GET /api/v1/documents/{id}/versions/{version}", s.documentVersionDetail)
	s.handle("POST /api/v1/documents/{id}/versions/{version}/restore", s.restoreVersion)
	s.handle("GET /api/v1/documents/{id}/backlinks", s.graphBacklinks)
	s.handle("GET /api/v1/documents/{id}/comments", s.discussionComments)
	s.handle("POST /api/v1/documents/{id}/comments", s.discussionAddComment)
	s.handle("GET /api/v1/documents/{id}/shares", s.documentShares)
	s.handle("PUT /api/v1/documents/{id}/shares", s.setDocumentShare)
	s.handle("POST /api/v1/documents/{id}/shares", s.addDocumentShare)
	s.handle("POST /api/v1/documents/{id}/approval", s.approval)
	s.handle("GET /api/v1/graph", s.knowledgeGraph)
	s.handle("GET /api/v1/tasks", s.tasks)
	s.handle("PUT /api/v1/tasks", s.updateTask)
}
func (s *Server) document(r *http.Request, id string) (map[string]any, error) {
	v, e := s.one(r.Context(), "SELECT "+docJSON+" FROM documents d WHERE d.id=$2", current(r).ID, id)
	if e != nil {
		return nil, e
	}
	p := current(r)
	v["can_write"] = s.canDocument(r.Context(), p, id, true) && hasIntegrationScope(p, "document:write") && v["deleted_at"] == nil
	var canComment bool
	_ = s.DB.QueryRow(r.Context(), "SELECT madi_document_comment_allowed($1,$2)", p.ID, id).Scan(&canComment)
	v["can_comment"] = p.Role != "viewer" && canComment && s.canDocument(r.Context(), p, id, false) && hasIntegrationScope(p, "document:write") && v["deleted_at"] == nil
	cfg, err := s.settings(r.Context())
	if err != nil {
		return nil, err
	}
	v["can_approve"] = false
	if boolean(cfg, "approval_enabled") && str(v, "status") == "review" && v["deleted_at"] == nil {
		v["can_approve"], err = s.canApproveDocument(r.Context(), p, id)
		if err != nil {
			return nil, err
		}
	}
	return v, nil
}
func (s *Server) listDocuments(w http.ResponseWriter, r *http.Request) {
	p := current(r)
	q := r.URL.Query()
	wid := q.Get("workspace_id")
	if p.WorkspaceID != "" {
		if wid != "" && wid != p.WorkspaceID {
			apiError(w, 403, "API 키의 워크스페이스 범위를 벗어났습니다")
			return
		}
		wid = p.WorkspaceID
	}
	if wid != "" && !validID(wid) {
		apiError(w, 400, "워크스페이스 ID 형식을 확인하세요")
		return
	}
	limit := 100
	if n, e := strconv.Atoi(q.Get("limit")); e == nil && n > 0 && n <= 2000 {
		limit = n
	}
	offset := 0
	if n, e := strconv.Atoi(q.Get("offset")); e == nil && n > 0 {
		offset = n
	}
	term := strings.TrimSpace(q.Get("q"))
	query, args := documentListQuery(p.ID, wid, term, q.Get("tag"), q.Get("trash") == "1", q.Get("favorite") == "1", limit, offset)
	v, e := s.rows(r.Context(), query, args...)
	if term != "" {
		s.audit(r, "SEARCH", wid, map[string]any{"query_bytes": len(term), "results": len(v)})
	}
	if !hasIntegrationScope(p, "document:write") {
		for _, d := range v {
			d["can_write"] = false
		}
	}
	respond(w, v, e)
}
func parseFrontMatter(md string) (map[string]any, error) {
	v := map[string]any{}
	if !strings.HasPrefix(md, "---\n") && !strings.HasPrefix(md, "---\r\n") {
		return v, nil
	}
	normalized := strings.ReplaceAll(md, "\r\n", "\n")
	end := strings.Index(normalized[4:], "\n---")
	if end < 0 {
		return v, fmt.Errorf("Front Matter 닫는 구분자(---)가 없습니다")
	}
	if err := yaml.Unmarshal([]byte(normalized[4:4+end]), &v); err != nil {
		return nil, fmt.Errorf("Front Matter YAML 형식을 확인하세요: %w", err)
	}
	return v, nil
}
func listStrings(v any) []string {
	out := []string{}
	switch a := v.(type) {
	case []any:
		for _, x := range a {
			if t, ok := x.(string); ok && t != "" {
				out = append(out, t)
			}
		}
	case []string:
		out = a
	case string:
		for _, t := range strings.Split(a, ",") {
			if t = strings.TrimSpace(t); t != "" {
				out = append(out, t)
			}
		}
	}
	return out
}
func (s *Server) createDocument(w http.ResponseWriter, r *http.Request) {
	var in map[string]any
	if decode(r, &in) != nil {
		apiError(w, 400, "문서 입력값을 확인하세요")
		return
	}
	wid := str(in, "workspace_id")
	if !s.canWorkspace(r.Context(), current(r), wid, true) {
		apiError(w, 403, "문서 작성 권한이 없습니다")
		return
	}
	title := strings.TrimSpace(str(in, "title"))
	if title == "" || len(title) > 500 {
		apiError(w, 400, "문서 제목은 1~500바이트여야 합니다")
		return
	}
	md := str(in, "markdown")
	if len(md) > 4<<20 {
		apiError(w, 400, "문서 크기는 4MB 이하여야 합니다")
		return
	}
	fm, e := parseFrontMatter(md)
	if e != nil {
		apiError(w, 400, e.Error())
		return
	}
	tags := listStrings(in["tags"])
	aliases := listStrings(in["aliases"])
	if t, ok := fm["tags"]; ok {
		tags = listStrings(t)
	}
	if t, ok := fm["aliases"]; ok {
		aliases = listStrings(t)
	}
	visibility := str(in, "visibility")
	if visibility == "" {
		visibility = "workspace"
	}
	if !oneOf(visibility, "workspace", "private", "selected") {
		apiError(w, 400, "문서 공개 범위를 확인하세요")
		return
	}
	parent := str(in, "parent_id")
	if parent != "" && (!s.validParent(r, "", wid, parent) || !s.canDocument(r.Context(), current(r), parent, true)) {
		apiError(w, 400, "같은 워크스페이스의 접근 가능한 상위 문서를 선택하세요")
		return
	}
	space := str(in, "space_id")
	if _, specified := in["space_id"]; !specified && parent != "" {
		_ = s.DB.QueryRow(r.Context(), "SELECT coalesce(space_id::text,'') FROM documents WHERE id=$1", parent).Scan(&space)
	}
	if !s.canSpace(r.Context(), current(r), wid, space, true) {
		apiError(w, 403, "문서를 만들 공간의 작성 권한이 없습니다")
		return
	}
	icon := str(in, "icon")
	if icon == "" {
		icon = "file"
	}
	if len(icon) > 64 {
		icon = "file"
	}
	metadata := in["block_metadata"]
	if metadata == nil {
		metadata = map[string]any{}
	}
	if _, ok := metadata.(map[string]any); !ok || len(jsonValue(metadata)) > 1<<20 {
		apiError(w, 400, "블록 메타데이터는 1MB 이하 JSON 객체여야 합니다")
		return
	}
	id := newID()
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	if e = s.validateAISelectionCreateTx(r, tx, wid, md); e != nil {
		apiError(w, 409, e.Error())
		return
	}
	if e = s.validateQuestionDraftTx(r, tx, wid, md); e != nil {
		apiError(w, 409, e.Error())
		return
	}
	if capture, ok := r.Context().Value(captureContextKey{}).(captureInput); ok && capture.RequestID != "" {
		if _, e = tx.Exec(r.Context(), "SELECT pg_advisory_xact_lock(hashtextextended($1,81))", current(r).ID+"/"+capture.RequestID); e != nil {
			respond(w, nil, e)
			return
		}
		var previousID, previousHash, previousWorkspace string
		e = tx.QueryRow(r.Context(), "SELECT coalesce(document_id::text,''),payload_hash,workspace_id::text FROM capture_receipts WHERE user_id=$1 AND request_id=$2", current(r).ID, capture.RequestID).Scan(&previousID, &previousHash, &previousWorkspace)
		if e == nil {
			// Receipt rows are immutable. Release the connection before current
			// ACL/response reads so concurrent replays cannot exhaust the pool.
			_ = tx.Rollback(r.Context())
			if previousHash != capture.PayloadHash || previousWorkspace != wid {
				apiError(w, 409, "같은 요청 ID로 다른 내용을 저장할 수 없습니다")
				return
			}
			if previousID == "" {
				apiError(w, 410, "이 요청으로 저장한 문서가 영구 삭제되었습니다")
				return
			}
			if !s.canDocument(r.Context(), current(r), previousID, false) {
				apiError(w, 404, "기존 수집 문서에 접근할 수 없습니다")
				return
			}
			v, err := s.document(r, previousID)
			if err == nil && v["deleted_at"] != nil {
				apiError(w, 410, "이 요청으로 저장한 문서가 휴지통에 있습니다")
				return
			}
			respond(w, v, err)
			return
		}
		if e != pgx.ErrNoRows {
			respond(w, nil, e)
			return
		}
		e = nil
	}
	if e = s.treePlacementTx(r, tx, "documents", wid, id, parent); e != nil {
		apiError(w, 409, e.Error())
		return
	}
	protected, e := s.protectCanonicalDocumentTx(r.Context(), tx, current(r), id, wid, title, md, tags, aliases, metadata)
	if e != nil {
		if !WriteProtectionError(w, e) {
			respond(w, nil, e)
		}
		return
	}
	title, md, tags, aliases, metadata = protected.Title, protected.Markdown, protected.Tags, protected.Aliases, protected.Metadata
	_, e = tx.Exec(r.Context(), "INSERT INTO documents(id,workspace_id,parent_id,title,markdown,tags,aliases,icon,visibility,owner_id,block_metadata,space_id) VALUES($1,$2,NULLIF($3,'')::uuid,$4,$5,$6,$7,$8,$9,$10,$11,NULLIF($12,'')::uuid)", id, wid, parent, title, md, jsonValue(tags), jsonValue(aliases), icon, visibility, current(r).ID, jsonValue(metadata), space)
	if e == nil {
		_, e = tx.Exec(r.Context(), "INSERT INTO document_versions(document_id,version,title,markdown,tags,user_id,block_metadata) SELECT id,version,title,markdown,tags,$2,block_metadata FROM documents WHERE id=$1", id, current(r).ID)
	}
	if capture, ok := r.Context().Value(captureContextKey{}).(captureInput); e == nil && ok {
		var source ProtectionMetadataResult
		source, e = s.ProtectDocumentMetadataTx(r.Context(), tx, current(r), id, wid, capture.Source)
		if e != nil {
			if !WriteProtectionError(w, e) {
				respond(w, nil, e)
			}
			return
		}
		protected.Changed = protected.Changed || source.Changed
		protected.Findings = append(protected.Findings, source.Findings...)
		_, e = tx.Exec(r.Context(), "INSERT INTO knowledge_document_meta(document_id,kind,system_metadata) VALUES($1,'inbox',$2)", id, jsonValue(source.Value))
		if e == nil && capture.RequestID != "" {
			_, e = tx.Exec(r.Context(), "INSERT INTO capture_receipts(user_id,request_id,workspace_id,document_id,payload_hash) VALUES($1,$2,$3,$4,$5)", current(r).ID, capture.RequestID, wid, id, capture.PayloadHash)
		}
	}
	if e == nil {
		if err := s.recordTemplateDocumentTx(r.Context(), tx, id); err != nil {
			apiError(w, 409, err.Error())
			return
		}
	}
	if e == nil {
		e = s.enqueueEvent(r.Context(), tx, Event{Type: "document.created", WorkspaceID: wid, ResourceID: id, ResourceType: "document", After: map[string]any{"id": id, "title": title, "status": "draft", "version": 1, "tags": tags}})
	}
	if e == nil {
		e = s.recordAutomationEffect(r.Context(), tx, map[string]any{"id": id, "version": 1})
	}
	if e == nil {
		e = s.finishQuestionDraftTx(r, tx, wid, id)
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "DOCUMENT_CREATE", id, nil)
	v, e := s.document(r, id)
	if e == nil {
		v["protection"] = protected.response()
	}
	respond(w, v, e)
}
func (s *Server) validParent(r *http.Request, id, wid, parent string) bool {
	if parent == "" {
		return true
	}
	if id == parent || !s.canDocument(r.Context(), current(r), parent, false) {
		return false
	}
	var actual string
	if s.DB.QueryRow(r.Context(), "SELECT workspace_id FROM documents WHERE id=$1 AND deleted_at IS NULL", parent).Scan(&actual) != nil || actual != wid {
		return false
	}
	if id != "" {
		var cycle bool
		if s.DB.QueryRow(r.Context(), "WITH RECURSIVE ancestors AS (SELECT id,parent_id FROM documents WHERE id=$1 UNION SELECT d.id,d.parent_id FROM documents d JOIN ancestors a ON d.id=a.parent_id) SELECT EXISTS(SELECT 1 FROM ancestors WHERE id=$2)", parent, id).Scan(&cycle) != nil || cycle {
			return false
		}
	}
	return true
}
func (s *Server) getDocument(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !s.canDocument(r.Context(), current(r), id, false) {
		apiError(w, 404, "문서를 찾을 수 없거나 접근 권한이 없습니다")
		return
	}
	v, e := s.document(r, id)
	s.audit(r, "DOCUMENT_READ", id, nil)
	respond(w, v, e)
}
func (s *Server) updateDocument(w http.ResponseWriter, r *http.Request) {
	var in map[string]any
	if decode(r, &in) != nil {
		apiError(w, 400, "문서 입력값을 확인하세요")
		return
	}
	s.saveDocument(w, r, r.PathValue("id"), in)
}
func (s *Server) saveDocument(w http.ResponseWriter, r *http.Request, id string, in map[string]any) {
	if !s.canDocument(r.Context(), current(r), id, true) {
		apiError(w, 403, "문서 수정 권한이 없습니다")
		return
	}
	old, e := s.document(r, id)
	if e != nil {
		respond(w, nil, e)
		return
	}
	if old["deleted_at"] != nil {
		apiError(w, 400, "휴지통 문서를 먼저 복원하세요")
		return
	}
	expected := number(in, "version", 0)
	if expected < 1 {
		apiError(w, 400, "충돌 방지를 위해 현재 문서 version이 필요합니다")
		return
	}
	title, md := str(old, "title"), str(old, "markdown")
	if v, ok := in["title"].(string); ok {
		title = strings.TrimSpace(v)
	}
	if v, ok := in["markdown"].(string); ok {
		md = v
	}
	if title == "" || len(title) > 500 || len(md) > 4<<20 {
		apiError(w, 400, "제목 또는 문서 크기를 확인하세요")
		return
	}
	fm, e := parseFrontMatter(md)
	if e != nil {
		apiError(w, 400, e.Error())
		return
	}
	tags, aliases := listStrings(old["tags"]), listStrings(old["aliases"])
	if v, ok := in["tags"]; ok {
		tags = listStrings(v)
	}
	if v, ok := in["aliases"]; ok {
		aliases = listStrings(v)
	}
	if v, ok := fm["tags"]; ok {
		tags = listStrings(v)
	}
	if v, ok := fm["aliases"]; ok {
		aliases = listStrings(v)
	}
	visibility, icon, status := str(old, "visibility"), str(old, "icon"), str(old, "status")
	if v, ok := in["visibility"].(string); ok {
		if current(r).ID != str(old, "owner_id") {
			apiError(w, 403, "소유자만 공유 범위를 변경할 수 있습니다")
			return
		}
		visibility = v
	}
	if !oneOf(visibility, "private", "selected", "workspace") {
		apiError(w, 400, "공유 범위를 확인하세요")
		return
	}
	if v, ok := in["icon"].(string); ok && len(v) < 65 {
		icon = v
	}
	if v, ok := in["status"].(string); ok {
		if !oneOf(v, "draft", "published", "stale", "archived") {
			apiError(w, 400, "문서 상태를 확인하세요")
			return
		}
		status = v
	}
	parent := str(old, "parent_id")
	if v, ok := in["parent_id"]; ok {
		parent, _ = v.(string)
	}
	if parent != str(old, "parent_id") && (current(r).ID != str(old, "owner_id") || (parent != "" && !s.canDocument(r.Context(), current(r), parent, true))) {
		apiError(w, 403, "소유자만 작성 권한이 있는 상위 문서로 이동할 수 있습니다")
		return
	}
	if !s.validParent(r, id, str(old, "workspace_id"), parent) {
		apiError(w, 400, "상위 문서 선택에 순환 또는 권한 오류가 있습니다")
		return
	}
	space := str(old, "space_id")
	if value, supplied := in["space_id"]; supplied {
		space, _ = value.(string)
		if space != str(old, "space_id") && current(r).ID != str(old, "owner_id") {
			apiError(w, 403, "소유자만 문서의 공간을 변경할 수 있습니다")
			return
		}
	}
	if !s.canSpace(r.Context(), current(r), str(old, "workspace_id"), space, true) {
		apiError(w, 403, "대상 공간의 작성 권한이 없습니다")
		return
	}
	metadata := old["block_metadata"]
	if md != str(old, "markdown") {
		metadata = map[string]any{}
	}
	if v, ok := in["block_metadata"]; ok {
		metadata = v
	}
	if metadata == nil {
		metadata = map[string]any{}
	}
	if _, ok := metadata.(map[string]any); !ok || len(jsonValue(metadata)) > 1<<20 {
		apiError(w, 400, "블록 메타데이터는 1MB 이하 JSON 객체여야 합니다")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	if e = s.prepareDocumentSplitTx(r, tx, str(old, "workspace_id")); e != nil {
		apiError(w, 409, e.Error())
		return
	}
	if e = s.prepareProposalConflictMergeTx(r, tx); e != nil {
		apiError(w, 409, e.Error())
		return
	}
	if parent != str(old, "parent_id") {
		if e = s.treePlacementTx(r, tx, "documents", str(old, "workspace_id"), id, parent); e != nil {
			apiError(w, 409, e.Error())
			return
		}
	}
	// Keep the same resource-before-policy lock order as collaboration and
	// approvals. Check the expected version before recording any policy event.
	var currentVersion int
	e = tx.QueryRow(r.Context(), "SELECT version FROM documents WHERE id=$1 AND deleted_at IS NULL AND madi_document_allowed($2,id,true) FOR UPDATE", id, current(r).ID).Scan(&currentVersion)
	if e == pgx.ErrNoRows {
		apiError(w, 403, "문서 수정 권한이 변경되었습니다")
		return
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	if currentVersion != expected {
		apiError(w, 409, "다른 사용자가 문서를 수정했습니다. 변경 내용을 보관하고 문서를 다시 불러오세요")
		return
	}
	if e = s.validateDocumentAccessChangeTx(r, tx, id, old, in, parent, space, visibility); e != nil {
		apiError(w, 409, e.Error())
		return
	}
	if e = s.validateProposalMergeTx(r, tx, id, expected, str(old, "markdown"), md); e != nil {
		apiError(w, 409, e.Error())
		return
	}
	if e = s.validateDocumentSplitTx(r, tx, id, old, expected, md); e != nil {
		apiError(w, 409, e.Error())
		return
	}
	protected, e := s.protectCanonicalDocumentTx(r.Context(), tx, current(r), id, str(old, "workspace_id"), title, md, tags, aliases, metadata)
	if e != nil {
		if !WriteProtectionError(w, e) {
			respond(w, nil, e)
		}
		return
	}
	title, md, tags, aliases, metadata = protected.Title, protected.Markdown, protected.Tags, protected.Aliases, protected.Metadata
	// Retain ordinary rename aliases, but do not silently reintroduce an old
	// sensitive title when a strict-policy edit is trying to remove it.
	if title != str(old, "title") && !oneOf(protected.Mode, "block", "mask") {
		aliases = append(aliases, str(old, "title"))
	}
	status, e = approvalSaveStatusTx(r.Context(), tx, id, old, map[string]any{"title": title, "markdown": md, "tags": tags, "aliases": aliases, "visibility": visibility, "parent_id": parent, "space_id": space, "block_metadata": metadata}, status)
	if e != nil {
		approvalRespondError(w, e)
		return
	}
	tag, e := tx.Exec(r.Context(), "UPDATE documents SET title=$1,markdown=$2,tags=$3,aliases=$4,visibility=$5,icon=$6,status=$7,parent_id=NULLIF($8,'')::uuid,version=version+1,updated_at=now(),block_metadata=$11,space_id=NULLIF($12,'')::uuid WHERE id=$9 AND version=$10 AND deleted_at IS NULL AND madi_document_allowed($13,id,true) AND (NULLIF($8,'') IS NULL OR EXISTS(SELECT 1 FROM documents target WHERE target.id=NULLIF($8,'')::uuid AND target.workspace_id=documents.workspace_id AND target.deleted_at IS NULL AND madi_document_allowed($13,target.id,true))) AND madi_space_allowed($13,NULLIF($12,'')::uuid,true)", title, md, jsonValue(tags), jsonValue(aliases), visibility, icon, status, parent, id, expected, jsonValue(metadata), space, current(r).ID)
	if e != nil {
		respond(w, nil, e)
		return
	}
	if tag.RowsAffected() == 0 {
		apiError(w, 409, "다른 사용자가 문서를 수정했습니다. 변경 내용을 보관하고 문서를 다시 불러오세요")
		return
	}
	if e = s.saveTaskDetailsTx(r, tx, id, str(old, "workspace_id")); e != nil {
		if e == errTaskAssignee || e == errTaskReviewDisabled {
			apiError(w, 400, e.Error())
		} else {
			respond(w, nil, e)
		}
		return
	}
	_, e = tx.Exec(r.Context(), "INSERT INTO document_versions(document_id,version,title,markdown,tags,user_id,block_metadata) SELECT id,version,title,markdown,tags,$2,block_metadata FROM documents WHERE id=$1", id, current(r).ID)
	if e == nil {
		e = s.enqueueEvent(r.Context(), tx, Event{Type: "document.updated", WorkspaceID: str(old, "workspace_id"), ResourceID: id, ResourceType: "document", Before: old, After: map[string]any{"id": id, "title": title, "status": status, "version": expected + 1, "tags": tags}})
	}
	if e == nil && status != str(old, "status") {
		e = s.enqueueEvent(r.Context(), tx, Event{Type: "document.status_changed", WorkspaceID: str(old, "workspace_id"), ResourceID: id, ResourceType: "document", Before: old, After: map[string]any{"id": id, "title": title, "status": status, "version": expected + 1, "tags": tags}})
	}
	if e == nil {
		e = s.enqueueTaskCompletions(r.Context(), tx, str(old, "workspace_id"), id, title, str(old, "markdown"), md)
	}
	if e == nil {
		e = s.recordAutomationEffect(r.Context(), tx, map[string]any{"id": id, "version": expected + 1})
	}
	if e == nil {
		e = s.finishProposalMergeTx(r, tx, id, md, expected+1)
	}
	if e == nil {
		e = s.finishDocumentSplitTx(r, tx, id, old, md, expected+1)
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		if WriteProtectionError(w, e) {
			return
		}
		respond(w, nil, e)
		return
	}
	auditChange := map[string]any{"version": expected + 1}
	if intent, ok := r.Context().Value(documentProposalMergeKey{}).(documentProposalMerge); ok {
		auditChange["proposal_id"] = intent.ID
	}
	s.audit(r, "DOCUMENT_UPDATE", id, auditChange)
	v, e := s.document(r, id)
	if e == nil {
		v["protection"] = protected.response()
	}
	respond(w, v, e)
}
func (s *Server) deleteDocument(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !s.canDocument(r.Context(), current(r), id, true) {
		apiError(w, 403, "문서 삭제 권한이 없습니다")
		return
	}
	expected, e := documentTrashExpectedVersion(r)
	if e != nil {
		apiError(w, 400, e.Error())
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	var version int
	var deleted bool
	e = tx.QueryRow(r.Context(), "SELECT version,deleted_at IS NOT NULL FROM documents WHERE id=$1 AND madi_document_allowed($2,id,true) FOR UPDATE", id, current(r).ID).Scan(&version, &deleted)
	if e != nil {
		respond(w, nil, e)
		return
	}
	if expected != nil && version != *expected {
		apiError(w, 409, "문서 버전이 변경되었습니다. 현재 문서를 확인하고 다시 삭제하세요")
		return
	}
	if deleted {
		jsonResponse(w, 200, map[string]any{"ok": true, "id": id, "version": version, "changed": false})
		return
	}
	var held bool
	e = tx.QueryRow(r.Context(), "SELECT EXISTS(SELECT 1 FROM knowledge_document_meta WHERE document_id=$1 AND (legal_hold OR retain_until>now()))", id).Scan(&held)
	if e != nil {
		respond(w, nil, e)
		return
	}
	if held {
		apiError(w, 409, "법적 보존 또는 보존 기한이 적용된 문서는 삭제할 수 없습니다")
		return
	}
	var raw []byte
	e = tx.QueryRow(r.Context(), "UPDATE documents SET deleted_at=now(),updated_at=now(),version=version+1 WHERE id=$1 AND deleted_at IS NULL RETURNING to_jsonb(documents)", id).Scan(&raw)
	var d map[string]any
	if e == nil {
		e = json.Unmarshal(raw, &d)
	}
	if e == nil {
		_, e = tx.Exec(r.Context(), "INSERT INTO document_versions(document_id,version,title,markdown,tags,user_id,block_metadata) SELECT id,version,title,markdown,tags,$2,block_metadata FROM documents WHERE id=$1", id, current(r).ID)
	}
	if e == nil {
		e = s.enqueueEvent(r.Context(), tx, Event{Type: "document.deleted", WorkspaceID: str(d, "workspace_id"), ResourceID: id, Before: d})
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "DOCUMENT_DELETE", id, map[string]any{"version": version + 1, "cas": expected != nil})
	respond(w, map[string]any{"ok": true, "id": id, "version": version + 1, "changed": true}, e)
}
func (s *Server) restoreDocument(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !s.canDocument(r.Context(), current(r), id, true) {
		apiError(w, 403, "문서 복원 권한이 없습니다")
		return
	}
	expected, e := documentTrashExpectedVersion(r)
	if e != nil {
		apiError(w, 400, e.Error())
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	var old map[string]any
	e = tx.QueryRow(r.Context(), `SELECT to_jsonb(d) FROM documents d WHERE id=$1 AND madi_document_allowed($2,id,true) FOR UPDATE`, id, current(r).ID).Scan(&old)
	if e != nil {
		respond(w, nil, e)
		return
	}
	if expected != nil && number(old, "version", 0) != *expected {
		apiError(w, 409, "휴지통의 문서 버전이 변경되었습니다. 현재 상태를 확인한 뒤 복원하세요")
		return
	}
	if old["deleted_at"] == nil {
		_ = tx.Rollback(r.Context())
		v, err := s.document(r, id)
		respond(w, v, err)
		return
	}
	_, e = tx.Exec(r.Context(), "UPDATE documents SET deleted_at=NULL WHERE id=$1", id)
	if e != nil {
		respond(w, nil, e)
		return
	}
	protected, e := s.protectCanonicalDocumentTx(r.Context(), tx, current(r), id, str(old, "workspace_id"), str(old, "title"), str(old, "markdown"), listStrings(old["tags"]), listStrings(old["aliases"]), old["block_metadata"])
	if e != nil {
		if !WriteProtectionError(w, e) {
			respond(w, nil, e)
		}
		return
	}
	status, e := approvalSaveStatusTx(r.Context(), tx, id, old, map[string]any{"title": protected.Title, "markdown": protected.Markdown, "tags": protected.Tags, "aliases": protected.Aliases, "block_metadata": protected.Metadata}, str(old, "status"))
	if e != nil {
		approvalRespondError(w, e)
		return
	}
	_, e = tx.Exec(r.Context(), "UPDATE documents SET title=$2,markdown=$3,tags=$4,aliases=$5,block_metadata=$6,status=$7,version=version+1,updated_at=now() WHERE id=$1", id, protected.Title, protected.Markdown, jsonValue(protected.Tags), jsonValue(protected.Aliases), jsonValue(protected.Metadata), status)
	if e == nil {
		_, e = tx.Exec(r.Context(), "INSERT INTO document_versions(document_id,version,title,markdown,tags,user_id,block_metadata) SELECT id,version,title,markdown,tags,$2,block_metadata FROM documents WHERE id=$1", id, current(r).ID)
	}
	if e == nil {
		e = s.enqueueEvent(r.Context(), tx, Event{Type: "document.updated", WorkspaceID: str(old, "workspace_id"), ResourceID: id, ResourceType: "document", Before: old, After: map[string]any{"id": id, "version": number(old, "version", 0) + 1, "restored": true}})
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "DOCUMENT_RESTORE", id, map[string]any{"version": number(old, "version", 0) + 1, "cas": expected != nil})
	v, e := s.document(r, id)
	if e == nil {
		v["protection"] = protected.response()
	}
	respond(w, v, e)
}
func (s *Server) favoriteDocument(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !s.canDocument(r.Context(), current(r), id, false) {
		apiError(w, 403, "문서 접근 권한이 없습니다")
		return
	}
	var removed string
	e := s.DB.QueryRow(r.Context(), "DELETE FROM favorites WHERE user_id=$1 AND document_id=$2 RETURNING document_id", current(r).ID, id).Scan(&removed)
	if e == pgx.ErrNoRows {
		_, e = s.DB.Exec(r.Context(), "INSERT INTO favorites VALUES($1,$2) ON CONFLICT DO NOTHING", current(r).ID, id)
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	v, e := s.document(r, id)
	respond(w, v, e)
}
func (s *Server) documentVersions(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !s.canDocument(r.Context(), current(r), id, false) {
		apiError(w, 403, "문서 접근 권한이 없습니다")
		return
	}
	limit, before := 50, 2147483647
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 100 {
			apiError(w, 400, "이력 조회 개수는 1~100개입니다")
			return
		}
		limit = n
	}
	if raw := r.URL.Query().Get("before"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 2147483647 {
			apiError(w, 400, "이력 페이지 버전을 확인하세요")
			return
		}
		before = n
	}
	v, e := s.rows(r.Context(), `SELECT jsonb_build_object('version',v.version,'title',v.title,'created_at',v.created_at,'user_name',u.name,'markdown_bytes',octet_length(v.markdown),'current_version',d.version) FROM document_versions v JOIN users u ON u.id=v.user_id JOIN documents d ON d.id=v.document_id WHERE v.document_id=$2 AND v.version<$3 AND madi_document_allowed($1,d.id,false) AND ($4='' OR d.workspace_id::text=$4) ORDER BY v.version DESC LIMIT $5`, current(r).ID, id, before, current(r).WorkspaceID, limit)
	respond(w, v, e)
}
func (s *Server) restoreVersion(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !s.canDocument(r.Context(), current(r), id, true) {
		apiError(w, 403, "문서 복원 권한이 없습니다")
		return
	}
	version, e := strconv.Atoi(r.PathValue("version"))
	if e != nil || version < 1 || version > 2147483647 {
		apiError(w, 400, "버전을 확인하세요")
		return
	}
	var in struct {
		ExpectedVersion int `json:"expected_version"`
	}
	if decode(r, &in) != nil || in.ExpectedVersion < 1 || in.ExpectedVersion > 2147483647 {
		apiError(w, 400, "복원할 때 화면에서 확인한 expected_version이 필요합니다")
		return
	}
	v, e := s.versionSnapshot(r, id, version)
	if e != nil {
		respond(w, nil, e)
		return
	}
	v["version"] = in.ExpectedVersion
	s.saveDocument(w, r, id, v)
}
func wikiTargets(md string) []string {
	return indexMarkdown(md).Links
}
func documentNames(d map[string]any) []string {
	return append([]string{str(d, "title"), str(d, "id")}, listStrings(d["aliases"])...)
}
func (s *Server) comments(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !s.canDocument(r.Context(), current(r), id, false) {
		apiError(w, 403, "문서 접근 권한이 없습니다")
		return
	}
	v, e := s.rows(r.Context(), "SELECT to_jsonb(c)||jsonb_build_object('user_name',u.name) FROM comments c JOIN users u ON c.user_id=u.id WHERE document_id=$1 ORDER BY c.created_at", id)
	respond(w, v, e)
}
func (s *Server) addComment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var canComment bool
	if validID(id) {
		_ = s.DB.QueryRow(r.Context(), "SELECT madi_document_comment_allowed($1,$2)", current(r).ID, id).Scan(&canComment)
	}
	if !canComment || !s.canDocument(r.Context(), current(r), id, false) || current(r).Role == "viewer" || !hasIntegrationScope(current(r), "document:write") {
		apiError(w, 403, "댓글 작성 권한이 없습니다")
		return
	}
	var role string
	s.DB.QueryRow(r.Context(), "SELECT m.role FROM workspace_members m JOIN documents d ON d.workspace_id=m.workspace_id WHERE d.id=$1 AND m.user_id=$2", id, current(r).ID).Scan(&role)
	if role == "viewer" {
		apiError(w, 403, "댓글 작성 권한이 없습니다")
		return
	}
	var in struct{ Body string }
	if decode(r, &in) != nil || strings.TrimSpace(in.Body) == "" || len(in.Body) > 10000 {
		apiError(w, 400, "댓글은 1~10000바이트여야 합니다")
		return
	}
	cid := newID()
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	var wid, title string
	e = tx.QueryRow(r.Context(), "SELECT workspace_id::text,title FROM documents WHERE id=$1 AND deleted_at IS NULL FOR SHARE", id).Scan(&wid, &title)
	if e == nil {
		_, e = tx.Exec(r.Context(), "INSERT INTO comments(id,document_id,user_id,body) VALUES($1,$2,$3,$4)", cid, id, current(r).ID, in.Body)
	}
	if e == nil {
		_, e = tx.Exec(r.Context(), "INSERT INTO notifications(id,user_id,title,document_id) SELECT $1,owner_id,$2,id FROM documents WHERE id=$3 AND owner_id!=$4", newID(), current(r).Name+"님이 댓글을 남겼습니다", id, current(r).ID)
	}
	if e == nil {
		e = s.enqueueEvent(r.Context(), tx, Event{Type: "comment.created", WorkspaceID: wid, ResourceID: id, After: map[string]any{"id": cid, "document_id": id, "title": title}})
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "COMMENT_CREATE", id, nil)
	v, e := s.one(r.Context(), "SELECT to_jsonb(c)||jsonb_build_object('user_name',$2::text) FROM comments c WHERE id=$1", cid, current(r).Name)
	respond(w, v, e)
}
func (s *Server) documentShares(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !s.canDocument(r.Context(), current(r), id, false) {
		apiError(w, 403, "문서 접근 권한이 없습니다")
		return
	}
	v, e := s.rows(r.Context(), "SELECT to_jsonb(sh)||jsonb_build_object('name',u.name,'email',u.email,'role',CASE WHEN sh.permission='write' THEN 'editor' ELSE 'viewer' END) FROM document_shares sh JOIN users u ON u.id=sh.user_id JOIN documents d ON d.id=sh.document_id WHERE sh.document_id=$1 AND d.owner_id=$2", id, current(r).ID)
	respond(w, v, e)
}
func (s *Server) addDocumentShare(w http.ResponseWriter, r *http.Request) {
	var in struct{ Email, Role string }
	if decode(r, &in) != nil || !oneOf(in.Role, "viewer", "editor") {
		apiError(w, 400, "공유 사용자와 역할을 확인하세요")
		return
	}
	var uid string
	if s.DB.QueryRow(r.Context(), "SELECT id FROM users WHERE lower(email)=$1 AND NOT disabled", strings.ToLower(strings.TrimSpace(in.Email))).Scan(&uid) != nil {
		apiError(w, 404, "등록된 사용자를 찾을 수 없습니다")
		return
	}
	permission := "read"
	if in.Role == "editor" {
		permission = "write"
	}
	r.Body = io.NopCloser(strings.NewReader(string(jsonValue(map[string]string{"user_id": uid, "permission": permission}))))
	s.setDocumentShare(w, r)
}
func (s *Server) setDocumentShare(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !s.canDocument(r.Context(), current(r), id, true) {
		apiError(w, 403, "공유 설정 권한이 없습니다")
		return
	}
	d, e := s.document(r, id)
	if e != nil {
		respond(w, nil, e)
		return
	}
	if str(d, "owner_id") != current(r).ID {
		apiError(w, 403, "문서 소유자만 공유 설정을 변경할 수 있습니다")
		return
	}
	var in struct {
		UserID     string `json:"user_id"`
		Permission string `json:"permission"`
	}
	if decode(r, &in) != nil || !validID(in.UserID) || !oneOf(in.Permission, "read", "write", "remove") {
		apiError(w, 400, "공유 사용자와 권한을 확인하세요")
		return
	}
	var member bool
	s.DB.QueryRow(r.Context(), "SELECT EXISTS(SELECT 1 FROM workspace_members WHERE workspace_id=$1 AND user_id=$2)", str(d, "workspace_id"), in.UserID).Scan(&member)
	if !member {
		apiError(w, 400, "같은 워크스페이스의 사용자를 선택하세요")
		return
	}
	if in.Permission == "remove" {
		_, e = s.DB.Exec(r.Context(), "DELETE FROM document_shares WHERE document_id=$1 AND user_id=$2", id, in.UserID)
	} else {
		_, e = s.DB.Exec(r.Context(), "INSERT INTO document_shares VALUES($1,$2,$3) ON CONFLICT(document_id,user_id) DO UPDATE SET permission=EXCLUDED.permission", id, in.UserID, in.Permission)
	}
	s.audit(r, "SHARE_CREATE", id, in)
	respond(w, map[string]bool{"ok": true}, e)
}
func (s *Server) approval(w http.ResponseWriter, r *http.Request) {
	s.approvalAdvanced(w, r)
}
func (s *Server) tasks(w http.ResponseWriter, r *http.Request) {
	wid := r.URL.Query().Get("workspace_id")
	if wid != "" && !s.canWorkspace(r.Context(), current(r), wid, false) {
		apiError(w, 403, "워크스페이스 접근 권한이 없습니다")
		return
	}
	if id := r.URL.Query().Get("document_id"); id != "" && !validID(id) {
		apiError(w, 400, "문서 ID를 확인하세요")
		return
	}
	v, e := s.taskRows(r)
	if e != nil {
		respond(w, nil, e)
		return
	}
	respond(w, v["items"], nil)
}
func (s *Server) updateTask(w http.ResponseWriter, r *http.Request) {
	var in struct {
		DocumentID string `json:"document_id"`
		Line       int    `json:"line"`
		Done       bool   `json:"done"`
		Version    int    `json:"version"`
	}
	if decode(r, &in) != nil || !s.canDocument(r.Context(), current(r), in.DocumentID, true) {
		apiError(w, 403, "할 일 수정 권한이 없습니다")
		return
	}
	d, e := s.document(r, in.DocumentID)
	if e != nil {
		respond(w, nil, e)
		return
	}
	lines := strings.Split(str(d, "markdown"), "\n")
	actualTask := false
	for _, task := range indexMarkdown(str(d, "markdown")).Tasks {
		if task.Line == in.Line {
			actualTask = true
			break
		}
	}
	if in.Line < 0 || in.Line >= len(lines) || !taskPattern.MatchString(lines[in.Line]) || !actualTask {
		apiError(w, 409, "할 일 위치가 변경되었습니다. 새로고침하세요")
		return
	}
	line := lines[in.Line]
	start := strings.Index(line, "[")
	mark := " "
	if in.Done {
		mark = "x"
	}
	lines[in.Line] = line[:start+1] + mark + line[start+2:]
	s.saveDocument(w, r, in.DocumentID, map[string]any{"markdown": strings.Join(lines, "\n"), "version": in.Version})
}
