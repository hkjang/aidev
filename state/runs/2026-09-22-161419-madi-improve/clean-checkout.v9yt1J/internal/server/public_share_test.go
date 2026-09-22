package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestPostgresPublicSharesPasswordRotationACLAndDownloads(t *testing.T) {
	s, admin, ctx, p, wid := jobTestFixture(t)
	admin.request("PUT", "/api/v1/admin/settings", map[string]any{"storage_path": t.TempDir()}, 200)
	policy := defaultProtectionSettings()
	policy["public_shares_enabled"] = true
	policy["public_share_require_password"] = true
	policy["public_share_allow_download"] = true
	if _, e := s.DB.Exec(ctx, "UPDATE protection_settings SET data=$1", jsonValue(policy)); e != nil {
		t.Fatal(e)
	}
	doc := testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "명시적으로 공유한 개인 문서", "markdown": "private explicitly shared", "visibility": "private"}, 200))
	did := str(doc, "id")
	file := testJSONObject(t, storageMultipart(t, admin, "/api/v1/attachments?document_id="+did, "notes.txt", []byte("shared attachment"), "", 200))
	form := map[string]any{"expires_at": time.Now().Add(time.Hour), "password": "Shared-private-password", "ip_allowlist": []string{"127.0.0.1/32"}, "allow_download": false, "allow_copy": false, "confirm_public": true}
	created := testJSONObject(t, admin.request("POST", "/api/v1/documents/"+did+"/public-shares", form, 200))
	share := created["share"].(map[string]any)
	sid := str(share, "id")
	token := strings.Split(str(created, "url"), "#")[1]
	list := admin.request("GET", "/api/v1/documents/"+did+"/public-shares", nil, 200)
	if strings.Contains(string(list), token) || strings.Contains(string(list), "Shared-private-password") || strings.Contains(string(list), "password_hash") {
		t.Fatal("share secret persisted in response")
	}
	public := func(method, path, key, grant string, body any, status int) []byte {
		t.Helper()
		var reader io.Reader
		if body != nil {
			reader = bytes.NewReader(jsonValue(body))
		}
		r, _ := http.NewRequest(method, admin.base+"/api/v1/public-shares/"+sid+path, reader)
		r.Header.Set("X-Madi-Share-Token", key)
		r.Header.Set("X-Madi-Share-Access", grant)
		r.Header.Set("Content-Type", "application/json")
		response, e := http.DefaultClient.Do(r)
		if e != nil {
			t.Fatal(e)
		}
		defer response.Body.Close()
		raw, e := io.ReadAll(response.Body)
		if e != nil {
			t.Fatal(e)
		}
		if response.StatusCode != status {
			t.Fatalf("public %s status=%d want=%d %s", path, response.StatusCode, status, raw)
		}
		if !strings.Contains(response.Header.Get("X-Robots-Tag"), "noindex") || !strings.Contains(response.Header.Get("Cache-Control"), "no-store") || response.Header.Get("Referrer-Policy") != "no-referrer" {
			t.Fatal("public anti-index/cache headers absent")
		}
		return raw
	}
	public("GET", "", "wrong-token", "", nil, 404)
	challenge := public("GET", "", token, "", nil, 401)
	if strings.Contains(string(challenge), "explicitly shared") {
		t.Fatal("password challenge revealed body")
	}
	public("POST", "/unlock", token, "", map[string]any{"password": "wrong"}, 401)
	unlocked := testJSONObject(t, public("POST", "/unlock", token, "", map[string]any{"password": "Shared-private-password"}, 200))
	grant := str(unlocked, "access_token")
	data := testJSONObject(t, public("GET", "", token, grant, nil, 200))
	if str(data, "markdown") != "private explicitly shared" || boolean(data, "allow_download") || boolean(data, "allow_copy") {
		t.Fatal("share view contract")
	}
	public("GET", "/attachments/"+str(file, "id"), token, grant, nil, 403)
	form["revision"] = share["revision"]
	form["allow_download"] = true
	form["password"] = ""
	updated := testJSONObject(t, admin.request("PUT", "/api/v1/documents/"+did+"/public-shares/"+sid, form, 200))
	public("GET", "", token, grant, nil, 401)
	grant = str(testJSONObject(t, public("POST", "/unlock", token, "", map[string]any{"password": "Shared-private-password"}, 200)), "access_token")
	if raw := public("GET", "/attachments/"+str(file, "id"), token, grant, nil, 200); string(raw) != "shared attachment" {
		t.Fatal("download mismatch")
	}
	public("GET", "/attachments/"+newID(), token, grant, nil, 404)
	// Files created before a stricter policy must not bypass it on public delivery.
	legacy := testJSONObject(t, storageMultipart(t, admin, "/api/v1/attachments?document_id="+did, "legacy.txt", []byte("owner@example.test"), "", 200))
	policy["enabled"], policy["mode"] = true, "mask"
	if _, e := s.DB.Exec(ctx, "UPDATE protection_settings SET data=$1", jsonValue(policy)); e != nil {
		t.Fatal(e)
	}
	public("GET", "/attachments/"+str(legacy, "id"), token, grant, nil, 403)
	policy["unscannable"], policy["mode"] = "block", "warn"
	if _, e := s.DB.Exec(ctx, "UPDATE protection_settings SET data=$1", jsonValue(policy)); e != nil {
		t.Fatal(e)
	}
	if _, e := s.DB.Exec(ctx, "UPDATE attachments SET content_type='application/octet-stream' WHERE id=$1", file["id"]); e != nil {
		t.Fatal(e)
	}
	public("GET", "/attachments/"+str(file, "id"), token, grant, nil, 403)
	policy["enabled"], policy["unscannable"] = false, "warn"
	if _, e := s.DB.Exec(ctx, "UPDATE protection_settings SET data=$1", jsonValue(policy)); e != nil {
		t.Fatal(e)
	}
	rotated := testJSONObject(t, admin.request("POST", "/api/v1/documents/"+did+"/public-shares/"+sid+"/rotate", map[string]any{"revision": updated["revision"]}, 200))
	public("GET", "", token, grant, nil, 404)
	token = strings.Split(str(rotated, "url"), "#")[1]
	grant = str(testJSONObject(t, public("POST", "/unlock", token, "", map[string]any{"password": "Shared-private-password"}, 200)), "access_token")
	if _, e := s.DB.Exec(ctx, "INSERT INTO knowledge_document_meta(document_id,classification) VALUES($1,'restricted') ON CONFLICT(document_id) DO UPDATE SET classification='restricted'", did); e != nil {
		t.Fatal(e)
	}
	public("GET", "", token, grant, nil, 404)
	if _, e := s.DB.Exec(ctx, "UPDATE knowledge_document_meta SET classification='internal' WHERE document_id=$1", did); e != nil {
		t.Fatal(e)
	}
	if _, e := s.DB.Exec(ctx, "UPDATE workspace_members SET role='viewer' WHERE user_id=$1 AND workspace_id=$2", p.ID, wid); e != nil {
		t.Fatal(e)
	}
	// Read ACL still exists for a private owner; changing account state revokes it.
	if _, e := s.DB.Exec(ctx, "UPDATE users SET disabled=true WHERE id=$1", p.ID); e != nil {
		t.Fatal(e)
	}
	public("GET", "", token, grant, nil, 404)
	if _, e := s.DB.Exec(ctx, "UPDATE users SET disabled=false WHERE id=$1", p.ID); e != nil {
		t.Fatal(e)
	}
	if _, e := s.DB.Exec(ctx, "UPDATE workspace_members SET role='owner' WHERE user_id=$1 AND workspace_id=$2", p.ID, wid); e != nil {
		t.Fatal(e)
	}
	admin.request("DELETE", "/api/v1/documents/"+did+"/public-shares/"+sid, nil, 200)
	public("GET", "", token, grant, nil, 404)
	var visits []map[string]any
	json.Unmarshal(admin.request("GET", "/api/v1/documents/"+did+"/public-shares/"+sid+"/visits", nil, 200), &visits)
	if len(visits) < 3 {
		t.Fatal("share audit absent")
	}
}

func TestPostgresProtectionFilesAndYAML(t *testing.T) {
	s, admin, ctx, p, wid := jobTestFixture(t)
	policy := defaultProtectionSettings()
	policy["enabled"] = true
	policy["mode"] = "mask"
	if _, e := s.DB.Exec(ctx, "UPDATE protection_settings SET data=$1", jsonValue(policy)); e != nil {
		t.Fatal(e)
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback(ctx)
	protected, e := s.ProtectDocumentTx(ctx, tx, p, "", wid, "owner@example.test", "---\ntags: [owner@example.test]\naliases:\n  - 010-1234-5678\n---\n\n# Memo")
	if e != nil {
		t.Fatal(e)
	}
	front, e := parseFrontMatter(protected.Markdown)
	if e != nil || len(listStrings(front["tags"])) != 1 || len(listStrings(front["aliases"])) != 1 {
		t.Fatal("mask broke YAML", protected.Markdown, e)
	}
	file, e := s.ProtectAttachmentTx(ctx, tx, p, "", wid, "owner@example.test.txt", "text/plain", []byte("010-1234-5678"))
	if e != nil || !file.Changed || strings.Contains(string(file.Data), "010-") || strings.Contains(file.Name, "owner@") {
		t.Fatal("text attachment not sanitized", e)
	}
	binary, e := s.ProtectAttachmentTx(ctx, tx, p, "", wid, "image.png", "image/png", []byte{0, 1, 2})
	if e != nil || !binary.Unscannable {
		t.Fatal("binary not explicitly unscannable", e)
	}
	tx.Rollback(ctx)
	policy["unscannable"] = "block"
	if _, e = s.DB.Exec(ctx, "UPDATE protection_settings SET data=$1", jsonValue(policy)); e != nil {
		t.Fatal(e)
	}
	doc := testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "텍스트 검사"}, 200))
	storageMultipart(t, admin, "/api/v1/attachments?document_id="+str(doc, "id"), "image.png", []byte{0, 1, 2}, "", 422)
}

func TestPostgresPublicShareOwnerAncestorIPExpiryAndRate(t *testing.T) {
	s, admin, ctx, p, wid := jobTestFixture(t)
	policy := defaultProtectionSettings()
	policy["public_shares_enabled"] = true
	if _, e := s.DB.Exec(ctx, "UPDATE protection_settings SET data=$1", jsonValue(policy)); e != nil {
		t.Fatal(e)
	}
	other := testJSONObject(t, admin.request("POST", "/api/v1/admin/users", map[string]any{"email": "public-share-editor@example.test", "name": "편집 동료", "role": "editor", "password": "Public-Editor-Password!"}, 200))
	admin.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": "public-share-editor@example.test", "role": "editor"}, 200)
	colleague := newIntegrationTestClient(t, admin.base)
	colleague.request("POST", "/api/v1/auth/login", map[string]any{"email": "public-share-editor@example.test", "password": "Public-Editor-Password!"}, 200)
	parent := testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "공유 상위 ACL"}, 200))
	doc := testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "공개 공유 하위", "parent_id": parent["id"]}, 200))
	form := map[string]any{"expires_at": time.Now().Add(time.Hour), "allow_copy": true, "confirm_public": true, "ip_allowlist": []string{}}
	colleague.request("POST", "/api/v1/documents/"+str(doc, "id")+"/public-shares", form, 403)
	created := testJSONObject(t, admin.request("POST", "/api/v1/documents/"+str(doc, "id")+"/public-shares", form, 200))
	sid := str(created["share"].(map[string]any), "id")
	token := strings.Split(str(created, "url"), "#")[1]
	get := func(want int) {
		t.Helper()
		r, _ := http.NewRequest("GET", admin.base+"/api/v1/public-shares/"+sid, nil)
		r.Header.Set("X-Madi-Share-Token", token)
		r.Header.Set("X-Forwarded-For", "192.0.2.9")
		res, e := http.DefaultClient.Do(r)
		if e != nil {
			t.Fatal(e)
		}
		defer res.Body.Close()
		body, _ := io.ReadAll(res.Body)
		if res.StatusCode != want {
			t.Fatalf("public status=%d want=%d %s", res.StatusCode, want, body)
		}
	}
	get(200)
	if _, e := s.DB.Exec(ctx, "UPDATE public_shares SET ip_allowlist='[\"192.0.2.0/24\"]' WHERE id=$1", sid); e != nil {
		t.Fatal(e)
	}
	get(404)
	if _, e := s.DB.Exec(ctx, "UPDATE public_shares SET ip_allowlist='[]',expires_at=now()-interval '1 second' WHERE id=$1", sid); e != nil {
		t.Fatal(e)
	}
	get(404)
	if _, e := s.DB.Exec(ctx, "UPDATE public_shares SET expires_at=now()+interval '1 hour' WHERE id=$1", sid); e != nil {
		t.Fatal(e)
	}
	if _, e := s.DB.Exec(ctx, "UPDATE documents SET owner_id=$2,visibility='selected' WHERE id=$1", parent["id"], other["id"]); e != nil {
		t.Fatal(e)
	}
	if _, e := s.DB.Exec(ctx, "INSERT INTO document_shares(document_id,user_id,permission) VALUES($1,$2,'read')", parent["id"], p.ID); e != nil {
		t.Fatal(e)
	}
	get(200)
	if _, e := s.DB.Exec(ctx, "DELETE FROM document_shares WHERE document_id=$1 AND user_id=$2", parent["id"], p.ID); e != nil {
		t.Fatal(e)
	}
	get(404)
	if !s.publicShareRate(ctx, "isolated-rate-test", 2, time.Minute) || !s.publicShareRate(ctx, "isolated-rate-test", 2, time.Minute) || s.publicShareRate(ctx, "isolated-rate-test", 2, time.Minute) {
		t.Fatal("public rate threshold not enforced")
	}
}

func TestPostgresPublicSharesRestoreRevokesExternalAccess(t *testing.T) {
	s, admin, ctx, _, wid := jobTestFixture(t)
	admin.request("PUT", "/api/v1/admin/settings", map[string]any{"storage_path": t.TempDir()}, 200)
	policy := defaultProtectionSettings()
	policy["public_shares_enabled"] = true
	if _, e := s.DB.Exec(ctx, "UPDATE protection_settings SET data=$1", jsonValue(policy)); e != nil {
		t.Fatal(e)
	}
	doc := testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "복원 후 공유 재동의"}, 200))
	created := testJSONObject(t, admin.request("POST", "/api/v1/documents/"+str(doc, "id")+"/public-shares", map[string]any{"expires_at": time.Now().Add(time.Hour), "confirm_public": true}, 200))
	sid := str(created["share"].(map[string]any), "id")
	if _, e := s.DB.Exec(ctx, "INSERT INTO public_share_access(token_hash,share_id,revision,ip_hash,expires_at) VALUES($1,$2,1,$3,now()+interval '1 hour')", digest("fixture-grant"), sid, digest("127.0.0.1")); e != nil {
		t.Fatal(e)
	}
	backup := admin.request("GET", "/api/v1/admin/backup", nil, 200)
	storageMultipart(t, admin, "/api/v1/admin/restore", "backup.zip", backup, "RESTORE", 200)
	var disabled, revoked bool
	var grants int
	if e := s.DB.QueryRow(ctx, "SELECT NOT (data->>'public_shares_enabled')::boolean FROM protection_settings WHERE id=1").Scan(&disabled); e != nil {
		t.Fatal(e)
	}
	if e := s.DB.QueryRow(ctx, "SELECT revoked_at IS NOT NULL FROM public_shares WHERE id=$1", sid).Scan(&revoked); e != nil {
		t.Fatal(e)
	}
	if e := s.DB.QueryRow(ctx, "SELECT count(*) FROM public_share_access").Scan(&grants); e != nil {
		t.Fatal(e)
	}
	if !disabled || !revoked || grants != 0 {
		t.Fatal("restored share/access is still live")
	}
}
