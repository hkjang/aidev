package server

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// A disposable, opt-in projection/query-plan measurement. No service/global
// statistics or planner switches are altered. These small synthetic bodies
// are not a ten-million-document or large-Markdown capacity claim.
func TestSearchKoreanScalePlan(t *testing.T) {
	if os.Getenv("MADI_SEARCH_KOREAN_SCALE") != "1" {
		t.Skip("set MADI_SEARCH_KOREAN_SCALE=1 for isolated 10k actual GIN plan")
	}
	s, admin, _, wid, actor := collaborationTestSetup(t)
	owner := testJSONObject(t, admin.request("GET", "/api/v1/profile", nil, 200))
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
	defer cancel()
	admin.request("PUT", "/api/v1/workspaces/"+wid+"/search-dictionary", map[string]any{"revision": 0, "confirm_shared": true, "entries": []searchDictionaryEntry{{Canonical: "쿠버네티스", Aliases: []string{"k8s", "Kubernetes"}}}}, 200)
	if _, e := s.DB.Exec(ctx, `INSERT INTO documents(id,workspace_id,title,markdown,owner_id,visibility) SELECT gen_random_uuid(),$1,CASE WHEN n%100=1 THEN '쿠버네티스 장애 대응 ' ELSE '일반 지식 운영 ' END||n::text,repeat('조직 운영을 위한 합성 회귀 문서입니다. ',10),$2,CASE WHEN n%5=0 THEN 'private' ELSE 'workspace' END FROM generate_series(1,10000) n`, wid, str(owner, "id")); e != nil {
		t.Fatal(e)
	}
	rows, e := s.DB.Query(ctx, `SELECT id::text,version,title,markdown FROM documents WHERE workspace_id=$1`, wid)
	if e != nil {
		t.Fatal(e)
	}
	values := [][]any{}
	for rows.Next() {
		var id, title, markdown string
		var version int
		if e = rows.Scan(&id, &version, &title, &markdown); e != nil {
			t.Fatal(e)
		}
		normalized := searchFold(title + " " + markdown)
		grams, complete := searchGrams(normalized)
		values = append(values, []any{id, version, normalized, strings.Join(grams, " "), complete})
	}
	rows.Close()
	if rows.Err() != nil {
		t.Fatal(rows.Err())
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback(ctx)
	if _, e = tx.Exec(ctx, `CREATE TEMP TABLE search_scale_stage(id uuid,version int,normalized text,grams text,complete bool) ON COMMIT DROP`); e != nil {
		t.Fatal(e)
	}
	if _, e = tx.CopyFrom(ctx, pgx.Identifier{"search_scale_stage"}, []string{"id", "version", "normalized", "grams", "complete"}, pgx.CopyFromRows(values)); e != nil {
		t.Fatal(e)
	}
	if _, e = tx.Exec(ctx, `INSERT INTO search_folded_documents(document_id,document_version,normalized,gram_vector,grams_complete) SELECT id,version,normalized,to_tsvector('simple',grams),complete FROM search_scale_stage ON CONFLICT(document_id) DO UPDATE SET document_version=EXCLUDED.document_version,normalized=EXCLUDED.normalized,gram_vector=EXCLUDED.gram_vector,grams_complete=EXCLUDED.grams_complete`); e != nil {
		t.Fatal(e)
	}
	if e = tx.Commit(ctx); e != nil {
		t.Fatal(e)
	}
	if _, e = s.DB.Exec(ctx, `ANALYZE documents;ANALYZE search_folded_documents;ANALYZE workspace_members`); e != nil {
		t.Fatal(e)
	}
	reports := []map[string]any{}
	for _, q := range []string{"k8s장애", "쿠 버 네 티 스 장애"} {
		request := httptest.NewRequest("GET", "/", nil).WithContext(context.WithValue(ctx, principalKey, &Principal{ID: actor, Role: "editor", Kind: "user"}))
		result, observation, status := s.observedSearch(request, wid, map[string]string{"q": q, "type": "document", "limit": "30", "group": "document"})
		if status != 200 || len(result["results"].([]any)) != 30 {
			t.Fatal(status, result)
		}
		for _, raw := range result["results"].([]any) {
			value := raw.(map[string]any)
			if !strings.Contains(str(value, "title"), "쿠버네티스 장애") {
				t.Fatal("normalization returned unrelated title", value)
			}
		}
		planTx, e := s.DB.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
		if e != nil {
			t.Fatal(e)
		}
		var plan []byte
		e = planTx.QueryRow(ctx, "EXPLAIN(ANALYZE,BUFFERS,FORMAT JSON) "+observation.SQL, observation.Arguments...).Scan(&plan)
		_ = planTx.Rollback(ctx)
		if e != nil {
			t.Fatal(e)
		}
		var decoded any
		if e = json.Unmarshal(plan, &decoded); e != nil {
			t.Fatal(e)
		}
		reports = append(reports, map[string]any{"query": q, "elapsed_ms": float64(observation.Elapsed.Microseconds()) / 1000, "page_results": len(result["results"].([]any)), "plan": decoded, "interpretation": result["interpretation"]})
	}
	report := map[string]any{"created_at": time.Now().UTC(), "go": runtime.Version(), "documents": 10000, "private_percent": 20, "concurrency": 1, "scope": "synthetic small-body actual PostgreSQL GIN path; isolated-schema ANALYZE only; no shared-service changes", "results": reports}
	raw, _ := json.MarshalIndent(report, "", "  ")
	output := filepath.Join("..", "..", "test-results", "search-operations", "korean-10000-plan.json")
	if e = os.MkdirAll(filepath.Dir(output), 0700); e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(output, raw, 0600); e != nil {
		t.Fatal(e)
	}
	for _, result := range reports {
		t.Logf("%s: %.2fms, %d results", result["query"], result["elapsed_ms"], result["page_results"])
	}
}
