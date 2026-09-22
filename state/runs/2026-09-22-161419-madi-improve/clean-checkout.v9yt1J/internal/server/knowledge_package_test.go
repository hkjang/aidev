package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestKnowledgePackageBudgetPreservesPolicyAndExactSpans(t *testing.T) {
	quote := func(id, body string, mandatory bool) knowledgePackageQuote {
		return knowledgePackageQuote{evidenceQuote: evidenceQuote{sourceFromChunk(id, "자료", 1, ragChunk{Start: 0, End: len(body), StartLine: 1, EndLine: 1, Content: body, Hash: digest(body)}), body}, Mandatory: mandatory, Reason: "사람이 지정한 포함 이유", Representation: "original"}
	}
	mandatory := quote(newID(), strings.Repeat("필수정책 한글\n", 20), true)
	optional := quote(newID(), strings.Repeat("선택참고 영어test\n", 30), false)
	input := knowledgePackagePayload{Purpose: "운영 변경", Model: "unknown-local", TokenBudget: len(renderKnowledgePackage(knowledgePackagePayload{Purpose: "운영 변경", Model: "unknown-local"}, []knowledgePackageQuote{mandatory})) + 1}
	out, err := fitKnowledgePackage(context.Background(), input, []knowledgePackageQuote{optional, mandatory, optional}, estimatePackageTokens)
	if err != nil || len(out.Quotes) != 1 || !out.Quotes[0].Mandatory || out.Quotes[0].Text != mandatory.Text || out.TokenCount != len(out.Prompt) || out.TokenCount > out.TokenBudget || out.OmittedChunks != 1 || out.DuplicateChunks != 1 {
		t.Fatal(out, err)
	}
	input.TokenBudget = 10
	if _, err = fitKnowledgePackage(context.Background(), input, []knowledgePackageQuote{mandatory}, estimatePackageTokens); err == nil {
		t.Fatal("mandatory policy silently truncated")
	}
}

func TestPostgresKnowledgePackageMCPConsentSnapshotAndRevocation(t *testing.T) {
	s, c, ctx, _, wid := jobTestFixture(t)
	doc := testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "필수 운영 정책", "markdown": "PACKAGE_POLICY_SENTINEL: 변경 전에 검토하세요."}, 200))
	did := str(doc, "id")
	input := map[string]any{"workspace_id": wid, "purpose": "운영 설정 변경안 검토", "allowed_scope": "문서 검토만; 명령 실행 없음", "model": "사내 모델", "token_budget": 8192, "documents": []map[string]any{{"id": did, "version": 1, "mandatory": true, "reason": "필수 변경 규정"}}, "counter": "estimate", "consent": true}
	id := str(testJSONObject(t, c.request("POST", "/api/v1/knowledge/packages", input, 201)), "id")
	record := testJSONObject(t, c.request("GET", "/api/v1/knowledge/packages/"+id, nil, 200))
	pkg := record["package"].(map[string]any)
	if !strings.Contains(str(pkg, "prompt"), "PACKAGE_POLICY_SENTINEL") || boolean(pkg, "execution_permission") || number(pkg, "token_count", 0) != len(str(pkg, "prompt")) {
		t.Fatal(record)
	}
	var cipher string
	if err := s.DB.QueryRow(ctx, `SELECT ciphertext FROM knowledge_packages WHERE id=$1`, id).Scan(&cipher); err != nil || strings.Contains(cipher, "PACKAGE_POLICY_SENTINEL") {
		t.Fatal(err, "plaintext package persisted")
	}
	key := testJSONObject(t, c.request("POST", "/api/v1/keys", map[string]any{"name": "자료 조회 전용", "workspace_id": wid, "scopes": []string{"document:read"}}, 201))
	c.token = str(key, "token")
	meta := c.request("GET", "/api/v1/knowledge/packages/"+id, nil, 200)
	if strings.Contains(string(meta), "PACKAGE_POLICY_SENTINEL") || !strings.Contains(string(meta), "export_required") {
		t.Fatal("key obtained body without export consent", string(meta))
	}
	c.request("POST", "/api/v1/knowledge/packages/"+id+"/export", map[string]any{"target": "검토 에이전트"}, 400)
	for _, name := range []string{"inspect_knowledge_package", "export_knowledge_package"} {
		args := map[string]any{"package_id": id}
		if strings.HasPrefix(name, "export") {
			args["target"] = "검토 에이전트"
			args["consent"] = true
		}
		out := testJSONObject(t, c.request("POST", "/api/v1/mcp", map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": name, "arguments": args}}, 200))
		result := out["result"].(map[string]any)
		if boolean(result, "isError") {
			t.Fatal(out)
		}
		if strings.HasPrefix(name, "export") && !strings.Contains(string(jsonValue(result)), "PACKAGE_POLICY_SENTINEL") {
			t.Fatal("MCP export missing canonical source")
		}
	}
	input["counter"] = "responses"
	input["model_consent"] = true
	c.request("POST", "/api/v1/knowledge/packages", input, 403)
	c.token = ""
	c.request("PUT", "/api/v1/documents/"+did, map[string]any{"version": 1, "markdown": "변경된 정책"}, 200)
	stale := testJSONObject(t, c.request("GET", "/api/v1/knowledge/packages/"+id, nil, 200))
	if !boolean(stale, "stale") {
		t.Fatal("package didn't detect source change")
	}
	c.request("POST", "/api/v1/knowledge/packages/"+id+"/export", map[string]any{"target": "검토 에이전트", "consent": true}, 409)
	input["counter"] = "estimate"
	c.request("POST", "/api/v1/knowledge/packages", input, 409)
	if _, err := s.DB.Exec(ctx, `UPDATE documents SET deleted_at=now() WHERE id=$1`, did); err != nil {
		t.Fatal(err)
	}
	c.request("GET", "/api/v1/knowledge/packages/"+id, nil, 404)
}

func TestPostgresKnowledgePackageProviderCounterExplicitConsent(t *testing.T) {
	_, c, _, _, wid := jobTestFixture(t)
	doc := testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "토큰 근거", "markdown": "TOKEN_COUNTER_SENTINEL 한글 정책"}, 200))
	did := str(doc, "id")
	calls := 0
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/v1/responses/input_tokens" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		var in map[string]any
		if json.NewDecoder(r.Body).Decode(&in) != nil || str(in, "model") != "local-model" {
			t.Error("model counter contract")
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"object":"response.input_tokens","input_tokens":100}`)
	}))
	defer provider.Close()
	c.request("PUT", "/api/v1/admin/settings", map[string]any{"ai_enabled": true, "ai_base_url": provider.URL + "/v1", "ai_model": "local-model"}, 200)
	c.request("PUT", "/api/v1/admin/knowledge-packages/policy", map[string]any{"enabled": true, "token_counter": "responses", "allow_http": true, "retention_hours": 24, "version": 1}, 200)
	input := map[string]any{"workspace_id": wid, "purpose": "검토", "model": "local-model", "token_budget": 512, "documents": []map[string]any{{"id": did, "version": 1, "mandatory": true}}, "counter": "responses", "consent": true}
	c.request("POST", "/api/v1/knowledge/packages", input, 403)
	if calls != 0 {
		t.Fatal("source transmitted without explicit consent")
	}
	input["model_consent"] = true
	result := testJSONObject(t, c.request("POST", "/api/v1/knowledge/packages", input, 201))
	if calls == 0 || str(result, "counter") != "responses" || number(result, "token_count", 0) != 100 {
		t.Fatal(result, calls)
	}
}

func TestPackageCounterRefusesMalformedAndRedirectResponses(t *testing.T) {
	for _, body := range []string{`{"input_tokens":0,"object":"response.input_tokens"}`, `{"input_tokens":1.5,"object":"response.input_tokens"}`, `{"input_tokens":10}`, `{"input_tokens":12,"object":"other"}`} {
		provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, body)
		}))
		_, err := responsePackageTokens(context.Background(), map[string]any{"ai_base_url": provider.URL + "/v1", "ai_model": "model"}, true, "원문")
		provider.Close()
		if err == nil {
			t.Fatal("invalid count accepted", body)
		}
	}
	for _, code := range []int{301, 302, 303, 307, 308} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			received := make(chan bool, 1)
			target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				received <- true
				fmt.Fprint(w, `{"object":"response.input_tokens","input_tokens":1}`)
			}))
			defer target.Close()
			provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, code) }))
			defer provider.Close()
			if _, e := responsePackageTokens(context.Background(), map[string]any{"ai_base_url": provider.URL + "/v1", "ai_model": "model", "ai_api_key": "test-only-secret"}, true, "PRIVATE-REDIRECT-SENTINEL"); e == nil {
				t.Fatal("redirect accepted")
			}
			select {
			case <-received:
				t.Fatal("source or credentials followed a redirect")
			default:
			}
		})
	}
}
