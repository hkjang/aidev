## 2026-09-23
- 선택: 파괴적 명령 전 체크포인트가 untracked 파일까지 담도록 고치고 첫 테스트를 붙이기 (가치 5 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `createCheckpoint`의 `git stash create`(추적 변경만 담김)를 `os.tmpdir()` 아래 임시 index 기반 스냅샷(`read-tree`→`add -A`→`write-tree`→`commit-tree`)으로 교체해, 아직 `git add` 하지 않은 새 파일도 `rm -rf` 직전에 보호되게 했다. 스냅샷 커밋은 stash 형식이 아니므로 노트의 복원 블록을 `git diff`/`git checkout <commit> -- .` 로 바꾸고, 커밋이 없는 저장소는 부모 없는 커밋으로 처리했으며, 설정 설명(ko/en)과 `docs/autonomous-goal-workflow.md` 문구를 정정했다. 검증은 `fs.mkdtempSync`로 만든 **진짜 git 저장소**를 통과하는 신규 `tests/unit/checkpoints.test.ts` 9개(가짜 git 대역 없음 — 복원 블록은 노트에서 명령을 파싱해 실제로 실행해 파일 복구를 확인)로 했고, 고치기 전 4개가 예상대로 실패(untracked 누락·`git stash apply` 잔존·빈 저장소)하는 것을 확인한 뒤 고쳤다. `npm run check` 전체 통과(53 tests, typecheck, esbuild 빌드).
- 보류 아이디어: plans.ts(523줄, 테스트 0개) 순수 함수 테스트 보강 — selectPlanName/planMeta/sortByPriority/linkedPlans/openPlanItems (3/1/M) · computeGoalMetrics가 `details.exitCode` 없는(undefined) 검증 항목을 통과로 집계 — 실제 감사 로그에 그런 줄이 나오는지 먼저 확인 필요 (2/1/S) · vibe-coders-proxy.ts provider 저장/복원 경로 테스트 — vscode 설정 API 스텁 확장이 필요해 대역 위주가 되기 쉬움 (3/2/M) · `createCheckpoint`의 catch가 `update-ref` 실패 시에도 "git 저장소가 아님" 노트를 남겨 commit/note가 어긋남 (2/1/S) · 거대한 untracked 디렉터리(.gitignore 미등록 node_modules 등)가 있을 때 동기 `git add -A` 지연 측정과 가드 (2/2/M)
- 과제서: 채택 — 과제서의 근거(stash create가 untracked를 제외, stash apply가 비-stash 커밋 거부)를 실제 git 2.43으로 재현해 확인한 뒤 그대로 구현했다.

- 릴리즈: v1.4.1 (2026-09-23, run 2026-09-23-123422-vibe-code-improve)
## 2026-09-26
- 선택: archiveDonePlans 가 같은 이름의 보관 계획을 말없이 지우는 것을 막고 보관 로직을 테스트 가능한 함수로 분리 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `vibe-code.archiveDonePlans` 는 `archive/<이름>` 이 이미 있으면 `fs.unlinkSync(dest)` 로 기존 보관본을 지운 뒤 덮어써서, "계획 → done → 보관" 루프만 반복해도 과거 계획 기록이 휴지통 없이 사라졌다. 보관 루프를 vscode 에 의존하지 않는 `archiveDonePlans(plansDir, archiveDir, stamp)` 로 분리하고 충돌 시 `restoreArchivedPlan` 과 같은 방식(`path.parse` + 14자리 숫자 스탬프)으로 `<이름>-<stamp>.md` 에 보관하도록 바꿨으며, 스탬프 생성을 `conflictStamp()`(KST) 로 통일해 두 경로가 같은 형식을 쓰게 했다. 검증은 `fs.mkdtempSync` 로 만든 실제 디렉터리 트리에서 도는 신규 `tests/unit/plans.test.ts` 10개로 했고, 고치기 전에 "같은 이름 두 번 보관" 테스트 2개가 기존 보관본 내용이 사라지는 형태로 실패하는 것을 먼저 확인한 뒤 고쳤다(가짜 fs·가짜 vscode 명령 대역 없음). `npm run check` 전체 통과 — typecheck + 63 tests(기준선 53 + 신규 10) + esbuild 빌드.
- 보류 아이디어: vibe-coders-proxy.ts provider 저장/복원 경로 테스트 — 스텁 확장 없이 순수 함수로 뺄 수 있는지 먼저 확인 (3/2/M) · plans.ts 나머지 순수 함수(planMeta, sortByPriority, linkedPlans) 테스트 보강 (2/1/S) · createCheckpoint 의 catch 가 update-ref 실패를 '저장소 아님' 으로 뭉개 audit 의 commit/note 가 어긋남 (2/1/S) · 거대한 untracked 디렉터리에서 동기 `git add -A` 지연 측정 후 필요할 때만 가드 (2/2/M) · 보관 후 `.active-plan` 포인터가 옮겨진 이름을 가리킨 채 남음 — 이제 보관 함수가 분리돼 싸게 얹을 수 있음 (2/1/S)
- 과제서: 채택 — 과제서의 근거(275줄 unlinkSync, restoreArchivedPlan 과의 정책 모순)를 코드에서 그대로 확인했고, 수용 기준 1~3 을 실패 재현 후 구현으로 모두 충족했다.

- 릴리즈: v1.4.2 (2026-09-26, run 2026-09-26-070059-vibe-code-improve)
