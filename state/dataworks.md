
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
