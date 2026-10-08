## 2026-10-09
- 선택: 가이드 추적 캡처의 CSP 보고에 제한 시간을 적용해 전역 설정 복원 경계로 빠져나오게 한다 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: 커밋 de09ba8; 프로덕션 guide-shots.mjs 1파일과 guide-settings-check.test.mjs 1파일을 수정해 기존 AGENTHUB_GUIDE_REQUEST_TIMEOUT_MS(미설정 30000ms)를 실제 captureTracking 호출부와 page.evaluate 직렬화 인자를 거쳐 CSP fetch의 AbortSignal.timeout에 연결했다. 실제 호출부→실제 captureTracking→실제 withGuideSettings를 실행하는 신규 회귀 7건은 정상 POST/Content-Type/본문/204 판정/촬영·정리 순서/skip, 진짜 AbortSignal의 50ms 중단 뒤 네 원본 복원 시도와 복원 실패보다 원래 보고 오류 우선을 검증하며, signal·호출부 전달·직렬화 인자를 각각 제거하면 5/6/6건 실패하고 원복 뒤 통과했다. node --check web/scripts/guide-shots.mjs, cd web && node --test scripts/guide-settings-check.test.mjs(80 pass), cd web && node --test scripts/*.test.mjs(154 pass, fail/skip 0), npm ci && npm run lint && npm run test:sso && npm run build, go test -race -p 1 ./cmd/... ./internal/..., 이미지 check-versions, kubectl kustomize, docker compose config --quiet, git diff --check가 통과했으며 실제 브라우저·관리자 DB는 미검증(AGENTHUB_TEST_DSN 없음), npm ci의 기존 의존성 high 경고 2건은 별도 후보로 기록했다.
- 실패 재현: `not ok 52 - a stalled real tracking report restores all settings despite no restoration failure` / `actual: 'the request was sent without a deadline and would never return'` (프로덕션 수정 전 80건 중 75 pass / 5 fail; 정상 경로도 timeout: undefined 대 75/30000으로 실패).
- 보류 아이디어: 복원 실패가 guide-shots의 problems 요약 출력을 건너뛰는 문제 (2/1/S).
  guide-shots 추적 캡처의 기존 CSP 위반 전체 삭제 방지 (3/2/M).
  web/scripts Node 회귀 테스트를 CI 기본 검증에 포함 (3/2/S).
  가이드 요청 제한 시간 환경변수의 유효 범위를 시작 시 검사 (2/1/S).
- 과제서: 채택 — CSP fetch에만 signal이 없고 외부 withGuideSettings 안에서 대기하는 구조가 현재 코드와 일치했으며, 내부 finally를 옮기지 않고 제한 시간 배선만 추가했다.
