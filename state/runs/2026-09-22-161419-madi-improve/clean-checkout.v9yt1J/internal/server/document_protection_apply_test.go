package server

import (
	"strings"
	"testing"
)

func TestPostgresDocumentProtectionCanonicalAPITransactions(t *testing.T) {
	s, admin, ctx, _, wid := jobTestFixture(t)
	legacy := testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "old-owner@example.test"}, 200))
	policy := defaultProtectionSettings()
	policy["enabled"], policy["mode"] = true, "mask"
	setPolicy := func() {
		t.Helper()
		if _, e := s.DB.Exec(ctx, "UPDATE protection_settings SET data=$1,revision=revision+1", jsonValue(policy)); e != nil {
			t.Fatal(e)
		}
	}
	setPolicy()
	doc := testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{
		"workspace_id": wid, "title": "문의 owner@example.test",
		"markdown":       "---\ntags: [tag@example.test]\naliases: [alias@example.test]\n---\n# 연락처\n\nowner@example.test\n",
		"block_metadata": map[string]any{"hidden": "owner@example.test"},
	}, 200))
	id := str(doc, "id")
	if strings.Contains(string(jsonValue(doc)), "@example.test") || !boolean(doc["protection"].(map[string]any), "changed") || len(doc["block_metadata"].(map[string]any)) != 0 {
		t.Fatal("response retained original or source-bound metadata", doc)
	}
	fm, e := parseFrontMatter(str(doc, "markdown"))
	if e != nil || len(listStrings(fm["tags"])) != 1 || len(listStrings(fm["aliases"])) != 1 {
		t.Fatal("mask corrupted YAML structure", e, fm)
	}
	doc = testJSONObject(t, admin.request("PUT", "/api/v1/documents/"+id, map[string]any{
		"version": 1, "markdown": "# 정제된 새 본문\n", "tags": []string{"new-tag@example.test"}, "aliases": []string{},
		"block_metadata": map[string]any{"blocks": []any{map[string]any{"id": "block-safe", "text": "meta@example.test"}}},
	}, 200))
	if number(doc, "version", 0) != 2 || strings.Contains(string(jsonValue(doc)), "@example.test") || !boolean(doc["protection"].(map[string]any), "changed") {
		t.Fatal("metadata-only transform not returned", doc)
	}
	var retained bool
	for _, query := range []string{
		"SELECT EXISTS(SELECT 1 FROM document_versions WHERE document_id=$1 AND (title||markdown||tags::text||block_metadata::text) LIKE '%@example.test%')",
		"SELECT EXISTS(SELECT 1 FROM automation_events WHERE resource_id=$1 AND payload::text LIKE '%@example.test%')",
		"SELECT EXISTS(SELECT 1 FROM protection_events WHERE document_id=$1 AND findings::text LIKE '%@example.test%')",
	} {
		if e = s.DB.QueryRow(ctx, query, id).Scan(&retained); e != nil || retained {
			t.Fatal("unmasked derived or audit data", e, query)
		}
	}
	policy["mode"] = "block"
	setPolicy()
	admin.request("PUT", "/api/v1/documents/"+id, map[string]any{"version": 2, "markdown": "blocked@example.test"}, 422)
	admin.request("PUT", "/api/v1/documents/"+id, map[string]any{"version": 1, "markdown": "blocked@example.test"}, 409)
	admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "blocked@example.test"}, 422)
	var version, versions, events int
	if e = s.DB.QueryRow(ctx, "SELECT version,(SELECT count(*) FROM document_versions WHERE document_id=$1),(SELECT count(*) FROM automation_events WHERE resource_id=$1::text) FROM documents WHERE id=$1", id).Scan(&version, &versions, &events); e != nil || version != 2 || versions != 2 || events != 2 {
		t.Fatal("blocked or conflicting save was not atomic", e, version, versions, events)
	}
	// Renaming an old sensitive title must not silently re-add it as an alias.
	cleaned := testJSONObject(t, admin.request("PUT", "/api/v1/documents/"+str(legacy, "id"), map[string]any{"version": 1, "title": "정리된 제목", "aliases": []string{}}, 200))
	if strings.Contains(string(jsonValue(cleaned)), "old-owner@example.test") {
		t.Fatal("strict cleanup reintroduced old title", cleaned)
	}
}
