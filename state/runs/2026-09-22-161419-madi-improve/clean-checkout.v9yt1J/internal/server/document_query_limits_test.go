package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestPostgresDocumentQueryBoundsTimeoutAndExpiredSession(t *testing.T) {
	s, c, ctx, p, wid := jobTestFixture(t)
	query := `{"version":1,"source":"documents","columns":[{"field":"title"}],"limit":100}`
	id := documentQueryCreate(t, c, wid, "문서 후보 조회", "```madi-query\n"+query+"\n```", "workspace")
	in := documentQueryInput(t, c, id, nil)
	if _, e := s.DB.Exec(ctx, `INSERT INTO documents(id,workspace_id,title,markdown,owner_id,status,visibility) SELECT gen_random_uuid(),$1,'범위 후보 '||i,'본문',$2,'draft','workspace' FROM generate_series(1,2100) i`, wid, p.ID); e != nil {
		t.Fatal(e)
	}
	out := testJSONObject(t, c.request("POST", "/api/v1/documents/"+id+"/queries/execute", in, 200))
	diag := out["diagnostics"].(map[string]any)
	if !boolean(diag, "truncated") || number(diag, "candidates", 0) != 2000 || len(out["rows"].([]any)) != 100 {
		t.Fatal(diag)
	}
	if _, e := s.DB.Exec(ctx, `UPDATE documents SET markdown=repeat('x',270000),updated_at=clock_timestamp() WHERE id=(SELECT id FROM documents WHERE id<>$1 AND workspace_id=$2 LIMIT 1)`, id, wid); e != nil {
		t.Fatal(e)
	}
	q2 := `{"version":1,"source":"documents","columns":[{"field":"title"},{"field":"property.번호"}],"properties":{"번호":"number"},"limit":20}`
	large := documentQueryCreate(t, c, wid, "속성 범위 조회", "```madi-query\n"+q2+"\n```", "workspace")
	largeIn := documentQueryInput(t, c, large, nil)
	out = testJSONObject(t, c.request("POST", "/api/v1/documents/"+large+"/queries/execute", largeIn, 200))
	diag = out["diagnostics"].(map[string]any)
	if number(diag, "oversized_documents", 0) != 1 || !boolean(diag, "truncated") {
		t.Fatal(diag)
	}
	// A metadata-only query must not repeatedly load full 5MiB Markdown bodies.
	if _, e := s.DB.Exec(ctx, `UPDATE documents SET markdown=repeat('x',1048577),version=version+1 WHERE id=$1`, large); e != nil {
		t.Fatal(e)
	}
	c.request("GET", "/api/v1/documents/"+large+"/queries", nil, 413)
	for n := 0; n < cap(s.documentQuerySlots); n++ {
		s.documentQuerySlots <- struct{}{}
	}
	c.request("POST", "/api/v1/documents/"+id+"/queries/execute", in, 429)
	for n := 0; n < cap(s.documentQuerySlots); n++ {
		<-s.documentQuerySlots
	}
	blocker, e := s.DB.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer blocker.Rollback(ctx)
	if _, e = blocker.Exec(ctx, "LOCK TABLE knowledge_document_meta IN ACCESS EXCLUSIVE MODE"); e != nil {
		t.Fatal(e)
	}
	started := time.Now()
	c.request("POST", "/api/v1/documents/"+id+"/queries/execute", in, 408)
	t.Logf("actual SQL-lock timeout: %s", time.Since(started))
	if e = blocker.Rollback(ctx); e != nil {
		t.Fatal(e)
	}
	if len(s.documentQuerySlots) != 0 {
		t.Fatal("timeout retained query slot")
	}
	blocker, e = s.DB.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer blocker.Rollback(ctx)
	if _, e = blocker.Exec(ctx, "SELECT id FROM users WHERE id=$1 FOR UPDATE", p.ID); e != nil {
		t.Fatal(e)
	}
	pid := blocker.Conn().PgConn().PID()
	result := make(chan int, 1)
	go func() {
		raw, _ := json.Marshal(in)
		req, _ := http.NewRequest("POST", c.base+"/api/v1/documents/"+id+"/queries/execute", bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Madi-Request", "1")
		response, e := c.client.Do(req)
		if e != nil {
			result <- 0
			return
		}
		io.Copy(io.Discard, response.Body)
		response.Body.Close()
		result <- response.StatusCode
	}()
	waiting := false
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if e = s.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE $1::integer=ANY(pg_blocking_pids(pid)))`, pid).Scan(&waiting); e != nil {
			t.Fatal(e)
		}
		if waiting {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !waiting {
		t.Fatal("actual query never waited on actor lock")
	}
	if _, e = s.DB.Exec(ctx, `UPDATE sessions SET expires_at=clock_timestamp()-interval '1 second' WHERE user_id=$1`, p.ID); e != nil {
		t.Fatal(e)
	}
	if e = blocker.Commit(ctx); e != nil {
		t.Fatal(e)
	}
	select {
	case status := <-result:
		if status != 403 {
			t.Fatal("expired session survived query lock", status)
		}
	case <-time.After(4 * time.Second):
		t.Fatal("query failed to stop")
	}
}
func TestPostgresDocumentQueryPIIRevisionAndConnectionCancellation(t *testing.T) {
	s, c, ctx, _, wid := jobTestFixture(t)
	id := documentQueryCreate(t, c, wid, "조회 안전성", "```madi-query\n{\"version\":1,\"source\":\"documents\",\"columns\":[{\"field\":\"title\"}],\"limit\":20}\n```", "workspace")
	documentQueryCreate(t, c, wid, "관측 표본", "원문", "workspace")
	in := documentQueryInput(t, c, id, nil)
	out := testJSONObject(t, c.request("POST", "/api/v1/documents/"+id+"/queries/execute", in, 200))
	if _, e := s.DB.Exec(ctx, "UPDATE protection_settings SET revision=revision+1 WHERE id=1"); e != nil {
		t.Fatal(e)
	}
	c.request("POST", "/api/v1/documents/"+id+"/queries/check", map[string]any{"validation_token": out["validation_token"]}, 409)
	blocker, e := s.DB.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer blocker.Rollback(ctx)
	if _, e = blocker.Exec(ctx, "LOCK TABLE knowledge_document_meta IN ACCESS EXCLUSIVE MODE"); e != nil {
		t.Fatal(e)
	}
	pid := blocker.Conn().PgConn().PID()
	requestCtx, cancel := context.WithCancel(ctx)
	raw, _ := json.Marshal(in)
	request, _ := http.NewRequestWithContext(requestCtx, "POST", c.base+"/api/v1/documents/"+id+"/queries/execute", strings.NewReader(string(raw)))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Madi-Request", "1")
	done := make(chan struct{})
	go func() {
		response, e := c.client.Do(request)
		if e == nil {
			response.Body.Close()
		}
		close(done)
	}()
	deadline := time.Now().Add(time.Second)
	waiting := false
	for time.Now().Before(deadline) {
		if e = s.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE $1::integer=ANY(pg_blocking_pids(pid)))`, pid).Scan(&waiting); e != nil {
			t.Fatal(e)
		}
		if waiting {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if !waiting {
		cancel()
		t.Fatal("cancel fixture never reached an actual SQL wait")
	}
	cancel()
	<-done
	deadline = time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) && len(s.documentQuerySlots) > 0 {
		time.Sleep(10 * time.Millisecond)
	}
	if len(s.documentQuerySlots) != 0 {
		t.Fatal("cancelled request retained slot")
	}
}
