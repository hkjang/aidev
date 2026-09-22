package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const scimUserSchema = "urn:ietf:params:scim:schemas:core:2.0:User"
const scimGroupSchema = "urn:ietf:params:scim:schemas:core:2.0:Group"
const scimErrorSchema = "urn:ietf:params:scim:api:messages:2.0:Error"
const scimListSchema = "urn:ietf:params:scim:api:messages:2.0:ListResponse"
const scimPatchSchema = "urn:ietf:params:scim:api:messages:2.0:PatchOp"

type scimError struct {
	status       int
	kind, detail string
}

func (e *scimError) Error() string                      { return e.detail }
func scimProblem(status int, kind, detail string) error { return &scimError{status, kind, detail} }
func scimJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/scim+json; charset=utf-8")
	w.WriteHeader(status)
	if status != 204 {
		_ = json.NewEncoder(w).Encode(value)
	}
}
func scimFailure(w http.ResponseWriter, e error) {
	status, kind, detail := 500, "", "SCIM 요청을 처리하지 못했습니다"
	var problem *scimError
	var pgError *pgconn.PgError
	if errors.As(e, &problem) {
		status, kind, detail = problem.status, problem.kind, problem.detail
	} else if errors.Is(e, pgx.ErrNoRows) {
		status, detail = 404, "SCIM 리소스를 찾을 수 없습니다"
	} else if errors.As(e, &pgError) && pgError.Code == "23505" {
		status, kind, detail = 409, "uniqueness", "동일한 사용자명·이메일·외부 ID 또는 그룹이 있습니다"
	}
	value := map[string]any{"schemas": []string{scimErrorSchema}, "status": strconv.Itoa(status), "detail": detail}
	if kind != "" {
		value["scimType"] = kind
	}
	scimJSON(w, status, value)
}

func (s *Server) registerSCIM() {
	register := func(pattern string, handler http.HandlerFunc) {
		s.apiRoutes = append(s.apiRoutes, pattern)
		s.mux.HandleFunc(pattern, s.scimAuth(handler))
	}
	for _, resource := range []string{"Users", "Groups"} {
		register("GET /api/v1/scim/v2/"+resource, s.scimList)
		register("POST /api/v1/scim/v2/"+resource, s.scimWrite)
		register("GET /api/v1/scim/v2/"+resource+"/{id}", s.scimGet)
		for _, method := range []string{"PUT", "PATCH", "DELETE"} {
			register(method+" /api/v1/scim/v2/"+resource+"/{id}", s.scimWrite)
		}
	}
	for _, resource := range []string{"ServiceProviderConfig", "ResourceTypes", "Schemas"} {
		register("GET /api/v1/scim/v2/"+resource, s.scimDiscovery)
		if resource != "ServiceProviderConfig" {
			register("GET /api/v1/scim/v2/"+resource+"/{id}", s.scimDiscovery)
		}
	}
}

func (s *Server) scimAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		p, e := s.tokenPrincipal(r)
		if e != nil {
			scimFailure(w, scimProblem(integrationAuthStatus(e), "", "만료되지 않은 SCIM 서비스 계정 키가 필요합니다"))
			return
		}
		if p.Kind != "service" || p.ScopeRestricted || !hasIntegrationScope(p, scimProvisionScope) {
			scimFailure(w, scimProblem(403, "", "SCIM 서비스 계정 권한이 필요합니다"))
			return
		}
		cfg, e := s.settings(r.Context())
		if e != nil {
			scimFailure(w, e)
			return
		}
		if !boolean(cfg, "scim_enabled") {
			scimFailure(w, scimProblem(403, "", "관리자가 SCIM을 활성화해야 합니다"))
			return
		}
		ctx, cancel := context.WithTimeout(context.WithValue(r.Context(), principalKey, p), 30*time.Second)
		defer cancel()
		next(w, r.WithContext(ctx))
	}
}

type scimQuerier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func scimKind(r *http.Request) string {
	if strings.Contains(r.URL.Path, "/Groups") {
		return "Groups"
	}
	return "Users"
}
func scimETag(version int64) string { return fmt.Sprintf(`W/"%d"`, version) }

func scimResource(ctx context.Context, q scimQuerier, kind, wid, id string) (map[string]any, error) {
	var name, external string
	var created, updated time.Time
	var version int64
	out := map[string]any{}
	if kind == "Users" {
		var raw []byte
		var active bool
		e := q.QueryRow(ctx, `SELECT username,external_id,profile,active,version,created_at,updated_at FROM scim_users WHERE workspace_id=$1 AND id=$2 AND deleted_at IS NULL`, wid, id).Scan(&name, &external, &raw, &active, &version, &created, &updated)
		if e != nil {
			return nil, e
		}
		if e = json.Unmarshal(raw, &out); e != nil {
			return nil, e
		}
		out["schemas"] = []string{scimUserSchema}
		out["userName"] = name
		out["active"] = active
		rows, e := q.Query(ctx, `SELECT g.id::text,g.display_name FROM scim_groups g JOIN scim_group_members m ON m.group_id=g.id WHERE m.workspace_id=$1 AND m.user_id=$2 ORDER BY g.display_name`, wid, id)
		if e != nil {
			return nil, e
		}
		groups := []any{}
		for rows.Next() {
			var gid, display string
			if e = rows.Scan(&gid, &display); e != nil {
				rows.Close()
				return nil, e
			}
			groups = append(groups, map[string]any{"value": gid, "display": display, "type": "direct", "$ref": "/api/v1/scim/v2/Groups/" + gid})
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return nil, e
		}
		out["groups"] = groups
	} else {
		e := q.QueryRow(ctx, `SELECT display_name,external_id,version,created_at,updated_at FROM scim_groups WHERE workspace_id=$1 AND id=$2`, wid, id).Scan(&name, &external, &version, &created, &updated)
		if e != nil {
			return nil, e
		}
		out["schemas"] = []string{scimGroupSchema}
		out["displayName"] = name
		rows, e := q.Query(ctx, `SELECT u.id::text,u.username FROM scim_users u JOIN scim_group_members m ON m.user_id=u.id WHERE m.workspace_id=$1 AND m.group_id=$2 AND u.deleted_at IS NULL ORDER BY u.username`, wid, id)
		if e != nil {
			return nil, e
		}
		members := []any{}
		for rows.Next() {
			var uid, display string
			if e = rows.Scan(&uid, &display); e != nil {
				rows.Close()
				return nil, e
			}
			members = append(members, map[string]any{"value": uid, "display": display, "type": "User", "$ref": "/api/v1/scim/v2/Users/" + uid})
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return nil, e
		}
		out["members"] = members
	}
	out["id"] = id
	out["externalId"] = external
	out["meta"] = map[string]any{"resourceType": strings.TrimSuffix(kind, "s"), "created": created.UTC().Format(time.RFC3339Nano), "lastModified": updated.UTC().Format(time.RFC3339Nano), "version": scimETag(version), "location": "/api/v1/scim/v2/" + kind + "/" + id}
	return out, nil
}

func (s *Server) scimGet(w http.ResponseWriter, r *http.Request) {
	if !validID(r.PathValue("id")) {
		scimFailure(w, pgx.ErrNoRows)
		return
	}
	out, e := scimResource(r.Context(), s.DB, scimKind(r), current(r).WorkspaceID, r.PathValue("id"))
	if e != nil {
		scimFailure(w, e)
		return
	}
	w.Header().Set("ETag", str(out["meta"].(map[string]any), "version"))
	scimJSON(w, 200, out)
}

var scimFilterPattern = regexp.MustCompile(`(?i)^\s*(userName|displayName|externalId|id)\s+eq\s+("(?:[^"\\]|\\.)*")\s*$`)

func (s *Server) scimList(w http.ResponseWriter, r *http.Request) {
	kind, wid := scimKind(r), current(r).WorkspaceID
	table, name := "scim_users", "username"
	where := "workspace_id=$1 AND deleted_at IS NULL"
	if kind == "Groups" {
		table, name, where = "scim_groups", "display_name", "workspace_id=$1"
	}
	args := []any{wid}
	if filter := r.URL.Query().Get("filter"); filter != "" {
		match := scimFilterPattern.FindStringSubmatch(filter)
		if match == nil {
			scimFailure(w, scimProblem(400, "invalidFilter", "지원 필터: userName/displayName/externalId/id eq \"값\""))
			return
		}
		var value string
		if json.Unmarshal([]byte(match[2]), &value) != nil {
			scimFailure(w, scimProblem(400, "invalidFilter", "문자열 필터를 확인하세요"))
			return
		}
		column := ""
		switch strings.ToLower(match[1]) {
		case "username":
			if kind == "Users" {
				column = name
			}
		case "displayname":
			if kind == "Groups" {
				column = name
			}
		case "externalid":
			column = "external_id"
		case "id":
			column = "id::text"
		}
		if column == "" {
			scimFailure(w, scimProblem(400, "invalidFilter", "이 리소스에 없는 필터 속성입니다"))
			return
		}
		args = append(args, value)
		where += " AND lower(" + column + ")=lower($2)"
	}
	start, count := 1, 100
	for key, target := range map[string]*int{"startIndex": &start, "count": &count} {
		if text := r.URL.Query().Get(key); text != "" {
			v, e := strconv.Atoi(text)
			if e != nil || v < 0 || v > 10000000 {
				scimFailure(w, scimProblem(400, "invalidValue", "페이지 매개변수를 확인하세요"))
				return
			}
			*target = v
		}
	}
	if start < 1 {
		start = 1
	}
	if count > 200 {
		count = 200
	}
	if r.URL.Query().Get("sortBy") != "" {
		scimFailure(w, scimProblem(400, "invalidValue", "SCIM 서버 정렬은 지원하지 않습니다"))
		return
	}
	var total int
	if e := s.DB.QueryRow(r.Context(), `SELECT count(*) FROM `+table+` WHERE `+where, args...).Scan(&total); e != nil {
		scimFailure(w, e)
		return
	}
	args = append(args, count, start-1)
	rows, e := s.DB.Query(r.Context(), `SELECT id::text FROM `+table+` WHERE `+where+` ORDER BY id LIMIT $`+strconv.Itoa(len(args)-1)+` OFFSET $`+strconv.Itoa(len(args)), args...)
	if e != nil {
		scimFailure(w, e)
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
		scimFailure(w, e)
		return
	}
	resources := []any{}
	for _, id := range ids {
		out, e := scimResource(r.Context(), s.DB, kind, wid, id)
		if errors.Is(e, pgx.ErrNoRows) {
			continue
		}
		if e != nil {
			scimFailure(w, e)
			return
		}
		resources = append(resources, out)
	}
	scimJSON(w, 200, map[string]any{"schemas": []string{scimListSchema}, "totalResults": total, "startIndex": start, "itemsPerPage": len(resources), "Resources": resources})
}

func scimUserInput(input map[string]any) (map[string]any, string, string, string, bool, error) {
	userName := strings.TrimSpace(str(input, "userName"))
	if userName == "" || len(userName) > 254 || strings.ContainsAny(userName, "\x00\r\n") {
		return nil, "", "", "", false, scimProblem(400, "invalidValue", "userName은 필수이며 254자 이하입니다")
	}
	active := true
	if v, exists := input["active"]; exists {
		var ok bool
		active, ok = v.(bool)
		if !ok {
			return nil, "", "", "", false, scimProblem(400, "invalidValue", "active는 boolean입니다")
		}
	}
	email := ""
	emails, ok := input["emails"].([]any)
	if !ok || len(emails) == 0 || len(emails) > 20 {
		return nil, "", "", "", false, scimProblem(400, "invalidValue", "emails는 1~20개 이메일 배열입니다")
	}
	primary := 0
	for i, raw := range emails {
		m, ok := raw.(map[string]any)
		if !ok {
			return nil, "", "", "", false, scimProblem(400, "invalidValue", "emails 형식을 확인하세요")
		}
		value := strings.ToLower(strings.TrimSpace(str(m, "value")))
		parsed, e := mail.ParseAddress(value)
		if e != nil || parsed.Address != value || len(value) > 254 {
			return nil, "", "", "", false, scimProblem(400, "invalidValue", "올바른 이메일 주소가 필요합니다")
		}
		m["value"] = value
		if i == 0 {
			email = value
		}
		if boolean(m, "primary") {
			primary++
			email = value
		}
	}
	if primary > 1 {
		return nil, "", "", "", false, scimProblem(400, "invalidValue", "primary 이메일은 하나만 가능합니다")
	}
	profile := map[string]any{"userName": userName, "active": active, "emails": emails}
	for _, key := range []string{"displayName", "nickName", "title", "userType", "preferredLanguage", "locale", "timezone", "profileUrl", "externalId"} {
		if raw, exists := input[key]; exists {
			value, ok := raw.(string)
			if !ok || len(value) > 2000 {
				return nil, "", "", "", false, scimProblem(400, "invalidValue", key+" 문자열을 확인하세요")
			}
			profile[key] = value
		}
	}
	if raw, exists := input["name"]; exists {
		m, ok := raw.(map[string]any)
		if !ok {
			return nil, "", "", "", false, scimProblem(400, "invalidValue", "name 형식을 확인하세요")
		}
		name := map[string]any{}
		for _, key := range []string{"formatted", "familyName", "givenName", "middleName", "honorificPrefix", "honorificSuffix"} {
			if raw, exists := m[key]; exists {
				v, ok := raw.(string)
				if !ok || len(v) > 250 {
					return nil, "", "", "", false, scimProblem(400, "invalidValue", "name 속성을 확인하세요")
				}
				name[key] = v
			}
		}
		profile["name"] = name
	}
	for _, key := range []string{"password", "roles", "entitlements", "x509Certificates"} {
		if _, exists := input[key]; exists {
			return nil, "", "", "", false, scimProblem(400, "mutability", key+"는 SCIM으로 변경할 수 없습니다")
		}
	}
	name := str(profile, "displayName")
	if name == "" {
		if m, ok := profile["name"].(map[string]any); ok {
			name = str(m, "formatted")
		}
	}
	if name == "" {
		name = userName
	}
	if len(name) > 250 {
		return nil, "", "", "", false, scimProblem(400, "invalidValue", "표시 이름은 250자 이하입니다")
	}
	return profile, userName, email, name, active, nil
}

var scimMemberPath = regexp.MustCompile(`(?i)^members\[value\s+eq\s+("(?:[^"\\]|\\.)*")\]$`)

func scimPatch(kind string, current, input map[string]any) (map[string]any, error) {
	schemas := listStrings(input["schemas"])
	if !slices.Contains(schemas, scimPatchSchema) {
		return nil, scimProblem(400, "invalidSyntax", "PatchOp schema가 필요합니다")
	}
	operations, ok := input["Operations"].([]any)
	if !ok || len(operations) == 0 || len(operations) > 100 {
		return nil, scimProblem(400, "invalidValue", "Operations는 1~100개 배열입니다")
	}
	result := map[string]any{}
	for k, v := range current {
		result[k] = v
	}
	for _, raw := range operations {
		operation, ok := raw.(map[string]any)
		if !ok {
			return nil, scimProblem(400, "invalidSyntax", "잘못된 PATCH 작업입니다")
		}
		op, path := strings.ToLower(str(operation, "op")), str(operation, "path")
		if !oneOf(op, "add", "replace", "remove") {
			return nil, scimProblem(400, "invalidSyntax", "지원 작업은 add/replace/remove입니다")
		}
		if path == "" {
			object, ok := operation["value"].(map[string]any)
			if !ok || op == "remove" {
				return nil, scimProblem(400, "noTarget", "경로 없는 PATCH에는 객체 값이 필요합니다")
			}
			for key, value := range object {
				if key == "" {
					return nil, scimProblem(400, "invalidPath", "빈 속성 경로는 허용하지 않습니다")
				}
				nested := map[string]any{"schemas": []any{scimPatchSchema}, "Operations": []any{map[string]any{"op": op, "path": key, "value": value}}}
				var e error
				result, e = scimPatch(kind, result, nested)
				if e != nil {
					return nil, e
				}
			}
			continue
		}
		if oneOf(strings.ToLower(path), "id", "meta", "schemas", "groups", "password", "roles") {
			return nil, scimProblem(400, "mutability", "읽기 전용 속성입니다")
		}
		for _, canonical := range []string{"userName", "active", "emails", "displayName", "nickName", "name", "title", "userType", "preferredLanguage", "locale", "timezone", "profileUrl", "externalId", "members", "name.formatted", "name.familyName", "name.givenName", "name.middleName", "name.honorificPrefix", "name.honorificSuffix"} {
			if strings.EqualFold(path, canonical) {
				path = canonical
				break
			}
		}
		if op == "remove" && ((kind == "Users" && oneOf(path, "userName", "active", "emails")) || (kind == "Groups" && path == "displayName")) {
			return nil, scimProblem(400, "mutability", "필수 속성은 삭제할 수 없습니다")
		}
		if kind == "Groups" {
			if match := scimMemberPath.FindStringSubmatch(path); match != nil {
				if op != "remove" {
					return nil, scimProblem(400, "invalidPath", "필터 members 경로는 remove만 지원합니다")
				}
				var uid string
				if json.Unmarshal([]byte(match[1]), &uid) != nil {
					return nil, scimProblem(400, "invalidPath", "잘못된 members 필터입니다")
				}
				values, _ := result["members"].([]any)
				out := []any{}
				for _, raw := range values {
					m, _ := raw.(map[string]any)
					if str(m, "value") != uid {
						out = append(out, raw)
					}
				}
				result["members"] = out
				continue
			}
			if !oneOf(path, "members", "displayName", "externalId") {
				return nil, scimProblem(400, "invalidPath", "지원하지 않는 그룹 속성 경로입니다")
			}
		} else if !oneOf(path, "userName", "active", "emails", "displayName", "nickName", "name", "title", "userType", "preferredLanguage", "locale", "timezone", "profileUrl", "externalId") && !strings.HasPrefix(path, "name.") {
			return nil, scimProblem(400, "invalidPath", "지원하지 않는 사용자 속성 경로입니다")
		}
		if strings.HasPrefix(path, "name.") {
			key := strings.TrimPrefix(path, "name.")
			if !oneOf(key, "formatted", "familyName", "givenName", "middleName", "honorificPrefix", "honorificSuffix") {
				return nil, scimProblem(400, "invalidPath", "지원하지 않는 name 경로입니다")
			}
			old, _ := result["name"].(map[string]any)
			clone := map[string]any{}
			for k, v := range old {
				clone[k] = v
			}
			if op == "remove" {
				delete(clone, key)
			} else {
				clone[key] = operation["value"]
			}
			result["name"] = clone
			continue
		}
		if op == "remove" {
			delete(result, path)
		} else if op == "add" && (path == "members" || path == "emails") {
			old, _ := result[path].([]any)
			add, ok := operation["value"].([]any)
			if !ok {
				return nil, scimProblem(400, "invalidValue", "다중 값은 배열이어야 합니다")
			}
			result[path] = append(append([]any{}, old...), add...)
		} else {
			result[path] = operation["value"]
		}
	}
	return result, nil
}

func (s *Server) scimWrite(w http.ResponseWriter, r *http.Request) {
	kind, wid, id := scimKind(r), current(r).WorkspaceID, r.PathValue("id")
	create := r.Method == "POST"
	if !create && !validID(id) {
		scimFailure(w, pgx.ErrNoRows)
		return
	}
	input := map[string]any{}
	if r.Method != "DELETE" {
		r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
		if e := decode(r, &input); e != nil {
			scimFailure(w, scimProblem(400, "invalidSyntax", "JSON 요청을 확인하세요"))
			return
		}
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		scimFailure(w, e)
		return
	}
	defer tx.Rollback(r.Context())
	if _, e = tx.Exec(r.Context(), `SELECT pg_advisory_xact_lock(729421587)`); e != nil {
		scimFailure(w, e)
		return
	}
	if create {
		id = newID()
	} else {
		old, e := scimResource(r.Context(), tx, kind, wid, id)
		if e != nil {
			scimFailure(w, e)
			return
		}
		etag := str(old["meta"].(map[string]any), "version")
		if match := r.Header.Get("If-Match"); match != "" && match != "*" && match != etag {
			scimFailure(w, scimProblem(412, "", "리소스가 변경되었습니다. 새 ETag로 다시 요청하세요"))
			return
		}
		if r.Method == "PATCH" {
			input, e = scimPatch(kind, old, input)
			if e != nil {
				scimFailure(w, e)
				return
			}
		}
	}
	if value := str(input, "id"); value != "" && value != id {
		scimFailure(w, scimProblem(400, "mutability", "리소스 ID는 변경할 수 없습니다"))
		return
	}
	if r.Method != "DELETE" && r.Method != "PATCH" {
		expected := scimUserSchema
		if kind == "Groups" {
			expected = scimGroupSchema
		}
		schemas := listStrings(input["schemas"])
		if !slices.Contains(schemas, expected) || len(schemas) != 1 {
			scimFailure(w, scimProblem(400, "invalidValue", "지원하는 SCIM core schema 하나가 필요합니다"))
			return
		}
	}
	var rawCfg []byte
	if e = tx.QueryRow(r.Context(), `SELECT data FROM settings WHERE id=1`).Scan(&rawCfg); e != nil {
		scimFailure(w, e)
		return
	}
	cfg, e := s.decodeSettings(rawCfg)
	if e != nil {
		scimFailure(w, e)
		return
	}
	if !boolean(cfg, "scim_enabled") || !slices.Contains(settingStrings(cfg, "allowed_key_scopes", keyScopes), scimProvisionScope) {
		scimFailure(w, scimProblem(403, "", "SCIM 프로비저닝이 비활성화되었습니다"))
		return
	}
	if kind == "Users" {
		e = s.scimWriteUser(r.Context(), tx, cfg, wid, id, input, create, r.Method == "DELETE")
	} else {
		e = s.scimWriteGroup(r.Context(), tx, cfg, wid, id, input, create, r.Method == "DELETE")
	}
	if e != nil {
		scimFailure(w, e)
		return
	}
	var out map[string]any
	if r.Method != "DELETE" {
		out, e = scimResource(r.Context(), tx, kind, wid, id)
		if e != nil {
			scimFailure(w, e)
			return
		}
	}
	if e = tx.Commit(r.Context()); e != nil {
		scimFailure(w, e)
		return
	}
	s.audit(r, "SCIM_"+r.Method, id, map[string]any{"resource_type": kind, "workspace_id": wid})
	if r.Method == "DELETE" {
		scimJSON(w, 204, nil)
		return
	}
	w.Header().Set("Location", "/api/v1/scim/v2/"+kind+"/"+id)
	w.Header().Set("ETag", str(out["meta"].(map[string]any), "version"))
	status := 200
	if create {
		status = 201
	}
	scimJSON(w, status, out)
}

func (s *Server) scimWriteUser(ctx context.Context, tx pgx.Tx, cfg map[string]any, wid, id string, input map[string]any, create, remove bool) error {
	if !create {
		var protected bool
		if e := tx.QueryRow(ctx, `SELECT kind!='user' OR role='admin' FROM users WHERE id=$1 FOR UPDATE`, id).Scan(&protected); e != nil {
			return e
		}
		if protected {
			return scimProblem(403, "mutability", "서비스 관리자나 서비스 계정은 SCIM으로 변경할 수 없습니다")
		}
		// User names are embedded in Group.members responses, and deletion changes
		// their membership arrays. Invalidate each affected group's ETag atomically.
		if _, e := tx.Exec(ctx, `UPDATE scim_groups SET version=version+1,updated_at=now() WHERE workspace_id=$1 AND id IN (SELECT group_id FROM scim_group_members WHERE workspace_id=$1 AND user_id=$2)`, wid, id); e != nil {
			return e
		}
	}
	if remove {
		if _, e := tx.Exec(ctx, `UPDATE scim_users SET active=false,deleted_at=now(),updated_at=now(),version=version+1 WHERE id=$1 AND workspace_id=$2`, id, wid); e != nil {
			return e
		}
		if _, e := tx.Exec(ctx, `UPDATE users SET disabled=true WHERE id=$1`, id); e != nil {
			return e
		}
		if _, e := tx.Exec(ctx, `DELETE FROM scim_group_members WHERE workspace_id=$1 AND user_id=$2`, wid, id); e != nil {
			return e
		}
	} else {
		profile, username, email, name, active, e := scimUserInput(input)
		if e != nil {
			return e
		}
		if create {
			if _, e = tx.Exec(ctx, `INSERT INTO users(id,email,name,role,kind,disabled) VALUES($1,$2,$3,'viewer','user',$4)`, id, email, name, !active); e != nil {
				return e
			}
			if _, e = tx.Exec(ctx, `INSERT INTO scim_users(id,workspace_id,username,external_id,profile,active) VALUES($1,$2,$3,$4,$5,$6)`, id, wid, username, str(profile, "externalId"), jsonValue(profile), active); e != nil {
				return e
			}
		} else {
			if _, e = tx.Exec(ctx, `UPDATE users SET email=$2,name=$3,disabled=$4 WHERE id=$1`, id, email, name, !active); e != nil {
				return e
			}
			if _, e = tx.Exec(ctx, `UPDATE scim_users SET username=$3,external_id=$4,profile=$5,active=$6,version=version+1,updated_at=now() WHERE id=$1 AND workspace_id=$2`, id, wid, username, str(profile, "externalId"), jsonValue(profile), active); e != nil {
				return e
			}
		}
	}
	if e := s.scimSyncUser(ctx, tx, cfg, wid, id); e != nil {
		return e
	}
	var disabled bool
	if e := tx.QueryRow(ctx, `SELECT disabled FROM users WHERE id=$1`, id).Scan(&disabled); e != nil {
		return e
	}
	if disabled {
		if _, e := tx.Exec(ctx, `DELETE FROM sessions WHERE user_id=$1`, id); e != nil {
			return e
		}
	}
	return nil
}

func (s *Server) scimSyncUser(ctx context.Context, tx pgx.Tx, cfg map[string]any, wid, uid string) error {
	var active bool
	if e := tx.QueryRow(ctx, `SELECT active AND deleted_at IS NULL FROM scim_users WHERE workspace_id=$1 AND id=$2`, wid, uid).Scan(&active); e != nil {
		return e
	}
	groups := []string{}
	if active {
		rows, e := tx.Query(ctx, `SELECT g.display_name FROM scim_groups g JOIN scim_group_members m ON m.group_id=g.id WHERE m.workspace_id=$1 AND m.user_id=$2 ORDER BY g.display_name`, wid, uid)
		if e != nil {
			return e
		}
		for rows.Next() {
			var group string
			if e = rows.Scan(&group); e != nil {
				rows.Close()
				return e
			}
			groups = append(groups, group)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, `INSERT INTO identity_grants(provider,issuer,user_id,workspace_id,role) VALUES('scim-base',$1::text,$2,$1::uuid,'viewer') ON CONFLICT(provider,issuer,user_id,workspace_id) DO NOTHING`, wid, uid); e != nil {
			return e
		}
	} else {
		if _, e := tx.Exec(ctx, `DELETE FROM identity_grants WHERE provider='scim-base' AND issuer=$1 AND user_id=$2`, wid, uid); e != nil {
			return e
		}
	}
	if _, e := tx.Exec(ctx, `INSERT INTO identity_links(id,user_id,provider,issuer,subject,groups) VALUES($1,$2::uuid,'scim',$3,$2::text,$4) ON CONFLICT(provider,issuer,subject) DO UPDATE SET groups=EXCLUDED.groups,updated_at=now()`, newID(), uid, wid, jsonValue(groups)); e != nil {
		return e
	}
	return s.identitySyncGroups(ctx, tx, cfg, uid, "scim", wid, groups)
}

func (s *Server) scimWriteGroup(ctx context.Context, tx pgx.Tx, cfg map[string]any, wid, id string, input map[string]any, create, remove bool) error {
	affected := map[string]bool{}
	if !create {
		rows, e := tx.Query(ctx, `SELECT user_id::text FROM scim_group_members WHERE workspace_id=$1 AND group_id=$2`, wid, id)
		if e != nil {
			return e
		}
		for rows.Next() {
			var uid string
			if e = rows.Scan(&uid); e != nil {
				rows.Close()
				return e
			}
			affected[uid] = true
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return e
		}
	}
	if remove {
		if _, e := tx.Exec(ctx, `DELETE FROM scim_groups WHERE workspace_id=$1 AND id=$2`, wid, id); e != nil {
			return e
		}
	} else {
		name, external := strings.TrimSpace(str(input, "displayName")), str(input, "externalId")
		if name == "" || len(name) > 250 || len(external) > 2000 {
			return scimProblem(400, "invalidValue", "displayName은 1~250자, externalId는 2,000자 이하입니다")
		}
		members := []any{}
		if raw, exists := input["members"]; exists {
			var ok bool
			members, ok = raw.([]any)
			if !ok || len(members) > 1000 {
				return scimProblem(400, "invalidValue", "members는 최대 1,000개 배열입니다")
			}
		}
		ids := map[string]bool{}
		for _, raw := range members {
			m, ok := raw.(map[string]any)
			if !ok || !validID(str(m, "value")) || (str(m, "type") != "" && !strings.EqualFold(str(m, "type"), "User")) {
				return scimProblem(400, "invalidValue", "그룹에는 이 워크스페이스의 SCIM User ID만 추가할 수 있습니다")
			}
			uid := str(m, "value")
			var allowed bool
			if e := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM scim_users su JOIN users u ON u.id=su.id WHERE su.workspace_id=$1 AND su.id=$2 AND su.deleted_at IS NULL AND u.role!='admin' AND u.kind='user')`, wid, uid).Scan(&allowed); e != nil {
				return e
			}
			if !allowed {
				return scimProblem(400, "invalidValue", "접근할 수 없는 SCIM 사용자입니다")
			}
			ids[uid] = true
			affected[uid] = true
		}
		if create {
			if _, e := tx.Exec(ctx, `INSERT INTO scim_groups(id,workspace_id,display_name,external_id) VALUES($1,$2,$3,$4)`, id, wid, name, external); e != nil {
				return e
			}
		} else {
			if _, e := tx.Exec(ctx, `UPDATE scim_groups SET display_name=$3,external_id=$4,version=version+1,updated_at=now() WHERE workspace_id=$1 AND id=$2`, wid, id, name, external); e != nil {
				return e
			}
			if _, e := tx.Exec(ctx, `DELETE FROM scim_group_members WHERE workspace_id=$1 AND group_id=$2`, wid, id); e != nil {
				return e
			}
		}
		for uid := range ids {
			if _, e := tx.Exec(ctx, `INSERT INTO scim_group_members(workspace_id,group_id,user_id) VALUES($1,$2,$3)`, wid, id, uid); e != nil {
				return e
			}
		}
	}
	ids := []string{}
	for uid := range affected {
		ids = append(ids, uid)
	}
	slices.Sort(ids)
	for _, uid := range ids {
		if e := s.scimSyncUser(ctx, tx, cfg, wid, uid); e != nil {
			return e
		}
		if _, e := tx.Exec(ctx, `UPDATE scim_users SET version=version+1,updated_at=now() WHERE id=$1`, uid); e != nil {
			return e
		}
	}
	return nil
}

func (s *Server) scimDiscovery(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, "ServiceProviderConfig") {
		scimJSON(w, 200, map[string]any{"schemas": []string{"urn:ietf:params:scim:schemas:core:2.0:ServiceProviderConfig"}, "patch": map[string]any{"supported": true}, "bulk": map[string]any{"supported": false, "maxOperations": 0, "maxPayloadSize": 0}, "filter": map[string]any{"supported": true, "maxResults": 200}, "changePassword": map[string]any{"supported": false}, "sort": map[string]any{"supported": false}, "etag": map[string]any{"supported": true}, "authenticationSchemes": []any{map[string]any{"type": "oauthbearertoken", "name": "madi service-account token", "description": "workspace-scoped identity:provision API key", "specUri": "https://www.rfc-editor.org/rfc/rfc6750", "primary": true}}})
		return
	}
	resources := []any{}
	for _, kind := range []string{"User", "Group"} {
		schema := scimUserSchema
		name := "userName"
		if kind == "Group" {
			schema, name = scimGroupSchema, "displayName"
		}
		if strings.Contains(r.URL.Path, "/ResourceTypes") {
			resources = append(resources, map[string]any{"schemas": []string{"urn:ietf:params:scim:schemas:core:2.0:ResourceType"}, "id": kind, "name": kind, "endpoint": "/" + kind + "s", "schema": schema, "schemaExtensions": []any{}})
		} else {
			attributes := []any{map[string]any{"name": name, "type": "string", "multiValued": false, "required": true, "caseExact": false, "mutability": "readWrite", "returned": "default", "uniqueness": "server"}, map[string]any{"name": "externalId", "type": "string", "multiValued": false, "required": false, "mutability": "readWrite", "returned": "default", "uniqueness": "none"}}
			if kind == "User" {
				attributes = append(attributes, map[string]any{"name": "active", "type": "boolean", "multiValued": false, "mutability": "readWrite", "returned": "default"}, map[string]any{"name": "emails", "type": "complex", "multiValued": true, "required": true, "mutability": "readWrite", "returned": "default", "subAttributes": []any{map[string]any{"name": "value", "type": "string", "required": true}, map[string]any{"name": "primary", "type": "boolean"}, map[string]any{"name": "type", "type": "string"}}})
				for _, field := range []string{"displayName", "nickName", "title", "userType", "preferredLanguage", "locale", "timezone", "profileUrl"} {
					attributes = append(attributes, map[string]any{"name": field, "type": "string", "multiValued": false, "mutability": "readWrite", "returned": "default"})
				}
				names := []any{}
				for _, field := range []string{"formatted", "familyName", "givenName", "middleName", "honorificPrefix", "honorificSuffix"} {
					names = append(names, map[string]any{"name": field, "type": "string", "multiValued": false, "mutability": "readWrite"})
				}
				attributes = append(attributes, map[string]any{"name": "name", "type": "complex", "multiValued": false, "mutability": "readWrite", "returned": "default", "subAttributes": names}, map[string]any{"name": "groups", "type": "complex", "multiValued": true, "mutability": "readOnly", "returned": "default", "subAttributes": []any{map[string]any{"name": "value", "type": "string"}, map[string]any{"name": "display", "type": "string"}, map[string]any{"name": "type", "type": "string"}, map[string]any{"name": "$ref", "type": "reference", "referenceTypes": []string{"Group"}}}})
			} else {
				attributes = append(attributes, map[string]any{"name": "members", "type": "complex", "multiValued": true, "mutability": "readWrite", "returned": "default", "subAttributes": []any{map[string]any{"name": "value", "type": "string", "required": true}}})
			}
			resources = append(resources, map[string]any{"schemas": []string{"urn:ietf:params:scim:schemas:core:2.0:Schema"}, "id": schema, "name": kind, "attributes": attributes})
		}
	}
	if id := r.PathValue("id"); id != "" {
		for _, raw := range resources {
			item := raw.(map[string]any)
			if str(item, "id") == id {
				scimJSON(w, 200, item)
				return
			}
		}
		scimFailure(w, pgx.ErrNoRows)
		return
	}
	scimJSON(w, 200, map[string]any{"schemas": []string{scimListSchema}, "totalResults": len(resources), "startIndex": 1, "itemsPerPage": len(resources), "Resources": resources})
}
