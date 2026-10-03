# fix-summary — git-ctx PR #44

- 문제: CI 의 `Known vulnerabilities`(govulncheck) 만 실패. 원인은 이 PR 이 아니다 — `git diff origin/main...HEAD -- go.mod go.sum` 가 비어 있고, 취약점 DB 가 **GO-2026-6505**(OpenTelemetry OTLP trace exporter 가 info 로그에 엔드포인트 URL 을 흘림, `go.opentelemetry.io/otel/exporters/otlp/otlptrace@v1.44.0`, fixed in v1.45.0)를 새로 실었다. main 과 go.mod 가 동일하므로 main 도 같은 실패를 낸다.
- 재현: `govulncheck ./...` 로컬 실행 → `Vulnerability #1: GO-2026-6505 … Found in …otlptrace@v1.44.0 / Fixed in …@v1.45.0`, exit 3. CI 로그의 71개 trace 와 같은 호출 경로(`internal/observability/manager.go`, `internal/search/service.go`).
- 고친 방법: otel 계열(otel·trace·sdk·otlptrace·otlptracehttp)을 v1.45.0 으로 올리고 `go mod tidy`. 코드·테스트는 한 줄도 안 고쳤다(PR 의 budget.go 변경은 무관하고 그대로 둠).
- 검증: `govulncheck ./...` → `No vulnerabilities found.` (exit 0). `gofmt -l ./cmd ./internal` 빈 출력, `go build -tags sqlite_fts5 ./...`, `go vet ./...`, `go test -tags sqlite_fts5 -count=1 ./...` 전체 통과, `-race ./internal/observability ./internal/mcp` 통과, `scripts/verify-version-sync.sh` 통과. 못 돈 것: 외부 DB/Vault 통합 잡, `test/store/upgrade.test.sh`, 전체 `-race ./...`.
- 딸린 전이 업데이트: logr v1.4.4, proto/otlp v1.11.0, genproto api/rpc(tidy 결과). 릴리즈 노트에 보안 수정 한 줄이 필요하다.
