## VisitFlow v2.8.0

- 관리자가 시스템 설정의 `방문 추적` 탭에서 방문 추적 스크립트를 붙일 수 있다 (tracking.* 설정, 기본 꺼짐, 마이그레이션 0014). provider는 momento·ga4·gtm·matomo와 8KB 이하의 custom 스니펫. 페이지는 script-src 'self'로 잠겨 있으므로 SPA 핸들러가 스니펫의 모든 `<script>`에 요청별 nonce를 붙이고 스니펫이 부르는 http(s) 출처를 정책에 넣으며, 'unsafe-inline'은 쓰지 않는다. 끄면 정책이 이전과 바이트 단위로 같아지고 API·health·metrics 경로에는 스니펫이 들어가지 않는다. Momento는 같은 오리진 프록시(`/momento/*`, 쿠키·Authorization 제거)가 기본이며, 브라우저의 CSP 신고(`/api/v1/tracking/csp-report`)로 차단된 출처를 최대 100개 기록해 설정 화면에서 한 번에 허용 목록에 넣을 수 있다.
- `oidc.auto_login` 설정(기본 꺼짐)을 켜면 Keycloak에 세션이 있는 방문자는 로그인 화면을 보기 전에 prompt=none으로 조용히 로그인되어 원래 주소로 돌아온다. 탭당 한 번 시도·로그아웃 후 억제·`login?sso=none` 표시의 세 가드로 반복을 막고, 설정이 꺼져 있으면 서버가 prompt=none을 일반 로그인으로 낮춘다.
- 사용자 가이드와 관리자 가이드를 실제 화면 캡처 중심으로 다시 쓰고 PDF로 함께 배포한다. 방문 추적 설정과 CSP 통과 방식도 관리자 가이드에 실었다.

버전 표기(README, docs/index.html, web/package.json, web/package-lock.json)를 2.8.0으로 올린다.

**Full Changelog**: https://github.com/hkjang/visitflow/compare/v2.7.4...v2.8.0
