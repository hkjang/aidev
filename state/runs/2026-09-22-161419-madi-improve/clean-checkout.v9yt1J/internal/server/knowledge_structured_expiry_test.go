package server

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestPostgresStructuredHTTPExpiresDuringResourceLock(t *testing.T) {
	f := newStructuredFixture(t)
	preview := f.preview(t)
	for _, operation := range []string{"preview", "apply", "save"} {
		t.Run(operation, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 8*time.Second)
			defer cancel()
			secret := "structured-expiry-fixture-" + operation
			var expires time.Time
			if e := f.server.DB.QueryRow(ctx, `INSERT INTO sessions(token_hash,user_id,expires_at) VALUES($1,$2,clock_timestamp()+interval '1500 milliseconds') RETURNING expires_at`, digest(secret), f.ticket.OwnerID).Scan(&expires); e != nil {
				t.Fatal(e)
			}
			blocker, e := f.server.DB.Begin(ctx)
			if e != nil {
				t.Fatal(e)
			}
			defer blocker.Rollback(context.WithoutCancel(ctx))
			if _, e = blocker.Exec(ctx, `SELECT role FROM workspace_members WHERE workspace_id=$1 AND user_id=$2 FOR UPDATE`, f.wid, f.ticket.OwnerID); e != nil {
				t.Fatal(e)
			}
			route := "/api/v1/knowledge/structured-drafts/" + f.id + "/" + operation
			var body any = map[string]any{"revision": 1, "fields": []structuredProviderField{{PropertyID: "amount", Value: 2, Quote: "2개"}}}
			if operation == "apply" {
				body = map[string]any{"review_ticket": preview["review_ticket"], "consent": true}
			}
			if operation == "save" {
				route = "/api/v1/knowledge/structured-drafts"
				body = f.save
			}
			req, e := http.NewRequestWithContext(ctx, "POST", f.owner.base+route, bytes.NewReader(jsonValue(body)))
			if e != nil {
				t.Fatal(e)
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Madi-Request", "1")
			req.AddCookie(&http.Cookie{Name: "madi_session", Value: secret})
			type result struct {
				status int
				raw    []byte
				err    error
			}
			done := make(chan result, 1)
			go func() {
				res, e := http.DefaultClient.Do(req)
				if e != nil {
					done <- result{err: e}
					return
				}
				defer res.Body.Close()
				raw, e := io.ReadAll(res.Body)
				done <- result{res.StatusCode, raw, e}
			}()
			waiting := false
			for time.Now().Before(expires) {
				if e = f.server.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE $1::integer=ANY(pg_blocking_pids(pid)))`, int32(blocker.Conn().PgConn().PID())).Scan(&waiting); e != nil {
					t.Fatal(e)
				}
				if waiting {
					break
				}
				time.Sleep(10 * time.Millisecond)
			}
			if !waiting {
				t.Fatal("actual HTTP request did not reach controlled DB lock")
			}
			if delay := time.Until(expires) + 30*time.Millisecond; delay > 0 {
				time.Sleep(delay)
			}
			if e = blocker.Rollback(ctx); e != nil {
				t.Fatal(e)
			}
			r := <-done
			if r.err != nil || r.status != 403 || strings.Contains(string(r.raw), "review_ticket") {
				t.Fatal(r.status, string(r.raw), r.err)
			}
			var n int
			if e = f.server.DB.QueryRow(ctx, `SELECT count(*) FROM database_rows WHERE database_id=$1`, f.dbID).Scan(&n); e != nil || n != 0 {
				t.Fatal("expired HTTP request created row", n, e)
			}
		})
	}
}
