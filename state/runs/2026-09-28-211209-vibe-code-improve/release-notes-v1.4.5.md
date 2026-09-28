## 1.4.5 — CRLF 파일에서 검증 로그가 중복 섹션을 만들지 않습니다

A fix release for the verification log. `src/features/verification.ts` still carried its own copy of
the `## 검증 로그` section regex — the one place the 1.4.3 CRLF consolidation did not reach — and it
cannot match a CRLF heading. So on a CRLF `.vibe-code/*.md` file every verification run appended a
*new* `## 검증 로그` section to the end of the file instead of adding a line to the existing one, and
because the goal status reads only the first section of that name, the result of a run appeared never
to update. CRLF is what you get from `core.autocrlf` on Windows or from a zip round-trip, both of
which `docs/autonomous-goal-workflow.md` treats as supported.

### Fixes (수정)

- **A verification run no longer adds a duplicate `## 검증 로그` section.** `appendVerificationLog`
  tested for the section with its own `/(?:^|\n)## 검증 로그\n/`, which allows exactly one `\n` after
  the heading and therefore never matches `## 검증 로그\r\n`. On a CRLF file the existing section was
  never found, so each run fell through to the "create the section" path and appended another one:
  three verifications left three sections. The check is now the shared `hasSection()`, so one section
  is found and appended to whatever the file's line endings are.
- **`최근 검증` in the goal status shows the latest runs again.** `goals.ts` reads the log through
  `section(text, "검증 로그")`, which returns the *first* section of that name — and the first one was
  the stale original, while every new record went into a duplicate further down the file. Nothing was
  wrong at that call site; it was reading a file the writer had broken. With the duplicates gone it
  reports the most recent runs.
- **Ticking a verification checklist item no longer rewrites the whole file's line endings.**
  `checkLine` split on `/\r?\n/` and rejoined with `\n`, so checking one box in a CRLF file converted
  every line in it to LF and turned a one-line change into a whole-file diff. It now remembers the
  input's line ending with `detectEol` and restores it afterwards — the same rule `writeSection`,
  `toggleCheckbox`, `touchPlan` and `setLine` have followed since 1.4.3.
- **Notes and indentation already in the log section survive an append.** The existing records were
  re-read with `lines()`, which drops blank lines and trims each one, so an indented output line
  (`  출력: 78 tests passed`) or a blank line inside the section was flattened or deleted as soon as
  the next record was appended. The section is now read with `sectionLines()`, exactly as written.

### Internal (내부)

- `restoreEol` is exported from `src/util/markdown.ts` — a one-word change, the function itself is
  unchanged — so `verification.ts` can restore line endings through the shared helper instead of a
  second copy of it. With `hasSection`, `sectionLines`, `detectEol`, `normalizeEol` and `restoreEol`
  all coming from `markdown.ts`, `verification.ts` no longer parses sections on its own; there is one
  section parser again. Two production files changed.
- Unchanged behaviour worth stating: `writeSection` and `toggleCheckbox` restore line endings
  themselves, so `restoreEol` is applied only on the new-section path and in `checkLine`, never twice
  (applying it to un-normalised text would produce `\r\r\n`). `runAndRecord` depends on `vscode` and
  is untouched and still uncovered.

### Tests (테스트)

- `tests/unit/verification.test.ts` gains a `verification log on CRLF files` block (4 tests). They
  apply `appendVerificationLog` three times in a row to a real CRLF file — the production
  `goalTemplate` / `planTemplate` output with `\n` replaced by `\r\n`, not a hand-assembled fixture —
  and read the result back through the production `sectionLines()` and `normalizeEol()`, with no fake
  `fs` and no fake `vscode`. Covered: exactly one `## 검증 로그` section in a CRLF goal file and in a
  CRLF plan file after repeated appends, with all three records present and in order; CRLF preserved
  in the output for both templates and by `checkLine`; and indented and blank lines already in the
  section kept. All four were confirmed failing against the previous code — `expected 3 to be 1`
  (goal file, three sections), `expected 2 to be 1` (plan file), `expected false to be true` (a
  non-CRLF line in the output) and the lost indentation on `  출력: 78 tests passed` — before the fix
  landed. `npm run check` is now 82 tests (78 + 4).
