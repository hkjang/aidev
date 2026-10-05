## 1.4.7 — `마지막 갱신` 의 명시적 UTC/오프셋 표기를 KST 로 오독하지 않습니다

A fix release for the goal age calculation. The `마지막 갱신:` line is read by two thresholds — the
stalled-goal warning (`vibe-code.goalStaleHours`, 48 hours by default) and auto-resume (72 hours by
default) — and each had its own character-for-character copy of the same regex and the same
KST-fixed arithmetic (`+m[4] - 9`), one exported from `src/features/goal-health.ts` and one kept
privately in `src/features/goal-loop.ts` with no test of its own. That shared arithmetic assumed the
stamp was *always* KST, so `마지막 갱신: 2026-10-05T03:00:00Z` — a stamp three hours old — was
computed as twelve hours old, and a `-04:00` stamp was off by thirteen. This is a path that actually
receives such values: `assets/demo/.vibe/commands/goal.md` instructs the AI to write that line as
`<ISO 또는 로컬 시각>`, and ISO with `Z` is exactly what that produces.

### Fixes (수정)

- **A `마지막 갱신:` stamp with an explicit `Z` or `±HH:MM` zone is read in that zone.** The nine-hour
  overestimate on UTC stamps and the thirteen-hour error on `-04:00` are gone. A stamp without a zone
  is still read as KST, which is the format `kstStamp().human` writes, so nothing about the normal
  in-product stamp changes.
- **The two thresholds that read that one line now read it the same way.** Two copies of one rule is
  two places a correction has to land, and only one of them was reachable from a test.

### Internal (내부)

- The age calculation is now one exported pure function, `stampAgeHours(stamp, now)` in
  `src/util/kst.ts` — no `vscode` import, so it is testable directly. It matches the stamp with one
  regex and checks for a trailing `Z`/`±HH:MM` with a second one, using the offset when present and
  9 hours when not. The whitespace is a single-space class, never `\s*`.
- `goal-loop.ts`'s second definition is deleted and it imports the `goal-health` export instead,
  which keeps `goal-loop` out of the existing `goals`↔`goal-health` cycle.
- Deliberately unchanged: `goalAgeHours(text, now?)` keeps its name, signature and export, and a
  value with no time or with a date only still returns `Number.POSITIVE_INFINITY` — so a date-only
  `마지막 갱신` still silences both the warning and auto-resume. Interpreting such a value as midnight
  would create a warning that does not fire today, which is a contract decision rather than a fix.
  Three production files changed.

### Tests (테스트)

- `tests/unit/goal-age.test.ts` is new (5 tests) and passes a fixed `now` to the pure boundary: `Z`,
  `+09:00` and a bare KST stamp for one instant agree; `-04:00` is read as `-04:00`; the
  `kstStamp().human` format and the `Infinity` cases are pinned. The two zone-sensitive ones were
  confirmed failing first, with the body extracted into `stampAgeHours` and both call sites wired but
  the arithmetic still KST-fixed — `expected [ 18, 9, 9 ] to deeply equal [ 9, 9, 9 ]` and
  `expected 22 to be 9`, value mismatches rather than missing symbols. The other three pin behaviour
  that was already correct and passed throughout. Reverting only the offset line afterwards
  reproduced the same two failures with the same messages. `npm run check` is now 90 tests (85 + 5).
- Scope worth stating: this pins the age arithmetic, not the surrounding flow. `assessGoalHealth` and
  `scheduleAutoResume` need `config()`, `vscode.window` and `workspaceState`, which the shared
  `tests/unit/vscode-stub.ts` does not provide, so "the warning does or does not appear" is not
  reproduced end to end. That `goal-loop` now uses the shared value is guaranteed by the import and
  the type check rather than by executing `scheduleAutoResume`.

### Verification (검증)

- `npm run check` on Node 22.23.1: typecheck + **12 files / 90 tests** + esbuild build, all passing;
  `node --check dist/extension.js` and `node --check dist/extension.core.js` both exit 0. The build
  leaves its usual `dist/ is missing runtime assets` warning, which is expected in a fresh clone.
- Not verified here: Windows packaging (`npm run vsix` / `verify` / `smoke:vscode`) needs PowerShell
  and runtime assets restored from a published VSIX, neither available in this environment. A passing
  build is not a passing package.
