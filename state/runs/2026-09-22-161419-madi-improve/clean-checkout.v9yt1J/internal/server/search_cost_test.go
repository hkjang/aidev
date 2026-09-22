package server

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestSearchCaseIndependentLiteralBoundary(t *testing.T) {
	query := interpretedSearchSQL("쿠버네티스")
	for _, expected := range []string{"FROM matching_fragments match", "FROM matching_comments match", "FROM search_fragments f", "f.content LIKE $4"} {
		if !strings.Contains(query, expected) {
			t.Fatal("candidate optimization no longer matches the original SQL boundary", expected)
		}
	}
	for _, term := range []string{"쿠버네티스", "장애 123", "ㄱㅏ", "한", "100%_\\", ""} {
		if !searchCaseIndependentLiteral(term) {
			t.Fatal("case-independent literal not recognized", term)
		}
	}
	for _, term := range []string{"k8s장애", "Kubernetes", "İ", "ß", "Σ", "\u0307", "東京", "K"} {
		if searchCaseIndependentLiteral(term) {
			t.Fatal("locale-sensitive or unsupported alphabet bypassed ILIKE", term)
		}
	}
}

func TestPostgresSearchCostOptimizationsMatchOriginal(t *testing.T) {
	s, admin, _, wid, viewerID := collaborationTestSetup(t)
	ctx := context.Background()
	owner := str(testJSONObject(t, admin.request("GET", "/api/v1/profile", nil, 200)), "id")
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, e := s.DB.Exec(ctx, sql, args...); e != nil {
			t.Fatal(e)
		}
	}
	texts := []string{"쿠버네티스 장애", "쿠 버 네 티 스 운영", "Kubernetes INCIDENT", "k8s장애", "100%_\\ 운영", "한글 ㄱㅏ 한", "İiı ßẞ ſS Σσς KK \u0307", "일반 지식"}
	ids := []string{}
	for i, text := range texts {
		id := newID()
		ids = append(ids, id)
		visibility := "workspace"
		if i%3 == 2 {
			visibility = "private"
		}
		exec(`INSERT INTO documents(id,workspace_id,owner_id,title,markdown,tags,aliases,visibility) VALUES($1,$2,$3,$4,$5,'["운영","쿠버네티스"]','["별칭문서"]',$6)`, id, wid, owner, text, "# "+text+"\n\n- [ ] "+text+"\n\n```txt\n"+text+"\n```", visibility)
		exec(`INSERT INTO comments(id,document_id,user_id,body) VALUES($1,$2,$3,$4)`, newID(), id, owner, text)
		exec(`INSERT INTO attachments(id,document_id,user_id,name,content_type,size,path) VALUES($1,$2,$3,$4,'application/pdf',1,$5)`, newID(), id, owner, text+".pdf", "search-cost-metadata/"+id)
		if _, e := s.indexSearchDocument(ctx, id); e != nil {
			t.Fatal(e)
		}
	}
	exec(`UPDATE documents SET parent_id=$2 WHERE id=$1`, ids[0], ids[2])
	// A missing folded projection must preserve the original source fallback.
	exec(`DELETE FROM search_folded_documents WHERE document_id=$1`, ids[3])
	exec(`DELETE FROM search_folded_fragments WHERE document_id=$1`, ids[3])
	entries, e := normalizeSearchDictionary([]searchDictionaryEntry{{Canonical: "쿠버네티스", Aliases: []string{"k8s", "Kubernetes"}}})
	if e != nil {
		t.Fatal(e)
	}
	queries := []string{"쿠버네티스", "장애", "k8s장애", "운영", "100%_\\", "ㄱㅏ", "한", "별칭문서", "İ", "σ", "\u0307", `"Kubernetes" OR incident`, ""}
	for _, actor := range []string{owner, viewerID} {
		for _, q := range queries {
			interpretation := interpretSearch(q, 1, entries)
			pattern := "%" + strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_").Replace(q) + "%"
			args := []any{actor, wid, q, pattern, "", "", "", "", "", nil, nil, false, true, true, true, true, 100, 0, jsonValue(interpretation.Parts), jsonValue(interpretation.FoldedParts), interpretation.GramQuery, time.Date(2026, 9, 9, 0, 0, 0, 0, time.UTC)}
			read := func(sql string) []map[string]any {
				t.Helper()
				sql = strings.ReplaceAll(sql, "now()", "$22::timestamptz") + " ORDER BY h.kind,h.id LIMIT $17 OFFSET $18"
				got, e := s.rows(ctx, sql, args...)
				if e != nil {
					t.Fatal(q, e)
				}
				return got
			}
			for _, kind := range []string{"", "document", "block", "task", "comment", "file"} {
				args[4] = kind
				before, after := read(legacyInterpretedSearchSQL()), read(interpretedSearchSQL(q, kind))
				if !reflect.DeepEqual(before, after) {
					t.Fatalf("actor %s query %q kind %q altered matches/ranking/metadata: original=%v optimized=%v", actor, q, kind, before, after)
				}
			}
			if searchCaseIndependentLiteral(q) {
				for _, text := range texts {
					var equal bool
					if e := s.DB.QueryRow(ctx, `SELECT ($1::text LIKE $2) IS NOT DISTINCT FROM ($1::text ILIKE $2)`, text, pattern).Scan(&equal); e != nil || !equal {
						t.Fatal("PostgreSQL case-free literal equivalence", text, q, equal, e)
					}
				}
			}
		}
	}
	secret := "DELETED_COMMENT_VECTOR_MUST_NOT_LEAK"
	cid := newID()
	exec(`INSERT INTO comments(id,document_id,user_id,body) VALUES($1,$2,$3,$4)`, cid, ids[1], owner, secret)
	active := admin.request("GET", "/api/v1/documents/"+ids[1]+"/comments", nil, 200)
	if strings.Contains(string(active), "search_vector") {
		t.Fatal("internal search projection returned in comment JSON")
	}
	exec(`UPDATE comments SET deleted_at=clock_timestamp() WHERE id=$1`, cid)
	deleted := admin.request("GET", "/api/v1/documents/"+ids[1]+"/comments", nil, 200)
	if strings.Contains(string(deleted), secret) || strings.Contains(string(deleted), "search_vector") {
		t.Fatal("deleted comment content leaked through generated projection", string(deleted))
	}
	var matches int
	if e = s.DB.QueryRow(ctx, `SELECT count(*) FROM comments WHERE id=$1 AND deleted_at IS NULL AND to_tsvector('simple',body)@@plainto_tsquery('simple',$2)`, cid, secret).Scan(&matches); e != nil || matches != 0 {
		t.Fatal("deleted comment remained searchable", matches, e)
	}
	t.Log(fmt.Sprintf("%d query/actor/kind differential cases preserve full result JSON and rank", len(queries)*2*6))
}
