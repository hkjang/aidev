package server

import (
	"strings"
	"testing"
)

func TestPostgresDocumentAccessRequestsPrivacyAndAtomicGrant(t *testing.T) {
	s, ts := integrationTestServer(t)
	owner := newIntegrationTestClient(t, ts.URL)
	owner.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200)
	wid := str(testJSONObject(t, owner.request("POST", "/api/v1/workspaces", map[string]any{"name": "문서 접근"}, 200)), "id")
	u := testJSONObject(t, owner.request("POST", "/api/v1/admin/users", map[string]any{"email": "requester@example.test", "name": "요청 사용자", "role": "editor", "password": "Access-Request-Password-2026!"}, 200))
	owner.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": u["email"], "role": "editor"}, 200)
	requester := newIntegrationTestClient(t, ts.URL)
	requester.request("POST", "/api/v1/auth/login", map[string]any{"email": u["email"], "password": "Access-Request-Password-2026!"}, 200)
	doc := testJSONObject(t, owner.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "PRIVATE_TITLE_NOT_FOR_REQUESTER", "markdown": "private body", "visibility": "private"}, 200))
	id := str(doc, "id")
	input := map[string]any{"workspace_id": wid, "permission": "read", "reason": "공동 작업을 위한 열람 요청"}
	valid := requester.request("POST", "/api/v1/documents/"+id+"/access-requests", input, 202)
	unknown := newID()
	missing := requester.request("POST", "/api/v1/documents/"+unknown+"/access-requests", input, 202)
	if string(valid) != string(missing) || strings.Contains(string(valid), id) {
		t.Fatal("generic response exposed target existence")
	}
	for i := 0; i < 3; i++ {
		requester.request("POST", "/api/v1/documents/"+id+"/access-requests", input, 202)
	}
	var count int
	if e := s.DB.QueryRow(t.Context(), "SELECT count(*) FROM document_access_requests WHERE requester_id=$1", u["id"]).Scan(&count); e != nil || count != 2 {
		t.Fatal("duplicate request or missing indistinguishable receipt", count, e)
	}
	mineRaw := requester.request("GET", "/api/v1/access-requests?workspace_id="+wid, nil, 200)
	if strings.Contains(string(mineRaw), str(doc, "title")) || strings.Contains(string(mineRaw), "private body") {
		t.Fatal("requester list leaked private source")
	}
	mine := testJSONObject(t, mineRaw)["mine"].([]any)
	if len(mine) != 2 {
		t.Fatal("missing-target receipt oracle")
	}
	list := testJSONObject(t, owner.request("GET", "/api/v1/access-requests?workspace_id="+wid, nil, 200))["incoming"].([]any)
	if len(list) != 1 {
		t.Fatal("unknown target delivered to owner", list)
	}
	req := list[0].(map[string]any)
	rid := str(req, "id")
	if str(req, "reason") != input["reason"] {
		t.Fatal("request reason did not roundtrip")
	}
	var sealed string
	if e := s.DB.QueryRow(t.Context(), "SELECT reason_ciphertext FROM document_access_requests WHERE id=$1", rid).Scan(&sealed); e != nil || strings.Contains(sealed, "공동 작업") {
		t.Fatal("request reason not encrypted", e)
	}
	requester.request("PUT", "/api/v1/access-requests/"+rid, map[string]any{"revision": 1, "action": "grant", "expected_document_version": 1}, 403)
	owner.request("PUT", "/api/v1/access-requests/"+rid, map[string]any{"revision": 1, "action": "grant", "expected_document_version": 9}, 409)
	owner.request("PUT", "/api/v1/access-requests/"+rid, map[string]any{"revision": 1, "action": "grant", "expected_document_version": 1}, 200)
	granted := testJSONObject(t, requester.request("GET", "/api/v1/documents/"+id, nil, 200))
	if str(granted, "visibility") != "selected" || number(granted, "version", 0) != 2 || granted["can_write"] != false {
		t.Fatal("read grant exceeded authority", granted)
	}
	owner.request("PUT", "/api/v1/access-requests/"+rid, map[string]any{"revision": 1, "action": "grant", "expected_document_version": 2}, 409)
	parent := testJSONObject(t, owner.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "상위 제한", "visibility": "private", "markdown": "private parent"}, 200))
	child := testJSONObject(t, owner.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "parent_id": parent["id"], "title": "하위 제한", "visibility": "private", "markdown": "private child"}, 200))
	cid := str(child, "id")
	requester.request("POST", "/api/v1/documents/"+cid+"/access-requests", input, 202)
	list = testJSONObject(t, owner.request("GET", "/api/v1/access-requests?workspace_id="+wid, nil, 200))["incoming"].([]any)
	var blocked map[string]any
	for _, item := range list {
		row := item.(map[string]any)
		if str(row, "document_id") == cid {
			blocked = row
		}
	}
	owner.request("PUT", "/api/v1/access-requests/"+str(blocked, "id"), map[string]any{"revision": 1, "action": "grant", "expected_document_version": 1}, 409)
	unchanged := testJSONObject(t, owner.request("GET", "/api/v1/documents/"+cid, nil, 200))
	if str(unchanged, "visibility") != "private" || number(unchanged, "version", 0) != 1 {
		t.Fatal("failed inherited grant partially changed source")
	}
	if e := s.DB.QueryRow(t.Context(), "SELECT count(*) FROM document_shares WHERE document_id=$1", cid).Scan(&count); e != nil || count != 0 {
		t.Fatal("failed inherited grant leaked share", count, e)
	}
	owner.request("PUT", "/api/v1/access-requests/"+str(blocked, "id"), map[string]any{"revision": 1, "action": "reject", "expected_document_version": 1}, 200)
	// A rejected request cannot create new owner notifications during cooldown.
	requester.request("POST", "/api/v1/documents/"+cid+"/access-requests", input, 202)
	if e := s.DB.QueryRow(t.Context(), "SELECT count(*) FROM document_access_requests WHERE requester_id=$1 AND requested_document_id=$2", u["id"], cid).Scan(&count); e != nil || count != 1 {
		t.Fatal("same-target cooldown missing", count, e)
	}
	for _, item := range mine {
		row := item.(map[string]any)
		if str(row, "document_id") == unknown {
			requester.request("PUT", "/api/v1/access-requests/"+str(row, "id"), map[string]any{"revision": 1, "action": "cancel"}, 200)
		}
	}
}
