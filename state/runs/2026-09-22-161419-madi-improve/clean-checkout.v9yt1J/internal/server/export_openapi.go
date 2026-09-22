package server

import "strings"

func transferOpenAPI(path, method string, op map[string]any) {
	object := func(props map[string]any, required ...string) map[string]any {
		return map[string]any{"type": "object", "properties": props, "required": required}
	}
	text := map[string]any{"type": "string"}
	body := func(schema map[string]any) {
		op["requestBody"] = map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{"schema": schema}}}
	}
	if path == "/exports" && method == "post" {
		body(object(map[string]any{"workspace_id": map[string]any{"type": "string", "format": "uuid"}, "format": map[string]any{"type": "string", "enum": []string{"markdown", "portable", "html", "json", "csv"}}, "document_ids": map[string]any{"type": "array", "minItems": 1, "maxItems": 1000, "uniqueItems": true, "items": map[string]any{"type": "string", "format": "uuid"}}, "database_id": map[string]any{"type": "string", "format": "uuid"}}, "workspace_id", "format"))
		op["description"] = "비동기 내보내기. CSV는 database_id, 나머지는 document_ids가 필요합니다. 원문 Markdown은 바이트 그대로, portable은 상대 링크 변환. 문서 scope와 DB scope는 각 실제 자원별로 검사합니다. 결과는 암호화·24시간 보관하며 원래 세션/키와 현재 원본/ACL/정책 지문에 묶입니다."
	}
	if strings.HasPrefix(path, "/exports/") && strings.HasSuffix(path, "/download") {
		responses, _ := op["responses"].(map[string]any)
		if responses == nil {
			responses = map[string]any{}
			op["responses"] = responses
		}
		responses["200"] = map[string]any{"description": "ZIP / UTF-8 JSON / UTF-8 CSV. no-store. 현재 ACL·원본 버전·요청 세션/키를 재검사하며 중간 변경 시 다운로드를 중단합니다.", "content": map[string]any{"application/octet-stream": map[string]any{"schema": map[string]any{"type": "string", "format": "binary"}}}}
		responses["409"] = map[string]any{"description": "원문/첨부/권한/설정/원래 세션 변경. 재생성 필요"}
		responses["410"] = map[string]any{"description": "임시 결과 파일이 만료되거나 폐기됨"}
	}
	if path == "/admin/exports/settings" && method == "put" {
		body(object(map[string]any{"enabled": map[string]any{"type": "boolean"}, "max_archive_mb": map[string]any{"type": "integer", "minimum": 1, "maximum": 100}, "revision": map[string]any{"type": "integer", "minimum": 1}}, "enabled", "max_archive_mb", "revision"))
	}
	if path == "/migrations" && method == "post" {
		op["description"] = "UTF-8 Markdown/Obsidian/Notion/HTML/CSV/JSON의 암호화 원본 staging과 미리보기 PG job. workspace_id,space_id(선택),format 쿼리. multipart file<=50MB. JSON은 madi-json-vault v1 또는 title/markdown/tags/aliases 목록. 원래 ID/권한으로 기존 문서를 덮어쓰지 않습니다."
		op["requestBody"] = map[string]any{"required": true, "content": map[string]any{"multipart/form-data": map[string]any{"schema": object(map[string]any{"file": map[string]any{"type": "string", "format": "binary"}}, "file")}}}
		op["parameters"] = []any{map[string]any{"name": "workspace_id", "in": "query", "required": true, "schema": text}, map[string]any{"name": "space_id", "in": "query", "schema": text}, map[string]any{"name": "format", "in": "query", "required": true, "schema": map[string]any{"type": "string", "enum": []string{"markdown", "obsidian", "notion", "html", "csv", "json"}}}}
	}
	if strings.HasPrefix(path, "/migrations/") && strings.HasSuffix(path, "/run") && method == "post" {
		body(object(map[string]any{"confirmation": map[string]any{"type": "string", "const": "IMPORT"}}, "confirmation"))
	}
}
