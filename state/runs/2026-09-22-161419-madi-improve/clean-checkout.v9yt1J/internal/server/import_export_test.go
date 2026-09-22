package server

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"sort"
	"strings"
	"testing"
)

func makeTestVault(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var out bytes.Buffer
	writer := zip.NewWriter(&out)
	names := []string{}
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		file, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = file.Write(files[name]); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func uploadTestVault(t *testing.T, client *integrationTestClient, wid string, vault []byte, status int) map[string]any {
	t.Helper()
	var body bytes.Buffer
	multipartWriter := multipart.NewWriter(&body)
	file, err := multipartWriter.CreateFormFile("file", "vault.zip")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = file.Write(vault)
	_ = multipartWriter.Close()
	r, err := http.NewRequest("POST", client.base+"/api/v1/import?workspace_id="+wid, &body)
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set("Content-Type", multipartWriter.FormDataContentType())
	r.Header.Set("X-Madi-Request", "1")
	response, err := client.client.Do(r)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	out, _ := io.ReadAll(response.Body)
	if response.StatusCode != status {
		t.Fatalf("import got %d want %d: %s", response.StatusCode, status, out)
	}
	return testJSONObject(t, out)
}

func TestVaultSafetyAndLiteralCode(t *testing.T) {
	for _, name := range []string{"../escape.md", "/absolute.md", "a/../../b.md", "folder\\evil.md", "C:/drive.md", "folder//nested.md"} {
		if _, err := parseVaultInput("vault.zip", makeTestVault(t, map[string][]byte{name: []byte("# bad")})); err == nil {
			t.Errorf("unsafe ZIP accepted: %s", name)
		}
	}
	var out bytes.Buffer
	writer := zip.NewWriter(&out)
	header := &zip.FileHeader{Name: "link.md"}
	header.SetMode(os.ModeSymlink | 0600)
	file, _ := writer.CreateHeader(header)
	_, _ = file.Write([]byte("../../outside"))
	_ = writer.Close()
	if _, err := parseVaultInput("vault.zip", out.Bytes()); err == nil {
		t.Fatal("symlink accepted")
	}
	md := "---\naliases: ['[[Target]]']\n---\n[[Target]] and `[[Target]]`\n```md\n[[Target]]\n```\n    [[Target]]\n"
	got := rewriteVaultContent(md, func(text string) string { return strings.ReplaceAll(text, "[[Target]]", "[[Changed]]") })
	if strings.Count(got, "[[Changed]]") != 1 || strings.Count(got, "[[Target]]") != 4 {
		t.Fatalf("code/metadata were modified: %s", got)
	}
}

func TestPostgresVaultRejectsDeepHierarchyAtomically(t *testing.T) {
	s, client, ctx, _, wid := jobTestFixture(t)
	var before int
	if e := s.DB.QueryRow(ctx, "SELECT count(*) FROM documents WHERE workspace_id=$1", wid).Scan(&before); e != nil {
		t.Fatal(e)
	}
	deep := strings.Repeat("folder/", 20) + "unreachable.md"
	uploadTestVault(t, client, wid, makeTestVault(t, map[string][]byte{deep: []byte("# Too deep"), "asset.txt": []byte("not persisted")}), 400)
	var documents, attachments int
	if e := s.DB.QueryRow(ctx, "SELECT count(*) FROM documents WHERE workspace_id=$1", wid).Scan(&documents); e != nil {
		t.Fatal(e)
	}
	if e := s.DB.QueryRow(ctx, "SELECT count(*) FROM attachments").Scan(&attachments); e != nil {
		t.Fatal(e)
	}
	if documents != before || attachments != 0 {
		t.Fatalf("rejected hierarchy persisted documents=%d attachments=%d", documents, attachments)
	}
}

func TestPostgresVaultRoundTripAttachmentsHierarchyAndRollback(t *testing.T) {
	s, server := integrationTestServer(t)
	admin := newIntegrationTestClient(t, server.URL)
	admin.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200)
	storage := t.TempDir()
	admin.request("PUT", "/api/v1/admin/settings", map[string]any{"storage_path": storage}, 200)
	workspace := testJSONObject(t, admin.request("POST", "/api/v1/workspaces", map[string]any{"name": "Vault import"}, 200))
	wid := str(workspace, "id")
	guide := `---
title: 운영 가이드
tags: [운영, 테스트]
aliases: [Runbook]
visibility: private
---
# 운영 가이드

[[Other]]
![[../assets/picture.png]]
[매뉴얼](../assets/notes.pdf)

` + "`[[Other]]`\n\n```markdown\n[[Other]]\n```\n"
	vault := makeTestVault(t, map[string][]byte{"Team/Guide.md": []byte(guide), "Team/Other.md": []byte("# Other\n\n[[Guide]]\n[가이드](Guide.md)\n"), "assets/picture.png": []byte("\x89PNG\r\nfixture"), "assets/notes.pdf": []byte("%PDF-test-fixture")})
	result := uploadTestVault(t, admin, wid, vault, 200)
	if result["imported"] != float64(2) || result["folders"] != float64(1) || result["attachments"] != float64(2) {
		t.Fatalf("incorrect import report: %v", result)
	}
	loadDocs := func(workspaceID string) map[string]map[string]any {
		t.Helper()
		var docs []map[string]any
		if err := json.Unmarshal(admin.request("GET", "/api/v1/documents?workspace_id="+workspaceID, nil, 200), &docs); err != nil {
			t.Fatal(err)
		}
		out := map[string]map[string]any{}
		for _, doc := range docs {
			out[str(doc, "title")] = testJSONObject(t, admin.request("GET", "/api/v1/documents/"+str(doc, "id"), nil, 200))
		}
		return out
	}
	docs := loadDocs(wid)
	if len(docs) != 3 || str(docs["운영 가이드"], "parent_id") != str(docs["Team"], "id") || str(docs["Other"], "parent_id") != str(docs["Team"], "id") {
		t.Fatalf("hierarchy lost: %v", docs)
	}
	if str(docs["운영 가이드"], "visibility") != "private" || len(listStrings(docs["운영 가이드"]["tags"])) != 2 || !strings.Contains(str(docs["Other"], "markdown"), "[[운영 가이드]]") {
		t.Fatal("front matter, private visibility or links lost")
	}
	if !strings.Contains(str(docs["Other"], "markdown"), "/app/documents/"+str(docs["운영 가이드"], "id")) {
		t.Fatal("ordinary Markdown document link not resolved")
	}
	assertAttachments := func(documentID, markdown string) {
		t.Helper()
		rows, err := s.rows(context.Background(), "SELECT to_jsonb(a) FROM attachments a WHERE document_id=$1", documentID)
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != 2 {
			t.Fatalf("attachments lost: %v", rows)
		}
		for _, a := range rows {
			if !strings.Contains(markdown, "/api/v1/attachments/"+str(a, "id")) {
				t.Fatalf("attachment link not rewritten: %s", markdown)
			}
			data := admin.request("GET", "/api/v1/attachments/"+str(a, "id"), nil, 200)
			if !bytes.Contains(data, []byte("fixture")) {
				t.Fatal("attachment bytes changed")
			}
		}
	}
	assertAttachments(str(docs["운영 가이드"], "id"), str(docs["운영 가이드"], "markdown"))
	exported := admin.request("GET", "/api/v1/export?workspace_id="+wid, nil, 200)
	parsed, err := parseVaultInput("export.zip", exported)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.manifest == nil || len(parsed.manifest.Documents) != 3 || len(parsed.manifest.Attachments) != 2 {
		t.Fatalf("manifest missing: %+v", parsed.manifest)
	}
	if _, exists := parsed.files["Team/운영 가이드.md"]; !exists {
		t.Fatal("human-readable folder/title paths missing")
	}
	exportedGuide := string(parsed.files["Team/운영 가이드.md"])
	if !strings.Contains(exportedGuide, "[[Team/Other|Other]]") || !strings.Contains(exportedGuide, "../attachments/") || strings.Contains(exportedGuide, "/api/v1/attachments/") {
		t.Fatalf("external vault links not portable: %s", exportedGuide)
	}
	for _, match := range vaultMarkdownLink.FindAllStringSubmatch(exportedGuide, -1) {
		target := strings.TrimSpace(match[2])
		if strings.HasPrefix(target, "../attachments/") {
			if resolveVaultFile(parsed.files, "Team/운영 가이드.md", target) == "" {
				t.Fatalf("exported file missing: %s", target)
			}
		}
	}
	second := testJSONObject(t, admin.request("POST", "/api/v1/workspaces", map[string]any{"name": "Round trip"}, 200))
	secondID := str(second, "id")
	uploadTestVault(t, admin, secondID, exported, 200)
	roundtrip := loadDocs(secondID)
	if len(roundtrip) != 3 || str(roundtrip["운영 가이드"], "visibility") != "private" || str(roundtrip["운영 가이드"], "parent_id") != str(roundtrip["Team"], "id") || !strings.Contains(str(roundtrip["Other"], "markdown"), "/app/documents/"+str(roundtrip["운영 가이드"], "id")) {
		t.Fatalf("madi round trip lost metadata/hierarchy/link: %v", roundtrip)
	}
	assertAttachments(str(roundtrip["운영 가이드"], "id"), str(roundtrip["운영 가이드"], "markdown"))
	before := len(roundtrip)
	uploadTestVault(t, admin, secondID, makeTestVault(t, map[string][]byte{"../escape.md": []byte("# denied")}), 400)
	if len(loadDocs(secondID)) != before {
		t.Fatal("rejected traversal changed documents")
	}
	entriesBefore, err := os.ReadDir(storage)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.DB.Exec(context.Background(), `ALTER TABLE attachments ADD CONSTRAINT reject_fixture CHECK (name <> 'trigger-failure.pdf')`); err != nil {
		t.Fatal(err)
	}
	uploadTestVault(t, admin, secondID, makeTestVault(t, map[string][]byte{"Rollback.md": []byte("[file](trigger-failure.pdf)"), "trigger-failure.pdf": []byte("failure")}), 500)
	if len(loadDocs(secondID)) != before {
		t.Fatal("failed attachment import did not rollback documents")
	}
	entriesAfter, err := os.ReadDir(storage)
	if err != nil {
		t.Fatal(err)
	}
	if len(entriesBefore) != len(entriesAfter) {
		t.Fatal("failed import left file directory behind")
	}
	for _, entry := range entriesAfter {
		if path.Base(entry.Name()) != entry.Name() {
			t.Fatal("unexpected storage path")
		}
	}
}

func TestVaultExportFilenameCollisionsAndManifestCycle(t *testing.T) {
	docs := []*vaultDocument{{ID: "00000001-0000-4000-8000-000000000000", Title: "Duplicate"}, {ID: "00000002-0000-4000-8000-000000000000", Title: "Duplicate"}, {ID: "00000003-0000-4000-8000-000000000000", Title: "Child", ParentID: "00000002-0000-4000-8000-000000000000"}}
	if err := vaultExportPaths(docs); err != nil {
		t.Fatal(err)
	}
	if docs[0].File == docs[1].File || !strings.HasPrefix(docs[2].File, strings.TrimSuffix(docs[1].File, ".md")+"/") {
		t.Fatalf("filename collision: %+v", docs)
	}
	input := &vaultInput{files: map[string][]byte{"A.md": []byte("A"), "B.md": []byte("B")}, manifest: &vaultManifest{Documents: []*vaultDocument{{ID: "a", File: "A.md", Title: "A", ParentID: "b"}, {ID: "b", File: "B.md", Title: "B", ParentID: "a"}}}}
	if _, _, err := prepareVaultDocuments(input); err == nil {
		t.Fatal("cyclic manifest accepted")
	}
}

func TestPostgresVaultExportExceeds2000Documents(t *testing.T) {
	s, server := integrationTestServer(t)
	admin := newIntegrationTestClient(t, server.URL)
	admin.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200)
	workspace := testJSONObject(t, admin.request("POST", "/api/v1/workspaces", map[string]any{"name": "Large vault"}, 200))
	wid := str(workspace, "id")
	_, err := s.DB.Exec(context.Background(), `INSERT INTO documents(id,workspace_id,title,markdown,owner_id) SELECT md5($1::text || '-' || g::text)::uuid,$1::uuid,'문서 '||g::text,'# document',u.id FROM generate_series(1,2001) g CROSS JOIN users u WHERE u.email='admin@example.test'`, wid)
	if err != nil {
		t.Fatal(err)
	}
	data := admin.request("GET", "/api/v1/export?workspace_id="+wid, nil, 200)
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, file := range reader.File {
		if strings.HasSuffix(file.Name, ".md") {
			count++
		}
	}
	if count != 2001 {
		t.Fatalf("whole workspace export truncated: got %d want 2001", count)
	}
}
