package server

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

type runbookAWXFixture struct {
	mu                sync.Mutex
	server            *httptest.Server
	launches, cancels int
	status            string
	ambiguous         bool
	modified          string
	parameters        map[string]any
}

func newRunbookAWXFixture(t *testing.T) *runbookAWXFixture {
	t.Helper()
	f := &runbookAWXFixture{status: "successful", modified: "stable-revision"}
	f.server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if r.Header.Get("Authorization") != "Bearer test-runner-secret" {
			http.Error(w, "secret", 401)
			return
		}
		switch r.Method + " " + r.URL.Path {
		case "GET /api/v2/job_templates/7/":
			jsonResponse(w, 200, map[string]any{"id": 7, "name": "격리 점검", "timeout": 30, "project": 9, "execution_environment": 17, "modified": f.modified, "playbook": "check.yml", "ask_variables_on_launch": true, "extra_vars": "hidden-template-secret"})
		case "GET /api/v2/execution_environments/17/":
			jsonResponse(w, 200, map[string]any{"id": 17, "image": "offline/ansible@sha256:" + strings.Repeat("b", 64), "modified": "stable-ee"})
		case "GET /api/v2/projects/9/":
			jsonResponse(w, 200, map[string]any{"id": 9, "scm_revision": "fixed-git-sha", "scm_update_on_launch": false})
		case "GET /api/v2/job_templates/7/launch/":
			jsonResponse(w, 200, map[string]any{"passwords_needed_to_start": []any{}})
		case "POST /api/v2/job_templates/7/launch/":
			f.launches++
			_ = json.NewDecoder(r.Body).Decode(&f.parameters)
			if f.ambiguous {
				http.Error(w, "launch accepted but response lost", 503)
				return
			}
			jsonResponse(w, 201, map[string]any{"id": 41, "ignored_fields": map[string]any{}})
		case "GET /api/v2/jobs/41/":
			jsonResponse(w, 200, map[string]any{"id": 41, "job_template": 7, "scm_revision": "fixed-git-sha", "status": f.status})
		case "GET /api/v2/jobs/41/stdout/":
			w.Write([]byte("점검 완료\ntest-runner-secret\n"))
		case "POST /api/v2/jobs/41/cancel/":
			f.cancels++
			f.status = "canceled"
			jsonResponse(w, 202, map[string]any{})
		default:
			http.Error(w, "unexpected path", 404)
		}
	}))
	t.Cleanup(f.server.Close)
	return f
}
func (f *runbookAWXFixture) config() map[string]any {
	return map[string]any{"base_url": f.server.URL, "token": "test-runner-secret", "ca_pem": string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: f.server.Certificate().Raw}))}
}

func runbookTestSetup(t *testing.T) (*Server, *integrationTestClient, *integrationTestClient, string, string, string, *runbookAWXFixture) {
	t.Helper()
	s, admin, editor, wid, uid := collaborationTestSetup(t)
	if e := s.migrateRunbook(context.Background()); e != nil {
		t.Fatal(e)
	}
	if s.approvalAdapters["runbook"].Lock == nil {
		s.registerRunbook()
	}
	fixture := newRunbookAWXFixture(t)
	created := testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "격리 운영 절차", "markdown": "# 목적\n서버 안에서 명령을 실행하지 않습니다."}, 200))
	doc := str(created, "id")
	admin.request("GET", "/api/v1/documents/"+doc+"/runbook", nil, 404)
	admin.request("PUT", "/api/v1/admin/runbook/settings", map[string]any{"enabled": true, "revision": 1}, 200)
	runner := testJSONObject(t, admin.request("POST", "/api/v1/admin/runbook/runners", map[string]any{"workspace_id": wid, "name": "사내 AWX", "kind": "awx", "enabled": true, "config": fixture.config(), "actions": []runbookAction{{ID: "check", Name: "안전 점검", Roles: []string{"editor"}, TemplateID: 7, Timeout: 30, Parameters: []runbookParameter{{Name: "target", Label: "대상", Type: "enum", Required: true, Options: []string{"staging", "production"}}}}}}, 200))
	rid := str(runner, "id")
	if bytes.Contains(jsonValue(runner), []byte("test-runner-secret")) {
		t.Fatal("secret exposed in admin response")
	}
	step := runbookStep{Name: "대상 점검", RunnerID: rid, ActionID: "check", Parameters: map[string]any{"target": "staging"}}
	admin.request("PUT", "/api/v1/documents/"+doc+"/runbook", runbookDefinition{Version: 0, Purpose: "격리 점검", Steps: []runbookStep{step}, ValidationSteps: []runbookStep{step}, RollbackSteps: []runbookStep{step}}, 200)
	return s, admin, editor, wid, uid, doc, fixture
}
func runbookPrepare(t *testing.T, c *integrationTestClient, doc, phase string) map[string]any {
	t.Helper()
	return testJSONObject(t, c.request("POST", "/api/v1/documents/"+doc+"/runbook/prepare", map[string]any{"phase": phase}, 201))
}
func runbookConfirm(t *testing.T, c *integrationTestClient, p map[string]any) map[string]any {
	t.Helper()
	id := str(p, "id")
	return testJSONObject(t, c.request("POST", "/api/v1/runbook/executions/"+id+"/execute", map[string]any{"version": p["version"], "confirmation": "EXECUTE " + id, "approval_version": number(runbookObject(p, "approval"), "version", 0)}, 202))
}
func runbookJobForTest(t *testing.T, s *Server, id string) Job {
	t.Helper()
	var j Job
	e := s.DB.QueryRow(context.Background(), `SELECT id::text,kind,workspace_id::text,owner_id::text,actor_id::text,payload FROM automation_jobs WHERE id=$1`, id).Scan(&j.ID, &j.Kind, &j.WorkspaceID, &j.OwnerID, &j.ActorID, &j.Payload)
	if e != nil {
		t.Fatal(e)
	}
	return j
}

func TestPostgresRunbookAWXSnapshotExecutionAndReplay(t *testing.T) {
	s, admin, _, _, _, doc, f := runbookTestSetup(t)
	plan := runbookPrepare(t, admin, doc, "validate")
	if bytes.Contains(jsonValue(plan), []byte("hidden-template-secret")) {
		t.Fatal("AWX template secret in snapshot")
	}
	id := str(plan, "id")
	admin.request("POST", "/api/v1/runbook/executions/"+id+"/execute", map[string]any{"version": 1, "confirmation": "yes"}, 400)
	queued := runbookConfirm(t, admin, plan)
	admin.request("POST", "/api/v1/runbook/executions/"+id+"/execute", map[string]any{"version": 1, "confirmation": "EXECUTE " + id}, 409)
	j := runbookJobForTest(t, s, str(queued, "job_id"))
	if _, e := s.executeRunbookJob(context.Background(), j); e != nil {
		t.Fatal(e)
	}
	if _, e := s.executeRunbookJob(context.Background(), j); e != nil {
		t.Fatal("re-delivery of terminal execution", e)
	}
	out := testJSONObject(t, admin.request("GET", "/api/v1/runbook/executions/"+id, nil, 200))
	if str(out, "status") != "succeeded" {
		t.Fatalf("execution: %s", jsonValue(out))
	}
	steps := runbookArray(out, "steps")
	if len(steps) != 1 || str(steps[0].(map[string]any), "state") != "succeeded" {
		t.Fatalf("step API contract: %s", jsonValue(out))
	}
	f.mu.Lock()
	launches, params := f.launches, f.parameters
	f.mu.Unlock()
	if launches != 1 || str(runbookObject(params, "extra_vars"), "target") != "staging" {
		t.Fatalf("launch count/typed vars: %d %v", launches, params)
	}
	var logs string
	if e := s.DB.QueryRow(context.Background(), `SELECT COALESCE(string_agg(message,E'\n'),'') FROM runbook_events WHERE execution_id=$1`, id).Scan(&logs); e != nil {
		t.Fatal(e)
	}
	if strings.Contains(logs, "test-runner-secret") || !strings.Contains(logs, "점검 완료") {
		t.Fatalf("safe logs: %s", logs)
	}
	definition := testJSONObject(t, admin.request("GET", "/api/v1/documents/"+doc+"/runbook", nil, 200))
	if definition["last_tested_at"] == nil {
		t.Fatal("successful validation did not mark exact definition tested")
	}
}

func TestPostgresRunbookAWXUnknownNeverRelaunches(t *testing.T) {
	s, admin, _, _, _, doc, f := runbookTestSetup(t)
	f.mu.Lock()
	f.ambiguous = true
	f.mu.Unlock()
	plan := runbookPrepare(t, admin, doc, "execute")
	queued := runbookConfirm(t, admin, plan)
	j := runbookJobForTest(t, s, str(queued, "job_id"))
	for i := 0; i < 2; i++ {
		if _, e := s.executeRunbookJob(context.Background(), j); e == nil {
			t.Fatal("ambiguous launch succeeded")
		}
	}
	out := testJSONObject(t, admin.request("GET", "/api/v1/runbook/executions/"+str(plan, "id"), nil, 200))
	f.mu.Lock()
	launches := f.launches
	f.mu.Unlock()
	if str(out, "status") != "unknown" || launches != 1 {
		t.Fatalf("unknown must not repeat: %v launches=%d", out, launches)
	}
	second := runbookPrepare(t, admin, doc, "execute")
	admin.request("POST", "/api/v1/runbook/executions/"+str(second, "id")+"/execute", map[string]any{"version": 1, "confirmation": "EXECUTE " + str(second, "id")}, 409)
	id := str(plan, "id")
	admin.request("POST", "/api/v1/admin/runbook/executions/"+id+"/resolve", map[string]any{"confirmation": "RESOLVE " + id, "reason": "짧음", "external_checked": true}, 400)
	admin.request("POST", "/api/v1/admin/runbook/executions/"+id+"/resolve", map[string]any{"confirmation": "RESOLVE " + id, "reason": "격리 실행기에서 추적 ID와 시간을 대조하여 미실행 상태를 직접 확인함", "external_checked": true}, 200)
	resolved := testJSONObject(t, admin.request("GET", "/api/v1/runbook/executions/"+id, nil, 200))
	if str(resolved, "status") != "cancelled" || !strings.Contains(str(resolved, "last_error"), "자동 검증") {
		t.Fatal("operator attestation incorrectly presented as remote verified success")
	}
	_ = runbookConfirm(t, admin, second)
}

func TestPostgresRunbookSourceDriftRevocationAndCancellation(t *testing.T) {
	s, admin, editor, wid, uid, doc, f := runbookTestSetup(t)
	plan := runbookPrepare(t, admin, doc, "execute")
	_, e := s.DB.Exec(context.Background(), `UPDATE runbook_documents SET purpose='changed',version=version+1 WHERE document_id=$1`, doc)
	if e != nil {
		t.Fatal(e)
	}
	admin.request("POST", "/api/v1/runbook/executions/"+str(plan, "id")+"/execute", map[string]any{"version": 1, "confirmation": "EXECUTE " + str(plan, "id")}, 409)
	plan = runbookPrepare(t, editor, doc, "execute")
	queued := runbookConfirm(t, editor, plan)
	f.mu.Lock()
	f.status = "running"
	f.mu.Unlock()
	j := runbookJobForTest(t, s, str(queued, "job_id"))
	done := make(chan error, 1)
	go func() { _, err := s.executeRunbookJob(context.Background(), j); done <- err }()
	deadline := time.Now().Add(5 * time.Second)
	for {
		f.mu.Lock()
		started := f.launches > 0
		f.mu.Unlock()
		if started {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("worker launch timeout")
		}
		time.Sleep(20 * time.Millisecond)
	}
	if _, e = s.DB.Exec(context.Background(), `DELETE FROM workspace_members WHERE workspace_id=$1 AND user_id=$2`, wid, uid); e != nil {
		t.Fatal(e)
	}
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("revoked run succeeded")
		}
	case <-time.After(8 * time.Second):
		t.Fatal("revoked execution was not cancelled")
	}
	f.mu.Lock()
	cancelled := f.cancels
	f.mu.Unlock()
	if cancelled < 1 {
		t.Fatal("ACL revocation did not call remote cancel")
	}
	out := testJSONObject(t, admin.request("GET", "/api/v1/runbook/executions/"+str(plan, "id"), nil, 200))
	if str(out, "status") != "cancelled" {
		t.Fatalf("state after revocation: %v", out)
	}
	editor.request("GET", "/api/v1/runbook/executions/"+str(plan, "id"), nil, 404)
	plan = runbookPrepare(t, admin, doc, "rollback")
	queued = runbookConfirm(t, admin, plan)
	admin.request("POST", "/api/v1/runbook/executions/"+str(plan, "id")+"/cancel", map[string]any{"confirmation": "CANCEL " + str(plan, "id")}, 202)
	if e = s.reconcileRunbooks(context.Background()); e != nil {
		t.Fatal(e)
	}
	out = testJSONObject(t, admin.request("GET", "/api/v1/runbook/executions/"+str(plan, "id"), nil, 200))
	if str(out, "status") != "cancelled" {
		t.Fatalf("queued cancellation: %v", out)
	}
}

func TestRunbookTypedParametersAndFixedArgv(t *testing.T) {
	base := runbookRunner{WorkspaceID: newID(), Name: "격리 명령", Kind: "kubernetes", Enabled: true, Config: map[string]any{"base_url": "https://cluster.example.test", "namespace": "madi-execution", "token": "secret"}, Actions: []runbookAction{{ID: "check", Name: "점검", Roles: []string{"editor"}, Timeout: 30, Image: "internal/check@sha256:" + strings.Repeat("a", 64), Argv: []string{"/usr/bin/check", "${target}"}, CPUMilli: 100, MemoryMi: 64, Parameters: []runbookParameter{{Name: "target", Type: "enum", Required: true, Options: []string{"staging", "production"}}}}}}
	if e := validateRunbookRunner(&base); e != nil {
		t.Fatal(e)
	}
	for _, input := range []map[string]any{{"target": "staging; rm -rf /"}, {"target": "staging", "arbitrary": true}, {"target": 123}} {
		if _, _, e := runbookParameters(base.Actions[0], input); e == nil {
			t.Fatalf("unsafe typed parameter accepted: %v", input)
		}
	}
	_, argv, e := runbookParameters(base.Actions[0], map[string]any{"target": "staging"})
	if e != nil || fmt.Sprint(argv) != "[/usr/bin/check staging]" {
		t.Fatalf("fixed argv: %v %v", argv, e)
	}
	for _, args := range [][]string{{"/bin/sh", "-c", "${target}"}, {"${target}"}, {"/usr/bin/check", "prefix-${target}"}} {
		bad := base
		bad.Actions = append([]runbookAction{}, base.Actions...)
		bad.Actions[0].Argv = args
		if validateRunbookRunner(&bad) == nil {
			t.Fatalf("unsafe argv accepted %v", args)
		}
	}
}

func TestPostgresRunbookExplicitApprovalConsumesOnce(t *testing.T) {
	s, admin, reviewer, wid, uid, doc, f := runbookTestSetup(t)
	admin.request("PUT", "/api/v1/admin/settings", map[string]any{"approval_enabled": true}, 200)
	plan := runbookPrepare(t, admin, doc, "execute")
	id := str(plan, "id")
	admin.request("POST", "/api/v1/runbook/executions/"+id+"/execute", map[string]any{"version": 1, "confirmation": "EXECUTE " + id}, 409)
	admin.request("POST", "/api/v1/runbook/executions/"+id+"/approval", map[string]any{"version": 1}, 409)
	admin.request("POST", "/api/v1/admin/approval/policies", map[string]any{"name": "Runbook 실행 검토", "workspace_id": wid, "resource_kind": "runbook", "stages": []any{map[string]any{"name": "별도 운영자 검토", "mode": "all", "gates": []any{map[string]any{"name": "운영 검토자", "kind": "user", "id": uid}}}}}, 200)
	req := testJSONObject(t, admin.request("POST", "/api/v1/runbook/executions/"+id+"/approval", map[string]any{"version": 1}, 200))
	admin.request("POST", "/api/v1/approvals/requests/"+str(req, "id")+"/decisions", map[string]any{"action": "approve", "request_version": req["version"]}, 403)
	reviewer.request("POST", "/api/v1/approvals/requests/"+str(req, "id")+"/decisions", map[string]any{"action": "approve", "request_version": req["version"]}, 200)
	approved := testJSONObject(t, admin.request("GET", "/api/v1/runbook/executions/"+id, nil, 200))
	queued := runbookConfirm(t, admin, approved)
	j := runbookJobForTest(t, s, str(queued, "job_id"))
	if _, e := s.executeRunbookJob(context.Background(), j); e != nil {
		t.Fatal(e)
	}
	var status string
	if e := s.DB.QueryRow(context.Background(), `SELECT status FROM approval_requests WHERE id=$1`, str(req, "id")).Scan(&status); e != nil {
		t.Fatal(e)
	}
	f.mu.Lock()
	launches := f.launches
	f.mu.Unlock()
	if status != "consumed" || launches != 1 {
		t.Fatalf("approval consumed once: %s launches %d", status, launches)
	}
}
