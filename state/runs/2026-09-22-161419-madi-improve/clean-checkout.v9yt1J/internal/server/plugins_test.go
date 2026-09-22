package server

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func pluginTestManifest() pluginManifest {
	return pluginManifest{ID: "knowledge-helper", Name: "지식 도우미", Version: "1.0.0", APIVersion: 1, Entry: "main.js", Capabilities: []string{"document:read", "document:write", "database:read", "database:write", "ai:execute", "storage:personal", "ui:notify", "ui:navigate", "file:import", "file:export"}, Contributions: map[string][]pluginContribution{"commands": {{ID: "search", Title: "문서 검색"}}, "importers": {{ID: "markdown", Title: "Markdown 가져오기", Extensions: []string{".md"}}}, "exporters": {{ID: "markdown", Title: "Markdown 내보내기", Extensions: []string{".md"}}}, "ai_providers": {{ID: "workspace-ai", Title: "워크스페이스 AI", Provider: "workspace"}}}}
}
func pluginTestZIP(t *testing.T, m pluginManifest, extras map[string][]byte) []byte {
	t.Helper()
	var out bytes.Buffer
	writer := zip.NewWriter(&out)
	files := map[string][]byte{"manifest.json": jsonValue(m), "main.js": []byte("madi.ready().then(()=>madi.ui.render({type:'text',text:'안녕하세요'}));")}
	for name, data := range extras {
		files[name] = data
	}
	for name, data := range files {
		file, e := writer.Create(name)
		if e != nil {
			t.Fatal(e)
		}
		file.Write(data)
	}
	if e := writer.Close(); e != nil {
		t.Fatal(e)
	}
	return out.Bytes()
}
func pluginTestUpload(t *testing.T, client *integrationTestClient, data []byte, replace string, want int) []byte {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, _ := writer.CreateFormFile("file", "plugin.zip")
	part.Write(data)
	writer.WriteField("replace", replace)
	writer.Close()
	r, _ := http.NewRequest("POST", client.base+"/api/v1/admin/plugins", &body)
	r.Header.Set("Content-Type", writer.FormDataContentType())
	r.Header.Set("X-Madi-Request", "1")
	response, e := client.client.Do(r)
	if e != nil {
		t.Fatal(e)
	}
	defer response.Body.Close()
	result, _ := io.ReadAll(response.Body)
	if response.StatusCode != want {
		t.Fatalf("install=%d want=%d body=%s", response.StatusCode, want, result)
	}
	return result
}
func TestPluginPackageBoundsManifestAndTraversal(t *testing.T) {
	m := pluginTestManifest()
	if _, e := readPluginPackage(pluginTestZIP(t, m, nil)); e != nil {
		t.Fatal(e)
	}
	for _, name := range []string{"../outside.js", "/absolute.js", "nested/../../outside.js", "C:\\outside.js", "a/../main.js", "index.html", "image.svg"} {
		if _, e := readPluginPackage(pluginTestZIP(t, m, map[string][]byte{name: []byte("unsafe")})); e == nil {
			t.Fatalf("accepted %s", name)
		}
	}
	var out bytes.Buffer
	writer := zip.NewWriter(&out)
	header := &zip.FileHeader{Name: "link.js", Method: zip.Store}
	header.SetMode(os.ModeSymlink | 0777)
	file, _ := writer.CreateHeader(header)
	file.Write([]byte("main.js"))
	writer.Close()
	if _, e := readPluginPackage(out.Bytes()); e == nil {
		t.Fatal("accepted symlink")
	}
	if _, e := readPluginPackage(pluginTestZIP(t, m, map[string][]byte{"bomb.txt": bytes.Repeat([]byte("a"), 2<<20)})); e == nil {
		t.Fatal("compression bomb accepted")
	}
	m.Capabilities = append(m.Capabilities, "system:admin")
	if validatePluginManifest(&m) == nil {
		t.Fatal("unknown permission accepted")
	}
	m = pluginTestManifest()
	m.Contributions["ai_providers"][0].Provider = "https://evil.example"
	if validatePluginManifest(&m) == nil {
		t.Fatal("external AI provider accepted")
	}
	m = pluginTestManifest()
	m.Capabilities = append(m.Capabilities, m.Capabilities[0])
	if validatePluginManifest(&m) == nil {
		t.Fatal("duplicate permission accepted")
	}
}

func TestPluginRequestLimitIsIndependentFromLogin(t *testing.T) {
	s := &Server{}
	for i := 0; i < 120; i++ {
		if !s.pluginRequestAllowed("user", "knowledge-helper") {
			t.Fatalf("early rejection %d", i)
		}
	}
	if s.pluginRequestAllowed("user", "knowledge-helper") {
		t.Fatal("plugin flood accepted")
	}
	if !s.allowLoginAttempt("127.0.0.1", "person@example.test", time.Now()) {
		t.Fatal("plugin requests consumed login budget")
	}
	if !s.pluginRequestAllowed("another-user", "knowledge-helper") {
		t.Fatal("users share plugin request budget")
	}
}
func TestPostgresPluginGrantsACLIsolationAndUpgrade(t *testing.T) {
	app, server := integrationTestServer(t)
	admin := newIntegrationTestClient(t, server.URL)
	admin.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200)
	var workspaces []map[string]any
	json.Unmarshal(admin.request("GET", "/api/v1/workspaces", nil, 200), &workspaces)
	wid := str(workspaces[0], "id")
	admin.request("POST", "/api/v1/admin/users", map[string]any{"email": "plugin-user@example.test", "name": "플러그인 사용자", "role": "editor", "password": "Plugin-User-Password-2026!"}, 200)
	admin.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": "plugin-user@example.test", "role": "editor"}, 200)
	user := newIntegrationTestClient(t, server.URL)
	user.request("POST", "/api/v1/auth/login", map[string]any{"email": "plugin-user@example.test", "password": "Plugin-User-Password-2026!"}, 200)
	manifest := pluginTestManifest()
	archive := pluginTestZIP(t, manifest, nil)
	pluginTestUpload(t, admin, archive, "", 200)
	pluginTestUpload(t, user, archive, "", 403)
	endpoint := "/api/v1/plugins/" + manifest.ID
	grant := "/api/v1/workspaces/" + wid + "/plugins/" + manifest.ID
	request := func(client *integrationTestClient, operation string, args map[string]any, status int) []byte {
		return client.request("POST", endpoint+"/bridge", map[string]any{"workspace_id": wid, "operation": operation, "args": args}, status)
	}
	admin.request("GET", endpoint+"/runtime?workspace_id="+wid, nil, 403)
	user.request("PUT", grant, map[string]any{"enabled": true, "capabilities": manifest.Capabilities}, 403)
	admin.request("PUT", grant, map[string]any{"enabled": true, "capabilities": []string{"system:admin"}}, 400)
	admin.request("PUT", grant, map[string]any{"enabled": true, "capabilities": []string{"document:read", "storage:personal", "ui:navigate"}}, 200)
	private := testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "PLUGIN_PRIVATE_TITLE", "markdown": "PLUGIN_PRIVATE_SENTINEL", "visibility": "private"}, 200))
	request(user, "documents.get", map[string]any{"id": private["id"]}, 404)
	if data := string(request(user, "documents.list", map[string]any{}, 200)); strings.Contains(data, "PLUGIN_PRIVATE_SENTINEL") || strings.Contains(data, "PLUGIN_PRIVATE_TITLE") {
		t.Fatal("plugin list leaked private document")
	}
	// Legacy/front-matter metadata may be large; summary SQL must bound tags
	// before rows are decoded and before the plugin's 4MB response backstop.
	large := testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "큰 메타데이터의 제한된 목록", "markdown": "원문은 개별 API로 읽습니다"}, 200))
	tags := []string{strings.Repeat("OVERSIZED_PLUGIN_TAG_", 65536)}
	for i := 0; i < 40; i++ {
		tags = append(tags, fmt.Sprintf("정상 태그 %d", i))
	}
	if _, err := app.DB.Exec(context.Background(), "UPDATE documents SET tags=$2 WHERE id=$1", large["id"], jsonValue(tags)); err != nil {
		t.Fatal(err)
	}
	bounded := request(user, "documents.list", map[string]any{}, 200)
	if len(bounded) > 128<<10 || bytes.Contains(bounded, []byte("OVERSIZED_PLUGIN_TAG_")) {
		t.Fatal("plugin summary allocated unbounded metadata")
	}
	var entries []map[string]any
	if err := json.Unmarshal(bounded, &entries); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, entry := range entries {
		if entry["id"] == large["id"] {
			found = true
			if len(listStrings(entry["tags"])) != 32 || entry["tags_truncated"] != true {
				t.Fatalf("invalid bounded metadata: %v", entry)
			}
		}
	}
	if !found {
		t.Fatal("bounded document missing from plugin summary")
	}
	request(admin, "documents.create", map[string]any{"title": "권한 상승"}, 403)
	request(user, "ui.navigate", map[string]any{"path": "/app/documents/" + str(private, "id")}, 403)
	request(user, "ui.navigate", map[string]any{"path": "https://evil.example"}, 403)
	request(user, "ui.navigate", map[string]any{"path": "/admin"}, 403)
	request(user, "storage.set", map[string]any{"key": "preference", "value": "USER_PRIVATE_SETTING"}, 200)
	if data := string(request(admin, "storage.get", map[string]any{"key": "preference"}, 200)); strings.Contains(data, "USER_PRIVATE_SETTING") {
		t.Fatal("plugin personal storage leak")
	}
	admin.request("PUT", grant, map[string]any{"enabled": true, "capabilities": manifest.Capabilities}, 200)
	created := testJSONObject(t, request(user, "documents.create", map[string]any{"title": "플러그인으로 만든 문서", "markdown": "실제 Markdown"}, 200))
	if str(created, "workspace_id") != wid {
		t.Fatal("workspace scope escaped")
	}
	other := testJSONObject(t, admin.request("POST", "/api/v1/workspaces", map[string]any{"name": "다른 워크스페이스"}, 200))
	foreign := testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": other["id"], "title": "OTHER_SCOPE_SENTINEL"}, 200))
	request(admin, "documents.get", map[string]any{"id": foreign["id"]}, 404)
	request(admin, "file.export", map[string]any{"contribution_id": "markdown", "name": "../../file.md"}, 400)
	request(admin, "file.export", map[string]any{"contribution_id": "markdown", "name": "file.html"}, 400)
	request(admin, "file.export", map[string]any{"contribution_id": "markdown", "name": "knowledge.md"}, 200)
	admin.request("PUT", grant, map[string]any{"enabled": false, "capabilities": manifest.Capabilities}, 200)
	request(user, "documents.list", map[string]any{}, 403)
	user.request("GET", endpoint+"/runtime?workspace_id="+wid, nil, 403)
	admin.request("PUT", grant, map[string]any{"enabled": true, "capabilities": manifest.Capabilities}, 200)
	pluginTestUpload(t, admin, archive, "", 409)
	pluginTestUpload(t, admin, archive, "REPLACE", 200)
	request(admin, "documents.list", map[string]any{}, 403)
	admin.request("PUT", grant, map[string]any{"enabled": true, "capabilities": manifest.Capabilities}, 200)
	admin.request("PUT", "/api/v1/admin/plugins/"+manifest.ID, map[string]any{"enabled": false}, 200)
	request(user, "documents.list", map[string]any{}, 403)
}
func TestPostgresPluginAIStreamAndRevocation(t *testing.T) {
	_, server := integrationTestServer(t)
	admin := newIntegrationTestClient(t, server.URL)
	admin.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200)
	var workspaces []map[string]any
	json.Unmarshal(admin.request("GET", "/api/v1/workspaces", nil, 200), &workspaces)
	wid := str(workspaces[0], "id")
	manifest := pluginTestManifest()
	pluginTestUpload(t, admin, pluginTestZIP(t, manifest, nil), "", 200)
	grant := "/api/v1/workspaces/" + wid + "/plugins/" + manifest.ID
	admin.request("PUT", grant, map[string]any{"enabled": true, "capabilities": manifest.Capabilities}, 200)
	started, canceled := make(chan struct{}), make(chan struct{})
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer PLUGIN_PROVIDER_SECRET" {
			t.Error("configured AI secret missing")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"실제 스트리밍 응답\"}}]}\n\n")
		w.(http.Flusher).Flush()
		<-r.Context().Done()
		close(canceled)
	}))
	defer provider.Close()
	admin.request("PUT", "/api/v1/admin/settings", map[string]any{"ai_enabled": true, "ai_base_url": provider.URL, "ai_model": "test", "ai_api_key": "PLUGIN_PROVIDER_SECRET"}, 200)
	runtime := admin.request("GET", "/api/v1/plugins/"+manifest.ID+"/runtime?workspace_id="+wid, nil, 200)
	if strings.Contains(string(runtime), "PLUGIN_PROVIDER_SECRET") {
		t.Fatal("provider secret leaked to runtime")
	}
	finished := make(chan []byte, 1)
	go func() {
		req, _ := http.NewRequest("POST", admin.base+"/api/v1/plugins/"+manifest.ID+"/bridge", bytes.NewReader(jsonValue(map[string]any{"workspace_id": wid, "operation": "ai.chat", "args": map[string]any{"prompt": "기록을 요약해 주세요"}})))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Madi-Request", "1")
		res, err := admin.client.Do(req)
		if err != nil {
			finished <- []byte(err.Error())
			return
		}
		defer res.Body.Close()
		var body bytes.Buffer
		scanner := bufio.NewScanner(res.Body)
		sawText := false
		for scanner.Scan() {
			line := scanner.Bytes()
			body.Write(line)
			body.WriteByte('\n')
			// Revoke after the client receives the first checked delta, not
			// merely after the provider writes it. Revoking earlier correctly
			// prevents that delta from passing the current-permission guard.
			if !sawText && bytes.Contains(line, []byte("실제 스트리밍 응답")) {
				sawText = true
				close(started)
			}
		}
		finished <- body.Bytes()
	}()
	select {
	case <-started:
	case <-time.After(4 * time.Second):
		t.Fatal("AI did not start")
	}
	admin.request("PUT", grant, map[string]any{"enabled": true, "capabilities": []string{"document:read"}}, 200)
	select {
	case <-canceled:
	case <-time.After(4 * time.Second):
		t.Fatal("revoked AI stream was not canceled")
	}
	select {
	case body := <-finished:
		if !bytes.Contains(body, []byte("실제 스트리밍 응답")) {
			t.Fatal(string(body))
		}
	case <-time.After(4 * time.Second):
		t.Fatal("stream response did not finish")
	}
}
