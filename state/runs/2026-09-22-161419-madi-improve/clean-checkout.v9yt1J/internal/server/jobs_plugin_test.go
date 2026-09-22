package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestPostgresJobsPluginScopeSnapshotAndRevocation(t *testing.T) {
	s, client, ctx, p, wid := jobTestFixture(t)
	manifest := pluginTestManifest()
	pluginTestUpload(t, client, pluginTestZIP(t, manifest, nil), "", 200)
	grant := "/api/v1/workspaces/" + wid + "/plugins/" + manifest.ID
	client.request("PUT", grant, map[string]any{"enabled": true, "capabilities": []string{"document:read", "document:write"}}, 200)
	constrained := *p
	constrained.PluginID = manifest.ID
	constrained.ScopeRestricted = true
	constrained.WorkspaceID = wid
	constrained.Scopes = []string{"document:read"}
	pluginCtx := context.WithValue(ctx, principalKey, &constrained)
	id, e := s.EnqueueJob(pluginCtx, nil, "test.plugin.snapshot", p.ID, wid, map[string]any{})
	if e != nil {
		t.Fatal(e)
	}
	client.request("PUT", grant, map[string]any{"enabled": true, "capabilities": manifest.Capabilities}, 200)
	var calls atomic.Int32
	s.RegisterJobHandler("test.plugin.snapshot", func(ctx context.Context, j Job) (map[string]any, error) {
		current, e := s.workerPrincipal(ctx, j.ActorID, j.TokenID, j.WorkspaceID)
		if e != nil {
			return nil, e
		}
		if !current.ScopeRestricted || current.PluginID != manifest.ID || !jobScope(current, "document:read") || jobScope(current, "document:write") || jobScope(current, "ai:execute") {
			t.Errorf("enqueued plugin scope expanded: %#v", current)
		}
		child, e := s.EnqueueJob(ctx, nil, "test.plugin.child", p.ID, wid, map[string]any{})
		calls.Add(1)
		return map[string]any{"child": child}, e
	})
	s.RegisterJobHandler("test.plugin.child", func(ctx context.Context, j Job) (map[string]any, error) {
		current, e := s.workerPrincipal(ctx, j.ActorID, j.TokenID, j.WorkspaceID)
		if e != nil {
			return nil, e
		}
		if current.PluginID != manifest.ID || jobScope(current, "document:write") {
			t.Errorf("child job lost attenuation: %#v", current)
		}
		calls.Add(1)
		return map[string]any{}, nil
	})
	drainJobs(t, s)
	if calls.Load() != 2 {
		t.Fatalf("snapshot jobs calls=%d", calls.Load())
	}
	j, e := scanJob(s.DB.QueryRow(ctx, "SELECT "+jobSelect+" FROM automation_jobs WHERE id=$1", id))
	if e != nil || j.Constraints.PluginID != manifest.ID || !j.Constraints.Restricted {
		t.Fatalf("durable constraints missing: %#v %v", j, e)
	}
	id, e = s.EnqueueJob(pluginCtx, nil, "test.plugin.snapshot", p.ID, wid, map[string]any{})
	if e != nil {
		t.Fatal(e)
	}
	client.request("PUT", grant, map[string]any{"enabled": false, "capabilities": manifest.Capabilities}, 200)
	drainJobs(t, s)
	j, e = scanJob(s.DB.QueryRow(ctx, "SELECT "+jobSelect+" FROM automation_jobs WHERE id=$1", id))
	if e != nil || j.Status != "failed" || calls.Load() != 2 {
		t.Fatalf("revoked plugin executed: %#v %v", j, e)
	}
}

func TestPostgresPluginOutboxChildWebhookRevocation(t *testing.T) {
	s, client, ctx, _, wid := jobTestFixture(t)
	manifest := pluginTestManifest()
	pluginTestUpload(t, client, pluginTestZIP(t, manifest, nil), "", 200)
	grant := "/api/v1/workspaces/" + wid + "/plugins/" + manifest.ID
	caps := []string{"document:read", "document:write"}
	client.request("PUT", grant, map[string]any{"enabled": true, "capabilities": caps}, 200)
	var calls atomic.Int32
	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(204) }))
	defer receiver.Close()
	client.request("POST", "/api/v1/webhooks", map[string]any{"workspace_id": wid, "name": "plugin outbox", "url": receiver.URL, "secret": "plugin-outbox-test-secret", "events": []string{"document.created"}, "enabled": true, "max_attempts": 2, "timeout_seconds": 2}, 200)
	client.request("POST", "/api/v1/plugins/"+manifest.ID+"/bridge", map[string]any{"workspace_id": wid, "operation": "documents.create", "args": map[string]any{"title": "플러그인 원본", "markdown": "private body", "visibility": "workspace"}}, 200)
	dispatch, e := s.claimJob(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if dispatch.Kind != "event.dispatch" || dispatch.Constraints.PluginID != manifest.ID || !dispatch.Constraints.Restricted {
		t.Fatalf("outbox constraints lost: %#v", dispatch)
	}
	s.runJob(ctx, dispatch)
	var count int
	if e = s.DB.QueryRow(ctx, "SELECT count(*) FROM automation_jobs WHERE kind='webhook.deliver' AND actor_constraints->>'plugin_id'=$1 AND actor_constraints->>'scope_restricted'='true'", manifest.ID).Scan(&count); e != nil || count != 1 {
		t.Fatalf("child outbox constraints count=%d err=%v", count, e)
	}
	client.request("PUT", grant, map[string]any{"enabled": false, "capabilities": caps}, 200)
	drainJobs(t, s)
	if calls.Load() != 0 {
		t.Fatal("revoked plugin sent a webhook")
	}
}
