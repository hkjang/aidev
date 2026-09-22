package server

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func structuredGenerationInput(f structuredFixture, meta map[string]any) map[string]any {
	return map[string]any{"database_id": f.dbID, "expected_version": meta["version"], "start_byte": 0, "end_byte": f.ticket.End, "selected_text": "근거 수량은 2개입니다.", "property_ids": []string{"amount"}, "provider_fingerprint": meta["provider"].(map[string]any)["fingerprint"], "schema_hash": meta["schema_hash"], "destination_hash": meta["destination_hash"], "consent": true}
}

func TestPostgresStructuredGeneratedPIINeverLeavesSSE(t *testing.T) {
	for _, mode := range []string{"block", "mask"} {
		t.Run(mode, func(t *testing.T) {
			f := newStructuredFixture(t)
			f.owner.request("PUT", "/api/v1/databases/"+f.dbID, map[string]any{"properties": []any{map[string]any{"id": "amount", "name": "수량 설명", "type": "text"}}}, 200)
			provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				// The sensitive term was not in the selected input and crosses deltas.
				for _, delta := range []string{`{"fields":[{"property_id":"amount","value":"BLOCKED_`, `SECRET_TOKEN","quote":"2개"}]}`} {
					fmt.Fprintf(w, "data: %s\n\n", jsonValue(map[string]any{"choices": []any{map[string]any{"delta": map[string]string{"content": delta}}}}))
					w.(http.Flusher).Flush()
				}
				fmt.Fprint(w, "data: [DONE]\n\n")
			}))
			defer provider.Close()
			f.owner.request("PUT", "/api/v1/admin/settings", map[string]any{"ai_base_url": provider.URL + "/v1"}, 200)
			cfg := defaultProtectionSettings()
			cfg["enabled"], cfg["mode"], cfg["custom_terms"] = true, mode, []string{"BLOCKED_SECRET_TOKEN"}
			if _, e := f.server.DB.Exec(t.Context(), `UPDATE protection_settings SET data=$1`, jsonValue(cfg)); e != nil {
				t.Fatal(e)
			}
			path := "/api/v1/documents/" + f.docID
			meta := testJSONObject(t, f.owner.request("GET", path+"/structured-context?database_id="+f.dbID, nil, 200))
			raw := string(f.owner.request("POST", path+"/structured-draft", structuredGenerationInput(f, meta), 200))
			for _, forbidden := range []string{"BLOCKED_", "SECRET_TOKEN", `"text":`, `"proposal":`, "draft_ticket"} {
				if strings.Contains(raw, forbidden) {
					t.Errorf("unvalidated output escaped via SSE (%s): %s", forbidden, raw)
				}
			}
			if !strings.Contains(raw, `"phase":"generating"`) || !strings.Contains(raw, `"retract":true`) {
				t.Fatal("missing content-free progress or rejection", raw)
			}
			var drafts, rows int
			if e := f.server.DB.QueryRow(t.Context(), `SELECT (SELECT count(*) FROM knowledge_structured_drafts),(SELECT count(*) FROM database_rows WHERE database_id=$1)`, f.dbID).Scan(&drafts, &rows); e != nil || drafts != 1 || rows != 0 {
				t.Fatal("rejected generated PII was persisted", drafts, rows, e)
			}
		})
	}
}

// Pause immediately before the requested policy read, then let PostgreSQL itself
// block on its FOR SHARE. This selects the stream's pre-transmission guard (the
// second read), not merely the first HTTP preflight. No production test hook.
type structuredPolicyReadTrace struct {
	target int32
	reads  atomic.Int32
	ready  chan struct{}
	resume chan struct{}
}

func (trace *structuredPolicyReadTrace) TraceQueryStart(ctx context.Context, _ *pgx.Conn, q pgx.TraceQueryStartData) context.Context {
	if q.SQL == "SELECT data FROM protection_settings WHERE id=1 FOR SHARE" && trace.reads.Add(1) == trace.target {
		close(trace.ready)
		select {
		case <-trace.resume:
		case <-ctx.Done():
		}
	}
	return ctx
}
func (*structuredPolicyReadTrace) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}

func TestPostgresStructuredFinalPolicyLockRevalidation(t *testing.T) {
	for _, operation := range []string{"context", "generate"} {
		for _, change := range []string{"session", "ancestor", "destination", "provider"} {
			t.Run(operation+"/"+change, func(t *testing.T) {
				f := newStructuredFixture(t)
				ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
				defer cancel()
				originalPool := f.server.DB
				parent := newID()
				otherID := str(testJSONObject(t, f.other.request("GET", "/api/v1/auth/me", nil, 200)), "id")
				if _, e := originalPool.Exec(ctx, `INSERT INTO documents(id,workspace_id,title,markdown,owner_id,visibility) VALUES($1,$2,'경계 상위','',$3,'workspace');`, parent, f.wid, otherID); e != nil {
					t.Fatal(e)
				}
				if _, e := originalPool.Exec(ctx, `UPDATE documents SET parent_id=$2 WHERE id=$1`, f.docID, parent); e != nil {
					t.Fatal(e)
				}
				var transmissions atomic.Int32
				provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					transmissions.Add(1)
					w.Header().Set("Content-Type", "text/event-stream")
					fmt.Fprintf(w, "data: %s\n\ndata: [DONE]\n\n", jsonValue(map[string]any{"choices": []any{map[string]any{"delta": map[string]string{"content": `{"fields":[{"property_id":"amount","value":2,"quote":"2개"}]}`}}}}))
				}))
				defer provider.Close()
				f.owner.request("PUT", "/api/v1/admin/settings", map[string]any{"ai_base_url": provider.URL + "/v1"}, 200)
				path := "/api/v1/documents/" + f.docID
				meta := testJSONObject(t, f.owner.request("GET", path+"/structured-context?database_id="+f.dbID, nil, 200))
				// Missing override is intentional: insertion is not fenced by an
				// existing workspace-settings row's FOR SHARE lock.
				if _, e := originalPool.Exec(ctx, `DELETE FROM workspace_settings WHERE workspace_id=$1`, f.wid); e != nil {
					t.Fatal(e)
				}
				trace := &structuredPolicyReadTrace{target: 1, ready: make(chan struct{}), resume: make(chan struct{})}
				if operation == "generate" {
					trace.target = 2
				}
				config := originalPool.Config()
				config.ConnConfig.Tracer = trace
				pool, e := pgxpool.NewWithConfig(ctx, config)
				if e != nil {
					t.Fatal(e)
				}
				f.server.DB = pool
				defer func() { f.server.DB = originalPool; pool.Close() }()
				method, route := "GET", path+"/structured-context?database_id="+f.dbID
				var body io.Reader
				if operation == "generate" {
					method, route, body = "POST", path+"/structured-draft", bytes.NewReader(jsonValue(structuredGenerationInput(f, meta)))
				}
				req, e := http.NewRequestWithContext(ctx, method, f.owner.base+route, body)
				if e != nil {
					t.Fatal(e)
				}
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-Madi-Request", "1")
				client := f.owner.client
				var expires time.Time
				if change == "session" {
					secret := "structured-final-policy-" + operation
					if e = originalPool.QueryRow(ctx, `INSERT INTO sessions(token_hash,user_id,expires_at) VALUES($1,$2,clock_timestamp()+interval '1500 milliseconds') RETURNING expires_at`, digest(secret), f.ticket.OwnerID).Scan(&expires); e != nil {
						t.Fatal(e)
					}
					req.AddCookie(&http.Cookie{Name: "madi_session", Value: secret})
					client = http.DefaultClient
				}
				type result struct {
					code int
					raw  string
					err  error
				}
				done := make(chan result, 1)
				go func() {
					response, e := client.Do(req)
					if e != nil {
						done <- result{err: e}
						return
					}
					defer response.Body.Close()
					raw, e := io.ReadAll(response.Body)
					done <- result{response.StatusCode, string(raw), e}
				}()
				select {
				case <-trace.ready:
				case <-ctx.Done():
					t.Fatal("policy guard was not reached", ctx.Err())
				}
				blocker, e := originalPool.Begin(ctx)
				if e != nil {
					t.Fatal(e)
				}
				defer blocker.Rollback(context.WithoutCancel(ctx))
				if _, e = blocker.Exec(ctx, `SELECT data FROM protection_settings WHERE id=1 FOR UPDATE`); e != nil {
					t.Fatal(e)
				}
				close(trace.resume)
				waiting := false
				for deadline := time.Now().Add(time.Second); time.Now().Before(deadline); {
					if e = originalPool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE $1::integer=ANY(pg_blocking_pids(pid)))`, int32(blocker.Conn().PgConn().PID())).Scan(&waiting); e != nil {
						t.Fatal(e)
					}
					if waiting {
						break
					}
					time.Sleep(5 * time.Millisecond)
				}
				if !waiting {
					t.Fatal("HTTP did not wait on actual protection row lock")
				}
				switch change {
				case "session":
					if delay := time.Until(expires) + 25*time.Millisecond; delay > 0 {
						time.Sleep(delay)
					}
				case "ancestor":
					_, e = originalPool.Exec(ctx, `UPDATE documents SET visibility='private' WHERE id=$1`, parent)
				case "destination":
					_, e = originalPool.Exec(ctx, `UPDATE workspace_members SET role=CASE WHEN role='viewer' THEN 'editor' ELSE 'viewer' END WHERE workspace_id=$1 AND user_id=$2`, f.wid, otherID)
				case "provider":
					_, e = originalPool.Exec(ctx, `INSERT INTO workspace_settings(workspace_id,data) VALUES($1,'{"ai_model":"changed-after-consent"}')`, f.wid)
				}
				if e != nil {
					t.Fatal(e)
				}
				if e = blocker.Rollback(ctx); e != nil {
					t.Fatal(e)
				}
				got := <-done
				if got.err != nil || got.code < 400 || got.code >= 500 || strings.Contains(got.raw, "근거 수량") || strings.Contains(got.raw, "draft_ticket") {
					t.Errorf("late %s not rejected: HTTP%d %s %v", change, got.code, got.raw, got.err)
				}
				if transmissions.Load() != 0 {
					t.Errorf("sent selected source after %s changed: %d requests", change, transmissions.Load())
				}
			})
		}
	}
}
