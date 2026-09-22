package server

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPostgresRAGGrantRevocationRetractsChatStream(t *testing.T) {
	for _, mode := range []string{"semantic", "hybrid"} {
		t.Run(mode, func(t *testing.T) {
			s, c, wid, _ := ragTestSetup(t)
			provider, _, _ := ragTestProvider(t, nil)
			ragTestConfigure(t, c, provider.URL)
			release := make(chan struct{})
			closed := false
			chat := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"AUTHORIZED_FIRST\"}}]}\n\n")
				w.(http.Flusher).Flush()
				select {
				case <-release:
				case <-r.Context().Done():
					return
				}
				fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"REVOKED_SHOULD_NOT_ESCAPE\"}}]}\n\ndata: [DONE]\n\n")
			}))
			defer func() {
				if !closed {
					close(release)
				}
				chat.Close()
			}()
			c.request("PUT", "/api/v1/admin/settings", map[string]any{"ai_enabled": true, "ai_base_url": chat.URL + "/v1", "ai_model": "test-chat", "rag_search_mode": mode}, 200)
			id := str(testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "GPU 인용 원문", "markdown": "# GPU 정책\n\n이 원문은 인용 검증을 위한 내용입니다."}, 200)), "id")
			consent := ragTestConsent(t, c, id, 1, false, true)
			ragTestRun(t, s, str(consent, "job_id"))
			req, _ := http.NewRequest("POST", c.base+"/api/v1/ai/chat", bytes.NewReader(jsonValue(map[string]any{"document_id": id, "prompt": "GPU 정책"})))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Madi-Request", "1")
			res, e := c.client.Do(req)
			if e != nil {
				t.Fatal(e)
			}
			defer res.Body.Close()
			if res.StatusCode != 200 {
				body, _ := io.ReadAll(res.Body)
				t.Fatal(res.StatusCode, string(body))
			}
			reader := bufio.NewReader(res.Body)
			var before strings.Builder
			for {
				line, e := reader.ReadString('\n')
				if e != nil {
					t.Fatal(e)
				}
				before.WriteString(line)
				if strings.Contains(line, "AUTHORIZED_FIRST") {
					break
				}
			}
			if !strings.Contains(before.String(), `"retrieval"`) || !strings.Contains(before.String(), `"citation_id"`) || strings.Contains(before.String(), `"RAGGrantID"`) {
				t.Fatal("missing diagnostics or hidden metadata exposed", before.String())
			}
			c.request("DELETE", "/api/v1/documents/"+id+"/rag-index", map[string]any{"grant_revision": consent["grant_revision"]}, 200)
			close(release)
			closed = true
			rest, e := io.ReadAll(reader)
			if e != nil {
				t.Fatal(e)
			}
			if strings.Contains(string(rest), "REVOKED_SHOULD_NOT_ESCAPE") || !strings.Contains(string(rest), `"retract":true`) {
				t.Fatal("grant revoke did not retract semantic/rerank stream", string(rest))
			}
		})
	}
}
