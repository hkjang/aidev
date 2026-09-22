package server

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"
)

//go:embed spaces.sql
var spacesSchema string

func (s *Server) migrateSpaces(ctx context.Context) error {
	_, e := s.DB.Exec(ctx, spacesSchema)
	return e
}
func (s *Server) registerSpaces() {
	s.handle("GET /api/v1/spaces", s.listSpaces)
	s.handle("GET /api/v1/spaces/{id}/documents", s.spaceDocuments)
	s.handle("POST /api/v1/spaces", s.createSpace)
	s.handle("PUT /api/v1/spaces/{id}", s.updateSpace)
	s.handle("DELETE /api/v1/spaces/{id}", s.deleteSpace)
	s.handle("GET /api/v1/spaces/{id}/members", s.spaceMembers)
	s.handle("PUT /api/v1/spaces/{id}/members", s.setSpaceMember)
	s.handle("GET /api/v1/workspaces/{id}/settings", s.getWorkspaceSettings)
	s.handle("GET /api/v1/workspaces/{id}/presentation", s.workspacePresentation)
	s.handle("PUT /api/v1/workspaces/{id}/settings", s.saveWorkspaceSettings)
	s.handle("GET /api/v1/workspaces/{id}/settings/history", s.workspaceSettingsHistory)
	s.handle("POST /api/v1/workspaces/{id}/settings/history/{historyId}/restore", s.restoreWorkspaceSettings)
	s.handle("GET /api/v1/organizations", s.listOrganizations)
	s.handle("POST /api/v1/organizations", s.createOrganization)
	s.handle("GET /api/v1/organizations/{id}/members", s.organizationMembers)
	s.handle("PUT /api/v1/organizations/{id}/members", s.setOrganizationMember)
	s.handle("PUT /api/v1/workspaces/{id}/organization", s.setWorkspaceOrganization)
}
func (s *Server) canSpace(ctx context.Context, p *Principal, wid, id string, write bool) bool {
	if !s.canWorkspace(ctx, p, wid, write) {
		return false
	}
	if id == "" {
		return true
	}
	if !validID(id) {
		return false
	}
	var allowed bool
	e := s.DB.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM spaces WHERE id=$1 AND workspace_id=$2 AND madi_space_allowed($3,id,$4))", id, wid, p.ID, write).Scan(&allowed)
	return e == nil && allowed
}
func (s *Server) listSpaces(w http.ResponseWriter, r *http.Request) {
	wid := r.URL.Query().Get("workspace_id")
	if !s.canWorkspace(r.Context(), current(r), wid, false) {
		apiError(w, 403, "워크스페이스 접근 권한이 없습니다")
		return
	}
	manage := r.URL.Query().Get("manage") == "1" && s.workspaceAdmin(r, wid)
	v, e := s.rows(r.Context(), `SELECT to_jsonb(s)||jsonb_build_object('can_read',madi_space_allowed($1,s.id,false),'can_write',madi_space_allowed($1,s.id,true),'can_manage',$3::boolean) FROM spaces s WHERE workspace_id=$2 AND ($3 OR madi_space_allowed($1,s.id,false)) ORDER BY name`, current(r).ID, wid, manage)
	respond(w, v, e)
}
func (s *Server) spaceDocuments(w http.ResponseWriter, r *http.Request) {
	if !hasIntegrationScope(current(r), "document:read") {
		apiError(w, 403, "문서 조회 키 권한이 필요합니다")
		return
	}
	id := r.PathValue("id")
	if !validID(id) {
		apiError(w, 404, "공간을 찾을 수 없습니다")
		return
	}
	var wid string
	if s.DB.QueryRow(r.Context(), "SELECT workspace_id::text FROM spaces WHERE id=$1", id).Scan(&wid) != nil || !s.canSpace(r.Context(), current(r), wid, id, false) {
		apiError(w, 403, "공간 접근 권한이 없습니다")
		return
	}
	v, e := s.rows(r.Context(), "WITH RECURSIVE descendants AS (SELECT id FROM spaces WHERE id=$2 UNION SELECT s.id FROM spaces s JOIN descendants d ON s.parent_id=d.id) SELECT "+docSummaryJSON+" FROM documents d WHERE "+docACL+" AND d.space_id IN (SELECT id FROM descendants) AND d.deleted_at IS NULL ORDER BY d.updated_at DESC LIMIT 2000", current(r).ID, id)
	if !hasIntegrationScope(current(r), "document:write") {
		for _, d := range v {
			d["can_write"] = false
		}
	}
	respond(w, v, e)
}

var simpleSlug = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,79}$`)

func (s *Server) validSpaceParent(r *http.Request, id, wid, parent string) bool {
	if parent == "" {
		return true
	}
	if parent == id || !s.canSpace(r.Context(), current(r), wid, parent, true) {
		return false
	}
	var allowed bool
	e := s.DB.QueryRow(r.Context(), `WITH RECURSIVE chain AS (SELECT id,parent_id,ARRAY[id] path FROM spaces WHERE id=$1 UNION ALL SELECT s.id,s.parent_id,c.path||s.id FROM spaces s JOIN chain c ON s.id=c.parent_id WHERE NOT s.id=ANY(c.path) AND cardinality(c.path)<21) SELECT count(*)<20 AND NOT bool_or(id::text=$2) FROM chain`, parent, id).Scan(&allowed)
	return e == nil && allowed
}
func (s *Server) createSpace(w http.ResponseWriter, r *http.Request) {
	var in map[string]any
	if decode(r, &in) != nil {
		apiError(w, 400, "공간 입력값을 확인하세요")
		return
	}
	wid := str(in, "workspace_id")
	if !s.workspaceAdmin(r, wid) {
		apiError(w, 403, "워크스페이스 관리자만 공간을 만들 수 있습니다")
		return
	}
	id := newID()
	name := strings.TrimSpace(str(in, "name"))
	slug := str(in, "slug")
	if slug == "" {
		slug = "space-" + id[:8]
	}
	visibility := str(in, "visibility")
	if visibility == "" {
		visibility = "workspace"
	}
	classification := str(in, "classification")
	if classification == "" {
		classification = "internal"
	}
	parent := str(in, "parent_id")
	if name == "" || len(name) > 200 || !simpleSlug.MatchString(slug) || !oneOf(visibility, "workspace", "restricted") || !oneOf(classification, "public", "internal", "confidential", "restricted") || !s.validSpaceParent(r, id, wid, parent) {
		apiError(w, 400, "공간 이름·주소·상위 공간·공개 범위를 확인하세요")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	if e = s.treePlacementTx(r, tx, "spaces", wid, id, parent); e != nil {
		apiError(w, 409, e.Error())
		return
	}
	_, e = tx.Exec(r.Context(), `INSERT INTO spaces(id,workspace_id,parent_id,name,slug,visibility,classification,owner_id) VALUES($1,$2,NULLIF($3,'')::uuid,$4,$5,$6,$7,$8)`, id, wid, parent, name, slug, visibility, classification, current(r).ID)
	if e == nil {
		_, e = tx.Exec(r.Context(), "INSERT INTO space_members VALUES($1,$2,'admin')", id, current(r).ID)
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		apiError(w, 409, "공간을 만들 수 없습니다. 중복 주소를 확인하세요")
		return
	}
	s.audit(r, "SPACE_CREATE", id, map[string]any{"workspace_id": wid})
	v, e := s.one(r.Context(), "SELECT to_jsonb(s) FROM spaces s WHERE id=$1", id)
	respond(w, v, e)
}
func (s *Server) managedSpace(w http.ResponseWriter, r *http.Request) (map[string]any, bool) {
	id := r.PathValue("id")
	if !validID(id) {
		apiError(w, 404, "공간을 찾을 수 없습니다")
		return nil, false
	}
	v, e := s.one(r.Context(), "SELECT to_jsonb(s) FROM spaces s WHERE id=$1", id)
	if e != nil || !s.workspaceAdmin(r, str(v, "workspace_id")) {
		apiError(w, 403, "공간 관리 권한이 없습니다")
		return nil, false
	}
	return v, true
}
func (s *Server) updateSpace(w http.ResponseWriter, r *http.Request) {
	old, ok := s.managedSpace(w, r)
	if !ok {
		return
	}
	var in map[string]any
	if decode(r, &in) != nil {
		apiError(w, 400, "공간 입력값을 확인하세요")
		return
	}
	name, slug, parent, visibility, classification := str(old, "name"), str(old, "slug"), str(old, "parent_id"), str(old, "visibility"), str(old, "classification")
	if v, ok := in["name"].(string); ok {
		name = strings.TrimSpace(v)
	}
	if v, ok := in["slug"].(string); ok {
		slug = v
	}
	if v, ok := in["parent_id"]; ok {
		parent, _ = v.(string)
	}
	if v, ok := in["visibility"].(string); ok {
		visibility = v
	}
	if v, ok := in["classification"].(string); ok {
		classification = v
	}
	if name == "" || len(name) > 200 || !simpleSlug.MatchString(slug) || !oneOf(visibility, "workspace", "restricted") || !oneOf(classification, "public", "internal", "confidential", "restricted") || !s.validSpaceParent(r, str(old, "id"), str(old, "workspace_id"), parent) {
		apiError(w, 400, "공간 설정 또는 순환하는 상위 공간을 확인하세요")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	if e = s.treePlacementTx(r, tx, "spaces", str(old, "workspace_id"), str(old, "id"), parent); e != nil {
		apiError(w, 409, e.Error())
		return
	}
	_, e = tx.Exec(r.Context(), `UPDATE spaces SET name=$2,slug=$3,parent_id=NULLIF($4,'')::uuid,visibility=$5,classification=$6,updated_at=now() WHERE id=$1`, str(old, "id"), name, slug, parent, visibility, classification)
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		apiError(w, 409, "공간 주소가 중복되거나 설정을 저장하지 못했습니다")
		return
	}
	s.audit(r, "SPACE_UPDATE", str(old, "id"), in)
	v, e := s.one(r.Context(), "SELECT to_jsonb(s) FROM spaces s WHERE id=$1", str(old, "id"))
	respond(w, v, e)
}

// Serialize structural mutations only; ordinary CRDT/document edits remain
// concurrent. Validate both ancestors and the moved subtree under the same lock.
func (s *Server) treePlacementTx(r *http.Request, tx pgx.Tx, table, wid, id, parent string) error {
	if !oneOf(table, "spaces", "documents") {
		return errors.New("허용되지 않은 트리 유형입니다")
	}
	if _, e := tx.Exec(r.Context(), "SELECT pg_advisory_xact_lock(hashtextextended($1,22))", table+"/"+wid); e != nil {
		return e
	}
	if parent != "" {
		var allowed bool
		fn := "madi_document_allowed"
		if table == "spaces" {
			fn = "madi_space_allowed"
		}
		if e := tx.QueryRow(r.Context(), "SELECT EXISTS(SELECT 1 FROM "+table+" WHERE id=$1 AND workspace_id=$2 AND "+fn+"($3,id,true))", parent, wid, current(r).ID).Scan(&allowed); e != nil || !allowed {
			return errors.New("상위 항목의 작성 권한이 변경되었습니다")
		}
	}
	var safe bool
	e := tx.QueryRow(r.Context(), `WITH RECURSIVE parents AS (
 SELECT id,parent_id,ARRAY[id] path,false cycle FROM `+table+` WHERE id=NULLIF($1,'')::uuid
 UNION ALL SELECT d.id,d.parent_id,p.path||d.id,d.id=ANY(p.path) FROM `+table+` d JOIN parents p ON d.id=p.parent_id WHERE NOT p.cycle AND cardinality(p.path)<21
 ), children AS (
 SELECT id,parent_id,ARRAY[id] path,false cycle FROM `+table+` WHERE id=$2
 UNION ALL SELECT d.id,d.parent_id,c.path||d.id,d.id=ANY(c.path) FROM `+table+` d JOIN children c ON d.parent_id=c.id WHERE NOT c.cycle AND cardinality(c.path)<21
 ) SELECT NOT EXISTS(SELECT 1 FROM parents WHERE id=$2 OR cycle) AND NOT EXISTS(SELECT 1 FROM children WHERE cycle) AND coalesce((SELECT max(cardinality(path)) FROM parents),0)+coalesce((SELECT max(cardinality(path)) FROM children),1)<=20`, parent, id).Scan(&safe)
	if e != nil {
		return e
	}
	if !safe {
		return errors.New("순환하는 위치로 이동하거나 최대 20단계의 탐색 깊이를 초과할 수 없습니다")
	}
	return nil
}
func (s *Server) deleteSpace(w http.ResponseWriter, r *http.Request) {
	v, ok := s.managedSpace(w, r)
	if !ok {
		return
	}
	// FK RESTRICT prevents content loss, including hidden/private and trashed pages.
	_, e := s.DB.Exec(r.Context(), "DELETE FROM spaces WHERE id=$1", str(v, "id"))
	if e != nil {
		apiError(w, 409, "하위 공간·문서·데이터베이스를 이동한 뒤 빈 공간만 삭제할 수 있습니다")
		return
	}
	s.audit(r, "SPACE_DELETE", str(v, "id"), nil)
	jsonResponse(w, 200, map[string]bool{"ok": true})
}
func (s *Server) spaceMembers(w http.ResponseWriter, r *http.Request) {
	v, ok := s.managedSpace(w, r)
	if !ok {
		return
	}
	rows, e := s.rows(r.Context(), `SELECT jsonb_build_object('id',u.id,'name',u.name,'email',u.email,'role',m.role) FROM space_members m JOIN users u ON u.id=m.user_id WHERE space_id=$1 ORDER BY u.name`, str(v, "id"))
	respond(w, rows, e)
}
func (s *Server) setSpaceMember(w http.ResponseWriter, r *http.Request) {
	v, ok := s.managedSpace(w, r)
	if !ok {
		return
	}
	var in struct {
		UserID string `json:"user_id"`
		Role   string `json:"role"`
	}
	if decode(r, &in) != nil || !validID(in.UserID) || !oneOf(in.Role, "admin", "editor", "commenter", "viewer", "remove") {
		apiError(w, 400, "사용자와 공간 권한을 확인하세요")
		return
	}
	var member bool
	_ = s.DB.QueryRow(r.Context(), "SELECT EXISTS(SELECT 1 FROM workspace_members WHERE workspace_id=$1 AND user_id=$2)", str(v, "workspace_id"), in.UserID).Scan(&member)
	if !member {
		apiError(w, 400, "먼저 워크스페이스에 사용자를 추가하세요")
		return
	}
	var e error
	if in.Role == "remove" {
		_, e = s.DB.Exec(r.Context(), "DELETE FROM space_members WHERE space_id=$1 AND user_id=$2", str(v, "id"), in.UserID)
	} else {
		_, e = s.DB.Exec(r.Context(), "INSERT INTO space_members VALUES($1,$2,$3) ON CONFLICT(space_id,user_id) DO UPDATE SET role=excluded.role", str(v, "id"), in.UserID, in.Role)
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "SPACE_PERMISSION_CHANGE", str(v, "id"), in)
	jsonResponse(w, 200, map[string]bool{"ok": true})
}

var workspaceSettingKeys = append([]string{"ai_enabled", "ai_base_url", "ai_api_key", "ai_model", "ai_max_tokens", "ai_system_prompt", "site_name", "theme_primary", "logo_url", "favicon_url", "review_period_days", "lifecycle_enabled", "feature_flags"}, ragSettingKeys()...)

func (s *Server) workspacePresentation(w http.ResponseWriter, r *http.Request) {
	wid := r.PathValue("id")
	if !s.canWorkspace(r.Context(), current(r), wid, false) {
		apiError(w, 403, "워크스페이스 접근 권한이 없습니다")
		return
	}
	cfg, e := s.effectiveSettings(r.Context(), wid)
	if e != nil {
		respond(w, nil, e)
		return
	}
	jsonResponse(w, 200, map[string]any{"site_name": cfg["site_name"], "theme_primary": cfg["theme_primary"], "logo_url": cfg["logo_url"], "favicon_url": cfg["favicon_url"], "feature_flags": s.featureFlagsFor(r.Context(), current(r), wid), "ai_enabled": cfg["ai_enabled"]})
}

func (s *Server) effectiveSettings(ctx context.Context, wid string) (map[string]any, error) {
	cfg, e := s.settings(ctx)
	if e != nil || wid == "" {
		return cfg, e
	}
	var raw []byte
	e = s.DB.QueryRow(ctx, "SELECT data FROM workspace_settings WHERE workspace_id=$1", wid).Scan(&raw)
	if errors.Is(e, pgx.ErrNoRows) {
		return cfg, nil
	}
	if e != nil {
		return nil, e
	}
	var overrides map[string]any
	if e = json.Unmarshal(raw, &overrides); e != nil {
		return nil, e
	}
	// Workspace administrators may select their own provider, but must never
	// receive a service-wide secret implicitly at a different endpoint.
	for _, pair := range [][2]string{{"ai_base_url", "ai_api_key"}, {"rag_embedding_base_url", "rag_embedding_api_key"}, {"rag_rerank_base_url", "rag_rerank_api_key"}} {
		if _, exists := overrides[pair[0]]; exists && str(overrides, pair[0]) != str(cfg, pair[0]) {
			cfg[pair[1]] = ""
		}
	}
	for key, value := range overrides {
		if isWorkspaceSecret(key) {
			plain, e := s.decrypt(str(overrides, key))
			if e != nil {
				return nil, e
			}
			cfg[key] = plain
		} else {
			cfg[key] = value
		}
	}
	return cfg, nil
}
func redactWorkspaceSettings(v map[string]any) map[string]any {
	out := map[string]any{}
	for k, val := range v {
		if isWorkspaceSecret(k) {
			out[k] = ""
			out[k+"_configured"] = val != ""
		} else {
			out[k] = val
		}
	}
	return out
}
func (s *Server) getWorkspaceSettings(w http.ResponseWriter, r *http.Request) {
	wid := r.PathValue("id")
	if !s.workspaceAdmin(r, wid) {
		apiError(w, 403, "워크스페이스 설정 권한이 없습니다")
		return
	}
	v, e := s.one(r.Context(), "SELECT to_jsonb(x) FROM workspace_settings x WHERE workspace_id=$1", wid)
	if errors.Is(e, pgx.ErrNoRows) {
		v = map[string]any{"data": map[string]any{}, "version": 0}
		e = nil
	}
	if e == nil {
		data, _ := v["data"].(map[string]any)
		v["data"] = redactWorkspaceSettings(data)
	}
	respond(w, v, e)
}
func (s *Server) saveWorkspaceSettings(w http.ResponseWriter, r *http.Request) {
	var in map[string]any
	if decode(r, &in) != nil {
		apiError(w, 400, "워크스페이스 설정 형식을 확인하세요")
		return
	}
	s.writeWorkspaceSettings(w, r, in, false)
}
func (s *Server) writeWorkspaceSettings(w http.ResponseWriter, r *http.Request, in map[string]any, replace bool) {
	wid := r.PathValue("id")
	if !s.workspaceAdmin(r, wid) {
		apiError(w, 403, "워크스페이스 설정 권한이 없습니다")
		return
	}
	data, ok := in["data"].(map[string]any)
	if !ok {
		apiError(w, 400, "data 설정 객체가 필요합니다")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	_, e = tx.Exec(r.Context(), "INSERT INTO workspace_settings(workspace_id) VALUES($1) ON CONFLICT DO NOTHING", wid)
	if e != nil {
		respond(w, nil, e)
		return
	}
	var version int
	var raw []byte
	e = tx.QueryRow(r.Context(), "SELECT version,data FROM workspace_settings WHERE workspace_id=$1 FOR UPDATE", wid).Scan(&version, &raw)
	if e != nil {
		respond(w, nil, e)
		return
	}
	expected := number(in, "version", -1)
	if expected != version && !(expected == 0 && version == 1 && string(raw) == "{}") {
		apiError(w, 409, "다른 관리자가 설정을 변경했습니다. 다시 불러오세요")
		return
	}
	stored := map[string]any{}
	if !replace {
		_ = json.Unmarshal(raw, &stored)
	}
	for key, value := range data {
		if !oneOf(key, workspaceSettingKeys...) {
			apiError(w, 400, "지원하지 않는 워크스페이스 설정: "+key)
			return
		}
		if value == nil {
			delete(stored, key)
			continue
		}
		if isWorkspaceSecret(key) {
			secret, ok := value.(string)
			if !ok {
				apiError(w, 400, "AI 키 형식을 확인하세요")
				return
			}
			if secret == "" && !replace {
				continue
			}
			if replace && strings.HasPrefix(secret, "enc:") {
				if _, e = s.decrypt(secret); e != nil {
					apiError(w, 400, "이전 키를 복호화하지 못했습니다")
					return
				}
				stored[key] = secret
			} else {
				encrypted, e := s.encrypt(secret)
				if e != nil {
					respond(w, nil, e)
					return
				}
				stored[key] = encrypted
			}
		} else {
			stored[key] = value
		}
	}
	cfg, e := s.settings(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	for key, value := range stored {
		cfg[key] = value
		if isWorkspaceSecret(key) {
			cfg[key], e = s.decrypt(value.(string))
			if e != nil {
				respond(w, nil, e)
				return
			}
		}
	}
	if e = validateSettings(cfg); e != nil {
		apiError(w, 400, e.Error())
		return
	}
	if n, exists := stored["review_period_days"]; exists {
		f, ok := n.(float64)
		if !ok || f < 1 || f > 3650 || f != float64(int(f)) {
			apiError(w, 400, "검토 주기는 1~3650일로 입력하세요")
			return
		}
	}
	if flags, exists := stored["feature_flags"]; exists {
		if e := validateFeatureFlags(flags); e != nil {
			apiError(w, 400, e.Error())
			return
		}
	}
	if v, exists := stored["lifecycle_enabled"]; exists {
		if _, ok := v.(bool); !ok {
			apiError(w, 400, "문서 생명주기 자동화는 활성 여부로 지정하세요")
			return
		}
	}
	if e := validateWorkspaceBranding(r.Context(), tx, wid, stored); e != nil {
		apiError(w, 400, e.Error())
		return
	}
	_, e = tx.Exec(r.Context(), "INSERT INTO workspace_settings_history(id,workspace_id,user_id,version,data) VALUES($1,$2,$3,$4,$5) ON CONFLICT(workspace_id,version) DO NOTHING", newID(), wid, current(r).ID, version, raw)
	if e == nil {
		_, e = tx.Exec(r.Context(), "UPDATE workspace_settings SET data=$2,version=version+1,updated_at=now() WHERE workspace_id=$1", wid, jsonValue(stored))
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "WORKSPACE_SETTINGS_UPDATE", wid, map[string]any{"version": version + 1, "restored": replace})
	jsonResponse(w, 200, map[string]any{"data": redactWorkspaceSettings(stored), "version": version + 1})
}
func (s *Server) workspaceSettingsHistory(w http.ResponseWriter, r *http.Request) {
	wid := r.PathValue("id")
	if !s.workspaceAdmin(r, wid) {
		apiError(w, 403, "설정 이력 권한이 없습니다")
		return
	}
	v, e := s.rows(r.Context(), "SELECT to_jsonb(h)-'data' FROM workspace_settings_history h WHERE workspace_id=$1 ORDER BY version DESC LIMIT 100", wid)
	respond(w, v, e)
}
func (s *Server) restoreWorkspaceSettings(w http.ResponseWriter, r *http.Request) {
	wid, hid := r.PathValue("id"), r.PathValue("historyId")
	if !s.workspaceAdmin(r, wid) || !validID(hid) {
		apiError(w, 403, "설정 복원 권한이 없습니다")
		return
	}
	var in map[string]any
	if decode(r, &in) != nil {
		apiError(w, 400, "현재 설정 version이 필요합니다")
		return
	}
	var raw []byte
	if e := s.DB.QueryRow(r.Context(), "SELECT data FROM workspace_settings_history WHERE id=$1 AND workspace_id=$2", hid, wid).Scan(&raw); e != nil {
		apiError(w, 404, "설정 이력이 없습니다")
		return
	}
	var data map[string]any
	if e := json.Unmarshal(raw, &data); e != nil {
		respond(w, nil, e)
		return
	}
	in["data"] = data
	s.writeWorkspaceSettings(w, r, in, true)
}

func (s *Server) organizationAdmin(r *http.Request, id string) bool {
	p := current(r)
	if !validID(id) || p.TokenID != "" || p.ScopeRestricted || p.Role == "viewer" {
		return false
	}
	var allowed bool
	e := s.DB.QueryRow(r.Context(), "SELECT EXISTS(SELECT 1 FROM organization_members WHERE organization_id=$1 AND user_id=$2 AND role IN ('owner','admin'))", id, p.ID).Scan(&allowed)
	return e == nil && allowed
}
func (s *Server) listOrganizations(w http.ResponseWriter, r *http.Request) {
	v, e := s.rows(r.Context(), "SELECT to_jsonb(o)||jsonb_build_object('role',m.role) FROM organizations o JOIN organization_members m ON m.organization_id=o.id WHERE m.user_id=$1 ORDER BY o.name", current(r).ID)
	respond(w, v, e)
}
func (s *Server) createOrganization(w http.ResponseWriter, r *http.Request) {
	p := current(r)
	if p.TokenID != "" || p.ScopeRestricted || p.Role == "viewer" {
		apiError(w, 403, "조직 생성 권한이 없습니다")
		return
	}
	var in map[string]any
	if decode(r, &in) != nil {
		apiError(w, 400, "조직 입력값을 확인하세요")
		return
	}
	name, slug := strings.TrimSpace(str(in, "name")), str(in, "slug")
	id := newID()
	if slug == "" {
		slug = "org-" + id[:8]
	}
	if name == "" || len(name) > 200 || !simpleSlug.MatchString(slug) {
		apiError(w, 400, "조직 이름과 주소를 확인하세요")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	_, e = tx.Exec(r.Context(), "INSERT INTO organizations(id,name,slug) VALUES($1,$2,$3)", id, name, slug)
	if e == nil {
		_, e = tx.Exec(r.Context(), "INSERT INTO organization_members VALUES($1,$2,'owner')", id, p.ID)
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		apiError(w, 409, "조직 주소가 중복됩니다")
		return
	}
	s.audit(r, "ORGANIZATION_CREATE", id, nil)
	jsonResponse(w, 200, map[string]any{"id": id, "name": name, "slug": slug, "role": "owner"})
}
func (s *Server) organizationMembers(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !s.organizationAdmin(r, id) {
		apiError(w, 403, "조직 관리 권한이 없습니다")
		return
	}
	v, e := s.rows(r.Context(), "SELECT jsonb_build_object('id',u.id,'name',u.name,'email',u.email,'role',m.role) FROM organization_members m JOIN users u ON u.id=m.user_id WHERE organization_id=$1 ORDER BY u.name", id)
	respond(w, v, e)
}
func (s *Server) setOrganizationMember(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !s.organizationAdmin(r, id) {
		apiError(w, 403, "조직 관리 권한이 없습니다")
		return
	}
	var in struct {
		UserID string `json:"user_id"`
		Email  string `json:"email"`
		Role   string `json:"role"`
	}
	if decode(r, &in) != nil || !oneOf(in.Role, "admin", "member", "remove") {
		apiError(w, 400, "사용자와 조직 권한을 확인하세요")
		return
	}
	if !validID(in.UserID) {
		_ = s.DB.QueryRow(r.Context(), "SELECT id::text FROM users WHERE email=$1 AND NOT disabled", strings.ToLower(strings.TrimSpace(in.Email))).Scan(&in.UserID)
	}
	if !validID(in.UserID) {
		apiError(w, 400, "등록된 사용자를 선택하세요")
		return
	}
	var role string
	_ = s.DB.QueryRow(r.Context(), "SELECT role FROM organization_members WHERE organization_id=$1 AND user_id=$2", id, in.UserID).Scan(&role)
	if role == "owner" {
		apiError(w, 403, "조직 소유자의 권한을 삭제하거나 낮출 수 없습니다")
		return
	}
	var e error
	if in.Role == "remove" {
		_, e = s.DB.Exec(r.Context(), "DELETE FROM organization_members WHERE organization_id=$1 AND user_id=$2", id, in.UserID)
	} else {
		_, e = s.DB.Exec(r.Context(), "INSERT INTO organization_members VALUES($1,$2,$3) ON CONFLICT(organization_id,user_id) DO UPDATE SET role=excluded.role", id, in.UserID, in.Role)
	}
	if e != nil {
		apiError(w, 400, "사용자가 존재하는지 확인하세요")
		return
	}
	s.audit(r, "ORGANIZATION_PERMISSION_CHANGE", id, in)
	jsonResponse(w, 200, map[string]bool{"ok": true})
}
func (s *Server) setWorkspaceOrganization(w http.ResponseWriter, r *http.Request) {
	wid := r.PathValue("id")
	if !s.workspaceAdmin(r, wid) {
		apiError(w, 403, "워크스페이스 관리 권한이 없습니다")
		return
	}
	var in struct {
		OrganizationID string `json:"organization_id"`
	}
	if decode(r, &in) != nil {
		apiError(w, 400, "조직을 선택하세요")
		return
	}
	var old string
	_ = s.DB.QueryRow(r.Context(), "SELECT coalesce(organization_id::text,'') FROM workspaces WHERE id=$1", wid).Scan(&old)
	if (old != "" && !s.organizationAdmin(r, old)) || (in.OrganizationID != "" && !s.organizationAdmin(r, in.OrganizationID)) {
		apiError(w, 403, "현재 조직과 대상 조직의 관리 권한이 모두 필요합니다")
		return
	}
	_, e := s.DB.Exec(r.Context(), "UPDATE workspaces SET organization_id=NULLIF($2,'')::uuid WHERE id=$1", wid, in.OrganizationID)
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "WORKSPACE_ORGANIZATION_CHANGE", wid, in)
	jsonResponse(w, 200, map[string]bool{"ok": true})
}
