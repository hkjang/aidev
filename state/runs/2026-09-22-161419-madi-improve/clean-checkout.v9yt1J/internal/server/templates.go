package server

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
)

//go:embed templates.sql
var templatesSchema string

func (s *Server) migrateTemplates(ctx context.Context) error {
	_, err := s.DB.Exec(ctx, templatesSchema)
	return err
}
func (s *Server) registerTemplates() {
	s.handle("GET /api/v1/templates/builtins", s.templateBuiltins)
	s.handle("GET /api/v1/templates", s.listTemplates)
	s.handle("POST /api/v1/templates", s.createTemplate)
	s.handle("GET /api/v1/templates/{id}", s.getTemplate)
	s.handle("PUT /api/v1/templates/{id}", s.updateTemplate)
	s.handle("DELETE /api/v1/templates/{id}", s.deleteTemplate)
	s.handle("POST /api/v1/templates/{id}/restore", s.restoreTemplate)
	s.handle("GET /api/v1/templates/{id}/versions", s.templateVersions)
	s.handle("POST /api/v1/templates/{id}/versions/{version}/restore", s.restoreTemplateVersion)
	s.handle("POST /api/v1/templates/{id}/duplicate", s.duplicateTemplate)
	s.handle("POST /api/v1/templates/{id}/documents", s.documentFromTemplate)
}

type templateInput struct {
	WorkspaceID string   `json:"workspace_id"`
	SpaceID     string   `json:"space_id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Icon        string   `json:"icon"`
	Kind        string   `json:"kind"`
	Markdown    string   `json:"markdown"`
	Tags        []string `json:"tags"`
	Visibility  string   `json:"visibility"`
	Version     int      `json:"version"`
}

func normalizeTemplate(in *templateInput) error {
	in.Name = strings.TrimSpace(in.Name)
	in.Category = strings.TrimSpace(in.Category)
	if in.Icon == "" {
		in.Icon = "file"
	}
	if in.Kind == "" {
		in.Kind = "page"
	}
	if in.Visibility == "" {
		in.Visibility = "private"
	}
	if !validID(in.WorkspaceID) || (in.SpaceID != "" && !validID(in.SpaceID)) || !oneOf(in.Visibility, "private", "workspace", "space") || (in.Visibility == "space" && in.SpaceID == "") {
		return errors.New("워크스페이스와 공개 범위·공간을 확인하세요")
	}
	if !oneOf(in.Kind, "page", "note", "daily", "meeting", "decision", "runbook") {
		return errors.New("템플릿의 문서 종류를 확인하세요")
	}
	if len(in.Name) < 1 || len(in.Name) > 500 || len(in.Description) > 4000 || len(in.Category) > 120 || len(in.Icon) > 64 || len(in.Markdown) > 4<<20 {
		return errors.New("이름 500바이트·설명 4KB·분류 120바이트·Markdown 4MB 제한을 확인하세요")
	}
	for _, v := range []string{in.Name, in.Description, in.Category, in.Icon, in.Markdown} {
		if !utf8.ValidString(v) || strings.ContainsRune(v, 0) {
			return errors.New("올바른 UTF-8 텍스트를 입력하세요")
		}
	}
	fm, err := parseFrontMatter(in.Markdown)
	if err != nil {
		return err
	}
	if tags, ok := fm["tags"]; ok {
		in.Tags = listStrings(tags)
	}
	if len(in.Tags) > 100 {
		return errors.New("태그는 100개까지 지정할 수 있습니다")
	}
	tags := []string{}
	seen := map[string]bool{}
	for _, tag := range in.Tags {
		tag = strings.TrimSpace(tag)
		if len(tag) > 200 || !utf8.ValidString(tag) || strings.ContainsRune(tag, 0) {
			return errors.New("태그 길이와 UTF-8 형식을 확인하세요")
		}
		if tag != "" && !seen[tag] {
			tags = append(tags, tag)
			seen[tag] = true
		}
	}
	in.Tags = tags
	return nil
}
func (s *Server) canTemplate(r *http.Request, id string, write bool) bool {
	p := current(r)
	if p == nil || !validID(id) || !hasIntegrationScope(p, map[bool]string{false: "document:read", true: "document:write"}[write]) {
		return false
	}
	var allowed bool
	err := s.DB.QueryRow(r.Context(), "SELECT EXISTS(SELECT 1 FROM document_templates t WHERE t.id=$1 AND ($3='' OR t.workspace_id::text=$3) AND madi_template_allowed($2,t.id,$4))", id, p.ID, p.WorkspaceID, write).Scan(&allowed)
	return err == nil && allowed
}
func (s *Server) templateRecord(r *http.Request, id string) (map[string]any, error) {
	p := current(r)
	v, err := s.one(r.Context(), "SELECT to_jsonb(t)||jsonb_build_object('owner_name',u.name,'can_manage',madi_template_allowed($2,t.id,true) AND $4,'can_write',madi_template_allowed($2,t.id,true) AND $4 AND t.deleted_at IS NULL) FROM document_templates t JOIN users u ON u.id=t.owner_id WHERE t.id=$1 AND ($3='' OR t.workspace_id::text=$3) AND madi_template_allowed($2,t.id,false)", id, p.ID, p.WorkspaceID, hasIntegrationScope(p, "document:write"))
	if err != nil {
		return nil, err
	}
	v["builtin"] = false
	fm, err := parseFrontMatter(str(v, "markdown"))
	if err != nil {
		return nil, err
	}
	v["frontmatter"] = fm
	return v, nil
}
func (s *Server) listTemplates(w http.ResponseWriter, r *http.Request) {
	p := current(r)
	q := r.URL.Query()
	wid := q.Get("workspace_id")
	if !s.canWorkspace(r.Context(), p, wid, false) || !hasIntegrationScope(p, "document:read") {
		apiError(w, 403, "워크스페이스 접근 권한이 없습니다")
		return
	}
	term := q.Get("q")
	category := q.Get("category")
	scope := q.Get("scope")
	if len(term) > 500 || len(category) > 120 || (scope != "" && !oneOf(scope, "private", "workspace", "space")) {
		apiError(w, 400, "검색 조건을 확인하세요")
		return
	}
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 50
	}
	offset, _ := strconv.Atoi(q.Get("offset"))
	if offset < 0 || offset > 100000 {
		offset = 0
	}
	// List metadata only; Markdown is fetched for the selected template after a
	// fresh ACL check, not bulk cached in the browser for the entire workspace.
	v, err := s.rows(r.Context(), `SELECT (to_jsonb(t)-'markdown')||jsonb_build_object('owner_name',u.name,'builtin',false,'can_manage',madi_template_allowed($1,t.id,true) AND $9,'can_write',madi_template_allowed($1,t.id,true) AND $9 AND t.deleted_at IS NULL) FROM document_templates t JOIN users u ON u.id=t.owner_id WHERE t.workspace_id=$2 AND madi_template_allowed($1,t.id,false) AND (t.deleted_at IS NOT NULL)=$3 AND ($4='' OR t.name ILIKE '%'||$4||'%' OR t.description ILIKE '%'||$4||'%' OR t.tags::text ILIKE '%'||$4||'%') AND ($5='' OR t.category=$5) AND ($6='' OR t.visibility=$6) ORDER BY t.updated_at DESC,t.id LIMIT $7 OFFSET $8`, p.ID, wid, q.Get("trash") == "1", term, category, scope, limit+1, offset, hasIntegrationScope(p, "document:write"))
	if err != nil {
		respond(w, nil, err)
		return
	}
	more := len(v) > limit
	if more {
		v = v[:limit]
	}
	jsonResponse(w, 200, map[string]any{"items": v, "has_more": more, "limit": limit, "offset": offset})
}
func (s *Server) getTemplate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !s.canTemplate(r, id, false) {
		apiError(w, 404, "템플릿을 찾을 수 없습니다")
		return
	}
	v, err := s.templateRecord(r, id)
	respond(w, v, err)
}
func (s *Server) protectTemplateTx(r *http.Request, tx pgx.Tx, in *templateInput) (map[string]any, error) {
	protected, err := s.ProtectDocumentTx(r.Context(), tx, current(r), "", in.WorkspaceID, in.Name, in.Markdown)
	if err != nil {
		return nil, err
	}
	meta, err := s.ProtectDocumentMetadataTx(r.Context(), tx, current(r), "", in.WorkspaceID, map[string]any{"description": in.Description, "category": in.Category, "icon": in.Icon, "tags": in.Tags})
	if err != nil {
		return nil, err
	}
	fields := meta.Value.(map[string]any)
	in.Name, in.Markdown = protected.Title, protected.Markdown
	in.Description, in.Category, in.Icon = str(fields, "description"), str(fields, "category"), str(fields, "icon")
	in.Tags = listStrings(fields["tags"])
	if err = normalizeTemplate(in); err != nil {
		return nil, err
	}
	return map[string]any{"changed": protected.Changed || meta.Changed, "mode": protected.Mode, "findings": append(protected.Findings, meta.Findings...)}, nil
}
func templateSnapshotTx(ctx context.Context, tx pgx.Tx, id, actor string) error {
	_, err := tx.Exec(ctx, `INSERT INTO document_template_versions(template_id,version,user_id,data) SELECT id,version,$2,to_jsonb(t)-'created_at'-'updated_at' FROM document_templates t WHERE id=$1`, id, actor)
	return err
}
func (s *Server) createTemplate(w http.ResponseWriter, r *http.Request) {
	var in templateInput
	if decode(r, &in) != nil {
		apiError(w, 400, "템플릿 입력을 확인하세요")
		return
	}
	s.insertTemplate(w, r, in)
}
func (s *Server) insertTemplate(w http.ResponseWriter, r *http.Request, in templateInput) {
	if err := normalizeTemplate(&in); err != nil {
		apiError(w, 400, err.Error())
		return
	}
	if !hasIntegrationScope(current(r), "document:write") || !s.canSpace(r.Context(), current(r), in.WorkspaceID, in.SpaceID, true) {
		apiError(w, 403, "템플릿을 저장할 작성 권한이 없습니다")
		return
	}
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		respond(w, nil, err)
		return
	}
	defer tx.Rollback(r.Context())
	if err = validateTemplateSourceTx(r.Context(), tx); err != nil {
		apiError(w, 409, err.Error())
		return
	}
	protected, err := s.protectTemplateTx(r, tx, &in)
	if err != nil {
		if !WriteProtectionError(w, err) {
			respond(w, nil, err)
		}
		return
	}
	id := newID()
	_, err = tx.Exec(r.Context(), `INSERT INTO document_templates(id,workspace_id,space_id,owner_id,name,description,category,icon,kind,markdown,tags,visibility) VALUES($1,$2,NULLIF($3,'')::uuid,$4,$5,$6,$7,$8,$9,$10,$11,$12)`, id, in.WorkspaceID, in.SpaceID, current(r).ID, in.Name, in.Description, in.Category, in.Icon, in.Kind, in.Markdown, jsonValue(in.Tags), in.Visibility)
	if err == nil {
		err = templateSnapshotTx(r.Context(), tx, id, current(r).ID)
	}
	if err == nil {
		err = tx.Commit(r.Context())
	}
	if err != nil {
		respond(w, nil, err)
		return
	}
	s.audit(r, "TEMPLATE_CREATE", id, map[string]any{"workspace_id": in.WorkspaceID, "version": 1})
	v, err := s.templateRecord(r, id)
	if err == nil {
		v["protection"] = protected
	}
	respond(w, v, err)
}
func (s *Server) updateTemplate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var in templateInput
	if !s.canTemplate(r, id, true) {
		apiError(w, 403, "템플릿 소유자의 작성 권한이 필요합니다")
		return
	}
	if decode(r, &in) != nil {
		apiError(w, 400, "템플릿 입력을 확인하세요")
		return
	}
	s.saveTemplate(w, r, id, in, false)
}
func (s *Server) saveTemplate(w http.ResponseWriter, r *http.Request, id string, in templateInput, restoring bool) {
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		respond(w, nil, err)
		return
	}
	defer tx.Rollback(r.Context())
	var wid string
	var version int
	var deleted bool
	err = tx.QueryRow(r.Context(), `SELECT workspace_id::text,version,deleted_at IS NOT NULL FROM document_templates WHERE id=$1 AND madi_template_allowed($2,id,true) FOR UPDATE`, id, current(r).ID).Scan(&wid, &version, &deleted)
	if err != nil {
		apiError(w, 403, "템플릿 작성 권한을 다시 확인하세요")
		return
	}
	if in.Version != version || (deleted && !restoring) {
		apiError(w, 409, "템플릿이 변경되거나 삭제되었습니다. 현재 버전을 다시 불러오세요")
		return
	}
	if in.WorkspaceID != "" && in.WorkspaceID != wid {
		apiError(w, 400, "템플릿은 다른 워크스페이스로 이동할 수 없습니다")
		return
	}
	in.WorkspaceID = wid
	if err = normalizeTemplate(&in); err != nil {
		apiError(w, 400, err.Error())
		return
	}
	if !s.canSpace(r.Context(), current(r), wid, in.SpaceID, true) {
		apiError(w, 403, "대상 공간의 작성 권한이 없습니다")
		return
	}
	protected, err := s.protectTemplateTx(r, tx, &in)
	if err != nil {
		if !WriteProtectionError(w, err) {
			respond(w, nil, err)
		}
		return
	}
	_, err = tx.Exec(r.Context(), `UPDATE document_templates SET space_id=NULLIF($2,'')::uuid,name=$3,description=$4,category=$5,icon=$6,kind=$7,markdown=$8,tags=$9,visibility=$10,version=version+1,updated_at=now(),deleted_at=NULL WHERE id=$1`, id, in.SpaceID, in.Name, in.Description, in.Category, in.Icon, in.Kind, in.Markdown, jsonValue(in.Tags), in.Visibility)
	if err == nil {
		err = templateSnapshotTx(r.Context(), tx, id, current(r).ID)
	}
	if err == nil {
		err = tx.Commit(r.Context())
	}
	if err != nil {
		respond(w, nil, err)
		return
	}
	s.audit(r, "TEMPLATE_UPDATE", id, map[string]any{"version": version + 1, "restored": restoring})
	v, err := s.templateRecord(r, id)
	if err == nil {
		v["protection"] = protected
	}
	respond(w, v, err)
}
func (s *Server) deleteTemplate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	version, err := strconv.Atoi(r.URL.Query().Get("version"))
	if !s.canTemplate(r, id, true) {
		apiError(w, 403, "템플릿 소유자의 작성 권한이 필요합니다")
		return
	}
	if err != nil || version < 1 {
		apiError(w, 400, "현재 버전을 지정하세요")
		return
	}
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		respond(w, nil, err)
		return
	}
	defer tx.Rollback(r.Context())
	tag, err := tx.Exec(r.Context(), `UPDATE document_templates SET deleted_at=now(),updated_at=now(),version=version+1 WHERE id=$1 AND version=$2 AND deleted_at IS NULL AND madi_template_allowed($3,id,true)`, id, version, current(r).ID)
	if err == nil && tag.RowsAffected() != 1 {
		apiError(w, 409, "템플릿이 변경되거나 삭제되었습니다")
		return
	}
	if err == nil {
		err = templateSnapshotTx(r.Context(), tx, id, current(r).ID)
	}
	if err == nil {
		err = tx.Commit(r.Context())
	}
	if err != nil {
		respond(w, nil, err)
		return
	}
	s.audit(r, "TEMPLATE_DELETE", id, map[string]any{"version": version + 1})
	jsonResponse(w, 200, map[string]any{"ok": true, "recoverable": true})
}
func templateFromMap(v map[string]any) templateInput {
	return templateInput{WorkspaceID: str(v, "workspace_id"), SpaceID: str(v, "space_id"), Name: str(v, "name"), Description: str(v, "description"), Category: str(v, "category"), Icon: str(v, "icon"), Kind: str(v, "kind"), Markdown: str(v, "markdown"), Tags: listStrings(v["tags"]), Visibility: str(v, "visibility"), Version: int(number(v, "version", 0))}
}
func (s *Server) restoreTemplate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var request struct {
		Version int `json:"version"`
	}
	if !s.canTemplate(r, id, true) {
		apiError(w, 403, "템플릿 소유자의 작성 권한이 필요합니다")
		return
	}
	if decode(r, &request) != nil {
		apiError(w, 400, "현재 버전을 지정하세요")
		return
	}
	v, err := s.templateRecord(r, id)
	if err != nil {
		respond(w, nil, err)
		return
	}
	if v["deleted_at"] == nil {
		apiError(w, 409, "휴지통의 템플릿만 복원할 수 있습니다")
		return
	}
	in := templateFromMap(v)
	in.Version = request.Version
	s.saveTemplate(w, r, id, in, true)
}
func (s *Server) templateVersions(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	// Old snapshots may contain formerly private content. Even shared readers
	// never receive history; only the current owner with write ACL can inspect it.
	if !s.canTemplate(r, id, true) {
		apiError(w, 403, "템플릿 소유자의 작성 권한이 필요합니다")
		return
	}
	v, err := s.rows(r.Context(), `SELECT jsonb_build_object('version',v.version,'created_at',v.created_at,'user_name',u.name,'data',v.data-'markdown') FROM document_template_versions v JOIN users u ON u.id=v.user_id JOIN document_templates t ON t.id=v.template_id WHERE v.template_id=$1 AND madi_template_allowed($2,t.id,true) AND ($3='' OR t.workspace_id::text=$3) ORDER BY v.version DESC LIMIT 100`, id, current(r).ID, current(r).WorkspaceID)
	respond(w, v, err)
}
func (s *Server) restoreTemplateVersion(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	revision, err := strconv.Atoi(r.PathValue("version"))
	var request struct {
		Version int `json:"version"`
	}
	if !s.canTemplate(r, id, true) {
		apiError(w, 403, "템플릿 소유자의 작성 권한이 필요합니다")
		return
	}
	if err != nil || revision < 1 || decode(r, &request) != nil {
		apiError(w, 400, "복원할 버전과 현재 버전을 확인하세요")
		return
	}
	v, err := s.one(r.Context(), "SELECT v.data FROM document_template_versions v JOIN document_templates t ON t.id=v.template_id WHERE v.template_id=$1 AND v.version=$2 AND madi_template_allowed($3,t.id,true) AND ($4='' OR t.workspace_id::text=$4)", id, revision, current(r).ID, current(r).WorkspaceID)
	if err != nil {
		apiError(w, 404, "템플릿 버전이 없습니다")
		return
	}
	in := templateFromMap(v)
	in.Version = request.Version
	s.saveTemplate(w, r, id, in, true)
}

type templateCreationContext struct {
	Kind, SourceID, WorkspaceID string
	Version                     int
}

// Recheck the source inside the destination transaction. A shared row lock
// prevents a template edit/private toggle/delete racing its derived write.
func validateTemplateSourceTx(ctx context.Context, tx pgx.Tx) error {
	source, ok := ctx.Value(templateCreationContext{}).(templateCreationContext)
	if !ok || strings.HasPrefix(source.SourceID, "builtin-") {
		return nil
	}
	p, _ := ctx.Value(principalKey).(*Principal)
	if p == nil || !hasIntegrationScope(p, "document:read") {
		return errors.New("템플릿 열람 권한이 변경되었습니다")
	}
	var version int
	err := tx.QueryRow(ctx, `SELECT version FROM document_templates WHERE id=$1 AND workspace_id=$2 AND deleted_at IS NULL AND madi_template_allowed($3,id,false) FOR SHARE`, source.SourceID, source.WorkspaceID, p.ID).Scan(&version)
	if err != nil || version != source.Version {
		return errors.New("템플릿 버전 또는 접근 권한이 변경되었습니다. 다시 불러오세요")
	}
	return nil
}

func (s *Server) recordTemplateDocumentTx(ctx context.Context, tx pgx.Tx, id string) error {
	source, ok := ctx.Value(templateCreationContext{}).(templateCreationContext)
	if !ok {
		return nil
	}
	if err := validateTemplateSourceTx(ctx, tx); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `INSERT INTO knowledge_document_meta(document_id,kind,system_metadata) VALUES($1,$2,$3) ON CONFLICT(document_id) DO UPDATE SET kind=EXCLUDED.kind,system_metadata=knowledge_document_meta.system_metadata||EXCLUDED.system_metadata`, id, source.Kind, jsonValue(map[string]any{"template": map[string]any{"id": source.SourceID, "version": source.Version}}))
	return err
}
func (s *Server) templateSource(r *http.Request, id, wid string) (map[string]any, error) {
	for _, builtin := range builtinTemplates() {
		if str(builtin, "id") == id {
			if !s.canWorkspace(r.Context(), current(r), wid, false) || !hasIntegrationScope(current(r), "document:read") {
				return nil, errors.New("템플릿에 접근할 수 없습니다")
			}
			return builtin, nil
		}
	}
	if !s.canTemplate(r, id, false) {
		return nil, errors.New("템플릿에 접근할 수 없습니다")
	}
	v, err := s.templateRecord(r, id)
	if err != nil {
		return nil, err
	}
	if str(v, "workspace_id") != wid || v["deleted_at"] != nil {
		return nil, errors.New("현재 워크스페이스의 활성 템플릿을 선택하세요")
	}
	return v, nil
}
func (s *Server) duplicateTemplate(w http.ResponseWriter, r *http.Request) {
	var request struct {
		WorkspaceID     string `json:"workspace_id"`
		ExpectedVersion int    `json:"expected_version"`
		Name            string `json:"name"`
	}
	if decode(r, &request) != nil {
		apiError(w, 400, "복제할 템플릿을 확인하세요")
		return
	}
	v, err := s.templateSource(r, r.PathValue("id"), request.WorkspaceID)
	if err != nil {
		apiError(w, 404, err.Error())
		return
	}
	if request.ExpectedVersion != int(number(v, "version", 0)) {
		apiError(w, 409, "템플릿이 변경되었습니다. 미리보기를 다시 불러오세요")
		return
	}
	in := templateFromMap(v)
	in.WorkspaceID = request.WorkspaceID
	in.Visibility = "private"
	in.SpaceID = ""
	in.Name = request.Name
	if in.Name == "" {
		in.Name = str(v, "name") + " 사본"
	}
	in.Version = 0
	derived := r.Clone(context.WithValue(r.Context(), templateCreationContext{}, templateCreationContext{SourceID: r.PathValue("id"), WorkspaceID: request.WorkspaceID, Version: request.ExpectedVersion}))
	s.insertTemplate(w, derived, in)
}
func expandTemplateDates(source string, now time.Time, zone string) string {
	loc, err := time.LoadLocation(zone)
	if err != nil {
		loc = time.UTC
	}
	now = now.In(loc)
	return strings.NewReplacer("{{date}}", now.Format("2006-01-02"), "{{datetime}}", now.Format(time.RFC3339)).Replace(source)
}
func (s *Server) documentFromTemplate(w http.ResponseWriter, r *http.Request) {
	var request struct {
		WorkspaceID     string `json:"workspace_id"`
		ExpectedVersion int    `json:"expected_version"`
		Title           string `json:"title"`
		SpaceID         string `json:"space_id"`
		ParentID        string `json:"parent_id"`
		Visibility      string `json:"visibility"`
	}
	if decode(r, &request) != nil {
		apiError(w, 400, "새 문서 입력을 확인하세요")
		return
	}
	if !hasIntegrationScope(current(r), "document:write") {
		apiError(w, 403, "문서 작성 권한이 필요합니다")
		return
	}
	v, err := s.templateSource(r, r.PathValue("id"), request.WorkspaceID)
	if err != nil {
		apiError(w, 404, err.Error())
		return
	}
	if request.ExpectedVersion != int(number(v, "version", 0)) {
		apiError(w, 409, "템플릿이 변경되었습니다. 미리보기를 다시 불러오세요")
		return
	}
	zone := "UTC"
	var preferences []byte
	if s.DB.QueryRow(r.Context(), "SELECT preferences FROM users WHERE id=$1", current(r).ID).Scan(&preferences) == nil {
		var p map[string]any
		if json.Unmarshal(preferences, &p) == nil && str(p, "timezone") != "" {
			zone = str(p, "timezone")
		}
	}
	now := time.Now()
	title := request.Title
	if title == "" {
		title = str(v, "name")
	}
	title = expandTemplateDates(title, now, zone)
	visibility := request.Visibility
	if visibility == "" {
		visibility = "private"
	}
	data := map[string]any{"workspace_id": request.WorkspaceID, "title": title, "markdown": expandTemplateDates(str(v, "markdown"), now, zone), "tags": v["tags"], "icon": v["icon"], "space_id": request.SpaceID, "parent_id": request.ParentID, "visibility": visibility}
	// Use the same canonical mutation handler as the editor. It applies current
	// destination ACL, PII policy, draft lifecycle, versions and event hooks.
	derived := r.Clone(context.WithValue(r.Context(), templateCreationContext{}, templateCreationContext{Kind: str(v, "kind"), SourceID: r.PathValue("id"), WorkspaceID: request.WorkspaceID, Version: request.ExpectedVersion}))
	body := jsonValue(data)
	derived.Body = io.NopCloser(bytes.NewReader(body))
	derived.ContentLength = int64(len(body))
	s.createDocument(w, derived)
}
