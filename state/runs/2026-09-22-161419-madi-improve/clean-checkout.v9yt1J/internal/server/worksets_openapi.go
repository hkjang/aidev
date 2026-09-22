package server

func worksetsOpenAPI(paths, schemas map[string]any) {
	uuid := map[string]any{"type": "string", "format": "uuid"}
	version := map[string]any{"type": "integer", "minimum": 1, "maximum": 2147483647}
	schemas["WorksetItem"] = map[string]any{"type": "object", "additionalProperties": false, "required": []string{"kind", "resource_id", "context"}, "properties": map[string]any{"kind": map[string]any{"type": "string", "enum": []string{"document", "database", "task"}}, "resource_id": uuid, "context": map[string]any{"type": "object", "description": "문서: version,mode(read/edit),line,scroll_y,숫자 선택 위치/range(DOM 숫자경로32단계 이하). 할 일: task_id,version,line. DB: view_id 또는 기존 저장된 보기의 view/filters/sorts/columns/board_property_id/date_property_id. 종류별 키 허용목록,32768바이트 한도. 원문·임의 URL·HTML 불가."}}}
	schemas["WorksetWrite"] = map[string]any{"type": "object", "required": []string{"name", "kind", "items"}, "properties": map[string]any{"workspace_id": uuid, "name": map[string]any{"type": "string", "description": "비어 있지 않은 이름, UTF-8 최대200바이트. 현재 민감정보 정책 적용"}, "kind": map[string]any{"type": "string", "enum": []string{"workset", "reference"}}, "items": map[string]any{"type": "array", "maxItems": 20, "items": map[string]any{"$ref": "#/components/schemas/WorksetItem"}}, "expected_version": version}, "description": "POST workspace_id 필수,PUT expected_version 필수. 본문 최대65536바이트, 개인당 workspace당 최대40개. reference는 문서만 최대6개. 현재 읽기 가능한 같은 workspace의 참조만 신규추가하며 기존 비공개 슬롯은 원래 ID/문맥 그대로만 보존 가능."}
	schemas["WorksetCAS"] = map[string]any{"type": "object", "required": []string{"expected_version"}, "properties": map[string]any{"expected_version": version}}
	schemas["DocumentCleanupPreview"] = map[string]any{"type": "object", "required": []string{"expected_version"}, "properties": map[string]any{"expected_version": version, "title": map[string]any{"type": "string", "description": "반환된 제목 후보 중 명시 선택한 값"}, "tags": map[string]any{"type": "array", "maxItems": 32, "items": map[string]any{"type": "string"}, "description": "반환된 태그 후보 중 선택한 값"}}, "description": "현재 원문1MiB 이하·현재 정책 기반 규칙 후보 및 선택 변경 미리보기. 쓰기는 하지 않음. after의 title/tags/markdown을 기존 PUT 문서 version CAS로 명시 적용. Front Matter tags 변경 시 YAML 재직렬화하며 다른 키/주석/본문 보존."}
	for _, entry := range []struct{ path, method, summary, description, schema string }{
		{"/worksets", "get", "나의 작업 묶음 목록", "workspace_id 필수. 본인 소유만 최대40개, 원문 없이 이름/종류/저장참조 수/수정시각. 서비스 관리자도 다른 사용자 목록 우회 불가.", ""},
		{"/worksets", "post", "개인 작업 묶음 또는 참고 선반 생성", "공개 범위는 본인 고정. 원문을 복제하거나 원문 권한을 바꾸지 않습니다.", "WorksetWrite"},
		{"/worksets/{id}", "get", "현재 권한으로 저장한 맥락 조회", "workset과 resolved 반환. 이미 저장한 접근 불가 슬롯은 kind/resource_id/available:false만 반환. 버전이 바뀐 문서 위치는 복원하지 않으며 DB 보기/필터는 현재 속성과 ACL을 검사합니다.", ""},
		{"/worksets/{id}", "put", "확인한 작업 묶음 버전만 변경", "전체 참조 구성을 같은 TX에서 검증·교체. CAS충돌은409. 숨긴 기존 슬롯을 새로 복제하거나 다른 숨긴 ID로 바꿀 수 없습니다.", "WorksetWrite"},
		{"/worksets/{id}", "delete", "개인 참조 묶음만 삭제", "expected_version 필수. 실제 문서/DB/할 일은 삭제하지 않습니다.", "WorksetCAS"},
		{"/documents/{id}/passport", "get", "현재 문서 여권 조회", "현재 읽기 권한으로 소유/버전/등급/게시상태/업무 유효기간의 사실 목록만 반환. 원문과 숨긴 대상 정보 없음. 정확성 인증이나 게시 승인이 아닙니다.", ""},
		{"/documents/{id}/cleanup-preview", "post", "본문 기반 제목·태그 후보와 정본 변경 비교", "현재 작성 권한 및 버전 필수. 후보는 외부AI가 아닌 제한된 Markdown 분석으로 만들고 민감정보 정책을 적용합니다. 자동 이동·공유·게시하지 않습니다.", "DocumentCleanupPreview"},
	} {
		methods, _ := paths[entry.path].(map[string]any)
		op, _ := methods[entry.method].(map[string]any)
		if op == nil {
			continue
		}
		op["summary"], op["description"] = entry.summary, entry.description
		op["security"] = []any{map[string]any{"cookieAuth": []string{}}}
		if entry.schema != "" {
			op["requestBody"] = map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": "#/components/schemas/" + entry.schema}}}}
		}
		if entry.path == "/worksets" && entry.method == "get" {
			op["parameters"] = []any{map[string]any{"name": "workspace_id", "in": "query", "required": true, "schema": uuid}}
		}
	}
}
