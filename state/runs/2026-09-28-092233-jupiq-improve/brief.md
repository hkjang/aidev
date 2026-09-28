- 과제: `/usage`가 같은 `group_by`를 두 경로에 넘기면서 `consumption`만 조용히 `user` 그룹으로 떨어지는 것을 응답에 드러낸다 (가치 3 / 위험 1 / 작업량 S)

- 왜: `GET /api/v1/usage?group_by=project`는 `trend`·`server_trend`를 project로 묶어 주지만(`internal/store/settings.go:1033`·`1059`가 `project`를 지원), 같은 핸들러가 같은 값을 그대로 넘기는 `ResourceConsumption`은 `project`를 모르고 말없이 `user`로 되돌린다(`internal/store/resource_usage.go:220-229`) — 한 응답 안에서 추세는 프로젝트별, `consumption`은 사용자별인데 응답에는 그 사실이 어디에도 없다. 프런트가 프로젝트 필터를 걸면 실제로 `group_by=project`를 보내므로(`web/src/utils/usage.ts:13,18`) 도달 가능한 경로다. `resource_usage_hourly`에 project 열이 없어(migrations/013_resource_usage_rollup.sql:11-20) project 소비량은 만들 수 없으니, 고칠 것은 "어떤 그룹으로 집계된 소비량인지"를 응답이 스스로 말하게 하는 것이다.

- 수용 기준:
  1) `GET /api/v1/usage?group_by=project` 응답에 `consumption_group_by`가 있고 값이 `"user"`다 — 같은 응답의 `trend[].group`은 project 라벨이라 두 그룹이 다르다는 것이 호출자에게 보인다(지금은 구분 불가).
  2) `group_by=department`(또는 `hub`·`network`·`user`) 응답은 `consumption_group_by`가 요청값과 같고, `consumption` 행 내용은 지금과 완전히 같다.
  3) `group_by`를 주지 않았거나 문서 밖 값(`group_by=nonsense`)이면 `consumption_group_by == "user"`이고 `consumption` 내용은 지금과 같다 — 400으로 바꾸지 않는다(문서화된 계약은 `/usage`가 5개 enum, `/usage/consumption`이 4개이며 이 과제는 계약을 좁히지 않는다).
  4) `GET /api/v1/usage/consumption`은 응답·상태코드 모두 무변경(`group_by=project`도 계속 200 + user 그룹). 기존 `internal/store/resource_usage_integration_test.go`·`list_limit_integration_test.go` 단언이 그대로 통과해야 한다.
  5) 테스트가 증명할 것: 실제 `store.Store`와 실제 `Server` 핸들러(실제 PostgreSQL)로 `/api/v1/usage?group_by=project`를 호출해 (a) 수정 전에는 `consumption_group_by`가 없어 실패하고, (b) 수정 후 `"user"`이며, (c) 같은 응답의 `trend` 라벨이 project 값임을 한 테스트에서 함께 단언 — 손으로 만든 대역 Store 금지.

- 건드릴 파일 (프로덕션 2개):
  - `internal/store/resource_usage.go:ResourceConsumption` — 함수 안의 `columns` 맵을 패키지 수준으로 올리고, 그 맵을 유일한 근거로 쓰는 노출 함수 하나를 더한다(권장: `func ConsumptionGroupBy(groupBy string) string` — 아는 값이면 그대로, 모르면 `"user"`). `ResourceConsumption`은 같은 맵/같은 함수를 쓰게 해 두 곳이 갈리지 않게 한다. SQL·인자·limit(`boundedLimit(limit,100,500)`)·반환 형식은 무변경.
  - `internal/api/core_handlers.go:usage`(390-411행) — 이미 읽고 있는 `groupBy := r.URL.Query().Get("group_by")`(404행)로 `effective := store.ConsumptionGroupBy(groupBy)`를 구해 `result["consumption_group_by"] = effective`를 넣는다. `ResourceConsumption` 호출 인자(405행)는 그대로 둔다 — store가 이미 같은 표로 되돌리므로 값이 어긋날 수 없다. `ResourceConsumption`이 에러면(지금처럼 Warn 후 `consumption` 생략) `consumption_group_by`도 넣지 말 것.
  - 테스트(새 파일 권장): `internal/api/usage_group_by_integration_test.go` — `internal/api/metrics_feature_gate_integration_test.go`·`dashboard_feature_gate_integration_test.go`가 실제 Store+핸들러를 띄우는 선례다. 필요하면 `internal/store/resource_usage_test.go` 성격의 DB 없는 표 기반 단위 테스트로 `ConsumptionGroupBy`의 5+1 경우(user·hub·network·department·project·빈값·nonsense)를 함께 고정(`bounded_limit_test.go`가 같은 모양의 선례).

- 검증 명령:
  - `gofmt -l .` (무출력), `go vet ./...`, `go test -count=1 ./...`
  - `go test -count=1 -run 'OpenAPI|UndocumentedRoute' ./internal/api`
  - 통합: 임시 DB를 띄운 뒤 `export JUPIQ_INTEGRATION_TEST_DSN=…` → `make test-integration` (= `go test -count=1 -p=1 -run Integration ./internal/store ./internal/api`). 이전 회차들이 쓴 방식: `docker run --rm -d -e POSTGRES_PASSWORD=… -p 5433:5432 postgres:16-alpine` 후 컨테이너 제거. **DSN 없이 도는 `go test ./...`는 통합이 전부 skip되므로 증거가 아니다** — `-v`로 SKIP 0건을 확인할 것.
  - 오늘 정찰 시점 사실: `go test -count=1 ./internal/store ./internal/api`는 DSN 없이 통과(통합 skip). VERSION 1.8.4, base main@944e82b.

- 위험과 피할 것:
  - **`ResourceConsumption`에 project 그룹을 추가하려 하지 말 것** — `resource_usage_hourly`에 project 열이 없어 마이그레이션(보호 경로)과 롤업 수집기까지 번진다. 이번 회차 범위 밖이고 파일 수가 터진다.
  - `openapi/openapi.yaml`은 손대지 않아도 된다 — `/usage`의 200은 `$ref: Success`(186-187행)로 자유 형식이라 키 추가가 계약 위반이 아니다. 굳이 문서화하려면 인용/들여쓰기를 깨지 말 것(695591e가 같은 자리에서 깨졌고 지금은 `-run OpenAPI` 테스트가 파싱까지 본다).
  - 프런트(`web/src/utils/usage.ts`·`buildUsageOverlay`·`ConsumptionPanel.tsx`)는 무변경 — `consumption`을 읽는 화면 요소는 현재 없으며(2026-09-28 grep 기준 `usage_stats`로만 실림) 이번 과제는 API 응답의 자기서술성만 고친다.
  - `/usage/consumption`의 문서화된 enum(openapi 206행: user·hub·network·department)을 넓히거나 400으로 좁히지 말 것.
  - `internal/store/settings.go:Usage`의 SQL(문자열로 조립되는 `groupExpr`·`serverGroupExpr`)은 건드리지 말 것 — 허용 맵 밖 값이 SQL에 닿지 않게 막아 둔 자리다.

- 차선 후보: `/mail/deliveries`의 `status`가 문서화된 enum 밖 값을 조용히 삼키는 것을 저장소 관례대로 400 `invalid_query`로 거부 — openapi 637행이 `enum [queued, sent, failed]`로 문서화했는데 `internal/api/mail_handlers.go:24`는 원문을 그대로 store로 넘기고, `internal/store/mail.go:124`의 `WHERE $1='' OR status=$1`은 `?status=Sent`·`?status=bogus`를 "0건"으로 답하면서 `Total`·`Summary`는 전체 집계를 돌려준다(빈 목록 + 큰 Total). 프로덕션 파일 1개(`mail_handlers.go`), 검증은 `internal/store/mail_integration_test.go` 옆에 핸들러 테스트 추가.
