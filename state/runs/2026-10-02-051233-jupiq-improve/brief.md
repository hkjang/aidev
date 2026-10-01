# 과제서 (2026-10-02 정찰, base main@92ff88c = tag v1.8.7)

## 0단계 (필수, 먼저): 배정된 `npm run lint` 실패를 재현해 사실을 확정한다

배정된 우선 과제는 "마지막 회차가 `npm run lint` (exit 1) 로 verify-failed" 다. **정찰은 이것을 재현하지 못했다 — 이 환경에서 `npm`·`gh` 실행이 권한으로 거부됐다(`npm ci`, `npm run lint`, `gh run list` 모두 "requires approval"). 그래서 실패 여부는 미확인이다.** 다만 아래는 실제로 확인했다:

- `web/node_modules` 가 이 작업 트리에 **없다**(`ls -d node_modules` → No such file or directory). 설치 없이 `npm run lint`(= `tsc -b --pretty false`)를 돌리면 타입 검사에 닿기도 전에 0 이 아닌 코드로 끝난다.
- 저장소 루트에 `package.json` 이 **없다**(루트 `ls` 결과: Dockerfile, LICENSE, Makefile, README.md, SECURITY.md, VERSION, cmd, compose.example.yaml, docs, go.mod, go.sum, internal, migrations, openapi, scripts, web). 루트에서 `npm run lint` 를 돌리면 ENOENT 로 exit 1 이다. CI 는 `web` 에서 돌린다(`.github/workflows/ci.yml:62-67` `defaults.run.working-directory: web`, 87-88행 "타입·린트 검사: npm run lint").
- base 의 web **소스는 PR #30 머지 시점과 동일하다**: `git log --name-only -- web` 으로 보면 92ff88c(v1.8.7 릴리즈 커밋)는 `web/package.json`·`web/package-lock.json` 의 버전만 바꿨고, 소스를 바꾼 마지막 커밋은 84ebd9d(`web/src/api/client.ts`, `client_json_body.test.ts`)다.
- 그 트리에 **tag v1.8.7 이 붙어 있고**(`git tag --points-at 92ko…` → `v1.8.7`, origin/main 과 같은 커밋) `.github/workflows/release.yml:81-86` 은 깨끗한 러너에서 `npm ci` → `npm run lint` → `npm test` → `npm run build` 를 전부 돌린다. 원장에 v1.8.7 릴리즈가 기록돼 있으므로 이 트리에서 `npm run lint` 는 GitHub 러너에서 통과했을 가능성이 높다(원격 run 로그는 `gh` 차단으로 미확인).
- `tsBuildInfoFile` 은 `./node_modules/.tmp/tsconfig.app.tsbuildinfo`(`web/tsconfig.app.json:3`)이라 `npm ci` 가 캐시를 지운다 — "로컬은 증분 캐시로 통과, CI 는 실패" 가설은 성립하지 않는다.

그러므로 0단계는 이렇게 한다:

1. `npm --prefix web ci` 후 `npm --prefix web run lint` 를 실행하고 **출력 전문을 회차 노트에 남긴다.**
2. **실패하면** tsc 가 가리킨 파일 1개를 고쳐 통과시킨다(그것이 이번 회차 과제다). `tsconfig*.json` 의 `strict`·`noUnusedLocals`·`noUnusedParameters`·`skipLibCheck` 를 끄거나 `include` 에서 파일을 빼는 식의 통과는 금지 — 워크플로·설정 느슨화는 반려 사유다.
3. **통과하면**(정찰의 예상) 그 사실을 증거(명령 + 출력)로 적고 2026-09-29 회차와 같이 아래 1단계 과제를 이번 회차의 작업으로 구현한다. 재현 없이 소스를 "예방적으로" 고치지 마라.

---

## 1단계 (0단계가 통과했을 때의 이번 회차 과제)

- 과제: `page_size` 상한이 OpenAPI(`maximum: 100`)와 실제 동작(`pageBounds` 200)으로 갈린 목록 엔드포인트를, **실제 HTTP 응답으로 어느 쪽이 사실인지 먼저 정한 뒤** 한쪽만 맞춘다 (가치 2 / 위험 2 / 작업량 S)
- 왜: 같은 계약 안에서 `page_size` 상한이 두 값으로 문서화돼 있고(`openapi/openapi.yaml:324` `maximum: 100` vs `:559` `maximum: 200`), 코드는 `internal/store/store.go:208 pageBounds` 에서 **200** 으로만 자른다(215-217행). 문서가 100 이라고 말하는 경로에 `?page_size=150` 을 보내면 150건까지 돌아오므로 클라이언트가 문서를 믿을 수 없다. 여섯 회차째 차선에만 올라 있었고 실제 경계 응답은 한 번도 확인되지 않았다.
- 확인된 것 / 미확인:
  - 확인: `pageBounds`(store.go:208-227)는 `pageSize < 1 → 20`, `pageSize > 200 → 200` 으로 자르고 `maxInt/pageSize` 로 page 를 클램프한다. `internal/store/users.go:141 ListLocalUsers`·`:864`·`user_detail.go:132`·`hubs.go:629`·`:733`·`settings.go:556` 이 이 함수를 공유한다.
  - 확인: `/users` 핸들러는 `internal/api/core_handlers.go:1326 (*Server).localUsers` 이고 1327행에서 `pageQuery(w, r, 20)`(`internal/api/helpers.go:109-115`, `queryIntOrReject` 로 잘못된 정수·음수를 400 `invalid_query` 로 거부)을 쓴 뒤 `ListLocalUsers` 를 호출한다.
  - **미확인(구현자가 먼저 확정할 것)**: `openapi.yaml:324` 의 `maximum: 100` 이 붙은 operation 은 `operationId: listHubUsers`("Hub별 통합 사용자와 현재 실행 자원 조회", 315-330행)다. 그 **path 문자열과 그 path 의 핸들러 함수**는 정찰이 확인하지 못했다(`internal/api` 에 `ListHubUsers` 호출이 grep 되지 않았다). 1단계의 첫 일은 `openapi.yaml:324` 위쪽에서 그 path 를 읽고, 라우터에서 그 path 의 핸들러가 `pageQuery` → `pageBounds` 를 타는지 **코드로 따라가 확인**하는 것이다. 타지 않는다면(별도 상한을 가진다면) 이 과제는 성립하지 않으니 차선 후보로 넘어가라.
- 수용 기준:
  1. 실제 PostgreSQL + 실제 `store.Store` + 실제 `New(...).Handler()` 관리자 세션으로 그 path 에 `?page_size=150` 을 호출해 **수정 전 응답**(`meta.page_size` 와 반환 행 수)을 테스트 실패 메시지로 기록한다. 대역 Store·소스 문자열 검색은 증거로 쓰지 않는다(fixture 는 150건 이상 필요).
  2. 그 사실에 맞춰 **한쪽만** 고친다. 문서가 틀렸다면 `openapi/openapi.yaml:324` 의 `maximum: 100` → `200` 한 토큰만(`/audit`:559 가 이미 200 이고 `pageBounds` 도 200 이므로 문서가 외톨이라는 쪽이 정찰의 예상이다). 코드를 상한 100 으로 내리는 쪽을 고른다면 `pageBounds` 를 공유하는 여섯 호출처의 동작을 바꾸지 말고 그 핸들러에만 상한을 주되, 왜 그 엔드포인트만 100 인지 한국어 주석으로 남겨라.
  3. 수정 후 같은 테스트가 통과하고, 되돌리면 다시 실패하는 것을 확인한다.
  4. 기존 `page_size <= 상한` 동작과 기본값 20, 잘못된 정수의 400 `invalid_query` 는 무변경임을 같은 테스트 안에서 고정한다.
  5. OpenAPI 계약 테스트가 깨지지 않는다.
- 건드릴 파일 (프로덕션 2개 이내):
  - `openapi/openapi.yaml:324` — `page_size` 의 `maximum` (문서가 틀렸을 때) **또는** 해당 핸들러 1곳(`internal/api/core_handlers.go` 또는 `resource_handlers.go`) — 둘 다 고치지 말고 하나만.
  - `internal/api/<해당>_integration_test.go` (신규 또는 기존 통합 테스트에 서브테스트 추가) — 경계 증거.
- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `gofmt -l .` (무출력), `go vet ./...`, `go build ./...`
  - `go test -count=1 ./internal/api ./internal/store`
  - `go test -count=1 -run 'OpenAPI|UndocumentedRoute' ./internal/api`
  - 임시 DB: `docker run -d --name jupiq-it -e POSTGRES_DB=jupiq_test -e POSTGRES_USER=jupiq -e POSTGRES_PASSWORD=it -p 5434:5432 postgres:16-alpine` → `export JUPIQ_INTEGRATION_TEST_DSN='postgres://jupiq:it@127.0.0.1:5434/jupiq_test?sslmode=disable'` → `make test-integration` (= `go test -count=1 -p=1 -run Integration ./internal/store ./internal/api`). 끝나면 컨테이너 제거. DSN 없는 `go test ./...` 통과에는 통합 skip 이 포함되므로 `-v` 로 SKIP 0건을 확인하라.
  - 0단계 때문에 web 도 돌려야 한다: `npm --prefix web ci`, `npm --prefix web run lint`, `npm --prefix web test`, `npm --prefix web run build`.
- 위험과 피할 것:
  - `pageBounds` 자체를 바꾸지 마라 — 여섯 호출처(users 2, user_detail, hubs 2, settings)가 공유하고 `internal/store/page_bounds_test.go`·`list_limit_integration_test.go` 가 200 을 고정한다. 상한 자르기 과제(2026-09-27)는 이미 머지됐으니 되돌리지 마라.
  - `openapi.yaml` 은 들여쓰기·인용에 민감하다(과거 반복 실패 지점). 인라인 `{ ... }` 형식을 그대로 유지하라.
  - `.github/workflows`·`internal/auth`·`migrations` 는 건드리지 않는다. 문서(`docs/*.md`)와 PDF·VERSION 정합을 깨지 않는다 — `./scripts/check-version.sh` 로 확인.
  - 통합 테스트 fixture 는 고유 marker 로 넣고 지우며, 전역 `total` 을 fixture 개수로 단정하지 마라.
  - 150건 fixture 를 만들 때 `generate_series` 를 쓰되 유니크 제약(username·email)을 marker + i 로 피하라 — `list_limit_integration_test.go:36-44` 가 같은 패턴을 쓴다.
- 차선 후보: `internal/store` 순수 헬퍼 5개(`firstLabel`·`latestMetricSample`·`countBool`·`countRuntime`·`filterBool`) 표 기반 테스트 — 일곱 회차 연속 차선이다. 1단계가 성립하지 않으면(위 "미확인" 이 무너지면) 이것을 하고, 끝나면 보류 목록에서 내려라.
