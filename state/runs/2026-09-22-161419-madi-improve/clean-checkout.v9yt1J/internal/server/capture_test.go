package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

func TestPostgresCapturePrivateClassifyAndScopes(t *testing.T) {
	s, ts := integrationTestServer(t)
	ctx := context.Background()
	admin := newIntegrationTestClient(t, ts.URL)
	admin.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200)
	ws := testJSONObject(t, admin.request("POST", "/api/v1/workspaces", map[string]any{"name": "수집함 검증"}, 200))
	wid := str(ws, "id")
	u := testJSONObject(t, admin.request("POST", "/api/v1/admin/users", map[string]any{"name": "수집 담당자", "email": "capture@example.test", "role": "editor", "password": "Capture-password-2026!"}, 200))
	admin.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": u["email"], "role": "editor"}, 200)
	member := newIntegrationTestClient(t, ts.URL)
	member.request("POST", "/api/v1/auth/login", map[string]any{"email": u["email"], "password": "Capture-password-2026!"}, 200)
	var calls atomic.Int32
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.Write([]byte("must not fetch")) }))
	defer source.Close()
	for _, bad := range []string{"javascript:alert(1)", "file:///etc/passwd", "https://user:secret@example.test", "https://example.test/\ncontent"} {
		member.request("POST", "/api/v1/captures", map[string]any{"workspace_id": wid, "url": bad}, 400)
	}
	doc := testJSONObject(t, member.request("POST", "/api/v1/captures", map[string]any{"workspace_id": wid, "text": "# 개인 수집 원문\n\n아직 분류하지 않은 생각", "url": source.URL + "/private-resource", "tags": []string{"아이디어"}}, 200))
	id := str(doc, "id")
	if str(doc, "visibility") != "private" || !strings.Contains(str(doc, "markdown"), "출처: "+source.URL) || calls.Load() != 0 {
		t.Fatal("capture must be private and never fetch a URL", doc, calls.Load())
	}
	admin.request("GET", "/api/v1/documents/"+id, nil, 404)
	if strings.Contains(string(admin.request("GET", "/api/v1/captures?workspace_id="+wid, nil, 200)), id) {
		t.Fatal("administrator bypassed private inbox")
	}
	var list []map[string]any
	json.Unmarshal(member.request("GET", "/api/v1/captures?workspace_id="+wid, nil, 200), &list)
	if len(list) != 1 || str(list[0], "source_url") != source.URL+"/private-resource" {
		t.Fatal(list)
	}
	meta := testJSONObject(t, member.request("GET", "/api/v1/documents/"+id+"/knowledge", nil, 200))
	if str(meta, "kind") != "inbox" {
		t.Fatal(meta)
	}
	var count int
	if e := s.DB.QueryRow(ctx, "SELECT count(*) FROM automation_events WHERE resource_id=$1 AND type='document.created'", id).Scan(&count); e != nil || count != 1 {
		t.Fatal("capture transactional outbox", count, e)
	}
	space := testJSONObject(t, admin.request("POST", "/api/v1/spaces", map[string]any{"workspace_id": wid, "name": "제한 공간", "visibility": "restricted"}, 200))
	path := "/api/v1/captures/" + id + "/classify"
	member.request("POST", path, map[string]any{"version": 1, "space_id": space["id"], "visibility": "workspace", "kind": "note"}, 403)
	member.request("POST", path, map[string]any{"version": 0, "visibility": "workspace", "kind": "note"}, 409)
	result := testJSONObject(t, member.request("POST", path, map[string]any{"version": 1, "visibility": "workspace", "kind": "note"}, 200))
	if number(result, "version", 0) != 2 || str(result, "visibility") != "workspace" {
		t.Fatal(result)
	}
	admin.request("GET", "/api/v1/documents/"+id, nil, 200)
	if strings.Contains(string(member.request("GET", "/api/v1/captures?workspace_id="+wid, nil, 200)), id) {
		t.Fatal("classified document stayed in inbox")
	}
	member.request("POST", path, map[string]any{"version": 2, "visibility": "workspace", "kind": "note"}, 404)
	key := testJSONObject(t, member.request("POST", "/api/v1/keys", map[string]any{"name": "수집 읽기 전용", "workspace_id": wid, "scopes": []string{"document:read"}, "expires_in_days": 7}, 201))
	reader := newIntegrationTestClient(t, ts.URL)
	reader.token = str(key, "token")
	reader.request("GET", "/api/v1/captures?workspace_id="+wid, nil, 200)
	reader.request("POST", "/api/v1/captures", map[string]any{"workspace_id": wid, "text": "scope denied"}, 403)
	key = testJSONObject(t, member.request("POST", "/api/v1/keys", map[string]any{"name": "클리퍼 쓰기", "workspace_id": wid, "scopes": []string{"document:write"}, "expires_in_days": 7}, 201))
	reader.token = str(key, "token")
	reader.request("POST", "/api/v1/captures", map[string]any{"workspace_id": wid, "text": strings.Repeat("😀", 180)}, 200)
	other := testJSONObject(t, admin.request("POST", "/api/v1/workspaces", map[string]any{"name": "키 범위 외부"}, 200))
	reader.request("POST", "/api/v1/captures", map[string]any{"workspace_id": other["id"], "text": "scope denied"}, 403)
}

func TestPostgresCaptureAndAttachmentIdempotency(t *testing.T) {
	s, ts := integrationTestServer(t)
	c := newIntegrationTestClient(t, ts.URL)
	c.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200)
	c.request("PUT", "/api/v1/admin/settings", map[string]any{"storage_path": t.TempDir()}, 200)
	w := testJSONObject(t, c.request("POST", "/api/v1/workspaces", map[string]any{"name": "수집 중복 방지"}, 200))
	input := map[string]any{"workspace_id": w["id"], "title": "연결 단절 재시도", "text": "중복 없는 수집", "client_request_id": newID()}
	var wg sync.WaitGroup
	results := make(chan map[string]any, 6)
	failures := make(chan string, 6)
	for range 6 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req, _ := http.NewRequest("POST", ts.URL+"/api/v1/captures", bytes.NewReader(jsonValue(input)))
			req.Header.Set("X-Madi-Request", "1")
			req.Header.Set("Content-Type", "application/json")
			res, e := c.client.Do(req)
			if e != nil {
				failures <- e.Error()
				return
			}
			defer res.Body.Close()
			raw, _ := io.ReadAll(res.Body)
			if res.StatusCode != 200 {
				failures <- string(raw)
				return
			}
			var d map[string]any
			json.Unmarshal(raw, &d)
			results <- d
		}()
	}
	wg.Wait()
	close(results)
	close(failures)
	for err := range failures {
		t.Fatal(err)
	}
	id := ""
	for d := range results {
		if id != "" && id != str(d, "id") {
			t.Fatal("duplicate document", id, d)
		}
		id = str(d, "id")
	}
	input["text"] = "다른 내용"
	c.request("POST", "/api/v1/captures", input, 409)
	input["text"] = "중복 없는 수집"
	upload := func(requestID, content string, want int) map[string]any {
		t.Helper()
		var body bytes.Buffer
		mp := multipart.NewWriter(&body)
		f, _ := mp.CreateFormFile("file", "운영 자료.txt")
		f.Write([]byte(content))
		mp.Close()
		req, _ := http.NewRequest("POST", ts.URL+"/api/v1/attachments?document_id="+id+"&client_request_id="+requestID, &body)
		req.Header.Set("X-Madi-Request", "1")
		req.Header.Set("Content-Type", mp.FormDataContentType())
		res, e := c.client.Do(req)
		if e != nil {
			t.Fatal(e)
		}
		defer res.Body.Close()
		raw, _ := io.ReadAll(res.Body)
		if res.StatusCode != want {
			t.Fatalf("upload: %d %s", res.StatusCode, raw)
		}
		return testJSONObject(t, raw)
	}
	requestID := newID()
	a := upload(requestID, "첨부파일 내용", 200)
	b := upload(strings.ToUpper(requestID), "첨부파일 내용", 200)
	if a["id"] != b["id"] {
		t.Fatal("duplicate attachment", a, b)
	}
	upload(requestID, "다른 첨부 내용", 409)
	var attachments []map[string]any
	json.Unmarshal(c.request("GET", "/api/v1/documents/"+id+"/attachments", nil, 200), &attachments)
	if len(attachments) != 1 {
		t.Fatal(attachments)
	}
	var n int
	if e := s.DB.QueryRow(context.Background(), "SELECT count(*) FROM automation_events WHERE resource_id=$1 AND type='document.created'", id).Scan(&n); e != nil || n != 1 {
		t.Fatal("duplicate event", n, e)
	}
	c.request("DELETE", "/api/v1/documents/"+id, nil, 200)
	c.request("POST", "/api/v1/captures", input, 410)
}

func TestPostgresNumberedMarkdownTaskUpdate(t *testing.T) {
	_, ts := integrationTestServer(t)
	c := newIntegrationTestClient(t, ts.URL)
	c.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200)
	ws := testJSONObject(t, c.request("POST", "/api/v1/workspaces", map[string]any{"name": "번호 할 일"}, 200))
	d := testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": ws["id"], "title": "번호 목록", "markdown": "1. [ ] 첫 번째 점검\n2. [ ] 두 번째 점검\n\n```md\n- [ ] 예시만\n```\n"}, 200))
	updated := testJSONObject(t, c.request("PUT", "/api/v1/tasks", map[string]any{"document_id": d["id"], "version": 1, "line": 0, "done": true}, 200))
	if !strings.HasPrefix(str(updated, "markdown"), "1. [x] 첫 번째 점검") {
		t.Fatal(updated)
	}
	c.request("PUT", "/api/v1/tasks", map[string]any{"document_id": d["id"], "version": 2, "line": 4, "done": true}, 409)
}
