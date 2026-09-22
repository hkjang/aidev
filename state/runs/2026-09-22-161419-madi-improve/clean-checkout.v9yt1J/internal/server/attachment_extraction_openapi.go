package server

import "strings"

func attachmentExtractionOpenAPI(path, method string, op map[string]any) {
	if !strings.Contains(path, "/attachment-extractions") && !strings.HasSuffix(path, "/extraction-context") && !(strings.HasPrefix(path, "/attachments/") && strings.HasSuffix(path, "/extractions")) && !(strings.HasPrefix(path, "/documents/") && strings.HasSuffix(path, "/extractions")) && path != "/admin/attachment-extraction/settings" {
		return
	}
	integer := func(min, max int) map[string]any {
		return map[string]any{"type": "integer", "minimum": min, "maximum": max}
	}
	uuid := map[string]any{"type": "string", "format": "uuid"}
	hash := map[string]any{"type": "string", "pattern": "^[0-9a-f]{64}$"}
	object := func(props map[string]any, required ...string) map[string]any {
		return map[string]any{"type": "object", "properties": props, "required": required}
	}
	body := func(schema map[string]any) {
		op["requestBody"] = map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{"schema": schema}}}
	}
	op["description"] = "관리자 기본 끔. 첨부의 현재 부모 문서 ACL·워크스페이스·키 scope를 확인합니다. 추출 결과는 원본 SHA-256·정책/추출 revision·현재 활성 head에 결합되며 복원 후 재생성해야 합니다. 원본 첨부 바이트를 수정하지 않습니다."
	switch {
	case path == "/admin/attachment-extraction/settings":
		op["description"] = "서비스 관리자 쿠키 로그인 전용. revision CAS 정책 변경/이력 기록. 정책 변경은 이전 결과의 현재 검색·인용을 무효화하며 재추출이 필요합니다."
		if method == "put" {
			boolean := map[string]any{"type": "boolean"}
			body(object(map[string]any{"revision": integer(1, 2147483647), "data": object(map[string]any{"enabled": boolean, "ocr_enabled": boolean, "max_file_bytes": integer(1048576, 52428800), "max_pages": integer(1, 500), "max_text_bytes": integer(65536, 8388608), "max_fragments": integer(100, 20000), "timeout_seconds": integer(30, 600)}, "enabled", "ocr_enabled", "max_file_bytes", "max_pages", "max_text_bytes", "max_fragments", "timeout_seconds")}, "revision", "data"))
		}
	case strings.HasPrefix(path, "/attachments/") && strings.HasSuffix(path, "/extractions") && method == "post":
		body(object(map[string]any{"document_version": integer(1, 2147483647), "checksum": map[string]any{"type": "string", "description": "현재 첨부 SHA-256. 기존 자료의 미기록 값은 빈 문자열이며 서버가 원본을 읽어 CAS 보완합니다."}, "ocr_pages": map[string]any{"type": "array", "maxItems": 20, "uniqueItems": true, "items": integer(1, 500)}, "confirmation": map[string]any{"type": "string", "description": "OCR 페이지가 있으면 정확히 OCR"}}, "document_version", "checksum"))
		op["description"] = "document:read + document:write, 직접 사용자/서비스 키 요청만 허용(플러그인 제외). 최대 개인 대기 5개/첨부 동시 1개. 현재 원본+세션/키+정책을 작업 시작·진행·최종 원자 저장에서 재검사합니다. 202는 접수이며 완료가 아닙니다. OCR은 이미 텍스트가 없는 명시 선택 PDF 페이지에만 적용됩니다."
	case strings.HasSuffix(path, "/ai") && method == "post":
		body(object(map[string]any{"fragment_id": uuid, "document_version": integer(1, 2147483647), "revision": integer(1, 2147483647), "start_byte": integer(0, 16384), "end_byte": integer(1, 16384), "hash": hash, "provider_fingerprint": map[string]any{"type": "string"}, "consent": map[string]any{"const": true}, "prompt": map[string]any{"type": "string", "maxLength": 4096}}, "fragment_id", "document_version", "revision", "start_byte", "end_byte", "hash", "provider_fingerprint", "consent"))
		op["description"] = "document:read + ai:execute. UTF-8 fragment text의 정확한 바이트 범위(최대 8KiB)와 SHA-256, 공급자 지문에 단건 명시 동의를 결합합니다. 선택 원문+질문만 전송하며 첨부 바이너리/부모 본문/이웃 구간/대화/RAG를 자동 추가하지 않습니다. SSE 기본, 관리자 max token 최대262144; 철회·원본/정책 변경 시 retract 후 종료. 결과는 자동 적용하지 않습니다."
	case strings.HasSuffix(path, "/fragments") && method == "get":
		parameters, _ := op["parameters"].([]any)
		op["parameters"] = append(parameters, map[string]any{"name": "offset", "in": "query", "schema": integer(0, 20000)})
		op["description"] = "현재 활성 결과의 위치 포함 텍스트 구간 최대100개와 has_more. 각 구간 최대16KiB. PDF page/bounds/page_size/OCR, Office slide/sheet/cell/paragraph/table/row/column을 지원하며 형식에 해당하는 위치만 제공합니다."
	case strings.HasSuffix(path, "/citation"):
		op["description"] = "현재 원본·활성 추출·정책·부모 ACL을 재확인한 위치 인용. StartByte/EndByte는 부모 Markdown이 아니라 attachment_fragment_id의 UTF-8 text에 대한 오프셋입니다. 첨부 SHA-256/추출 revision/구간 hash/position을 포함합니다. stale 결과410."
	case method == "delete":
		op["description"] = "현재 문서 읽기·쓰기 권한을 가진 원래 실행자의 진행 작업 취소. 생성된 원본 첨부나 문서는 삭제하지 않습니다. 관리자는 별도 공통 작업 관리 화면에서 실행을 중단할 수 있습니다."
	}
}
