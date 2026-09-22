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
	"time"
)

func TestPostgresAISelectionQuietStreamRevokes(t *testing.T) {
	for _, reason := range []string{"document_version", "document_acl", "key_scope", "provider", "session"} {
		t.Run(reason, func(t *testing.T) {
			_, ts := integrationTestServer(t)
			admin := newIntegrationTestClient(t, ts.URL)
			admin.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200)
			wid := str(testJSONObject(t, admin.request("POST", "/api/v1/workspaces", map[string]any{"name": "선택 AI 회수"}, 200)), "id")
			user := testJSONObject(t, admin.request("POST", "/api/v1/admin/users", map[string]any{"email": "selection-stream@example.test", "name": "작성자", "role": "editor", "password": "Selection-stream-Password-2026!"}, 200))
			admin.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": user["email"], "role": "editor"}, 200)
			doc := testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "회수할 참조", "markdown": "선택 원문"}, 200))
			id := str(doc, "id")
			member := newIntegrationTestClient(t, ts.URL)
			member.request("POST", "/api/v1/auth/login", map[string]any{"email": user["email"], "password": "Selection-stream-Password-2026!"}, 200)
			keyID := ""
			if reason == "key_scope" {
				key := testJSONObject(t, member.request("POST", "/api/v1/keys", map[string]any{"name": "선택 AI 조회", "workspace_id": wid, "scopes": []string{"document:read", "ai:execute"}}, 201))
				member.token = str(key, "token")
				keyID = str(key["key"].(map[string]any), "id")
			}
			cancelled := make(chan struct{}, 1)
			provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"FIRST_SELECTED\"}}]}\n\n")
				w.(http.Flusher).Flush()
				<-r.Context().Done()
				cancelled <- struct{}{}
			}))
			defer provider.Close()
			admin.request("PUT", "/api/v1/admin/settings", map[string]any{"ai_enabled": true, "ai_base_url": provider.URL + "/v1", "ai_model": "selected"}, 200)
			path := "/api/v1/documents/" + id + "/ai-selection"
			meta := testJSONObject(t, member.request("GET", path, nil, 200))
			fp := str(meta["provider"].(map[string]any), "fingerprint")
			req, _ := http.NewRequestWithContext(t.Context(), "POST", ts.URL+path, bytes.NewReader(jsonValue(map[string]any{"expected_version": 1, "start_byte": 0, "end_byte": len("선택 원문"), "selected_text": "선택 원문", "provider_fingerprint": fp, "consent": true, "action": "rewrite"})))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Madi-Request", "1")
			if member.token != "" {
				req.Header.Set("Authorization", "Bearer "+member.token)
			}
			res, err := member.client.Do(req)
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
				line, e := reader.ReadString('\n')
				if e != nil {
					t.Fatal(e)
				}
				if strings.Contains(line, "FIRST_SELECTED") {
					break
				}
			}
			switch reason {
			case "document_version":
				admin.request("PUT", "/api/v1/documents/"+id, map[string]any{"version": 1, "markdown": "바뀐 원문"}, 200)
			case "document_acl":
				admin.request("PUT", "/api/v1/documents/"+id, map[string]any{"version": 1, "visibility": "private"}, 200)
			case "key_scope":
				admin.request("PUT", "/api/v1/keys/"+keyID, map[string]any{"scopes": []string{"document:read"}}, 200)
			case "provider":
				admin.request("PUT", "/api/v1/admin/settings", map[string]any{"ai_model": "changed"}, 200)
			case "session":
				member.request("POST", "/api/v1/auth/logout", nil, 200)
			}
			select {
			case <-cancelled:
			case <-time.After(5 * time.Second):
				t.Fatal("quiet provider not cancelled after revocation")
			}
			rest, err := io.ReadAll(reader)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(rest), `"retract":true`) || strings.Contains(string(rest), `"proposal":`) {
				t.Fatal("revoked response was not retracted", string(rest))
			}
		})
	}
}
