package server

import (
	"net/http"
	"net/url"
	"strings"
)

// OpenAPI is assembled from a compact route catalogue. URLs stay relative so the
// same contract works behind an internal reverse proxy without an internet host.
func (s *Server) openAPI(w http.ResponseWriter, r *http.Request) {
	type route struct {
		method, path, summary, tag, body string
		public                           bool
	}
	catalogue := []route{
		{"get", "/public", "서비스 버전과 공개 설정", "인증", "", true},
		{"get", "/public-shares/{share}", "비밀 링크와 현재 정책 검증 후 외부 공유 문서 조회", "공개 공유", "", true},
		{"post", "/public-shares/{share}/unlock", "공유 암호를 검증하고 짧은 접근 토큰 발급", "공개 공유", "ModuleRequest", true},
		{"get", "/public-shares/{share}/attachments/{file}", "현재 공유·정보보호 정책 내 첨부 다운로드", "공개 공유", "", true},
		{"post", "/auth/login", "로컬 계정 로그인", "인증", "Login", true},
		{"post", "/auth/logout", "현재 세션 종료", "인증", "", false},
		{"get", "/auth/me", "인증 사용자 조회", "인증", "", false},
		{"get", "/auth/oidc/start", "Keycloak OIDC 로그인 시작", "인증", "", true},
		{"get", "/auth/oidc/callback", "OIDC 코드 교환 콜백", "인증", "", true},
		{"post", "/auth/ldap/login", "TLS 디렉터리 계정 로그인", "인증", "LDAPLogin", true},
		{"get", "/auth/saml/start", "서명된 SAML 인증 요청 시작", "인증", "", true},
		{"get", "/auth/saml/metadata", "SAML SP 공개 Metadata XML", "인증", "", true},
		{"post", "/auth/saml/acs", "서명·브라우저·재사용 검증 후 SAML 로그인", "인증", "SAMLResponse", true},
		{"get", "/profile", "개인 프로필과 설정 조회", "개인화", "", false},
		{"put", "/profile", "프로필·개인 설정·비밀번호 변경", "개인화", "Profile", false},
		{"get", "/workspaces", "접근 가능한 워크스페이스 조회", "워크스페이스", "", false},
		{"post", "/workspaces", "워크스페이스 생성", "워크스페이스", "Name", false},
		{"get", "/workspaces/{id}/members", "워크스페이스 멤버 조회", "워크스페이스", "", false},
		{"put", "/workspaces/{id}/members", "멤버 추가 또는 역할 변경", "워크스페이스", "Member", false},
		{"delete", "/workspaces/{id}/members/{userId}", "멤버 제거", "워크스페이스", "", false},
		{"get", "/documents", "문서 제목·본문·태그 검색 및 목록", "문서", "", false},
		{"post", "/documents", "Markdown 문서 생성", "문서", "DocumentCreate", false},
		{"get", "/documents/{id}", "Markdown 원문·메타데이터 조회", "문서", "", false},
		{"put", "/documents/{id}", "버전 충돌 검사 후 문서 수정", "문서", "DocumentUpdate", false},
		{"delete", "/documents/{id}", "문서를 휴지통으로 이동", "문서", "", false},
		{"post", "/documents/{id}/restore", "휴지통 문서 복원", "문서", "", false},
		{"post", "/documents/{id}/favorite", "즐겨찾기 토글", "문서", "", false},
		{"get", "/documents/{id}/versions", "문서 버전 이력 조회", "문서", "", false},
		{"post", "/documents/{id}/versions/{version}/restore", "과거 버전을 새 버전으로 복원", "문서", "", false},
		{"get", "/documents/{id}/backlinks", "권한 내 백링크 조회", "문서", "", false},
		{"get", "/documents/{id}/citation", "현재 권한·버전·UTF-8 byte 범위·SHA-256로 인용 원문 확인", "검색", "", false},
		{"post", "/documents/{id}/relations", "원본 작성·대상 열람 권한 및 버전 검사 후 수동 관계 연결", "문서", "DocumentRelation", false},
		{"delete", "/documents/{id}/relations/{target}", "명시적 수동 관계 해제 (문서는 삭제하지 않음)", "문서", "", false},
		{"get", "/documents/{id}/rag-index", "현재 공급자와 문서별 색인 동의·진행 상태 조회", "AI 검색", "", false},
		{"post", "/documents/{id}/rag-index", "저장본·공급자 명시 동의 후 비동기 임베딩 색인", "AI 검색", "RAGIndexConsent", false},
		{"delete", "/documents/{id}/rag-index", "동의 철회·벡터 삭제·진행 중 응답 무효화", "AI 검색", "RAGIndexRevoke", false},
		{"post", "/documents/{id}/rag-index/cancel", "색인 작업 중지 및 자동 재색인 중지", "AI 검색", "RAGIndexCancel", false},
		{"get", "/workspaces/{id}/search-ai", "워크스페이스 AI 검색 유효 설정·상속·비밀 설정 여부", "AI 검색", "", false},
		{"post", "/workspaces/{id}/search-ai/test", "저장된 공급자로 고정된 비민감 문구만 전송하여 진단", "AI 검색", "", false},
		{"post", "/admin/search-ai/test", "서비스 임베딩 공급자 고정 문구 진단", "AI 검색", "", false},
		{"get", "/documents/{id}/collaboration", "쿠키 세션 기반 Yjs 실시간 협업 WebSocket", "문서", "", false},
		{"get", "/documents/{id}/comments", "댓글 조회", "문서", "", false},
		{"post", "/documents/{id}/comments", "문서 댓글 작성", "문서", "Comment", false},
		{"get", "/documents/{id}/shares", "문서 공유 대상 조회", "문서", "", false},
		{"put", "/documents/{id}/shares", "사용자별 공유 권한 설정", "문서", "Share", false},
		{"post", "/documents/{id}/shares", "이메일로 문서 공유", "문서", "Member", false},
		{"post", "/documents/{id}/approval", "관리자 정책에 따른 검토·승인·반려", "문서", "Approval", false},
		{"get", "/documents/{id}/approval", "문서의 현재 승인 정책·단계·담당자 조회", "승인", "", false},
		{"get", "/approvals/resources/{kind}/{id}", "현재 권한 범위 내 리소스 승인 현황", "승인", "", false},
		{"get", "/approvals/requests/{id}", "검토 요청의 고정된 원본·정책·결정 이력", "승인", "", false},
		{"post", "/approvals/requests/{id}/decisions", "검토 버전 검사 후 승인·반려·취소", "승인", "ApprovalDecision", false},
		{"get", "/approvals/inbox", "현재 담당 단계의 개인 검토함", "승인", "", false},
		{"get", "/documents/{id}/runbook", "현재 문서 권한 내 격리 운영 절차·마지막 검증 조회", "격리 실행", "", false},
		{"put", "/documents/{id}/runbook", "버전 검사 후 허용 명령 기반 실행·검증·롤백 절차 저장", "격리 실행", "ModuleRequest", false},
		{"post", "/documents/{id}/runbook/prepare", "실제 원격 격리 검증 후 불변 실행 계획 준비 (사용자 세션 전용)", "격리 실행", "RunbookPrepare", false},
		{"get", "/documents/{id}/runbook/executions", "문서 실행 이력 조회", "격리 실행", "", false},
		{"get", "/runbook/runners", "현재 사용자에게 허용된 실행기·명령 목록", "격리 실행", "", false},
		{"get", "/runbook/executions/{id}", "고정된 실행 계획·승인·단계 상태 조회", "격리 실행", "", false},
		{"post", "/runbook/executions/{id}/approval", "Runbook 명시 정책으로 계획 검토 제출", "격리 실행", "RunbookSubmit", false},
		{"post", "/runbook/executions/{id}/execute", "정확한 사용자 확인·선택적 승인 소비·단일 작업 등록", "격리 실행", "RunbookConfirm", false},
		{"post", "/runbook/executions/{id}/cancel", "원격 Job ID의 중지 요청 (확인 전 성공으로 표시하지 않음)", "격리 실행", "RunbookConfirm", false},
		{"get", "/runbook/executions/{id}/events", "현재 세션·문서·실행 권한 재검증 SSE 출력", "격리 실행", "", false},
		{"get", "/admin/runbook/settings", "서비스 격리 실행 사용 설정 조회", "격리 실행 관리", "", false},
		{"put", "/admin/runbook/settings", "설정 revision 검사 후 기능 켜기·끄기", "격리 실행 관리", "ModuleRequest", false},
		{"get", "/admin/runbook/runners", "비밀값을 제외한 실행기 설정 조회", "격리 실행 관리", "", false},
		{"post", "/admin/runbook/runners", "AWX/Kubernetes 고정 허용 작업·암호화 자격 증명 등록", "격리 실행 관리", "ModuleRequest", false},
		{"put", "/admin/runbook/runners/{id}", "실행기 버전 검사 후 새 설정 버전 저장", "격리 실행 관리", "ModuleRequest", false},
		{"post", "/admin/runbook/runners/{id}/test", "저장된 실행기 TLS·고정 이미지·격리 검사 (실행 없음)", "격리 실행 관리", "", false},
		{"get", "/admin/runbook/runners/{id}/versions", "실행기 설정 버전 이력", "격리 실행 관리", "", false},
		{"get", "/admin/runbook/runners/{id}/versions/{version}", "비밀값 제외 과거 실행기 설정", "격리 실행 관리", "", false},
		{"get", "/admin/runbook/uncertain", "운영자 확인이 필요한 불확실 원격 실행", "격리 실행 관리", "", false},
		{"post", "/admin/runbook/executions/{id}/resolve", "외부 확인 근거·명시 RESOLVE 문구로 수동 취소 종결 (자동 검증 아님)", "격리 실행 관리", "RunbookResolve", false},
		{"get", "/graph", "문서 연결 그래프 조회", "탐색", "", false},
		{"get", "/tasks", "문서의 Markdown 할 일 조회", "탐색", "", false},
		{"put", "/tasks", "할 일 완료 상태 변경", "탐색", "Task", false},
		{"get", "/tasks/board", "문서·표 안 할 일과 담당자·기한·상태 조회", "할 일", "", false},
		{"put", "/tasks/details", "원문 위치·버전 검사 후 할 일 속성 변경", "할 일", "TaskDetails", false},
		{"get", "/tasks/calendar", "현재 문서·DB 권한과 키 범위 내 통합 일정", "할 일", "", false},
		{"post", "/tasks/calendar/events", "회의·마일스톤 일정 생성", "할 일", "CalendarEvent", false},
		{"put", "/tasks/calendar/events/{id}", "소유자의 일정 버전 검사 후 변경", "할 일", "CalendarEvent", false},
		{"delete", "/tasks/calendar/events/{id}", "소유자의 일정 삭제 (연결 원문 유지)", "할 일", "", false},
		{"get", "/captures", "소유한 비공개 수집함 목록", "수집", "", false},
		{"post", "/capture-hooks/{id}", "HMAC 서명·재전송 방지 후 비공개 수집", "수집", "ModuleRequest", true},
		{"get", "/search", "현재 권한 내 문서·블록·코드·할 일·파일·댓글·태그·DB 통합 검색", "검색", "", false},
		{"post", "/captures", "텍스트·URL을 멱등한 비공개 문서로 수집", "수집", "Capture", false},
		{"post", "/captures/{id}/classify", "수집 문서 버전 검사 후 명시적 분류·공유", "수집", "CaptureClassify", false},
		{"get", "/documents/{id}/attachments", "문서 권한 내 첨부파일 메타데이터 목록", "파일", "", false},
		{"get", "/databases", "데이터베이스 목록 조회", "데이터베이스", "", false},
		{"post", "/databases", "속성 정의와 데이터베이스 생성", "데이터베이스", "Database", false},
		{"get", "/databases/{id}", "데이터베이스 속성 조회", "데이터베이스", "", false},
		{"put", "/databases/{id}", "데이터베이스 이름·속성 변경", "데이터베이스", "Database", false},
		{"delete", "/databases/{id}", "데이터베이스와 모든 행 영구 삭제", "데이터베이스", "", false},
		{"get", "/databases/{id}/rows", "행 조회", "데이터베이스", "", false},
		{"post", "/databases/{id}/rows", "유형 검증 후 행 생성", "데이터베이스", "Row", false},
		{"put", "/databases/{id}/rows/{rowId}", "유형 검증 후 행 수정", "데이터베이스", "Row", false},
		{"delete", "/databases/{id}/rows/{rowId}", "행 삭제", "데이터베이스", "", false},
		{"post", "/attachments", "최대 50MB 첨부파일 업로드", "파일", "Upload", false},
		{"get", "/attachments/{id}", "문서 권한 검사 후 첨부파일 다운로드", "파일", "", false},
		{"get", "/export", "워크스페이스 Markdown Vault ZIP 내보내기", "파일", "", false},
		{"post", "/import", "Markdown 또는 Vault ZIP 가져오기", "파일", "Upload", false},
		{"get", "/keys", "개인 또는 위임 서비스 계정 API 키 조회", "연동", "", false},
		{"post", "/keys", "워크스페이스 범위 제한 API 키 발급", "연동", "APIKey", false},
		{"put", "/keys/{id}", "API 키 권한·IP·호출 정책 변경", "연동", "APIKey", false},
		{"delete", "/keys/{id}", "API 키 즉시 폐기", "연동", "", false},
		{"post", "/keys/{id}/rotate", "기존 키를 즉시 폐기하고 새 키 발급", "연동", "", false},
		{"post", "/ai/chat", "문서 권한을 적용한 AI SSE 스트리밍", "연동", "AIChat", false},
		{"post", "/search/ai/proposal", "명시 동의 후 자연어 검색 조건 제안 (자동 검색 없음)", "검색", "NaturalSearch", false},
		{"get", "/ai/actions", "사용 가능한 AI 문서 지원 작업", "연동", "", false},
		{"post", "/ai/conversations", "명시 동의 후 완료한 개인 AI 답변 저장", "개인화", "AIHistorySave", false},
		{"get", "/ai/conversations", "현재 출처 권한을 적용한 내 대화 검색", "개인화", "", false},
		{"get", "/ai/conversations/{id}", "본인의 저장한 질문·답변·인용 조회", "개인화", "", false},
		{"delete", "/ai/conversations/{id}", "본인 대화 기록 영구 삭제", "개인화", "AIHistoryDelete", false},
		{"post", "/mcp", "HTTP MCP JSON-RPC 요청", "연동", "MCP", false},
		{"get", "/notifications", "개인 알림 조회", "개인화", "", false},
		{"post", "/notifications/{id}/read", "알림 읽음 처리", "개인화", "", false},
		{"get", "/admin/stats", "관리자 통계 조회", "관리자", "", false},
		{"get", "/admin/users", "서비스 사용자 조회", "관리자", "", false},
		{"post", "/admin/users", "로컬 사용자·서비스 계정 생성", "관리자", "User", false},
		{"put", "/admin/users/{id}", "사용자 권한·활성·비밀번호 변경", "관리자", "User", false},
		{"get", "/admin/settings", "비밀을 숨긴 서비스 설정 조회", "관리자", "", false},
		{"put", "/admin/settings", "검증·암호화 후 서비스 설정 변경", "관리자", "Settings", false},
		{"get", "/admin/settings/history", "설정 변경 이력 조회", "관리자", "", false},
		{"post", "/admin/settings/history/{id}/restore", "이전 운영 설정 복원", "관리자", "", false},
		{"get", "/admin/audit", "감사 로그 조회", "관리자", "", false},
		{"get", "/admin/backup", "서비스 논리 백업 ZIP 다운로드", "관리자", "", false},
		{"post", "/admin/restore", "명시적 확인 후 서비스 논리 백업 복원", "관리자", "Restore", false},
	}
	// Modules cannot silently disappear from the API catalogue. Exact schemas
	// for common operations are above; newly registered module routes retain a
	// permissive JSON schema until their module guide supplies detailed fields.
	known := map[string]bool{}
	for _, entry := range catalogue {
		known[entry.method+" "+entry.path] = true
	}
	for _, pattern := range s.apiRoutes {
		method, path, ok := strings.Cut(pattern, " ")
		if !ok || !strings.HasPrefix(path, "/api/v1/") {
			continue
		}
		method = strings.ToLower(method)
		path = strings.TrimPrefix(path, "/api/v1")
		if known[method+" "+path] {
			continue
		}
		known[method+" "+path] = true
		body := ""
		if oneOf(method, "post", "put", "patch") {
			body = "ModuleRequest"
		}
		tag := "확장 API"
		if strings.HasPrefix(path, "/admin/") {
			tag = "관리자"
		}
		verbs := map[string]string{"get": "조회", "post": "실행·생성", "put": "설정 변경", "patch": "부분 변경", "delete": "삭제"}
		catalogue = append(catalogue, route{method, path, verbs[method] + " · " + path, tag, body, false})
	}
	textType := map[string]any{"type": "string"}
	integerType := map[string]any{"type": "integer"}
	booleanType := map[string]any{"type": "boolean"}
	objectType := map[string]any{"type": "object", "additionalProperties": true}
	stringsType := map[string]any{"type": "array", "items": textType}
	object := func(properties map[string]any, required ...string) map[string]any {
		v := map[string]any{"type": "object", "properties": properties}
		if len(required) > 0 {
			v["required"] = required
		}
		return v
	}
	schemas := map[string]any{
		"LDAPLogin":        object(map[string]any{"username": textType, "password": map[string]any{"type": "string", "format": "password", "writeOnly": true}}, "username", "password"),
		"SAMLResponse":     object(map[string]any{"RelayState": textType, "SAMLResponse": textType}, "RelayState", "SAMLResponse"),
		"RunbookPrepare":   object(map[string]any{"phase": map[string]any{"type": "string", "enum": []string{"execute", "validate", "rollback"}}}, "phase"),
		"RunbookSubmit":    object(map[string]any{"version": map[string]any{"type": "integer", "minimum": 1}, "comment": textType}, "version"),
		"RunbookConfirm":   object(map[string]any{"version": map[string]any{"type": "integer", "minimum": 1}, "approval_version": map[string]any{"type": "integer", "minimum": 1}, "confirmation": map[string]any{"type": "string", "description": "EXECUTE 실행-ID 또는 CANCEL 실행-ID 정확히 입력; 로그인한 실제 요청자만 실행 가능"}}, "confirmation"),
		"RunbookResolve":   object(map[string]any{"confirmation": textType, "reason": map[string]any{"type": "string", "minLength": 20, "maxLength": 2000}, "external_checked": map[string]any{"type": "boolean", "const": true}}, "confirmation", "reason", "external_checked"),
		"ModuleRequest":    objectType,
		"Login":            object(map[string]any{"email": textType, "password": textType}, "email", "password"),
		"Name":             object(map[string]any{"name": textType}, "name"),
		"Profile":          object(map[string]any{"name": textType, "preferences": objectType, "password": textType, "current_password": textType}),
		"Member":           object(map[string]any{"email": textType, "role": textType}, "email", "role"),
		"DocumentCreate":   object(map[string]any{"workspace_id": textType, "title": textType, "markdown": textType, "tags": stringsType, "aliases": stringsType, "parent_id": textType, "visibility": map[string]any{"enum": []string{"workspace", "private", "selected"}}, "icon": textType}, "workspace_id", "title"),
		"DocumentUpdate":   object(map[string]any{"version": integerType, "title": textType, "markdown": textType, "tags": stringsType, "aliases": stringsType, "parent_id": map[string]any{"type": []string{"string", "null"}}, "visibility": textType, "status": textType}, "version"),
		"DocumentRelation": object(map[string]any{"target_id": map[string]any{"type": "string", "format": "uuid"}, "type": map[string]any{"type": "string", "enum": []string{"related", "reference"}}, "expected_version": map[string]any{"type": "integer", "minimum": 1}}, "target_id", "type", "expected_version"),
		"RAGIndexConsent":  object(map[string]any{"expected_version": map[string]any{"type": "integer", "minimum": 1}, "provider_fingerprint": textType, "consent": map[string]any{"type": "boolean", "const": true}, "auto_reindex": booleanType, "allow_rerank": booleanType, "rerank_fingerprint": textType}, "expected_version", "provider_fingerprint", "consent"),
		"RAGIndexRevoke":   object(map[string]any{"grant_revision": map[string]any{"type": "integer", "minimum": 1}}, "grant_revision"),
		"RAGIndexCancel":   object(map[string]any{"job_id": map[string]any{"type": "string", "format": "uuid"}}, "job_id"),
		"Comment":          object(map[string]any{"body": textType}, "body"),
		"Share":            object(map[string]any{"user_id": textType, "permission": map[string]any{"enum": []string{"read", "write", "remove"}}}, "user_id", "permission"),
		"Approval":         object(map[string]any{"action": map[string]any{"enum": []string{"submit", "approve", "reject", "cancel"}}, "comment": textType, "request_id": textType, "request_version": integerType, "gate_index": integerType}, "action"),
		"ApprovalDecision": object(map[string]any{"action": map[string]any{"enum": []string{"approve", "reject", "cancel"}}, "comment": textType, "request_version": integerType, "gate_index": integerType}, "action", "request_version"),
		"Task":             object(map[string]any{"document_id": textType, "line": integerType, "done": booleanType, "version": integerType}, "document_id", "line", "done", "version"),
		"TaskDetails":      object(map[string]any{"document_id": textType, "line": integerType, "source_start": integerType, "version": integerType, "task_id": textType, "status": map[string]any{"enum": []string{"backlog", "todo", "doing", "review", "done"}}, "priority": map[string]any{"enum": []string{"low", "normal", "high", "urgent"}}, "assignee_id": textType, "due_date": textType, "separate": booleanType}, "document_id", "line", "version", "status", "priority"),
		"CalendarEvent":    object(map[string]any{"workspace_id": textType, "title": textType, "kind": map[string]any{"enum": []string{"meeting", "milestone"}}, "document_id": textType, "start_date": textType, "end_date": textType, "visibility": map[string]any{"enum": []string{"private", "workspace"}}, "version": integerType}, "workspace_id", "title", "kind", "start_date", "end_date", "visibility"),
		"Capture":          object(map[string]any{"workspace_id": textType, "title": textType, "text": textType, "url": textType, "tags": stringsType, "client_request_id": textType}, "workspace_id"),
		"CaptureClassify":  object(map[string]any{"version": integerType, "space_id": textType, "parent_id": textType, "visibility": textType, "kind": textType}, "version", "visibility", "kind"),
		"Database":         object(map[string]any{"workspace_id": textType, "name": textType, "properties": map[string]any{"type": "array", "items": object(map[string]any{"id": textType, "name": textType, "type": textType, "options": stringsType}, "id", "name", "type")}}),
		"Row":              object(map[string]any{"values": objectType}, "values"),
		"APIKey":           object(map[string]any{"name": textType, "workspace_id": textType, "user_id": textType, "scopes": stringsType, "expires_in_days": integerType, "ip_allowlist": stringsType, "rate_limit": integerType}),
		"AIChat":           object(map[string]any{"prompt": textType, "document_id": textType, "workspace_id": textType, "conversation_id": textType, "conversation_version": map[string]any{"type": "integer", "minimum": 1}, "action": map[string]any{"enum": []string{"ask", "write", "rewrite", "summarize", "translate", "tags", "links", "duplicates", "meeting", "template", "gaps"}, "default": "ask"}, "max_tokens": map[string]any{"type": "integer", "minimum": 1, "maximum": 262144}}, "prompt"),
		"AIHistorySave":    object(map[string]any{"ticket": textType, "consent": map[string]any{"const": true}}, "ticket", "consent"),
		"NaturalSearch":    object(map[string]any{"workspace_id": textType, "prompt": map[string]any{"type": "string", "maxLength": 8000}, "consent": map[string]any{"const": true}}, "workspace_id", "prompt", "consent"),
		"AIHistoryDelete":  object(map[string]any{"confirmation": map[string]any{"const": "DELETE"}, "version": integerType}, "confirmation", "version"),
		"MCP":              object(map[string]any{"jsonrpc": map[string]any{"const": "2.0"}, "id": map[string]any{}, "method": textType, "params": objectType}, "jsonrpc", "method"),
		"User":             object(map[string]any{"email": textType, "name": textType, "password": textType, "role": map[string]any{"enum": []string{"admin", "editor", "viewer"}}, "kind": map[string]any{"enum": []string{"user", "service"}}, "disabled": booleanType}),
		"Settings":         objectType,
		"Upload":           object(map[string]any{"file": map[string]any{"type": "string", "format": "binary"}}, "file"),
		"Restore":          object(map[string]any{"file": map[string]any{"type": "string", "format": "binary"}, "confirmation": map[string]any{"const": "RESTORE"}}, "file", "confirmation"),
		"Error":            object(map[string]any{"error": textType, "request_id": textType}, "error"),
	}
	paths := map[string]any{}
	for _, entry := range catalogue {
		parameters := []any{}
		for _, segment := range strings.Split(entry.path, "/") {
			if strings.HasPrefix(segment, "{") {
				name := strings.Trim(segment, "{}")
				parameters = append(parameters, map[string]any{"name": name, "in": "path", "required": true, "schema": textType})
			}
		}
		if oneOf(entry.path, "/documents", "/graph", "/tasks", "/tasks/board", "/tasks/calendar", "/captures", "/databases", "/import", "/export", "/spaces", "/canvases", "/jobs", "/automations", "/webhooks") {
			parameters = append(parameters, map[string]any{"name": "workspace_id", "in": "query", "schema": textType})
		}
		if entry.path == "/documents" && entry.method == "get" {
			for _, name := range []string{"q", "tag", "trash", "favorite", "limit", "offset"} {
				parameters = append(parameters, map[string]any{"name": name, "in": "query", "schema": textType})
			}
		}
		if entry.path == "/documents/{id}/citation" {
			for _, name := range []string{"version", "start", "end", "hash"} {
				parameters = append(parameters, map[string]any{"name": name, "in": "query", "required": true, "schema": textType})
			}
		}
		if entry.path == "/documents/{id}/backlinks" {
			parameters = append(parameters, map[string]any{"name": "offset", "in": "query", "schema": map[string]any{"type": "integer", "minimum": 0, "maximum": 10000}})
		}
		if entry.path == "/documents/{id}/relations/{target}" {
			for _, name := range []string{"type", "expected_version"} {
				parameters = append(parameters, map[string]any{"name": name, "in": "query", "required": true, "schema": textType})
			}
		}
		if entry.path == "/attachments" {
			parameters = append(parameters, map[string]any{"name": "document_id", "in": "query", "required": true, "schema": textType})
			parameters = append(parameters, map[string]any{"name": "client_request_id", "in": "query", "schema": textType, "description": "선택 UUID. 같은 파일의 재전송을 중복 업로드하지 않습니다."})
		}
		if entry.path == "/tasks/calendar" {
			parameters = append(parameters, map[string]any{"name": "month", "in": "query", "required": true, "schema": textType, "description": "YYYY-MM"})
			parameters = append(parameters, map[string]any{"name": "document_id", "in": "query", "schema": map[string]any{"type": "string", "format": "uuid"}, "description": "선택 문서의 현재 권한을 적용하여 해당 문서 날짜·할 일·실제 연결 일정을 조회합니다. 독립 일정과 문서 관계가 없는 DB 행은 포함하지 않으며, 잘못된 ID를 전체 범위로 대체하지 않습니다."})
		}
		if entry.path == "/tasks/board" {
			for _, name := range []string{"document_id", "task"} {
				parameters = append(parameters, map[string]any{"name": name, "in": "query", "schema": textType})
			}
		}
		if entry.path == "/captures" || entry.path == "/documents/{id}/attachments" {
			parameters = append(parameters, map[string]any{"name": "after", "in": "query", "schema": textType, "description": "직전 목록의 마지막 ID"})
		}
		if entry.path == "/keys" && entry.method == "get" {
			parameters = append(parameters, map[string]any{"name": "user_id", "in": "query", "description": "관리자 세션에서 서비스 계정 등 다른 소유자의 키를 조회할 때 지정", "schema": textType})
		}
		if entry.path == "/search" {
			for _, name := range []string{"workspace_id", "q", "type", "space_id", "author_id", "tag", "status", "from", "to", "has_attachment", "sort", "limit", "offset"} {
				parameters = append(parameters, map[string]any{"name": name, "in": "query", "required": name == "workspace_id", "schema": textType})
			}
		}
		responses := map[string]any{"200": map[string]any{"description": "성공. JSON 객체/배열 또는 명시된 스트림/파일을 반환합니다."}, "400": map[string]any{"description": "입력값 검증 오류"}, "401": map[string]any{"description": "인증 필요 또는 키/세션 만료"}, "403": map[string]any{"description": "권한, 키 scope, 워크스페이스 또는 CSRF 정책 위반"}, "404": map[string]any{"description": "리소스 없음 또는 접근 불가"}, "409": map[string]any{"description": "문서 버전 충돌 또는 중복 데이터"}, "429": map[string]any{"description": "호출량 제한 (Retry-After 헤더)"}}
		op := map[string]any{"summary": entry.summary, "tags": []string{entry.tag}, "operationId": entry.method + strings.NewReplacer("/", "_", "{", "", "}", "").Replace(entry.path), "responses": responses}
		if entry.path == "/tasks/board" {
			responses["200"] = map[string]any{"description": "현재 권한을 통과한 최신 문서 최대 2,000개·본문 16MiB·할 일 10,000개 범위. total_documents_exact=true일 때만 total_documents가 정확한 수입니다. 2,001개 도달 시 total_documents_is_lower_bound=true로 하한을 반환하며 워크스페이스 전체 문서 수가 아닙니다. truncated와 limits를 함께 확인하세요.", "content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"type": "object", "properties": map[string]any{
				"items": map[string]any{"type": "array", "items": map[string]any{"type": "object"}}, "documents_scanned": map[string]any{"type": "integer", "minimum": 0, "maximum": 2000},
				"total_documents": map[string]any{"type": "integer", "minimum": 0, "maximum": 2001}, "total_documents_exact": map[string]any{"type": "boolean"}, "total_documents_is_lower_bound": map[string]any{"type": "boolean"},
				"truncated": map[string]any{"type": "boolean"}, "notice": textType, "limits": map[string]any{"type": "object"},
			}}}}}
		}
		if len(parameters) > 0 {
			op["parameters"] = parameters
		}
		if entry.public {
			op["security"] = []any{}
		} else if entry.path == "/mcp" {
			op["security"] = []any{map[string]any{"bearerAuth": []string{}}}
		} else if strings.HasPrefix(entry.path, "/admin/") || strings.HasPrefix(entry.path, "/keys") || strings.HasPrefix(entry.path, "/profile") || strings.Contains(entry.path, "/members") || strings.HasPrefix(entry.path, "/notifications") || entry.path == "/auth/logout" || (entry.path == "/workspaces" && entry.method != "get") {
			op["security"] = []any{map[string]any{"cookieAuth": []string{}}}
		}
		if entry.path == "/capture-hooks/{id}" {
			op["description"] = "관리자 허용 정책과 소유자가 만든 채널의 HMAC-SHA256 서명이 필요합니다. X-Madi-Timestamp(Unix 초, ±5분) + '.' + X-Madi-ID(UUID) + '.' + 원본 요청 바이트를 채널 비밀로 서명합니다. X-Madi-Signature는 sha256=와 16진수 서명입니다. 동일 ID·동일 payload는 202 duplicate=true, 다른 내용 재사용은 409. JSON {title,text,url?,attachments?:[{name,type,data:base64}]} 최대 10MB. 쿠키·Bearer 키로 서명을 대체할 수 없습니다."
			for _, name := range []string{"X-Madi-ID", "X-Madi-Timestamp", "X-Madi-Signature"} {
				parameters = append(parameters, map[string]any{"name": name, "in": "header", "required": true, "schema": textType})
			}
			op["parameters"] = parameters
			responses["202"] = map[string]any{"description": "수집 작업 접수 또는 멱등한 중복 확인"}
			delete(responses, "200")
		}
		if !entry.public {
			probe := &http.Request{Method: strings.ToUpper(entry.method), URL: &url.URL{Path: "/api/v1" + entry.path}}
			if !integrationScopeAllowed(&Principal{TokenID: "catalogue", Scopes: keyScopes}, probe) || strings.HasSuffix(entry.path, "/collaboration") {
				op["security"] = []any{map[string]any{"cookieAuth": []string{}}}
			}
		}
		if strings.HasPrefix(entry.path, "/scim/v2/") {
			op["security"] = []any{map[string]any{"bearerAuth": []string{}}}
			op["description"] = "SCIM 2.0: 활성 서비스 계정의 workspace 고정 identity:provision 키만 허용합니다. 관리자 정책에서 해당 scope와 SCIM 사용을 모두 활성화해야 합니다. 개인 키·플러그인·쿠키 세션으로 접근할 수 없습니다. SCIM 필드는 RFC 규격의 camelCase입니다."
			op["x-madi-required-scope"] = scimProvisionScope
		}
		if !entry.public && entry.method != "get" {
			parameters = append(parameters, map[string]any{"name": "X-Madi-Request", "in": "header", "description": "쿠키 세션의 변경 요청에는 1을 지정합니다. Bearer 요청은 제외됩니다.", "schema": map[string]any{"type": "string", "enum": []string{"1"}}})
			op["parameters"] = parameters
		}
		if entry.path == "/keys" && entry.method == "post" {
			responses["201"] = responses["200"]
			delete(responses, "200")
		}
		if strings.HasPrefix(entry.path, "/auth/oidc/") {
			responses["302"] = map[string]any{"description": "SSO 제공자 또는 로그인된 앱으로 리디렉션"}
			delete(responses, "200")
		}
		if entry.path == "/auth/saml/start" || entry.path == "/auth/saml/acs" {
			responses["302"] = map[string]any{"description": "IdP 또는 인증된 앱으로 리디렉션"}
			delete(responses, "200")
		}
		if entry.path == "/auth/saml/metadata" {
			responses["200"] = map[string]any{"description": "SP 공개 인증서 및 지원 SAML 바인딩", "content": map[string]any{"application/samlmetadata+xml": map[string]any{"schema": textType}}}
		}
		if entry.body != "" {
			contentType := "application/json"
			if entry.body == "Upload" || entry.body == "Restore" {
				contentType = "multipart/form-data"
			}
			if entry.body == "SAMLResponse" {
				contentType = "application/x-www-form-urlencoded"
			}
			if strings.HasPrefix(entry.path, "/scim/v2/") {
				contentType = "application/scim+json"
			}
			op["requestBody"] = map[string]any{"required": true, "content": map[string]any{contentType: map[string]any{"schema": map[string]any{"$ref": "#/components/schemas/" + entry.body}}}}
		}
		if strings.HasPrefix(entry.path, "/scim/v2/") {
			if entry.method == "post" {
				responses["201"] = responses["200"]
				delete(responses, "200")
			}
			if entry.method == "delete" {
				responses["204"] = map[string]any{"description": "리소스 삭제 완료"}
				delete(responses, "200")
			}
			if oneOf(entry.method, "put", "patch", "delete") {
				responses["412"] = map[string]any{"description": "If-Match ETag 불일치"}
				parameters = append(parameters, map[string]any{"name": "If-Match", "in": "header", "schema": textType, "description": "직전 조회의 ETag. 오래된 버전에 대한 변경을 거부합니다."})
				op["parameters"] = parameters
			}
		}
		if entry.path == "/ai/chat" {
			responses["200"] = map[string]any{"description": "SSE: {sources:[{id,title,version,citation_id,start_byte,end_byte,start_line,end_line,content_hash,citation_url}],retrieval:{mode,backend,scanned,truncated,reranked,warnings},action}, {text}, 완료 시 개인 브라우저만 {history_ticket,history_expires_in:3600}, optional {error,retract:true}, [DONE]. retract 수신 시 이전 답변·출처·저장 티켓을 모두 지워야 합니다. 기록 저장은 별도 명시 동의 POST입니다.", "content": map[string]any{"text/event-stream": map[string]any{"schema": textType}}}
		}
		if entry.path == "/ai/conversations" && entry.method == "post" {
			responses["201"] = map[string]any{"description": "개인 대화 생성; 동일 응답 티켓 재시도는 200과 기존 ID"}
		}
		if entry.path == "/search/ai/proposal" {
			responses["200"] = map[string]any{"description": "SSE {text} → 검증된 {proposal:{explanation,plan,query,date_basis},automatic_apply:false} → [DONE]. 실패/권한 변경의 {error,retract:true}는 이전 출력과 제안을 제거해야 합니다. 개인 쿠키 전용. 사용자 확인 후 일반 GET /search에 반환된 query를 적용하며 직접 선택한 공간/소유자 필터를 유지합니다.", "content": map[string]any{"text/event-stream": map[string]any{"schema": textType}}}
		}
		if entry.path == "/documents/{id}/rag-index" && entry.method == "post" {
			responses["202"] = map[string]any{"description": "동의와 job을 같은 트랜잭션에 저장함. 완료 전에는 벡터 검색에 사용하지 않음."}
			delete(responses, "200")
		}
		if strings.HasSuffix(entry.path, "/collaboration") {
			responses["101"] = map[string]any{"description": "WebSocket로 업그레이드. 동일 Origin/활성 쿠키와 현재 문서 ACL을 확인합니다."}
			delete(responses, "200")
			op["x-websocket"] = true
		}
		if entry.path == "/export" || entry.path == "/admin/backup" {
			responses["200"] = map[string]any{"description": "ZIP 파일", "content": map[string]any{"application/zip": map[string]any{"schema": map[string]any{"type": "string", "format": "binary"}}}}
		}
		if paths[entry.path] == nil {
			paths[entry.path] = map[string]any{}
		}
		protectionOpenAPI(entry.path, entry.method, op)
		transferOpenAPI(entry.path, entry.method, op)
		migrationResumeOpenAPI(entry.path, entry.method, op)
		attachmentExtractionOpenAPI(entry.path, entry.method, op)
		knowledgePathsOpenAPI(entry.path, entry.method, op)
		structuredDraftOpenAPI(entry.path, entry.method, op)
		paths[entry.path].(map[string]any)[entry.method] = op
	}
	workspaceAgentOpenAPI(paths, schemas)
	supportOpenAPI(paths, schemas)
	graphAIOpenAPI(paths, schemas)
	documentSummaryOpenAPI(paths, schemas)
	searchOperationsOpenAPI(paths, schemas)
	uxReviewOpenAPI(paths, schemas)
	knowledgeOperationsOpenAPI(paths, schemas)
	distributionOpenAPI(paths, schemas)
	systemStatusOpenAPI(paths, schemas)
	impactExceptionOpenAPI(paths, schemas)
	knowledgeQuestionsOpenAPI(paths, schemas)
	knowledgeConflictsOpenAPI(paths)
	documentQueryOpenAPI(paths, schemas)
	schemas["SearchHistorySettings"] = object(map[string]any{"enabled": booleanType, "retention_days": map[string]any{"type": "integer", "minimum": 7, "maximum": 365}, "revision": integerType, "consent": booleanType}, "enabled", "retention_days", "revision")
	schemas["SearchHistoryDelete"] = object(map[string]any{"workspace_id": textType, "confirmation": map[string]any{"const": "DELETE_ALL"}}, "confirmation")
	schemas["SearchHistoryGaps"] = object(map[string]any{"workspace_id": textType, "entries": map[string]any{"type": "array", "minItems": 1, "maxItems": 30, "items": object(map[string]any{"id": textType, "revision": integerType}, "id", "revision")}, "consent": map[string]any{"const": true}}, "workspace_id", "entries", "consent")
	for _, entry := range []struct{ path, method, schema string }{{"/profile/search-history/settings", "put", "SearchHistorySettings"}, {"/search/history", "delete", "SearchHistoryDelete"}, {"/search/history/gaps", "post", "SearchHistoryGaps"}} {
		if path, ok := paths[entry.path].(map[string]any); ok {
			if op, ok := path[entry.method].(map[string]any); ok {
				op["requestBody"] = map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": "#/components/schemas/" + entry.schema}}}}
				op["description"] = "개인 사용자 쿠키 세션 전용. 다른 사용자·관리자·API 키·플러그인 우회 없음. 기본 검색어 저장 동의 꺼짐."
				if entry.schema == "SearchHistoryGaps" {
					op["responses"].(map[string]any)["200"] = map[string]any{"description": "SSE {text} → {proposal:{notice,suggestions:[{title,reason,outline,references}],sources},automatic_apply:false} → [DONE]. 현재 개인 동의·선택 기록 revision/만료·세션·공급자 변경 시 {error,retract:true}로 이전 출력 제거. 생성만으로 저장·게시하지 않음.", "content": map[string]any{"text/event-stream": map[string]any{"schema": textType}}}
				}
			}
		}
	}
	schemas["AIChat"].(map[string]any)["properties"].(map[string]any)["selected_documents"] = map[string]any{"type": "array", "maxItems": 10, "items": object(map[string]any{"id": textType, "version": map[string]any{"type": "integer", "minimum": 1}}, "id", "version"), "description": "선택한 현재 검색 문서만 요약. 각 문서 원문 조각 1개(약6KiB), 선택 외 문서로 검색 확장 없음. document_id/conversation_id와 함께 사용 불가."}
	jsonResponse(w, 200, map[string]any{"openapi": "3.1.0", "info": map[string]any{"title": "madi REST API", "version": s.Version, "description": "한국어 자체 호스팅 지식관리 API. JSON은 snake_case. 쿠키 세션 변경 요청은 X-Madi-Request: 1 및 동일 Origin이 필요합니다. API 키는 Bearer 인증을 사용하며 관리자/개인 설정 변경은 세션으로만 가능합니다. 키 scope와 소유자의 현재 ACL이 모두 적용됩니다."}, "servers": []any{map[string]any{"url": "/api/v1"}}, "security": []any{map[string]any{"bearerAuth": []string{}}, map[string]any{"cookieAuth": []string{}}}, "paths": paths, "components": map[string]any{"schemas": schemas, "securitySchemes": map[string]any{"bearerAuth": map[string]any{"type": "http", "scheme": "bearer", "bearerFormat": "madi API key"}, "cookieAuth": map[string]any{"type": "apiKey", "in": "cookie", "name": "madi_session"}}}})
}
