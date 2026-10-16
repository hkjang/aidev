## 2026-10-17
- 선택: DB 차단기 백오프가 동시 요청에서 누적되지 않는 것 고치기 (가치 4 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: `server/db.ts:trip` 이 `baseTx` 가 호출 시작 시점에 캡처한 breaker 대신 트립하는 순간 `breakers.get(key)` 를 다시 읽어 strikes 를 더하게 했고, 키별 `closedAt` 을 두어 DB 가 답해 차단기를 닫은 뒤에 끝난 실패는 무시하게 했다(거짓 "잠시 점검 중입니다" 방지, `resetDatabaseBreakers()` 도 함께 비움). `tests/db-breaker.ts` 에 `Promise.allSettled` 로 겹쳐 돌리는 동시 실패 테스트와, 드로퍼의 새 'hold' 모드(연결을 받고 답하지 않아 5초 connect timeout)로 "닫힌 뒤 늦게 실패" 를 재현하는 테스트를 먼저 추가해 실패를 확인한 뒤 고쳤다. 검증은 새로 띄운 127.0.0.1:55462 postgres:17-alpine + `db:migrate` 에서 `node --test artifacts/db-breaker.mjs` 12/12(3회), `npm test` 216/216, `npm run typecheck`, `npm run build`, `npm run test:runtime` 2/2.
- 실패 재현: `not ok 6 - 동시에 실패한 호출들의 strikes 가 누적된다` → `actual: { open: true, retryAfterSeconds: 10, strikes: 1 }` / `expected: { retryAfterSeconds: 20, strikes: 2 }`; `not ok 7 - 차단기가 닫힌 뒤에 끝난 실패는 차단기를 다시 열지 않는다` → `actual: { open: true, retryAfterSeconds: 10, strikes: 1 }` / `expected: { open: false, retryAfterSeconds: 0, strikes: 0 }`
- 보류 아이디어: 차단기 재개 순간의 thundering herd 를 단일 프로브로 좁히기(3/3/M, 이번 과제서의 차선 후보 — 그대로 유효) / `src/api.ts` 가 JSON 이 아닌 503 에서 db_unavailable 신호를 놓치는 것 보강(3/2/S) / 기한 초과 카운트와 알림 다이제스트의 grace_days 불일치(3/3/M, 제품 판단 필요) / `DegradedNotice` 가 숨겨진 탭에서도 30초마다 폴링(2/1/S) / 통합 테스트가 재사용된 로컬 DB 에서 누적 데이터로 깨지는 것(새로 발견, 3/2/M — 같은 DB 에 `npm test` 를 두 번째·세 번째로 돌리면 integration·record-handoff 에서 403/404·403/409 불일치가 나고, 새 DB 에서는 216/216 통과한다)
- 과제서: 채택 — 과제서의 근거(`trip(key, breaker)` 가 캡처한 값을 쓰는 것)가 코드와 정확히 일치했고 제안한 수용 기준 4개를 그대로 구현·검증했다.

