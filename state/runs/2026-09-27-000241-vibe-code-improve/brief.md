# 과제서 (2026-09-28 정찰, base main@b30117e / v1.4.3)

- 과제: `advanceCurrentPlan` 이 Now/Next 섹션의 체크리스트가 아닌 줄을 말없이 지우는 것을 막고, advance 로직을 테스트 가능한 순수 함수로 분리 (가치 4 / 위험 2 / 작업량 M)

- 왜: `src/features/plans.ts:226-243` 의 `vibe-code.advanceCurrentPlan` 은 Now/Next 를 `taskLines(section(text, …))` — 즉 **체크리스트 줄만, 그것도 `lines()` 가 trim 한 형태로** — 읽은 뒤 `writeSection(text, "Now", now)` 로 섹션 본문을 통째로 덮어쓴다. 그래서 사용자가 Now/Next 에 적어 둔 메모 줄("참고: …", "- 전제: …")과 들여쓴 하위 체크 항목이 "계획 한 단계 진행" 한 번에 사라지고, 되돌릴 수단이 없다(이 명령은 체크포인트 대상이 아니다). 계획 파일은 손으로 쓰라고 만든 정본이므로 내용 유실은 실제 손해다.
  같은 뿌리가 `src/util/markdown.ts:187` `moveTaskToSection` 에도 있다 — 대상 섹션을 `lines(section(without, target))` 로 재조립해 들여쓰기와 빈 줄을 날린다(CodeLens "완료로 이동"/"Now로 승격" 경로).

- 수용 기준:
  1) Now 와 Next 에 체크리스트가 아닌 줄(메모)이 섞인 계획에서 advance 를 돌린 뒤에도 그 메모 줄이 원래 자리·원래 문자열 그대로 남는다.
  2) 들여쓴 줄(`  - [ ] 하위`, `  참고: …`)의 들여쓰기가 advance 후에도 보존된다. `moveTaskToSection` 으로 항목을 옮겼을 때 대상 섹션에 이미 있던 들여쓴 줄도 보존된다.
  3) 기존 동작은 그대로다 — Now 의 첫 체크리스트 줄이 `- [x] <내용>` 으로 Done 끝에 붙고, Next 의 첫 체크리스트 줄이 Now 로 올라가며, `상태: active` 와 `마지막 갱신: <stamp>` 이 갱신된다. Now 에 체크리스트 줄이 없으면 아무것도 바꾸지 않는다.
  4) 테스트는 **프로덕션 `planTemplate(…)` 이 만든 실제 계획 문자열**과, 거기에 메모/들여쓴 줄을 더한 문자열로 위 1~3 을 증명한다. 고치기 전 1)·2) 테스트가 실패하는 것을 먼저 확인할 것(가짜 fs·가짜 vscode 대역 금지 — 순수 함수에 문자열을 그대로 넣는다).

- 건드릴 파일 (프로덕션 2개):
  - `src/features/plans.ts` — `advanceCurrentPlan` 콜백(226~247줄) 안의 텍스트 변환을 `export function advancePlanText(text: string, stamp: string): { text: string; completed: string; promoted: string } | null` 로 뽑아내고(위치는 `openPlanItems` 옆, vscode import 금지), 섹션을 taskLines 로 재조립하지 말고 **`section(text, name)` 의 원본 줄 배열을 splice** 하도록 고친다. 콜백은 이 함수를 부르고, null 이면 기존 경고("진행할 Now 항목이 없습니다.")를 띄우며, `writeAudit` 의 `completed`/`promoted` 는 반환값에서 받는다.
  - `src/util/markdown.ts` — `moveTaskToSection`(179~188줄)의 `[...lines(section(without, target)), …]` 를 원본 줄 배열 기반으로 바꾼다. `writeSection`/`section`/`lines` 자체의 계약은 건드리지 말 것(여러 경로가 쓴다).
  - 테스트: `tests/unit/plans.test.ts` 에 advance 케이스 추가, `tests/unit/markdown.test.ts` 에 moveTaskToSection 보존 케이스 추가.

- 검증 명령 (이 워크트리에는 `node_modules` 가 없다 — 반드시 `npm ci` 부터):
  - `npm ci`
  - `npx vitest run tests/unit/plans.test.ts tests/unit/markdown.test.ts`
  - `npm run check` (typecheck + vitest 전체 + esbuild). 기준선은 70 tests.
  - (PowerShell 전용인 `npm run vsix`/`verify`/`smoke:vscode` 는 리눅스에서 돌지 않는다 — 시도하지 말 것.)

- 위험과 피할 것:
  - **`section()` 은 본문을 trim 해서 돌려준다.** 빈 섹션(`## Done` 다음 바로 `## Risks`)이면 `""` 이고 `"".split("\n")` 은 `[""]` 이라 그대로 쓰면 빈 줄이 하나 끼어든다 — 빈 본문은 `[]` 로 다뤄야 한다. Done 섹션은 템플릿에서 비어 있으므로 이 경로를 반드시 탄다.
  - **줄바꿈**: v1.4.3 에서 `writeSection`/`touchPlan` 은 입력의 CRLF 를 보존한다. 새 코드가 `section()`(정규화된 LF 본문) 의 줄을 `writeSection` 에 넘기는 것은 안전하지만, 원본 텍스트를 직접 `split("\n")` 하면 CRLF 에서 `\r` 이 줄 끝에 남는다. `src/util/markdown.ts` 의 `normalizeEol` 을 쓰거나 `section()` 의 결과만 쪼갤 것. 기존 CRLF 테스트(markdown.test.ts 하단)가 깨지지 않는지 확인.
  - **현행 동작 중 바꾸지 말 것**: 지금 `now.shift()` 는 Now 의 첫 체크리스트 줄을 체크 상태와 무관하게(이미 `- [x]` 여도) 완료 처리한다. 이번 과제의 범위가 아니다 — 고치지 말고, 이번 테스트가 그 동작을 못 박지도 말 것(별도 판단이 필요하다).
  - 같은 패턴이 `src/features/verification.ts:41`(검증 로그 섹션을 `lines()` 로 재조립)에도 있지만 **이번 회차 범위 밖**이다. 파일 수를 늘리지 말 것.
  - 보호 경로 금지: `vendor/`(번들 수정 금지), `.github/workflows/`, `scripts/*.ps1`, `release/`, `src/features/checkpoints.ts`.
  - 테스트는 `tests/unit/vscode-stub.ts` 위에서 돈다. 새 순수 함수는 vscode 를 import 하지 않게 하고, 테스트도 vscode 를 건드리지 말 것.

- 차선 후보: `countChecks` / `taskLines` / `isTaskLine` 이 들여쓴 체크리스트 항목(`  - [ ] 하위`)을 세 갈래로 다르게 판정하는 것을 한 판정 함수로 통일 (가치 3 / 위험 1 / 작업량 S). `countChecks`(markdown.ts:101 `/^- \[( |x|X)\] /gm`)는 들여쓴 항목을 세지 않고, `taskLines`(82줄)는 `lines()` 가 먼저 trim 하므로 센다. `plan-codelens.ts:29` 와 `toggleCheckbox`(169줄)는 trim 하지 않은 원문으로 `isTaskLine` 을 불러 렌즈를 안 달고 체크도 안 되는데, `moveTaskToSection`(183줄)과 `verification.ts:239` 는 `line.trim()` 으로 받아들인다. 결과적으로 같은 계획에서 진행률 표시(countChecks)와 트리·린트 항목 수(taskLines)가 어긋난다.

## 정찰이 확인한 것 / 확인 못 한 것
- 확인: 위 줄번호와 코드는 이번 세션에 `src/features/plans.ts`, `src/util/markdown.ts`, `src/features/plan-codelens.ts`, `src/features/verification.ts` 를 직접 열어 읽은 것이다. `writeSection` 이 섹션 본문을 인자 배열로 통째로 교체하는 것(markdown.ts:61-70), advance 가 `taskLines`(체크리스트만) 를 그 인자로 넘기는 것(plans.ts:226-243) 을 코드에서 확인했다.
- **미확인(실행 검증 없음)**: 이 세션에서는 `npm ci` 가 승인되지 않아 테스트를 한 번도 돌리지 못했다. 위 유실 시나리오는 코드 독해로만 판정한 것이므로, 구현자는 **먼저 실패 재현**(메모 줄이 있는 계획 → advance → 메모 줄 사라짐)을 테스트로 찍어 확인한 뒤 고칠 것. 재현되지 않으면 차선 후보로 갈 것.
