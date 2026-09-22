package server

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestTaskDocumentQueryBoundsAfterACLBeforeStatistics(t *testing.T) {
	for _, tc := range []struct {
		workspace, document string
		args                int
	}{
		{"", "", 1}, {newID(), "", 2}, {"", newID(), 2}, {newID(), newID(), 3},
	} {
		query, args := taskDocumentQuery(newID(), tc.workspace, tc.document)
		if len(args) != tc.args || strings.Contains(query, "workspace_id::text") || strings.Contains(query, "count(*) OVER") {
			t.Fatal("query must retain UUID predicates and bounded statistics", query, args)
		}
		acl, limit, window := strings.Index(query, docACL), strings.Index(query, "LIMIT 2001"), strings.Index(query, "row_number()")
		if acl < 0 || limit <= acl || window <= limit || !strings.Contains(query, "visible AS MATERIALIZED") {
			t.Fatal("visibility must precede limit, and statistics must follow it", query)
		}
	}
}

func TestPostgresTaskReadBoundsZeroBudgetAndIndexes(t *testing.T) {
	s, server := integrationTestServer(t)
	c := newIntegrationTestClient(t, server.URL)
	me := testJSONObject(t, c.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200))
	actor := str(me, "id")
	workspace := testJSONObject(t, c.request("POST", "/api/v1/workspaces", map[string]any{"name": "할 일 읽기 상한"}, 200))
	wid := str(workspace, "id")
	other := testJSONObject(t, c.request("POST", "/api/v1/admin/users", map[string]any{"email": "task-read-owner@example.test", "name": "비공개 문서 소유자", "password": "Task-Read-Test-Password!", "role": "editor"}, 200))
	c.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": other["email"], "role": "editor"}, 200)
	ctx := t.Context()
	// Newer private documents are deliberately before all visible ones. They
	// must neither consume the visible limit nor inflate the reported count.
	_, e := s.DB.Exec(ctx, `INSERT INTO documents(id,workspace_id,title,markdown,owner_id,visibility,updated_at)
 SELECT md5('task-visible-'||g)::uuid,$1,'표시 문서 '||g,'- [ ] 표시 작업 '||g,$2,'workspace',now()-g*interval '1 second' FROM generate_series(1,12000) g`, wid, actor)
	if e != nil {
		t.Fatal(e)
	}
	_, e = s.DB.Exec(ctx, `INSERT INTO documents(id,workspace_id,title,markdown,owner_id,visibility,updated_at)
 SELECT md5('task-private-'||g)::uuid,$1,'비공개 표식 '||g,'- [ ] 숨겨야 하는 작업 '||g,$2,'private',now()+g*interval '1 second' FROM generate_series(1,1000) g`, wid, other["id"])
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.DB.Exec(ctx, "ANALYZE documents"); e != nil {
		t.Fatal(e)
	}
	board := testJSONObject(t, c.request("GET", "/api/v1/tasks/board?workspace_id="+wid, nil, 200))
	items, ok := board["items"].([]any)
	if !ok || len(items) != 2000 || number(board, "documents_scanned", 0) != 2000 || number(board, "total_documents", 0) != 2001 || boolean(board, "total_documents_exact") || !boolean(board, "total_documents_is_lower_bound") || !boolean(board, "truncated") {
		t.Fatal("bounded visible count and sentinel", len(items), board["documents_scanned"], board["total_documents"], board["truncated"])
	}
	for _, item := range items {
		if strings.Contains(string(jsonValue(item)), "비공개") || strings.Contains(string(jsonValue(item)), "숨겨야") {
			t.Fatal("private data leaked")
		}
	}
	first := items[0].(map[string]any)
	last := items[len(items)-1].(map[string]any)
	if str(first, "text") != "표시 작업 1" || str(last, "text") != "표시 작업 2000" {
		t.Fatal("visible order changed", first, last)
	}
	query, args := taskDocumentQuery(actor, wid, "")
	var raw []byte
	if e = s.DB.QueryRow(ctx, "EXPLAIN (ANALYZE,BUFFERS,FORMAT JSON) "+query, args...).Scan(&raw); e != nil {
		t.Fatal(e)
	}
	var plan any
	if e = json.Unmarshal(raw, &plan); e != nil {
		t.Fatal(e)
	}
	foundIndex, foundWindow := false, false
	walkTaskReadPlan(plan, func(node map[string]any) {
		if str(node, "Index Name") == "documents_active_workspace_recent_idx" {
			foundIndex = true
		}
		if str(node, "Node Type") == "WindowAgg" {
			foundWindow = true
			if number(node, "Actual Rows", 0) > 2001 {
				t.Fatal("window inspected unbounded visible documents", node)
			}
		}
	})
	if !foundIndex || !foundWindow {
		t.Fatal("ordered task index and bounded window expected", string(raw))
	}
	if reportPath := os.Getenv("MADI_TASK_PLAN_REPORT"); reportPath != "" {
		recordTaskPreparedPlans(t, s, query, actor, wid, reportPath)
	}
	// Document filtering is not limited to the first workspace page.
	var older, private string
	if e = s.DB.QueryRow(ctx, "SELECT md5('task-visible-2200')::uuid::text,md5('task-private-1')::uuid::text").Scan(&older, &private); e != nil {
		t.Fatal(e)
	}
	exact := testJSONObject(t, c.request("GET", "/api/v1/tasks/board?workspace_id="+wid+"&document_id="+older, nil, 200))
	if number(exact, "total_documents", 0) != 1 || !boolean(exact, "total_documents_exact") || boolean(exact, "truncated") {
		t.Fatal(exact)
	}
	hidden := testJSONObject(t, c.request("GET", "/api/v1/tasks/board?workspace_id="+wid+"&document_id="+private, nil, 200))
	if number(hidden, "total_documents", -1) != 0 || !boolean(hidden, "total_documents_exact") || boolean(hidden, "truncated") {
		t.Fatal("hidden document existence leaked", hidden)
	}
	// The legacy all-workspace read uses the same canonical parser and UUID
	// argument builder without supplying nonexistent query parameters.
	c.request("GET", "/api/v1/tasks", nil, 200)
	// Single-document comment/attachment lists have stable index-backed order.
	_, e = s.DB.Exec(ctx, `INSERT INTO comments(id,document_id,user_id,body,created_at)
 SELECT md5('task-read-comment-'||g)::uuid,md5('task-visible-'||((g-1)%2200+1))::uuid,$1,'내용',now()+g*interval '1 second' FROM generate_series(1,8800) g`, actor)
	if e != nil {
		t.Fatal(e)
	}
	_, e = s.DB.Exec(ctx, `INSERT INTO attachments(id,document_id,user_id,name,content_type,size,path,created_at)
 SELECT md5('task-read-file-'||g)::uuid,md5('task-visible-'||((g-1)%2200+1))::uuid,$1,'문서.txt','text/plain',0,'fixture-unused',now()+g*interval '1 second' FROM generate_series(1,8800) g`, actor)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.DB.Exec(ctx, "ANALYZE comments; ANALYZE attachments"); e != nil {
		t.Fatal(e)
	}
	for _, table := range []string{"comments", "attachments"} {
		var report []byte
		if e = s.DB.QueryRow(ctx, fmt.Sprintf("EXPLAIN (ANALYZE,BUFFERS,FORMAT JSON) SELECT id FROM %s WHERE document_id=$1 ORDER BY created_at,id LIMIT 200", table), older).Scan(&report); e != nil {
			t.Fatal(e)
		}
		var parsed any
		if e = json.Unmarshal(report, &parsed); e != nil {
			t.Fatal(e)
		}
		found := false
		walkTaskReadPlan(parsed, func(node map[string]any) {
			if str(node, "Index Name") == table+"_document_created_idx" {
				found = true
			}
		})
		if !found {
			t.Fatal("single document read missed index", table, string(report))
		}
	}
	// Direct legacy data can exceed today's API body limit. Keep the bounded
	// count/truncation sentinel even when no source fits in the body budget.
	_, e = s.DB.Exec(ctx, `INSERT INTO documents(id,workspace_id,title,markdown,owner_id,visibility,updated_at)
 VALUES($1,$2,'예산을 넘는 과거 원문',repeat('x ',8388609),$3,'workspace',now()+interval '1 day')`, newID(), wid, actor)
	if e != nil {
		t.Fatal(e)
	}
	budget := testJSONObject(t, c.request("GET", "/api/v1/tasks/board?workspace_id="+wid, nil, 200))
	if number(budget, "documents_scanned", -1) != 0 || number(budget, "total_documents", 0) != 2001 || !boolean(budget, "truncated") || boolean(budget, "total_documents_exact") {
		t.Fatal("zero-budget state lost", budget)
	}
}

// Opt-in diagnostic only: all samples share one connection and the same
// read-only snapshot. Alternating order avoids attributing a warm-cache effect
// to generic/custom planning. The plans contain fixture UUIDs, not user data.
func recordTaskPreparedPlans(t *testing.T, s *Server, query, actor, workspace, reportPath string) {
	t.Helper()
	if !validID(actor) || !validID(workspace) {
		t.Fatal("prepared-plan diagnostic requires fixture UUIDs")
	}
	ctx := t.Context()
	tx, e := s.DB.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback(ctx)
	if _, e = tx.Exec(ctx, "PREPARE madi_task_plan_fixture(uuid,uuid) AS "+query); e != nil {
		t.Fatal(e)
	}
	defer tx.Exec(ctx, "DEALLOCATE madi_task_plan_fixture")
	execute := fmt.Sprintf("EXPLAIN (ANALYZE,BUFFERS,FORMAT JSON) EXECUTE madi_task_plan_fixture('%s'::uuid,'%s'::uuid)", actor, workspace)
	report := []map[string]any{}
	for _, mode := range []string{"force_custom_plan", "force_generic_plan", "force_generic_plan", "force_custom_plan"} {
		if _, e = tx.Exec(ctx, "SET LOCAL plan_cache_mode="+mode); e != nil {
			t.Fatal(e)
		}
		var raw []byte
		if e = tx.QueryRow(ctx, execute).Scan(&raw); e != nil {
			t.Fatal(e)
		}
		var plan []map[string]any
		if e = json.Unmarshal(raw, &plan); e != nil || len(plan) != 1 {
			t.Fatalf("invalid plan report: %v", e)
		}
		report = append(report, map[string]any{"mode": mode, "plan": json.RawMessage(raw)})
		t.Logf("task prepared mode=%s execution_ms=%v planning_ms=%v", mode, plan[0]["Execution Time"], plan[0]["Planning Time"])
	}
	data, e := json.MarshalIndent(map[string]any{"fixture_documents": 13000, "visible_documents": 12000, "same_readonly_snapshot": true, "samples": report}, "", "  ")
	if e != nil {
		t.Fatal(e)
	}
	if e = os.MkdirAll(filepath.Dir(reportPath), 0o700); e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(reportPath, append(data, '\n'), 0o600); e != nil {
		t.Fatal(e)
	}
}

func walkTaskReadPlan(value any, visit func(map[string]any)) {
	switch v := value.(type) {
	case map[string]any:
		visit(v)
		for _, child := range v {
			walkTaskReadPlan(child, visit)
		}
	case []any:
		for _, child := range v {
			walkTaskReadPlan(child, visit)
		}
	}
}
