package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestPostgresEvidenceActualResponseSnapshotsCurrentACLAndReviews(t *testing.T) {
	s, ts := integrationTestServer(t)
	admin := newIntegrationTestClient(t, ts.URL)
	admin.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200)
	wid := str(testJSONObject(t, admin.request("POST", "/api/v1/workspaces", map[string]any{"name": "근거 경계"}, 200)), "id")
	user := testJSONObject(t, admin.request("POST", "/api/v1/admin/users", map[string]any{"email": "evidence@example.test", "name": "근거 담당자", "password": "Evidence-Password-2026!", "role": "editor"}, 200))
	admin.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": user["email"], "role": "editor"}, 200)
	member := newIntegrationTestClient(t, ts.URL)
	member.request("POST", "/api/v1/auth/login", map[string]any{"email": user["email"], "password": "Evidence-Password-2026!"}, 200)
	doc := testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "GPU 정책", "markdown": "# GPU 정책\n\nEVIDENCE_ORIGINAL_SENTINEL 동시 작업은 2개입니다.\n"}, 200))
	did := str(doc, "id")
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var in map[string]any
		if json.NewDecoder(r.Body).Decode(&in) != nil || !boolean(in, "stream") {
			w.WriteHeader(400)
			return
		}
		if !strings.Contains(string(jsonValue(in)), "EVIDENCE_ORIGINAL_SENTINEL") {
			t.Error("archived source was not actually sent to provider")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"동시 작업은 2개입니다 [1].\"}}]}\n\ndata: [DONE]\n\n")
	}))
	defer provider.Close()
	admin.request("PUT", "/api/v1/admin/settings", map[string]any{"ai_enabled": true, "ai_base_url": provider.URL + "/v1", "ai_model": "evidence-model"}, 200)
	ticket := aiHistoryTestTicket(t, member, wid, did)
	member.request("POST", "/api/v1/ai/evidence", map[string]any{"ticket": ticket}, 400)
	member.request("POST", "/api/v1/ai/evidence", map[string]any{"ticket": ticket, "consent": true}, 403)
	admin.request("PUT", "/api/v1/admin/evidence-policy", map[string]any{"enabled": true, "retention_days": 90, "version": 1}, 200)
	admin.request("POST", "/api/v1/ai/evidence", map[string]any{"ticket": ticket, "consent": true}, 400)
	saved := testJSONObject(t, member.request("POST", "/api/v1/ai/evidence", map[string]any{"ticket": ticket, "consent": true}, 201))
	id := str(saved, "id")
	replayed := testJSONObject(t, member.request("POST", "/api/v1/ai/evidence", map[string]any{"ticket": ticket, "consent": true}, 200))
	if str(replayed, "id") != id || !boolean(replayed, "replayed") {
		t.Fatal(replayed)
	}
	var cipher string
	var count int
	if e := s.DB.QueryRow(t.Context(), `SELECT ciphertext FROM knowledge_evidence WHERE id=$1`, id).Scan(&cipher); e != nil || strings.Contains(cipher, "EVIDENCE_ORIGINAL_SENTINEL") {
		t.Fatal("plaintext at rest", e)
	}
	if e := s.DB.QueryRow(t.Context(), `SELECT count(*) FROM ai_conversations`).Scan(&count); e != nil || count != 0 {
		t.Fatal("evidence implicitly persisted personal conversation", e, count)
	}
	record := testJSONObject(t, member.request("GET", "/api/v1/ai/evidence/"+id, nil, 200))
	if str(record, "model") != "evidence-model" || str(record, "review_status") != "unreviewed" {
		t.Fatal(record)
	}
	quote := record["sources"].([]any)[0].(map[string]any)
	if str(quote, "integrity") != "match" || str(quote, "freshness") != "current" || !strings.Contains(str(quote, "text"), "EVIDENCE_ORIGINAL_SENTINEL") {
		t.Fatal(quote)
	}
	admin.request("GET", "/api/v1/ai/evidence/"+id, nil, 404)
	if list := admin.request("GET", "/api/v1/ai/evidence?workspace_id="+wid, nil, 200); bytes.Contains(list, []byte(id)) {
		t.Fatal("admin private snapshot bypass")
	}
	member.request("POST", "/api/v1/ai/evidence/"+id+"/reviews", map[string]any{"version": 1, "status": "insufficient", "note": "범위 조건 추가 확인"}, 200)
	member.request("POST", "/api/v1/ai/evidence/"+id+"/reviews", map[string]any{"version": 1, "status": "supported"}, 409)
	admin.request("PUT", "/api/v1/documents/"+did, map[string]any{"version": 1, "markdown": "# GPU 정책\n\n동시 작업을 4개로 변경했습니다.\n"}, 200)
	record = testJSONObject(t, member.request("GET", "/api/v1/ai/evidence/"+id, nil, 200))
	quote = record["sources"].([]any)[0].(map[string]any)
	if str(quote, "freshness") != "changed" || str(quote, "integrity") != "match" || !strings.Contains(str(quote, "text"), "EVIDENCE_ORIGINAL_SENTINEL") || str(record, "review_status") != "insufficient" {
		t.Fatal("staleness/integrity/claim review collapsed", record)
	}
	protection := defaultProtectionSettings()
	protection["enabled"], protection["mode"], protection["custom_terms"] = true, "mask", []string{"EVIDENCE_ORIGINAL_SENTINEL"}
	if _, err := s.DB.Exec(t.Context(), `UPDATE protection_settings SET data=$1`, jsonValue(protection)); err != nil {
		t.Fatal(err)
	}
	member.request("GET", "/api/v1/ai/evidence/"+id, nil, 409)
	protection["enabled"] = false
	if _, err := s.DB.Exec(t.Context(), `UPDATE protection_settings SET data=$1`, jsonValue(protection)); err != nil {
		t.Fatal(err)
	}
	member.request("GET", "/api/v1/ai/evidence/"+id, nil, 200)
	member.request("POST", "/api/v1/ai/evidence", map[string]any{"ticket": ticket, "consent": true}, 409)
	key := testJSONObject(t, member.request("POST", "/api/v1/keys", map[string]any{"name": "근거 키 제한", "workspace_id": wid, "scopes": []string{"ai:execute", "document:read"}}, 201))
	member.token = str(key, "token")
	member.request("GET", "/api/v1/ai/evidence/"+id, nil, 404)
	member.token = ""
	admin.request("PUT", "/api/v1/documents/"+did, map[string]any{"version": 2, "visibility": "private"}, 200)
	member.request("GET", "/api/v1/ai/evidence/"+id, nil, 404)
	if list := member.request("GET", "/api/v1/ai/evidence?workspace_id="+wid, nil, 200); bytes.Contains(list, []byte(id)) {
		t.Fatal("revoked source leaked evidence existence")
	}
	member.request("DELETE", "/api/v1/ai/evidence/"+id, map[string]any{"version": 2, "confirmation": "DELETE"}, 200)
}

func TestPostgresEvidenceRetentionHoldPolicyAndSourceDeletion(t *testing.T) {
	s, c, ctx, p, wid := jobTestFixture(t)
	c.request("PUT", "/api/v1/admin/evidence-policy", map[string]any{"enabled": true, "retention_days": 90, "version": 1}, 200)
	doc := testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "보존 의무", "markdown": "당시 보존 근거"}, 200))
	did := str(doc, "id")
	cfg, err := s.effectiveSettings(ctx, wid)
	if err != nil {
		t.Fatal(err)
	}
	src := sourceFromChunk(did, "보존 의무", 1, ragChunk{Start: 0, End: len("당시 보존 근거"), StartLine: 1, EndLine: 1, Content: "당시 보존 근거", Hash: digest("당시 보존 근거")})
	ticket, err := s.sealAIHistory(p, wid, "질문", "답변", "ask", []aiSource{src}, cfg)
	if err != nil {
		t.Fatal(err)
	}
	id := str(testJSONObject(t, c.request("POST", "/api/v1/ai/evidence", map[string]any{"ticket": ticket, "consent": true}, 201)), "id")
	if _, err = s.DB.Exec(ctx, `INSERT INTO knowledge_document_meta(document_id,legal_hold) VALUES($1,true) ON CONFLICT(document_id) DO UPDATE SET legal_hold=true`, did); err != nil {
		t.Fatal(err)
	}
	c.request("DELETE", "/api/v1/ai/evidence/"+id, map[string]any{"version": 1, "confirmation": "DELETE"}, 409)
	if _, err = s.DB.Exec(ctx, `UPDATE knowledge_evidence SET created_at=now()-interval '2 days' WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}
	c.request("PUT", "/api/v1/admin/evidence-policy", map[string]any{"enabled": true, "retention_days": 1, "version": 2}, 200)
	c.request("GET", "/api/v1/ai/evidence/"+id, nil, 404)
	if err = s.purgeEvidence(ctx); err != nil {
		t.Fatal(err)
	}
	var count int
	if err = s.DB.QueryRow(ctx, `SELECT count(*) FROM knowledge_evidence WHERE id=$1`, id).Scan(&count); err != nil || count != 1 {
		t.Fatal("hold discarded encrypted snapshot", count, err)
	}
	c.request("PUT", "/api/v1/admin/evidence-policy", map[string]any{"enabled": true, "retention_days": 3650, "version": 3}, 200)
	c.request("GET", "/api/v1/ai/evidence/"+id, nil, 404)
	if _, err = s.DB.Exec(ctx, `UPDATE knowledge_document_meta SET legal_hold=false WHERE document_id=$1`, did); err != nil {
		t.Fatal(err)
	}
	if err = s.purgeEvidence(ctx); err != nil {
		t.Fatal(err)
	}
	if err = s.DB.QueryRow(ctx, `SELECT count(*) FROM knowledge_evidence WHERE id=$1`, id).Scan(&count); err != nil || count != 0 {
		t.Fatal("expired snapshot not purged", count, err)
	}
	// The snapshot remains private when its last source disappears. A JSON
	// reference must not become an empty/all-allowed FK collection on deletion.
	ticket, err = s.sealAIHistory(p, wid, "다른 질문", "다른 답변", "ask", []aiSource{src}, cfg)
	if err != nil {
		t.Fatal(err)
	}
	id = str(testJSONObject(t, c.request("POST", "/api/v1/ai/evidence", map[string]any{"ticket": ticket, "consent": true}, 201)), "id")
	if _, err = s.DB.Exec(ctx, `DELETE FROM documents WHERE id=$1`, did); err != nil {
		t.Fatal(err)
	}
	c.request("GET", "/api/v1/ai/evidence/"+id, nil, 404)
	// TTL is checked in the authorization predicate, not only by an hourly job.
	if _, err = s.DB.Exec(ctx, `UPDATE knowledge_evidence SET expires_at=$2 WHERE id=$1`, id, time.Now().Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	c.request("GET", "/api/v1/ai/evidence/"+id, nil, 404)
}
