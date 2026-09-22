package server

func searchOperationsOpenAPI(paths, schemas map[string]any) {
	ragGenerationsOpenAPI(paths, schemas)
	text := map[string]any{"type": "string"}
	if path, ok := paths["/search"].(map[string]any); ok {
		if op, ok := path["get"].(map[string]any); ok {
			parameters, _ := op["parameters"].([]any)
			for _, item := range []struct{ name, description string }{{"group", "document: 같은 문서의 결과를 먼저 묶은 뒤 페이지 분할. metadata.matches 최대3개, matched_count 포함"}, {"cursor", "암호화된 next_cursor. 현재 사용자/워크스페이스/검색조건/사전 버전에 결합. 15분 만료, offset과 동시 사용 불가"}} {
				parameters = append(parameters, map[string]any{"name": item.name, "in": "query", "schema": text, "description": item.description})
			}
			op["parameters"] = parameters
			op["description"] = "현재 권한의 문서·블록·코드·할 일·파일·댓글·DB 검색. NFKC·공백 정규화와 워크스페이스 공유 용어 사전을 적용하며 한국어 형태소 분석기는 아닙니다. interpretation에 적용된 부분/동의어/경고를 반환합니다. 원문 조각은 현재 문서 버전만 사용합니다. 결과는 실시간이며 cursor는 고정 DB 스냅샷이 아닙니다. raw 검색어는 감사로그에 남기지 않으며 개인 검색 이력은 별도 명시 동의가 필요합니다."
			if responses, ok := op["responses"].(map[string]any); ok {
				responses["200"] = map[string]any{"description": "{results,has_more,next_cursor,next_offset,group,interpretation,outcome,history_status}. outcome=matched/no_match_in_current_scope/index_pending. index_pending은 현재 접근 가능한 색인의 갱신 대기일 뿐 해당 검색어의 비공개 문서 존재를 뜻하지 않습니다."}
				responses["409"] = map[string]any{"description": "outcome=search_changed: cursor 만료/사용자·필터·사전 변경. 첫 페이지부터 재조회"}
				responses["504"] = map[string]any{"description": "outcome=timeout: 8초 읽기 예산 초과. 조건을 좁히거나 재시도"}
			}
		}
	}
	schemas["SearchDictionaryEntry"] = map[string]any{"type": "object", "required": []string{"canonical", "aliases"}, "properties": map[string]any{"id": map[string]any{"type": "string", "format": "uuid"}, "canonical": map[string]any{"type": "string", "description": "한 줄, UTF-8 200바이트 이하"}, "aliases": map[string]any{"type": "array", "maxItems": 12, "items": text}}}
	schemas["SearchDictionarySave"] = map[string]any{"type": "object", "required": []string{"revision", "entries", "confirm_shared"}, "properties": map[string]any{"revision": map[string]any{"type": "integer", "minimum": 0}, "confirm_shared": map[string]any{"type": "boolean", "const": true}, "entries": map[string]any{"type": "array", "maxItems": 300, "items": map[string]any{"$ref": "#/components/schemas/SearchDictionaryEntry"}}}}
	schemas["SearchDiagnostic"] = map[string]any{"type": "object", "required": []string{"confirm"}, "properties": map[string]any{"confirm": map[string]any{"type": "boolean", "const": true}, "run_plan": map[string]any{"type": "boolean"}, "filters": map[string]any{"type": "object", "additionalProperties": text, "description": "GET /search 허용 필터. limit=20으로 제한"}}}
	schemas["SearchEvaluation"] = map[string]any{"type": "object", "required": []string{"confirm", "top_k", "cases"}, "properties": map[string]any{"confirm": map[string]any{"type": "boolean", "const": true}, "top_k": map[string]any{"type": "integer", "minimum": 1, "maximum": 20}, "cases": map[string]any{"type": "array", "minItems": 1, "maxItems": 30, "items": map[string]any{"type": "object", "required": []string{"query", "expected_document_ids"}, "properties": map[string]any{"query": text, "expected_document_ids": map[string]any{"type": "array", "minItems": 1, "maxItems": 20, "items": map[string]any{"type": "string", "format": "uuid"}}}}}}}
	for _, entry := range []struct{ path, method, summary, schema string }{{"search-dictionary", "get", "조직 용어 사전과 CAS revision 조회", ""}, {"search-dictionary", "put", "공유 동의 후 조직 용어 사전 CAS 저장", "SearchDictionarySave"}, {"search-diagnostics", "post", "현재 권한의 검색 해석·실행 시간·읽기 전용 실제 실행 계획", "SearchDiagnostic"}, {"search-evaluation/sample", "get", "공개 합성 한국어 평가셋 다운로드", ""}, {"search-evaluation", "post", "제공한 현재 권한의 정답 문서로 Recall@k/MRR@k 실제 계산", "SearchEvaluation"}} {
		path := "/workspaces/{id}/" + entry.path
		methods, _ := paths[path].(map[string]any)
		if methods == nil {
			methods = map[string]any{}
			paths[path] = methods
		}
		op := map[string]any{"summary": entry.summary, "tags": []string{"검색 운영"}, "security": []any{map[string]any{"cookieAuth": []string{}}}, "parameters": []any{map[string]any{"name": "id", "in": "path", "required": true, "schema": map[string]any{"type": "string", "format": "uuid"}}}, "description": "현재 활성 일반 사용자 쿠키와 워크스페이스 owner/admin 역할이 모두 필요합니다. 서비스 관리자라도 멤버십을 우회하지 않으며 키·플러그인은 허용하지 않습니다. 진단·평가 원문과 미열람 모집단 수는 감사로그/개인 검색 이력에 저장하지 않습니다. 실제 EXPLAIN ANALYZE는 읽기 전용 TX/3초 statement 제한으로만 실행하며 전역 설정을 변경하지 않습니다. 사전은 구성원 검색에 공유되고 정보보호 정책을 적용합니다.", "responses": map[string]any{"200": map[string]any{"description": "실제 결과"}, "400": map[string]any{"description": "형식 또는 명시 확인 필요"}, "403": map[string]any{"description": "현재 권한 없음"}, "409": map[string]any{"description": "사전 revision 또는 접근 범위 변경"}, "422": map[string]any{"description": "정보 보호 정책 위반"}}}
		if entry.schema != "" {
			op["requestBody"] = map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": "#/components/schemas/" + entry.schema}}}}
		}
		methods[entry.method] = op
	}
}
