package server

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestResumeCSVInferenceAndExactText(t *testing.T) {
	csv, e := parseResumeCSV([]byte("이름,수량,전화,완료,날짜\n서버,12,01012345678,true,2026-09-08\n노드,8,01011112222,false,2026-09-09\n"))
	if e != nil {
		t.Fatal(e)
	}
	cols := inferMigrationCSV(csv)
	for i, want := range []string{"text", "number", "text", "checkbox", "date"} {
		if cols[i].Type != want {
			t.Fatalf("column %d type %q expected %q", i, cols[i].Type, want)
		}
	}
	if e = validateMigrationCSVColumns(csv, cols); e != nil {
		t.Fatal(e)
	}
	for _, raw := range []string{"0012", "9007199254740993", "NaN", "Infinity"} {
		if _, ok := migrationNumber(raw); ok {
			t.Fatalf("lossy number inferred: %s", raw)
		}
	}
	items := []migrationSessionItem{{SourceID: "one", FilePath: "a.md", TargetID: newID(), Kind: "document"}, {SourceID: "two", FilePath: "b.md", TargetID: newID(), Kind: "document"}}
	index := newMigrationLinkIndex(items)
	md := "---\ntitle: 원문\n---\n\n  정확한 공백\n\n`[[b]]`\n"
	got, _, _ := migrationContentLinks(items[0], md, index)
	if got != md {
		t.Fatal("Markdown source was normalized")
	}
	got, _, report := migrationContentLinks(items[0], "[[b]]", index)
	if !strings.Contains(got, items[1].TargetID) || number(report, "resolved", 0) != 1 || number(report, "unresolved", 0) != 0 {
		t.Fatalf("wiki link report: %s %#v", got, report)
	}
}

func resumeTestUpload(t *testing.T, client *integrationTestClient, wid, key string, files map[string]string) string {
	t.Helper()
	run := testJSONObject(t, client.request("POST", "/api/v1/migrations/sessions", map[string]any{"workspace_id": wid, "source_key": key, "label": "재개 이관 검증", "format": "markdown"}, 200))
	id := str(run, "id")
	for filename, raw := range files {
		kind := "document"
		if strings.HasSuffix(filename, ".csv") {
			kind = "csv"
		}
		if strings.HasSuffix(filename, ".txt") || strings.HasSuffix(filename, ".png") {
			kind = "attachment"
		}
		manifest := testJSONObject(t, client.request("POST", "/api/v1/migrations/sessions/"+id+"/items", map[string]any{"items": []map[string]any{{"source_id": filename, "path": filename, "kind": kind, "bytes": len(raw), "sha256": digest(raw)}}}, 200))
		item := manifest["items"].([]any)[0].(map[string]any)
		for offset := 0; offset < len(raw); offset += migrationChunkBytes {
			end := min(offset+migrationChunkBytes, len(raw))
			url := client.base + "/api/v1/migrations/sessions/" + id + "/items/" + str(item, "id") + "/chunks/" + fmt.Sprint(offset/migrationChunkBytes)
			req, e := http.NewRequest("PUT", url, strings.NewReader(raw[offset:end]))
			if e != nil {
				t.Fatal(e)
			}
			req.Header.Set("Content-Type", "application/octet-stream")
			req.Header.Set("Origin", client.base)
			req.Header.Set("X-Madi-Request", "1")
			if client.token != "" {
				req.Header.Set("Authorization", "Bearer "+client.token)
			}
			res, e := client.client.Do(req)
			if e != nil {
				t.Fatal(e)
			}
			body, _ := io.ReadAll(res.Body)
			res.Body.Close()
			if res.StatusCode != 200 {
				t.Fatalf("chunk HTTP %d %s", res.StatusCode, body)
			}
		}
	}
	return id
}

func TestPostgresResumableMigrationCSVAndAttachmentIsolation(t *testing.T) {
	s, client, ctx, _, wid := jobTestFixture(t)
	id := resumeTestUpload(t, client, wid, "attachment-source", map[string]string{"첫째.md": "![파일](files/data.txt)\n[[둘째]]", "둘째.md": "[[첫째]]\n[참조](files/data.txt)", "files/data.txt": "검증 첨부 원본", "업무.csv": "이름,수량,완료\n서버,12,true\n노드,8,false\n"})
	v := testJSONObject(t, client.request("GET", "/api/v1/migrations/sessions/"+id, nil, 200))
	client.request("POST", "/api/v1/migrations/sessions/"+id+"/prepare", map[string]any{"revision": v["revision"]}, 200)
	drainJobs(t, s)
	v = testJSONObject(t, client.request("GET", "/api/v1/migrations/sessions/"+id, nil, 200))
	if str(v, "status") != "ready" {
		t.Fatalf("prepare %#v", v)
	}
	client.request("POST", "/api/v1/migrations/sessions/"+id+"/commit", map[string]any{"revision": v["revision"], "plan_hash": v["plan_hash"], "confirmation": "IMPORT"}, 409)
	var csvID string
	if e := s.DB.QueryRow(ctx, "SELECT id::text FROM migration_session_items WHERE session_id=$1 AND kind='csv'", id).Scan(&csvID); e != nil {
		t.Fatal(e)
	}
	detail := testJSONObject(t, client.request("GET", "/api/v1/migrations/sessions/"+id+"/items/"+csvID, nil, 200))
	client.request("PUT", "/api/v1/migrations/sessions/"+id+"/items/"+csvID+"/review", map[string]any{"revision": v["revision"], "plan_hash": v["plan_hash"], "decision": "apply", "confirm_types": true, "types": detail["inferred_columns"]}, 200)
	v = testJSONObject(t, client.request("GET", "/api/v1/migrations/sessions/"+id, nil, 200))
	client.request("POST", "/api/v1/migrations/sessions/"+id+"/commit", map[string]any{"revision": v["revision"], "plan_hash": v["plan_hash"], "confirmation": "IMPORT"}, 200)
	drainJobs(t, s)
	v = testJSONObject(t, client.request("GET", "/api/v1/migrations/sessions/"+id, nil, 200))
	if str(v, "status") != "completed" {
		var reason string
		s.DB.QueryRow(ctx, "SELECT last_error FROM automation_jobs WHERE id=$1", str(v, "job_id")).Scan(&reason)
		t.Fatalf("commit %#v error=%s", v, reason)
	}
	var files, keys, rows int
	if e := s.DB.QueryRow(ctx, `SELECT count(*),count(DISTINCT object_key) FROM attachments WHERE document_id IN(SELECT target_id FROM migration_session_items WHERE session_id=$1 AND kind IN('document','folder'))`, id).Scan(&files, &keys); e != nil {
		t.Fatal(e)
	}
	if files != 3 || keys != 3 {
		t.Fatalf("attachment ownership not isolated files=%d keys=%d", files, keys)
	}
	if e := s.DB.QueryRow(ctx, "SELECT count(*) FROM database_rows WHERE database_id=(SELECT target_id FROM migration_session_items WHERE id=$1)", csvID).Scan(&rows); e != nil {
		t.Fatal(e)
	}
	if rows != 2 {
		t.Fatal("CSV rows lost")
	}
}

func TestPostgresResumableMigrationAtomicRepeatAndConflict(t *testing.T) {
	s, client, ctx, _, wid := jobTestFixture(t)
	var before int
	if e := s.DB.QueryRow(ctx, "SELECT count(*) FROM documents WHERE workspace_id=$1", wid).Scan(&before); e != nil {
		t.Fatal(e)
	}
	raw := "---\ntitle: 보존할 원문\ntags: [지식]\n---\n\n첫째 줄  \n둘째 줄\n"
	for iteration := 0; iteration < 2; iteration++ {
		id := resumeTestUpload(t, client, wid, "stable-source", map[string]string{"원문.md": raw})
		v := testJSONObject(t, client.request("GET", "/api/v1/migrations/sessions/"+id, nil, 200))
		client.request("POST", "/api/v1/migrations/sessions/"+id+"/prepare", map[string]any{"revision": v["revision"]}, 200)
		drainJobs(t, s)
		v = testJSONObject(t, client.request("GET", "/api/v1/migrations/sessions/"+id, nil, 200))
		if str(v, "status") != "ready" {
			t.Fatalf("prepare failed %#v", v)
		}
		var count int
		_ = s.DB.QueryRow(ctx, "SELECT count(*) FROM documents WHERE workspace_id=$1", wid).Scan(&count)
		if count != before+min(iteration, 1) {
			t.Fatal("staging published partial documents")
		}
		client.request("POST", "/api/v1/migrations/sessions/"+id+"/commit", map[string]any{"revision": v["revision"], "plan_hash": v["plan_hash"], "confirmation": "IMPORT"}, 200)
		drainJobs(t, s)
		v = testJSONObject(t, client.request("GET", "/api/v1/migrations/sessions/"+id, nil, 200))
		if str(v, "status") != "completed" {
			var jobs any
			s.DB.QueryRow(ctx, "SELECT jsonb_agg(jsonb_build_object('status',status,'error',last_error)) FROM automation_jobs WHERE id=$1", str(v, "job_id")).Scan(&jobs)
			t.Fatalf("commit failed %#v jobs=%v", v, jobs)
		}
		var md, visibility string
		var version int
		if e := s.DB.QueryRow(ctx, "SELECT markdown,visibility,version FROM documents WHERE id=(SELECT target_id FROM migration_source_bindings WHERE workspace_id=$1 AND source_key='stable-source' AND source_id='원문.md')", wid).Scan(&md, &visibility, &version); e != nil {
			t.Fatal(e)
		}
		if md != raw || visibility != "private" || version != 1 {
			t.Fatalf("source/identity changed: %q %s %d", md, visibility, version)
		}
	}
	_, e := s.DB.Exec(ctx, "UPDATE documents SET markdown='사용자가 수정함',version=version+1 WHERE id=(SELECT target_id FROM migration_source_bindings WHERE workspace_id=$1 AND source_key='stable-source' AND source_id='원문.md')", wid)
	if e != nil {
		t.Fatal(e)
	}
	id := resumeTestUpload(t, client, wid, "stable-source", map[string]string{"원문.md": raw + "원격 변경"})
	v := testJSONObject(t, client.request("GET", "/api/v1/migrations/sessions/"+id, nil, 200))
	client.request("POST", "/api/v1/migrations/sessions/"+id+"/prepare", map[string]any{"revision": v["revision"]}, 200)
	drainJobs(t, s)
	v = testJSONObject(t, client.request("GET", "/api/v1/migrations/sessions/"+id, nil, 200))
	if number(v["report"].(map[string]any), "conflicts", 0) != 1 {
		t.Fatalf("local change not detected %#v", v)
	}
	client.request("POST", "/api/v1/migrations/sessions/"+id+"/commit", map[string]any{"revision": v["revision"], "plan_hash": v["plan_hash"], "confirmation": "IMPORT"}, 409)
}

func TestPostgresResumableMigrationRepeatedCSVExplicitSchema(t *testing.T) {
	s, c, ctx, _, wid := jobTestFixture(t)
	var targetID string
	for iteration, desired := range []string{"number", "text"} {
		id := resumeTestUpload(t, c, wid, "csv-schema", map[string]string{"수량.csv": "수량\n12\n8\n"})
		v := resumePrepare(t, s, c, id)
		if str(v, "status") != "ready" {
			t.Fatal(v)
		}
		var itemID, target string
		if e := s.DB.QueryRow(ctx, "SELECT id::text,target_id::text FROM migration_session_items WHERE session_id=$1 AND kind='csv'", id).Scan(&itemID, &target); e != nil {
			t.Fatal(e)
		}
		if iteration == 0 {
			targetID = target
		} else if targetID != target {
			t.Fatal("same source created a duplicate CSV database")
		}
		detail := testJSONObject(t, c.request("GET", "/api/v1/migrations/sessions/"+id+"/items/"+itemID, nil, 200))
		columns := detail["inferred_columns"].([]any)
		columns[0].(map[string]any)["type"] = desired
		c.request("PUT", "/api/v1/migrations/sessions/"+id+"/items/"+itemID+"/review", map[string]any{"revision": v["revision"], "plan_hash": v["plan_hash"], "decision": "apply", "confirm_types": true, "types": columns}, 200)
		v = testJSONObject(t, c.request("GET", "/api/v1/migrations/sessions/"+id, nil, 200))
		if iteration == 1 && number(v["report"].(map[string]any), "changed", 0) != 1 {
			t.Fatalf("explicit schema change was ignored: %#v", v)
		}
		resumeQueueCommit(t, c, id, v)
		drainJobs(t, s)
		v = testJSONObject(t, c.request("GET", "/api/v1/migrations/sessions/"+id, nil, 200))
		if str(v, "status") != "completed" {
			t.Fatal(v)
		}
		var typ string
		var count int
		if e := s.DB.QueryRow(ctx, "SELECT properties->0->>'type' FROM databases WHERE id=$1", targetID).Scan(&typ); e != nil {
			t.Fatal(e)
		}
		s.DB.QueryRow(ctx, "SELECT count(*) FROM database_rows WHERE database_id=$1", targetID).Scan(&count)
		if typ != desired || count != 2 {
			t.Fatalf("explicit schema/rows not applied: type=%s count=%d", typ, count)
		}
	}
}
