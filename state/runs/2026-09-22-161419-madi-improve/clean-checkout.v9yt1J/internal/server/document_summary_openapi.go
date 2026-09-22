package server

func documentSummaryOpenAPI(paths, schemas map[string]any) {
	properties := map[string]any{}
	for _, key := range []string{"id", "workspace_id", "owner_id"} {
		properties[key] = map[string]any{"type": "string", "format": "uuid"}
	}
	for _, key := range []string{"parent_id", "space_id"} {
		properties[key] = map[string]any{"type": []string{"string", "null"}, "format": "uuid"}
	}
	for _, key := range []string{"created_at", "updated_at"} {
		properties[key] = map[string]any{"type": "string", "format": "date-time"}
	}
	properties["deleted_at"] = map[string]any{"type": []string{"string", "null"}, "format": "date-time"}
	for key, max := range map[string]int{"title": 500, "icon": 64, "status": 32, "visibility": 32, "excerpt": 160} {
		properties[key] = map[string]any{"type": "string", "maxLength": max}
	}
	properties["version"] = map[string]any{"type": "integer", "minimum": 1}
	for _, key := range []string{"can_write", "is_favorite", "tags_truncated", "aliases_truncated"} {
		properties[key] = map[string]any{"type": "boolean"}
	}
	for _, key := range []string{"tags", "aliases"} {
		properties[key] = map[string]any{"type": "array", "maxItems": 32, "items": map[string]any{"type": "string", "maxLength": 200}, "description": "원형을 유지하는 값만. 각 UTF-8 200바이트 및 JSON 인코딩 256바이트 이하. 생략 여부는 *_truncated로 표시하며 전체 속성은 개별 GET"}
	}
	schemas["DocumentSummary"] = map[string]any{"type": "object", "additionalProperties": false, "properties": properties, "required": []string{"id", "workspace_id", "title", "excerpt", "tags", "aliases", "tags_truncated", "aliases_truncated"}, "description": "탐색 전용의 크기가 제한된 문서 요약. markdown과 block_metadata는 포함하지 않으며 GET /documents/{id}에서만 원문과 전체 속성 조회"}
	if path, ok := paths["/documents"].(map[string]any); ok {
		if op, ok := path["get"].(map[string]any); ok {
			op["summary"] = "문서 요약 목록·검색 (전체 원문 제외)"
			op["description"] = "document:read 필수. q 유무·공백과 무관하며 search:read만으로 호출 불가. 제목·본문·태그로 찾되 응답은 제한된 요약만 반환합니다. excerpt 160문자, 태그/alias 각각 최대32개. 전체 원문·블록메타·속성은 GET /documents/{id}. 기본100개/최대2000개 및 offset 페이지 사용. 검색어 원문은 공용 감사 로그에 저장하지 않습니다."
			op["responses"].(map[string]any)["200"] = map[string]any{"description": "크기가 제한된 문서 요약 배열", "content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"type": "array", "maxItems": 2000, "items": map[string]any{"$ref": "#/components/schemas/DocumentSummary"}}}}}
		}
	}
	if path, ok := paths["/spaces/{id}/documents"].(map[string]any); ok {
		if op, ok := path["get"].(map[string]any); ok {
			op["summary"] = "공간·하위 공간 문서 요약 목록 (최대2000개)"
			op["description"] = "document:read 필수. database:read 또는 search:read만으로 호출 불가. 현재 공간과 문서 ACL을 모두 적용하며 markdown/block_metadata를 반환하지 않습니다. 전체 원문과 속성은 개별 GET /documents/{id}."
			op["responses"].(map[string]any)["200"] = map[string]any{"description": "크기가 제한된 문서 요약 배열", "content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"type": "array", "maxItems": 2000, "items": map[string]any{"$ref": "#/components/schemas/DocumentSummary"}}}}}
		}
	}
}
