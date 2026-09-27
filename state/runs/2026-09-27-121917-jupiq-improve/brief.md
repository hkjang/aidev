- 과제: 목록 `limit`이 상한을 넘으면 상한으로 자르지 않고 기본값으로 **줄어드는** 세 store 함수를 저장소 관례(pageBounds·GetUserDetail)대로 고친다 (가치 3 / 위험 1 / 작업량 S)

- 왜: `ListMailDeliveries`·`ResourceConsumption`·`Metrics` 세 곳이 `if limit > MAX { limit = 기본값 }` 형태라, 크게 요청할수록 **적게** 돌려준다 — `?limit=500`인 발송 기록은 200건이 아니라 50건, `?limit=1000`인 사용량은 500건이 아니라 100건, `?limit=10000`인 지표는 5000건이 아니라 1000건이다. 세 상한은 `openapi/openapi.yaml`이 각각 `maximum: 200`(638행)·`500`(207행)·`5000`(573행)으로 문서화한 값이고 같은 저장소의 `store.go:pageBounds`(195행)와 `user_detail.go:GetUserDetail`(132~134행)은 초과분을 상한으로 자르므로, 이 세 곳만 문서와 관례 양쪽에서 어긋나 있다.

- 수용 기준:
  1) `ListMailDeliveries(ctx, "", 500)`이 최대 200건을 돌려준다(현재 50건). 201건 이상 있을 때 `len(Items) == 200`.
  2) `ResourceConsumption(..., 1000)`이 최대 500건, `Metrics(..., 10000)`이 최대 5000건을 돌려준다.
  3) 종전 계약이 그대로다 — `limit` 미지정/0/음수는 여전히 각각 기본값 50·100·1000이고(`GET /metrics?limit=0`은 openapi가 `minimum: 0`이라 허용하므로 1000건 유지), 상한 **이하** 값(예: 200·500·5000·1)은 지금과 같은 결과를 낸다.
  4) 테스트가 "상한 초과가 기본값으로 떨어지는" 회귀를 잡는다: 자름 로직을 되돌리면(`= MAX` → `= 기본값`) 테스트가 실제로 빨개지는 것을 한 번 확인해 기록한다.
  5) `go vet ./...`·`go test -count=1 ./...`·`gofmt -l .`(무출력) 통과, OpenAPI 계약 테스트(`go test -run 'OpenAPI|UndocumentedRoute' ./internal/api`) 통과. openapi.yaml의 `maximum` 값은 **바꾸지 않는다** — 코드를 문서에 맞추는 과제다.

- 건드릴 파일 (프로덕션 3개 + 테스트):
  - `internal/store/store.go` — `pageBounds` 바로 옆에 작은 헬퍼를 하나 추가: `func boundedLimit(limit, fallback, max int) int { if limit < 1 { limit = fallback }; if limit > max { limit = max }; return limit }`. 주석은 한국어로 "왜"(크게 요청한 쪽이 적게 받지 않도록 초과는 상한으로 자른다)를 적을 것. 세 기본값은 모두 상한 이하라(50≤200, 100≤500, 1000≤5000) 순서 문제는 없다.
  - `internal/store/mail.go:ListMailDeliveries` 120행 — `if limit < 1 || limit > 200 { limit = 50 }` → `limit = boundedLimit(limit, 50, 200)`.
  - `internal/store/resource_usage.go:ResourceConsumption` 230행 — `if limit < 1 || limit > 500 { limit = 100 }` → `limit = boundedLimit(limit, 100, 500)`.
  - `internal/store/metrics.go:Metrics` 724행 — `if limit <= 0 || limit > 5000 { limit = 1000 }` → `limit = boundedLimit(limit, 1000, 5000)`. (`limit <= 0`과 `limit < 1`은 int에서 동일하다.)
  - `internal/store/page_bounds_test.go` 또는 새 `bounded_limit_test.go` — `page_bounds_test.go`와 같은 표 기반 단위 테스트(0·음수·1·기본값·상한·상한+1·아주 큰 값 × 세 조합). 이 파일은 DB 없이 돈다.
  - 통합 근거(권장): `internal/store/mail_integration_test.go`에 발송 기록 201건을 넣고 `ListMailDeliveries(ctx, "", 500)`이 200건임을 실제 PostgreSQL로 단언. 이전 회차들이 쓴 방법 — 임시 `docker run --rm -d -e POSTGRES_PASSWORD=… -p …:5432 postgres:16-alpine` 뒤 `export JUPIQ_INTEGRATION_TEST_DSN=…` → `make test-integration`, 끝나면 컨테이너 제거.

- 검증 명령:
  - `gofmt -l .` (무출력) / `go vet ./...`
  - `go test -count=1 ./internal/store ./internal/api` — 기준선은 오늘 실측으로 ok(store 0.009s, api 0.052s).
  - `go test -count=1 ./...`
  - DSN 있으면 `make test-integration` (= `go test -count=1 -p=1 -run Integration ./internal/store ./internal/api`), `-v`로 store 통합 SKIP 0건 확인. **DSN 없이 도는 `go test ./...`는 통합이 전부 skip이라 1)·2)의 증거가 되지 못한다** — 단위 테스트(헬퍼)로도 4)는 증명되므로 DSN이 없으면 그 사실을 그대로 적을 것.

- 위험과 피할 것:
  - `openapi/openapi.yaml`의 `maximum`을 손대지 말 것. 이번 변경은 코드를 이미 문서화된 계약에 맞추는 것이고, 문서를 코드에 맞추면 "크게 요청하면 적게 준다"가 계약으로 굳는다.
  - `internal/api/*`의 `queryIntOrReject` 호출부(mail_handlers.go:20, core_handlers.go:424·1692)는 **그대로 둘 것**. 비정수·음수 400 거부는 2026-09-20에 이미 정착한 계약이고, 상한 초과를 400으로 바꾸는 것은 별개의 계약 변경이다(저장소 관례는 조용히 자르는 쪽 — pageBounds).
  - `internal/store/user_detail.go:GetUserDetail`(132행)은 이미 올바르게 자른다. 여기는 `pageBounds`가 먼저 돌아 `<1`을 20으로 바꾼 **뒤** 100으로 자르는 다른 순서이므로 헬퍼로 갈아끼우지 말 것(기본값 의미가 달라진다).
  - `Metrics`의 `limit=0`은 openapi `minimum: 0`이 허용하는 값이다 — 0을 거부하거나 0건으로 만들면 계약 위반. 기본값 1000 유지.
  - `ListMailDeliveries`의 `Summary`/`Total`은 `limit`과 무관하게 전체를 GROUP BY로 세므로(mail.go:142~155) 이번 변경으로 값이 바뀌지 않아야 한다 — 통합 테스트에서 함께 확인하면 좋다.
  - 보호 경로(internal/auth/, migrations/, .github/workflows/)와 목록 접근제어 SQL(dataAccessSQL/countAccessSQL)은 건드리지 않는다. 이 과제는 그중 어느 것도 지나지 않는다.
  - 프런트(`web/src/pages/SettingsPage.tsx:489 MailDeliveriesCard`)는 `limit`을 아예 보내지 않고 "최근 50건입니다" 문구를 쓴다 — 기본값 50이 유지되므로 프런트는 무변경이다.

- 차선 후보: **발송 기록의 `Total`이 `status` 필터와 무관하게 전체 건수를 돌려주는 것**(`internal/store/mail.go:142` 집계 쿼리에 status 조건이 없다 — `?status=failed`로 걸러도 `Total`은 모든 상태의 합) — 다만 `MailDeliveryPage`의 주석("상태별 집계")상 의도일 수 있고 프런트가 `total`을 화면에 쓰는지 확인하지 않았다(미확인). 착수 전 의도부터 판정할 것. 그다음은 「OpenAPI page_size 상한 불일치 정리(/users 100 vs /audit 200 vs pageBounds 200)」.
