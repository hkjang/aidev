package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"
)

func TestPostgresApprovalSessionExpiresBehindResourceLock(t *testing.T) {
	for _, mode := range []string{"advanced_submit", "resource_status", "request_detail", "decision"} {
		t.Run(mode, func(t *testing.T) {
			s, c, reviewer, wid, source, _, _ := impactExceptionFixture(t)
			var reviewerID string
			if e := s.DB.QueryRow(t.Context(), `SELECT id::text FROM users WHERE email='exception-reviewer@example.test'`).Scan(&reviewerID); e != nil {
				t.Fatal(e)
			}
			c.request("POST", "/api/v1/admin/approval/policies", map[string]any{"workspace_id": wid, "resource_kind": "document", "name": "세션 만료 회귀", "enabled": true, "stages": []any{map[string]any{"name": "검토", "mode": "all", "gates": []any{map[string]any{"name": "검토자", "kind": "user", "id": reviewerID}}}}}, 200)
			method, path, client, actor := "POST", "/api/v1/documents/"+source+"/approval", c, ""
			var payload any = map[string]any{"action": "submit"}
			if e := s.DB.QueryRow(t.Context(), `SELECT id::text FROM users WHERE email='admin@example.test'`).Scan(&actor); e != nil {
				t.Fatal(e)
			}
			if mode != "advanced_submit" {
				c.request("POST", path, payload, 200)
				status := testJSONObject(t, c.request("GET", path, nil, 200))
				req := status["request"].(map[string]any)
				method, payload, client, actor = "GET", nil, reviewer, reviewerID
				switch mode {
				case "resource_status":
					path = "/api/v1/approvals/resources/document/" + source
				case "request_detail":
					path = "/api/v1/approvals/requests/" + str(req, "id")
				case "decision":
					method = "POST"
					path = "/api/v1/approvals/requests/" + str(req, "id") + "/decisions"
					payload = map[string]any{"action": "approve", "request_version": req["version"]}
				}
			}
			blocker, e := s.DB.Begin(t.Context())
			if e != nil {
				t.Fatal(e)
			}
			defer blocker.Rollback(t.Context())
			if _, e = blocker.Exec(t.Context(), `SELECT id FROM documents WHERE id=$1 FOR UPDATE`, source); e != nil {
				t.Fatal(e)
			}
			pid := blocker.Conn().PgConn().PID()
			result := make(chan int, 1)
			go func() {
				raw, _ := json.Marshal(payload)
				req, _ := http.NewRequest(method, c.base+path, bytes.NewReader(raw))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-Madi-Request", "1")
				res, e := client.client.Do(req)
				if e != nil {
					result <- 0
					return
				}
				io.Copy(io.Discard, res.Body)
				res.Body.Close()
				result <- res.StatusCode
			}()
			waiting := false
			deadline := time.Now().Add(3 * time.Second)
			for time.Now().Before(deadline) {
				if e = s.DB.QueryRow(t.Context(), `SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE $1::integer=ANY(pg_blocking_pids(pid)))`, pid).Scan(&waiting); e != nil {
					t.Fatal(e)
				}
				if waiting {
					break
				}
				time.Sleep(10 * time.Millisecond)
			}
			if !waiting {
				t.Fatal("HTTP request never waited on its actual document lock")
			}
			if _, e = s.DB.Exec(t.Context(), `UPDATE sessions SET expires_at=clock_timestamp()-interval '1 second' WHERE user_id=$1`, actor); e != nil {
				t.Fatal(e)
			}
			if e = blocker.Commit(t.Context()); e != nil {
				t.Fatal(e)
			}
			select {
			case code := <-result:
				if code != 403 {
					t.Fatal("expired credential survived resource wait", code)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("request did not finish")
			}
			var decisions int
			if e = s.DB.QueryRow(t.Context(), `SELECT count(*) FROM approval_decisions`).Scan(&decisions); e != nil || decisions != 0 {
				t.Fatal("decision survived rollback", e, decisions)
			}
			if mode == "advanced_submit" {
				var n int
				s.DB.QueryRow(t.Context(), `SELECT count(*) FROM approval_requests WHERE resource_kind='document'`).Scan(&n)
				if n != 0 {
					t.Fatal("expired submission persisted", n)
				}
			}
		})
	}
}
