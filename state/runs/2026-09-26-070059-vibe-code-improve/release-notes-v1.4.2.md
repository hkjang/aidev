## 1.4.2 — 완료 계획 보관이 과거 기록을 지우지 않습니다

A fix release for plan archiving. `Vibe Code: 완료 계획 보관` was a destructive operation on its own
history: archiving a plan whose name was already taken in `archive/` deleted the older copy first.

### Fixes (수정)

- **An archived plan of the same name is no longer destroyed.** `vibe-code.archiveDonePlans` moves
  every `상태: done` plan into `.vibe-code/plans/archive/`, and on a name collision it ran
  `fs.unlinkSync(dest)` on the existing archived copy before renaming the new one over it. Repeating
  the ordinary "계획 → done → 보관" loop was therefore enough to erase the earlier plan's record,
  with no trash can to recover it from. A collision now preserves both: the incoming file is archived
  as `<이름>-<stamp>.md` (and `<이름>-<stamp>-<n>.md` if that is taken too) and the existing archived
  plan is left untouched. This also resolves a policy contradiction with `restoreArchivedPlan`, which
  already renamed rather than overwrote on collision.
- **Archiving and restoring stamp names the same way.** Both paths now take the collision suffix from
  one shared `conflictStamp()` — `YYYYMMDDHHMMSS` in Asia/Seoul. `restoreArchivedPlan` previously
  derived its stamp from `toISOString()`, i.e. UTC.
- **The notification and the audit trail name the files that exist.** The archive message and the
  `archiveDonePlans` audit entry report the destination names files were actually written under,
  rather than the names they had in `plans/`.
- `docs/autonomous-goal-workflow.md` states the collision behaviour.

### Internal (내부)

- The archiving loop is extracted from the command handler into an exported, vscode-free
  `archiveDonePlans(plansDir, archiveDir, stamp)` returning `{ name, dest }` per plan, alongside
  `freeName()` and `conflictStamp()`. This follows the `checkpoints.ts` precedent and is what makes
  the behaviour testable without growing the test stub for `vscode.commands`.

### Tests (테스트)

- `tests/unit/plans.test.ts` (10 tests): `plans.ts` had no coverage at all. The tests run against
  real directory trees created with `fs.mkdtempSync` — no fake `fs` and no fake vscode command
  surface. Covered: the same-name archive collision (both copies survive, existing content intact),
  only `상태: done` plans move, the `archive/` subdirectory is not descended into, `freeName`
  fallbacks, `conflictStamp` format, `selectPlanName` fallback, and `openPlanItems`. The two
  collision tests were confirmed failing against the previous code — the existing archived plan's
  content disappeared — before the fix landed. `npm run check` is now 63 tests (53 + 10).

### 검증 (Verification)

`npm run check` passes on this tag: typecheck, 63 unit tests across 11 files, esbuild build, plus
`node --check` on `dist/extension.js` and `dist/extension.core.js`.

No VSIX asset is attached to this release, as with 1.4.1. Packaging runs through
`scripts/package-vsix.ps1` on Windows and needs the runtime assets (`node_modules`, `i18n`,
`workers`, `*.wasm`) restored from an earlier release VSIX; install from source with
`npm run check` until a packaged build is published.
