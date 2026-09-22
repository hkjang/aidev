package server

func supportOpenAPI(paths, schemas map[string]any) {
	schemas["SupportStart"] = map[string]any{"type": "object", "required": []string{"target_id", "reason", "duration_minutes"}, "properties": map[string]any{"target_id": map[string]any{"type": "string", "format": "uuid", "description": "본인과 다른 활성 일반 사용자. 공통 워크스페이스 필수"}, "reason": map[string]any{"type": "string", "minLength": 10, "maxLength": 500, "description": "감사 기록에 남을 사유. 정보보호 정책에 따라 차단·마스킹될 수 있습니다"}, "duration_minutes": map[string]any{"type": "integer", "minimum": 1, "maximum": 30, "description": "현재 support_max_minutes 이하"}}}
	for path, value := range paths {
		if len(path) < 9 || path[:9] != "/support/" {
			continue
		}
		entry, ok := value.(map[string]any)
		if !ok {
			continue
		}
		for method, raw := range entry {
			op, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			op["tags"] = []string{"읽기 전용 지원 진단"}
			op["security"] = []any{map[string]any{"cookieAuth": []string{}}}
			op["description"] = "서비스 관리자 원래 쿠키 세션 전용. 키·서비스 계정·플러그인 사용 불가. support_enabled 및 명시적 support_operator_ids 권한이 필요하며 계정을 전환하지 않습니다. 관리자 현재 ACL ∩ 대상 현재 ACL로 문서·DB·참조를 제한합니다. 개인 문서 권한 우회·개인 AI 이력·쓰기·AI·외부 전송·첨부 다운로드는 제공하지 않습니다. 다른 일반 서비스 API에는 원래 관리자 권한이 그대로 적용됩니다."
			if path == "/support/sessions" && method == "post" {
				op["requestBody"] = map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": "#/components/schemas/SupportStart"}}}}
				if responses, ok := op["responses"].(map[string]any); ok {
					delete(responses, "200")
					responses["201"] = map[string]any{"description": "원 세션에 결합된 진단 및 마스킹 여부; 담당자당 활성 진단 최대 3개"}
				}
			}
			if path == "/support/sessions/{id}" && method == "get" {
				parameters, _ := op["parameters"].([]any)
				for _, key := range []string{"document_ids", "database_ids", "formula_workspace_id"} {
					parameters = append(parameters, map[string]any{"name": key, "in": "query", "schema": map[string]any{"type": "string"}, "description": "화면에 표시 중인 자료의 현재 접근을 재검증하는 선택적 heartbeat. ID 목록은 쉼표 구분, 종류별 최대 101개. formula_workspace_id는 단일 UUID"})
				}
				op["parameters"] = parameters
			}
		}
	}
}
