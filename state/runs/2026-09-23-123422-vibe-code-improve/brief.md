# 과제서 (2026-09-23, 정찰)

- 과제: 파괴적 명령 전 체크포인트가 untracked 파일까지 담도록 고치고 첫 테스트를 붙이기 (가치 5 / 위험 2 / 작업량 M)

- 왜: `src/features/checkpoints.ts:createCheckpoint`는 `git stash create`로 스냅샷을 만드는데, 이 명령은 **추적 중인 변경만** 담고 untracked 파일은 담지 않습니다. 그런데 노트와 설정 설명은 "git 작업 트리 스냅샷"이라고만 적혀 있어, `rm -rf` 직전에 새로 만든(아직 `git add` 안 한) 파일이 보호된다고 사용자가 오해합니다. 고치면 이 기능이 실제로 막아야 할 사고(새 파일 삭제)를 막고, 안전장치인데도 테스트가 0개인 모듈에 실제 git을 통과하는 회귀 테스트가 생깁니다.

- 수용 기준:
  1) `createCheckpoint`가 만드는 스냅샷 커밋에 **추적 파일의 미커밋 변경 + untracked 파일(.gitignore 제외분 제외)** 이 모두 들어간다. 호출 전후로 사용자의 작업 트리·index·stash 스택이 전혀 바뀌지 않는다(`git status --porcelain` 동일, `git stash list` 비어 있음).
  2) 체크포인트 노트(`.vibe-code/checkpoints/*.md`)의 "복원" 블록이 **실제로 도는 명령**이다. `git stash create`를 버리면 그 커밋은 stash 형식이 아니므로 `git stash apply <commit>` 은 더 이상 쓰면 안 되고(`is not a stash-like commit` 으로 실패), `git checkout <commit> -- .` / `git restore --source=<commit> .` / `git diff HEAD <commit>` 계열로 바꿔야 한다. 스냅샷에 없던 파일은 이 복원으로 지워지지 않는다는 점을 노트에 한 줄로 적을 것.
  3) 새 테스트 `tests/unit/checkpoints.test.ts` 가 증명할 것: (a) `fs.mkdtempSync`로 만든 **진짜 git 저장소**에서 `createCheckpoint(ws, dir, cmd, reason)` 를 호출해 `git ls-tree -r --name-only <commit>` 에 수정된 추적 파일과 untracked 파일이 **둘 다** 나온다, (b) 호출 후 `git status --porcelain` 이 호출 전과 같고 `git stash list` 가 비어 있다, (c) git 저장소가 아닌 디렉터리에서 호출해도 예외 없이 노트 파일만 남는다(기존 catch 경로 회귀 방지), (d) `isDestructiveCommand` 표 테스트 — 참: `rm -rf build`, `git reset --hard`, `git push origin main --force`, `kubectl delete pod x`, `DROP TABLE users`; 거짓: `npm test`, `git status`, `ls -R`, `git restore --staged file.ts`. 가짜 git 대역이나 소스 문자열 검사로 대신하지 말 것.

- 건드릴 파일:
  - `src/features/checkpoints.ts:createCheckpoint` — `git(ws, ["stash","create"])` 를 임시 index 기반 스냅샷으로 교체. 권장 절차(모두 `execFileSync`, `env: { ...process.env, GIT_INDEX_FILE: tmp }`): `git read-tree HEAD` → `git add -A` → `git write-tree` → `git commit-tree <tree> -p HEAD -m "vibe checkpoint"`. 임시 index 파일은 `os.tmpdir()` 아래에 만들고 finally에서 삭제할 것(`.vibe-code/checkpoints/` 안에 두지 말 것 — `listCheckpoints`가 보는 디렉터리다). 결과 커밋은 지금처럼 `git update-ref refs/vibe-checkpoints/<stamp>` 로 고정.
  - `src/features/checkpoints.ts:createCheckpoint` 의 `body` 배열 — 복원 블록 명령 교체(수용 기준 2).
  - `package.nls.json` / `package.nls.ko.json` 의 `settings.checkpointBeforeDestructive.description` — 추적/미추적 파일을 모두 담는다는 문구로 정정(설정 **키 이름은 바꾸지 말 것**).
  - `tests/unit/checkpoints.test.ts` — 신규. 기존 `tests/unit/verification.test.ts` 의 `describe/it/expect` 스타일을 따를 것.
  - (선택) `readme.md` 의 체크포인트 언급이 있으면 같은 문구로 정정.

- 검증 명령:
  ```
  npm ci            # 이 워크트리에는 node_modules 가 없다. 먼저 실행할 것
  npm run typecheck # tsc -p tsconfig.json --noEmit
  npm test          # vitest run (tests/unit/**/*.test.ts)
  npm run check     # typecheck + test + esbuild 빌드까지. 마무리로 한 번
  ```
  (`npm run vsix` 등 PowerShell 스크립트는 Windows 전용이므로 이번 과제에서 돌리지 말 것.)

- 위험과 피할 것:
  - **`git stash push` / `git stash pop` 을 절대 쓰지 말 것.** stash 스택은 다른 워크트리·세션과 공유되고, 사용자의 작업 트리를 건드리는 순간 "파괴적 명령 직전 스냅샷"이라는 계약이 깨진다. 임시 index 방식은 작업 트리와 index를 둘 다 건드리지 않아 이 계약을 지킨다.
  - 커밋이 하나도 없는 저장소(HEAD 없음)에서는 `rev-parse HEAD` 와 `-p HEAD` 가 실패한다. 지금은 통째로 catch 로 떨어져 "git 저장소가 아님" 노트가 남는다 — 이 경로를 **더 나쁘게 만들지 말 것**(부모 없이 `commit-tree <tree>` 로 처리하거나, 최소한 현재와 같은 동작을 유지).
  - `checkpointBeforeCommand` 는 명령 승인 훅에서 **동기**로 불린다. 비동기로 바꾸거나 오래 걸리는 작업을 넣지 말 것.
  - 보호 경로 금지: `.github/workflows/ci.yml`, `vendor/extension.core.js`, `scripts/*.ps1` 은 이번 과제에서 건드리지 말 것.
  - 감사 로그(`writeAudit`)에 넣는 `details` 에는 지금처럼 명령·파일명·ref·commit 만 넣고, 명령 출력이나 파일 내용을 새로 흘려 넣지 말 것.
  - 효과 없는 변경 금지: `DESTRUCTIVE` 정규식을 "혹시 몰라서" 넓히지 말 것. 이번 과제의 정규식 작업은 **테스트로 현재 동작을 고정하는 것까지**다.
  - 미확인: 이 세션에서는 샌드박스 때문에 임시 git 저장소 실험과 `npm test` 를 **실행하지 못했다**. `git stash create`가 untracked 를 제외한다는 것과 `git stash apply` 가 stash 형식이 아닌 커밋을 거부한다는 것은 git 문서 기준 판단이며, 구현자는 1번 테스트를 먼저 작성해 **현재 코드에서 실패하는 것을 눈으로 확인**한 뒤 고칠 것(그 실패가 이 과제의 근거다). 만약 현재 코드가 이미 untracked 를 담는다면 이 과제는 성립하지 않으니 차선 후보로 넘어갈 것.

- 차선 후보: `src/features/plans.ts`(523줄, 테스트 0개)의 순수 함수 테스트 보강 — `selectPlanName`(.active-plan 포인터가 없는/깨진/존재하지 않는 파일을 가리키는 경우), `planMeta`, `sortByPriority`(P0~P3 + 미지정), `linkedPlans`, `openPlanItems` 를 `tests/unit/plans.test.ts` 에 고정. 가치 3 / 위험 1 / M.
