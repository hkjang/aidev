# 회차 노트 2026-10-01-074222-vibe-code-improve — vibe-code
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 07:42] base pinned — main@43fd7a1
- [러너 07:42] autonomy release — 

## 정찰 노트
- 고른 이유: 보류 1순위였던 goal-metrics.ts:51 exitCode 단독 수정은 **한쪽만 고치기**가 되어 운영자 금지 항목에 걸린다. 같은 값을 읽는 세 번째 경로 journal-summary.ts:66 이 정반대로 읽는 것을 이번에 찾았고, command-audit.ts:20/core/host.ts:65 가 exitCode 누락이 실제 경로임을 증명해 값이 4로 올랐다. 프로덕션 2파일로 끝난다.
- 추측으로 적은 것: PR #13 ↔ `auto/2026-10-01-0632`(head `1dd2151`) 대응은 **추정**이다 — diff 가 직전 회차 산출물과 일치하는 것만 확인했다. 원격 CI 상태·annotation 은 이번에도 미확인(gh 미인증).
- 미확인: 이 정찰 세션은 `npx vitest run` 이 **권한 거부**돼 테스트 기준선(82)을 직접 재측정하지 못했다. 구현자는 반드시 실측할 것. docs/README/CHANGELOG 에 통과율 집계 규칙을 단언하는 줄이 있는지도 확인 못 했다.
- 조심할 것: 기존 픽스처는 두 테스트 파일 모두 exitCode 를 0/1/2/null 로 갖고 있어 기대값이 안 바뀌는 것을 정찰이 직접 열어 확인했다 — 기대값을 고치고 싶어지면 그건 구현이 틀린 것이다. `num()`(goal-metrics.ts:14)은 tokens/cost 가 쓰므로 지우지 말 것. `ci.yml`·`command-audit.ts`(쓰는 쪽)는 건드리지 말 것.
- [러너 07:47] scout done — 수정 과제 — PR #13 head `1dd2151` 로 CI check job 로컬 재현 판정 + 감사 `exitCode` **누락**을 세 읽기 경로가 서로 

## 구현 노트
- 무엇/왜: 감사 `exitCode` **누락**을 읽는 경로가 서로 반대였다 — `journal-summary.ts:66` 은 `Number(undefined)=NaN!==0` 이라 실패로 세고 `(exit undefined)` 를 찍는데 `goal-metrics.ts:51/52/69/110` 은 `num(undefined)=0` 이라 성공으로 셌다. 순수 헬퍼 `auditExitCode()`(누락·null·비수치 → `null`=미상, 미상은 성공 아님)를 `journal-summary.ts` 에 두고 다섯 지점이 전부 그 한 술어를 지나게 했다. 커밋 `bbee934`, 프로덕션 2파일.
- 확신 없는 곳/검증 못 한 것: **PR #13 ↔ `1dd2151` 대응은 정찰의 추정을 그대로 썼다** — 원격 CI 상태·annotation 은 이번에도 직접 못 읽었다(gh 미인증, push 불가, WebFetch 거부). 로컬 4단계는 전부 exit 0(`validation/pr13-node20-check.log`)이라 "코드 결함 아님" 이 여덟 번째 판정이지만, 배정의 "같은 이유로 두 번 실패" 는 **미확인**이다. CI `package` job(windows-latest)도 미확인 — 이 세션에 Windows 수단이 없다.
- 일부러 하지 않은 것: `command-audit.ts:20`(쓰는 쪽) — 조건부 쓰기는 계약이고 이미 쌓인 감사 파일이 그대로라 읽는 쪽을 맞추는 게 맞다. `ci.yml` 은 읽기만(완화 금지). `num()` 은 tokens/cost/archived 가 쓰므로 남겼다. 차선 후보 `readCommandHistory` 날짜 창은 다른 결함이라 같은 커밋에 얹지 않았다(shotgun 금지) — `ideas.json` 에 pending.
- 다음 역할이 조심할 것: (1) 워크트리에 `node_modules` 가 없어 `npm ci` 를 먼저 돌려야 `npx vitest run` 이 된다. 기준선은 main@43fd7a1 에서 **11 files/82 tests** 실측, 이 변경으로 88. (2) 기존 기대값은 한 줄도 안 바꿨다(`git diff --numstat tests/` = `40/1`·`10/0`, 유일한 삭제는 import 줄) — 기대값을 고치고 싶어지면 구현이 틀린 것이다. (3) 인과 확인 중 `git checkout HEAD -- .` 로 작업을 한 번 날려 동일 내용을 재적용했다 — 최종 트리는 검증된 상태이고 `npm run check` 를 재적용 후 다시 돌렸다(`validation/fix-node20-check.log`). (4) 같은 값을 읽는 **네 번째 표시 경로**가 `command-audit.ts:73` 에 남아 `! 종료 undefined` 를 찍는다 — 판정은 맞고 표시만 문제라 이번 범위 밖, `ideas.json` 에 신규 pending 으로 적었다.
- [러너 07:55] brief accepted — 채택 — 0단계 게이트를 지시대로 Node 20.19.2 에서 4단계 모두 재현해 "코드 결함 아님" 을 증거 로그로 판정하고, 멈추지 �
- [러너 07:55] verify passed — 검증 6개 통과 (auto)

## 비평 노트
- 인과를 직접 재현했다: 프로덕션 2파일만 main 으로 되돌리니 원장과 **같은 6개가 같은 메시지로** 빨개졌고, 복원 후 `npm run typecheck`/`npx vitest run`(11 files·88 tests)/`npm run build` 전부 exit 0. 기대값 삭제는 import 2줄뿐 — 테스트가 기존 계약을 깎아 통과한 자리는 없다.
- 못 본 것: 원격 CI·PR #13 대응·windows `package` job·Extension Host. 로컬 Node 는 v22.23.1 로 돌렸다(순수 함수라 20.19.2 와 결과가 갈릴 여지 없다고 판단).
- 승인이어도 남는 우려 ①: journal-summary.ts:44 JSDoc 의 "이 값을 읽는 **모든** 경로가 이 술어를 쓴다" 는 거짓이다 — `command-audit.ts:51/73` 이 네 번째 경로로 남아 `! 종료 undefined` 를 그대로 찍는다. 커밋 제목("세 읽기 경로")은 정확하니, 다음 회차는 주석을 좁히거나 네 번째 경로를 같이 접어라.
- 승인이어도 남는 우려 ②(릴리즈 노트에 적을 것): 실제로 수치가 바뀌는 곳은 `command`/`exited` 줄뿐이다(runVerification 은 verification.ts:119 가 항상 exitCode 를 쓰므로 하드닝). 셸 통합이 exitCode 를 못 주는 터미널에서는 `실패 종료`·회고 병목이 갑자기 커지는데, 이는 buildSessionSummary 가 이미 하던 판정에 goal-metrics 를 맞춘 결과다.
- 보안·법무 차단 없음. 다만 `.vibe-code/` 가 .gitignore 에 없어 일지에 남는 실패 명령 **원문**의 건수가 미상-종료만큼 늘어난다 — 새 공격 경로는 아니지만 자격증명이 섞인 명령이 커밋될 노출량은 커진다.
- [러너 07:59] review approved — 리뷰 승인 (risk=low)
- [러너 07:59] pr created — https://github.com/hkjang/vibe-code/pull/14
- [러너 07:59] ci failed — 성공이 아닌 검사: typecheck + unit tests + build=failure
