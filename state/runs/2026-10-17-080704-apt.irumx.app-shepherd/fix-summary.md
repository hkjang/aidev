# 수리 요약 (51df83c)

- 비평 1 재현 확인: 같은 주소로 동시 10건을 실패시키면 `{open:true, retryAfterSeconds:120, strikes:10}`. 실제 오류는 전부 `ECONNRESET` — 비평가가 말한 블립 모양 그대로였다.
- 고침: `Breaker.openedAt` 추가. 차단기가 열린 순간 이미 돌고 있던 호출의 실패는 그 라운드의 strike 를 유지하고 `openUntil` 만 늦은 쪽으로 미룬다. 대기가 끝난 뒤 시작한 호출의 실패만 strike 를 올려 10→20→…→120 램프를 지킨다.
- 비평 2(N=2 는 의미 구분 못 함) 수용: N=2 테스트를 N=10 묶음이 `strikes 1 / 10초` 임을 못 박는 테스트로 바꾸고, "늦게 끝난 같은 라운드 실패가 쌓인 strikes 를 되돌리지 않는다"(strikes 2 유지)를 더했다. 뒤 테스트는 b14cc46 과 23e2a12 양쪽에서 실패한다 — 원래 배선 결함도 계속 지킨다.
- 검증: `node --test artifacts/db-breaker.mjs` 13/13(3회), 새 컨테이너+`db:migrate` 후 `npm test` 217/217(3회, 재실행 1회 포함), `npm run typecheck`, `npm run build`, `npm run test:runtime` 2/2. e2e·test:assets 는 돌리지 않았다.
- 남은 점: 첫 전체 실행 1회에서 16건이 실패했으나 이후 네 번(새 DB 3회·재실행 1회) 모두 217/217 로 재현되지 않았고 실패 목록을 남기지 못했다 — 환경 쪽 흔들림으로 보지만 단정하지 않는다.
