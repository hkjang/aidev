- 과제: `cd web && npm test --silent` 가 깨끗한 체크아웃에서 항상 실패 — `web/package.json` 의 `test` 를 패키지 경로로 바꾸고 `pretest` 설치 가드 추가 (가치 5 / 위험 1 / 작업량 S)
- 왜: base `b782c5f` 의 `web/package.json:10` 만 `"vitest run"` 이라 `node_modules/.bin` 심링크와 "이미 설치돼 있음" 을 전제하는데, 저장소의 다른 모든 호출처(`Makefile:17`, `.github/workflows/ci.yml:57`, `scripts/build-ui.sh:7-8`)는 이미 `node node_modules/<pkg>/...` 패키지 경로를 쓴다. CI 는 바로 위 `ci.yml:56` 의 `npm ci` 덕에 초록이지만, 러너의 검증 명령은 `web/node_modules` 가 없는 워크트리에서 그대로 떨어진다 — 이번 회차 verify-failed 의 원인이다. 고치면 깨끗한 체크아웃에서도 `npm test` 가 돌고 vitest 실행 문장이 저장소에 한 곳만 남는다.

- 수용 기준:
  1) `rm -rf web/node_modules && cd web && npm test --silent` 이 exit 0 이고, `src/utils/format.test.ts` + `src/App.test.tsx` 의 테스트가 **실제로 실행**되며 skip 0 (`-- --reporter=verbose` 로 테스트명을 눈으로 확인 — "no tests" 로 통과하는 것은 실패로 간주)
  2) 바로 재실행하면 설치를 건너뛰고 수 초 안에 다시 exit 0 — 가드가 설치돼 있을 때 no-op
  3) `git grep -n vitest -- Makefile .github/workflows web/package.json scripts/` 결과 vitest 를 **실행**하는 문장이 `web/package.json` 한 곳뿐이고 `Makefile:17`·`ci.yml:57` 은 `npm test --silent` 를 부른다
  4) `cd web && node node_modules/typescript/bin/tsc -b` exit 0 (CI 가 테스트 다음에 하는 것)
  5) `make lint`·`make test`·`go test ./...` 모두 exit 0 (Go 쪽 회귀 없음)

- 건드릴 파일 (프로덕션 3개, 신규 없음):
  - `web/package.json:10 scripts.test` — `"vitest run"` → vitest 를 패키지 경로 `node_modules/vitest/vitest.mjs` 로 직접 호출. (`package-lock.json:3603` 이 `"vitest": "vitest.mjs"` 를 bin 으로 선언하므로 이 경로가 맞다 — 확인함.) 인터프리터는 `"${npm_node_execpath:-node}"` 로 고정할 것 — 아래 "위험" 참고.
  - `web/package.json` `scripts.pretest` 신규 — `scripts/build-ui.sh:6` 과 **글자까지 같은** 가드 `[ -d node_modules ] || npm ci --no-audit --no-fund`
  - `Makefile:17` — `cd web && node node_modules/vitest/vitest.mjs run` → `cd web && npm test --silent`
  - `.github/workflows/ci.yml:57` — `node node_modules/vitest/vitest.mjs run` → `npm test --silent` (바로 위 56줄의 `npm ci` 는 **그대로 둘 것**; 가드가 no-op 이 된다)

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - 실패 재현(고치기 전): `cd web && npm test --silent` → 이번 회차에 직접 실행해 `sh: 1: vitest: not found` 확인
  - 기준1: `cd web && rm -rf node_modules && npm test --silent -- --reporter=verbose`
  - 기준4: `cd web && node node_modules/typescript/bin/tsc -b`
  - 기준5: `make lint` / `make test` / `go test ./...`

- 위험과 피할 것:
  - **`.github/workflows/release.yml` 은 건드리지 말 것.** 이번 회차에 전문(37줄)을 읽었는데 npm·web·테스트 단계가 **하나도 없다**(`verify-version.sh` → `package-offline.sh` → `verify-offline.sh` → `gh release upload` 뿐). 즉 "릴리즈 워크플로가 같은 이유로 두 번 실패" 는 `release.yml` 이 아니라 **러너 자신의 검증 단계**(`cd web && npm test --silent`)를 가리킨다. release.yml 에서 원인을 찾으려 하지 말 것.
  - **`npm_node_execpath` 고정에 대한 정정 — 과거 세 회차의 전제가 이번 환경에서는 성립하지 않는다.** 그 회차들은 npm 이 lifecycle 스크립트 PATH 앞에 모든 상위 디렉터리의 `node_modules/.bin` 을 붙이고 `/home/hkjang/node_modules/.bin/node` 가 v20.19.2 라서, 바레 `node` 로는 jsdom 30 의 undici 가 `markAsUncloneable` 없음으로 죽는다고 실측했다. 이번 회차에 확인해 보니 **그 경로가 존재하지 않는다**(`/home/hkjang/node_modules/.bin/` 자체가 비어 있거나 없음; PATH 의 `node` 는 v22.23.1). 그래서 고정은 지금은 no-op 일 수 있다. 그래도 **넣어 둘 것** — 비용이 0 이고 과거 세 번 실제로 터진 함정을 막는다. 다만 "고정이 없으면 실패한다" 는 주장을 검증 근거로 쓰지는 말 것(이번 환경에서는 뮤테이션으로 재현되지 않을 수 있다).
  - **느슨하게 통과시키기 금지**: `--passWithNoTests`, `|| true`, `continue-on-error`, 테스트 파일 제외, 워크플로 완화 — 전부 금지. 테스트가 실제로 실행되어 통과해야 한다.
  - `web/package-lock.json` 을 바꾸지 말 것(`npm ci` 만 쓰고 `npm install` 금지). `web/dist`·`internal/ui/dist` 같은 빌드 산출물을 커밋하지 말 것.
  - `web/package.json` 의 `build`·`dev`·`preview`·`screenshots` 는 이번 범위 밖(`tsc`/`vite` 의 `.bin` 의존이 남지만 CI·`build-ui.sh` 는 패키지 경로를 쓴다). 보류 아이디어로 남길 것 — 범위를 늘리지 말 것.
  - Go 쪽과 보호 경로(`internal/server/auth.go`·`oidc.go`·`keys.go`·`oauth.go`, `internal/store/migrations`, `scripts/package-offline.sh`, `VERSION`)는 건드리지 않는다.
  - **미확인 하나**: 이번 회차에 `npm ci` 를 직접 돌려 레지스트리 도달을 재확인하려 했으나 권한 거부로 실행하지 못했다. 과거 두 회차가 이 환경에서 성공(191 packages, ~9s)을 실측했으므로 성립할 것으로 보지만, 구현자는 기준1 을 돌릴 때 `npm ci` 가 레지스트리에 닿는지부터 확인할 것. 만약 오프라인이면 가드를 캐시 기반으로 바꾸지 말고(그건 과제의 본질이 아니다) 그 사실을 보고할 것.

- 차선 후보: `internal/mattermost` 의 남은 경로 테스트 — `FileContent` 의 `*[]byte` 우회, `PostList.Ordered()` 가 없는 id 를 조용히 버림, `New()` 의 깨진 CA PEM 거부. 배선은 `internal/server/integration_test.go:30 fakeMM` 관례대로 `httptest.Server` + 실제 `*mattermost.Client` 로 할 것(손으로 만든 인터페이스 대역은 이 저장소 관례가 아니다). 단 1순위가 러너의 검증 명령 자체를 막고 있으므로 1순위를 먼저 끝낼 것.
