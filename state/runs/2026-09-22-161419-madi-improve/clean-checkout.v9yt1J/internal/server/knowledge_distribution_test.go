package server

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func distributionTestManifest() (distributionManifest, ed25519.PrivateKey) {
	_, private, _ := ed25519.GenerateKey(rand.Reader)
	meta := distributionMetadata{Title: "망별 운영 자료", Tags: []string{"운영"}, Aliases: []string{}, Icon: ""}
	f := distributionFile{SourceID: newID(), Path: "운영.md", Kind: "document", Bytes: len64("서명 원본\n"), SHA256: digest("서명 원본\n"), SourceVersion: 3, Metadata: meta, MetadataHash: digest(string(jsonValue(meta.value())))}
	return distributionManifest{Format: "madi-distribution-v1", BundleID: newID(), SourceInstance: newID(), SourceWorkspace: newID(), ReceiverInstance: newID(), KeyID: newID(), ProtectionRevision: 1, CreatedAt: time.Now().Unix(), ExpiresAt: time.Now().Add(time.Hour).Unix(), Files: []distributionFile{f}}, private
}
func len64(v string) int64 { return int64(len(v)) }
func distributionTestZIP(t *testing.T, m distributionManifest, private ed25519.PrivateKey, changed bool, symlink bool) []byte {
	t.Helper()
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	for _, f := range m.Files {
		h := &zip.FileHeader{Name: f.Path, Method: zip.Deflate}
		if symlink {
			h.SetMode(os.ModeSymlink | 0600)
		}
		w, e := z.CreateHeader(h)
		if e != nil {
			t.Fatal(e)
		}
		body := "서명 원본\n"
		if changed {
			body = "서명 변조\n"
		}
		if _, e = w.Write([]byte(body)); e != nil {
			t.Fatal(e)
		}
	}
	raw := jsonValue(m)
	w, _ := z.Create(distributionManifestPath)
	w.Write(raw)
	w, _ = z.Create(distributionSignaturePath)
	w.Write([]byte(base64.StdEncoding.EncodeToString(ed25519.Sign(private, raw))))
	if e := z.Close(); e != nil {
		t.Fatal(e)
	}
	return b.Bytes()
}
func TestDistributionCanonicalSignatureAndArchiveBounds(t *testing.T) {
	m, private := distributionTestManifest()
	raw := jsonValue(m)
	parsed, e := parseDistributionManifest(raw)
	if e != nil || parsed.BundleID != m.BundleID {
		t.Fatal(parsed, e)
	}
	public := base64.StdEncoding.EncodeToString(private.Public().(ed25519.PublicKey))
	sig := base64.StdEncoding.EncodeToString(ed25519.Sign(private, raw))
	if e = verifyDistributionSignature(raw, sig, public); e != nil {
		t.Fatal(e)
	}
	for _, bad := range [][]byte{append([]byte(" "), raw...), []byte(strings.Replace(string(raw), `"format":`, `"format":"madi-distribution-v1","format":`, 1)), []byte(strings.Replace(string(raw), `"bundle_id":`, `"unknown":1,"bundle_id":`, 1))} {
		if _, e = parseDistributionManifest(bad); e == nil {
			t.Fatal("accepted ambiguous canonical JSON")
		}
	}
	changed := m
	changed.ReceiverInstance = newID()
	if verifyDistributionSignature(jsonValue(changed), sig, public) == nil {
		t.Fatal("accepted receiver substitution")
	}
	for _, edit := range []func(*distributionManifest){func(v *distributionManifest) { v.Files[0].Path = "../outside" }, func(v *distributionManifest) { v.Files[0].Bytes = distributionMaxBytes + 1 }, func(v *distributionManifest) { v.Files[0].Metadata.Title = "changed" }, func(v *distributionManifest) { v.Files = append(v.Files, v.Files[0]) }, func(v *distributionManifest) { v.Files[0].ParentSourceID = v.Files[0].SourceID }, func(v *distributionManifest) { v.Files[0].Approval.Required = true }} {
		var v distributionManifest
		json.Unmarshal(raw, &v)
		edit(&v)
		if _, e = parseDistributionManifest(jsonValue(v)); e == nil {
			t.Fatal("accepted invalid manifest", v)
		}
	}
	for _, item := range []struct{ changed, symlink bool }{{true, false}, {false, true}} {
		if _, _, _, _, e = readDistributionArchive(context.Background(), distributionTestZIP(t, m, private, item.changed, item.symlink)); e == nil {
			t.Fatal("accepted modified or symlink archive")
		}
	}
	_, got, signature, files, e := readDistributionArchive(context.Background(), distributionTestZIP(t, m, private, false, false))
	if e != nil || len(files) != 1 || verifyDistributionSignature(got, signature, public) != nil {
		t.Fatal(e)
	}
	if distributionCurrentTime(m, time.Unix(m.ExpiresAt, 0), 7) == nil || distributionCurrentTime(m, time.Unix(m.CreatedAt-301, 0), 7) == nil {
		t.Fatal("accepted expired/future signature")
	}
}

func uploadDistributionTestFiles(t *testing.T, c *integrationTestClient, id string, m distributionManifest) {
	t.Helper()
	for _, f := range m.Files {
		out := testJSONObject(t, c.request("POST", "/api/v1/migrations/sessions/"+id+"/items", map[string]any{"items": []map[string]any{{"source_id": f.SourceID, "path": f.Path, "kind": f.Kind, "sha256": f.SHA256, "bytes": f.Bytes, "parent_source_id": f.ParentSourceID, "metadata": f.Metadata.value()}}}, 200))
		item := out["items"].([]any)[0].(map[string]any)
		req, e := http.NewRequest("PUT", c.base+"/api/v1/migrations/sessions/"+id+"/items/"+str(item, "id")+"/chunks/0", strings.NewReader("서명 원본\n"))
		if e != nil {
			t.Fatal(e)
		}
		req.Header.Set("Origin", c.base)
		req.Header.Set("X-Madi-Request", "1")
		req.Header.Set("Content-Type", "application/octet-stream")
		res, e := c.client.Do(req)
		if e != nil {
			t.Fatal(e)
		}
		raw, _ := io.ReadAll(res.Body)
		res.Body.Close()
		if res.StatusCode != 200 {
			t.Fatalf("upload %d %s", res.StatusCode, raw)
		}
	}
}
func TestPostgresDistributionTrustedImportAtomicReceiptAndRevocation(t *testing.T) {
	s, c, ctx, _, wid := jobTestFixture(t)
	settings := testJSONObject(t, c.request("GET", "/api/v1/admin/knowledge-distribution", nil, 200))
	policy := settings["policy"].(map[string]any)
	c.request("PUT", "/api/v1/admin/knowledge-distribution", map[string]any{"revision": 1, "enabled": true, "max_valid_days": 7, "consent": true}, 200)
	m, private := distributionTestManifest()
	m.ReceiverInstance = str(policy, "instance_id")
	public := private.Public().(ed25519.PublicKey)
	key := testJSONObject(t, c.request("POST", "/api/v1/admin/knowledge-distribution/keys", map[string]any{"kind": "trusted", "label": "별도 확인한 반출망", "source_instance": m.SourceInstance, "source_key_id": m.KeyID, "public_key": base64.StdEncoding.EncodeToString(public), "fingerprint": digest(string(public)), "consent": true}, 201))
	prepare := func(manifest distributionManifest, bindStatus int) string {
		v := testJSONObject(t, c.request("POST", "/api/v1/migrations/sessions", map[string]any{"workspace_id": wid, "source_key": manifest.sourceKey(), "label": "검증된 반입", "format": "markdown"}, 200))
		id := str(v, "id")
		raw := jsonValue(manifest)
		c.request("POST", "/api/v1/migrations/sessions/"+id+"/signed-source", map[string]any{"manifest_base64": base64.StdEncoding.EncodeToString(raw), "signature": base64.StdEncoding.EncodeToString(ed25519.Sign(private, raw)), "consent": true}, bindStatus)
		if bindStatus != 200 {
			return id
		}
		uploadDistributionTestFiles(t, c, id, manifest)
		v = testJSONObject(t, c.request("GET", "/api/v1/migrations/sessions/"+id, nil, 200))
		c.request("POST", "/api/v1/migrations/sessions/"+id+"/prepare", map[string]any{"revision": v["revision"]}, 200)
		drainJobs(t, s)
		v = testJSONObject(t, c.request("GET", "/api/v1/migrations/sessions/"+id, nil, 200))
		if str(v, "status") != "ready" {
			t.Fatal(v)
		}
		return id
	}
	foreign := m
	foreign.ReceiverInstance = newID()
	prepare(foreign, 422)
	id := prepare(m, 200)
	var before int
	s.DB.QueryRow(ctx, `SELECT count(*) FROM documents WHERE workspace_id=$1`, wid).Scan(&before)
	commit := func(id string) map[string]any {
		v := testJSONObject(t, c.request("GET", "/api/v1/migrations/sessions/"+id, nil, 200))
		c.request("POST", "/api/v1/migrations/sessions/"+id+"/commit", map[string]any{"revision": v["revision"], "plan_hash": v["plan_hash"], "confirmation": "IMPORT"}, 200)
		drainJobs(t, s)
		return testJSONObject(t, c.request("GET", "/api/v1/migrations/sessions/"+id, nil, 200))
	}
	// Receipt write failure must roll back actual documents, not just the label.
	_, e := s.DB.Exec(ctx, `CREATE FUNCTION distribution_fail_fixture() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.imported_at IS NOT NULL THEN RAISE EXCEPTION 'fixture receipt failure'; END IF; RETURN NEW; END $$; CREATE TRIGGER distribution_fail_fixture BEFORE UPDATE ON knowledge_distribution_imports FOR EACH ROW EXECUTE FUNCTION distribution_fail_fixture()`)
	if e != nil {
		t.Fatal(e)
	}
	if out := commit(id); str(out, "status") == "completed" {
		t.Fatal(out)
	}
	var after int
	s.DB.QueryRow(ctx, `SELECT count(*) FROM documents WHERE workspace_id=$1`, wid).Scan(&after)
	if after != before {
		t.Fatal("partial publication", before, after)
	}
	s.DB.Exec(ctx, `DROP TRIGGER distribution_fail_fixture ON knowledge_distribution_imports`)
	m.BundleID = newID()
	id = prepare(m, 200)
	if out := commit(id); str(out, "status") != "completed" {
		var failure string
		s.DB.QueryRow(ctx, `SELECT last_error FROM automation_jobs WHERE id=$1`, str(out, "job_id")).Scan(&failure)
		t.Fatal(out, failure)
	}
	receipt := testJSONObject(t, c.request("GET", "/api/v1/migrations/sessions/"+id+"/signed-source", nil, 200))
	mapping := receipt["mapping"].([]any)
	if receipt["imported_at"] == nil || len(mapping) != 1 || number(mapping[0].(map[string]any), "source_version", 0) != 3 {
		t.Fatal(receipt)
	}
	target := str(mapping[0].(map[string]any), "target_id")
	doc := testJSONObject(t, c.request("GET", "/api/v1/documents/"+target, nil, 200))
	if str(doc, "visibility") != "private" || str(doc, "status") != "draft" {
		t.Fatal("import granted publication", doc)
	}
	m.BundleID = newID()
	id = prepare(m, 200)
	c.request("POST", "/api/v1/admin/knowledge-distribution/keys/"+str(key, "id")+"/revoke", map[string]any{"revision": 1, "confirmation": "REVOKE"}, 200)
	if out := commit(id); str(out, "status") == "completed" {
		t.Fatal("revoked trust committed", out)
	}
	var imported *time.Time
	s.DB.QueryRow(ctx, `SELECT imported_at FROM knowledge_distribution_imports WHERE session_id=$1`, id).Scan(&imported)
	if imported != nil {
		t.Fatal("revoked trust wrote receipt")
	}
}
