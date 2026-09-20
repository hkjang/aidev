# 과제서 2026-09-20 — jupiq

- 과제: 목록·지표 API의 정수 질의 파라미터(page·page_size·limit·hub_id)가 정수가 아니거나 음수이면 400 `invalid_query`로 거부 (가치 3 / 위험 2 / 작업량 S)
- 왜: `internal/api/helpers.go:queryInt`는 `strconv.Atoi` 실패를 조용히 기본값으로 바꾸므로 `GET /api/v1/users?hub_id=abc`는 "모든 Hub", `?page=abc`는 1쪽, `?page_size=-5`는 20건으로 응답한다 — 호출자는 오타를 알 수 없고 OpenAPI(`openapi/openapi.yaml` 317~319행, 546~559행: `minimum: 1`)와도 어긋난다. 같은 저장소의 `/users/{username}`은 이미 `boundedUserDetailQueryInt`(core_handlers.go:1276)로 잘못된 값을 `invalid_pagination` 400으로 돌려주므로, 그 규칙을 나머지 7개 핸들러(12개 호출)에 맞추면 API 전체가 한 가지 계약을 지킨다.

## 수용 기준
1) `GET /api/v1/users|servers|local-users|audit|{resources}?page=abc`(또는 `page_size=abc`, `hub_id=abc`, `page=-1`)가 400과 `{"error":{"code":"invalid_query","message":"<이름>은(는) 0 이상의 정수여야 합니다", …}}` 형태로 응답하고, 저장소(`s.Store`)는 호출되지 않는다.
2) 값이 비어 있으면 종전과 같은 기본값(page 1, page_size 20/50, limit 100/1000, hub_id 0), `0`·큰 값은 종전과 같이 store가 보정(`pageBounds`, `Metrics` 5000, `ResourceConsumption` 500) — 즉 정수이면서 0 이상인 입력의 응답은 단 하나도 바뀌지 않는다. 프런트(`web/src/components/ResourceListPage.tsx:143`, `pages/SettingsPage.tsx:213`, `ConsumptionPanel.tsx:29`)는 항상 숫자를 보내므로 화면 영향 없음.
3) `/usage/consumption?limit=abc`·`/metrics?limit=abc`도 같은 400. `/users/{username}`의 기존 `invalid_pagination` 동작·메시지는 그대로 둔다(기존 테스트 `TestUserDetailQueryBounds` 통과).
4) 테스트: (a) `helpers_test.go`(신규)에서 새 헬퍼가 빈 값→fallback, `"7"`→7, `"0"`→0, `"abc"`·`"-1"`·`" 3"`→error 를 표로 검증; (b) `Server{}`(Store nil)에 `httptest.NewRequest`로 `auditList`·`servers`(또는 users)·`metrics`·`usageConsumption` 핸들러를 직접 호출해 `?page=abc` 등이 store에 닿기 전에 400 `invalid_query`를 내는 것을 확인 — Store가 nil이므로 파싱을 통과하면 panic으로 드러난다(가짜 store 주입 금지); (c) 기존 통합 테스트 `TestScopedRBACHTTPFailClosedIntegration`(scoped_rbac_integration_test.go:109~112, `page=1&page_size=1`)가 DSN 환경에서 그대로 통과해 정상 값 경로가 프로덕션 `Handler()` 배선으로 무변경임을 증명.
5) `openapi/openapi.yaml`의 해당 목록·지표 경로에 `'400'` 응답(설명: 정수가 아니거나 음수인 page·page_size·limit·hub_id)을 추가하고 `TestOpenAPIDocumentCoversEveryRegisteredRoute` 등 계약 테스트 통과. 관리자 가이드·PDF는 건드리지 않는다(API 오류 코드 표가 가이드에 없음 — 미확인이면 `grep -n invalid_pagination docs/ADMIN_GUIDE.md`로 확인 후 없으면 그대로).

## 건드릴 파일
- `internal/api/helpers.go:queryInt` — 시그니처를 `queryInt(r, name string, fallback int) (int, error)`로 바꾸거나, 새 `queryIntParam`을 두고 기존 함수는 제거. 규칙: `""`→fallback,nil / `strconv.Atoi` 실패 또는 `< 0` → `fmt.Errorf("%s은(는) 0 이상의 정수여야 합니다", name)`. 반복을 줄이려면 `pageQuery(w, r, defaultSize) (page, size int, ok bool)` 같이 400을 직접 쓰는 헬퍼를 하나 더 두어도 됨(선택).
- `internal/api/core_handlers.go` — `usageConsumption`(423행 limit), `users`(1228행 page·page_size·hub_id), `localUsers`(1288행), `servers`(1300행 page·page_size·hub_id), `auditList`(1626행), `metrics`(1641행 limit): 오류 시 `apiError(w, r, http.StatusBadRequest, "invalid_query", err.Error())` 후 return. 파싱은 `boundedTimeRange` 검사 뒤·store 호출 전에.
- `internal/api/resource_handlers.go:49` — `ListResources` 호출의 page·page_size 동일 처리.
- `internal/api/core_handlers.go:1276 boundedUserDetailQueryInt` — 그대로 두거나 새 헬퍼 위에 min/max만 얹어 재구성(선택, 동작 불변 조건).
- `internal/api/helpers_test.go`(신규), 핸들러 400 테스트(기존 `resource_handlers_test.go` 또는 신규 `query_params_test.go`).
- `openapi/openapi.yaml` — 목록·지표 경로 `'400'` 응답 추가.

## 검증 명령
```
gofmt -l . && go vet ./... && go test -race ./internal/api/ ./internal/store/
go test ./...
cd web && npm test          # 프런트 무변경 확인용(선택)
JUPIQ_INTEGRATION_TEST_DSN=postgres://… make test-integration   # postgres:16-alpine 있을 때만(README "릴리스 전 로컬 검증" 절)
```
`make test-integration`은 DSN이 없으면 exit 1로 안내를 내므로, 컨테이너를 띄울 수 없으면 그 사실을 회차 노트에 적고 (b)·(a) 단위 테스트로 대신한다.

## 위험과 피할 것
- `internal/store/store.go:pageBounds`와 store 쪽 limit 보정(`metrics.go:724`, `resource_usage.go:230`)은 건드리지 않는다 — 0·상한 초과의 보정 동작을 바꾸면 "정수 입력의 응답 무변경" 조건이 깨지고 기존 클라이언트를 깰 수 있다. 상한(page_size 200 등)을 400으로 바꾸는 확장도 금지(OpenAPI 318행 max 100 vs 547행 max 200 불일치가 있어 이번 범위 밖).
- `hub_id=0`은 "모든 Hub"라는 기존 의미를 유지한다(음수만 거부).
- `internal/auth/*`, `migrations/`, `.github/workflows/*`, 설정 검증(`settings` 영역)은 무관 — 손대지 않는다.
- 가짜 Store·인터페이스 주입으로 테스트하지 말 것(운영자 규칙). nil Store로 "store 이전에 400"을, 통합 테스트로 "정상 값 무변경"을 증명한다.
- 미머지 브랜치 `auto/2026-09-16-0552`(메일)·`auto/2026-09-17-2243`(MCP OAuth)가 `core_handlers.go`·`server.go`를 넓게 건드리므로 충돌을 줄이려면 호출 줄 자체만 최소로 바꾸고 함수 이동·대규모 재배치는 하지 않는다.
- 커밋 메시지는 한국어 `fix: …` 관례. VERSION은 올리지 않는다(릴리즈 단계가 함).

## 차선 후보
- `internal/store` 순수 헬퍼 5개(`firstLabel`·`latestMetricSample`·`countBool`·`countRuntime`·`filterBool`) 표 기반 단위 테스트 추가 (가치 2 / 위험 1 / 작업량 S) — DB 불필요, `go test ./internal/store/`만으로 검증. 1순위가 기존 클라이언트 호환 문제로 성립하지 않을 때 고른다.
