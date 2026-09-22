## 2026-09-22
- 선택: Vite 개발 서버의 OpenAPI 링크 프록시 누락 보완 (가치 2 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `/api/openapi.json`을 기존 Go 백엔드 8080으로 전달하는 프록시 항목과 개발 실행 안내를 추가했다(커밋 `7f171d0`); API 탐색기 링크와 Go 공개 라우트는 유지하고, Node 환경 회귀 테스트를 위해 공통 setup의 브라우저 초기화만 조건부로 실행한다. 실제 운영 Vite 설정·임시 HTTP upstream·native fetch 회귀 9개에서 수정 전 두 실패(브라우저 Accept는 HTML, JSON Accept는 404)를 확인한 뒤 모두 통과했으며, 별도 Go 공개 routes/httptest→실제 Vite 왕복에서도 두 Accept 모두 JSON의 `openapi: 3.1.0` 및 Go 직접 응답과 본문 동일을 확인했다. Node 24.21.0의 npm ci·전체 53개 테스트·lint·build, go test ./..., 최종 ./scripts/verify.sh가 모두 exit 0이며 PostgreSQL 포함 서비스 기동·브라우저 클릭은 미검증이고 요청한 technology 스킬/Skill 도구는 없어 로컬 superpowers debugging·TDD·verification-before-completion을 대신 적용했다.
- 보류 아이디어:
  - settings GET 파생 필드의 PUT 왕복 저장 문제 (2/1/S): 인접 변경 충돌 회피.
  - 감사 로그 보존 자동 정리 (3/3/M): 삭제·보존 계약 확정 필요.
  - baoKVWrite create/update 판정 TOCTOU (3/3/M): 트랜잭션·동시성 검증 필요.
  - Transit 프리뷰 안내의 batch 지원 계약 정합성 (2/1/S): 선택 과제 해결로 차선 미실행.
- 과제서: 채택 — 누락된 프록시와 실제 HTTP 실패가 과제서 근거에 일치했으며 좁은 항목 추가로 해결했고, Node 테스트에 필요한 setup 가드 외 범위를 확장하지 않았다.
