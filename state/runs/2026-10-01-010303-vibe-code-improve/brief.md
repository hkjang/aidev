# 과제서 — 2026-10-01-010303-vibe-code-improve

- 과제: 수정 과제 — PR #9(`origin/auto/2026-09-30-2217`, head `bb54c5807c44d4d33d6afb82641425fe1de27860`) CI 실패를 Node 20 check job 로컬 재현으로 판정하고, 코드 결함이 아니면 멈추지 말고 1단계로 `initJournal` 이 재활성화 기록을 `## 세션` 이 아니라 파일 끝(= `## 다음 세션을 위한 노트`)에 쌓는 것을 고친다 (가치 3 / 위험 1 / 작업량 S)
- 왜: 자동 배정이 "릴리즈 워크플로가 같은 이유로 두 번 실패" 라고 했지만 PR #9 의 diff 는 `src/features/plans.ts` 18줄 + 테스트 + 문서 2줄뿐이라(확인: `git diff --stat main...origin/auto/2026-09-30-2217` → 4 files, 75+/8-) check job 이 코드 때문에 깨질 표면이 거의 없다. 직전 3회차(PR #7·#8·#6)가 모두 같은 게이트를 Node 20 에서 전부 exit 0 으로 재현하고 "코드 결함 아님" 으로 판정했으니, 0단계에서 같은 결과가 나오면 거기서 멈추지 말고 1단계 실제 결함을 고쳐 회차를 산출물 있는 상태로 끝내라. 1단계 결함은 `src/features/journal.ts:34-35` 로, 오늘 일지가 이미 있으면 `fs.appendFileSync(file, "- HH:MM — 확장 재활성화\n")` 로 **파일 끝에** 붙는데 프로덕션 템플릿(`journalTemplate`, journal.ts:9-23)의 마지막 섹션은 `## 다음 세션을 위한 노트` 다. 즉 `## 세션` 섹션은 최초 활성화 한 줄만 갖고, 재활성화 기록은 전부 "다음 세션을 위한 노트" 안으로 흘러든다. 고치면 하루에 여러 번 VS Code 를 여는 실사용에서 세션 기록이 제 섹션에 순서대로 쌓이고, 노트 섹션이 타임스탬프로 오염되지 않는다.

## 0단계 — PR #9 CI check job 로컬 재현 (먼저, 게이트)

이것을 먼저 하고 결과를 원장에 로그로 남겨라. 통과하면 1단계로 넘어간다. 실패하면 **1단계를 버리고** 그 실패 원인을 고친다.

```
git -C <worktree> archive origin/auto/2026-09-30-2217 | (mkdir -p <run>/validation/pr9 && tar -x -C <run>/validation/pr9)
# CI 와 같은 Node 20 확보 (2026-09-30 회차가 동작 확인한 절차):
#   npm install --no-save node@20.19.2   후 <dir>/bin/node 를 PATH 앞에 둔다. 20.19.2 는 engines.node 정확 핀.
cd <run>/validation/pr9
npm ci
npm run check
node --check dist/extension.js && node --check dist/extension.core.js
```
`.github/workflows/ci.yml:17-24` 의 순서와 같다(확인: 이번 정찰이 ci.yml 을 읽었다). 네 단계 exit code 를 전부 로그에 남겨라.

- **원격 상태는 이번 정찰에서 미확인이다.** 이 세션의 `gh` 는 승인되지 않아 PR #9 의 check 결과·annotation 을 읽지 못했다. PR 번호 ↔ 브랜치 대응(`#9` = `auto/2026-09-30-2217`)은 배정 URL 과 "PR #8 = auto/2026-09-30-2132"(2026-09-30 원장)로 미루어 본 **추정**이다. 로컬 `origin/auto/2026-09-30-2217` head 는 `bb54c58` 로 확인했다.
- 2026-09-29 회차가 보관한 annotation("The job was not started because recent account payments have failed or your spending limit needs to be increased.", `steps=[]`, `runner_id=0`)이 유효하면 원격은 결제/한도로 job 이 시작조차 안 한 것이다. **재확인하지는 못했다.** 결제 설정 변경·CI 재시도·원격 push 는 하지 마라.

## 1단계 — `initJournal` 의 세션 기록이 `## 세션` 에 쌓이게 한다

- 수용 기준:
  1. 프로덕션 `journalTemplate(ymd, clock)` 이 만든 실제 일지에 재활성화 기록을 3회 연속 넣으면, 세 줄이 **모두 `## 세션` 섹션 안에** 최초 활성화 줄 다음에 시간 순으로 있다 — 프로덕션 파서 `section(text, "세션")`/`sectionLines(text, "세션")` 로 확인한다.
  2. 같은 결과에서 `section(text, "다음 세션을 위한 노트")` 에 `확장 재활성화` 가 **하나도 없다** (현재 동작은 전부 여기에 있다).
  3. 템플릿의 다른 섹션(`## 작업 내용` 의 안내 문장, `## 다음 세션을 위한 노트`)과 제목 줄은 원문 그대로 남는다 — 메모·빈 줄 유실 없음.
  4. CRLF 일지(같은 템플릿을 `.replace(/\n/g, "\r\n")` 한 것)에 넣어도 출력에 LF-only 줄이 없다(`detectEol`/`normalizeEol` 로 확인). v1.4.5 가 `appendVerificationLog`/`checkLine` 에서 고친 것과 같은 계열이다.
  5. `## 세션` 섹션이 없는 일지(사용자가 제목을 지운 경우)에도 기록이 유실되지 않는다 — `hasSection` 이 false 면 기존처럼 파일 끝에 붙인다.
  6. `npm run check` 전체 통과. 테스트 수는 기준선 + 신규만 늘어난다(main@43fd7a1 기준선 **82 tests** — 2026-09-30 회차가 실측한 값, 이번 정찰은 `npm ci` 를 돌리지 않아 재실측하지 않았다).

- 건드릴 파일 (프로덕션 1개 + 테스트 1개):
  - `src/features/journal.ts:9 journalTemplate` — `export` 추가(테스트가 프로덕션 템플릿을 그대로 쓰기 위해서. 한 단어 변경).
  - `src/features/journal.ts` — 신규 `export function appendSessionLine(text: string, line: string): string`. `hasSection(text, "세션")` 이 true 면 `normalizeEol(text).split("\n")` 에서 `## 세션` 다음의 첫 `## ` 제목(또는 EOF) 직전, 마지막 비어 있지 않은 줄 다음에 `line` 을 끼우고 `restoreEol(…, detectEol(text))` 로 돌려준다. false 면 지금처럼 끝에 붙인다. 세 함수 모두 `src/util/markdown.ts` 에서 이미 export 되어 있다(확인: markdown.ts:9/15/20/29).
  - `src/features/journal.ts:35` — `fs.appendFileSync(file, …)` 를 `fs.writeFileSync(file, appendSessionLine(fs.readFileSync(file, "utf8"), \`- ${clock} — 확장 재활성화\`))` 로 바꾼다. 34행(신규 생성 경로)과 37-47행(디렉터리 생성·`log`)은 건드리지 마라.
  - `tests/unit/journal.test.ts` — 신규. `journalTemplate` 과 `appendSessionLine`, 그리고 프로덕션 `section`/`sectionLines`/`hasSection`/`detectEol`/`normalizeEol` 만 쓴다. **손으로 조립한 일지 문자열·가짜 `CoreHost`·가짜 vscode 객체를 쓰지 마라** — `initJournal` 은 `vscode`/`workspaceRoot()`/`CoreHost` 에 묶여 있어 대역 없이는 호출할 수 없으니, 테스트 경계는 순수 함수 `appendSessionLine` + 프로덕션 템플릿으로 잡는다(이 저장소가 `appendVerificationLog`·`advancePlanText` 에서 쓰는 것과 같은 경계).

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - 실패 재현 먼저: `npx vitest run tests/unit/journal.test.ts` → 수용 기준 1·2·4 가 빨개지는 것을 로그로 남긴다.
  - 고친 뒤: `npx vitest run tests/unit/journal.test.ts tests/unit/journal-summary.test.ts`
  - 전체: `npm run check` (typecheck + vitest run + esbuild build)
  - 인과 확인: `git checkout HEAD -- src/features/journal.ts` 후 같은 테스트가 다시 빨개지는 것을 확인한다.

- 위험과 피할 것:
  - **`.github/workflows/ci.yml` 을 완화하지 마라.** 0단계를 통과시키려고 조건·엔진 핀을 손대는 것은 금지다.
  - **main 에 아직 머지되지 않은 open PR 과 같은 파일을 건드리지 마라.** PR #6·#7 = `src/util/markdown.ts`(+`src/features/goal-lint.ts`), PR #8·#9 = `src/features/plans.ts`. `journal.ts` 는 이 네 PR 이 손대지 않았다(확인: PR #9 diff 4 files). markdown.ts 는 **읽기만**(export 된 함수 호출만) 하라.
  - `writeSection` 을 쓰지 마라 — 본문 전체를 교체하므로 `taskLines`/필터로 걸러 쓰면 메모·들여쓰기가 날아간다. `sectionLines` 도 `section().trim()` 을 거쳐 섹션 첫 줄 들여쓰기와 앞뒤 빈 줄을 지우므로 **쓰기 경로에 쓰지 말고 검증(읽기)에만** 써라. 삽입은 `normalizeEol().split("\n")` 인덱스 계산으로 해라.
  - 공백 매칭에 `\s*` 를 쓰지 마라 — 줄바꿈을 먹어 다음 줄을 흡수한다(2026-09-30 `headingTitle` 회차의 실제 결함). 필요하면 `[ \t]*`.
  - 같은 EOF-append 형태인 `src/features/journal-summary.ts:101 appendSessionSummary`(일지 끝에 새 `## 세션 요약` 섹션을 붙이는 것 — 계약이 다르다)와 `src/features/handoff-on-exit.ts:48`(다른 파일)은 이번 범위가 아니다. 건드리지 마라.
  - 보호 경로 회피: `checkpoints.ts`, `vibe-coders-proxy.ts`, `vendor/`, `scripts/*.ps1`, `release/` 는 열지 않는다.
  - 버전 bump·`CHANGELOG` 는 릴리즈 역할의 몫이다. 문서는 `## 세션` 계약을 단언하는 줄이 실제로 있을 때만 고쳐라(이번 정찰은 그런 줄을 찾지 않았다 — 미확인).

- 차선 후보: **`appendSessionSummary` 가 CRLF 일지에 LF-only 요약 블록을 붙여 줄바꿈이 섞이는 것 고치기** (2/1/S) — `journal-summary.ts:101` 이 `(journal.endsWith("\n") ? "" : "\n") + "\n" + summary` 를 그대로 append 하고 `buildSessionSummary` 는 `lines.join("\n")` 로만 만든다(확인: journal-summary.ts:85·101). v1.4.5 와 같은 계열이고 `detectEol`/`restoreEol` 로 끝난다. 1순위가 성립하지 않을 때(예: 0단계가 실패해 시간을 다 쓰거나, `## 세션` 삽입 계약에 막힐 때) 이것을 골라라. 단 실사용 일지가 CRLF 가 되는 경로는 미확인이라 가치가 낮다.
