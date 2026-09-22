package server

func systemStatusMCPTools() []mcpTool {
	uuid := map[string]any{"type": "string", "format": "uuid"}
	number := map[string]any{"type": "integer", "minimum": 1, "maximum": 9007199254740990}
	stringField := map[string]any{"type": "string", "maxLength": 500}
	return []mcpTool{
		{Name: "get_system_status", Description: "현재 문서 ACL에 따른 운영 기대값과 보고 주체의 관측값·TTL·drift를 읽습니다. 서버 직접 배포 검증이 아닙니다.", Scope: "document:read", InputSchema: map[string]any{"type": "object", "additionalProperties": false, "properties": map[string]any{"document_id": uuid}, "required": []string{"document_id"}}, Annotations: map[string]any{"readOnlyHint": true, "destructiveHint": false, "openWorldHint": false}},
		{Name: "report_system_status", Description: "명시 허용된 보고 주체가 자신의 실제 관측을 저장합니다. document:read+write, 카드의 reporter allowlist와 모든 현재 revision/epoch, confirm:true가 필수입니다. 관측을 서버 직접 검증으로 인증하지 않으며 문서·배포를 변경하지 않습니다.", Scope: "document:write", InputSchema: map[string]any{"type": "object", "additionalProperties": false, "properties": map[string]any{"document_id": uuid, "request_id": uuid, "card_revision": number, "report_revision": map[string]any{"type": "integer", "minimum": 0, "maximum": 9007199254740990}, "verification_epoch": number, "document_version": map[string]any{"type": "integer", "minimum": 1, "maximum": 2147483647}, "observed_at": map[string]any{"type": "string", "format": "date-time"}, "confirm": map[string]any{"type": "boolean", "const": true}, "observation": map[string]any{"type": "object", "additionalProperties": false, "properties": map[string]any{"version": stringField, "deployment": stringField, "health": map[string]any{"type": "string", "enum": []string{"healthy", "degraded", "down", "unknown"}}, "note": map[string]any{"type": "string", "maxLength": 4000}}, "required": []string{"health"}}}, "required": []string{"document_id", "request_id", "card_revision", "report_revision", "verification_epoch", "document_version", "observed_at", "observation", "confirm"}}, Annotations: map[string]any{"readOnlyHint": false, "destructiveHint": false, "openWorldHint": false}},
	}
}
