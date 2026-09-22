package server

import "strings"

func structuredDraftOpenAPI(path, method string, op map[string]any) {
	if !strings.HasPrefix(path, "/knowledge/structured-drafts") && !strings.HasSuffix(path, "/structured-context") && !strings.HasSuffix(path, "/structured-draft") {
		return
	}
	uuid := map[string]any{"type": "string", "format": "uuid"}
	positive := map[string]any{"type": "integer", "minimum": 1}
	consent := map[string]any{"const": true}
	obj := func(p map[string]any, required ...string) map[string]any {
		return map[string]any{"type": "object", "additionalProperties": false, "properties": p, "required": required}
	}
	body := func(v any) {
		op["requestBody"] = map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{"schema": v}}}
	}
	field := obj(map[string]any{"property_id": map[string]any{"type": "string"}, "value": map[string]any{"description": "선택한 일반 속성 타입에 맞는 JSON 값. 실행·계산·관계·사용자 속성은 제외"}, "quote": map[string]any{"type": "string", "description": "선택 원문의 정확한 연속 UTF-8 인용1~4096바이트"}, "start_byte": map[string]any{"type": "integer", "minimum": 0, "description": "선택 구간 안의 상대 UTF-8 시작 위치. 생략하면 인용이 정확히 한 번 나타나야 함"}}, "property_id", "value", "quote")
	op["description"] = "원문→한 개 데이터 행의 근거 연결 초안. AI 출력은 값·인용 위치·타입을 별도로 검사하며 사실성을 보증하지 않습니다. 생성은 선택 원문32KiB와 일반 속성1~32개만 전송하고 SSE기본/설정 max_tokens최대262144, 실제 응답64KiB 한도입니다. 완료만으로 저장하지 않습니다. 미반영 초안24시간, 완료 근거는 현재 원문/DB ACL과 보존 정책 하에 명시 삭제 전까지 유지. 원문 물리 삭제 시 근거 사본만 삭제되고 이미 공유된 DB 행은 회수되지 않습니다."
	if method != "get" || strings.HasSuffix(path, "/structured-context") {
		op["security"] = []any{map[string]any{"sessionCookie": []string{}}}
	}
	switch {
	case strings.HasSuffix(path, "/structured-context"):
		args, _ := op["parameters"].([]any)
		op["parameters"] = append(args, map[string]any{"name": "database_id", "in": "query", "required": true, "schema": uuid})
	case strings.HasSuffix(path, "/structured-draft"):
		body(obj(map[string]any{"database_id": uuid, "expected_version": positive, "start_byte": map[string]any{"type": "integer", "minimum": 0}, "end_byte": positive, "selected_text": map[string]any{"type": "string"}, "property_ids": map[string]any{"type": "array", "minItems": 1, "maxItems": 32, "uniqueItems": true, "items": map[string]any{"type": "string"}}, "provider_fingerprint": map[string]any{"type": "string"}, "schema_hash": map[string]any{"type": "string"}, "destination_hash": map[string]any{"type": "string"}, "consent": consent}, "database_id", "expected_version", "start_byte", "end_byte", "selected_text", "property_ids", "provider_fingerprint", "schema_hash", "destination_hash", "consent"))
		op["responses"] = map[string]any{"200": map[string]any{"description": "text/event-stream: text(미완료), proposal{draft_ticket,fields}(완료), 오류시retract. 저장·자동 행 생성 없음", "content": map[string]any{"text/event-stream": map[string]any{"schema": map[string]any{"type": "string"}}}}}
	case path == "/knowledge/structured-drafts" && method == "post":
		body(obj(map[string]any{"ticket": map[string]any{"type": "string", "description": "완료된 제안의 암호화30분 ticket; 최대1MiB"}, "consent": consent}, "ticket", "consent"))
	case path == "/knowledge/structured-drafts" && method == "get":
		op["parameters"] = []any{map[string]any{"name": "workspace_id", "in": "query", "required": true, "schema": uuid}, map[string]any{"name": "after", "in": "query", "schema": uuid}}
	case strings.HasSuffix(path, "/preview"):
		body(obj(map[string]any{"revision": positive, "fields": map[string]any{"type": "array", "minItems": 1, "maxItems": 32, "items": field}}, "revision", "fields"))
	case strings.HasSuffix(path, "/apply"):
		body(obj(map[string]any{"review_ticket": map[string]any{"type": "string", "description": "현재 초안·원문·대상 속성/공유 fingerprint 및 검토한 값에 결합된10분 ticket"}, "consent": consent}, "review_ticket", "consent"))
	case method == "delete":
		body(obj(map[string]any{"revision": positive, "consent": consent}, "revision", "consent"))
	}
}
func structuredDraftMCPTools() []mcpTool {
	return []mcpTool{
		{Name: "list_structured_drafts", Scope: "document:read", Description: "본인의 현재 원문+데이터베이스 ACL에서 구조화 초안/반영 이력을20건씩 읽습니다. database:read도 필요하며 AI 호출·값 검토·새 행 생성은 대행하지 않습니다.", InputSchema: map[string]any{"type": "object", "additionalProperties": false, "required": []string{}, "properties": map[string]any{"workspace_id": map[string]any{"type": "string", "format": "uuid"}, "after": map[string]any{"type": "string", "format": "uuid"}}}, Annotations: map[string]any{"readOnlyHint": true, "destructiveHint": false, "openWorldHint": false}},
		{Name: "get_structured_draft", Scope: "document:read", Description: "본인의 구조화 초안과 값별 인용·사람이 수정한 값·현재성·반영 이력을 조회합니다. database:read와 현재 원문/대상 DB ACL도 필요합니다. 인용 해시 일치와 값의 사실성은 별개입니다.", InputSchema: map[string]any{"type": "object", "additionalProperties": false, "required": []string{"draft_id"}, "properties": map[string]any{"draft_id": map[string]any{"type": "string", "format": "uuid"}}}, Annotations: map[string]any{"readOnlyHint": true, "destructiveHint": false, "openWorldHint": false}},
	}
}
