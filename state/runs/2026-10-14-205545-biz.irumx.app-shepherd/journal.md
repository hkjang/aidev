# PR 처리기 노트 2026-10-14-205545-biz.irumx.app-shepherd — biz.irumx.app PR #7
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-14-203508-biz.irumx.app-improve)
# 회차 노트 2026-10-14-203508-biz.irumx.app-improve — biz.irumx.app
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 20:35] base pinned — main@714a660
- [러너 20:35] autonomy release — 

## 구현 노트
- 배정 오류는 러너 쪽 사건이었다(직전 stages.json: scout done 16:42 → improve error 16:45, 저장소에 실패 단계 없음). 정찰 1순위였던 `services:`→명시적 단계 변경은 **기각**: `.github/workflows/check.yml` 은 13d52c5(v0.1.1) 이후 무변경인데 그 사이 PR #2·#3·#4 가 머지되고 v0.1.2~0.1.4 가 나갔다 — 같은 파일로 통과하던 CI 가 #5·#6 에서 3초·0단계로 죽으므로 파일은 무죄다. 워크플로는 한 줄도 안 건드렸다. 대신 보류 1순위였던 쿼터 창 계약을 새 파일 `packages/core/src/__tests__/time.test.ts`(15개)로 고정했다. 프로덕션 0파일.
- **확신 없는 곳**: ① PR #2~#4 의 CI 가 실제로 green 이었는지는 gh 미인증(+GH_TOKEN 0건, 비공개 404)으로 직접 확인하지 못했다 — 머지와 릴리즈가 진행된 사실에서 추론했다. 이 추론이 틀리면 "파일 무죄" 결론도 약해진다. ② 3초·annotation 2건의 원문은 세 회차 연속 못 읽었다. 원인은 여전히 미확정이며 계정·쿼터 가능성은 추정이다. ③ CI 의 브라우저 구간(check.yml 61~70행: playwright install·seed-demo·ui-flows·screenshots·a11y)은 **실행하지 않았다** — chromium 내려받기와 apt 의존성이 필요하다. 그 아래 저장소 게이트는 전부 로컬에서 돌렸다(check 153 passed, build, ci-env, migrate 10, seed, e2e 300 passed/0 failed).
- 일부러 안 한 것: 새로 찾은 실제 결함 — quota.ts `retryAfterSeconds` 의 day·month 분기가 실제 초기화 시각(UTC 15:00)보다 **정확히 9시간 긴** Retry-After 를 낸다 — 를 고치지 않았다. 비공개 함수이고 `consume()` 이 Sql 을 요구해 DB 없는 vitest 로는 프로덕션 배선을 지나는 증거를 만들 수 없고(손 대역은 운영자 금지), day/month 429 를 실제 노출하는 경로가 거의 없다(agent.ts:97·client-errors.ts:49 가 retryAfter 를 버린다). ideas.json 에 분석을 그대로 남겼다. 미사용 표면(kstDayStart·kstWindowOn·overlaps·isoDay·formatKst·backoffMs)도 늘리지 않았다.
- 다음 역할이 조심할 것: 새 시험은 DB·브라우저 없이 `npx vitest run packages/core/src/__tests__/time.test.ts` 로 돈다(15 passed). `domain.test.ts` 를 건드리지 않았으므로 미머지 PR #6(같은 파일 끝에 +58행)과 텍스트 충돌이 없다. 뒷정리 확인: docker 컨테이너 biz-ci-pg 삭제, fake·서버 프로세스 0개, `.env.dev`·`dist/` 는 .gitignore 에 있어 커밋에 안 들어갔다(커밋 3a84b16 = 시험 1파일 +63행).
- [러너 20:45] verify passed — 검증 6개 통과 (auto)

## 비평 노트
- 승인(approve, risk low, blocking 없음). diff 는 `packages/core/src/__tests__/time.test.ts` 한 파일 +63행뿐이고, 15개 단언이 모두 실제 `util/time.js` 의 kstDateKey·minuteStart 를 호출해 구체 문자열을 비교한다 — 14:59:59.999Z/15:00:00.000Z 피벗이 오프셋에 민감해 KST_OFFSET_MINUTES·padStart 가 틀리면 깨진다. 윤년(2028-02-28→02-29)·평년(2026-02-28→03-01)·월·해 넘김 산술을 손으로 재검산해 전부 맞았다. 로컬에서 `npm run check` 153 passed(eslint+tsc×3+vitest), 신규 파일 단독 15 passed.
- 구현자가 보류한 quota.ts:33 retryAfterSeconds 결함은 스크래치 스크립트로 재현했다 — day 창은 항상 **정확히 9시간(32,400초)** 과대(at=2026-03-15T14:59Z: 실제 60초, 응답 32,460초). 다만 '노출 경로가 거의 없다'는 판단은 틀렸다: http/respond.ts:68-69 이 retryAfter 를 가진 모든 AppError 에 헤더를 붙이고, day 상한은 mcp.ts:64·auth.ts:45·auth.ts:241·helpers.ts:135 에서 평범한 HTTP 로 도달한다. main 에 이미 있던 결함이라 이번 거절 사유는 아니며 신규 시험이 수정을 막지도 않는다 → **다음 회차 1순위**.
- 남는 우려: ① 월 창 시험(time.test.ts:38-43)은 quota.ts:29 의 표현식을 손으로 베낀 사본이라 그쪽이 바뀌어도 통과한다(windowStart 비공개 탓, 불가피). ② PR #5·#6 의 3초 CI 실패 원문은 이 세션도 못 읽었다 — `gh auth status` 미인증. 이 PR 의 green 은 로컬 검증뿐이다.
- 릴리즈가 볼 것: VERSION 은 0.1.4 그대로다(직전 시험 전용 커밋 9b864f8 은 v0.1.3 으로 올려 나갔음 — 올릴지 판단 필요). 브라우저·e2e 구간은 이 비평에서 돌리지 않았다.
- [러너 20:48] review approved — 리뷰 승인 (risk=low)
- [러너 20:48] pr created — https://github.com/hkjang/biz.irumx.app/pull/7
- [러너 20:49] ci failed — 성공이 아닌 검사: check=failure · 실패한 검사: ? 잡: check 

## 수리 노트
- **커밋 없음** — 지적(CI 'check' 실패)은 이 변경의 결함이 아니다. 3초·**0단계** 실패는 `actions/checkout@v4` 보다 앞(러너 배정·컨테이너 초기화)에서 죽은 것이고, 그 시점에 저장소는 체크아웃조차 안 됐으므로 시험 1파일(+63행)이 원인일 수 없다. 워크플로도 무죄 재확인: `check.yml` YAML 파싱 정상(잡 1개), 13d52c5 이후 무변경, 이 브랜치는 `.github/` 무변경(`git log origin/main..HEAD -- .github/` 공백).
- **맞았던 지적**: 없음(CI 가 실제로 red 인 것은 사실이지만 코드 결함은 아니다). **틀렸던 것**: "이 PR 때문"이라는 함의. 비평 노트의 '남는 우려 ②'(로컬 green 뿐)는 이번에 상당히 메웠다.
- **이번에 새로 돌린 것** — 앞선 세 회차가 건너뛴 브라우저 구간(check.yml 61~70행)을 실제로 통과시켰다: `playwright install chromium`, `seed-demo`, `ui-flows` 16 passed/0 problem, `screenshots` 152장(콘솔오류·5xx·오버플로 0), `a11y` 심각한 위반 0. 그 아래도 전부: check 153 passed, build, ci-env, migrate 10 + seed, e2e 300 passed/0 failed. 즉 check.yml 의 **모든 단계**가 로컬에서 통과한다.
- 타임존 가설도 배제했다(CI 와 로컬의 유일한 현실적 차이): 신규 시험이 `TZ=UTC`·`America/New_York`·`Asia/Seoul`·`Pacific/Kiritimati` 각각 15 passed, 전체 스위트도 `TZ=UTC` 153 passed. 뒷정리: 컨테이너 `biz-fix-pg` 삭제, fake·서버 프로세스 0개, `screenshots/`·`*.log` 제거, 작업나무 clean, HEAD 는 3a84b16 그대로.
- **여전히 확신 없는 곳**: 실패 로그·annotation 원문은 네 회차 연속 못 읽었다(`gh` 미인증, `GH_TOKEN`·`GITHUB_*` 0건, 비공개라 비인증 API 404, HOME 에 자격증명 없음). 계정 수준 사유(Actions 분·지출 한도, Actions 비활성화)가 #5·#6·#7 의 동일한 3초·0단계 패턴을 가장 잘 설명하지만 **추정**이다 — 확정에는 운영자가 Actions 사용량 화면이나 그 잡의 annotation 을 봐 줘야 한다. 수리 에이전트가 손댈 수 있는 범위(저장소 코드) 안에는 고칠 것이 없다.
