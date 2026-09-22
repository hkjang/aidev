package server

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/hkjang/madi/internal/extract"
)

func extractionTestFixture(t *testing.T) (*Server, *integrationTestClient, context.Context, *Principal, string) {
	s, c, ctx, p, wid := jobTestFixture(t)
	runtime := s.jobRuntime()
	runtime.mu.RLock()
	_, exists := runtime.handlers["attachment.extract"]
	runtime.mu.RUnlock()
	if !exists {
		s.registerAttachmentExtraction()
	}
	return s, c, ctx, p, wid
}

func extractionTestPolicy(t *testing.T, c *integrationTestClient, enabled bool) {
	t.Helper()
	value := testJSONObject(t, c.request("GET", "/api/v1/admin/attachment-extraction/settings", nil, 200))
	data := value["data"].(map[string]any)
	data["enabled"] = enabled
	data["ocr_enabled"] = true
	c.request("PUT", "/api/v1/admin/attachment-extraction/settings", map[string]any{"revision": value["revision"], "data": data}, 200)
}

func extractionTestAttachment(t *testing.T, c *integrationTestClient, wid, text string) (string, string, string) {
	t.Helper()
	doc := testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "첨부 추출 검증", "markdown": "첨부 원본", "visibility": "private"}, 200))
	id := str(doc, "id")
	file := testJSONObject(t, storageMultipart(t, c, "/api/v1/attachments?document_id="+id, "local.txt", []byte(text), "", 200))
	return id, str(file, "id"), str(file, "checksum_sha256")
}

func TestPostgresAttachmentExtractionAtomicProjectionAndMask(t *testing.T) {
	s, c, ctx, p, wid := extractionTestFixture(t)
	doc, id, hash := extractionTestAttachment(t, c, wid, "첫 번째 근거\nuser@example.test\n세 번째 근거")
	body := map[string]any{"document_version": 1, "checksum": hash}
	c.request("POST", "/api/v1/attachments/"+id+"/extractions", body, 403)
	extractionTestPolicy(t, c, true)
	// Existing source bytes remain unchanged; only the new projection is masked.
	_, e := s.DB.Exec(ctx, `UPDATE protection_settings SET data=data||'{"enabled":true,"mode":"mask"}'::jsonb`)
	if e != nil {
		t.Fatal(e)
	}
	queued := testJSONObject(t, c.request("POST", "/api/v1/attachments/"+id+"/extractions", body, 202))
	runID := str(queued, "id")
	var count int
	s.DB.QueryRow(ctx, "SELECT count(*) FROM attachment_extraction_fragments").Scan(&count)
	if count != 0 {
		t.Fatal("partial projection before durable job")
	}
	drainJobs(t, s)
	detail := testJSONObject(t, c.request("GET", "/api/v1/attachment-extractions/"+runID, nil, 200))
	run := detail["extraction"].(map[string]any)
	if str(run, "status") != "ready" || !boolean(detail, "active") {
		t.Fatalf("not ready: %s", jsonValue(detail))
	}
	fragments := testJSONObject(t, c.request("GET", "/api/v1/attachment-extractions/"+runID+"/fragments", nil, 200))
	if strings.Contains(string(jsonValue(fragments)), "user@example.test") {
		t.Fatal("raw PII in derived projection")
	}
	if len(fragments["fragments"].([]any)) != 3 {
		t.Fatalf("fragment count %v", fragments)
	}
	if string(c.request("GET", "/api/v1/attachments/"+id, nil, 200)) != "첫 번째 근거\nuser@example.test\n세 번째 근거" {
		t.Fatal("source attachment silently rewritten")
	}
	var secrets, temp string
	s.DB.QueryRow(ctx, "SELECT session_hash||token_hash,temp_path FROM attachment_extractions WHERE id=$1", runID).Scan(&secrets, &temp)
	if secrets != "" || temp != "" {
		t.Fatal("completed extraction retained request/temp capability")
	}
	// Same source can be read after an unrelated parent edit using current CAS.
	_, e = s.DB.Exec(ctx, "UPDATE documents SET version=version+1 WHERE id=$1", doc)
	if e != nil {
		t.Fatal(e)
	}
	fragments = testJSONObject(t, c.request("GET", "/api/v1/attachment-extractions/"+runID+"/fragments", nil, 200))
	if number(fragments, "document_version", 0) != 2 {
		t.Fatal("parent current version missing")
	}
	other := *p
	other.ID = newID()
	if _, _, e = s.activeExtraction(ctx, &other, runID); e == nil {
		t.Fatal("foreign actor projection exposed")
	}
	extractionTestPolicy(t, c, false)
	c.request("GET", "/api/v1/attachment-extractions/"+runID+"/fragments", nil, 410)
}

func TestPostgresAttachmentExtractionCurrentGuardsAndLegacyChecksum(t *testing.T) {
	for _, mode := range []string{"source", "session", "policy", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			s, c, ctx, _, wid := extractionTestFixture(t)
			extractionTestPolicy(t, c, true)
			doc, id, hash := extractionTestAttachment(t, c, wid, "checkpoint source")
			if mode == "source" {
				if _, e := s.DB.Exec(ctx, "UPDATE attachments SET checksum_sha256='' WHERE id=$1", id); e != nil {
					t.Fatal(e)
				}
				hash = ""
			}
			queued := testJSONObject(t, c.request("POST", "/api/v1/attachments/"+id+"/extractions", map[string]any{"document_version": 1, "checksum": hash}, 202))
			run := str(queued, "id")
			if mode == "source" {
				if len(str(queued, "checksum")) != 64 {
					t.Fatal("legacy checksum not backfilled")
				}
				s.DB.Exec(ctx, "UPDATE documents SET version=version+1 WHERE id=$1", doc)
			}
			if mode == "session" {
				s.DB.Exec(ctx, "UPDATE sessions SET expires_at=clock_timestamp()-interval '1 second'")
			}
			if mode == "policy" {
				extractionTestPolicy(t, c, false)
			}
			if mode == "cancel" {
				c.request("DELETE", "/api/v1/attachment-extractions/"+run, nil, 200)
			}
			drainJobs(t, s)
			var count int
			s.DB.QueryRow(ctx, "SELECT count(*) FROM attachment_extraction_fragments WHERE extraction_id=$1", run).Scan(&count)
			if count != 0 {
				t.Fatal("changed/cancelled projection published")
			}
			var status string
			s.DB.QueryRow(ctx, "SELECT status FROM attachment_extractions WHERE id=$1", run).Scan(&status)
			if status == "ready" || status == "running" {
				t.Fatalf("stale run: %s", status)
			}
		})
	}
}

func TestAttachmentTextByteAndFragmentLimits(t *testing.T) {
	limits := extract.DefaultLimits()
	text := strings.Repeat("한", 6000) + "\n문단"
	r, e := extractionPlainText([]byte(text), limits)
	if e != nil || len(r.Fragments) != 3 {
		t.Fatalf("UTF8 split %v %+v", e, r)
	}
	for _, f := range r.Fragments {
		if len(f.Text) > 16384 {
			t.Fatal("unbounded fragment")
		}
	}
	limits.Fragments = 1
	if _, e = extractionPlainText([]byte("a\nb"), limits); e == nil {
		t.Fatal("fragment limit ignored")
	}
	if _, e = extractionPlainText([]byte{0xff}, extract.DefaultLimits()); e == nil {
		t.Fatal("invalid UTF8 accepted")
	}
	if e = cleanExtractionTemp("/"); e == nil {
		t.Fatal("broad cleanup path accepted")
	}
}

func TestPostgresAttachmentFinalLockWaitExpiry(t *testing.T) {
	s, c, ctx, p, wid := extractionTestFixture(t)
	extractionTestPolicy(t, c, true)
	_, id, hash := extractionTestAttachment(t, c, wid, "현재 세션 만료 경계")
	queued := testJSONObject(t, c.request("POST", "/api/v1/attachments/"+id+"/extractions", map[string]any{"document_version": 1, "checksum": hash}, 202))
	v, e := scanExtractionRun(s.DB.QueryRow(ctx, "SELECT "+extractionRunSelect+" FROM attachment_extractions WHERE id=$1", str(queued, "id")))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.DB.Exec(ctx, "UPDATE sessions SET expires_at=clock_timestamp()+interval '400 milliseconds' WHERE token_hash=$1", v.SessionHash); e != nil {
		t.Fatal(e)
	}
	block, e := s.DB.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer block.Rollback(ctx)
	if _, e = block.Exec(ctx, "SELECT id FROM attachment_extraction_settings WHERE id=1 FOR UPDATE"); e != nil {
		t.Fatal(e)
	}
	done := make(chan error, 1)
	go func() {
		tx, e := s.DB.Begin(ctx)
		if e != nil {
			done <- e
			return
		}
		defer tx.Rollback(ctx)
		done <- s.extractionGuardTx(ctx, tx, p, v)
	}()
	time.Sleep(700 * time.Millisecond)
	if e = block.Commit(ctx); e != nil {
		t.Fatal(e)
	}
	select {
	case e = <-done:
		if e == nil {
			t.Fatal("expired session survived final lock wait")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("final guard failed to finish")
	}
}
