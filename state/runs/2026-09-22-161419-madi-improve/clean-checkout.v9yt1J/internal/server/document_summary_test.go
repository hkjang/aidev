package server

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestPostgresDocumentListBoundedSummaryAndFullDetail(t *testing.T) {
	s, c, wid, uid := agentTestSetup(t)
	id := newID()
	markdown := strings.Repeat("x", 4<<20)
	metadata := map[string]any{"private_raw_metadata": strings.Repeat("Z", (1<<20)-64)}
	tags, aliases := []string{}, []string{}
	for i := range 32 {
		tags = append(tags, fmt.Sprintf("%02d-", i)+strings.Repeat("\t", 45)+strings.Repeat("t", 152))
		aliases = append(aliases, fmt.Sprintf("%02d-", i)+strings.Repeat("\t", 45)+strings.Repeat("a", 152))
	}
	tags = append(tags, strings.Repeat("overlong-tag", 30))
	aliases = append(aliases, strings.Repeat("overlong-alias", 30))
	_, e := s.DB.Exec(t.Context(), `INSERT INTO documents(id,workspace_id,title,markdown,owner_id,tags,aliases,block_metadata) VALUES($1,$2,'원문 개별 조회 검증',$3,$4,$5,$6,$7)`, id, wid, markdown, uid, jsonValue(tags), jsonValue(aliases), jsonValue(metadata))
	if e != nil {
		t.Fatal(e)
	}
	_, e = s.DB.Exec(t.Context(), `INSERT INTO documents(id,workspace_id,title,markdown,owner_id,tags,aliases) SELECT gen_random_uuid(),$1,'용량 검증 '||n::text,'본문 미리보기',$2,$3,$4 FROM generate_series(1,1999) n`, wid, uid, jsonValue(tags), jsonValue(aliases))
	if e != nil {
		t.Fatal(e)
	}
	query, args := documentListQuery(uid, wid, "", "", false, false, 100, 0)
	plan, e := s.DB.Query(t.Context(), "EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT) "+query, args...)
	if e != nil {
		t.Fatal(e)
	}
	for plan.Next() {
		var line string
		if e := plan.Scan(&line); e != nil {
			t.Fatal(e)
		}
		t.Log(line)
	}
	plan.Close()
	start := time.Now()
	raw := c.request("GET", "/api/v1/documents?workspace_id="+wid+"&limit=2000", nil, 200)
	t.Logf("2,000 summaries with maximal encoded tag/alias entries: %d bytes in %s", len(raw), time.Since(start))
	if len(raw) > 36<<20 {
		t.Fatalf("unbounded summary response: %d bytes", len(raw))
	}
	if strings.Contains(string(raw), "private_raw_metadata") || strings.Contains(string(raw), `"markdown":`) || strings.Contains(string(raw), `"block_metadata":`) {
		t.Fatal("raw content in document list")
	}
	var rows []map[string]any
	if e = json.Unmarshal(raw, &rows); e != nil {
		t.Fatal(e)
	}
	if len(rows) != 2000 {
		t.Fatalf("pagination compatibility %d", len(rows))
	}
	for _, row := range rows {
		if len([]rune(str(row, "excerpt"))) > 160 || len(listStrings(row["tags"])) != 32 || len(listStrings(row["aliases"])) != 32 || !boolean(row, "tags_truncated") || !boolean(row, "aliases_truncated") {
			t.Fatal("summary bound/truncation missing")
		}
		for i, value := range listStrings(row["aliases"]) {
			if value != aliases[i] {
				t.Fatal("summary invented a clipped alias")
			}
		}
	}
	detail := testJSONObject(t, c.request("GET", "/api/v1/documents/"+id, nil, 200))
	if str(detail, "markdown") != markdown || len(listStrings(detail["aliases"])) != 33 || len(jsonValue(detail["block_metadata"])) < 1<<19 {
		t.Fatal("full detail lost canonical content")
	}
}

func TestDocumentListQueryIndexFriendlyAndParameterized(t *testing.T) {
	query, args := documentListQuery("actor", "workspace", "", "", false, false, 100, 0)
	if strings.Contains(query, "CASE WHEN $2") || strings.Contains(query, "ts_rank_cd") || strings.Contains(query, "workspace_id::text") || !strings.Contains(query, "d.workspace_id=$2::uuid") || !strings.Contains(query, "ORDER BY d.updated_at DESC LIMIT $3 OFFSET $4") || len(args) != 4 {
		t.Fatalf("default indexed list contract %s %#v", query, args)
	}
	query, args = documentListQuery("actor", "", "needle';DROP", "tag';DROP", true, true, 20, 5)
	if strings.Contains(query, "needle") || strings.Contains(query, "tag';DROP") || !strings.Contains(query, "ts_rank_cd") || !strings.Contains(query, "d.deleted_at IS NOT NULL") || len(args) != 5 {
		t.Fatalf("search filter parameterization %s %#v", query, args)
	}
}
