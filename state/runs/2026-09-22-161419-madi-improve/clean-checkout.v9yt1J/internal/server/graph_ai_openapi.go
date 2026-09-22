package server

import "strings"

func graphAIOpenAPI(paths, schemas map[string]any) {
	schemas["GraphAISnapshot"] = map[string]any{"type": "object", "additionalProperties": false, "required": []string{"document_id", "version", "document_hash", "start_byte", "end_byte", "content_hash"}, "properties": map[string]any{
		"document_id": map[string]any{"type": "string", "format": "uuid"}, "version": map[string]any{"type": "integer", "minimum": 1}, "document_hash": map[string]any{"type": "string"}, "start_byte": map[string]any{"type": "integer", "minimum": 0}, "end_byte": map[string]any{"type": "integer", "minimum": 1}, "content_hash": map[string]any{"type": "string"},
	}}
	schemas["GraphAIAnalyze"] = map[string]any{"type": "object", "required": []string{"request_id", "snapshots", "provider_fingerprint", "kinds", "consent"}, "properties": map[string]any{
		"request_id":           map[string]any{"type": "string", "format": "uuid", "description": "새 요청 UUID. 같은 ID 재전송은 409; 전경 요청 종료 후 자동 재시도하지 않음"},
		"snapshots":            map[string]any{"type": "array", "minItems": 1, "maxItems": 12, "items": map[string]any{"$ref": "#/components/schemas/GraphAISnapshot"}, "description": "context 미리보기의 정확한 원문 버전·해시·UTF-8 바이트 구간"},
		"provider_fingerprint": map[string]any{"type": "string", "description": "현재 미리보기 공급자 지문. 공급자 설정·모델·비밀값 변경 시 재동의 필수"},
		"kinds":                map[string]any{"type": "array", "uniqueItems": true, "minItems": 1, "maxItems": 5, "items": map[string]any{"type": "string", "enum": graphAIKinds}, "description": "topic은 단일 입력 문서만 허용. duplicate는 연결 제안이며 병합·삭제하지 않음"},
		"consent":              map[string]any{"type": "boolean", "const": true, "description": "표시된 공급자에게 선택 구간을 보내는 명시적 동의"},
	}}
	schemas["GraphAIConfirm"] = map[string]any{"type": "object", "required": []string{"action_hash"}, "properties": map[string]any{"action_hash": map[string]any{"type": "string", "description": "수정하지 않은 후보 지문"}, "confirm": map[string]any{"type": "boolean"}, "reject": map[string]any{"type": "boolean"}}, "description": "confirm=true 또는 reject=true 중 하나만. 적용은 실제 일반 사용자의 원래 쿠키 세션에서 한 건씩 확인하며 키·서비스 계정·플러그인 확인은 금지"}
	for path, value := range paths {
		if !strings.Contains(path, "/graph-ai/") {
			continue
		}
		entry, ok := value.(map[string]any)
		if !ok {
			continue
		}
		for method, raw := range entry {
			op, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			op["tags"] = []string{"AI 지식 제안"}
			op["description"] = "현재 사용자 ACL 및 document:read·ai:execute 교집합. 원본 1~12개, 실제 전송 구간 합계 48KiB, 인코딩 입력/모델 출력 각각 64KiB, 후보 최대 30개. 모든 출처의 현재 버전·해시·권한, 공급자 동의, 원래 세션/키/플러그인과 ai-graph 기능 정책을 다시 검사합니다. 출처 원문을 기록에 복사하지 않으며 raw 검색 이력을 사용하지 않습니다. 후보 적용 전 자동 변경은 없습니다."
			if strings.HasSuffix(path, "/context") {
				parameters, _ := op["parameters"].([]any)
				op["parameters"] = append(parameters, map[string]any{"name": "document_ids", "in": "query", "required": true, "schema": map[string]any{"type": "string"}, "description": "현재 접근 가능한 같은 워크스페이스 문서 UUID 1~12개, 쉼표 구분"})
			}
			if method == "post" && strings.HasSuffix(path, "/analyze") {
				op["requestBody"] = map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": "#/components/schemas/GraphAIAnalyze"}}}}
				op["responses"] = map[string]any{"200": map[string]any{"description": "SSE: phase, text, 검증된 proposal 또는 error+retract, [DONE]. X-Madi-Graph-Run-ID 헤더. JSON 문자열은 후보 스키마·정확 인용·PII 검증 뒤에만 영속화", "content": map[string]any{"text/event-stream": map[string]any{"schema": map[string]any{"type": "string"}}}}, "400": map[string]any{"description": "입력/동의/종류 오류"}, "403": map[string]any{"description": "현재 권한 또는 기능 제한"}, "409": map[string]any{"description": "원문·공급자 변경 또는 중복 요청 ID"}, "413": map[string]any{"description": "인코딩 입력 한도 초과"}, "429": map[string]any{"description": "사용자당 동시 전경 분석 2개 한도"}}
			}
			if strings.HasSuffix(path, "/confirm") {
				op["security"] = []any{map[string]any{"cookieAuth": []string{}}}
				op["requestBody"] = map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": "#/components/schemas/GraphAIConfirm"}}}}
				op["description"] = "실제 요청 소유자 쿠키 세션의 명시적 단건 확인 전용. document:write 및 원래 키/플러그인 제약도 상한으로 적용. 정확 해시·전체 출처 현재 ACL/버전·정보보호 정책을 같은 TX에서 검사하고 변경과 결과 영수증을 함께 커밋합니다. 동시 재확인은 같은 영수증 반환. 엔터티/공백은 가장 높은 입력 보안 등급의 개인 초안. topic은 단일 입력 원문의 시스템 메타데이터만 변경. API 키/서비스 계정/플러그인은 적용 불가."
			}
			if method == "delete" {
				op["description"] = "본인 분석 기록과 미적용 후보 삭제. 이미 적용한 정식 문서·관계·주제는 보존. 다른 사용자의 기록은 관리자도 조회·삭제하지 않음."
			}
		}
	}
}
