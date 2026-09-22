package server

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPostgresDatabaseEditingAtomicCASAndValidation(t *testing.T) {
	s, admin, ctx, _, wid := jobTestFixture(t)
	props := []any{map[string]any{"id": "title", "name": "이름", "type": "text"}, map[string]any{"id": "amount", "name": "수량", "type": "number"}, map[string]any{"id": "state", "name": "상태", "type": "select", "options": []any{"대기", "완료"}}}
	db := testJSONObject(t, admin.request("POST", "/api/v1/databases", map[string]any{"workspace_id": wid, "name": "원자 셀 편집", "properties": props}, 200))
	id := str(db, "id")
	first := testJSONObject(t, admin.request("POST", "/api/v1/databases/"+id+"/rows", map[string]any{"values": map[string]any{"title": "첫째", "amount": 1, "state": "대기"}}, 200))
	second := testJSONObject(t, admin.request("POST", "/api/v1/databases/"+id+"/rows", map[string]any{"values": map[string]any{"title": "둘째", "amount": 2, "state": "대기"}}, 200))
	if number(first, "version", 0) != 1 || number(second, "version", 0) != 1 {
		t.Fatal("new row version missing", first, second)
	}
	metadata := testJSONObject(t, admin.request("GET", "/api/v1/databases/"+id+"/editing", nil, 200))
	schema := str(metadata, "schema_fingerprint")
	body := map[string]any{"schema_fingerprint": schema, "cells": []any{map[string]any{"row_id": first["id"], "property_id": "amount", "expected_version": 1, "value": 10}, map[string]any{"row_id": second["id"], "property_id": "state", "expected_version": 1, "value": "잘못된선택"}}, "commit": false}
	preview := testJSONObject(t, admin.request("POST", "/api/v1/databases/"+id+"/edit-cells", body, 200))
	if boolean(preview, "valid") || boolean(preview, "committed") || len(preview["errors"].([]any)) == 0 {
		t.Fatal("invalid cell preview accepted", preview)
	}
	body["commit"], body["consent"] = true, true
	invalid := testJSONObject(t, admin.request("POST", "/api/v1/databases/"+id+"/edit-cells", body, 200))
	if boolean(invalid, "committed") {
		t.Fatal("invalid batch partially committed")
	}
	var amount, version int
	if e := s.DB.QueryRow(ctx, "SELECT (values->>'amount')::int,version FROM database_rows WHERE id=$1", first["id"]).Scan(&amount, &version); e != nil || amount != 1 || version != 1 {
		t.Fatal("invalid batch changed earlier cell", amount, version, e)
	}
	cells := body["cells"].([]any)
	cells[1].(map[string]any)["value"] = "완료"
	// Failure injected after an actual row mutation rolls every row and version back.
	if _, e := s.DB.Exec(ctx, "CREATE FUNCTION madi_cells_fixture_fail()RETURNS trigger LANGUAGE plpgsql AS $$BEGIN RAISE EXCEPTION 'fixture cells rollback';END$$;CREATE TRIGGER madi_cells_fixture_fail BEFORE INSERT ON automation_events FOR EACH ROW WHEN(NEW.type='database.row.updated')EXECUTE FUNCTION madi_cells_fixture_fail()"); e != nil {
		t.Fatal(e)
	}
	admin.request("POST", "/api/v1/databases/"+id+"/edit-cells", body, 500)
	if e := s.DB.QueryRow(ctx, "SELECT (values->>'amount')::int,version FROM database_rows WHERE id=$1", first["id"]).Scan(&amount, &version); e != nil || amount != 1 || version != 1 {
		t.Fatal("failed event did not roll back", amount, version, e)
	}
	if _, e := s.DB.Exec(ctx, "DROP TRIGGER madi_cells_fixture_fail ON automation_events"); e != nil {
		t.Fatal(e)
	}
	result := testJSONObject(t, admin.request("POST", "/api/v1/databases/"+id+"/edit-cells", body, 200))
	if !boolean(result, "committed") || len(result["rows"].([]any)) != 2 {
		t.Fatal("valid atomic batch missing", result)
	}
	admin.request("POST", "/api/v1/databases/"+id+"/edit-cells", body, 409)
	for _, bad := range []any{nil, 0, -1, 1.5, "2", 2147483648.0} {
		admin.request("PUT", "/api/v1/databases/"+id+"/rows/"+str(first, "id"), map[string]any{"values": map[string]any{"amount": 50}, "expected_version": bad}, 400)
	}
	admin.request("PUT", "/api/v1/databases/"+id+"/rows/"+str(first, "id"), map[string]any{"values": map[string]any{"amount": 50}, "expected_version": 1}, 409)
	saved := testJSONObject(t, admin.request("PUT", "/api/v1/databases/"+id+"/rows/"+str(first, "id"), map[string]any{"values": map[string]any{"amount": 50}, "expected_version": 2}, 200))
	if number(saved, "version", 0) != 3 {
		t.Fatal("CAS update version", saved)
	}
	legacy := testJSONObject(t, admin.request("PUT", "/api/v1/databases/"+id+"/rows/"+str(first, "id"), map[string]any{"values": map[string]any{"amount": 51}}, 200))
	if number(legacy, "version", 0) != 4 {
		t.Fatal("legacy update version", legacy)
	}
	admin.request("DELETE", "/api/v1/databases/"+id+"/rows/"+str(first, "id"), nil, 200)
	admin.request("PUT", "/api/v1/databases/"+id+"/rows/"+str(first, "id"), map[string]any{"values": map[string]any{"amount": 90}, "expected_version": 4}, 404)
	recreated := testJSONObject(t, admin.request("POST", "/api/v1/databases/"+id+"/rows", map[string]any{"id": first["id"], "values": map[string]any{"title": "새 행"}}, 200))
	if recreated["id"] == first["id"] || number(recreated, "version", 0) != 1 {
		t.Fatal("retired row ID reused", recreated)
	}
	// Changing the schema invalidates the old range preview even with current row CAS.
	cells[0].(map[string]any)["row_id"], cells[0].(map[string]any)["expected_version"] = recreated["id"], 1
	cells[1].(map[string]any)["expected_version"] = 2
	props[0].(map[string]any)["name"] = "새 이름"
	admin.request("PUT", "/api/v1/databases/"+id, map[string]any{"properties": props}, 200)
	admin.request("POST", "/api/v1/databases/"+id+"/edit-cells", body, 409)
	// SQL revision updates from other code paths are also versioned, and overflow is atomic.
	if _, e := s.DB.Exec(ctx, "UPDATE database_rows SET values=values||'{\"amount\":99}'::jsonb WHERE id=$1", second["id"]); e != nil {
		t.Fatal(e)
	}
	if e := s.DB.QueryRow(ctx, "SELECT version FROM database_rows WHERE id=$1", second["id"]).Scan(&version); e != nil || version != 3 {
		t.Fatal("non-REST value change bypassed version", version, e)
	}
	maxID := newID()
	if _, e := s.DB.Exec(ctx, "INSERT INTO database_rows(id,database_id,values,version)VALUES($1,$2,'{}',2147483647)", maxID, id); e != nil {
		t.Fatal(e)
	}
	admin.request("PUT", "/api/v1/databases/"+id+"/rows/"+maxID, map[string]any{"values": map[string]any{"title": "overflow"}, "expected_version": 2147483647}, 500)
	if e := s.DB.QueryRow(ctx, "SELECT version FROM database_rows WHERE id=$1", maxID).Scan(&version); e != nil || version != 2147483647 {
		t.Fatal("overflow changed revision")
	}
}

func TestDatabaseEditingCellAndViewValidation(t *testing.T) {
	var cell databaseEditCell
	if json.Unmarshal([]byte(`{"row_id":"x","property_id":"a","expected_version":1}`), &cell) == nil {
		t.Fatal("missing cell value silently cleared")
	}
	props := []map[string]any{{"id": "title", "type": "text"}, {"id": "when", "type": "date"}, {"id": "state", "type": "select"}}
	valid := map[string]any{"view": "board", "board_property_id": "state", "columns": []any{"title", "state"}, "filters": []any{}, "sorts": []any{}}
	if e := validateDatabaseViewData(valid, props); e != nil {
		t.Fatal(e)
	}
	for _, bad := range []map[string]any{{"view": "javascript"}, {"view": "table", "columns": []any{"title", "title"}}, {"view": "table", "filters": nil}, {"view": "board", "board_property_id": "when"}, {"view": "table", "sorts": []any{map[string]any{"property_id": "hidden", "direction": "asc"}}}, {"view": "table", "custom_css": strings.Repeat("x", 10)}} {
		if validateDatabaseViewData(bad, props) == nil {
			t.Fatal("invalid view accepted", bad)
		}
	}
}
