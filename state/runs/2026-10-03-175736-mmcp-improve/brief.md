- 과제: `cd web && npm test --silent` 가 깨끗한 체크아웃에서 항상 실패 — `web/package.json` 의 `test` 를 패키지 경로로 바꾸고 `pretest` 설치 가드 추가, vitest 를 돌리는 Node 를 `npm_node_execpath` 로 고정 (가치 5 / 위험 1 / 작업량 S)

- 왜: base `b782c5f` 의 `web/package.json:10` 만 `"test": "vitest run"` 으로 `node_modules/.bin/vitest` 심링크와 "이미 설치돼 있음" 을 전제한다. 이 워크트리에서 그대로 재현했다 — `web/node_modules` 가 없고(`ls: cannot access 'node_modules'`) `cd web && npm test --silent` → `sh: 1: vitest: not found`. 이것이 러너가 두 회차 연속 `verify-failed` 로 끝난 바로 그 명령이다. 고치면 러너 검증·`make test`·사람 손 체크아웃이 모두 CI(`npm ci` 선행)와 같은 경로를 밟는다.

- 수용 기준:
  1) `rm -rf web/node_modules && cd web && npm test --silent` 가 **exit 0**, 테스트가 실제로 돌아 `src/utils/format.test.ts` + `src/App.test.tsx` 가 통과한다(skip 0, "no tests" 아님 — `--reporter=verbose` 로 테스트명을 눈으로 확인할 것).
  2) 바로 재실행하면 재설치 없이(수 초 안에) 다시 exit 0 — `pretest` 가드가 설치돼 있을 때 no-op 임을 증명.
  3) vitest 를 호출하는 문장이 저장소에 **한 곳만** 남는다 — `git grep -n "vitest.*run"` 이 `web/package.json` 한 줄만 보여주고, `Makefile`·`ci.yml` 은 `npm test --silent` 를 부른다.
  4) `cd web && node node_modules/typescript/bin/tsc -b` exit 0 (CI 가 테스트 다음에 하는 것 — 프런트 타입 빌드를 깨뜨리지 않았음).
  5) `make lint` exit 0 (`gofmt -l .` 빈 출력 + `go vet ./...` + `./scripts/verify-version.sh`).

- 건드릴 파일 (프로덕션 3개, 신규 파일 없음):
  - `web/package.json:10 scripts.test` — `"vitest run"` → `node_modules/vitest/vitest.mjs run` 을 **`"${npm_node_execpath:-node}"` 로** 실행. 이 저장소 관례는 이미 패키지 경로 직접 호출이다(`Makefile:17`, `.github/workflows/ci.yml:57`, `scripts/build-ui.sh:7-8` 이 `node node_modules/typescript/bin/tsc`·`node_modules/vite/bin/vite.js` 를 쓴다).
  - `web/package.json scripts.pretest` (신규 한 줄) — `scripts/build-ui.sh:6` 의 `[ -d node_modules ] || npm ci --no-audit --no-fund` 를 **글자까지 같게** 복사. 가드를 새로 발명하지 말 것.
  - `Makefile:17` — `cd web && node node_modules/vitest/vitest.mjs run` → `cd web && npm test --silent`. (`Makefile:15-17 test:` 타깃에는 지금 설치 가드가 없어서 깨끗한 체크아웃에서 `MODULE_NOT_FOUND` 로 떨어진다. 이 교체가 그것까지 같이 고친다.)
  - `.github/workflows/ci.yml:57` — `node node_modules/vitest/vitest.mjs run` → `npm test --silent`. 바로 위 56줄의 `npm ci --no-audit --no-fund` 는 **그대로 둘 것**(CI 는 캐시를 쓰는 별도 경로다).

- 검증 명령 (이 저장소에서 실제로 도는 것):
  ```
  cd web && rm -rf node_modules && npm test --silent          # 기준 1 — 핵심
  cd web && npm test --silent                                  # 기준 2 — 재실행
  git grep -n "vitest.*run" -- . ':!web/package-lock.json'     # 기준 3
  cd web && node node_modules/typescript/bin/tsc -b            # 기준 4
  make lint                                                    # 기준 5
  go test ./... && make test                                   # 회귀 — 둘 다 exit 0
  ```

- 위험과 피할 것:
  - **`npm_node_execpath` 고정을 빼지 말 것.** npm 은 lifecycle 스크립트의 PATH 에 **모든 상위 디렉터리**의 `node_modules/.bin` 을 앞에 붙인다. 2026-10-03 회차가 이 환경에서 `/home/hkjang/node_modules/.bin/node` = **v20.19.2** 이고 `worker_threads.markAsUncloneable` 이 `undefined` 임을 실측했고, 그 결과 바레 `node` 로는 `web/vitest.config.ts:7` 의 `environment: 'jsdom'` → jsdom 30 → undici(`engines: >=22.19.0`)가 로드 중 `TypeError: webidl.util.markAsUncloneable is not a function` 으로 죽어 테스트가 **0개** 돌고 exit 1 이 된다. 이번 회차는 샌드박스가 그 경로 조회를 막아 **재확인하지 못했다(미확인)** — 다만 PATH 의 node 는 v22.23.1 이고 `jsdom` 요구는 유효하므로, 고정은 hijack 이 없으면 no-op 이고 있으면 필수다. 구현자는 수용 기준 1 이 떨어지면 먼저 이 줄을 의심할 것.
  - **느슨하게 만들어 통과시키는 것은 금지.** 테스트 제외, `--passWithNoTests`, `|| true`, `continue-on-error`, 기준 하향 모두 안 된다. "exit 0" 만으로는 부족하고 기준 1 의 "테스트가 실제로 돌았다" 를 확인해야 한다.
  - `web/package-lock.json` 을 바꾸지 말 것 — `npm ci` 만 쓰고 `npm install` 은 쓰지 말 것. `web/dist`·`internal/ui/dist` 빌드 산출물을 커밋에 넣지 말 것.
  - `web/package.json` 의 `build`·`dev`·`preview`·`screenshots` 는 **이번 범위 밖**이다(아직 `.bin` 의 `tsc`/`vite` 에 의존하지만 CI 와 `build-ui.sh` 는 패키지 경로를 쓰므로 깨지지 않는다). 같은 파일이라 손이 가지만 참을 것 — 파일 수를 늘리지 않는 쪽이 머지된다.
  - `.github/workflows/release.yml` 은 **건드리지 말 것.** 확인했다: `grep -n "npm\|vitest\|node\|test"` 결과 release.yml 에는 npm·node·web 테스트 단계가 **없다**(릴리즈는 로컬 `make package-offline` + `gh release create` 로 컷한다). 즉 이번 실패는 GitHub release 워크플로가 아니라 **러너의 검증 명령**이다. 릴리즈 경로(`VERSION`, `scripts/verify-version.sh`, `scripts/package-offline.sh`)는 손대지 말 것.
  - 보호 경로(`internal/server/oauth.go`·`oidc.go`·`auth.go`·`identity.go`, `internal/store/migrations`)는 이 과제와 무관하다 — 전혀 열지 말 것.
  - **먼저 확인할 것:** 이 처방과 동일한 PR 이 두 번 `review-pending` 으로 남아 있다. 작업 시작 전에 `web/package.json:10` 을 열어 보라. 이미 `node_modules/vitest/vitest.mjs` 로 되어 있으면 **이 과제는 끝난 것이므로 아래 차선 후보로 넘어갈 것.** base `b782c5f` 에서는 아직 `"vitest run"` 임을 이번 회차에 확인했다.

- 차선 후보: `internal/mattermost/client.go` 의 남은 경로에 테스트 추가 (가치 4 / 위험 2 / 작업량 M) — `FileContent` 의 `*[]byte` 우회, `PostList.Ordered()` 가 응답에 없는 id 를 조용히 버리는 것, `New()` 의 깨진 CA PEM 거부. `internal/server/integration_test.go:30 fakeMM` 관례대로 `httptest.Server` + 실제 `*mattermost.Client` 로 배선할 것(손으로 만든 인터페이스 대역은 이 저장소 관례가 아니다). 주의: base 에 `client_test.go` 가 **없음**을 이번 회차에 `git grep` 으로 확인했으나, 2026-10-02 회차가 같은 이름의 파일을 만든 적이 있어 그 PR 이 되살아나면 충돌한다 — 시작 전에 `ls internal/mattermost/` 로 확인할 것.
