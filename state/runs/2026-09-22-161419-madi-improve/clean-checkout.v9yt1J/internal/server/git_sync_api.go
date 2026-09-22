package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
)

var gitSyncSecretFields = []string{"username", "password", "private_key", "private_key_password"}

func (s *Server) registerGitSync() {
	s.admin("GET /api/v1/admin/git-sync/settings", func(w http.ResponseWriter, r *http.Request) {
		value, revision, e := s.gitSyncSettings(r.Context())
		respond(w, map[string]any{"settings": value, "revision": revision}, e)
	})
	s.admin("PUT /api/v1/admin/git-sync/settings", s.saveGitSyncSettings)
	s.handle("GET /api/v1/git-sync/connections", s.listGitSyncConnections)
	s.admin("POST /api/v1/git-sync/connections", s.saveGitSyncConnection)
	s.admin("PUT /api/v1/git-sync/connections/{id}", s.saveGitSyncConnection)
	s.handle("POST /api/v1/git-sync/connections/{id}/test", s.testGitSyncConnection)
	s.handle("POST /api/v1/git-sync/connections/{id}/preview", s.queueGitSyncPreview)
	s.handle("GET /api/v1/git-sync/connections/{id}/runs", s.listGitSyncRuns)
	s.handle("GET /api/v1/git-sync/runs/{run}", s.getGitSyncRun)
	s.handle("POST /api/v1/git-sync/runs/{run}/confirm", s.confirmGitSync)
	s.handle("POST /api/v1/git-sync/runs/{run}/cancel", s.cancelGitSync)
	s.handle("POST /api/v1/git-sync/runs/{run}/inspect", s.inspectGitSync)
	s.RegisterJobHandler("git-sync.preview", s.executeGitSyncJob)
	s.RegisterJobHandler("git-sync.execute", s.executeGitSyncJob)
}

func (s *Server) testGitSyncConnection(w http.ResponseWriter, r *http.Request) {
	c, e := s.loadGitSyncConnection(r.Context(), r.PathValue("id"))
	if e != nil || !s.gitSyncManager(r, c) {
		apiError(w, 404, "진단할 Git 연결이 없습니다")
		return
	}
	policy, _, e := s.gitSyncSettings(r.Context())
	if e != nil || !boolean(policy, "enabled") || !c.Enabled {
		apiError(w, 400, "관리자 Git 정책과 저장된 연결을 먼저 활성화하세요")
		return
	}
	config, e := s.gitSyncDecryptedConfig(c)
	if e != nil {
		apiError(w, 400, e.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	remote, e := newGitSyncRemote(ctx, config, listStrings(policy["allowed_hosts"]), boolean(policy, "allow_private_networks"))
	if e != nil {
		apiError(w, 400, e.Error())
		return
	}
	defer remote.Close()
	session, e := remote.Transport.NewUploadPackSession(remote.Endpoint, remote.Auth)
	if e != nil {
		apiError(w, 400, "Git 읽기 연결에 실패했습니다. TLS·SSH 키·인증을 확인하세요")
		return
	}
	defer session.Close()
	refs, e := session.AdvertisedReferencesContext(ctx)
	if errors.Is(e, transport.ErrEmptyRemoteRepository) {
		jsonResponse(w, 200, map[string]any{"ok": true, "empty": true, "message": "빈 저장소의 읽기 연결을 확인했습니다. 쓰기 권한은 아직 검증하지 않았습니다"})
		return
	}
	if e != nil || refs == nil || len(refs.References) > 10000 {
		apiError(w, 400, "Git 브랜치 목록이 올바르지 않거나 한도를 초과했습니다")
		return
	}
	head := refs.References[plumbing.NewBranchReferenceName(str(config, "branch")).String()]
	s.audit(r, "GIT_SYNC_TEST", c.ID, nil)
	jsonResponse(w, 200, map[string]any{"ok": true, "commit": head.String(), "empty": head.IsZero(), "message": "저장된 대상의 브랜치 읽기 연결만 확인했습니다. 쓰기 권한은 아직 검증하지 않았습니다"})
}

func (s *Server) gitSyncSettings(ctx context.Context) (map[string]any, int64, error) {
	var raw []byte
	var revision int64
	e := s.DB.QueryRow(ctx, "SELECT data,revision FROM git_sync_settings WHERE id=1").Scan(&raw, &revision)
	if e != nil {
		return nil, 0, e
	}
	value := defaultGitSyncSettings()
	var stored map[string]any
	if json.Unmarshal(raw, &stored) != nil {
		return nil, 0, errors.New("Git 정책을 읽을 수 없습니다")
	}
	for k, v := range stored {
		value[k] = v
	}
	return value, revision, nil
}

func (s *Server) saveGitSyncSettings(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Revision int64          `json:"revision"`
		Settings map[string]any `json:"settings"`
	}
	if decode(r, &in) != nil {
		apiError(w, 400, "Git 정책 입력을 확인하세요")
		return
	}
	value := defaultGitSyncSettings()
	for key, v := range in.Settings {
		if _, ok := value[key]; !ok {
			apiError(w, 400, "지원하지 않는 Git 정책 항목입니다")
			return
		}
		value[key] = v
	}
	for _, key := range []string{"enabled", "allow_private_networks"} {
		if _, ok := value[key].(bool); !ok {
			apiError(w, 400, "Git 정책은 참/거짓 값이어야 합니다")
			return
		}
	}
	hosts, e := notificationHostList(value["allowed_hosts"])
	if e != nil || len(hosts) > 100 || (boolean(value, "enabled") && len(hosts) == 0) {
		apiError(w, 400, "Git 사용 전에 정확한 허용 호스트를 1~100개 지정하세요")
		return
	}
	value["allowed_hosts"] = hosts
	max, ok := value["max_snapshot_mb"].(float64)
	if !ok {
		if i, yes := value["max_snapshot_mb"].(int); yes {
			max = float64(i)
			ok = true
		}
	}
	if !ok || max < 1 || max > 100 || float64(int(max)) != max {
		apiError(w, 400, "Git 미리보기 한도는 1~100MB 정수입니다")
		return
	}
	tag, e := s.DB.Exec(r.Context(), "UPDATE git_sync_settings SET data=$1,revision=revision+1,updated_at=now(),updated_by=$2 WHERE id=1 AND revision=$3", jsonValue(value), current(r).ID, in.Revision)
	if e != nil {
		respond(w, nil, e)
		return
	}
	if tag.RowsAffected() != 1 {
		apiError(w, 409, "Git 정책이 변경되었습니다. 다시 불러오세요")
		return
	}
	s.audit(r, "GIT_SYNC_POLICY", "git-sync", map[string]any{"enabled": value["enabled"]})
	jsonResponse(w, 200, map[string]any{"settings": value, "revision": in.Revision + 1})
}

func (s *Server) loadGitSyncConnection(ctx context.Context, id string) (gitSyncConnection, error) {
	var c gitSyncConnection
	var raw []byte
	e := s.DB.QueryRow(ctx, "SELECT id::text,workspace_id::text,owner_id::text,coalesce(space_id::text,''),name,config,enabled,revision,last_remote_commit FROM git_sync_connections WHERE id=$1", id).Scan(&c.ID, &c.WorkspaceID, &c.OwnerID, &c.SpaceID, &c.Name, &raw, &c.Enabled, &c.Revision, &c.LastRemoteCommit)
	if e != nil {
		return c, e
	}
	if json.Unmarshal(raw, &c.Config) != nil {
		return c, errors.New("Git 연결 설정을 읽을 수 없습니다")
	}
	return c, nil
}
func gitSyncPublicConnection(c gitSyncConnection) map[string]any {
	config := map[string]any{}
	for k, v := range c.Config {
		config[k] = v
	}
	for _, key := range gitSyncSecretFields {
		config[key+"_configured"] = str(config, key) != ""
		delete(config, key)
	}
	return map[string]any{"id": c.ID, "workspace_id": c.WorkspaceID, "owner_id": c.OwnerID, "space_id": c.SpaceID, "name": c.Name, "config": config, "enabled": c.Enabled, "revision": c.Revision, "last_remote_commit": c.LastRemoteCommit}
}
func (s *Server) gitSyncDecryptedConfig(c gitSyncConnection) (map[string]any, error) {
	config := map[string]any{}
	for k, v := range c.Config {
		config[k] = v
	}
	for _, key := range gitSyncSecretFields {
		if str(config, key) != "" {
			value, e := s.decrypt(str(config, key))
			if e != nil {
				return nil, errors.New("Git 인증 정보를 복호화하지 못했습니다")
			}
			config[key] = value
		}
	}
	return config, nil
}
func (s *Server) gitSyncManager(r *http.Request, c gitSyncConnection) bool {
	p := current(r)
	return p != nil && p.Kind == "user" && p.TokenID == "" && !p.ScopeRestricted && s.automationManager(r.Context(), p, c.WorkspaceID) && s.canSpace(r.Context(), p, c.WorkspaceID, c.SpaceID, false)
}
func (s *Server) listGitSyncConnections(w http.ResponseWriter, r *http.Request) {
	wid := r.URL.Query().Get("workspace_id")
	if !s.gitSyncManager(r, gitSyncConnection{WorkspaceID: wid}) {
		apiError(w, 403, "워크스페이스 Git 관리 권한이 필요합니다")
		return
	}
	rows, e := s.DB.Query(r.Context(), "SELECT id::text FROM git_sync_connections WHERE workspace_id=$1 ORDER BY created_at", wid)
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
	if e == nil {
		e = rows.Err()
	}
	rows.Close()
	if e != nil {
		respond(w, nil, e)
		return
	}
	result := []any{}
	for _, id := range ids {
		c, e := s.loadGitSyncConnection(r.Context(), id)
		if e != nil {
			respond(w, nil, e)
			return
		}
		if s.gitSyncManager(r, c) {
			result = append(result, gitSyncPublicConnection(c))
		}
	}
	jsonResponse(w, 200, result)
}
func (s *Server) saveGitSyncConnection(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name        string         `json:"name"`
		WorkspaceID string         `json:"workspace_id"`
		OwnerID     string         `json:"owner_id"`
		SpaceID     string         `json:"space_id"`
		Config      map[string]any `json:"config"`
		Enabled     bool           `json:"enabled"`
		Revision    int64          `json:"revision"`
	}
	if decode(r, &in) != nil || in.Name == "" || len(in.Name) > 120 || !validID(in.WorkspaceID) || !validID(in.OwnerID) {
		apiError(w, 400, "Git 연결 이름·워크스페이스·실행 계정을 확인하세요")
		return
	}
	for _, value := range in.Config {
		if _, ok := value.(string); !ok {
			apiError(w, 400, "Git 연결 설정은 문자열이어야 합니다")
			return
		}
	}
	c := gitSyncConnection{ID: r.PathValue("id"), WorkspaceID: in.WorkspaceID, OwnerID: in.OwnerID, SpaceID: in.SpaceID, Name: in.Name, Config: map[string]any{}, Enabled: in.Enabled}
	if c.ID != "" {
		old, e := s.loadGitSyncConnection(r.Context(), c.ID)
		if e != nil {
			apiError(w, 404, "Git 연결이 없습니다")
			return
		}
		if old.WorkspaceID != c.WorkspaceID || old.OwnerID != c.OwnerID || old.SpaceID != c.SpaceID {
			apiError(w, 409, "Git 대상 워크스페이스·공간·실행 계정 변경은 새 연결로 등록하세요")
			return
		}
		c.Config, e = s.gitSyncDecryptedConfig(old)
		if e != nil {
			apiError(w, 400, e.Error())
			return
		}
		for _, key := range []string{"url", "branch", "prefix"} {
			if value, ok := in.Config[key]; ok && value != old.Config[key] {
				apiError(w, 409, "Git 저장소·브랜치·전용 폴더 변경은 새 연결로 등록하세요")
				return
			}
		}
	}
	for key, v := range in.Config {
		if str(in.Config, key) == "" {
			secret := false
			for _, field := range gitSyncSecretFields {
				if field == key {
					secret = true
				}
			}
			if secret {
				continue
			}
		}
		c.Config[key] = v
	}
	if e := gitSyncValidateConfig(c.Config); e != nil {
		apiError(w, 400, e.Error())
		return
	}
	actor, e := s.workerPrincipal(r.Context(), c.OwnerID, "", c.WorkspaceID)
	if e != nil || !s.canWorkspace(r.Context(), actor, c.WorkspaceID, true) || !s.canSpace(r.Context(), actor, c.WorkspaceID, c.SpaceID, true) {
		apiError(w, 403, "실행 계정에 대상 워크스페이스·공간 작성 권한이 필요합니다")
		return
	}
	if !s.gitSyncManager(r, c) {
		apiError(w, 403, "현재 관리자가 대상 공간에 접근할 수 없습니다")
		return
	}
	for _, key := range gitSyncSecretFields {
		if value := str(c.Config, key); value != "" {
			encrypted, e := s.encrypt(value)
			if e != nil {
				respond(w, nil, e)
				return
			}
			c.Config[key] = encrypted
		}
	}
	if c.ID == "" {
		c.ID = newID()
		_, e = s.DB.Exec(r.Context(), "INSERT INTO git_sync_connections(id,workspace_id,owner_id,space_id,name,config,enabled) VALUES($1,$2,$3,NULLIF($4,'')::uuid,$5,$6,$7)", c.ID, c.WorkspaceID, c.OwnerID, c.SpaceID, c.Name, jsonValue(c.Config), c.Enabled)
	} else {
		tag, err := s.DB.Exec(r.Context(), "UPDATE git_sync_connections SET name=$2,config=$3,enabled=$4,revision=revision+1,updated_at=now() WHERE id=$1 AND revision=$5", c.ID, c.Name, jsonValue(c.Config), c.Enabled, in.Revision)
		e = err
		if e == nil && tag.RowsAffected() != 1 {
			apiError(w, 409, "Git 연결 설정이 변경되었습니다")
			return
		}
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	saved, e := s.loadGitSyncConnection(r.Context(), c.ID)
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "GIT_SYNC_CONNECTION", c.ID, map[string]any{"workspace_id": c.WorkspaceID, "enabled": c.Enabled})
	jsonResponse(w, 200, gitSyncPublicConnection(saved))
}
