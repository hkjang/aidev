# SeatOn API 및 MCP

## 인증

관리자 화면의 프로필 메뉴에서 개인 API 키를 생성한다. 키 원문은 생성 또는 회전 직후 한 번만 표시되고 서버에는 설치별 마스터 키로 계산한 HMAC-SHA-256 값만 저장된다.

```http
Authorization: Bearer seat_xxxxxxxxxxxxxxxxxxxxxxxxx
```

범위는 다음과 같다.

| 범위 | 권한 |
| --- | --- |
| `read` | REST 조회 |
| `write` | REST 변경 및 MCP 배정 도구 |
| `mcp` | MCP 연결 |

키 회전 시 새 버전이 생성되고, 기존 키는 관리자가 설정한 유예시간 동안만 계속 유효하다. 즉시 폐기하면 유예 없이 무효화된다.

### MCP 의 SSO(OAuth 2.1) 인증

관리자가 `mcp.oauth.enabled` 를 켜면 `/mcp` 는 같은 `Authorization: Bearer` 헤더로 Keycloak 액세스 토큰도 받는다(`seat_` 접두사면 키, JWT 모양이면 토큰). SeatOn 은 리소스 서버이고 인증 서버는 Keycloak 이다.

- `GET /.well-known/oauth-protected-resource` · `GET /.well-known/oauth-protected-resource/mcp` — RFC 9728 메타데이터. 인증 없음, 맨 JSON, `Access-Control-Allow-Origin: *`. 꺼져 있으면 `404 mcp_oauth_disabled`
- `/mcp` 의 `401` 에 `WWW-Authenticate: Bearer realm="SeatOn", resource_metadata="…/.well-known/oauth-protected-resource/mcp"`(거부된 토큰이면 `, error="invalid_token"`). REST 401 에는 붙지 않는다
- 검사: JWKS 서명(RS/ES/PS 만), `iss`=`oidc.issuer_url`, `exp`·`nbf`, `typ=ID` 거부, `cnf` 있으면 거부, `sub` 필수, 대상(`aud` 에 리소스 식별자 또는 `aud`/`azp` 가 `mcp.oauth.audience` 에)
- 계정: `preferred_username`(없으면 `email`)으로 이미 웹 SSO 로그인으로 등록된 활성 계정만. 없으면 `401 account_not_registered`. 범위는 `mcp.oauth.scopes`(기본 `read mcp`)가 정하고, 토큰이 `read`/`write`/`mcp` 를 실어 오면 교집합. 교집합이 비면 `403 insufficient_scope`
- OAuth 토큰은 `/mcp` 에서만 받는다. REST 경로에 내면 `401 authentication_required`
- Keycloak discovery 에 닿지 못하면 `503 sso_unavailable`(도전 헤더 없음)

## REST API

OpenAPI 3.1 문서는 실행 중인 SeatOn의 `/api/v1/openapi.json`에서 확인한다. 대표 경로는 다음과 같다.

- `GET /api/v1/employees?q=&status=&assignment=` 직원/조직 및 재직·배정 상태 검색
- `GET /api/v1/seats?floorMapId=` 좌석 및 배정 조회
- `PATCH /api/v1/seats/bulk` 다중 좌석 위치·회전 일괄 저장
- `POST /api/v1/seat-assignments` 좌석 배정
- `POST /api/v1/seat-assignments/bulk` CSV/XLSX 일괄 배정
- `DELETE /api/v1/seat-assignments/{seatID}` 좌석 배정 해제. 좌석이 비고 이력이 남습니다. 배정된 직원이 없으면 `404 not_assigned`
- `POST /api/v1/floor-maps/{id}/unpublish` 게시 내리기. 게시 중이 아니면 `409 map_not_published`
- `DELETE /api/v1/floor-maps/{id}` 도면 버전 삭제. 게시 중이면 `409 map_published`, 배정·변경 이력이 걸린 좌석이 있으면 `409 map_in_use`
- `POST /api/v1/floor-maps/{id}/analyze?engine=cv|vlm|hybrid` 도면 분석 시작, `202`와 함께 `jobId` 반환. `engine`을 생략하면 `ai.engine` 설정값을 사용
- `GET /api/v1/analysis-jobs/{jobId}` 분석 진행 상태, 결과 건수, 경고 목록 조회
- `POST /api/v1/settings/ai/vlm/test` 사내 VLM 엔드포인트 연결·응답 형식 시험
- `GET /api/v1/settings/tracking/violations` 방문 추적 CSP가 차단한 출처 목록(시스템 관리자). `DELETE`는 기록 비우기
- `POST /api/v1/tracking/csp-report` 브라우저의 CSP 위반 신고 수신. 인증 없음, 추적이 켜진 동안만 메모리에 기록
- `GET /momento/tracker.js` · `POST /momento/collect/v1/events` Momento 같은 오리진 프록시. 추적이 Momento·프록시 구성일 때만 열리고 그 밖에는 `404`
- `GET /api/v1/floor-maps/{id}/preview` 좌석 오버레이 기준 래스터 이미지, PDF는 첫 페이지를 PNG로 변환해 제공
- `PUT /api/v1/floor-maps/{id}/grid` 도면 좌석 격자 보정값 저장, 빈 본문 `{}`은 해제
- `POST /api/v1/floor-maps/{id}/seats/align` 좌석을 도면 격자에 정렬, `seatIds` 생략 시 도면 전체
- `GET /api/v1/seat-history?q=&source=&from=&to=&limit=` 변경 이력 검색. `from`/`to`는 RFC3339 시각이며 `to`는 열린 구간
- `GET /api/v1/dashboard` 운영 준비도와 처리 필요 건수
- `GET /api/v1/dashboard/action-count` 처리 필요 건수만 집계, 상단 배지처럼 자주 부르는 곳에 사용
- `GET /api/v1/dashboard/issues?kind=` 처리 필요 상세 작업 큐
- `POST /api/v1/dashboard/issues/{kind}/{id}/resolve` 퇴직자 좌석 해제, AI 후보 승인, 조직 영역 보정
- `GET /api/v1/users` 사용자 목록(시스템 관리자). `active`로 사용 여부를 함께 준다
- `PATCH /api/v1/users/{id}` `{"role":"…","email":"…","active":true|false}` 중 보낸 항목만 바꾼다. `email`은 빈 문자열이면 지운다. 자기 계정 비활성화는 `400 self_deactivation`, 자기 계정의 `role`을 `system_admin`이 아닌 값으로 낮추면 `400 self_demotion`, SSO 사용자의 `email`은 Keycloak 프로필을 따르므로 `409 sso_managed_email`

## MCP

SeatOn은 MCP Streamable HTTP 형식의 단일 엔드포인트를 제공한다.

```text
POST https://seaton.example.intra/mcp
Authorization: Bearer seat_...          # 개인 키
Authorization: Bearer eyJhbGciOi...     # 또는 Keycloak 액세스 토큰 (mcp.oauth.enabled 일 때)
```

지원 도구:

- `search_employees`: 이름, 사번, 이메일, 조직으로 직원/좌석 검색
- `list_available_seats`: 층별 빈 좌석 조회
- `get_floor_map`: 도면 메타데이터와 비율 좌표 조회
- `get_action_items`: 관리자 처리 필요 항목 조회
- `assign_seat`: 좌석 관리자 + `write` 범위로 좌석 배정

변경 도구 호출은 REST와 동일하게 감사로그와 좌석 변경 이력을 남긴다.
