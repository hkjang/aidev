# 과제서 (2026-10-17, 기준 커밋 `2734aff` / v0.34.63)

- 과제: `make verify` 를 만들어 CI 의 게이트를 로컬에서 빠짐없이 돌리고, ci.yml 에 게이트가 늘어났는데 Makefile 이 안 따라오는 드리프트를 테스트로 막는다 (가치 4 / 위험 1 / 작업량 S)

- 왜: `Makefile:test`(실제 내용 확인함 — `go test ./cmd/... ./internal/...` / `go vet` / `cd sdk && npm run typecheck && npm test` / `cd web && npm run lint && npm test && npm run build`)에는 CI 가 돌리는 **`-race`·`govulncheck`·`npm ci`·`npm audit`·sdk 의 `npm run build`** 가 하나도 없다. 그래서 로컬에서 `make test` 녹색을 보고 올린 커밋이 CI 에서 빨간불을 만난다 — 이 저장소의 최근 verify-failed 두 건(2026-10-08 의 `x/text` 권고, 2026-10-16 의 표준 라이브러리 권고 10건)이 전부 `govulncheck` 단계였고, 그 단계는 `make test` 에 **없다**. 고치면 커밋 전에 같은 게이트를 한 명령으로 돌릴 수 있고, CI 에 게이트가 늘었을 때 로컬이 뒤처지는 것을 테스트가 알려 준다.

- 수용 기준:
  1) `make verify` 가 `.github/workflows/ci.yml` 의 `test` job 이 돌리는 다섯 명령을 **같은 글자·같은 순서**(Go → sdk → web; ci.yml 에서 sdk 가 web 보다 먼저다)로 실행하고, 전부 통과하면 exit 0 이다. 기존 `make test` 는 **지우지 않고 그대로** 둔다(빠른 루프용).
  2) 새 Go 테스트가 ci.yml 과 Makefile 을 둘 다 읽어 **드리프트일 때만** 빨간불이 된다. 판정은 "게이트가 최신인가"(읽지 않고 갱신되는 숫자가 된다)가 아니라 `internal/version/toolchain_test.go` 와 같은 **"두 선언이 일치하는가"** 다. 구체적으로:
     - 테스트 안에 CI 의 `run:` 단계 ↔ 그것을 덮는 Makefile 레시피 줄의 **표**를 둔다.
     - ci.yml 의 `run:` 단계 **개수**가 표의 항목 수와 다르면 실패한다(= CI 에 단계를 더하고 표를 안 고치면 빨간불). ci.yml 의 현재 `run:` 단계는 **6개**다 — Go 3개(`go test -race`, `go vet`, `go run …govulncheck@latest`), `working-directory: sdk` 의 npm 한 줄, `working-directory: web` 의 npm 한 줄, 그리고 `name: Build production image` 아래의 `docker build`.
     - 표의 각 항목이 가리키는 로컬 문자열이 `Makefile` 본문에 실제로 들어 있어야 한다.
     - `docker build` 단계는 **면제**로 적고 이유를 주석에 쓴다 — `make docker` 가 같은 이미지를 빌드하지만 `--build-arg VERSION`/`COMMIT` 값이 의도적으로 다르다(CI 는 `ci`·`${GITHUB_SHA::12}`, Makefile 은 `DECLARED_VERSION`·`git rev-parse --short`). 면제 항목은 `docker build` 가 Makefile 에 있는지만 본다.
  3) 지금은 두 선언이 일치하므로 **그 테스트가 녹색**이다. 빨간불을 눈으로 보려면 ci.yml 에 `- run: echo drift` 한 줄을 **로컬에서만** 더해 실패를 확인하고 `git checkout -- .github/workflows/ci.yml` 로 되돌릴 것 — ci.yml 변경을 커밋하면 안 된다.

- 건드릴 파일 (프로덕션 1 + 테스트 1):
  - `Makefile` — `.PHONY` 줄에 `verify` 추가, `test:` 아래에 `verify:` 타깃 신설. 레시피는 ci.yml 순서 그대로:
    ```
    go test -race ./cmd/... ./internal/...
    go vet ./cmd/... ./internal/...
    go run golang.org/x/vuln/cmd/govulncheck@latest ./cmd/... ./internal/...
    cd sdk && npm ci && npm audit && npm run typecheck && npm test && npm run build
    cd web && npm ci && npm audit && npm run lint && npm test && npm run build
    ```
    (make 는 레시피 **줄마다 새 셸**을 쓰므로 `cd sdk` 가 다음 줄로 새지 않는다 — 기존 `test:` 타깃이 이미 그 전제로 쓰여 있다. README 의 누적 `cd` 문제와 혼동하지 말 것: 그건 문서의 한 셸 블록이었다.)
    주석으로 두 가지를 적을 것: (a) `govulncheck` 와 `npm audit` 는 **네트워크와 살아 있는 권고 DB** 를 읽으므로 오프라인에서는 이 타깃이 실패한다 — 숨기지 말고 그대로 실패하게 둔다, (b) `MOMENTO_TEST_POSTGRES_DSN` 이 없으면 DB 통합 테스트는 **skip** 되므로 `make verify` 녹색이 CI 녹색을 보장하지 않는다(CI 는 `postgres:17-alpine` 서비스를 띄우고 그 DSN 을 env 로 준다).
  - `internal/version/buildgates_test.go` (새 파일) — 위 드리프트 가드. **이 패키지에 두는 이유**: 같은 패키지의 `docs_test.go`·`toolchain_test.go` 가 이미 저장소 루트의 파일(`docs/`, `go.mod`, `Dockerfile`)을 읽는 저장소 수준 메타 테스트이고, `version_test.go:148` 의 `func repoRoot(t *testing.T) string` 헬퍼를 **그대로 재사용**할 수 있다(실제로 확인함). 새 패키지를 만들지 말 것. `.go` 파일 중 ci.yml 을 읽는 것은 현재 **하나도 없다**(확인함).

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `go test ./internal/version/` — 현재 `ok … 0.004s`(실측). 새 테스트만 보려면 `go test ./internal/version/ -run BuildGate -v`.
  - `go vet ./cmd/... ./internal/...`
  - `make verify` — **끝까지 한 번 돌려 exit 0 을 찍을 것.** 오래 걸린다: `go test -race` 가 DB 없이도 수십 초, `govulncheck` 는 모듈 다운로드, `npm ci` 가 sdk·web 두 번. 몇 분을 예상할 것.
  - `make test` 가 여전히 도는 것도 확인(회귀 금지). `make -n verify` 로 레시피 전개를 먼저 눈으로 볼 것.
  - 빨간불 증명: ci.yml 에 `run:` 단계를 로컬에서 한 줄 더해 테스트 실패를 확인한 뒤 `git checkout -- .github/workflows/ci.yml`. 커밋 전에 `git status` 로 **Makefile 과 새 테스트 파일 2개만** 담긴 것을 확인할 것.

- 위험과 피할 것:
  - **`.github/workflows/*.yml` 은 읽기만 한다.** 게이트를 느슨하게 하거나(`continue-on-error`, `-skip`, `-race` 제거) 단계를 옮기지 말 것 — 지난 회차들이 지킨 선이다.
  - `make test` 를 `verify` 로 대체하거나 그 내용을 바꾸지 말 것. 둘은 다른 용도다(빠른 루프 vs 커밋 전 전체).
  - 가드를 **문자열 전체 동일성**으로 만들지 말 것 — `docker build` 처럼 인자가 의도적으로 다른 단계가 있어서 곧 거짓 실패를 낸다. 표 + 개수 비교로 하라는 것이 이 이유다.
  - 가드가 "sdk 의 `npm audit` 과 web 의 `npm audit` 을 구분한다" 고 **주장하지 말 것**. 부분 문자열 판정은 디렉터리별 동일성까지는 증명하지 않는다 — 테스트 주석에 그 한계를 정직하게 적을 것(과거 교훈: 소스 문자열 검사는 동작의 증거가 아니다).
  - `internal/auth`·`internal/database/migrations`·`docs/openapi.yaml`·릴리즈 워크플로는 이 과제와 무관하다. 건드리지 말 것.
  - YAML 파서 의존성을 새로 추가하지 말 것. ci.yml 은 **텍스트로 정규식**으로 읽는다(`toolchain_test.go` 가 go.mod·Dockerfile 에 쓴 방식과 같다). `go.mod` 에 새 의존성을 넣는 변경은 이 과제의 범위 밖이다.
  - 2026-10-08 에 같은 취지의 「make verify 진입점」 과제가 **no-change** 로 끝났다(구현자가 판정을 남기지 않았다). 그때 무엇이 막았는지는 **미확인**이다. 막히면 범위를 Makefile 한 타깃으로만 줄이고(가드 테스트는 다음 회차로) 무엇이 막았는지 반드시 적을 것.

- 차선 후보: **`cidrRule.ts` 의 호스트 비트 안내를 서버의 정규화 저장 결과로 고친다** (가치 2 / 위험 1 / S). 지금 `web/src/pages/cidrRule.ts:judgeCIDR` 의 마지막 분기는 `hint: \`대역의 시작 주소는 ${dotted(start)}/${bits} 입니다.\`` 를 돌려주고, 바로 위 주석은 "Postgres `cidr` 컬럼이 이것을 어떻게 다루는지는 확인하지 못했으니 아무 주장도 하지 않는다" 고 적혀 있다. 그 전제가 **이제 거짓**이다 — PR #32 가 머지되어(`eba5371`, 후속 `2865192`) `internal/httpapi/admin.go:820 normalizeCIDR` 이 `net.ParseCIDR` 이 계산한 대역을 그대로 저장하고 `createNetwork`(admin.go:847)가 그 값을 INSERT 한다. 그래서 안내가 결과를 말하지 않는다. 안내 문장을 "저장하면 …" 꼴로 고치고 주석의 '미확인' 서술을 `normalizeCIDR` 을 가리키게 고칠 것. 범위: `web/src/pages/cidrRule.ts` + `web/test/cidrRule.test.mjs`. 기존 테스트는 문장을 글자 그대로 보지 않고 `assert.match(judgement.hint ?? "", /10\.20\.30\.0\/24/)` 처럼 주소만 정규식으로 본다(확인함) — 그래서 문장 교체가 기존 단언을 깨지 않는다. 검증: `cd web && npm ci && npm run lint && npm test && npm run build`. `hint` 는 `blocking`·`error` 없이 두어 **저장을 막지 않는 것**을 유지할 것 — 서버가 받아 주는 값이다.
