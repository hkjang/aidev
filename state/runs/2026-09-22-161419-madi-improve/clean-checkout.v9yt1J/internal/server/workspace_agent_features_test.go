package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPostgresWorkspaceAgentFeatureStopsActiveProvider(t *testing.T) {
	s, c, wid, _ := agentTestSetup(t)
	doc := testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "기능 정책", "markdown": "변하지 않는 원문"}, 200))
	did := str(doc, "id")
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.request("PUT", "/api/v1/admin/settings", map[string]any{"feature_flags": map[string]bool{"workspace-agents": false}}, 200)
		agentTestSSE(w, "FEATURE_REVOKED_OUTPUT", "", "")
	}))
	defer provider.Close()
	c.request("PUT", "/api/v1/admin/settings", map[string]any{"ai_enabled": true, "ai_base_url": provider.URL, "ai_model": "test"}, 200)
	a := agentTestConfig(t, c, wid, did, []string{"get_document"})
	aid := str(a, "id")
	rid := str(testJSONObject(t, c.request("POST", "/api/v1/agents/"+aid+"/runs", map[string]any{"prompt": "검사", "expected_agent_version": 1}, 202)), "id")
	agentTestPump(t, s, rid, "failed")
	var text string
	s.DB.QueryRow(t.Context(), `SELECT coalesce(string_agg(data::text,''),'') FROM agent_events WHERE run_id=$1`, rid).Scan(&text)
	if strings.Contains(text, "FEATURE_REVOKED_OUTPUT") {
		t.Fatal("feature revoked content persisted")
	}
	c.request("POST", "/api/v1/agents/"+aid+"/runs", map[string]any{"prompt": "blocked", "expected_agent_version": 1}, 403)
	c.request("GET", "/api/v1/agent-runs/"+rid, nil, 200)
	a["name"] = "비활성 기능도 설정 복구 가능"
	c.request("PUT", "/api/v1/agents/"+aid, a, 200)
}
func TestPostgresCollaborationFeatureRevocationKeepsMarkdown(t *testing.T) {
	s, c, _, wid, _ := collaborationTestSetup(t)
	doc := testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "정책 원문", "markdown": "original\n"}, 200))
	id := str(doc, "id")
	session := collaborationTestSession(t, c)
	initial, e := s.collaborationState(t.Context(), id, session, nil)
	if e != nil {
		t.Fatal(e)
	}
	quiet, _, e := collaborationTestDial(t, c, id, c.base)
	if e != nil {
		t.Fatal(e)
	}
	_ = collaborationTestRead(t, quiet, "hello")
	c.request("PUT", "/api/v1/admin/settings", map[string]any{"feature_flags": map[string]bool{"collaboration": false}}, 200)
	notice := collaborationTestRead(t, quiet, "error")
	if notice.Code != "feature_disabled" || len(notice.State) != 0 {
		t.Fatal("quiet client did not receive feature stop notice")
	}
	_, e = s.collaborationState(t.Context(), id, session, &collaborationMessage{Type: "seed", Schema: collaborationSchemaID, Epoch: initial.Epoch, State: []byte{1, 2, 3}})
	var fault *collaborationFault
	if !errors.As(e, &fault) || fault.code != "feature_disabled" {
		t.Fatal("disabled collaboration accepted state", e)
	}
	current := testJSONObject(t, c.request("GET", "/api/v1/documents/"+id, nil, 200))
	if str(current, "markdown") != "original\n" || number(current, "version", 0) != 1 {
		t.Fatal("feature policy changed canonical document")
	}
	c.request("PUT", "/api/v1/documents/"+id, map[string]any{"version": 1, "markdown": "일반 원문 편집은 유지"}, 200)
}
