# 수리 요약 — Momento PR #29

- 문제: CI `test` 잡의 govulncheck 단계가 GO-2026-6629(`golang.org/x/text` v0.39.0, `precis.Profile.String`, `database.Open`→pgxpool 경유)로 exit 3. 로컬에서 같은 명령으로 같은 자리 재현했다.
- 이 PR 의 웹 변경과 무관한 **main 에 이미 있던 실패**다(`git diff origin/main...HEAD` 는 web 파일 5개뿐, Go/go.mod 무수정). 새 권고가 공개돼 main 의 핀이 빨간불이 된 것.
- 고친 방법: `go get golang.org/x/text@v0.41.0` + `go mod tidy`. go.mod/go.sum 만 바뀌었다(x/text v0.39.0→v0.41.0, 그 모듈이 요구하는 x/sync v0.21.0→v0.22.0). 코드·테스트·워크플로는 손대지 않았다.
- 재검증: govulncheck `No vulnerabilities found` exit 0, `go vet` 통과, `go test -race ./cmd/... ./internal/...` 전부 ok, web `npm ci && lint && test(259/259) && build` 통과. (Postgres 가 없어 Go 통합 테스트는 조용히 skip — DB 동작 근거로 쓰지 말 것.)
