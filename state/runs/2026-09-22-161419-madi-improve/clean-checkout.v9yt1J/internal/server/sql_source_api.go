package server

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"slices"
	"strings"
	"time"
)

//go:embed sql_source_schema.sql
var sqlSourceSchema string

func (s *Server) migrateSQLSources(ctx context.Context) error {
	_, e := s.DB.Exec(ctx, sqlSourceSchema)
	if e != nil {
		return e
	}
	return s.migrateSQLSourceAI(ctx)
}
func (s *Server) loadSQLSource(ctx context.Context, id string) (sqlSource, error) {
	c := sqlSource{}
	var raw []byte
	var cipher string
	e := s.DB.QueryRow(ctx, `SELECT id::text,workspace_id::text,coalesce(space_id::text,''),owner_id::text,service_account_id::text,name,kind,config,credentials_ciphertext,enabled,revision FROM sql_sources WHERE id=$1`, id).Scan(&c.ID, &c.WorkspaceID, &c.SpaceID, &c.OwnerID, &c.ServiceID, &c.Name, &c.Kind, &raw, &cipher, &c.Enabled, &c.Revision)
	if e != nil {
		return c, e
	}
	if json.Unmarshal(raw, &c.Config) != nil {
		return c, errors.New("데이터 소스 설정을 읽을 수 없습니다")
	}
	secret, e := s.decrypt(cipher)
	if e != nil || json.Unmarshal([]byte(secret), &c.Credentials) != nil {
		return c, errors.New("데이터 소스 자격 증명을 해독하지 못했습니다")
	}
	return c, nil
}
func (s *Server) sqlSourceAccess(r *http.Request, c sqlSource, manage bool) bool {
	p := current(r)
	if manage {
		return p.TokenID == "" && !p.ScopeRestricted && s.automationManager(r.Context(), p, c.WorkspaceID) && s.canSpace(r.Context(), p, c.WorkspaceID, c.SpaceID, true)
	}
	return hasIntegrationScope(p, "database:read") && s.canSpace(r.Context(), p, c.WorkspaceID, c.SpaceID, false)
}
func sqlSourcePublic(c sqlSource, manage bool) map[string]any {
	v := map[string]any{"id": c.ID, "workspace_id": c.WorkspaceID, "space_id": c.SpaceID, "name": c.Name, "kind": c.Kind, "enabled": c.Enabled, "ai_enabled": boolean(c.Config, "allow_ai"), "revision": c.Revision, "can_manage": manage}
	if manage {
		v["config"] = c.Config
		v["service_account_id"] = c.ServiceID
		v["credentials"] = map[string]bool{"username_configured": str(c.Credentials, "username") != "", "password_configured": str(c.Credentials, "password") != ""}
	}
	return v
}
func (s *Server) registerSQLSources() {
	s.registerSQLSourceAI()
	s.handle("GET /api/v1/data-sources", s.listSQLSources)
	s.handle("POST /api/v1/data-sources", s.saveSQLSource)
	s.handle("PUT /api/v1/data-sources/{id}", s.saveSQLSource)
	s.handle("GET /api/v1/data-sources/{id}", s.getSQLSource)
	s.handle("POST /api/v1/data-sources/{id}/inspect", s.inspectSQLSource)
	s.handle("PUT /api/v1/data-sources/{id}/table", s.annotateSQLSourceTable)
	s.handle("POST /api/v1/data-sources/{id}/queries", s.saveSQLSourceQuery)
	s.handle("PUT /api/v1/data-sources/{id}/queries/{query}", s.saveSQLSourceQuery)
	s.handle("POST /api/v1/data-sources/{id}/queries/{query}/execute", s.executeSQLSourceQuery)
}
func validateSQLSource(c sqlSource) error {
	if c.Name == "" || len(c.Name) > 120 || !oneOf(c.Kind, "postgres", "mysql", "mariadb", "mssql", "oracle") {
		return errors.New("데이터 소스 이름과 유형을 확인하세요")
	}
	for k, v := range c.Config {
		switch k {
		case "host", "database", "ca_pem":
			value, ok := v.(string)
			if !ok || len(value) > 65536 || strings.ContainsAny(value, "\x00\r") {
				return errors.New("접속 설정 형식이 올바르지 않습니다")
			}
		case "port", "timeout_seconds", "max_rows":
			n, ok := v.(float64)
			if !ok || n != float64(int(n)) {
				return errors.New("연결 숫자 설정은 정수여야 합니다")
			}
		case "allow_plaintext", "insecure_tls", "acknowledge_readonly", "acknowledge_acl", "allow_ai":
			if _, ok := v.(bool); !ok {
				return errors.New("연결 보안 설정은 참/거짓이어야 합니다")
			}
		case "tables":
			if _, ok := v.([]any); !ok {
				return errors.New("허용 테이블은 문자열 배열이어야 합니다")
			}
		default:
			return errors.New("지원하지 않는 데이터 소스 설정: " + k)
		}
	}
	host := str(c.Config, "host")
	if len(host) > 253 || (!connectorHostPattern.MatchString(host) && net.ParseIP(host) == nil) || number(c.Config, "port", 0) < 1 || number(c.Config, "port", 0) > 65535 || str(c.Config, "database") == "" || len(str(c.Config, "database")) > 128 {
		return errors.New("호스트·포트·데이터베이스 또는 Oracle 서비스 이름을 확인하세요")
	}
	if !boolean(c.Config, "acknowledge_readonly") || !boolean(c.Config, "acknowledge_acl") {
		return errors.New("원격 읽기 전용 계정과 대상 공간 ACL 적용을 확인해야 합니다")
	}
	if n := number(c.Config, "timeout_seconds", 15); n < 1 || n > 60 {
		return errors.New("조회 시간 제한은 1~60초입니다")
	}
	if n := number(c.Config, "max_rows", 500); n < 1 || n > 2000 {
		return errors.New("최대 결과 행은 1~2000입니다")
	}
	tables := listStrings(c.Config["tables"])
	if len(tables) == 0 || len(tables) > 100 {
		return errors.New("허용 테이블을 1~100개 지정하세요")
	}
	seen := map[string]bool{}
	for _, name := range tables {
		parts := strings.Split(name, ".")
		if len(parts) != 2 || !sqlSourceIdentifier.MatchString(parts[0]) || !sqlSourceIdentifier.MatchString(parts[1]) || seen[name] {
			return errors.New("허용 테이블은 중복 없는 schema.table 이름이어야 합니다")
		}
		seen[name] = true
	}
	for key, v := range c.Credentials {
		if !oneOf(key, "username", "password") {
			return errors.New("허용되지 않은 자격 증명입니다")
		}
		value, ok := v.(string)
		if !ok || len(value) > 8192 || strings.ContainsAny(value, "\x00\r\n") {
			return errors.New("자격 증명 형식을 확인하세요")
		}
	}
	if str(c.Credentials, "username") == "" {
		return errors.New("읽기 전용 원격 계정 이름이 필요합니다")
	}
	return nil
}
func (s *Server) saveSQLSource(w http.ResponseWriter, r *http.Request) {
	var in map[string]any
	if decode(r, &in) != nil {
		apiError(w, 400, "데이터 소스 입력을 확인하세요")
		return
	}
	c := sqlSource{ID: r.PathValue("id"), WorkspaceID: str(in, "workspace_id"), SpaceID: str(in, "space_id"), OwnerID: current(r).ID, ServiceID: str(in, "service_account_id"), Name: strings.TrimSpace(str(in, "name")), Kind: str(in, "kind"), Enabled: boolean(in, "enabled")}
	c.Config, _ = in["config"].(map[string]any)
	c.Credentials, _ = in["credentials"].(map[string]any)
	if c.Credentials == nil {
		c.Credentials = map[string]any{}
	}
	if c.ID != "" {
		old, e := s.loadSQLSource(r.Context(), c.ID)
		if e != nil || !s.sqlSourceAccess(r, old, true) {
			apiError(w, 404, "관리할 데이터 소스가 없습니다")
			return
		}
		if c.WorkspaceID != old.WorkspaceID || c.SpaceID != old.SpaceID || c.ServiceID != old.ServiceID || c.Kind != old.Kind {
			apiError(w, 409, "데이터 소스 유형·공간·서비스 계정은 생성 후 고정됩니다")
			return
		}
		if number(in, "revision", 0) != old.Revision {
			apiError(w, 409, "데이터 소스 설정이 변경되었습니다. 다시 불러오세요")
			return
		}
		c.Revision = old.Revision
		for _, key := range []string{"username", "password"} {
			if str(c.Credentials, key) == "" {
				c.Credentials[key] = str(old.Credentials, key)
			}
		}
	}
	if !s.sqlSourceAccess(r, c, true) {
		apiError(w, 403, "데이터 소스 관리 권한이 없습니다")
		return
	}
	if e := validateSQLSource(c); e != nil {
		apiError(w, 400, e.Error())
		return
	}
	service, e := s.workerPrincipal(r.Context(), c.ServiceID, "", c.WorkspaceID)
	if e != nil || service.Kind != "service" || !s.canSpace(r.Context(), service, c.WorkspaceID, c.SpaceID, false) {
		apiError(w, 400, "대상 공간에 접근할 전용 서비스 계정이 필요합니다")
		return
	}
	if c.Enabled {
		db, e := s.openSQLSource(r.Context(), c)
		if e != nil {
			apiError(w, 400, e.Error())
			return
		}
		_ = db.Close()
	}
	cipher, e := s.encrypt(string(jsonValue(c.Credentials)))
	if e != nil {
		respond(w, nil, e)
		return
	}
	if c.ID == "" {
		c.ID = newID()
		_, e = s.DB.Exec(r.Context(), `INSERT INTO sql_sources(id,workspace_id,space_id,owner_id,service_account_id,name,kind,config,credentials_ciphertext,enabled) VALUES($1,$2,NULLIF($3,'')::uuid,$4,$5,$6,$7,$8,$9,$10)`, c.ID, c.WorkspaceID, c.SpaceID, c.OwnerID, c.ServiceID, c.Name, c.Kind, jsonValue(c.Config), cipher, c.Enabled)
	} else {
		tag, err := s.DB.Exec(r.Context(), `UPDATE sql_sources SET name=$2,config=$3,credentials_ciphertext=$4,enabled=$5,revision=revision+1,updated_at=now() WHERE id=$1 AND revision=$6`, c.ID, c.Name, jsonValue(c.Config), cipher, c.Enabled, c.Revision)
		e = err
		if e == nil && tag.RowsAffected() != 1 {
			apiError(w, 409, "다른 관리자가 설정을 변경했습니다")
			return
		}
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "DATA_SOURCE_CONFIGURE", c.ID, map[string]any{"kind": c.Kind, "enabled": c.Enabled})
	c, e = s.loadSQLSource(r.Context(), c.ID)
	respond(w, sqlSourcePublic(c, true), e)
}
func (s *Server) listSQLSources(w http.ResponseWriter, r *http.Request) {
	wid := r.URL.Query().Get("workspace_id")
	if !s.sqlSourceAccess(r, sqlSource{WorkspaceID: wid}, false) {
		apiError(w, 403, "데이터 소스 읽기 권한이 없습니다")
		return
	}
	rows, e := s.rows(r.Context(), "SELECT jsonb_build_object('id',id) FROM sql_sources WHERE workspace_id=$1 ORDER BY name", wid)
	if e != nil {
		respond(w, nil, e)
		return
	}
	out := []map[string]any{}
	for _, row := range rows {
		c, e := s.loadSQLSource(r.Context(), str(row, "id"))
		if e != nil {
			respond(w, nil, e)
			return
		}
		if s.sqlSourceAccess(r, c, false) {
			out = append(out, sqlSourcePublic(c, s.sqlSourceAccess(r, c, true)))
		}
	}
	respond(w, out, nil)
}
func (s *Server) requestedSQLSource(w http.ResponseWriter, r *http.Request, manage bool) (sqlSource, bool) {
	c, e := s.loadSQLSource(r.Context(), r.PathValue("id"))
	if e != nil || !s.sqlSourceAccess(r, c, manage) {
		apiError(w, 404, "접근할 데이터 소스가 없습니다")
		return c, false
	}
	return c, true
}
func (s *Server) getSQLSource(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requestedSQLSource(w, r, false)
	if !ok {
		return
	}
	tables, e := s.rows(r.Context(), "SELECT to_jsonb(t) FROM sql_source_tables t WHERE source_id=$1 ORDER BY schema_name,table_name", c.ID)
	if e != nil {
		respond(w, nil, e)
		return
	}
	filtered := []map[string]any{}
	for _, table := range tables {
		if !sqlSourceTableAllowed(c, str(table, "schema_name"), str(table, "table_name")) {
			continue
		}
		ids := []string{}
		for _, id := range listStrings(table["document_ids"]) {
			if s.canDocument(r.Context(), current(r), id, false) {
				ids = append(ids, id)
			}
		}
		table["document_ids"] = ids
		filtered = append(filtered, table)
	}
	queries, e := s.rows(r.Context(), "SELECT to_jsonb(q) FROM sql_source_queries q WHERE source_id=$1 ORDER BY name", c.ID)
	if e != nil {
		respond(w, nil, e)
		return
	}
	visibleQueries := []map[string]any{}
	for _, q := range queries {
		p, e := sqlSourcePlan(q["plan"])
		if e == nil && sqlSourceTableAllowed(c, p.Schema, p.Table) {
			visibleQueries = append(visibleQueries, q)
		}
	}
	runs := []map[string]any{}
	if s.sqlSourceAccess(r, c, true) {
		runs, e = s.rows(r.Context(), "SELECT to_jsonb(x) FROM sql_source_runs x WHERE source_id=$1 ORDER BY created_at DESC LIMIT 100", c.ID)
	}
	respond(w, map[string]any{"source": sqlSourcePublic(c, s.sqlSourceAccess(r, c, true)), "tables": filtered, "queries": visibleQueries, "runs": runs}, e)
}
func (s *Server) inspectSQLSource(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requestedSQLSource(w, r, true)
	if !ok {
		return
	}
	service, e := s.workerPrincipal(r.Context(), c.ServiceID, "", c.WorkspaceID)
	if e != nil || service.Kind != "service" || !s.canSpace(r.Context(), service, c.WorkspaceID, c.SpaceID, false) {
		apiError(w, 403, "서비스 계정의 현재 공간 읽기 권한이 없습니다")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(number(c.Config, "timeout_seconds", 15))*time.Second)
	defer cancel()
	db, e := s.openSQLSource(ctx, c)
	if e != nil {
		apiError(w, 400, e.Error())
		return
	}
	defer db.Close()
	remote, e := beginSQLSource(ctx, db, c.Kind)
	if e != nil {
		apiError(w, 400, e.Error())
		return
	}
	defer remote.Rollback()
	items := []map[string]any{}
	for _, table := range listStrings(c.Config["tables"]) {
		parts := strings.Split(table, ".")
		columns, e := inspectSQLTable(ctx, remote, c, parts[0], parts[1])
		if e != nil {
			apiError(w, 400, e.Error())
			return
		}
		items = append(items, map[string]any{"schema_name": parts[0], "table_name": parts[1], "columns": columns})
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(ctx)
	var rev int
	if e = tx.QueryRow(ctx, "SELECT revision FROM sql_sources WHERE id=$1 FOR SHARE", c.ID).Scan(&rev); e != nil || rev != c.Revision {
		apiError(w, 409, "검사 중 데이터 소스 설정이 변경되었습니다")
		return
	}
	for _, item := range items {
		_, e = tx.Exec(ctx, `INSERT INTO sql_source_tables(source_id,schema_name,table_name,columns) VALUES($1,$2,$3,$4) ON CONFLICT(source_id,schema_name,table_name) DO UPDATE SET columns=excluded.columns,inspected_at=now()`, c.ID, item["schema_name"], item["table_name"], jsonValue(item["columns"]))
		if e != nil {
			respond(w, nil, e)
			return
		}
	}
	e = tx.Commit(ctx)
	if e == nil {
		s.audit(r, "DATA_SOURCE_INSPECT", c.ID, map[string]any{"tables": len(items)})
	}
	respond(w, map[string]any{"tables": items, "readonly_verified": true}, e)
}
func (s *Server) sourceColumns(ctx context.Context, c sqlSource, p SQLSourcePlan) ([]sqlColumn, error) {
	if !sqlSourceTableAllowed(c, p.Schema, p.Table) {
		return nil, errors.New("관리자 허용 테이블이 아닙니다")
	}
	var raw []byte
	if e := s.DB.QueryRow(ctx, "SELECT columns FROM sql_source_tables WHERE source_id=$1 AND schema_name=$2 AND table_name=$3", c.ID, p.Schema, p.Table).Scan(&raw); e != nil {
		return nil, errors.New("먼저 연결 진단과 메타데이터 검사를 실행하세요")
	}
	var cols []sqlColumn
	e := json.Unmarshal(raw, &cols)
	return cols, e
}
func (s *Server) saveSQLSourceQuery(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requestedSQLSource(w, r, true)
	if !ok {
		return
	}
	var in map[string]any
	if decode(r, &in) != nil || str(in, "name") == "" || len(str(in, "name")) > 120 {
		apiError(w, 400, "저장 쿼리 이름을 확인하세요")
		return
	}
	plan, e := sqlSourcePlan(in["plan"])
	if e != nil {
		apiError(w, 400, e.Error())
		return
	}
	cols, e := s.sourceColumns(r.Context(), c, plan)
	if e == nil {
		_, _, e = buildSQLSourceQuery(c.Kind, plan, cols, number(c.Config, "max_rows", 500))
	}
	if e != nil {
		apiError(w, 400, e.Error())
		return
	}
	id := r.PathValue("query")
	if id == "" {
		id = newID()
		_, e = s.DB.Exec(r.Context(), `INSERT INTO sql_source_queries(id,source_id,owner_id,name,plan,enabled) VALUES($1,$2,$3,$4,$5,$6)`, id, c.ID, current(r).ID, str(in, "name"), jsonValue(plan), boolean(in, "enabled"))
	} else {
		tag, err := s.DB.Exec(r.Context(), `UPDATE sql_source_queries SET name=$3,plan=$4,enabled=$5,updated_at=now() WHERE id=$1 AND source_id=$2 AND origin_proposal_id IS NULL`, id, c.ID, str(in, "name"), jsonValue(plan), boolean(in, "enabled"))
		e = err
		if e == nil && tag.RowsAffected() != 1 {
			apiError(w, 409, "쿼리를 수정할 수 없습니다. AI 확인·승인 계획은 새 제안으로 변경하세요")
			return
		}
	}
	if e == nil {
		s.audit(r, "DATA_SOURCE_QUERY_CONFIGURE", id, map[string]any{"source_id": c.ID})
	}
	respond(w, map[string]any{"id": id}, e)
}
func (s *Server) executeSQLSourceQuery(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requestedSQLSource(w, r, false)
	if !ok {
		return
	}
	if !c.Enabled {
		apiError(w, 403, "데이터 소스가 중지되었습니다")
		return
	}
	if e := s.validateSQLProposalQuery(r, c.ID, r.PathValue("query")); e != nil {
		sqlProposalError(w, e)
		return
	}
	service, e := s.workerPrincipal(r.Context(), c.ServiceID, "", c.WorkspaceID)
	if e != nil || service.Kind != "service" || !s.canSpace(r.Context(), service, c.WorkspaceID, c.SpaceID, false) {
		apiError(w, 403, "서비스 계정의 현재 공간 읽기 권한이 없습니다")
		return
	}
	var raw []byte
	e = s.DB.QueryRow(r.Context(), "SELECT plan FROM sql_source_queries WHERE id=$1 AND source_id=$2 AND enabled", r.PathValue("query"), c.ID).Scan(&raw)
	if e != nil {
		apiError(w, 404, "활성화된 저장 쿼리를 찾을 수 없습니다")
		return
	}
	var p SQLSourcePlan
	if json.Unmarshal(raw, &p) != nil {
		apiError(w, 400, "저장 쿼리 형식이 잘못되었습니다")
		return
	}
	cols, e := s.sourceColumns(r.Context(), c, p)
	if e != nil {
		apiError(w, 400, e.Error())
		return
	}
	query, args, e := buildSQLSourceQuery(c.Kind, p, cols, number(c.Config, "max_rows", 500))
	if e != nil {
		apiError(w, 400, e.Error())
		return
	}
	started := time.Now()
	status := "failed"
	message := "외부 조회를 완료하지 못했습니다"
	count := 0
	defer func() {
		auditCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_, _ = s.DB.Exec(auditCtx, "INSERT INTO sql_source_runs(id,source_id,query_id,user_id,status,row_count,duration_ms,message) VALUES($1,$2,$3,$4,$5,$6,$7,$8)", newID(), c.ID, r.PathValue("query"), current(r).ID, status, count, time.Since(started).Milliseconds(), message)
		s.audit(r, "DATA_SOURCE_QUERY", r.PathValue("query"), map[string]any{"source_id": c.ID, "status": status, "rows": count})
	}()
	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(number(c.Config, "timeout_seconds", 15))*time.Second)
	defer cancel()
	db, e := s.openSQLSource(ctx, c)
	if e != nil {
		apiError(w, 400, e.Error())
		return
	}
	defer db.Close()
	tx, e := beginSQLSource(ctx, db, c.Kind)
	if e != nil {
		apiError(w, 400, e.Error())
		return
	}
	defer tx.Rollback()
	if e = checkSQLSourcePrivileges(ctx, tx, c, p.Schema, p.Table); e != nil {
		apiError(w, 403, e.Error())
		return
	}
	rows, truncated, e := sqlSourceRows(ctx, tx, query, args, p.Limit)
	if e != nil {
		apiError(w, 400, e.Error())
		return
	}
	// A slow remote query cannot return data after the local grant/source is revoked.
	if e := s.validateSQLProposalQuery(r, c.ID, r.PathValue("query")); e != nil {
		sqlProposalError(w, e)
		return
	}
	fresh, e := s.loadSQLSource(ctx, c.ID)
	if e != nil || fresh.Revision != c.Revision || !fresh.Enabled || !s.sqlSourceAccess(r, fresh, false) {
		apiError(w, 403, "조회 중 연결 설정 또는 권한이 변경되어 결과를 반환하지 않습니다")
		return
	}
	_, e = s.workerPrincipal(ctx, current(r).ID, current(r).TokenID, c.WorkspaceID)
	if e != nil {
		apiError(w, 403, "조회 중 사용자 또는 API 키 권한이 회수되었습니다")
		return
	}
	if current(r).PluginID != "" {
		_, caps, e := s.pluginGrant(r, current(r).PluginID, c.WorkspaceID)
		if e != nil || !slices.Contains(caps, "database:read") {
			apiError(w, 403, "조회 중 플러그인 권한이 회수되었습니다")
			return
		}
	}
	var queryCurrent bool
	if s.DB.QueryRow(ctx, "SELECT enabled AND plan=$3::jsonb FROM sql_source_queries WHERE id=$1 AND source_id=$2", r.PathValue("query"), c.ID, raw).Scan(&queryCurrent) != nil || !queryCurrent {
		apiError(w, 403, "조회 중 저장 쿼리가 중지되거나 변경되었습니다")
		return
	}
	service, e = s.workerPrincipal(ctx, c.ServiceID, "", c.WorkspaceID)
	if e != nil || !s.canSpace(ctx, service, c.WorkspaceID, c.SpaceID, false) {
		apiError(w, 403, "조회 중 서비스 계정 권한이 회수되었습니다")
		return
	}
	status = "succeeded"
	message = ""
	count = len(rows)
	respond(w, map[string]any{"columns": p.Columns, "rows": rows, "truncated": truncated, "duration_ms": time.Since(started).Milliseconds(), "sql": query}, nil)
}
func (s *Server) annotateSQLSourceTable(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requestedSQLSource(w, r, true)
	if !ok {
		return
	}
	var in map[string]any
	if decode(r, &in) != nil || !sqlSourceTableAllowed(c, str(in, "schema_name"), str(in, "table_name")) || len(str(in, "description")) > 20000 {
		apiError(w, 400, "테이블 설명을 확인하세요")
		return
	}
	ids := listStrings(in["document_ids"])
	if len(ids) > 100 {
		apiError(w, 400, "관련 문서는 최대 100개입니다")
		return
	}
	for _, id := range ids {
		if !s.canDocument(r.Context(), current(r), id, false) {
			apiError(w, 403, "관련 문서에 접근할 수 없습니다")
			return
		}
	}
	tag, e := s.DB.Exec(r.Context(), "UPDATE sql_source_tables SET description=$4,document_ids=$5 WHERE source_id=$1 AND schema_name=$2 AND table_name=$3", c.ID, str(in, "schema_name"), str(in, "table_name"), str(in, "description"), jsonValue(ids))
	if e == nil && tag.RowsAffected() != 1 {
		apiError(w, 404, "검사된 테이블이 없습니다")
		return
	}
	respond(w, map[string]bool{"ok": e == nil}, e)
}
