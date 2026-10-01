## 1.4.6 — 예산 초과 경고가 KST 하루에 한 번만 발동합니다

A fix release for the weekly budget guard. `enforceBudget` in `src/features/usage.ts` keyed its
"already notified today" check on `new Date().toISOString().slice(0, 10)` — a *UTC* date — and the
UTC date rolls over at 09:00 KST. One KST day therefore spans two UTC dates, so a user warned at
08:00 KST was warned *again* at 10:00 KST, autonomy was lowered to `assist` a second time, and one
KST audit file collected two `budgetExceeded` lines. The "once per day" that
`vibe-code.weeklyBudgetKrw` and `docs/autonomous-goal-workflow.md` promise was broken for every
exceeded budget whose first notice landed before 09:00. Every other day boundary in this repository —
the audit filenames, the journal and retro filenames, the goal-metrics windows — is already KST; this
was the last one that was not.

### Fixes (수정)

- The weekly budget warning could fire twice on one Korean calendar day. `enforceBudget` in `src/features/usage.ts` keyed its "already notified today" check on `new Date().toISOString().slice(0, 10)` — a *UTC* date, and the UTC date rolls over at 09:00 KST. One KST day therefore spans two UTC dates, so a user warned at 08:00 KST was warned *again* at 10:00 KST: the stored `vibeCode.budgetNotifiedDate` no longer matched, `writeAutonomyPreset(host, "assist", true)` lowered autonomy a second time, and one KST audit file collected two `budgetExceeded` lines. The "once per day" that `vibe-code.weeklyBudgetKrw` promises in `package.nls.json` / `package.nls.ko.json` and in `docs/autonomous-goal-workflow.md` was broken for every exceeded budget whose first notice landed before 09:00. The notice day is now `kstDate()`, so one KST day is one key and the notice is once per day as described.
- The notice, the autonomy downgrade and the audit trail now agree on where a day ends. `writeAudit` names `.vibe-code/audit/<KST 날짜>.jsonl`, so the audit file was already per-KST-day while the notice writing into it was per-UTC-day — a reader of that file could not reconcile the two. All three now share the one boundary.

### Internal (내부)

- The day key is extracted into an exported, vscode-free `budgetNoticeDay(now = new Date())` in `src/features/usage.ts` whose body is `kstDate(now)` from `src/util/kst.ts` — the same helper the audit filenames (`workspace.ts`), the journal and retro filenames and the goal-metrics windows already use. `src/util/kst.ts` uses only `Intl.DateTimeFormat` and imports no `vscode`, so the test graph stays clean. This was the last `toISOString().slice(0, 10)` in `src/`.
- Unchanged behaviour worth stating: `enforceBudget`'s control flow, its warning text, the `writeAudit` fields and the `vibeCode.budgetNotifiedDate` globalState key are all untouched, and nothing else reads that key. A key written by 1.4.5 under a UTC date can fail to match once across the upgrade — at most one extra notice on the first exceeded budget after updating, and nothing after that; the stored UTC date is never *ahead* of the KST date, so no notice is suppressed. One production file changed.

### Tests (테스트)

- `tests/unit/usage.test.ts` gains a `budget notice day key` block (3 tests). They pass fixed `Date` instances for four boundary moments of one KST day — 00:30, 08:00, 10:00 and 23:30 KST, straddling the 09:00 KST UTC rollover — and pin that all four produce a single key, and that the key equals `kstDate()` at each of them, which is the rule the audit filename follows. All three were confirmed failing first, with the key extracted but still computed as `toISOString().slice(0, 10)`: `expected '2026-09-30' to be '2026-10-01'`. `npm run check` is now 85 tests (82 + 3).
- Scope worth stating: these pin the day-key contract, not the whole notification path. `enforceBudget` itself needs `config()`, `host.context.globalState`, `vscode.window.showWarningMessage` and `writeAutonomyPreset`, none of which the shared `tests/unit/vscode-stub.ts` provides, so the twice-in-one-day firing is established by the key comparison rather than reproduced end to end.
