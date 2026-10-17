# PR 처리기 노트 2026-10-17-233656-insight.yeopjari.bid-shepherd — insight.yeopjari.bid PR #5
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-17-230856-insight.yeopjari.bid-improve)
# 회차 노트 2026-10-17-230856-insight.yeopjari.bid-improve — insight.yeopjari.bid
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 23:09] base pinned — main@601659d
- [러너 23:09] autonomy release — 

## 정찰 노트
- 자동 배정된 수정 과제를 그대로 실었다. 새 아이디어를 제친 이유는 PR #4 가 막혀 있는 한 다음 회차도 같은 자리에서 멈추기 때문이다.
- 확정된 증거: check_run 117331978771 이 3초(06:36:37→06:36:40Z) 만에 failure, annotations 2건, 단계 정보 없음. 같은 커밋의 로컬 check(7파일/113개)·build 는 이전 회차 verify.txt 에서 통과.
- 추측으로 적은 것: "실패가 단계 스크립트 밖(잡 기동)" 은 3초라는 소요 시간에서 끌어낸 추론이다. 주석 2건의 원문은 **못 받았다** — 이 환경에서 gh 호출이 승인되지 않았다.
- 구현자가 조심할 것: 주석 원문을 먼저 받고 분기할 것. 원인을 못 본 채 테스트를 손보면 같은 실패가 반복된다. 워크플로를 느슨하게 만드는 수정은 금지다.
- ci-env.mjs·dev-reset.sh 를 기존 개발 환경에 돌리지 말 것(.env.dev 덮어쓰기·DB 초기화). e2e 는 폐기용 Postgres 16 + 분리 포트, 역할은 irumx_gateway→insight_app.
- [러너 23:13] scout done — PR #4 의 check 워크플로가 3초 만에 실패하는 원인을 확정해 고치기 (수정 과제, 가치 4 / 위험 2 / 작업량 M)

## 구현 노트
- 수정 과제는 분기 B 로 끝냈다(코드 미수정, check.yml 한 줄도 안 건드림). 과제서의 "결정적 증거" 를 반증했다: `annotations_count: 2` 는 이 워크플로의 **성공 회차 5건에도 모두 2** 여서 실패 신호가 아니다(2 = 쓰는 액션 수). `.github/` 는 마지막 녹색 CI(739ea33)부터 동일하고, 같은 날 같은 계정 다른 저장소 잡들은 정상 실행돼 과금·러너 전면 장애도 아니다.
- 그 다음 과제서가 허용한 차선 후보를 했다: 새 파일 `apps/web/src/__tests__/client-guards.test.ts`(12개)로 CSRF 쿠키 선택·세션 힌트·상태별 문구·retry-after·401 알림을 실제 http 서버로 고정했고, 그 과정에서 찾은 **204 쓰기의 캐시 무효화 누락**을 먼저 실패시킨 뒤 client.ts 5줄로 고쳤다(프로덕션 파일 1개).
- **확신 없는 곳·검증 못 한 것:** ① 3초 CI 실패의 실제 이유는 **모른다** — 비공개 저장소라 자격증명 없이 로그·주석을 읽을 길이 없었다(익명 API 404). 추정으로 단정하지 않았다. ② 204 결함은 지금 **잠복**이다: `respond.ts:noContent` 호출처 0, `api.cached/prefetch` 화면 소비 0 — 즉 오늘의 사용자 피해는 증명하지 못했고 고친 것은 cache.ts 가 문서로 약속한 규칙이다. ③ 계약 가드 12개는 결함 수정이 아니라 기존 동작 고정이라 red 단계가 없다 — 변이 검증(쿠키 선택·isAdminSurface·403 문구를 틀리게 바꿔 3개 실패)으로 대신했다.
- **일부러 하지 않은 것:** `node scripts/e2e.mjs`·`scripts/screenshots.mjs` 미실행 — 가짜 OAuth 포트 :18999 가 고정 공유라 동시에 도는 다른 회차의 서비스와 충돌할 수 있다(끄지 말라는 지시). `ci-env.mjs`·`dev-reset.sh` 도 돌리지 않았다. `download()` 의 signal·x-insight-stale 보완은 PR #4 와 같은 줄이라 미뤘다.
- **다음 역할이 조심할 것:** 이 테스트는 DB 없이 돌지만 포트 0 으로 로컬 http 서버를 띄운다(샌드박스에서 루프백 바인딩 필요). `window`/`document` 는 node 전역에 세우고 `afterEach` 에서 지운다. PR #4 와는 파일이 달라 충돌하지 않지만 둘 다 `client.ts` 를 건드린다(이쪽은 174행 근처, 저쪽은 234행 근처).
- [러너 23:25] brief fallback — 차선 — 과제서가 결정적 증거로 지목한 주석 2건은 성공 회차에도 똑같이 2건이어서 실패 신호가 아니고(새로 반증함), �
- [러너 23:25] verify passed — 검증 6개 통과 (auto)

## 비평 노트
- 확인한 것: 범위는 cb269c5 한 건(client.ts 5줄 + 새 테스트 209줄). 204 테스트의 red 는 실재 — invalidateAll() 이 캐시를 비우는 유일한 경로이고 수정 전 204 분기는 그 앞에서 return 했다. 이 워크트리에서 `npm run check` 재실행 전부 통과(eslint, core/server/web typecheck, vitest 7파일/121개). 나머지 11개는 결함 수정이 아닌 회귀 가드임을 확인했고 쿠키 정확 일치·403 문구는 변이에 민감하다.
- 못 본 것: e2e·screenshots 미실행(실 Postgres·:18999 공유 포트), 브라우저에서의 실제 배너 확인 없음, PR #4 의 check_run annotations 원문(자격증명 없음).
- 승인이어도 남는 우려 ①: 이 수정은 잠복 결함 정비다 — noContent 호출처 0, api.cached/prefetch 소비처 0, 204 생산지는 OPTIONS 프리플라이트와 핸들러가 undefined 를 돌려줄 때의 toResponse 뿐. 릴리즈 노트에 사용자 체감 결함 수정으로 적지 말 것.
- 승인이어도 남는 우려 ②: 204 분기가 x-insight-stale→reportDegraded() 앞에서 return 하므로 204+stale 쓰기는 열화 배너를 못 띄운다(지금 그런 경로는 없음). 보류된 download() 정렬 작업이 같은 자리다.
- 다음 회차가 알아야 할 것: 이 브랜치는 601659d 위에 쌓여 있어 PR #4 를 막는 원인이 이 PR 도 막는다. 3초 CI 실패는 여전히 미확정이니 annotations 원문 확보를 최우선으로. 보안·법무 차단 사유는 없다(인가·개인정보·의존성·비밀값 변화 없음).
- [러너 23:29] review approved — 리뷰 승인 (risk=low)
- [러너 23:29] pr created — https://github.com/hkjang/insight.yeopjari.bid/pull/5
- [러너 23:30] ci failed — 성공이 아닌 검사: check=failure · 실패한 검사: ? 잡: check 

## 수리 노트
- 맞았던 지적: 없다. CI 가 빨간 것은 사실이지만 그 원인이 이 diff(client.ts 5줄 + 새 테스트 209줄)라는 전제가 틀렸다. 커밋 없이 끝냈다.
- 확정한 반증: 이 워크트리에서 `npm ci`(231 packages, 12s) → `npm run check`(eslint·typecheck 3개·vitest 7파일/121개) → `npm run build`(web+server) 전부 통과. 3초 실패는 `npm ci` 가 끝나기도 전이라 어떤 검증 단계도 밟지 않았다.
- 저장소 내용 배제: 마지막 녹색 CI(739ea33) 이후 `.github/` diff 가 빈 결과이고, package.json 4개 변경은 0.1.4→0.1.6 버전 줄뿐(의존성 무변동), 락파일 동기도 `npm ci` 성공으로 확인했다. PR #4 도 겹치지 않는 diff 로 같은 3초 failure — 원인은 변경 밖이다.
- 여전히 확신 없는 곳: 실제 원인. 남는 후보는 `postgres:16-alpine` 서비스 컨테이너 풀 실패(Docker Hub 레이트리밋)·저장소 Actions 한도·러너 기동 실패이며 **증명하지 못했다**. 어느 쪽이든 고치려면 check.yml 을 건드려야 해서 절대 규칙상 손대지 않았다.
- 다음 역할이 할 일: 코드가 아니라 GH_TOKEN 확보다. 자격증명 없이는(`gh auth` 미설정, 익명 API 404) 잡 117420253640 의 로그·주석을 아무도 읽을 수 없고, 그 상태로는 이 PR 뿐 아니라 이후 모든 회차가 같은 자리에서 멈춘다.
