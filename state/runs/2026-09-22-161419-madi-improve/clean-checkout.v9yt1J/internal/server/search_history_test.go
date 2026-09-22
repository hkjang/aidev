package server

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestPostgresSearchHistoryProtectionRetentionAndTokenDenial(t *testing.T) {
	s, c, ctx, p, wid := jobTestFixture(t)
	c.request("PUT", "/api/v1/profile/search-history/settings", map[string]any{"enabled": true, "retention_days": 30, "revision": 0, "consent": true}, 200)
	policy := defaultProtectionSettings()
	policy["enabled"], policy["mode"], policy["detectors"] = true, "mask", []string{"email"}
	if _, e := s.DB.Exec(ctx, "UPDATE protection_settings SET data=$1", jsonValue(policy)); e != nil {
		t.Fatal(e)
	}
	search := func(term string) map[string]any {
		return testJSONObject(t, c.request("GET", "/api/v1/search?workspace_id="+wid+"&q="+url.QueryEscape(term)+"&type=document", nil, 200))
	}
	if str(search("private-query@example.test"), "history_status") != "saved" {
		t.Fatal("masked query not saved")
	}
	var body string
	if e := s.DB.QueryRow(ctx, `SELECT query FROM search_history_entries WHERE user_id=$1`, p.ID).Scan(&body); e != nil || strings.Contains(body, "private-query@example.test") {
		t.Fatal("unmasked personal query", body, e)
	}
	policy["mode"] = "block"
	if _, e := s.DB.Exec(ctx, "UPDATE protection_settings SET data=$1", jsonValue(policy)); e != nil {
		t.Fatal(e)
	}
	if str(search("blocked-query@example.test"), "history_status") != "protection_blocked" {
		t.Fatal("blocked query persisted")
	}
	var count int
	if e := s.DB.QueryRow(ctx, `SELECT count(*) FROM search_history_entries WHERE user_id=$1`, p.ID).Scan(&count); e != nil || count != 1 {
		t.Fatal(count, e)
	}
	key := testJSONObject(t, c.request("POST", "/api/v1/keys", map[string]any{"name": "personal history denial", "workspace_id": wid, "scopes": []string{"search:read", "document:read", "ai:execute"}}, 201))
	c.token = str(key, "token")
	c.request("GET", "/api/v1/search/history?workspace_id="+wid, nil, 403)
	c.request("GET", "/api/v1/profile/search-history/settings", nil, 403)
	c.request("DELETE", "/api/v1/search/history", map[string]any{"workspace_id": wid, "confirmation": "DELETE_ALL"}, 403)
	if str(search("key search without personal capture"), "history_status") != "not_saved" {
		t.Fatal("key search captured")
	}
	c.token = ""
	// Decreasing the policy shortens old rows; increasing it never resurrects or extends them.
	if _, e := s.DB.Exec(ctx, `UPDATE search_history_entries SET first_seen=now()-interval '10 days',last_seen=now()-interval '10 days',expires_at=now()+interval '20 days' WHERE user_id=$1`, p.ID); e != nil {
		t.Fatal(e)
	}
	c.request("PUT", "/api/v1/profile/search-history/settings", map[string]any{"enabled": true, "retention_days": 7, "revision": 1, "consent": true}, 200)
	var before, after time.Time
	if e := s.DB.QueryRow(ctx, `SELECT expires_at FROM search_history_entries WHERE user_id=$1`, p.ID).Scan(&before); e != nil || !before.Before(time.Now()) {
		t.Fatal(before, e)
	}
	list := testJSONObject(t, c.request("GET", "/api/v1/search/history?workspace_id="+wid, nil, 200))
	if len(list["items"].([]any)) != 0 {
		t.Fatal("expired history still visible", list)
	}
	c.request("PUT", "/api/v1/profile/search-history/settings", map[string]any{"enabled": true, "retention_days": 365, "revision": 2, "consent": true}, 200)
	if e := s.DB.QueryRow(ctx, `SELECT expires_at FROM search_history_entries WHERE user_id=$1`, p.ID).Scan(&after); e != nil || !before.Equal(after) {
		t.Fatal("retention increase extended historical rows", before, after, e)
	}
	if e := s.expireSearchHistory(ctx); e != nil {
		t.Fatal(e)
	}
	if e := s.DB.QueryRow(ctx, `SELECT count(*) FROM search_history_entries WHERE user_id=$1`, p.ID).Scan(&count); e != nil || count != 0 {
		t.Fatal("expiry cleanup", count, e)
	}
}

func TestPostgresSearchHistoryQuietStreamConsentRevocation(t *testing.T) {
	_, c, _, _, wid := jobTestFixture(t)
	c.request("PUT", "/api/v1/profile/search-history/settings", map[string]any{"enabled": true, "retention_days": 30, "revision": 0, "consent": true}, 200)
	c.request("GET", "/api/v1/search?workspace_id="+wid+"&q=gap-quiet-test", nil, 200)
	list := testJSONObject(t, c.request("GET", "/api/v1/search/history?workspace_id="+wid, nil, 200))
	entry := list["items"].([]any)[0].(map[string]any)
	started, stopped := make(chan struct{}), make(chan struct{})
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"{\"}}]}\n\n")
		w.(http.Flusher).Flush()
		<-r.Context().Done()
		close(stopped)
	}))
	defer provider.Close()
	c.request("PUT", "/api/v1/admin/settings", map[string]any{"ai_enabled": true, "ai_base_url": provider.URL, "ai_model": "quiet-gap"}, 200)
	done := make(chan []byte, 1)
	go func() {
		req, _ := http.NewRequest("POST", c.base+"/api/v1/search/history/gaps", bytes.NewReader(jsonValue(map[string]any{"workspace_id": wid, "entries": []any{map[string]any{"id": entry["id"], "revision": entry["revision"]}}, "consent": true})))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Madi-Request", "1")
		response, e := c.client.Do(req)
		if e != nil {
			done <- []byte(e.Error())
			return
		}
		defer response.Body.Close()
		reader := bufio.NewReader(response.Body)
		var body strings.Builder
		ready := false
		for {
			line, e := reader.ReadString('\n')
			body.WriteString(line)
			if !ready && strings.Contains(line, `"text"`) {
				ready = true
				close(started)
			}
			if e != nil {
				if e != io.EOF {
					body.WriteString(e.Error())
				}
				break
			}
		}
		done <- []byte(body.String())
	}()
	select {
	case <-started:
	case <-time.After(4 * time.Second):
		t.Fatal("no first search gap delta")
	}
	c.request("PUT", "/api/v1/profile/search-history/settings", map[string]any{"enabled": false, "retention_days": 30, "revision": 1}, 200)
	select {
	case body := <-done:
		if !bytes.Contains(body, []byte(`"retract":true`)) || bytes.Contains(body, []byte(`"proposal":`)) {
			t.Fatal("consent-revoked answer survived", string(body))
		}
	case <-time.After(4 * time.Second):
		t.Fatal("quiet gap stream not revoked")
	}
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("quiet provider not cancelled")
	}
}

func TestSearchGapClosedSchema(t *testing.T) {
	valid := `{"notice":"선택한 검색 기록만 분석","suggestions":[{"title":"점검 안내","reason":"실패 검색 보완","outline":"# 질문\n\n담당자 확인","references":[1]}]}`
	if _, e := parseSearchGapProposal(valid, 1); e != nil {
		t.Fatal(e)
	}
	for _, bad := range []string{strings.Replace(valid, `[1]`, `[2]`, 1), strings.Replace(valid, `[1]`, `[1,1]`, 1), strings.Replace(valid, `"title"`, `"sql"`, 1), valid + `{}`} {
		if _, e := parseSearchGapProposal(bad, 1); e == nil {
			t.Fatal("invalid gap plan accepted")
		}
	}
}
func TestPostgresSearchHistoryConsentPrivacyGroupingGapsAndErasure(t *testing.T) {
	s, ts := integrationTestServer(t)
	c := newIntegrationTestClient(t, ts.URL)
	me := testJSONObject(t, c.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200))
	wid := str(testJSONObject(t, c.request("POST", "/api/v1/workspaces", map[string]any{"name": "Private search knowledge gaps"}, 200)), "id")
	query := "PRIVATE_SEARCH_NO_RESULT"
	path := "/api/v1/search?workspace_id=" + wid + "&q=" + url.QueryEscape(query) + "&type=document"
	cfg := testJSONObject(t, c.request("GET", "/api/v1/profile/search-history/settings", nil, 200))
	if boolean(cfg, "enabled") || number(cfg, "revision", -1) != 0 {
		t.Fatal("history default not off", cfg)
	}
	c.request("GET", path, nil, 200)
	var n int
	if e := s.DB.QueryRow(t.Context(), "SELECT count(*) FROM search_history_entries").Scan(&n); e != nil || n != 0 {
		t.Fatal("raw query stored without consent", n, e)
	}
	c.request("PUT", "/api/v1/profile/search-history/settings", map[string]any{"enabled": true, "retention_days": 30, "revision": 0}, 400)
	c.request("PUT", "/api/v1/profile/search-history/settings", map[string]any{"enabled": true, "retention_days": 30, "revision": 0, "consent": true}, 200)
	c.request("PUT", "/api/v1/profile/search-history/settings", map[string]any{"enabled": false, "retention_days": 30, "revision": 0}, 409)
	for i := 0; i < 2; i++ {
		response := testJSONObject(t, c.request("GET", path, nil, 200))
		if str(response, "history_status") != "saved" {
			t.Fatal("personal capture failed", response)
		}
	}
	list := testJSONObject(t, c.request("GET", "/api/v1/search/history?workspace_id="+wid+"&zero=1", nil, 200))
	rows := list["items"].([]any)
	if len(rows) != 1 {
		t.Fatal("same daily query not grouped", list)
	}
	entry := rows[0].(map[string]any)
	if number(entry, "searches", 0) != 2 || number(entry, "zero_results", 0) != 2 || str(entry, "query") != query {
		t.Fatal(entry)
	}
	if e := s.DB.QueryRow(t.Context(), "SELECT count(*) FROM audit_logs WHERE details::text LIKE '%PRIVATE_SEARCH_NO_RESULT%'").Scan(&n); e != nil || n != 0 {
		t.Fatal("private query in shared audit", n, e)
	}
	other := testJSONObject(t, c.request("POST", "/api/v1/admin/users", map[string]any{"email": "query-other@example.test", "name": "다른 사용자", "role": "editor", "password": "Other-Query-Password-2026!"}, 200))
	c.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": other["email"], "role": "editor"}, 200)
	o := newIntegrationTestClient(t, ts.URL)
	o.request("POST", "/api/v1/auth/login", map[string]any{"email": other["email"], "password": "Other-Query-Password-2026!"}, 200)
	if bytes.Contains(o.request("GET", "/api/v1/search/history?workspace_id="+wid, nil, 200), []byte(query)) {
		t.Fatal("other user read raw query")
	}
	var calls atomic.Int32
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		_ = json.NewDecoder(r.Body).Decode(&payload)
		calls.Add(1)
		if !boolean(payload, "stream") || !strings.Contains(string(jsonValue(payload)), query) {
			t.Error("selected history not sent as stream")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		answer := `{"notice":"선택한 실패 검색에 한정한 제안","suggestions":[{"title":"점검 가이드","reason":"반복된 검색의 보완 후보","outline":"# 목적\n\n담당자와 기준을 확인하세요.","references":[1]}]}`
		fmt.Fprintf(w, "data: %s\n\ndata: [DONE]\n\n", jsonValue(map[string]any{"choices": []any{map[string]any{"delta": map[string]any{"content": answer}}}}))
	}))
	defer provider.Close()
	c.request("PUT", "/api/v1/admin/settings", map[string]any{"ai_enabled": true, "ai_base_url": provider.URL, "ai_model": "personal-gap"}, 200)
	in := map[string]any{"workspace_id": wid, "entries": []any{map[string]any{"id": entry["id"], "revision": entry["revision"]}}}
	c.request("POST", "/api/v1/search/history/gaps", in, 400)
	if calls.Load() != 0 {
		t.Fatal("model called without additional consent")
	}
	in["consent"] = true
	body := c.request("POST", "/api/v1/search/history/gaps", in, 200)
	if !bytes.Contains(body, []byte(`"suggestions"`)) || !bytes.Contains(body, []byte(`"automatic_apply":false`)) {
		t.Fatal(string(body))
	}
	c.request("GET", path, nil, 200)
	c.request("POST", "/api/v1/search/history/gaps", in, 409)
	if calls.Load() != 1 {
		t.Fatal("stale history transmitted")
	}
	c.request("PUT", "/api/v1/profile/search-history/settings", map[string]any{"enabled": false, "retention_days": 7, "revision": 1}, 200)
	response := testJSONObject(t, c.request("GET", path, nil, 200))
	if str(response, "history_status") != "disabled" {
		t.Fatal("history kept recording after opt-out")
	}
	c.request("POST", "/api/v1/search/history/gaps", in, 403)
	c.request("DELETE", "/api/v1/search/history", map[string]any{"workspace_id": wid, "confirmation": "DELETE_ALL"}, 200)
	if e := s.DB.QueryRow(t.Context(), "SELECT count(*) FROM search_history_entries WHERE user_id=$1", str(me, "id")).Scan(&n); e != nil || n != 0 {
		t.Fatal("history not erased", n, e)
	}
}
