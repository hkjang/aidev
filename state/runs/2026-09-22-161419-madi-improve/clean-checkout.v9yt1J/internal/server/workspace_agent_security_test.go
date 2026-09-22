package server

import (
	"context"
	"github.com/jackc/pgx/v5"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPostgresWorkspaceAgentDBDependenciesAndHistoryACL(t *testing.T) {
	s, c, wid, uid := agentTestSetup(t)
	props := []map[string]any{{"id": "name", "name": "이름", "type": "text"}, {"id": "count", "name": "수량", "type": "number"}, {"id": "total", "name": "합계", "type": "formula", "expression": "prop(\"count\") * 2"}}
	db := testJSONObject(t, c.request("POST", "/api/v1/databases", map[string]any{"workspace_id": wid, "name": "수량 DB", "properties": props}, 200))
	dbID := str(db, "id")
	row := testJSONObject(t, c.request("POST", "/api/v1/databases/"+dbID+"/rows", map[string]any{"values": map[string]any{"name": "GPU", "count": 4}}, 200))
	doc := testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "원본", "markdown": "내용"}, 200))
	did := str(doc, "id")
	p := &Principal{ID: uid, Role: "admin", WorkspaceID: wid}
	a := workspaceAgent{WorkspaceID: wid, DatabaseIDs: []string{dbID}, DocumentIDs: []string{did}, Tools: []string{"query_database"}}
	q := agentDBQuery{DatabaseID: dbID, PropertyIDs: []string{"name", "total"}, Limit: 10}
	result, hash, e := s.agentQueryDatabase(t.Context(), p, a, q)
	if e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(string(jsonValue(result)), `"total":8`) {
		t.Fatalf("actual formula result %s", jsonValue(result))
	}
	deps := result["_dependencies"]
	if deps == nil {
		t.Fatal("missing actual row/schema dependencies")
	}
	// Persist a real run source so the same-transaction guard is exercised.
	agent := agentTestConfig(t, c, wid, did, []string{"get_document"})
	rid := newID()
	_, e = s.DB.Exec(t.Context(), `INSERT INTO agent_runs(id,agent_id,workspace_id,owner_id,agent_revision,provider_fingerprint,prompt) VALUES($1,$2,$3,$4,1,'x','DB검증')`, rid, str(agent, "id"), wid, uid)
	if e != nil {
		t.Fatal(e)
	}
	src := agentSource{Key: "db:query", Kind: "database", ResourceID: dbID, Snapshot: map[string]any{"query": q, "dependencies": deps}, Fingerprint: hash}
	_, e = s.DB.Exec(t.Context(), `INSERT INTO agent_sources(run_id,source_key,kind,resource_id,snapshot,fingerprint) VALUES($1,$2,$3,$4,$5,$6)`, rid, src.Key, src.Kind, src.ResourceID, jsonValue(src.Snapshot), hash)
	if e != nil {
		t.Fatal(e)
	}
	tx, e := s.DB.Begin(t.Context())
	if e != nil {
		t.Fatal(e)
	}
	e = s.agentSourceTx(t.Context(), tx, p, a, rid, "", 0)
	tx.Rollback(t.Context())
	if e != nil {
		t.Fatalf("unchanged DB guard: %v", e)
	}
	c.request("PUT", "/api/v1/databases/"+dbID+"/rows/"+str(row, "id"), map[string]any{"values": map[string]any{"name": "GPU", "count": 5}}, 200)
	if e = s.validateAgentSources(t.Context(), p, a, rid, true); e == nil {
		t.Fatal("changed computed input accepted")
	}
	tx, e = s.DB.Begin(t.Context())
	if e != nil {
		t.Fatal(e)
	}
	e = s.agentSourceTx(t.Context(), tx, p, a, rid, "", 0)
	tx.Rollback(t.Context())
	if e == nil {
		t.Fatal("same-tx row change accepted")
	}
	// Even if the user can access a relation target, the configured Agent may
	// not traverse it unless that DB is in the explicit allowlist.
	outside := testJSONObject(t, c.request("POST", "/api/v1/databases", map[string]any{"workspace_id": wid, "name": "범위 밖", "properties": []any{map[string]any{"id": "amount", "name": "금액", "type": "number"}}}, 200))
	outsideID := str(outside, "id")
	otherRow := testJSONObject(t, c.request("POST", "/api/v1/databases/"+outsideID+"/rows", map[string]any{"values": map[string]any{"amount": 999}}, 200))
	props = append(props, map[string]any{"id": "relation", "name": "참조", "type": "relation", "target_database_id": outsideID}, map[string]any{"id": "rollup", "name": "집계", "type": "rollup", "relation_property_id": "relation", "target_property_id": "amount", "aggregation": "sum"})
	c.request("PUT", "/api/v1/databases/"+dbID, map[string]any{"name": "수량 DB", "properties": props}, 200)
	c.request("PUT", "/api/v1/databases/"+dbID+"/rows/"+str(row, "id"), map[string]any{"values": map[string]any{"name": "GPU", "count": 5, "relation": []string{str(otherRow, "id")}}}, 200)
	q.PropertyIDs = []string{"rollup"}
	if _, _, e = s.agentQueryDatabase(t.Context(), p, a, q); e == nil {
		t.Fatal("out-of-allowlist rollup traversed")
	}
	a.DatabaseIDs = append(a.DatabaseIDs, outsideID)
	result, _, e = s.agentQueryDatabase(t.Context(), p, a, q)
	if e != nil || !strings.Contains(string(jsonValue(result)), `"rollup":999`) {
		t.Fatalf("authorized actual rollup %v %s", e, jsonValue(result))
	}
}

func TestPostgresWorkspaceAgentStalePlanAndSessionRevocation(t *testing.T) {
	for _, mode := range []string{"stale_plan", "logout"} {
		t.Run(mode, func(t *testing.T) {
			s, c, wid, _ := agentTestSetup(t)
			doc := testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "검증 문서", "markdown": "원문"}, 200))
			did := str(doc, "id")
			call := 0
			provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				call++
				if mode == "logout" {
					c.request("POST", "/api/v1/auth/logout", nil, 200)
					agentTestSSE(w, "LOGGED_OUT_SECRET", "", "")
					return
				}
				if call == 1 {
					agentTestSSE(w, "", "get_document", string(jsonValue(map[string]any{"document_id": did, "start_byte": 0})))
				} else {
					agentTestSSE(w, "", "update_document", string(jsonValue(map[string]any{"document_id": did, "expected_version": 1, "title": "검증 문서", "markdown": "AI 제안"})))
				}
			}))
			defer provider.Close()
			c.request("PUT", "/api/v1/admin/settings", map[string]any{"ai_enabled": true, "ai_base_url": provider.URL, "ai_model": "test"}, 200)
			a := agentTestConfig(t, c, wid, did, []string{"get_document", "update_document"})
			rid := str(testJSONObject(t, c.request("POST", "/api/v1/agents/"+str(a, "id")+"/runs", map[string]any{"prompt": "수정", "expected_agent_version": 1}, 202)), "id")
			if mode == "logout" {
				agentTestPump(t, s, rid, "failed")
				var text string
				s.DB.QueryRow(t.Context(), `SELECT coalesce(string_agg(data::text,''),'') FROM agent_events WHERE run_id=$1`, rid).Scan(&text)
				if strings.Contains(text, "LOGGED_OUT_SECRET") {
					t.Fatal("post-logout output persisted")
				}
				return
			}
			agentTestPump(t, s, rid, "awaiting_confirmation")
			view := testJSONObject(t, c.request("GET", "/api/v1/agent-runs/"+rid, nil, 200))
			action := view["actions"].([]any)[0].(map[string]any)
			c.request("PUT", "/api/v1/documents/"+did, map[string]any{"version": 1, "markdown": "사용자가 먼저 수정"}, 200)
			c.request("POST", "/api/v1/agent-runs/"+rid+"/actions/"+str(action, "id")+"/confirm", map[string]any{"action_hash": action["action_hash"], "confirm": true}, 409)
			current := testJSONObject(t, c.request("GET", "/api/v1/documents/"+did, nil, 200))
			if str(current, "markdown") != "사용자가 먼저 수정" {
				t.Fatal("stale AI plan overwrote document")
			}
			c.request("POST", "/api/v1/agent-runs/"+rid+"/cancel", map[string]any{}, 200)
		})
	}
}

func TestPostgresMutationGuardRollsBackWithoutEffectContext(t *testing.T) {
	s, c, wid, uid := agentTestSetup(t)
	p := &Principal{ID: uid, Role: "admin"}
	ctx := withMutationGuard(context.Background(), func(context.Context, pgx.Tx) error { return errAgentChanged })
	r := automationRequest(ctx, p, "POST", map[string]any{"workspace_id": wid, "title": "MUST_ROLLBACK", "markdown": "not committed"})
	rec := httptest.NewRecorder()
	s.createDocument(rec, r)
	if rec.Code < 400 {
		t.Fatalf("guard bypass HTTP %d", rec.Code)
	}
	var n int
	s.DB.QueryRow(t.Context(), `SELECT count(*) FROM documents WHERE title='MUST_ROLLBACK'`).Scan(&n)
	if n != 0 {
		t.Fatal("mutation committed despite guard without receipt context")
	}
	_ = c
}
