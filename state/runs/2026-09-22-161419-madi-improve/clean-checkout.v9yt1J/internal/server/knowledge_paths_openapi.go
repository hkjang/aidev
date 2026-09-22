package server

import "strings"

func knowledgePathsOpenAPI(path, method string, op map[string]any) {
	if !strings.HasPrefix(path, "/knowledge-paths") && !strings.HasPrefix(path, "/learning-progress/") {
		return
	}
	uuid := map[string]any{"type": "string", "format": "uuid"}
	revision := map[string]any{"type": "integer", "minimum": 1}
	boolean := map[string]any{"type": "boolean"}
	object := func(p map[string]any, required ...string) map[string]any {
		return map[string]any{"type": "object", "properties": p, "required": required}
	}
	body := func(v any) {
		op["requestBody"] = map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{"schema": v}}}
	}
	op["description"] = "역할 이름은 안내용입니다. 경로 ACL과 원문 ACL을 각각 확인하며 역할·실행·문서 게시 권한을 부여하지 않습니다. 목록/단건 GET만 document:read 키/MCP 지원, 실습 원문은 키에 반환하지 않습니다. 개인 진행은 현재 경로 revision·원문 version·현재 승인 정책에 결합됩니다."
	if method != "get" || strings.HasSuffix(path, "/history") || strings.HasSuffix(path, "/review") {
		op["security"] = []any{map[string]any{"sessionCookie": []string{}}}
	}
	switch {
	case path == "/knowledge-paths" && method == "get":
		args, _ := op["parameters"].([]any)
		op["parameters"] = append(args, map[string]any{"name": "workspace_id", "in": "query", "required": true, "schema": uuid}, map[string]any{"name": "archived", "in": "query", "schema": map[string]any{"type": "string", "enum": []string{"0", "1"}}})
		op["description"] = "현재 접근 가능한 최근 경로 최대200개와 has_more입니다. 제목/역할 필터는 현재 목록의 클라이언트 필터입니다. 원문·개인 실습 원문은 포함하지 않습니다."
	case (path == "/knowledge-paths" || path == "/knowledge-paths/{id}") && (method == "post" || method == "put"):
		step := object(map[string]any{"id": uuid, "document_id": uuid, "kind": map[string]any{"enum": []string{"read", "practice", "review"}}, "title": map[string]any{"type": "string", "description": "UTF-8 최대250바이트"}, "instruction": map[string]any{"type": "string", "description": "UTF-8 최대4000바이트"}}, "document_id", "kind", "title")
		body(object(map[string]any{"workspace_id": uuid, "space_id": map[string]any{"type": "string", "description": "공간 UUID 또는 빈 문자열"}, "revision": revision, "title": map[string]any{"type": "string", "description": "UTF-8 1~250바이트"}, "description": map[string]any{"type": "string", "description": "UTF-8 최대4000바이트"}, "role_labels": map[string]any{"type": "array", "maxItems": 20, "items": map[string]any{"type": "string", "description": "UTF-8 1~100바이트"}}, "visibility": map[string]any{"enum": []string{"private", "workspace"}}, "archived": boolean, "confirm_shared": boolean, "steps": map[string]any{"type": "array", "minItems": 1, "maxItems": 100, "items": step}}, "workspace_id", "title", "visibility", "steps"))
		op["description"] = "워크스페이스 소유자/관리자 쿠키 전용, 현재 원문 read+대상 공간 write 필요. 변경 PUT은 revision CAS. 새 공유/대상 공간 변경은 confirm_shared:true. 기존 단계 ID는 같은 경로·동일 문서/종류만 유지하며 제외된 단계의 개인 이력은 보존됩니다. 모든 경로 변경은 새 revision으로 현재 진행 재확인. 3초 lock/12초 statement timeout의 짧은 ACL 직렬화 TX입니다."
	case strings.HasSuffix(path, "/confirm"):
		body(object(map[string]any{"path_revision": revision, "document_version": revision, "revision": map[string]any{"type": "integer", "minimum": 0}, "proof": map[string]any{"type": "string", "description": "실습/검토 사유는 비어 있지 않은 UTF-8 최대16KiB. 읽기 확인에서는 저장하지 않음."}, "confirmation": map[string]any{"const": "CONFIRM"}}, "path_revision", "document_version", "revision", "confirmation"))
		op["description"] = "사람의 본인 쿠키 전용. 이전 비제외 단계를 현재 버전으로 먼저 완료해야 합니다. 기존 개인 revision CAS, 원문/경로 CAS, PII 적용 후 proof 암호화. review는 관리자 명시 learning_step 정책이 있는 경우에만 기존 승인 엔진으로 제출하며 접수는 완료가 아닙니다. 응답 protection.changed/mode/findings 및 정제된 proof. 원문·역할·실행 상태를 변경하지 않습니다."
	case strings.HasSuffix(path, "/history"):
		op["description"] = "본인 쿠키 전용 최근200개 개인 이력과 has_more. 이전 proof는 암호화 보관하며 현재 경로+참조 문서 ACL/PII를 통과한 원문만 표시합니다. 접근 상실 후에는 상세 메타데이터/원문 숨김. 이력은 현재 완료와 별개입니다."
	case strings.HasSuffix(path, "/review"):
		op["description"] = "제출자 또는 해당 revision에 실제 지정된 검토자 쿠키 전용. 현재 경로/모든 선행 문서 ACL과 고정된 실습 기록·승인 snapshot/version/hash를 확인해 원문을 반환합니다. 선행 기록/정책/원문 변경409로 새 미제출 기록을 노출하지 않습니다. 승인 snapshot 자체는 내용이 아닌 hash/version 참조만 보관. 이 승인으로 실행 권한이 생기지 않습니다."
	}
}
