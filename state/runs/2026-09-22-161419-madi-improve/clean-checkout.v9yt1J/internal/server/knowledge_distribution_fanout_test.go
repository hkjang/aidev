package server

import (
	"crypto/ed25519"
	"encoding/base64"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestPostgresDistributionAllAttachmentCopiesMappedAndACLFiltered(t *testing.T) {
	s, c, ctx, _, wid := jobTestFixture(t)
	cfg := testJSONObject(t, c.request("GET", "/api/v1/admin/knowledge-distribution", nil, 200))
	c.request("PUT", "/api/v1/admin/knowledge-distribution", map[string]any{"revision": 1, "enabled": true, "max_valid_days": 7, "consent": true}, 200)
	m, private := distributionTestManifest()
	m.ReceiverInstance = str(cfg["policy"].(map[string]any), "instance_id")
	second := m.Files[0]
	second.SourceID = newID()
	second.Path = "참고.md"
	second.Metadata.Title = "다른 운영 문서"
	second.MetadataHash = digest(string(jsonValue(second.Metadata.value())))
	asset := distributionFile{SourceID: newID(), Path: "첨부.txt", Kind: "attachment", ParentSourceID: m.Files[0].SourceID, Metadata: distributionMetadata{Title: "첨부.txt", Tags: []string{}, Aliases: []string{}}}
	asset.MetadataHash = digest(string(jsonValue(asset.Metadata.value())))
	bodies := map[string]string{m.Files[0].SourceID: "첫 운영 자료\n\n[동일 첨부](첨부.txt)", second.SourceID: "다른 운영 자료\n\n[동일 첨부](첨부.txt)", asset.SourceID: "같은 파일 근거"}
	m.Files = append(m.Files, second, asset)
	for i := range m.Files {
		m.Files[i].Bytes = int64(len(bodies[m.Files[i].SourceID]))
		m.Files[i].SHA256 = digest(bodies[m.Files[i].SourceID])
	}
	public := private.Public().(ed25519.PublicKey)
	c.request("POST", "/api/v1/admin/knowledge-distribution/keys", map[string]any{"kind": "trusted", "label": "반입 사본 회귀", "source_instance": m.SourceInstance, "source_key_id": m.KeyID, "public_key": base64.StdEncoding.EncodeToString(public), "fingerprint": digest(string(public)), "consent": true}, 201)
	v := testJSONObject(t, c.request("POST", "/api/v1/migrations/sessions", map[string]any{"workspace_id": wid, "source_key": m.sourceKey(), "label": "문서별 첨부 대응", "format": "markdown"}, 200))
	id := str(v, "id")
	path := "/api/v1/migrations/sessions/" + id
	raw := jsonValue(m)
	c.request("POST", path+"/signed-source", map[string]any{"manifest_base64": base64.StdEncoding.EncodeToString(raw), "signature": base64.StdEncoding.EncodeToString(ed25519.Sign(private, raw)), "consent": true}, 200)
	for _, f := range m.Files {
		added := testJSONObject(t, c.request("POST", path+"/items", map[string]any{"items": []any{map[string]any{"source_id": f.SourceID, "path": f.Path, "kind": f.Kind, "sha256": f.SHA256, "bytes": f.Bytes, "parent_source_id": f.ParentSourceID, "metadata": f.Metadata.value()}}}, 200))
		item := added["items"].([]any)[0].(map[string]any)
		req, e := http.NewRequest("PUT", c.base+path+"/items/"+str(item, "id")+"/chunks/0", strings.NewReader(bodies[f.SourceID]))
		if e != nil {
			t.Fatal(e)
		}
		req.Header.Set("X-Madi-Request", "1")
		req.Header.Set("Origin", c.base)
		req.Header.Set("Content-Type", "application/octet-stream")
		response, e := c.client.Do(req)
		if e != nil {
			t.Fatal(e)
		}
		body, _ := io.ReadAll(response.Body)
		response.Body.Close()
		if response.StatusCode != 200 {
			t.Fatal(response.StatusCode, string(body))
		}
	}
	v = testJSONObject(t, c.request("GET", path, nil, 200))
	c.request("POST", path+"/prepare", map[string]any{"revision": v["revision"]}, 200)
	drainJobs(t, s)
	v = testJSONObject(t, c.request("GET", path, nil, 200))
	if str(v, "status") != "ready" {
		t.Fatal(v)
	}
	c.request("POST", path+"/commit", map[string]any{"revision": v["revision"], "plan_hash": v["plan_hash"], "confirmation": "IMPORT"}, 200)
	drainJobs(t, s)
	v = testJSONObject(t, c.request("GET", path, nil, 200))
	if str(v, "status") != "completed" {
		var failure string
		s.DB.QueryRow(ctx, `SELECT last_error FROM automation_jobs WHERE id=$1`, v["job_id"]).Scan(&failure)
		t.Fatal(v, failure)
	}
	receipt := testJSONObject(t, c.request("GET", path+"/signed-source", nil, 200))
	mapped := map[string]bool{}
	copies := 0
	oneDoc := ""
	for _, item := range receipt["mapping"].([]any) {
		m := item.(map[string]any)
		if str(m, "kind") != "attachment" {
			continue
		}
		mapped[str(m, "target_id")] = true
		if str(m, "disposition") == "reference_copy" {
			copies++
			oneDoc = str(m, "target_document_id")
			if str(m, "target_content_sha256") != digest(bodies[asset.SourceID]) {
				t.Fatal(m)
			}
		}
	}
	if copies != 2 {
		t.Fatal("expected both document reference copies", receipt)
	}
	rows, e := s.DB.Query(ctx, `SELECT a.id::text FROM attachments a JOIN documents d ON d.id=a.document_id WHERE d.workspace_id=$1`, wid)
	if e != nil {
		t.Fatal(e)
	}
	for rows.Next() {
		var aid string
		rows.Scan(&aid)
		if !mapped[aid] {
			t.Fatal("imported attachment missing from signed mapping", aid)
		}
	}
	rows.Close()
	// Transfer one target's ownership and make it private; receipt must not leak
	// that target's document or attachment IDs to its former owner/admin.
	other := testJSONObject(t, c.request("POST", "/api/v1/admin/users", map[string]any{"email": "distribution-copies@example.test", "name": "사본 소유자", "role": "editor", "password": "Distribution-Copies-Password-2026!"}, 200))
	c.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": other["email"], "role": "editor"}, 200)
	if _, e = s.DB.Exec(ctx, `UPDATE documents SET owner_id=$2,visibility='private' WHERE id=$1`, oneDoc, other["id"]); e != nil {
		t.Fatal(e)
	}
	hidden := string(c.request("GET", path+"/signed-source", nil, 200))
	if strings.Contains(hidden, oneDoc) {
		t.Fatal("revoked copy mapping leaked target", hidden)
	}
}
