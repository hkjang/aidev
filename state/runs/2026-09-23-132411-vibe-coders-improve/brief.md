# 정찰 과제서 — 2026-09-23 (HEAD d67267b)

- **과제**: 수정 과제 — `pnpm run <script> --silent` 가 tsc·vite 에 그대로 전달돼 검증이 죽는 것을 `web/scripts/run-tool.mjs` 복원으로 해소 (가치 5 / 위험 1 / 작업량 S)

- **왜**: 외부 러너의 `run_verify` 는 npm 관례대로 `cd web && pnpm run typecheck --silent` 를 만드는데, pnpm 은 npm 과 달리 `--silent` 를 소비하지 않고 스크립트 줄 끝에 이어 붙인다. 현재 `web/package.json:13` 의 `"typecheck": "tsc -b --pretty false"` 는 그래서 `tsc -b --pretty false --silent` 로 실행되고 tsc 가 TS5072(Unknown build option)로 거부해 exit 1 — 코드에 타입 오류가 하나도 없어도 회차가 verify-failed 로 끝난다(직전 회차 실패 사유가 정확히 이 명령). 고치면 같은 명령이 exit 0 이 되고, 진짜 타입 오류만 검증을 실패시키게 된다.

- **핵심 사실(직접 확인)**:
  - 이 수정은 이미 네 번 작성됐고 **한 번도 master 에 병합되지 않았다**. `git merge-base --is-ancestor 964d2b0 HEAD` → exit 1. `git branch -a --contains 964d2b0` → `auto/2026-09-17-2153` 하나뿐.
  - 같은 내용의 커밋 4개가 저장소에 남아 있다: `964d2b0`·`92bc502`·`6bbc8fc`(테스트 55줄, 동일) 와 **`3a17c4f`(테스트 95줄, 가장 강함)**. → **`3a17c4f` 를 기준으로 복원할 것.**
  - `.github/workflows/ci.yml:58` 은 `pnpm typecheck`(플래그 없음)라 CI 는 이 문제와 무관하고, 이 수정으로 **느슨해지지 않는다** — tsc·vite 는 그대로 전부 실행되고, 끝에 붙은 quiet 플래그만 버린다.
  - 이 워크트리에 `web/node_modules` 가 **없다**(`web/node_modules/typescript` 부재 확인). 먼저 설치해야 한다.

- **수용 기준**:
  1. `cd web && pnpm run typecheck --silent` 가 exit 0, `cd web && pnpm run build --silent` 가 exit 0 이고 `web/dist/index.html` 과 비어 있지 않은 `web/dist/assets` 가 생긴다.
  2. **수정 전 상태에서 두 명령이 실패하는 것을 먼저 기록**한다(typecheck → TS5072, build → CACError "Unknown option --silent"). 로그를 회차 디렉터리에 남길 것.
  3. 회귀 테스트가 "느슨해지지 않았음"을 증명한다: `web/src` 아래에 의도적 타입 오류(`const invalid: number = "not a number"`) fixture 를 넣으면 `typecheck --silent`·`build --silent` 가 **둘 다 0 이 아닌 코드로 실패**하고 출력에 `invalid.ts … TS2322` 가 있으며 Vite 빌드 로그가 뒤이어 나오지 않는다(타입 실패가 빌드로 덮이지 않음). fixture 는 try/finally 로 반드시 제거.
  4. 플래그 없는 `pnpm typecheck` / `pnpm build` / `pnpm test` / `pnpm lint` / `pnpm format:check` / `pnpm openapi:check` 가 모두 exit 0 (기존 동작 불변).
  5. 새 테스트 파일이 `test` 스크립트에서 실제로 돌아간다(`node --test … scripts/run-tool.test.mjs`).

- **건드릴 파일** (이 3개만, `3a17c4f` 의 diff 그대로):
  - `web/scripts/run-tool.mjs` — 신규. `stripForwardedFlags`(끝에 붙은 `--silent|-s|--quiet|-q` 만 pop, 중간에 쓴 플래그는 보존) / `resolveBin`(`node_modules/.bin` 대신 패키지 manifest 의 `bin` 으로 해석 — 윈도 `.cmd` shim 회피, vite 의 `exports` 잠금 우회) / `runTool`(`spawnSync(process.execPath, …, {stdio:"inherit"})`, 종료 코드 그대로 반환).
  - `web/package.json:12-14,20` — `build`: `node scripts/run-tool.mjs typescript:tsc -b && node scripts/run-tool.mjs vite build`, `typecheck`: `node scripts/run-tool.mjs typescript:tsc -b --pretty false`, `test`: 끝에 `scripts/run-tool.test.mjs` 추가.
  - `web/scripts/run-tool.test.mjs` — 신규(node:test 6개). 3a17c4f 판본을 쓸 것: `stripForwardedFlags` 단위, `resolveBin` 성공·실패, 실제 tsc/vite 가 `--version --silent` 로 통과, 실제 tsc 가 `--no-such-option` 에 0 아닌 코드, **중간 위치 quiet 플래그와 미지원 플래그는 실제 도구가 여전히 거부**, 그리고 수용 기준 3의 TS2322 fixture 회귀.

- **검증 명령** (전부 `web/` 에서, 순서대로):
  ```
  cd web && pnpm install --frozen-lockfile
  pnpm run typecheck --silent        # 수정 전 실패 로그 먼저 → 수정 후 exit 0
  pnpm run build --silent            # 수정 전 실패 로그 먼저 → 수정 후 exit 0
  pnpm typecheck && pnpm build
  node --test scripts/run-tool.test.mjs
  pnpm test && pnpm lint && pnpm format:check && pnpm openapi:check
  ```
  Go 쪽은 이 변경과 무관하지만 트리 무결성 확인용으로 루트에서 `go build ./...` 와 `go vet ./...` 만(전체 `go test ./...` 는 약 80초, 이 diff 로는 불필요).

- **위험과 피할 것**:
  - **`.github/workflows/*.yml` 를 건드리지 말 것.** CI 는 `--silent` 를 쓰지 않으므로 고칠 것이 없고, 워크플로를 느슨하게 만드는 것은 이번 회차 금지 사항이다. 수정은 `web/` 안 3개 파일로 끝난다.
  - `--silent` 를 `--logLevel silent` 로 **번역하지 말 것**. 그러면 검증이 존재하는 이유인 빌드 오류 자체가 숨는다. 드롭이 맞다.
  - 끝에 붙은 것만 벗겨야 한다. 중간에 의도적으로 쓴 플래그까지 벗기면 스크립트가 조용히 의미를 바꾼다(수용 기준 3의 테스트가 이 경계를 지킨다).
  - `pnpm-lock.yaml` 만 있으므로 `npm ci` 금지. lockfile·의존성 버전을 바꾸지 말 것(`--frozen-lockfile` 유지).
  - 새 테스트는 안에서 `pnpm run build --silent` 를 두 번 spawn 하므로 `pnpm test` 가 눈에 띄게 느려진다(각 120s timeout). 예상 총 소요 15–35분(80% 신뢰), 기본 20분. 지연 요인 1순위는 frozen install 실패 — 그때는 lockfile 을 고치지 말고 회차 노트에 적고 멈출 것.
  - `web/node_modules`·`web/dist` 는 gitignore 대상이다. 커밋 전 `git status` 가 정확히 3개 파일만 보여야 한다.
  - 저장소 파일은 CRLF 가 섞여 있을 수 있다 — 새 파일 2개는 저장소 기존 `web/scripts/*.mjs` 의 줄바꿈·prettier 스타일을 따를 것(`pnpm format:check` 가 이를 잡는다).
  - 커밋 메시지는 영어 conventional 스타일, Claude 귀속 트레일러 금지. 1커밋 권장: `build(web): restore forwarded quiet flag compatibility`.

- **차선 후보**: `docs/APP_UI_ROADMAP.md:28` 의 추적 활성 시 캐시 설명 정정 (가치 2 / 위험 1 / 작업량 S) — 문서는 `index.html` 이 항상 `no-cache` 라고 하나 추적이 켜지면 `internal/appui/handler.go:147` 의 `serveTrackedIndex` 가 `no-store` 를 보내고 validator 를 붙이지 않는다. 문서만 고치는 과제라 1순위가 어떤 이유로든 성립하지 않을 때 안전하게 집을 수 있다.

## 미확인(정직하게)
- 정찰인 나는 `pnpm run typecheck --silent` 를 **이번 회차에 직접 실행하지 못했다**: 이 워크트리에 `node_modules` 가 없고 `pnpm run` 실행이 이 세션에서 승인되지 않았다. TS5072·CACError 라는 구체적 오류는 09-17·09-20 회차 기록과 `964d2b0` 커밋 메시지에 남은 것이고, 원인(package.json 이 tsc 를 직접 부름 + 이 수정이 HEAD 의 조상이 아님)은 이번 회차에 코드와 `git merge-base` 로 직접 확인했다. **구현자는 수용 기준 2대로 수정 전 실패를 자기 손으로 먼저 재현해 로그로 남길 것** — 재현되지 않으면 원인이 바뀐 것이므로 복원을 강행하지 말고 다시 진단할 것.
