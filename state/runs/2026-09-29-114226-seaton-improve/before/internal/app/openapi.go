package app

import "net/http"

func (s *Server) openAPI(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, 200, map[string]any{
		"openapi":    "3.1.0",
		"info":       map[string]any{"title": "SeatOn API", "version": s.version, "description": "오프라인 사내 좌석 관리 REST API. 브라우저 세션 또는 개인 Bearer API 키를 사용합니다. /mcp 는 관리자가 MCP SSO 인증을 켜면 Keycloak 액세스 토큰(OAuth 2.1)도 받으며, 인증 서버 위치는 /.well-known/oauth-protected-resource/mcp (RFC 9728) 에 있습니다."},
		"servers":    []map[string]string{{"url": "/api/v1"}},
		"components": map[string]any{"securitySchemes": map[string]any{"bearerApiKey": map[string]string{"type": "http", "scheme": "bearer", "bearerFormat": "SeatOn personal API key"}, "cookieSession": map[string]string{"type": "apiKey", "in": "cookie", "name": sessionCookie}}},
		"security":   []map[string]any{{"bearerApiKey": []string{}}, {"cookieSession": []string{}}},
		"paths": map[string]any{
			"/version":                map[string]any{"get": operation("서비스 버전 조회", false)},
			"/dashboard":              map[string]any{"get": operation("관리자 운영 요약", true)},
			"/dashboard/action-count": map[string]any{"get": operation("처리 필요 건수만 집계 (상단 배지용)", true)},
			"/dashboard/issues":       map[string]any{"get": operation("처리 필요 상세 목록", true)},
			"/dashboard/issues/{kind}/{issueID}/resolve": map[string]any{"post": operation("처리 필요 항목 즉시 조치", true)},
			"/employees":                      map[string]any{"get": operation("직원 검색", true), "post": operation("직원 등록/갱신", true)},
			"/organizations":                  map[string]any{"get": operation("조직 조회", true), "post": operation("조직 등록/갱신", true)},
			"/buildings":                      map[string]any{"get": operation("사업장 조회", true), "post": operation("사업장 등록", true)},
			"/floors":                         map[string]any{"get": operation("층 조회", true), "post": operation("층 등록", true)},
			"/floor-maps":                     map[string]any{"get": operation("도면 버전 조회", true), "post": operation("도면 업로드", true)},
			"/floor-maps/{mapID}":             map[string]any{"delete": operation("도면 버전 삭제 (게시 중이거나 이력이 있으면 409)", true)},
			"/floor-maps/{mapID}/unpublish":   map[string]any{"post": operation("도면 게시 내리기", true)},
			"/floor-maps/{mapID}/analyze":     map[string]any{"post": operation("좌석 후보 분석 시작 (202, engine=cv|vlm|hybrid)", true)},
			"/analysis-jobs/{jobID}":          map[string]any{"get": operation("분석 잡 진행 상태와 결과 조회", true)},
			"/settings/ai/vlm/test":           map[string]any{"post": operation("사내 VLM 엔드포인트 연결 시험", true)},
			"/settings/tracking/violations":   map[string]any{"get": operation("방문 추적 CSP 가 차단한 출처 목록", true), "delete": operation("차단 기록 비우기", true)},
			"/tracking/csp-report":            map[string]any{"post": operation("브라우저의 CSP 위반 신고 수신 (추적이 켜진 동안만 기록)", false)},
			"/floor-maps/{mapID}/preview":     map[string]any{"get": operation("좌석 오버레이 기준 래스터 이미지", true)},
			"/floor-maps/{mapID}/grid":        map[string]any{"put": operation("도면 좌석 격자 보정값 저장/해제", true)},
			"/floor-maps/{mapID}/seats/align": map[string]any{"post": operation("좌석을 도면 격자에 정렬", true)},
			"/seats":                          map[string]any{"get": operation("좌석 조회", true), "post": operation("좌석 생성", true)},
			"/seats/bulk":                     map[string]any{"patch": operation("좌석 위치 일괄 이동", true)},
			"/seats/{seatID}":                 map[string]any{"delete": operation("좌석 삭제", true)},
			"/seat-assignments":               map[string]any{"post": operation("직원 좌석 배정", true)},
			"/seat-assignments/{seatID}":      map[string]any{"delete": operation("좌석 배정 해제 (배정된 직원이 없으면 404 not_assigned)", true)},
			"/seat-assignments/bulk":          map[string]any{"post": operation("CSV/XLSX 좌석 일괄 배정", true)},
			"/seat-history":                   map[string]any{"get": operation("좌석 변경 이력", true)},
			"/api-keys":                       map[string]any{"get": operation("내 API 키 조회", true), "post": operation("개인 API 키 생성", true)},
			"/users":                          map[string]any{"get": operation("사용자 목록 (시스템 관리자)", true)},
			"/users/{userID}":                 map[string]any{"patch": operation("사용자 권한·메일 주소·사용 여부 변경 ({role, email, active}; 자기 계정 비활성화는 400 self_deactivation, 자기 계정의 권한 강등은 400 self_demotion, SSO 사용자 email 은 409 sso_managed_email)", true)},
		},
	})
}

func operation(summary string, auth bool) map[string]any {
	m := map[string]any{"summary": summary, "responses": map[string]any{"200": map[string]string{"description": "성공"}, "400": map[string]string{"description": "잘못된 요청"}, "403": map[string]string{"description": "권한 없음"}}}
	if !auth {
		m["security"] = []any{}
	}
	return m
}
