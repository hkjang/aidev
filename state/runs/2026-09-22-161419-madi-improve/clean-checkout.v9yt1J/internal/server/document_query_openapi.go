package server

func documentQueryMCPTools() []mcpTool {
	props := map[string]any{"document_id": map[string]any{"type": "string", "format": "uuid"}, "document_version": map[string]any{"type": "integer", "minimum": 1}, "query_hash": map[string]any{"type": "string", "minLength": 64, "maxLength": 64}, "parameters": map[string]any{"type": "object", "maxProperties": 8}}
	return []mcpTool{
		{Name: "list_document_queries", Description: "현재 원문 안의 madi-query 선언형 조회 정의와 hash/version을 읽습니다. 실행하거나 원문을 수정하지 않습니다.", Scope: "document:read", InputSchema: map[string]any{"type": "object", "additionalProperties": false, "required": []string{"document_id"}, "properties": map[string]any{"document_id": props["document_id"]}}, Annotations: map[string]any{"readOnlyHint": true, "destructiveHint": false, "openWorldHint": false}},
		{Name: "execute_document_query", Description: "방금 읽은 문서 버전과 원문 조회 hash로 제한된 문서·할 일·관계 표를 실행합니다. 현재 ACL/키 범위만 적용하며 SQL/JS나 임의 정의는 받지 않습니다. 결과는 스냅샷이고 원문·승인·공유 사본에 저장하지 않습니다.", Scope: "document:read", InputSchema: map[string]any{"type": "object", "additionalProperties": false, "required": []string{"document_id", "document_version", "query_hash"}, "properties": props}, Annotations: map[string]any{"readOnlyHint": true, "destructiveHint": false, "openWorldHint": false}},
	}
}
func documentQueryOpenAPI(paths, schemas map[string]any) {
	schemas["DocumentQueryExecute"] = map[string]any{"type": "object", "additionalProperties": false, "required": []string{"document_version", "query_hash"}, "properties": map[string]any{"document_version": map[string]any{"type": "integer", "minimum": 1}, "query_hash": map[string]any{"type": "string", "minLength": 64, "maxLength": 64}, "parameters": map[string]any{"type": "object", "maxProperties": 8}}}
	for _, route := range []struct{ Suffix, Method, Summary string }{{"", "get", "원문의 선언형 조회 정의 조회"}, {"/execute", "post", "저장된 조회 정의의 현재 ACL 표 실행"}, {"/check", "post", "표시 결과의 현재 권한·버전 검증"}} {
		op := map[string]any{"summary": route.Summary, "tags": []string{"문서 선언형 조회"}, "description": "document:read 및 현재 부모/모든 사용 문서 ACL 필요. 읽기 전용 POST에도 브라우저 CSRF 적용. 기능 flag document-queries. 플러그인 브리지는 미노출. 후보 최대2,000·8MiB, 본문256KiB/조회문서1MiB, 행100, 실행슬롯4/2초 예산. 제한·실행시각을 응답하며 전체 결과/최신 지속성을 보증하지 않습니다.", "security": []map[string]any{{"cookieAuth": []string{}}, {"bearerAuth": []string{}}}, "parameters": []any{map[string]any{"name": "id", "in": "path", "required": true, "schema": map[string]any{"type": "string", "format": "uuid"}}}, "responses": map[string]any{"200": map[string]any{"description": "정의 또는 제한된 결과/현재성"}, "403": map[string]any{"description": "계정·키·기능 접근 불가"}, "409": map[string]any{"description": "정의·원문·현재성 변경"}, "408": map[string]any{"description": "시간 예산 초과"}, "413": map[string]any{"description": "정의/결과 크기 초과"}, "429": map[string]any{"description": "동시 실행 슬롯 초과"}}}
		if route.Method == "post" {
			schema := any(map[string]any{"$ref": "#/components/schemas/DocumentQueryExecute"})
			if route.Suffix == "/check" {
				schema = map[string]any{"type": "object", "additionalProperties": false, "required": []string{"validation_token"}, "properties": map[string]any{"validation_token": map[string]any{"type": "string", "maxLength": 786432}}}
			}
			op["requestBody"] = map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{"schema": schema}}}
		}
		paths["/documents/{id}/queries"+route.Suffix] = map[string]any{route.Method: op}
	}
}
