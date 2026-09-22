package server

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"
)

func TestPostgresSearchGroupedCursorDiagnosticsAndEvaluation(t *testing.T) {
	s, admin, viewer, wid, _ := collaborationTestSetup(t)
	ids := []string{}
	for _, title := range []string{"가나다", "가나다", "라마바", "사아자", "차카타"} {
		doc := testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": title, "markdown": "# 공통 검색\n\n공통 검색 문단\n\n- [ ] 공통 검색 할 일\n\n```text\n공통 검색 코드\n```"}, 200))
		id := str(doc, "id")
		ids = append(ids, id)
		if _, e := s.indexSearchDocument(t.Context(), id); e != nil {
			t.Fatal(e)
		}
	}
	for _, sort := range []string{"relevance", "newest", "oldest", "title"} {
		base := "/api/v1/search?workspace_id=" + wid + "&q=" + url.QueryEscape("공통 검색") + "&group=document&limit=2&sort=" + sort
		cursor := ""
		seen := map[string]bool{}
		pages := 0
		for {
			path := base
			if cursor != "" {
				path += "&cursor=" + url.QueryEscape(cursor)
			}
			data := testJSONObject(t, viewer.request("GET", path, nil, 200))
			for _, row := range data["results"].([]any) {
				v := row.(map[string]any)
				id := str(v, "document_id")
				if seen[id] {
					t.Fatal("keyset duplicate", sort, id)
				}
				seen[id] = true
				metadata := v["metadata"].(map[string]any)
				if number(metadata, "matched_count", 0) < 3 {
					t.Fatal("grouped after pagination instead of before", v)
				}
				if matches, ok := metadata["matches"].([]any); !ok || len(matches) > 3 {
					t.Fatal("group snippets not bounded", v)
				}
			}
			pages++
			if !boolean(data, "has_more") {
				break
			}
			cursor = str(data, "next_cursor")
			if cursor == "" || strings.Contains(cursor, "공통") || pages > 5 {
				t.Fatal("invalid encrypted cursor", pages)
			}
		}
		if len(seen) != len(ids) || pages != 3 {
			t.Fatal("keyset omitted grouped documents", sort, len(seen), pages)
		}
		viewer.request("GET", base+"&cursor="+url.QueryEscape(cursor)+"&tag=changed", nil, 409)
		viewer.request("GET", base+"&cursor="+url.QueryEscape(cursor+"!"), nil, 409)
		admin.request("GET", base+"&cursor="+url.QueryEscape(cursor), nil, 409)
	}
	diagnosticPath := "/api/v1/workspaces/" + wid + "/search-diagnostics"
	viewer.request("POST", diagnosticPath, map[string]any{"confirm": true, "run_plan": true}, 403)
	diagnostic := testJSONObject(t, admin.request("POST", diagnosticPath, map[string]any{"confirm": true, "run_plan": true, "filters": map[string]string{"q": "공통 검색", "group": "document"}}, 200))
	if !boolean(diagnostic, "plan_executed") {
		t.Fatal("no real execution plan", diagnostic)
	}
	planRaw := string(jsonValue(diagnostic["plan"]))
	for _, unsafe := range []string{"공통 검색", "Rows Removed", "Actual Rows", "Filter", "Plan Rows", ids[0]} {
		if strings.Contains(planRaw, unsafe) {
			t.Fatal("private query/population leaked in plan", unsafe, planRaw)
		}
	}
	if boolean(diagnostic, "history_recorded") {
		t.Fatal("diagnostic recorded raw query history")
	}
	evalPath := "/api/v1/workspaces/" + wid + "/search-evaluation"
	evaluation := testJSONObject(t, admin.request("POST", evalPath, map[string]any{"confirm": true, "top_k": 5, "cases": []any{map[string]any{"query": "공통 검색", "expected_document_ids": ids}}}, 200))
	if evaluation["recall_at_k"] != float64(1) {
		t.Fatal("evaluation did not use real ranked results", evaluation)
	}
	admin.request("POST", evalPath, map[string]any{"confirm": true, "top_k": 5, "cases": []any{map[string]any{"query": "개인 평가 원문", "expected_document_ids": []string{newID()}}}}, 400)
	var count int
	if e := s.DB.QueryRow(t.Context(), `SELECT count(*) FROM search_history_entries WHERE query='공통 검색'`).Scan(&count); e != nil || count != 0 {
		t.Fatal("evaluation persisted query without consent", e, count)
	}
	var sample map[string]any
	if e := json.Unmarshal(admin.request("GET", evalPath+"/sample", nil, 200), &sample); e != nil || len(sample["cases"].([]any)) != 12 {
		t.Fatal("synthetic evaluation dataset unavailable", e)
	}
}

func TestSearchPlanRedaction(t *testing.T) {
	value := map[string]any{"Node Type": "Index Scan", "Index Name": "search_folded_documents_grams_idx", "Actual Rows": 123, "Rows Removed by Filter": 777, "Filter": "PRIVATE_QUERY", "Plans": []any{map[string]any{"Node Type": "Seq Scan", "Plan Rows": 999, "Filter": "HIDDEN_VALUE"}}}
	encoded := string(jsonValue(safeSearchPlan(value)))
	if strings.Contains(encoded, "PRIVATE") || strings.Contains(encoded, "HIDDEN") || strings.Contains(encoded, "777") || strings.Contains(encoded, "999") {
		t.Fatal("diagnostic population or literal leak", encoded)
	}
}
