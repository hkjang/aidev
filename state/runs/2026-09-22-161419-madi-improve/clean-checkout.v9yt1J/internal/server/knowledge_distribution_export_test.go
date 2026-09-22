package server

import (
	"bytes"
	"testing"
)

func TestPostgresDistributionExportPublishedSourceEncryptionPolicyAndKey(t *testing.T) {
	s, c, ctx, _, wid := jobTestFixture(t)
	c.request("PUT", "/api/v1/admin/settings", map[string]any{"storage_path": t.TempDir()}, 200)
	settings := testJSONObject(t, c.request("GET", "/api/v1/admin/knowledge-distribution", nil, 200))
	instance := str(settings["policy"].(map[string]any), "instance_id")
	c.request("PUT", "/api/v1/admin/knowledge-distribution", map[string]any{"revision": 1, "enabled": true, "max_valid_days": 7, "consent": true}, 200)
	key := testJSONObject(t, c.request("POST", "/api/v1/admin/knowledge-distribution/keys", map[string]any{"kind": "signing", "label": "반출 서명키", "consent": true}, 201))
	if str(key, "source_instance") != instance || str(key, "source_key_id") != str(key, "id") {
		t.Fatal(key)
	}
	doc := testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "확인된 운영 정책", "markdown": "서명 보존 원본 SIGNED_SOURCE_SENTINEL\n", "visibility": "workspace"}, 200))
	did := str(doc, "id")
	attachment := testJSONObject(t, storageMultipart(t, c, "/api/v1/attachments?document_id="+did, "운영.txt", []byte("파일 근거 ATTACHMENT_SENTINEL"), "", 200))
	if str(attachment, "id") == "" {
		t.Fatal(attachment)
	}
	makeBase := func() string {
		out := testJSONObject(t, c.request("POST", "/api/v1/exports", map[string]any{"workspace_id": wid, "format": "markdown", "document_ids": []string{did}}, 200))
		drainJobs(t, s)
		return str(out, "id")
	}
	makeDistribution := func(base string) string {
		version := testJSONObject(t, c.request("GET", "/api/v1/documents/"+did, nil, 200))["version"]
		out := testJSONObject(t, c.request("POST", "/api/v1/knowledge/distribution/exports", map[string]any{"export_id": base, "signing_key_id": str(key, "id"), "receiver_instance": newID(), "valid_days": 7, "consent": true, "documents": []map[string]any{{"id": did, "version": version}}}, 202))
		drainJobs(t, s)
		return str(out, "id")
	}
	// Drafts can be exported for personal migration, but cannot be signed as
	// published cross-network knowledge.
	failed := makeDistribution(makeBase())
	var status string
	s.DB.QueryRow(ctx, `SELECT status FROM knowledge_distribution_exports WHERE id=$1`, failed).Scan(&status)
	if status != "failed" {
		t.Fatal("signed draft", status)
	}
	current := testJSONObject(t, c.request("GET", "/api/v1/documents/"+did, nil, 200))
	c.request("PUT", "/api/v1/documents/"+did, map[string]any{"version": current["version"], "status": "published"}, 200)
	base := makeBase()
	id := makeDistribution(base)
	if e := s.DB.QueryRow(ctx, `SELECT status FROM knowledge_distribution_exports WHERE id=$1`, id).Scan(&status); e != nil || status != "ready" {
		var failure string
		s.DB.QueryRow(ctx, `SELECT last_error FROM automation_jobs WHERE id=(SELECT job_id FROM knowledge_distribution_exports WHERE id=$1)`, id).Scan(&failure)
		t.Fatal(status, e, failure)
	}
	raw := c.request("GET", "/api/v1/knowledge/distribution/exports/"+id+"/download", nil, 200)
	m, manifest, sig, files, e := readDistributionArchive(ctx, raw)
	if e != nil {
		t.Fatal(e)
	}
	if len(m.Files) != 2 || len(files) != 2 || m.SourceInstance != instance || m.SourceWorkspace != wid || m.Files[0].Approval.Required {
		t.Fatal(m)
	}
	if e = verifyDistributionSignature(manifest, sig, str(key, "public_key")); e != nil {
		t.Fatal(e)
	}
	var artifact []byte
	var private, stored string
	if e = s.DB.QueryRow(ctx, `SELECT ciphertext FROM knowledge_distribution_artifacts WHERE export_id=$1`, id).Scan(&artifact); e != nil || bytes.Contains(artifact, []byte("SIGNED_SOURCE_SENTINEL")) || bytes.HasPrefix(artifact, []byte("PK")) {
		t.Fatal("unencrypted archive", e)
	}
	if e = s.DB.QueryRow(ctx, `SELECT private_ciphertext FROM knowledge_distribution_keys WHERE id=$1`, str(key, "id")).Scan(&private); e != nil || private == "" {
		t.Fatal(e)
	}
	if e = s.DB.QueryRow(ctx, `SELECT manifest_ciphertext FROM knowledge_distribution_exports WHERE id=$1`, id).Scan(&stored); e != nil || bytes.Contains([]byte(stored), []byte("확인된 운영 정책")) {
		t.Fatal("unencrypted manifest", e)
	}
	// Current protection policy revision changes invalidate already prepared
	// bytes, even if the document revision is unchanged.
	if _, e = s.DB.Exec(ctx, `UPDATE protection_settings SET revision=revision+1 WHERE id=1`); e != nil {
		t.Fatal(e)
	}
	c.request("GET", "/api/v1/knowledge/distribution/exports/"+id+"/download", nil, 409)
	id = makeDistribution(base)
	c.request("GET", "/api/v1/knowledge/distribution/exports/"+id+"/download", nil, 200)
	c.request("POST", "/api/v1/admin/knowledge-distribution/keys/"+str(key, "id")+"/revoke", map[string]any{"revision": 1, "confirmation": "REVOKE"}, 200)
	c.request("GET", "/api/v1/knowledge/distribution/exports/"+id+"/download", nil, 409)
	s.expireDistributions(ctx)
	var count int
	s.DB.QueryRow(ctx, `SELECT count(*) FROM knowledge_distribution_artifacts WHERE export_id=$1`, id).Scan(&count)
	if count != 0 {
		t.Fatal("revoked artifact retained")
	}
	// A published label created before the approval policy was enabled is not
	// an approval receipt under the newly enabled process.
	key = testJSONObject(t, c.request("POST", "/api/v1/admin/knowledge-distribution/keys", map[string]any{"kind": "signing", "label": "회전한 키", "consent": true}, 201))
	c.request("PUT", "/api/v1/admin/settings", map[string]any{"approval_enabled": true}, 200)
	id = makeDistribution(makeBase())
	s.DB.QueryRow(ctx, `SELECT status FROM knowledge_distribution_exports WHERE id=$1`, id).Scan(&status)
	if status != "failed" {
		t.Fatal("legacy published label bypassed approval", status)
	}
}
