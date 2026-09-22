package server

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPostgresDiscussionThreadsMentionsAndACL(t *testing.T) {
	_, ts := integrationTestServer(t)
	admin := newIntegrationTestClient(t, ts.URL)
	admin.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200)
	ws := testJSONObject(t, admin.request("POST", "/api/v1/workspaces", map[string]any{"name": "문서 토론 검증"}, 200))
	wid := str(ws, "id")
	createUser := func(email, role string) (*integrationTestClient, string) {
		u := testJSONObject(t, admin.request("POST", "/api/v1/admin/users", map[string]any{"email": email, "name": email, "role": "editor", "password": "Discussion-password-2026!"}, 200))
		admin.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": email, "role": role}, 200)
		c := newIntegrationTestClient(t, ts.URL)
		c.request("POST", "/api/v1/auth/login", map[string]any{"email": email, "password": "Discussion-password-2026!"}, 200)
		return c, str(u, "id")
	}
	author, authorID := createUser("discussion-author@example.test", "editor")
	reviewer, reviewerID := createUser("discussion-reviewer@example.test", "commenter")
	viewer, viewerID := createUser("discussion-viewer@example.test", "viewer")
	block := newID()
	doc := testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "검토 문서", "markdown": "# 운영 지식\n\n검토 대상 문장\n", "block_metadata": map[string]any{"blocks": []any{map[string]any{"id": block, "type": "paragraph", "text": "검토 대상 문장"}}}}, 200))
	did := str(doc, "id")
	base := "/api/v1/documents/" + did + "/comments"
	team := testJSONObject(t, admin.request("POST", "/api/v1/teams", map[string]any{"workspace_id": wid, "name": "운영팀"}, 200))
	tid := str(team, "id")
	admin.request("PUT", "/api/v1/teams/"+tid+"/members", map[string]any{"user_id": reviewerID}, 200)
	admin.request("PUT", "/api/v1/teams/"+tid+"/members", map[string]any{"user_id": viewerID}, 200)
	author.request("POST", "/api/v1/teams", map[string]any{"workspace_id": wid, "name": "비인가"}, 403)
	body := "@[운영팀](team:" + tid + ") @[검토 담당자](user:" + reviewerID + ") 확인 부탁합니다."
	comment := testJSONObject(t, author.request("POST", base, map[string]any{"body": body, "quote": "검토 대상 문장", "block_id": block, "document_version": 1, "assigned_to": reviewerID}, 200))
	cid := str(comment, "id")
	if !boolean(comment, "anchor_current") {
		t.Fatal(comment)
	}
	author.request("POST", base, map[string]any{"body": "오래된 인용", "quote": "검토 대상 문장", "document_version": 0}, 409)
	author.request("POST", base, map[string]any{"body": "없는 인용", "quote": "없는 문장", "document_version": 1}, 400)
	author.request("POST", base, map[string]any{"body": "없는 블록", "block_id": newID(), "document_version": 1}, 400)
	reviewer.request("PATCH", base+"/"+cid, map[string]any{"body": "타인 댓글 편집"}, 403)
	reviewer.request("PATCH", base+"/"+cid, map[string]any{"resolved": true}, 200)
	viewer.request("POST", base, map[string]any{"body": "조회자 댓글"}, 403)
	for range 2 {
		reviewer.request("POST", base+"/"+cid+"/reactions", map[string]any{"reaction": "thanks"}, 200)
	}
	var comments []map[string]any
	json.Unmarshal(author.request("GET", base+"?limit=1", nil, 200), &comments)
	reactions := comments[0]["reactions"].([]any)
	if len(reactions) != 1 || number(reactions[0].(map[string]any), "count", 0) != 1 {
		t.Fatal("reaction duplicate", reactions)
	}
	reply := testJSONObject(t, reviewer.request("POST", base, map[string]any{"body": "검토 완료했습니다.", "parent_id": cid}, 200))
	parent := str(reply, "id")
	for range 3 {
		v := testJSONObject(t, reviewer.request("POST", base, map[string]any{"body": "추가 답글", "parent_id": parent}, 200))
		parent = str(v, "id")
	}
	reviewer.request("POST", base, map[string]any{"body": "너무 깊은 답글", "parent_id": parent}, 400)
	json.Unmarshal(author.request("GET", base+"?limit=1&after="+cid, nil, 200), &comments)
	if len(comments) != 1 || str(comments[0], "id") != str(reply, "id") {
		t.Fatal("cursor", comments)
	}
	author.request("PATCH", base+"/"+cid, map[string]any{"body": "수정한 댓글"}, 200)
	reviewer.request("DELETE", base+"/"+cid, nil, 404)
	author.request("DELETE", base+"/"+cid, nil, 200)
	response := author.request("GET", base, nil, 200)
	if strings.Contains(string(response), "수정한 댓글") || !strings.Contains(string(response), "검토 완료했습니다.") {
		t.Fatal("deleted parent or surviving reply", string(response))
	}
	// Team mentions do not grant document access and notices disappear on revocation.
	notices := viewer.request("GET", "/api/v1/notifications", nil, 200)
	if !strings.Contains(string(notices), did) {
		t.Fatal("team mention missing")
	}
	admin.request("DELETE", "/api/v1/workspaces/"+wid+"/members/"+viewerID, nil, 200)
	notices = viewer.request("GET", "/api/v1/notifications", nil, 200)
	if strings.Contains(string(notices), did) {
		t.Fatal("revoked notification revealed document")
	}
	secret := testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "비공개 원문", "visibility": "private"}, 200))
	admin.request("POST", "/api/v1/documents/"+str(secret, "id")+"/comments", map[string]any{"body": "@[작성자](user:" + authorID + ") 비공개 언급"}, 200)
	notices = author.request("GET", "/api/v1/notifications", nil, 200)
	if strings.Contains(string(notices), str(secret, "id")) {
		t.Fatal("mention granted private read")
	}
	author.request("GET", "/api/v1/documents/"+str(secret, "id")+"/comments", nil, 403)
}
