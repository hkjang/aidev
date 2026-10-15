# PR 처리기 노트 2026-10-15-144524-review.yeopjari.bid-shepherd — review.yeopjari.bid PR #7
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-15-141852-review.yeopjari.bid-improve)
# 회차 노트 2026-10-15-141852-review.yeopjari.bid-improve — review.yeopjari.bid
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 14:18] base pinned — main@6ea982c
- [러너 14:18] autonomy release — 

## 정찰 노트
- 골랐다: util/text.ts 순수 함수 단위 테스트(7회차 연속 보류). 열린 PR 셋(#4 quota.ts / #5 pipeline.ts+text.ts / #6 workflows+README+ci-local.sh)의 파일 목록을 `git diff --name-only main..<branch>` 로 뽑아 보니 쓸 만한 후보 대부분이 그 파일들과 겹쳐 금지였고, `text.test.ts` 만 어느 PR 도 건드리지 않는다 — 지금 겹침 0으로 끝낼 수 있는 몇 안 되는 과제다.
- 제친 것: perf 예산 경고(차선으로 남김, CI 단계라 실패 기준을 건드릴 위험), clientAddress IPv6 정규화·purgeOldCounters(각각 PR #5·#4 와 같은 파일), josa.ts 통합(운영자 금지 + 영향 0으로 기각).
- 확신 없는 곳: 과제서의 기대값 전부가 **손계산**이다(이 샌드박스에서 npm 실행 거부). 특히 maskEmail 의 `'ho**@ex*****.com'`·`'a*@ex*****.com'`·`'ho**@ab*.com'` 과 excerpt 의 120 경계는 실행해서 확정할 것. 다르면 관측값이 맞다.
- 신규 관측(이번엔 안 고름): `mailboxKey` 가 저장소 전역 grep 에서 참조 0건이다 — 주석이 선언한 '한 메일함 한 번 인증' 이 이 함수로는 지켜지지 않는다. 다른 곳(유일 인덱스·sso.ts)에서 지켜질 수 있으니 단정 말 것. ideas.json 에 가치 4 / 위험 4 로 올려 뒀다.
- 구현자가 조심할 것: `clientIpFrom` 테스트 금지(PR #5 가 삭제), `util/text.ts` 수정 금지(PR #5 와 충돌), 새 테스트 파일 만들지 말고 기존 `text.test.ts` 에 덧붙일 것.
- [러너 14:23] scout done — `util/text.ts` 의 나머지 순수 함수에 단위 테스트를 입히기 — `koreanParticle`·`maskEmail`·`tidyBody`·`excerpt`·`mailb

## 구현 노트
- `text.test.ts` 에 koreanParticle·maskEmail·tidyBody·excerpt·mailboxKey·emailDomain·normalizeEmail·charCount 의 엣지케이스를 덧붙였다(기존 표 관례 그대로). 프로덕션 코드 0줄, 변경 파일 1개. 118 tests / 6 files → **187 tests / 6 files**, `npm run check` 통과.
- 동작 변경이 없어 "실패 재현" 대신 변이 주입으로 실효를 증명했다: `%28 === 0`→`!== 0` 이면 9 failed, `maskEmail` 의 `Math.min(2,…)`→`3` 이면 8 failed. 둘 다 되돌렸고 `git diff -- packages/core/src/util/text.ts` 가 비어 있음을 확인했다.
- 확신 없는 곳: 전부 실행으로 확정했으므로 기대값에 미확인은 없다. 다만 **이 테스트들은 현재 동작을 그대로 기록한 characterization 테스트**다 — 특히 `maskEmail('a@b@example.com')`→`'a@*@ex*****.com'` 과 `mailboxKey('+tag@company.com')`→`'+tag@company.com'`(폴백) 두 건은 "맞는 동작" 이라고 단정하지 않았고 주석으로 의도를 적어 뒀다. 의도를 바꾸려면 별 회차다.
- 일부러 하지 않은 것: `clientIpFrom` 테스트(PR #5 가 그 함수를 삭제 — 쓰면 머지 시 빨간 테스트), `apps/web/src/lib/josa.ts` 프런트 사본과의 통합(운영자 지시: 같은 종류의 파서는 통합하지 말 것), `randomAlias`/`suggestNickname`/`randomDigits`(난수라 값 단정 불가, 가치 낮음), `util/text.ts` 수정 일체(PR #5 와 충돌).
- 다음 역할이 조심할 것: DB·브라우저 불필요 — `npm run check` 하나로 전부 돈다(이 워크트리는 `npm ci` 가 필요했다). `node scripts/e2e.mjs` 는 이번에 돌리지 않았다 — 프로덕션 코드를 한 줄도 건드리지 않았으므로 회귀 경로가 없다고 판단했다. 열린 PR #4·#5·#6 과 파일 겹침 0 이다.
- [러너 14:28] brief accepted — 채택 — 근거가 코드와 그대로 맞았고(`text.test.ts` 108줄이 `isSafeLink` 뿐, 열린 PR 셋과 파일 겹침 0, 기준선 118 tests / 6 files �
- [러너 14:28] verify passed — 검증 6개 통과 (auto)

## 비평 노트
- 확인했다: diff 는 `text.test.ts` 한 파일(237줄 추가, 프로덕션 0줄), 이 워크트리에서 `npm run check` 통과(187 passed / 6 files), 표의 기대값 전수 손계산 재확인(maskEmail 별 개수·excerpt 120/121·tidyBody 3/4 newline 경계 포함), 열린 브랜치 4개와 파일 겹침 0, PR #5 의 `util/text.ts` 변경은 `clientIpFrom` 삭제뿐이라 머지 순서 무관.
- 구현자 주장을 그대로 믿지 않고 **다른 변이 9종**을 직접 주입했다 — 7종이 잡혔다(tidyBody·excerpt·mailboxKey 폴백·koreanParticle 숫자표·charCount·normalizeEmail·maskEmail 각 1~3 failed). 매번 되돌리고 최종 `git diff` 가 빈 것을 확인했다.
- **살아남은 변이 2건**(결함 아니라 커버리지 구멍, 다음 회차 후보): `text.ts:34` `at < 0`→`at <= 0` (emailDomain 표에 `@company.com` 류 없음), `text.ts:49` `dot > 0`→`dot >= 0` (`hong@.com` 류 없음).
- 못 본 것: `e2e.mjs`·`ui-flows.mjs`·`a11y.mjs` 미실행(프로덕션 코드 0줄이라 회귀 경로 없음에 동의). 릴리즈 노트는 "테스트만 추가, 동작 변경 없음"으로 적을 것.
- 승인이어도 남는 우려: `mailboxKey` 는 프로덕션 호출 0건이고 실제 유일성은 `0001_foundation.sql:61` 의 `ux_accounts_email_norm(email_norm)` 기준 — `text.ts:15-22` 가 약속한 '한 메일함 한 번 인증'(태그 주소로 계정 쌓기 방지)은 집행되지 않는다. 이번 변경이 만든 문제는 아니지만 테스트가 붙어 '커버됨'으로 보이게 된다. 보안·법무 차단 없음(합성 주소만, 새 수집·전송 없음).
- [러너 14:32] review approved — 리뷰 승인 (risk=low)
- [러너 14:32] pr created — https://github.com/hkjang/review.yeopjari.bid/pull/7
- [러너 14:33] ci failed — 성공이 아닌 검사: check=failure · 실패한 검사: ? 잡: check 

## 수리 노트
- 지적이 **틀렸다** — 정확히는 코드 지적이 아니었다. CI 'check' 는 잡이 시작조차 못 해 실패했다: `"steps":[]`·3초·러너 미배정에 런 주석이 *"The job was not started because recent account payments have failed or your spending limit needs to be increased"*. `--log-failed` 가 빈 출력이던 것도 이 때문이다(과제서 로그 칸이 비어 있던 이유).
- 브랜치 무관 확인: 10-14 01:14 이후 런 5개(브랜치 5개, PR #4·#5·#6·#7)가 전부 같은 billing 주석으로 실패, 마지막 성공은 그 직전 main. 다른 런 2개에서 주석을 직접 확인했다.
- 고친 것 없음 / 커밋 없음. 워크플로를 손대 통과시키는 것은 금지이고 저장소 안에 결함이 없다 — GitHub 계정 Billing & plans 조치만이 해법이며 사람 몫이다.
- 로컬에서 같은 검증을 돌려 브랜치는 초록임을 확정했다: `npm ci` → `npm run check` **187 passed / 6 files**, lint·typecheck 무경고, `npm run build` 성공, `scripts/perf.mjs` 223.7 KB / 230 KB 통과.
- 확신 없는 곳: `e2e.mjs`·`a11y.mjs` 는 안 돌렸다 — 공유 `yeopjari-review-dev` 컨테이너가 떠 있어 `dev-reset.sh` 의 DROP DATABASE 가 동시 회차를 깰 위험이 있었다. 프로덕션 0줄이라 회귀 경로는 없다고 보지만 CI 가 살아나면 거기서 처음 돈다.
