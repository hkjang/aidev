## 2026-10-02
- 선택: Vitest 도입 + 순수 게임 로직(rankPlayers / derivePhase / computeCounts) 회귀 테스트 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: 테스트가 0개였던 저장소에 `vitest@2.1.9` + `npm test` 를 깔고 순수 로직 28개 테스트를 추가했다. `derivePhase` 와 `counts` 집계를 `src/lib/phase.ts`(Supabase 를 import 하지 않는 순수 모듈)로 **그대로 옮기고** `useRoom.ts` 가 재수출해 기존 호출부 4곳(Timer/Display/Admin/Play)은 한 줄도 바뀌지 않았다. 과제서는 `useRoom.ts` 에서 추출만 하라고 했지만, 그러면 테스트가 `useRoom.ts` → `supabase.ts` 의 모듈 로드 부작용(`createClient` 가 RealtimeClient 생성)을 끌어와 터진다 — 자기 모듈을 mock 하거나 WebSocket 셰임을 넣는 대신 부작용 없는 모듈로 분리했다. 검증: `npm test`(28 passed) + `npm run build`(tsc --noEmit && vite build, 녹색) 둘 다 실제 실행. 추가로 변이 테스트로 테스트가 프로덕션 경로를 실제로 지나는지 확인했다 — `Math.max(0,…)` 제거 / 공동순위 건너뛰기 제거 / `now < e` → `now <= e` 세 변이에 각각 해당 테스트 5개가 빨개졌고, 되돌리니 다시 녹색이 됐다.
- 실패 재현: 추출 전 `npm test`:
  `FAIL src/lib/useRoom.test.ts — Error: Node.js detected but native WebSocket not found.` (← `computeCounts` 가 없는 게 아니라 `supabase.ts:7` 의 `createClient` 가 모듈 로드 때 터진 것. 이것이 phase.ts 분리의 직접 근거다.)
  변이 후 `npx vitest run` (테스트가 바뀐 경로를 지나는 증거):
  `AssertionError: expected [ 1, 2, 3 ] to deeply equal [ 1, 1, 3 ]` / `AssertionError: expected -2 to be +0` / `AssertionError: expected 'choosing' to be 'locked'` — 5 failed | 23 passed
- 보류 아이디어: ①탈락 모드 정답률·미선택 분모 불일치(0003_elimination.sql:114 `v_total` 은 전체, Display.tsx:50 은 그걸로 정답률 계산, Field 는 생존자 기준 `counts.alive` → 분모 3중 불일치. 보호 경로 + SQL 검증 수단 없음) ②별도 `ci.yml` 로 PR 에 `npm test` + `npm run build` 돌리기(이제 test 스크립트가 있으므로 성립. pages.yml 은 절대 건드리지 말 것) ③`reveal.eliminated` 가 이번 라운드가 아닌 누적 탈락자 수(0003:148) ④`useRoom` 4초 전역 폴링을 단계별로 조정(finished/waiting 에서 간격 늘리기) ⑤`react-router-dom@6` 의 moderate 2건(open redirect / deserializeErrors) — 수정이 v7 메이저라 단독 회차 필요
- 과제서: 채택 — 1순위를 그대로 구현했고, 수용 기준 5개를 모두 만족시켰다(추출 위치만 모듈 로드 부작용 때문에 `useRoom.ts` 내부 → 새 `phase.ts` 로 바꿨고 재수출로 런타임 동작은 동일하다).

