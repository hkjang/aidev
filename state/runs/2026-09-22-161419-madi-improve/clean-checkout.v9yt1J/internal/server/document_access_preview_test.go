package server

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestPostgresDocumentAccessPreviewTicketAndPlacement(t *testing.T) {
	s, owner, ctx, p, wid := jobTestFixture(t)
	create := func(title, visibility string) map[string]any {
		return testJSONObject(t, owner.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": title, "visibility": visibility, "markdown": "原文_SOURCE_NEVER_PREVIEW\n"}, 200))
	}
	source, parent := create("이동 원문", "private"), create("대상 위치", "workspace")
	id, pid := str(source, "id"), str(parent, "id")
	input := map[string]any{"expected_version": 1, "parent_id": pid, "visibility": "workspace"}
	preview := func() map[string]any {
		return testJSONObject(t, owner.request("POST", "/api/v1/documents/"+id+"/access-preview", input, 200))
	}
	check := preview()
	if check["potential_expansion"] != true || check["requires_confirmation"] != true || strings.Contains(string(jsonValue(check)), "SOURCE_NEVER_PREVIEW") {
		t.Fatal("preview risk/source projection", check)
	}
	if number(testJSONObject(t, owner.request("GET", "/api/v1/documents/"+id, nil, 200)), "version", 0) != 1 {
		t.Fatal("preview mutated source")
	}
	apply := func(ticket string, status int) {
		owner.request("PUT", "/api/v1/documents/"+id, map[string]any{"version": 1, "parent_id": pid, "visibility": "workspace", "access_preview_ticket": ticket}, status)
	}
	apply(str(check, "ticket")+"tamper", 409)
	owner.request("PUT", "/api/v1/documents/"+pid, map[string]any{"version": 1, "visibility": "private"}, 200)
	apply(str(check, "ticket"), 409)
	check = preview()
	user := testJSONObject(t, owner.request("POST", "/api/v1/admin/users", map[string]any{"email": "preview-share@example.test", "name": "공유 사용자", "role": "editor", "password": "Preview-Share-Password-2026!"}, 200))
	owner.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": user["email"], "role": "editor"}, 200)
	apply(str(check, "ticket"), 409) // Workspace membership/role ceiling changed.
	check = preview()
	owner.request("PUT", "/api/v1/documents/"+id+"/shares", map[string]any{"user_id": user["id"], "permission": "read"}, 200)
	apply(str(check, "ticket"), 409) // Existing dormant share can affect expansion.
	check = preview()
	plain, e := s.decrypt(str(check, "ticket"))
	if e != nil {
		t.Fatal(e)
	}
	var ticket documentAccessPreviewTicket
	if e = json.Unmarshal([]byte(plain), &ticket); e != nil {
		t.Fatal(e)
	}
	ticket.Expires = time.Now().Add(-time.Second).Unix()
	expired, e := s.encrypt(string(jsonValue(ticket)))
	if e != nil {
		t.Fatal(e)
	}
	apply(expired, 409)
	apply(str(check, "ticket"), 200)
	updated := testJSONObject(t, owner.request("GET", "/api/v1/documents/"+id, nil, 200))
	if number(updated, "version", 0) != 2 || str(updated, "parent_id") != pid || str(updated, "markdown") != str(source, "markdown") {
		t.Fatal("confirmed placement changed canonical contents", updated)
	}
	// Legacy callers without a ticket remain compatible but cannot bypass a
	// destination revoked after their original read/check.
	legacy := create("기존 호출 상한", "workspace")
	if _, e = s.DB.Exec(ctx, "UPDATE documents SET deleted_at=now() WHERE id=$1", pid); e != nil {
		t.Fatal(e)
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback(ctx)
	r := httptest.NewRequest("PUT", "/", nil).WithContext(context.WithValue(ctx, principalKey, p))
	base, _ := url.Parse(owner.base)
	for _, c := range owner.client.Jar.Cookies(base) {
		r.AddCookie(c)
	}
	if s.validateDocumentAccessChangeTx(r, tx, str(legacy, "id"), legacy, map[string]any{"version": 1}, pid, "", "workspace") == nil {
		t.Fatal("legacy placement accepted deleted destination")
	}
}
