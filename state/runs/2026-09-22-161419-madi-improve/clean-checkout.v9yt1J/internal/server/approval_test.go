package server

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresAdvancedApprovalWorkflow(t *testing.T) {
	s, server := integrationTestServer(t)
	admin := newIntegrationTestClient(t, server.URL)
	admin.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200)
	var workspaces []map[string]any
	json.Unmarshal(admin.request("GET", "/api/v1/workspaces", nil, 200), &workspaces)
	wid := str(workspaces[0], "id")
	clients := []*integrationTestClient{}
	uids := []string{}
	for _, name := range []string{"teamlead", "security", "director", "observer"} {
		u := testJSONObject(t, admin.request("POST", "/api/v1/admin/users", map[string]any{"email": name + "@example.test", "name": name, "password": "Approval-password-2026!", "role": "editor"}, 200))
		uids = append(uids, str(u, "id"))
		admin.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": name + "@example.test", "role": "viewer"}, 200)
		c := newIntegrationTestClient(t, server.URL)
		c.request("POST", "/api/v1/auth/login", map[string]any{"email": name + "@example.test", "password": "Approval-password-2026!"}, 200)
		clients = append(clients, c)
	}
	doc := testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "변경 승인", "markdown": "검토할 정확한 원본", "visibility": "workspace"}, 200))
	did := str(doc, "id")
	base := "/api/v1/documents/" + did
	admin.request("GET", base+"/approval", nil, 404)
	clients[0].request("GET", "/api/v1/approvals/inbox", nil, 404)
	admin.request("PUT", "/api/v1/admin/settings", map[string]any{"approval_enabled": true}, 200)
	team := newID()
	ctx := context.Background()
	if _, e := s.DB.Exec(ctx, `INSERT INTO teams(id,workspace_id,name) VALUES($1,$2,'운영 부서')`, team, wid); e != nil {
		t.Fatal(e)
	}
	if _, e := s.DB.Exec(ctx, `INSERT INTO team_members(team_id,user_id) VALUES($1,$2)`, team, uids[0]); e != nil {
		t.Fatal(e)
	}
	policy := map[string]any{"workspace_id": wid, "resource_kind": "document", "name": "부서 병렬 및 최종 승인", "enabled": true, "stages": []any{
		map[string]any{"name": "부서 검토", "mode": "all", "gates": []any{map[string]any{"name": "운영 부서", "kind": "team", "id": team}, map[string]any{"name": "보안 담당", "kind": "user", "id": uids[1]}}},
		map[string]any{"name": "최종 승인", "mode": "any", "gates": []any{map[string]any{"name": "지정 책임자", "kind": "user", "id": uids[2]}}},
	}}
	policy = testJSONObject(t, admin.request("POST", "/api/v1/admin/approval/policies", policy, 200))
	admin.request("POST", "/api/v1/admin/approval/policies", policy, 409)
	status := func(c *integrationTestClient) map[string]any {
		return testJSONObject(t, c.request("GET", base+"/approval", nil, 200))
	}
	request := func(c *integrationTestClient) map[string]any { return status(c)["request"].(map[string]any) }
	decision := func(c *integrationTestClient, request map[string]any, action, comment string, want int) {
		c.request("POST", base+"/approval", map[string]any{"action": action, "comment": comment, "request_id": request["id"], "request_version": request["version"]}, want)
	}
	admin.request("POST", base+"/approval", map[string]any{"action": "submit"}, 200)
	r := request(clients[0])
	if !boolean(testJSONObject(t, clients[0].request("GET", base, nil, 200)), "can_approve") {
		t.Fatal("read-only designated reviewer must be able to approve")
	}
	decision(admin, r, "approve", "", 403)
	decision(clients[2], r, "approve", "", 403)
	decision(clients[3], r, "approve", "", 403)
	clients[0].request("POST", base+"/approval", map[string]any{"action": "approve"}, 400)
	admin.request("POST", base+"/approval", map[string]any{"action": "submit"}, 409)
	// Revoked department membership immediately removes the assignment from
	// the current inbox and denies the otherwise valid immutable request.
	if _, e := s.DB.Exec(ctx, `DELETE FROM team_members WHERE team_id=$1 AND user_id=$2`, team, uids[0]); e != nil {
		t.Fatal(e)
	}
	decision(clients[0], r, "approve", "", 403)
	if string(clients[0].request("GET", "/api/v1/approvals/inbox", nil, 200)) != "[]\n" {
		t.Fatal("revoked team reviewer remained in inbox")
	}
	if _, e := s.DB.Exec(ctx, `INSERT INTO team_members(team_id,user_id) VALUES($1,$2)`, team, uids[0]); e != nil {
		t.Fatal(e)
	}
	decision(clients[0], r, "approve", "운영 확인", 200)
	decision(clients[1], r, "approve", "", 409)
	r = request(clients[1])
	decision(clients[1], r, "approve", "보안 확인", 200)
	r = request(clients[2])
	if number(r, "current_stage", 0) != 1 {
		t.Fatal("parallel gates did not advance sequential stage")
	}
	decision(clients[2], r, "reject", "", 400)
	decision(clients[2], r, "reject", "롤백 절차를 추가하세요", 200)
	if str(testJSONObject(t, admin.request("GET", base, nil, 200)), "status") != "rejected" {
		t.Fatal("rejection not reflected in document")
	}
	admin.request("POST", base+"/approval", map[string]any{"action": "submit"}, 200)
	r = request(clients[0])
	decision(clients[2], r, "approve", "", 403)
	// Source mutation supersedes the reviewed snapshot; request detail still
	// shows the exact old Markdown for audit, not the new source body.
	doc = testJSONObject(t, admin.request("GET", base, nil, 200))
	admin.request("PUT", base, map[string]any{"version": doc["version"], "markdown": "다른 원본"}, 200)
	decision(clients[0], r, "approve", "", 409)
	detail := testJSONObject(t, clients[0].request("GET", "/api/v1/approvals/requests/"+str(r, "id"), nil, 200))
	if !boolean(detail, "stale") || str(detail["snapshot"].(map[string]any), "markdown") != "검토할 정확한 원본" {
		t.Fatal("immutable review snapshot lost")
	}
	admin.request("POST", base+"/approval", map[string]any{"action": "submit"}, 200)
	r = request(clients[0])
	// Off/on must never resurrect prior approvals even with the same policy.
	admin.request("PUT", "/api/v1/admin/settings", map[string]any{"approval_enabled": false}, 200)
	clients[0].request("GET", "/api/v1/approvals/requests/"+str(r, "id"), nil, 404)
	admin.request("PUT", "/api/v1/admin/settings", map[string]any{"approval_enabled": true}, 200)
	decision(clients[0], r, "approve", "", 409)
	admin.request("POST", base+"/approval", map[string]any{"action": "submit"}, 200)
	r = request(clients[0])
	policy["name"] = "수정된 검토 정책"
	policy = testJSONObject(t, admin.request("PUT", "/api/v1/admin/approval/policies/"+str(policy, "id"), policy, 200))
	decision(clients[0], r, "approve", "", 409)
	admin.request("POST", base+"/approval", map[string]any{"action": "submit"}, 200)
	r = request(clients[0])
	// A key belonging to an assigned human may read the request, never approve.
	issued := testJSONObject(t, admin.request("POST", "/api/v1/keys", map[string]any{"user_id": uids[0], "name": "검토 읽기", "workspace_id": wid, "scopes": []string{"document:read", "document:write"}}, 201))
	key := newIntegrationTestClient(t, server.URL)
	key.token = str(issued, "token")
	key.request("GET", base+"/approval", nil, 200)
	decision(key, r, "approve", "", 403)
	// Small pool plus concurrent decisions catches use of the outer pool from
	// inside a locked transaction. Only one same-version decision can commit.
	config := s.DB.Config()
	config.MaxConns = 4
	pool, e := pgxpool.NewWithConfig(ctx, config)
	if e != nil {
		t.Fatal(e)
	}
	original := s.DB
	s.DB = pool
	var wg sync.WaitGroup
	codes := make(chan int, 6)
	for range 6 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			payload := string(jsonValue(map[string]any{"action": "approve", "request_id": r["id"], "request_version": r["version"]}))
			req, _ := http.NewRequest("POST", server.URL+base+"/approval", strings.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Madi-Request", "1")
			res, e := clients[0].client.Do(req)
			if e != nil {
				codes <- 0
				return
			}
			io.Copy(io.Discard, res.Body)
			res.Body.Close()
			codes <- res.StatusCode
		}()
	}
	wg.Wait()
	close(codes)
	s.DB = original
	pool.Close()
	won := 0
	for code := range codes {
		if code == 200 {
			won++
		} else if code != 409 {
			t.Fatalf("concurrent decision HTTP %d", code)
		}
	}
	if won != 1 {
		t.Fatalf("committed decisions %d", won)
	}
	r = request(clients[1])
	decision(clients[1], r, "approve", "", 200)
	r = request(clients[2])
	decision(clients[2], r, "approve", "", 200)
	doc = testJSONObject(t, admin.request("GET", base, nil, 200))
	if str(doc, "status") != "published" {
		t.Fatal("completed workflow did not publish")
	}
	// Visibility revocation blocks both snapshot and assignment/history reads.
	admin.request("PUT", base, map[string]any{"version": doc["version"], "visibility": "private"}, 200)
	clients[0].request("GET", "/api/v1/approvals/requests/"+str(r, "id"), nil, 404)
	clients[0].request("GET", base+"/approval", nil, 404)
	if str(testJSONObject(t, admin.request("GET", base, nil, 200)), "status") != "draft" {
		t.Fatal("snapshot metadata mutation remained published")
	}
}
