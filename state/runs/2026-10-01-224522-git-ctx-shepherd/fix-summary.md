# 수리 요약 — git-ctx PR #43

- 문제: `sectionCount`·`cutAtBoundary` 의 새 `\n#### ` 단이 무조건이라 **파일 본문**의 `#### ` 제목에도 걸렸다. `#### ` 소제목 뒤에 `### ` 가 오는 마크다운을 read-file 하면 cut 이 더 이른 본문 `#### ` 를 골라 같은 예산에서 전달 바이트가 줄었다.
- 재현(커밋 전): 새 테스트 `TestReadFileCutIgnoresContentSubsectionHeadings`(예산 3400·4000)·`TestReadFileCountIgnoresContentSubsectionHeadings` 가 HEAD 에서 FAIL, budget.go 를 main 으로 되돌리면 PASS — 회귀임을 양방향으로 확인.
- 고침: `codeSearchHits` 를 추가해 `## Code Search\n` 로 시작하고 `\n### Source Matches (` 가 있는 응답의 **그 지점 이후 구간**에서만 `#### ` 를 세고 자른다(두 단 모두). 두 문자열은 `formatCodeSearch` 만 쓰므로 앵커가 유일하다. 어긋났던 주석도 실제 범위대로 다시 썼다.
- 검증: 24KiB·3400 예산 실측으로 read-file 전달 바이트가 main 과 **정확히 일치**(23,798·2,570; 수정 전 HEAD 는 23,780·2,552). search-code 테스트 2개는 main 의 budget.go 에서 여전히 FAIL — 원래 고치려던 버그는 그대로 잡는다.
- 전체 green: `gofmt -l ./cmd ./internal`(빈 출력), `go build`/`go vet -tags sqlite_fts5 ./...`, `go test -tags sqlite_fts5 -count=1 ./...`(internal/app 101.0s 포함). 커밋 d286e60. 외부 Postgres/pgvector/Vault·Docker·실브라우저·`-race`·govulncheck 는 미실행.
