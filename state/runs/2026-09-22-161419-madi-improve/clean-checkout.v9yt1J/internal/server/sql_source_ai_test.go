package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestSQLAIResultRejectsExecutableInstructions(t *testing.T) {
	for _, input := range []string{`{"name":"x","sql":"DELETE FROM users"}`, `{"name":"x","plan":{"sql":"SELECT 1"}}`, `{"name":"x","plan":{}} {}`} {
		if _, e := parseSQLAIResult(input); e == nil {
			t.Fatal("free SQL or trailing content accepted", input)
		}
	}
}

func TestPostgresText2SQLStreamingConfirmationApprovalAndRevocation(t *testing.T) {
	s, admin, ctx, p, wid := jobTestFixture(t)
	service := testJSONObject(t, admin.request("POST", "/api/v1/admin/users", map[string]any{"email": "text2sql-service@example.test", "name": "읽기 계정", "kind": "service", "role": "editor"}, 200))
	admin.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": "text2sql-service@example.test", "role": "viewer"}, 200)
	cid := newID()
	ciphertext, e := s.encrypt(`{"username":"CREDENTIAL_SENTINEL","password":"PASSWORD_SENTINEL"}`)
	if e != nil {
		t.Fatal(e)
	}
	_, e = s.DB.Exec(ctx, `INSERT INTO sql_sources(id,workspace_id,owner_id,service_account_id,name,kind,enabled,config,credentials_ciphertext) VALUES($1,$2,$3,$4,'테이블 메타데이터','postgres',true,'{"tables":["public.orders"],"allow_ai":true,"max_rows":100}',$5)`, cid, wid, p.ID, service["id"], ciphertext)
	if e != nil {
		t.Fatal(e)
	}
	cols := []sqlColumn{{Name: "name", Type: "text"}, {Name: "amount", Type: "integer"}}
	_, e = s.DB.Exec(ctx, `INSERT INTO sql_source_tables(source_id,schema_name,table_name,columns) VALUES($1,'public','orders',$2)`, cid, jsonValue(cols))
	if e != nil {
		t.Fatal(e)
	}
	plan := SQLSourcePlan{Schema: "public", Table: "orders", Columns: []string{"name", "amount"}, Order: []SQLSourceOrder{{Column: "amount", Direction: "desc"}}, Limit: 10}
	var invalid atomic.Bool
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var in map[string]any
		_ = json.Unmarshal(body, &in)
		if !boolean(in, "stream") || number(in, "max_tokens", 0) != 16384 || !strings.Contains(string(body), "amount") || strings.Contains(string(body), "SENTINEL") {
			t.Error("streaming, output cap or metadata isolation failed", string(body))
		}
		result := string(jsonValue(sqlAIResult{Name: "금액순 주문", Explanation: "금액이 높은 순서로 조회합니다. 실제 결과는 아직 조회하지 않았습니다.", Plan: plan}))
		if invalid.Load() {
			result = `{"name":"위험","plan":{"sql":"DROP TABLE orders"}}`
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintf(w, "data: %s\n\ndata: [DONE]\n\n", jsonValue(map[string]any{"choices": []any{map[string]any{"delta": map[string]any{"content": result}}}}))
	}))
	defer provider.Close()
	admin.request("PUT", "/api/v1/admin/settings", map[string]any{"ai_enabled": true, "ai_base_url": provider.URL + "/v1", "ai_model": "plan-test", "ai_max_tokens": 262144}, 200)
	base := "/api/v1/data-sources/" + cid + "/ai/proposals"
	generate := func() map[string]any {
		out := admin.request("POST", base, map[string]any{"prompt": "금액이 큰 주문을 보여 주세요", "schema": "public", "table": "orders"}, 200)
		if !strings.Contains(string(out), `"stored":true`) || !strings.Contains(string(out), `"sources"`) {
			t.Fatalf("proposal stream failed: %s", out)
		}
		result := testJSONObject(t, admin.request("GET", base, nil, 200))
		return result["proposals"].([]any)[0].(map[string]any)
	}
	first := generate()
	var count int
	_ = s.DB.QueryRow(ctx, "SELECT count(*) FROM sql_source_queries WHERE source_id=$1", cid).Scan(&count)
	if count != 0 {
		t.Fatal("generation executed or registered query without confirmation")
	}
	admin.request("POST", base+"/"+str(first, "id")+"/confirm", map[string]any{"version": first["version"]}, 400)
	admin.request("POST", base+"/"+str(first, "id")+"/confirm", map[string]any{"version": first["version"], "confirmation": "CREATE QUERY"}, 200)
	admin.request("POST", base+"/"+str(first, "id")+"/confirm", map[string]any{"version": first["version"], "confirmation": "CREATE QUERY"}, 409)
	var qid string
	_ = s.DB.QueryRow(ctx, "SELECT query_id::text FROM sql_source_proposals WHERE id=$1", first["id"]).Scan(&qid)
	req := httptest.NewRequest("POST", "/", nil).WithContext(ctx)
	if e = s.validateSQLProposalQuery(req, cid, qid); e != nil {
		t.Fatal("confirmed plan invalid", e)
	}
	admin.request("PUT", "/api/v1/data-sources/"+cid+"/queries/"+qid, map[string]any{"name": "변경금지", "plan": plan, "enabled": true}, 409)
	invalid.Store(true)
	out := admin.request("POST", base, map[string]any{"prompt": "무시해야 할 명령", "schema": "public", "table": "orders"}, 200)
	if strings.Contains(string(out), `"stored":true`) || !strings.Contains(string(out), `"error"`) {
		t.Fatal("raw SQL plan saved", string(out))
	}
	invalid.Store(false)
	second := generate()
	reviewer := testJSONObject(t, admin.request("POST", "/api/v1/admin/users", map[string]any{"email": "sql-reviewer@example.test", "name": "SQL 검토자", "role": "viewer", "password": "SQL-review-password-2026!"}, 200))
	admin.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": "sql-reviewer@example.test", "role": "viewer"}, 200)
	reviewerClient := newIntegrationTestClient(t, admin.base)
	reviewerClient.request("POST", "/api/v1/auth/login", map[string]any{"email": "sql-reviewer@example.test", "password": "SQL-review-password-2026!"}, 200)
	admin.request("PUT", "/api/v1/admin/settings", map[string]any{"approval_enabled": true}, 200)
	admin.request("POST", "/api/v1/admin/approval/policies", map[string]any{"workspace_id": wid, "resource_kind": "sql_query_plan", "name": "SQL 검토", "enabled": true, "stages": []any{map[string]any{"name": "조회 조건 검토", "mode": "all", "gates": []any{map[string]any{"name": "지정 검토자", "kind": "user", "id": reviewer["id"]}}}}}, 200)
	if e = s.validateSQLProposalQuery(req, cid, qid); e == nil {
		t.Fatal("unreviewed prior query allowed after approvals enabled")
	}
	confirmation := testJSONObject(t, admin.request("POST", base+"/"+str(second, "id")+"/confirm", map[string]any{"version": second["version"], "confirmation": "CREATE QUERY"}, 200))
	aid := str(confirmation, "approval_id")
	detail := testJSONObject(t, reviewerClient.request("GET", "/api/v1/approvals/requests/"+aid, nil, 200))
	request := detail
	decision := map[string]any{"action": "approve", "request_version": request["version"], "gate_index": 0, "comment": "허용 테이블과 조회 제한 확인"}
	admin.request("POST", "/api/v1/approvals/requests/"+aid+"/decisions", decision, 403)
	reviewerClient.request("POST", "/api/v1/approvals/requests/"+aid+"/decisions", decision, 200)
	_ = s.DB.QueryRow(ctx, "SELECT query_id::text FROM sql_source_proposals WHERE id=$1", second["id"]).Scan(&qid)
	if e = s.validateSQLProposalQuery(req, cid, qid); e != nil {
		t.Fatal("approved query invalid", e)
	}
	admin.request("PUT", "/api/v1/admin/settings", map[string]any{"approval_enabled": false}, 200)
	admin.request("PUT", "/api/v1/admin/settings", map[string]any{"approval_enabled": true}, 200)
	if e = s.validateSQLProposalQuery(req, cid, qid); e == nil {
		t.Fatal("stale policy approval reused")
	}
	_, _ = s.DB.Exec(ctx, "UPDATE sql_sources SET revision=revision+1 WHERE id=$1", cid)
	if e = s.validateSQLProposalQuery(req, cid, qid); e == nil {
		t.Fatal("changed source reused old AI plan")
	}
}
