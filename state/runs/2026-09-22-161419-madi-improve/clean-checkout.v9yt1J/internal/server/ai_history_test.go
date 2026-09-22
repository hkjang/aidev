package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAIHistoryTicketBoundToOwnerDomainExpiry(t *testing.T) {
	s := &Server{EncryptionKey: bytes.Repeat([]byte{7}, 32)}
	p := &Principal{ID: newID(), Kind: "user"}
	raw, err := s.sealAIHistory(p, newID(), "질문", "답변", "ask", nil, map[string]any{"ai_model": "fixture"})
	if err != nil {
		t.Fatal(err)
	}
	ticket, err := s.openAIHistory(raw, p)
	if err != nil || ticket.Answer != "답변" {
		t.Fatal(ticket, err)
	}
	for _, other := range []*Principal{{ID: newID()}, {ID: p.ID, TokenID: newID()}, {ID: p.ID, PluginID: "plugin", ScopeRestricted: true}, {ID: p.ID, Kind: "service"}} {
		if _, err = s.openAIHistory(raw, other); err == nil {
			t.Fatal("ticket accepted for another principal")
		}
	}
	for _, mutate := range []func(*aiHistoryTicket){func(v *aiHistoryTicket) { v.Kind = "other-purpose" }, func(v *aiHistoryTicket) { v.Expires = time.Now().Add(-time.Minute).Unix() }, func(v *aiHistoryTicket) { v.UserID = newID() }} {
		v := ticket
		mutate(&v)
		bad, _ := s.encrypt(string(jsonValue(v)))
		if _, err = s.openAIHistory(bad, p); err == nil {
			t.Fatal("invalid sealed ticket accepted")
		}
	}
	if _, err = s.openAIHistory(raw[:len(raw)-8]+"AAAAAAAA", p); err == nil {
		t.Fatal("forged encrypted ticket accepted")
	}
}

func aiHistoryTestTicket(t *testing.T, client *integrationTestClient, wid, did string, extra ...map[string]any) string {
	t.Helper()
	input := map[string]any{"workspace_id": wid, "document_id": did, "prompt": "개인 기록의 근거를 알려 주세요"}
	for _, fields := range extra {
		for k, v := range fields {
			input[k] = v
		}
	}
	body := client.request("POST", "/api/v1/ai/chat", input, 200)
	for _, line := range strings.Split(string(body), "\n") {
		if strings.HasPrefix(line, "data: ") {
			var data map[string]any
			if json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &data) == nil && str(data, "history_ticket") != "" {
				return str(data, "history_ticket")
			}
		}
	}
	t.Fatalf("no completed history ticket: %s", body)
	return ""
}

func TestPostgresAIHistoryOwnerConsentReplayAndSourceACL(t *testing.T) {
	s, ts := integrationTestServer(t)
	admin := newIntegrationTestClient(t, ts.URL)
	admin.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200)
	wid := str(testJSONObject(t, admin.request("POST", "/api/v1/workspaces", map[string]any{"name": "개인 대화 경계"}, 200)), "id")
	user := testJSONObject(t, admin.request("POST", "/api/v1/admin/users", map[string]any{"email": "history@example.test", "name": "개인 사용자", "password": "Personal-history-2026!", "role": "editor"}, 200))
	admin.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": user["email"], "role": "editor"}, 200)
	member := newIntegrationTestClient(t, ts.URL)
	member.request("POST", "/api/v1/auth/login", map[string]any{"email": user["email"], "password": "Personal-history-2026!"}, 200)
	d := testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "참고 문서", "markdown": "HISTORY_SOURCE_SENTINEL"}, 200))
	did := str(d, "id")
	requests := make(chan map[string]any, 32)
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		if json.NewDecoder(r.Body).Decode(&payload) != nil {
			w.WriteHeader(400)
			return
		}
		requests <- payload
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"HISTORY_PRIVATE_ANSWER [1]\"}}]}\n\ndata: [DONE]\n\n")
	}))
	defer provider.Close()
	admin.request("PUT", "/api/v1/admin/settings", map[string]any{"ai_enabled": true, "ai_base_url": provider.URL + "/v1", "ai_model": "history"}, 200)
	ticket := aiHistoryTestTicket(t, member, wid, did)
	var count int
	if e := s.DB.QueryRow(t.Context(), "SELECT count(*) FROM ai_conversations").Scan(&count); e != nil || count != 0 {
		t.Fatal("history saved without consent", count, e)
	}
	member.request("POST", "/api/v1/ai/conversations", map[string]any{"ticket": ticket}, 400)
	admin.request("POST", "/api/v1/ai/conversations", map[string]any{"ticket": ticket, "consent": true}, 400)
	saved := testJSONObject(t, member.request("POST", "/api/v1/ai/conversations", map[string]any{"ticket": ticket, "consent": true}, 201))
	id := str(saved, "id")
	replay := testJSONObject(t, member.request("POST", "/api/v1/ai/conversations", map[string]any{"ticket": ticket, "consent": true}, 200))
	if str(replay, "id") != id || !boolean(replay, "replayed") {
		t.Fatal("replay duplicated history")
	}
	body := member.request("GET", "/api/v1/ai/conversations/"+id, nil, 200)
	if !bytes.Contains(body, []byte("HISTORY_PRIVATE_ANSWER")) || bytes.Contains(body, []byte("provider_fingerprint")) || bytes.Contains(body, []byte("grant_refs")) {
		t.Fatal(string(body))
	}
	admin.request("GET", "/api/v1/ai/conversations/"+id, nil, 404)
	if list := admin.request("GET", "/api/v1/ai/conversations?workspace_id="+wid, nil, 200); bytes.Contains(list, []byte(id)) {
		t.Fatal("admin bypassed personal history")
	}
	<-requests // First request has no previous conversation.
	continued := aiHistoryTestTicket(t, member, wid, did, map[string]any{"conversation_id": id, "conversation_version": 1, "prompt": "구체적인 다음 단계를 설명하세요"})
	payload := <-requests
	sawPrevious := false
	for _, raw := range payload["messages"].([]any) {
		message := raw.(map[string]any)
		if str(message, "role") == "assistant" && strings.Contains(str(message, "content"), "HISTORY_PRIVATE_ANSWER") {
			sawPrevious = true
		}
	}
	if !sawPrevious {
		t.Fatal("selected history was not sent to the same provider")
	}
	competing := aiHistoryTestTicket(t, member, wid, did, map[string]any{"conversation_id": id, "conversation_version": 1})
	second := testJSONObject(t, member.request("POST", "/api/v1/ai/conversations", map[string]any{"ticket": continued, "consent": true}, 201))
	if str(second, "id") != id || number(second, "version", 0) != 2 {
		t.Fatal("follow-up did not append to the same versioned conversation", second)
	}
	member.request("POST", "/api/v1/ai/conversations", map[string]any{"ticket": competing, "consent": true}, 409)
	member.request("POST", "/api/v1/ai/chat", map[string]any{"workspace_id": wid, "conversation_id": id, "conversation_version": 1, "prompt": "stale conversation"}, 409)
	admin.request("PUT", "/api/v1/admin/settings", map[string]any{"ai_model": "different-provider-model"}, 200)
	member.request("POST", "/api/v1/ai/chat", map[string]any{"workspace_id": wid, "conversation_id": id, "conversation_version": 2, "prompt": "must not resend historical secrets"}, 409)
	admin.request("PUT", "/api/v1/admin/settings", map[string]any{"ai_model": "history"}, 200)
	key := testJSONObject(t, member.request("POST", "/api/v1/keys", map[string]any{"name": "history restricted", "workspace_id": wid, "scopes": []string{"ai:execute", "document:read", "search:read"}}, 201))
	member.token = str(key, "token")
	member.request("GET", "/api/v1/ai/conversations?workspace_id="+wid, nil, 403)
	member.request("POST", "/api/v1/ai/conversations", map[string]any{"ticket": ticket, "consent": true}, 403)
	member.token = ""
	stale := aiHistoryTestTicket(t, member, wid, did)
	admin.request("PUT", "/api/v1/documents/"+did, map[string]any{"version": 1, "visibility": "private"}, 200)
	member.request("POST", "/api/v1/ai/conversations", map[string]any{"ticket": stale, "consent": true}, 409)
	member.request("GET", "/api/v1/ai/conversations/"+id, nil, 404)
	if list := member.request("GET", "/api/v1/ai/conversations?workspace_id="+wid+"&q=HISTORY_PRIVATE_ANSWER", nil, 200); bytes.Contains(list, []byte(id)) {
		t.Fatal("revoked source leaked conversation metadata")
	}
	member.request("DELETE", "/api/v1/ai/conversations/"+id, map[string]any{"confirmation": "DELETE", "version": 2}, 200)
	member.request("DELETE", "/api/v1/ai/conversations/"+id, map[string]any{"confirmation": "DELETE", "version": 2}, 409)
}
