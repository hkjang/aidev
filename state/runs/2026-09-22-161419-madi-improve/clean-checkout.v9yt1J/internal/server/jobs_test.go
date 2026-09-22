package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func TestWebhookSignature(t *testing.T) {
	now := time.Unix(1800000000, 0)
	stamp := "1800000000"
	id := newID()
	body := []byte(`{"type":"document.created"}`)
	secret := "a-long-independent-test-secret"
	signature := WebhookSignature(secret, stamp, id, body)
	if !VerifyWebhookSignature(secret, stamp, id, signature, body, now) {
		t.Fatal("valid signature rejected")
	}
	if VerifyWebhookSignature(secret, stamp, id, signature, []byte("changed"), now) || VerifyWebhookSignature(secret, stamp, id, signature, body, now.Add(6*time.Minute)) || VerifyWebhookSignature(secret, stamp, newID(), signature, body, now) {
		t.Fatal("tampering/replay accepted")
	}
	if _, e := webhookDial(context.Background(), "tcp", "169.254.169.254:80"); e == nil {
		t.Fatal("metadata network accepted")
	}
}
func jobTestFixture(t *testing.T) (*Server, *integrationTestClient, context.Context, *Principal, string) {
	s, server := integrationTestServer(t)
	client := newIntegrationTestClient(t, server.URL)
	client.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200)
	var uid, wid string
	if e := s.DB.QueryRow(context.Background(), "SELECT user_id::text,workspace_id::text FROM workspace_members WHERE role='owner' LIMIT 1").Scan(&uid, &wid); e != nil {
		t.Fatal(e)
	}
	p, e := s.workerPrincipal(context.Background(), uid, "", wid)
	if e != nil {
		t.Fatal(e)
	}
	return s, client, context.WithValue(context.Background(), principalKey, p), p, wid
}
func drainJobs(t *testing.T, s *Server) {
	t.Helper()
	ctx := context.Background()
	for index := 0; index < 100; index++ {
		_, _ = s.DB.Exec(ctx, "UPDATE automation_jobs SET run_after=now() WHERE status='pending'")
		j, e := s.claimJob(ctx)
		if errors.Is(e, pgx.ErrNoRows) {
			return
		}
		if e != nil {
			t.Fatal(e)
		}
		s.runJob(ctx, j)
	}
	t.Fatal("job drain exceeded limit (possible recursion)")
}
func TestJobsTransactionalOutboxWebhookRetry(t *testing.T) {
	s, client, ctx, p, wid := jobTestFixture(t)
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	eventID := newID()
	if e = s.enqueueEvent(ctx, tx, Event{ID: eventID, Type: "date.reached", WorkspaceID: wid, ActorID: p.ID, ResourceType: "schedule"}); e != nil {
		t.Fatal(e)
	}
	_ = tx.Rollback(ctx)
	var count int
	_ = s.DB.QueryRow(ctx, "SELECT count(*) FROM automation_jobs WHERE event_id=$1", eventID).Scan(&count)
	if count != 0 {
		t.Fatal("rolled back mutation left an outbox job")
	}
	secret := "actual-local-receiver-secret-2026"
	var calls atomic.Int32
	var mutex sync.Mutex
	deliveries := []string{}
	var invalid atomic.Bool
	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if !VerifyWebhookSignature(secret, r.Header.Get("X-Madi-Timestamp"), r.Header.Get("X-Madi-Delivery"), r.Header.Get("X-Madi-Signature"), body, time.Now()) {
			invalid.Store(true)
		}
		if strings.Contains(string(body), "DO-NOT-SEND-BODY") {
			invalid.Store(true)
		}
		mutex.Lock()
		deliveries = append(deliveries, r.Header.Get("X-Madi-Delivery"))
		mutex.Unlock()
		if calls.Add(1) == 1 {
			w.WriteHeader(503)
			return
		}
		w.WriteHeader(204)
	}))
	defer receiver.Close()
	hook := testJSONObject(t, client.request("POST", "/api/v1/webhooks", map[string]any{"workspace_id": wid, "name": "실제 HTTP", "url": receiver.URL, "secret": secret, "events": []string{"document.created"}, "enabled": true, "max_attempts": 3, "timeout_seconds": 2}, 200))
	if strings.Contains(string(jsonValue(hook)), secret) {
		t.Fatal("webhook secret returned")
	}
	var cipher string
	_ = s.DB.QueryRow(ctx, "SELECT secret_ciphertext FROM automation_webhooks WHERE id=$1", str(hook, "id")).Scan(&cipher)
	if cipher == secret || cipher == "" {
		t.Fatal("secret not encrypted")
	}
	doc := testJSONObject(t, client.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "재시도 문서", "markdown": "DO-NOT-SEND-BODY", "visibility": "workspace"}, 200))
	drainJobs(t, s)
	if calls.Load() != 2 || invalid.Load() {
		t.Fatalf("calls=%d invalid=%v", calls.Load(), invalid.Load())
	}
	if deliveries[0] != deliveries[1] {
		t.Fatal("retry did not retain delivery ID")
	}
	detail := testJSONObject(t, client.request("GET", "/api/v1/jobs/"+deliveries[0], nil, 200))
	if str(detail["job"].(map[string]any), "status") != "succeeded" || len(detail["attempts"].([]any)) != 2 {
		t.Fatalf("job history mismatch: %v", detail)
	}
	client.request("GET", "/api/v1/webhooks/"+str(hook, "id")+"/deliveries", nil, 200)
	// Public source was readable at enqueue, but account disable before execution
	// must prevent a subsequent real network call.
	client.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "권한 회수", "visibility": "workspace"}, 200)
	_, _ = s.DB.Exec(ctx, "UPDATE users SET disabled=true WHERE id=$1", p.ID)
	drainJobs(t, s)
	if calls.Load() != 2 {
		t.Fatal("disabled actor sent webhook")
	}
	_, _ = s.DB.Exec(ctx, "UPDATE users SET disabled=false WHERE id=$1", p.ID)
	client.request("PUT", "/api/v1/documents/"+str(doc, "id"), map[string]any{"version": 1, "title": "변경", "markdown": "- [ ] 확인"}, 200)
	client.request("PUT", "/api/v1/tasks", map[string]any{"document_id": str(doc, "id"), "version": 2, "line": 0, "done": true}, 200)
	_ = s.DB.QueryRow(ctx, "SELECT count(*) FROM automation_events WHERE type='task.completed' AND resource_id=$1", str(doc, "id")).Scan(&count)
	if count != 1 {
		t.Fatal("task completion missing from transaction")
	}
}
func TestJobsLeaseCancellationAndPanic(t *testing.T) {
	s, _, ctx, p, wid := jobTestFixture(t)
	id, e := s.EnqueueJob(ctx, nil, "test.noop", p.ID, wid, map[string]any{})
	if e != nil {
		t.Fatal(e)
	}
	var claimed atomic.Int32
	var mu sync.Mutex
	var first Job
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			j, e := s.claimJob(ctx)
			if e == nil {
				claimed.Add(1)
				mu.Lock()
				first = j
				mu.Unlock()
			} else if !errors.Is(e, pgx.ErrNoRows) {
				t.Error(e)
			}
		}()
	}
	wg.Wait()
	if claimed.Load() != 1 || first.ID != id {
		t.Fatal("SKIP LOCKED duplicate claim")
	}
	_, e = s.DB.Exec(ctx, "UPDATE automation_jobs SET lease_until=now()-interval '1 second' WHERE id=$1", id)
	if e != nil {
		t.Fatal(e)
	}
	recovered, e := s.claimJob(ctx)
	if e != nil || recovered.ID != id || recovered.LeaseID == first.LeaseID {
		t.Fatal("expired lease was not recovered")
	}
	s.RegisterJobHandler("test.noop", func(context.Context, Job) (map[string]any, error) { panic("secret-must-not-appear") })
	s.runJob(ctx, recovered)
	var status, message string
	_ = s.DB.QueryRow(ctx, "SELECT status,last_error FROM automation_jobs WHERE id=$1", id).Scan(&status, &message)
	if status != "failed" || strings.Contains(message, "secret") {
		t.Fatal("panic not safely contained")
	}
	started := make(chan struct{})
	s.RegisterJobHandler("test.cancel", func(ctx context.Context, _ Job) (map[string]any, error) {
		close(started)
		<-ctx.Done()
		return nil, ctx.Err()
	})
	id, e = s.EnqueueJob(ctx, nil, "test.cancel", p.ID, wid, nil)
	if e != nil {
		t.Fatal(e)
	}
	j, e := s.claimJob(ctx)
	if e != nil {
		t.Fatal(e)
	}
	finished := make(chan struct{})
	go func() { s.runJob(ctx, j); close(finished) }()
	<-started
	_, _ = s.DB.Exec(ctx, "UPDATE automation_jobs SET cancel_requested=true WHERE id=$1", id)
	select {
	case <-finished:
	case <-time.After(3 * time.Second):
		t.Fatal("running job did not cancel")
	}
	_ = s.DB.QueryRow(ctx, "SELECT status FROM automation_jobs WHERE id=$1", id).Scan(&status)
	if status != "cancelled" {
		t.Fatal("cancelled job status mismatch")
	}
}
func TestAutomationEffectsScheduleAndACL(t *testing.T) {
	s, client, ctx, p, wid := jobTestFixture(t)
	rule := testJSONObject(t, client.request("POST", "/api/v1/automations", map[string]any{"workspace_id": wid, "name": "한 번 생성", "enabled": true, "trigger": "document.created", "conditions": map[string]any{"tag": "run"}, "actions": []any{map[string]any{"type": "create_document", "title": "자동화 결과 {{title}}", "markdown": "예약 생성"}, map[string]any{"type": "notification", "title": "처리 완료"}}}, 200))
	source := testJSONObject(t, client.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "원본", "tags": []string{"run"}}, 200))
	drainJobs(t, s)
	var count int
	_ = s.DB.QueryRow(ctx, "SELECT count(*) FROM documents WHERE title='자동화 결과 원본' AND visibility='private'").Scan(&count)
	if count != 1 {
		t.Fatal("automation create failed")
	}
	var jobID string
	_ = s.DB.QueryRow(ctx, "SELECT id::text FROM automation_jobs WHERE kind='automation.execute' AND automation_id=$1", str(rule, "id")).Scan(&jobID)
	if jobID == "" {
		t.Fatal("automation job absent")
	}
	// Simulate retry after success persisted in the effect ledger but before the
	// outer job completion record survived. Already committed creates stay single.
	_, _ = s.DB.Exec(ctx, "UPDATE automation_jobs SET status='pending',run_after=now() WHERE id=$1", jobID)
	drainJobs(t, s)
	_ = s.DB.QueryRow(ctx, "SELECT count(*) FROM documents WHERE title='자동화 결과 원본'").Scan(&count)
	if count != 1 {
		t.Fatal("automation retry duplicated document")
	}
	// Another schedule rule must not run when only this rule becomes due.
	for i, due := range []time.Time{time.Now().Add(-time.Minute), time.Now().Add(time.Hour)} {
		client.request("POST", "/api/v1/automations", map[string]any{"workspace_id": wid, "name": []string{"도래", "미도래"}[i], "enabled": true, "trigger": "date.reached", "schedule_at": due, "actions": []any{map[string]any{"type": "notification", "title": []string{"예약 A", "예약 B"}[i]}}}, 200)
	}
	s.scheduleDueAutomations(ctx)
	drainJobs(t, s)
	_ = s.DB.QueryRow(ctx, "SELECT count(*) FROM notifications WHERE title IN ('예약 A','예약 B')").Scan(&count)
	if count != 1 {
		t.Fatal("schedule dispatched unrelated rule")
	}
	_, _ = s.DB.Exec(ctx, "UPDATE job_settings SET paused=true")
	_, _ = s.DB.Exec(ctx, "UPDATE automation_rules SET schedule_at=now()-interval '1 second' WHERE trigger='date.reached'")
	s.scheduleDueAutomations(ctx)
	_ = s.DB.QueryRow(ctx, "SELECT count(*) FROM automation_events WHERE type='date.reached'").Scan(&count)
	if count != 1 {
		t.Fatal("paused scheduler created event")
	}
	_, _ = s.DB.Exec(ctx, "UPDATE job_settings SET paused=false")
	// Rule owner can see source, but original actor is now a viewer: no writes.
	other := newID()
	_, e := s.DB.Exec(ctx, "INSERT INTO users(id,email,name,role) VALUES($1,'viewer-job@test.local','읽기 사용자','viewer')", other)
	if e != nil {
		t.Fatal(e)
	}
	_, _ = s.DB.Exec(ctx, "INSERT INTO workspace_members VALUES($1,$2,'viewer')", wid, other)
	tx, _ := s.DB.Begin(ctx)
	e = s.enqueueEvent(ctx, tx, Event{Type: "document.created", WorkspaceID: wid, ActorID: other, ResourceID: str(source, "id"), After: map[string]any{"title": "권한 초과", "tags": []string{"run"}}})
	if e != nil {
		t.Fatal(e)
	}
	_ = tx.Commit(ctx)
	drainJobs(t, s)
	_ = s.DB.QueryRow(ctx, "SELECT count(*) FROM documents WHERE title='자동화 결과 권한 초과'").Scan(&count)
	if count != 0 {
		t.Fatal("automation elevated viewer")
	}
	_ = p
}

func TestAutomationStreamingDatabaseAndApproval(t *testing.T) {
	s, client, ctx, _, wid := jobTestFixture(t)
	var calls atomic.Int32
	var valid atomic.Bool
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		_ = json.NewDecoder(r.Body).Decode(&payload)
		valid.Store(boolean(payload, "stream") && number(payload, "max_tokens", 0) == 262144)
		calls.Add(1)
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"자동화 AI 결과 [1]\"}}]}\n\ndata: [DONE]\n\n")
	}))
	defer provider.Close()
	client.request("PUT", "/api/v1/admin/settings", map[string]any{"ai_enabled": true, "ai_base_url": provider.URL + "/v1", "ai_model": "test", "ai_max_tokens": 262144}, 200)
	doc := testJSONObject(t, client.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "AI 문서", "markdown": "원본 본문"}, 200))
	drainJobs(t, s)
	client.request("POST", "/api/v1/automations", map[string]any{"workspace_id": wid, "name": "AI 요약", "trigger": "document.updated", "enabled": true, "conditions": map[string]any{"document_id": doc["id"]}, "actions": []any{map[string]any{"type": "ai", "prompt": "요약해주세요"}}}, 200)
	client.request("PUT", "/api/v1/documents/"+str(doc, "id"), map[string]any{"version": 1, "markdown": "변경한 원본"}, 200)
	drainJobs(t, s)
	result := testJSONObject(t, client.request("GET", "/api/v1/documents/"+str(doc, "id"), nil, 200))
	if calls.Load() != 1 || !valid.Load() || !strings.Contains(str(result, "markdown"), "자동화 AI 결과") {
		t.Fatalf("AI automation did not stream/append: %v", result)
	}
	db := testJSONObject(t, client.request("POST", "/api/v1/databases", map[string]any{"workspace_id": wid, "name": "자동화 DB"}, 200))
	row := testJSONObject(t, client.request("POST", "/api/v1/databases/"+str(db, "id")+"/rows", map[string]any{"values": map[string]any{"title": "행", "status": "시작 전"}}, 200))
	client.request("POST", "/api/v1/automations", map[string]any{"workspace_id": wid, "name": "속성 변경", "trigger": "comment.created", "enabled": true, "conditions": map[string]any{"document_id": doc["id"]}, "actions": []any{map[string]any{"type": "update_property", "database_id": db["id"], "row_id": row["id"], "values": map[string]any{"status": "완료"}}}}, 200)
	client.request("POST", "/api/v1/documents/"+str(doc, "id")+"/comments", map[string]any{"body": "상태 바꾸기"}, 200)
	drainJobs(t, s)
	var status string
	_ = s.DB.QueryRow(ctx, "SELECT values->>'status' FROM database_rows WHERE id=$1", row["id"]).Scan(&status)
	if status != "완료" {
		t.Fatal("database action failed")
	}
	client.request("PUT", "/api/v1/admin/settings", map[string]any{"approval_enabled": true}, 200)
	client.request("POST", "/api/v1/automations", map[string]any{"workspace_id": wid, "name": "승인 우회 금지", "trigger": "comment.created", "enabled": true, "conditions": map[string]any{"document_id": doc["id"]}, "actions": []any{map[string]any{"type": "update_document", "status": "published"}}}, 200)
	client.request("POST", "/api/v1/documents/"+str(doc, "id")+"/comments", map[string]any{"body": "승인 정책"}, 200)
	drainJobs(t, s)
	_ = s.DB.QueryRow(ctx, "SELECT status FROM documents WHERE id=$1", doc["id"]).Scan(&status)
	if status == "published" {
		t.Fatal("automation bypassed approval")
	}
	var failures int
	_ = s.DB.QueryRow(ctx, "SELECT count(*) FROM automation_jobs WHERE kind='automation.execute' AND status='failed'").Scan(&failures)
	if failures == 0 {
		t.Fatal("approval failure not recorded")
	}
}

func TestWebhookRedirectAndRevokedToken(t *testing.T) {
	s, client, ctx, _, wid := jobTestFixture(t)
	var redirected atomic.Int32
	destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { redirected.Add(1); w.WriteHeader(204) }))
	defer destination.Close()
	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, destination.URL, 307) }))
	defer receiver.Close()
	hook := testJSONObject(t, client.request("POST", "/api/v1/webhooks", map[string]any{"workspace_id": wid, "name": "redirect forbidden", "url": receiver.URL, "secret": "test-secret-with-enough-entropy", "events": []string{"document.created"}, "enabled": true, "max_attempts": 3, "timeout_seconds": 2}, 200))
	test := testJSONObject(t, client.request("POST", "/api/v1/webhooks/"+str(hook, "id")+"/test", map[string]any{}, 200))
	drainJobs(t, s)
	var status string
	var attempts int
	_ = s.DB.QueryRow(ctx, "SELECT status,attempts FROM automation_jobs WHERE id=$1", test["job_id"]).Scan(&status, &attempts)
	if redirected.Load() != 0 || status != "failed" || attempts != 1 {
		t.Fatal("redirect followed or permanently invalid response retried")
	}
	issued := testJSONObject(t, client.request("POST", "/api/v1/keys", map[string]any{"name": "job-key", "workspace_id": wid, "scopes": []string{"document:read", "document:write"}, "expires_in_days": 1, "rate_limit": 100}, 201))
	tokenClient := newIntegrationTestClient(t, client.base)
	tokenClient.token = str(issued, "token")
	doc := testJSONObject(t, tokenClient.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "폐기 전 이벤트"}, 200))
	key := issued["key"].(map[string]any)
	client.request("DELETE", "/api/v1/keys/"+str(key, "id"), nil, 200)
	drainJobs(t, s)
	_ = s.DB.QueryRow(ctx, "SELECT status FROM automation_jobs WHERE kind='event.dispatch' AND resource_id=$1", doc["id"]).Scan(&status)
	if status != "failed" {
		t.Fatal("revoked token did not stop queued work")
	}
}
