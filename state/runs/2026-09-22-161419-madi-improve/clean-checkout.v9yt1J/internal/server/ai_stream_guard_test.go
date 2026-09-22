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

func TestPostgresAIStreamRevocationStopsNewContent(t *testing.T) {
	for _, reason := range []string{"document_acl", "document_version", "key_scope", "session"} {
		t.Run(reason, func(t *testing.T) {
			_, ts := integrationTestServer(t)
			admin := newIntegrationTestClient(t, ts.URL)
			admin.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200)
			wid := str(testJSONObject(t, admin.request("POST", "/api/v1/workspaces", map[string]any{"name": "AI 권한회수"}, 200)), "id")
			u := testJSONObject(t, admin.request("POST", "/api/v1/admin/users", map[string]any{"email": "stream@example.test", "name": "AI 사용자", "password": "Streaming-password-2026!", "role": "editor"}, 200))
			admin.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": u["email"], "role": "editor"}, 200)
			d := testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "스트림 참조", "markdown": "authorized context"}, 200))
			member := newIntegrationTestClient(t, ts.URL)
			member.request("POST", "/api/v1/auth/login", map[string]any{"email": u["email"], "password": "Streaming-password-2026!"}, 200)
			keyID := ""
			if reason == "key_scope" {
				k := testJSONObject(t, member.request("POST", "/api/v1/keys", map[string]any{"name": "AI 스트림", "workspace_id": wid, "scopes": []string{"document:read", "ai:execute"}}, 201))
				member.token = str(k, "token")
				keyID = str(k["key"].(map[string]any), "id")
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
				fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"SHOULD_NOT_ESCAPE\"}}]}\n\ndata: [DONE]\n\n")
			}))
			defer func() {
				if !closed {
					close(release)
				}
				provider.Close()
			}()
			admin.request("PUT", "/api/v1/admin/settings", map[string]any{"ai_enabled": true, "ai_base_url": provider.URL + "/v1", "ai_model": "stream-guard"}, 200)
			req, _ := http.NewRequest("POST", ts.URL+"/api/v1/ai/chat", bytes.NewReader(jsonValue(map[string]any{"document_id": d["id"], "prompt": "문서 요약"})))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Madi-Request", "1")
			if member.token != "" {
				req.Header.Set("Authorization", "Bearer "+member.token)
			}
			res, e := member.client.Do(req)
			if e != nil {
				t.Fatal(e)
			}
			defer res.Body.Close()
			if res.StatusCode != 200 {
				raw, _ := io.ReadAll(res.Body)
				t.Fatal(res.StatusCode, string(raw))
			}
			reader := bufio.NewReader(res.Body)
			for {
				line, e := reader.ReadString('\n')
				if e != nil {
					t.Fatal(e)
				}
				if strings.Contains(line, "FIRST_AUTHORIZED") {
					break
				}
			}
			switch reason {
			case "document_acl":
				admin.request("PUT", "/api/v1/documents/"+str(d, "id"), map[string]any{"version": 1, "visibility": "private"}, 200)
			case "document_version":
				admin.request("PUT", "/api/v1/documents/"+str(d, "id"), map[string]any{"version": 1, "markdown": "changed reference"}, 200)
			case "key_scope":
				admin.request("PUT", "/api/v1/keys/"+keyID, map[string]any{"scopes": []string{"document:read"}}, 200)
			case "session":
				member.request("POST", "/api/v1/auth/logout", nil, 200)
			}
			close(release)
			closed = true
			rest, e := io.ReadAll(reader)
			if e != nil {
				t.Fatal(e)
			}
			if strings.Contains(string(rest), "SHOULD_NOT_ESCAPE") || !strings.Contains(string(rest), `"retract":true`) {
				t.Fatal("stream did not retract after revocation", string(rest))
			}
		})
	}
}
