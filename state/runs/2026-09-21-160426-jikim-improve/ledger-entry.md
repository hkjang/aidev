## 2026-09-21
- 선택: streamChat의 콜백 예외를 JSON 파싱 오류로 오인해 원문을 다시 전달하는 문제 수정 (가치 2 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: JSON.parse만 catch하도록 경계를 좁혀 onChunk의 Error/SyntaxError를 동일 객체로 전파하고 해당 이벤트 중복 호출 및 뒤 이벤트 전달을 막았다(커밋 4e21a71). 실제 loopback HTTP 서버→native fetch→Response/ReadableStream→운영 streamChat 회귀 테스트 9개를 먼저 작성해 수정 전 6개 실패를 확인했고, 수정 후 9개·전체 41개와 Node 24.21.0의 lint/build 및 ./scripts/verify.sh(Go test·vet, npm ci·test·lint·build, 문서·Compose)가 통과했다. 요청한 technology 스킬/Skill 도구는 제공 목록과 로컬 검색에서 발견하지 못했으며 로컬 superpowers systematic-debugging·test-driven-development·verification-before-completion을 대신 적용했다; 실제 AiPage의 자연 발생 예외·PostgreSQL opt-in E2E는 미검증이다.
- 보류 아이디어:
  - Vite 개발 서버의 OpenAPI 링크 프록시 누락 보완 (2/1/S): 차선 유지, 런타임 재현은 다음 회차.
  - settings GET 파생 필드의 PUT 왕복 저장 문제 (2/1/S): 인접 변경 충돌 회피.
  - audit_retention_days 자동 정리 (3/3/M): 삭제·보존 계약 필요.
  - baoKVWrite create/update 판정 TOCTOU (3/3/M): 트랜잭션·동시성 검증 필요.
- 과제서: 채택 — HEAD 9ea43d0의 catch 범위와 실제 HTTP 실패 재현이 과제서 근거에 일치했고 최소 경계 수정으로 수용 기준을 충족했다.
