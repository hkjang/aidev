package server

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

type structuredFixture struct {
	server               *Server
	owner, other         *integrationTestClient
	wid, docID, dbID, id string
	ticket               structuredTicket
	save                 map[string]any
}

func newStructuredFixture(t *testing.T) structuredFixture {
	t.Helper()
	s, c, other, wid, _ := collaborationTestSetup(t)
	c.request("PUT", "/api/v1/admin/settings", map[string]any{"ai_enabled": true, "ai_base_url": "http://127.0.0.1:11434/v1", "ai_model": "sealed-fixture-no-provider-call"}, 200)
	props := []map[string]any{{"id": "amount", "name": "수량", "type": "number"}}
	dbID := str(testJSONObject(t, c.request("POST", "/api/v1/databases", map[string]any{"workspace_id": wid, "name": "현재 대상", "properties": props}, 200)), "id")
	md := "근거 수량은 2개입니다."
	docID := str(testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "구조화 근거", "markdown": md, "visibility": "private"}, 200)), "id")
	meta := testJSONObject(t, c.request("GET", "/api/v1/documents/"+docID+"/structured-context?database_id="+dbID, nil, 200))
	uid := str(testJSONObject(t, c.request("GET", "/api/v1/auth/me", nil, 200)), "id")
	payload, e := parseStructuredOutput(`{"fields":[{"property_id":"amount","value":2,"quote":"2개"}]}`, md, 0, props)
	if e != nil {
		t.Fatal(e)
	}
	ticket := structuredTicket{Kind: "madi-structured-draft-v1", ID: newID(), OwnerID: uid, DocumentID: docID, DatabaseID: dbID, WorkspaceID: wid, Version: 1, SourceHash: digest(md), Start: 0, End: len(md), Schema: str(meta, "schema_hash"), Destination: str(meta, "destination_hash"), Provider: str(meta["provider"].(map[string]any), "fingerprint"), Model: "sealed-fixture-no-provider-call", Payload: payload, Expires: time.Now().Add(30 * time.Minute).Unix()}
	sealed, e := s.encrypt(string(jsonValue(ticket)))
	if e != nil {
		t.Fatal(e)
	}
	save := map[string]any{"ticket": sealed, "consent": true}
	c.request("POST", "/api/v1/knowledge/structured-drafts", save, 201)
	return structuredFixture{s, c, other, wid, docID, dbID, ticket.ID, ticket, save}
}
func (f structuredFixture) preview(t *testing.T) map[string]any {
	t.Helper()
	return testJSONObject(t, f.owner.request("POST", "/api/v1/knowledge/structured-drafts/"+f.id+"/preview", map[string]any{"revision": 1, "fields": []structuredProviderField{{PropertyID: "amount", Value: 2, Quote: "2개"}}}, 200))
}
func TestPostgresStructuredDraftCurrentScopePIIExpiryAndMCP(t *testing.T) {
	for _, reason := range []string{"source", "schema", "destination", "protection", "expiry", "key"} {
		t.Run(reason, func(t *testing.T) {
			f := newStructuredFixture(t)
			s, c, ctx := f.server, f.owner, t.Context()
			path := "/api/v1/knowledge/structured-drafts/" + f.id
			preview := f.preview(t)
			apply := map[string]any{"review_ticket": preview["review_ticket"], "consent": true}
			switch reason {
			case "source":
				c.request("PUT", "/api/v1/documents/"+f.docID, map[string]any{"version": 1, "markdown": "수량은3개"}, 200)
			case "schema":
				c.request("PUT", "/api/v1/databases/"+f.dbID, map[string]any{"properties": []any{map[string]any{"id": "amount", "name": "새 수량", "type": "number"}}}, 200)
			case "destination":
				otherID := str(testJSONObject(t, f.other.request("GET", "/api/v1/auth/me", nil, 200)), "id")
				if _, e := s.DB.Exec(ctx, `UPDATE workspace_members SET role=CASE WHEN role='viewer' THEN 'editor' ELSE 'viewer' END WHERE workspace_id=$1 AND user_id=$2`, f.wid, otherID); e != nil {
					t.Fatal(e)
				}
			case "protection":
				policy := defaultProtectionSettings()
				policy["enabled"], policy["mode"], policy["custom_terms"] = true, "mask", []string{"2개"}
				if _, e := s.DB.Exec(ctx, `UPDATE protection_settings SET data=$1`, jsonValue(policy)); e != nil {
					t.Fatal(e)
				}
				c.request("GET", path, nil, 403)
				c.request("POST", path+"/apply", apply, 403)
				return
			case "expiry":
				if _, e := s.DB.Exec(ctx, `UPDATE knowledge_structured_drafts SET expires_at=now()-interval '1 second' WHERE id=$1`, f.id); e != nil {
					t.Fatal(e)
				}
				c.request("GET", path, nil, 404)
				c.request("POST", path+"/apply", apply, 404)
				return
			case "key":
				key := testJSONObject(t, c.request("POST", "/api/v1/keys", map[string]any{"name": "구조화 읽기", "workspace_id": f.wid, "scopes": []string{"document:read", "database:read"}}, 201))
				c.token = str(key, "token")
				c.request("GET", path, nil, 200)
				c.request("POST", path+"/apply", apply, 403)
				c.request("GET", "/api/v1/documents/"+f.docID+"/structured-context?database_id="+f.dbID, nil, 403)
				for _, tool := range []struct {
					name string
					args map[string]any
				}{{"get_structured_draft", map[string]any{"draft_id": f.id}}, {"list_structured_drafts", map[string]any{}}} {
					result := testJSONObject(t, c.request("POST", "/api/v1/mcp", map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": tool.name, "arguments": tool.args}}, 200))
					response, ok := result["result"].(map[string]any)
					if !ok || boolean(response, "isError") || !strings.Contains(string(jsonValue(response)), f.id) {
						t.Fatal(result)
					}
				}
				c.token = ""
				c.request("PUT", "/api/v1/keys/"+str(key["key"].(map[string]any), "id"), map[string]any{"scopes": []string{"document:read"}}, 200)
				c.token = str(key, "token")
				c.request("GET", path, nil, 403)
				return
			}
			v := testJSONObject(t, c.request("GET", path, nil, 200))
			if boolean(v, "fresh") {
				t.Fatal("changed dependency marked current", reason, v)
			}
			c.request("POST", path+"/apply", apply, 409)
			var count int
			if e := s.DB.QueryRow(ctx, `SELECT count(*) FROM database_rows WHERE database_id=$1`, f.dbID).Scan(&count); e != nil || count != 0 {
				t.Fatal("stale preview wrote row", reason, count, e)
			}
		})
	}
}
func TestPostgresStructuredDraftConcurrentApplyAndSourceDeletionBoundary(t *testing.T) {
	f := newStructuredFixture(t)
	preview := f.preview(t)
	body := jsonValue(map[string]any{"review_ticket": preview["review_ticket"], "consent": true})
	type result struct {
		status int
		raw    []byte
		err    error
	}
	results := make(chan result, 2)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			req, e := http.NewRequestWithContext(t.Context(), "POST", f.owner.base+"/api/v1/knowledge/structured-drafts/"+f.id+"/apply", bytes.NewReader(body))
			if e != nil {
				results <- result{err: e}
				return
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Madi-Request", "1")
			res, e := f.owner.client.Do(req)
			if e != nil {
				results <- result{err: e}
				return
			}
			defer res.Body.Close()
			raw, e := io.ReadAll(res.Body)
			results <- result{res.StatusCode, raw, e}
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	successes := 0
	for r := range results {
		if r.err != nil || r.status != 200 && r.status != 409 {
			t.Fatal(r.status, string(r.raw), r.err)
		}
		if r.status == 200 {
			successes++
		}
	}
	if successes < 1 {
		t.Fatal("no request completed")
	}
	var count int
	if e := f.server.DB.QueryRow(t.Context(), `SELECT count(*) FROM database_rows WHERE database_id=$1`, f.dbID).Scan(&count); e != nil || count != 1 {
		t.Fatal("concurrent apply duplicated", count, e)
	}
	// Provenance deletion is separate from the canonical shared row.
	f.owner.request("DELETE", "/api/v1/knowledge/structured-drafts/"+f.id, map[string]any{"revision": 2, "consent": true}, 200)
	f.owner.request("POST", "/api/v1/knowledge/structured-drafts", f.save, 409)
	if e := f.server.DB.QueryRow(t.Context(), `SELECT count(*) FROM database_rows WHERE database_id=$1`, f.dbID).Scan(&count); e != nil || count != 1 {
		t.Fatal("deleted receipt deleted shared row", count, e)
	}
}
