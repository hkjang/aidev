# 수리 요약 (5b23838)

- 비평 1건 맞았다. 재현: 동시 10건 실패 → `strikes 10 / 120초`(테스트가 `10 !== 1` 로 실패). `trip` 이 실패 1건마다 strike 를 더해 첫 블립이 바로 상한으로 점프했다.
- 고친 것: `Breaker` 에 `openedAt` 을 두고, 차단기가 열린 순간 이미 in-flight 였던 호출(`startedAt <= openedAt`)은 같은 라운드로 접어 strike 를 더하지 않는다. 대기가 지난 뒤 새로 시작한 호출의 실패만 strike 를 올린다. 접힌 실패도 `openUntil` 은 늦은 쪽으로 늘려, 대기가 지나고 늦게 끝난 실패가 차단기를 닫힌 채 두지 않는다. workerd 의 멈춘 시계를 위해 비교는 `<=`.
- 비평 2건(테스트가 N=2 만 고정)도 맞았다. 기존 테스트를 N=10 으로 바꿔 `strikes 1 / 10초` 를 못 박고, "먼저 쌓인 strikes 를 늦게 끝난 같은 라운드 실패가 되돌리지 않는다"(hold 모드로 2라운드 뒤 착지 → 20초/strikes 2) 를 새로 더했다. 두 테스트 모두 `server/db.ts` 를 b14cc46 으로 되돌리면 실패하므로 원래 고치려던 배선 결함도 그대로 지킨다.
- 확인: `db-breaker` 13/13(3회), `npm test` 217/217, `typecheck`, `build`, `test:runtime` 2/2. 테스트 삭제·단언 완화·검증 명령 수정은 없다.
- 범위 밖 발견(수정 안 함): 전체 묶음에 **원래부터 있던** 간헐 실패가 있다 — `tests/bill-analysis.ts:237`(`order by created_at desc limit 1`, 같은 테넌트에 다른 target 의 `bill.analysis.read` 감사행이 여럿) 과 `ending staff role revokes authored proxy cases…`. `server/db.ts` 를 b14cc46 으로 되돌린 5회 실행에서도 나왔고, `bill-analysis.mjs` 는 `db-breaker.mjs` 보다 먼저 돌며 `trip` 은 DB 에 못 닿을 때만 실행되므로 이 변경과 무관하다.
