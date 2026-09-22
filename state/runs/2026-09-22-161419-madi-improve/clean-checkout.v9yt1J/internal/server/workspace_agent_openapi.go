package server

func workspaceAgentOpenAPI(paths, schemas map[string]any) {
	text := map[string]any{"type": "string"}
	ids := map[string]any{"type": "array", "maxItems": 100, "uniqueItems": true, "items": map[string]any{"type": "string", "format": "uuid"}}
	schemas["WorkspaceAgent"] = map[string]any{"type": "object", "required": []string{"name", "enabled", "tools", "max_steps", "max_tokens"}, "properties": map[string]any{"name": map[string]any{"type": "string", "maxLength": 160}, "instructions": map[string]any{"type": "string", "maxLength": 16000}, "enabled": map[string]any{"type": "boolean"}, "revision": map[string]any{"type": "integer", "minimum": 1, "description": "PUT 때 직전 설정 revision 필수"}, "space_ids": ids, "document_ids": ids, "database_ids": ids, "tools": map[string]any{"type": "array", "uniqueItems": true, "items": map[string]any{"type": "string", "enum": agentToolNames}}, "max_steps": map[string]any{"type": "integer", "minimum": 1, "maximum": 24}, "max_tokens": map[string]any{"type": "integer", "minimum": 1, "maximum": 262144}}}
	schemas["AgentRunCreate"] = map[string]any{"type": "object", "required": []string{"prompt", "expected_agent_version"}, "properties": map[string]any{"prompt": map[string]any{"type": "string", "maxLength": 32000}, "expected_agent_version": map[string]any{"type": "integer", "minimum": 1}}}
	schemas["AgentActionDecision"] = map[string]any{"type": "object", "required": []string{"action_hash"}, "properties": map[string]any{"action_hash": map[string]any{"type": "string", "pattern": "^[0-9a-f]{64}$"}, "confirm": map[string]any{"type": "boolean", "description": "confirm 또는 reject 중 하나만 true"}, "reject": map[string]any{"type": "boolean"}}}
	for _, route := range []struct {
		path, method, summary, body string
		accepted                    bool
	}{{"/workspaces/{id}/agents", "get", "현재 권한 내 워크스페이스 Agent 목록", "", false}, {"/workspaces/{id}/agents", "post", "워크스페이스 관리자의 지식·도구 범위 설정", "WorkspaceAgent", false}, {"/agents/{id}", "put", "revision 검사 후 Agent 정책 변경", "WorkspaceAgent", false}, {"/agents/{id}/runs", "get", "본인과 현재 사용 원본 ACL로 제한한 실행 이력", "", false}, {"/agents/{id}/runs", "post", "현재 AI 공급자와 Agent 버전을 고정한 비동기 실행", "AgentRunCreate", true}, {"/agent-runs/{id}", "get", "현재 원본 권한 내 실행 상태·불변 변경 계획", "", false}, {"/agent-runs/{id}/events", "get", "현재 세션·권한을 검증하는 영속 SSE 이벤트", "", false}, {"/agent-runs/{id}/cancel", "post", "후속 호출·출력·대기 중 변경 취소", "", false}, {"/agent-runs/{id}/actions/{action}/confirm", "post", "실제 사용자 세션에서 고정 계획 한 건 확인 또는 거절", "AgentActionDecision", true}} {
		entry, ok := paths[route.path].(map[string]any)
		if !ok {
			continue
		}
		op, ok := entry[route.method].(map[string]any)
		if !ok {
			continue
		}
		op["summary"] = route.summary
		op["tags"] = []string{"워크스페이스 Agent"}
		op["description"] = "사용자 현재 ACL/scopes ∩ 관리자 지식/도구 allowlist. 모델은 권한을 부여하지 않습니다. 설정/공급자/원본/키/세션 변경 시 중단합니다. 문서 변경은 기존 REST/버전/선택형 승인/정보보호 정책과 같은-TX 효과 영수증을 적용합니다."
		if route.body != "" {
			op["requestBody"] = map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": "#/components/schemas/" + route.body}}}}
		}
		responses, _ := op["responses"].(map[string]any)
		if route.accepted && responses != nil {
			responses["202"] = map[string]any{"description": "영속 작업 접수. 완료 또는 사용자 확인까지 이벤트를 조회하세요"}
			delete(responses, "200")
		}
		if route.path == "/agent-runs/{id}/events" {
			responses["200"] = map[string]any{"description": "SSE event: step_start, delta{text,step}, tool_result, step_end, action_required, action_decision, action_applied, status, retract. retract 수신 즉시 기존 출력·결과를 숨기세요. id/Last-Event-ID 또는 after로 재접속합니다. 완료/확인대기에서 연결은 정상 종료됩니다.", "content": map[string]any{"text/event-stream": map[string]any{"schema": text}}}
			parameters, _ := op["parameters"].([]any)
			parameters = append(parameters, map[string]any{"name": "after", "in": "query", "schema": map[string]any{"type": "integer", "minimum": 0}}, map[string]any{"name": "Last-Event-ID", "in": "header", "schema": text})
			op["parameters"] = parameters
		}
	}
}
