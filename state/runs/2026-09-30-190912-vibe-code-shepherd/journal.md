# PR 처리기 노트 2026-09-30-190912-vibe-code-shepherd — vibe-code PR #7
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-09-30-182136-vibe-code-improve)
# 회차 노트 2026-09-30-182136-vibe-code-improve — vibe-code
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 18:21] base pinned — main@43fd7a1
- [러너 18:21] autonomy release — 

## 정찰 노트
- 이 과제를 고른 이유: 4회 보류된 "들여쓰기 계약" 을 호출부 전수 grep 으로 풀었다 — 6곳(taskLines/isChecklistLine/moveTaskToSection/verification.ts:243/checkLine)이 trim 판정, 2곳(countChecks·isTaskLine)만 열 0 고정이라 계약 결정이 아니라 낙오자 정리다. 프로덕션 1파일로 끝나고 CodeLens 무동작이라는 관찰 가능한 증상이 있다.
- 제친 후보: 빈 제목(차선으로 남김, 린트 쪽 연결이 미확인), sectionLines 첫 줄 들여쓰기(기존 테스트 2곳이 현재 trim 동작을 기대값으로 고정해 위험이 큼), PR #6 CI(계정 결제 blocker라 기각), 워크플로 패키징(보호 경로 + Windows 필요).
- 추측으로 적은 것: 기준선 82 tests, `countChecks(goalTemplate)`=5 유지, goal-health/goal-metrics 영향 없음 — 전부 미확인. 이번 세션은 Bash 실행 승인을 못 받아 테스트를 한 번도 돌리지 못했다. 준비한 probe-indent.mjs 를 run 디렉터리에 남겼다.
- 구현자가 조심할 것: `\s*` 대신 반드시 `[ \t]*`(2026-09-29 회차의 줄 경계 사고 원인). toggleCheckbox 의 replace 두 줄도 같이 고쳐야 CodeLens 가 눌러서 동작한다 — 안 고치면 액션만 뜨고 무동작.
- PR #6(open, main 미포함)이 같은 markdown.ts 의 matchLine/setLine/touchPlan 을 건드린다. 그 함수들은 건드리지 말 것.
- [러너 18:26] scout done — 들여쓴 체크리스트 항목을 `countChecks`/`isTaskLine` 도 항목으로 인정해 진행률·CodeLens 가 트리·advance 와 어긋

## 구현 노트
- `countChecks`/`isTaskLine`/`isDoneTask` 의 열 0 앵커를 `[ \t]*` 로 완화하고 `toggleCheckbox` 의 replace 두 줄을 들여쓰기 캡처(`$1`)로 고쳤다. 들여쓴 항목이 트리·`openPlanItems` 에는 열린 항목으로 나오는데 진행률에는 안 세이고 CodeLens 도 안 붙던 불일치를 없앤다. 프로덕션 1파일(`src/util/markdown.ts`), 테스트 1파일, 커밋 2ac2a0d.
- 과제서에 없던 `isDoneTask` 를 함께 고쳤다 — plan-codelens.ts:31/37 이 원문 줄을 넘기므로 `isTaskLine` 만 완화하면 들여쓴 `- [x]` 에 "완료로 이동" 이 붙고 `toggleCheckbox`(markdown.ts:180)도 체크 해제를 못 한다. 이건 테스트로 재현해 확인했다.
- 확신 없는 곳: (1) `countChecks` 완화로 코드블록 안의 들여쓴 `- [ ]` 도 세어진다 — 의도한 범위 밖이고 실제 사용자 파일에 그런 블록이 있는지는 미확인. (2) 기존 목표·계획 파일에 들여쓴 항목이 있으면 진행률 숫자가 달라진다 — 의도된 변화지만 실제 워크스페이스로 확인하지는 않았다(goal-health/goal-metrics 테스트는 영향 없음을 실행해 확인). (3) VS Code Extension Host 에서 CodeLens 가 실제로 들여쓴 줄에 뜨고 눌러서 동작하는지는 실행 검증하지 않았다 — 순수 함수 수준까지만 증명했다.
- 일부러 하지 않은 것: `plans.ts:194 isChecklistLine` 위임(엄격도가 달라 `advancePlanText` 동작이 바뀜), `stripTask` 완화(모든 호출부가 이미 trim 된 입력을 넘겨 불필요), `countChecks` 의 섹션 범위 제한, `sectionLines` 첫 줄 trim. 전부 ideas.json 에 남겼다.
- 다음 역할이 조심할 것: `moveTaskToSection`(markdown.ts:198)과 `advancePlanText` 는 여전히 `line.trim()` 으로 항목을 옮기므로, 이제 CodeLens 가 붙은 들여쓴 하위 항목을 "완료로 이동" 하면 Done 에서 열 0 으로 펴진다. 계약 판단이 필요해 이번 범위에서 뺐고 ideas.json 에 신규 항목으로 적었다.
- PR #6 이 건드리는 `matchLine`/`setLine`/`touchPlan` 은 손대지 않았다. 다만 같은 파일이라 머지 충돌은 날 수 있다. 기준선은 실측으로 main@43fd7a1 에서 82 tests 였고 이번에 89 tests.
- [러너 18:31] brief accepted — 채택 — 과제서의 판정표(6곳 trim vs 2곳 열 0 고정)를 소스에서 그대로 확인하고 수용 기준 1~5 를 실패 재현 후 모두 충족�
- [러너 18:31] verify passed — 검증 6개 통과 (auto)

## 비평 노트
- 확인함: markdown.ts 만 main 으로 되돌려 신규 7개 중 6개가 빨개지는 것을 직접 재현(원장의 실패 출력과 일치) — 테스트는 프로덕션 planTemplate/writeSection/section/taskLines/openPlanItems 를 실제로 지난다. npm run check 전체 통과(89 tests). stripTask 비완화는 호출부 전수 확인 결과 안전(모두 trim 된 입력).
- 구현자 의심 (1) 코드블록 오산정은 실측으로 재현됨(펜스 안 `  - [ ] 예시` 로 total 6→7). 다만 goalTemplate/planTemplate/handoffTemplate/docs/README 에 해당 패턴이 하나도 없어 기본 상태 파일 진행률은 불변. ideas.json 에 남아 있어 별도 회차로 충분.
- 인수 노트의 "완료로 이동 평탄화" 는 신규 결함이 아님을 확인: main 코드에서도 advancePlanText 가 들여쓴 Now 항목을 Done 에 열 0 으로 옮긴다. 이번 변경은 도달 경로만 하나 늘렸고 텍스트는 보존된다. sectionLines 첫 줄 trim 도 기존 동작(프로필 반복 금지 항목).
- 못 본 것: Extension Host 에서 CodeLens 가 실제로 들여쓴 줄에 뜨고 눌러 동작하는지(Linux, vscode-stub 는 import 경계용). 배선 읽기와 순수 함수 수준까지만 확인했다.
- 승인이어도 남는 우려: (a) 릴리즈 노트에 "들여쓴 하위 항목이 진행률에 포함되어 기존 파일 숫자가 올라갈 수 있다" 를 사용자 가시 변경으로 적을 것. (b) open PR #6 이 같은 파일의 matchLine/setLine/touchPlan 을 건드려 머지 충돌 가능(함수는 겹치지 않음). 보안·법무 차단 사유 없음.
- [러너 18:35] review approved — 리뷰 승인 (risk=low)
- [러너 18:35] pr created — https://github.com/hkjang/vibe-code/pull/7
- [러너 18:36] ci failed — 성공이 아닌 검사: typecheck + unit tests + build=failure

## 수리 노트
- 지적이 틀렸다: "typecheck + unit tests + build = failure" 는 코드 실패가 아니다. job 109827554769 annotation 이 "recent account payments have failed or your spending limit needs to be increased" 이고, API 상 `runner_id=0`/`steps_count=0`/빈 `--log-failed` — 러너 배정 전에 죽었다. 정찰 노트의 2026-09-29 run 36518589343 과 같은 유형이며, 이제 "같은 이유 두 번" 이 확인됐다.
- 고친 것 없음, 커밋 없음. 대신 CI 와 동일한 Node 20.20.2 를 설치해 워크플로 단계를 그대로 돌렸다: `npm ci` → `npm run check`(89 tests 통과) → `node --check` 두 번들, 전부 exit 0. Node 22.23.1 에서도 동일.
- 되돌림 검증까지 했다 — markdown.ts 만 origin/main 으로 되돌리면 6 failed / 23 passed, 복원하면 다시 녹색. 변경과 테스트는 건강하다. 워크트리 clean 확인.
- 여전히 확신 없는 곳: 결제 해제 후 재실행하면 통과한다는 것은 로컬 재현에 근거한 추론이지 실제 CI 녹색을 본 것이 아니다. 그리고 windows package job 은 여전히 미실행이라 이번에도 포장·검증 증거는 없다.
- 다음 역할에게: 코드로 할 일은 없다. 계정 결제 한도를 푼 뒤 `gh run rerun 36697099434`(GH_CONFIG_DIR=/home/hkjang/.config/gh 로 Linux gh 인증됨) 이 유일한 조치다.
