package server

import (
	"strings"
	"testing"
)

func TestPostgresEnterpriseCatalogEntityLinksAndACL(t *testing.T) {
	s, admin, ctx, p, wid := jobTestFixture(t)
	user := testJSONObject(t, admin.request("POST", "/api/v1/admin/users", map[string]any{"email": "entity-viewer@example.test", "name": "엔터티 조회자", "role": "viewer", "password": "Entity-viewer-password-2026!"}, 200))
	_ = user
	admin.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": "entity-viewer@example.test", "role": "viewer"}, 200)
	viewer := newIntegrationTestClient(t, admin.base)
	viewer.request("POST", "/api/v1/auth/login", map[string]any{"email": "entity-viewer@example.test", "password": "Entity-viewer-password-2026!"}, 200)
	entity := testJSONObject(t, admin.request("POST", "/api/v1/enterprise/entities", map[string]any{"workspace_id": wid, "title": "PostgreSQL 엔터티", "entity_type": "technology", "description": "운영 데이터베이스 기술"}, 200))
	eid := str(entity, "id")
	private := testJSONObject(t, admin.request("POST", "/api/v1/enterprise/entities", map[string]any{"workspace_id": wid, "title": "PRIVATE_ENTITY_SENTINEL", "entity_type": "project", "visibility": "private"}, 200))
	admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "실제 참조", "markdown": "[[PostgreSQL 엔터티]]"}, 200)
	admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "코드 예시 제외", "markdown": "```text\n[[PostgreSQL 엔터티]]\n```"}, 200)
	admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "PRIVATE_BACKLINK_SENTINEL", "markdown": "[[PostgreSQL 엔터티]]", "visibility": "private"}, 200)
	space := testJSONObject(t, admin.request("POST", "/api/v1/spaces", map[string]any{"workspace_id": wid, "name": "비밀 데이터", "slug": "secret-data", "visibility": "restricted"}, 200))
	cipher, e := s.encrypt("{}")
	if e != nil {
		t.Fatal(e)
	}
	for i, sid := range []string{"", str(space, "id")} {
		id := newID()
		name := "공개 소스"
		if i == 1 {
			name = "PRIVATE_SOURCE_SENTINEL"
		}
		_, e = s.DB.Exec(ctx, `INSERT INTO sql_sources(id,workspace_id,space_id,owner_id,service_account_id,name,kind,config,credentials_ciphertext) VALUES($1,$2,NULLIF($3,'')::uuid,$4,$4,$5,'postgres','{"tables":["public.orders"]}',$6)`, id, wid, sid, p.ID, name, cipher)
		if e != nil {
			t.Fatal(e)
		}
		_, e = s.DB.Exec(ctx, `INSERT INTO sql_source_tables(source_id,schema_name,table_name,columns,document_ids) VALUES($1,'public','orders','[{"name":"name","type":"text"}]',$2)`, id, jsonValue([]string{eid}))
		if e != nil {
			t.Fatal(e)
		}
	}
	result := testJSONObject(t, viewer.request("GET", "/api/v1/enterprise/entities/"+eid, nil, 200))
	if len(result["related_documents"].([]any)) != 1 || len(result["related_tables"].([]any)) != 1 || strings.Contains(string(jsonValue(result)), "PRIVATE_") {
		t.Fatalf("entity source ACL leak %#v", result)
	}
	viewer.request("GET", "/api/v1/enterprise/entities/"+str(private, "id"), nil, 404)
	viewer.request("POST", "/api/v1/enterprise/entities", map[string]any{"workspace_id": wid, "title": "권한 없음", "entity_type": "team"}, 403)
	search := viewer.request("GET", "/api/v1/enterprise/search?workspace_id="+wid, nil, 200)
	if strings.Contains(string(search), "PRIVATE_") {
		t.Fatal("catalog disclosed private resource")
	}
	admin.request("PUT", "/api/v1/enterprise/entities/"+eid, map[string]any{"entity_type": "database"}, 200)
	result = testJSONObject(t, viewer.request("GET", "/api/v1/enterprise/entities/"+eid, nil, 200))
	if str(result["metadata"].(map[string]any)["system_metadata"].(map[string]any), "entity_type") != "database" {
		t.Fatal("entity type not persisted")
	}
}
