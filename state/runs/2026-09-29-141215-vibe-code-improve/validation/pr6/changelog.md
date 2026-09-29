# Changelog

## 1.4.5

### Fixes (수정)

- Running a verification on a CRLF file added a new `## 검증 로그` section every time. `appendVerificationLog` in `src/features/verification.ts` still carried its own copy of the section regex — `/(?:^|\n)## 검증 로그\n/` — which the 1.4.3 CRLF consolidation did not reach, and it cannot match `## 검증 로그\r\n`. So on a CRLF `.vibe-code/*.md` file the existing section was never found and a fresh one was appended to the end of the file on every run: three verifications left three sections. The check is now the shared `hasSection()`, so one section is found and appended to, whatever the file's line endings are.
- The result of a verification appeared never to update. `goals.ts` reads `최근 검증` from `section(text, "검증 로그")`, which returns the *first* section of that name — and the first one was the stale original, while every new record went into a duplicate further down the file. With duplicates gone, the goal status shows the latest runs again.
- Ticking a verification checklist item no longer rewrites the whole file's line endings. `checkLine` split on `/\r?\n/` and rejoined with `\n`, so checking one box in a CRLF file converted every line in it to LF — one click produced a whole-file diff. It now remembers the input's line ending with `detectEol` and restores it, the same rule the other editing functions have followed since 1.4.3.
- Notes and indentation already in the `## 검증 로그` section survive an append. The existing records were re-read through `lines()`, which drops blank lines and trims each line, so an indented output line (`  출력: 78 tests passed`) or a blank line inside the section was flattened or deleted when the next record was appended. The section is now read with `sectionLines()` — exactly as written.

### Internal (내부)

- `restoreEol` is exported from `src/util/markdown.ts` (a one-word change; the function is unchanged) so `verification.ts` can restore line endings through the shared helper rather than a second copy of it. With `hasSection`, `sectionLines`, `detectEol`, `normalizeEol` and `restoreEol` all coming from `markdown.ts`, `verification.ts` no longer parses sections on its own — there is one section parser again. Two production files changed.
- Unchanged behaviour worth stating: `runAndRecord` depends on `vscode` and is untouched and still uncovered; `writeSection` and `toggleCheckbox` restore line endings themselves, so `restoreEol` is applied only on the new-section path and in `checkLine`, never twice.

### Tests (테스트)

- `tests/unit/verification.test.ts` gains a `verification log on CRLF files` block (4 tests). They apply `appendVerificationLog` three times in a row to a real CRLF file — the production `goalTemplate` / `planTemplate` output with `\n` replaced by `\r\n`, not a hand-assembled fixture — and read the result back through the production `sectionLines()` / `normalizeEol()`. Covered: exactly one `## 검증 로그` section in a CRLF goal file and in a CRLF plan file after repeated appends with all three records in order, CRLF preserved on the output of both templates and by `checkLine`, and indented and blank lines already in the section kept. All four were confirmed failing against the previous code — `expected 3 to be 1` (goal file, three sections), `expected 2 to be 1` (plan file), `expected false to be true` (a non-CRLF line in the output) and the lost indentation on `  출력: 78 tests passed` — before the fix landed. `npm run check` is now 82 tests (78 + 4).

## 1.4.4

### Fixes (수정)

- Advancing a plan one step silently deleted your own notes. `vibe-code.advanceCurrentPlan` read `## Now` and `## Next` through `taskLines(section(...))` — checklist lines only, and trimmed at that — then wrote the result back with `writeSection`, which overwrites the whole section body. So a note line you had typed under Now (`참고: …`) and any indented sub-item (`  - [ ] 하위 항목`) were gone after a single `계획 한 단계 진행`, along with their indentation. This command is not covered by the destructive-command checkpoint, so there was nothing to restore from either. Both sections are now read as written and only the one item is spliced out, so every other line — notes, indentation, blank lines inside the section — comes back unchanged.
- The plan CodeLens `완료로 이동` did the same thing to `## Done`. `moveTaskToSection` rebuilt the *target* section from `lines(section(...))`, so moving a task into Done flattened the indentation of everything already recorded there. The target section is now reassembled from its original lines.

### Internal (내부)

- `sectionLines(text, name)` is added to `src/util/markdown.ts`: the lines of a `## <name>` section exactly as written, with an empty or missing section returning `[]` rather than `[""]`. It is the reading half that `writeSection` was missing — editing a section no longer means filtering it first.
- The advance step is extracted from the command callback into an exported, vscode-free `advancePlanText(text, stamp)` returning `{ text, completed, promoted }`, or `null` when Now has no checklist line — the same pattern `archiveDonePlans` and `createCheckpoint` follow, which is what makes it testable against real plan text instead of a stubbed `vscode` surface. The audit entry takes `completed` / `promoted` from the return value.
- Unchanged behaviour worth stating: a Next item promoted into Now is still `trim()`ed, and the very first and last blank lines of a section body are still dropped by `section()`. Two production files changed.

### Tests (테스트)

- `tests/unit/plans.test.ts` gains an `advancePlanText` block (6 tests) and `tests/unit/markdown.test.ts` gains `sectionLines` and `moveTaskToSection` cases (2 tests). They run against a plan string built by the production `planTemplate(...)` — with notes and indented lines added through the production `writeSection` — rather than hand-written fixtures, and there is no fake `fs` and no fake `vscode`. Covered: non-checklist and indented lines in Now/Next surviving an advance, the first Now item landing at the end of Done with 상태/마지막 갱신 bumped, existing indented Done records kept, `null` when Now has no checklist line, an empty Next promoting nothing, and the same result on a CRLF plan with line endings preserved. Three of them were confirmed failing against the previous code — `expected '- [ ] 하위 항목\n…' to be '참고: Now 메모\n  - [ ] 하위 항목\n…'` and the flattened Done indentation — before the fix landed. `npm run check` is now 78 tests (70 + 8).

## 1.4.3

### Fixes (수정)

- Plan and goal files with CRLF line endings were unreadable to the section parser. The section regexes in `src/util/markdown.ts` allowed exactly one `\n` after a `## <이름>` heading, so on a CRLF `.vibe-code/*.md` file every section-shaped operation failed at once: `section()` and `subsection()` returned the empty string, `writeSection()` returned its input untouched, and `moveTaskToSection()` returned `null` — which is why the plan CodeLens `완료로 이동` / `Next로 되돌리기` did nothing at all, silently, with no error anywhere. The reading functions now normalise CRLF to LF at entry (`normalizeEol`) before matching, so the same file reads the same either way. CRLF is what you get from `core.autocrlf` on Windows or from a plan file that has been through a zip round-trip, both of which `docs/autonomous-goal-workflow.md` treats as supported.
- Editing a plan no longer rewrites the whole file's line endings. `writeSection`, `moveTaskToSection`, `toggleCheckbox`, `touchPlan` and `setLine` remember the input's line ending with `detectEol` (a mixed file follows its first line break, the same rule as VS Code's `TextDocument.eol`) and restore it after editing, so ticking one checkbox in a CRLF file produces a one-line diff instead of converting every line to LF.
- Goal and plan lint stopped reporting sections that are present. `goal-lint.ts` carried its own copy of the section regex with the same `\n` assumption, so a CRLF plan was flagged with `## 단계 섹션이 없습니다` and one warning per required section. The existence check is now the single shared `hasSection()`, so the parser and the lint cannot disagree about whether a section exists.

### Internal (내부)

- `detectEol`, `normalizeEol` and `hasSection` are exported from `src/util/markdown.ts`, and the two duplicated section regexes in `src/features/goal-lint.ts` are gone — there is one section parser again. Two production files changed.

### Tests (테스트)

- `tests/unit/markdown.test.ts` gains a `CRLF 줄바꿈` block (7 tests): EOL detection on mixed input, reading functions returning the same values under CRLF as under LF, `writeSection` and `moveTaskToSection` producing the same edit with line endings preserved, `toggleCheckbox`/`touchPlan`/`setLine` preserving them too, and — through the production `planTemplate` / `goalTemplate` rather than hand-written fixtures — `lintPlan`, `lintGoal`, `parseGoal` and `openPlanItems` reading CRLF files identically, with no missing-section warnings. Six of them were confirmed failing against the previous code (`expected '' to be '- [ ] A\n- [ ] B'`, `expected null not to be null`, the LF-converted `toggleCheckbox` output, and the spurious lint warnings) before the fix landed. `npm run check` is now 70 tests (63 + 7).

## 1.4.2

### Fixes (수정)

- Archiving a completed plan could silently destroy an older one. `vibe-code.archiveDonePlans` moved every `상태: done` plan into `.vibe-code/plans/archive/`, and when a plan of the same name was already archived it ran `fs.unlinkSync(dest)` on the existing copy before renaming the new one over it — so simply repeating the "plan → done → archive" loop erased the earlier plan's record, with no trash can to recover it from. A collision now preserves both: the incoming file is archived as `<이름>-<stamp>.md` (and `<이름>-<stamp>-<n>.md` if that is taken too) and the existing archived plan is left alone. This also removes a policy contradiction with `restoreArchivedPlan`, which already renamed rather than overwrote on collision.
- The collision suffix is now produced by one shared `conflictStamp()` (`YYYYMMDDHHMMSS` in Asia/Seoul), so archiving and restoring stamp names in the same format. `restoreArchivedPlan` previously derived its stamp from `toISOString()`, i.e. UTC.
- The archive notification and the `archiveDonePlans` audit entry now report the names files were actually written under, not the names they had in `plans/`, so the audit trail points at files that exist.
- `docs/autonomous-goal-workflow.md` states the collision behaviour.

### Internal (내부)

- The archiving loop is extracted from the command handler into an exported, vscode-free `archiveDonePlans(plansDir, archiveDir, stamp)` returning `{ name, dest }` per plan, alongside `freeName()` and `conflictStamp()` — the same pattern `checkpoints.ts` uses, which is what makes it testable without growing the test stub for `vscode.commands`.

### Tests (테스트)

- `tests/unit/plans.test.ts` (10 tests): `plans.ts` had no coverage at all. The tests run against real directory trees created with `fs.mkdtempSync` — no fake `fs`, no fake vscode command surface. Covered: the same-name archive collision (both copies survive, existing content intact), only `상태: done` plans move, the `archive/` subdirectory is not descended into, `freeName` fallbacks, `conflictStamp` format, `selectPlanName` fallback, and `openPlanItems`. The two collision tests were confirmed failing against the previous code — the existing archived plan's content disappeared — before the fix landed. `npm run check` is now 63 tests (53 + 10).

## 1.4.1

### Fixes (수정)

- Checkpoints snapshotted only tracked changes. `createCheckpoint` used `git stash create`, which never includes untracked files, so a brand-new file that had not been `git add`ed yet was *not* protected when a destructive command (`rm -rf`, `git clean -f`, …) ran — exactly the case the 1.4.0 safety net was meant to cover. The snapshot is now built through a throwaway index (`GIT_INDEX_FILE` under `os.tmpdir()`, then `read-tree` → `add -A` → `write-tree` → `commit-tree`), so uncommitted changes to tracked files and untracked files alike (ignored files still excluded) are pinned under `refs/vibe-checkpoints/<stamp>`. The working tree, the index and the stash stack are left untouched.
- The restore block in the checkpoint note told you to run `git stash apply`, which fails: a snapshot commit is not in stash format and git rejects it. Notes now carry commands that actually work — `git diff <commit>` to review and `git checkout <commit> -- .` to restore.
- A repository with no commits yet is snapshotted as a parentless commit instead of only leaving a note.
- `vibe-code.checkpointBeforeDestructive` descriptions (en/ko) and `docs/autonomous-goal-workflow.md` now state the tracked/untracked behaviour instead of the vaguer "working tree snapshot".

### Tests (테스트)

- `tests/unit/checkpoints.test.ts` (9 tests): the checkpoint feature had no coverage. The tests run against real temporary git repositories created with `fs.mkdtempSync` rather than a fake git, and the restore test parses the commands out of the generated note and executes them to prove the files come back. Covered: untracked files in the snapshot, working tree / index / stash stack left untouched, restore, empty repository, non-git directory, and the `isDestructiveCommand` table. Four of them failed against the 1.4.0 code for the reasons above before the fix landed. `npm run check` is now 53 tests.

## 1.4.0

### Goal loop (목표 기반 지속 개발)

- The extension now drives the `/goal` loop instead of relying on the slash-command prompt alone. Three new core hooks (`onTask`, `promptContext`, plus the existing `onCommand`) are documented in `vendor/PATCHES.md`.
- Prompt context: every system prompt ends with a compact `VIBE CODE GOAL STATE` block (title, status, phase, next action, open Now/Next items, completion progress, loop rules) built from `.vibe-code/goals/current.md`; cached on the file's mtime.
- Auto resume: `vibe-code.goalAutoResume` (`ask` | `always` | `never`) offers `/goal 이어서` a few seconds after activation when the active goal still has Now items and was updated within `vibe-code.goalResumeWithinHours` (72). `Vibe Code: 목표 이어서 진행` does the same on demand.
- Task completion check: when a task finishes without touching the goal file you get a warning with `목표 열기` / `이어서 갱신 요청`; when Now items remain you get `계속 진행`. Every completion is audited (`goal/taskCompleted` with tokens, cost, duration, whether the goal was updated).
- Verification runner: `## 검증 계획` / `## 완료 기준` lines that contain a command (backticked or starting with a known runner such as `npm`, `go`, `pytest`) get a `검증 실행` CodeLens. The command runs in the workspace root with output streamed to the Vibe Code channel; the result is appended to `## 검증 로그` of the plan and the goal file, the line is ticked on success, and a `plan/runVerification` audit entry is written. `Vibe Code: 검증 실행` offers the active plan's commands or free input.
- `Vibe Code: 변경 파일 테스트 추천`: maps `git status` changes to test commands (sibling `.test`/`.spec` files, `tests/unit/<name>.test.ts`, `go test ./pkg/...`, `pytest tests/test_<name>.py`, plus `npm run check` / `npm test` / `go test ./...`), lets you pick several, runs them in order and records each result.
- Completion gates: choosing `done` interactively for a plan with unchecked `단계` / `Now` / `검증 계획` items, or for a goal (`Vibe Code: 목표 상태 변경`) with unchecked `완료 기준`, shows a modal listing them; overriding is audited (`completionGateOverridden`). Programmatic calls with an explicit status skip the gate.
- Stall detection: the goal status bar turns to the warning color (with the reason in the tooltip) when the active goal has not been updated for `vibe-code.goalStaleHours` (48) or the same first Now item has survived three sessions; activation also shows a one-time warning with `목표 열기` / `최신 계획 열기` and audits `goal/healthWarning`.
- Goal/plan lint: open `.vibe-code/goals/*.md` or `.vibe-code/plans/*.md` files get diagnostics for a missing title line, missing required sections, unknown `상태:` / `우선순위:` values, too many open Now items (goal > 4, plan > 3), `done` with unchecked items, and a missing `연결 목표:` line.
- Handoff on exit: when the extension deactivates with an active goal and audit events newer than the last handoff, a handoff file is written automatically (in addition to the journal session summary).
- Goal catalog: a new `목표 목록` activity-bar view lists `.vibe-code/goals/*.md` (current first, then by last update) and switches on click. `Vibe Code: 현재 목표 전환` saves current.md back into its own goal file (adding a `목표 파일:` line when missing) before loading another. `Vibe Code: 새 목표` asks for a title, a preset (기능 개발 / 버그 수정 / 리팩터링 / 문서화 / 자유 형식 — each with its own 완료 기준) and a description, creates the goal file, makes it current, and can start `/goal <title>` immediately.
- Plan → criteria link: `Vibe Code: 계획을 완료 기준과 연결` stores `연결 완료 기준: 1, 3` in the active plan; when that plan is set to `done` the listed 완료 기준 items in the goal file are ticked automatically.
- Metrics and retro: `Vibe Code: 목표 진행 지표 (7일)` aggregates the audit trail (completed items per day, verification pass rate and failing commands, approved/denied/failed commands, tasks with and without a goal update, tokens and USD cost from task completions, handoffs, stall warnings, plan status changes). `Vibe Code: 주간 회고 생성` writes `.vibe-code/retro/<date>-weekly.md` with those numbers, the vibe-coders weekly usage when available, completed items, repeated failures as bottlenecks, and next-week items from the goal's open Now / 완료 기준.
- Checkpoints: before an approved destructive command runs (`rm -rf`, `git reset --hard`, `git clean -f`, force push, `DROP TABLE`, `Remove-Item -Recurse`, `del /s`, `npm publish`, `kubectl delete`, `terraform destroy`, …) the working tree is snapshotted with `git stash create` and pinned under `refs/vibe-checkpoints/<stamp>`, and a note with restore commands is written to `.vibe-code/checkpoints/` (a note is still written outside git). `vibe-code.checkpointBeforeDestructive` (default on), `Vibe Code: 체크포인트 생성` and `체크포인트 목록`.
- Budget guard: `vibe-code.weeklyBudgetKrw` (0 = off). When the weekly vibe-coders cost exceeds it, `auto`/`yolo` autonomy is lowered to `assist` once per day, a warning is shown, the proxy status bar turns to the warning color with `예산 초과`, and `proxy/budgetExceeded` is audited.
- Auto handoff: when the core condenses a task's context, a handoff file is written to `.vibe-code/sessions/` and noted in the goal file.

## 1.3.0

### Build & CI

- Added `.github/workflows/ci.yml`: `npm run check` on every push/PR (ubuntu), then a Windows job that restores runtime assets from the latest GitHub release VSIX, packages, verifies, and uploads the VSIX with a SHA256 file.
- Added `scripts/restore-dist-assets.mjs`: extracts `extension/dist/**` (node_modules, i18n, workers, wasm) from a release VSIX into `dist/` with no dependencies, so a fresh clone can build a working extension.

### Features

- `Vibe Code: 시작 설정` (`vibe-code.setupWizard`): three-step QuickPick for provider (vibe-coders proxy / cloud API / local model), network mode (`offlineMode`) and autonomy level; writes the settings, applies the autonomy preset immediately, and hands off to the proxy apply command or the sidebar settings. The first-run welcome toast offers it.
- Plan file CodeLens (`.vibe-code/plans/*.md`): `완료로 이동` / `Next로 되돌리기` on Now items, `Now로 승격` on Next items, `체크`/`체크 해제` on other checklist lines, and `현재 계획으로 선택` on the title. Each action edits the open document, saves it, bumps `마지막 갱신:` and writes an audit entry.
- Command execution audit: the core now reports `execute_command` approvals, denials and shell exits through a third hook (`onCommand`, see `vendor/PATCHES.md`); they land in `.vibe-code/audit/<date>.jsonl` as `kind: "command"`, appear in the journal session summary, and `Vibe Code: 명령 실행 이력 보기` lists the last 30.
- vibe-coders usage: `Vibe Code: vibe-coders 사용량 리포트 보기` prints the weekly and monthly `/me/report` (requests, tokens, KRW cost vs prior window, success rate, latency, cache rate, top models, potential savings) and `Vibe Code: vibe-coders 사용량 대시보드` opens the same data as a webview panel (KPI tiles, single-hue cost bars with direct labels, table view, refresh). While the proxy route is active the `VC: ACTIVE` status bar item also shows the weekly cost and tokens (cached 5 minutes, skipped in `offlineMode = offline`).
- Session summary in the journal: on deactivate, today's audit events since the last summary (completed plan items, status changes, archives, handoffs, proxy apply/restore, failures) plus the goal's `다음 행동` and open Now items are appended to `.vibe-code/journal/<date>.md` under `## 세션 요약`.

### UX

- Goal Tracker, Plan Board and Goal Plan Map items now have stable ids, so expand/collapse state survives the periodic refresh.
- Tree views poll every 30s instead of 10s; file watchers remain the primary refresh signal.
- The mode status bar no longer sends a network probe every 15s: `vibe-code.offlineMode = offline|online` is asserted from settings without any request, and `auto` probes once a minute.
- The vibe-coders proxy status bar refreshes on settings changes and right after apply/restore, with a 60s fallback timer instead of 15s.

## 1.2.0

### Source Restored (소스 복원)

- Vibe Code's own features now live in `src/` as TypeScript and are built into `dist/extension.js` with esbuild (`npm run build`). The upstream core bundle is vendored as `vendor/extension.core.js`, staged to `dist/extension.core.js`, and only calls two hooks (`beforeCore`, `mergeLocaleOverrides`); see `vendor/PATCHES.md`.
- Added `npm run typecheck`, `npm run test` (vitest unit tests for the markdown/plan helpers), and `npm run check`.
- `package.json` no longer lists the upstream monorepo dependencies (`workspace:^` entries broke `npm install`); runtime modules keep shipping in `dist/node_modules`.
- Verification scripts read the version from `package.json` instead of hardcoding `1.0.0`, and check that `dist/extension.core.js` is present and in sync with `vendor/`.

### Fixes (수정)

- Plan `## Now` / `## Next` / `## Done` / `## 단계` / `## Risks` sections were never parsed (a broken `[\s\S]` escape matched only the letter `s`), so `현재 계획 진행` always reported "진행할 Now 항목이 없습니다" and the Plan Board showed empty sections. Section parsing is now a tested helper (`src/util/markdown.ts`).
- `vibe-coders` proxy commands threw a `ReferenceError` on their final audit call because the audit helper was block-scoped elsewhere; the apply command reported failure after actually applying the profile. All commands now share `writeAudit`.
- Multi-line `## 검증 로그` sections are read in full instead of only their first line.
- Goal Tracker / Plan Board / Goal Plan Map no longer pop "워크스페이스를 먼저 열어주세요" warnings on every refresh when no folder is open.
- The local-release update source path followed the old `vibe-code-1.0.0` folder name; it now points at `projects/vibe-code/release`.

---

## 1.1.0

### Plan Management (계획 관리)

- Added `Vibe Code: 최신 계획 열기` — opens or creates `.vibe-code/plans/current-plan.md` and shows plan details (단계/Now/Next/Done/Risks) in the output channel; records access in workspace audit JSONL.
- Added `Vibe Code: 현재 계획 진행` — moves the first `Now` item into `Done`, promotes the next queued step, and records the action in audit JSONL.
- Added `Vibe Code: 현재 계획 상태 변경` — updates `상태:` in the latest plan file via quick-pick and records the change in audit JSONL.
- Added `Vibe Code: 현재 계획 우선순위 변경` — stores `우선순위: P0..P3` in plan files; priority is shown in Plan Board and Goal Plan Map views.
- Added `Vibe Code: 현재 계획 목표 연결` — writes goal linkage metadata into the latest plan file and records it in audit JSONL.
- Added `Vibe Code: 완료 계획 보관` — moves `상태: done` plans into `.vibe-code/plans/archive/` and records the action in audit JSONL.
- Added `Vibe Code: 보관 계획 복원` — restores archived plan files back into active plan space.
- Added `Vibe Code: 현재 계획 선택` — maintains `.vibe-code/plans/.active-plan` as the current plan pointer; all plan commands follow this pointer.
- Added `Vibe Code: 최고 우선순위 계획 선택` — selects the highest-priority active plan (P0→P3), updates the active plan pointer, and records the action in audit JSONL.
- Added `Vibe Code: 계획 목록 보기` — prints active/archive plan summaries sorted by priority (P0→P3) to the Output channel.
- Added `Vibe Code: 계획 이력 보기` — shows recent plan audit events in the Output channel.
- Added `Vibe Code: 목표-계획 연결 보기` — prints the current goal and its linked active/archive plan files to the Output channel.

### New Views (신규 뷰)

- Added `목표 추적기` activity-bar tree view — tracks the current goal state.
- Added `계획 보드` activity-bar tree view — lists active plans with quick actions (advance, set status, set priority, link goal, archive, restore, catalog, history).
- Added `목표-계획 맵` activity-bar tree view — keeps the current goal's linked active/archive plans visible with quick actions; plans sorted by priority.

### Slash Commands

- Renamed `/목표` slash command to `/goal` for international compatibility; both forms are seeded.

### Docs

- Updated `docs/` with autonomous goal workflow, improvement roadmap, maintenance guide, and source analysis for v1.1.0.

---

## 1.0.0

- Rebranded the extension package, commands, views, settings, webview UI, and localization as `vibe-code`.
- Korean is now enforced as the first-run runtime default even when VS Code itself is using another UI language.
- Added `scripts/verify-package.ps1` plus npm helper scripts (`vsix`, `vsix:lean`, `verify`) for repeatable release checks.
- Added `scripts/smoke-vscode-cli.ps1` plus npm helper script `smoke:vscode` to install the VSIX into isolated VS Code directories and verify the installed extension.
- Added an Extension Host test entry plus `scripts/test-extension-host.ps1` and `npm run test:extension-host`; it uses `@vscode/test-electron` when available and falls back to the installed VS Code CLI.
- Added `vibe-coders` proxy routing commands. Vibe Code can now create/activate an OpenAI-compatible provider profile pointing at `http://localhost:8080/v1` so model calls pass through `vibe-coders` and are aggregated there. The extension does not start the gateway.
- Added `Vibe Code: 이전 프로바이더로 복원`, which re-activates the provider profile that was active before applying the `vibe-coders` proxy.
- Added `Vibe Code: vibe-coders 프록시 상태 보기`, which shows the current provider route and last proxy apply snapshot without making network or model calls.
- Hardened the Extension Host smoke test so it executes the no-network proxy status command and uses a per-process temporary directory.
- Added slash command `/goal` for persistent autonomous project development. It manages `.vibe-code/goals/current.md`, goal-specific logs, session handoff files, and verification records so long-running vibe coding can continue across sessions.
- Slash command seeding now backfills missing command files into existing `.vibe/commands/` directories without overwriting user-edited commands.
- Added `Vibe Code: 현재 목표 열기` and `Vibe Code: 목표 핸드오프 생성` for command-palette access to persistent goal state and session handoff creation.
- Added an active-goal status bar item that appears when `.vibe-code/goals/current.md` exists and opens the current goal on click.
- Added `Vibe Code: 목표 상태 보기` and checklist progress in the active-goal status bar.
- Added `Vibe Code: 최신 계획 열기` plus the `현재 계획` activity-bar tree view. The extension now opens or creates `.vibe-code/plans/current-plan.md`, shows the latest plan's 단계/Now/Next/Done/Risks, and records plan access in workspace audit JSONL.
- Added `Vibe Code: 현재 계획 진행` and `Vibe Code: 현재 계획 상태 변경`. The extension can now move the first `Now` item into `Done`, promote the next queued step, update `상태:` in the latest plan file, and record those actions in audit JSONL.
- Added `Vibe Code: 현재 계획 목표 연결` and `Vibe Code: 완료 계획 보관`. The extension now writes goal linkage metadata into the latest plan file, moves `상태: done` plans into `.vibe-code/plans/archive/`, exposes both actions in the Plan Board quick actions, and records them in workspace audit JSONL.
- Added `Vibe Code: 보관 계획 복원` and `Vibe Code: 현재 계획 선택`. The extension now restores archived plan files back into active plan space, maintains `.vibe-code/plans/.active-plan` as the current plan pointer, makes plan commands follow that pointer, and exposes both actions in the Plan Board quick actions.
- Added `Vibe Code: 계획 목록 보기` and `Vibe Code: 계획 이력 보기`. The extension now prints active/archive plan summaries and recent plan audit events to the Output channel, and exposes both actions in the Plan Board quick actions.
- Added `Vibe Code: 목표-계획 연결 보기`. The extension now prints the current goal and its linked active/archive plan files to the Output channel, and records `showGoalPlanMap` in workspace audit JSONL.
- Added the `목표-계획 맵` activity-bar tree view. The extension now keeps the current goal's linked active/archive plans visible in a dedicated view with quick actions for the map, current goal, and latest plan.
- Added `Vibe Code: 현재 계획 우선순위 변경`. The extension now stores `우선순위: P0..P3` in plan files, shows priority in the Plan Board and Goal Plan Map views, and includes priority in plan catalog / goal-plan map output.
- Improved plan ordering and linked-plan actions. `계획 목록 보기` and `목표-계획 맵` now sort plans by `P0 -> P3`, and clicking a linked plan in `목표-계획 맵` now selects it as the active plan or restores it from archive directly.
- Added `Vibe Code: 최고 우선순위 계획 선택`. The command selects the highest-priority active plan by `P0 -> P3`, updates `.vibe-code/plans/.active-plan`, opens the selected plan, and records `selectHighestPriorityPlan` in audit JSONL.
- Added maintenance documentation under `docs/` covering source analysis, release workflow, and improvement roadmap.
- Fixed VSIX packaging to include `readme.ko.md`, which is opened by the Korean first-run welcome action.
- Added a repeatable Windows VSIX packaging script.
- Prepared the VS Code release artifact layout for `release/vibe-code-1.0.0.vsix`.
- Default UI and assistant language set to Korean (`ko`). `fallbackLng` kept as `en` so missing keys do not blank out.
- New setting `vibe-code.language` (enum: `ko`, `en`) — synced into `globalState` on activation when no in-app value is set.
- System prompt's Language Preference section hardened: model MUST respond exclusively in the user's chosen language; code, paths, and identifiers stay original.
- Code-action prompts (Explain, Fix, Improve, Terminal Fix, Terminal Explain) tail-appended with explicit language directive.
- Workspace-level translation overrides: drop JSON files at `~/.vibe-code/locales/<lang>/<ns>.json` to deep-merge over the bundled translations at activation.
- `autoCondenseContextPercent` default lowered to 75 (from 90) to better fit Korean's higher token-per-character ratio.
- Filled missing `settings.providers.*` keys in `dist/i18n/locales/en/common.json` so English users no longer see blank labels for Groq and Claude Code settings.

### Phase 2 — UX / Ops

- New setting `vibe-code.tone` (formal / casual / terse) — applied to Korean responses via Language Preference suffix.
- New setting `vibe-code.autonomy` (safe / assist / auto / yolo) — auto-applies always-allow flags on activation.
- New command `Vibe Code: Show Context Stats` — dumps current mode, language, autonomy, telemetry, and token usage to output channel.
- New command `Vibe Code: Open Today's Journal` — opens `.vibe-code/journal/YYYY-MM-DD.md` for the current day.
- Status bar item showing current mode and accumulated tokens; click opens the sidebar.
- Editor context menu submenu extended to include `Fix Code` and `New Task` alongside the existing entries.
- 4 Korean custom modes seeded to `.vibemodes` on first workspace activation: 한국어 코드 리뷰어, 한국어 학습 동반자, TDD 동반자, 한국어 아키텍트.
- 7 Korean slash commands seeded to `.vibe/commands/`: `/설명`, `/수정`, `/개선`, `/검토`, `/테스트`, `/한국어주석`, `/병렬`.
- `.vibeignore` template seeded with Korean privacy-sensitive patterns (주민번호 파일, 개인정보 CSV, 회원/고객 정보, secrets, cloud credentials).
- Telemetry defaults to disabled until the user explicitly opts in.
- Daily auto-check for newer VSIX in `release/` — prompts install when found.
- System prompt OBJECTIVE section now includes a SELF-RECOVERY block (5 failure categories, no silent retries, root-cause-first, preserve user work).
- Demo scenarios seeded to `.vibe-code/demo-scenarios.md` for first-run onboarding.
- Korean user guide: `readme.ko.md`.

### Build pipeline

- `scripts/package-vsix.ps1` slims `dist/node_modules` (drops .md, .d.ts, .ts, .map, test/spec/docs directories).
- Added `-Lean` switch — removes rarely-used tree-sitter WASMs to further reduce VSIX size.

### Phase 3 — Safety / productivity / UX

- `vibe-code.deniedCommands` default expanded with 23 destructive patterns (rm -rf /, force push, drop table, curl | sh, fork bomb, etc.).
- System prompt extended with three new sections: DESTRUCTIVE COMMAND POLICY (explicit confirm for irreversible ops), SECRET HANDLING (API key / Korean PII / private key redaction), COST-AWARE WORK STYLE (match effort to task scope).
- 3 new Korean slash commands seeded: `/커밋` (한국어 conventional commit), `/PR` (PR 제목+본문 자동 작성), `/계획` (큰 작업 계획서 .vibe-code/plans/ 자동 저장).
- 8 default keybindings: Ctrl+; (sidebar), Ctrl+Shift+L (add to context), Ctrl+Alt+E/F/I (explain/fix/improve), Ctrl+Alt+N (new task), Ctrl+Alt+S (stats), Ctrl+Alt+J (journal).
- First-run welcome notification in Korean with "Open Sidebar" / "Open Guide" actions.
- New command `Vibe Code: 임베딩 인덱스 정보` — shows globalStorage size and workspace vector DB status.
- New command `Vibe Code: 환경 점검` — single-screen diagnostic of all Korean-related settings + seeded files + override detection.
- Workspace activation now also creates `.vibe-code/plans/` for the plan-mode workflow.

### Phase 4 — Git / MCP / templates / KST / team / verification

- 6 more Korean slash commands seeded: `/훅` (git hooks 설치), `/CRUD` (엔티티 보일러플레이트), `/API` (REST 엔드포인트), `/리팩터` (안전 리팩터링), `/컨벤션` (스타일 자동 감지 → `.vibe/rules-code.md`), bringing total to **15 slash commands**.
- `.vibe-code/mcp-recommendations.json` seeded with 9 curated MCP server entries (filesystem, github, memory, fetch, puppeteer, sqlite, postgres, slack, time).
- KST timezone applied to journal filenames and timestamps via `Intl.DateTimeFormat` with `Asia/Seoul`.
- System prompt gained TIMEZONE block — interprets Korean user's unqualified times as KST, surfaces converted timestamps.
- System prompt gained POST-CHANGE VERIFICATION block — typecheck, lint, focused tests after non-trivial edits; no suppression to make checks pass; explicit when checks aren't possible.
- New commands `Vibe Code: 팀 설정 내보내기` / `가져오기` — package `.vibemodes`, `.vibeignore`, `.vibe/commands/`, `mcp-recommendations.json` into a zip for team sharing (uses adm-zip if available, falls back to PowerShell Compress-Archive).
- Korean README extended with font (D2Coding, Pretendard Coding, Sarasa Mono K, JetBrains Mono + Noto Sans KR), IME, word-separator, and ruler-width recommendations.

### Phase 5 — Offline / air-gapped network operation

- System prompt gained OFFLINE-FIRST OPERATION block — bans `npm install`, `npx <unknown>`, public-URL `git clone`, WebFetch to public sites; tells the agent to search local sources first; explicit "(인터넷 필요)" tagging required when proposing online commands.
- Status bar item now performs a 2s HEAD probe every 15s; icon switches between `$(cloud)` (online) and `$(cloud-offline)` (offline). Tooltip shows network state in Korean.
- 4 new settings: `vibe-code.offlineMode` (auto/online/offline), `vibe-code.proxyUrl`, `vibe-code.extraCaCertsPath`, `vibe-code.updateSourcePath`.
- Activation applies proxy/CA env vars to `process.env` so MCP servers and child processes inherit them: `HTTPS_PROXY`, `HTTP_PROXY`, `NODE_EXTRA_CA_CERTS`.
- `Vibe Code: 외부 통신 점검` command — probes 1.1.1.1 / public LLM APIs / npm registry / GitHub / HuggingFace with HEAD requests; reports current API provider, base URL, and any MCP server that depends on `npx` (flagged as inter-net-requiring).
- `mcp-recommendations.json` rewritten with offline-safe entries (filesystem, memory, sqlite, time, git) explicitly marked `_offline_safe: true`, plus install-from-tarball guide for npx-based servers in air-gapped envs.
- New slash command `/문서` — searches local README/docs/wiki/journal/plans/embeddings only, never touches the public web.
- Auto-update check now consults `vibe-code.updateSourcePath` first (UNC paths supported) before falling back to local `release/` folder.
- PR slash command updated to support GitHub Enterprise (`GH_HOST=...`) and to fall back to "copy PR body" output when no CLI is available.
- Hook slash command branches on internet availability (husky vs raw `.git/hooks/`).
- Korean README extended with Offline / air-gapped operation guide: local LLM options (Ollama, LM Studio, vLLM, llama.cpp, internal gateway), proxy/cert settings, per-feature internet-dependency matrix, internal update-source setup.

### Phase 6 — License removal + Roo-Code parity

- `expiredDate` getter now returns empty string and `invalidLicense` returns false unconditionally — time-based licence expiration is fully removed.
- All `[VibeCode#ask] Invalid License` throw sites stripped from `ask`/`say` paths.
- System prompt gained WRITE-PROTECTED PATHS block — explicit guard on `.vscode/`, `.git/`, lock files, `.env*`, CI/CD configs, build artifacts, and the extension's own state directories.
- System prompt gained NATIVE TOOL CALLING block — prefer native function calling on supported models, batch independent reads in parallel, fall back to XML on local/older models.
- Default mode for fresh installations now `architect` (was `code`), reflecting Roo-Code v3.22.6 onward and matching the "think-first" workflow.
- Status bar tooltip extended with vibe-code version and applied autonomy preset alongside the mode and network state.
- README extended with timer-based follow-up auto-approve, default-mode-architect, and parallel native tool calls tips (parity with Roo-Code v3.22.x and v3.34.x features).
