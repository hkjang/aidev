package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAISelectionUTF8Bounds(t *testing.T) {
	md := "앞😀선택한 문장\n뒤"
	start := strings.Index(md, "선택")
	end := strings.Index(md, "\n")
	if !validAISelection(md, start, end, md[start:end]) {
		t.Fatal("valid Unicode selection rejected")
	}
	for _, test := range []struct {
		start, end int
		text       string
	}{{-1, end, ""}, {0, len(md) + 1, md}, {start + 1, end, md[start+1 : end]}, {start, end - 1, md[start : end-1]}, {start, start, ""}, {start, end, "다른 원문"}} {
		if validAISelection(md, test.start, test.end, test.text) {
			t.Fatal("invalid selection accepted", test.start, test.end)
		}
	}
	if validAISelection(strings.Repeat("a", aiSelectionMaxBytes+1), 0, aiSelectionMaxBytes+1, strings.Repeat("a", aiSelectionMaxBytes+1)) {
		t.Fatal("oversized selection accepted")
	}
}

func TestPostgresAISelectionExactBoundaryAndCAS(t *testing.T) {
	s, ts := integrationTestServer(t)
	c := newIntegrationTestClient(t, ts.URL)
	c.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200)
	w := testJSONObject(t, c.request("POST", "/api/v1/workspaces", map[string]any{"name": "선택 AI 격리"}, 200))
	wid := str(w, "id")
	requests := make(chan map[string]any, 10)
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var data map[string]any
		if json.NewDecoder(r.Body).Decode(&data) != nil {
			w.WriteHeader(400)
			return
		}
		requests <- data
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"개선된 한글 문장 😀\"}}]}\n\ndata: [DONE]\n\n")
	}))
	defer provider.Close()
	c.request("PUT", "/api/v1/admin/settings", map[string]any{"ai_enabled": true, "ai_base_url": provider.URL + "/v1?secret=not-in-display", "ai_model": "selection", "ai_max_tokens": 262144}, 200)
	md := "---\nprivate: 앞부분은 절대로 전송하지 않음\n---\n\n선택한 원문 😀\n\n뒷부분 비밀 MARKER_NOT_SELECTED"
	d := testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "TITLE_NOT_TRANSMITTED", "markdown": md, "visibility": "private", "tags": []string{"TAG_NOT_TRANSMITTED"}}, 200))
	id := str(d, "id")
	path := "/api/v1/documents/" + id + "/ai-selection"
	meta := testJSONObject(t, c.request("GET", path, nil, 200))
	providerMeta := meta["provider"].(map[string]any)
	if strings.Contains(str(providerMeta, "base_url"), "secret") {
		t.Fatal("provider display leaked query")
	}
	selected := "선택한 원문 😀"
	start := strings.Index(md, selected)
	input := map[string]any{"expected_version": number(d, "version", 0), "start_byte": start, "end_byte": start + len(selected), "selected_text": selected, "provider_fingerprint": str(providerMeta, "fingerprint"), "consent": true, "action": "rewrite", "prompt": "명료하게"}
	response := string(c.request("POST", path, input, 200))
	if !strings.Contains(response, `"proposal":`) || !strings.Contains(response, "개선된 한글 문장 😀") || strings.Contains(response, `"retract"`) {
		t.Fatal(response)
	}
	upstream := <-requests
	var proposal map[string]any
	for line := range strings.SplitSeq(response, "\n") {
		if strings.HasPrefix(line, "data: ") && strings.Contains(line, `"proposal":`) {
			var event map[string]json.RawMessage
			if json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &event) == nil {
				_ = json.Unmarshal(event["proposal"], &proposal)
			}
		}
	}
	if str(proposal, "draft_ticket") == "" {
		t.Fatal("missing scoped private draft ticket")
	}
	copyInput := map[string]any{"ticket": str(proposal, "draft_ticket"), "title": "개인 AI 선택 제안", "markdown": str(proposal, "markdown")}
	copy := testJSONObject(t, c.request("POST", "/api/v1/ai/selection-drafts", copyInput, 200))
	if str(copy, "visibility") != "private" || str(copy, "markdown") != str(proposal, "markdown") {
		t.Fatal("copy not private/exact")
	}
	copyInput["markdown"] = "변조한 내용"
	c.request("POST", "/api/v1/ai/selection-drafts", copyInput, 409)
	copyInput["markdown"] = str(proposal, "markdown")
	encoded := string(jsonValue(upstream))
	for _, forbidden := range []string{"MARKER_NOT_SELECTED", "TITLE_NOT_TRANSMITTED", "TAG_NOT_TRANSMITTED", "앞부분은 절대로"} {
		if strings.Contains(encoded, forbidden) {
			t.Fatal("selection request exposed unselected data", forbidden)
		}
	}
	if upstream["stream"] != true || upstream["max_tokens"] != float64(262144) || !strings.Contains(encoded, selected) {
		t.Fatal("stream/boundary/token contract changed", upstream)
	}
	unchanged := testJSONObject(t, c.request("GET", "/api/v1/documents/"+id, nil, 200))
	if str(unchanged, "markdown") != md || number(unchanged, "version", 0) != number(d, "version", 0) {
		t.Fatal("AI proposal modified source")
	}
	input["consent"] = false
	c.request("POST", path, input, 400)
	input["consent"] = true
	input["selected_text"] = "다른 원문"
	c.request("POST", path, input, 400)
	input["selected_text"] = selected
	input["provider_fingerprint"] = "stale"
	c.request("POST", path, input, 409)
	input["provider_fingerprint"] = str(providerMeta, "fingerprint")
	input["action"] = "execute"
	c.request("POST", path, input, 400)
	input["action"] = "rewrite"
	if len(requests) != 0 {
		t.Fatal("invalid request reached provider")
	}
	c.request("PUT", "/api/v1/documents/"+id, map[string]any{"version": number(d, "version", 0), "markdown": md + "\n다른 변경"}, 200)
	c.request("POST", path, input, 409)
	c.request("POST", "/api/v1/ai/selection-drafts", copyInput, 409)
	other := testJSONObject(t, c.request("POST", "/api/v1/admin/users", map[string]any{"email": "selection-viewer@example.test", "name": "조회자", "password": "Selection-Viewer-Password-2026!", "role": "viewer"}, 200))
	c.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": str(other, "email"), "role": "viewer"}, 200)
	viewer := newIntegrationTestClient(t, ts.URL)
	viewer.request("POST", "/api/v1/auth/login", map[string]any{"email": "selection-viewer@example.test", "password": "Selection-Viewer-Password-2026!"}, 200)
	viewer.request("GET", path, nil, 403)
	viewer.request("POST", path, input, 403)
	var count int
	if err := s.DB.QueryRow(t.Context(), "SELECT count(*) FROM document_versions WHERE document_id=$1", id).Scan(&count); err != nil || count != 2 {
		t.Fatal("unexpected document revisions", count, err)
	}
}
