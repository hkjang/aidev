package server

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"
)

func exportTestFiles(t *testing.T, raw []byte) map[string][]byte {
	t.Helper()
	z, e := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if e != nil {
		t.Fatal(e)
	}
	out := map[string][]byte{}
	for _, f := range z.File {
		r, e := f.Open()
		if e != nil {
			t.Fatal(e)
		}
		out[f.Name], e = io.ReadAll(r)
		r.Close()
		if e != nil {
			t.Fatal(e)
		}
	}
	return out
}
func TestMigrationNotionLocalImagesLinksCSVAndJSONVault(t *testing.T) {
	files := map[string][]byte{"Folder/A.html": []byte(`<title>A</title><p><img src="image.png" alt="로컬"><img src="https://outside.invalid/secret"><a href="B.html#hello">B</a></p>`), "Folder/B.html": []byte(`<title>B</title><h1>hello</h1>`), "Folder/image.png": {137, 80, 78, 71}, "Folder/tasks.csv": []byte("제목,상태\n업무,완료\n")}
	input, _, e := prepareMigrationInput("notion.zip", "notion", makeTestVault(t, files))
	if e != nil {
		t.Fatal(e)
	}
	md := string(input.files["Folder/A.md"])
	if !strings.Contains(md, "image.png") || !strings.Contains(md, "B.md#hello") || strings.Contains(md, "outside.invalid") || len(input.databases) != 1 {
		t.Fatalf("local conversion incorrect %q %#v", md, input.databases)
	}
	if _, _, e = prepareMigrationInput("deep.html", "html", []byte(strings.Repeat("<div>", 120)+"x"+strings.Repeat("</div>", 120))); e == nil {
		t.Fatal("deep HTML accepted")
	}
	rawMD := "---\r\ntitle: 원문\r\n---\r\n\r\n# 원문  \r\n\tcode\r\n"
	v := transferJSONVault{Format: "madi-json-vault", Version: 1, Manifest: vaultManifest{Format: "madi-vault", Version: 1, Documents: []*vaultDocument{{ID: "untrusted-old", File: "원문.md", Title: "원문", Visibility: "workspace", Status: "published"}}, Attachments: []*vaultAttachment{}}, Files: []transferJSONFile{{Path: "원문.md", Data: []byte(rawMD)}}}
	raw, _ := json.Marshal(v)
	input, _, e = prepareMigrationInput("data.json", "json", raw)
	if e != nil {
		t.Fatal(e)
	}
	docs, _, e := prepareVaultDocuments(input)
	if e != nil || len(docs) != 1 || docs[0].Markdown != rawMD || docs[0].Visibility != "private" || docs[0].Status != "draft" || docs[0].NewID == "untrusted-old" {
		t.Fatalf("JSON identity/raw contract: %#v %v", docs, e)
	}
	v.Files[0].Path = "../escape.md"
	raw, _ = json.Marshal(v)
	if _, e = parseJSONVault(raw); e == nil {
		t.Fatal("JSON path traversal accepted")
	}
	for _, cell := range []string{"=cmd()", " +SUM(1)", "\t@evil", "-1"} {
		if !strings.HasPrefix(csvSafeCell(cell), "'") {
			t.Fatal("CSV formula not escaped")
		}
	}
}

func TestExportHTMLRetainsTablesButRejectsActiveOrRemoteContent(t *testing.T) {
	files := map[string][]byte{"assets/photo.png": {137, 80, 78, 71}}
	body, e := sanitizeExportHTML(`<table onclick="bad()"><tbody><tr><td colspan="2">표 내용<input type="checkbox" checked onchange="bad()"></td></tr></tbody></table><script>alert(1)</script><img src="https://outside.invalid/track"><img src="file://outside.invalid/x"><img src="assets/photo.png" onerror="bad()"><a href="javascript:alert(1)">unsafe</a><div style="background:url(https://outside.invalid)">안내</div>`, "page.html", files)
	if e != nil {
		t.Fatal(e)
	}
	for _, disallowed := range []string{"onclick", "onchange", "onerror", "outside.invalid", "javascript:", "<script", "style="} {
		if strings.Contains(body, disallowed) {
			t.Fatalf("active HTML preserved: %s", body)
		}
	}
	for _, want := range []string{"<table>", `colspan="2"`, "표 내용", `src="assets/photo.png"`, `disabled=""`} {
		if !strings.Contains(body, want) {
			t.Fatalf("safe structure missing %q in %s", want, body)
		}
	}
}
func TestPostgresExportFormatsRawBytesCurrentSourceAndExpiry(t *testing.T) {
	s, client, ctx, _, wid := jobTestFixture(t)
	client.request("PUT", "/api/v1/admin/settings", map[string]any{"storage_path": t.TempDir()}, 200)
	original := "---\ntags: [portable]\n---\n\n# 원문 보존  \n\n[[다른 문서]]\n\n```go\nfmt.Println(1)\n```\n"
	doc := testJSONObject(t, client.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "바이트 보존", "visibility": "private", "markdown": original}, 200))
	id := str(doc, "id")
	currentDoc := testJSONObject(t, client.request("GET", "/api/v1/documents/"+id, nil, 200))
	original = str(currentDoc, "markdown")
	runs := map[string]string{}
	for _, format := range []string{"markdown", "portable", "html", "json"} {
		created := testJSONObject(t, client.request("POST", "/api/v1/exports", map[string]any{"workspace_id": wid, "format": format, "document_ids": []string{id}}, 200))
		run := str(created, "id")
		runs[format] = run
		drainJobs(t, s)
		got := testJSONObject(t, client.request("GET", "/api/v1/exports/"+run, nil, 200))
		if str(got, "status") != "ready" {
			t.Fatalf("%s not ready: %#v", format, got)
		}
		raw := client.request("GET", "/api/v1/exports/"+run+"/download", nil, 200)
		var cipher []byte
		if e := s.DB.QueryRow(ctx, "SELECT ciphertext FROM export_artifact_blobs WHERE run_id=$1", run).Scan(&cipher); e != nil || bytes.Contains(cipher, []byte("바이트 보존")) || bytes.HasPrefix(cipher, []byte("PK")) {
			t.Fatal("artifact not encrypted", e)
		}
		if format == "json" {
			var vault transferJSONVault
			if json.Unmarshal(raw, &vault) != nil || vault.Format != "madi-json-vault" || len(vault.Files) != 1 || string(vault.Files[0].Data) != original {
				t.Fatal("JSON raw bytes changed")
			}
		} else {
			files := exportTestFiles(t, raw)
			if format == "markdown" && string(files["바이트 보존.md"]) != original {
				t.Fatal("Markdown bytes changed")
			}
			if format == "html" {
				if !strings.Contains(string(files["바이트 보존.html"]), "Content-Security-Policy") || files["index.html"] == nil {
					t.Fatal("offline HTML missing")
				}
			}
		}
	}
	client.request("PUT", "/api/v1/documents/"+id, map[string]any{"title": "변경됨", "markdown": "현재 버전", "version": number(currentDoc, "version", 1)}, 200)
	client.request("GET", "/api/v1/exports/"+runs["markdown"]+"/download", nil, 409)
	_, e := s.DB.Exec(ctx, "UPDATE export_runs SET expires_at=now()-interval '1 second' WHERE id=$1", runs["json"])
	if e != nil {
		t.Fatal(e)
	}
	s.expireExports(ctx)
	var count int
	if e = s.DB.QueryRow(ctx, "SELECT count(*) FROM export_artifact_blobs WHERE run_id=$1", runs["json"]).Scan(&count); e != nil || count != 0 {
		t.Fatal("expired blob retained", e)
	}
	client.request("DELETE", "/api/v1/exports/"+runs["portable"], nil, 200)
}
func TestPostgresExportCSVAndNotionMixedAtomicImport(t *testing.T) {
	s, client, ctx, _, wid := jobTestFixture(t)
	client.request("PUT", "/api/v1/admin/settings", map[string]any{"storage_path": t.TempDir()}, 200)
	stage := testJSONObject(t, storageMultipart(t, client, "/api/v1/migrations?workspace_id="+wid+"&format=notion", "notion.zip", makeTestVault(t, map[string][]byte{"Page.html": []byte("<title>페이지</title><p>혼합 가져오기</p>"), "Task.csv": []byte("제목,상태\n=evil(),완료\n")}), "", 200))
	drainJobs(t, s)
	id := str(stage, "id")
	preview := testJSONObject(t, client.request("GET", "/api/v1/migrations/"+id, nil, 200))
	if number(preview["preview"].(map[string]any), "databases", 0) != 1 {
		t.Fatal("CSV preview missing")
	}
	client.request("POST", "/api/v1/migrations/"+id+"/run", map[string]any{"confirmation": "IMPORT"}, 200)
	drainJobs(t, s)
	var dbID string
	if e := s.DB.QueryRow(ctx, "SELECT id::text FROM databases WHERE workspace_id=$1 AND name='Task'", wid).Scan(&dbID); e != nil {
		t.Fatal(e)
	}
	run := testJSONObject(t, client.request("POST", "/api/v1/exports", map[string]any{"workspace_id": wid, "format": "csv", "database_id": dbID}, 200))
	drainJobs(t, s)
	raw := client.request("GET", "/api/v1/exports/"+str(run, "id")+"/download", nil, 200)
	if !strings.Contains(string(raw), "'=evil()") || !strings.Contains(string(raw), "完成") && !strings.Contains(string(raw), "완료") {
		t.Fatalf("CSV contract %q", raw)
	}
	backup := client.request("GET", "/api/v1/admin/backup", nil, 200)
	storageMultipart(t, client, "/api/v1/admin/restore", "exports.zip", backup, "RESTORE", 200)
	var status string
	var count int
	var enabled bool
	if e := s.DB.QueryRow(ctx, "SELECT status FROM export_runs WHERE id=$1", str(run, "id")).Scan(&status); e != nil || status != "expired" {
		t.Fatal("restored artifact still ready", status, e)
	}
	_ = s.DB.QueryRow(ctx, "SELECT count(*) FROM export_artifact_blobs").Scan(&count)
	_ = s.DB.QueryRow(ctx, "SELECT enabled FROM export_settings WHERE id").Scan(&enabled)
	if count != 0 || enabled {
		t.Fatal("restored artifact/policy retained")
	}
}

func TestPostgresExportSessionKeyCancellationAndJSONAttachmentIdentity(t *testing.T) {
	s, client, ctx, p, wid := jobTestFixture(t)
	client.request("PUT", "/api/v1/admin/settings", map[string]any{"storage_path": t.TempDir()}, 200)
	doc := testJSONObject(t, client.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "첨부 원문", "markdown": "# 파일", "visibility": "private"}, 200))
	id := str(doc, "id")
	attachment := testJSONObject(t, storageMultipart(t, client, "/api/v1/attachments?document_id="+id, "file.txt", []byte("attachment-fixture-payload"), "", 200))
	aid := str(attachment, "id")
	client.request("PUT", "/api/v1/documents/"+id, map[string]any{"title": "첨부 원문", "markdown": "# 파일\n\n[첨부](/api/v1/attachments/" + aid + ")\n", "version": number(doc, "version", 1)}, 200)
	create := func(c *integrationTestClient, format string) string {
		v := testJSONObject(t, c.request("POST", "/api/v1/exports", map[string]any{"workspace_id": wid, "format": format, "document_ids": []string{id}}, 200))
		return str(v, "id")
	}
	jsonID := create(client, "json")
	drainJobs(t, s)
	raw := client.request("GET", "/api/v1/exports/"+jsonID+"/download", nil, 200)
	stage := testJSONObject(t, storageMultipart(t, client, "/api/v1/migrations?workspace_id="+wid+"&format=json", "vault.json", raw, "", 200))
	drainJobs(t, s)
	client.request("POST", "/api/v1/migrations/"+str(stage, "id")+"/run", map[string]any{"confirmation": "IMPORT"}, 200)
	drainJobs(t, s)
	var imported, markdown, visibility string
	var targetAttachment string
	if e := s.DB.QueryRow(ctx, `SELECT id::text,markdown,visibility FROM documents WHERE workspace_id=$1 AND title='첨부 원문' AND id<>$2`, wid, id).Scan(&imported, &markdown, &visibility); e != nil {
		t.Fatal(e)
	}
	if e := s.DB.QueryRow(ctx, `SELECT id::text FROM attachments WHERE document_id=$1`, imported).Scan(&targetAttachment); e != nil {
		t.Fatal(e)
	}
	if visibility != "private" || strings.Contains(markdown, aid) || !strings.Contains(markdown, targetAttachment) {
		t.Fatalf("JSON attachment identity not remapped: %s", markdown)
	}
	cancelled := create(client, "markdown")
	client.request("DELETE", "/api/v1/exports/"+cancelled, nil, 200)
	drainJobs(t, s)
	var count int
	_ = s.DB.QueryRow(ctx, `SELECT count(*) FROM export_artifact_blobs WHERE run_id=$1`, cancelled).Scan(&count)
	if count != 0 {
		t.Fatal("cancelled export produced blob")
	}
	issued := testJSONObject(t, client.request("POST", "/api/v1/keys", map[string]any{"name": "내보내기 전용", "workspace_id": wid, "scopes": []string{"document:read"}, "expires_in_days": 30, "rate_limit": 1000}, 201))
	key := newIntegrationTestClient(t, client.base)
	key.token = str(issued, "token")
	keyRun := create(key, "markdown")
	drainJobs(t, s)
	key.request("GET", "/api/v1/exports/"+keyRun+"/download", nil, 200)
	if _, e := s.DB.Exec(ctx, `UPDATE api_keys SET scopes='{}' WHERE id=(SELECT token_id FROM export_runs WHERE id=$1)`, keyRun); e != nil {
		t.Fatal(e)
	}
	key.request("GET", "/api/v1/exports/"+keyRun+"/download", nil, 403)
	if _, e := s.DB.Exec(ctx, `DELETE FROM sessions WHERE user_id=$1`, p.ID); e != nil {
		t.Fatal(e)
	}
	client.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200)
	client.request("GET", "/api/v1/exports/"+jsonID+"/download", nil, 409)
}

func TestPostgresExportGenerationSourceChangeNeverReady(t *testing.T) {
	s, client, ctx, _, wid := jobTestFixture(t)
	d := testJSONObject(t, client.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "작업 중 변경", "markdown": "시작 본문"}, 200))
	_, e := s.DB.Exec(ctx, `CREATE FUNCTION export_fixture_change() RETURNS trigger LANGUAGE plpgsql AS $$BEGIN UPDATE documents SET version=version+1,markdown='생성 중 바뀐 본문' WHERE id=NEW.document_ids[1]; RETURN NEW; END$$; CREATE TRIGGER export_fixture_running BEFORE UPDATE ON export_runs FOR EACH ROW WHEN (OLD.status='queued' AND NEW.status='running') EXECUTE FUNCTION export_fixture_change();`)
	if e != nil {
		t.Fatal(e)
	}
	run := testJSONObject(t, client.request("POST", "/api/v1/exports", map[string]any{"workspace_id": wid, "format": "markdown", "document_ids": []string{str(d, "id")}}, 200))
	drainJobs(t, s)
	result := testJSONObject(t, client.request("GET", "/api/v1/exports/"+str(run, "id"), nil, 200))
	var blobs int
	_ = s.DB.QueryRow(ctx, `SELECT count(*) FROM export_artifact_blobs WHERE run_id=$1`, str(run, "id")).Scan(&blobs)
	if str(result, "status") != "failed" || blobs != 0 {
		t.Fatalf("changed source marked ready: %#v blobs=%d", result, blobs)
	}
}
