package server

import (
	"fmt"
	"strings"
	"testing"
)

func TestPostgresEnterpriseRelatedSourceBudgetAndIndexedRelations(t *testing.T) {
	s, c, wid, uid := agentTestSetup(t)
	entity := testJSONObject(t, c.request("POST", "/api/v1/enterprise/entities", map[string]any{"workspace_id": wid, "title": "bounded entity", "entity_type": "technology"}, 200))
	eid := str(entity, "id")
	// JSON encoding doubles these backslashes. The budget must account for that
	// expansion before transferring any complete document row to Go.
	markdown := strings.Repeat(`\`, 1<<20) + "\n\n[[bounded entity]]"
	for i := range 12 {
		_, e := s.DB.Exec(t.Context(), `INSERT INTO documents(id,workspace_id,title,markdown,owner_id) VALUES($1,$2,$3,$4,$5)`, newID(), wid, fmt.Sprintf("미색인 문서 %02d", i), markdown, uid)
		if e != nil {
			t.Fatal(e)
		}
	}
	manual, indexed := newID(), newID()
	for _, id := range []string{manual, indexed} {
		_, e := s.DB.Exec(t.Context(), `INSERT INTO documents(id,workspace_id,title,markdown,owner_id,updated_at) VALUES($1,$2,$3,$4,$5,now()-interval '1 day')`, id, wid, id, markdown, uid)
		if e != nil {
			t.Fatal(e)
		}
	}
	if _, e := s.DB.Exec(t.Context(), `INSERT INTO document_relations(source_id,target_id,type,created_by) VALUES($1,$2,'related',$3)`, eid, manual, uid); e != nil {
		t.Fatal(e)
	}
	if _, e := s.DB.Exec(t.Context(), `INSERT INTO search_index_documents(document_id,document_version,source_hash,chunk_count,links,links_indexed) VALUES($1,1,$2,0,'["bounded entity"]',true)`, indexed, digest(markdown)); e != nil {
		t.Fatal(e)
	}
	rows, e := s.rows(t.Context(), enterpriseRelatedDocumentsSQL, wid, eid, uid, []string{"bounded entity", eid})
	if e != nil {
		t.Fatal(e)
	}
	bytes, deferred, noSource := 0, 0, map[string]bool{}
	for _, row := range rows {
		bytes += len(jsonValue(str(row, "markdown")))
		if boolean(row, "deferred") {
			deferred++
		}
		if str(row, "id") == manual || str(row, "id") == indexed {
			noSource[str(row, "id")] = str(row, "markdown") == "" && !boolean(row, "deferred")
		}
	}
	if bytes > 16<<20 || deferred == 0 || !noSource[manual] || !noSource[indexed] {
		t.Fatalf("source budget bytes=%d deferred=%d source-free=%v", bytes, deferred, noSource)
	}
	result := testJSONObject(t, c.request("GET", "/api/v1/enterprise/entities/"+eid, nil, 200))
	if !boolean(result, "backlink_truncated") || number(result, "backlink_source_bytes_limit", 0) != 16<<20 {
		t.Fatal("bounded scan diagnostic missing")
	}
	linked := map[string]bool{}
	for _, raw := range result["related_documents"].([]any) {
		row := raw.(map[string]any)
		linked[str(row, "id")] = true
		if _, exists := row["markdown"]; exists {
			t.Fatal("related document response retained source body")
		}
	}
	if !linked[manual] || !linked[indexed] {
		t.Fatal("source budget hid explicit or indexed relation")
	}
}
