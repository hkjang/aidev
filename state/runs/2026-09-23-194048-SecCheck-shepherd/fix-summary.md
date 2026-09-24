# 수리 요약 — 커밋 없음 (지적이 이 변경의 결함이 아님)

- CI 실패는 재현했다: HEAD f4df4c2 에서 `govulncheck ./...` → exit 3, `GO-2026-6452`, 트레이스 `internal/store/seed.go:66:25` — CI 로그와 글자까지 같다.
- 그런데 **이 PR 이 만든 실패가 아니다**: 같은 스캔을 기준점 `origin/main`(eb2a3b0) 워크트리에서 돌려도 exit 3, 같은 취약점·같은 트레이스 줄로 실패한다. PR diff 는 `internal/web/mcp.go` + `internal/web/mcp_report_test.go` 뿐이고 `internal/store/`·`go.mod`·`go.sum` 변경은 0건이다.
- **고칠 업스트림이 없다**: govulncheck 이 `Fixed in: N/A` 라 하고, `go list -m -versions github.com/xuri/excelize/v2` 의 마지막이 이미 고정된 `v2.11.0` 이다. 올릴 버전 자체가 존재하지 않는다.
- 초록으로 만드는 길은 (a) ci.yml 의 vuln 단계에 무시/허용을 넣기 — 검증 명령 수정 금지에 걸림, (b) 무관한 `store.ExtractWorkbookDefaults` 의 excelize 호출 제거 — 이 PR 범위 밖의 기능 변경. 둘 다 절대 규칙 위반이라 손대지 않고 커밋 없이 끝낸다.
- 이 PR 자체는 건강하다: 실 PostgreSQL 16 으로 CI 와 같은 `go test -race -coverprofile=coverage.out ./...` exit 0(전 패키지 ok, internal/web 197.199s), `go vet ./...` exit 0, `TestMCPReviewReportIgnoresTheRequestQueryString` 는 SKIP 이 아니라 1.03s PASS. 작업 트리 clean, HEAD 그대로 f4df4c2.
