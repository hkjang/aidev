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
	"sync/atomic"
	"testing"
	"time"
)

const naturalSearchFixture = `{"explanation":"GPU 장애 문서를 수정 날짜로 찾습니다","plan":{"q":"GPU 장애","type":"document","from":"2026-08-01","to":"2026-08-31","tag":"","status":"published","has_attachment":false,"sort":"newest"}}`

func TestNaturalSearchClosedSchema(t *testing.T) {
	v, e := parseNaturalSearchProposal(naturalSearchFixture)
	if e != nil || !strings.Contains(v.Plan.query(), "from=2026-08-01") {
		t.Fatal(v, e)
	}
	for _, value := range []string{strings.Replace(naturalSearchFixture, `"q":"GPU 장애"`, `"sql":"select * from users"`, 1), strings.Replace(naturalSearchFixture, `"type":"document"`, `"type":"all-tables"`, 1), strings.Replace(naturalSearchFixture, "2026-08-31", "2026-02-30", 1), strings.Replace(naturalSearchFixture, "2026-08-01", "2026-09-01", 1), naturalSearchFixture + `{}`, strings.Replace(naturalSearchFixture, `"q":"GPU 장애"`, `"q":"`+strings.Repeat("x", 501)+`"`, 1)} {
		if _, e := parseNaturalSearchProposal(value); e == nil {
			t.Fatal("invalid plan accepted")
		}
	}
}

func TestPostgresNaturalSearchQuietStreamRevocation(t *testing.T) {
	_, ts := integrationTestServer(t)
	c := newIntegrationTestClient(t, ts.URL)
	c.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200)
	wid := str(testJSONObject(t, c.request("POST", "/api/v1/workspaces", map[string]any{"name": "Quiet proposal"}, 200)), "id")
	started, stopped := make(chan struct{}), make(chan struct{})
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"{\"}}]}\n\n")
		w.(http.Flusher).Flush()
		<-r.Context().Done()
		close(stopped)
	}))
	defer provider.Close()
	c.request("PUT", "/api/v1/admin/settings", map[string]any{"ai_enabled": true, "ai_base_url": provider.URL, "ai_model": "quiet"}, 200)
	done := make(chan []byte, 1)
	go func() {
		req, _ := http.NewRequest("POST", ts.URL+"/api/v1/search/ai/proposal", bytes.NewReader(jsonValue(map[string]any{"workspace_id": wid, "prompt": "문서 찾기", "consent": true})))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Madi-Request", "1")
		response, e := c.client.Do(req)
		if e != nil {
			done <- []byte(e.Error())
			return
		}
		defer response.Body.Close()
		reader := bufio.NewReader(response.Body)
		var result strings.Builder
		ready := false
		for {
			line, e := reader.ReadString('\n')
			result.WriteString(line)
			if !ready && strings.Contains(line, `"text"`) {
				ready = true
				close(started)
			}
			if e != nil {
				if e != io.EOF {
					result.WriteString(e.Error())
				}
				break
			}
		}
		done <- []byte(result.String())
	}()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("no upstream request")
	}
	// Wait for the first client delta, not just an upstream write, before revoking.
	c.request("PUT", "/api/v1/admin/settings", map[string]any{"ai_enabled": false}, 200)
	select {
	case body := <-done:
		if !bytes.Contains(body, []byte(`"retract":true`)) || bytes.Contains(body, []byte(`"proposal":`)) {
			t.Fatal("stale quiet proposal", string(body))
		}
	case <-time.After(4 * time.Second):
		t.Fatal("quiet stream not revoked")
	}
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("upstream not cancelled")
	}
}

func TestPostgresSelectedAISourcesVersionACLAndNoExpansion(t *testing.T) {
	s, ts := integrationTestServer(t)
	c := newIntegrationTestClient(t, ts.URL)
	c.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200)
	wid := str(testJSONObject(t, c.request("POST", "/api/v1/workspaces", map[string]any{"name": "Selected AI sources"}, 200)), "id")
	selected := []selectedAIDocument{}
	for _, name := range []string{"SELECTED_ONE", "SELECTED_TWO", "EXCLUDED_THIRD"} {
		d := testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": name, "markdown": "# " + name + "\n\nGPU test content."}, 200))
		if len(selected) < 2 {
			selected = append(selected, selectedAIDocument{str(d, "id"), 1})
		}
	}
	var requests atomic.Int32
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		_ = json.NewDecoder(r.Body).Decode(&payload)
		requests.Add(1)
		raw := string(jsonValue(payload))
		if !strings.Contains(raw, "SELECTED_ONE") || !strings.Contains(raw, "SELECTED_TWO") || strings.Contains(raw, "EXCLUDED_THIRD") {
			t.Error("selection expanded or omitted")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"요약 [1] [2]\"}}]}\n\ndata: [DONE]\n\n")
	}))
	defer provider.Close()
	c.request("PUT", "/api/v1/admin/settings", map[string]any{"ai_enabled": true, "ai_base_url": provider.URL, "ai_model": "selection"}, 200)
	in := map[string]any{"workspace_id": wid, "prompt": "GPU 요약", "selected_documents": selected, "action": "summarize"}
	body := c.request("POST", "/api/v1/ai/chat", in, 200)
	if !bytes.Contains(body, []byte(`"mode":"selected"`)) || !bytes.Contains(body, []byte(`"content_hash"`)) {
		t.Fatal(string(body))
	}
	c.request("PUT", "/api/v1/documents/"+selected[0].ID, map[string]any{"version": 1, "markdown": "updated source"}, 200)
	c.request("POST", "/api/v1/ai/chat", in, 403)
	if requests.Load() != 1 {
		t.Fatal("stale selection sent to AI")
	}
	if _, e := s.DB.Exec(t.Context(), "UPDATE documents SET deleted_at=now() WHERE id=$1", selected[1].ID); e != nil {
		t.Fatal(e)
	}
	selected[0].Version = 2
	c.request("POST", "/api/v1/ai/chat", in, 403)
	if requests.Load() != 1 {
		t.Fatal("deleted selection sent to AI")
	}
}
func TestPostgresNaturalSearchStreamingConsentPrivacyAndClosedPlan(t *testing.T) {
	s, ts := integrationTestServer(t)
	c := newIntegrationTestClient(t, ts.URL)
	c.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200)
	wid := str(testJSONObject(t, c.request("POST", "/api/v1/workspaces", map[string]any{"name": "Natural search scope"}, 200)), "id")
	c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "PRIVATE_TITLE_NEVER_SENT", "markdown": "PRIVATE_BODY_NEVER_SENT", "visibility": "private"}, 200)
	var requests atomic.Int32
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var in map[string]any
		if json.NewDecoder(r.Body).Decode(&in) != nil {
			w.WriteHeader(400)
			return
		}
		requests.Add(1)
		if !boolean(in, "stream") || number(in, "max_tokens", 0) != 262144 || strings.Contains(string(jsonValue(in)), "PRIVATE_TITLE_NEVER_SENT") || strings.Contains(string(jsonValue(in)), "PRIVATE_BODY_NEVER_SENT") {
			t.Error("wrong AI payload or leaked document", in)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		for _, part := range []string{naturalSearchFixture[:40], naturalSearchFixture[40:]} {
			fmt.Fprintf(w, "data: %s\n\n", jsonValue(map[string]any{"choices": []any{map[string]any{"delta": map[string]any{"content": part}}}}))
			w.(http.Flusher).Flush()
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer provider.Close()
	c.request("PUT", "/api/v1/admin/settings", map[string]any{"ai_enabled": true, "ai_base_url": provider.URL, "ai_model": "natural", "ai_max_tokens": 262144}, 200)
	in := map[string]any{"workspace_id": wid, "prompt": "SEARCH_PRIVATE_PROMPT 지난달 GPU 장애 문서를 찾아줘"}
	c.request("POST", "/api/v1/search/ai/proposal", in, 400)
	if requests.Load() != 0 {
		t.Fatal("model called without consent")
	}
	in["consent"] = true
	body := c.request("POST", "/api/v1/search/ai/proposal", in, 200)
	if !bytes.Contains(body, []byte(`"proposal"`)) || !bytes.Contains(body, []byte(`"automatic_apply":false`)) || !bytes.Contains(body, []byte("data: [DONE]")) {
		t.Fatal(string(body))
	}
	var count int
	if e := s.DB.QueryRow(t.Context(), `SELECT count(*) FROM audit_logs WHERE details::text LIKE '%SEARCH_PRIVATE_PROMPT%' OR action='SEARCH'`).Scan(&count); e != nil || count != 0 {
		t.Fatal("query audit leak or automatic search", count, e)
	}
	token := testJSONObject(t, c.request("POST", "/api/v1/keys", map[string]any{"name": "natural denied", "workspace_id": wid, "scopes": []string{"ai:execute", "search:read"}}, 201))
	k := newIntegrationTestClient(t, ts.URL)
	k.token = str(token, "token")
	k.request("POST", "/api/v1/search/ai/proposal", in, 403)
}
