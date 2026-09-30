- 과제: 수정 과제 — PR #8(`origin/auto/2026-09-30-2132`, head `9b50afc`) CI 실패를 Node 20 check job 으로 재현 판정하고, 코드 결함이 아니면 `advancePlanText` 가 이미 `- [x]` 인 Now 항목을 "완료" 처리하는 것을 고치기 (가치 4 / 위험 2 / 작업량 M)
- 왜: PR #8 은 main@43fd7a1 위의 커밋 하나(`9b50afc`, `headingTitle`/제목 린트 수정)뿐이고 CI 실패로 자동 배정됐지만, 정찰은 이번 회차에 **원격 상태를 전혀 읽지 못했다**(`gh auth status` 실행 권한 거부, `WebFetch` 권한 없음 — 둘 다 미확인). 지난 두 회차(PR #6/#7)는 같은 check job 을 로컬 Node 20 에서 전부 exit 0 으로 재현하고 원격 실패가 계정 결제/사용 한도로 job 이 시작되지 않은 것(annotation: "The job was not started because recent account payments have failed…", `steps=[]`, `runner_id=0`)임을 기록했다. 따라서 0단계로 같은 게이트를 한 번 더 돌려 "코드 결함인지" 를 증거 로그로 판정하고, 결함이 아니면 남은 시간을 실제 사용자 오작동 1건(`advanceCurrentPlan` 이 이미 완료된 항목을 다시 완료 처리하고 Next 를 헛되게 소진하는 것)에 쓴다.

## 0단계 (게이트, 먼저 이것만 — 15분 안)
`.github/workflows/ci.yml:13-24` 의 `check` job 과 **같은 순서·같은 명령**으로 PR #8 head 를 돌린다. `9b50afc` 는 로컬 `origin/auto/2026-09-30-2132` 에 이미 있으므로 fetch 가 필요 없다(원격 fetch 는 인증이 없어 실패한다).

1. 별도 디렉터리에 head 를 추출: `git archive origin/auto/2026-09-30-2132 | tar -x -C <run>/validation/pr8` (작업 트리를 오염시키지 말 것).
2. Node 20 확보 — CI 는 `node-version: 20`(20.20.x 로 해석)이고 `package.json:14 engines.node` 는 정확 핀 `"20.19.2"` 다. 지난 회차에서 동작 확인된 절차: `npm install --no-save node@20.19.2` 후 그 디렉터리의 `bin/node` 를 PATH 앞에 둔다. **두 가지가 다르므로 20.19.2 로 재현하고, 시간이 남으면 20.20.x 로도 한 번 더 돌려 EBADENGINE 이 경고인지 에러인지 로그로 남긴다**(`.npmrc` 가 없어 `engine-strict` 는 꺼져 있으므로 경고일 것으로 보이나 이번 정찰은 실행하지 않았다 — 미확인).
3. `npm ci` → `npm run check`(= `npm run typecheck && vitest run && node scripts/build.mjs`) → `node --check dist/extension.js` → `node --check dist/extension.core.js`.
4. 판정을 로그 파일로 남긴다(`validation/pr8-node20-check.log`, 각 단계 EXIT_CODE 포함).
   - **실패가 재현되면** 그것이 이번 회차의 과제다. 실패한 단계의 스크립트(`scripts/build.mjs`) 또는 테스트를 읽고 원인을 고친 뒤 같은 4개 명령을 다시 통과시켜라. 아래 1단계는 버린다.
   - **전부 exit 0 이면** "PR #8 의 원격 실패는 코드 결함이 아니다" 를 로그와 함께 원장에 적고 1단계로 넘어가라. 멈추지 말 것.

## 1단계 (0단계가 전부 통과했을 때의 실제 수정)
`src/features/plans.ts:209 advancePlanText` 는 Now/Next 의 **첫 체크리스트 줄**을 `isChecklistLine`(plans.ts:194, `line.trim().startsWith("- [")`)로 집는데, 이 판정은 `- [x]` 도 통과한다.
- `plans.ts:213` `now.findIndex(isChecklistLine)` → Now 첫 줄이 `- [x] 이미 끝난 작업` 이면 그 줄을 Now 에서 빼고 `markdown.ts:100 stripTask`(`/^- \[(?: |x|X)\]\s*/` — `x` 도 벗긴다)로 접두어를 벗겨 `Done` 에 `- [x] 이미 끝난 작업` 으로 또 붙인다. 실제로 진행돼야 할 **열린 Now 항목은 건너뛰어진다.**
- `plans.ts:220` `next.findIndex(isChecklistLine)` → Next 첫 줄이 `- [x]` 여도 Now 로 승격돼, 다음 advance 의 거짓 "완료" 재료가 된다.
- 부수 피해: `plans.ts:274` 가 감사에 `completed: advanced.completed` 를 기록하므로 감사 로그가 "이미 끝난 작업을 이번에 완료했다" 는 거짓을 남기고, 사용자 입장에서는 버튼을 눌러도 Next 만 Now 로 빠져나가며 일이 진행되지 않는다.

**제품 계약은 이미 정해져 있다 — 새로 판단할 것 없음.** 같은 파일 `plans.ts:236 openPlanItems` 가 `taskLines(...).filter((l) => !/^- \[[xX]\]/.test(l))` 로 `- [x]` 를 "열린 항목이 아님" 으로 보고, `plans.ts:286` 의 done 전환 게이트가 그 값을 쓴다. 따라서 advance 도 **첫 열린(unchecked) 항목**에 작용해야 한다.

- 수용 기준:
  1) 0단계 4개 명령의 종료 코드가 담긴 로그 파일이 `<run>/validation/` 에 남고, "코드 결함인지" 판정이 원장에 한 문장으로 적혀 있다.
  2) (1단계) Now 첫 줄이 `- [x]` 이고 그 아래에 열린 `- [ ]` 항목이 있는 실제 계획에서 `advancePlanText` 가 **열린 항목**을 완료하고, `- [x]` 줄은 Now 에 그대로 남는다(`completed` 도 열린 항목의 텍스트다).
  3) Next 첫 줄이 `- [x]` 일 때 그것을 승격하지 않고 첫 열린 Next 항목을 승격한다. Next 에 열린 항목이 없으면 `promoted === ""` 이고 Next 는 그대로다.
  4) Now 에 체크리스트 줄이 있지만 **전부 `- [x]`** 면 `advancePlanText` 는 `null` 을 돌려주고(명령이 "진행할 Now 항목이 없습니다." 를 띄운다) 텍스트를 한 글자도 바꾸지 않는다.
  5) 기존 보존 계약 회귀 없음 — 메모 줄(`참고: …`)·들여쓴 하위 항목·섹션 빈 줄이 그대로 남고, 들여쓴 `  - [ ] 하위` 는 여전히 항목으로 인정된다(`isChecklistLine` 의 trim 계약 유지). CRLF 계획에서도 같은 결과(기존 plans.test.ts:232 의 crlf 동등성 테스트가 계속 통과).
  6) 실패 재현을 먼저 보일 것: 고치기 전 새 테스트가 빨갛고, 고친 뒤 통과하며, `src/features/plans.ts` 만 `git checkout HEAD --` 로 되돌리면 같은 테스트가 다시 빨개진다(인과 확인 로그).

- 건드릴 파일 (프로덕션 1개 · 테스트 1개):
  - `src/features/plans.ts:194` — `isChecklistLine` 옆에 "열린 항목" 판정을 하나 더 둔다(예: `isOpenChecklistLine = (l) => isChecklistLine(l) && !/^- \[[xX]\]/.test(l.trim())`). **`src/util/markdown.ts` 의 `isDoneTask`/`isTaskLine` 을 쓰지 말 것** — main 에서 그 둘은 열 0 고정이라 들여쓴 `  - [x] 하위` 를 놓치고, 게다가 markdown.ts 는 PR #7·#8 이 둘 다 수정 중인 파일이라 손대면 충돌이 난다.
  - `src/features/plans.ts:213,220` — 두 `findIndex(isChecklistLine)` 를 새 판정으로 바꾼다. `splice`/`writeSection` 구조와 `stripTask` 는 그대로 둔다(v1.4.4 의 줄 보존 수정을 되돌리지 말 것).
  - `tests/unit/plans.test.ts:174 describe("advancePlanText")` — 신규 테스트를 이 describe 안에 더한다. 입력은 **프로덕션 함수로만** 만든다: `planTemplate(...)` 로 실제 계획을 만들고, Now 항목을 체크하는 것도 프로덕션 `toggleCheckbox(text, lineIndex)`(`markdown.ts:175`)로 한다 — 손으로 조립한 문자열이나 가짜 객체로 증명하지 말 것. 검증도 프로덕션 `sectionLines`/`section`/`taskLines`/`openPlanItems` 로 읽어 비교한다. CRLF 케이스는 기존 `crlf()` 헬퍼를 쓴다.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - 집중: `npx vitest run tests/unit/plans.test.ts` (빨강→초록 재현용)
  - 전체 게이트: `npm ci` (worktree 에 `node_modules` 가 없다 — 이번 정찰이 확인) → `npm run check` (typecheck + `vitest run` 전체 + esbuild). main@43fd7a1 기준선은 지난 회차 실측 **82 tests** 다(이번 정찰은 `npm ci` 를 돌리지 않았으므로 직접 다시 재 보고 신규 개수를 더해 보고할 것).
  - 번들: `node --check dist/extension.js`, `node --check dist/extension.core.js`
  - Node: 0단계와 1단계 모두 **Node 20.19.2**(= `engines.node`)에서 최소 한 번 통과시켜라. 개발 셸 기본은 v22.23.1 이다(이번 정찰 확인).

- 위험과 피할 것:
  - **`.github/workflows/ci.yml` 을 고치지 말 것.** 이 회차의 목적은 워크플로를 통과시키는 것이 아니다. `continue-on-error: true`(ci.yml:57)와 `if: steps.assets.outputs.found == 'true'`(ci.yml:60/65/70/80) 때문에 Windows `package` job 은 릴리즈 VSIX 를 못 받으면 포장·검증·업로드를 조용히 건너뛰고 초록이 된다 — 알려진 별 과제(보류 아이디어 4/2/M)이고 Windows PowerShell 이 필요해 이번 회차 범위가 아니다. 조건 완화는 금지다.
  - `src/util/markdown.ts`, `src/features/goal-lint.ts` 를 건드리지 말 것 — PR #7(`origin/auto/2026-09-30-1821`, markdown.ts)과 PR #8(markdown.ts + goal-lint.ts)이 모두 open 이고 main 에 없다. 이번 변경을 `plans.ts` 안에 가두면 세 브랜치가 충돌 없이 머지된다.
  - `checkpoints.ts`(임시 Git index/commit-tree/update-ref), `vibe-coders-proxy.ts`(전역 provider 설정), `vendor/`, `scripts/*.ps1`, `release/` — 보호 경로. 이번 과제와 무관하다.
  - 감사(`writeAudit`)에 새 원문 필드를 추가하지 말 것. `completed`/`promoted` 가 올바른 값이 되면 충분하다.
  - 임시 디렉터리를 상위 Git 저장소 안에 만들지 말고, `TMPDIR` 을 쓰면 `GIT_CEILING_DIRECTORIES=$TMPDIR` 도 함께 지정하라(과거 교훈).
  - 원격 push·CI 재시도·결제 설정 변경은 하지 말 것. 원격 상태를 확인할 수 없으면 "미확인" 으로 적어라 — 지난 회차가 확인했다는 사실을 이번 회차의 확인으로 옮겨 적지 말 것.

- 차선 후보: `advancePlanText`/`moveTaskToSection` 이 항목을 옮길 때 `line.trim()` 으로 붙여 들여쓴 하위 항목의 들여쓰기를 잃는 것 (3/2/S). 단 "하위 항목을 옮길 때 원래 들여쓰기를 지켜야 하나(부모가 따라오지 않아 고아 들여쓰기가 된다), 펴는 것이 맞나" 라는 제품 계약 판단이 먼저다 — 1순위와 달리 저장소 안에 정답 근거가 없으므로, 고른다면 그 판단을 과제 첫 줄에 명시하고 시작하라. 1순위와 같은 함수(`advancePlanText`)를 건드리므로 둘을 같은 회차에 섞지 말 것.
