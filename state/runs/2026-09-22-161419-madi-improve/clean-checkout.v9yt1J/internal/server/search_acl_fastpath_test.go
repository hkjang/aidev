package server

import (
	"fmt"
	"reflect"
	"sort"
	"testing"
)

// Differential verification against the existing recursive authorization
// function. The search optimization is only a read-model fast path; it must
// never become an independent or weaker permission model.
func TestPostgresSearchFlatACLMatchesRecursivePolicy(t *testing.T) {
	s, c, ctx, owner, wid := jobTestFixture(t)
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := s.DB.Exec(ctx, q, args...); err != nil {
			t.Fatal(err)
		}
	}
	actors := []string{owner.ID}
	for i, role := range []string{"viewer", "commenter", "editor"} {
		u := testJSONObject(t, c.request("POST", "/api/v1/admin/users", map[string]any{"email": fmt.Sprintf("fast-acl-%d@example.test", i), "name": role, "role": "editor", "password": "Fast-ACL-Test-Password-2026!"}, 200))
		id := str(u, "id")
		actors = append(actors, id)
		exec(`INSERT INTO workspace_members(workspace_id,user_id,role) VALUES($1,$2,$3)`, wid, id, role)
	}
	otherWorkspace := newID()
	exec(`INSERT INTO workspaces(id,name,slug) VALUES($1,'other',$2)`, otherWorkspace, otherWorkspace)
	exec(`INSERT INTO workspace_members(workspace_id,user_id,role) VALUES($1,$2,'owner')`, otherWorkspace, owner.ID)
	space := newID()
	exec(`INSERT INTO spaces(id,workspace_id,name,slug,visibility,owner_id) VALUES($1,$2,'restricted','restricted','restricted',$3)`, space, wid, owner.ID)
	exec(`INSERT INTO space_members(space_id,user_id,role) VALUES($1,$2,'viewer')`, space, actors[1])
	doc := func(workspace, author, visibility string, parent, docSpace any) string {
		t.Helper()
		id := newID()
		exec(`INSERT INTO documents(id,workspace_id,owner_id,title,markdown,visibility,parent_id,space_id) VALUES($1,$2,$3,'FAST_ACL','FAST_ACL',$4,$5,$6)`, id, workspace, author, visibility, parent, docSpace)
		return id
	}
	for _, author := range actors {
		for _, visibility := range []string{"workspace", "private", "selected"} {
			id := doc(wid, author, visibility, nil, nil)
			for i, permission := range []string{"read", "write"} {
				exec(`INSERT INTO document_shares(document_id,user_id,permission) VALUES($1,$2,$3)`, id, actors[i+1], permission)
			}
			_ = doc(wid, actors[3], "workspace", id, nil)
		}
	}
	_ = doc(wid, actors[2], "workspace", nil, space)
	_ = doc(wid, actors[2], "private", nil, space)
	foreignParent := doc(otherWorkspace, owner.ID, "workspace", nil, nil)
	_ = doc(wid, owner.ID, "workspace", foreignParent, nil)
	cycleA := doc(wid, owner.ID, "workspace", nil, nil)
	cycleB := doc(wid, owner.ID, "workspace", cycleA, nil)
	exec(`UPDATE documents SET parent_id=$2 WHERE id=$1`, cycleA, cycleB)
	deleted := doc(wid, owner.ID, "workspace", nil, nil)
	exec(`UPDATE documents SET deleted_at=now() WHERE id=$1`, deleted)
	check := func(stage string) {
		t.Helper()
		for _, actor := range actors {
			want := []string{}
			rows, err := s.DB.Query(ctx, `SELECT id::text FROM documents WHERE workspace_id=$1 AND deleted_at IS NULL AND madi_document_allowed($2,id,false) ORDER BY id`, wid, actor)
			if err != nil {
				t.Fatal(err)
			}
			for rows.Next() {
				var id string
				if err := rows.Scan(&id); err != nil {
					t.Fatal(err)
				}
				want = append(want, id)
			}
			rows.Close()
			if err := rows.Err(); err != nil {
				t.Fatal(err)
			}
			found, err := s.rows(ctx, universalSearchSQL, actor, wid, "", "%", "document", "", "", "", "", nil, nil, false, true, false, false, true)
			if err != nil {
				t.Fatal(err)
			}
			got := []string{}
			for _, hit := range found {
				got = append(got, str(hit, "id"))
			}
			sort.Strings(got)
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("%s actor=%s optimized=%v recursive=%v", stage, actor, got, want)
			}
		}
	}
	check("initial roles, owner, shares, ancestors, restricted space, cycle, deletion")
	exec(`DELETE FROM document_shares WHERE user_id=$1`, actors[1])
	exec(`DELETE FROM space_members WHERE user_id=$1`, actors[1])
	check("share and space membership revoked")
	exec(`DELETE FROM workspace_members WHERE workspace_id=$1 AND user_id=$2`, wid, actors[2])
	check("workspace membership revoked")
	exec(`UPDATE users SET disabled=true WHERE id=$1`, actors[3])
	check("actor disabled")
}
