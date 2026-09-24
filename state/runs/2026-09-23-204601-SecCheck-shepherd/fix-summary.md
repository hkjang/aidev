# 수리 요약 — 커밋 없음 (지적이 이 변경과 무관)

- CI `test-build-scan` 실패 원인은 `Go vulnerability scan` 단계 하나뿐(exit 3): GO-2026-6452, `github.com/xuri/excelize/v2@v2.11.0`, 도달 경로 `internal/store/seed.go:66 ExtractWorkbookDefaults → excelize.File.GetRows`. 로컬 `govulncheck ./...` 로 HEAD(f4df4c2)에서 동일 재현.
- **기준선에서도 동일하게 실패**: `origin/main` 을 별도 워크트리로 체크아웃해 같은 명령을 돌려 동일 결과 + RC=3. 이 PR 의 diff 는 `internal/web/mcp.go`, `internal/web/mcp_report_test.go` 2개뿐이며 `go.mod`·`internal/store` 를 건드리지 않았다 → 이 변경이 만든 실패가 아니다.
- **고칠 수 있는 업스트림이 없다**: vulndb `Fixed in: N/A`, 프록시 기준 excelize 최신 태그가 v2.11.0(2026-07-06). 버전 올리기로는 해소 불가. 남는 선택지는 워크플로 게이트 완화 또는 무관한 `internal/store/seed.go` 개조뿐이라 절대 규칙(게이트 수정 금지, 무관 파일 금지)에 걸려 커밋하지 않았다.
- 이 PR 자체는 로컬에서 깨끗하다: `gofmt -l internal/` 무출력, `go vet ./...` RC=0, `go test ./... -count=1` 전 패키지 ok. 단 `TEST_POSTGRES_DSN` 이 없어 `internal/web` DB 테스트(신규 `mcp_report_test.go` 포함)는 **SKIP** 되었다 — 이 실행을 DB 검증으로 읽지 말 것. CI 로그상 `Unit and database integration tests` 단계는 통과한 뒤 govulncheck 에서 멈췄다.
- 판단: 이 게이트는 `origin/main` 과 공유하는 선결 부채이므로 별도 과제로 다뤄야 한다(호출부는 개발자 스크립트 `scripts/extract-defaults` 전용이며 서버 요청 경로가 아님). 중재자가 base 의 동일 실패를 근거로 이 PR 을 판정하길 권한다.
