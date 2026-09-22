package server

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
)

func TestDocumentHistoryDiffScriptsAndLimits(t *testing.T) {
	rng := rand.New(rand.NewSource(46))
	for trial := 0; trial < 500; trial++ {
		a, b := []string{}, []string{}
		for i := 0; i < rng.Intn(30); i++ {
			a = append(a, fmt.Sprintf("줄%d\n", rng.Intn(8)))
		}
		for i := 0; i < rng.Intn(30); i++ {
			b = append(b, fmt.Sprintf("줄%d\n", rng.Intn(8)))
		}
		script, ok := documentMyersDiff(a, b)
		if !ok {
			t.Fatal("small input exceeded limit")
		}
		left, right := []string{}, []string{}
		for _, row := range script {
			if row.Kind != "add" {
				left = append(left, row.Text)
			}
			if row.Kind != "remove" {
				right = append(right, row.Text)
			}
		}
		if strings.Join(a, "") == strings.Join(left, "") && strings.Join(b, "") == strings.Join(right, "") {
			continue
		}
		t.Fatalf("invalid script a=%v b=%v script=%v", a, b, script)
	}
	diff := boundedDocumentDiff("---\r\ntags: [문서]\r\n---\r\n# 제목\r\n\r\n수정 전\r\n", "---\r\ntags: [문서]\r\n---\r\n# 제목\r\n\r\n수정 후\r\n")
	if diff.Added != 1 || diff.Removed != 1 || diff.Truncated {
		t.Fatalf("precise CRLF diff %v", diff)
	}
	newline := boundedDocumentDiff("본문", "본문\n")
	if newline.Added != 1 || newline.Removed != 1 {
		t.Fatal("trailing newline difference lost")
	}
	long := boundedDocumentDiff("", strings.Repeat("긴", 10000))
	if !long.Truncated || len(long.Rows) > 2000 || len(jsonValue(long)) > 600<<10 {
		t.Fatal("long line budget")
	}
	many := boundedDocumentDiff("", strings.Repeat("\n", 50001))
	if !many.Truncated || len(many.Rows) != 0 {
		t.Fatal("line budget")
	}
}

func TestDocumentHistoryMetadataDetailDiffAndRestoreCAS(t *testing.T) {
	s, admin, ctx, _, wid := jobTestFixture(t)
	md := "---\ntags: [운영]\n---\n# 이전\n\nHISTORY_PRIVATE_SENTINEL\n\n"
	doc := testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "원본 이력", "markdown": md, "visibility": "private"}, 200))
	id := str(doc, "id")
	path := "/api/v1/documents/" + id + "/versions"
	doc = testJSONObject(t, admin.request("PUT", "/api/v1/documents/"+id, map[string]any{"version": doc["version"], "title": "변경 이력", "markdown": strings.Replace(md, "이전", "이후", 1)}, 200))
	metadata := string(admin.request("GET", path, nil, 200))
	if strings.Contains(metadata, "HISTORY_PRIVATE_SENTINEL") || strings.Contains(metadata, `"markdown":`) {
		t.Fatal("history list leaked full content")
	}
	version := testJSONObject(t, admin.request("GET", path+"/1", nil, 200))
	if str(version, "markdown") != md {
		t.Fatal("history source changed")
	}
	diff := string(admin.request("GET", path+"/diff?from=1&to=2", nil, 200))
	if !strings.Contains(diff, `"added":1`) || !strings.Contains(diff, `"removed":1`) {
		t.Fatalf("diff route %s", diff)
	}
	admin.request("POST", path+"/1/restore", nil, 400)
	newer := testJSONObject(t, admin.request("PUT", "/api/v1/documents/"+id, map[string]any{"version": doc["version"], "markdown": "다른 사용자의 더 최신 내용\n"}, 200))
	admin.request("POST", path+"/1/restore", map[string]any{"expected_version": doc["version"]}, 409)
	preserved := testJSONObject(t, admin.request("GET", "/api/v1/documents/"+id, nil, 200))
	if str(preserved, "markdown") != "다른 사용자의 더 최신 내용\n" {
		t.Fatal("stale restoration overwrote edits")
	}
	restored := testJSONObject(t, admin.request("POST", path+"/1/restore", map[string]any{"expected_version": newer["version"]}, 200))
	if str(restored, "markdown") != md || str(restored, "visibility") != "private" || number(restored, "version", 0) != 4 {
		t.Fatalf("exact restored source %v", restored)
	}
	admin.request("POST", "/api/v1/admin/users", map[string]any{"email": "history-viewer@example.test", "name": "이력 열람자", "role": "viewer", "password": "History-viewer-password-2026!"}, 200)
	admin.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": "history-viewer@example.test", "role": "viewer"}, 200)
	viewer := newIntegrationTestClient(t, admin.base)
	viewer.request("POST", "/api/v1/auth/login", map[string]any{"email": "history-viewer@example.test", "password": "History-viewer-password-2026!"}, 200)
	for _, endpoint := range []string{path, path + "/1", path + "/diff?from=1&to=2"} {
		viewer.request("GET", endpoint, nil, 403)
	}
	restored = testJSONObject(t, admin.request("PUT", "/api/v1/documents/"+id, map[string]any{"version": restored["version"], "visibility": "workspace"}, 200))
	viewer.request("GET", path+"/1", nil, 200)
	viewer.request("POST", path+"/1/restore", map[string]any{"expected_version": restored["version"]}, 403)
	_, err := s.DB.Exec(ctx, "UPDATE documents SET visibility='private' WHERE id=$1", id)
	if err != nil {
		t.Fatal(err)
	}
	viewer.request("GET", path+"/diff?from=1&to=2", nil, 403)
	admin.request("GET", path+"?limit=101", nil, 400)
	admin.request("GET", path+"?before=0", nil, 400)
}
