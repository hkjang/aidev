package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
)

func graphAITestSetup(t *testing.T) (*Server, *integrationTestClient, string, string) {
	t.Helper()
	s, c, wid, uid := agentTestSetup(t)
	if e := s.migrateGraphAI(t.Context()); e != nil {
		t.Fatal(e)
	}
	if !slices.Contains(s.apiRoutes, "GET /api/v1/workspaces/{id}/graph-ai/context") {
		s.registerGraphAI()
	}
	return s, c, wid, uid
}
func graphAITestGenerate(t *testing.T, c *integrationTestClient, wid string, ids []string, kinds []string) string {
	t.Helper()
	preview := testJSONObject(t, c.request("GET", "/api/v1/workspaces/"+wid+"/graph-ai/context?document_ids="+strings.Join(ids, ","), nil, 200))
	id := newID()
	stream := c.request("POST", "/api/v1/workspaces/"+wid+"/graph-ai/analyze", map[string]any{"request_id": id, "snapshots": preview["snapshots"], "provider_fingerprint": preview["provider"].(map[string]any)["fingerprint"], "kinds": kinds, "consent": true}, 200)
	if !strings.Contains(string(stream), `"proposal":`) || strings.Contains(string(stream), `"retract":true`) {
		t.Fatalf("AI graph proposal stream: %s", stream)
	}
	return id
}
func graphAITestCandidates(a, b string) []graphAICandidate {
	proofA := graphAIEvidence{DocumentID: a, Quote: "PostgreSQL 운영 기준"}
	proofB := graphAIEvidence{DocumentID: b, Quote: "PostgreSQL 장애 복구"}
	return []graphAICandidate{
		{Kind: "relation", SourceID: a, TargetID: b, RelationType: "reference", Reason: "운영 기준과 복구 절차를 함께 확인할 수 있습니다.", Evidence: []graphAIEvidence{proofA, proofB}},
		{Kind: "duplicate", SourceID: a, TargetID: b, RelationType: "related", Reason: "제공된 구간의 PostgreSQL 설명은 비슷하지만 운영 기준과 장애 복구라는 목적 차이가 있습니다. 자동 병합하지 않습니다.", Evidence: []graphAIEvidence{proofA, proofB}},
		{Kind: "topic", SourceID: a, Topic: "데이터베이스 운영", Reason: "운영 기준의 핵심 주제입니다.", Evidence: []graphAIEvidence{proofA}},
		{Kind: "entity", Title: "PostgreSQL", EntityType: "technology", Description: "제공 자료에서 운영하는 데이터베이스 기술입니다.", Reason: "문서에 직접 언급한 기술의 비공개 엔터티 초안입니다.", Evidence: []graphAIEvidence{proofA, proofB}},
		{Kind: "gap", Title: "복구 검증 체크리스트", Description: "## 확인할 질문\n\n- 복구 성공을 어떤 절차로 확인하는가?\n- 마지막 테스트는 언제인가?\n\n제공 구간만으로는 답할 수 없어 확인이 필요합니다.", Reason: "현재 제공된 구간에는 검증 방법이 구체적으로 제시되지 않습니다. 다른 문서의 부재를 뜻하지 않습니다.", Evidence: []graphAIEvidence{proofB}},
	}
}
func TestGraphAICandidateSchemaAndExactEvidence(t *testing.T) {
	a, b := newID(), newID()
	sources := []aiSource{{ID: a, Version: 1, Markdown: "# PostgreSQL 운영 기준\n", StartLine: 1}, {ID: b, Version: 2, Markdown: "PostgreSQL 장애 복구", StartLine: 1}}
	valid := graphAITestCandidates(a, b)
	result, e := graphAIParseCandidates(string(jsonValue(map[string]any{"candidates": valid})), graphAIKinds, sources)
	if e != nil || len(result) != 5 {
		t.Fatalf("valid candidates: %v", e)
	}
	encoded := string(jsonValue(result))
	if strings.Contains(encoded, `"quote"`) || !strings.Contains(encoded, `"citation_id"`) {
		t.Fatal("durable candidate must not copy source quote")
	}
	for _, mode := range []string{"unknown_id", "invented_quote", "fake_citation", "wrong_kind", "invalid_relation", "missing_both"} {
		t.Run(mode, func(t *testing.T) {
			candidate := graphAITestCandidates(a, b)[0]
			switch mode {
			case "unknown_id":
				candidate.TargetID = newID()
			case "invented_quote":
				candidate.Evidence[0].Quote = "본문에 없는 인용"
			case "fake_citation":
				candidate.Evidence[0].Citation = &aiSource{ID: a}
			case "wrong_kind":
				candidate.Kind = "run_shell"
			case "invalid_relation":
				candidate.RelationType = "parent"
			case "missing_both":
				candidate.Evidence = candidate.Evidence[:1]
			}
			if _, e := graphAIParseCandidates(string(jsonValue(map[string]any{"candidates": []graphAICandidate{candidate}})), graphAIKinds, sources); e == nil {
				t.Fatal("malformed model output accepted")
			}
		})
	}
}
func TestPostgresGraphAIActualStreamAndExplicitActions(t *testing.T) {
	s, c, wid, _ := graphAITestSetup(t)
	a := testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "운영 기준", "markdown": "# PostgreSQL 운영 기준\n\n기본 원칙을 기록합니다.\n", "visibility": "private"}, 200))
	b := testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "장애 복구", "markdown": "# PostgreSQL 장애 복구\n\n서비스 정상화를 위한 운영 지식입니다.\n"}, 200))
	aid, bid := str(a, "id"), str(b, "id")
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		if json.NewDecoder(r.Body).Decode(&payload) != nil || !boolean(payload, "stream") || number(payload, "max_tokens", 0) != 4096 {
			http.Error(w, "bad provider request", 400)
			return
		}
		messages := payload["messages"].([]any)
		var prompt map[string]any
		json.Unmarshal([]byte(str(messages[len(messages)-1].(map[string]any), "content")), &prompt)
		allowed := listStrings(prompt["allowed_kinds"])
		candidates := []graphAICandidate{}
		for _, candidate := range graphAITestCandidates(aid, bid) {
			if slices.Contains(allowed, candidate.Kind) {
				candidates = append(candidates, candidate)
			}
		}
		agentTestSSE(w, string(jsonValue(map[string]any{"candidates": candidates})), "", "")
	}))
	defer provider.Close()
	c.request("PUT", "/api/v1/admin/settings", map[string]any{"ai_enabled": true, "ai_base_url": provider.URL, "ai_model": "graph-contract", "ai_max_tokens": 4096}, 200)
	id := graphAITestGenerate(t, c, wid, []string{aid, bid}, []string{"relation", "duplicate", "entity", "gap"})
	view := testJSONObject(t, c.request("GET", "/api/v1/graph-ai/runs/"+id, nil, 200))
	if !boolean(view, "can_apply") {
		t.Fatalf("can_apply missing: %s", jsonValue(view))
	}
	var count int
	s.DB.QueryRow(t.Context(), `SELECT count(*) FROM document_relations`).Scan(&count)
	if count != 0 {
		t.Fatal("model relations were applied automatically")
	}
	actions := view["actions"].([]any)
	if len(actions) != 4 {
		t.Fatal("not all candidate kinds persisted")
	}
	for _, raw := range actions {
		action := raw.(map[string]any)
		path := "/api/v1/graph-ai/actions/" + str(action, "id") + "/confirm"
		c.request("POST", path, map[string]any{"action_hash": action["action_hash"], "confirm": false}, 400)
		result := testJSONObject(t, c.request("POST", path, map[string]any{"action_hash": action["action_hash"], "confirm": true}, 200))
		if str(result, "status") != "applied" {
			t.Fatal("not applied")
		}
		c.request("POST", path, map[string]any{"action_hash": action["action_hash"], "confirm": true}, 200)
		if oneOf(str(action, "kind"), "entity", "gap") {
			did := str(result["result"].(map[string]any), "document_id")
			doc := testJSONObject(t, c.request("GET", "/api/v1/documents/"+did, nil, 200))
			if str(doc, "visibility") != "private" {
				t.Fatal("AI source disclosed through public draft")
			}
			if str(action, "kind") == "entity" {
				entity := testJSONObject(t, c.request("GET", "/api/v1/enterprise/entities/"+did, nil, 200))
				if len(entity["related_documents"].([]any)) != 2 {
					t.Fatalf("entity source relations missing: %s", jsonValue(entity))
				}
			}
		}
	}
	topicRun := graphAITestGenerate(t, c, wid, []string{aid}, []string{"topic"})
	topicView := testJSONObject(t, c.request("GET", "/api/v1/graph-ai/runs/"+topicRun, nil, 200))
	topicAction := topicView["actions"].([]any)[0].(map[string]any)
	c.request("POST", "/api/v1/graph-ai/actions/"+str(topicAction, "id")+"/confirm", map[string]any{"action_hash": topicAction["action_hash"], "confirm": true}, 200)
	for _, before := range []map[string]any{a, b} {
		after := testJSONObject(t, c.request("GET", "/api/v1/documents/"+str(before, "id"), nil, 200))
		if str(after, "markdown") != str(before, "markdown") || number(after, "version", 0) != 1 {
			t.Fatal("analysis mutated source Markdown/version")
		}
	}
	var topics string
	if e := s.DB.QueryRow(t.Context(), `SELECT system_metadata->'ai_topics' FROM knowledge_document_meta WHERE document_id=$1`, aid).Scan(&topics); e != nil || !strings.Contains(topics, "데이터베이스 운영") {
		t.Fatalf("topic annotation missing %s %v", topics, e)
	}
	s.DB.QueryRow(t.Context(), `SELECT count(*) FROM documents WHERE workspace_id=$1`, wid).Scan(&count)
	if count != 4 {
		t.Fatalf("duplicate confirmed doc creations: %d", count)
	}
	c.request("DELETE", "/api/v1/graph-ai/runs/"+id, nil, 200)
	s.DB.QueryRow(t.Context(), `SELECT count(*) FROM graph_ai_annotations`).Scan(&count)
	if count != 3 {
		t.Fatal("history deletion removed approved annotations")
	}
}
