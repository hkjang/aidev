package server

func knowledgeQuestionsOpenAPI(paths, schemas map[string]any) {
	uuid := map[string]any{"type": "string", "format": "uuid"}
	version := map[string]any{"type": "integer", "minimum": 1, "maximum": 2147483646}
	refs := map[string]any{"type": "array", "minItems": 1, "maxItems": 32, "items": map[string]any{"type": "object", "required": []string{"id", "version"}, "properties": map[string]any{"id": uuid, "version": version, "hash": map[string]any{"type": "string", "description": "조회 결과의 원문SHA256. 신규 등록은 서버가 계산합니다."}, "attachment": packageAttachmentOpenAPISchema()["properties"].(map[string]any)["source"]}}}
	schemas["ManagedQuestionInput"] = map[string]any{"type": "object", "required": []string{"document_id", "version", "owner_id", "question", "reason", "review_due", "sources", "consent"}, "properties": map[string]any{"document_id": uuid, "version": version, "revision": version, "owner_id": uuid, "question": map[string]any{"type": "string", "description": "UTF-8 최대2000바이트"}, "reason": map[string]any{"type": "string", "description": "UTF-8 최대4000바이트"}, "review_due": map[string]any{"type": "string", "format": "date", "description": "UTC 날짜, 당일까지 유효"}, "sources": refs, "message_id": uuid, "consent": map[string]any{"type": "boolean", "enum": []bool{true}}}}
	for path, raw := range paths {
		methods, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		for method, operation := range methods {
			op, ok := operation.(map[string]any)
			if !ok {
				continue
			}
			switch path {
			case "/knowledge/questions", "/knowledge/questions/{id}":
				op["description"] = "정본 답변 문서와 모든 근거의 현재 ACL/키 scope/PII 조건을 적용합니다. 공식 표시는 담당자 확인·현재 버전·검토 기한·현재 게시 승인의 결합이며 사실성 보증이 아닙니다. 조회는 키 사용 가능, 쓰기는 사람의 브라우저 세션만 가능합니다."
				if method == "post" || method == "put" {
					op["requestBody"] = map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": "#/components/schemas/ManagedQuestionInput"}}}}
					op["security"] = []any{map[string]any{"sessionCookie": []string{}}}
					if method == "post" {
						op["responses"].(map[string]any)["201"] = map[string]any{"description": "정본 답변에 연결한 제안 상태; 자동 게시/공식 확인 없음"}
					}
				}
			case "/knowledge/questions/{id}/decision":
				op["description"] = "현재 담당자의 명시 confirmed 또는 archived 처리. 게시 승인은 답변 문서의 기존 정책을 따르며 이 작업이 게시하거나 문서를 변경하지 않습니다."
				op["requestBody"] = map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"type": "object", "required": []string{"revision", "state", "note", "consent"}, "properties": map[string]any{"revision": version, "state": map[string]any{"type": "string", "enum": []string{"confirmed", "archived"}}, "note": map[string]any{"type": "string", "description": "UTF-8 1~4000바이트"}, "consent": map[string]any{"type": "boolean", "enum": []bool{true}}}}}}}
			case "/ai/messages/{id}/question-preview":
				op["description"] = "본인의 단일 개인 질문·답변·현재 근거 미리보기. 읽기만으로 조직에 공유되지 않습니다. 직접 승격 시 본인 비공개 정본 답변에 먼저 연결하며 다른 메시지/대화 ID는 공유하지 않습니다."
			case "/ai/messages/{id}/question-draft":
				op["description"] = "현재 미리보기 해시와 consent:true로 본인의 단일 답변을 비공개 정본 초안으로 복사합니다. createDocument 공통 트랜잭션 안에서 현재 기록·근거·세션과 원자 receipt를 검증합니다. 같은 메시지 재시도는 같은 미변경 비공개 초안을 반환하며 동시 생성 충돌은409 후 재시도합니다. 질문 등록은 별도 단계입니다."
				op["requestBody"] = map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"type": "object", "required": []string{"preview_hash", "consent"}, "properties": map[string]any{"preview_hash": map[string]any{"type": "string", "pattern": "^[a-f0-9]{64}$"}, "consent": map[string]any{"type": "boolean", "enum": []bool{true}}}}}}}
			}
		}
	}
}
func knowledgeQuestionMCPTools() []mcpTool {
	return []mcpTool{
		{Name: "get_managed_question", Description: "현재 답변·모든 근거 ACL로 관리 질문과 공식 표시의 개별 조건을 조회합니다. 사실성 자동 보증이 아니며 사람의 확인/공유/게시를 대행하지 않습니다.", Scope: "document:read", InputSchema: map[string]any{"type": "object", "additionalProperties": false, "required": []string{"question_id"}, "properties": map[string]any{"question_id": map[string]any{"type": "string", "format": "uuid"}}}, Annotations: map[string]any{"readOnlyHint": true, "destructiveHint": false, "openWorldHint": false}},
		{Name: "list_managed_questions", Description: "현재 워크스페이스와 모든 근거 ACL에서 관리 질문을 최대20건씩 조회합니다. 응답 next_after로 다음 페이지를 읽습니다.", Scope: "document:read", InputSchema: map[string]any{"type": "object", "additionalProperties": false, "required": []string{}, "properties": map[string]any{"workspace_id": map[string]any{"type": "string", "format": "uuid"}, "after": map[string]any{"type": "string", "format": "uuid"}}}, Annotations: map[string]any{"readOnlyHint": true, "destructiveHint": false, "openWorldHint": false}},
	}
}
