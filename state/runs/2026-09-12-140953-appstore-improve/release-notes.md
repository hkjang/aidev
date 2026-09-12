AppStore v2.6.0

관리자가 화면에서 방문 추적 스크립트를 붙이고, 서비스가 nonce 기반 CSP로 그 스크립트만 허용하는 minor 릴리스입니다.

- 관리자 → 방문 추적(`/admin/analytics`) 화면에서 provider(Momento가 첫 자리)와 스니펫, 삽입 위치, 관리자 화면 포함 여부, 허용 출처를 설정합니다. 설정은 `system_settings.analytics` 한 행에 저장되고 기본값은 꺼짐이라, 새로 설치한 곳과 기존 설치는 화면을 손대기 전까지 아무것도 달라지지 않습니다.
- 모든 화면이 `script-src 'self'`로 잠겨 있어 스니펫을 그냥 붙이면 조용히 차단되므로, 새 `ContentSecurityPolicy` 미들웨어가 추적이 켜진 페이지 요청마다 nonce를 만들어 `script-src`에 넣고 SPA 핸들러가 같은 nonce를 스니펫의 모든 `<script>`에 붙여 index.html에 삽입합니다. provider 출처, 스니펫에서 읽어 낸 http(s) 출처, 관리자가 더한 허용 출처가 `script-src`·`connect-src`·`img-src`에 더해지고 `'unsafe-inline'`은 쓰지 않습니다. 추적을 끄면 정책은 원래 문자열로 돌아가고, `/api/*`·`/mcp`·`/health*`·`/momento/*` 같은 비화면 경로는 `default-src 'none'`으로 더 좁아집니다.
- 추적이 켜진 동안에만 `report-uri`를 정책에 넣어 브라우저가 거부한 요청을 `POST /api/v1/analytics/csp-report`(8KB 상한, 언제나 204)로 받아 메모리에 출처·지시어 기준 100건까지 보관하고, 관리 화면의 차단된 출처 표에서 한 번 눌러 허용 목록에 넣을 수 있습니다.
- Momento용 같은 오리진 프록시(`/momento/*`)를 기본으로 켜 두면 스니펫이 `/momento/tracker.js`와 `/momento`만 보므로 외부 출처가 정책에 등장하지 않습니다. 프록시는 전달 전에 이 서비스의 세션 쿠키와 Authorization 헤더를 떼어 내고 15초·256KB 상한을 두며, 추적이 꺼져 있으면 404입니다.
- 관리자 라우트 `GET/PUT /admin/analytics`, `GET/DELETE /admin/analytics/violations`, `POST /admin/analytics/allowed-hosts`와 `AnalyticsSettings`·`CspViolation` 스키마를 OpenAPI에 추가했습니다.
- 관리자 가이드에 4.9 절(설정표, Momento 프록시, CSP 동작, 차단된 출처 사용법)과 장애 대응·보안 기본값 항목을 더하고 PDF를 다시 구웠습니다. 새 라우트 캡처 2장과 사이드바 항목이 늘어난 관리자 캡처를 갱신했습니다.
- Schema 변경이 없어 기존 설치는 image만 교체하면 됩니다. 실제 수집기로 수집이 들어오는지는 이 환경에 Momento가 없어 확인하지 못했습니다.
