package server

import "net/http"

func builtinTemplates() []map[string]any {
	return []map[string]any{
		{"id": "builtin-meeting", "name": "회의록", "description": "안건과 결정, 다음 행동을 함께 남기는 회의 기록입니다.", "category": "팀 협업", "icon": "💬", "kind": "meeting", "markdown": "---\ntags: [회의]\n---\n# 회의록 · {{date}}\n\n## 일시와 참석자\n\n- 일시: {{datetime}}\n- 참석자: \n\n## 안건\n\n1. \n\n## 논의 내용\n\n## 결정 사항\n\n## 다음 행동\n\n- [ ] \n", "tags": []string{"회의"}, "version": 1, "builtin": true},
		{"id": "builtin-decision", "name": "의사결정 기록 (ADR)", "description": "배경과 선택지, 결정을 내린 이유를 보존합니다.", "category": "의사결정", "icon": "🧭", "kind": "decision", "markdown": "---\ntags: [ADR]\n---\n# 의사결정 기록\n\n- 작성일: {{date}}\n- 작성자: \n\n## 배경\n\n## 검토한 선택지\n\n| 선택지 | 장점 | 단점 |\n| --- | --- | --- |\n| | | |\n\n## 결정\n\n## 결정 사유\n\n## 예상 영향과 후속 작업\n\n- [ ] \n\n## 관련 문서\n", "tags": []string{"ADR"}, "version": 1, "builtin": true},
		{"id": "builtin-runbook", "name": "운영 런북", "description": "사전 조건, 검증, 되돌리기 절차까지 포함하는 운영 가이드입니다.", "category": "IT 운영", "icon": "🛠️", "kind": "runbook", "markdown": "---\ntags: [운영, 런북]\n---\n# 운영 런북\n\n## 목적\n\n## 소유자와 마지막 검증\n\n- 소유자: \n- 마지막 검증: {{date}}\n\n## 사전 조건\n\n## 실행 절차\n\n1. \n\n```shell\n# 실행 전 내용을 작성하고 검토하세요.\n```\n\n## 결과 검증\n\n- [ ] 정상 상태 확인\n\n## 되돌리기\n\n## 관련 문서\n", "tags": []string{"운영", "런북"}, "version": 1, "builtin": true},
		{"id": "builtin-daily", "name": "오늘의 노트 · {{date}}", "description": "오늘의 생각과 할 일을 가볍게 정리합니다. 기존 일일 노트 기능도 그대로 사용할 수 있습니다.", "category": "개인 기록", "icon": "☀️", "kind": "daily", "markdown": "---\ntags: [일일노트]\n---\n# {{date}}\n\n## 오늘의 목표\n\n- [ ] \n\n## 메모\n\n## 배운 점\n\n## 내일 이어갈 일\n\n- [ ] \n", "tags": []string{"일일노트"}, "version": 1, "builtin": true},
		{"id": "builtin-project", "name": "프로젝트 계획", "description": "목표, 범위, 일정과 위험 요소를 한곳에 모읍니다.", "category": "프로젝트", "icon": "🚀", "kind": "page", "markdown": "# 프로젝트 계획\n\n## 목표\n\n## 범위\n\n### 포함하는 것\n\n### 제외하는 것\n\n## 일정\n\n| 단계 | 담당자 | 목표일 |\n| --- | --- | --- |\n| | | |\n\n## 위험과 대응\n\n## 다음 행동\n\n- [ ] \n", "tags": []string{}, "version": 1, "builtin": true},
		{"id": "builtin-retrospective", "name": "주간 회고", "description": "성과와 배운 점을 돌아보고 다음 주의 작은 실험을 정합니다.", "category": "팀 협업", "icon": "🌱", "kind": "note", "markdown": "# 주간 회고 · {{date}}\n\n## 이번 주 성과\n\n## 잘한 점\n\n## 개선할 점\n\n## 배운 점\n\n## 다음 주 시도할 일\n\n- [ ] \n", "tags": []string{}, "version": 1, "builtin": true},
		{"id": "builtin-knowledge", "name": "지식 정리", "description": "핵심 개념과 적용 예시를 연결해 오래 남는 지식을 만듭니다.", "category": "지식 관리", "icon": "📚", "kind": "page", "markdown": "# 지식 정리\n\n## 한 줄 요약\n\n> \n\n## 핵심 개념\n\n## 적용 예시\n\n## 참고 자료\n\n## 연결된 지식\n\n[[관련 문서]]\n", "tags": []string{}, "version": 1, "builtin": true},
	}
}
func (s *Server) templateBuiltins(w http.ResponseWriter, r *http.Request) {
	if !s.canWorkspace(r.Context(), current(r), r.URL.Query().Get("workspace_id"), false) || !hasIntegrationScope(current(r), "document:read") {
		apiError(w, 403, "워크스페이스 접근 권한이 없습니다")
		return
	}
	jsonResponse(w, 200, builtinTemplates())
}
