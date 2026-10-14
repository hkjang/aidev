# 회차 노트 2026-10-14-095838-review.yeopjari.bid-improve — review.yeopjari.bid
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 09:58] base pinned — main@6ea982c
- [러너 09:58] autonomy release — 

## 정찰 노트
- 고른 이유: quota.ts 의 Retry-After 가 일 한도에서 최대 9시간 과장된다 — 프로덕션 파일 1개, 저장 키를 안 건드리면 위험 1, 단위 테스트로 증명 가능(지금 quota 테스트가 아예 없다). 차선이었던 text.ts 테스트 보강은 결함을 고치지 않아 뒤로 뒀고, e2e 멱등성(M)·라우트 통합 테스트(L)는 한 회차에 안 들어간다.
- 확신 없는 곳: 23h→14h 숫자는 손계산이다(샌드박스에서 `node -e` 승인 거부). 또 이 헤더를 지금 화면이 표시하지 않는다 — 효과는 API 응답 정확성까지이고 화면 변화는 없다. 과대 포장하지 말 것.
- 조심할 것: `windowStart` 반환값(= `quota_counters.window_start` 저장 키)을 고치면 배포 순간 진행 중인 일·월 카운터가 전부 리셋된다. 재시도 계산만 바꿀 것. `purgeOldCounters` 의 3일 간격은 월 창에 대해 조기 삭제지만 `window: 'month'` 호출자가 없어 이번 범위 밖(ideas.json 에 분리해 남겼다).
- 검증: `npm run check` 는 반드시, `node scripts/e2e.mjs` 는 `bash scripts/dev-reset.sh` 직후 218/0 기준선과 비교.
- [러너 10:02] scout done — 하루 한도 429 의 Retry-After 가 최대 9시간 과장되는 것 — 재시도 시점을 KST 실제 경계로 계산 (가치 3 / 위험

## 구현 노트
- 무엇·왜: `quota.ts` 의 `retryAfterSeconds` 가 저장 키 라벨(`windowStart`, KST 날짜의 UTC 자정)을 실제 창 경계로 믿어 일·월 한도 429 의 `retry-after` 가 **언제나 정확히 9시간** 길게 나갔다(시각 무관 상수 오차 — 정찰이 적은 "최대 9시간"보다 나쁘다). 실제 경계를 구하는 `nextWindowReset(window, at)` 를 새로 두고 `retryAfterSeconds` 가 그것만 쓰게 했다. 프로덕션 파일 1개.
- 저장 키는 안 건드렸다: `windowStart` 는 그대로이고(diff 로 확인) 주석만 "이건 저장 라벨이고 실제 경계가 아니다"로 보강했다. 배포 시점 진행 중인 카운터는 리셋되지 않는다.
- 확신 없는 곳: 없는 쪽에 가깝다 — 정찰의 손계산(50400 vs 82800)을 단위 테스트로 확정했고, e2e 에 실제 HTTP 429 헤더 검사를 넣어 고치기 전 코드로 빌드·재시작해 `retry-after=82351 expected≈49951` 로 재현한 뒤 고친 코드에서 통과시켰다. 다만 `minute` 분기는 식을 바꿨다(결과는 동등하게 유지되도록 경계 3점을 테스트로 고정) — 리뷰는 여기를 먼저 봐도 좋다.
- 일부러 안 한 것: `purgeOldCounters` 의 `interval '3 days'`(월 창 조기 삭제, 지금 month 호출자 0건), 화면에 재시도 시각 표시(별건 — ideas.json 에 선행 조건 해소로 기록), `windowStart` 통합. 범위·파일 수를 지키기 위함.
- 정찰 과제서 정정: "e2e 에 review.submit 일 한도 회귀가 있다"는 사실이 아니었다 — e2e 에 쿼터 한도 검사가 아예 없었다. 그래서 `product.request`(일 10회) 경로로 '하루 한도' 섹션을 추가했다(`resetQuotas()` 로 스스로 격리, 11번 POST).
- 다음 역할이 조심할 것: `node scripts/e2e.mjs` 는 PostgreSQL(127.0.0.1:55443, 컨테이너 `yeopjari-review-dev`)과 `:8796` 서버가 떠 있어야 돈다. 이 회차는 `npm ci` → `node scripts/ci-env.mjs` → `npm run build -w @review/server` → `bash scripts/dev-reset.sh` 로 세워 220 passed / 0 failed(기준선 218 + 신규 2). 새 '하루 한도' 검사는 벽시계를 쓰므로 ±5초 허용으로 되어 있다.
- 돌린 것: `npm run check` → eslint + 타입 3개 + 126 tests 통과. `npx vitest run packages/core/src/__tests__/quota.test.ts` → 8 passed(고치기 전 7 failed). `node scripts/e2e.mjs` → 220 passed / 0 failed. a11y·ui-flows 는 화면 변화가 없어 돌리지 않았다.
- [러너 10:09] brief accepted — 채택 — 근거·수용 기준·손계산 수치(50400/82800)가 모두 코드와 일치했고 범위도 과제서대로 프로덕션 파일 1개로 끝났다
- [러너 10:09] verify passed — 검증 6개 통과 (auto)

## 비평 노트
- 판정 approve(위험 low, 차단 없음). 전부 직접 돌렸다: `eslint` 통과, `npm run check` 126 tests, 새 테스트 단독 8 passed, `dev-reset.sh` 후 `node scripts/e2e.mjs` 220/0 — 원장 수치와 일치. 구현자가 지목한 `minute` 분기는 `ceil((60000-r)/1000) == 60-floor(r/1000)` 로 동등함을 대수로 확인했다.
- 실패 재현을 믿지 않고 다시 했다: 고치기 전 `retryAfterSeconds` 를 독립 스크립트로 재구현해 새 단언 10개를 돌려 day·month 7개가 전부 **정확히 32400초(9시간)** 오차로 실패함을 확인(원장의 82800/50400·32460/60 과 동일). 테스트는 바뀐 경로를 실제로 지난다.
- 승인이어도 남는 것 ① `quota.test.ts:11`·`e2e.mjs:1125` 주석이 '최대 9시간'이라 적었지만 실제는 시각 무관 '정확히 9시간'(커밋 본문은 맞다) — 주석만 정정 ② 새 e2e 검사는 벽시계 ±5초라 15:00Z(한국 자정)를 걸친 실행에서 플레이크 ③ `nextWindowReset` 은 모듈 밖 호출자 0건 — 아이디어 ② 를 집지 않으면 노출 표면만 남는다.
- 범위 밖 추적(구현자 ideas ③ 과 동일): `purgeOldCounters` 의 `interval '3 days'` 가 month 창 카운터를 매달 4일째 지운다. `window:'month'` 호출자 0건을 grep 으로 확인 — 월 한도를 처음 쓰는 사람이 조용히 당한다.
- 보안·법무 소견 모두 없음: 인가·세션·비밀값·저장 키(`windowStart`)·개인정보·의존성에 변화 없고 마이그레이션도 없어 revert 로 완전히 돌아온다. 못 본 것은 a11y·ui-flows·perf(화면 변화 없어 생략, 구현자와 동일 판단).
- [러너 10:14] review approved — 리뷰 승인 (risk=low)
- [러너 10:14] pr created — https://github.com/hkjang/review.yeopjari.bid/pull/3
- [러너 10:15] ci failed — 성공이 아닌 검사: check=failure · 실패한 검사: ? 잡: check 
