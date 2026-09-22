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
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type scaleCapturedQuery struct {
	SQL  string `json:"sql"`
	Args []any  `json:"arguments"`
}
type scaleQueryTrace struct {
	mu      sync.Mutex
	queries map[string]scaleCapturedQuery
}

func (s *scaleQueryTrace) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	label := ""
	if strings.HasPrefix(data.SQL, "WITH active_workspace AS MATERIALIZED (") && strings.Contains(data.SQL, "FROM hits h") {
		label = "search"
	} else if strings.Contains(data.SQL, docSummaryJSON) && strings.Contains(data.SQL, "ORDER BY") && strings.Contains(data.SQL, "LIMIT") {
		label = "list"
	}
	if label != "" {
		s.mu.Lock()
		s.queries[label] = scaleCapturedQuery{data.SQL, append([]any(nil), data.Args...)}
		s.mu.Unlock()
	}
	return ctx
}
func (*scaleQueryTrace) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}
func (trace *scaleQueryTrace) explain(t *testing.T, s *Server, count int) {
	t.Helper()
	trace.mu.Lock()
	queries := make(map[string]scaleCapturedQuery, len(trace.queries))
	for k, v := range trace.queries {
		queries[k] = v
	}
	trace.mu.Unlock()
	if len(queries) != 2 {
		t.Fatalf("representative list/search SQL capture missing: %d", len(queries))
	}
	for label, query := range queries {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		tx, e := s.DB.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
		if e != nil {
			cancel()
			t.Fatal(e)
		}
		var plan []byte
		e = tx.QueryRow(ctx, "EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON) "+query.SQL, query.Args...).Scan(&plan)
		_ = tx.Rollback(ctx)
		cancel()
		if e != nil {
			t.Fatal(e)
		}
		dir := filepath.Join("..", "..", "test-results", "scale-plans")
		if e = os.MkdirAll(dir, 0700); e != nil {
			t.Fatal(e)
		}
		out, e := json.MarshalIndent(map[string]any{"documents": count, "query": query, "explain": json.RawMessage(plan)}, "", "  ")
		if e != nil {
			t.Fatal(e)
		}
		path := filepath.Join(dir, fmt.Sprintf("%d-%s.json", count, label))
		if e = os.WriteFile(path, append(out, '\n'), 0600); e != nil {
			t.Fatal(e)
		}
		t.Logf("Read-only query plan: %s", path)
	}
}

type scaleObservation struct {
	MS     float64
	Bytes  int64
	Status int
	Error  string
}
type scaleResult struct {
	Documents   int         `json:"documents"`
	Endpoint    string      `json:"endpoint"`
	Concurrency int         `json:"concurrency"`
	Requests    int         `json:"requests"`
	P50MS       float64     `json:"p50_ms"`
	P95MS       float64     `json:"p95_ms"`
	MaxMS       float64     `json:"max_ms"`
	Errors      int         `json:"errors"`
	ErrorRate   float64     `json:"error_rate"`
	MeanBytes   int64       `json:"mean_response_bytes"`
	MaxBytes    int64       `json:"max_response_bytes"`
	Status      map[int]int `json:"http_statuses"`
	HeapBytes   uint64      `json:"go_heap_bytes_after"`
	RSSKB       int64       `json:"process_rss_kb_after"`
}

func scaleProcValue(path, key string) int64 {
	raw, e := os.ReadFile(path)
	if e != nil {
		return 0
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.HasPrefix(line, key+":") {
			fields := strings.Fields(line)
			if len(fields) > 1 {
				v, _ := strconv.ParseInt(fields[1], 10, 64)
				return v
			}
		}
	}
	return 0
}
func scaleHTTP(client *http.Client, base, endpoint string) scaleObservation {
	started := time.Now()
	r, e := client.Get(base + endpoint)
	result := scaleObservation{}
	if e != nil {
		result.Error = e.Error()
	} else {
		result.Status = r.StatusCode
		result.Bytes, e = io.Copy(io.Discard, io.LimitReader(r.Body, 8<<20))
		r.Body.Close()
		if e != nil {
			result.Error = e.Error()
		}
		if r.StatusCode != 200 {
			result.Error = "unexpected status"
		}
	}
	result.MS = float64(time.Since(started).Microseconds()) / 1000
	return result
}
func measureScale(client *http.Client, base, endpoint string, count, concurrency, requests int) scaleResult {
	for i := 0; i < 5; i++ {
		scaleHTTP(client, base, endpoint)
	}
	samples := make([]scaleObservation, requests)
	queue := make(chan int)
	var wg sync.WaitGroup
	for n := 0; n < concurrency; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range queue {
				samples[i] = scaleHTTP(client, base, endpoint)
			}
		}()
	}
	for i := range samples {
		queue <- i
	}
	close(queue)
	wg.Wait()
	result := scaleResult{Documents: count, Endpoint: strings.SplitN(endpoint, "?", 2)[0], Concurrency: concurrency, Requests: requests, Status: map[int]int{}}
	latency := make([]float64, requests)
	for i, s := range samples {
		latency[i] = s.MS
		result.MeanBytes += s.Bytes
		result.Status[s.Status]++
		if s.Bytes > result.MaxBytes {
			result.MaxBytes = s.Bytes
		}
		if s.Error != "" {
			result.Errors++
		}
	}
	sort.Float64s(latency)
	result.P50MS = latency[(requests-1)*50/100]
	result.P95MS = latency[(requests-1)*95/100]
	result.MaxMS = latency[requests-1]
	result.MeanBytes /= int64(requests)
	result.ErrorRate = float64(result.Errors) / float64(requests)
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	result.HeapBytes = mem.HeapAlloc
	result.RSSKB = scaleProcValue("/proc/self/status", "VmRSS")
	return result
}

// Opt-in measurement only. CREATE/DROP uses integrationTestServer's random
// schema; no request is sent to the shared development or production service.
func TestReadOnlyScaleBenchmark(t *testing.T) {
	if os.Getenv("MADI_SCALE_BENCH") != "1" {
		t.Skip("opt-in: MADI_SCALE_BENCH=1 with an isolated test PostgreSQL DSN")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	s, admin, _, owner, _ := jobTestFixture(t)
	var trace *scaleQueryTrace
	wantsPlan := os.Getenv("MADI_SCALE_EXPLAIN") == "1" || os.Getenv("MADI_SCALE_EXPLAIN_ONLY") == "1"
	if wantsPlan || os.Getenv("MADI_SCALE_POOL") != "" {
		config := s.DB.Config()
		if wantsPlan {
			trace = &scaleQueryTrace{queries: map[string]scaleCapturedQuery{}}
			config.ConnConfig.Tracer = trace
		}
		if raw := os.Getenv("MADI_SCALE_POOL"); raw != "" {
			size, e := strconv.Atoi(raw)
			if e != nil || size < 1 || size > 100 {
				t.Fatal("MADI_SCALE_POOL must be 1..100")
			}
			config.MaxConns = int32(size)
		}
		pool, e := pgxpool.NewWithConfig(ctx, config)
		if e != nil {
			t.Fatal(e)
		}
		s.DB = pool
		t.Cleanup(pool.Close)
	}
	wid := str(testJSONObject(t, admin.request("POST", "/api/v1/workspaces", map[string]any{"name": "격리된 규모 측정"}, 200)), "id")
	viewer := testJSONObject(t, admin.request("POST", "/api/v1/admin/users", map[string]any{"email": "scale-viewer@example.test", "name": "규모 측정 열람자", "role": "viewer", "password": "Scale-Viewer-Password-2026!"}, 200))
	admin.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": str(viewer, "email"), "role": "viewer"}, 200)
	reader := newIntegrationTestClient(t, admin.base)
	reader.client.Timeout = 30 * time.Second
	reader.request("POST", "/api/v1/auth/login", map[string]any{"email": "scale-viewer@example.test", "password": "Scale-Viewer-Password-2026!"}, 200)
	body := "# 규모 측정 문서\n\nbenchmark_knowledge 운영 지식의 저장·검색·열람을 확인합니다.\n\n" + strings.Repeat("문서 원문 바이트를 목록에 포함하지 않습니다. ", 100)
	results := []scaleResult{}
	previous := 0
	for _, count := range []int{1000, 10000} {
		if _, e := s.DB.Exec(ctx, `INSERT INTO documents(id,workspace_id,title,markdown,tags,owner_id,visibility)
SELECT gen_random_uuid(),$1,'규모 측정 '||n::text,$2,'["benchmark"]'::jsonb,$3,CASE WHEN n%5=0 THEN 'private' ELSE 'workspace' END FROM generate_series($4::integer,$5::integer) n`, wid, body, owner.ID, previous+1, count); e != nil {
			t.Fatal(e)
		}
		previous = count
		if _, e := s.DB.Exec(ctx, "ANALYZE documents; ANALYZE workspace_members"); e != nil {
			t.Fatal(e)
		}
		var id string
		if e := s.DB.QueryRow(ctx, "SELECT id::text FROM documents WHERE workspace_id=$1 AND visibility='workspace' AND title='규모 측정 1' LIMIT 1", wid).Scan(&id); e != nil {
			t.Fatal(e)
		}
		list := reader.request("GET", "/api/v1/documents?workspace_id="+wid+"&limit=100", nil, 200)
		if bytes.Contains(list, []byte(`"markdown"`)) {
			t.Fatal("metadata list exposed raw Markdown")
		}
		for _, endpoint := range []string{"/api/v1/documents?workspace_id=" + wid + "&limit=100", "/api/v1/search?workspace_id=" + wid + "&q=benchmark_knowledge&type=document&limit=30", "/api/v1/documents/" + id} {
			if os.Getenv("MADI_SCALE_EXPLAIN_ONLY") == "1" {
				reader.request("GET", endpoint, nil, 200)
				continue
			}
			result := measureScale(reader.client, reader.base, endpoint, count, 10, 60)
			if strings.HasPrefix(result.Endpoint, "/api/v1/documents/") {
				result.Endpoint = "/api/v1/documents/{id}"
			}
			results = append(results, result)
			raw, _ := json.Marshal(result)
			t.Log(string(raw))
		}
		if trace != nil {
			trace.explain(t, s, count)
		}
	}
	var pgVersion string
	_ = s.DB.QueryRow(ctx, "SHOW server_version").Scan(&pgVersion)
	report := map[string]any{"created_at": time.Now().UTC().Format(time.RFC3339), "go_version": runtime.Version(), "goos": runtime.GOOS, "goarch": runtime.GOARCH, "logical_cpus": runtime.NumCPU(), "gomaxprocs": runtime.GOMAXPROCS(0), "host_memory_kb": scaleProcValue("/proc/meminfo", "MemTotal"), "postgres_version": pgVersion, "postgres_pool_max_connections": s.DB.Config().MaxConns, "document_body_bytes": len(body), "private_document_percent": 20, "warmup_requests_per_endpoint": 5, "results": results, "limitations": []string{"10 concurrent readers, 60 timed requests per endpoint; not a capacity guarantee", "Go API and PostgreSQL on shared host; unrelated tasks can affect latency", "single-level document ACL, 20% private hidden from viewer", "native document FTS only; no semantic provider or derived block index workload", "seed writes and normal GET audit logs are restricted to a disposable schema", "no browser rendering, TLS proxy, network latency or attachments measured", "Go race detector overhead depends on invocation; run without -race for representative timing"}}
	path := os.Getenv("MADI_SCALE_REPORT")
	if path == "" {
		path = filepath.Join("..", "..", "test-results", "scale-benchmark.json")
		if os.Getenv("MADI_SCALE_EXPLAIN_ONLY") == "1" {
			path = filepath.Join("..", "..", "test-results", "scale-explain-environment.json")
		}
	}
	if e := os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		t.Fatal(e)
	}
	raw, e := json.MarshalIndent(report, "", "  ")
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(path, append(raw, '\n'), 0600); e != nil {
		t.Fatal(e)
	}
	t.Logf("Report: %s", path)
	for _, r := range results {
		if r.Errors > 0 {
			t.Error(fmt.Sprintf("%s (%d documents): %d/%d errors", r.Endpoint, r.Documents, r.Errors, r.Requests))
		}
	}
}
