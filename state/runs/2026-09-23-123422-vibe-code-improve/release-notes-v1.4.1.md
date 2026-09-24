## 1.4.1 — 체크포인트가 미추적 파일까지 담습니다

A fix release for the destructive-command checkpoint introduced in 1.4.0. The safety net did not
cover the case it most needed to: a file you had just created and not yet `git add`ed.

### Fixes (수정)

- **Untracked files are now in the snapshot.** `createCheckpoint` used `git stash create`, which by
  design never includes untracked files — so a brand-new file was *not* protected when an approved
  destructive command (`rm -rf`, `git clean -f`, …) ran. The snapshot is now built through a
  throwaway index (`GIT_INDEX_FILE` under `os.tmpdir()`, then `read-tree` → `add -A` → `write-tree`
  → `commit-tree`), so uncommitted changes to tracked files and untracked files alike (ignored files
  still excluded) are pinned under `refs/vibe-checkpoints/<stamp>`. Your working tree, index and
  stash stack are left untouched.
- **Restore commands in the note actually work.** The note told you to run `git stash apply`, which
  fails — a snapshot commit is not in stash format and git rejects it. Notes now carry
  `git diff <commit>` to review and `git checkout <commit> -- .` to restore.
- A repository with **no commits yet** is snapshotted as a parentless commit instead of only leaving
  a note.
- `vibe-code.checkpointBeforeDestructive` descriptions (en/ko) and `docs/autonomous-goal-workflow.md`
  now state the tracked/untracked behaviour instead of the vaguer "working tree snapshot".

### Tests (테스트)

- `tests/unit/checkpoints.test.ts` (9 tests) — the checkpoint feature shipped in 1.4.0 with no
  coverage. These run against **real temporary git repositories** (`fs.mkdtempSync`), not a fake
  git, and the restore test parses the commands out of the generated note and executes them to prove
  the files come back. Covered: untracked files in the snapshot, working tree / index / stash stack
  left untouched, restore, empty repository, non-git directory, and the `isDestructiveCommand` table.
  Four failed against the 1.4.0 code for the reasons above before the fix landed.
- `npm run check` is now 53 tests (typecheck + vitest + esbuild build).

### Upgrading

No configuration changes. Checkpoints created by 1.4.0 remain readable, but their notes contain the
`git stash apply` command that does not work — use `git checkout <commit> -- .` against the pinned
ref instead. Checkpoints created by 1.4.1 carry the corrected commands.

### Known limitation

`checkpointBeforeCommand` is synchronous and now walks untracked files as well, so a large directory
that is not covered by `.gitignore` (an uncommitted `node_modules`, for example) can make the
pre-command snapshot slower. Not yet measured or guarded.
