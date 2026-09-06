
## 2026-09-02
- 선택: API surface audit 복구 + 미문서화 라우트 86건 카탈로그 등록 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `cmd/api-surface-audit` 가 리브랜딩으로 사라진 `cmd/vibe/main.go`·`sdk/typescript/vibe.ts` 를 읽으며 경고 후 빈 문자열을 반환해, CLI/SDK 계약 검사가 0건 기준으로 항상 통과하는 무력 상태였다. 소스 경로를 정정하고 읽기 실패를 즉시 실패로 바꿨으며, mux 에 등록됐지만 `apiEndpoints` 에 없어 `/openapi.json` 에서 빠져 있던 라우트 86건을 핸들러가 실제 허용하는 메서드대로 추가했다. 같은 불변식을 `go test ./...` 안의 회귀 테스트(라우트 커버리지·죽은 항목·항목 형식)로 고정했고, 가짜 라우트를 넣어 가드가 실제로 실패하는 것까지 확인했다. 검증: `go build ./...`, `go test ./...` 전체 통과, `go run ./cmd/api-surface-audit` gap 0 종료.
- 보류 아이디어: GitHub Actions CI 부재(build/vet/test/api-surface-audit 게이트 추가) / `go vet ./...` 잔여 4건(테스트의 err 미검사 3건, json 태그 중복 1건) 정리 / `gofmt -l` 미정렬 64개 파일 일괄 포맷 / `parseWindow` 가 음수·과대 duration 을 그대로 받아 미래 시각 since 를 만드는 엣지케이스 / `internal/dataworks` 패키지 테스트 커버리지 보강
- 릴리즈: v0.9.36 (2026-09-02)

## 2026-09-03
- 선택: GitHub Actions CI 게이트 추가 + `go vet ./...` 잔여 4건 정리(그중 1건은 실제 응답 버그) (가치 4 / 위험 1 / 작업량 M)
- 결과: 성공
- 요약: 저장소에 워크플로가 하나도 없어 README와 `cmd/api-surface-audit` 주석이 전제하는 검사들이 아무것도 강제되지 않았다. `.github/workflows/ci.yml` 에 Go 잡(build·vet·test·api-surface-audit)과 Web 잡(npm ci·lint·test·build)을 추가하고 README 에 백엔드 검증 명령을 문서화했다. vet 를 하드 게이트로 만들기 위해 잔여 4건을 정리했는데, `admin_k8s_collect_slo.go` 의 json 태그 중복은 실제 버그였다: `failView` 가 `store.K8sCollectRun` 과 `analyzer.CollectGap` 을 같은 깊이로 임베드해 두 `category` 태그가 충돌, encoding/json 이 `recent_failures[].category` 를 통째로 누락시켜 Collect Gap RCA 화면에 원인이 표시되지 않았다. 분류 필드를 평탄하게 명시하고 회귀 테스트를 추가했으며, 옛 구조로 되돌려 테스트가 `category=nil` 로 실제 실패하는 것까지 확인했다. 검증: `go build ./...`·`go vet ./...`(0건)·`go test ./...` 전체 통과, `go run ./cmd/api-surface-audit` gap 0, web 에서 `npm ci && npm run lint && npm test && npm run build` 전부 통과(vitest 14건).
- 보류 아이디어: `gofmt -l` 미정렬 64개 파일 일괄 포맷(현재 CI 게이트 제외) / `parseWindow` 가 음수·과대 duration 을 그대로 받아 미래 시각 since 를 만드는 엣지케이스(30여 개 엔드포인트 영향) / `internal/dataworks` 패키지 테스트 커버리지 보강 / Playwright e2e 를 서비스 컨테이너 기반 CI 잡으로 편입 / `npm run build` 가 추적 파일 `web/dist/.gitkeep` 을 삭제하는 문제를 vite 설정으로 해결
- 릴리즈: v0.9.37 (2026-09-03)

## 2026-09-03 (2)
- 선택: `parseWindow` 가 음수·0 duration 을 그대로 받아 미래 시각 since 를 만드는 엣지케이스 수정 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `internal/proxy/admin_analytics.go` 의 `parseWindow` 가 `time.ParseDuration` 결과를 부호 검사 없이 적용해 `?window=-24h`·`?window=0s` 같은 값이 since 를 현재 시각 이후로 밀어냈고, 이 함수를 쓰는 70여 개 호출부(analytics·cost·MCP·personalization·xview 등)가 기본 윈도우로 폴백하는 대신 빈 데이터를 반환했다. 양수 lookback 만 허용하고 그 외에는 호출부 fallback 을 쓰도록 고쳤으며, 유일하게 fallback 0("무제한" 의도)을 넘기는 `mcpFilterFromQuery` 를 위해 비양수 결과는 store 계층 관례대로 zero time 을 반환하게 했다. 검증: 단위 테스트 2개 + `/admin/timeseries?window=-24h` HTTP 회귀 테스트를 추가하고 옛 코드로 되돌려 3개 모두 실제 실패하는 것을 확인, `go build ./...`·`go vet ./...`·`go test ./...` 전체 통과, `go run ./cmd/api-surface-audit` gap 0.
- 보류 아이디어: `gofmt -l` 미정렬 64개 파일 일괄 포맷(현재 CI 게이트 제외) / `internal/dataworks` 패키지 테스트 커버리지 보강 / Playwright e2e 를 서비스 컨테이너 기반 CI 잡으로 편입 / `npm run build` 가 추적 파일 `web/dist/.gitkeep` 을 삭제하는 문제를 vite 설정으로 해결 / `parseWindow` 의 미사용 `bucket` 인자 제거(호출부 70곳 일괄 정리)
- 릴리즈: v0.9.38 (2026-09-03)

## 2026-09-03 (3)
- 선택: `tokenSet` 이 한글 단어를 전부 구분자로 버려 한국어 제품의 고객 적합도 점수가 낮게 나오는 버그 수정 (가치 4 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: `internal/dataworks/domain.go` 의 `tokenSet` 이 `[a-z0-9_]` 이외 모든 룬을 구분자로 취급해, 한글로 작성된 `name_ko`·`description`·`pain_points` 는 토큰이 하나도 생성되지 않았다(실측 `tokenSet("여신 승인 신용 위험 스코어") == map[]`). 그 결과 `ComputeCustomerFitScore` 의 "shared positioning terms" 항목(최대 +35 및 `product_positioning`·`segment_pain_points` evidence ref)이 한국어 우선 제품에서는 세그먼트 pain point 가 글자 그대로 일치해도 절대 발동하지 않았고, 동일 내용의 영어 제품은 70+, 한국어 제품은 66 을 받았다. 구분자 판정을 `unicode.IsLetter`/`IsDigit` 기반으로 바꾸고 최소 토큰 길이를 바이트가 아닌 룬 수로 세도록 해(1음절 조사·단일 ASCII 문자를 동일하게 제외) ASCII 동작은 그대로 유지했다. 검증: 한국어 적합도 회귀 테스트(무관한 세그먼트는 여전히 더 낮게 나오는지 포함)와 `tokenSet` 단위 테스트를 추가하고 옛 코드로 되돌려 둘 다 실제 실패하는 것을 확인, `go build ./...`·`go vet ./...`(0건)·`go test ./...` 전체 통과, `gofmt -l internal/dataworks` 클린, `go run ./cmd/api-surface-audit` gap 0.
- 보류 아이디어: `gofmt -l` 미정렬 64개 파일 일괄 포맷(현재 CI 게이트 제외) / Playwright e2e 를 서비스 컨테이너 기반 CI 잡으로 편입 / `npm run build` 가 추적 파일 `web/dist/.gitkeep` 을 삭제하는 문제를 vite 설정으로 해결 / `parseWindow` 의 미사용 `bucket` 인자 제거(호출부 70곳 일괄 정리) / `bestApprovalStatus` 는 `ExpiresAt` 파싱 실패를 expired 로, `expiredAt` 은 not-expired 로 처리하는 불일치 정리
- 릴리즈: v0.9.39 (2026-09-03)

## 2026-09-03 (4)
- 선택: 데이터 상품 접근 창(access window) 타임스탬프 검증 부재와 admin/runtime 판정 불일치 수정 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `POST /admin/dataworks/products/{key}/contract-scopes` 와 `.../entitlements` 는 `valid_from`·`valid_to`·`expires_at` 를 형식 검사 없이 그대로 저장했는데(바로 옆 approval trace 핸들러는 RFC3339 검증을 함), 런타임 게이트 `entitlementActive`/`contractScopeActive` 는 파싱 실패를 inactive 로 처리한다. 그래서 `"2026-12-31"` 같은 자연스러운 입력이 200 으로 저장된 뒤 `/v1/data-products/{key}/query` 가 전부 403(`inactive_entitlement`/`contract_scope_inactive`)이 되었다. 반대로 admin 쪽 `entitlementExpired`·`dataworks.expiredAt` 은 파싱 실패를 "만료 아님" 으로, action center 는 파싱 불가 `valid_to` 를 아예 건너뛰어, 런타임이 거부하는 규칙이 화면에서는 정상으로 보이고 retirement 점수에서도 활성 사용으로 집계됐다. 쓰기 경로에 기존 approval trace 와 동일한 메시지·에러코드(`invalid_valid_to` 등) 검증을 추가하고 trim 후 저장하며, admin 판독기 3곳을 런타임과 같은 fail-closed 로 맞춰 레거시 행이 만료로 노출되게 했다. 검증: HTTP 회귀 테스트 2개(형식 거부, action center 노출)와 도메인 단위 테스트 1개를 추가하고 옛 동작으로 되돌려 각 단언이 실제로 실패하는 것을 확인, `go build ./...`·`go vet ./...`(0건)·`go test ./...` 전체 통과, 수정 파일 `gofmt -l` 클린, `go run ./cmd/api-surface-audit` gap 0.
- 보류 아이디어: `gofmt -l` 미정렬 64개 파일 일괄 포맷(현재 CI 게이트 제외) / Playwright e2e 를 서비스 컨테이너 기반 CI 잡으로 편입 / `npm run build` 가 추적 파일 `web/dist/.gitkeep` 을 삭제하는 문제를 vite 설정으로 해결 / `parseWindow` 의 미사용 `bucket` 인자 제거(호출부 70곳 일괄 정리) / 계약·엔타이틀먼트 만료 임박 기준(30일 고정)을 쿼리 파라미터로 노출
- 릴리즈: v0.9.40 (2026-09-03)

## 2026-09-04
- 선택: Contract Scope `rate_limit`(계약별 분당 호출 한도)이 런타임에서 전혀 강제되지 않던 문제 수정 (가치 5 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `dw_contract_scopes.rate_limit` 은 저장되고, 런타임 응답 `rate_limit` 필드로 반환되고, `BuildDynamicOpenAPIDocument` 가 상품 OpenAPI 에 `429 Rate limit exceeded` 로 광고까지 하는데 `POST /v1/data-products/{key}/query` 는 값을 읽기만 할 뿐 검사하지 않아, 분당 60회로 계약한 고객도 무제한 호출이 가능했고 `dw_usage_metering.over_limit_calls` 는 설계만 되고 영원히 0 이었다. 계약 키별 고정 1분 창(벽시계 분 경계 정렬) 카운터를 `internal/proxy/dataworks_ratelimit.go` 에 추가해 허용 호출에는 `X-DataWorks-RateLimit-{Limit,Used,Reset}` 를, 초과 호출에는 `429 contract_rate_limited` + `Retry-After` 를 반환하게 했다. 거부된 호출은 창을 소모하지 않고 `rate_limit_exceeded:<contract>` 로 감사 로그에 남으며 `failed_calls` 가 아니라 `over_limit_calls` 로 집계된다(`IncrementUsageMetering` 에 `overLimit` 파라미터 추가). 음수 `rate_limit` 은 런타임에서 "무제한" 으로 읽히므로 쓰기 경로에서 `400 invalid_rate_limit` 으로 거부하도록 했다. 검증: 리미터 창 롤오버·Retry-After 올림 단위 테스트와 HTTP 회귀 테스트 2개(3번째 호출 429·메터링 집계, 음수 입력 거부)를 추가하고 enforcement 를 꺼서 둘 다 실제 실패하는 것을 확인, `go build ./...`·`go vet ./...`(0건)·`go test ./...` 전체 통과, 수정 파일 `gofmt -l` 클린(기존 CRLF 파일 제외), `go run ./cmd/api-surface-audit` gap 0. README 에 Contract Rate Limit 절 추가.
- 보류 아이디어: `gofmt -l` 미정렬 64개 파일 일괄 포맷(현재 CI 게이트 제외) / Playwright e2e 를 서비스 컨테이너 기반 CI 잡으로 편입 / `npm run build` 가 추적 파일 `web/dist/.gitkeep` 을 삭제하는 문제를 vite 설정으로 해결 / 런타임 entitlement scope 검사가 `strings.Contains(scope,"query")` 라 `no-query` 같은 값도 통과하는 문제 / 계약·엔타이틀먼트 만료 임박 기준(30일 고정)을 쿼리 파라미터로 노출
- 릴리즈: v0.9.41 (2026-09-04)

## 2026-09-05
- 선택: 런타임 entitlement `scope` 를 부분 문자열로 검사해 조회 권한 없는 계약도 통과하던 문제 수정 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `internal/proxy/dataworks_runtime.go` 의 `POST /v1/data-products/{key}/query` 게이트가 `strings.Contains(strings.ToLower(ent.Scope), "query")` 로 권한을 판정해, `no-query`·`query:denied`·`data_product:subquery` 처럼 조회를 허용하지 않거나 명시적으로 거부하는 scope 가 그대로 통과해 상품 데이터가 반환됐다(fail-open). 반대로 문서화된 와일드카드 `data_product:*` 는 `*` 와 정확히 같지도 "query" 를 포함하지도 않아 403 `scope_denied` 로 막혔다. scope 를 쉼표·세미콜론·파이프·공백 구분 권한 목록으로 읽고 각 권한을 `*`·`query`·`data_product:*`·`data_product:query` 와 정확히 비교하는 `entitlementAllowsQuery` 로 분리했으며, 빈 scope 는 필드 도입 이전 행을 위해 기존대로 무제한 유지했다(다른 판독부는 없어 admin 화면과의 불일치 없음). 검증: 권한 판정 테이블 단위 테스트와 HTTP 회귀 테스트(거부 마커 403 `scope_denied`, 와일드카드 200)를 추가하고 옛 substring 로직으로 되돌려 둘 다 실제 실패하는 것을 확인, `go build ./...`·`go vet ./...`(0건)·`go test ./...` 전체 통과, `go run ./cmd/api-surface-audit` gap 0, 수정 파일 gofmt 상태는 변경 전과 동일(기존 CRLF 파일). `docs/OPERATIONS.md` 에 허용 권한 목록을 명시하고 v0.9.42 로 릴리즈 정렬(web 변경은 e2e/capture 픽스처의 버전 문자열뿐이라 node_modules 부재로 웹 체크는 미실행).
- 보류 아이디어: `gofmt -l` 미정렬 64개 파일 일괄 포맷(현재 CI 게이트 제외) / Playwright e2e 를 서비스 컨테이너 기반 CI 잡으로 편입 / `npm run build` 가 추적 파일 `web/dist/.gitkeep` 을 삭제하는 문제를 vite 설정으로 해결 / 계약·엔타이틀먼트 만료 임박 기준(30일 고정)을 쿼리 파라미터로 노출 / `parseWindow` 의 미사용 `bucket` 인자 제거(호출부 70곳 일괄 정리)
- 릴리즈: v0.9.42 (2026-09-05)
- 릴리즈: v0.9.42 (2026-09-05)
## 2026-09-06
- 선택: 런타임이 강제하지 않는 계약 `masking_policy` 를 마스킹 근거로 인정하던 문제 수정 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `applyMasking` 이 구현한 정책은 `redact`·`hash` 뿐이고 그 외 값은 `default` 분기에서 원본 값을 그대로 반환하는데, `POST /admin/dataworks/products/{key}/contract-scopes` 는 `masking_policy` 를 형식 검사 없이 저장했고 민감 상품 Publish Gate 는 비어 있지 않고 `none` 이 아니면 무조건 마스킹 구성으로 집계했다. 그래서 `"개인 단위 원천값 제외"` 같은 서술형 문구나 `redact_pii` 같은 오타를 저장하면 `masking_configured` 가 통과해 민감 상품이 게시되지만 `POST /v1/data-products/{key}/query` 응답은 마스킹 없이 원본 샘플 값을 그대로 내보냈다(fail-open). 쓰기 경로에서 런타임이 실제로 이해하는 `none`(빈 값 포함)·`redact`·`hash` 만 허용하고 그 외에는 `400 invalid_masking_policy` 로 거부하며 소문자·trim 정규화 후 저장하도록 했고, Publish Gate 판독기도 `applyMasking` 이 값을 실제로 바꾸는 정책만 세도록 맞춰 레거시 서술형 행이 `masking_configured=false`(`missing_evidence: masking_policy`) 로 노출되게 했다. 검증: HTTP 회귀 테스트 2건(쓰기 거부+정규화 저장, Publish Gate 판정)과 정책 분류 단위 테스트, 허용 목록과 `applyMasking` 동기화 테스트를 추가하고 옛 동작으로 되돌려 두 회귀 테스트가 실제로 실패하는 것을 확인, `go build ./...`·`go vet ./...`(0건)·`go test ./...` 전체 통과, `go run ./cmd/api-surface-audit` gap 0, web 에서 `npm ci && npm run lint && npm test && npm run build` 전부 통과(vitest 14건). `docs/OPERATIONS.md` 에 허용 값을 명시하고 v0.9.43 으로 릴리즈 정렬.
- 보류 아이디어: `gofmt -l` 미정렬 64개 파일 일괄 포맷(현재 CI 게이트 제외, 대부분 CRLF) / Playwright e2e 를 서비스 컨테이너 기반 CI 잡으로 편입 / `npm run build` 가 추적 파일 `web/dist/.gitkeep` 을 삭제하는 문제를 vite 설정으로 해결(이번 회차에도 재현·수동 복원) / 계약·엔타이틀먼트 만료 임박 기준(30일 고정)을 쿼리 파라미터로 노출 / `parseWindow` 의 미사용 `bucket` 인자 제거(호출부 70곳 일괄 정리)
- 릴리즈: v0.9.43 (2026-09-06)

## 2026-09-07
- 선택: 한 API 키가 같은 상품에 엔타이틀먼트를 여러 개 가질 때 활성 행 대신 최근 행을 평가하던 문제 수정 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `dw_api_entitlements` 는 `id` 만 PRIMARY KEY 라 admin POST 마다 새 행이 생겨 한 API 키가 같은 상품에 여러 엔타이틀먼트를 가질 수 있는데(만료된 체험 옆에 발급한 갱신, 감사용으로 남긴 `revoked` 행), `store.FindAPIEntitlement` 는 `ORDER BY updated_at DESC LIMIT 1` 로 마지막에 쓰인 행 하나만 읽었다. 그래서 활성 계약이 revoked/만료 행보다 먼저 쓰였다면 유효한 권한이 있는데도 `POST /v1/data-products/{key}/query` 가 403 `inactive_entitlement` 로 막혔고, 활성 계약이 둘이면 어느 쪽 `allowed_fields`·`rate_limit`·과금으로 처리될지 임의였다(신규 두 행은 `updated_at` 이 같은 값으로 저장될 수 있어 순서가 비결정적). 활성·미만료 행을 우선하고 그중 가장 최근 것을 고르도록 바꿨으며, 활성 행이 없으면 종전처럼 최근 행을 돌려줘 게이트가 `missing_entitlement` 가 아닌 `inactive_entitlement` 로 응답하게 유지했다. 활성 판정은 `store.EntitlementActive`(파싱 불가 만료일은 만료로 취급) 하나로 모으고 런타임 게이트 `entitlementActive` 가 이를 위임하게 해 선택 규칙과 게이트 규칙이 어긋나지 않도록 했다. 검증: 스토어 선택 테스트(활성 우선, 활성 부재 시 최근 행 반환)·`EntitlementActive` 표 테스트·HTTP 회귀 테스트(만료 행이 나중에 쓰여도 200 이고 활성 계약 키로 처리)를 추가하고 활성 우선 로직을 꺼서 두 회귀 테스트가 실제로 실패하는 것을 확인, `go build ./...`·`go vet ./...`(0건)·`go test ./...` 전체 통과, 수정·신규 파일 `gofmt -l` 클린, `go run ./cmd/api-surface-audit` gap 0. `docs/OPERATIONS.md` 5절에 선택 규칙을 문서화했다. web 변경이 없어 웹 체크는 미실행이고, 직전 회차(v0.9.43)가 아직 main 에 병합되지 않아 버전 충돌을 피하려 이번 회차는 릴리즈 커밋 없이 수정 커밋만 남겼다.
- 보류 아이디어: Playwright e2e 를 서비스 컨테이너 기반 CI 잡으로 편입 / `npm run build` 가 추적 파일 `web/dist/.gitkeep` 을 삭제하는 문제를 vite 설정으로 해결 / 계약·엔타이틀먼트 만료 임박 기준(30일 고정)을 쿼리 파라미터로 노출 / `contractResponseFields` 가 `allowed_fields` 와 다른 대소문자로 요청한 필드를 요청 표기 그대로 응답(계약 표기로 정규화 필요) / `internal/dataworks` 도메인 함수(EvaluatePublishGateV2·EvaluateRetirementCandidate) 단위 테스트 보강

