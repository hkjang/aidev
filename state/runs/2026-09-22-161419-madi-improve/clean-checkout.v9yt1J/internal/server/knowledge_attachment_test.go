package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPostgresKnowledgeAttachmentEvidenceAndExplicitPackage(t *testing.T) {
	s, c, ctx, p, wid := extractionTestFixture(t)
	extractionTestPolicy(t, c, true)
	did, aid, hash := extractionTestAttachment(t, c, wid, "ATTACHMENT_KNOWLEDGE_SENTINEL 페이지 근거입니다.\n다음 문단")
	queued := testJSONObject(t, c.request("POST", "/api/v1/attachments/"+aid+"/extractions", map[string]any{"document_version": 1, "checksum": hash}, 202))
	eid := str(queued, "id")
	drainJobs(t, s)
	v, version, e := s.activeExtraction(ctx, p, eid)
	if e != nil {
		t.Fatal(e)
	}
	f, e := scanAttachmentFragment(s.DB.QueryRow(ctx, `SELECT id::text,extraction_id::text,ordinal,text,content_hash,position FROM attachment_extraction_fragments WHERE extraction_id=$1 ORDER BY ordinal LIMIT 1`, eid))
	if e != nil {
		t.Fatal(e)
	}
	src, e := attachmentAISource(v, version, f, 0, len(f.Text))
	if e != nil {
		t.Fatal(e)
	}
	c.request("PUT", "/api/v1/admin/evidence-policy", map[string]any{"version": 1, "enabled": true, "retention_days": 90}, 200)
	cfg, e := s.effectiveSettings(ctx, wid)
	if e != nil {
		t.Fatal(e)
	}
	ticket, e := s.sealAIHistory(p, wid, "첨부 근거 확인", "근거를 확인했습니다", "ask", []aiSource{src}, cfg)
	if e != nil {
		t.Fatal(e)
	}
	evidence := str(testJSONObject(t, c.request("POST", "/api/v1/ai/evidence", map[string]any{"ticket": ticket, "consent": true}, 201)), "id")
	conversation := str(testJSONObject(t, c.request("POST", "/api/v1/ai/conversations", map[string]any{"ticket": ticket, "consent": true}, 201)), "id")
	historyRequest := httptest.NewRequest(http.MethodPost, "/", nil).WithContext(context.WithValue(ctx, principalKey, p))
	history, e := s.loadAIHistoryContext(historyRequest, conversation, 1, wid, cfg)
	if e != nil || len(history.Sources) != 1 || history.Sources[0].Markdown != f.Text {
		t.Fatal("attachment history interpreted parent source", history, e)
	}
	getQuote := func() map[string]any {
		return testJSONObject(t, c.request("GET", "/api/v1/ai/evidence/"+evidence, nil, 200))["sources"].([]any)[0].(map[string]any)
	}
	quote := getQuote()
	if str(quote, "text") != f.Text || str(quote, "freshness") != "current" || !boolean(quote, "same_span") {
		t.Fatal("attachment offsets were interpreted as parent Markdown", quote)
	}
	request := map[string]any{"workspace_id": wid, "purpose": "첨부 근거 패키지", "model": "local-fixture", "token_budget": 16384, "counter": "estimate", "consent": true, "documents": []any{}}
	c.request("POST", "/api/v1/knowledge/packages", request, 400) // Never implicitly include attachments.
	request["attachments"] = []packageAttachmentInput{{Source: src, Mandatory: true, Reason: "위치를 직접 확인함"}}
	packageID := str(testJSONObject(t, c.request("POST", "/api/v1/knowledge/packages", request, 201)), "id")
	readPackage := func() map[string]any {
		return testJSONObject(t, c.request("GET", "/api/v1/knowledge/packages/"+packageID, nil, 200))
	}
	pkg := readPackage()
	if boolean(pkg, "stale") || !strings.Contains(string(jsonValue(pkg)), "ATTACHMENT_KNOWLEDGE_SENTINEL") || !strings.Contains(string(jsonValue(pkg)), "extracted_text") || !strings.Contains(string(jsonValue(pkg)), "바이트 범위는 해당 조각 기준") {
		t.Fatal("missing explicit extraction provenance", pkg)
	}
	c.request("POST", "/api/v1/knowledge/packages/"+packageID+"/export", map[string]any{"consent": true, "target": "사내 검토"}, 200)
	var sealed string
	if e = s.DB.QueryRow(ctx, `SELECT ciphertext FROM knowledge_packages WHERE id=$1`, packageID).Scan(&sealed); e != nil || strings.Contains(sealed, "ATTACHMENT_KNOWLEDGE_SENTINEL") {
		t.Fatal("raw attachment persisted", e)
	}
	bad := src
	bad.ContentHash = digest("wrong")
	request["attachments"] = []packageAttachmentInput{{Source: bad}}
	c.request("POST", "/api/v1/knowledge/packages", request, 409)
	// Malicious presentation fields are not carried through a valid source.
	bad = src
	bad.Title, bad.URL, bad.CitationURL = "FORGED-TITLE", "https://tracking.invalid", "/admin"
	request["attachments"] = []packageAttachmentInput{{Source: bad}}
	id := str(testJSONObject(t, c.request("POST", "/api/v1/knowledge/packages", request, 201)), "id")
	if raw := c.request("GET", "/api/v1/knowledge/packages/"+id, nil, 200); bytes.Contains(raw, []byte("FORGED-TITLE")) || bytes.Contains(raw, []byte("tracking.invalid")) {
		t.Fatal("caller citation presentation trusted")
	}
	// Disabling a derived index stops new AI transmission/export, but does not
	// invent a new 'current' quote or erase permitted historical evidence.
	extractionTestPolicy(t, c, false)
	if _, e = s.loadAIHistoryContext(historyRequest, conversation, 1, wid, cfg); e == nil {
		t.Fatal("stale attachment history resent")
	}
	quote = getQuote()
	if str(quote, "text") != f.Text || str(quote, "current_text") != "" || str(quote, "freshness") != "changed" || str(quote, "integrity") != "match" {
		t.Fatal(quote)
	}
	if !boolean(readPackage(), "stale") {
		t.Fatal("obsolete extraction reported current")
	}
	c.request("POST", "/api/v1/knowledge/packages/"+packageID+"/export", map[string]any{"consent": true, "target": "사내 검토"}, 409)
	r := httptest.NewRequest(http.MethodPost, "/", nil).WithContext(context.WithValue(ctx, principalKey, p))
	if e = s.validateAIStream(r, p, wid, []aiSource{src}, cfg); e == nil {
		t.Fatal("obsolete attachment was allowed to stream")
	}
	// Attachment deletion removes derivative copies, not unrelated document
	// evidence. List endpoints must not expose now inaccessible snapshot IDs.
	if _, e = s.DB.Exec(ctx, `DELETE FROM attachments WHERE id=$1 AND document_id=$2`, aid, did); e != nil {
		t.Fatal(e)
	}
	c.request("GET", "/api/v1/ai/evidence/"+evidence, nil, 404)
	c.request("GET", "/api/v1/knowledge/packages/"+packageID, nil, 404)
	c.request("GET", "/api/v1/ai/conversations/"+conversation, nil, 404)
	for _, path := range []string{"/api/v1/ai/evidence?workspace_id=", "/api/v1/knowledge/packages?workspace_id="} {
		var items []any
		if e = json.Unmarshal(c.request("GET", path+wid, nil, 200), &items); e != nil || len(items) != 0 {
			t.Fatal("deleted attachment snapshot in list", items, e)
		}
	}
}
