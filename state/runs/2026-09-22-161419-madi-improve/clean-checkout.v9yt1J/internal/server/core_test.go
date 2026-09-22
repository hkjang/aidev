package server

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestPostgresDocumentLifecycleAndPermissions(t *testing.T) {
	_, server := integrationTestServer(t)
	admin := newIntegrationTestClient(t, server.URL)
	admin.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200)
	var workspaces []map[string]any
	json.Unmarshal(admin.request("GET", "/api/v1/workspaces", nil, 200), &workspaces)
	wid := str(workspaces[0], "id")
	viewerUser := testJSONObject(t, admin.request("POST", "/api/v1/admin/users", map[string]any{"email": "viewer@example.test", "name": "읽기 사용자", "password": "Viewer-test-password!", "role": "viewer"}, 200))
	editorUser := testJSONObject(t, admin.request("POST", "/api/v1/admin/users", map[string]any{"email": "editor@example.test", "name": "팀장", "password": "Editor-test-password!", "role": "editor"}, 200))
	admin.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": "viewer@example.test", "role": "viewer"}, 200)
	admin.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": "editor@example.test", "role": "admin"}, 200)
	viewer := newIntegrationTestClient(t, server.URL)
	viewer.request("POST", "/api/v1/auth/login", map[string]any{"email": "viewer@example.test", "password": "Viewer-test-password!"}, 200)
	editor := newIntegrationTestClient(t, server.URL)
	editor.request("POST", "/api/v1/auth/login", map[string]any{"email": "editor@example.test", "password": "Editor-test-password!"}, 200)
	create := func(title, md, visibility string) map[string]any {
		return testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": title, "markdown": md, "visibility": visibility}, 200))
	}
	private := create("비밀 문서", "CLASSIFIED_BODY", "private")
	pid := str(private, "id")
	viewer.request("GET", "/api/v1/documents/"+pid, nil, 404)
	viewer.request("GET", "/api/v1/documents/"+pid+"/versions", nil, 403)
	if strings.Contains(string(viewer.request("GET", "/api/v1/documents?q=CLASSIFIED", nil, 200)), "CLASSIFIED") {
		t.Fatal("private search leak")
	}
	public := create("운영 지침", "---\ntags: [운영, 서버]\n---\n# 운영 지침\n\n- [ ] 점검\n", "workspace")
	did := str(public, "id")
	viewer.request("PUT", "/api/v1/documents/"+did, map[string]any{"title": "수정 공격", "version": 1}, 403)
	viewer.request("POST", "/api/v1/documents/"+did+"/comments", map[string]any{"body": "unauthorized"}, 403)
	viewer.request("GET", "/api/v1/admin/stats", nil, 403)
	selected := create("선택 공유", "SHARED_ONLY", "selected")
	sid := str(selected, "id")
	viewer.request("GET", "/api/v1/documents/"+sid, nil, 404)
	admin.request("PUT", "/api/v1/documents/"+sid+"/shares", map[string]any{"user_id": viewerUser["id"], "permission": "read"}, 200)
	viewer.request("GET", "/api/v1/documents/"+sid, nil, 200)
	admin.request("PUT", "/api/v1/documents/"+sid+"/shares", map[string]any{"user_id": viewerUser["id"], "permission": "remove"}, 200)
	viewer.request("GET", "/api/v1/documents/"+sid, nil, 404)
	// Exactly one writer wins a simultaneous version update.
	var wg sync.WaitGroup
	statuses := make(chan int, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, _ := http.NewRequest("PUT", server.URL+"/api/v1/documents/"+did, strings.NewReader(`{"title":"운영 표준","version":1}`))
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("X-Madi-Request", "1")
			res, e := admin.client.Do(r)
			if e != nil {
				statuses <- 0
				return
			}
			defer res.Body.Close()
			io.Copy(io.Discard, res.Body)
			statuses <- res.StatusCode
		}()
	}
	wg.Wait()
	close(statuses)
	success, conflict := 0, 0
	for status := range statuses {
		if status == 200 {
			success++
		}
		if status == 409 {
			conflict++
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatalf("optimistic concurrency success=%d conflict=%d", success, conflict)
	}
	currentDoc := testJSONObject(t, admin.request("GET", "/api/v1/documents/"+did, nil, 200))
	if !oneOf("운영 지침", listStrings(currentDoc["aliases"])...) {
		t.Fatal("rename alias lost")
	}
	source := create("링크 문서", "[[운영 지침]]", "workspace")
	_ = source
	if !strings.Contains(string(admin.request("GET", "/api/v1/documents/"+did+"/backlinks", nil, 200)), "링크 문서") {
		t.Fatal("alias backlinks broken")
	}
	admin.request("POST", "/api/v1/documents/"+did+"/versions/1/restore", map[string]any{"expected_version": currentDoc["version"]}, 200)
	restored := testJSONObject(t, admin.request("GET", "/api/v1/documents/"+did, nil, 200))
	if str(restored, "title") != "운영 지침" || number(restored, "version", 0) != 3 {
		t.Fatalf("restore failed: %v", restored)
	}
	admin.request("PUT", "/api/v1/documents/"+did, map[string]any{"version": 3, "parent_id": did}, 400)
	admin.request("POST", "/api/v1/documents/"+did+"/approval", map[string]any{"action": "submit"}, 404)
	admin.request("PUT", "/api/v1/admin/settings", map[string]any{"approval_enabled": true}, 200)
	admin.request("PUT", "/api/v1/documents/"+did, map[string]any{"version": 3, "status": "published"}, 403)
	admin.request("POST", "/api/v1/documents/"+did+"/approval", map[string]any{"action": "submit"}, 200)
	approvalStatus := testJSONObject(t, editor.request("GET", "/api/v1/documents/"+did+"/approval", nil, 200))
	approvalRequest := approvalStatus["request"].(map[string]any)
	decision := map[string]any{"action": "approve", "request_id": approvalRequest["id"], "request_version": approvalRequest["version"]}
	admin.request("POST", "/api/v1/documents/"+did+"/approval", decision, 403)
	editor.request("POST", "/api/v1/documents/"+did+"/approval", decision, 200)
	admin.request("DELETE", "/api/v1/documents/"+did, nil, 200)
	if strings.Contains(string(admin.request("GET", "/api/v1/documents?workspace_id="+wid, nil, 200)), `"id":"`+did+`"`) {
		t.Fatal("deleted doc listed")
	}
	admin.request("POST", "/api/v1/documents/"+did+"/restore", nil, 200)
	admin.request("PUT", "/api/v1/admin/users/"+str(editorUser, "id"), map[string]any{"disabled": true}, 200)
	editor.request("GET", "/api/v1/auth/me", nil, 401)
}

func TestHTTPStatusMetricsPreserveStreamingAndRedactPanics(t *testing.T) {
	s := &Server{mux: http.NewServeMux()}
	s.mux.HandleFunc("GET /ok", func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, "ok") })
	s.mux.HandleFunc("GET /forbidden", func(w http.ResponseWriter, r *http.Request) { apiError(w, 403, "denied") })
	s.mux.HandleFunc("GET /panic", func(w http.ResponseWriter, r *http.Request) { panic("private-panic-payload") })
	s.mux.HandleFunc("GET /stream", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: first\n\n")
		if err := http.NewResponseController(w).Flush(); err != nil {
			t.Error(err)
		}
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
	})
	var logs bytes.Buffer
	oldLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
	defer slog.SetDefault(oldLogger)
	for _, path := range []string{"/ok", "/forbidden", "/panic", "/stream"} {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if path == "/stream" && (!w.Flushed || !strings.Contains(w.Body.String(), "[DONE]")) {
			t.Fatal("response accounting broke SSE flush")
		}
	}
	if s.requests.Load() != 4 || s.errors.Load() != 2 || s.responseClasses[2].Load() != 2 || s.responseClasses[4].Load() != 1 || s.responseClasses[5].Load() != 1 {
		t.Fatal("HTTP status counters do not match responses")
	}
	if strings.Contains(logs.String(), "private-panic-payload") || !strings.Contains(logs.String(), `"status":403`) {
		t.Fatal("panic redaction/status logging failed")
	}
}

func TestLoginLimiterBoundedConcurrentAttempts(t *testing.T) {
	s := &Server{}
	now := time.Now()
	var workers sync.WaitGroup
	for worker := range 32 {
		workers.Add(1)
		go func(worker int) {
			defer workers.Done()
			for i := range 400 {
				s.allowLoginAttempt(fmt.Sprintf("client-%d-%d", worker, i), fmt.Sprintf("user-%d@example.test", i), now)
			}
		}(worker)
	}
	workers.Wait()
	s.limiterMu.Lock()
	count := len(s.attempts)
	s.limiterMu.Unlock()
	if count > maxLoginLimiterEntries {
		t.Fatalf("login limiter grew beyond bound: %d", count)
	}
	if !s.allowLoginAttempt("after-expiration", "user@example.test", now.Add(16*time.Minute)) {
		t.Fatal("expired limiter entries were not reclaimed")
	}
	limited := &Server{}
	for i := range 15 {
		if !limited.allowLoginAttempt("same-ip", "user@example.test", now) {
			t.Fatalf("too early rate limit at %d", i)
		}
	}
	if limited.allowLoginAttempt("same-ip", " USER@example.test ", now) {
		t.Fatal("whitespace/case bypassed account limiter")
	}
}

func TestHTTPLogsUseMatchedRouteAndNeverPrivateDriverMessage(t *testing.T) {
	var logs bytes.Buffer
	old := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
	defer slog.SetDefault(old)
	s := &Server{mux: http.NewServeMux()}
	s.mux.HandleFunc("GET /records/{id}", func(w http.ResponseWriter, r *http.Request) { respond(w, nil, fmt.Errorf("PRIVATE_DRIVER_PAYLOAD")) })
	s.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/records/PRIVATE_RESOURCE_ID?secret=PRIVATE_QUERY", nil))
	s.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/UNKNOWN_PRIVATE_PATH", nil))
	for _, private := range []string{"PRIVATE_DRIVER_PAYLOAD", "PRIVATE_RESOURCE_ID", "PRIVATE_QUERY", "UNKNOWN_PRIVATE_PATH"} {
		if strings.Contains(logs.String(), private) {
			t.Fatal("private request data entered logs")
		}
	}
	if !strings.Contains(logs.String(), `"route":"GET /records/{id}"`) || !strings.Contains(logs.String(), `"route":"unmatched"`) {
		t.Fatal("matched template missing", logs.String())
	}
}

func TestPostgresSettingsConcurrencyAuditHistoryAndServiceAccounts(t *testing.T) {
	s, server := integrationTestServer(t)
	admin := newIntegrationTestClient(t, server.URL)
	u := testJSONObject(t, admin.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200))
	uid := str(u, "id")
	var attribution int
	if err := s.DB.QueryRow(context.Background(), `SELECT count(*) FROM audit_logs WHERE action='LOGIN' AND user_id=$1 AND resource=$1::text`, uid).Scan(&attribution); err != nil || attribution < 1 {
		t.Fatalf("login audit lost verified user: %d %v", attribution, err)
	}
	public := testJSONObject(t, admin.request("GET", "/api/v1/public", nil, 200))
	if str(public, "reviewer_role") != "admin" {
		t.Fatal("public metadata lacks reviewer policy")
	}
	var workspaces []map[string]any
	_ = json.Unmarshal(admin.request("GET", "/api/v1/workspaces", nil, 200), &workspaces)
	wid := str(workspaces[0], "id")
	doc := testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "댓글 조인 검증"}, 200))
	admin.request("POST", "/api/v1/documents/"+str(doc, "id")+"/comments", map[string]any{"body": "댓글 테스트"}, 200)
	if !strings.Contains(string(admin.request("GET", "/api/v1/documents/"+str(doc, "id")+"/comments", nil, 200)), "댓글 테스트") {
		t.Fatal("comment query lost row")
	}
	patches := []map[string]any{{"site_name": "concurrent-madi"}, {"session_hours": 12}, {"trash_retention_days": 21}, {"default_key_days": 45}, {"ai_max_tokens": 2048}, {"reviewer_role": "editor"}}
	var wg sync.WaitGroup
	results := make(chan string, len(patches))
	start := make(chan struct{})
	for _, patch := range patches {
		wg.Add(1)
		go func(patch map[string]any) {
			defer wg.Done()
			<-start
			payload, _ := json.Marshal(patch)
			r, _ := http.NewRequest("PUT", server.URL+"/api/v1/admin/settings", bytes.NewReader(payload))
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("X-Madi-Request", "1")
			response, err := admin.client.Do(r)
			if err != nil {
				results <- err.Error()
				return
			}
			defer response.Body.Close()
			body, _ := io.ReadAll(response.Body)
			if response.StatusCode != 200 {
				results <- fmt.Sprintf("%d %s", response.StatusCode, body)
			}
		}(patch)
	}
	close(start)
	wg.Wait()
	close(results)
	for result := range results {
		t.Error(result)
	}
	cfg := testJSONObject(t, admin.request("GET", "/api/v1/admin/settings", nil, 200))
	for _, patch := range patches {
		for key, value := range patch {
			expected, _ := json.Marshal(value)
			actual, _ := json.Marshal(cfg[key])
			if string(expected) != string(actual) {
				t.Fatalf("concurrent partial update lost %s: %s want %s", key, actual, expected)
			}
		}
	}
	var history []map[string]any
	if err := json.Unmarshal(admin.request("GET", "/api/v1/admin/settings/history", nil, 200), &history); err != nil || len(history) != len(patches) {
		t.Fatalf("history JOIN/rows failed: %v %v", history, err)
	}
	admin.request("POST", "/api/v1/admin/settings/history/"+str(history[len(history)-1], "id")+"/restore", nil, 200)
	for _, patch := range []map[string]any{{"ai_model": []string{"invalid"}}, {"ai_api_key_clear": []string{"invalid"}}, {"site_url": "https://example.test/unsupported-prefix"}} {
		admin.request("PUT", "/api/v1/admin/settings", patch, 400)
	}
	badHistory := newID()
	if _, err := s.DB.Exec(context.Background(), `INSERT INTO settings_history(id,user_id,data) VALUES($1,$2,'{"site_name":7}')`, badHistory, uid); err != nil {
		t.Fatal(err)
	}
	admin.request("POST", "/api/v1/admin/settings/history/"+badHistory+"/restore", nil, 400)
	service := testJSONObject(t, admin.request("POST", "/api/v1/admin/users", map[string]any{"email": "connector@example.test", "name": "연동 계정", "role": "editor", "kind": "service"}, 200))
	admin.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": "connector@example.test", "role": "editor"}, 200)
	issued := testJSONObject(t, admin.request("POST", "/api/v1/keys", map[string]any{"user_id": service["id"], "name": "서비스 연동", "workspace_id": wid, "scopes": []string{"document:read"}}, 201))
	serviceClient := newIntegrationTestClient(t, server.URL)
	serviceClient.request("POST", "/api/v1/auth/login", map[string]any{"email": "connector@example.test", "password": "not-a-user-password"}, 401)
	serviceClient.token = str(issued, "token")
	serviceClient.request("GET", "/api/v1/documents/"+str(doc, "id"), nil, 200)
	if !strings.Contains(string(admin.request("GET", "/api/v1/keys?user_id="+str(service, "id"), nil, 200)), "서비스 연동") {
		t.Fatal("admin delegated key list failed")
	}
	admin.request("PUT", "/api/v1/admin/users/"+str(service, "id"), map[string]any{"disabled": true}, 200)
	serviceClient.request("GET", "/api/v1/documents", nil, 401)
	editor := testJSONObject(t, admin.request("POST", "/api/v1/admin/users", map[string]any{"email": "manager@example.test", "name": "워크스페이스 관리자", "password": "Manager-password-for-test!", "role": "editor"}, 200))
	admin.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": "manager@example.test", "role": "admin"}, 200)
	manager := newIntegrationTestClient(t, server.URL)
	manager.request("POST", "/api/v1/auth/login", map[string]any{"email": "manager@example.test", "password": "Manager-password-for-test!"}, 200)
	admin.request("PUT", "/api/v1/admin/users/"+str(editor, "id"), map[string]any{"role": "viewer"}, 200)
	manager.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": "admin@example.test", "role": "viewer"}, 403)
}

func TestPostgresDatabaseSettingsCSRFAndBackup(t *testing.T) {
	_, server := integrationTestServer(t)
	c := newIntegrationTestClient(t, server.URL)
	user := testJSONObject(t, c.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200))
	var workspaces []map[string]any
	json.Unmarshal(c.request("GET", "/api/v1/workspaces", nil, 200), &workspaces)
	wid := str(workspaces[0], "id")
	props := []any{map[string]any{"id": "title", "name": "제목", "type": "text"}, map[string]any{"id": "status", "name": "상태", "type": "select", "options": []string{"대기", "완료"}}, map[string]any{"id": "tags", "name": "태그", "type": "multi_select", "options": []string{"A", "B"}}, map[string]any{"id": "amount", "name": "수량", "type": "number"}}
	db := testJSONObject(t, c.request("POST", "/api/v1/databases", map[string]any{"workspace_id": wid, "name": "프로젝트", "properties": props}, 200))
	id := str(db, "id")
	row := testJSONObject(t, c.request("POST", "/api/v1/databases/"+id+"/rows", map[string]any{"values": map[string]any{"title": "테스트", "status": "대기", "tags": []string{"A", "B"}, "amount": 12}}, 200))
	c.request("PUT", "/api/v1/databases/"+id+"/rows/"+str(row, "id"), map[string]any{"values": map[string]any{"status": "완료"}}, 200)
	c.request("POST", "/api/v1/databases/"+id+"/rows", map[string]any{"values": map[string]any{"status": "invalid"}}, 400)
	c.request("POST", "/api/v1/databases/"+id+"/rows", map[string]any{"values": map[string]any{"amount": "not number"}}, 400)
	for _, v := range []any{"document:read", []any{12}, []any{"bogus"}} {
		c.request("PUT", "/api/v1/admin/settings", map[string]any{"allowed_key_scopes": v}, 400)
	}
	c.request("PUT", "/api/v1/admin/settings", map[string]any{"ai_max_tokens": 262145}, 400)
	c.request("PUT", "/api/v1/admin/settings", map[string]any{"ai_max_tokens": 1.5}, 400)
	c.request("PUT", "/api/v1/admin/settings", map[string]any{"ai_api_key": []string{"secret"}}, 400)
	c.request("PUT", "/api/v1/admin/settings", map[string]any{"ai_max_tokens": 262144, "ai_base_url": "https://azure.example/openai/deployments/m/chat/completions?api-version=2024-10-21"}, 200)
	c.request("PUT", "/api/v1/admin/users/"+str(user, "id"), map[string]any{"disabled": true}, 400)
	for _, test := range []struct {
		header, value string
		status        int
	}{{"Origin", "https://attacker.example", 403}, {"Authorization", "Basic attacker", 401}, {"X-Madi-Request", "", 403}} {
		r, _ := http.NewRequest("PUT", server.URL+"/api/v1/profile", strings.NewReader(`{"name":"bad"}`))
		r.Header.Set("X-Madi-Request", "1")
		r.Header.Set(test.header, test.value)
		res, e := c.client.Do(r)
		if e != nil {
			t.Fatal(e)
		}
		io.Copy(io.Discard, res.Body)
		res.Body.Close()
		if res.StatusCode != test.status {
			t.Fatalf("%s validation: %d", test.header, res.StatusCode)
		}
	}
	c.request("PUT", "/api/v1/admin/settings", map[string]any{"storage_path": t.TempDir()}, 200)
	raw := c.request("GET", "/api/v1/admin/backup", nil, 200)
	z, e := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if e != nil {
		t.Fatal(e)
	}
	if len(z.File) != 1 || z.File[0].Name != "backup.json" {
		t.Fatal("backup manifest missing")
	}
	f, e := z.File[0].Open()
	if e != nil {
		t.Fatal(e)
	}
	data, _ := io.ReadAll(f)
	f.Close()
	if !strings.Contains(string(data), "madi-logical-backup") || strings.Contains(string(data), "madi_session") {
		t.Fatal("invalid backup")
	}
	added := testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "복원하면 제거될 검증 문서"}, 200))
	uploadRestore := func(confirmation string, want int) {
		t.Helper()
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		part, err := writer.CreateFormFile("file", "backup.zip")
		if err != nil {
			t.Fatal(err)
		}
		part.Write(raw)
		writer.WriteField("confirmation", confirmation)
		writer.Close()
		r, _ := http.NewRequest("POST", server.URL+"/api/v1/admin/restore", &body)
		r.Header.Set("Content-Type", writer.FormDataContentType())
		r.Header.Set("X-Madi-Request", "1")
		res, err := c.client.Do(r)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		payload, _ := io.ReadAll(res.Body)
		if res.StatusCode != want {
			t.Fatalf("restore status=%d want=%d: %s", res.StatusCode, want, payload)
		}
	}
	uploadRestore("", 400)
	c.request("GET", "/api/v1/documents/"+str(added, "id"), nil, 200)
	uploadRestore("RESTORE", 200)
	c.request("GET", "/api/v1/documents/"+str(added, "id"), nil, 404)
	c.request("GET", "/api/v1/databases/"+id+"/rows", nil, 200)
}
