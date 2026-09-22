package server

import (
	"archive/zip"
	"bytes"
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"regexp"
	"strings"
)

//go:embed plugins.sql
var pluginsSchema string
var pluginIDPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{2,63}$`)
var pluginVersionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:-[a-zA-Z0-9.-]+)?$`)
var pluginContributionPattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,63}$`)
var pluginCapabilities = []string{"document:read", "document:write", "database:read", "database:write", "ai:execute", "storage:personal", "ui:notify", "ui:navigate", "file:import", "file:export"}

type pluginContribution struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Extensions  []string `json:"extensions,omitempty"`
	Provider    string   `json:"provider,omitempty"`
}
type pluginManifest struct {
	ID            string                          `json:"id"`
	Name          string                          `json:"name"`
	Version       string                          `json:"version"`
	APIVersion    int                             `json:"api_version"`
	Description   string                          `json:"description"`
	Author        string                          `json:"author"`
	Entry         string                          `json:"entry"`
	Style         string                          `json:"style,omitempty"`
	Capabilities  []string                        `json:"capabilities"`
	Contributions map[string][]pluginContribution `json:"contributions"`
}
type pluginFile struct {
	MIME string `json:"mime"`
	Data string `json:"data"`
}
type pluginPackage struct {
	Manifest pluginManifest
	Files    map[string]pluginFile
}

func (s *Server) migratePlugins(ctx context.Context) error {
	_, e := s.DB.Exec(ctx, pluginsSchema)
	return e
}
func (s *Server) registerPlugins() {
	s.admin("GET /api/v1/admin/plugins", s.adminPlugins)
	s.admin("POST /api/v1/admin/plugins", s.installPlugin)
	s.admin("PUT /api/v1/admin/plugins/{id}", s.configurePlugin)
	s.handle("GET /api/v1/workspaces/{id}/plugins", s.workspacePlugins)
	s.handle("PUT /api/v1/workspaces/{id}/plugins/{pluginId}", s.configureWorkspacePlugin)
	s.handle("GET /api/v1/plugins", s.listPlugins)
	s.handle("GET /api/v1/plugins/{id}/runtime", s.pluginRuntime)
	s.handle("POST /api/v1/plugins/{id}/bridge", s.pluginBridge)
}
func pluginContains(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}
func validatePluginManifest(m *pluginManifest) error {
	if !pluginIDPattern.MatchString(m.ID) || !pluginVersionPattern.MatchString(m.Version) || len(m.Version) > 64 || m.APIVersion != 1 || strings.TrimSpace(m.Name) == "" || len(m.Name) > 200 || len(m.Description) > 4000 || len(m.Author) > 200 || m.Entry != "main.js" || (m.Style != "" && m.Style != "style.css") {
		return errors.New("manifest의 ID·버전·이름·API 버전(1)·진입 파일(main.js)을 확인하세요")
	}
	if len(m.Capabilities) > len(pluginCapabilities) {
		return errors.New("요청 권한이 너무 많습니다")
	}
	seen := map[string]bool{}
	for _, cap := range m.Capabilities {
		if seen[cap] || !pluginContains(pluginCapabilities, cap) {
			return fmt.Errorf("잘못되거나 중복된 권한: %s", cap)
		}
		seen[cap] = true
	}
	if m.Capabilities == nil {
		m.Capabilities = []string{}
	}
	if m.Contributions == nil {
		m.Contributions = map[string][]pluginContribution{}
	}
	count := 0
	for kind, items := range m.Contributions {
		if !oneOf(kind, "blocks", "commands", "sidebars", "menus", "importers", "exporters", "ai_providers") {
			return errors.New("알 수 없는 플러그인 확장 유형입니다")
		}
		ids := map[string]bool{}
		for _, item := range items {
			count++
			if !pluginContributionPattern.MatchString(item.ID) || ids[item.ID] || strings.TrimSpace(item.Title) == "" || len(item.Title) > 160 || len(item.Description) > 1000 || len(item.Extensions) > 20 {
				return errors.New("확장 ID·제목·파일 확장자를 확인하세요")
			}
			ids[item.ID] = true
			for _, ext := range item.Extensions {
				if !regexp.MustCompile(`^\.[a-z0-9]{1,12}$`).MatchString(ext) {
					return errors.New("가져오기·내보내기 확장자는 .md와 같은 형식이어야 합니다")
				}
			}
			if kind == "importers" && !seen["file:import"] || kind == "exporters" && !seen["file:export"] {
				return errors.New("가져오기·내보내기 기능에는 명시적인 파일 권한이 필요합니다")
			}
			if kind == "ai_providers" && (item.Provider != "workspace" || !seen["ai:execute"]) {
				return errors.New("AI 확장은 관리자가 설정한 workspace 공급자만 사용할 수 있습니다")
			}
		}
	}
	if count > 100 {
		return errors.New("확장은 100개까지 등록할 수 있습니다")
	}
	return nil
}
func readPluginPackage(content []byte) (pluginPackage, error) {
	result := pluginPackage{Files: map[string]pluginFile{}}
	if len(content) > 10<<20 {
		return result, errors.New("플러그인 ZIP은 10MB 이하여야 합니다")
	}
	archive, e := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if e != nil {
		return result, errors.New("올바른 ZIP 파일을 선택하세요")
	}
	if len(archive.File) > 200 {
		return result, errors.New("ZIP 파일 항목은 200개까지 지원합니다")
	}
	var total uint64
	seen := map[string]bool{}
	for _, f := range archive.File {
		name := f.Name
		if name == "" || len(name) > 240 || strings.ContainsAny(name, "\\\x00:") || strings.HasPrefix(name, "/") || path.Clean(name) != strings.TrimSuffix(name, "/") || strings.HasPrefix(name, "../") || name == ".." || (!f.FileInfo().IsDir() && !f.Mode().IsRegular()) {
			return result, errors.New("ZIP에 안전하지 않은 경로 또는 특수 파일이 있습니다")
		}
		if seen[strings.ToLower(name)] {
			return result, errors.New("ZIP에 중복 파일이 있습니다")
		}
		seen[strings.ToLower(name)] = true
		if f.FileInfo().IsDir() {
			continue
		}
		if f.UncompressedSize64 > 4<<20 || total+f.UncompressedSize64 > 20<<20 || f.UncompressedSize64 > 1<<20 && f.CompressedSize64 > 0 && f.UncompressedSize64/f.CompressedSize64 > 200 {
			return result, errors.New("ZIP 압축 해제 크기 또는 압축 비율 한도를 초과했습니다")
		}
		total += f.UncompressedSize64
		mime := map[string]string{".js": "text/javascript", ".css": "text/css", ".json": "application/json", ".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".gif": "image/gif", ".webp": "image/webp", ".woff2": "font/woff2", ".txt": "text/plain", ".md": "text/plain"}[strings.ToLower(path.Ext(name))]
		if mime == "" {
			return result, errors.New("ZIP에는 JavaScript·CSS·JSON·문서·래스터 이미지·WOFF2만 포함할 수 있습니다")
		}
		file, e := f.Open()
		if e != nil {
			return result, e
		}
		data, e := io.ReadAll(io.LimitReader(file, 4<<20+1))
		file.Close()
		if e != nil || len(data) > 4<<20 || uint64(len(data)) != f.UncompressedSize64 {
			return result, errors.New("ZIP 파일 크기 또는 CRC 검증에 실패했습니다")
		}
		result.Files[name] = pluginFile{MIME: mime, Data: base64.StdEncoding.EncodeToString(data)}
	}
	manifest, ok := result.Files["manifest.json"]
	if !ok {
		return result, errors.New("ZIP 루트에 manifest.json이 필요합니다")
	}
	raw, _ := base64.StdEncoding.DecodeString(manifest.Data)
	if len(raw) > 65536 {
		return result, errors.New("manifest.json은 64KB 이하여야 합니다")
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if e = dec.Decode(&result.Manifest); e != nil {
		return result, fmt.Errorf("manifest.json 형식을 확인하세요: %w", e)
	}
	var trailing any
	if dec.Decode(&trailing) != io.EOF {
		return result, errors.New("manifest.json에는 단일 JSON 객체만 허용됩니다")
	}
	if e = validatePluginManifest(&result.Manifest); e != nil {
		return result, e
	}
	if _, ok = result.Files[result.Manifest.Entry]; !ok {
		return result, errors.New("main.js가 없습니다")
	}
	if result.Manifest.Style != "" {
		if _, ok = result.Files[result.Manifest.Style]; !ok {
			return result, errors.New("style.css가 없습니다")
		}
	}
	return result, nil
}
func (s *Server) adminPlugins(w http.ResponseWriter, r *http.Request) {
	items, e := s.rows(r.Context(), `SELECT (to_jsonb(p)-'files')||jsonb_build_object('workspace_count',(SELECT count(*) FROM workspace_plugins wp WHERE wp.plugin_id=p.id AND wp.enabled)) FROM plugins p ORDER BY created_at DESC`)
	respond(w, map[string]any{"plugins": items, "capabilities": pluginCapabilities}, e)
}
func (s *Server) installPlugin(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 11<<20)
	if e := r.ParseMultipartForm(1 << 20); e != nil {
		apiError(w, 400, "10MB 이하 플러그인 ZIP을 선택하세요")
		return
	}
	defer r.MultipartForm.RemoveAll()
	file, _, e := r.FormFile("file")
	if e != nil {
		apiError(w, 400, "ZIP 파일이 필요합니다")
		return
	}
	defer file.Close()
	raw, e := io.ReadAll(io.LimitReader(file, 10<<20+1))
	if e != nil {
		apiError(w, 400, "ZIP 파일을 읽지 못했습니다")
		return
	}
	pkg, e := readPluginPackage(raw)
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
	var existing bool
	if e = tx.QueryRow(r.Context(), "SELECT EXISTS(SELECT 1 FROM plugins WHERE id=$1)", pkg.Manifest.ID).Scan(&existing); e != nil {
		respond(w, nil, e)
		return
	}
	if existing && r.FormValue("replace") != "REPLACE" {
		apiError(w, 409, "이미 설치된 ID입니다. 업데이트 시 모든 워크스페이스 권한이 해제됩니다. REPLACE 확인이 필요합니다")
		return
	}
	_, e = tx.Exec(r.Context(), `INSERT INTO plugins(id,manifest,files,installed_by) VALUES($1,$2,$3,$4) ON CONFLICT(id) DO UPDATE SET manifest=excluded.manifest,files=excluded.files,version=plugins.version+1,enabled=true,installed_by=excluded.installed_by,updated_at=now()`, pkg.Manifest.ID, jsonValue(pkg.Manifest), jsonValue(pkg.Files), current(r).ID)
	if e == nil && existing {
		_, e = tx.Exec(r.Context(), "UPDATE workspace_plugins SET enabled=false,capabilities='[]',version=version+1,updated_at=now() WHERE plugin_id=$1", pkg.Manifest.ID)
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "PLUGIN_INSTALL", pkg.Manifest.ID, map[string]any{"version": pkg.Manifest.Version, "replaced": existing})
	jsonResponse(w, 200, map[string]any{"id": pkg.Manifest.ID, "manifest": pkg.Manifest, "permissions_reset": existing})
}
func (s *Server) configurePlugin(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Enabled bool `json:"enabled"`
	}
	if decode(r, &in) != nil {
		apiError(w, 400, "설정을 확인하세요")
		return
	}
	result, e := s.DB.Exec(r.Context(), "UPDATE plugins SET enabled=$2,version=version+1,updated_at=now() WHERE id=$1", r.PathValue("id"), in.Enabled)
	if e != nil {
		respond(w, nil, e)
		return
	}
	if result.RowsAffected() == 0 {
		apiError(w, 404, "플러그인을 찾을 수 없습니다")
		return
	}
	s.audit(r, "PLUGIN_CONFIGURE", r.PathValue("id"), map[string]any{"enabled": in.Enabled})
	jsonResponse(w, 200, map[string]any{"ok": true})
}
func (s *Server) workspacePlugins(w http.ResponseWriter, r *http.Request) {
	wid := r.PathValue("id")
	if !s.workspaceAdmin(r, wid) {
		apiError(w, 403, "워크스페이스 관리자 권한이 필요합니다")
		return
	}
	items, e := s.rows(r.Context(), `SELECT jsonb_build_object('id',p.id,'manifest',p.manifest,'global_enabled',p.enabled,'enabled',coalesce(wp.enabled,false),'capabilities',coalesce(wp.capabilities,'[]'::jsonb),'version',coalesce(wp.version,0)) FROM plugins p LEFT JOIN workspace_plugins wp ON wp.plugin_id=p.id AND wp.workspace_id=$1 ORDER BY p.manifest->>'name'`, wid)
	respond(w, items, e)
}
func (s *Server) configureWorkspacePlugin(w http.ResponseWriter, r *http.Request) {
	wid, id := r.PathValue("id"), r.PathValue("pluginId")
	if !s.workspaceAdmin(r, wid) {
		apiError(w, 403, "워크스페이스 관리자 권한이 필요합니다")
		return
	}
	var in struct {
		Enabled      bool     `json:"enabled"`
		Capabilities []string `json:"capabilities"`
	}
	if decode(r, &in) != nil {
		apiError(w, 400, "활성 상태와 권한을 확인하세요")
		return
	}
	if in.Capabilities == nil {
		in.Capabilities = []string{}
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	var raw []byte
	var enabled bool
	if e = tx.QueryRow(r.Context(), "SELECT manifest,enabled FROM plugins WHERE id=$1 FOR SHARE", id).Scan(&raw, &enabled); e != nil {
		apiError(w, 404, "플러그인을 찾을 수 없습니다")
		return
	}
	var manifest pluginManifest
	json.Unmarshal(raw, &manifest)
	seen := map[string]bool{}
	for _, cap := range in.Capabilities {
		if seen[cap] || !pluginContains(manifest.Capabilities, cap) {
			apiError(w, 400, "manifest에서 요청한 권한만 중복 없이 허용할 수 있습니다")
			return
		}
		seen[cap] = true
	}
	if in.Enabled && !enabled {
		apiError(w, 409, "서비스 관리자가 비활성화한 플러그인입니다")
		return
	}
	_, e = tx.Exec(r.Context(), `INSERT INTO workspace_plugins(workspace_id,plugin_id,enabled,capabilities,updated_by) VALUES($1,$2,$3,$4,$5) ON CONFLICT(workspace_id,plugin_id) DO UPDATE SET enabled=excluded.enabled,capabilities=excluded.capabilities,updated_by=excluded.updated_by,version=workspace_plugins.version+1,updated_at=now()`, wid, id, in.Enabled, jsonValue(in.Capabilities), current(r).ID)
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "PLUGIN_GRANT", id, map[string]any{"workspace_id": wid, "enabled": in.Enabled, "capabilities": in.Capabilities})
	jsonResponse(w, 200, map[string]any{"ok": true})
}
func (s *Server) listPlugins(w http.ResponseWriter, r *http.Request) {
	wid := r.URL.Query().Get("workspace_id")
	if !s.canWorkspace(r.Context(), current(r), wid, false) {
		apiError(w, 403, "워크스페이스 접근 권한이 없습니다")
		return
	}
	items, e := s.rows(r.Context(), `SELECT jsonb_build_object('id',p.id,'manifest',p.manifest,'capabilities',wp.capabilities,'version',p.version,'grant_version',wp.version) FROM plugins p JOIN workspace_plugins wp ON wp.plugin_id=p.id WHERE wp.workspace_id=$1 AND p.enabled AND wp.enabled ORDER BY p.manifest->>'name'`, wid)
	w.Header().Set("Cache-Control", "no-store")
	respond(w, items, e)
}
func (s *Server) pluginGrant(r *http.Request, id, wid string) (pluginManifest, []string, error) {
	var manifest pluginManifest
	caps := []string{}
	if !pluginIDPattern.MatchString(id) || !s.canWorkspace(r.Context(), current(r), wid, false) {
		return manifest, caps, errors.New("플러그인 또는 워크스페이스 접근 권한이 없습니다")
	}
	if !s.canFeature(r.Context(), current(r), wid, "plugins") {
		return manifest, caps, errors.New("현재 기능 공개 정책에서 플러그인 실행이 비활성화되었습니다")
	}
	var raw, grants []byte
	e := s.DB.QueryRow(r.Context(), `SELECT p.manifest,wp.capabilities FROM plugins p JOIN workspace_plugins wp ON p.id=wp.plugin_id WHERE p.id=$1 AND wp.workspace_id=$2 AND p.enabled AND wp.enabled AND EXISTS(SELECT 1 FROM users WHERE id=$3 AND NOT disabled) AND madi_feature_allowed($3,$2,'plugins')`, id, wid, current(r).ID).Scan(&raw, &grants)
	if e != nil {
		return manifest, caps, errors.New("플러그인 사용 권한이 취소되었거나 비활성화되었습니다")
	}
	if json.Unmarshal(raw, &manifest) != nil || json.Unmarshal(grants, &caps) != nil {
		return manifest, caps, errors.New("플러그인 설정을 읽지 못했습니다")
	}
	for _, cap := range caps {
		if !pluginContains(manifest.Capabilities, cap) {
			return manifest, nil, errors.New("플러그인 권한을 다시 승인하세요")
		}
	}
	return manifest, caps, nil
}
func (s *Server) pluginRuntime(w http.ResponseWriter, r *http.Request) {
	id, wid := r.PathValue("id"), r.URL.Query().Get("workspace_id")
	manifest, caps, e := s.pluginGrant(r, id, wid)
	if e != nil {
		apiError(w, 403, e.Error())
		return
	}
	var files []byte
	var version, grant int
	e = s.DB.QueryRow(r.Context(), `SELECT p.files,p.version,wp.version FROM plugins p JOIN workspace_plugins wp ON wp.plugin_id=p.id WHERE p.id=$1 AND wp.workspace_id=$2 AND p.enabled AND wp.enabled AND madi_feature_allowed($3,$2,'plugins')`, id, wid, current(r).ID).Scan(&files, &version, &grant)
	if e != nil {
		apiError(w, 403, "플러그인 사용 권한을 확인하세요")
		return
	}
	var bundle map[string]pluginFile
	json.Unmarshal(files, &bundle)
	w.Header().Set("Cache-Control", "no-store")
	jsonResponse(w, 200, map[string]any{"manifest": manifest, "capabilities": caps, "files": bundle, "version": version, "grant_version": grant})
}
