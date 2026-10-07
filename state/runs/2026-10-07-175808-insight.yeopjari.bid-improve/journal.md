# 회차 노트 2026-10-07-175808-insight.yeopjari.bid-improve — insight.yeopjari.bid
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 17:58] base pinned — main@7a0955e
- [러너 17:58] autonomy release — 

## 정찰 노트
- `quota.ts:retryAfterSeconds` 의 +9시간 오차를 골랐다: 원인이 한 함수 안에 있고(프로덕션 파일 1개 + 테스트 1개), `retry-after` 헤더로 사용자까지 실제로 도달하며(`respond.ts:68` → `apps/web/src/api/client.ts:181`), `day` 제한이 auth.code·study.apply·report·mail 캡 등 전역에 쓰여 영향 범위가 넓다. 테스트 공백 보강(util/time.ts)·죽은 코드 정리(isSafeLink)는 가치가 더 낮아 미뤘다.
- 추측으로 적은 것: e2e 151개 중 Retry-After 값을 단정하는 것이 있는지 확인하지 않았다. 또 `windowStart()` 라벨이 9시간 밀려 있어도 **창의 길이·전환 시점은 올바르다**는 판단은 코드 독해로만 확인했다(DB 행을 보지 못했다).
- 샌드박스가 node/npm 실행을 거부해 `npm run test` 를 한 번도 돌리지 못했다 — 숫자 검산은 손으로 했다(`2026-10-07T16:00Z` → 현재 32h, 올바름 23h). 구현자는 먼저 현재 코드로 그 값을 재현해 전제를 확인하고 시작할 것.
- 구현자가 조심할 것: `windowStart()` 를 "고치고 싶어지는" 함수다. 절대 건드리지 말 것 — `release()`·`usage()`·DB 유일 키가 모두 그 값에 걸려 있고, 바꾸면 운영 중 모든 제한이 한 번 리셋된다. `util/time.ts` 도 읽기만.
- [러너 18:03] scout done — 일·월 단위 쓰기 제한의 Retry-After 가 KST 경계를 9시간 넘겨 잡히는 것 고치기 (가치 4 / 위험 2 / 작업량 S)

## 구현 노트
- `quota.ts:retryAfterSeconds` 가 다음 창 전환을 `windowStart()`(KST 날짜의 **UTC** 자정 라벨)로 계산해 day/month 제한에 항상 +9h 를 더 얹던 것을 고쳤다. 전환 시각을 `kstDayStart(at)+days(1)` / `Date.UTC(kstY, kstM, 1) - 9h` 로 직접 계산한다. 프로덕션 파일 1개, 테스트 1개.
- **`windowStart()` 본문은 한 줄도 바꾸지 않았다.** `export` 만 붙였고(테스트가 ISO 라벨 4개를 못 박는다), 그 이유를 주석으로 남겼다 — 값이 `insight_priv.quota_counters` 유일 키라 바꾸면 모든 제한이 한 번 리셋된다.
- **확신 없는 곳**: e2e 151개와 screenshots 는 **돌리지 못했다**(실 Postgres·Chromium 필요). grep 으로 `retry-after` 값을 단정하는 e2e·클라이언트 코드가 없음은 확인했지만(`apps/web/src/api/client.ts:181` 은 숫자를 그대로 읽어 표시만 한다), CI 의 e2e 결과는 봐야 한다. 또 테스트는 실제 `consume()` 배선을 지나지 않는다 — DB 없이 429 를 만들 길이 없어 순수 함수를 직접 부른다(과제서가 지정한 방식).
- **일부러 하지 않은 것**: `usage()` 의 "one round trip" 주석(측정 없이 건드릴 수 없고 범위 밖), `util/text.ts:isSafeLink` 정리(차선 후보로 남김), `util/time.ts` 는 읽기만 했다.
- **다음 역할이 조심할 것**: `npm install` 이 `package-lock.json` 에 `peer: true` 26줄을 추가한다 — 커밋에서 제외했다. `npm ci` 를 쓰거나 커밋 전 `git status` 로 확인할 것.
- 검증: `npm run check` → lint 통과, typecheck 3개 workspace 통과, `vitest run` **44 passed** (이전 40). `npm run build` 통과. 커밋 `098cc43`.
- [러너 18:07] brief accepted — 채택 — 과제서의 근거·수용 기준·손으로 한 검산이 지금 코드와 전부 일치했고 수정 범위도 프로덕션 파일 1개로 끝났�
- [러너 18:07] verify passed — 검증 6개 통과 (auto)

## 비평 노트
- 원장에 `- 실패 재현:` 줄이 없어 직접 재현했다: main 의 `quota.ts` 로 되돌려 새 테스트를 돌리니 day/month 가 정확히 +32400s(9h)로 실패(115200↔82800, 2620800↔2588400) — 증상이 커밋 메시지와 일치한다. 고친 뒤 `npm run check` 44 passed, 트리 깨끗.
- `windowStart` 는 `export` 만 붙고 본문 무변경(diff 확인), `index.ts` 는 명시 배럴이라 패키지 표면도 안 넓어진다. 경계값(한국 자정 정각·짧은 달·연 경계·ms 꼬리)은 손으로 검산해 전부 맞다.
- 못 본 것: e2e 151개·screenshots(실 Postgres·Chromium 필요). 다만 `scripts/e2e.mjs` 에 429·retry 단정이 0개이고 web 변경이 없어 위험은 낮다 — CI 결과만 확인하면 된다.
- 승인이어도 남는 우려 둘: (1) `window: 'month'` 를 쓰는 제한이 지금 하나도 없어 month 분기는 아직 미사용이다(실효는 day 뿐). (2) 그래서 더 중요한데 `quota.ts:157` `purgeOldCounters` 의 `now() - interval '3 days'` 는 월 라벨(1일)을 달 중간에 지운다 — **월 단위 제한을 도입하기 전에 고칠 것**.
- 릴리즈 노트 표현: `retryAfter` 는 `apps/web/src/api/client.ts:181` 에서 파싱만 되고 화면에 렌더되지 않는다. "사용자에게 보이는 시각" 이 아니라 "HTTP `retry-after` 헤더 값 정정" 으로 적어야 정확하다.
- [러너 18:10] review approved — 리뷰 승인 (risk=low)
- [러너 18:11] pr created — https://github.com/hkjang/insight.yeopjari.bid/pull/1
- [러너 18:15] ci passed — 검사 1개 모두 success
- [러너 18:15] merge done — 098cc43
- [러너 18:26] release published — v0.1.1
- [러너 18:26] assets n/a — 이전 릴리즈에도 자산 없음
