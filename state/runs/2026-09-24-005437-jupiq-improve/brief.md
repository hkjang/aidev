# 정찰 과제서 (2026-09-24, main@c5b50b4, VERSION 1.7.2)

- 과제: 목록 API의 `search` 질의가 `%`·`_`·`\`를 와일드카드로 흘려보내는 것을 전역 검색과 같은 규칙(`searchPattern`)으로 이스케이프 (가치 3 / 위험 2 / 작업량 S)

- 왜: `internal/store/search.go:searchPattern`은 `\`·`%`·`_`를 이스케이프하고 `ILIKE $1 ESCAPE '\'`로 조회하는데, 목록 4곳(`users.go:142`, `hubs.go:630`, `hubs.go:733`, `settings.go:547`)은 `pattern := "%" + search + "%"`로 원문을 그대로 붙여 쓴다. 그래서 같은 문자열을 전역 검색창과 목록 필터가 다르게 읽는다 — 사용자 목록에서 `hong_gildong`을 찾으면 `_`가 임의의 한 글자와 맞아 `hong1gildong`까지 나오고, `_` 한 글자나 `%` 한 글자를 넣으면 필터가 걸린 척하면서 전체 행을 돌려준다(개수 쿼리와 데이터 쿼리 둘 다 같은 pattern을 쓰므로 total까지 함께 틀린다). 한 헬퍼로 모으면 전역 검색·목록 필터·MCP `jupiq.list_servers`가 같은 질의를 같게 읽는다.

- 수용 기준:
  1) `ListLocalUsers`·`ListManagedUsersWithAccess`·`ListServersWithAccess`·`ListResources`가 `%`·`_`·`\`를 리터럴로 취급한다. 예: `users`에 `hong_gildong`과 `hong1gildong`이 있을 때 `search=hong_gildong`은 전자만 돌려주고 `Page.Total`도 1이다.
  2) `search`가 빈 문자열이거나 공백뿐이면 지금처럼 필터가 걸리지 않는다(전체 목록). 이스케이프 결과가 `%%`가 되어 "전체"와 "아무것도 아닌 것"이 뒤바뀌지 않게, 빈 판정에 쓰는 값과 pattern을 만드는 값이 **같은 문자열**이어야 한다(둘 다 trim 후 값으로 통일할 것).
  3) 이스케이프 규칙이 `search.go:searchPattern`과 한 벌이다 — 새 헬퍼를 만들지 말고 `searchPattern`을 그대로 재사용하거나, 재사용이 어려우면 `searchPattern`을 목록에서도 쓸 수 있게 고쳐 한 함수만 남긴다(같은 규칙이 두 벌로 갈라지면 이번 수정의 의미가 없다).
  4) 테스트가 증명할 것: (a) 순수 단위 — `searchPattern`(또는 통합된 헬퍼)이 `%`·`_`·`\`·한글·공백을 어떻게 바꾸는지 표 기반으로. 이미 `search_test.go:TestSearchPatternEscapesLikeMetacharacters`가 있으므로 목록에서 쓰는 입력(빈 문자열, 공백뿐)을 더할 것. (b) 실제 PostgreSQL 통합 — 실제 행을 넣고 `ListLocalUsers`(권한 인자 없어 가장 단순)로 `hong_gildong`/`hong1gildong` 구분과 `search="_"`가 0건임을, 그리고 `Page.Total`이 `data` 길이와 맞음을 확인. 가짜 store·SQL 문자열 단언으로 대신하지 말 것.
  5) 수정을 되돌리면(이스케이프 제거) 통합 테스트가 실제로 빨개지는 것을 확인하고 기록에 남긴다.

- 건드릴 파일:
  - `internal/store/users.go:142` — `ListLocalUsers`의 `pattern := "%" + search + "%"`와 `$1=''` 비교에 쓰는 `search` 인자.
  - `internal/store/hubs.go:630` — `ListManagedUsersWithAccess`의 count(635행)·data(681행) 두 쿼리가 같은 pattern을 공유하니 둘 다.
  - `internal/store/hubs.go:733` — `ListServersWithAccess`의 count(738행)·data(744행). MCP `jupiq.list_servers`(`internal/api/mcp_handlers.go:155`)도 이 경로를 탄다.
  - `internal/store/settings.go:547` — `ListResources`의 count(549행)·data(553행).
  - `internal/store/search.go:searchPattern` — 재사용(필요하면 trim 위치만 조정). `GlobalSearch`의 동작은 바꾸지 말 것.
  - `internal/store/search_test.go` 또는 새 `*_test.go` — 순수 단위 표.
  - 새 `internal/store/list_search_integration_test.go`(가칭) — 기존 통합 테스트 관례(`os.Getenv("JUPIQ_INTEGRATION_TEST_DSN")` 없으면 `t.Skip`, 예: `internal/store/feature_write_gate_integration_test.go:15`)를 그대로 따를 것. 파일명·테스트명에 `Integration`이 들어가야 `make test-integration`의 `-run Integration`에 잡힌다.
  - SQL의 `ILIKE $n` 뒤에 `ESCAPE '\'`를 붙이는 것은 `search.go`와의 모양 맞추기용이다(PostgreSQL 기본 이스케이프 문자가 이미 `\`라 동작은 이스케이프 자체에서 나온다). 붙여도 되고 안 붙여도 되지만, 붙인다면 네 곳 모두 일관되게.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `go test -count=1 ./internal/store/ ./internal/api/` — 정찰에서 방금 통과 확인(`ok … store 0.006s`, `ok … api 0.050s`).
  - 통합(필수, DB 있을 때): `docker run --rm -d -p 55432:5432 -e POSTGRES_PASSWORD=jupiq -e POSTGRES_DB=jupiq --name jupiq-it postgres:16-alpine` → `export JUPIQ_INTEGRATION_TEST_DSN=postgres://postgres:jupiq@127.0.0.1:55432/jupiq?sslmode=disable` → `make test-integration`(내부적으로 `go test -count=1 -p=1 -run Integration ./internal/store ./internal/api`). 끝나면 컨테이너 제거.
  - `gofmt -l .`(무출력), `go vet ./...`, `go test -count=1 ./...`.
  - 릴리스 수준까지 볼 여유가 있으면 `make release-check`(수 분·네트워크 필요).
  - **미확인**: 이 정찰 세션에서는 `docker` 실행 권한을 얻지 못해 컨테이너 기동을 직접 확인하지 못했다. 도커가 없으면 통합 테스트는 skip될 뿐 실패하지 않으므로, 그 경우 "통합 미검증"을 결과에 명시하고 순수 단위 테스트만으로 완료를 주장하지 말 것.

- 위험과 피할 것:
  - **빈 검색 의미가 뒤집히는 것이 이 과제의 유일한 진짜 위험이다.** `searchPattern`은 `TrimSpace`를 하므로 `search="   "`가 pattern `%%`가 되는데, 쿼리의 `$n=''` 비교에 원문을 그대로 넘기면 "필터는 걸렸는데 전부 매치"가 된다. 비교 값과 pattern 값을 같은 trim 결과로 통일할 것(수용 기준 2).
  - `internal/auth/`, `migrations/`, `.github/workflows/`는 건드리지 말 것. 이번 과제는 `internal/store` 4개 함수 + 테스트로 끝난다.
  - `orderBy`/`pageBounds`/접근제어 SQL(`dataAccessSQL`·`countAccessSQL`)과 인자 번호($1,$2,…)를 함께 흔들지 말 것 — pattern 인자를 빼거나 순서를 바꾸면 접근제어 인자가 밀린다. 인자 개수는 유지하고 값만 바꾸는 것이 가장 안전하다.
  - 과거 교훈: "한쪽 경로만 넓히지 말 것" — 목록 4곳 중 일부만 고치면 같은 검색어가 화면마다 다르게 동작한다. 넷 다 고칠 것.
  - 과거 교훈: "출력이 실제로 바뀌지 않는 수정은 넣지 말 것" — 그래서 `ESCAPE '\'` 추가만으로 끝내지 말고 반드시 pattern 이스케이프를 넣어야 한다(기본 이스케이프 문자가 이미 `\`라 절 추가만으로는 아무것도 달라지지 않는다).
  - 프런트(`web/`)는 검색어를 그대로 보내므로 무변경. 가이드/PDF는 검색 메타문자 설명이 없어(확인함: `docs/*.md`에 관련 절 없음 — 다만 전수 확인은 아님) 손대지 않아도 된다. 굳이 문서를 고치겠다면 PDF 재굽기까지 함께 해야 하므로 이번 범위에서 빼는 편을 권한다.

- 차선 후보: **OpenAPI `page_size` 상한 불일치 정리** — `openapi/openapi.yaml`의 `/users`는 `maximum: 100`, `/audit`는 `200`인데 실제 `internal/store/store.go:pageBounds`는 200으로 보정한다. 문서가 거짓말을 하는 쪽(`/users`)을 실제 동작에 맞추거나, 반대로 핸들러가 문서대로 거절하게 만들되 어느 쪽인지 한 방향만 고르고 `internal/api/openapi_contract_test.go`가 계속 통과하는지 확인할 것. (그 다음: `internal/store` 순수 헬퍼 5개 표 기반 테스트 — 가치는 낮지만 위험도 거의 없다.)
