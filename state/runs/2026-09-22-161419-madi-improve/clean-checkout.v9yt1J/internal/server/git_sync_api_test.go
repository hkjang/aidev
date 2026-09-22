package server

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPostgresGitSyncConfigEncryptionAndSelectedExport(t *testing.T) {
	s, admin, ctx, p, wid := jobTestFixture(t)
	if e := s.migrateGitSync(ctx); e != nil {
		t.Fatal(e)
	}
	registered := false
	for _, route := range s.apiRoutes {
		if route == "GET /api/v1/admin/git-sync/settings" {
			registered = true
		}
	}
	if !registered {
		s.registerGitSync()
	}
	initial := testJSONObject(t, admin.request("GET", "/api/v1/admin/git-sync/settings", nil, 200))
	if boolean(initial["settings"].(map[string]any), "enabled") {
		t.Fatal("Git must default off")
	}
	admin.request("PUT", "/api/v1/admin/git-sync/settings", map[string]any{"revision": initial["revision"], "settings": map[string]any{"enabled": true}}, 400)
	admin.request("PUT", "/api/v1/admin/git-sync/settings", map[string]any{"revision": initial["revision"], "settings": map[string]any{"enabled": true, "allowed_hosts": []string{"git.example.invalid"}}}, 200)
	config := map[string]any{"url": "https://git.example.invalid/team/knowledge.git", "branch": "main", "prefix": "madi", "username": "git-fixture-user", "password": "git-fixture-private-token"}
	form := map[string]any{"workspace_id": wid, "owner_id": p.ID, "space_id": "", "name": "암호화 Git 연결", "enabled": true, "config": config}
	created := testJSONObject(t, admin.request("POST", "/api/v1/git-sync/connections", form, 200))
	id := str(created, "id")
	list := admin.request("GET", "/api/v1/git-sync/connections?workspace_id="+wid, nil, 200)
	if strings.Contains(string(list), "git-fixture-user") || strings.Contains(string(list), "git-fixture-private-token") {
		t.Fatal("Git credentials leaked through list")
	}
	var cipher string
	if e := s.DB.QueryRow(ctx, "SELECT config->>'password' FROM git_sync_connections WHERE id=$1", id).Scan(&cipher); e != nil || !strings.HasPrefix(cipher, "enc:") {
		t.Fatal("Git token not encrypted", e)
	}
	form["revision"] = created["revision"]
	config["password"] = ""
	updated := testJSONObject(t, admin.request("PUT", "/api/v1/git-sync/connections/"+id, form, 200))
	stored, e := s.loadGitSyncConnection(ctx, id)
	if e != nil {
		t.Fatal(e)
	}
	decrypted, e := s.gitSyncDecryptedConfig(stored)
	if e != nil || str(decrypted, "password") != "git-fixture-private-token" {
		t.Fatal("blank update did not preserve token", e)
	}
	form["revision"] = updated["revision"]
	config["url"] = "https://git.example.invalid/other.git"
	admin.request("PUT", "/api/v1/git-sync/connections/"+id, form, 409)
	doc := testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "선택한 문서", "visibility": "private", "markdown": "# 선택 원본"}, 200))
	admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "내보내지 않은 문서", "markdown": "must not export"}, 200)
	export := func(selection vaultExportSelection, want int) []byte {
		t.Helper()
		ctx := context.WithValue(ctx, principalKey, p)
		ctx = context.WithValue(ctx, vaultExportSelectionKey{}, selection)
		r := httptest.NewRequest("GET", "/api/v1/export?workspace_id="+wid, nil).WithContext(ctx)
		w := httptest.NewRecorder()
		s.exportMarkdown(w, r)
		if w.Code != want {
			t.Fatalf("selected export status=%d want=%d %s", w.Code, want, w.Body.String())
		}
		return w.Body.Bytes()
	}
	raw := export(vaultExportSelection{IDs: []string{str(doc, "id")}, AdditionalActorID: p.ID, MaxBytes: 1 << 20}, 200)
	archive, e := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if e != nil {
		t.Fatal(e)
	}
	found := false
	for _, file := range archive.File {
		r, _ := file.Open()
		data, _ := io.ReadAll(r)
		r.Close()
		if strings.Contains(string(data), "must not export") {
			t.Fatal("unselected private content exported")
		}
		if file.Name == "madi-manifest.json" {
			var manifest vaultManifest
			if json.Unmarshal(data, &manifest) != nil || len(manifest.Documents) != 1 || manifest.Documents[0].Version != 1 {
				t.Fatal("source version not preserved")
			}
			found = true
		}
	}
	if !found {
		t.Fatal("selected manifest absent")
	}
	export(vaultExportSelection{IDs: []string{str(doc, "id")}, AdditionalActorID: newID(), MaxBytes: 1 << 20}, 403)
	export(vaultExportSelection{IDs: []string{str(doc, "id")}, AdditionalActorID: p.ID, MaxBytes: 1}, 413)
}
