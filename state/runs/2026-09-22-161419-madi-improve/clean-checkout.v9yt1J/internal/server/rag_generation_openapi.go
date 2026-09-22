package server

func ragGenerationsOpenAPI(paths, schemas map[string]any) {
	strSchema := map[string]any{"type": "string"}
	uuid := map[string]any{"type": "string", "format": "uuid"}
	revision := map[string]any{"type": "integer", "minimum": 1}
	confirmation := map[string]any{"type": "boolean", "const": true}
	mode := map[string]any{"type": "string", "enum": []string{"exact", "ann", "verify"}}
	schemas["RAGGenerationCreate"] = map[string]any{"type": "object", "required": []string{"name", "dimensions", "config", "confirm"}, "properties": map[string]any{"name": strSchema, "dimensions": map[string]any{"type": "integer", "minimum": 1, "maximum": 8192}, "config": map[string]any{"type": "object", "description": "불변 임베딩/재정렬 공급자 설정 rag_embedding_*,rag_rerank_*,rag_allow_http,rag_ca_pem만. 새 주소에는 이전 키를 자동 전달하지 않음"}, "confirm": confirmation}}
	schemas["RAGGenerationVerify"] = map[string]any{"type": "object", "required": []string{"revision", "provider_fingerprint", "query", "mode", "consent", "documents"}, "properties": map[string]any{"revision": revision, "provider_fingerprint": strSchema, "query": map[string]any{"type": "string", "description": "질문 전송 명시 동의, UTF-8 4000바이트 이내; 원문 이력 저장하지 않음"}, "mode": mode, "consent": confirmation, "documents": map[string]any{"type": "array", "minItems": 1, "maxItems": 30, "items": map[string]any{"type": "object", "required": []string{"document_id", "version"}, "properties": map[string]any{"document_id": uuid, "version": revision}}}}}
	schemas["RAGGenerationActivate"] = map[string]any{"type": "object", "required": []string{"validation_id", "state_revision", "mode", "confirm", "confirm_coverage"}, "properties": map[string]any{"validation_id": uuid, "state_revision": revision, "mode": mode, "confirm": confirmation, "confirm_coverage": confirmation}}
	schemas["RAGGenerationConfirmation"] = map[string]any{"type": "object", "required": []string{"revision", "confirm"}, "properties": map[string]any{"revision": revision, "confirm": confirmation}}
	for _, entry := range []struct{ path, method, title, schema, description string }{
		{"/workspaces/{id}/rag-generations", "get", "현재 권한의 벡터 세대 목록·상태", "", "활성 포인터/state_revision/모드/현재 RAG 설정 fingerprint, 현재 열람 가능한 문서의 동의/준비 수만 반환. 공급자 비밀과 URL query 비노출."},
		{"/workspaces/{id}/rag-generations", "post", "불변 모델·차원의 새 그림자 세대 준비", "RAGGenerationCreate", "기존 공급자/동의를 기준 세대로 보존하며 새 공급자에 자동 전송하거나 검색을 전환하지 않음. 세대별로 문서 동의 필요. 최대20개."},
		{"/rag-generations/{id}/index", "post", "실제 HNSW 인덱스 생성 작업 요청", "RAGGenerationConfirmation", "202 job_id. pgvector0.8+ 수동설치/1~2000차원. CREATE INDEX CONCURRENTLY, schema당 한 작업, 현재세션/권한/설정 및 공통jobs 취소 확인. 외부전송 없음."},
		{"/rag-generations/{id}/verify", "post", "현재 문서 cohort의 실제 정확/ANN 비교", "RAGGenerationVerify", "45초 예산. query 임베딩 실제 호출 후 현재 ACL·버전·원래 동의주체 권한 확인. 선택1~30문서/질문1건만 평가, 전체 조직 품질 보증 아님. 15분 현재세션 결합 validation_id 반환. 원문 대신 query hash와 ID/version/영수증만 저장."},
		{"/workspaces/{id}/rag-generations/activate", "post", "검증 영수증의 세대·공급자·모드 CAS 전환", "RAGGenerationActivate", "같은 세션/현재state_revision/현재문서버전·동의·설정이 일치하고 정확기준 잘림없음/Recall>=0.9인 경우만 동일TX로 settings+active pointer 갱신. 한 영수증 한 번. 이전세대 복귀도 새 검증 필수."},
		{"/rag-generations/{id}", "delete", "비활성 세대와 파생 색인 삭제", "RAGGenerationConfirmation", "현재 검색 세대 또는 진행중 작업이 있는 세대 삭제불가. 정확한 세대 HNSW 인덱스만 500ms lock timeout/5초 TX로 삭제. 원본 문서 유지. 세대 설정·동의·검증기록·파생벡터 복구에는 새 동의 필요."},
	} {
		methods, _ := paths[entry.path].(map[string]any)
		if methods == nil {
			methods = map[string]any{}
			paths[entry.path] = methods
		}
		op := map[string]any{"summary": entry.title, "description": "현재 일반 사용자 쿠키+워크스페이스 owner/admin만, 키/서비스계정/플러그인 관리호출 금지. " + entry.description, "tags": []string{"벡터 색인 세대"}, "security": []any{map[string]any{"cookieAuth": []string{}}}, "parameters": []any{map[string]any{"name": "id", "in": "path", "required": true, "schema": uuid}}, "responses": map[string]any{"200": map[string]any{"description": "현재 검증된 결과"}, "201": map[string]any{"description": "세대 생성"}, "202": map[string]any{"description": "작업 큐 등록"}, "400": map[string]any{"description": "유효한 설정/명시 동의 필요"}, "403": map[string]any{"description": "현재 관리 권한 없음"}, "409": map[string]any{"description": "세션/동의/ACL/버전/설정/CAS 변경 또는 인덱스 미준비"}, "422": map[string]any{"description": "정보보호 정책 또는 공급자 차원 불일치"}}}
		if entry.schema != "" {
			op["requestBody"] = map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": "#/components/schemas/" + entry.schema}}}}
		}
		methods[entry.method] = op
	}
	for _, path := range []string{"/documents/{id}/rag-index", "/documents/{id}/rag-index/cancel"} {
		if methods, ok := paths[path].(map[string]any); ok {
			for _, value := range methods {
				if op, ok := value.(map[string]any); ok {
					parameters, _ := op["parameters"].([]any)
					op["parameters"] = append(parameters, map[string]any{"name": "generation_id", "in": "query", "schema": uuid, "description": "지정 세대의 동의/상태/취소. 생략 시 현재 활성 세대. 다른 세대 동의나 벡터에는 영향 없음."})
				}
			}
		}
	}
}
