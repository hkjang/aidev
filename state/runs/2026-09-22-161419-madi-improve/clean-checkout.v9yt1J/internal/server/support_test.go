package server

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

func supportTestSetup(t *testing.T) (*Server, *integrationTestClient, *integrationTestClient, string, string, string) {
	t.Helper()
	s, c, wid, uid := agentTestSetup(t)
	if e := s.migrateSupport(t.Context()); e != nil {
		t.Fatal(e)
	}
	if !slices.Contains(s.apiRoutes, "GET /api/v1/support/meta") {
		s.registerSupport()
	}
	target := testJSONObject(t, c.request("POST", "/api/v1/admin/users", map[string]any{"email": "support-target@example.test", "name": "지원 대상", "password": "Support-Target-Password!", "role": "editor"}, 200))
	tid := str(target, "id")
	c.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": "support-target@example.test", "role": "editor"}, 200)
	other := newIntegrationTestClient(t, c.base)
	other.request("POST", "/api/v1/auth/login", map[string]any{"email": "support-target@example.test", "password": "Support-Target-Password!"}, 200)
	return s, c, other, wid, uid, tid
}
func supportTestStart(t *testing.T, c *integrationTestClient, tid string) string {
	t.Helper()
	v := testJSONObject(t, c.request("POST", "/api/v1/support/sessions", map[string]any{"target_id": tid, "reason": "문서 메뉴 접근 오류를 함께 확인합니다", "duration_minutes": 5}, 201))
	return str(v["session"].(map[string]any), "id")
}
func TestPostgresSupportACLIntersectionAndNoImpersonation(t *testing.T) {
	s, c, target, wid, uid, tid := supportTestSetup(t)
	c.request("POST", "/api/v1/support/sessions", map[string]any{"target_id": tid, "reason": "사유가 충분한 진단 시작 요청", "duration_minutes": 5}, 403)
	c.request("PUT", "/api/v1/admin/settings", map[string]any{"support_enabled": true, "support_operator_ids": []string{uid}, "support_max_minutes": 15}, 200)
	makeDoc := func(client *integrationTestClient, title, visibility string) map[string]any {
		return testJSONObject(t, client.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": title, "markdown": title + "_BODY", "visibility": visibility}, 200))
	}
	public := makeDoc(c, "공통 문서", "workspace")
	private := makeDoc(target, "TARGET_PRIVATE_SECRET", "private")
	operatorPrivate := makeDoc(c, "OPERATOR_PRIVATE_SECRET", "private")
	selected := makeDoc(c, "선택 공유", "selected")
	c.request("PUT", "/api/v1/documents/"+str(selected, "id")+"/shares", map[string]any{"user_id": tid, "permission": "read"}, 200)
	id := supportTestStart(t, c, tid)
	base := "/api/v1/support/sessions/" + id
	issued := testJSONObject(t, c.request("POST", "/api/v1/keys", map[string]any{"name": "진단 접근 금지 검증", "workspace_id": wid, "scopes": []string{"document:read", "ai:execute"}, "expires_in_days": 1, "rate_limit": 60}, 201))
	keyClient := newIntegrationTestClient(t, c.base)
	keyClient.token = str(issued, "token")
	keyClient.request("GET", base, nil, 403)
	keyClient.request("POST", "/api/v1/support/sessions", map[string]any{"target_id": tid, "reason": "키로 지원 권한 상승 시도 차단", "duration_minutes": 5}, 403)
	list := string(c.request("GET", base+"/documents?workspace_id="+wid, nil, 200))
	if strings.Contains(list, "PRIVATE_SECRET") || !strings.Contains(list, "공통 문서") || !strings.Contains(list, "선택 공유") {
		t.Fatalf("intersection list %s", list)
	}
	c.request("GET", base+"/documents/"+str(private, "id"), nil, 403)
	c.request("GET", base+"/documents/"+str(operatorPrivate, "id"), nil, 403)
	c.request("GET", base+"/documents/"+str(selected, "id"), nil, 200)
	c.request("GET", base+"?document_ids="+str(selected, "id"), nil, 200)
	for _, path := range []string{"/attachments/" + newID(), "/ai/conversations", "/profile", "/keys", "/auth/oidc/start"} {
		c.request("GET", base+path, nil, 403)
	}
	for _, method := range []string{"POST", "PUT", "PATCH", "DELETE"} {
		c.request(method, base+"/documents/"+str(public, "id"), map[string]any{"markdown": "ATTACK", "version": 1}, 403)
	}
	me := testJSONObject(t, c.request("GET", "/api/v1/auth/me", nil, 200))
	if str(me, "id") != uid {
		t.Fatal("support replaced original principal")
	}
	target.request("GET", base, nil, 403)
	// Even a different service-admin session cannot reuse a support ID.
	c.request("PUT", "/api/v1/documents/"+str(selected, "id")+"/shares", map[string]any{"user_id": tid, "permission": "remove"}, 200)
	c.request("GET", base+"/documents/"+str(selected, "id"), nil, 403)
	c.request("GET", base+"?document_ids="+str(selected, "id"), nil, 403)
	c.request("POST", base+"/end", nil, 200)
	c.request("GET", base+"/documents/"+str(public, "id"), nil, 403)
	var rows []map[string]any
	raw := c.request("GET", "/api/v1/admin/audit", nil, 200)
	if json.Unmarshal(raw, &rows) != nil {
		t.Fatal("audit JSON")
	}
	encoded := string(raw)
	for _, event := range []string{"SUPPORT_START", "SUPPORT_DOCUMENT_READ", "SUPPORT_FORBIDDEN", "SUPPORT_END"} {
		if !strings.Contains(encoded, event) {
			t.Fatalf("missing audit %s", event)
		}
	}
	if strings.Contains(encoded, "TARGET_PRIVATE_SECRET_BODY") {
		t.Fatal("private body leaked into support audit")
	}
	var body string
	s.DB.QueryRow(t.Context(), `SELECT markdown FROM documents WHERE id=$1`, str(public, "id")).Scan(&body)
	if body != "공통 문서_BODY" {
		t.Fatal("support modified source")
	}
}

func TestPostgresSupportOriginalCookieAndCurrentAccounts(t *testing.T) {
	s, c, _, wid, uid, tid := supportTestSetup(t)
	c.request("PUT", "/api/v1/admin/settings", map[string]any{"support_enabled": true, "support_operator_ids": []string{uid}}, 200)
	id := supportTestStart(t, c, tid)
	base := "/api/v1/support/sessions/" + id
	other := newIntegrationTestClient(t, c.base)
	other.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200)
	other.request("GET", base, nil, 403)
	other.request("POST", base+"/end", nil, 404)
	if _, e := s.DB.Exec(t.Context(), `UPDATE users SET disabled=true WHERE id=$1`, tid); e != nil {
		t.Fatal(e)
	}
	c.request("GET", base, nil, 403)
	if _, e := s.DB.Exec(t.Context(), `UPDATE users SET disabled=false WHERE id=$1`, tid); e != nil {
		t.Fatal(e)
	}
	c.request("GET", base, nil, 200)
	if _, e := s.DB.Exec(t.Context(), `DELETE FROM workspace_members WHERE workspace_id=$1 AND user_id=$2`, wid, tid); e != nil {
		t.Fatal(e)
	}
	c.request("GET", base, nil, 403)
	if _, e := s.DB.Exec(t.Context(), `INSERT INTO workspace_members(workspace_id,user_id,role) VALUES($1,$2,'viewer')`, wid, tid); e != nil {
		t.Fatal(e)
	}
	c.request("GET", base, nil, 200)
	c.request("POST", "/api/v1/auth/logout", nil, 200)
	c.request("GET", base, nil, 401)
	c.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200)
	c.request("GET", base, nil, 403)
}

func TestPostgresSupportDatabaseRelationsUseBothACLs(t *testing.T) {
	s, c, _, wid, uid, tid := supportTestSetup(t)
	c.request("PUT", "/api/v1/admin/settings", map[string]any{"support_enabled": true, "support_operator_ids": []string{uid}}, 200)
	db := testJSONObject(t, c.request("POST", "/api/v1/databases", map[string]any{"workspace_id": wid, "name": "공통 수량", "properties": []any{map[string]any{"id": "amount", "name": "수량", "type": "number"}}}, 200))
	dbID := str(db, "id")
	row := testJSONObject(t, c.request("POST", "/api/v1/databases/"+dbID+"/rows", map[string]any{"values": map[string]any{"amount": 4}}, 200))
	outside := testJSONObject(t, c.request("POST", "/api/v1/databases", map[string]any{"workspace_id": wid, "name": "참조 비공개", "properties": []any{map[string]any{"id": "secret", "name": "내부 숫자", "type": "number"}}}, 200))
	outsideID := str(outside, "id")
	otherRow := testJSONObject(t, c.request("POST", "/api/v1/databases/"+outsideID+"/rows", map[string]any{"values": map[string]any{"secret": 927351}}, 200))
	props := []any{map[string]any{"id": "amount", "name": "수량", "type": "number"}, map[string]any{"id": "double", "name": "두 배", "type": "formula", "expression": "prop(\"amount\") * 2"}, map[string]any{"id": "relation", "name": "참조", "type": "relation", "target_database_id": outsideID}, map[string]any{"id": "total", "name": "집계", "type": "rollup", "relation_property_id": "relation", "target_property_id": "secret", "aggregation": "sum"}}
	c.request("PUT", "/api/v1/databases/"+dbID, map[string]any{"name": "공통 수량", "properties": props}, 200)
	c.request("PUT", "/api/v1/databases/"+dbID+"/rows/"+str(row, "id"), map[string]any{"values": map[string]any{"amount": 4, "relation": []string{str(otherRow, "id")}}}, 200)
	id := supportTestStart(t, c, tid)
	base := "/api/v1/support/sessions/" + id
	body := string(c.request("GET", base+"/databases/"+dbID, nil, 200))
	if !strings.Contains(body, `"double":8`) || !strings.Contains(body, `"total":927351`) {
		t.Fatalf("authorized formula %s", body)
	}
	space := newID()
	if _, e := s.DB.Exec(t.Context(), `INSERT INTO spaces(id,workspace_id,name,slug,visibility,owner_id) VALUES($1,$2,'비공개','support-private','restricted',$3)`, space, wid, uid); e != nil {
		t.Fatal(e)
	}
	if _, e := s.DB.Exec(t.Context(), `INSERT INTO space_members(space_id,user_id,role) VALUES($1,$2,'admin')`, space, uid); e != nil {
		t.Fatal(e)
	}
	if _, e := s.DB.Exec(t.Context(), `UPDATE databases SET space_id=$1 WHERE id=$2`, space, outsideID); e != nil {
		t.Fatal(e)
	}
	c.request("GET", base+"?database_ids="+dbID+","+outsideID, nil, 403)
	body = string(c.request("GET", base+"/databases/"+dbID, nil, 200))
	if strings.Contains(body, "927351") || strings.Contains(body, str(otherRow, "id")) || !strings.Contains(body, `"double":8`) {
		t.Fatalf("hidden relation leaked %s", body)
	}
	c.request("GET", base+"/databases/"+outsideID, nil, 403)
	c.request("GET", base+"/databases/"+dbID+"?limit=101", nil, 400)
}
func TestPostgresSupportRevocationExpiryAndReasonProtection(t *testing.T) {
	s, c, _, _, uid, tid := supportTestSetup(t)
	c.request("PUT", "/api/v1/admin/settings", map[string]any{"support_enabled": true, "support_operator_ids": []string{uid}}, 200)
	id := supportTestStart(t, c, tid)
	c.request("PUT", "/api/v1/admin/settings", map[string]any{"support_operator_ids": []string{}}, 200)
	c.request("GET", "/api/v1/support/sessions/"+id, nil, 403)
	c.request("POST", "/api/v1/support/sessions/"+id+"/end", nil, 200)
	c.request("PUT", "/api/v1/admin/settings", map[string]any{"support_operator_ids": []string{uid}}, 200)
	id = supportTestStart(t, c, tid)
	if _, e := s.DB.Exec(t.Context(), `UPDATE support_sessions SET created_at=now()-interval '10 minutes',expires_at=now()-interval '1 minute' WHERE id=$1`, id); e != nil {
		t.Fatal(e)
	}
	c.request("GET", "/api/v1/support/sessions/"+id, nil, 403)
	if e := s.expireSupportSessions(t.Context()); e != nil {
		t.Fatal(e)
	}
	var count int
	s.DB.QueryRow(t.Context(), `SELECT count(*) FROM audit_logs WHERE action='SUPPORT_EXPIRED' AND resource=$1`, id).Scan(&count)
	if count != 1 {
		t.Fatal("missing terminal expiry audit")
	}
	if e := s.expireSupportSessions(t.Context()); e != nil {
		t.Fatal(e)
	}
	s.DB.QueryRow(t.Context(), `SELECT count(*) FROM audit_logs WHERE action='SUPPORT_EXPIRED' AND resource=$1`, id).Scan(&count)
	if count != 1 {
		t.Fatal("duplicate expiry audit")
	}
	if _, e := s.DB.Exec(t.Context(), `UPDATE protection_settings SET data=data||' {"enabled":true,"mode":"mask","detectors":["email"],"custom_terms":[]}'::jsonb WHERE id=1`); e != nil {
		t.Fatal(e)
	}
	v := testJSONObject(t, c.request("POST", "/api/v1/support/sessions", map[string]any{"target_id": tid, "reason": "문의 연락처 secret@example.test 를 포함한 지원 사유입니다", "duration_minutes": 5}, 201))
	if strings.Contains(string(jsonValue(v)), "secret@example.test") {
		t.Fatal("reason sensitive value not masked")
	}
	if !boolean(v, "reason_masked") {
		t.Fatal("mask notice missing")
	}
	if _, e := s.DB.Exec(t.Context(), `UPDATE protection_settings SET data=jsonb_set(data,'{mode}','"block"'::jsonb) WHERE id=1`); e != nil {
		t.Fatal(e)
	}
	c.request("POST", "/api/v1/support/sessions", map[string]any{"target_id": tid, "reason": "차단 대상 secret@example.test 이메일이 포함된 사유입니다", "duration_minutes": 5}, 422)
	var leaked int
	if e := s.DB.QueryRow(t.Context(), `SELECT count(*) FROM support_sessions WHERE reason LIKE '%secret@example.test%'`).Scan(&leaked); e != nil || leaked != 0 {
		t.Fatalf("blocked reason persisted: %d %v", leaked, e)
	}
}
