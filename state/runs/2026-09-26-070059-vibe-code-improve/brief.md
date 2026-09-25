- 과제: `archiveDonePlans` 가 같은 이름의 보관 계획을 말없이 지우는 것을 막고, 보관 로직을 테스트 가능한 함수로 분리해 첫 테스트를 붙이기 (가치 4 / 위험 2 / 작업량 M)

- 왜: `src/features/plans.ts:265` 의 `vibe-code.archiveDonePlans` 명령은 `archive/<이름>` 이 이미 있으면 `fs.unlinkSync(dest)` 로 **기존 보관본을 지운 뒤** 덮어쓴다(plans.ts:275). 같은 파일의 `restoreArchivedPlan`(plans.ts:296-303)은 정반대로 충돌 시 `-restored-<stamp>` 를 붙여 보존하므로, 한 파일 안에서 같은 상황을 한쪽은 보존하고 한쪽은 파괴한다. `openLatestPlan` 은 plans 디렉터리가 비면 기본 이름 `current-plan.md` 를 다시 만들어 주므로(plans.ts:70, 166-170) "계획 작성 → done → 보관 → 새 계획(또 current-plan.md) → done → 보관" 이라는 정상 사용 루프만으로 이전 보관본이 휴지통도 없이 영구 삭제된다. 고치면 사용자의 과거 계획 기록이 보존되고, 저장소에서 가장 큰 파일(plans.ts 523줄, 단위 테스트 0개)에 첫 테스트가 생긴다.

- 수용 기준:
  1) `archive/<이름>` 이 이미 존재할 때 보관을 실행하면 **기존 보관본 파일의 내용이 그대로 남아 있고**, 새로 보관되는 파일은 충돌하지 않는 다른 이름(예: `<이름>-<stamp>.md`)으로 `archive/` 에 들어간다. 어떤 경우에도 `unlinkSync` 로 기존 보관본을 지우지 않는다.
  2) 충돌이 없을 때의 동작은 그대로다 — `상태: done` 인 `.md` 만 `plans/` 에서 `archive/` 로 옮겨지고, done 이 아닌 파일과 `archive` 하위 디렉터리는 건드리지 않는다. `writeAudit("plan","archiveDonePlans", …)` 의 `files` 에는 **실제로 기록된 목적지 이름**이 들어간다(이름이 바뀌었으면 바뀐 이름).
  3) 테스트는 `fs.mkdtempSync(os.tmpdir())` 로 만든 **진짜 디렉터리 트리**에서 실제 파일을 만들고 옮겨 위 1)·2)를 증명한다. 특히 "같은 이름으로 두 번 보관" 시나리오를 재현해 **고치기 전에는 기존 보관본 내용이 사라지는 것**을 먼저 확인(테스트 실패)한 뒤 고친다. 가짜 fs·가짜 vscode 명령 대역을 만들어 증명하지 말 것.

- 건드릴 파일:
  - `src/features/plans.ts` — 보관 루프(현재 265~285줄, `vibe-code.archiveDonePlans` 콜백 안)를 vscode 에 의존하지 않는 **exported 순수 함수**로 뽑아낸다. 예: `export function archiveDonePlans(plansDir: string, archiveDir: string, stamp: string): { name: string; dest: string }[]`. 명령 콜백은 `paths.plans`/`paths.archive` 와 스탬프를 넘겨 호출하고, 반환값으로 알림 문구와 `writeAudit` 의 `files` 를 만든다. 충돌 시 이름은 `restoreArchivedPlan`(plans.ts:299-301)의 방식(`path.parse` + 숫자 스탬프 접미사)을 그대로 따라 일관되게 할 것 — 새 규칙을 발명하지 말 것.
  - `tests/unit/plans.test.ts`(신규) — 위 함수에 대한 테스트. 선례는 `tests/unit/checkpoints.test.ts`(`tempDir()` 헬퍼로 `fs.mkdtempSync` + `fs.realpathSync`, `afterEach` 에서 정리). 여력이 있으면 같은 파일에서 `selectPlanName`(포인터 없음/깨짐/존재하지 않는 파일 지목 → 마지막 파일 폴백, 빈 디렉터리 → `current-plan.md`)과 `openPlanItems`(`단계`/`Now`/`검증 계획` 의 미체크 항목만, `- [x]` 제외) 도 같이 덮되, **1순위는 보관 충돌 회귀 테스트**다.
  - (문자열이 사용자에게 보이면) `readme.ko.md`/`docs/` 에 보관 충돌 동작이 설명돼 있는지 확인하고 어긋나면 한국어로 정정. — 실제로 그런 문장이 있는지는 **미확인**.

- 검증 명령:
  - `npm ci` (워크트리에 `node_modules` 없음 — 반드시 먼저)
  - `npx vitest run tests/unit/plans.test.ts` (빠른 반복용)
  - `npm run check` (= `typecheck` + `vitest run` + esbuild `build`. 현재 기준선은 53 tests 통과)
  - `npm run vsix` / `verify` / `smoke:vscode` / `test:extension-host` 는 **PowerShell 전용이라 리눅스에서 돌지 않는다** — 시도하지 말 것.

- 위험과 피할 것:
  - `vendor/extension.core.js`, `src/core/hooks.ts`, `.github/workflows/`, `scripts/*.ps1`, `release/` 는 건드리지 말 것(릴리즈 경로).
  - `src/features/checkpoints.ts` 는 이번 회차와 무관하다. 지난 회차에서 방금 고쳤으므로 같이 손대지 말 것.
  - `tests/unit/vscode-stub.ts` 의 `commands` 는 빈 객체라 `registerCommand` 가 없다. **등록된 명령을 테스트에서 호출하려고 스텁을 확장하지 말 것** — 그렇게 하면 운영자가 반복해 지적한 "손으로 만든 대역" 이 된다. 로직을 vscode 밖으로 빼서 실제 fs 로 검증하는 것이 이 저장소의 확립된 방식이다(checkpoints 선례).
  - `restoreArchivedPlan` 의 충돌 규칙과 새 보관 충돌 규칙이 서로 다른 스탬프 형식을 쓰지 않게 할 것 — 같은 값을 읽는 두 경로가 어긋나면 안 된다.
  - 보관 후 `.active-plan` 포인터가 옮겨진 이름을 가리킨 채 남는 문제는 `selectPlanName`(plans.ts:68) 이 `files.includes(active)` 로 막아 폴백하므로 **이번 범위 밖**이다. 같이 고치려다 scope 를 키우지 말 것.
  - `fs.statSync(src).isFile()` 체크와 `.md` 필터(`listMarkdown`, workspace.ts:83)는 그대로 유지할 것.

- 차선 후보: `plans.ts` 순수 함수 단위 테스트만 붙이기 — `selectPlanName`(`.active-plan` 없음/깨짐/없는 파일 지목/빈 디렉터리), `planMeta`, `sortByPriority`(P0~P3 + 미지정 + 동순위 이름순), `linkedPlans`(`연결 목표` 없는 파일 제외), `openPlanItems`. 프로덕션 코드 변경 0, 위험 1, 작업량 M. 1순위의 재현(같은 이름 두 번 보관)이 어떤 이유로 성립하지 않으면 이쪽으로 갈 것. 단, `sortByPriority` 가 입력 배열을 제자리 정렬(`rows.sort`)하는 점은 테스트에서 전제로 삼지 말 것.
