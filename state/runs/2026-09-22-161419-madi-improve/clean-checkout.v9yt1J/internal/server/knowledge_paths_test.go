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

func learningFixture(t *testing.T) (*Server, *integrationTestClient, *integrationTestClient, string, string, string) {
	t.Helper()
	s, owner, reader, wid, _ := collaborationTestSetup(t)
	doc := testJSONObject(t, owner.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "운영자 지식 경로 근거", "markdown": "현재 운영 기준", "visibility": "workspace"}, 200))
	input := map[string]any{"workspace_id": wid, "title": "운영자 온보딩", "description": "자동 실행 없이 원문 확인과 직접 실습을 구분합니다", "role_labels": []string{"운영자"}, "visibility": "workspace", "confirm_shared": true, "steps": []any{
		map[string]any{"document_id": doc["id"], "kind": "read", "title": "원문 읽기"}, map[string]any{"document_id": doc["id"], "kind": "practice", "title": "직접 실습"}, map[string]any{"document_id": doc["id"], "kind": "review", "title": "독립 검토"}}}
	made := testJSONObject(t, owner.request("POST", "/api/v1/knowledge-paths", input, 200))
	return s, owner, reader, wid, str(doc, "id"), str(made, "id")
}
func learningView(t *testing.T, c *integrationTestClient, id string) map[string]any {
	t.Helper()
	return testJSONObject(t, c.request("GET", "/api/v1/knowledge-paths/"+id, nil, 200))
}
func learningStepAt(v map[string]any, i int) map[string]any {
	return v["steps"].([]any)[i].(map[string]any)
}
func learningConfirm(t *testing.T, c *integrationTestClient, id string, v map[string]any, i int, proof string, status int) map[string]any {
	t.Helper()
	step := learningStepAt(v, i)
	revision := float64(0)
	if g, ok := step["progress"].(map[string]any); ok {
		revision, _ = g["revision"].(float64)
	}
	raw := c.request("POST", "/api/v1/knowledge-paths/"+id+"/steps/"+str(step, "id")+"/confirm", map[string]any{"path_revision": v["path"].(map[string]any)["revision"], "document_version": step["document_version"], "revision": revision, "proof": proof, "confirmation": "CONFIRM"}, status)
	if status != 200 {
		return nil
	}
	return testJSONObject(t, raw)
}
func TestPostgresLearningPathsOrderedProgressPrivacyAndSourceRecheck(t *testing.T) {
	s, owner, reader, wid, did, id := learningFixture(t)
	v := learningView(t, reader, id)
	if number(v, "total", 0) != 2 || number(v, "omitted_reviews", 0) != 1 || number(v, "completed", -1) != 0 {
		t.Fatal(v)
	}
	reader.request("GET", "/api/v1/documents/"+did, nil, 200)
	if number(learningView(t, reader, id), "completed", -1) != 0 {
		t.Fatal("read implicitly completed")
	}
	learningConfirm(t, reader, id, v, 1, "실습 기록", 409)
	learningConfirm(t, reader, id, v, 0, "읽기에는 proof를 저장하지 않습니다", 200)
	learningConfirm(t, reader, id, v, 0, "", 409)
	v = learningView(t, reader, id)
	learningConfirm(t, reader, id, v, 1, "LEARNING_PRIVATE_SENTINEL 직접 실행 후 결과를 확인했습니다", 200)
	v = learningView(t, reader, id)
	if number(v, "completed", 0) != 2 {
		t.Fatal(v)
	}
	if bytes.Contains(owner.request("GET", "/api/v1/knowledge-paths/"+id, nil, 200), []byte("LEARNING_PRIVATE_SENTINEL")) {
		t.Fatal("other user's proof leaked")
	}
	var raw string
	if e := s.DB.QueryRow(t.Context(), `SELECT proof FROM knowledge_path_progress WHERE step_id=$1 AND user_id=$2`, str(learningStepAt(v, 1), "id"), str(testJSONObject(t, reader.request("GET", "/api/v1/auth/me", nil, 200)), "id")).Scan(&raw); e != nil || bytes.Contains([]byte(raw), []byte("LEARNING_PRIVATE_SENTINEL")) {
		t.Fatal("proof plaintext", e)
	}
	learningConfirm(t, reader, id, v, 2, "검토 요청", 409)
	owner.request("PUT", "/api/v1/documents/"+did, map[string]any{"version": 1, "markdown": "수정된 운영 기준"}, 200)
	v = learningView(t, reader, id)
	if number(v, "completed", -1) != 0 || !boolean(learningStepAt(v, 1), "needs_recheck") {
		t.Fatal(v)
	}
	learningConfirm(t, reader, id, v, 0, "", 200)
	v = learningView(t, reader, id)
	learningConfirm(t, reader, id, v, 1, "새 버전 재실습", 200)
	history := reader.request("GET", "/api/v1/knowledge-paths/"+id+"/history", nil, 200)
	if !bytes.Contains(history, []byte("LEARNING_PRIVATE_SENTINEL")) {
		t.Fatal("past encrypted proof lost", string(history))
	}
	owner.request("PUT", "/api/v1/documents/"+did, map[string]any{"version": 2, "visibility": "private"}, 200)
	v = learningView(t, reader, id)
	if boolean(learningStepAt(v, 0), "available") || str(learningStepAt(v, 0), "document_id") != "" {
		t.Fatal("inaccessible reference leaked", v)
	}
	history = reader.request("GET", "/api/v1/knowledge-paths/"+id+"/history", nil, 200)
	if bytes.Contains(history, []byte("LEARNING_PRIVATE_SENTINEL")) {
		t.Fatal("revoked source retained proof")
	}
	reader.request("POST", "/api/v1/knowledge-paths", map[string]any{"workspace_id": wid, "title": "권한 없는 생성", "visibility": "private", "steps": []any{map[string]any{"document_id": did, "kind": "read", "title": "읽기"}}}, 404)
}
func TestPostgresLearningPathsIndependentReviewExplicitPolicyAndStaleProof(t *testing.T) {
	s, owner, reader, wid, did, id := learningFixture(t)
	ownerID := str(testJSONObject(t, owner.request("GET", "/api/v1/auth/me", nil, 200)), "id")
	owner.request("PUT", "/api/v1/admin/settings", map[string]any{"approval_enabled": true}, 200)
	v := learningView(t, reader, id)
	if boolean(v, "review_policy_configured") || number(v, "total", 0) != 2 {
		t.Fatal("document fallback activated learning review", v)
	}
	owner.request("POST", "/api/v1/admin/approval/policies", map[string]any{"workspace_id": wid, "resource_kind": "learning_step", "name": "지식 실습 검토", "enabled": true, "stages": []any{map[string]any{"name": "독립 검토", "mode": "all", "gates": []any{map[string]any{"name": "담당 검토자", "kind": "user", "id": ownerID}}}}}, 200)
	v = learningView(t, reader, id)
	learningConfirm(t, reader, id, v, 0, "", 200)
	v = learningView(t, reader, id)
	learningConfirm(t, reader, id, v, 1, "REVIEW_PRIVATE_SENTINEL 실습 결과", 200)
	v = learningView(t, reader, id)
	review := learningConfirm(t, reader, id, v, 2, "독립 확인을 요청합니다", 200)
	rid, aid := str(review, "id"), str(review, "approval_id")
	var snap string
	s.DB.QueryRow(t.Context(), `SELECT snapshot::text FROM approval_requests WHERE id=$1`, aid).Scan(&snap)
	if bytes.Contains([]byte(snap), []byte("REVIEW_PRIVATE_SENTINEL")) {
		t.Fatal("proof duplicated in snapshot")
	}
	body := owner.request("GET", "/api/v1/learning-progress/"+rid+"/review", nil, 200)
	if !bytes.Contains(body, []byte("REVIEW_PRIVATE_SENTINEL")) {
		t.Fatal("review missing submitted proof", string(body))
	}
	request := testJSONObject(t, owner.request("GET", "/api/v1/approvals/requests/"+aid, nil, 200))
	decision := map[string]any{"action": "approve", "request_version": request["version"], "gate_index": 0, "comment": "실습 원문 확인"}
	reader.request("POST", "/api/v1/approvals/requests/"+aid+"/decisions", decision, 403)
	owner.request("POST", "/api/v1/approvals/requests/"+aid+"/decisions", decision, 200)
	v = learningView(t, reader, id)
	if number(v, "completed", 0) != 3 {
		t.Fatal(v)
	}
	var version int
	s.DB.QueryRow(t.Context(), `SELECT version FROM documents WHERE id=$1`, did).Scan(&version)
	if version != 1 {
		t.Fatal("learning approval mutated source")
	}
	learningConfirm(t, reader, id, v, 1, "NEW_NOT_SUBMITTED_PRIVATE 실습 결과 변경", 200)
	owner.request("GET", "/api/v1/learning-progress/"+rid+"/review", nil, 409)
	v = learningView(t, reader, id)
	if number(v, "completed", 0) != 2 {
		t.Fatal("earlier practice changed without review invalidation", v)
	}
	owner.request("PUT", "/api/v1/admin/settings", map[string]any{"approval_enabled": false}, 200)
	v = learningView(t, reader, id)
	if number(v, "total", 0) != 2 || number(v, "omitted_reviews", 0) != 1 {
		t.Fatal(v)
	}
}

func TestPostgresLearningPathsPIIKeyMetadataOnlyAndConfigCAS(t *testing.T) {
	s, owner, reader, wid, did, id := learningFixture(t)
	ctx := t.Context()
	v := learningView(t, reader, id)
	learningConfirm(t, reader, id, v, 0, "", 200)
	policy := defaultProtectionSettings()
	policy["enabled"], policy["mode"] = true, "mask"
	if _, e := s.DB.Exec(ctx, `UPDATE protection_settings SET data=$1`, jsonValue(policy)); e != nil {
		t.Fatal(e)
	}
	v = learningView(t, reader, id)
	g := learningConfirm(t, reader, id, v, 1, "확인자 learning@example.test · 직접 수행", 200)
	if strings.Contains(str(g, "proof"), "learning@example.test") || !boolean(g["protection"].(map[string]any), "changed") {
		t.Fatal("proof mask failed", g)
	}
	policy["mode"] = "block"
	s.DB.Exec(ctx, `UPDATE protection_settings SET data=$1`, jsonValue(policy))
	v = learningView(t, reader, id)
	learningConfirm(t, reader, id, v, 1, "block@example.test", 422)
	policy["mode"] = "warn"
	policy["enabled"] = false
	s.DB.Exec(ctx, `UPDATE protection_settings SET data=$1`, jsonValue(policy))
	v = learningView(t, reader, id)
	learningConfirm(t, reader, id, v, 1, "COOKIE_ONLY_PERSONAL_PROOF", 200)
	uid := str(testJSONObject(t, reader.request("GET", "/api/v1/auth/me", nil, 200)), "id")
	issued := testJSONObject(t, owner.request("POST", "/api/v1/keys", map[string]any{"user_id": uid, "name": "학습 조회", "workspace_id": wid, "scopes": []string{"document:read"}, "expires_in_days": 1}, 201))
	key := newIntegrationTestClient(t, owner.base)
	key.token = str(issued, "token")
	if raw := key.request("GET", "/api/v1/knowledge-paths/"+id, nil, 200); bytes.Contains(raw, []byte("COOKIE_ONLY_PERSONAL_PROOF")) {
		t.Fatal("key leaked private practice body")
	}
	key.request("GET", "/api/v1/knowledge-paths/"+id+"/history", nil, 403)
	key.request("POST", "/api/v1/knowledge-paths/"+id+"/steps/"+str(learningStepAt(v, 0), "id")+"/confirm", map[string]any{}, 403)
	currentView := learningView(t, owner, id)
	p := currentView["path"].(map[string]any)
	p["steps"] = currentView["steps"]
	p["description"] = "원문 변경과 다른 경로 개정"
	p["confirm_shared"] = true
	owner.request("PUT", "/api/v1/knowledge-paths/"+id, p, 200)
	owner.request("PUT", "/api/v1/knowledge-paths/"+id, p, 409)
	if number(learningView(t, reader, id), "completed", -1) != 0 {
		t.Fatal("path config revision did not invalidate current progress")
	}
	p["revision"] = 2
	p["steps"] = []any{map[string]any{"id": str(learningStepAt(currentView, 0), "id"), "document_id": did, "kind": "practice", "title": "기존 유형 위조"}}
	owner.request("PUT", "/api/v1/knowledge-paths/"+id, p, 400)
}

func TestPostgresLearningPathsSessionExpiresWhileFinalMembershipLocked(t *testing.T) {
	s, _, reader, _, _, id := learningFixture(t)
	ctx := t.Context()
	v := learningView(t, reader, id)
	step := learningStepAt(v, 0)
	uid := str(testJSONObject(t, reader.request("GET", "/api/v1/auth/me", nil, 200)), "id")
	block, e := s.DB.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer block.Rollback(ctx)
	if _, e = block.Exec(ctx, `SELECT user_id FROM workspace_members WHERE user_id=$1 FOR UPDATE`, uid); e != nil {
		t.Fatal(e)
	}
	var expires time.Time
	if e = s.DB.QueryRow(ctx, `UPDATE sessions SET expires_at=clock_timestamp()+interval '1500 milliseconds' WHERE user_id=$1 RETURNING expires_at`, uid).Scan(&expires); e != nil {
		t.Fatal(e)
	}
	data, _ := json.Marshal(map[string]any{"path_revision": 1, "document_version": 1, "revision": 0, "confirmation": "CONFIRM"})
	request, _ := http.NewRequest("POST", reader.base+"/api/v1/knowledge-paths/"+id+"/steps/"+str(step, "id")+"/confirm", bytes.NewReader(data))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Madi-Request", "1")
	type result struct {
		status int
		err    error
	}
	ch := make(chan result, 1)
	go func() {
		r, e := reader.client.Do(request)
		if e != nil {
			ch <- result{err: e}
			return
		}
		io.Copy(io.Discard, r.Body)
		r.Body.Close()
		ch <- result{status: r.StatusCode}
	}()
	waiting := false
	for time.Now().Before(expires) {
		if e = s.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE wait_event_type='Lock' AND query LIKE 'SELECT role FROM workspace_members%' AND pid<>pg_backend_pid())`).Scan(&waiting); e != nil {
			t.Fatal(e)
		}
		if waiting {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !waiting {
		t.Fatal("confirmation did not reach final credential lock before expiry")
	}
	if delay := time.Until(expires) + 30*time.Millisecond; delay > 0 {
		time.Sleep(delay)
	}
	if e = block.Rollback(ctx); e != nil {
		t.Fatal(e)
	}
	got := <-ch
	if got.err != nil || got.status != 403 {
		t.Fatal("expired session confirmation", got)
	}
	var n int
	if e = s.DB.QueryRow(ctx, `SELECT count(*) FROM knowledge_path_progress WHERE user_id=$1`, uid).Scan(&n); e != nil || n != 0 {
		t.Fatal("expired action retained a completion", n, e)
	}
}
