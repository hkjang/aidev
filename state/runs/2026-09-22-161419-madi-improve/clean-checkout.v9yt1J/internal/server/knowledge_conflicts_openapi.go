package server

func knowledgeConflictMCPTools() []mcpTool {
	makeTool := func(name, description, key string) mcpTool {
		return mcpTool{Name: name, Description: description, Scope: "document:read", InputSchema: map[string]any{"type": "object", "additionalProperties": false, "properties": map[string]any{key: map[string]any{"type": "string", "format": "uuid"}}, "required": []string{key}}, Annotations: map[string]any{"readOnlyHint": true, "destructiveHint": false, "openWorldHint": false}}
	}
	return []mcpTool{makeTool("list_knowledge_conflicts", "본인의 규칙 기반 문서 비교 보고서 메타데이터만 읽습니다. 사실성·승인 판정이 아니며 원문을 변경하지 않습니다.", "workspace_id"), makeTool("get_knowledge_conflict", "본인 보고서의 양쪽 현재 문서 권한·보호 정책을 재검사하고 출처 버전/해시/구간과 개인 판단을 읽습니다. 현재 접근 불가 원문은 숨깁니다.", "run_id")}
}

func knowledgeConflictsOpenAPI(paths map[string]any) {
	for _, entry := range []struct{ path, method, summary string }{
		{"/knowledge/conflicts", "get", "본인 비교 보고서 목록(document:read)"},
		{"/knowledge/conflicts", "post", "현재 문서 2~32개 규칙 검사 및 암호화 보관(본인 로그인 세션)"},
		{"/knowledge/conflicts/{id}", "get", "양쪽 현재 ACL·정보 보호를 적용한 본인 비교 보고서"},
		{"/knowledge/conflicts/{id}", "delete", "연결된 변경안이 없는 개인 보고서 삭제(원문 유지)"},
		{"/knowledge/conflict-candidates/{id}", "get", "본인 비교 후보의 정확한 구간·판단 이력"},
		{"/knowledge/conflict-candidates/{id}/reviews", "post", "현재 원문·revision CAS를 적용한 개인 판단 기록(게시 승인 아님)"},
	} {
		item, ok := paths[entry.path].(map[string]any)
		if !ok {
			item = map[string]any{}
			paths[entry.path] = item
		}
		op, _ := item[entry.method].(map[string]any)
		if op == nil {
			continue
		}
		op["summary"], op["description"] = entry.summary, conflictNotice
		op["responses"].(map[string]any)["422"] = map[string]any{"description": "현재 정보 보호 정책으로 처리 불가"}
		if entry.method != "get" {
			op["security"] = []any{map[string]any{"cookieAuth": []string{}}}
		}
		if entry.method == "get" && entry.path == "/knowledge/conflicts" {
			op["parameters"] = []any{map[string]any{"name": "workspace_id", "in": "query", "required": true, "schema": map[string]any{"type": "string", "format": "uuid"}}}
		}
		if entry.method == "post" && entry.path == "/knowledge/conflicts" {
			op["requestBody"] = map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"type": "object", "required": []string{"workspace_id", "documents", "consent"}, "properties": map[string]any{"workspace_id": map[string]any{"type": "string", "format": "uuid"}, "consent": map[string]any{"type": "boolean", "const": true}, "documents": map[string]any{"type": "array", "minItems": 2, "maxItems": 32, "items": map[string]any{"type": "object", "required": []string{"id", "version"}, "properties": map[string]any{"id": map[string]any{"type": "string", "format": "uuid"}, "version": map[string]any{"type": "integer", "minimum": 1}}}}}}}}}
			op["responses"].(map[string]any)["201"] = map[string]any{"description": "개인 보고서 ID와 실제 검사 통계"}
		}
		if entry.method == "post" && entry.path != "/knowledge/conflicts" {
			op["requestBody"] = map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"type": "object", "required": []string{"revision", "decision", "note", "consent"}, "properties": map[string]any{"revision": map[string]any{"type": "integer", "minimum": 1}, "decision": map[string]any{"type": "string", "enum": []string{"conflict", "compatible", "needs_review"}}, "note": map[string]any{"type": "string", "minLength": 1, "maxLength": 4000}, "consent": map[string]any{"type": "boolean", "const": true}}}}}}
		}
		item[entry.method] = op
	}
}
