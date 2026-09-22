package server

import (
	"encoding/json"
	"testing"
)

func TestPostgresPropertyChangesPreserveValidRows(t *testing.T) {
	_, server := integrationTestServer(t)
	c := newIntegrationTestClient(t, server.URL)
	c.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200)
	var spaces []map[string]any
	json.Unmarshal(c.request("GET", "/api/v1/workspaces", nil, 200), &spaces)
	props := []any{map[string]any{"id": "title", "name": "제목", "type": "text"}, map[string]any{"id": "status", "name": "상태", "type": "select", "options": []string{"A", "B"}}}
	db := testJSONObject(t, c.request("POST", "/api/v1/databases", map[string]any{"workspace_id": spaces[0]["id"], "name": "속성 변경 검증", "properties": props}, 200))
	id := str(db, "id")
	returned := db["properties"].([]any)[0].(map[string]any)
	if _, exists := returned["options"]; !exists {
		t.Fatal("options must always serialize as an array for select UI")
	}
	row := testJSONObject(t, c.request("POST", "/api/v1/databases/"+id+"/rows", map[string]any{"values": map[string]any{"title": "원본", "status": "A"}}, 200))
	incompatible := []any{props[0], map[string]any{"id": "status", "name": "상태", "type": "multi_select", "options": []string{"A", "B"}}}
	c.request("PUT", "/api/v1/databases/"+id, map[string]any{"properties": incompatible}, 400)
	incompatible = []any{props[0], map[string]any{"id": "status", "name": "상태", "type": "select", "options": []string{"B"}}}
	c.request("PUT", "/api/v1/databases/"+id, map[string]any{"properties": incompatible}, 400)
	duplicate := []any{props[0], map[string]any{"id": "status", "name": "상태", "type": "select", "options": []string{"A", "A"}}}
	c.request("PUT", "/api/v1/databases/"+id, map[string]any{"properties": duplicate}, 400)
	unchanged := testJSONObject(t, c.request("GET", "/api/v1/databases/"+id, nil, 200))
	if str(unchanged["properties"].([]any)[1].(map[string]any), "type") != "select" {
		t.Fatal("invalid change must preserve existing schema")
	}
	c.request("PUT", "/api/v1/databases/"+id, map[string]any{"properties": []any{props[0]}}, 200)
	var rows []map[string]any
	json.Unmarshal(c.request("GET", "/api/v1/databases/"+id+"/rows", nil, 200), &rows)
	if _, exists := rows[0]["values"].(map[string]any)["status"]; exists {
		t.Fatal("deleted property value must not poison future updates")
	}
	c.request("PUT", "/api/v1/databases/"+id+"/rows/"+str(row, "id"), map[string]any{"values": map[string]any{"title": "수정"}}, 200)
}
