package server

import (
	"bytes"
	"context"
	"net/http"
	"os"
	"strings"
	"time"
)

var storageSecrets = []string{"access_key", "secret_key", "session_token"}

func storageRedact(p storageProvider) map[string]any {
	config := map[string]any{}
	for k, v := range p.Config {
		config[k] = v
	}
	for _, key := range storageSecrets {
		config[key+"_configured"] = str(config, key) != ""
		config[key] = ""
	}
	return map[string]any{"id": p.ID, "workspace_id": p.WorkspaceID, "name": p.Name, "kind": p.Kind, "enabled": p.Enabled, "config": config}
}
func (s *Server) storageManage(r *http.Request, p storageProvider) bool {
	if current(r).TokenID != "" || current(r).ScopeRestricted {
		return false
	}
	if p.WorkspaceID == "" {
		return current(r).Role == "admin"
	}
	return s.automationManager(r.Context(), current(r), p.WorkspaceID)
}
func (s *Server) registerStorage() {
	s.handle("GET /api/v1/storage/providers", s.listStorageProviders)
	s.handle("POST /api/v1/storage/providers", s.saveStorageProvider)
	s.handle("PUT /api/v1/storage/providers/{id}", s.saveStorageProvider)
	s.handle("POST /api/v1/storage/providers/{id}/test", s.testStorageProvider)
	s.handle("GET /api/v1/storage/assignment", s.getStorageAssignment)
	s.handle("PUT /api/v1/storage/assignment", s.setStorageAssignment)
	s.registerBackupJobs()
}
func (s *Server) listStorageProviders(w http.ResponseWriter, r *http.Request) {
	wid := r.URL.Query().Get("workspace_id")
	p := current(r)
	if p.TokenID != "" || p.ScopeRestricted || (wid == "" && p.Role != "admin") || (wid != "" && !s.automationManager(r.Context(), p, wid)) {
		apiError(w, 403, "저장소 관리 권한이 없습니다")
		return
	}
	rows, e := s.DB.Query(r.Context(), "SELECT id::text FROM storage_providers WHERE ($1='' AND $2) OR workspace_id::text=$1 OR workspace_id IS NULL ORDER BY name", wid, p.Role == "admin")
	if e != nil {
		respond(w, nil, e)
		return
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if e = rows.Scan(&id); e != nil {
			break
		}
		ids = append(ids, id)
	}
	rows.Close()
	if e != nil {
		respond(w, nil, e)
		return
	}
	out := []map[string]any{}
	for _, id := range ids {
		provider, e := s.storageProvider(r.Context(), id)
		if e != nil {
			respond(w, nil, e)
			return
		}
		v := storageRedact(provider)
		v["can_manage"] = s.storageManage(r, provider)
		if provider.WorkspaceID == "" && p.Role != "admin" {
			v["config"] = map[string]any{}
		}
		out = append(out, v)
	}
	jsonResponse(w, 200, out)
}
func (s *Server) saveStorageProvider(w http.ResponseWriter, r *http.Request) {
	var in struct {
		WorkspaceID, Name, Kind string
		Config                  map[string]any
		Enabled                 bool
	}
	var raw map[string]any
	if decode(r, &raw) != nil {
		apiError(w, 400, "저장소 설정을 확인하세요")
		return
	}
	in.WorkspaceID = str(raw, "workspace_id")
	in.Name = str(raw, "name")
	in.Kind = str(raw, "kind")
	in.Config, _ = raw["config"].(map[string]any)
	in.Enabled = boolean(raw, "enabled")
	id := r.PathValue("id")
	creating := id == ""
	p := storageProvider{WorkspaceID: in.WorkspaceID, Kind: in.Kind, Config: in.Config}
	var old storageProvider
	var e error
	if !creating {
		old, e = s.storageProvider(r.Context(), id)
		if e != nil {
			apiError(w, 404, "저장소 설정을 찾을 수 없습니다")
			return
		}
		p.WorkspaceID = old.WorkspaceID
	}
	if !s.storageManage(r, p) || (in.Kind == "local" && current(r).Role != "admin") {
		apiError(w, 403, "로컬 경로는 서비스 관리자만 설정하며 S3 연결은 공간 관리자가 설정합니다")
		return
	}
	if strings.TrimSpace(in.Name) == "" || len(in.Name) > 120 || in.Config == nil || len(jsonValue(in.Config)) > 128<<10 {
		apiError(w, 400, "저장소 이름과 설정을 확인하세요")
		return
	}
	if !creating {
		for _, key := range storageSecrets {
			if key == "session_token" && boolean(in.Config, "clear_session_token") {
				in.Config[key] = ""
			} else if str(in.Config, key) == "" {
				in.Config[key] = str(old.Config, key)
			}
		}
	}
	delete(in.Config, "clear_session_token")
	if e = validateStorageConfig(in.Kind, in.Config); e != nil {
		apiError(w, 400, e.Error())
		return
	}
	if !creating {
		changed := in.Kind != old.Kind
		for _, key := range []string{"endpoint", "bucket", "prefix", "root", "region"} {
			changed = changed || str(in.Config, key) != str(old.Config, key)
		}
		if changed {
			apiError(w, 409, "저장소 위치는 생성 후 변경하지 않습니다. 새 연결을 만든 후 기본 저장소를 전환하세요")
			return
		}
	}
	config := map[string]any{}
	for k, v := range in.Config {
		if !strings.HasSuffix(k, "_configured") {
			config[k] = v
		}
	}
	for _, key := range storageSecrets {
		if secret := str(config, key); secret != "" {
			config[key], e = s.encrypt(secret)
			if e != nil {
				respond(w, nil, e)
				return
			}
		}
	}
	if creating {
		id = newID()
		_, e = s.DB.Exec(r.Context(), "INSERT INTO storage_providers(id,workspace_id,owner_id,name,kind,config,enabled) VALUES($1,NULLIF($2,'')::uuid,$3,$4,$5,$6,$7)", id, p.WorkspaceID, current(r).ID, strings.TrimSpace(in.Name), in.Kind, jsonValue(config), in.Enabled)
	} else {
		_, e = s.DB.Exec(r.Context(), "UPDATE storage_providers SET name=$2,kind=$3,config=$4,enabled=$5,updated_at=now() WHERE id=$1", id, strings.TrimSpace(in.Name), in.Kind, jsonValue(config), in.Enabled)
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "STORAGE_CONFIGURE", id, map[string]any{"kind": in.Kind, "enabled": in.Enabled, "workspace_id": p.WorkspaceID})
	p, e = s.storageProvider(r.Context(), id)
	respond(w, storageRedact(p), e)
}
func (s *Server) getStorageAssignment(w http.ResponseWriter, r *http.Request) {
	wid := r.URL.Query().Get("workspace_id")
	if current(r).TokenID != "" || current(r).ScopeRestricted || (wid == "" && current(r).Role != "admin") || (wid != "" && !s.automationManager(r.Context(), current(r), wid)) {
		apiError(w, 403, "저장소 관리 권한이 없습니다")
		return
	}
	var id string
	query := "SELECT coalesce(provider_id::text,'') FROM storage_settings WHERE id=1"
	args := []any{}
	if wid != "" {
		query = "SELECT coalesce((SELECT provider_id::text FROM storage_assignments WHERE workspace_id=$1),'')"
		args = append(args, wid)
	}
	e := s.DB.QueryRow(r.Context(), query, args...).Scan(&id)
	respond(w, map[string]any{"workspace_id": wid, "provider_id": id}, e)
}
func (s *Server) setStorageAssignment(w http.ResponseWriter, r *http.Request) {
	var in struct {
		WorkspaceID string `json:"workspace_id"`
		ProviderID  string `json:"provider_id"`
	}
	if decode(r, &in) != nil {
		apiError(w, 400, "저장소 선택을 확인하세요")
		return
	}
	if current(r).TokenID != "" || current(r).ScopeRestricted || (in.WorkspaceID == "" && current(r).Role != "admin") || (in.WorkspaceID != "" && !s.automationManager(r.Context(), current(r), in.WorkspaceID)) {
		apiError(w, 403, "저장소 관리 권한이 없습니다")
		return
	}
	if in.ProviderID != "" {
		p, e := s.storageProvider(r.Context(), in.ProviderID)
		if e != nil || !p.Enabled || (p.WorkspaceID != "" && p.WorkspaceID != in.WorkspaceID) {
			apiError(w, 403, "같은 워크스페이스 또는 공용 활성 저장소를 선택하세요")
			return
		}
	}
	var e error
	if in.WorkspaceID == "" {
		_, e = s.DB.Exec(r.Context(), "UPDATE storage_settings SET provider_id=NULLIF($1,'')::uuid WHERE id=1", in.ProviderID)
	} else {
		_, e = s.DB.Exec(r.Context(), "INSERT INTO storage_assignments(workspace_id,provider_id) VALUES($1,NULLIF($2,'')::uuid) ON CONFLICT(workspace_id) DO UPDATE SET provider_id=EXCLUDED.provider_id", in.WorkspaceID, in.ProviderID)
	}
	if e == nil {
		s.audit(r, "STORAGE_ASSIGNMENT", in.WorkspaceID, map[string]string{"provider_id": in.ProviderID})
	}
	respond(w, map[string]bool{"ok": true}, e)
}
func (s *Server) testStorageProvider(w http.ResponseWriter, r *http.Request) {
	p, e := s.storageProvider(r.Context(), r.PathValue("id"))
	if e != nil || !s.storageManage(r, p) {
		apiError(w, 403, "저장소 진단 권한이 없습니다")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	content := []byte("madi storage connection verification\n")
	key := "diagnostics/" + newID()
	object, e := s.putStoredObject(ctx, p, key, bytes.NewReader(content), int64(len(content)), "text/plain")
	if e != nil {
		_ = s.cleanupStoredObject(ctx, object)
		apiError(w, 502, "저장소 쓰기 진단 실패: 주소·키·버킷 권한·TLS 설정을 확인하세요")
		return
	}
	cleanupCtx, cleanupCancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
	defer cleanupCancel()
	defer s.deleteStoredObject(cleanupCtx, object)
	file, e := s.materializeObject(ctx, object, 1024)
	if e != nil {
		apiError(w, 502, "저장소 읽기 또는 체크섬 진단에 실패했습니다")
		return
	}
	file.Close()
	os.Remove(file.Name())
	if e = s.deleteStoredObject(ctx, object); e != nil {
		apiError(w, 502, "저장소 삭제 진단에 실패했습니다. 진단 UUID 객체는 보존됩니다")
		return
	}
	s.audit(r, "STORAGE_TEST", p.ID, map[string]bool{"success": true})
	jsonResponse(w, 200, map[string]any{"ok": true, "write": true, "read": true, "checksum": true, "delete": true})
}
func (s *Server) cleanupStoredObject(ctx context.Context, o storedObject) error {
	cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	defer cancel()
	if o.Path == "" && o.ProviderID == "" {
		return nil
	}
	return s.deleteStoredObject(cleanup, o)
}
