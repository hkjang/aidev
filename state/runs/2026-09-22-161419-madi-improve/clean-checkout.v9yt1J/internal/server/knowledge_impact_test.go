package server

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"
)

func TestKnowledgeChangeClassificationDoesNotClaimSemanticCertainty(t *testing.T) {
	diff := boundedDocumentDiff("최대 32개\n실행 전에 검토하세요.\n", "최대 64개\n승인 이후 실행하세요.\n")
	got := classifyKnowledgeChange(diff, true, false)
	for _, want := range []string{"number", "procedure", "access"} {
		if !slices.Contains(got, want) {
			t.Fatal(got)
		}
	}
	if got = classifyKnowledgeChange(boundedDocumentDiff("용어", "설명"), false, false); !slices.Equal(got, []string{"wording_candidate"}) {
		t.Fatal(got)
	}
}

func TestPostgresKnowledgeImpactTypedDependenciesReviewCASAndACL(t *testing.T) {
	s, c, ctx, _, wid := jobTestFixture(t)
	create := func(title string) string {
		return str(testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": title, "markdown": "운영 기준 32개"}, 200)), "id")
	}
	source, target, indirect, hidden := create("운영 정책"), create("설치 Runbook"), create("서비스 절차"), create("PRIVATE_IMPACT_SENTINEL")
	relate := func(dependent, dependency, kind string) {
		c.request("POST", "/api/v1/documents/"+dependent+"/relations", map[string]any{"target_id": dependency, "type": kind, "expected_version": 1}, 200)
	}
	relate(target, source, "policy")
	relate(indirect, target, "execution")
	relate(hidden, source, "data")
	c.request("PUT", "/api/v1/documents/"+source, map[string]any{"version": 1, "markdown": "승인 후 운영 기준 64개"}, 200)
	get := func(depth int) map[string]any {
		return testJSONObject(t, c.request("GET", fmt.Sprintf("/api/v1/documents/%s/impact?from=1&depth=%d", source, depth), nil, 200))
	}
	first := get(1)
	if len(first["documents"].([]any)) != 2 || !strings.Contains(string(jsonValue(first["categories"])), "number") {
		t.Fatal(first)
	}
	second := get(2)
	if len(second["documents"].([]any)) != 3 {
		t.Fatal(second)
	}
	request := map[string]any{"source_version": 2, "target_id": target, "target_version": 1, "relation_type": "policy"}
	c.request("POST", "/api/v1/documents/"+source+"/impact-reviews", request, 403)
	c.request("PUT", "/api/v1/admin/settings", map[string]any{"approval_enabled": true}, 200)
	review := str(testJSONObject(t, c.request("POST", "/api/v1/documents/"+source+"/impact-reviews", request, 201)), "id")
	c.request("POST", "/api/v1/documents/"+source+"/impact-reviews", request, 409)
	c.request("PUT", "/api/v1/knowledge/impact-reviews/"+review, map[string]any{"revision": 1, "target_version": 1, "status": "needs_change", "note": "검토 근거 SECRET_REVIEW_NOTE"}, 200)
	c.request("PUT", "/api/v1/knowledge/impact-reviews/"+review, map[string]any{"revision": 1, "target_version": 1, "status": "done", "note": "오래된 화면"}, 409)
	var ciphertext string
	if err := s.DB.QueryRow(ctx, `SELECT note_ciphertext FROM knowledge_impact_review_events WHERE review_id=$1`, review).Scan(&ciphertext); err != nil || strings.Contains(ciphertext, "SECRET_REVIEW_NOTE") {
		t.Fatal(err, ciphertext)
	}
	if !strings.Contains(string(c.request("GET", "/api/v1/knowledge/impact-reviews/"+review+"/history", nil, 200)), "SECRET_REVIEW_NOTE") {
		t.Fatal("missing encrypted history")
	}
	c.request("PUT", "/api/v1/documents/"+target, map[string]any{"version": 1, "markdown": "64개로 수정 완료"}, 200)
	c.request("PUT", "/api/v1/knowledge/impact-reviews/"+review, map[string]any{"revision": 2, "target_version": 1, "status": "done", "note": "오래된 대상"}, 409)
	c.request("PUT", "/api/v1/knowledge/impact-reviews/"+review, map[string]any{"revision": 2, "target_version": 2, "status": "done", "note": "변경 원문과 수정 결과 확인"}, 200)
	c.request("PUT", "/api/v1/documents/"+source, map[string]any{"version": 2, "markdown": "기준 128개"}, 200)
	c.request("PUT", "/api/v1/knowledge/impact-reviews/"+review, map[string]any{"revision": 3, "target_version": 2, "status": "no_impact", "note": "새 원문은 재분석해야 함"}, 409)
	if !strings.Contains(string(c.request("GET", "/api/v1/knowledge/impact-reviews?workspace_id="+wid, nil, 200)), `"stale":true`) {
		t.Fatal("old review not marked stale")
	}
	// A read-only scoped agent can inspect current allowed results, but not write.
	key := testJSONObject(t, c.request("POST", "/api/v1/keys", map[string]any{"name": "영향 조회", "workspace_id": wid, "scopes": []string{"document:read"}}, 201))
	c.token = str(key, "token")
	get(1)
	c.request("PUT", "/api/v1/knowledge/impact-reviews/"+review, map[string]any{"revision": 3, "target_version": 2, "status": "done", "note": "권한 없음"}, 403)
	c.token = ""
	// Remove current access without deleting its references; existence must vanish.
	other := newID()
	if _, err := s.DB.Exec(ctx, `WITH inserted AS(INSERT INTO users(id,email,name,password_hash,role) VALUES($1,'impact-owner@example.test','비공개 소유자','','editor') RETURNING id), member AS(INSERT INTO workspace_members(workspace_id,user_id,role) SELECT $2,id,'editor' FROM inserted) UPDATE documents SET owner_id=$1,visibility='private' WHERE id=$3`, other, wid, hidden); err != nil {
		t.Fatal(err)
	}
	value := get(2)
	if strings.Contains(string(jsonValue(value)), "PRIVATE_IMPACT_SENTINEL") || strings.Contains(string(jsonValue(value)), hidden) {
		t.Fatal("private dependency leaked")
	}
	if _, err := s.DB.Exec(ctx, `UPDATE documents SET owner_id=$1,visibility='private' WHERE id=$2`, other, target); err != nil {
		t.Fatal(err)
	}
	value = get(2)
	if len(value["documents"].([]any)) != 0 {
		t.Fatal("traversed inaccessible intermediate", value)
	}
	var list []any
	if json.Unmarshal(c.request("GET", "/api/v1/knowledge/impact-reviews?workspace_id="+wid, nil, 200), &list) != nil || len(list) != 0 {
		t.Fatal("review existence leaked", list)
	}
	c.request("GET", "/api/v1/knowledge/impact-reviews/"+review+"/history", nil, 404)
}
