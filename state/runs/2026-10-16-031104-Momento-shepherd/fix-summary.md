# 수리 요약 — Momento PR #32

- 문제: CI `test` 잡이 `govulncheck` 단계에서 exit 3. PR 의 CIDR 수정과 무관하고, 고정된 `toolchain go1.26.7` 의 표준 라이브러리 권고 10건(`net/http` 8, `net/textproto` 1, `crypto/tls` 1 — GO-2026-6603/6605/6607/6608/6609/6610/6611/6612/6613/6617)이 **모두 호출 경로가 확인되어** 걸린 것이다. 전부 go1.26.9 에서 수정됐다.
- 재현: pre-fix 트리에서 CI 와 같은 `go run golang.org/x/vuln/cmd/govulncheck@latest ./cmd/... ./internal/...` 를 돌려 같은 10건과 CI 로그의 같은 추적줄(`internal/service/automation.go:506:16`, `internal/secret/cipher.go:155:24`)을 재현했다.
- 수정: `go.mod` 의 `toolchain` 과 Dockerfile 빌더 이미지를 1.26.7 → 1.26.9 (`go 1.26.0` 지시자와 의존성은 그대로). 선례 5a9e55e 가 같은 두 자리를 함께 올렸고, 빌더 이미지를 두면 릴리즈 바이너리에 취약한 stdlib 가 남는다.
- 검증(go1.26.9): govulncheck exit 0 (호출되는 취약점 0), `go vet` exit 0, 실제 postgres:17-alpine DSN 을 준 `go test -race ./cmd/... ./internal/...` 전 패키지 ok — PR 의 CIDR 테스트 7개 모두 skip 없이 PASS. sdk·web `npm audit` 0건. Docker 빌드는 `golang:1.26.9-bookworm` 매니페스트 확인 후 로컬 재빌드.
- 손대지 않은 것: `admin.go`·`network_cidr_test.go`(PR 본체), 워크플로·검증 명령·단언 일절 무변경.
