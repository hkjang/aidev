오프라인망 반입용 단일 Docker 이미지입니다.

- Docker image: `jupiq:v1.6.0`
- Architecture: `linux/amd64`
- PostgreSQL: `14 이상`

데이터베이스 migration은 기동 시 자동 적용됩니다(이번 버전에는 새 마이그레이션이 없습니다). 업그레이드 전에 PostgreSQL과 `ENCRYPTION_KEY`를 백업하세요. 이전 버전으로 내릴 때 schema downgrade는 자동으로 수행되지 않습니다.

## What's Changed
* feat: 관리자가 화면에서 방문 추적 스크립트를 붙일 수 있게 한다 — 관리자 설정 "방문 추적" 탭(Momento·GA4·GTM·Matomo·직접 붙여넣기), SPA 응답 CSP의 요청별 nonce, 같은 오리진 Momento 프록시(`GET /momento/tracker.js`, `POST /momento/collect/*`), `POST /api/v1/analytics/csp-report`, `GET/DELETE /api/v1/analytics/violations`. 기본값은 꺼짐이라 기존 설치에서는 아무것도 달라지지 않습니다.
* docs: 관리자 가이드에 방문 추적 설정과 CSP 설명을 더한다 (3.2·3.5·7.3절, PDF 갱신)

**Full Changelog**: https://github.com/hkjang/jupiq/compare/v1.5.0...v1.6.0
