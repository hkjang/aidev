package server

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"regexp"
	"strings"
)

//go:embed connector_schema.sql
var connectorSchema string

func (s *Server) migrateConnectors(ctx context.Context) error {
	_, e := s.DB.Exec(ctx, connectorSchema)
	return e
}
func (s *Server) loadConnector(ctx context.Context, id string) (connectorConfig, error) {
	c := connectorConfig{}
	var raw []byte
	var ciphertext string
	e := s.DB.QueryRow(ctx, `SELECT id::text,workspace_id::text,coalesce(space_id::text,''),owner_id::text,service_account_id::text,name,kind,base_url,credentials_ciphertext,config,enabled,revision,interval_minutes,cursor FROM connector_configs WHERE id=$1`, id).Scan(&c.ID, &c.WorkspaceID, &c.SpaceID, &c.OwnerID, &c.ServiceID, &c.Name, &c.Kind, &c.BaseURL, &ciphertext, &raw, &c.Enabled, &c.Revision, &c.Interval, &c.Cursor)
	if e != nil {
		return c, e
	}
	if json.Unmarshal(raw, &c.Config) != nil {
		return c, errors.New("연결 설정을 읽을 수 없습니다")
	}
	c.Credentials = map[string]any{}
	if ciphertext != "" {
		plain, e := s.decrypt(ciphertext)
		if e != nil || json.Unmarshal([]byte(plain), &c.Credentials) != nil {
			return c, errors.New("연결 자격 증명 복호화에 실패했습니다")
		}
	}
	return c, nil
}
func connectorPublic(c connectorConfig) map[string]any {
	flags := map[string]bool{}
	for _, key := range []string{"token", "username", "password"} {
		flags[key+"_configured"] = str(c.Credentials, key) != ""
	}
	return map[string]any{"id": c.ID, "workspace_id": c.WorkspaceID, "space_id": c.SpaceID, "owner_id": c.OwnerID, "service_account_id": c.ServiceID, "name": c.Name, "kind": c.Kind, "base_url": c.BaseURL, "config": c.Config, "credentials": flags, "enabled": c.Enabled, "revision": c.Revision, "interval_minutes": c.Interval, "has_checkpoint": c.Cursor != ""}
}
func (s *Server) connectorManage(r *http.Request, c connectorConfig) bool {
	return current(r).TokenID == "" && !current(r).ScopeRestricted && s.automationManager(r.Context(), current(r), c.WorkspaceID) && s.canSpace(r.Context(), current(r), c.WorkspaceID, c.SpaceID, true)
}
func (s *Server) registerConnectors() {
	s.admin("GET /api/v1/admin/connectors/settings", func(w http.ResponseWriter, r *http.Request) {
		v, e := s.one(r.Context(), "SELECT to_jsonb(c) FROM connector_settings c WHERE id=1")
		respond(w, v, e)
	})
	s.admin("PUT /api/v1/admin/connectors/settings", s.saveConnectorSettings)
	s.handle("GET /api/v1/connectors", s.listConnectors)
	s.handle("POST /api/v1/connectors", s.saveConnector)
	s.handle("PUT /api/v1/connectors/{id}", s.saveConnector)
	s.handle("GET /api/v1/connectors/options", s.connectorOptions)
	s.handle("GET /api/v1/connectors/{id}/records", s.connectorRecords)
	s.handle("GET /api/v1/connectors/{id}/runs", s.connectorRuns)
	s.handle("POST /api/v1/connectors/{id}/preview", s.queueConnector)
	s.handle("POST /api/v1/connectors/{id}/run", s.queueConnector)
	s.RegisterJobHandler("connector.preview", s.executeConnectorJob)
	s.RegisterJobHandler("connector.sync", s.executeConnectorJob)
}

var connectorHostPattern = regexp.MustCompile(`^[a-zA-Z0-9](?:[a-zA-Z0-9.-]*[a-zA-Z0-9])?$`)

func (s *Server) saveConnectorSettings(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Enabled      bool     `json:"enabled"`
		AllowedHosts []string `json:"allowed_hosts"`
	}
	if decode(r, &in) != nil || len(in.AllowedHosts) > 100 || (in.Enabled && len(in.AllowedHosts) == 0) {
		apiError(w, 400, "커넥터 활성화 전에 정확한 호스트를 1~100개 등록하세요")
		return
	}
	for i, h := range in.AllowedHosts {
		h = strings.ToLower(strings.TrimSpace(h))
		if len(h) > 253 || (!connectorHostPattern.MatchString(h) && net.ParseIP(h) == nil) {
			apiError(w, 400, "허용 호스트에는 스킴·경로·와일드카드 없이 호스트 이름 또는 IP를 입력하세요")
			return
		}
		in.AllowedHosts[i] = h
	}
	_, e := s.DB.Exec(r.Context(), "UPDATE connector_settings SET enabled=$1,allowed_hosts=$2 WHERE id=1", in.Enabled, jsonValue(in.AllowedHosts))
	if e == nil {
		s.audit(r, "CONNECTOR_POLICY", "connectors", in)
	}
	respond(w, in, e)
}
func validateConnector(c connectorConfig) error {
	if c.Name == "" || len(c.Name) > 120 || !oneOf(c.Kind, "github", "gitlab", "jira", "confluence", "drive", "sharepoint", "rest") || c.Interval < 0 || c.Interval > 525600 || (c.Interval > 0 && c.Interval < 5) {
		return errors.New("연결 이름·유형과 예약 간격(0 또는 5~525600분)을 확인하세요")
	}
	endpoint, e := connectorURL(c.BaseURL, "")
	if e != nil || endpoint.RawQuery != "" || len(c.BaseURL) > 2048 {
		return errors.New("API 기본 URL에는 쿼리·비밀·프래그먼트를 넣지 마세요")
	}
	if !boolean(c.Config, "acknowledge_acl") {
		return errors.New("원격 권한 대신 대상 공간 ACL이 적용됨을 확인해야 합니다")
	}
	for key, value := range c.Config {
		switch key {
		case "allow_http", "insecure_tls", "acknowledge_acl":
			if _, ok := value.(bool); !ok {
				return errors.New("연결 정책은 참/거짓 값이어야 합니다")
			}
		case "max_items":
			n, ok := value.(float64)
			if !ok || n < 100 || n > 10000 || n != float64(int(n)) {
				return errors.New("한 번에 처리할 항목은 100~10000개입니다")
			}
		case "ca_pem", "auth_mode", "conflict_policy", "project_id", "repository", "ref", "path_prefix", "jql", "variant", "remote_space", "query", "site_id", "list_path", "items_path", "id_path", "title_path", "content_path", "url_path", "next_path":
			text, ok := value.(string)
			if !ok || len(text) > 65536 || strings.ContainsAny(text, "\x00\r") {
				return errors.New("연결 세부 설정 형식을 확인하세요")
			}
		default:
			return errors.New("지원하지 않는 연결 설정: " + key)
		}
	}
	if endpoint.Scheme == "http" && !boolean(c.Config, "allow_http") {
		return errors.New("내부 HTTP 연결을 명시적으로 허용하세요")
	}
	if mode := str(c.Config, "auth_mode"); mode != "" && !oneOf(mode, "basic", "bearer", "private_token") {
		return errors.New("지원하지 않는 인증 방식입니다")
	}
	if policy := str(c.Config, "conflict_policy"); policy != "" && !oneOf(policy, "preserve_local", "replace_local") {
		return errors.New("충돌 처리 정책을 확인하세요")
	}
	required := map[string][]string{"github": {"repository"}, "gitlab": {"project_id"}, "jira": {"jql"}, "sharepoint": {"site_id"}, "rest": {"id_path", "title_path", "content_path"}}[c.Kind]
	for _, key := range required {
		if strings.TrimSpace(str(c.Config, key)) == "" {
			return errors.New("필수 연결 설정: " + key)
		}
	}
	for key, value := range c.Credentials {
		if !oneOf(key, "token", "username", "password") {
			return errors.New("지원하지 않는 자격 증명 필드입니다")
		}
		text, ok := value.(string)
		if !ok || len(text) > 8192 || strings.ContainsAny(text, "\r\n\x00") {
			return errors.New("연결 자격 증명 형식 또는 크기를 확인하세요")
		}
	}
	return nil
}
func (s *Server) saveConnector(w http.ResponseWriter, r *http.Request) {
	var in map[string]any
	if decode(r, &in) != nil {
		apiError(w, 400, "연결 정보를 확인하세요")
		return
	}
	c := connectorConfig{ID: r.PathValue("id"), WorkspaceID: str(in, "workspace_id"), SpaceID: str(in, "space_id"), OwnerID: current(r).ID, ServiceID: str(in, "service_account_id"), Name: strings.TrimSpace(str(in, "name")), Kind: str(in, "kind"), BaseURL: strings.TrimSuffix(str(in, "base_url"), "/"), Enabled: boolean(in, "enabled"), Interval: number(in, "interval_minutes", 0)}
	c.Config, _ = in["config"].(map[string]any)
	c.Credentials, _ = in["credentials"].(map[string]any)
	if c.Config == nil {
		c.Config = map[string]any{}
	}
	if c.Credentials == nil {
		c.Credentials = map[string]any{}
	}
	creating := c.ID == ""
	var old connectorConfig
	var e error
	if !creating {
		old, e = s.loadConnector(r.Context(), c.ID)
		if e != nil || !s.connectorManage(r, old) {
			apiError(w, 404, "관리할 연결을 찾을 수 없습니다")
			return
		}
		if c.WorkspaceID != old.WorkspaceID || c.SpaceID != old.SpaceID || c.ServiceID != old.ServiceID || c.Kind != old.Kind || c.BaseURL != old.BaseURL {
			apiError(w, 409, "연결 위치·서비스 계정·대상 공간은 변경하지 않습니다. 새 연결을 만드세요")
			return
		}
		if number(in, "revision", 0) != old.Revision {
			apiError(w, 409, "다른 관리자가 연결 설정을 변경했습니다. 새로 불러오세요")
			return
		}
		for _, key := range []string{"token", "username", "password"} {
			if str(c.Credentials, key) == "" {
				c.Credentials[key] = str(old.Credentials, key)
			}
		}
	}
	if !s.connectorManage(r, c) {
		apiError(w, 403, "대상 공간을 관리할 권한이 없습니다")
		return
	}
	principal, e := s.workerPrincipal(r.Context(), c.ServiceID, "", c.WorkspaceID)
	if e != nil || principal.Kind != "service" || !s.canSpace(r.Context(), principal, c.WorkspaceID, c.SpaceID, true) {
		apiError(w, 400, "대상 공간의 쓰기 권한을 가진 전용 서비스 계정이 필요합니다")
		return
	}
	if e = validateConnector(c); e != nil {
		apiError(w, 400, e.Error())
		return
	}
	if c.Enabled {
		if _, e = s.connectorHTTP(r.Context(), c); e != nil {
			apiError(w, 400, e.Error())
			return
		}
	}
	cipher, e := s.encrypt(string(jsonValue(c.Credentials)))
	if e != nil {
		respond(w, nil, e)
		return
	}
	if creating {
		c.ID = newID()
		_, e = s.DB.Exec(r.Context(), `INSERT INTO connector_configs(id,workspace_id,space_id,owner_id,service_account_id,name,kind,base_url,credentials_ciphertext,config,enabled,interval_minutes,next_run) VALUES($1,$2,NULLIF($3,'')::uuid,$4,$5,$6,$7,$8,$9,$10,$11,$12,CASE WHEN $11 AND $12>0 THEN now()+make_interval(mins=>$12) ELSE NULL END)`, c.ID, c.WorkspaceID, c.SpaceID, c.OwnerID, c.ServiceID, c.Name, c.Kind, c.BaseURL, cipher, jsonValue(c.Config), c.Enabled, c.Interval)
	} else {
		tag, err := s.DB.Exec(r.Context(), `UPDATE connector_configs SET name=$2,credentials_ciphertext=$3,config=$4,enabled=$5,interval_minutes=$6,next_run=CASE WHEN $5 AND $6>0 THEN now()+make_interval(mins=>$6) ELSE NULL END,revision=revision+1,cursor='',owner_id=$7,updated_at=now() WHERE id=$1 AND revision=$8`, c.ID, c.Name, cipher, jsonValue(c.Config), c.Enabled, c.Interval, current(r).ID, old.Revision)
		e = err
		if e == nil && tag.RowsAffected() != 1 {
			apiError(w, 409, "다른 관리자가 연결 설정을 변경했습니다")
			return
		}
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "CONNECTOR_CONFIGURE", c.ID, map[string]any{"kind": c.Kind, "enabled": c.Enabled, "workspace_id": c.WorkspaceID})
	c, e = s.loadConnector(r.Context(), c.ID)
	respond(w, connectorPublic(c), e)
}
func (s *Server) listConnectors(w http.ResponseWriter, r *http.Request) {
	wid := r.URL.Query().Get("workspace_id")
	if !s.connectorManage(r, connectorConfig{WorkspaceID: wid}) {
		apiError(w, 403, "워크스페이스 연결 관리자 권한이 필요합니다")
		return
	}
	rows, e := s.rows(r.Context(), "SELECT jsonb_build_object('id',id) FROM connector_configs WHERE workspace_id=$1 ORDER BY name", wid)
	if e != nil {
		respond(w, nil, e)
		return
	}
	out := []map[string]any{}
	for _, v := range rows {
		c, e := s.loadConnector(r.Context(), str(v, "id"))
		if e != nil {
			respond(w, nil, e)
			return
		}
		if s.connectorManage(r, c) {
			out = append(out, connectorPublic(c))
		}
	}
	respond(w, out, nil)
}
func (s *Server) connectorOptions(w http.ResponseWriter, r *http.Request) {
	wid := r.URL.Query().Get("workspace_id")
	if !s.connectorManage(r, connectorConfig{WorkspaceID: wid}) {
		apiError(w, 403, "연결 관리 권한이 없습니다")
		return
	}
	v, e := s.rows(r.Context(), "SELECT jsonb_build_object('id',u.id,'name',u.name,'email',u.email,'role',m.role) FROM users u JOIN workspace_members m ON m.user_id=u.id WHERE m.workspace_id=$1 AND u.kind='service' AND NOT u.disabled AND u.role<>'viewer' AND m.role IN ('owner','admin','editor') ORDER BY u.name", wid)
	respond(w, v, e)
}
func (s *Server) connectorRecords(w http.ResponseWriter, r *http.Request) {
	c, e := s.loadConnector(r.Context(), r.PathValue("id"))
	if e != nil || !s.connectorManage(r, c) {
		apiError(w, 404, "연결을 찾을 수 없습니다")
		return
	}
	v, e := s.rows(r.Context(), `SELECT to_jsonb(x)||jsonb_build_object('title',d.title) FROM connector_records x LEFT JOIN documents d ON d.id=x.document_id WHERE x.connector_id=$1 AND (x.document_id IS NULL OR madi_document_allowed($2,x.document_id,false)) ORDER BY last_seen_at DESC LIMIT 1000`, c.ID, current(r).ID)
	respond(w, v, e)
}
func (s *Server) connectorRuns(w http.ResponseWriter, r *http.Request) {
	c, e := s.loadConnector(r.Context(), r.PathValue("id"))
	if e != nil || !s.connectorManage(r, c) {
		apiError(w, 404, "연결을 찾을 수 없습니다")
		return
	}
	v, e := s.rows(r.Context(), `SELECT to_jsonb(x)-'cursor'||jsonb_build_object('status',j.status,'last_error',j.last_error,'kind',j.kind) FROM connector_runs x JOIN automation_jobs j ON j.id=x.job_id WHERE x.connector_id=$1 ORDER BY x.created_at DESC LIMIT 100`, c.ID)
	respond(w, v, e)
}
