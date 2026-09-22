package server

import (
	"errors"
	"strings"
	"testing"
)

func TestPostgresInformationProtectionCanonicalMetadataAndBackstop(t *testing.T) {
	s, admin, ctx, p, wid := jobTestFixture(t)
	if e := s.migrateInformationProtection(ctx); e != nil {
		t.Fatal(e)
	}
	doc := testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "보호 정책 검증"}, 200))
	id := str(doc, "id")
	policy := defaultProtectionSettings()
	policy["enabled"] = true
	policy["mode"] = "mask"
	policy["custom_terms"] = []string{"비밀프로젝트"}
	setPolicy := func() {
		if _, e := s.DB.Exec(ctx, "UPDATE protection_settings SET data=$1", jsonValue(policy)); e != nil {
			t.Fatal(e)
		}
	}
	setPolicy()
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback(ctx)
	raw := "010-1234-5678 900101-1234567 4111 1111 1111 1111 123-456-789012 비밀프로젝트"
	cleaned, e := s.ProtectDocumentTx(ctx, tx, p, id, wid, "작성자 owner@example.test", raw)
	if e != nil {
		t.Fatal(e)
	}
	if !cleaned.Changed || len(cleaned.Findings) < 5 || strings.Contains(cleaned.Title, "owner@example.test") || strings.Contains(cleaned.Markdown, "010-1234") || strings.Contains(cleaned.Markdown, "비밀프로젝트") {
		t.Fatalf("PII canonical mask failed %+v", cleaned)
	}
	metadata, e := s.ProtectDocumentMetadataTx(ctx, tx, p, id, wid, map[string]any{"tags": []string{"owner@example.test"}, "aliases": []string{"010-1234-5678"}, "block_metadata": map[string]any{"blocks": []any{map[string]any{"id": "block-safe", "text": "비밀프로젝트"}}}})
	if e != nil || !metadata.Changed {
		t.Fatal("metadata mask", e)
	}
	value := metadata.Value.(map[string]any)
	if strings.Contains(string(jsonValue(value)), "owner@example.test") || strings.Contains(string(jsonValue(value)), "비밀프로젝트") {
		t.Fatal("masked metadata retained original")
	}
	if _, e = tx.Exec(ctx, "UPDATE documents SET title=$2,markdown=$3,tags=$4,aliases=$5,block_metadata=$6,version=version+1 WHERE id=$1", id, cleaned.Title, cleaned.Markdown, jsonValue(value["tags"]), jsonValue(value["aliases"]), jsonValue(value["block_metadata"])); e != nil {
		t.Fatal(e)
	}
	if _, e = tx.Exec(ctx, "INSERT INTO document_versions(document_id,version,title,markdown,tags,user_id,block_metadata) SELECT id,version,title,markdown,tags,owner_id,block_metadata FROM documents WHERE id=$1", id); e != nil {
		t.Fatal(e)
	}
	if e = tx.Commit(ctx); e != nil {
		t.Fatal(e)
	}
	for _, q := range []string{"UPDATE documents SET markdown='raw@example.test' WHERE id=$1", "UPDATE documents SET tags='[\"raw@example.test\"]' WHERE id=$1", "UPDATE documents SET block_metadata='{\"value\":\"010-1234-5678\"}' WHERE id=$1"} {
		if _, e = s.DB.Exec(ctx, q, id); e == nil || !strings.Contains(e.Error(), "MADI_PROTECTION_") {
			t.Fatal("unsanitized canonical path bypass", e)
		}
	}
	tx, e = s.DB.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	_, e = s.ProtectDocumentMetadataTx(ctx, tx, p, id, wid, map[string]any{"raw@example.test": "value"})
	var blocked ProtectionError
	if !errors.As(e, &blocked) || blocked.Code != "mask_required" {
		t.Fatal("metadata key mutation must fail closed", e)
	}
	tx.Rollback(ctx)
	tx, e = s.DB.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	_, e = s.ProtectCollaborationTx(ctx, tx, p, id, wid, "clean", "clean projected markdown")
	if !errors.As(e, &blocked) || blocked.Code != "collaboration_policy" {
		t.Fatal("hidden Yjs history not protected", e)
	}
	tx.Rollback(ctx)
	policy["mode"] = "block"
	setPolicy()
	tx, e = s.DB.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	_, e = s.ProtectDocumentTx(ctx, tx, p, id, wid, "clean", "owner@example.test")
	if !errors.As(e, &blocked) || blocked.Code != "blocked" {
		t.Fatal("block policy did not reject", e)
	}
	tx.Rollback(ctx)
	policy["mode"] = "audit"
	setPolicy()
	if _, e = s.DB.Exec(ctx, "UPDATE documents SET markdown='audit@example.test' WHERE id=$1", id); e != nil {
		t.Fatal(e)
	}
	var events string
	if e = s.DB.QueryRow(ctx, "SELECT coalesce(jsonb_agg(to_jsonb(e)),'[]')::text FROM protection_events e WHERE document_id=$1", id).Scan(&events); e != nil {
		t.Fatal(e)
	}
	if strings.Contains(events, "audit@example.test") || strings.Contains(events, "owner@example.test") || !strings.Contains(events, "document.detected") {
		t.Fatal("PII audit leaked payload or is absent", events)
	}
}

func TestPostgresInformationProtectionClassificationInheritance(t *testing.T) {
	s, admin, ctx, _, wid := jobTestFixture(t)
	if e := s.migrateInformationProtection(ctx); e != nil {
		t.Fatal(e)
	}
	parent := testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "기밀 상위 문서"}, 200))
	child := testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "분류 상속 문서", "parent_id": parent["id"]}, 200))
	if _, e := s.DB.Exec(ctx, "INSERT INTO knowledge_document_meta(document_id,classification) VALUES($1,'restricted'),($2,'public')", parent["id"], child["id"]); e != nil {
		t.Fatal(e)
	}
	var classification string
	if e := s.DB.QueryRow(ctx, "SELECT madi_effective_classification($1)", child["id"]).Scan(&classification); e != nil || classification != "restricted" {
		t.Fatal("ancestor classification bypass", classification, e)
	}
}
