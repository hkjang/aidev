package server

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"
)

func TestJobsCredentialDeletionProvenanceAndLegacyMigration(t *testing.T) {
	s, client, ctx, p, wid := jobTestFixture(t)
	issued := testJSONObject(t, client.request("POST", "/api/v1/keys", map[string]any{"name": "작업키", "workspace_id": wid, "scopes": []string{"document:read"}, "expires_in_days": 30}, 201))
	var keyID string
	if e := s.DB.QueryRow(ctx, `SELECT id::text FROM api_keys WHERE user_id=$1 AND name='작업키'`, p.ID).Scan(&keyID); e != nil {
		t.Fatal(e)
	}
	keyPrincipal := *p
	keyPrincipal.TokenID = keyID
	keyPrincipal.Scopes = []string{"document:read"}
	keyPrincipal.WorkspaceID = wid
	keyCtx := context.WithValue(ctx, principalKey, &keyPrincipal)
	var calls atomic.Int32
	s.RegisterJobHandler("fixture.credential", func(context.Context, Job) (map[string]any, error) {
		calls.Add(1)
		return map[string]any{"ok": true}, nil
	})
	id, e := s.EnqueueJob(keyCtx, nil, "fixture.credential", p.ID, wid, map[string]any{})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.DB.Exec(ctx, `DELETE FROM api_keys WHERE id=$1`, keyID); e != nil {
		t.Fatal(e)
	}
	var stored string
	if e = s.DB.QueryRow(ctx, `SELECT coalesce(token_id::text,'') FROM automation_jobs WHERE id=$1`, id).Scan(&stored); e != nil || stored != keyID {
		t.Fatal("common jobs must retain key ID provenance", stored, e)
	}
	drainJobs(t, s)
	if calls.Load() != 0 {
		t.Fatal("deleted-key job executed")
	}
	if _, e = s.DB.Exec(ctx, `UPDATE automation_jobs SET token_id=NULL,status='pending',attempts=0,run_after=now() WHERE id=$1`, id); e != nil {
		t.Fatal(e)
	}
	drainJobs(t, s)
	if calls.Load() != 0 {
		t.Fatal("cleared key ID escalated job")
	}
	legacy, e := s.EnqueueJob(ctx, nil, "fixture.credential", p.ID, wid, map[string]any{})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.DB.Exec(ctx, `UPDATE automation_jobs SET actor_constraints='{"scope_restricted":true,"scopes":["document:read"]}' WHERE id=$1`, legacy); e != nil {
		t.Fatal(e)
	}
	if e = s.migrateJobs(ctx); e != nil {
		t.Fatal(e)
	}
	drainJobs(t, s)
	if calls.Load() != 0 {
		t.Fatal("ambiguous legacy scope was promoted")
	}
	ordinary, e := s.EnqueueJob(ctx, nil, "fixture.credential", p.ID, wid, map[string]any{})
	if e != nil {
		t.Fatal(e)
	}
	drainJobs(t, s)
	if calls.Load() != 1 {
		t.Fatal("ordinary cookie job was blocked", ordinary)
	}
	_ = issued
}
func TestJobsCurrentKeyRevocationCancelsRunningJob(t *testing.T) {
	s, client, ctx, p, wid := jobTestFixture(t)
	client.request("POST", "/api/v1/keys", map[string]any{"name": "실행 중 키", "workspace_id": wid, "scopes": []string{"document:read"}, "expires_in_days": 30}, 201)
	var keyID string
	_ = s.DB.QueryRow(ctx, `SELECT id::text FROM api_keys WHERE user_id=$1 AND name='실행 중 키'`, p.ID).Scan(&keyID)
	copy := *p
	copy.TokenID = keyID
	copy.Scopes = []string{"document:read"}
	copy.WorkspaceID = wid
	started := make(chan struct{})
	s.RegisterJobHandler("fixture.long-key", func(c context.Context, _ Job) (map[string]any, error) {
		close(started)
		<-c.Done()
		return nil, c.Err()
	})
	_, e := s.EnqueueJob(context.WithValue(ctx, principalKey, &copy), nil, "fixture.long-key", p.ID, wid, map[string]any{})
	if e != nil {
		t.Fatal(e)
	}
	j, e := s.claimJob(ctx)
	if e != nil {
		t.Fatal(e)
	}
	done := make(chan struct{})
	go func() { s.runJob(ctx, j); close(done) }()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("job did not start")
	}
	if _, e = s.DB.Exec(ctx, `UPDATE api_keys SET scopes='{}' WHERE id=$1`, keyID); e != nil {
		t.Fatal(e)
	}
	select {
	case <-done:
	case <-time.After(4 * time.Second):
		t.Fatal("key reduction did not cancel worker")
	}
	var status string
	_ = s.DB.QueryRow(ctx, `SELECT status FROM automation_jobs WHERE id=$1`, j.ID).Scan(&status)
	if status != "cancelled" {
		t.Fatal("key-invalidated job was retried", status)
	}
}
func TestAutomationNestedWebhookRetainsActorConstraints(t *testing.T) {
	s, _, ctx, p, wid := jobTestFixture(t)
	hook, eventID := newID(), newID()
	cipher, e := s.encrypt("fixture-secret")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.DB.Exec(ctx, `INSERT INTO automation_webhooks(id,workspace_id,owner_id,name,url,secret_ciphertext) VALUES($1,$2,$3,'fixture','https://example.invalid',$4)`, hook, wid, p.ID, cipher); e != nil {
		t.Fatal(e)
	}
	if _, e = s.DB.Exec(ctx, `INSERT INTO automation_events(id,type,workspace_id,actor_id) VALUES($1,'date.reached',$2,$3)`, eventID, wid, p.ID); e != nil {
		t.Fatal(e)
	}
	constraints := actorConstraints{PluginID: "fixture-plugin", Restricted: true, Scopes: []string{"document:read"}, TokenBound: true}
	j := Job{ID: newID(), WorkspaceID: wid, OwnerID: p.ID, ActorID: p.ID, EventID: eventID, TokenID: newID(), Constraints: constraints}
	if _, e = s.executeAutomationAction(ctx, j, Event{}, p, p, AutomationAction{Type: "webhook", WebhookID: hook}); e != nil {
		t.Fatal(e)
	}
	var raw []byte
	var got actorConstraints
	if e = s.DB.QueryRow(ctx, `SELECT actor_constraints FROM automation_jobs WHERE event_id=$1 AND kind='webhook.deliver'`, eventID).Scan(&raw); e != nil {
		t.Fatal(e)
	}
	if e = json.Unmarshal(raw, &got); e != nil || !got.TokenBound || got.PluginID != constraints.PluginID || !got.Restricted || len(got.Scopes) != 1 {
		t.Fatalf("nested webhook lost constraints: %s %v", raw, e)
	}
}
