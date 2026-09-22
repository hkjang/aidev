package server

// Review APIs supplement the registered route catalogue. They never invent
// server routes; existing operations retain their middleware-derived security.
func uxReviewOpenAPI(paths, schemas map[string]any) {
	worksetsOpenAPI(paths, schemas)
	text := map[string]any{"type": "string"}
	version := map[string]any{"type": "integer", "minimum": 1, "maximum": 2147483647}
	uuid := map[string]any{"type": "string", "format": "uuid"}
	schemas["AISelectionRequest"] = map[string]any{"type": "object", "required": []string{"expected_version", "start_byte", "end_byte", "selected_text", "provider_fingerprint", "consent", "action"}, "properties": map[string]any{
		"expected_version":     version,
		"start_byte":           map[string]any{"type": "integer", "minimum": 0, "description": "저장된 Markdown UTF-8 바이트의 시작 위치. UTF-16 문자 인덱스가 아님"},
		"end_byte":             map[string]any{"type": "integer", "minimum": 1, "description": "끝 위치(미포함). 시작보다 크고 범위 최대32768바이트, 양쪽 UTF-8 경계 필수"},
		"selected_text":        map[string]any{"type": "string", "description": "현재 원문의 선택 범위와 정확히 일치하는 비어 있지 않은 UTF-8 문자열. 최대32768바이트"},
		"provider_fingerprint": text,
		"consent":              map[string]any{"type": "boolean", "const": true},
		"action":               map[string]any{"type": "string", "enum": []string{"rewrite", "summarize", "translate", "write"}},
		"prompt":               map[string]any{"type": "string", "description": "사용자가 명시한 추가 지시, UTF-8 최대4096바이트"},
	}}
	schemas["AISelectionDraft"] = map[string]any{"type": "object", "required": []string{"ticket", "title", "markdown"}, "properties": map[string]any{
		"ticket":   map[string]any{"type": "string", "description": "완료 SSE proposal.draft_ticket. 사용자·원문버전·범위해시·답변해시·공급자 결합,30분 만료, 최대8192바이트"},
		"title":    map[string]any{"type": "string", "description": "기존 문서 생성의 제목 검증 적용"},
		"markdown": map[string]any{"type": "string", "description": "완료 제안과 바이트 단위로 동일한 Markdown. 최대65536바이트"},
	}}
	schemas["DocumentAccessRequest"] = map[string]any{"type": "object", "required": []string{"workspace_id", "permission"}, "properties": map[string]any{
		"workspace_id": uuid,
		"permission":   map[string]any{"type": "string", "enum": []string{"read", "write"}},
		"reason":       map[string]any{"type": "string", "description": "최대2000 UTF-8바이트. 현재 정보보호 정책 적용 후 암호화 보관, 실제 문서 소유자에게만 표시"},
	}}
	schemas["DocumentAccessDecision"] = map[string]any{"type": "object", "required": []string{"revision", "action"}, "properties": map[string]any{
		"revision":                  version,
		"action":                    map[string]any{"type": "string", "enum": []string{"grant", "reject", "cancel"}},
		"expected_document_version": map[string]any{"type": "integer", "minimum": 1, "description": "grant/reject 필수. cancel은 요청 revision만 사용"},
	}}
	schemas["DocumentTrashCAS"] = map[string]any{"type": "object", "properties": map[string]any{"expected_version": version}, "description": "UI는 항상 현재 버전을 전송합니다. 생략한 기존 API 호출은 처리 시 최신 상태에 대해 동작합니다. 명시한 null/문자열/소수/0은 거절됩니다. 성공한 실제 삭제·복구는 버전과 불변 스냅샷을1개씩 증가시키며, 이미 같은 상태인 동일 버전 호출은 변경하지 않습니다."}
	schemas["DocumentAccessPreview"] = map[string]any{"type": "object", "required": []string{"expected_version"}, "properties": map[string]any{"expected_version": version, "parent_id": map[string]any{"type": []string{"string", "null"}, "format": "uuid"}, "space_id": map[string]any{"type": []string{"string", "null"}, "format": "uuid"}, "visibility": map[string]any{"type": "string", "enum": []string{"private", "selected", "workspace"}}}}
	schemas["DocumentSplitPreview"] = map[string]any{"type": "object", "required": []string{"expected_version", "start_byte", "end_byte", "selected_text", "title"}, "properties": map[string]any{"expected_version": version, "start_byte": map[string]any{"type": "integer", "minimum": 0}, "end_byte": map[string]any{"type": "integer", "minimum": 1}, "selected_text": map[string]any{"type": "string", "description": "저장 원문과 정확히 일치하는 완전한 Markdown 블록. UTF-8 최대256KiB, 원문 최대1MiB. Front Matter/HTML 확장/부분 구조 거절"}, "title": map[string]any{"type": "string", "description": "공백이 아닌 제목, 최대500 UTF-8바이트"}}}
	schemas["DocumentSplitCommit"] = map[string]any{"type": "object", "required": []string{"ticket", "client_request_id", "consent"}, "properties": map[string]any{"ticket": map[string]any{"type": "string", "description": "미리보기의 actor·source/version·sourcehash·범위·제목·childID 결합 암호화 티켓,10분 만료"}, "client_request_id": uuid, "consent": map[string]any{"type": "boolean", "const": true}}}
	schemas["DatabaseRowMutation"] = map[string]any{"type": "object", "properties": map[string]any{"expected_version": version, "values": map[string]any{"type": "object"}}, "description": "PUT values 필수, expected_version 제공 시 엄격한 CAS(브라우저는항상전송). DELETE는 values 없이 선택적 expected_version. 생략한 기존API는최신값에작동. 실제값변경은행version자동증가,새행항상새UUID,삭제ID로PUT은복구하지않음."}
	schemas["DatabaseCellReview"] = map[string]any{"type": "object", "required": []string{"schema_fingerprint", "cells"}, "properties": map[string]any{"schema_fingerprint": text, "commit": map[string]any{"type": "boolean", "default": false}, "consent": map[string]any{"type": "boolean"}, "cells": map[string]any{"type": "array", "minItems": 1, "maxItems": 500, "items": map[string]any{"type": "object", "required": []string{"row_id", "property_id", "expected_version", "value"}, "properties": map[string]any{"row_id": uuid, "property_id": text, "expected_version": version, "value": map[string]any{}}}}}}
	schemas["DatabaseSavedView"] = map[string]any{"type": "object", "required": []string{"name", "visibility", "data"}, "properties": map[string]any{"name": text, "visibility": map[string]any{"type": "string", "enum": []string{"private", "workspace"}}, "expected_version": version, "share_consent": map[string]any{"type": "boolean"}, "data": map[string]any{"type": "object", "description": "최대32KiB. view(table/board/calendar/list/gallery/timeline), filters최대30, sorts최대10, columns현재속성ID최대100, board_property_id(select/status), date_property_id(date). 임의CSS/JS/알수없는필드거절"}}}
	schemas["DatabaseViewPreference"] = map[string]any{"type": "object", "required": []string{"view_id"}, "properties": map[string]any{"view_id": map[string]any{"type": "string", "description": "현재읽기권한보기UUID 또는빈문자열(개인기본해제)"}}}
	for _, entry := range []struct {
		path, method, summary, description, schema string
		personal                                   bool
	}{
		{"/databases/{id}/editing", "get", "현재 속성 기준과 원자 셀 편집 한도", "개인브라우저와현재DB읽기권한필수. schema_fingerprint와속성/100행500셀한도를반환합니다.", "", true},
		{"/databases/{id}/edit-cells", "post", "타입·현재행CAS 검사 후 범위 원자 반영", "개인브라우저현재DB작성권한. 요청최대4MiB/100행500셀,비교최대8MiB. commit:false는변경없이valid/errors/changes반환. commit:true는consent:true필수이며원문/관계/속성/각행version을같은TX에서다시검사. 한셀이라도오류면committed:false로전체미변경. 속성/행기준변경409,SQL실패전체rollback.", "DatabaseCellReview", true},
		{"/databases/{id}/views", "get", "내 개인 보기와 접근 가능한 팀 보기", "현재DB ACL적용,개인보기소유자외서비스관리자도조회불가. views+본인default_view_id 반환,삭제속성참조보기는compatible:false.", "", true},
		{"/databases/{id}/views", "post", "현재 구성을 개인 또는 명시적 팀 보기로 저장", "개인보기는현재DB읽기권한,팀보기는작성권한과share_consent:true필수. 팀보기는행/원문ACL을확대하지않습니다. 이름정보보호적용,최대100개접근가능보기.", "DatabaseSavedView", true},
		{"/databases/{id}/views/{viewID}", "put", "소유한 보기의 확인한 버전만 변경", "expected_version필수,다른소유자개인보기우회불가. 팀보기저장과공유철회는작성권한,팀저장매번명시동의. 동시변경409.", "DatabaseSavedView", true},
		{"/databases/{id}/views/{viewID}", "delete", "확인한 버전의 내 저장 보기 삭제", "expected_version필수. 행/문서는삭제하지않고해당보기를쓰던개인기본참조만해제. 현재소유권/DB권한/CAS적용.", "DatabaseRowMutation", true},
		{"/databases/{id}/view-preference", "put", "본인의 데이터베이스 기본 보기 설정", "현재읽을수있는개인/팀보기만선택. 다른사용자기본은변경하지않으며보기공유철회/삭제후접근이사라지면기본보기로조회하지않습니다.", "DatabaseViewPreference", true},
		{"/databases/{id}/rows/{rowId}", "put", "행 값 일부를 현재 버전과 비교하여 저장", "expected_version은선택이며제공시엄격한양의정수CAS,브라우저는항상사용. 기존값타입/관계/감사/outbox/automation경로유지. SQL직접·AI·버튼의값변경도version을증가시킵니다.", "DatabaseRowMutation", false},
		{"/databases/{id}/rows/{rowId}", "delete", "확인한 현재 행 삭제", "expected_version은선택이며제공시다른편집자의후속변경409. 문서휴지통과달리행물리삭제이며새행은새UUID입니다.", "DatabaseRowMutation", false},
		{"/documents/{id}/split-preview", "post", "선택한 완전한 Markdown 블록의 분리 영향 확인", "본인 일반 브라우저와 현재 문서 작성 권한이 필요합니다. UTF-8 범위를 원문과 비교하고 부분 문단·목록·표·코드/Front Matter/HTML 확장을 거절합니다. 변경 없이 선택 원문·참조 대체문·권한 상속 설명·10분 티켓만 반환합니다.", "DocumentSplitPreview", true},
		{"/documents/{id}/split", "post", "원문 참조와 새 하위 초안을 원자적으로 저장", "현재 쿠키·멤버십·문서 CAS·권한·트리 깊이·정보보호·게시 정책을 같은 TX에서 검사합니다. 원본의 선택 부분에 위키 참조를 남기고 원본/공간 ACL을 상속하는 초안을 생성합니다. 원문버전·새문서·task ID·검색·outbox·receipt 모두 성공하거나 모두 rollback합니다. 같은 사용자/client_request_id/티켓 재시도는 현재 ACL에서 기존 결과를 반환합니다. 다른 payload409, 삭제된 결과410. 이전 CRDT epoch는 reset하며 자동 병합하지 않습니다.", "DocumentSplitCommit", true},
		{"/documents/{id}/ai-selection", "get", "선택 영역 AI 전송 대상과 현재 버전 확인", "document:read와 ai:execute 및 현재 문서 ACL이 필요합니다. provider.base_url은 인증 쿼리를 생략한 표시용 주소이며 API키를 반환하지 않습니다. automatic_apply=false.", "", false},
		{"/documents/{id}/ai-selection", "post", "선택한 원문만 AI로 스트리밍 비교", "선택 문자열과 명시한 추가 지시만 관리자 설정 공급자로 전송합니다. 전체 문서·제목·태그·RAG를 자동 첨부하지 않습니다. stream=true, 관리자 max_tokens 한도(최대262144), 출력 최대65536바이트. SSE는 text 조각, 최종 proposal, [DONE] 순서이며 현재 원문·세션·키·ACL·공급자 변경 시 retract하고 종료합니다. 완성 제안은 문서를 자동 수정하지 않으며 적용은 기존 PUT /documents/{id} CAS 경로입니다.", "AISelectionRequest", false},
		{"/ai/selection-drafts", "post", "완료된 선택 AI 제안을 새 개인 문서로 저장", "본인 일반 사용자 브라우저 쿠키만 허용합니다. 원문·계정·멤버십·세션·공급자를 같은 생성 TX에서 다시 검사하고 기존 문서 생성의 정보보호·게시승인·스냅샷·감사 경로로 나만 보기 문서를 생성합니다. 티켓은 답변 변경/다른 사용자/만료/참조 변경에 재사용할 수 없습니다.", "AISelectionDraft", true},
		{"/documents/{id}/access-requests", "post", "문서 존재를 공개하지 않고 접근 권한 요청 접수", "일반 사용자 브라우저와 현재 워크스페이스 멤버십이 필요합니다. 유효/비존재/다른 워크스페이스/이미 접근 가능한 대상에 동일202 및 제목 없는 동일 형식 접수 기록을 반환합니다. 실제 전달은 현재 요청 가능한 문서 소유자에게만 합니다. 시간당20개·열린20개, 같은 대상 열린 요청 중복 방지 및 처리 뒤10분 cooldown. 보안 접근 요청은 문서 게시 검토·승인과 별개입니다.", "DocumentAccessRequest", true},
		{"/access-requests", "get", "내 접근 요청과 현재 소유 문서의 수신 요청 조회", "workspace_id 필수. 내 요청에는 내가 입력한 문서 ID와 요청 상태만 포함하고 숨겨진 제목·원문·사유를 공개하지 않습니다. 수신 요청은 현재 소유권·문서/상위/공간 ACL을 다시 적용합니다. 각 최근100개, 서비스 관리자도 개인 문서 소유권을 우회하지 않습니다.", "", true},
		{"/access-requests/{id}", "put", "요청·문서 CAS 확인 후 접근 요청 처리", "cancel은 본인 요청만, grant/reject는 현재 문서 소유자만 가능합니다. 계정·세션·멤버십·역할 상한을 같은 TX에서 확인합니다. private→selected 시 비활성 공유가 남아 있으면409로 거절하며 다른 사용자 권한을 자동 활성화하지 않습니다. 상위/공간 ACL 때문에 실제 접근을 제공할 수 없으면 요청·문서·share·스냅샷 전체를 rollback합니다. grant는 게시 승인이 아니며 기존 게시 정책을 적용합니다.", "DocumentAccessDecision", true},
		{"/documents/{id}/access-preview", "post", "현재 공유·이동 영향과 10분 확인 티켓", "문서 소유자의 일반 브라우저만 사용합니다. 내부 범위·이동 위치·보수적인 확대 가능성을 설명하며 숨겨진 하위문서 수나 열람 인원을 추정하지 않습니다. 원문을 반환하거나 외부로 보내지 않습니다. ticket은 사용자/문서version/변경속성/기존·목적지 조상과 직접공유·공간권한·현재멤버 역할에 결합합니다. 기존 PUT /documents/{id}의 access_preview_ticket으로 전송하고, 확인 이후 정책이 달라졌거나 실제시각 10분이 지나면409입니다. ticket 없는 기존 호출도 최종목적지 ACL을 우회하지 않습니다.", "DocumentAccessPreview", true},
		{"/documents/{id}", "delete", "현재 버전을 확인하여 문서를 휴지통으로 이동", "expected_version 제공 시 CAS. 법적 보존/보존기한 정책은 그대로 적용됩니다. 성공 응답 {ok,id,version,changed}의 version으로 즉시 복구하세요. 다른 사용자의 후속 삭제/복구가 있으면 오래된 되돌리기는409입니다.", "DocumentTrashCAS", false},
		{"/documents/{id}/restore", "post", "휴지통 문서의 확인한 상태만 복구", "expected_version 제공 시 CAS. 같은 TX에서 현재 문서 권한·정보보호·게시 정책을 적용하고 canonical 응답과 새 버전을 반환합니다. 후속 변경을 오래된 undo가 덮지 않도록 UI는 삭제 응답 버전을 항상 전송합니다.", "DocumentTrashCAS", false},
	} {
		methods, _ := paths[entry.path].(map[string]any)
		op, _ := methods[entry.method].(map[string]any)
		if op == nil {
			continue
		}
		op["summary"], op["description"] = entry.summary, entry.description
		if entry.personal {
			op["security"] = []any{map[string]any{"cookieAuth": []string{}}}
		}
		if entry.schema != "" {
			required := entry.schema != "DocumentTrashCAS" && !(entry.path == "/databases/{id}/rows/{rowId}" && entry.method == "delete")
			op["requestBody"] = map[string]any{"required": required, "content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": "#/components/schemas/" + entry.schema}}}}
		}
		responses, _ := op["responses"].(map[string]any)
		if responses == nil {
			responses = map[string]any{}
			op["responses"] = responses
		}
		responses["409"] = map[string]any{"description": "기준 버전·요청 상태·공급자·현재 권한 변경. 현재 상태를 다시 확인한 뒤 새 명시 동의로 재시도"}
		if entry.path == "/documents/{id}/access-requests" {
			responses["202"] = map[string]any{"description": "문서 존재를 나타내지 않는 동일 접수 응답 {accepted:true,message}"}
			delete(responses, "200")
			responses["429"] = map[string]any{"description": "요청자 자신의 시간당/열린 요청 한도"}
		}
		if entry.path == "/documents/{id}/ai-selection" && entry.method == "post" {
			responses["200"] = map[string]any{"description": "완료되지 않은 proposal은 적용 불가. retract 발생 시 이미 표시한 내용도 폐기", "content": map[string]any{"text/event-stream": map[string]any{"schema": text}}}
		}
		if entry.path == "/access-requests" {
			op["parameters"] = []any{map[string]any{"name": "workspace_id", "in": "query", "required": true, "schema": uuid}}
		}
	}
}
