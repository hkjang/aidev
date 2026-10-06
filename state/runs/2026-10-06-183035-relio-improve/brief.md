# 과제서 (2026-10-06 정찰 — 확정)

> 단 하나의 미확인 전제: **이 세션은 네트워크·npm 실행이 차단돼 `npm audit --audit-level=high` 의 현재 exit code 를 확인하지 못했다.** 기준 1 을 가장 먼저 돌려 그 사실부터 확정하라. 그 밖의 모든 수치는 실제로 열어 본 파일에서 나왔다.

- 과제: 수정 과제 — CI·release 를 두 번 깨뜨린 `npm audit --audit-level=high` 게이트를 **로컬 검증 경로에 재현 가능하게 추가**하고, lockfile 에만 손으로 박힌 `source-map-js` 패치 버전을 `web/package.json` 의 `overrides` 로 **선언**한다 (가치 4 / 위험 2 / 작업량 S)
- 왜: `ci-failure-e00f2446444f.txt` 가 실패 단계를 `test / Frontend security audit` 로 특정한다 — 깨진 것은 코드가 아니라 `npm audit --audit-level=high` 다. 그런데 이 명령은 `.github/workflows/ci.yml:26` 과 `.github/workflows/release.yml:38` **두 곳에만** 있고 `Makefile:13-16`·루트 `package.json`·`scripts/` 어디에도 없어(저장소 전체 grep 으로 확인), 에이전트는 `make test` 를 green 으로 보고 PR 을 열고 CI 에서만 터진다. 게이트를 **느슨하게 만들지 않고** 로컬 경로에 그대로 복제하면 같은 실패가 PR 전에 잡힌다.
- 수용 기준:
  1) `npm --prefix web ci` 뒤 `npm --prefix web audit --audit-level=high` 를 **실제로 돌려** exit code 와 전체 출력을 기록한다. 0 이든 1 이든 그 출력이 이번 회차의 사실 근거다.
  2) exit 1 이면 그것이 살아 있는 결함이다 — 남은 advisory 의 패키지·심각도·patched 버전을 적고, 직접 의존(`web/package.json` 의 `vite`/`@vitejs/plugin-react` 등) 상향 또는 `overrides` 로 고친 뒤 `npm install` 로 lockfile 을 **정상 재생성**한다(손 편집 금지).
  3) `web/package.json` 에 `"overrides": { "source-map-js": "^1.2.2" }` 를 추가하고 `npm install --package-lock-only` 로 lockfile 에 override 가 기록되게 한다. 6b95f32 는 `web/package-lock.json` **한 파일만** 손으로 고쳤고(`git show --stat 6b95f32` = 1 file, 3+/3-) `web/package.json` 에는 1.2.2 를 요구하는 것이 아무것도 없다.
  4) `Makefile:13` 의 `test` 타깃에 `npm audit --audit-level=high` 를 CI 와 **같은 문구로** 추가한다(`cd web && npm ci && npm audit --audit-level=high && npm run typecheck && npm test && npm run build`). 워크플로 파일은 **0줄** 건드린다.
  5) `npm --prefix web ci` → `audit --audit-level=high` → `run typecheck` → `test`(27건) → `run build` 가 전부 exit 0, `make test` exit 0, `go test ./...` 23 ok / FAIL 0.
  6) `README.md` 의 검증 명령 목록에 audit 한 줄을 더한다(문서 1문장, 선택).
- 건드릴 파일 (4개):
  - `web/package.json` — `overrides` 블록 추가(현재 `dependencies` 5개·`devDependencies` 2개뿐, `overrides` 없음).
  - `web/package-lock.json` — `npm install --package-lock-only` 로 재생성. 현재 `lockfileVersion: 3`, `node_modules/source-map-js` 는 1621-1626 행 한 자리.
  - `Makefile:13-16` `test` 타깃 — audit 한 단계 추가.
  - `README.md` — 검증 명령 목록(선택, 1문장).
- 검증 명령:
  - `npm --prefix web ci`
  - `npm --prefix web audit --audit-level=high`  ← **이번 회차의 핵심. 반드시 돌리고 출력을 원장에 붙일 것**
  - `npm --prefix web run typecheck` · `npm --prefix web test` · `npm --prefix web run build`
  - `make test`
  - `go test ./...` · `go vet ./...` · `gofmt -l ./cmd ./internal`
  - `./scripts/check-env-contract.sh` · `./scripts/check-static-assets.sh` · `./scripts/previous-release-tag-test.sh`
  - 빌드 뒤 `cmp internal/webui/dist/README web/public/README` (embed 앵커 보존)
- 위험과 피할 것:
  - `.github/workflows/` 는 **0줄**. `--audit-level=high` 를 `critical` 로 낮추거나 `|| true` 를 붙이는 것은 명시 금지(게이트 느슨화).
  - `npm install` 이 lockfile 을 **대폭** 재생성하면(수십 패키지 버전 이동) 범위가 터진다 — `git diff --stat web/package-lock.json` 을 보고 source-map-js 와 override 기록 외에 큰 이동이 있으면 `npm install --package-lock-only` 로 최소화하고, 그래도 크면 기준 3 을 버리고 기준 1·2·4 만 하라.
  - `web/package.json:10` 의 `"$npm_node_execpath"` 와 `**` 글롭은 한 글자도 바꾸지 말 것(2026-10-02 회차가 이 자리에서 중첩 npm v20 그림자를 겪었다).
  - 빌드 뒤 `internal/webui/dist/README`·`web/public/README` embed 앵커 보존 확인.
  - `make test` 는 `npm ci` 로 네트워크를 탄다 — audit 도 네트워크를 탄다. 네트워크 없는 환경에서 `make test` 를 돌릴 수 없게 되는 것은 **의도된 비용**(CI 와 같아지는 것이 목적)이니 회피 플래그를 넣지 말 것.
- 차선 후보: `release.yml` 의 `Test source` 단계에 `npm --prefix web test` 추가 — ci.yml·Makefile 은 프런트 27건을 돌리는데 release.yml 만 안 돌린다. 보호 경로 단독 1줄 변경이므로 로컬에서 `npm --prefix web test` exit 0 을 재현한 뒤에만.
