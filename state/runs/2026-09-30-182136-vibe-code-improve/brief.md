# 과제서 — 2026-09-30-182136-vibe-code-improve (base main@43fd7a1, v1.4.5)

- 과제: 들여쓴 체크리스트 항목을 `countChecks`/`isTaskLine` 도 항목으로 인정해 진행률·CodeLens 가 트리·advance 와 어긋나지 않게 하기 (가치 3 / 위험 2 / 작업량 S)

- 왜: 이 저장소의 체크리스트 판정은 이미 "들여쓴 항목도 항목이다" 로 굳어져 있다 — `taskLines`(markdown.ts:83,91 `lines()` 가 trim 후 `startsWith("- [")`), `plans.ts:194 isChecklistLine`(주석에 "들여쓴 체크리스트 항목(`  - [ ] 하위`)도 항목으로 본다" 라고 명시), `moveTaskToSection`(markdown.ts:193 `isTaskLine(line.trim())`), `verification.ts:243`(`isTaskLine(line.trim())`), `checkLine`(verification.ts:53 `all[lineIndex].trim()` 로 판정). 그런데 `countChecks`(markdown.ts:111 `/^- \[( |x|X)\] /gm`)와 `isTaskLine`(markdown.ts:170 `/^- \[( |x|X)\] /`) 둘만 열 0 고정이라, 같은 `  - [ ] 하위` 줄을 `openPlanItems`(plans.ts:236)·트리는 "열린 항목" 으로 세는데 진행률(plans.ts:478 `progressLabel(countChecks(...))`, goals.ts:93/238)은 존재하지 않는 것처럼 세고, CodeLens(plan-codelens.ts:29 `if (!isTaskLine(line)) continue;` — `document.lineAt(i).text` 원문)는 그 줄에 아무 액션도 달지 않아 사용자가 계획 보드에서 체크할 수단 자체가 없다. 고치면 진행률이 트리가 말하는 열린 항목 수와 맞고, 들여쓴 하위 항목도 체크/완료 이동을 할 수 있게 된다.

- 수용 기준:
  1) `countChecks` 가 `  - [ ] 하위` / `\t- [x] 하위` 를 센다 — 프로덕션 `planTemplate(...)` 에 `writeSection` 으로 들여쓴 항목을 넣은 실제 계획 문자열에서, `countChecks(plan).total` 이 `["단계","Now","Next","검증 계획","Done"] .flatMap(n => taskLines(section(plan,n)))` 의 개수와 일치한다(현재는 들여쓴 개수만큼 적다).
  2) `isTaskLine("  - [ ] 하위")` 가 `true` 다. 그 결과 plan-codelens.ts:29 의 게이트가 들여쓴 줄을 통과하고, `toggleCheckbox(text, <들여쓴 줄 index>)` 가 **들여쓰기를 보존한 채** `- [ ]` ↔ `- [x]` 를 뒤집는다(입력과 다른 문자열을 돌려준다). 현재 `toggleCheckbox` 는 markdown.ts:179 에서 `isTaskLine(line)`(trim 없음) 으로 걸러 입력을 그대로 돌려준다 — 이게 먼저 실패해야 한다.
  3) 판정 완화가 열 0 항목과 비체크리스트 줄에는 영향이 없다 — `isTaskLine("- [ ] x") === true`, `isTaskLine("참고: 메모") === false`, `isTaskLine("  참고: 메모") === false`, `isTaskLine("  - 그냥 불릿") === false`, `isTaskLine("- [] 잘못된 표기") === false`(strict `- [ ] `/`- [x] ` 계약 유지).
  4) 기존 82 tests 가 그대로 통과한다. 특히 markdown.test.ts:154 `countChecks(goalTemplate(STAMP))` 가 `{done:0,total:5}` 로 남는지 확인할 것(템플릿에 들여쓴 항목이 없어 영향 없을 것으로 보이나 **미확인** — 실행해서 확인하라).
  5) 테스트는 가짜 문자열 조립이 아니라 프로덕션 `planTemplate`/`goalTemplate` + `writeSection` 으로 만든 실제 계획·목표 문자열에 대해, 프로덕션 `taskLines`/`section`/`sectionLines` 로 비교해서 증명한다. LF 와 CRLF(`.replace(/\n/g,"\r\n")`) 두 경우 모두.

- 건드릴 파일 (프로덕션 1개, 테스트 1~2개):
  - `src/util/markdown.ts:countChecks` — `/^- \[( |x|X)\] /gm` → 선행 공백/탭 허용(`/^[ \t]*- \[( |x|X)\] /gm`). `\s*` 는 `\n` 을 먹으니 쓰지 말 것.
  - `src/util/markdown.ts:isTaskLine` — 같은 방식으로 선행 공백/탭 허용. JSDoc 에 "들여쓴 항목도 항목" 계약을 한 줄 적어 `plans.ts:194` 주석과 짝을 맞출 것.
  - `src/util/markdown.ts:toggleCheckbox` — markdown.ts:180 의 `line.replace(/^- \[[xX]\]/, "- [ ]")` / `line.replace(/^- \[ \]/, "- [x]")` 가 열 0 고정이라 들여쓴 줄에서 **아무것도 바꾸지 않는다**. 들여쓰기를 캡처해 되돌리거나(`/^([ \t]*)- \[[xX]\]/` → `"$1- [ ]"`) `checkLine`(verification.ts:53) 처럼 비앵커 replace 로 바꿀 것. 여기를 같이 고치지 않으면 CodeLens 는 뜨는데 눌러도 무동작이 된다.
  - `tests/unit/markdown.test.ts` — 위 수용 기준 1~3 의 신규 테스트.
  - (선택, 권장하지 않음) `src/features/plans.ts:194 isChecklistLine` 를 `isTaskLine` 으로 위임 — **주의**: `isChecklistLine` 은 `startsWith("- [")` 로 느슨하고 `isTaskLine` 은 `- [ ] `/`- [x] ` 로 엄격하다. 위임하면 `- [] `·`- [-] ` 같은 표기에서 `advancePlanText` 동작이 바뀐다. 이번 과제는 **들여쓰기 축만** 통일하고 엄격도는 건드리지 않는다. 손대려면 별도 회차로.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - 실패 재현 먼저: `npx vitest run tests/unit/markdown.test.ts` (신규 테스트가 빨간 것을 확인한 뒤 고칠 것)
  - 집중: `npx vitest run tests/unit/markdown.test.ts tests/unit/plans.test.ts tests/unit/verification.test.ts`
  - 전체: `npm run check` (typecheck + vitest run 82 tests + esbuild build). 기준선은 main@43fd7a1 에서 82 tests — **이번 정찰은 실행 승인을 못 받아 직접 재확인하지 못했다(미확인). 첫 명령으로 기준선을 먼저 찍을 것.**

- 위험과 피할 것:
  - `countChecks` 는 계획뿐 아니라 **목표 파일에도** 쓰인다(goals.ts:93 상태 표시, goals.ts:238). 완화하면 들여쓴 항목이 있는 기존 목표의 진행률 숫자가 바뀐다 — 의도된 변화지만 goal-health/goal-metrics 테스트가 이 숫자에 의존하는지 확인할 것.
  - `countChecks` 는 섹션 구분 없이 파일 전체를 훑는다. 코드블록 안의 들여쓴 `- [ ]` 도 이제 세어진다. 섹션 제한을 새로 넣는 것은 이번 범위 밖(별도 과제).
  - `isTaskLine` 완화의 파급은 4곳뿐이고 전부 안전 방향이다: plan-codelens.ts:29(액션이 더 붙음), markdown.ts:179 toggleCheckbox(위에서 같이 고침), markdown.ts:193·verification.ts:243(이미 `.trim()` 을 넘기므로 동작 불변).
  - `\s*` / `\s+` 를 쓰지 말 것. 2026-09-29 회차가 정확히 이것 때문에 메타데이터 줄 경계를 고쳐야 했다. 반드시 `[ \t]*`.
  - **PR #6(head 1855dcb) 이 아직 open 이고 main 에 없다.** 그 PR 도 `src/util/markdown.ts` 를 건드린다(`matchLine`/`setLine`/`touchPlan`). 이번 과제는 `countChecks`/`isTaskLine`/`toggleCheckbox` 만 만지므로 로직 충돌은 없지만 같은 파일이라 머지 충돌 가능성은 있다. PR #6 을 rebase 하거나 그 함수들을 건드리지 말 것.
  - 이미 해결된 것을 다시 하지 말 것: CRLF 섹션 파싱(v1.4.3), advance 의 메모·들여쓰기 보존(v1.4.4), 검증 로그 중복(v1.4.5), 메타데이터 줄 경계(PR #6).
  - 보호 경로 회피: `vendor/`, `.github/workflows/`, `scripts/*.ps1`, `release/`, `src/features/checkpoints.ts`, `src/features/vibe-coders-proxy.ts` 는 건드리지 않는다.
  - 2026-09-29 회차의 CI 실패는 코드가 아니라 **계정 결제/사용 한도로 job 자체가 시작되지 않은 것**이다. 원격 CI 가 빨개도 로컬 `npm run check` 결과로 판단하고, 워크플로 조건을 완화하지 말 것.

- 차선 후보: **빈 제목이 다음 줄을 제목으로 읽고 제목 린트를 통과하는 문제** (가치 3 / 위험 1 / 작업량 S) — `markdown.ts:41 headingTitle` 의 `"^#\\s*" + prefix + ":\\s*(.+)$"` 에서 `\s*` 가 줄바꿈을 포함하므로 `# 계획: ` (값 없음) 뒤의 `상태: draft` 줄이 제목으로 읽힌다. `goal-lint.ts` 의 제목 검사도 같은 뿌리를 쓰는지 확인하고(**린트 쪽은 미확인**) `[ \t]*` 로 좁힌 뒤 빈 제목이 fallback 을 돌려주고 린트가 경고를 내는지 함께 테스트한다. 건드릴 파일: `src/util/markdown.ts:headingTitle`, `src/features/goal-lint.ts`.

## 정찰이 확인한 것 / 확인 못 한 것
- 확인함(파일을 열어 읽음): `src/util/markdown.ts` 전체(199줄), `src/features/plan-codelens.ts` 전체, `src/features/plans.ts:185-237`, `src/features/verification.ts:36-55`, 그리고 `taskLines|countChecks|isTaskLine|checkLine|sectionLines` 전체 호출부 grep. 위 판정표(6곳은 trim, 2곳은 열 0 고정)는 전부 소스에서 직접 읽은 것이다.
- 확인 못 함(미확인): 테스트 실행. 이번 정찰 세션은 Bash 실행 승인을 받지 못해 `npx vitest run` / `npm run check` / 준비한 probe 스크립트(`probe-indent.mjs`, 이 run 디렉터리에 남겨 둠 — `node --experimental-strip-types probe-indent.mjs` 로 프로덕션 소스를 그대로 import 해 판정을 출력한다)를 한 번도 돌리지 못했다. 기준선 82 tests, `countChecks(goalTemplate)` = 5, goal-health/goal-metrics 영향 여부는 구현자가 첫 명령으로 직접 찍어서 확인할 것.
