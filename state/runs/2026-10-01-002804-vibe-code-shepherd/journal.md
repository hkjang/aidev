# PR 처리기 노트 2026-10-01-002804-vibe-code-shepherd — vibe-code PR #9
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-09-30-221722-vibe-code-improve)
# 회차 노트 2026-09-30-221722-vibe-code-improve — vibe-code
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 22:17] base pinned — main@43fd7a1
- [러너 22:17] autonomy release — 

## 정찰 노트
- PR #8 은 main 위 커밋 하나(`9b50afc`)뿐이고 원격 상태는 이번 회차에 **전혀 확인 못 했다**(`gh auth status` 실행 권한 거부, `WebFetch` 권한 없음). 그래서 0단계 게이트(Node 20.19.2 로 ci.yml check job 재현)를 먼저 두고, 통과하면 멈추지 말고 1단계로 넘어가게 썼다 — 지난 회차가 "코드 결함 아님" 만 남기고 no-change 로 끝난 전례를 피하기 위해서다.
- 1단계로 `advancePlanText` 의 `- [x]` 항목 오선택을 고른 이유: 제품 계약이 **같은 파일 안 `openPlanItems`(plans.ts:236)에 이미 있어** 판단이 필요 없고, 프로덕션 1파일로 끝나 PR #7(markdown.ts)·PR #8(markdown.ts+goal-lint.ts)과 충돌하지 않는다. 차선(들여쓰기 trim)과 `sectionLines`/`countChecks` 계열은 제품 계약 판단이나 기존 기대값 재협상이 선행이라 제쳤다. ci.yml 의 `continue-on-error`/`found` skip 문제는 실재하지만(이번에 직접 읽어 확인) 보호 경로 + Windows 필요로 제외했다.
- 추측으로 적은 것: main@43fd7a1 의 테스트 기준선 82 tests(지난 회차 실측 인용, 이번엔 `npm ci` 를 돌리지 않았다), `npm install --no-save node@20.19.2` 절차(지난 회차 기록 인용), EBADENGINE 이 경고에 그친다는 판단(`.npmrc` 부재는 확인, 실제 실행은 미확인).
- 구현자가 조심할 것: `src/util/markdown.ts` 의 `isDoneTask`/`isTaskLine` 은 main 에서 열 0 고정이므로 재사용하면 들여쓴 `  - [x]` 를 놓친다 — 판정을 `plans.ts` 안에 trim 계약으로 두어라. `splice`+`writeSection` 구조(v1.4.4 줄 보존)를 되돌리지 말고, 입력은 `planTemplate`+프로덕션 `toggleCheckbox` 로만 만들어라.
- [러너 22:23] scout done — 수정 과제 — PR #8(`origin/auto/2026-09-30-2132`, head `9b50afc`) CI 실패를 Node 20 check job 으로 재현 판정하고, 코드 결

## 구현 노트
- 0단계 게이트: PR #8 head(9b50afc)를 저장소 밖에 `git archive` 로 추출해 ci.yml check job 을 Node 20.19.2 와 20.20.0 양쪽에서 돌렸다 — 4단계 전부 exit 0(89 tests), EBADENGINE 은 `npm warn` 뿐. 원격 실패는 코드 결함이 아니다. 로그: validation/pr8-node20-check.log, validation/pr8-node2020-check.log.
- 1단계 변경: `src/features/plans.ts` 에 `isOpenChecklistLine` 을 더해 `advancePlanText` 의 두 `findIndex(isChecklistLine)` 를 바꿨다. 이미 `- [x]` 인 Now/Next 첫 항목을 집어 열린 항목을 건너뛰고 감사에 거짓 `completed` 를 남기던 것을 고친다. 계약 근거는 같은 파일 `openPlanItems`.
- 확신 없는 곳 / 검증 못 한 것: (1) **PR #8 의 원격 CI 상태는 확인하지 못했다** — gh 미인증, 이 세션에 GitHub 조회 권한 없음. "결제/사용 한도" 는 2026-09-29 회차의 보관 annotation 을 인용한 추정이고 이번 회차의 확인이 아니다. (2) VS Code Extension Host 에서 실제 `advanceCurrentPlan` 명령을 눌러 본 검증은 없다 — 순수 함수 경계까지만 테스트했다(vscode-stub 는 import 경계용). (3) `npm run build` 는 dist 런타임 자산 5종 누락 경고를 내고 성공한다 — 실사용 런타임은 릴리즈 VSIX 복원이 필요하고 이번에 하지 않았다. (4) Windows `npm run vsix`/`npm run verify` 미실행.
- 일부러 하지 않은 것: `.github/workflows/ci.yml` 완화(금지), `src/util/markdown.ts`·`src/features/goal-lint.ts` 수정(PR #7/#8 이 둘 다 open 이라 충돌 회피 — 그래서 열 0 고정인 `isDoneTask`/`isTaskLine` 을 재사용하지 않고 `plans.ts` 안에 trim 계약으로 판정을 뒀다), 감사에 새 필드 추가, 들여쓰기 소실(차선 후보) 수정 — 같은 함수라 같은 회차에 섞지 않았다. push·CI 재시도·결제 설정 변경도 하지 않았다.
- 다음 역할이 조심할 것: 신규 테스트 "체크된 첫 줄 다음이 들여쓴 열린 항목이면 그것을 완료한다" 는 **현재의 들여쓰기 소실 동작**(`completed === "들여쓴 하위"`)을 기대값으로 고정한다 — 차선 후보(들여쓰기 보존)를 구현하면 이 기대값을 먼저 재협상해야 한다. 테스트는 DB·네트워크 없이 돌지만 `tests/unit/plans.test.ts` 의 archive 계열은 실제 임시 fs 를 쓴다(TMPDIR 이 상위 Git 저장소 안이면 오작동).
- 재현 명령: `PATH=<node20.19.2>/bin:$PATH npx vitest run tests/unit/plans.test.ts` (22 tests) → `npm run check` (88 tests). Node 20.19.2 는 `npm install --no-save node@20.19.2` 로 확보했고, **설치 디렉터리에 자체 package.json 을 먼저 두어야 한다** — 없으면 npm 이 상위로 올라가 홈 디렉터리의 node_modules 에 설치한다(이번에 한 번 그렇게 됐다).
- [러너 22:31] brief accepted — 채택 — 0단계 게이트를 지시대로 Node 20.19.2(+ 20.20.0 추가)에서 재현해 "코드 결함 아님" 을 증거 로그로 판정하고, 멈추지 
- [러너 22:31] verify passed — 검증 6개 통과 (auto)

## 비평 노트
- 확인한 것: main 의 plans.ts 를 저장소 밖 사본에 되돌려 신규 6개 테스트가 전부 **맞는 증상으로** 실패함을 직접 봤고(completed 가 이미 체크된 첫 항목), HEAD 에서 `npm run check` 초록(88 tests, Node 22.23.1). `- [x]` 를 열린 항목으로 세지 않는 계약이 `openPlanItems`(plans.ts:244)·`goal-lint.ts:60`·`plan-codelens.ts:31` 에 이미 있어 이번 변경이 그 계약에 맞춘 것임을 확인. docs 두 줄과 코드·경고 문구 일치. 범위 이탈·보안·법무 표면 없음 → **approve, risk low, blocking 없음**.
- 못 본 것: PR #8 원격 CI(GitHub 권한 없음 — '결제/사용 한도' 는 여전히 전 회차 인용), Node 20 재실행, Extension Host 실클릭(extension-host 테스트는 전부 열린 템플릿이라 이 경로를 안 지난다), Windows vsix/verify, dist 자산 복원.
- 승인이어도 남는 우려(릴리즈 노트에 적을 것): 이제 **손으로 체크한 Now 항목은 Done 에 기록되지 않고 Now 에 영구히 남는다** — 비우는 경로는 codelens `체크 해제` → `완료로 이동` 뿐이다. '체크 후 진행' 습관을 가진 사용자에게는 진행이 그다음 열린 항목을 완료하는 것으로 보인다.
- 커밋 메시지의 "거짓 completed" 표현은 약하다. 실제 이득은 **Done 중복 줄 + audit `completed` 이중 계산**(goal-metrics.ts:49, journal-summary.ts:47)과 헛된 Next 승격 방지다. 다음 회차가 이 함수를 또 만지면(들여쓰기 보존 후보) 신규 테스트 `체크된 첫 줄 다음이 들여쓴 열린 항목…` 의 기대값을 먼저 재협상해야 한다.
- [러너 22:35] review approved — 리뷰 승인 (risk=low)
- [러너 22:35] pr created — https://github.com/hkjang/vibe-code/pull/9
- [러너 22:35] ci failed — 성공이 아닌 검사: typecheck + unit tests + build=failure

## 수리 노트
- **지적이 틀렸다.** "CI 실패" 는 코드 결함이 아니다 — job 이 시작조차 안 했다(`steps: []`, `runner_id: 0`, 3초). annotation 원문: "The job was not started because recent account payments have failed or your spending limit needs to be increased." 전 회차가 '추정' 으로만 적던 결제 문제를 이번엔 커밋 bb54c58 의 check-run 에서 직접 읽어 **확정**했다(Windows gh 기존 인증; Linux gh 는 미인증, 저장소가 private 이라 익명 API 는 404).
- **고친 것 없음, 커밋 없음.** 고칠 대상이 없다. 워크플로나 단언을 건드려야만 초록이 되는 상황이라 절대 규칙대로 멈췄다. 실제 해법은 GitHub Billing & plans 수정 후 run 36722736193 re-run — 사람 개입이 필요하다.
- **대신 CI 를 로컬에서 완전 재현해 코드 무결을 입증했다**: Node 20.19.2 로 `npm ci` → `npm run check` → 두 번들 `node --check` 전부 exit 0, 88 tests. Node 22.23.1 도 동일. 즉 job 이 돌았다면 초록이었다.
- **PR 의 테스트가 진짜 대상을 잡는지도 역검증**: `plans.ts` 만 main 판으로 되돌리니 신규 6개가 옳은 증상으로 실패, HEAD 에서 22/22 통과. 비평 노트의 approve 판단과 일치한다. 파일 복원 후 트리 clean 확인.
- **여전히 확신 없는 곳**: Extension Host 실클릭 검증, Windows `vsix`/`verify`, dist 런타임 자산 복원 — 셋 다 이번에도 못 했다. 비평가는 이 셋만 재심하면 되고, CI 빨강은 재심 대상이 아니다.
