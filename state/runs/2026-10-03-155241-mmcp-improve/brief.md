# 과제서 — 2026-10-03 (수정 과제: 직전 회차 verify-failed)

- 과제: `cd web && npm test --silent` 가 깨끗한 체크아웃에서 항상 실패 — `web/package.json` 의 `test` 를 패키지 경로 호출로 바꾸고 `pretest` 설치 가드를 추가 (가치 5 / 위험 1 / 작업량 S)

- 왜: base `b782c5f` 의 `web/package.json:10` 은 `"test": "vitest run"` 이라 `node_modules/.bin/vitest` 심링크와 "이미 설치돼 있음" 을 전제한다. 이 워크트리에는 `web/node_modules` 가 아예 없어서(`ls -d web/node_modules` → `No such file or directory` 로 확인) 러너의 검증 명령이 그대로 떨어진다. 저장소의 다른 두 호출처(`Makefile:17`, `.github/workflows/ci.yml:57`)는 이미 `node node_modules/vitest/vitest.mjs run` 패키지 경로를 쓰므로, `package.json` 한 곳만 관례에서 빠져 있다. 고치면 설치 상태와 무관하게 `npm test` 가 돌고 러너의 검증이 통과한다.

- 실패 재현 (이번 정찰이 실제로 실행함):
  ```
  $ cd web && npm test --silent
  sh: 1: vitest: not found        # exit 127 (npm 래퍼/옵션에 따라 1 로 보고됨)
  ```

- 수용 기준:
  1) `rm -rf web/node_modules && cd web && npm test --silent` 가 **exit 0**, 그리고 출력에 `src/utils/format.test.ts` 3건 + `src/App.test.tsx` 1건 = **4 passed / 0 skipped** 가 보인다. (테스트 제외·`--passWithNoTests`·`continue-on-error` 로 통과시키는 것은 실패로 간주)
  2) 바로 한 번 더 `npm test --silent` 를 돌리면 재설치 없이(수 초 안에) 다시 exit 0 — `pretest` 가드가 이미 설치된 경우 no-op 이다.
  3) `git grep -n "vitest.*run" -- Makefile .github web/package.json` 결과, vitest 를 실제로 호출하는 문장이 저장소에 **한 곳**만 남는다(`web/package.json` 의 `scripts.test`). `Makefile:17` 과 `.github/workflows/ci.yml:57` 은 `npm test --silent` 를 부른다.
  4) `cd web && node node_modules/typescript/bin/tsc -b` 가 exit 0 (CI 가 vitest 다음에 하는 것 — 프런트 타입 빌드가 깨지지 않았음을 확인)
  5) `make lint` 와 `make test` 가 exit 0

- 건드릴 파일 (3개):
  - `web/package.json` — `scripts.test` 를 `"${npm_node_execpath:-node}" node_modules/vitest/vitest.mjs run` 로, 새 `scripts.pretest` 에 `scripts/build-ui.sh:6` 과 같은 가드 `[ -d node_modules ] || npm ci --no-audit --no-fund` 를 추가. **`npm_node_execpath` 로 인터프리터를 고정하는 것이 핵심이다** — 아래 "위험" 참조. (`build`/`dev`/`preview` 는 이번에 건드리지 말 것 — 별건이고 CI 는 패키지 경로를 쓴다)
  - `Makefile:17` — `cd web && node node_modules/vitest/vitest.mjs run` → `cd web && npm test --silent` (가드가 묻어오고 호출 문장이 한 곳으로 모인다)
  - `.github/workflows/ci.yml:57` — `node node_modules/vitest/vitest.mjs run` → `npm test --silent` (바로 위 줄에 `npm ci` 가 있으므로 가드는 no-op. 이 줄만 바꾸고 `npm ci`/`tsc -b`/`vite build` 줄은 그대로 둘 것)

- 검증 명령 (이 저장소에서 실제로 도는 것):
  ```
  rm -rf web/node_modules && cd web && npm test --silent     # 수용 기준 1
  cd web && npm test --silent                                 # 기준 2 (재실행)
  cd web && node node_modules/typescript/bin/tsc -b           # 기준 4
  make lint                                                   # gofmt -l . + go vet ./... + scripts/verify-version.sh
  make test                                                   # go test ./... + web 테스트
  go test -race -count=1 ./...                                # 회귀 확인 (Postgres 없으면 통합 테스트는 조용히 skip)
  ```

- 위험과 피할 것:
  - **npm lifecycle 스크립트의 PATH 오염이 이 과제의 함정이다.** 2026-10-02 회차가 실측한 것: `pretest` 가드를 넣으면 그 뒤에 두 번째 실패가 드러난다 — npm 은 lifecycle 스크립트의 PATH 앞에 **모든 상위 디렉터리**의 `node_modules/.bin` 을 붙이고, 당시 `/home/hkjang/node_modules/.bin/node` 가 Node 20.19.2 를 가리켜 `worker_threads.markAsUncloneable` 이 없어 jsdom 30 의 undici 8.11.2(`engines: >=22.19.0`)가 로드 중 죽었다(`TypeError: webidl.util.markAsUncloneable is not a function`, `Test Files no tests / Errors 2 errors`). 그래서 바레 `node` 가 아니라 `"${npm_node_execpath:-node}"` 를 써야 한다. **이번 정찰은 이 상위 `.bin` 을 다시 확인하지 못했다(미확인 — 세션의 읽기 허용 경로 밖이라 `ls` 가 차단됐고, `npm ci` 도 권한으로 막혀 실측하지 못했다).** 다만 현재 PATH 의 node 는 `v22.23.1`(확인)로 undici 요구를 만족하므로, 오염이 없으면 `npm_node_execpath` 는 무해한 no-op 이고 오염이 있으면 유일한 해법이다 — 어느 쪽이든 쓰는 것이 맞다. 만약 기준 1 에서 `markAsUncloneable` 오류가 보이면 원인은 이것이다.
  - 2026-10-02 에 **같은 파일(`web/package.json`)을 같은 방향으로 고친 PR 이 있고 그 회차는 `review-pending` 으로 끝나 이 base 에 머지되지 않았다**. 사람이 반려한 것이 아니라 심사가 안 끝난 것이므로 재제출 금지 규칙에 걸리지 않는다. 다만 그 PR 이 나중에 머지되면 충돌한다 — 커밋 메시지에 "re-apply" 임을 밝히고, 같은 처방(패키지 경로 + 가드 + `npm_node_execpath`)을 쓰는 것이 충돌 해소를 쉽게 한다.
  - `web/package-lock.json` 을 바꾸지 말 것 — `npm ci` 만 쓰면 바뀌지 않는다. `npm install` 금지.
  - 빌드 산출물(`web/dist`, `internal/ui/dist`)을 커밋에 넣지 말 것.
  - 보호 경로를 건드리지 않는다: `internal/server/oauth.go`·`oidc.go`·`auth.go`·`keys.go`, `internal/server/identity.go`, `internal/store/migrations`, `scripts/package-offline.sh`, `VERSION`. `.github/workflows/ci.yml` 은 한 줄만 바꾸고 `release.yml` 은 **건드리지 말 것** — release.yml 은 web 테스트를 부르지 않는다(읽어서 확인했다. `verify-version.sh` → `package-offline.sh` → `verify-offline.sh` → `gh release upload` 뿐).
  - 검증을 느슨하게 만드는 어떤 수정도 금지(테스트 제외, `--passWithNoTests`, `|| true`, `continue-on-error`).

- 차선 후보: `logbuf.Handler` 가 `WithGroup` 이전에 붙인 attr 에도 그룹 접두사를 붙여 JSON tee 와 버퍼 출력이 갈리는 것 수정 (`internal/logbuf/logbuf.go:96-106`, 가치 3 / 위험 2 / 작업량 S). 2026-10-03 에 `logbuf_test.go` 가 들어와 동작이 못 박혀 있어 지금 고치기 안전하고, 프로덕션에 `WithGroup`/`logger.With` 호출처가 0건이라 회귀 위험이 낮다. 단 1순위가 성립하지 않는 경우에만 — 1순위는 러너의 검증 자체를 막고 있으므로 먼저 끝내야 한다.
