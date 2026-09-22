package server

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

type runbookKubernetesFixture struct {
	mu                               sync.Mutex
	server                           *httptest.Server
	job                              map[string]any
	creates, deletes                 int
	ambiguous, allowNetwork, sidecar bool
}

func runbookCloneJSON(m map[string]any) map[string]any {
	var v map[string]any
	_ = json.Unmarshal(jsonValue(m), &v)
	return v
}
func newRunbookKubernetesFixture(t *testing.T) *runbookKubernetesFixture {
	f := &runbookKubernetesFixture{}
	f.server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if r.Header.Get("Authorization") != "Bearer test-runner-secret" {
			http.Error(w, "denied", 401)
			return
		}
		switch {
		case r.Method == "GET" && r.URL.Path == "/api/v1/namespaces/madi-execution":
			jsonResponse(w, 200, map[string]any{"metadata": map[string]any{"uid": "f5e08796-84a6-43fc-b8b5-40f6eab7b431", "labels": map[string]any{"pod-security.kubernetes.io/enforce": "restricted", "pod-security.kubernetes.io/enforce-version": "latest"}}})
		case strings.HasSuffix(r.URL.Path, "/networkpolicies"):
			spec := map[string]any{"podSelector": map[string]any{}, "policyTypes": []string{"Ingress", "Egress"}}
			if f.allowNetwork {
				spec["egress"] = []any{map[string]any{}}
			}
			jsonResponse(w, 200, map[string]any{"items": []any{map[string]any{"metadata": map[string]any{"name": "deny-all", "uid": "3117b403-9f5c-41c0-a6c6-42ece0dfed5e"}, "spec": spec}}})
		case strings.HasSuffix(r.URL.Path, "/resourcequotas"):
			hard := map[string]any{"pods": "20", "count/jobs.batch": "20", "requests.cpu": "2", "limits.cpu": "2", "requests.memory": "2Gi", "limits.memory": "2Gi"}
			jsonResponse(w, 200, map[string]any{"items": []any{map[string]any{"metadata": map[string]any{"name": "runbook-limit", "uid": "2d7d3845-9cbd-42c1-b724-9984dcc664ff"}, "spec": map[string]any{"hard": hard}, "status": map[string]any{"hard": hard}}}})
		case r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/jobs"):
			if f.job != nil {
				http.Error(w, "exists", 409)
				return
			}
			f.creates++
			_ = json.NewDecoder(r.Body).Decode(&f.job)
			runbookObject(f.job, "metadata")["uid"] = "322ab30e-6a1b-4c1b-b90a-59df268c8f50"
			f.job["status"] = map[string]any{"succeeded": 1}
			if f.ambiguous {
				http.Error(w, "response lost", 503)
				return
			}
			jsonResponse(w, 201, f.job)
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/jobs/"):
			if f.job == nil {
				http.Error(w, "missing", 404)
				return
			}
			jsonResponse(w, 200, f.job)
		case r.Method == "DELETE" && strings.Contains(r.URL.Path, "/jobs/"):
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if f.job != nil && str(runbookObject(body, "preconditions"), "uid") != str(runbookObject(f.job, "metadata"), "uid") {
				http.Error(w, "UID precondition failed", 409)
				return
			}
			f.deletes++
			f.job = nil
			jsonResponse(w, 200, map[string]any{"status": "Success"})
		case strings.HasSuffix(r.URL.Path, "/pods"):
			pod := map[string]any{"metadata": map[string]any{"name": "madi-runner-pod", "ownerReferences": []any{map[string]any{"kind": "Job", "uid": "322ab30e-6a1b-4c1b-b90a-59df268c8f50"}}}, "spec": runbookObject(runbookObject(runbookObject(f.job, "spec"), "template"), "spec")}
			pod = runbookCloneJSON(pod)
			if f.sidecar {
				spec := runbookObject(pod, "spec")
				spec["containers"] = append(runbookArray(spec, "containers"), map[string]any{"name": "injected", "image": "bad"})
			}
			jsonResponse(w, 200, map[string]any{"items": []any{pod}})
		case strings.HasSuffix(r.URL.Path, "/log"):
			w.Write([]byte("격리된 컨테이너 검증 완료\n"))
		default:
			http.Error(w, "unexpected", 404)
		}
	}))
	t.Cleanup(f.server.Close)
	return f
}

func TestRunbookKubernetesTLSIsolationRecoveryAndUIDCancel(t *testing.T) {
	f := newRunbookKubernetesFixture(t)
	s := &Server{EncryptionKey: bytes.Repeat([]byte{7}, 32)}
	encrypted, e := s.encrypt("test-runner-secret")
	if e != nil {
		t.Fatal(e)
	}
	action := runbookAction{ID: "check", Name: "컨테이너 점검", Roles: []string{"editor"}, Timeout: 30, Image: "offline/check@sha256:" + strings.Repeat("a", 64), Argv: []string{"/usr/bin/check", "staging"}, CPUMilli: 1000, MemoryMi: 1024}
	runner := runbookRunner{ID: newID(), WorkspaceID: newID(), Name: "사내 Kubernetes", Kind: "kubernetes", Enabled: true, Revision: 1, Config: map[string]any{"base_url": f.server.URL, "token": encrypted, "namespace": "madi-execution", "ca_pem": string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: f.server.Certificate().Raw}))}, Actions: []runbookAction{action}}
	remote, e := s.newRunbookRemote(runner)
	if e != nil {
		t.Fatal(e)
	}
	defer remote.close()
	ctx := context.Background()
	report, e := remote.preflight(ctx, action)
	if e != nil {
		t.Fatal(e)
	}
	step := runbookPlanStep{Name: "격리 점검", RunnerID: runner.ID, RunnerRevision: 1, Kind: "kubernetes", Action: action, Argv: action.Argv, Parameters: map[string]any{}, Remote: report}
	id := newID()
	f.mu.Lock()
	f.ambiguous = true
	f.mu.Unlock()
	if _, e = remote.launch(ctx, id, 0, step); e == nil {
		t.Fatal("ambiguous response should be recoverable error")
	}
	external, e := remote.launch(ctx, id, 0, step)
	if e != nil {
		t.Fatal("deterministic Job recovery", e)
	}
	f.mu.Lock()
	creates := f.creates
	f.mu.Unlock()
	if creates != 1 {
		t.Fatal("created more than one distinct Job")
	}
	state, e := remote.poll(ctx, external, step, true)
	if e != nil || state.Status != "succeeded" || !strings.Contains(state.Output, "검증 완료") {
		t.Fatalf("poll durable recovered Job: %#v %v", state, e)
	}
	f.mu.Lock()
	f.sidecar = true
	f.mu.Unlock()
	if _, e = remote.poll(ctx, external, step, true); e == nil {
		t.Fatal("admission-injected sidecar accepted")
	}
	f.mu.Lock()
	f.allowNetwork = true
	f.mu.Unlock()
	if _, e = remote.preflight(ctx, action); e == nil {
		t.Fatal("egress allowance accepted")
	}
	if e = remote.cancelAndWait(ctx, external, step); e != nil {
		t.Fatal(e)
	}
	f.mu.Lock()
	deletes := f.deletes
	f.mu.Unlock()
	if deletes != 1 {
		t.Fatal("UID guarded remote cancellation not called")
	}
	bad := runner
	bad.Config = runbookCloneJSON(runner.Config)
	delete(bad.Config, "ca_pem")
	untrusted, e := s.newRunbookRemote(bad)
	if e != nil {
		t.Fatal(e)
	}
	defer untrusted.close()
	if _, e = untrusted.preflight(ctx, action); e == nil {
		t.Fatal("untrusted TLS certificate accepted")
	}
}
