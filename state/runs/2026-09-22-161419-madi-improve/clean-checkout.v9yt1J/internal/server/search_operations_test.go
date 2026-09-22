package server

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"
)

func TestSearchKoreanInterpretationAndDictionaryBoundaries(t *testing.T) {
	entries, e := normalizeSearchDictionary([]searchDictionaryEntry{{Canonical: "Kubernetes", Aliases: []string{"k8s", "쿠버네티스"}}, {Canonical: "인공지능", Aliases: []string{"AI"}}})
	if e != nil {
		t.Fatal(e)
	}
	for _, q := range []string{"k8s장애", "쿠 버 네 티 스 장애", "Ｋ８Ｓ 장애"} {
		got := interpretSearch(q, 2, entries)
		if len(got.MatchedTerms) != 1 || len(got.Parts) != 2 || got.GramQuery == "" {
			t.Fatal("Korean dictionary normalization", q, got)
		}
	}
	mail := interpretSearch("mail system", 2, entries)
	if len(mail.MatchedTerms) != 0 {
		t.Fatal("embedded ASCII abbreviation changed unrelated word", mail)
	}
	if got := interpretSearch(strings.Repeat("x ", 17), 2, entries); got.Strategy != "websearch_syntax" || len(got.Parts) != 0 {
		t.Fatal("long query silently dropped terms", got)
	}
	if got := interpretSearch(`"exact phrase" OR token`, 2, entries); got.Strategy != "websearch_syntax" {
		t.Fatal("advanced query syntax reinterpreted", got)
	}
	for _, q := range []string{"k8s -legacy", "k8s or database"} {
		if got := interpretSearch(q, 2, entries); got.Strategy != "websearch_syntax" {
			t.Fatal("boolean/negative syntax changed", q, got)
		}
	}
	if _, e = normalizeSearchDictionary([]searchDictionaryEntry{{Canonical: "GPU 서버"}, {Canonical: "gpu서버"}}); e == nil {
		t.Fatal("normalized duplicate alias accepted")
	}
	if searchFold(" ＧＰＵ\t서버 \n") != "gpu서버" {
		t.Fatal("NFKC/space normalization")
	}
}

func TestPostgresSearchKoreanEvaluationCurrentVersionACLAndCAS(t *testing.T) {
	s, admin, viewer, wid, _ := collaborationTestSetup(t)
	var fixture struct {
		Dictionary []searchDictionaryEntry                 `json:"dictionary"`
		Documents  []struct{ Key, Title, Markdown string } `json:"documents"`
		Cases      []struct {
			Query    string
			Expected []string
			Category string
		} `json:"cases"`
	}
	if e := json.Unmarshal(searchKoreanEvaluationJSON, &fixture); e != nil {
		t.Fatal(e)
	}
	path := "/api/v1/workspaces/" + wid + "/search-dictionary"
	viewer.request("GET", path, nil, 403)
	admin.request("PUT", path, map[string]any{"revision": 0, "entries": fixture.Dictionary, "confirm_shared": false}, 400)
	saved := testJSONObject(t, admin.request("PUT", path, map[string]any{"revision": 0, "entries": fixture.Dictionary, "confirm_shared": true}, 200))
	if number(saved, "revision", 0) != 1 {
		t.Fatal("dictionary revision")
	}
	admin.request("PUT", path, map[string]any{"revision": 0, "entries": fixture.Dictionary, "confirm_shared": true}, 409)
	ids := map[string]string{}
	for _, d := range fixture.Documents {
		created := testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": d.Title, "markdown": d.Markdown}, 200))
		id := str(created, "id")
		ids[d.Key] = id
		if _, e := s.indexSearchDocument(t.Context(), id); e != nil {
			t.Fatal(e)
		}
	}
	reciprocal := 0.0
	recall := 0.0
	for _, test := range fixture.Cases {
		value := testJSONObject(t, viewer.request("GET", "/api/v1/search?workspace_id="+wid+"&type=document&q="+url.QueryEscape(test.Query), nil, 200))
		rows := value["results"].([]any)
		found := map[string]int{}
		for i, row := range rows {
			found[str(row.(map[string]any), "id")] = i + 1
		}
		first, matched := 0, 0
		for _, key := range test.Expected {
			rank := found[ids[key]]
			if rank > 0 && rank <= 5 {
				matched++
				if first == 0 || rank < first {
					first = rank
				}
			}
		}
		if matched != len(test.Expected) {
			t.Fatalf("Korean evaluation %s (%s), recall@5=%d/%d result=%s", test.Query, test.Category, matched, len(test.Expected), jsonValue(value))
		}
		recall += float64(matched) / float64(len(test.Expected))
		if first > 0 {
			reciprocal += 1 / float64(first)
		}
	}
	t.Logf("KOREAN_EVALUATION cases=%d recall@5=%.4f MRR@5=%.4f", len(fixture.Cases), recall/float64(len(fixture.Cases)), reciprocal/float64(len(fixture.Cases)))
	// Derived normalized content must not survive an authority/version boundary.
	id := ids["guide"]
	if _, e := s.DB.Exec(t.Context(), `UPDATE documents SET visibility='private' WHERE id=$1`, id); e != nil {
		t.Fatal(e)
	}
	got := viewer.request("GET", "/api/v1/search?workspace_id="+wid+"&q="+url.QueryEscape("운영가이드"), nil, 200)
	if strings.Contains(string(got), id) {
		t.Fatal("normalized projection bypassed private ACL")
	}
	if _, e := s.DB.Exec(t.Context(), `UPDATE documents SET visibility='workspace',markdown='완전히 변경한 내용',title='다른 제목',version=version+1 WHERE id=$1`, id); e != nil {
		t.Fatal(e)
	}
	got = viewer.request("GET", "/api/v1/search?workspace_id="+wid+"&q="+url.QueryEscape("운영가이드"), nil, 200)
	if strings.Contains(string(got), id) {
		t.Fatal("stale normalized projection leaked old content")
	}
	var raw string
	if e := s.DB.QueryRow(t.Context(), `SELECT coalesce(string_agg(details::text,' '),'') FROM audit_logs WHERE action='SEARCH_DICTIONARY_UPDATE'`).Scan(&raw); e != nil {
		t.Fatal(e)
	}
	if strings.Contains(raw, "Kubernetes") || strings.Contains(raw, "쿠버네티스") {
		t.Fatal("dictionary plaintext entered shared audit")
	}
}
