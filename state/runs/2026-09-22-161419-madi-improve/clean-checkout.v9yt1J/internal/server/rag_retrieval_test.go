package server

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPostgresRAGExactCitationsCurrentVersionAndACL(t *testing.T) {
	s, ts := integrationTestServer(t)
	ctx := t.Context()
	c := newIntegrationTestClient(t, ts.URL)
	me := testJSONObject(t, c.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200))
	uid := str(me, "id")
	wid := str(testJSONObject(t, c.request("POST", "/api/v1/workspaces", map[string]any{"name": "정확한 인용"}, 200)), "id")
	md := "---\ntags: [운영]\n---\n# 긴 원문\n\n" + strings.Repeat("일반적인 운영 설명 문장입니다.\n", 500) + "\n## GPU 메모리 절차\n\nGPU OOM 대응 절차는 이 위치에 있습니다. 🚀\n"
	id := str(testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "운영 지식", "markdown": md, "visibility": "private"}, 200)), "id")
	p, e := s.workerPrincipal(ctx, uid, "", wid)
	if e != nil {
		t.Fatal(e)
	}
	r := httptest.NewRequest("GET", "/", nil).WithContext(context.WithValue(ctx, principalKey, p))
	sources, e := s.aiContext(r, p, id, wid, "GPU OOM 절차")
	if e != nil || len(sources) == 0 {
		t.Fatal(e, sources)
	}
	source := sources[0]
	if !strings.Contains(source.Markdown, "GPU OOM") || source.StartByte == 0 || source.Markdown != md[source.StartByte:source.EndByte] || source.ContentHash != digest(source.Markdown) {
		t.Fatal("retrieval lost exact source", source)
	}
	v := testJSONObject(t, c.request("GET", "/api/v1"+source.CitationURL, nil, 200))
	if !boolean(v, "verified") || str(v, "markdown") != source.Markdown {
		t.Fatal(v)
	}
	c.request("GET", "/api/v1/documents/"+id+"/citation?version=1&start=0&end=8&hash="+strings.Repeat("0", 64), nil, 409)
	c.request("GET", "/api/v1/documents/"+id+"/citation?version=1&start=-1&end=8&hash="+strings.Repeat("0", 64), nil, 400)
	c.request("PUT", "/api/v1/documents/"+id, map[string]any{"version": 1, "markdown": md + "\n수정"}, 200)
	c.request("GET", "/api/v1"+source.CitationURL, nil, 409)
	user := testJSONObject(t, c.request("POST", "/api/v1/admin/users", map[string]any{"email": "citation-reader@example.test", "name": "출처 검증", "role": "editor", "password": "Citation-Test-Password-2026!"}, 200))
	c.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": user["email"], "role": "viewer"}, 200)
	viewer := newIntegrationTestClient(t, ts.URL)
	viewer.request("POST", "/api/v1/auth/login", map[string]any{"email": user["email"], "password": "Citation-Test-Password-2026!"}, 200)
	viewer.request("GET", "/api/v1"+source.CitationURL, nil, 404)
	p, e = s.workerPrincipal(ctx, str(user, "id"), "", wid)
	if e != nil {
		t.Fatal(e)
	}
	r = r.WithContext(context.WithValue(ctx, principalKey, p))
	sources, e = s.aiContext(r, p, "", wid, "GPU OOM")
	if e != nil || len(sources) != 0 {
		t.Fatal("private source entered workspace context", e, sources)
	}
}
