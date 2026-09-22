package server

import "testing"

func TestPostgresDocumentTrashUndoCAS(t *testing.T) {
	s, ts := integrationTestServer(t)
	c := newIntegrationTestClient(t, ts.URL)
	c.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200)
	w := testJSONObject(t, c.request("POST", "/api/v1/workspaces", map[string]any{"name": "휴지통 실행 취소"}, 200))
	md := "---\ntags: [보존]\n---\n# 정확한 원문\n\n내용 😀\n"
	d := testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": w["id"], "title": "삭제와 복원", "markdown": md, "visibility": "private"}, 200))
	id := str(d, "id")
	path := "/api/v1/documents/" + id
	for _, bad := range []any{nil, 0, -1, 1.5, "1"} {
		c.request("DELETE", path, map[string]any{"expected_version": bad}, 400)
	}
	c.request("DELETE", path, map[string]any{"expected_version": 9}, 409)
	deleted := testJSONObject(t, c.request("DELETE", path, map[string]any{"expected_version": 1}, 200))
	if number(deleted, "version", 0) != 2 || !boolean(deleted, "changed") {
		t.Fatal("delete must return new version", deleted)
	}
	c.request("POST", path+"/restore", map[string]any{"expected_version": 1}, 409)
	restored := testJSONObject(t, c.request("POST", path+"/restore", map[string]any{"expected_version": 2}, 200))
	if number(restored, "version", 0) != 3 || str(restored, "markdown") != md || str(restored, "visibility") != "private" || restored["deleted_at"] != nil {
		t.Fatal("restore lost canonical data", restored)
	}
	c.request("POST", path+"/restore", map[string]any{"expected_version": 2}, 409)
	idempotent := testJSONObject(t, c.request("POST", path+"/restore", map[string]any{"expected_version": 3}, 200))
	if number(idempotent, "version", 0) != 3 {
		t.Fatal("active restore created extra version")
	}
	c.request("DELETE", path, map[string]any{"expected_version": 3}, 200)
	c.request("POST", path+"/restore", map[string]any{"expected_version": 2}, 409)
	// Legacy callers retain their latest-state behavior, but every state change
	// now has a new immutable version; UI callers always supply expected_version.
	legacy := testJSONObject(t, c.request("POST", path+"/restore", nil, 200))
	if number(legacy, "version", 0) != 5 {
		t.Fatal("legacy restore version", legacy)
	}
	legacy = testJSONObject(t, c.request("DELETE", path, nil, 200))
	if number(legacy, "version", 0) != 6 {
		t.Fatal("legacy delete version", legacy)
	}
	legacy = testJSONObject(t, c.request("DELETE", path, nil, 200))
	if number(legacy, "version", 0) != 6 || boolean(legacy, "changed") {
		t.Fatal("duplicate delete changed canonical version")
	}
	var count, distinct int
	if e := s.DB.QueryRow(t.Context(), "SELECT count(*),count(DISTINCT markdown) FROM document_versions WHERE document_id=$1", id).Scan(&count, &distinct); e != nil || count != 6 || distinct != 1 {
		t.Fatal("immutable snapshots mismatch", count, distinct, e)
	}
}
