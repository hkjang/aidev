package server

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPostgresAttachmentStreamRetractsChangedSource(t *testing.T) {
	for _, reason := range []string{"policy", "checksum"} {
		t.Run(reason, func(t *testing.T) {
			s, c, ctx, p, wid := extractionTestFixture(t)
			extractionTestPolicy(t, c, true)
			doc, id, hash := extractionTestAttachment(t, c, wid, "명시 선택된 첨부 근거")
			runID := str(testJSONObject(t, c.request("POST", "/api/v1/attachments/"+id+"/extractions", map[string]any{"document_version": 1, "checksum": hash}, 202)), "id")
			drainJobs(t, s)
			v, version, err := s.activeExtraction(ctx, p, runID)
			if err != nil {
				t.Fatal(err)
			}
			f, err := scanAttachmentFragment(s.DB.QueryRow(ctx, "SELECT id::text,extraction_id::text,ordinal,text,content_hash,position FROM attachment_extraction_fragments WHERE extraction_id=$1 ORDER BY ordinal LIMIT 1", runID))
			if err != nil {
				t.Fatal(err)
			}
			release := make(chan struct{})
			closed := false
			provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"FIRST_AUTHORIZED\"}}]}\n\n")
				w.(http.Flusher).Flush()
				select {
				case <-release:
				case <-r.Context().Done():
					return
				}
				fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"MUST_NOT_ESCAPE\"}}]}\n\ndata: [DONE]\n\n")
			}))
			defer func() {
				if !closed {
					close(release)
				}
				provider.Close()
			}()
			c.request("PUT", "/api/v1/admin/settings", map[string]any{"ai_enabled": true, "ai_base_url": provider.URL, "ai_model": "attachment-retract"}, 200)
			info := testJSONObject(t, c.request("GET", "/api/v1/documents/"+doc+"/ai-selection", nil, 200))
			in := map[string]any{"fragment_id": f.ID, "document_version": version, "revision": v.Revision, "start_byte": 0, "end_byte": len(f.Text), "hash": digest(f.Text), "provider_fingerprint": str(info["provider"].(map[string]any), "fingerprint"), "consent": true}
			req, _ := http.NewRequest("POST", c.base+"/api/v1/attachment-extractions/"+runID+"/ai", bytes.NewReader(jsonValue(in)))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Madi-Request", "1")
			res, err := c.client.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer res.Body.Close()
			if res.StatusCode != 200 {
				raw, _ := io.ReadAll(res.Body)
				t.Fatal(res.StatusCode, string(raw))
			}
			reader := bufio.NewReader(res.Body)
			for {
				line, err := reader.ReadString('\n')
				if err != nil {
					t.Fatal(err)
				}
				if strings.Contains(line, "FIRST_AUTHORIZED") {
					break
				}
			}
			if reason == "policy" {
				extractionTestPolicy(t, c, false)
			} else if _, err = s.DB.Exec(ctx, "UPDATE attachments SET checksum_sha256=$2 WHERE id=$1", id, strings.Repeat("1", 64)); err != nil {
				t.Fatal(err)
			}
			close(release)
			closed = true
			rest, err := io.ReadAll(reader)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(rest), "MUST_NOT_ESCAPE") || !strings.Contains(string(rest), `"retract":true`) {
				t.Fatal("attachment stream was not retracted", string(rest))
			}
		})
	}
}

func TestPostgresAttachmentSearchCitationAndExplicitAI(t *testing.T) {
	s, c, ctx, p, wid := extractionTestFixture(t)
	extractionTestPolicy(t, c, true)
	doc, id, hash := extractionTestAttachment(t, c, wid, "쿠버네티스 선택 근거 😀\nUNSELECTED_ATTACHMENT_TEXT")
	queued := testJSONObject(t, c.request("POST", "/api/v1/attachments/"+id+"/extractions", map[string]any{"document_version": 1, "checksum": hash}, 202))
	runID := str(queued, "id")
	drainJobs(t, s)
	c.request("PUT", "/api/v1/workspaces/"+wid+"/search-dictionary", map[string]any{"revision": 0, "confirm_shared": true, "entries": []searchDictionaryEntry{{Canonical: "쿠버네티스", Aliases: []string{"k8s"}}}}, 200)
	result := testJSONObject(t, c.request("GET", "/api/v1/search?workspace_id="+wid+"&type=file&q=k8s", nil, 200))
	rows := result["results"].([]any)
	if len(rows) != 1 {
		t.Fatalf("dictionary attachment search: %s", jsonValue(result))
	}
	hit := rows[0].(map[string]any)
	if !strings.Contains(str(hit, "url"), "/app/attachments/"+id) || !strings.Contains(str(hit, "snippet"), "쿠버네티스") {
		t.Fatal("missing attachment source position", hit)
	}
	fragments := testJSONObject(t, c.request("GET", "/api/v1/attachment-extractions/"+runID+"/fragments", nil, 200))
	first := fragments["fragments"].([]any)[0].(map[string]any)
	v, version, e := s.activeExtraction(ctx, p, runID)
	if e != nil {
		t.Fatal(e)
	}
	var f attachmentFragment
	if json.Unmarshal(jsonValue(first), &f) != nil {
		t.Fatal("fragment decode")
	}
	src, e := attachmentAISource(v, version, f, 0, len(f.Text))
	if e != nil {
		t.Fatal(e)
	}
	value := testJSONObject(t, c.request("GET", "/api/v1"+src.CitationURL, nil, 200))
	if str(value, "text") != f.Text {
		t.Fatal("wrong quote bytes")
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	history, e := s.attachmentCitationTx(ctx, tx, p, src, false)
	tx.Rollback(ctx)
	if e != nil || !history.Fresh {
		t.Fatal("initial archival freshness", e)
	}
	requests := make(chan map[string]any, 4)
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var input map[string]any
		if json.NewDecoder(r.Body).Decode(&input) != nil {
			w.WriteHeader(400)
			return
		}
		requests <- input
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"근거 요약 [1]\"}}]}\n\ndata: [DONE]\n\n")
	}))
	defer provider.Close()
	c.request("PUT", "/api/v1/admin/settings", map[string]any{"ai_enabled": true, "ai_base_url": provider.URL, "ai_model": "attachment-test", "ai_max_tokens": 262144}, 200)
	info := testJSONObject(t, c.request("GET", "/api/v1/documents/"+doc+"/ai-selection", nil, 200))
	fingerprint := str(info["provider"].(map[string]any), "fingerprint")
	input := map[string]any{"fragment_id": f.ID, "document_version": version, "revision": v.Revision, "start_byte": 0, "end_byte": len(f.Text), "hash": digest(f.Text), "provider_fingerprint": fingerprint, "consent": true, "prompt": "이 근거만 요약"}
	output := string(c.request("POST", "/api/v1/attachment-extractions/"+runID+"/ai", input, 200))
	if !strings.Contains(output, `"proposal":`) || strings.Contains(output, `"retract":`) {
		t.Fatal(output)
	}
	upstream := <-requests
	encoded := string(jsonValue(upstream))
	if upstream["stream"] != true || upstream["max_tokens"] != float64(262144) || !strings.Contains(encoded, f.Text) || strings.Contains(encoded, "UNSELECTED_ATTACHMENT_TEXT") || strings.Contains(encoded, "첨부 원본") || strings.Contains(encoded, "local.txt") {
		t.Fatal("AI attachment scope violated", encoded)
	}
	input["consent"] = false
	c.request("POST", "/api/v1/attachment-extractions/"+runID+"/ai", input, 400)
	input["consent"] = true
	input["hash"] = "bad"
	c.request("POST", "/api/v1/attachment-extractions/"+runID+"/ai", input, 409)
	if len(requests) != 0 {
		t.Fatal("invalid consent/hash reached provider")
	}
	// Current citation checks reject changed versions; archival evidence keeps
	// the authenticated old text and reports changed freshness instead.
	if _, e = s.DB.Exec(ctx, "UPDATE documents SET version=version+1 WHERE id=$1", doc); e != nil {
		t.Fatal(e)
	}
	c.request("GET", "/api/v1"+src.CitationURL, nil, 409)
	tx, e = s.DB.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	history, e = s.attachmentCitationTx(ctx, tx, p, src, false)
	tx.Rollback(ctx)
	if e != nil || history.Fresh || history.Text != f.Text {
		t.Fatal("archival parent edit", history, e)
	}
	if _, e = s.DB.Exec(ctx, "UPDATE attachments SET checksum_sha256=$2 WHERE id=$1", id, strings.Repeat("a", 64)); e != nil {
		t.Fatal(e)
	}
	tx, e = s.DB.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	history, e = s.attachmentCitationTx(ctx, tx, p, src, false)
	tx.Rollback(ctx)
	if e != nil || history.Fresh || history.Text != "" {
		t.Fatal("archival source changed", history, e)
	}
	result = testJSONObject(t, c.request("GET", "/api/v1/search?workspace_id="+wid+"&type=file&q=k8s", nil, 200))
	if len(result["results"].([]any)) != 0 {
		t.Fatal("stale attachment projection found")
	}
}
