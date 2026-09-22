package server

import "strings"

// These endpoints deliberately do not inherit cookie or Bearer authorization.
// A share secret conveys only this particular owner-authorized document grant.
func protectionOpenAPI(path, method string, op map[string]any) {
	text := map[string]any{"type": "string"}
	boolean := map[string]any{"type": "boolean"}
	object := func(properties map[string]any, required ...string) map[string]any {
		return map[string]any{"type": "object", "properties": properties, "required": required}
	}
	body := func(schema map[string]any) {
		op["requestBody"] = map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{"schema": schema}}}
	}
	responses := op["responses"].(map[string]any)
	if strings.HasPrefix(path, "/public-shares/") {
		op["security"] = []any{}
		op["description"] = "로그인 세션·Bearer 키가 아닌 X-Madi-Share-Token 비밀(43자)이 필수입니다. /share/{UUID}#비밀에서 fragment는 서버 URL로 보내지 않으며 이 헤더로만 전달합니다. 암호가 설정된 공유는 unlock 응답의 X-Madi-Share-Access 토큰도 필요합니다. 토큰은 공유 revision·직접 연결 IP·최대 1시간에 고정되며 회전·설정 변경·폐기·권한 회수로 무효화됩니다. 모든 응답은 no-store/noindex/no-referrer. 직접 연결 IP당 분당 120회(내부 재검증도 산입) 제한. X-Forwarded-For는 신뢰하지 않습니다."
		parameters, _ := op["parameters"].([]any)
		parameters = append(parameters, map[string]any{"name": "X-Madi-Share-Token", "in": "header", "required": true, "schema": map[string]any{"type": "string", "minLength": 43, "maxLength": 43}, "description": "일회 표시된 링크의 # 뒤 비밀. URL query에 넣지 마세요."})
		if strings.HasSuffix(path, "/unlock") {
			body(object(map[string]any{"password": map[string]any{"type": "string", "format": "password", "writeOnly": true, "description": "공유 암호. UTF-8 72바이트 이하."}}, "password"))
			responses["200"] = map[string]any{"description": "암호 확인 성공", "content": map[string]any{"application/json": map[string]any{"schema": object(map[string]any{"access_token": text, "expires_at": map[string]any{"type": "string", "format": "date-time"}}, "access_token", "expires_at")}}}
			responses["429"] = map[string]any{"description": "링크·직접 연결 IP당 5분간 암호 시도 5회 또는 공개 요청 제한"}
		} else {
			parameters = append(parameters, map[string]any{"name": "X-Madi-Share-Access", "in": "header", "required": false, "schema": text, "description": "암호가 설정된 공유에서 필수. unlock의 access_token, 메모리에만 보관."})
			responses["401"] = map[string]any{"description": "공유 암호 확인 필요; code=share_password_required, 문서 제목·본문은 제공하지 않음"}
		}
		if strings.Contains(path, "/attachments/") {
			responses["200"] = map[string]any{"description": "강제 attachment 다운로드. 기존 첨부도 현재 정보보호 정책을 재검사함", "content": map[string]any{"application/octet-stream": map[string]any{"schema": map[string]any{"type": "string", "format": "binary"}}}}
		}
		op["parameters"] = parameters
		responses["404"] = map[string]any{"description": "없는 링크, 틀린 비밀, 만료, IP 제한, 정책·현재 원문 소유권/ACL 위반을 동일하게 처리"}
		responses["409"] = map[string]any{"description": "읽는 동안 문서 또는 공유·보호 정책이 바뀜. 새로 확인 필요"}
		return
	}
	if strings.Contains(path, "/public-shares") {
		op["security"] = []any{map[string]any{"cookieAuth": []string{}}}
		op["description"] = "현재 문서 소유자의 활성 사용자 세션만 허용합니다. 관리자라도 다른 소유자의 문서를 대신 공개하지 못하며 API 키·서비스 계정·플러그인은 제외됩니다. 새 비밀은 생성/회전 응답 url에서 한 번만 표시하고 DB에는 해시만 저장합니다. 설정 변경과 회전은 이전 암호 확인 토큰을 무효화합니다."
		if strings.HasSuffix(path, "/rotate") {
			body(object(map[string]any{"revision": map[string]any{"type": "integer", "minimum": 1}}, "revision"))
		} else if method == "post" || method == "put" {
			required := []string{"expires_at", "confirm_public"}
			if method == "put" {
				required = append(required, "revision")
			}
			body(object(map[string]any{"expires_at": map[string]any{"type": "string", "format": "date-time", "description": "미래 시각이며 생성 시점부터 관리자 max_days 이내"}, "password": map[string]any{"type": "string", "format": "password", "writeOnly": true, "description": "새 값은 8~72 UTF-8 바이트. 수정 시 빈 값은 유지."}, "clear_password": boolean, "ip_allowlist": map[string]any{"type": "array", "maxItems": 50, "items": text}, "allow_download": boolean, "allow_copy": boolean, "confirm_public": map[string]any{"type": "boolean", "const": true}, "revision": map[string]any{"type": "integer", "minimum": 1}}, required...))
		}
	}
	if path == "/admin/information-protection" && method == "put" {
		body(object(map[string]any{"revision": map[string]any{"type": "integer", "minimum": 1}, "settings": object(map[string]any{"enabled": boolean, "mode": map[string]any{"type": "string", "enum": []string{"warn", "block", "mask", "audit"}}, "detectors": map[string]any{"type": "array", "items": map[string]any{"type": "string", "enum": []string{"rrn", "email", "phone", "payment", "account"}}}, "custom_terms": map[string]any{"type": "array", "maxItems": 100, "items": text}, "unscannable": map[string]any{"type": "string", "enum": []string{"warn", "block"}}, "watermark_enabled": boolean, "watermark_min_classification": text, "public_shares_enabled": boolean, "public_share_max_classification": text, "public_share_max_days": map[string]any{"type": "integer", "minimum": 1, "maximum": 365}, "public_share_require_password": boolean, "public_share_allow_download": boolean})}, "revision", "settings"))
		op["description"] = "설정 전체를 revision과 함께 저장합니다. 기본 비활성. 차단·마스킹 정책은 숨은 Yjs 이력의 안전한 검사 한계로 실시간 협업을 중지하고 원문 정본 저장을 요구합니다. 이전 버전·백업을 소급 삭제하지 않습니다."
	}
}
