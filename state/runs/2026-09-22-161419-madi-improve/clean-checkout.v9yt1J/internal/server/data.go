package server

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func (s *Server) registerData() {
	s.handle("GET /api/v1/databases", s.listDatabases)
	s.handle("POST /api/v1/databases", s.createDatabase)
	s.handle("GET /api/v1/databases/{id}", s.getDatabase)
	s.handle("PUT /api/v1/databases/{id}", s.updateDatabase)
	s.handle("DELETE /api/v1/databases/{id}", s.deleteDatabase)
	s.handle("GET /api/v1/databases/{id}/rows", s.listRows)
	s.handle("POST /api/v1/databases/{id}/rows", s.createRow)
	s.handle("PUT /api/v1/databases/{id}/rows/{rowId}", s.updateRow)
	s.handle("DELETE /api/v1/databases/{id}/rows/{rowId}", s.deleteRow)
	s.handle("POST /api/v1/attachments", s.uploadAttachment)
	s.handle("GET /api/v1/attachments/{id}", s.downloadAttachment)
	s.handle("GET /api/v1/documents/{id}/attachments", s.listAttachments)
	s.handle("GET /api/v1/export", s.exportMarkdown)
	s.handle("POST /api/v1/import", s.importMarkdown)
}
func (s *Server) canDatabase(r *http.Request, id string, write bool) bool {
	if !validID(id) {
		return false
	}
	var wid, space string
	if s.DB.QueryRow(r.Context(), "SELECT workspace_id,coalesce(space_id::text,'') FROM databases WHERE id=$1", id).Scan(&wid, &space) != nil {
		return false
	}
	return s.canSpace(r.Context(), current(r), wid, space, write)
}
func (s *Server) listDatabases(w http.ResponseWriter, r *http.Request) {
	p := current(r)
	wid := r.URL.Query().Get("workspace_id")
	if p.WorkspaceID != "" {
		if wid != "" && p.WorkspaceID != wid {
			apiError(w, 403, "API 키 범위를 벗어난 워크스페이스입니다")
			return
		}
		wid = p.WorkspaceID
	}
	v, e := s.rows(r.Context(), "SELECT to_jsonb(d)||jsonb_build_object('can_write',madi_space_allowed($1,d.space_id,true) AND m.role IN ('owner','admin','editor') AND $3) FROM databases d JOIN workspace_members m ON d.workspace_id=m.workspace_id WHERE m.user_id=$1 AND ($2='' OR d.workspace_id::text=$2) AND madi_space_allowed($1,d.space_id,false) ORDER BY d.created_at", p.ID, wid, p.Role != "viewer" && hasIntegrationScope(p, "database:write"))
	respond(w, v, e)
}
func validateProperties(v any) error {
	a, ok := v.([]any)
	if !ok {
		return fmt.Errorf("속성은 배열이어야 합니다")
	}
	if len(a) > 100 {
		return fmt.Errorf("속성은 100개까지 지원합니다")
	}
	seen := map[string]bool{}
	for _, raw := range a {
		p, ok := raw.(map[string]any)
		if !ok {
			return fmt.Errorf("속성 형식을 확인하세요")
		}
		id, name, kind := str(p, "id"), str(p, "name"), str(p, "type")
		if id == "" || name == "" || seen[id] || len(id) > 100 || len(name) > 100 {
			return fmt.Errorf("속성 ID와 이름은 필수이며 중복 ID는 사용할 수 없습니다")
		}
		seen[id] = true
		if !oneOf(kind, "text", "number", "select", "multi_select", "multiselect", "date", "checkbox", "user", "url", "status", "email", "phone") && !isAdvancedPropertyType(kind) {
			return fmt.Errorf("지원하지 않는 속성 유형: %s", kind)
		}
		if opts, exists := p["options"]; exists {
			options, ok := opts.([]any)
			if !ok {
				return fmt.Errorf("선택 옵션은 배열이어야 합니다")
			}
			if len(options) > 100 {
				return fmt.Errorf("선택 옵션은 100개까지 지원합니다")
			}
			seenOptions := map[string]bool{}
			for _, rawOption := range options {
				option, ok := rawOption.(string)
				if !ok || strings.TrimSpace(option) == "" || seenOptions[option] || len(option) > 200 {
					return fmt.Errorf("선택 옵션은 중복 없는 1~200바이트 문자열이어야 합니다")
				}
				seenOptions[option] = true
			}
		} else {
			p["options"] = []any{}
		}
	}
	properties, e := advancedProperties(v)
	if e != nil {
		return e
	}
	return validateAdvancedProperties(properties)
}
func (s *Server) createDatabase(w http.ResponseWriter, r *http.Request) {
	var in map[string]any
	if decode(r, &in) != nil {
		apiError(w, 400, "데이터베이스 정보를 확인하세요")
		return
	}
	wid := str(in, "workspace_id")
	space := str(in, "space_id")
	if !s.canSpace(r.Context(), current(r), wid, space, true) {
		apiError(w, 403, "데이터베이스 생성 권한이 없습니다")
		return
	}
	name := strings.TrimSpace(str(in, "name"))
	if name == "" || len(name) > 200 {
		apiError(w, 400, "데이터베이스 이름을 확인하세요")
		return
	}
	props := in["properties"]
	if props == nil {
		props = []any{map[string]any{"id": "title", "name": "이름", "type": "text"}, map[string]any{"id": "status", "name": "상태", "type": "select", "options": []any{"시작 전", "진행 중", "완료"}}}
	}
	if e := validateProperties(props); e != nil {
		apiError(w, 400, e.Error())
		return
	}
	id := newID()
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	propertyList, e := advancedProperties(props)
	if e == nil {
		e = s.validateAdvancedRelations(r, tx, wid, id, propertyList)
	}
	if e != nil {
		apiError(w, 400, e.Error())
		return
	}
	_, e = tx.Exec(r.Context(), "INSERT INTO databases(id,workspace_id,name,properties,space_id) VALUES($1,$2,$3,$4,NULLIF($5,'')::uuid)", id, wid, name, jsonValue(props), space)
	if e == nil {
		e = s.enqueueDatabaseEvent(r, tx, "database.created", id, "", nil)
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "DATABASE_CREATE", id, nil)
	v, e := s.one(r.Context(), "SELECT to_jsonb(d) FROM databases d WHERE id=$1", id)
	respond(w, v, e)
}
func (s *Server) getDatabase(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !s.canDatabase(r, id, false) {
		apiError(w, 404, "데이터베이스를 찾을 수 없거나 접근 권한이 없습니다")
		return
	}
	v, e := s.one(r.Context(), "SELECT to_jsonb(d) FROM databases d WHERE id=$1", id)
	if e == nil {
		v["can_write"] = s.canDatabase(r, id, true) && hasIntegrationScope(current(r), "database:write")
	}
	respond(w, v, e)
}
func (s *Server) updateDatabase(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !s.canDatabase(r, id, true) {
		apiError(w, 403, "데이터베이스 수정 권한이 없습니다")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	var oldRaw []byte
	e = tx.QueryRow(r.Context(), "SELECT to_jsonb(d) FROM databases d WHERE id=$1 FOR UPDATE", id).Scan(&oldRaw)
	if e != nil {
		respond(w, nil, e)
		return
	}
	old := map[string]any{}
	if e = json.Unmarshal(oldRaw, &old); e != nil {
		respond(w, nil, e)
		return
	}
	var in map[string]any
	if decode(r, &in) != nil {
		apiError(w, 400, "데이터베이스 정보를 확인하세요")
		return
	}
	name := str(old, "name")
	props := old["properties"]
	if v, ok := in["name"].(string); ok {
		name = strings.TrimSpace(v)
	}
	if v, ok := in["properties"]; ok {
		props = v
	}
	if name == "" || len(name) > 200 {
		apiError(w, 400, "이름을 확인하세요")
		return
	}
	if e = validateProperties(props); e != nil {
		apiError(w, 400, e.Error())
		return
	}
	var propertyList []map[string]any
	if e = json.Unmarshal(jsonValue(props), &propertyList); e != nil {
		respond(w, nil, e)
		return
	}
	if e = s.validateAdvancedRelations(r, tx, str(old, "workspace_id"), id, propertyList); e != nil {
		apiError(w, 400, e.Error())
		return
	}
	keys := []string{}
	keySet := map[string]bool{}
	for _, prop := range propertyList {
		key := str(prop, "id")
		keys = append(keys, key)
		keySet[key] = true
	}
	rows, e := tx.Query(r.Context(), "SELECT values FROM database_rows WHERE database_id=$1 FOR UPDATE", id)
	if e != nil {
		respond(w, nil, e)
		return
	}
	for rows.Next() {
		var raw []byte
		var values map[string]any
		if e = rows.Scan(&raw); e != nil {
			break
		}
		if e = json.Unmarshal(raw, &values); e != nil {
			break
		}
		for key := range values {
			if !keySet[key] {
				delete(values, key)
			}
		}
		if e = validateValuesForProperties(propertyList, values); e != nil {
			break
		}
	}
	rows.Close()
	if e == nil {
		e = rows.Err()
	}
	if e != nil {
		apiError(w, 400, "기존 행과 호환되지 않는 속성 변경입니다. 해당 값을 먼저 수정하세요: "+e.Error())
		return
	}
	_, e = tx.Exec(r.Context(), "UPDATE databases SET name=$1,properties=$2 WHERE id=$3", name, jsonValue(props), id)
	if e == nil {
		_, e = tx.Exec(r.Context(), "UPDATE database_rows SET values=(SELECT coalesce(jsonb_object_agg(key,value),'{}'::jsonb) FROM jsonb_each(database_rows.values) WHERE key=ANY($2::text[])) WHERE database_id=$1", id, keys)
	}
	if e == nil {
		e = s.enqueueDatabaseEvent(r, tx, "database.updated", id, "", map[string]any{"title": old["name"], "database_id": id})
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "DATABASE_UPDATE", id, nil)
	v, e := s.one(r.Context(), "SELECT to_jsonb(d) FROM databases d WHERE id=$1", id)
	respond(w, v, e)
}
func (s *Server) deleteDatabase(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !s.canDatabase(r, id, true) {
		apiError(w, 403, "데이터베이스 삭제 권한이 없습니다")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	var raw []byte
	e = tx.QueryRow(r.Context(), "DELETE FROM databases WHERE id=$1 RETURNING to_jsonb(databases)", id).Scan(&raw)
	var old map[string]any
	if e == nil {
		e = json.Unmarshal(raw, &old)
	}
	if e == nil {
		e = s.enqueueEvent(r.Context(), tx, Event{Type: "database.deleted", ResourceType: "database", WorkspaceID: str(old, "workspace_id"), ResourceID: id, Before: map[string]any{"database_id": id, "title": old["name"], "space_id": old["space_id"]}})
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "DATABASE_DELETE", id, nil)
	respond(w, map[string]bool{"ok": true}, e)
}
func (s *Server) listRows(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !s.canDatabase(r, id, false) {
		apiError(w, 403, "데이터베이스 접근 권한이 없습니다")
		return
	}
	v, e := s.rows(r.Context(), "SELECT to_jsonb(v) FROM database_rows v WHERE database_id=$1 ORDER BY created_at LIMIT 10000", id)
	if e == nil {
		v, e = s.enrichDatabaseRows(r, id, v)
	}
	s.audit(r, "DATABASE_QUERY", id, nil)
	respond(w, v, e)
}
func (s *Server) validateValues(r *http.Request, id string, values map[string]any) error {
	if len(jsonValue(values)) > 1<<20 {
		return fmt.Errorf("행 크기는 1MB 이하여야 합니다")
	}
	var raw []byte
	if e := s.DB.QueryRow(r.Context(), "SELECT properties FROM databases WHERE id=$1", id).Scan(&raw); e != nil {
		return e
	}
	var props []map[string]any
	if e := json.Unmarshal(raw, &props); e != nil {
		return e
	}
	if e := validateValuesForProperties(props, values); e != nil {
		return e
	}
	return s.validateAdvancedRowRelations(r, s.DB, id, props, values)
}
func validateValuesForProperties(props []map[string]any, values map[string]any) error {
	lookup := map[string]map[string]any{}
	for _, p := range props {
		lookup[str(p, "id")] = p
	}
	for key, v := range values {
		p, exists := lookup[key]
		if !exists {
			return fmt.Errorf("존재하지 않는 속성: %s", key)
		}
		kind := str(p, "type")
		if oneOf(kind, "formula", "rollup", "created_time", "updated_time", "created_by", "updated_by", "button") {
			return validateAdvancedValue(p, v)
		}
		if text, ok := v.(string); v == nil || (ok && text == "") {
			continue
		}
		if isAdvancedPropertyType(kind) {
			if e := validateAdvancedValue(p, v); e != nil {
				return e
			}
			continue
		}
		switch kind {
		case "number":
			if _, ok := v.(float64); !ok {
				return fmt.Errorf("%s: 숫자를 입력하세요", str(p, "name"))
			}
		case "checkbox":
			if _, ok := v.(bool); !ok {
				return fmt.Errorf("%s: 체크박스 값을 확인하세요", str(p, "name"))
			}
		case "select", "status":
			value, ok := v.(string)
			if !ok || !oneOf(value, listStrings(p["options"])...) {
				return fmt.Errorf("%s: 등록된 옵션을 선택하세요", str(p, "name"))
			}
		case "multi_select", "multiselect":
			a, ok := v.([]any)
			if !ok {
				return fmt.Errorf("다중 선택 값은 배열이어야 합니다")
			}
			for _, item := range a {
				value, ok := item.(string)
				if !ok || !oneOf(value, listStrings(p["options"])...) {
					return fmt.Errorf("등록된 다중 선택 옵션을 선택하세요")
				}
			}
		case "date":
			value, ok := v.(string)
			if !ok {
				return fmt.Errorf("날짜 형식을 확인하세요")
			}
			if _, e := time.Parse("2006-01-02", value); e != nil {
				if _, e = time.Parse(time.RFC3339, value); e != nil {
					return fmt.Errorf("날짜는 YYYY-MM-DD 형식이어야 합니다")
				}
			}
		default:
			if _, ok := v.(string); !ok {
				return fmt.Errorf("%s: 텍스트를 입력하세요", str(p, "name"))
			}
		}
	}
	return nil
}
func (s *Server) validateValuesTx(r *http.Request, tx pgx.Tx, id string, values map[string]any) error {
	if len(jsonValue(values)) > 1<<20 {
		return fmt.Errorf("행 크기는 1MB 이하여야 합니다")
	}
	var raw []byte
	if e := tx.QueryRow(r.Context(), "SELECT properties FROM databases WHERE id=$1 FOR SHARE", id).Scan(&raw); e != nil {
		return e
	}
	var props []map[string]any
	if e := json.Unmarshal(raw, &props); e != nil {
		return e
	}
	if e := validateValuesForProperties(props, values); e != nil {
		return e
	}
	return s.validateAdvancedRowRelations(r, tx, id, props, values)
}
func (s *Server) createRow(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !s.canDatabase(r, id, true) {
		apiError(w, 403, "데이터베이스 수정 권한이 없습니다")
		return
	}
	var in struct {
		Values map[string]any `json:"values"`
	}
	if decode(r, &in) != nil {
		apiError(w, 400, "행 입력값을 확인하세요")
		return
	}
	if in.Values == nil {
		in.Values = map[string]any{}
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	if e = s.prepareStructuredRowTx(r, tx, id, in.Values); e != nil {
		apiError(w, 409, "구조화 초안·원문·권한이 변경되었습니다. 현재 미리보기에서 다시 확인하세요")
		return
	}
	if e := s.validateValuesTx(r, tx, id, in.Values); e != nil {
		apiError(w, 400, e.Error())
		return
	}
	rid := newID()
	_, e = tx.Exec(r.Context(), "INSERT INTO database_rows(id,database_id,values,created_by,updated_by) VALUES($1,$2,$3,$4,$4)", rid, id, jsonValue(in.Values), current(r).ID)
	if e == nil {
		e = s.enqueueDatabaseEvent(r, tx, "database.row.created", id, rid, nil)
	}
	if e == nil {
		e = s.recordAutomationEffect(r.Context(), tx, map[string]any{"id": rid})
	}
	if e == nil {
		e = s.finishStructuredRowTx(r, tx, id, rid)
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "DATABASE_ROW_CREATE", rid, nil)
	v, e := s.one(r.Context(), "SELECT to_jsonb(v) FROM database_rows v WHERE id=$1", rid)
	respond(w, v, e)
}
func (s *Server) updateRow(w http.ResponseWriter, r *http.Request) {
	id, rid := r.PathValue("id"), r.PathValue("rowId")
	if !s.canDatabase(r, id, true) {
		apiError(w, 403, "데이터베이스 수정 권한이 없습니다")
		return
	}
	if !validID(rid) {
		apiError(w, 400, "행 ID를 확인하세요")
		return
	}
	var in struct {
		Values   map[string]any  `json:"values"`
		Expected json.RawMessage `json:"expected_version"`
	}
	if decode(r, &in) != nil || in.Values == nil {
		apiError(w, 400, "행 입력값을 확인하세요")
		return
	}
	var expected *int
	if len(in.Expected) > 0 {
		var n int
		if string(in.Expected) == "null" || json.Unmarshal(in.Expected, &n) != nil || n < 1 || n > 2147483647 {
			apiError(w, 400, "expected_version은1이상의 정수여야 합니다")
			return
		}
		expected = &n
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	if e := s.validateValuesTx(r, tx, id, in.Values); e != nil {
		apiError(w, 400, e.Error())
		return
	}
	if expected != nil {
		var actual int
		if e = tx.QueryRow(r.Context(), "SELECT version FROM database_rows WHERE id=$1 AND database_id=$2 FOR UPDATE", rid, id).Scan(&actual); e != nil {
			respond(w, nil, e)
			return
		}
		if actual != *expected {
			apiError(w, 409, "다른 편집자가 행을 변경했습니다. 현재 값과 다시 비교하세요")
			return
		}
	}
	var raw []byte
	e = tx.QueryRow(r.Context(), "UPDATE database_rows SET values=values||$1::jsonb,updated_at=now(),updated_by=$4 WHERE id=$2 AND database_id=$3 RETURNING to_jsonb(database_rows)", jsonValue(in.Values), rid, id, current(r).ID).Scan(&raw)
	if e == nil {
		e = s.enqueueDatabaseEvent(r, tx, "database.row.updated", id, rid, nil)
	}
	if e == nil {
		e = s.recordAutomationEffect(r.Context(), tx, map[string]any{"id": rid})
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	v := map[string]any{}
	if e == nil {
		e = json.Unmarshal(raw, &v)
	}
	s.audit(r, "DATABASE_ROW_UPDATE", rid, nil)
	respond(w, v, e)
}
func (s *Server) deleteRow(w http.ResponseWriter, r *http.Request) {
	id, rid := r.PathValue("id"), r.PathValue("rowId")
	if !s.canDatabase(r, id, true) || !validID(rid) {
		apiError(w, 403, "행 삭제 권한이 없습니다")
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
	if expected != nil {
		var actual int
		if e = tx.QueryRow(r.Context(), "SELECT version FROM database_rows WHERE id=$1 AND database_id=$2 FOR UPDATE", rid, id).Scan(&actual); e != nil {
			respond(w, nil, e)
			return
		}
		if actual != *expected {
			apiError(w, 409, "다른 편집자가 행을 변경했습니다. 삭제할 상태를 다시 확인하세요")
			return
		}
	}
	tag, e := tx.Exec(r.Context(), "DELETE FROM database_rows WHERE id=$1 AND database_id=$2", rid, id)
	if e == nil && tag.RowsAffected() > 0 {
		e = s.enqueueDatabaseEvent(r, tx, "database.row.deleted", id, rid, nil)
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "DATABASE_ROW_DELETE", rid, nil)
	respond(w, map[string]bool{"ok": true}, e)
}
func (s *Server) uploadAttachment(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("document_id")
	requestID := r.URL.Query().Get("client_request_id")
	if requestID != "" && !validID(requestID) {
		apiError(w, 400, "client_request_id는 UUID여야 합니다")
		return
	}
	requestID = strings.ToLower(requestID)
	if !s.canDocument(r.Context(), current(r), id, true) {
		apiError(w, 403, "첨부파일 업로드 권한이 없습니다")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 52<<20)
	if r.ParseMultipartForm(2<<20) != nil {
		apiError(w, 400, "최대 50MB 파일을 선택하세요")
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	file, h, e := r.FormFile("file")
	if e != nil {
		apiError(w, 400, "파일을 선택하세요")
		return
	}
	defer file.Close()
	if h.Size > 50<<20 {
		apiError(w, 400, "최대 50MB까지 업로드할 수 있습니다")
		return
	}
	name := filepath.Base(strings.ReplaceAll(h.Filename, "\\", "/"))
	payloadHash := ""
	if requestID != "" {
		hash := sha256.New()
		written, err := io.Copy(hash, io.LimitReader(file, (50<<20)+1))
		if err != nil || written > 50<<20 {
			apiError(w, 400, "첨부파일을 읽지 못했거나 50MB를 초과했습니다")
			return
		}
		if _, err = file.Seek(0, io.SeekStart); err != nil {
			respond(w, nil, err)
			return
		}
		payloadHash = digest(id + "\n" + name + "\n" + fmt.Sprintf("%x", hash.Sum(nil)))
	}
	var wid string
	if e = s.DB.QueryRow(r.Context(), "SELECT workspace_id::text FROM documents WHERE id=$1", id).Scan(&wid); e != nil {
		respond(w, nil, e)
		return
	}
	provider, e := s.resolveStorage(r.Context(), wid)
	if e != nil {
		respond(w, nil, e)
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	if requestID != "" {
		if _, e = tx.Exec(r.Context(), "SELECT pg_advisory_xact_lock(hashtextextended($1,82))", current(r).ID+"/"+requestID); e != nil {
			respond(w, nil, e)
			return
		}
		var oldHash, oldAttachment string
		e = tx.QueryRow(r.Context(), "SELECT payload_hash,coalesce(attachment_id::text,'') FROM attachment_receipts WHERE user_id=$1 AND request_id=$2", current(r).ID, requestID).Scan(&oldHash, &oldAttachment)
		if e == nil {
			_ = tx.Rollback(r.Context())
			if oldHash != payloadHash {
				apiError(w, 409, "같은 요청 ID로 다른 첨부파일을 저장할 수 없습니다")
				return
			}
			if oldAttachment == "" {
				apiError(w, 410, "이 요청으로 저장한 첨부파일이 삭제되었습니다")
				return
			}
			v, err := s.one(r.Context(), "SELECT jsonb_build_object('id',id,'name',name,'size',size,'url','/api/v1/attachments/'||id::text,'checksum_sha256',checksum_sha256) FROM attachments WHERE id=$1 AND document_id=$2", oldAttachment, id)
			respond(w, v, err)
			return
		}
		if e != pgx.ErrNoRows {
			respond(w, nil, e)
			return
		}
		e = nil
	}
	if e = tx.QueryRow(r.Context(), "SELECT workspace_id::text FROM documents WHERE id=$1 AND deleted_at IS NULL AND madi_document_allowed($2,id,true) FOR SHARE", id, current(r).ID).Scan(&wid); e != nil {
		respond(w, nil, e)
		return
	}
	fid := newID()
	contentType := mime.TypeByExtension(strings.ToLower(filepath.Ext(name)))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	fileBytes, e := io.ReadAll(io.LimitReader(file, (50<<20)+1))
	if e != nil || len(fileBytes) > 50<<20 {
		apiError(w, 400, "첨부파일을 읽을 수 없거나 50MB를 초과했습니다")
		return
	}
	fileProtection, e := s.ProtectAttachmentTx(r.Context(), tx, current(r), id, wid, name, contentType, fileBytes)
	if e != nil {
		if !WriteProtectionError(w, e) {
			respond(w, nil, e)
		}
		return
	}
	name = fileProtection.Name
	object, e := s.putStoredObject(r.Context(), provider, "attachments/"+fid, bytes.NewReader(fileProtection.Data), 50<<20, contentType)
	if e != nil {
		_ = tx.Rollback(r.Context())
		_ = s.cleanupStoredObject(r.Context(), object)
		apiError(w, 502, e.Error())
		return
	}
	_, e = tx.Exec(r.Context(), "INSERT INTO attachments(id,document_id,user_id,name,content_type,size,path,storage_provider_id,object_key,checksum_sha256) VALUES($1,$2,$3,$4,$5,$6,$7,NULLIF($8,'')::uuid,$9,$10)", fid, id, current(r).ID, name, contentType, object.Size, object.Path, object.ProviderID, object.Key, object.Checksum)
	if e == nil && requestID != "" {
		_, e = tx.Exec(r.Context(), "INSERT INTO attachment_receipts(user_id,request_id,document_id,attachment_id,payload_hash) VALUES($1,$2,$3,$4,$5)", current(r).ID, requestID, id, fid, payloadHash)
	}
	if e == nil {
		e = s.recordAutomationEffect(r.Context(), tx, map[string]any{"id": fid, "document_id": id, "checksum_sha256": object.Checksum})
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		// A lost commit acknowledgement does not prove rollback. Do not delete
		// a possibly committed object: confirm absence using a fresh connection.
		_ = tx.Rollback(r.Context())
		check, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		var exists bool
		if err := s.DB.QueryRow(check, "SELECT EXISTS(SELECT 1 FROM attachments WHERE id=$1)", fid).Scan(&exists); err == nil && !exists {
			_ = s.cleanupStoredObject(check, object)
		}
		cancel()
		respond(w, nil, e)
		return
	}
	s.audit(r, "FILE_UPLOAD", fid, nil)
	jsonResponse(w, 200, map[string]any{"id": fid, "name": name, "url": "/api/v1/attachments/" + fid, "size": object.Size, "checksum_sha256": object.Checksum, "protection": map[string]any{"findings": fileProtection.Findings, "changed": fileProtection.Changed, "unscannable": fileProtection.Unscannable}})
}
func (s *Server) downloadAttachment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validID(id) {
		apiError(w, 404, "첨부파일을 찾을 수 없습니다")
		return
	}
	var doc, name, kind string
	var object storedObject
	e := s.DB.QueryRow(r.Context(), "SELECT document_id,name,content_type,path,coalesce(storage_provider_id::text,''),object_key,checksum_sha256,size FROM attachments WHERE id=$1", id).Scan(&doc, &name, &kind, &object.Path, &object.ProviderID, &object.Key, &object.Checksum, &object.Size)
	if e != nil || !s.canDocument(r.Context(), current(r), doc, false) {
		apiError(w, 404, "첨부파일을 찾을 수 없거나 접근 권한이 없습니다")
		return
	}
	f, e := s.materializeObject(r.Context(), object, 50<<20)
	if e != nil {
		apiError(w, 404, "첨부파일이 저장소에 없습니다")
		return
	}
	defer f.Close()
	defer os.Remove(f.Name())
	stat, e := f.Stat()
	if e != nil {
		respond(w, nil, e)
		return
	}
	disposition := "attachment"
	if oneOf(kind, "image/png", "image/jpeg", "image/gif", "image/webp", "image/avif") {
		disposition = "inline"
	}
	w.Header().Set("Content-Type", kind)
	w.Header().Set("Content-Disposition", mime.FormatMediaType(disposition, map[string]string{"filename": name}))
	w.Header().Set("Content-Security-Policy", "default-src 'none'; sandbox")
	s.audit(r, "FILE_DOWNLOAD", id, nil)
	http.ServeContent(w, r, name, stat.ModTime(), f)
}
