package server

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
)

func resumePrepare(t *testing.T, s *Server, c *integrationTestClient, id string) map[string]any {
	t.Helper()
	v := testJSONObject(t, c.request("GET", "/api/v1/migrations/sessions/"+id, nil, 200))
	c.request("POST", "/api/v1/migrations/sessions/"+id+"/prepare", map[string]any{"revision": v["revision"]}, 200)
	drainJobs(t, s)
	return testJSONObject(t, c.request("GET", "/api/v1/migrations/sessions/"+id, nil, 200))
}
func resumeQueueCommit(t *testing.T, c *integrationTestClient, id string, v map[string]any) {
	t.Helper()
	c.request("POST", "/api/v1/migrations/sessions/"+id+"/commit", map[string]any{"revision": v["revision"], "plan_hash": v["plan_hash"], "confirmation": "IMPORT"}, 200)
}
func migrationTestJobContext(ctx context.Context, j Job) context.Context {
	return context.WithValue(ctx, jobContextKey{}, jobContext{ActorID: j.ActorID, TokenID: j.TokenID, JobID: j.ID, LeaseID: j.LeaseID, Constraints: j.Constraints})
}

func TestResumeHTMLQuality(t *testing.T) {
	html := `<title>구조 보존</title><h1>제목</h1><blockquote><p>인용</p></blockquote><ul><li>상위<ul><li>하위</li></ul></li></ul><table><tr><th>이름</th><th>값</th></tr><tr><td colspan="2">병합</td></tr></table><p><img src="assets/local.png"><img src="https://outside.invalid/secret"><a href="javascript:alert(1)">안전한 글</a></p><pre><code>a = 1;
[[code]]</code></pre><script>SECRET_JS</script>`
	title, md, loss, e := migrationHTMLQuality([]byte(html), "a.html", map[string][]byte{"assets/local.png": {1}})
	if e != nil {
		t.Fatal(e)
	}
	if title != "구조 보존" {
		t.Fatal(title)
	}
	for _, want := range []string{"# 제목", "> 인용", "- 상위", "하위", "| 이름 | 값 |", "assets/local.png", "[[code]]"} {
		if !strings.Contains(md, want) {
			t.Fatalf("missing %q: %s", want, md)
		}
	}
	for _, bad := range []string{"SECRET_JS", "javascript:", "outside.invalid"} {
		if strings.Contains(md, bad) {
			t.Fatal("unsafe HTML retained", md)
		}
	}
	if loss["removed_executable"] != 1 || loss["remote_images_omitted"] != 1 || loss["merged_cells_flattened"] != 1 {
		t.Fatalf("loss report %#v", loss)
	}
}

func TestPostgresResumableMigrationCheckpointLeaseRecovery(t *testing.T) {
	s, c, ctx, p, wid := jobTestFixture(t)
	id := resumeTestUpload(t, c, wid, "checkpoint", map[string]string{"a.md": "첫 항목", "b.md": strings.Repeat("x", migrationChunkBytes+37)})
	v := testJSONObject(t, c.request("GET", "/api/v1/migrations/sessions/"+id, nil, 200))
	c.request("POST", "/api/v1/migrations/sessions/"+id+"/prepare", map[string]any{"revision": v["revision"]}, 200)
	j, e := s.claimJob(ctx)
	if e != nil {
		t.Fatal(e)
	}
	jobctx := migrationTestJobContext(ctx, j)
	session, _, e := s.migrationSessionJob(jobctx, j)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.ensureMigrationFolders(jobctx, session); e != nil {
		t.Fatal(e)
	}
	if e = s.classifyMigrationItems(jobctx, p, session); e != nil {
		t.Fatal(e)
	}
	items, e := s.migrationSessionIndex(ctx, id)
	if e != nil {
		t.Fatal(e)
	}
	index := newMigrationLinkIndex(items)
	if e = index.validateParents(items); e != nil {
		t.Fatal(e)
	}
	if e = s.prepareMigrationSessionItem(jobctx, p, session, items[0], index, items); e != nil {
		t.Fatal(e)
	}
	var before, after []byte
	s.DB.QueryRow(ctx, "SELECT prepared_data FROM migration_session_items WHERE id=$1", items[0].ID).Scan(&before)
	if _, e = s.DB.Exec(ctx, "UPDATE automation_jobs SET lease_until=now()-interval '1 second' WHERE id=$1", j.ID); e != nil {
		t.Fatal(e)
	}
	// A separately initialized server recovers the same durable lease/checkpoint.
	restarted, e := New(ctx, s.DB, s.EncryptionKey, "0.1.0", "admin@example.test", "Integration-Test-Password-2026!", nil)
	if e != nil {
		t.Fatal(e)
	}
	defer restarted.CloseCollaboration()
	drainJobs(t, restarted)
	s.DB.QueryRow(ctx, "SELECT prepared_data FROM migration_session_items WHERE id=$1", items[0].ID).Scan(&after)
	if !bytes.Equal(before, after) || len(after) == 0 {
		t.Fatal("prepared checkpoint was recomputed or lost")
	}
	v = testJSONObject(t, c.request("GET", "/api/v1/migrations/sessions/"+id, nil, 200))
	if str(v, "status") != "ready" {
		t.Fatal(v)
	}
	var chunks int
	s.DB.QueryRow(ctx, "SELECT count(*) FROM migration_session_chunks WHERE item_id IN(SELECT id FROM migration_session_items WHERE session_id=$1)", id).Scan(&chunks)
	if chunks != 3 {
		t.Fatal(chunks)
	}
	resumeQueueCommit(t, c, id, v)
	drainJobs(t, restarted)
	var count int
	s.DB.QueryRow(ctx, "SELECT count(*) FROM documents WHERE id IN(SELECT target_id FROM migration_session_items WHERE session_id=$1)", id).Scan(&count)
	if count != 2 {
		t.Fatal(count)
	}
}

func TestPostgresResumableMigrationAtomicFailureAndRevocation(t *testing.T) {
	for _, scenario := range []string{"final_validation", "session_revoked"} {
		t.Run(scenario, func(t *testing.T) {
			s, c, ctx, p, wid := jobTestFixture(t)
			id := resumeTestUpload(t, c, wid, "rollback", map[string]string{"a.md": "먼저 반영될 항목", "z.md": "<final-block>"})
			v := resumePrepare(t, s, c, id)
			if str(v, "status") != "ready" {
				t.Fatal(v)
			}
			resumeQueueCommit(t, c, id, v)
			if scenario == "session_revoked" {
				if _, e := s.DB.Exec(ctx, "DELETE FROM sessions WHERE user_id=$1", p.ID); e != nil {
					t.Fatal(e)
				}
			} else {
				if _, e := s.DB.Exec(ctx, `UPDATE protection_settings SET data='{"enabled":true,"mode":"block","detectors":[],"custom_terms":["<final-block>"]}'`); e != nil {
					t.Fatal(e)
				}
			}
			drainJobs(t, s)
			var count, versions, events int
			var state string
			s.DB.QueryRow(ctx, "SELECT count(*) FROM documents WHERE id IN(SELECT target_id FROM migration_session_items WHERE session_id=$1)", id).Scan(&count)
			s.DB.QueryRow(ctx, "SELECT count(*) FROM document_versions WHERE document_id IN(SELECT target_id FROM migration_session_items WHERE session_id=$1)", id).Scan(&versions)
			s.DB.QueryRow(ctx, "SELECT count(*) FROM automation_events WHERE workspace_id=$1", wid).Scan(&events)
			s.DB.QueryRow(ctx, "SELECT status FROM migration_sessions WHERE id=$1", id).Scan(&state)
			if count != 0 || versions != 0 || events != 0 || state != "failed" {
				t.Fatalf("partial publication docs=%d versions=%d events=%d state=%s", count, versions, events, state)
			}
		})
	}
}

func TestPostgresResumableMigrationRepairsOnlyUnclaimedObject(t *testing.T) {
	s, c, ctx, _, wid := jobTestFixture(t)
	payload := "완전한 첨부 원본"
	id := resumeTestUpload(t, c, wid, "object-recovery", map[string]string{"a.md": "[첨부](asset.txt)", "asset.txt": payload})
	v := resumePrepare(t, s, c, id)
	resumeQueueCommit(t, c, id, v)
	j, e := s.claimJob(ctx)
	if e != nil {
		t.Fatal(e)
	}
	jobctx := migrationTestJobContext(ctx, j)
	session, _, e := s.migrationSessionJob(jobctx, j)
	if e != nil {
		t.Fatal(e)
	}
	var itemID string
	s.DB.QueryRow(ctx, "SELECT id::text FROM migration_session_items WHERE session_id=$1 AND kind='attachment'", id).Scan(&itemID)
	item, e := scanMigrationItem(s.DB.QueryRow(ctx, "SELECT "+migrationItemSelect+" FROM migration_session_items WHERE id=$1", itemID))
	if e != nil {
		t.Fatal(e)
	}
	prepared, e := s.migrationPreparedValue(item)
	if e != nil {
		t.Fatal(e)
	}
	provider, e := s.resolveStorage(ctx, wid)
	if e != nil {
		t.Fatal(e)
	}
	partial, e := s.putStoredObject(ctx, provider, "attachments/"+newID(), strings.NewReader("중간"), 50<<20, "text/plain")
	if e != nil {
		t.Fatal(e)
	}
	partial.Size = int64(len(payload))
	partial.Checksum = digest(payload)
	if _, e = s.DB.Exec(ctx, "UPDATE migration_session_items SET staged_object=$2,storage_fingerprint=$3 WHERE id=$1", itemID, jsonValue(partial), migrationStorageFingerprint(provider)); e != nil {
		t.Fatal(e)
	}
	release, e := s.migrationObjectLock(ctx, itemID, "")
	if e != nil {
		t.Fatal(e)
	}
	e = s.recoverMigrationObject(jobctx, j, session, itemID, "", provider, partial, prepared)
	release()
	if e != nil {
		t.Fatal(e)
	}
	reader, e := s.openStoredObject(ctx, partial)
	if e != nil {
		t.Fatal(e)
	}
	raw, e := io.ReadAll(reader)
	reader.Close()
	if e != nil || string(raw) != payload {
		t.Fatal("partial object not repaired", e)
	}
	// The existing immutable key is reused, then published once through the job.
	s.runJob(ctx, j)
	var state string
	s.DB.QueryRow(ctx, "SELECT status FROM migration_sessions WHERE id=$1", id).Scan(&state)
	if state != "completed" {
		t.Fatal(state)
	}
	c.request("DELETE", "/api/v1/migrations/sessions/"+id, nil, 409)
	s.cleanupMigrationObjects(ctx)
	reader, e = s.openStoredObject(ctx, partial)
	if e != nil {
		t.Fatal("public object collected", e)
	}
	reader.Close()
}

func TestPostgresResumableMigrationBoundsAttachmentAmplification(t *testing.T) {
	s, c, ctx, _, wid := jobTestFixture(t)
	if _, e := s.DB.Exec(ctx, "UPDATE migration_settings SET max_session_bytes=1048576"); e != nil {
		t.Fatal(e)
	}
	id := resumeTestUpload(t, c, wid, "amplification", map[string]string{"a.md": "[첨부](a.txt)", "b.md": "[첨부](a.txt)", "a.txt": strings.Repeat("a", 400<<10)})
	v := resumePrepare(t, s, c, id)
	if str(v, "status") != "failed" || !strings.Contains(str(v, "error"), "복제량") {
		t.Fatal(v)
	}
	var count int
	s.DB.QueryRow(ctx, "SELECT count(*) FROM attachments").Scan(&count)
	if count != 0 {
		t.Fatal("over-budget attachment was published")
	}
}

func TestPostgresResumableMigrationExcludedLinkClosureAndExpiry(t *testing.T) {
	s, c, ctx, _, wid := jobTestFixture(t)
	id := resumeTestUpload(t, c, wid, "link-cycle", map[string]string{"a.md": "[[b]]", "b.md": "[[a]]"})
	v := resumePrepare(t, s, c, id)
	items, e := s.migrationSessionIndex(ctx, id)
	if e != nil {
		t.Fatal(e)
	}
	for index, item := range items {
		c.request("PUT", "/api/v1/migrations/sessions/"+id+"/items/"+item.ID+"/review", map[string]any{"revision": v["revision"], "plan_hash": v["plan_hash"], "decision": "skip"}, 200)
		v = testJSONObject(t, c.request("GET", "/api/v1/migrations/sessions/"+id, nil, 200))
		if index == 0 {
			if number(v["report"].(map[string]any), "broken_dependencies", 0) != 1 {
				t.Fatal(v)
			}
			c.request("POST", "/api/v1/migrations/sessions/"+id+"/commit", map[string]any{"revision": v["revision"], "plan_hash": v["plan_hash"], "confirmation": "IMPORT"}, 409)
		}
	}
	if number(v["report"].(map[string]any), "broken_dependencies", 1) != 0 {
		t.Fatal(v)
	}
	if _, e = s.DB.Exec(ctx, "UPDATE migration_sessions SET expires_at=now()-interval '1 second' WHERE id=$1", id); e != nil {
		t.Fatal(e)
	}
	c.request("GET", "/api/v1/migrations/sessions/"+id+"/items/"+items[0].ID, nil, 410)
	c.request("GET", "/api/v1/migrations/sessions/"+id+"/items/"+items[0].ID+"/download", nil, 410)
	s.expireMigrationSessions(ctx)
	var remaining int
	if e = s.DB.QueryRow(ctx, "SELECT count(*) FROM migration_session_chunks WHERE item_id IN(SELECT id FROM migration_session_items WHERE session_id=$1)", id).Scan(&remaining); e != nil || remaining != 0 {
		t.Fatal("expired chunks remain", e)
	}
}
