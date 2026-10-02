# 회차 노트 2026-10-02-110726-ox-arena-improve — ox-arena
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 11:07] base pinned — main@f68a6fa
- [러너 11:07] autonomy release — 

## 정찰 노트
- 테스트 0개·test 스크립트 없음을 확인하고 Vitest 하네스를 1순위로 골랐다. 더 "진짜 버그"인 탈락 모드 정답률 분모 오류(0003_elimination.sql:114 v_total ↔ Display.tsx:50 rv.total ↔ useRoom counts.alive, 세 경로가 분모가 다름)는 마이그레이션=보호 경로이고 이 저장소에 SQL 검증 수단이 전혀 없어 차선으로 미뤘다.
- 추측으로 적은 것: vitest 버전(vite 5 호환은 ^2 계열로 적었으나 **미확인**), vitest 무설정 기본값으로 src/lib/*.test.ts 가 잡힌다는 것(문서 기억 기반, 실행 미확인), src/lib/supabase.ts 의 env fallback(파일 미열람).
- `npm ci`/`npm i` 가 이 샌드박스에서 승인 거부로 막혀 **build 도 test 도 한 번도 실행해 보지 못했다.** 구현자는 손대기 전에 `npm i && npm run build` 로 기준선 녹색부터 확인할 것.
- 조심할 것: rankPlayers 는 정렬에 correct 를 쓰고 랭크는 score 만 본다 — README 의 "동점 공동순위" 와 맞는 현재 동작이니 테스트로 **고정**하고 바꾸지 말 것. counts 추출은 순수 이동만, useRoom 의 폴링/채널은 건드리지 말 것.
- [러너 11:12] scout done — Vitest 도입 + 순수 게임 로직(rankPlayers / derivePhase / 집계) 회귀 테스트 (가치 4 / 위험 2 / 작업량 M)

## 구현 노트
- 테스트 0개였던 저장소에 vitest@2.1.9 + `npm test`(`vitest run`, 설정 파일 없음) 를 깔고 순수 로직 28개 테스트를 추가했다(commit 214f3e7). `derivePhase`/`counts` 집계는 **새 `src/lib/phase.ts`** 로 verbatim 이동하고 `useRoom.ts` 가 재수출 — 호출부 4곳(Timer/Display/Admin/Play)은 무변경. 프로덕션 파일 2개.
- **과제서와 다른 점(의도적)**: 과제서는 `useRoom.ts` 안에서 `computeCounts` 를 export 하라고 했으나, 그러면 테스트가 `useRoom.ts` → `supabase.ts` 의 모듈 로드 부작용(`createClient` 가 RealtimeClient 생성)을 끌어와 `Error: Node.js detected but native WebSocket not found.` 로 터진다(이 샌드박스의 npm 은 Node 20 을 집는다). 자기 모듈을 `vi.mock` 하거나 WebSocket 셰임을 넣는 대신 부작용 없는 모듈로 분리했다 — 이게 근본 원인 수정이다.
- **확신 없는 곳**: ①이 샌드박스 npm 이 Node 20(v20.19.2)을 쓰는 이유를 파악하지 못했다(`node -v` 는 22.23.1). CI 는 Node 22 라 어느 쪽이든 phase.ts 분리로 영향받지 않지만, 비평가 환경에서 Node 버전 경고가 보일 수 있다. ②vitest 가 dev 의존성에 advisory 3건을 추가한다(critical 1 = GHSA-5xrq-8626-4rwp, **Vitest UI 서버를 띄웠을 때만** 해당 — `vitest run` 은 서버를 띄우지 않는다). 패치된 버전이 범위 안에 없어 그대로 뒀다. 프로덕션 의존성(`npm audit --omit=dev`)은 변화 없음(기존 react-router moderate 2건).
- **일부러 하지 않은 것**: `joinUrl`(location 의존 → jsdom 필요), `useRoom` 훅 자체(DB 필요), `vitest.config.ts`/`vite.config.ts`(무설정으로 돌아서 불필요), `pages.yml`·`supabase/migrations/**`(보호 경로). 차선 후보(탈락 모드 분모 불일치)는 SQL 검증 수단이 없어 손대지 않았다 — ideas.json 에 유지.
- **다음 역할이 조심할 것**: 세 테스트 파일 모두 DB·네트워크·브라우저 없이 돈다 — `npm i && npm test` 면 끝. 테스트가 바뀐 경로를 지나는지는 변이 테스트로 확인했다(`Math.max(0,…)` 제거 / 공동순위 건너뛰기 제거 / `now < e`→`now <= e` 세 변이에 테스트 5개가 각각 빨개짐, 되돌리니 녹색). `rankPlayers` 테스트는 **현재 동작을 고정한 것**이지 바꾼 것이 아니다(랭크 기준은 점수만, correct 는 정렬에만). 테스트 파일이 `src` 안이지만 번들에는 섞이지 않음을 확인했다(dist js 에 테스트 문자열 0건, 90 modules / 434.17 kB).
- [러너 11:34] brief accepted — 채택 — 1순위를 그대로 구현했고, 수용 기준 5개를 모두 만족시켰다(추출 위치만 모듈 로드 부작용 때문에 `useRoom.ts` 내�
- [러너 11:34] verify passed — 검증 4개 통과 (auto)
