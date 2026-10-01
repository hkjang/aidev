- 과제: 서버 `writeJSON`이 인코딩 실패를 버려 **200 + 빈 JSON 본문**을 내보내는 것을 500 오류 봉투로 바꾼다 (가치 3 / 위험 2 / 작업량 S)
- 왜: `internal/api/helpers.go:29` `writeJSON`은 `w.WriteHeader(status)`를 **먼저** 하고 `_ = json.NewEncoder(w).Encode(value)`로 오류를 버린다. 그래서 인코딩이 실패하면 상태코드는 이미 200으로 확정됐고 본문은 비어 있어, 클라이언트는 "성공했는데 응답이 깨진" 상태를 받는다. 직렬화 전에 marshal 하면 실패를 아직 되돌릴 수 있는 시점에 알 수 있어 거짓 200 대신 저장소 관례의 `error.code` 봉투를 보낼 수 있다.

## 실제로 확인한 것 (추측과 구분)
- **증상 재현(확인)**: 지금 `writeJSON`과 똑같은 코드를 실제 `httptest.NewServer` + 실제 `http.Get`으로 돌려 `math.NaN()`이 섞인 `map[string]any{"data": rows}`를 넘겼다. 결과는
  `status=200 content-type="application/json; charset=utf-8" len=0 body=""` 이고 클라이언트 쪽 `json.Unmarshal`은 `unexpected end of JSON input`. (저장소 밖 `/tmp` 임시 모듈에서 실행했고 저장소는 건드리지 않았다.)
- **고칠 수 있다는 것(확인)**: 같은 값에 `json.Marshal`을 먼저 부르면 `json: unsupported value: NaN`을 **쓰기 전에** 돌려준다. 즉 marshal-first로 바꾸면 상태코드를 바꿀 여지가 남는다.
- **적용 범위(확인)**: `writeJSON` 호출자는 프로덕션에서 7곳뿐 — `helpers.go:36 data`, `helpers.go:40 list`, `helpers.go:46 apiError`, `core_handlers.go:1708·1748·1756`(대시보드 feature gate). `internal/api/ai_handlers.go`의 SSE 스트리밍은 `writeJSON`을 쓰지 않으므로 이 변경과 무관하다(grep 확인).
- **NaN이 응답까지 가는 실제 유입 경로: 확인하지 못했다.** 알려진 Prometheus 수집 경로는 `internal/integration/prometheus.go:81 validPrometheusSample`이 NaN·Inf·음수를 이미 거른다. `internal/api/resource_handlers.go:465 numberFromAny`의 `string` 분기는 `strconv.ParseFloat` 오류를 버려 `"NaN"`을 통과시키지만, 호출자 9곳 모두 즉시 `int64(...)`로 캐스팅하거나 비교에만 쓰므로 응답 맵에 NaN이 들어가지 않는다. 따라서 **이 과제는 "지금 터지는 버그"가 아니라 계약 방어**다. 정직하게 그렇게 다뤄라 — 유입 경로를 억지로 만들어 "실제 버그"라고 쓰지 마라.
- **왜 그래도 가치 3인가(확인)**: `internal/store/resource_usage.go:272-299`는 `cpu_core_seconds` 등 `double precision` 열의 합을 **생 `float64` 그대로** 응답 맵에 넣고, `internal/store/metrics.go:741-746`도 `metric_samples.value`를 그대로 넣는다. 가드는 수집 입구 한 곳(`validPrometheusSample`)에만 있고 출구에는 없다. 또 지난 회차에 프런트 `web/src/api/client.ts`가 바로 이 모양(200 + 빈 본문)을 `INVALID_RESPONSE` ApiError로 받아내게 고쳐졌다 — 서버 쪽 짝을 맞추는 일이다. 열이 `double precision NOT NULL DEFAULT 0`라 PostgreSQL이 `'NaN'`을 받아들인다는 것은 **미확인**(DDL은 `migrations/013_resource_usage_rollup.sql:23-29`에서 확인, DB로 실제 삽입은 안 해봤다).

- 수용 기준:
  1) `writeJSON`이 marshal → (성공 시) 헤더+본문, (실패 시) 500 + `{"error":{"code":"internal_error","message":...}}` 순서로 동작한다. 성공 경로의 바이트·상태코드·`Content-Type: application/json; charset=utf-8`은 **지금과 완전히 동일**하다(기존 테스트가 하나도 깨지지 않아야 한다).
  2) 실패 시 응답은 상태 500이고, 본문은 `error.code`/`error.message`를 가진 유효한 JSON이며 `json.Unmarshal`로 파싱된다. 빈 본문이 아니다.
  3) 새 테스트가 **수정 전에 빨갛고 수정 후에 초록**이어야 한다: `internal/api` 패키지 안에서 `data(rec, 200, math.NaN())`, `list(rec, []any{math.Inf(1)}, store.Page{})`, 중첩 구조 안의 NaN(`map[string]any{"cpu_core_seconds": math.NaN()}` — `ResourceConsumption` 응답 모양과 같게), 지원하지 않는 타입(`func(){}` 또는 `chan int`)을 각각 호출해 `(상태, 본문)`을 단언한다. 수정 전 기대 실패 모양은 `status=200 body=""`.
  4) 성공 경로 회귀 단언: `data`/`list`/`apiError`가 평범한 값에 대해 지금과 같은 JSON(줄바꿈 포함 여부까지)을 낸다. `json.NewEncoder(...).Encode`는 끝에 `\n`을 붙이므로 **marshal-first로 바꿀 때 이 개행을 유지**해야 기존 바이트 단위 기대와 어긋나지 않는다. 바꾸기 전에 `json.NewEncoder` 출력과 `json.Marshal` 출력의 차이(개행)를 테스트로 먼저 고정해라 — 여기가 유일한 함정이다.
  5) 되돌림 확인: 실패 분기를 지우면 3)의 새 테스트가 다시 빨개진다.
- 건드릴 파일 (프로덕션 1개):
  - `internal/api/helpers.go:writeJSON` — `payload, err := json.Marshal(value)`로 바꾸고, `err != nil`이면 500 + 고정 오류 봉투 바이트를 쓰고 반환. 성공이면 헤더 설정 → `WriteHeader(status)` → `w.Write(payload)` + 개행. 왜 marshal을 먼저 하는지 한국어 한 줄 주석(저장소 관례)으로 남긴다.
  - `internal/api/helpers_test.go`(또는 새 `internal/api/write_json_test.go`) — 위 표 기반 테스트. `httptest.NewRecorder()`를 쓰는 기존 관례를 따른다(`helpers_test.go`가 이미 `httptest`를 쓴다).
  - 그 외 파일은 건드리지 마라. `data`/`list`/`apiError` 시그니처·`core_handlers.go` 호출부·`openapi/openapi.yaml`·프런트는 무변경.
- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `gofmt -l .` (무출력이어야 한다)
  - `go vet ./...`
  - `go test -count=1 ./internal/api`  ← 1차 증거. DSN 없이 돌고 통합 테스트는 skip 된다(정상).
  - `go test -count=1 ./...`
  - `go test -count=1 -run 'OpenAPI|UndocumentedRoute' ./internal/api` (계약 테스트가 안 깨졌는지)
  - 여유가 있으면 `JUPIQ_INTEGRATION_TEST_DSN=...` 지정 후 `make test-integration` — `apiError`/`data`를 타는 기존 통합 테스트(`mail_deliveries_integration_test.go`, `usage_group_by_integration_test.go`)가 성공 경로 무변경을 증명한다. DSN 없으면 이 명령은 실패하니 그때는 생략하고 그렇다고 적어라.
- 위험과 피할 것:
  - **개행 1바이트**가 이번 과제의 전부다. `Encode`는 `\n`을 붙이고 `Marshal`은 붙이지 않는다. 기존 테스트나 프런트가 본문을 바이트로 비교하는 곳이 있으면 여기서 깨진다 — 개행을 유지하는 쪽을 기본으로 삼아라.
  - 메모리: `Encode`도 내부적으로 전체를 버퍼링하므로 marshal-first가 최대 메모리를 눈에 띄게 늘리지 않는다. 그래도 `/metrics?limit=5000` 같은 큰 응답이 있으니 `io.Writer`로 흘리는 방식으로 "최적화"하려 들지 마라 — 그러면 과제 목적(쓰기 전에 실패를 안다)이 사라진다.
  - 보호 경로를 건드리지 마라: `internal/auth`, `migrations/`, `.github/workflows/`, SMTP·권한 범위 SQL. 이 과제는 그 어느 것도 필요 없다.
  - SSE/스트리밍(`ai_handlers.go`)과 SPA 핸들러(`spa_test.go` 대상)는 `writeJSON`을 쓰지 않는다. 함께 "통일"하려 들지 마라.
  - 과거 교훈: 대역(fake) Store나 소스 문자열 검색을 결함 증거로 삼지 마라. 실제 함수를 실제 `ResponseWriter`로 호출해 상태코드와 본문을 단언해라.
  - 유입 경로가 확인되지 않았다는 사실을 커밋 메시지·PR 본문에서 숨기지 마라. "계약 방어 + 거짓 200 제거"로 쓰면 정확하다.
- 차선 후보: **OpenAPI page_size 상한 불일치 정리** — `openapi/openapi.yaml`의 `/users` `page_size` `maximum: 100`과 `internal/store/store.go:pageBounds`의 상한 200이 다르다(불일치 자체는 이전 회차가 기록, 이번에 실제 HTTP 경계는 미확인). 구현자는 먼저 `?page_size=150`을 실제로 쏴 보고 어느 쪽이 사실인지 정한 뒤, 코드가 아니라 문서를 맞출지 반대로 할지 한 파일에서 끝내라.
