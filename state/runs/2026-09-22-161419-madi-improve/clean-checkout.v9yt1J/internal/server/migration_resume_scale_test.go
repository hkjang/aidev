package server

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"testing"
	"time"
)

// Opt-in; never creates documents in an existing user workspace. The fixture
// owns a temporary schema and removes it even after a failed publication.
func TestResumableMigrationScale(t *testing.T) {
	raw := os.Getenv("MADI_MIGRATION_SCALE_ITEMS")
	if raw == "" {
		t.Skip("set MADI_MIGRATION_SCALE_ITEMS=1000 (or 10000/100000) for an isolated mixed migration measurement")
	}
	count, e := strconv.Atoi(raw)
	if e != nil || (count != 1000 && count != 10000 && count != 100000) {
		t.Fatal("MADI_MIGRATION_SCALE_ITEMS must be 1000, 10000 or 100000")
	}
	s, c, ctx, _, wid := jobTestFixture(t)
	if _, e = s.DB.Exec(ctx, "UPDATE migration_settings SET max_items=100000"); e != nil {
		t.Fatal(e)
	}
	files := map[string]string{"자산.csv": "이름,수량\n서버,12\n노드,8\n", "메모.txt": "혼합 이관의 원본 첨부"}
	// One inferred private attachment parent brings the exact final item count
	// to count. No remote services, production schemas or external credentials.
	for i := 0; i < count-3; i++ {
		name, body := fmt.Sprintf("문서-%06d.md", i), fmt.Sprintf("# 운영 %06d\n\n독립 이관 검증입니다.\n", i)
		if i%10 == 0 {
			name, body = fmt.Sprintf("자료-%06d.html", i), fmt.Sprintf("<h1>운영 %06d</h1><p>HTML 혼합 검증입니다.</p>", i)
		}
		files[name] = body
	}
	started := time.Now()
	id := resumeTestUpload(t, c, wid, "scale-mixed", files)
	uploaded := time.Now()
	v := resumePrepare(t, s, c, id)
	prepared := time.Now()
	if str(v, "status") != "ready" || int(number(v, "item_count", 0)) != count || int(number(v, "prepared_count", 0)) != count {
		t.Fatalf("scale prepare failed or lost checkpoints: %#v", v)
	}
	var published int
	if e = s.DB.QueryRow(ctx, "SELECT count(*) FROM documents WHERE id IN(SELECT target_id FROM migration_session_items WHERE session_id=$1)", id).Scan(&published); e != nil || published != 0 {
		t.Fatal("scale staging published partial documents", published, e)
	}
	var csvID string
	if e = s.DB.QueryRow(ctx, "SELECT id::text FROM migration_session_items WHERE session_id=$1 AND kind='csv'", id).Scan(&csvID); e != nil {
		t.Fatal(e)
	}
	detail := testJSONObject(t, c.request("GET", "/api/v1/migrations/sessions/"+id+"/items/"+csvID, nil, 200))
	c.request("PUT", "/api/v1/migrations/sessions/"+id+"/items/"+csvID+"/review", map[string]any{"revision": v["revision"], "plan_hash": v["plan_hash"], "decision": "apply", "confirm_types": true, "types": detail["inferred_columns"]}, 200)
	v = testJSONObject(t, c.request("GET", "/api/v1/migrations/sessions/"+id, nil, 200))
	confirmed := time.Now()
	resumeQueueCommit(t, c, id, v)
	job, e := s.claimJob(ctx)
	if e != nil || job.Kind != "migration.session.commit" {
		t.Fatalf("expected the queued migration publication: %s %v", job.Kind, e)
	}
	s.runJob(ctx, job)
	finished := time.Now()
	v = testJSONObject(t, c.request("GET", "/api/v1/migrations/sessions/"+id, nil, 200))
	if str(v, "status") != "completed" {
		t.Fatalf("scale atomic publication failed: %#v", v)
	}
	if e = s.DB.QueryRow(ctx, "SELECT count(*) FROM documents WHERE id IN(SELECT target_id FROM migration_session_items WHERE session_id=$1) AND visibility='private' AND status='draft'", id).Scan(&published); e != nil || published != count-2 {
		t.Fatal("scale publication lost or exposed documents", published, e)
	}
	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)
	report := map[string]any{"items": count, "documents": published, "database_count": 1, "attachment_count": 1, "upload_seconds": uploaded.Sub(started).Seconds(), "prepare_seconds": prepared.Sub(uploaded).Seconds(), "explicit_review_seconds": confirmed.Sub(prepared).Seconds(), "atomic_commit_seconds": finished.Sub(confirmed).Seconds(), "total_seconds": finished.Sub(started).Seconds(), "heap_inuse_bytes_after_run": memory.HeapInuse, "allocated_bytes_process_total": memory.TotalAlloc, "go": runtime.Version(), "logical_cpus": runtime.NumCPU(), "gomaxprocs": runtime.GOMAXPROCS(0), "note": "Isolated small-content mixed fixture, shared host; downstream event delivery not included; not a maximum-size or production throughput guarantee"}
	encoded, _ := json.MarshalIndent(report, "", "  ")
	t.Log(string(encoded))
	if output := os.Getenv("MADI_MIGRATION_SCALE_REPORT"); output != "" {
		if e = os.WriteFile(output, append(encoded, '\n'), 0600); e != nil {
			t.Fatal(e)
		}
	}
}
