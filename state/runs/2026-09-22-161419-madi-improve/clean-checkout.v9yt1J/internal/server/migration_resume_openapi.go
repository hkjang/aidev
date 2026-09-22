package server

import "strings"

func migrationResumeOpenAPI(path, method string, op map[string]any) {
	if !strings.HasPrefix(path, "/migrations/sessions") && path != "/admin/migration/settings" {
		return
	}
	text := map[string]any{"type": "string"}
	integer := map[string]any{"type": "integer", "minimum": 1}
	uuid := map[string]any{"type": "string", "format": "uuid"}
	object := func(props map[string]any, required ...string) map[string]any {
		return map[string]any{"type": "object", "properties": props, "required": required}
	}
	body := func(schema map[string]any) {
		op["requestBody"] = map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{"schema": schema}}}
	}
	op["description"] = "본인 소유의 비공개 재개 이관. 현재 workspace/space document:write ACL과 API 키·플러그인 제한을 확인합니다. CSV는 database:write도 필요합니다. 준비 원본/정본은 암호화되며 metadata 목록은 본문·암호문을 포함하지 않습니다."
	switch {
	case path == "/migrations/sessions" && method == "post":
		body(object(map[string]any{"workspace_id": uuid, "space_id": uuid, "source_key": map[string]any{"type": "string", "minLength": 1, "maxLength": 512}, "label": map[string]any{"type": "string", "minLength": 1, "maxLength": 200}, "format": map[string]any{"enum": []string{"markdown", "obsidian", "notion", "html", "csv", "json"}}}, "workspace_id", "source_key", "label", "format"))
	case strings.HasSuffix(path, "/items") && method == "post":
		item := object(map[string]any{"source_id": text, "path": text, "kind": map[string]any{"enum": []string{"document", "attachment", "csv"}}, "bytes": map[string]any{"type": "integer", "minimum": 0, "maximum": 52428800}, "sha256": map[string]any{"type": "string", "pattern": "^[0-9a-f]{64}$"}, "parent_source_id": text, "metadata": object(map[string]any{"title": text, "tags": map[string]any{"type": "array", "items": text}, "aliases": map[string]any{"type": "array", "items": text}, "icon": text})}, "source_id", "path", "kind", "bytes", "sha256")
		body(object(map[string]any{"items": map[string]any{"type": "array", "minItems": 1, "maxItems": 100, "items": item}}, "items"))
	case strings.HasSuffix(path, "/chunks/{chunk}") && method == "put":
		op["requestBody"] = map[string]any{"required": true, "content": map[string]any{"application/octet-stream": map[string]any{"schema": map[string]any{"type": "string", "format": "binary", "maxLength": 1048576}}}}
		op["description"] = "정확한 1MiB 청크(마지막은 manifest 나머지 크기), 0부터 시작하는 ordinal. 동일 ordinal/SHA-256 재전송은 성공하며 바이트가 다른 재전송은409. 문서 원본은 파일당4MiB, CSV/첨부50MiB."
	case strings.HasSuffix(path, "/prepare") && method == "post":
		body(object(map[string]any{"revision": integer}, "revision"))
	case strings.HasSuffix(path, "/review") && method == "put":
		column := object(map[string]any{"name": text, "type": map[string]any{"enum": []string{"text", "number", "checkbox", "date", "select", "url", "email", "phone"}}, "options": map[string]any{"type": "array", "maxItems": 100, "items": text}}, "name", "type")
		body(object(map[string]any{"revision": integer, "plan_hash": text, "decision": map[string]any{"enum": []string{"apply", "skip"}}, "confirm_types": map[string]any{"type": "boolean"}, "types": map[string]any{"type": "array", "maxItems": 100, "items": column}}, "revision", "plan_hash", "decision"))
	case strings.HasSuffix(path, "/commit") && method == "post":
		body(object(map[string]any{"revision": integer, "plan_hash": text, "confirmation": map[string]any{"const": "IMPORT"}}, "revision", "plan_hash", "confirmation"))
		op["description"] = "검토 해시/CAS+CSV 자료형 동의 후 비동기 전체 원자 확정. 신규 비공개 초안, 기존 원문/버전/ACL/원래 세션·키/PII/저장소를 다시 검사합니다. PG 리스/항목 체크포인트/객체 키 journal/트랜잭션 receipt로 중복 게시를 방지합니다. 외부 첨부 준비 후 권한 잠금5초·개별SQL10분·전체job1시간 제한. 최종TX 동안 공간/공유 권한 관리 쓰기가 기다릴 수 있습니다."
	case strings.HasSuffix(path, "/download") && method == "get":
		parameters, _ := op["parameters"].([]any)
		op["parameters"] = append(parameters, map[string]any{"name": "view", "in": "query", "schema": map[string]any{"enum": []string{"original", "canonical"}}})
		responses, _ := op["responses"].(map[string]any)
		if responses == nil {
			responses = map[string]any{}
			op["responses"] = responses
		}
		responses["200"] = map[string]any{"description": "현재 소유자 ACL 검사 후 원본 bytes 또는 변환 Markdown, no-store", "content": map[string]any{"application/octet-stream": map[string]any{"schema": map[string]any{"type": "string", "format": "binary"}}}}
	case path == "/admin/migration/settings" && method == "put":
		body(object(map[string]any{"revision": integer, "max_session_bytes": map[string]any{"type": "integer", "minimum": 1048576, "maximum": 8589934592}, "max_user_bytes": map[string]any{"type": "integer", "minimum": 1048576, "maximum": 17179869184}, "max_items": map[string]any{"type": "integer", "minimum": 1, "maximum": 100000}, "retention_days": map[string]any{"type": "integer", "minimum": 1, "maximum": 30}}, "revision", "max_session_bytes", "max_user_bytes", "max_items", "retention_days"))
	}
}
