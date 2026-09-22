package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type mixedScaleTrace struct {
	mu      sync.Mutex
	queries map[string]scaleCapturedQuery
}

func (trace *mixedScaleTrace) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	key := ""
	switch {
	case strings.Contains(data.SQL, "FROM hits h"):
		key = "search_mixed"
		if len(data.Args) > 4 {
			if kind, ok := data.Args[4].(string); ok && kind == "document" {
				key = "search_document"
			}
		}
	case strings.Contains(data.SQL, docSummaryJSON) && strings.Contains(data.SQL, "ORDER BY"):
		key = "documents"
	case strings.Contains(data.SQL, "WITH sampled AS (SELECT d.id,count(*) OVER()") || strings.HasPrefix(data.SQL, "WITH visible AS MATERIALIZED (") && strings.Contains(data.SQL, "LIMIT 2001"):
		key = "tasks"
	case strings.HasPrefix(data.SQL, discussionSelect) && strings.Contains(data.SQL, "LIMIT"):
		key = "comments"
	case strings.Contains(data.SQL, "FROM attachments a JOIN documents d") && strings.Contains(data.SQL, "LIMIT 200"):
		key = "attachments"
	case strings.Contains(data.SQL, "SELECT to_jsonb(v) FROM database_rows v"):
		key = "database_rows"
	}
	if key != "" {
		trace.mu.Lock()
		trace.queries[key] = scaleCapturedQuery{SQL: data.SQL, Args: append([]any(nil), data.Args...)}
		trace.mu.Unlock()
	}
	return ctx
}
func (*mixedScaleTrace) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}

type mixedScaleEndpoint struct {
	name, path, method string
	body               any
}

func mixedScaleRequest(ctx context.Context, client *integrationTestClient, endpoint mixedScaleEndpoint) (scaleObservation, []byte) {
	started := time.Now()
	result := scaleObservation{}
	var body io.Reader
	if endpoint.body != nil {
		body = bytes.NewReader(jsonValue(endpoint.body))
	}
	req, err := http.NewRequestWithContext(ctx, endpoint.method, client.base+endpoint.path, body)
	if err != nil {
		return scaleObservation{Error: err.Error()}, nil
	}
	req.Header.Set("X-Madi-Request", "1")
	req.Header.Set("Content-Type", "application/json")
	response, err := client.client.Do(req)
	var raw []byte
	if err != nil {
		result.Error = err.Error()
	} else {
		result.Status = response.StatusCode
		raw, err = io.ReadAll(io.LimitReader(response.Body, 20<<20))
		_ = response.Body.Close()
		result.Bytes = int64(len(raw))
		if err != nil {
			result.Error = err.Error()
		}
		if response.StatusCode != 200 {
			result.Error = "non-200 response"
		}
		if len(raw) >= 20<<20 {
			result.Error = "response reached 20MiB measurement cap"
		}
		if bytes.Contains(raw, []byte("MIXED_PRIVATE_SECRET")) || bytes.Contains(raw, []byte("MIXED_ANCESTOR_SECRET")) {
			result.Error = "ACL_LEAK"
		}
	}
	result.MS = float64(time.Since(started).Microseconds()) / 1000
	return result, raw
}
func mixedScaleMeasure(t *testing.T, ctx context.Context, readers []*integrationTestClient, endpoint mixedScaleEndpoint, count, parallel int) scaleResult {
	t.Helper()
	const requests = 16
	for range 2 {
		mixedScaleRequest(ctx, readers[0], endpoint)
	}
	samples := make([]scaleObservation, requests)
	queue := make(chan int)
	var workers sync.WaitGroup
	for n := 0; n < parallel; n++ {
		workers.Go(func() {
			for index := range queue {
				samples[index], _ = mixedScaleRequest(ctx, readers[n], endpoint)
			}
		})
	}
	for i := range samples {
		queue <- i
	}
	close(queue)
	workers.Wait()
	result := scaleResult{Documents: count, Endpoint: endpoint.name, Concurrency: parallel, Requests: requests, Status: map[int]int{}}
	latencies := []float64{}
	for _, sample := range samples {
		if sample.Error == "ACL_LEAK" {
			t.Fatal("current-scope response leaked private marker", endpoint.name)
		}
		latencies = append(latencies, sample.MS)
		result.Status[sample.Status]++
		result.MeanBytes += sample.Bytes
		result.MaxBytes = max(result.MaxBytes, sample.Bytes)
		if sample.Error != "" {
			result.Errors++
		}
	}
	sort.Float64s(latencies)
	result.P50MS = latencies[(requests-1)*50/100]
	result.P95MS = latencies[(requests-1)*95/100]
	result.MaxMS = latencies[requests-1]
	result.MeanBytes /= requests
	result.ErrorRate = float64(result.Errors) / requests
	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)
	result.HeapBytes = memory.HeapAlloc
	result.RSSKB = scaleProcValue("/proc/self/status", "VmRSS")
	return result
}

func mixedScaleWrite(t *testing.T, name string, value any) {
	t.Helper()
	dir := filepath.Join("..", "..", "test-results", "mixed-scale")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, name), raw, 0600); err != nil {
		t.Fatal(err)
	}
}

// Synthetic canonical templates are projected with the SAME pure parser as the
// service, then bulk-copied to disposable fixture tables. This measures reads,
// not the indexing worker's throughput or file/OCR processing.
func mixedScaleTemplates(t *testing.T, ctx context.Context, s *Server) {
	t.Helper()
	_, err := s.DB.Exec(ctx, `CREATE TABLE mixed_scale_templates(kind int PRIMARY KEY,title text,markdown text,source_hash text,normalized text,grams text,complete bool);
CREATE TABLE mixed_scale_fragments(kind int,ordinal int,fragment_kind text,start_byte int,end_byte int,start_line int,content text,metadata jsonb,normalized text,grams text,complete bool);
CREATE TABLE mixed_scale_chunks(kind int,ordinal int,content_hash text,start_byte int,end_byte int,start_line int,end_line int,heading text,content text);
CREATE TABLE mixed_scale_ids(n int PRIMARY KEY,id uuid UNIQUE,kind int)`)
	if err != nil {
		t.Fatal(err)
	}
	for kind := 0; kind < 6; kind++ {
		label := "일반 지식"
		if kind%2 == 1 {
			label = "쿠버네티스 장애"
		}
		secret := ""
		if kind/2 == 1 {
			secret = "MIXED_PRIVATE_SECRET "
		}
		if kind/2 == 2 {
			secret = "MIXED_ANCESTOR_SECRET "
		}
		title := secret + label + " 운영 기록"
		markdown := "---\nowner: 운영팀\npriority: 2\n---\n# " + title + "\n\n" + strings.Repeat("현장 운영 절차와 변경 이력을 함께 점검합니다. ", 8) + "\n\n## 점검 항목\n\n- [ ] " + secret + label + " 사전 확인\n- [x] " + secret + label + " 완료 확인\n\n> 검증용 합성 자료입니다.\n\n```sql\nSELECT 1;\n```\n"
		folded := searchFold(title + "\n" + markdown + "\nmixed\n운영")
		grams, complete := searchGrams(folded)
		if _, err = s.DB.Exec(ctx, "INSERT INTO mixed_scale_templates VALUES($1,$2,$3,$4,$5,$6,$7)", kind, title, markdown, digest(markdown), folded, strings.Join(grams, " "), complete); err != nil {
			t.Fatal(err)
		}
		for i, f := range projectSearchFragments(markdown) {
			fold := searchFold(f.Content)
			grams, complete := searchGrams(fold)
			if _, err = s.DB.Exec(ctx, "INSERT INTO mixed_scale_fragments VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)", kind, i, f.Kind, f.Start, f.End, f.Line, f.Content, f.Metadata, fold, strings.Join(grams, " "), complete); err != nil {
				t.Fatal(err)
			}
		}
		chunks, err := chunkMarkdown(markdown, 6144, 512)
		if err != nil {
			t.Fatal(err)
		}
		for _, c := range chunks {
			if _, err = s.DB.Exec(ctx, "INSERT INTO mixed_scale_chunks VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)", kind, c.Index, c.Hash, c.Start, c.End, c.StartLine, c.EndLine, c.Heading, c.Content); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func mixedScaleSeed(t *testing.T, ctx context.Context, s *Server, wid, owner, db string, from, to int) {
	t.Helper()
	started := time.Now()
	if _, err := s.DB.Exec(ctx, `INSERT INTO mixed_scale_ids SELECT n,gen_random_uuid(),(CASE WHEN n%5=0 THEN 2 WHEN n%100=2 THEN 4 ELSE 0 END)+(CASE WHEN n%100 IN(0,1,2) THEN 1 ELSE 0 END) FROM generate_series($1::int,$2::int)n`, from, to); err != nil {
		t.Fatal(err)
	}
	queries := []string{
		`INSERT INTO documents(id,workspace_id,owner_id,title,markdown,tags,visibility,created_at,updated_at,block_metadata) SELECT x.id,$1,$2,b.title,b.markdown,'["mixed","운영"]',CASE WHEN x.n%5=0 THEN 'private' ELSE 'workspace' END,'2026-09-09'::timestamptz-x.n*interval '1 millisecond','2026-09-09'::timestamptz-x.n*interval '1 millisecond',jsonb_build_object('blocks',jsonb_build_array(jsonb_build_object('id',md5(x.id::text||'heading')::uuid,'type','heading'),jsonb_build_object('id',md5(x.id::text||'paragraph')::uuid,'type','paragraph'),jsonb_build_object('id',md5(x.id::text||'task1')::uuid,'type','taskItem'),jsonb_build_object('id',md5(x.id::text||'task2')::uuid,'type','taskItem'))) FROM mixed_scale_ids x JOIN mixed_scale_templates b ON b.kind=x.kind WHERE x.n BETWEEN $3 AND $4`,
		`UPDATE documents d SET parent_id=(SELECT id FROM mixed_scale_ids WHERE n=5) FROM mixed_scale_ids x WHERE x.id=d.id AND x.n BETWEEN $3 AND $4 AND x.n%100=2 AND d.workspace_id=$1 AND d.owner_id=$2`,
		`INSERT INTO comments(id,document_id,user_id,body) SELECT gen_random_uuid(),x.id,$2,b.title||' 검증 댓글 '||c FROM mixed_scale_ids x JOIN mixed_scale_templates b ON b.kind=x.kind CROSS JOIN generate_series(1,3)c WHERE x.n BETWEEN $3 AND $4 AND EXISTS(SELECT 1 FROM documents d WHERE d.id=x.id AND d.workspace_id=$1)`,
		`INSERT INTO attachments(id,document_id,user_id,name,content_type,size,path) SELECT gen_random_uuid(),x.id,$2,b.title||'.pdf','application/pdf',12345,'fixture-metadata-only-no-file/'||x.id FROM mixed_scale_ids x JOIN mixed_scale_templates b ON b.kind=x.kind WHERE x.n BETWEEN $3 AND $4 AND EXISTS(SELECT 1 FROM documents d WHERE d.id=x.id AND d.workspace_id=$1)`,
		`INSERT INTO search_chunks SELECT x.id,c.ordinal,1,c.content_hash,c.start_byte,c.end_byte,c.start_line,c.end_line,c.heading,c.content FROM mixed_scale_ids x JOIN mixed_scale_chunks c ON c.kind=x.kind WHERE x.n BETWEEN $3 AND $4 AND EXISTS(SELECT 1 FROM documents d WHERE d.id=x.id AND d.workspace_id=$1 AND d.owner_id=$2)`,
		`INSERT INTO search_fragments(document_id,ordinal,document_version,kind,start_byte,end_byte,start_line,content,metadata) SELECT x.id,f.ordinal,1,f.fragment_kind,f.start_byte,f.end_byte,f.start_line,f.content,f.metadata FROM mixed_scale_ids x JOIN mixed_scale_fragments f ON f.kind=x.kind WHERE x.n BETWEEN $3 AND $4 AND EXISTS(SELECT 1 FROM documents d WHERE d.id=x.id AND d.workspace_id=$1 AND d.owner_id=$2)`,
		`INSERT INTO search_folded_documents(document_id,document_version,normalized,gram_vector,grams_complete) SELECT x.id,1,b.normalized,to_tsvector('simple',b.grams),b.complete FROM mixed_scale_ids x JOIN mixed_scale_templates b ON b.kind=x.kind WHERE x.n BETWEEN $3 AND $4 AND EXISTS(SELECT 1 FROM documents d WHERE d.id=x.id AND d.workspace_id=$1 AND d.owner_id=$2)`,
		`INSERT INTO search_folded_fragments(document_id,ordinal,document_version,normalized,gram_vector,grams_complete) SELECT x.id,f.ordinal,1,f.normalized,to_tsvector('simple',f.grams),f.complete FROM mixed_scale_ids x JOIN mixed_scale_fragments f ON f.kind=x.kind WHERE x.n BETWEEN $3 AND $4 AND EXISTS(SELECT 1 FROM documents d WHERE d.id=x.id AND d.workspace_id=$1 AND d.owner_id=$2)`,
		`INSERT INTO search_index_documents(document_id,document_version,source_hash,chunk_count,links,source_tags,links_indexed) SELECT x.id,1,b.source_hash,(SELECT count(*) FROM mixed_scale_chunks c WHERE c.kind=x.kind),'[]','[]',true FROM mixed_scale_ids x JOIN mixed_scale_templates b ON b.kind=x.kind WHERE x.n BETWEEN $3 AND $4 AND EXISTS(SELECT 1 FROM documents d WHERE d.id=x.id AND d.workspace_id=$1 AND d.owner_id=$2)`,
		`DELETE FROM search_index_queue q USING mixed_scale_ids x WHERE q.document_id=x.id AND x.n BETWEEN $3 AND $4 AND EXISTS(SELECT 1 FROM documents d WHERE d.id=x.id AND d.workspace_id=$1 AND d.owner_id=$2)`,
	}
	for i, query := range queries {
		if _, err := s.DB.Exec(ctx, query, wid, owner, from, to); err != nil {
			t.Fatalf("mixed seed query %d: %v", i, err)
		}
	}
	if _, err := s.DB.Exec(ctx, `INSERT INTO database_rows(id,database_id,values,created_at) SELECT gen_random_uuid(),$1,jsonb_build_object('title',CASE WHEN n%100=1 THEN '쿠버네티스 장애' ELSE '일반 운영' END||' 업무 '||n,'status',CASE WHEN n%2=0 THEN '완료' ELSE '진행 중' END,'priority',n%5),'2026-09-09'::timestamptz+n*interval '1 millisecond' FROM generate_series($2::int,$3::int)n`, db, from, to); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"documents", "workspace_members", "comments", "attachments", "database_rows", "databases", "search_chunks", "search_fragments", "search_folded_documents", "search_folded_fragments", "search_index_documents", "audit_logs"} {
		if _, err := s.DB.Exec(ctx, "ANALYZE "+pgx.Identifier{table}.Sanitize()); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("mixed fixtures %d→%d seeded/projected/analyzed in %.2fs", from-1, to, time.Since(started).Seconds())
}

func TestMixedKnowledgeScale(t *testing.T) {
	if os.Getenv("MADI_MIXED_SCALE") != "1" {
		t.Skip("opt-in disposable 10k/100k mixed knowledge read measurement")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Minute)
	defer cancel()
	s, admin, _, _, _ := collaborationTestSetup(t)
	wid := str(testJSONObject(t, admin.request("POST", "/api/v1/workspaces", map[string]any{"name": "혼합 규모 전용 격리 워크스페이스"}, 200)), "id")
	owner := str(testJSONObject(t, admin.request("GET", "/api/v1/profile", nil, 200)), "id")
	trace := &mixedScaleTrace{queries: map[string]scaleCapturedQuery{}}
	cfg := s.DB.Config()
	cfg.MaxConns = 16
	cfg.ConnConfig.Tracer = trace
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	s.DB = pool
	defer pool.Close()
	readers := make([]*integrationTestClient, 8)
	readerIDs := make([]string, 8)
	for i := range readers {
		email := fmt.Sprintf("mixed-reader-%d@example.test", i)
		readerIDs[i] = str(testJSONObject(t, admin.request("POST", "/api/v1/admin/users", map[string]any{"email": email, "name": "혼합 규모 열람자", "role": "viewer", "password": "Mixed-Scale-Password-2026!"}, 200)), "id")
		admin.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": email, "role": "viewer"}, 200)
		readers[i] = newIntegrationTestClient(t, admin.base)
		readers[i].client.Timeout = 30 * time.Second
		readers[i].request("POST", "/api/v1/auth/login", map[string]any{"email": email, "password": "Mixed-Scale-Password-2026!"}, 200)
	}
	db := str(testJSONObject(t, admin.request("POST", "/api/v1/databases", map[string]any{"workspace_id": wid, "name": "혼합 지식 업무 데이터", "properties": []map[string]any{{"id": "title", "name": "이름", "type": "text"}, {"id": "status", "name": "상태", "type": "select", "options": []string{"진행 중", "완료"}}, {"id": "priority", "name": "우선순위", "type": "number"}}}, 200)), "id")
	admin.request("PUT", "/api/v1/workspaces/"+wid+"/search-dictionary", map[string]any{"revision": 0, "confirm_shared": true, "entries": []searchDictionaryEntry{{Canonical: "쿠버네티스", Aliases: []string{"k8s", "Kubernetes"}}}}, 200)
	mixedScaleTemplates(t, ctx, s)
	all := []scaleResult{}
	planOnly := os.Getenv("MADI_MIXED_SCALE_PLAN_ONLY") == "1"
	previous := 0
	counts := []int{10000, 100000}
	if os.Getenv("MADI_MIXED_SCALE_10K_ONLY") == "1" {
		counts = counts[:1]
	}
	for _, count := range counts {
		mixedScaleSeed(t, ctx, s, wid, owner, db, previous+1, count)
		previous = count
		var visible int
		if err = s.DB.QueryRow(ctx, "SELECT count(*) FROM documents d JOIN mixed_scale_ids x ON x.id=d.id WHERE madi_document_allowed($1,d.id,false)", readerIDs[0]).Scan(&visible); err != nil {
			t.Fatal(err)
		}
		if visible != count*79/100 {
			t.Fatalf("current ACL oracle count=%d expected=%d", visible, count*79/100)
		}
		var public, private, hidden string
		for n, target := range map[int]*string{1: &public, 5: &private, 2: &hidden} {
			if err = s.DB.QueryRow(ctx, "SELECT id::text FROM mixed_scale_ids WHERE n=$1", n).Scan(target); err != nil {
				t.Fatal(err)
			}
		}
		for _, reader := range readers {
			reader.request("GET", "/api/v1/documents/"+private, nil, 404)
			reader.request("GET", "/api/v1/documents/"+hidden, nil, 404)
			reader.request("GET", "/api/v1/documents/"+private+"/attachments", nil, 404)
			reader.request("GET", "/api/v1/documents/"+hidden+"/comments", nil, 403)
		}
		endpoints := []mixedScaleEndpoint{{"document-list", "/api/v1/documents?workspace_id=" + wid + "&limit=100", "GET", nil}, {"korean-document-search", "/api/v1/search?workspace_id=" + wid + "&q=k8s장애&type=document&limit=30&group=document", "GET", nil}, {"mixed-kind-search", "/api/v1/search?workspace_id=" + wid + "&q=쿠버네티스&limit=30", "GET", nil}, {"task-board", "/api/v1/tasks/board?workspace_id=" + wid, "GET", nil}, {"document-comments", "/api/v1/documents/" + public + "/comments?limit=200", "GET", nil}, {"attachment-metadata", "/api/v1/documents/" + public + "/attachments", "GET", nil}, {"database-query", "/api/v1/databases/" + db + "/query", "POST", map[string]any{"limit": 100}}}
		observed := []map[string]any{}
		for _, endpoint := range endpoints {
			sample, raw := mixedScaleRequest(ctx, readers[0], endpoint)
			if sample.Error == "ACL_LEAK" {
				t.Fatal("private source leak", endpoint.name)
			}
			var parsed any
			_ = json.Unmarshal(raw, &parsed)
			detail := map[string]any{"endpoint": endpoint.name, "status": sample.Status, "response_bytes": sample.Bytes}
			if value, ok := parsed.(map[string]any); ok {
				for _, key := range []string{"documents_scanned", "total_documents", "total_documents_exact", "total_documents_is_lower_bound", "notice", "truncated", "limits", "total", "limit", "has_more", "pagination", "outcome", "interpretation"} {
					if v, exists := value[key]; exists {
						detail[key] = v
					}
				}
			}
			observed = append(observed, detail)
			if planOnly {
				continue
			}
			for _, parallel := range []int{1, 8} {
				result := mixedScaleMeasure(t, ctx, readers, endpoint, count, parallel)
				all = append(all, result)
				raw, _ := json.Marshal(result)
				t.Log(string(raw))
				mixedScaleWrite(t, "results.json", map[string]any{"go": runtime.Version(), "recorded_at": time.Now().UTC(), "scope": "isolated synthetic mixed metadata/Markdown; pool16; eight distinct viewers; no file bytes/OCR/remote network/worker throughput; no SLA", "results": all})
			}
		}
		trace.mu.Lock()
		queries := make(map[string]scaleCapturedQuery, len(trace.queries))
		for key, value := range trace.queries {
			queries[key] = value
		}
		trace.mu.Unlock()
		plans := map[string]any{}
		for key, query := range queries {
			tx, e := s.DB.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
			if e != nil {
				t.Fatal(e)
			}
			_, e = tx.Exec(ctx, "SET LOCAL statement_timeout='30s'")
			var plan []byte
			if e == nil {
				e = tx.QueryRow(ctx, "EXPLAIN(ANALYZE,BUFFERS,FORMAT JSON) "+query.SQL, query.Args...).Scan(&plan)
			}
			_ = tx.Rollback(ctx)
			if e != nil {
				plans[key] = map[string]any{"error": e.Error()}
			} else {
				plans[key] = map[string]any{"query": query, "plan": json.RawMessage(plan)}
			}
		}
		var countsRaw []byte
		if err = s.DB.QueryRow(ctx, `SELECT jsonb_build_object('documents',(SELECT count(*) FROM mixed_scale_ids),'markdown_bytes',(SELECT sum(octet_length(d.markdown)) FROM documents d JOIN mixed_scale_ids x ON x.id=d.id),'blocks',(SELECT count(*) FROM search_fragments WHERE kind='block'),'tasks',(SELECT count(*) FROM search_fragments WHERE kind='task'),'code',(SELECT count(*) FROM search_fragments WHERE kind='code'),'comments',(SELECT count(*) FROM comments),'attachment_metadata',(SELECT count(*) FROM attachments),'database_rows',(SELECT count(*) FROM database_rows),'folded_documents',(SELECT count(*) FROM search_folded_documents),'folded_fragments',(SELECT count(*) FROM search_folded_fragments))`).Scan(&countsRaw); err != nil {
			t.Fatal(err)
		}
		name := fmt.Sprintf("%d-detail.json", count)
		if planOnly {
			name = fmt.Sprintf("%d-plan-only.json", count)
		}
		mixedScaleWrite(t, name, map[string]any{"plan_only": planOnly, "counts": json.RawMessage(countsRaw), "observed_limits": observed, "plans": plans, "private_percent": 20, "additional_private_ancestor_percent": 1, "current_acl_visible_documents": visible, "database_pool_max": 16})
	}
	admin.request("DELETE", "/api/v1/workspaces/"+wid+"/members/"+readerIDs[1], nil, 200)
	if raw := readers[1].request("GET", "/api/v1/documents?workspace_id="+wid, nil, 200); string(bytes.TrimSpace(raw)) != "[]" {
		t.Fatalf("revoked member legacy document list must be empty: %s", raw)
	}
	readers[1].request("GET", "/api/v1/search?workspace_id="+wid+"&q=쿠버네티스", nil, 403)
	readers[1].request("GET", "/api/v1/tasks/board?workspace_id="+wid, nil, 403)
	readers[1].request("POST", "/api/v1/databases/"+db+"/query", map[string]any{"limit": 100}, 403)
	mixedScaleWrite(t, "current-acl.json", map[string]any{"eight_real_viewers": true, "private_and_private_ancestor_reads_denied": true, "revoked_member_legacy_document_list": "200 empty array", "revoked_member_new_search_tasks_database": "403"})
	// Collect every endpoint/plan and the revocation checks before failing. A
	// completed measurement must not be mistaken for an error-free capacity test.
	for _, result := range all {
		if result.Errors > 0 {
			t.Errorf("mixed scale limit observed: documents=%d endpoint=%s readers=%d errors=%d/%d statuses=%v (reports preserved)", result.Documents, result.Endpoint, result.Concurrency, result.Errors, result.Requests, result.Status)
		}
	}
}
