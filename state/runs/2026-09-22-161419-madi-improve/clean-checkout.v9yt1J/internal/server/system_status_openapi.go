package server

func systemStatusOpenAPI(paths, schemas map[string]any) {
	uuid := map[string]any{"type": "string", "format": "uuid"}
	revision := map[string]any{"type": "integer", "minimum": 1, "maximum": 9007199254740990}
	zeroRevision := map[string]any{"type": "integer", "minimum": 0, "maximum": 9007199254740990}
	confirm := map[string]any{"type": "boolean", "const": true}
	expected := map[string]any{"type": "object", "required": []string{"name"}, "description": "기대 version/deployment 중 하나는 필수. 사용자 등록 기대값이며 실제 검증 아님. 서버 UTF-8 길이 및 PII 정책 적용.", "properties": map[string]any{"name": map[string]any{"type": "string", "maxLength": 200}, "environment": map[string]any{"type": "string", "maxLength": 200}, "version": map[string]any{"type": "string", "maxLength": 500}, "deployment": map[string]any{"type": "string", "maxLength": 500}, "note": map[string]any{"type": "string", "maxLength": 4000}}}
	schemas["SystemStatusExpectation"] = map[string]any{"type": "object", "required": []string{"revision", "document_version", "owner_id", "reporter_ids", "ttl_seconds", "expected", "confirm"}, "properties": map[string]any{"revision": zeroRevision, "document_version": revision, "owner_id": uuid, "reporter_ids": map[string]any{"type": "array", "maxItems": 100, "uniqueItems": true, "items": uuid}, "ttl_seconds": map[string]any{"type": "integer", "minimum": 60, "maximum": 604800}, "expected": expected, "confirm": confirm}}
	for _, tool := range systemStatusMCPTools() {
		if tool.Name == "report_system_status" {
			properties := map[string]any{}
			for key, value := range tool.InputSchema["properties"].(map[string]any) {
				if key != "document_id" {
					properties[key] = value
				}
			}
			required := []string{}
			for _, key := range tool.InputSchema["required"].([]string) {
				if key != "document_id" {
					required = append(required, key)
				}
			}
			schemas["SystemStatusReport"] = map[string]any{"type": "object", "properties": properties, "required": required}
		}
	}
	schemas["SystemStatusDelete"] = map[string]any{"type": "object", "required": []string{"revision", "document_version", "confirm"}, "properties": map[string]any{"revision": revision, "document_version": revision, "confirm": confirm}}
	schemas["SystemStatusPolicy"] = map[string]any{"type": "object", "required": []string{"revision", "enabled", "max_ttl_seconds", "max_observation_age_seconds", "retention_days", "confirm"}, "properties": map[string]any{"revision": revision, "enabled": map[string]any{"type": "boolean"}, "max_ttl_seconds": map[string]any{"type": "integer", "minimum": 60, "maximum": 604800}, "max_observation_age_seconds": map[string]any{"type": "integer", "minimum": 60, "maximum": 604800}, "retention_days": map[string]any{"type": "integer", "minimum": 8, "maximum": 3650}, "confirm": confirm}}
	for _, entry := range []struct {
		path, method, title, schema, description string
		ordinary                                 bool
	}{
		{"/documents/{id}/system-status", "get", "현재 운영 기대값·관측·신선도·실행/수집 맥락", "", "현재 원문 ACL·document:read. 카드가 없으면 null, 최근 보고 최대50건. policy off/TTL/원문·기준변경은 별도 상태. 사용자가 관측했다고 보고한 값이며 independent verification false.", false},
		{"/documents/{id}/system-status", "put", "운영 기대값·담당·보고 주체 CAS 저장", "SystemStatusExpectation", "현재 문서 작성권한의 일반 사용자만. 전역정책 enabled 필수, 첫 revision0. 카드 기준/epoch는 증가하지만 Markdown 정본이나 게시 승인 상태는 변경하지 않음.", true},
		{"/documents/{id}/system-status", "delete", "운영 카드·보고 이력 삭제", "SystemStatusDelete", "현재 원문/카드 CAS와 명시 삭제확인. 정본·실제 시스템 유지, 운영카드/관측 이력만 삭제하며 백업 없으면 복구 불가.", true},
		{"/documents/{id}/system-status/reports", "post", "명시된 보고 주체의 실제 관측 보고", "SystemStatusReport", "document:read+write, 현재 카드 reporter_ids, 계정/키/워크스페이스/문서 권한의 교집합. 현재 card/report revision·document version·epoch CAS. UUID request_id 동일주체/동일본문 retry만200, 신규201. 미래60초/최대과거/관측시각 단조검사. 만료=min(observed_at+TTL,received_at+TTL), 재전송 연장 불가. 원문/배포/승인 변경 없음.", false},
		{"/admin/system-status/policy", "get", "운영 보고 정책과 변경 이력", "", "현재 서비스 관리자 일반 로그인만. 최대50개 정책 이력, 카드/보고 원문은 이 관리자 경로에 없음.", true},
		{"/admin/system-status/policy", "put", "운영 보고 허용·TTL·보존 정책 CAS", "SystemStatusPolicy", "기본off. 관리자 설정만으로 카드 reporter 권한 부여되지 않음. 보존8~3650일은 정리 배치에서 적용. 복원 후 off/카드epoch 변경.", true},
	} {
		methods, _ := paths[entry.path].(map[string]any)
		if methods == nil {
			methods = map[string]any{}
			paths[entry.path] = methods
		}
		security := []any{map[string]any{"cookieAuth": []string{}}}
		if !entry.ordinary {
			security = append(security, map[string]any{"bearerAuth": []string{}})
		}
		op := map[string]any{"summary": entry.title, "description": entry.description, "tags": []string{"시스템 운영 현황"}, "security": security, "responses": map[string]any{"200": map[string]any{"description": "확정 상태 또는 동일 보고 재전송 영수증"}, "201": map[string]any{"description": "새 보고 확정"}, "400": map[string]any{"description": "입력/관측시각/정책범위 오류"}, "403": map[string]any{"description": "현재 주체·키·세션·보고 권한 없음"}, "404": map[string]any{"description": "현재 접근 가능한 문서/카드 없음"}, "409": map[string]any{"description": "원문/기준/보고/epoch CAS 충돌 또는 비활성"}, "422": map[string]any{"description": "현재 정보보호 정책에서 표시/보관 불가"}}}
		if entry.path[1:6] != "admin" {
			op["parameters"] = []any{map[string]any{"name": "id", "in": "path", "required": true, "schema": uuid}}
		}
		if entry.schema != "" {
			op["requestBody"] = map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": "#/components/schemas/" + entry.schema}}}}
		}
		methods[entry.method] = op
	}
}
