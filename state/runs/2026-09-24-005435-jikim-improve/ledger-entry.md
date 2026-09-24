## 2026-09-24
- 선택: Vite 개발 서버의 MCP OAuth 메타데이터 프록시 누락 보완 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `mcp_oauth.go`가 `/.well-known/oauth-protected-resource`와 `…/mcp`를 서빙하는데 `vite.config.ts`의 프록시 목록에는 없어, 개발 서버에서 관리 화면이 안내하는 메타데이터 주소와 `/mcp` 401의 `resource_metadata` 주소가 백엔드에 닿지 못했다(실제 왕복에서 404 확인). 프록시 항목 하나를 좁게(`/.well-known` 전체가 아니라 해당 접두사만) 추가하고 CONTRIBUTING 개발 안내에 주소를 적었다. 검증은 실제 운영 `vite.config.ts`를 로드해 진짜 Vite 개발 서버·임시 HTTP upstream·native fetch로 도는 기존 하네스에 테스트 6개를 먼저 추가해 수정 전 5개 실패(404)를 본 뒤 수정 후 통과시켰고, 되돌려 다시 실패함도 확인했다. `./scripts/verify.sh` 전체(Go test·vet·gofmt, npm ci·vitest 59개·lint·build, 문서·Compose) exit 0. 커밋 `8c7a4fe`.
- 보류 아이디어: settings GET 파생 필드의 PUT 왕복 저장 문제 (2/1/S, 관찰 가능한 변화가 없어 사실상 기각 방향) / 감사 로그 보존(audit_retention_days) 자동 정리 (3/3/M, 삭제·보존 계약 확정 필요) / baoKVWrite create·update 판정 TOCTOU (3/3/M, 트랜잭션·동시성 검증 필요) / requestedOpenBaoVersion이 음수 version을 오류 대신 latest로 처리 (2/2/S, OpenBao 사양 확인 불가로 보류) / Go 라우트와 Vite 프록시 목록이 어긋나도 아무도 알려주지 않는 구조적 공백 (3/2/M)
- 과제서: 채택 — 정찰이 고른 항목이 맞았다. 다만 정찰이 "개발 서버가 HTML을 준다"고 추측한 것과 달리 실제로는 404였고, Vite 8의 문자열 단축 프록시는 `changeOrigin: true`를 함의해 Host가 백엔드로 바뀌므로(정찰의 "Host 폴백 유지" 전제와 다름) `/mcp`와 같은 Host로 도착하는지를 대신 검증했다.
