- 과제: `cd web && npm test` 가 `node_modules/.bin` 심링크와 "이미 설치돼 있음" 을 전제해 깨끗한 체크아웃에서 항상 실패한다 — 저장소 전체가 쓰는 `node node_modules/<pkg>/…` + 설치 가드 방식으로 통일한다 (가치 4 / 위험 1 / 작업량 S)

- 왜: `web/package.json:9` 의 `"test": "vitest run"` 만 유일하게 `node_modules/.bin/vitest` 심링크에 의존하고, 설치 가드도 없다. 그래서 `web/node_modules` 가 없는 깨끗한 워크트리에서 `cd web && npm test --silent` 는 `sh: 1: vitest: not found` 로 exit 1 이고(아래 '재현' 참고), 마지막 회차가 이 검증으로 verify-failed 했다. 저장소의 나머지 전부(`Makefile:17`, `.github/workflows/ci.yml:57-59`, `Dockerfile:9`, `scripts/build-ui.sh:7-8`)는 `node node_modules/vitest/vitest.mjs run` 처럼 패키지 경로를 직접 호출하고 `scripts/build-ui.sh:6` 은 `[ -d node_modules ] || npm ci --no-audit --no-fund` 가드까지 갖고 있다 — `npm test` 와 `make test` 만 이 관례에서 빠져 있다.

- 재현 (정찰이 실제로 돌려 확인한 것):
  - `ls web/node_modules` → 없음(.gitignore 대상이므로 새 워크트리에는 항상 없다)
  - `cd web && npm test --silent` → `sh: 1: vitest: not found`, exit 1
  - 테스트 파일은 실제로 존재한다: `web/src/App.test.tsx`, `web/src/utils/format.test.ts` (즉 "테스트가 없어서" 가 아니다)
  - `make test:` 타깃(`Makefile:16-17`)도 설치 가드가 없어 같은 상황에서 깨진다 — `go test ./...` 다음 줄에서 `Cannot find module …/vitest.mjs` 가 된다(모듈 부재 메시지 문구는 미확인, 실패하는 것은 확실)

- 수용 기준:
  1) `web/node_modules` 를 지운 상태에서 `cd web && npm test --silent` 가 exit 0 으로 끝나고, 출력에 위 2개 테스트 파일이 **실행되어** 통과한 것이 보인다(0개 통과·skip 은 실패로 볼 것).
  2) 그 직후 같은 명령을 다시 돌리면 재설치 없이 바로 테스트가 돈다(가드가 멱등).
  3) `make test` 와 `.github/workflows/ci.yml` 의 web 테스트가 `npm test` 와 **같은 한 경로**를 쓴다(vitest 호출 문장이 저장소에 한 곳만 남는다). CI 는 그 앞에서 이미 `npm ci` 를 하므로 가드는 no-op 이어야 한다.
  4) `cd web && node node_modules/typescript/bin/tsc -b` 가 통과한다(CI 가 vitest 다음에 하는 관문).

- 건드릴 파일 (프로덕션 3개 + 선택 1개):
  - `web/package.json` `scripts` — `"test": "vitest run"` → `"test": "node node_modules/vitest/vitest.mjs run"`, 그리고 `"pretest": "[ -d node_modules ] || npm ci --no-audit --no-fund"` 추가. `pretest` 문장은 `scripts/build-ui.sh:6` 과 **글자까지 같게** 쓸 것. npm 은 스크립트를 `sh -c` 로 돌리므로 이 POSIX 문법이 동작하고, `npm ci` 는 루트의 `pretest` 를 다시 실행하지 않으므로 재귀하지 않는다.
  - `Makefile:17` — `cd web && node node_modules/vitest/vitest.mjs run` → `cd web && npm test --silent`. 바로 위 `go test ./...` 줄은 그대로.
  - `.github/workflows/ci.yml:57` — `node node_modules/vitest/vitest.mjs run` → `npm test --silent`. 같은 블록의 `npm ci --no-audit --no-fund`(56행), `tsc -b`(58행), `vite build`(59행) 줄과 `setup-node` 캐시 설정은 **그대로 둘 것**.
  - (선택) `README.md:107` 주변 — `make test` 만 적혀 있고 `npm test` 언급은 저장소 전체에 없다(grep 확인). 꼭 필요하면 `cd web && npm test` 한 줄 추가 정도. 다른 문서는 손대지 말 것(`scripts/verify-version.sh` 가 문서의 버전 문자열을 검사한다).
  - `.github/workflows/release.yml` 은 **건드리지 말 것**: 읽어 보니 `verify-version.sh` → `package-offline.sh` → `verify-offline.sh` → `gh release upload` 뿐이고 web 테스트를 호출하지 않는다. 두 번 실패한 것은 러너의 verify 단계이지 이 워크플로가 아니다.

- 검증 명령 (이 저장소에서 실제로 도는 것, 순서대로):
  1) `cd web && rm -rf node_modules && npm test --silent` — 고치기 **전에** 먼저 돌려 `sh: 1: vitest: not found` 를 재현하고, 고친 뒤 같은 명령이 exit 0 인지 볼 것 (수용 기준 1)
  2) `cd web && npm test --silent` — 두 번째 실행 (수용 기준 2)
  3) `cd web && node node_modules/typescript/bin/tsc -b` (수용 기준 4)
  4) `make lint` — `gofmt -l .` 빈 출력 + `go vet ./...` + `./scripts/verify-version.sh`
  5) `make test` — `go test ./...` + web 테스트 (수용 기준 3)
  - `go test ./...` 가 `ok … 0.0xs` 로 끝나면 통합 테스트는 skip 된 것이다(`internal/server/integration_test.go:192` 가 `TEST_POSTGRES_DSN` 없으면 `t.Skip`). 이번 과제는 Go 코드를 한 줄도 바꾸지 않으므로 그대로 둬도 된다.

- 위험과 피할 것:
  - **미확인 — 먼저 확인할 것: 이 환경에서 `npm ci` 가 npm 레지스트리에 닿는가.** 정찰은 샌드박스가 `npm ci` 실행을 막아 확인할 수 없었다. 닿지 않으면 `pretest` 가드는 "vitest not found" 를 "registry 접속 실패" 로 바꿀 뿐 수용 기준 1을 만족하지 못한다. 그 경우: (a) `scripts.test` 를 패키지 경로 호출로 바꾸는 부분만 그대로 적용하고(이것만으로도 `.bin` 의존은 사라진다), (b) `pretest` 는 넣되 그대로 두고, (c) 수용 기준 1을 달성할 수 없다는 사실과 실제 오류 출력을 회차 노트에 그대로 적을 것. **오프라인을 이유로 기준을 낮추거나 테스트를 건너뛰게 만들지 말 것.**
  - **미확인: vitest 5.x 가 `node_modules/vitest/vitest.mjs` 를 패키지 루트에 두는가.** `Makefile`·CI·`Dockerfile` 이 이미 이 경로를 전제하므로 맞을 가능성이 높지만, 설치 후 `ls web/node_modules/vitest/` 로 한 번 눈으로 확인할 것. 없으면 `node_modules/vitest/dist/cli.js` 등 실제 경로를 쓰고 `Makefile`·`ci.yml` 의 기존 문장도 같이 맞출 것(그 둘은 어차피 `npm test` 로 대체된다).
  - **워크플로를 느슨하게 해서 통과시키는 것 금지.** `continue-on-error`, `|| true`, `--passWithNoTests`, 테스트 파일 제외, `vitest.config.ts` 의 `test.include` 축소 — 전부 금지.
  - `package-lock.json` 을 재생성하지 말 것(`npm install` 이 아니라 `npm ci`). lock 이 바뀌면 CI 의 `npm ci` 와 `Dockerfile:6` 이 동시에 흔들린다. `git status` 에 `web/package-lock.json` 이 뜨면 되돌릴 것.
  - 보호 경로를 전혀 건드리지 말 것: `internal/server/{oauth,oidc,auth,keys,identity}.go`, `internal/server/tools.go:321 approve()`, `internal/store/migrations`, `VERSION`·`scripts/verify-version.sh`·`scripts/package-offline.sh`.
  - `npm test --silent` 의 `--silent` 는 npm 의 로그 레벨 플래그이고 vitest 로 전달되지 않는다. vitest 에 인자를 넘길 일이 생기면 `--` 가 필요하다(이번에는 없다).
  - 프런트 소스(`web/src`)는 수정 대상이 아니다. 건드리면 `tsc -b` 가 vitest 보다 먼저 깨진다.

- 차선 후보: `internal/logbuf` 링 버퍼·`Subscribe()` 해제 경로 테스트 추가 (가치 3 / 위험 1 / 작업량 S) — 129줄에 테스트 0개. `add()` 의 용량 초과 폐기, `Snapshot()` 순서, `Subscribe()` 가 돌려주는 해제 함수를 호출한 뒤 구독자 채널에 더 쓰지 않는지(누수·블로킹), `slog.Handler` 의 `WithAttrs`/`WithGroup` 을 `go test -race ./internal/logbuf/` 로. 네트워크가 막혀 1순위의 수용 기준 1을 끝낼 수 없더라도 1순위의 (a)(b)(c) 는 먼저 끝낸 뒤 이쪽을 추가할 것.
