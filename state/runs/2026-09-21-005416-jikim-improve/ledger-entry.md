## 2026-09-21
- 선택: AI 채팅의 HTTP 오류 처리와 세션 만료 처리를 일반 API 클라이언트와 일치시키기 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: api와 streamChat의 비성공 HTTP 응답 처리를 공통화하여 JSON/text 메시지·status·details 계약 및 요청당 401 알림을 일치시켰고 사용자 가이드에 반영했습니다. 실제 loopback HTTP/native fetch/Response/ReadableStream 및 실제 AuthProvider 통합 테스트를 먼저 추가하여 수정 전 13개 실패를 확인했고, 수정 후 대상 43개와 전체 70개 테스트 및 Node 24.21.0의 ./scripts/verify.sh(Go test·vet, npm ci·test·lint·build, 문서·Compose), 별도 go test ./...를 통과했습니다. 커밋 d6dd63f; PostgreSQL opt-in/실제 브라우저 로그인·Keycloak은 미검증이며 요청한 technology 스킬/Skill 도구는 발견하지 못해 고유 절차 적용을 주장하지 않습니다(로컬 superpowers systematic-debugging·test-driven-development를 대신 읽고 적용).
- 보류 아이디어:
  - streamChat onChunk 예외가 JSON 파싱 실패로 오인되어 재호출되는 문제 (2/1/S): 차선은 이번 범위 밖.
  - settings GET 파생 필드가 PUT 왕복에서 저장되는 문제 (2/1/S): 인접 변경 충돌 회피.
  - audit_retention_days 자동 정리 (3/3/M): 삭제·보존 계약 필요.
  - baoKVWrite create/update 판정 TOCTOU (3/3/M): 트랜잭션·동시성 검증 필요.
- 과제서: 채택 — 현재 코드에도 streamChat의 JSON 오류 원문 노출·401 알림 누락이 남아 있었고 실제 HTTP 및 AuthProvider 실패 테스트로 재현했습니다.
