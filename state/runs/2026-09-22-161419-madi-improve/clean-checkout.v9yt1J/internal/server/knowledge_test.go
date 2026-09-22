package server

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestPostgresKnowledgePolicyACLHistoryAndHold(t *testing.T) {
	s, ts := integrationTestServer(t)
	admin := newIntegrationTestClient(t, ts.URL)
	admin.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200)
	var ws []map[string]any
	json.Unmarshal(admin.request("GET", "/api/v1/workspaces", nil, 200), &ws)
	wid := str(ws[0], "id")
	u := testJSONObject(t, admin.request("POST", "/api/v1/admin/users", map[string]any{"email": "knowledge@example.test", "name": "문서 담당자", "password": "Knowledge-password-2026!", "role": "editor"}, 200))
	admin.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": u["email"], "role": "editor"}, 200)
	member := newIntegrationTestClient(t, ts.URL)
	member.request("POST", "/api/v1/auth/login", map[string]any{"email": u["email"], "password": "Knowledge-password-2026!"}, 200)
	doc := testJSONObject(t, member.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "운영 결정 기록", "markdown": "# 배경\n\n운영 정책과 [[접근 가능한 참고자료]] #정책\n"}, 200))
	id := str(doc, "id")
	path := "/api/v1/documents/" + id + "/knowledge"
	member.request("PUT", path, map[string]any{"version": 1, "legal_hold": true}, 403)
	member.request("PUT", path, map[string]any{"version": 1, "retain_until": time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339)}, 403)
	member.request("PUT", path, map[string]any{"version": 1, "review_period_days": 1.5}, 400)
	updated := testJSONObject(t, member.request("PUT", path, map[string]any{"version": 1, "kind": "decision", "classification": "confidential", "review_period_days": 30}, 200))
	if str(updated, "kind") != "decision" || number(updated, "version", 0) != 2 {
		t.Fatal(updated)
	}
	member.request("PUT", path, map[string]any{"version": 1, "kind": "meeting"}, 409)
	member.request("POST", "/api/v1/documents/"+id+"/reviewed", nil, 200)
	var history []map[string]any
	json.Unmarshal(member.request("GET", path+"/history", nil, 200), &history)
	if len(history) != 2 || str(history[0], "action") != "reviewed" {
		t.Fatal(history)
	}
	if _, e := s.DB.Exec(context.Background(), "UPDATE knowledge_document_meta SET last_reviewed_at=now()-interval '31 days' WHERE document_id=$1", id); e != nil {
		t.Fatal(e)
	}
	health := testJSONObject(t, member.request("GET", "/api/v1/knowledge/health?workspace_id="+wid, nil, 200))
	if number(health["counts"].(map[string]any), "stale", 0) < 1 {
		t.Fatal("stale not detected", health)
	}
	admin.request("PUT", path, map[string]any{"version": 2, "legal_hold": true, "retain_until": time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339)}, 200)
	member.request("DELETE", "/api/v1/documents/"+id, nil, 409)
	admin.request("DELETE", "/api/v1/documents/"+id, nil, 409)
	admin.request("PUT", path, map[string]any{"version": 3, "legal_hold": false}, 200)
	admin.request("DELETE", "/api/v1/documents/"+id, nil, 409)
	admin.request("PUT", path, map[string]any{"version": 4, "retain_until": nil}, 200)
	member.request("DELETE", "/api/v1/documents/"+id, nil, 200)
	member.request("POST", "/api/v1/documents/"+id+"/reviewed", nil, 404)
	// Metadata, history and quality never expose another user's private document.
	secret := testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "PRIVATE_KNOWLEDGE_SECRET", "visibility": "private", "markdown": "PRIVATE_KNOWLEDGE_SECRET"}, 200))
	secretID := str(secret, "id")
	member.request("GET", "/api/v1/documents/"+secretID+"/knowledge", nil, 404)
	member.request("GET", "/api/v1/documents/"+secretID+"/knowledge/history", nil, 404)
	if strings.Contains(string(member.request("GET", "/api/v1/knowledge/health?workspace_id="+wid, nil, 200)), "PRIVATE_KNOWLEDGE_SECRET") {
		t.Fatal("quality leaked private source")
	}
	// Explicit ownership handover grants the destination and revokes the source.
	admin.request("PUT", "/api/v1/documents/"+secretID+"/knowledge", map[string]any{"version": 1, "owner_id": u["id"]}, 200)
	admin.request("GET", "/api/v1/documents/"+secretID, nil, 404)
	member.request("GET", "/api/v1/documents/"+secretID, nil, 200)
	member.request("GET", "/api/v1/knowledge/analytics?workspace_id="+wid, nil, 403)
	admin.request("GET", "/api/v1/knowledge/analytics?workspace_id="+wid, nil, 200)
}

func TestKnowledgeQualityIsTransparentAndBounded(t *testing.T) {
	now := time.Now().UTC()
	d := map[string]any{"markdown": "# 배경\n\n## 결정\n" + strings.Repeat("검증된 운영 지식 ", 60) + " [[문서]] #태그", "updated_at": now.Format(time.RFC3339), "review_period_days": 90}
	q := documentQuality(d, indexMarkdown(str(d, "markdown")), nil, true, now)
	if q.Score != 100 || len(q.Flags) != 0 {
		t.Fatal(q)
	}
	d["owner_disabled"] = true
	d["updated_at"] = now.Add(-100 * 24 * time.Hour).Format(time.RFC3339)
	q = documentQuality(d, indexMarkdown(str(d, "markdown")), []string{"문서"}, false, now)
	if q.Score != 45 || len(q.Flags) != 4 {
		t.Fatal(q)
	}
}

func TestPostgresKnowledgeLifecycleOptInAndApproval(t *testing.T) {
	s, ts := integrationTestServer(t)
	ctx := context.Background()
	admin := newIntegrationTestClient(t, ts.URL)
	user := testJSONObject(t, admin.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200))
	ws := testJSONObject(t, admin.request("POST", "/api/v1/workspaces", map[string]any{"name": "문서 생명주기 검증"}, 200))
	wid := str(ws, "id")
	doc := testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "정기 운영 표준", "visibility": "private"}, 200))
	id := str(doc, "id")
	path := "/api/v1/documents/" + id + "/knowledge"
	admin.request("PUT", path, map[string]any{"version": 1, "status": "published"}, 200)
	_, e := s.DB.Exec(ctx, "UPDATE documents SET updated_at=now()-interval '91 days' WHERE id=$1", id)
	if e != nil {
		t.Fatal(e)
	}
	if n, e := s.runWorkspaceLifecycle(ctx, wid); e != nil || n != 0 {
		t.Fatalf("disabled lifecycle ran: %d %v", n, e)
	}
	admin.request("PUT", "/api/v1/workspaces/"+wid+"/settings", map[string]any{"version": 0, "data": map[string]any{"lifecycle_enabled": true, "review_period_days": 90}}, 200)
	if n, e := s.runWorkspaceLifecycle(ctx, wid); e != nil || n != 1 {
		t.Fatalf("lifecycle: %d %v", n, e)
	}
	if n, e := s.runWorkspaceLifecycle(ctx, wid); e != nil || n != 0 {
		t.Fatalf("duplicate lifecycle: %d %v", n, e)
	}
	next := testJSONObject(t, admin.request("GET", path, nil, 200))
	if str(next, "status") != "stale" || number(next, "version", 0) != 3 {
		t.Fatal(next)
	}
	health := testJSONObject(t, admin.request("GET", "/api/v1/knowledge/health?workspace_id="+wid, nil, 200))
	if number(health["counts"].(map[string]any), "stale", 0) != 1 {
		t.Fatal("stale marker lost review baseline", health)
	}
	var count int
	e = s.DB.QueryRow(ctx, "SELECT count(*) FROM notifications WHERE document_id=$1 AND user_id=$2", id, user["id"]).Scan(&count)
	if e != nil || count != 1 {
		t.Fatal(count, e)
	}
	admin.request("POST", "/api/v1/documents/"+id+"/reviewed", nil, 200)
	admin.request("PUT", path, map[string]any{"version": 3, "status": "published"}, 200)
	if n, e := s.runWorkspaceLifecycle(ctx, wid); e != nil || n != 0 {
		t.Fatalf("reviewed document became stale: %d %v", n, e)
	}
	admin.request("PUT", path, map[string]any{"version": 4, "status": "archived"}, 200)
	admin.request("PUT", "/api/v1/admin/settings", map[string]any{"approval_enabled": true}, 200)
	admin.request("PUT", path, map[string]any{"version": 5, "status": "published"}, 403)
	admin.request("PUT", path, map[string]any{"version": 5, "status": "draft"}, 200)
	// A disabled owner cannot be used as a workflow actor by the scheduler.
	_, e = s.DB.Exec(ctx, "UPDATE documents SET status='published',updated_at=now()-interval '100 days' WHERE id=$1;", id)
	if e != nil {
		t.Fatal(e)
	}
	_, e = s.DB.Exec(ctx, "UPDATE knowledge_document_meta SET last_reviewed_at=now()-interval '100 days' WHERE document_id=$1", id)
	if e != nil {
		t.Fatal(e)
	}
	_, e = s.DB.Exec(ctx, "UPDATE users SET disabled=true WHERE id=$1", user["id"])
	if e != nil {
		t.Fatal(e)
	}
	if n, e := s.runWorkspaceLifecycle(ctx, wid); e != nil || n != 0 {
		t.Fatalf("disabled owner executed lifecycle: %d %v", n, e)
	}
}
