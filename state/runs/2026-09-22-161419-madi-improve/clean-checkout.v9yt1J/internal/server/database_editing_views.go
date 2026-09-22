package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

func validateDatabaseViewData(data map[string]any, properties []map[string]any) error {
	bad := errors.New("보기 유형·필터·정렬·표시 속성 구성을 확인하세요")
	if data == nil || len(jsonValue(data)) > 32768 || !oneOf(str(data, "view"), "table", "board", "calendar", "list", "gallery", "timeline") {
		return bad
	}
	allowed := map[string]bool{"view": true, "filters": true, "sorts": true, "columns": true, "board_property_id": true, "date_property_id": true}
	for key := range data {
		if !allowed[key] {
			return bad
		}
	}
	lookup := map[string]map[string]any{}
	for _, p := range properties {
		lookup[str(p, "id")] = p
	}
	for _, key := range []string{"board_property_id", "date_property_id"} {
		raw, ok := data[key]
		if !ok {
			continue
		}
		value, ok := raw.(string)
		if !ok {
			return bad
		}
		if value == "" {
			continue
		}
		p := lookup[value]
		if p == nil {
			return bad
		}
		if key == "board_property_id" && !oneOf(str(p, "type"), "select", "status") {
			return bad
		}
		if key == "date_property_id" && str(p, "type") != "date" {
			return bad
		}
	}
	if raw, ok := data["columns"]; ok {
		items, ok := raw.([]any)
		if !ok || len(items) > 100 {
			return bad
		}
		seen := map[string]bool{}
		for _, item := range items {
			key, ok := item.(string)
			if !ok || lookup[key] == nil || seen[key] {
				return bad
			}
			seen[key] = true
		}
	}
	if raw, ok := data["filters"]; ok {
		var items []advancedFilter
		if string(jsonValue(raw)) == "null" || json.Unmarshal(jsonValue(raw), &items) != nil || len(items) > 30 {
			return bad
		}
		for _, item := range items {
			if lookup[item.PropertyID] == nil || !oneOf(item.Operator, "eq", "neq", "contains", "not_contains", "is_empty", "not_empty", "gt", "gte", "lt", "lte", "in") {
				return bad
			}
		}
	}
	if raw, ok := data["sorts"]; ok {
		var items []advancedSort
		if string(jsonValue(raw)) == "null" || json.Unmarshal(jsonValue(raw), &items) != nil || len(items) > 10 {
			return bad
		}
		for _, item := range items {
			if lookup[item.PropertyID] == nil || !oneOf(item.Direction, "asc", "desc") {
				return bad
			}
		}
	}
	return nil
}
func (s *Server) listDatabaseViews(w http.ResponseWriter, r *http.Request) {
	id, p := r.PathValue("id"), current(r)
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	scope, e := s.databaseEditingScopeTx(r, tx, id, false)
	if e != nil {
		apiError(w, 403, e.Error())
		return
	}
	rows, e := tx.Query(r.Context(), `SELECT to_jsonb(v) FROM database_views v WHERE database_id=$1 AND(owner_id=$2 OR visibility='workspace') ORDER BY visibility,name,id LIMIT 101`, id, p.ID)
	if e != nil {
		respond(w, nil, e)
		return
	}
	result := []map[string]any{}
	for rows.Next() {
		var raw []byte
		if e = rows.Scan(&raw); e != nil {
			break
		}
		var v map[string]any
		if e = json.Unmarshal(raw, &v); e != nil {
			break
		}
		data, _ := v["data"].(map[string]any)
		v["compatible"] = validateDatabaseViewData(data, scope.Properties) == nil
		v["can_edit"] = str(v, "owner_id") == p.ID
		result = append(result, v)
	}
	rows.Close()
	if e == nil {
		e = rows.Err()
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	var defaultID string
	if e = tx.QueryRow(r.Context(), `SELECT coalesce((SELECT v.id::text FROM database_view_preferences pref JOIN database_views v ON v.id=pref.view_id WHERE pref.user_id=$1 AND pref.database_id=$2 AND v.database_id=$2 AND(v.owner_id=$1 OR v.visibility='workspace')),'')`, p.ID, id).Scan(&defaultID); e != nil {
		respond(w, nil, e)
		return
	}
	jsonResponse(w, 200, map[string]any{"views": result, "default_view_id": defaultID, "schema_fingerprint": scope.Schema})
}
func (s *Server) saveDatabaseView(w http.ResponseWriter, r *http.Request) {
	id, viewID, p := r.PathValue("id"), r.PathValue("viewID"), current(r)
	var in struct {
		Name       string         `json:"name"`
		Visibility string         `json:"visibility"`
		Data       map[string]any `json:"data"`
		Version    int            `json:"expected_version"`
		Consent    bool           `json:"share_consent"`
	}
	if decode(r, &in) != nil || strings.TrimSpace(in.Name) == "" || len(in.Name) > 200 || !oneOf(in.Visibility, "private", "workspace") || in.Visibility == "workspace" && !in.Consent {
		apiError(w, 400, "보기 이름·범위를 확인하세요. 팀 보기 저장은 별도 공유 동의가 필요합니다")
		return
	}
	creating := viewID == ""
	if !creating && (!validID(viewID) || in.Version < 1 || in.Version >= 2147483647) {
		apiError(w, 400, "보기 ID와 기준 버전을 확인하세요")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	scope, e := s.databaseEditingScopeTx(r, tx, id, in.Visibility == "workspace")
	if e != nil {
		apiError(w, 403, e.Error())
		return
	}
	if e = validateDatabaseViewData(in.Data, scope.Properties); e != nil {
		apiError(w, 400, e.Error())
		return
	}
	// Serialize each DB's quota/create path; existing view resource locks follow it.
	if _, e = tx.Exec(r.Context(), "SELECT pg_advisory_xact_lock(hashtextextended($1,23))", id); e != nil {
		respond(w, nil, e)
		return
	}
	if creating {
		var count int
		if e = tx.QueryRow(r.Context(), "SELECT count(*) FROM database_views WHERE database_id=$1 AND(owner_id=$2 OR visibility='workspace')", id, p.ID).Scan(&count); e != nil {
			respond(w, nil, e)
			return
		}
		if count >= 100 {
			apiError(w, 409, "현재 데이터베이스에서 접근 가능한 저장 보기는100개까지 지원합니다")
			return
		}
		viewID = newID()
	} else {
		var version int
		var oldVisibility string
		e = tx.QueryRow(r.Context(), "SELECT version,visibility FROM database_views WHERE id=$1 AND database_id=$2 AND owner_id=$3 FOR UPDATE", viewID, id, p.ID).Scan(&version, &oldVisibility)
		if e != nil {
			apiError(w, 404, "수정할 보기가 없거나 소유권이 없습니다")
			return
		}
		if version != in.Version {
			apiError(w, 409, "보기가 변경되었습니다. 현재 구성을 다시 확인하세요")
			return
		}
		if oldVisibility == "workspace" && in.Visibility == "private" {
			if _, e = s.databaseEditingScopeTx(r, tx, id, true); e != nil {
				apiError(w, 403, e.Error())
				return
			}
		}
	}
	protected, e := s.ProtectDocumentMetadataTx(r.Context(), tx, p, "", scope.WorkspaceID, map[string]any{"name": strings.TrimSpace(in.Name)})
	if e != nil {
		if !WriteProtectionError(w, e) {
			respond(w, nil, e)
		}
		return
	}
	metadata, _ := protected.Value.(map[string]any)
	name := str(metadata, "name")
	var raw []byte
	if creating {
		e = tx.QueryRow(r.Context(), "INSERT INTO database_views(id,database_id,owner_id,name,visibility,data)VALUES($1,$2,$3,$4,$5,$6)RETURNING to_jsonb(database_views)", viewID, id, p.ID, name, in.Visibility, jsonValue(in.Data)).Scan(&raw)
	} else {
		e = tx.QueryRow(r.Context(), "UPDATE database_views SET name=$1,visibility=$2,data=$3,version=version+1,updated_at=clock_timestamp()WHERE id=$4 RETURNING to_jsonb(database_views)", name, in.Visibility, jsonValue(in.Data), viewID).Scan(&raw)
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "DATABASE_VIEW_SAVE", viewID, map[string]any{"database_id": id, "visibility": in.Visibility})
	var value map[string]any
	e = json.Unmarshal(raw, &value)
	respond(w, value, e)
}
func (s *Server) deleteDatabaseView(w http.ResponseWriter, r *http.Request) {
	id, viewID, p := r.PathValue("id"), r.PathValue("viewID"), current(r)
	var in struct {
		Version int `json:"expected_version"`
	}
	if decode(r, &in) != nil || in.Version < 1 || !validID(viewID) {
		apiError(w, 400, "보기 ID와 기준 버전을 확인하세요")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	if _, e = s.databaseEditingScopeTx(r, tx, id, false); e != nil {
		apiError(w, 403, e.Error())
		return
	}
	var version int
	var visibility string
	e = tx.QueryRow(r.Context(), "SELECT version,visibility FROM database_views WHERE id=$1 AND database_id=$2 AND owner_id=$3 FOR UPDATE", viewID, id, p.ID).Scan(&version, &visibility)
	if e != nil {
		apiError(w, 404, "삭제할 보기가 없거나 소유권이 없습니다")
		return
	}
	if version != in.Version {
		apiError(w, 409, "보기가 변경되었습니다. 최신 구성을 확인하세요")
		return
	}
	if visibility == "workspace" {
		if _, e = s.databaseEditingScopeTx(r, tx, id, true); e != nil {
			apiError(w, 403, e.Error())
			return
		}
	}
	_, e = tx.Exec(r.Context(), "DELETE FROM database_views WHERE id=$1", viewID)
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "DATABASE_VIEW_DELETE", viewID, map[string]any{"database_id": id})
	jsonResponse(w, 200, map[string]any{"ok": true})
}
func (s *Server) saveDatabaseViewPreference(w http.ResponseWriter, r *http.Request) {
	id, p := r.PathValue("id"), current(r)
	var in struct {
		ViewID string `json:"view_id"`
	}
	if decode(r, &in) != nil || in.ViewID != "" && !validID(in.ViewID) {
		apiError(w, 400, "기본 보기 ID를 확인하세요")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	if _, e = s.databaseEditingScopeTx(r, tx, id, false); e != nil {
		apiError(w, 403, e.Error())
		return
	}
	if in.ViewID != "" {
		var found string
		e = tx.QueryRow(r.Context(), "SELECT id::text FROM database_views WHERE id=$1 AND database_id=$2 AND(owner_id=$3 OR visibility='workspace') FOR SHARE", in.ViewID, id, p.ID).Scan(&found)
		if e != nil {
			apiError(w, 404, "기본 보기로 사용할 수 없습니다")
			return
		}
	}
	_, e = tx.Exec(r.Context(), "INSERT INTO database_view_preferences(user_id,database_id,view_id)VALUES($1,$2,NULLIF($3,'')::uuid)ON CONFLICT(user_id,database_id)DO UPDATE SET view_id=excluded.view_id,updated_at=clock_timestamp()", p.ID, id, in.ViewID)
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	jsonResponse(w, 200, map[string]any{"default_view_id": in.ViewID})
}
