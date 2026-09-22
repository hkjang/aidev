package server

import (
	"strings"
	"testing"
)

func TestMigrationFormatParsingAndHTMLSafety(t *testing.T) {
	title, md, e := migrationHTML([]byte(`<html><head><title>운영 문서</title><style>secret-css</style></head><body><h1>제목</h1><p>Hello <strong>팀</strong></p><script>danger()</script><a href="javascript:alert(1)">label</a><a href="https://example.test/docs">안전 링크</a><iframe src="https://outside.invalid">invisible</iframe></body></html>`))
	if e != nil || title != "운영 문서" || !strings.Contains(md, "**팀**") || strings.Contains(md, "danger") || strings.Contains(md, "secret-css") || strings.Contains(md, "javascript:") || strings.Contains(md, "invisible") {
		t.Fatalf("HTML conversion failed title=%q md=%q error=%v", title, md, e)
	}
	csv, e := parseMigrationCSV([]byte("이름,상태\n\"줄\n바꿈\",완료\n"))
	if e != nil || len(csv.Rows) != 1 || csv.Rows[0][0] != "줄\n바꿈" {
		t.Fatalf("CSV parsing: %#v %v", csv, e)
	}
	for _, raw := range []string{"같은,같은\na,b", "A,B\n1,2,3", "A,\n1,2"} {
		if _, e := parseMigrationCSV([]byte(raw)); e == nil {
			t.Fatal("invalid CSV accepted")
		}
	}
	input, _, e := prepareMigrationInput("docs.json", "json", []byte(`[{"title":"JSON 문서","markdown":"# 본문","tags":["태그"],"aliases":["별칭"]}]`))
	if e != nil {
		t.Fatal(e)
	}
	docs, _, e := prepareVaultDocuments(input)
	if e != nil || len(docs) != 1 || docs[0].Title != "JSON 문서" || len(docs[0].Tags) != 1 || len(docs[0].Aliases) != 1 || docs[0].Visibility != "private" {
		t.Fatalf("JSON fields not preserved: %#v %v", docs, e)
	}
	if _, _, e = prepareMigrationInput("docs.json", "json", []byte(`[{"title":"bad","markdown":"x","owner_id":"someone"}]`)); e == nil {
		t.Fatal("JSON ownership injection accepted")
	}
	input, _, e = prepareMigrationInput("notion.zip", "notion", makeTestVault(t, map[string][]byte{"Folder/Page.html": []byte("<title>Notion HTML</title><h1>가져온 문서</h1>")}))
	if e != nil {
		t.Fatal(e)
	}
	docs, _, e = prepareVaultDocuments(input)
	if e != nil || len(docs) != 2 {
		t.Fatalf("Notion folder import failed: %d %v", len(docs), e)
	}
}
func TestPostgresMigrationPreviewImportIdempotenceAndCSV(t *testing.T) {
	s, client, ctx, _, wid := jobTestFixture(t)
	before := 0
	if e := s.DB.QueryRow(ctx, "SELECT count(*) FROM documents WHERE workspace_id=$1", wid).Scan(&before); e != nil {
		t.Fatal(e)
	}
	stage := testJSONObject(t, storageMultipart(t, client, "/api/v1/migrations?workspace_id="+wid+"&format=json", "docs.json", []byte(`[{"title":"검토 후 가져오기","markdown":"# 개인 노트"}]`), "", 200))
	id := str(stage, "id")
	client.request("POST", "/api/v1/migrations/"+id+"/run", map[string]any{"confirmation": "IMPORT"}, 409)
	drainJobs(t, s)
	preview := testJSONObject(t, client.request("GET", "/api/v1/migrations/"+id, nil, 200))
	if str(preview, "status") != "ready" || !boolean(preview["preview"].(map[string]any), "validated") {
		t.Fatalf("preview not ready: %#v", preview)
	}
	var count int
	_ = s.DB.QueryRow(ctx, "SELECT count(*) FROM documents WHERE workspace_id=$1", wid).Scan(&count)
	if count != before {
		t.Fatal("preview changed documents")
	}
	job := testJSONObject(t, client.request("POST", "/api/v1/migrations/"+id+"/run", map[string]any{"confirmation": "IMPORT"}, 200))
	drainJobs(t, s)
	imported := testJSONObject(t, client.request("GET", "/api/v1/migrations/"+id, nil, 200))
	if str(imported, "status") != "completed" || number(imported, "source_bytes", -1) != 0 {
		t.Fatalf("import failed %#v", imported)
	}
	_, e := s.DB.Exec(ctx, "UPDATE automation_jobs SET status='pending',run_after=now(),attempts=0 WHERE id=$1", str(job, "job_id"))
	if e != nil {
		t.Fatal(e)
	}
	drainJobs(t, s)
	_ = s.DB.QueryRow(ctx, "SELECT count(*) FROM documents WHERE workspace_id=$1", wid).Scan(&count)
	if count != before+1 {
		t.Fatal("import replay duplicated documents")
	}
	csvStage := testJSONObject(t, storageMultipart(t, client, "/api/v1/migrations?workspace_id="+wid+"&format=csv", "업무.csv", []byte("제목,상태\n테스트,진행 중\n회귀,완료\n"), "", 200))
	drainJobs(t, s)
	client.request("POST", "/api/v1/migrations/"+str(csvStage, "id")+"/run", map[string]any{"confirmation": "IMPORT"}, 200)
	drainJobs(t, s)
	csvResult := testJSONObject(t, client.request("GET", "/api/v1/migrations/"+str(csvStage, "id"), nil, 200))
	if str(csvResult, "status") != "completed" {
		t.Fatalf("CSV import failed %#v", csvResult)
	}
	dbID := str(csvResult["report"].(map[string]any), "database_id")
	_ = s.DB.QueryRow(ctx, "SELECT count(*) FROM database_rows WHERE database_id=$1", dbID).Scan(&count)
	if count != 2 {
		t.Fatal("CSV rows not imported")
	}
}
func TestPostgresMigrationCancellationAndExpiredSource(t *testing.T) {
	s, client, ctx, _, wid := jobTestFixture(t)
	stage := testJSONObject(t, storageMultipart(t, client, "/api/v1/migrations?workspace_id="+wid+"&format=html", "cancel.html", []byte("<h1>취소할 데이터</h1>"), "", 200))
	client.request("DELETE", "/api/v1/migrations/"+str(stage, "id"), nil, 200)
	drainJobs(t, s)
	v := testJSONObject(t, client.request("GET", "/api/v1/migrations/"+str(stage, "id"), nil, 200))
	if str(v, "status") != "cancelled" || number(v, "source_bytes", -1) != 0 {
		t.Fatal("cancelled source retained")
	}
	stage = testJSONObject(t, storageMultipart(t, client, "/api/v1/migrations?workspace_id="+wid+"&format=markdown", "expired.md", []byte("# 만료"), "", 200))
	_, e := s.DB.Exec(ctx, "UPDATE migration_imports SET expires_at=now()-interval '1 day' WHERE id=$1", str(stage, "id"))
	if e != nil {
		t.Fatal(e)
	}
	drainJobs(t, s)
	v = testJSONObject(t, client.request("GET", "/api/v1/migrations/"+str(stage, "id"), nil, 200))
	if str(v, "job_status") != "failed" {
		t.Fatalf("expired upload executed: %#v", v)
	}
}
