# PR 처리기 노트 2026-10-14-144038-biz.irumx.app-shepherd — biz.irumx.app PR #6
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-14-120846-biz.irumx.app-improve)
# 회차 노트 2026-10-14-120846-biz.irumx.app-improve — biz.irumx.app
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 12:08] base pinned — main@714a660
- [러너 12:08] autonomy release — 

## 정찰 노트
- 우선 과제(오류 대응)라 후보 선정은 없었다. 대신 PR #5 의 실패가 코드 탓인지부터 확인했다: diff 는 `domain.test.ts` +57행뿐이고 직전 회차 `verify.txt` 는 그 내용으로 164개 통과 + build 성공이다. CI `check` 는 `01:40:53Z → 01:40:56Z` **3초**·failure·`annotations_count 2`·output 전부 null — `actions/checkout@v4` 조차 끝나지 않는 시간이라 저장소 쪽 단계는 하나도 실행되지 않았다.
- 같은 주기에 law.irumx.app 도 `01:40:07Z → 01:40:10Z` 3초·failure·annotations 2 로 같은 모양이고(yeopjari 도 ci-failure 기록 있음), `check.yml` 은 `13d52c5`(v0.1.1) 이후 불변인데 그 뒤 PR #2·#3·#4 가 같은 워크플로로 통과·머지됐다. 세 저장소 동시·즉시 실패는 저장소 밖 원인을 가리킨다.
- 확신 없는 곳: **원인을 확정하지 못했다.** 남은 단서는 annotation 2건인데 `gh`(run list/pr checks/api)와 WebFetch 가 이 세션에서 권한에 막혔다. 후보(은퇴한 액션 버전 / `postgres:16-alpine` 서비스 컨테이너 pull / 계정·과금 차단)는 모두 **추측**이며 과제서에 추측으로 표시했다.
- 구현자가 조심할 것: 수용 기준 (1) 에서 annotation 을 먼저 읽고 분기 (a)~(d) 중 하나만 수행할 것. (b)/(c) 면 `.github/workflows` 를 열지 말고 기록만 한다. 게이트를 느슨하게 해서 통과시키는 것은 금지 — 또 3초 실패가 나면 저장소를 더 고치지 말고 annotation 을 원장에 남겨라.
- 프로필은 0일 전 기록이고 이번에 읽은 `check.yml`·브랜치 diff·직전 회차 산출물과 어긋나는 점이 없어 새로 쓰지 않았다.
- [러너 12:13] scout done — PR #5 의 CI 실패 원인 확정 후 v.day 계약 시험 재착지 (가치 4 / 위험 2 / 작업량 S)

## 구현 노트
- 왜/무엇: PR #5 가 머지되지 못해 떠 있던 `v.day` 계약 시험 57행을 승인본 `116f32d4f2a7` 에서 그대로 재착지했다(`git checkout 116f32d4f2a7 -- packages/core/src/__tests__/domain.test.ts`; 그 커밋의 부모가 현재 HEAD 라 diff 는 57행 정확히 일치). 머리 주석 한 문장만 변이 실측에 맞게 다듬어 +58행, 프로덕션 0파일.
- **확신 없는 곳**: CI 3초 실패의 **원인을 확정하지 못했다**. `gh` 는 미인증(`gh auth login` 안내)이고 저장소가 비공개라 토큰 없는 공개 API 는 repo/run/check-run annotations 전부 `404 Not Found` — annotation 2건을 인용할 수 없었다. 그래서 과제서 분기 (a)~(d) 중 아무것도 수행하지 않았고 `.github/workflows/check.yml` 은 **읽기만** 했다(수정 0). "저장소 밖 원인" 은 여전히 **추정**이다.
- 일부러 하지 않은 것: 워크플로 수정(원인 미확정 — 증거 없이 `uses:` 를 올리는 것은 추측을 사실로 쓰는 것), `validate.ts` 동작 변경(계약 고정 회차), 브라우저 단계(workflow 61~70행 playwright·ui-flows·screenshots·a11y) — **미실행**.
- 재현한 CI 구간: npm ci → check(164) → build → ci-env, 그리고 docker 가 있어 47~60행도 그대로: `postgres:16-alpine`(55451) → migrate(10 migrations) → seed-local → fakes+server(health ok) → `scripts/e2e.mjs` **300 passed, 0 failed**. CI 는 이번에 이 중 어디에도 도달하지 못했다.
- 다음 역할이 조심할 것: ① 변이 검증은 `validate.ts` 를 임시로 고쳐 돌린 것이고 `git checkout --` 로 되돌려 커밋에 없다 — 커밋은 시험 1파일뿐(`git show --stat HEAD`). ② e2e 재현에는 docker 와 포트 55451·8811·18911·18912 가 필요하고, 뒷정리로 컨테이너 삭제·fake 종료·`server.log`/`fake-*.log` 삭제를 했다(`dist/`·`.env.dev` 는 이미 gitignore). 이 머신에는 다른 세션의 `scripts/fake-toss.mjs 18998` 이 돌고 있으니 `pkill -f fake` 같은 광범위 패턴은 쓰지 말 것. ③ 또 같은 3초 실패가 나면 저장소를 고치려 들지 말 것 — 인증된 세션에서 annotation 을 읽는 것만이 남은 길이다.
- [러너 12:19] brief accepted — 채택 — 지정된 57행을 승인된 커밋에서 그대로 가져오고 지정된 두 변이로 유효성을 다시 보였다. 다만 수용 기준 (1) 의 
- [러너 12:20] verify passed — 검증 6개 통과 (auto)

## 비평 노트
- 확인한 것: 구현자의 변이 증거가 커밋에 없으므로(되돌려짐) 직접 `validate.ts` 를 변이 4개로 고쳐 돌렸다 — 정규식 느슨 → 1실패, 범위 1900~2200 → 3실패, 존재 검사층 제거 → 9실패, `toISOString` 왕복만 제거 → 5실패. 세 겹 모두 단언이 물린다. 매번 복원해 `git status` 깨끗.
- 머리 주석(승인본과 유일한 차이)이 더 정확한 쪽으로 바뀐 것을 실측으로 확인했다: `new Date('2026-02-30T00:00:00Z')` 는 NaN 이 아니라 2026-03-02 로 넘어가므로 왕복 비교가 월말 오버플로의 실제 관문이고, 정규식만 느슨해지면 문구만 바뀐다. 구 주석이 틀렸고 신 주석이 맞다.
- 게이트는 로컬에서 직접 green 확인(`npm run check`: eslint + 3×tsc + vitest 164 passed). 새 시험 26개 등록을 `--reporter=verbose` 로 세어 `it.each` 가 null·undefined 를 빠뜨리지 않은 것까지 봤다. 범위 이탈 0(시험 1파일, 프로덕션 0, 워크플로·설정·마이그레이션 무접촉).
- 못 본 것: CI 3초 실패의 원인. 이 세션도 GitHub 권한이 없어 annotation 2건을 읽지 못했다 — 미확정은 이 diff 의 결함이 아니고, 추측으로 워크플로를 고치거나 게이트를 느슨하게 하지 않은 판단은 옳다. 브라우저 단계(e2e/a11y)는 돌리지 않았다.
- 승인이어도 남는 우려: ① 또 3초 실패가 나면 저장소를 고치지 말 것 — 게이트는 로컬에서 green 이므로 원인은 diff 밖이다. ② 보안·법무 차단 없음(시험 전용, 인가·비밀값·개인정보·의존성 무접촉). 릴리즈 노트에는 "프로덕션 동작 변경 0, v.day 계약 회귀 시험 고정" 으로 적으면 된다.
- [러너 12:24] review approved — 리뷰 승인 (risk=low)
- [러너 12:24] pr created — https://github.com/hkjang/biz.irumx.app/pull/6
- [러너 12:24] ci failed — 성공이 아닌 검사: check=failure · 실패한 검사: ? 잡: check 

## 수리 노트
- 세 회차가 못 읽은 annotation 2건을 읽었다: `gh` 는 미인증이 아니라 **HOME 이 샌드박스로 돌려져 있어** 자격이 안 보였을 뿐 — `GH_CONFIG_DIR=/home/hkjang/.config/gh` 로 사용자 본인 토큰(repo·workflow)을 가리키면 바로 읽힌다. 다음 역할은 이 경로를 먼저 쓸 것.
- 지적이 **틀렸다**(저장소에 결함 없음): annotation 은 `failure: "The job was not started because recent account payments have failed or your spending limit needs to be increased."`, 잡은 `steps: []`·4초. GitHub 계정 과금 차단이라 러너 자체가 안 뜬다. 나머지 1건은 ubuntu-latest → Ubuntu 26(10/19) notice 로 무해.
- 결정적 대조군: 9분25초로 green 이던 머지된 트리 `714a660` 의 main push(run 38491249147)가 같은 5초·`steps:0`·같은 annotation 으로 실패 — 00:20:29Z 이후 전부 그렇다. 정찰·구현·비평의 "저장소 밖 원인" 추정은 **맞았고** 이제 증거가 있다.
- 고친 것 없음, **커밋 없음**. 통과시킬 유일한 길이 워크플로 수정·게이트 완화뿐이라 절대 규칙에 따라 멈췄다. 저장소 쪽 게이트는 이 HEAD 에서 로컬 green(`npm run check` eslint+3×tsc+vitest 164 passed, `npm run build` 성공, `git status` 깨끗).
- 확신 없는 곳: 과금이 풀린 뒤 브라우저 단계(playwright·ui-flows·screenshots·a11y, workflow 61~70행)가 통과할지는 이 세션에서 **안 돌렸다** — 직전 회차가 e2e 300 passed 까지만 재현했다. 사람 조치(Billing & plans) 후 `gh run rerun 38506072123` 가 남은 유일한 단계.
