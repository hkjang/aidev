# 회차 노트 2026-10-14-141349-review.yeopjari.bid-improve — review.yeopjari.bid
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 14:13] base pinned — main@6ea982c
- [러너 14:13] autonomy release — 

## 정찰 노트
- 골랐다: PR #3(쿼터 Retry-After) 을 main 위에 다시 올리면서 새 e2e '하루 한도' 검사의 벽시계 의존을 없애기. 이유는 머지가 안 된 것이 사실이기 때문 — main@6ea982c 에 `quota.test.ts` 가 없다(`ls`, `git diff --stat main auto/2026-10-14-0958` 로 확인). 새 아이디어(text.ts 테스트 보강 등)는 우선 과제가 있어 제쳤다.
- 수리 회차가 두 번 "고칠 것 없음, 재시도" 로 끝나 두 번 verify-failed 했다. 같은 결론을 세 번째로 내지 않도록, 아무도 읽지 않은 단서를 과제서 첫 지시로 넣었다: `ci-fe4eaba9edd2.json` 의 `output.annotations_count: 2` → `gh api .../check-runs/115627512290/annotations`.
- 미확인(추측으로 적지 않으려 표시함): CI 3초 실패의 진짜 원인. 정찰 샌드박스에서 `gh` 실행이 승인 거부돼 주석을 못 읽었다. 주석이 워크플로/서비스 컨테이너 오류를 가리키면 그것이 과제이고, 지출 한도면 코드로 고칠 것이 없다 — 과제서에 분기를 적어 뒀다.
- 구현자가 조심할 것: `windowStart` 는 손대지 말 것(저장 키), e2e 의 ±5초 허용 오차를 넓혀 통과시키지 말 것(기준 시각만 `date` 헤더로 교체), `.github/workflows/check.yml` 을 느슨하게 만들지 말 것.
- 프로필은 0일 전 판이 지금 코드와 맞아 새로 쓰지 않았다.
- [러너 14:18] scout done — PR #3 의 쿼터 수정을 main 위에 다시 올리면서, 새 e2e '하루 한도' 검사의 벽시계 의존(한국 자정 경계에서 86

## 구현 노트
- 무엇/왜: PR #3(머지 안 됨)의 `quota.ts` 수정 — `retryAfterSeconds` 가 저장 키(`windowStart`, KST 날짜를 UTC 자정으로 라벨링)를 실제 창 경계로 믿어 429 의 `retry-after` 가 언제나 정확히 9시간 길었다. `nextWindowReset` 을 두고 그것만 쓰게 했다. 더해서 그 PR 에 남아 있던 e2e 결함(기준 시각을 `Date.now()` 로 다시 읽어 한국 자정 걸침에서 `expected` 가 86400초 튀는 것)을 응답 `date` 헤더로 교체해 없앴다. 프로덕션 파일 1개. 커밋 62b3471.
- 확신 없는 곳·검증 못 한 것: **CI 'check' 3초 실패의 원인은 여전히 미확정이다.** 과제서의 첫 지시(annotations 2건 읽기)를 실행하지 못했다 — 이 세션도 `gh` 미인증(`gh auth status`: "You are not logged into any GitHub hosts")이고 로컬 JSON 에는 주석 본문이 없다. 그러니 이 변경이 CI 실패를 고친다는 주장은 하지 않는다. 고친 것은 "CI 가 거짓 실패할 수 있는 시각 의존" 하나다.
- 일부러 하지 않은 것: `.github/workflows/check.yml` 무수정(느슨하게 만들기 금지), `windowStart` 무수정(저장 키), ±5초 허용 오차 무수정(기준 시각만 교체), `purgeOldCounters` 의 월 창 보존 함정은 범위 밖이라 ideas.json 으로 넘겼다.
- 다음 역할이 조심할 것: 단위 8건은 DB 없이 돌지만 `scripts/e2e.mjs` 는 PostgreSQL(`127.0.0.1:55443`, 컨테이너 `yeopjari-review-dev`) + `:8796` 서버 + `dev-reset.sh` 직후를 전제로 한다(220 passed / 0 failed, `TZ=UTC` 도 동일).
- 재현 방법 메모: 자정 걸침 재현은 libfaketime 으로 했다 — 서버 `FAKETIME="@2026-10-14 14:59:38"`, e2e 는 그보다 29초 앞(TOTP 허용 드리프트 ±30초 안이라야 관리자 MFA 가 통과한다. 1시간 어긋내면 `mfa confirm 401` 로 조기 중단된다). 그 조건에서 서버 시각만 DB `now()` 보다 9.4시간 앞서 무관한 검사 1건('입금 확인 → 오늘부터 3개월 적용')이 깨지는데, 고치기 전/후 양쪽에서 똑같이 깨지므로 이번 변경과 무관하다.
- [러너 14:29] brief accepted — 채택 — 근거가 코드와 그대로 맞았다(main@6ea982c 에 `quota.test.ts` 없음, `quota.ts` 옛 판). 첫 지시였던 CI 주석 읽기는 `gh` 미�
- [러너 14:29] verify passed — 검증 6개 통과 (auto)

## 비평 노트
- 확인함: 구현자가 의심한 자리(CI 3초 실패)는 코드 결함이 아니라 미확인 사실로 남았고, 이 변경이 그것을 고친다는 주장을 안 한 것이 정직하다. 수정 로직은 직접 추적해 맞았다 — `nextWindowReset` 의 월 분기(`m===12 → y+1, 0`)·일 분기(KST 는 DST 없어 +24h 가 정확)·분 분기 모두 경계에서 옳고, `windowStart` 저장 키를 건드리지 않아 카운터 키가 그대로다.
- 테스트가 변경을 정말 검증한다: main 의 옛 `retryAfterSeconds` 를 그대로 옮겨 새 기대값 10건을 돌려 봤다 — day/month 7건 전부 실패하고 편차가 예외 없이 +32400초, 원장의 `expected 82800 to be 50400`·`expected 32460 to be 60` 과 정확히 일치한다(minute 3건은 옛 코드에서도 통과 — 특성 테스트).
- 직접 실행: `npm run check` 126/126(기준선 6파일 118 + 신규 8 확인), 실 PostgreSQL + `dev-reset.sh` 뒤 `node scripts/e2e.mjs` **220 passed / 0 failed** — '하루 한도' 2건 PASS. 범위 이탈·인증/권한/마이그레이션/워크플로 변경 없고 revert 로 완전히 돌아온다. 보안·법무 차단 사유 없음(새 엔드포인트·개인정보·의존성·암호 코드 0).
- 못 본 것: CI check-run 115627512290 의 annotations 2건(이 세션도 GitHub 권한 없음), a11y·ui-flows·perf(이 변경은 UI 무관), 그리고 e2e 를 옛 `quota.ts` 로 되돌려 돌려보는 확인(단위 재현으로 충분하다 판단).
- 승인이어도 남는 우려 — 릴리즈가 알 것: ① CI 가 같은 이유로 또 실패할 수 있다(원인 미확정). ② `window: 'month'` 한도 호출자가 0건이라 월 분기는 오늘 테스트 전용 경로다. ③ `purgeOldCounters` 의 `interval '3 days'` 는 월 한도가 생기는 순간 매달 4일에 진행 중 카운터를 지운다(ideas ④ — 월 한도 추가 전 선행 조건으로 묶을 것). ④ 새 '하루 한도' 절이 minji 의 `product.request` 한도를 소진한 채 끝난다 — 뒤에 minji 로 제안을 올리는 검사를 추가하려면 먼저 `resetQuotas()`. ⑤ `apps/web/src/api/client.ts:181` 이 `retryAfter` 를 받아 두지만 쓰는 화면이 없다 — 헤더가 정확해졌으니 ideas ③ 의 선행 조건이 풀렸다.
- [러너 14:34] review approved — 리뷰 승인 (risk=low)
- [러너 14:34] pr created — https://github.com/hkjang/review.yeopjari.bid/pull/4
- [러너 14:35] ci failed — 성공이 아닌 검사: check=failure · 실패한 검사: ? 잡: check 
