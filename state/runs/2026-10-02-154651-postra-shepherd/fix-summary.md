# fix-summary — PR #31 govulncheck 실패

- 문제: `govulncheck ./...` 가 GO-2026-6505(otlptrace exporter 설정 로깅이 info 로그로 endpoint URL 유출)로 exit 3. 로컬에서 govulncheck@v1.6.0 으로 같은 자리(동일 호출 트레이스, "1 vulnerability from 1 module")를 재현했다. PR 의 web/ 변경과 무관한 신규 권고이며 go.mod 는 이 PR 이 건드리지 않았으므로 origin/main 도 같이 실패한다.
- 원인: `go.opentelemetry.io/otel/{exporters/otlp/otlptrace,otlptracehttp,sdk}` 가 v1.44.0 이고 수정은 v1.45.0 에 들어 있다.
- 수정: otel 계열(otel·trace·metric·sdk·otlptrace·otlptracehttp)을 v1.45.0 으로 올리고 `go mod tidy`. Go 코드·테스트·워크플로·`toolchain` 은 손대지 않았고 전이 의존(logr 1.4.4, proto/otlp 1.11.0, genproto)만 따라 올라갔다.
- 검증: `govulncheck ./...` exit 0 (v1.44.0 으로 되돌리면 다시 exit 3) · `go build ./...` · `go vet ./...` · `go test -race -count=1 ./...` 전부 통과 · `go run ./cmd/postra-contracts -check` · `make lint`(gofmt+gosec 0 issues).
- 못 돌린 것: PostgreSQL(`POSTRA_TEST_PG`)·브라우저 e2e·npm(이번 변경은 web/ 과 Go 소스를 바꾸지 않음).
