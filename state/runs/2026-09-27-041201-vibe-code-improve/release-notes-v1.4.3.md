## 1.4.3 — CRLF 계획·목표 파일에서 섹션 파서와 편집이 동작합니다

A fix release for `.vibe-code/*.md` files with CRLF line endings. The section parser rejected them
outright, so on Windows (`core.autocrlf`) or after a zip round-trip — both of which
`docs/autonomous-goal-workflow.md` treats as supported — the plan CodeLens did nothing, silently,
and lint warned about sections that were right there in the file.

### Fixes (수정)

- **CRLF plan and goal files are readable again.** The section regexes in `src/util/markdown.ts`
  allowed exactly one `\n` after a `## <이름>` heading, so on a CRLF file every section-shaped
  operation failed at once: `section()` and `subsection()` returned the empty string,
  `writeSection()` returned its input untouched, and `moveTaskToSection()` returned `null` — which
  is why the plan CodeLens `완료로 이동` / `Next로 되돌리기` did nothing at all, with no error
  surfaced anywhere. The reading functions now normalise CRLF to LF at entry (`normalizeEol`) before
  matching, so the same file reads the same either way.
- **Editing a plan no longer rewrites the whole file's line endings.** `writeSection`,
  `moveTaskToSection`, `toggleCheckbox`, `touchPlan` and `setLine` remember the input's line ending
  with `detectEol` (a mixed file follows its first line break, the same rule as VS Code's
  `TextDocument.eol`) and restore it after editing. Ticking one checkbox in a CRLF file now produces
  a one-line diff instead of converting every line in the file to LF.
- **Lint stops reporting sections that are present.** `goal-lint.ts` carried its own copy of the
  section regex with the same `\n` assumption, so a CRLF plan was flagged with
  `## 단계 섹션이 없습니다` and one warning per required section. The existence check is now the
  single shared `hasSection()`, so the parser and the lint cannot disagree about whether a section
  exists.

### Internal (내부)

- `detectEol`, `normalizeEol` and `hasSection` are exported from `src/util/markdown.ts`, and the two
  duplicated section regexes in `src/features/goal-lint.ts` are gone — there is one section parser
  again. Two production files changed.

### Tests (테스트)

- `tests/unit/markdown.test.ts` gains a `CRLF 줄바꿈` block (7 tests): EOL detection on mixed input,
  reading functions returning the same values under CRLF as under LF, `writeSection` and
  `moveTaskToSection` producing the same edit with line endings preserved,
  `toggleCheckbox`/`touchPlan`/`setLine` preserving them too, and — through the production
  `planTemplate` / `goalTemplate` rather than hand-written fixtures — `lintPlan`, `lintGoal`,
  `parseGoal` and `openPlanItems` reading CRLF files identically, with no missing-section warnings.
  Six of them were confirmed failing against the previous code
  (`expected '' to be '- [ ] A\n- [ ] B'`, `expected null not to be null`, the LF-converted
  `toggleCheckbox` output, and the spurious lint warnings) before the fix landed. `npm run check` is
  now 70 tests (63 + 7).
