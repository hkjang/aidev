## 방문 추적 스크립트 + nonce 기반 CSP

- 관리자가 시스템 설정 화면에서 방문 추적 스니펫을 붙일 수 있습니다 (`tracking.*` 설정, 기본 꺼짐, provider 첫 자리 `momento`, `momento_proxy` 기본 켜짐).
- Web UI 가 요청마다 nonce 를 만들어 `script-src 'self' 'nonce-…'` CSP 로 잠급니다 (style 만 `'unsafe-inline'`). 레이아웃 자체 스크립트에 nonce 를 붙이고, 인라인 `onsubmit` 확인 핸들러는 `data-confirm` 위임 핸들러로 옮겼습니다.
- 추적이 켜진 화면에만 스니펫·출처·`report-uri /ui/csp-report` 가 붙고, CSP 신고는 메모리 기록기(최근 100건)로 모아 설정 화면 "차단된 출처" 표에서 한 번에 허용/지우기 할 수 있습니다.
- `/momento/*` 는 같은 오리진 리버스 프록시(쿠키·Authorization 제거)로 넘기고, `/api/*`·정적 파일·`/ui/jobs/status` 는 `default-src 'none'` 을 받습니다.
- ADMIN_GUIDE 6.3 절 추가.

**Full Changelog**: https://github.com/hkjang/postra/compare/v0.18.5...v0.18.6
