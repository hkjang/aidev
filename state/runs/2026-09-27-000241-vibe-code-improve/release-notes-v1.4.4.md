## 1.4.4 — 계획 한 단계 진행이 Now/Next 의 메모 줄과 들여쓰기를 지우지 않습니다

A fix release for `vibe-code.advanceCurrentPlan`. Advancing a plan one step rewrote `## Now` and
`## Next` from a filtered, trimmed copy of themselves, so anything in those sections that was not a
top-level checklist line — a note you had typed under Now, an indented sub-item — was deleted.
This command is not covered by the destructive-command checkpoint added in 1.4.0, so there was
nothing to restore from.

### Fixes (수정)

- **Hand-written notes and indentation in Now/Next survive an advance.** `vibe-code.advanceCurrentPlan`
  read both sections through `taskLines(section(...))` — checklist lines only, and trimmed at that —
  then wrote the result back with `writeSection`, which overwrites the whole section body. A single
  `계획 한 단계 진행` was therefore enough to lose a `참고: …` note line and to flatten
  `  - [ ] 하위 항목` into `- [ ] 하위 항목`. Both sections are now read exactly as written and only
  the one item being moved is spliced out, so every other line — notes, indentation, blank lines
  inside the section — comes back unchanged.
- **The plan CodeLens `완료로 이동` no longer flattens `## Done`.** `moveTaskToSection` rebuilt the
  *target* section from `lines(section(...))`, so moving a task into Done stripped the indentation
  from everything already recorded there. The target section is now reassembled from its original
  lines.

### Internal (내부)

- `sectionLines(text, name)` is added to `src/util/markdown.ts`: the lines of a `## <name>` section
  exactly as written, with an empty or missing section returning `[]` rather than `[""]`. It is the
  reading half `writeSection` was missing — editing a section no longer means filtering it first.
- The advance step is extracted from the command callback into an exported, vscode-free
  `advancePlanText(text, stamp)` returning `{ text, completed, promoted }`, or `null` when Now has no
  checklist line — the same pattern `archiveDonePlans` and `createCheckpoint` follow, which is what
  makes it testable against real plan text instead of a stubbed `vscode` surface. The audit entry
  takes `completed` / `promoted` from the return value.
- Unchanged behaviour worth stating: a Next item promoted into Now is still `trim()`ed, and the very
  first and last blank lines of a section body are still dropped by `section()`. Two production files
  changed.

### Tests (테스트)

- `tests/unit/plans.test.ts` gains an `advancePlanText` block (6 tests) and
  `tests/unit/markdown.test.ts` gains `sectionLines` and `moveTaskToSection` cases (2 tests). They
  run against a plan string built by the production `planTemplate(...)` — with notes and indented
  lines added through the production `writeSection` — rather than hand-written fixtures, and there is
  no fake `fs` and no fake `vscode`. Covered: non-checklist and indented lines in Now/Next surviving
  an advance, the first Now item landing at the end of Done with 상태/마지막 갱신 bumped, existing
  indented Done records kept, `null` when Now has no checklist line, an empty Next promoting nothing,
  and the same result on a CRLF plan with line endings preserved. Three of them were confirmed
  failing against the previous code — the lost `참고: Now 메모` line and the flattened Done
  indentation — before the fix landed. `npm run check` is now 78 tests (70 + 8).
