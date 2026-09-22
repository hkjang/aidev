package server

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// Keep the pre-optimization query as a policy/projection oracle. The production
// query must select precisely the same ordered, currently readable candidates.
func documentQueryRecursiveCandidateSQL(body, tasks bool) string {
	return `SELECT ` + documentQueryDocJSON(body, tasks) + ` FROM documents d LEFT JOIN knowledge_document_meta k ON k.document_id=d.id WHERE d.workspace_id=$2 AND d.deleted_at IS NULL AND madi_document_allowed($1,d.id,false) AND ($3::uuid IS NULL OR d.space_id=$3) AND (cardinality($4::uuid[])=0 OR d.id=ANY($4::uuid[])) ORDER BY d.updated_at DESC,d.id LIMIT 2001`
}

func TestPostgresDocumentQueryCandidateACLAndProjectionOracle(t *testing.T) {
	s, _, ctx, owner, wid := jobTestFixture(t)
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, e := s.DB.Exec(ctx, sql, args...); e != nil {
			t.Fatal(e)
		}
	}
	actors := []string{owner.ID}
	for n, role := range []string{"viewer", "commenter", "editor"} {
		id := newID()
		serviceRole := "editor"
		if role == "viewer" {
			serviceRole = "viewer"
		}
		exec(`INSERT INTO users(id,email,name,role) VALUES($1,$2,$3,$4)`, id, fmt.Sprintf("query-oracle-%d@example.test", n), strings.Repeat("이름", 110), serviceRole)
		exec(`INSERT INTO workspace_members(workspace_id,user_id,role) VALUES($1,$2,$3)`, wid, id, role)
		actors = append(actors, id)
	}
	foreign := newID()
	exec(`INSERT INTO workspaces(id,name,slug) VALUES($1,'다른 공간',$2)`, foreign, foreign)
	exec(`INSERT INTO workspace_members(workspace_id,user_id,role) VALUES($1,$2,'owner')`, foreign, owner.ID)
	space, subspace := newID(), newID()
	exec(`INSERT INTO spaces(id,workspace_id,name,slug,visibility,owner_id) VALUES($1,$2,'제한 공간','query-oracle','restricted',$3)`, space, wid, owner.ID)
	exec(`INSERT INTO spaces(id,workspace_id,parent_id,name,slug,visibility,owner_id) VALUES($1,$2,$3,'상속 공간','query-oracle-child','workspace',$4)`, subspace, wid, space, owner.ID)
	exec(`INSERT INTO space_members(space_id,user_id,role) VALUES($1,$2,'viewer'),($1,$3,'commenter')`, space, actors[1], actors[2])
	doc := func(workspace, author, visibility string, parent, docSpace any) string {
		t.Helper()
		id := newID()
		exec(`INSERT INTO documents(id,workspace_id,owner_id,title,markdown,tags,visibility,parent_id,space_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, id, workspace, author, strings.Repeat("문서", 260), "---\n번호: 42\n---\n- [ ] 실제 할 일\n", jsonValue([]string{"태그", strings.Repeat("x", 501)}), visibility, parent, docSpace)
		exec(`INSERT INTO task_details(document_id,task_id,assignee_id,updated_by) VALUES($1,$2,$3,$4)`, id, newID(), actors[1], owner.ID)
		return id
	}
	var selected string
	for _, author := range actors {
		for _, visibility := range []string{"workspace", "private", "selected"} {
			id := doc(wid, author, visibility, nil, nil)
			exec(`INSERT INTO document_shares(document_id,user_id,permission) VALUES($1,$2,'read'),($1,$3,'write')`, id, actors[1], actors[2])
			_ = doc(wid, actors[3], "workspace", id, nil)
			if visibility == "selected" && author == owner.ID {
				selected = id
			}
		}
	}
	_ = doc(wid, owner.ID, "workspace", nil, space)
	_ = doc(wid, owner.ID, "workspace", nil, subspace)
	parent := doc(foreign, owner.ID, "private", nil, nil)
	_ = doc(wid, owner.ID, "workspace", parent, nil)
	a := doc(wid, owner.ID, "workspace", nil, nil)
	b := doc(wid, owner.ID, "workspace", a, nil)
	exec(`UPDATE documents SET parent_id=$2 WHERE id=$1`, a, b)
	deep := doc(wid, owner.ID, "workspace", nil, nil)
	for n := 0; n < 21; n++ {
		deep = doc(wid, owner.ID, "workspace", deep, nil)
	}
	deleted := doc(wid, owner.ID, "workspace", nil, nil)
	exec(`UPDATE documents SET deleted_at=clock_timestamp() WHERE id=$1`, deleted)
	combinations := 0
	check := func(stage string) {
		t.Helper()
		tx, e := s.DB.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
		if e != nil {
			t.Fatal(e)
		}
		defer tx.Rollback(ctx)
		for _, actor := range actors {
			for _, scope := range []struct {
				space any
				ids   []string
			}{{nil, []string{}}, {space, []string{}}, {subspace, []string{}}, {nil, []string{selected, selected, deleted, parent}}} {
				for _, shape := range [][2]bool{{false, false}, {true, false}, {true, true}} {
					read := func(query string) []string {
						t.Helper()
						rows, e := tx.Query(ctx, query, actor, wid, scope.space, scope.ids)
						if e != nil {
							t.Fatal(e)
						}
						defer rows.Close()
						out := []string{}
						for rows.Next() {
							var raw []byte
							if e = rows.Scan(&raw); e != nil {
								t.Fatal(e)
							}
							out = append(out, string(raw))
						}
						if e = rows.Err(); e != nil {
							t.Fatal(e)
						}
						return out
					}
					want := read(documentQueryRecursiveCandidateSQL(shape[0], shape[1]))
					got := read(documentQueryCandidateSQL(shape[0], shape[1]))
					if !reflect.DeepEqual(got, want) {
						t.Fatalf("%s actor=%s space=%v ids=%v shape=%v: full JSON/order differs (%d/%d)", stage, actor, scope.space, scope.ids, shape, len(got), len(want))
					}
					combinations++
				}
			}
		}
	}
	check("initial: roles, selected shares, private owners, inherited ACL, cycles, depth and deletion")
	exec(`DELETE FROM document_shares WHERE user_id=$1`, actors[1])
	exec(`DELETE FROM space_members WHERE user_id=$1`, actors[2])
	check("explicit read share and inherited space access revoked")
	exec(`DELETE FROM workspace_members WHERE workspace_id=$1 AND user_id=$2`, wid, actors[2])
	exec(`UPDATE users SET disabled=true WHERE id=$1`, actors[3])
	check("workspace access revoked and actor disabled")
	t.Logf("%d exact recursive-policy/projection/order comparisons", combinations)
}

func TestPostgresDocumentQueryCandidatePlanning(t *testing.T) {
	if os.Getenv("MADI_DOCUMENT_QUERY_PLAN_REPORT") == "" {
		t.Skip("opt-in isolated cold/statistics query-plan diagnostic")
	}
	s, _, ctx, p, wid := jobTestFixture(t)
	if _, e := s.DB.Exec(ctx, `INSERT INTO documents(id,workspace_id,title,markdown,owner_id,status,visibility) SELECT gen_random_uuid(),$1,'범위 후보 '||i,'본문',$2,'draft','workspace' FROM generate_series(1,2100) i`, wid, p.ID); e != nil {
		t.Fatal(e)
	}
	reports := map[string]json.RawMessage{}
	for _, stage := range []string{"cold", "analyzed"} {
		if stage == "analyzed" {
			if _, e := s.DB.Exec(ctx, "ANALYZE documents"); e != nil {
				t.Fatal(e)
			}
		}
		for _, variant := range []string{"recursive", "current"} {
			query := documentQueryRecursiveCandidateSQL(false, false)
			if variant == "current" {
				query = documentQueryCandidateSQL(false, false)
			}
			started := time.Now()
			var raw []byte
			if e := s.DB.QueryRow(ctx, "EXPLAIN (ANALYZE,BUFFERS,FORMAT JSON) "+query, p.ID, wid, nil, []string{}).Scan(&raw); e != nil {
				t.Fatal(e)
			}
			reports[stage+"_"+variant] = raw
			var result []map[string]any
			if e := json.Unmarshal(raw, &result); e != nil {
				t.Fatal(e)
			}
			plan := result[0]["Plan"].(map[string]any)
			t.Logf("%s/%s wall=%s execution_ms=%v planning_ms=%v total_cost=%v shared_hits=%v jit=%v", stage, variant, time.Since(started), result[0]["Execution Time"], result[0]["Planning Time"], plan["Total Cost"], plan["Shared Hit Blocks"], result[0]["JIT"])
		}
	}
	// Compare real prepared statements on one read-only connection. These
	// settings affect nested SQL-function planning too; do not turn this
	// diagnostic into a production force_custom_plan/force_generic_plan knob.
	tx, e := s.DB.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback(ctx)
	for _, variant := range []string{"recursive", "current"} {
		query := documentQueryRecursiveCandidateSQL(false, false)
		if variant == "current" {
			query = documentQueryCandidateSQL(false, false)
		}
		name := "document_query_plan_" + variant
		if _, e = tx.Exec(ctx, "PREPARE "+name+"(uuid,uuid,uuid,uuid[]) AS "+query); e != nil {
			t.Fatal(e)
		}
		for _, mode := range []string{"force_custom_plan", "force_generic_plan"} {
			if _, e = tx.Exec(ctx, "SET LOCAL plan_cache_mode="+mode); e != nil {
				t.Fatal(e)
			}
			var raw []byte
			query = fmt.Sprintf("EXPLAIN (ANALYZE,BUFFERS,FORMAT JSON) EXECUTE %s('%s'::uuid,'%s'::uuid,NULL,'{}'::uuid[])", name, p.ID, wid)
			if e = tx.QueryRow(ctx, query).Scan(&raw); e != nil {
				t.Fatal(e)
			}
			reports[mode+"_"+variant] = raw
			var result []map[string]any
			if e = json.Unmarshal(raw, &result); e != nil {
				t.Fatal(e)
			}
			plan := result[0]["Plan"].(map[string]any)
			t.Logf("%s/%s execution_ms=%v planning_ms=%v shared_hits=%v jit=%v", mode, variant, result[0]["Execution Time"], result[0]["Planning Time"], plan["Shared Hit Blocks"], result[0]["JIT"])
		}
		if _, e = tx.Exec(ctx, "DEALLOCATE "+name); e != nil {
			t.Fatal(e)
		}
	}
	if e = tx.Rollback(ctx); e != nil {
		t.Fatal(e)
	}
	raw, e := json.MarshalIndent(reports, "", "  ")
	if e != nil {
		t.Fatal(e)
	}
	path := os.Getenv("MADI_DOCUMENT_QUERY_PLAN_REPORT")
	if e = os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(path, raw, 0600); e != nil {
		t.Fatal(e)
	}
}
