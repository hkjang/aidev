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

## 비평 노트
- 판정 approve / risk low / blocking 없음. 직접 돌린 것: `npm ci` → `npm test` 28 passed, `npm run build` 녹색, dist 에 테스트 문자열 0건. 변이 3개(클램프 제거 / `now <= e` / dense rank)를 내 손으로 넣어 테스트 4개가 해당 자리에서 빨개지는 것까지 확인했다 — 테스트는 실제로 바뀐 경로를 지난다.
- 구현 노트가 의심한 두 자리를 다 시험했다. ①Node 20 WebSocket 오류는 이 환경(node 22)에서 재현되지 않았다 — 근거는 확인 못 했지만 결과 설계는 과제서 위치보다 낫고 호출부를 보존했으므로 통과. ②advisory 는 사실이나 "패치된 버전이 범위 안에 없다"는 틀렸다: vitest@3.2.7 을 main 의 vite@5.4 에 얹어 설치해 critical GHSA-5xrq-8626-4rwp 가 사라지는 것을 확인했다. dev 전용·`vitest run` 은 서버를 안 띄우므로 차단은 아니고 다음 회차 한 줄 작업으로 넘긴다.
- 못 본 것: vitest 3.x 로 실제 `npm test` 를 돌려보지 않았고(설치·audit 만 확인), supabase/migrations 의 SQL 은 이번 diff 에 없어 열지 않았다.
- 승인이어도 남는 우려 두 개. (1) **`npm test` 를 돌리는 CI 가 없다** — pages.yml 은 build 만, PR 워크플로 없음. 28개 테스트는 지금 회귀를 자동으로 막지 못한다. pages.yml 이 보호 경로라 피한 판단은 타당하나 가장 값싼 후속 작업이다. (2) `phase.ts:20` 주석이 "none·alive 는 생존자 기준" 이라 말하지만 `choices` 는 `useRoom.ts:41-48` 에서 eliminated 필터 없이 읽으므로 o/x 는 탈락자 답까지 센다 — 기존 동작을 옮긴 것이고 테스트가 솔직히 고정해 두었지만, 프로필의 탈락 모드 분모 불일치와 같은 뿌리이고 이번에 고쳐지지 않았다.
- 릴리즈 노트: 사용자 눈에 보이는 동작 변화 0건(프로덕션 로직 무변경, 테스트 하네스 추가뿐). revert 로 완전히 되돌아온다 — 외부 상태·마이그레이션 변경 없음.
- [러너 11:39] review approved — 리뷰 승인 (risk=low)
- [러너 11:39] pr created — https://github.com/hkjang/ox-arena/pull/1
- [러너 11:43] ci no-ci — 이 커밋에 검사가 없음 (정책 allow_merge_without_ci 가 없으면 차단)
