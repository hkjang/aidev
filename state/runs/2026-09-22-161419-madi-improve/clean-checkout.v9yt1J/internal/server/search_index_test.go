package server

import (
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestPostgresSearchProjectionQueueVersionAndACL(t *testing.T) {
	s, ts := integrationTestServer(t)
	ctx := t.Context()
	c := newIntegrationTestClient(t, ts.URL)
	c.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200)
	wid := str(testJSONObject(t, c.request("POST", "/api/v1/workspaces", map[string]any{"name": "로컬 검색 색인"}, 200)), "id")
	md := "# 운영 지식\n\n" + strings.Repeat("검색과 원문 인용을 검증하는 한글 지식.\n", 200)
	id := str(testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "검색 지식", "markdown": md, "visibility": "private"}, 200)), "id")
	var pending int
	if e := s.DB.QueryRow(ctx, "SELECT count(*) FROM search_index_queue WHERE document_id=$1", id).Scan(&pending); e != nil || pending != 1 {
		t.Fatal(pending, e)
	}
	if worked, e := s.indexSearchDocument(ctx, id); e != nil || !worked {
		t.Fatal(worked, e)
	}
	status := testJSONObject(t, c.request("GET", "/api/v1/search/index-status?workspace_id="+wid, nil, 200))
	if number(status, "indexed", 0) != 1 || number(status, "chunks", 0) < 2 {
		t.Fatal(status)
	}
	rows, e := s.rows(ctx, "SELECT to_jsonb(c)-'search_vector' FROM search_chunks c WHERE document_id=$1 ORDER BY ordinal", id)
	if e != nil {
		t.Fatal(e)
	}
	for _, v := range rows {
		if str(v, "content") != md[number(v, "start_byte", 0):number(v, "end_byte", 0)] {
			t.Fatal("citation differs from original", v)
		}
	}
	c.request("PUT", "/api/v1/documents/"+id, map[string]any{"version": 1, "markdown": "# 변경된 원문\n\n새로운 내용"}, 200)
	if e = s.DB.QueryRow(ctx, "SELECT count(*) FROM search_chunks c JOIN documents d ON d.id=c.document_id WHERE c.document_id=$1 AND c.document_version=d.version", id).Scan(&pending); e != nil || pending != 0 {
		t.Fatal("stale projection used", pending, e)
	}
	var wg sync.WaitGroup
	errors := make(chan error, 6)
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, e := s.indexSearchDocument(ctx, id); errors <- e }()
	}
	wg.Wait()
	close(errors)
	for e := range errors {
		if e != nil {
			t.Fatal(e)
		}
	}
	if e = s.DB.QueryRow(ctx, "SELECT count(*) FROM search_chunks WHERE document_id=$1 AND document_version=2", id).Scan(&pending); e != nil || pending != 1 {
		t.Fatal(pending, e)
	}
	// Rolled-back source edits must not enqueue a committed indexing effect.
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	tx.Exec(ctx, "UPDATE documents SET version=version+1 WHERE id=$1", id)
	tx.Rollback(ctx)
	if e = s.DB.QueryRow(ctx, "SELECT count(*) FROM search_index_queue WHERE document_id=$1", id).Scan(&pending); e != nil || pending != 0 {
		t.Fatal("rollback leaked queue", pending, e)
	}
	u := testJSONObject(t, c.request("POST", "/api/v1/admin/users", map[string]any{"email": "index-viewer@example.test", "name": "검색 조회자", "role": "editor", "password": "Search-index-password-2026!"}, 200))
	c.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": u["email"], "role": "viewer"}, 200)
	viewer := newIntegrationTestClient(t, ts.URL)
	viewer.request("POST", "/api/v1/auth/login", map[string]any{"email": u["email"], "password": "Search-index-password-2026!"}, 200)
	status = testJSONObject(t, viewer.request("GET", "/api/v1/search/index-status?workspace_id="+wid, nil, 200))
	if number(status, "documents", -1) != 0 {
		t.Fatal("private index size disclosed", status)
	}
	viewer.request("POST", "/api/v1/search/reindex", map[string]any{"document_id": id}, 404)
	c.request("POST", "/api/v1/search/reindex", map[string]any{"document_id": id}, 202)
	c.request("DELETE", "/api/v1/documents/"+id, nil, 200)
	if e = s.DB.QueryRow(ctx, "SELECT chunk_count FROM search_index_documents WHERE document_id=$1", id).Scan(&pending); e != pgx.ErrNoRows {
		t.Fatal("trash index retained", e)
	}
	c.request("POST", "/api/v1/documents/"+id+"/restore", nil, 200)
	if worked, e := s.indexSearchNext(ctx); e != nil || !worked {
		t.Fatal(worked, e)
	}
	if _, e = s.DB.Exec(ctx, "DELETE FROM search_index_documents WHERE document_id=$1", id); e != nil {
		t.Fatal(e)
	}
	if _, _, e = s.searchIndexBackfill(ctx, ""); e != nil {
		t.Fatal(e)
	}
	if worked, e := s.indexSearchDocument(ctx, id); e != nil || !worked {
		t.Fatal("restore backfill", worked, e)
	}
}
