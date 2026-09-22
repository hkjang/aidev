package server

import (
	"strings"
	"testing"
)

func TestPostgresDistributionWholeBundleApproval(t *testing.T) {
	s, owner, reviewer, wid, _ := collaborationTestSetup(t)
	ctx := t.Context()
	uid := str(testJSONObject(t, reviewer.request("GET", "/api/v1/auth/me", nil, 200)), "id")
	owner.request("PUT", "/api/v1/admin/settings", map[string]any{"approval_enabled": true, "storage_path": t.TempDir()}, 200)
	for _, kind := range []string{"document", "knowledge_distribution"} {
		owner.request("POST", "/api/v1/admin/approval/policies", map[string]any{"name": "정확한 자료 검토 " + kind, "workspace_id": wid, "resource_kind": kind, "enabled": true, "stages": []any{map[string]any{"name": "독립 검토", "mode": "all", "gates": []any{map[string]any{"name": "검토자", "kind": "user", "id": uid}}}}}, 200)
	}
	owner.request("PUT", "/api/v1/admin/knowledge-distribution", map[string]any{"revision": 1, "enabled": true, "max_valid_days": 7, "consent": true}, 200)
	key := testJSONObject(t, owner.request("POST", "/api/v1/admin/knowledge-distribution/keys", map[string]any{"kind": "signing", "label": "배포 승인 검증키", "consent": true}, 201))
	doc := testJSONObject(t, owner.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "배포 승인 원문", "markdown": "BUNDLE_DOCUMENT_PRIVATE_SENTINEL", "visibility": "workspace"}, 200))
	did := str(doc, "id")
	storageMultipart(t, owner, "/api/v1/attachments?document_id="+did, "비밀이름.txt", []byte("BUNDLE_ATTACHMENT_PRIVATE_SENTINEL"), "", 200)
	owner.request("POST", "/api/v1/documents/"+did+"/approval", map[string]any{"action": "submit"}, 200)
	a := testJSONObject(t, owner.request("GET", "/api/v1/documents/"+did+"/approval", nil, 200))["request"].(map[string]any)
	reviewer.request("POST", "/api/v1/approvals/requests/"+str(a, "id")+"/decisions", map[string]any{"action": "approve", "request_version": a["version"]}, 200)
	doc = testJSONObject(t, owner.request("GET", "/api/v1/documents/"+did, nil, 200))
	base := str(testJSONObject(t, owner.request("POST", "/api/v1/exports", map[string]any{"workspace_id": wid, "format": "markdown", "document_ids": []string{did}}, 200)), "id")
	drainJobs(t, s)
	id := str(testJSONObject(t, owner.request("POST", "/api/v1/knowledge/distribution/exports", map[string]any{"export_id": base, "signing_key_id": str(key, "id"), "receiver_instance": newID(), "valid_days": 7, "consent": true, "documents": []map[string]any{{"id": did, "version": doc["version"]}}}, 202)), "id")
	drainJobs(t, s)
	var status, signature string
	if e := s.DB.QueryRow(ctx, `SELECT status,signature FROM knowledge_distribution_exports WHERE id=$1`, id).Scan(&status, &signature); e != nil || status != "awaiting_review" || signature != "" {
		var reason string
		s.DB.QueryRow(ctx, `SELECT last_error FROM automation_jobs WHERE id=(SELECT job_id FROM knowledge_distribution_exports WHERE id=$1)`, id).Scan(&reason)
		t.Fatal("document publication was incorrectly treated as full bundle approval", status, signature, e, reason)
	}
	path := "/api/v1/knowledge/distribution/exports/" + id
	owner.request("GET", path+"/download", nil, 409)
	view := testJSONObject(t, reviewer.request("GET", path+"/review", nil, 200))
	m := view["manifest"].(map[string]any)
	if len(m["files"].([]any)) != 2 || m["bundle_approval"] != nil {
		t.Fatal(view)
	}
	owner.request("POST", path+"/sign", map[string]any{"consent": true, "request_id": newID(), "request_version": 1}, 409)
	owner.request("POST", path+"/approval", map[string]any{"consent": true, "manifest_sha256": digest("wrong")}, 409)
	a = testJSONObject(t, owner.request("POST", path+"/approval", map[string]any{"consent": true, "manifest_sha256": view["manifest_sha256"]}, 201))
	aid := str(a, "id")
	owner.request("POST", path+"/approval", map[string]any{"consent": true, "manifest_sha256": view["manifest_sha256"]}, 409)
	var snapshot string
	if e := s.DB.QueryRow(ctx, `SELECT snapshot::text FROM approval_requests WHERE id=$1`, aid).Scan(&snapshot); e != nil || strings.Contains(snapshot, "비밀이름") || strings.Contains(snapshot, "PRIVATE_SENTINEL") {
		t.Fatal("plain data in shared review snapshot", e)
	}
	owner.request("POST", "/api/v1/approvals/requests/"+aid+"/decisions", map[string]any{"action": "approve", "request_version": a["version"]}, 403)
	a = testJSONObject(t, reviewer.request("POST", "/api/v1/approvals/requests/"+aid+"/decisions", map[string]any{"action": "approve", "request_version": a["version"]}, 200))
	if str(a, "status") != "approved" {
		t.Fatal(a)
	}
	var artifacts int
	s.DB.QueryRow(ctx, `SELECT count(*) FROM knowledge_distribution_artifacts WHERE export_id=$1`, id).Scan(&artifacts)
	if artifacts != 0 {
		t.Fatal("approval automatically signed/transmitted the bundle")
	}
	owner.request("POST", path+"/sign", map[string]any{"consent": true, "request_id": aid, "request_version": 1}, 409)
	reviewer.request("POST", path+"/sign", map[string]any{"consent": true, "request_id": aid, "request_version": a["version"]}, 404)
	owner.request("POST", path+"/sign", map[string]any{"consent": true, "request_id": aid, "request_version": a["version"]}, 202)
	drainJobs(t, s)
	raw := owner.request("GET", path+"/download", nil, 200)
	manifest, body, sig, _, e := readDistributionArchive(ctx, raw)
	if e != nil || manifest.BundleApproval == nil || manifest.BundleApproval.RequestID != aid {
		t.Fatal(manifest, e)
	}
	if e = verifyDistributionSignature(body, sig, str(key, "public_key")); e != nil {
		t.Fatal(e)
	}
	if e = validateReceivedDistributionApproval(manifest); e != nil {
		t.Fatal(e)
	}
	manifest.BundleApproval = nil
	if e = validateReceivedDistributionApproval(manifest); e == nil {
		t.Fatal("a document-only receipt satisfied a received bundle approval")
	}
	// Attaching another file changes the reviewed bundle even when document
	// Markdown/version remains unchanged. No new bytes may inherit approval.
	storageMultipart(t, owner, "/api/v1/attachments?document_id="+did, "추가.txt", []byte("AFTER_REVIEW"), "", 200)
	owner.request("GET", path+"/download", nil, 409)
	reviewer.request("GET", path+"/review", nil, 409)
}
