package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func connectorTestSetup(t *testing.T, endpoint string) (*Server, *integrationTestClient, context.Context, string, map[string]any) {
	t.Helper()
	s, client, ctx, _, wid := jobTestFixture(t)
	client.request("PUT", "/api/v1/admin/connectors/settings", map[string]any{"enabled": true, "allowed_hosts": []string{"127.0.0.1"}}, 200)
	service := testJSONObject(t, client.request("POST", "/api/v1/admin/users", map[string]any{"email": "remote-reader@example.test", "name": "원격 읽기 계정", "role": "editor", "kind": "service"}, 200))
	client.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": "remote-reader@example.test", "role": "editor"}, 200)
	input := map[string]any{"workspace_id": wid, "space_id": "", "service_account_id": str(service, "id"), "name": "실제 테스트 연결", "kind": "rest", "base_url": endpoint, "enabled": true, "config": map[string]any{"allow_http": true, "acknowledge_acl": true, "items_path": "items", "id_path": "id", "title_path": "title", "content_path": "content", "next_path": "next", "list_path": "list", "max_items": 100}, "credentials": map[string]any{"token": "connection-secret-not-public"}}
	return s, client, ctx, wid, input
}
func connectorRunResult(t *testing.T, s *Server, client *integrationTestClient, id, action string) map[string]any {
	t.Helper()
	queued := testJSONObject(t, client.request("POST", "/api/v1/connectors/"+id+"/"+action, map[string]any{"rewind": true}, 200))
	drainJobs(t, s)
	var reportRaw []byte
	var status, message string
	if e := s.DB.QueryRow(context.Background(), `SELECT r.report,j.status,j.last_error FROM connector_runs r JOIN automation_jobs j ON j.id=r.job_id WHERE r.job_id=$1`, str(queued, "job_id")).Scan(&reportRaw, &status, &message); e != nil {
		t.Fatal(e)
	}
	if status != "succeeded" {
		t.Fatalf("connector job %s: %s", status, message)
	}
	report := map[string]any{}
	_ = json.Unmarshal(reportRaw, &report)
	return report
}
func TestPostgresConnectorRealHTTPPreviewResumeConflictRevocation(t *testing.T) {
	var calls atomic.Int32
	var changed atomic.Bool
	var wrongAuth atomic.Bool
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.Header.Get("Authorization") != "Bearer connection-secret-not-public" {
			wrongAuth.Store(true)
		}
		if calls.Add(1) == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(429)
			return
		}
		content := "# upstream"
		if changed.Load() {
			content = "# upstream changed"
		}
		next := "?page=2"
		id := "first"
		if r.URL.Query().Get("page") == "2" {
			next = ""
			id = "second"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"items": []map[string]any{{"id": id, "title": id, "content": content}}, "next": next})
	}))
	defer upstream.Close()
	s, client, ctx, wid, input := connectorTestSetup(t, upstream.URL)
	created := testJSONObject(t, client.request("POST", "/api/v1/connectors", input, 200))
	id := str(created, "id")
	if strings.Contains(string(jsonValue(created)), "connection-secret-not-public") {
		t.Fatal("connector credential leaked")
	}
	report := connectorRunResult(t, s, client, id, "preview")
	if number(report, "preview_count", 0) != 1 || !boolean(report, "has_more") {
		t.Fatalf("preview %#v", report)
	}
	var count int
	_ = s.DB.QueryRow(ctx, "SELECT count(*) FROM connector_records WHERE connector_id=$1", id).Scan(&count)
	if count != 0 {
		t.Fatal("preview mutated documents")
	}
	report = connectorRunResult(t, s, client, id, "run")
	if number(report, "created", 0) != 2 || number(report, "pages", 0) != 2 || wrongAuth.Load() {
		t.Fatalf("paginated import %#v auth%v", report, wrongAuth.Load())
	}
	report = connectorRunResult(t, s, client, id, "run")
	if number(report, "unchanged", 0) != 2 {
		t.Fatalf("rewind not idempotent %#v", report)
	}
	var doc string
	_ = s.DB.QueryRow(ctx, "SELECT document_id::text FROM connector_records WHERE connector_id=$1 AND remote_id='first'", id).Scan(&doc)
	client.request("PUT", "/api/v1/documents/"+doc, map[string]any{"title": "사용자 편집", "markdown": "보존할 로컬 변경", "version": 1}, 200)
	changed.Store(true)
	report = connectorRunResult(t, s, client, id, "run")
	if number(report, "conflicts", 0) != 1 || number(report, "updated", 0) != 1 {
		t.Fatalf("local edit not preserved %#v", report)
	}
	local := testJSONObject(t, client.request("GET", "/api/v1/documents/"+doc, nil, 200))
	if str(local, "markdown") != "보존할 로컬 변경" {
		t.Fatal("local edit overwritten")
	}
	queued := testJSONObject(t, client.request("POST", "/api/v1/connectors/"+id+"/run", map[string]any{}, 200))
	beforeCalls := calls.Load()
	client.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": "remote-reader@example.test", "role": "viewer"}, 200)
	drainJobs(t, s)
	var status string
	_ = s.DB.QueryRow(ctx, "SELECT status FROM automation_jobs WHERE id=$1", str(queued, "job_id")).Scan(&status)
	if status != "failed" || calls.Load() != beforeCalls {
		t.Fatal("revoked service account fetched remote data")
	}
}
func TestPostgresConnectorAtomicPageAndHostBoundary(t *testing.T) {
	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"items": []map[string]any{{"id": "ok", "title": "ok", "content": "first"}, {"id": "bad", "title": "bad", "content": strings.Repeat("x", 4<<20+1)}}})
	}))
	defer receiver.Close()
	s, client, ctx, _, input := connectorTestSetup(t, receiver.URL)
	created := testJSONObject(t, client.request("POST", "/api/v1/connectors", input, 200))
	id := str(created, "id")
	client.request("POST", "/api/v1/connectors/"+id+"/run", map[string]any{}, 200)
	drainJobs(t, s)
	var count int
	_ = s.DB.QueryRow(ctx, "SELECT count(*) FROM connector_records WHERE connector_id=$1", id).Scan(&count)
	if count != 0 {
		t.Fatal("invalid page partially committed")
	}
	c, e := s.loadConnector(ctx, id)
	if e != nil {
		t.Fatal(e)
	}
	if _, _, e = s.connectorRead(ctx, c, "http://unapproved.example.invalid/page"); e == nil {
		t.Fatal("cross-origin credential forwarding accepted")
	}
	c.BaseURL = "http://169.254.169.254"
	client.request("PUT", "/api/v1/admin/connectors/settings", map[string]any{"enabled": true, "allowed_hosts": []string{"169.254.169.254"}}, 200)
	if _, _, e = s.connectorRead(ctx, c, ""); e == nil {
		t.Fatal("metadata target accepted")
	}
}
func TestPostgresConnectorProviderProtocols(t *testing.T) {
	var bad atomic.Bool
	sha := strings.Repeat("a", 40)
	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			bad.Store(true)
		}
		switch {
		case strings.Contains(r.URL.Path, "/wikis"):
			w.Header().Set("X-Next-Page", "2")
			fmt.Fprint(w, `[{"slug":"guide","title":"Guide","content":"# GitLab"}]`)
		case strings.HasSuffix(r.URL.Path, "/search/jql"):
			fmt.Fprint(w, `{"issues":[{"key":"OPS-1","fields":{"summary":"Jira","description":{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"Atlassian text"}]}]}}}],"nextPageToken":"token2"}`)
		case r.URL.Path == "/confluence/pages":
			fmt.Fprint(w, `{"results":[{"id":"3","title":"Wiki","body":{"storage":{"value":"<p>safe<script>bad</script></p>"}}}],"_links":{"next":"?cursor=2"}}`)
		case r.URL.Path == "/drive/files":
			fmt.Fprint(w, `{"files":[{"id":"file1","name":"Drive","mimeType":"application/vnd.google-apps.document"}],"nextPageToken":"next-drive"}`)
		case r.URL.Path == "/drive/files/file1/export":
			if r.URL.Query().Get("mimeType") != "text/plain" {
				bad.Store(true)
			}
			fmt.Fprint(w, "# exported Google doc")
		case strings.Contains(r.URL.Path, "microsoft.graph.sitePage"):
			fmt.Fprint(w, `{"value":[{"id":"sitepage","title":"SharePoint","canvasLayout":{"horizontalSections":[{"columns":[{"webparts":[{"innerHtml":"<p>Graph page</p>"}]}]}]}}]}`)
		case strings.Contains(r.URL.Path, "/git/trees/"):
			_ = json.NewEncoder(w).Encode(map[string]any{"sha": sha, "tree": []map[string]any{{"type": "blob", "path": "README.md", "sha": sha}}})
		case strings.Contains(r.URL.Path, "/git/blobs/"):
			_ = json.NewEncoder(w).Encode(map[string]any{"encoding": "base64", "content": base64.StdEncoding.EncodeToString([]byte("# GitHub source"))})
		default:
			bad.Store(true)
			http.NotFound(w, r)
		}
	}))
	defer receiver.Close()
	s, client, ctx, _, input := connectorTestSetup(t, receiver.URL)
	_ = client
	_ = input
	cases := []struct {
		kind, path, title, next string
		config                  map[string]any
	}{
		{"gitlab", "/gitlab", "Guide", "2", map[string]any{"project_id": "group/project"}},
		{"jira", "/jira", "OPS-1 · Jira", "token2", map[string]any{"jql": "project=OPS"}},
		{"confluence", "/confluence", "Wiki", receiver.URL + "/confluence/pages?cursor=2", nil},
		{"drive", "/drive", "Drive", "next-drive", nil},
		{"sharepoint", "/graph", "SharePoint", "", map[string]any{"site_id": "site"}},
		{"github", "/github", "README", "", map[string]any{"repository": "org/repo"}},
	}
	for _, tc := range cases {
		t.Run(tc.kind, func(t *testing.T) {
			config := map[string]any{"allow_http": true}
			for k, v := range tc.config {
				config[k] = v
			}
			page, e := s.fetchConnectorPage(ctx, connectorConfig{Kind: tc.kind, BaseURL: receiver.URL + tc.path, Config: config}, "")
			if e != nil || len(page.Items) != 1 || page.Items[0].Title != tc.title || page.Next != tc.next {
				t.Fatalf("protocol result %#v %v", page, e)
			}
			if strings.Contains(page.Items[0].Markdown, "<script>") {
				t.Fatal("unsafe HTML retained")
			}
		})
	}
	if bad.Load() {
		t.Fatal("unexpected protocol request")
	}
}
func TestConnectorRetryDelayAndURL(t *testing.T) {
	if (connectorRetry{time.Minute}).RetryAfter() != time.Minute {
		t.Fatal("retry delay lost")
	}
	if _, e := connectorURL("https://api.example.test/v1", "https://other.example.test/list"); e == nil {
		t.Fatal("origin boundary absent")
	}
	if _, e := connectorURL("https://user:secret@api.example.test", ""); e == nil {
		t.Fatal("embedded credential accepted")
	}
}
