# PR 처리기 노트 2026-10-15-165749-review.yeopjari.bid-shepherd — review.yeopjari.bid PR #9
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-15-161731-review.yeopjari.bid-improve)
# 회차 노트 2026-10-15-161731-review.yeopjari.bid-improve — review.yeopjari.bid
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 16:17] base pinned — main@6ea982c
- [러너 16:17] autonomy release — 

## 정찰 노트
- PR #8 은 `scripts/perf.mjs` +10/-1 한 파일뿐이고 diff 가 실패 기준(`let failed = total > BUDGET_KB`)과 기본 예산 230 을 건드리지 않는다 — repo 안에 CI 단계를 떨어뜨릴 변경이 없다. `gh` 는 이 세션에서도 승인 거부되어 CI 주석을 읽지 못했다(세션 비대화형). 그래서 0단계를 "로컬 전체 게이트로 배제 후 되돌리기"로 두고, 본 과제는 따로 골랐다 — 같은 구조가 2026-10-15 1508 회차에서 채택됐다.
- 본 과제로 `validate.ts` 단위 테스트를 고른 이유: 230줄짜리 입력 관문이 테스트 0건이고(ls 로 `__tests__` 5파일 확인), 라우트·도메인 14개 파일이 import 한다(grep 확인). 프로덕션 0줄·새 파일이라 열린 PR 다섯 개와 겹침이 0이다. 제친 후보 — 번들 축소(M·`perf.mjs`가 PR #8 과 겹칠 소지), e2e 멱등성(PR #4 와 파일 겹침), `mailboxKey` 조사(보호 구역 위험 4).
- 미확인(손계산): `v.int('0x10')`→16, `v.string(max:4)` 에 `'🙂🙂🙂'`→거부/`v.text` 는 통과, `'.a@b.com'` 통과, `v.instant('2026')` 통과, nil UUID 통과. 전부 실행해 확정하고 다르면 코드가 정답.
- 이 샌드박스에서 `npm`/`npx`/`gh`/셸 연산자가 승인 거부됐다 — 기준선 118 tests / 6 files 는 앞 두 회차 관측 인용이다. 구현자가 재확인할 것.
- 조심할 것: 테스트를 쓰다 발견될 세 자리(UTF-16 계수·느슨한 instant·선행 점 이메일)를 이번 회차에 고치지 말 것(사용자 노출 한도·수락 범위가 바뀐다). `__tests__/text.test.ts` 는 PR #7 과 충돌하므로 건드리지 말 것. 프로필은 지금 코드와 일치해 새로 쓰지 않았다.
- [러너 16:22] scout done — PR #8 의 CI 실패를 저장소 안에서 끝까지 배제한 뒤, 모든 라우트의 입력 관문인 `util/validate.ts` 의 `v.*` 검증

## 구현 노트
- 0단계: PR #8(`49334c4`, `scripts/perf.mjs` +10/-1)을 main 위에 얹어 `check.yml` 9단계를 로컬에서 전부 통과 — `118 passed (118)` / `223.7 KB … budget 230 KB` exit 0 / `applied 10 migrations` / e2e `218 passed, 0 failed` / a11y `심각한 접근성 위반 없음`. repo 쪽 결백이 증명됐으므로 `perf.mjs` 를 되돌렸고 중복 PR 을 만들지 않았다. 워크플로 0줄 변경(`git diff .github/` 빈 것 확인).
- 본 과제: `packages/core/src/__tests__/validate.test.ts` 신규(641줄 / 170 tests), 프로덕션 0줄. 전부 `parseOne`/`parse` → `invalid()` 의 `AppError` 경로로 돌려 필드 메시지 **전문**을 단정했다. 손계산 기대값 5건은 임시 probe 테스트로 실제 출력을 먼저 받아 확정했다(모두 과제서와 일치).
- 확신 없는 곳·검증 못 한 것: (1) `node scripts/e2e.mjs` 는 0단계에서 돌린 것이 마지막이다 — 그 뒤 추가한 것이 테스트 파일뿐이라 재실행하지 않았고, 이어서 `seed-demo.mjs` 를 돌려 DB 가 더러워져 지금 재실행하면 깨끗한 DB 전제가 깨진다(돌리려면 `bash scripts/dev-reset.sh` 먼저). (2) `fail` 의 중복 방지는 `v.*` 조합만으로는 한 path 에 두 번 실패시킬 수 없어 `Validator` 계약으로 두 검증기를 같은 path 에 얹는 합성 검증기로 재현했다 — 프로덕션에 그런 합성 호출자가 실제로 있는지는 확인하지 않았다(변이 ③ 은 이 테스트로 잡힌다).
- 일부러 하지 않은 것: `v.string` 의 UTF-16 계수, `v.instant` 의 느슨한 파싱, `v.email` 의 선·후행 점 — 셋 다 실행으로 확정했지만 사용자 노출 한도·수락 범위가 바뀌는 동작 변경이라 **현재 동작을 테스트로 고정하고 파일 머리말에 주석으로만** 남겼다(ideas.json 에 pending). `text.test.ts`(PR #7)·`e2e.mjs`(PR #4)·`perf.mjs`(PR #8)는 손대지 않았다.
- 다음 역할이 조심할 것: 이 테스트는 순수 함수라 DB 가 필요 없고 `npm run check` 만으로 돈다(7 files / 288 tests, exit 0 확인). 변이 주입 3건은 모두 되돌렸고 `git diff -- packages/core/src/util/validate.ts` 가 0줄인 것을 각각 확인했다. 커밋은 `f1f4c84` 하나, 새 파일 1개뿐이다.
- [러너 16:33] brief accepted — 채택 — 0단계 기대 수치(118 tests / 6 files, 223.7KB / 230KB, 218/0, 10 migrations)와 손계산 기대값 다섯 가지(`'0x10'`→16, `'🙂🙂🙂'
- [러너 16:33] verify passed — 검증 6개 통과 (auto)

## 비평 노트
- 확인한 것: diff 가 신규 테스트 1파일(641줄)뿐이고 프로덕션·워크플로 0줄 — `npm run check` 재실행으로 288/288 exit 0, 단일 파일 170 passed 를 직접 받았다. 테스트가 대상을 정말 지나는지는 구현자의 변이 3건과 다른 변이 6건을 내가 주입해 확인했고(5건 잡힘: `v.int` 빈문자열·`v.oneOf` 6/7 경계·`v.uuid` 소문자화·`v.text` 제어문자 strip·`parse` 의 undefined 키), 매번 복원해 `git status` 깨끗함을 봤다. 메시지 전문은 `koreanParticle` 규칙으로 손검산까지 맞췄다.
- 못 본 것: e2e·a11y·ui-flows 를 이 커밋에서 재실행하지 않았다(프로덕션 0줄이라 불필요하다고 판단). 0단계의 PR #8 통과 수치는 구현자 기록을 인용했고 재현하지 않았다.
- 승인이어도 남는 우려: ① `v.array` 의 `if (parsed !== undefined) out.push` 변이가 살아남는다 — validate.test.ts:433 주석이 단언(`toThrow`)보다 많이 말한다(등가 변이, 결함 아님). ② 이 테스트가 UTF-16 계수·느슨한 `v.instant`·선행 점 이메일을 **현재 동작으로 고정**하므로, 나중에 좁히는 회차는 해당 표를 같이 고쳐야 한다(릴리즈 노트에 넣을 것). ③ `v.string({min:0})`·`v.text` 의 `min>1` 분기는 아직 비어 있다.
- 다음 회차: `util/validate.ts` 를 건드리면 `bash scripts/dev-reset.sh` → `node scripts/e2e.mjs`(218/0) 를 먼저 다시 받을 것. 보안·법무 차단 사유 없음(신규 수집·권한·비밀값 없음, 테스트 데이터는 합성 값).
- [러너 16:36] review approved — 리뷰 승인 (risk=low)
- [러너 16:37] pr created — https://github.com/hkjang/review.yeopjari.bid/pull/9
- [러너 16:37] ci failed — 성공이 아닌 검사: check=failure · 실패한 검사: ? 잡: check 

## 수리 노트
- 지적이 틀렸다. CI 'check' 실패는 코드가 아니라 **GitHub Actions 과금 거절**이다 — 잡 `116305460362` 의 `steps` 가 빈 배열(3초, 단계 0개 실행)이고 런 주석이 "The job was not started because recent account payments have failed or your spending limit needs to be increased" 라고 말한다. 러너의 "실패한 검사: ?" 도 이 때문(읽을 로그가 없다).
- 그래서 고치지 않았고 커밋도 없다(작업 트리 깨끗, HEAD 그대로 `f1f4c84`). 워크플로를 고쳐 통과시키는 것은 절대 규칙 위반이고, 과금 문제는 그래도 안 풀린다.
- 대신 이 브랜치로 CI 9단계를 로컬에서 전부 받아 저장소의 결백을 증명했다: check 288/288 exit 0 · build exit 0 · perf 223.7/230 KB · migrate 10개 · (dev-reset 후) e2e **218 passed / 0 failed** · a11y 심각 위반 0건. 비평가가 못 봤다고 한 e2e·a11y 가 이번에 이 커밋에서 실제로 돌았다.
- 여전히 확신 없는 곳: 비평 노트의 우려 ①~③(등가 변이 1건, 현재 동작 고정 3건, 빈 분기 2건)은 이번 수리 범위가 아니어서 그대로 남아 있다. 내 로컬 a11y 는 CI 와 달리 캐시된 chromium(`PLAYWRIGHT_BROWSERS_PATH`)을 썼다.
- 다음 역할: 저장소 쪽에 할 일이 없다. 과금 해소 후 CI 재실행(`gh run rerun 38699953366`)만 하면 된다.
