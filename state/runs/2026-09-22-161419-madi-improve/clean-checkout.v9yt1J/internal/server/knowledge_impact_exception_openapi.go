package server

func impactExceptionOpenAPI(paths, schemas map[string]any) {
	uuid := map[string]any{"type": "string", "format": "uuid"}
	version := map[string]any{"type": "integer", "minimum": 1, "maximum": 2147483647}
	schemas["ImpactExceptionRequest"] = map[string]any{"type": "object", "required": []string{"review_revision", "source_version", "target_version", "reason", "valid_until", "confirm"}, "properties": map[string]any{"review_revision": version, "source_version": version, "target_version": version, "reason": map[string]any{"type": "string", "minLength": 1, "maxLength": 4000, "description": "최대 UTF-8 4000바이트, 현재 PII 정책 적용, 암호화 저장"}, "valid_until": map[string]any{"type": "string", "format": "date-time", "description": "현재 이후, 최대90일"}, "confirm": map[string]any{"type": "boolean", "const": true}}}
	for _, entry := range []struct{ path, method, title string }{
		{"/knowledge/impact-reviews/{id}/exceptions", "get", "현재 양쪽 ACL로 최근20개 예외 요청·현재 유효 여부 조회"},
		{"/knowledge/impact-reviews/{id}/exceptions", "post", "명시 정책·현재 버전·기간에 대한 별도 예외 승인 요청"},
		{"/knowledge/impact-exceptions/{id}", "get", "암호화된 예외 근거와 실제 승인 결과 조회"},
	} {
		methods, _ := paths[entry.path].(map[string]any)
		if methods == nil {
			methods = map[string]any{}
			paths[entry.path] = methods
		}
		security := []any{map[string]any{"cookieAuth": []string{}}}
		if entry.method == "get" {
			security = append(security, map[string]any{"bearerAuth": []string{}})
		}
		op := map[string]any{"summary": entry.title, "description": "approval_enabled=false이면 절차 비노출. impact_exception 명시 승인 정책 필수. 문서 정본·게시·실행 권한은 변경하지 않음. current/effective는 현재 원문·대상·검토 revision·관계 생성·정책·기한의 교집합이며 과거 approved 자체가 현재 유효함을 보장하지 않음. 결정·취소는 기존 /approvals/requests/{id}/decisions의 실제 사용자 세션에서 처리. 자기 승인 금지.", "security": security, "tags": []string{"변경 영향 예외"}, "parameters": []any{map[string]any{"name": "id", "in": "path", "required": true, "schema": uuid}}, "responses": map[string]any{"200": map[string]any{"description": "현재 권한에서 읽은 이력·근거"}, "201": map[string]any{"description": "예외와 승인 요청 같은 TX 확정"}, "400": map[string]any{"description": "입력·기간·명시확인 오류"}, "403": map[string]any{"description": "현재 계정·세션·권한 없음"}, "404": map[string]any{"description": "비활성 또는 접근 가능한 대상 없음"}, "409": map[string]any{"description": "현재 원문·검토·관계·정책 변경 또는 기존 pending 요청"}, "422": map[string]any{"description": "정보보호 정책 위반"}}}
		if entry.method == "post" {
			op["requestBody"] = map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": "#/components/schemas/ImpactExceptionRequest"}}}}
		}
		methods[entry.method] = op
	}
}
