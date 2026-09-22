package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPostgresAdvancedDatabaseFormulaRelationRollup(t *testing.T) {
	_, server := integrationTestServer(t)
	admin := newIntegrationTestClient(t, server.URL)
	admin.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200)
	var ws []map[string]any
	json.Unmarshal(admin.request("GET", "/api/v1/workspaces", nil, 200), &ws)
	wid := str(ws[0], "id")
	create := func(name string, props []any) map[string]any {
		return testJSONObject(t, admin.request("POST", "/api/v1/databases", map[string]any{"workspace_id": wid, "name": name, "properties": props}, 200))
	}
	target := create("항목", []any{map[string]any{"id": "title", "name": "이름", "type": "text"}, map[string]any{"id": "price", "name": "가격", "type": "number"}})
	targetID := str(target, "id")
	r1 := testJSONObject(t, admin.request("POST", "/api/v1/databases/"+targetID+"/rows", map[string]any{"values": map[string]any{"title": "하나", "price": 40}}, 200))
	r2 := testJSONObject(t, admin.request("POST", "/api/v1/databases/"+targetID+"/rows", map[string]any{"values": map[string]any{"title": "둘", "price": 60}}, 200))
	props := []any{map[string]any{"id": "title", "name": "이름", "type": "text"}, map[string]any{"id": "items", "name": "연결 항목", "type": "relation", "target_database_id": targetID}, map[string]any{"id": "total", "name": "합계", "type": "rollup", "relation_property_id": "items", "target_property_id": "price", "aggregation": "sum"}, map[string]any{"id": "tax", "name": "세금 포함", "type": "formula", "expression": `round(prop("total") * 1.1, 2)`}, map[string]any{"id": "created", "name": "생성 시간", "type": "created_time"}}
	source := create("주문", props)
	sourceID := str(source, "id")
	row := testJSONObject(t, admin.request("POST", "/api/v1/databases/"+sourceID+"/rows", map[string]any{"values": map[string]any{"title": "첫 주문", "items": []string{str(r1, "id"), str(r2, "id")}}}, 200))
	query := testJSONObject(t, admin.request("POST", "/api/v1/databases/"+sourceID+"/query", map[string]any{"filters": []any{map[string]any{"property_id": "tax", "operator": "gte", "value": 100}}, "sorts": []any{map[string]any{"property_id": "tax", "direction": "desc"}}}, 200))
	rows := query["rows"].([]any)
	if len(rows) != 1 {
		t.Fatalf("rows: %v", query)
	}
	computed := rows[0].(map[string]any)["computed_values"].(map[string]any)
	if computed["total"] != float64(100) || computed["tax"] != float64(110) || computed["created"] == nil {
		t.Fatalf("computed values: %#v", computed)
	}
	admin.request("PUT", "/api/v1/databases/"+sourceID+"/rows/"+str(row, "id"), map[string]any{"values": map[string]any{"tax": 999}}, 400)
	admin.request("PUT", "/api/v1/databases/"+sourceID+"/rows/"+str(row, "id"), map[string]any{"values": map[string]any{"items": []string{str(row, "id")}}}, 400)
	other := testJSONObject(t, admin.request("POST", "/api/v1/workspaces", map[string]any{"name": "격리된 공간"}, 200))
	outside := testJSONObject(t, admin.request("POST", "/api/v1/databases", map[string]any{"workspace_id": other["id"], "name": "외부"}, 200))
	bad := []any{map[string]any{"id": "rel", "name": "외부 관계", "type": "relation", "target_database_id": outside["id"]}}
	admin.request("PUT", "/api/v1/databases/"+sourceID, map[string]any{"properties": bad}, 400)
	admin.request("POST", "/api/v1/admin/users", map[string]any{"email": "viewer@example.test", "name": "조회자", "role": "viewer", "password": "Viewer-password-2026!"}, 200)
	admin.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": "viewer@example.test", "role": "viewer"}, 200)
	viewer := newIntegrationTestClient(t, server.URL)
	viewer.request("POST", "/api/v1/auth/login", map[string]any{"email": "viewer@example.test", "password": "Viewer-password-2026!"}, 200)
	viewer.request("POST", "/api/v1/databases/"+sourceID+"/query", map[string]any{}, 200)
	viewer.request("POST", "/api/v1/databases/"+sourceID+"/rows", map[string]any{"values": map[string]any{"title": "공격"}}, 403)
	viewer.request("POST", "/api/v1/databases/"+str(outside, "id")+"/query", map[string]any{}, 403)
	// A deleted relation target is rendered as a per-cell error instead of leaking rows or failing the page.
	admin.request("DELETE", "/api/v1/databases/"+targetID+"/rows/"+str(r1, "id"), nil, 200)
	query = testJSONObject(t, admin.request("POST", "/api/v1/databases/"+sourceID+"/query", map[string]any{}, 200))
	errors := query["rows"].([]any)[0].(map[string]any)["errors"].(map[string]any)
	if errors["total"] == nil {
		t.Fatal("missing relation should produce cell error")
	}
}

func TestPostgresAdvancedAIGenerationAndConcurrentChange(t *testing.T) {
	s, server := integrationTestServer(t)
	admin := newIntegrationTestClient(t, server.URL)
	admin.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200)
	var ws []map[string]any
	json.Unmarshal(admin.request("GET", "/api/v1/workspaces", nil, 200), &ws)
	props := []any{map[string]any{"id": "title", "name": "제목", "type": "text"}, map[string]any{"id": "summary", "name": "AI 요약", "type": "ai", "prompt": "한 문장으로 요약하세요", "source_property_ids": []string{"title"}}}
	db := testJSONObject(t, admin.request("POST", "/api/v1/databases", map[string]any{"workspace_id": ws[0]["id"], "name": "AI 속성", "properties": props}, 200))
	id := str(db, "id")
	row := testJSONObject(t, admin.request("POST", "/api/v1/databases/"+id+"/rows", map[string]any{"values": map[string]any{"title": "원본 내용"}}, 200))
	rid := str(row, "id")
	mutate := false
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		json.NewDecoder(r.Body).Decode(&payload)
		if payload["stream"] != true {
			t.Error("AI property must stream")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"검증된 요약\"}}]}\n\n")
		if mutate {
			s.DB.Exec(context.Background(), "UPDATE database_rows SET values=values||'{\"title\":\"다른 사용자의 수정\"}',updated_at=now() WHERE id=$1", rid)
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer provider.Close()
	admin.request("PUT", "/api/v1/admin/settings", map[string]any{"ai_enabled": true, "ai_base_url": provider.URL + "/v1", "ai_model": "test"}, 200)
	path := "/api/v1/databases/" + id + "/rows/" + rid + "/ai/summary"
	body := string(admin.request("POST", path, map[string]any{}, 200))
	if !strings.Contains(body, `"stored":true`) || !strings.Contains(body, "검증된 요약") {
		t.Fatalf("missing stored streamed value: %s", body)
	}
	mutate = true
	body = string(admin.request("POST", path, map[string]any{}, 200))
	if strings.Contains(body, `"stored":true`) || !strings.Contains(body, "행 내용이 변경") {
		t.Fatalf("concurrent edit overwritten: %s", body)
	}
}

func TestPostgresAdvancedRestrictedRelationAndButton(t *testing.T) {
	_, server := integrationTestServer(t)
	admin := newIntegrationTestClient(t, server.URL)
	admin.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200)
	var workspaces []map[string]any
	json.Unmarshal(admin.request("GET", "/api/v1/workspaces", nil, 200), &workspaces)
	wid := str(workspaces[0], "id")
	space := testJSONObject(t, admin.request("POST", "/api/v1/spaces", map[string]any{"workspace_id": wid, "name": "제한된 데이터", "visibility": "restricted"}, 200))
	target := testJSONObject(t, admin.request("POST", "/api/v1/databases", map[string]any{"workspace_id": wid, "space_id": space["id"], "name": "비공개 항목", "properties": []any{map[string]any{"id": "title", "name": "제목", "type": "text"}, map[string]any{"id": "price", "name": "금액", "type": "number"}, map[string]any{"id": "make", "name": "문서 생성", "type": "button", "action": "create_document", "template": "# {{제목}}\n금액: {{price}}"}}}, 200))
	targetID := str(target, "id")
	targetRow := testJSONObject(t, admin.request("POST", "/api/v1/databases/"+targetID+"/rows", map[string]any{"values": map[string]any{"title": "외부에 드러나면 안 되는 이름", "price": 100}}, 200))
	props := []any{map[string]any{"id": "relation", "name": "관계", "type": "relation", "target_database_id": targetID}, map[string]any{"id": "total", "name": "금액 합계", "type": "rollup", "relation_property_id": "relation", "target_property_id": "price", "aggregation": "sum"}}
	source := testJSONObject(t, admin.request("POST", "/api/v1/databases", map[string]any{"workspace_id": wid, "name": "공개 집계", "properties": props}, 200))
	sourceID := str(source, "id")
	admin.request("POST", "/api/v1/databases/"+sourceID+"/rows", map[string]any{"values": map[string]any{"relation": []string{str(targetRow, "id")}}}, 200)
	admin.request("POST", "/api/v1/admin/users", map[string]any{"email": "editor@example.test", "name": "편집자", "role": "editor", "password": "Editor-password-2026!"}, 200)
	admin.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": "editor@example.test", "role": "editor"}, 200)
	editor := newIntegrationTestClient(t, server.URL)
	editor.request("POST", "/api/v1/auth/login", map[string]any{"email": "editor@example.test", "password": "Editor-password-2026!"}, 200)
	result := string(editor.request("POST", "/api/v1/databases/"+sourceID+"/query", map[string]any{}, 200))
	if strings.Contains(result, "외부에 드러나면 안 되는 이름") || !strings.Contains(result, "접근 권한") {
		t.Fatalf("restricted relation leaked or missing cell error: %s", result)
	}
	editor.request("PUT", "/api/v1/databases/"+sourceID, map[string]any{"properties": props}, 400)
	editor.request("POST", "/api/v1/databases/"+sourceID+"/rows", map[string]any{"values": map[string]any{"relation": []string{str(targetRow, "id")}}}, 400)
	document := testJSONObject(t, admin.request("POST", "/api/v1/databases/"+targetID+"/rows/"+str(targetRow, "id")+"/button/make", map[string]any{}, 200))
	if document["space_id"] != space["id"] || !strings.Contains(str(document, "markdown"), "금액: 100") {
		t.Fatalf("button did not inherit space or template: %v", document)
	}
	editor.request("GET", "/api/v1/documents/"+str(document, "id"), nil, 404)
}

func TestPostgresAdvancedRelationCycleIsCellError(t *testing.T) {
	_, server := integrationTestServer(t)
	admin := newIntegrationTestClient(t, server.URL)
	admin.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200)
	var ws []map[string]any
	json.Unmarshal(admin.request("GET", "/api/v1/workspaces", nil, 200), &ws)
	db := testJSONObject(t, admin.request("POST", "/api/v1/databases", map[string]any{"workspace_id": ws[0]["id"], "name": "순환 집계", "properties": []any{map[string]any{"id": "title", "name": "이름", "type": "text"}}}, 200))
	id := str(db, "id")
	props := []any{map[string]any{"id": "title", "name": "이름", "type": "text"}, map[string]any{"id": "rel", "name": "관계", "type": "relation", "target_database_id": id}, map[string]any{"id": "roll", "name": "순환 합계", "type": "rollup", "relation_property_id": "rel", "target_property_id": "roll", "aggregation": "sum"}}
	admin.request("PUT", "/api/v1/databases/"+id, map[string]any{"properties": props}, 200)
	row := testJSONObject(t, admin.request("POST", "/api/v1/databases/"+id+"/rows", map[string]any{"values": map[string]any{"title": "자기 참조"}}, 200))
	admin.request("PUT", "/api/v1/databases/"+id+"/rows/"+str(row, "id"), map[string]any{"values": map[string]any{"title": "자기 참조", "rel": []string{str(row, "id")}}}, 200)
	result := string(admin.request("POST", "/api/v1/databases/"+id+"/query", map[string]any{}, 200))
	if !strings.Contains(result, "순환 참조") {
		t.Fatalf("cycle not detected: %s", result)
	}
}
