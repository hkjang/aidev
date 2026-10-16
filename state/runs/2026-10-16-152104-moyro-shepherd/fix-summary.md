# fix-summary — moyro PR #39 (수리)

- 문제: CI `verify` 의 `Test server` 단계에서 `govulncheck ./...` 가 exit 3. 북마크 `Reorder` 변경과 무관한, 새로 공개된 취약점이다 — 호출로 확인된 12건 전부가 Go 표준 라이브러리 go1.27(→ go1.27.2 수정)과 `golang.org/x/net` v0.58.0(→ v0.60.0 수정)이었다. 로컬에서 같은 명령으로 exit 1 / 동일 12건으로 **재현**했다.
- 고친 방법(패치 버전 상향만, 검사·테스트는 손대지 않음): `server/go.mod` `go 1.27.0` → `1.27.2`(CI setup-go 가 `go-version-file` 로 읽는 값), `golang.org/x/net` → v0.60.0(`go mod tidy` 가 crypto/mod/sync/sys/text 전이 상향 동반), `Dockerfile` server-build 베이스 `golang:1.27.0-alpine` → `1.27.2-alpine`(digest 재고정) — 이미지가 `GOTOOLCHAIN=local` 이라 베이스를 안 올리면 "Build service image" 가 깨진다.
- 검증(로컬, 전부 exit 0): `govulncheck ./...` → "No vulnerabilities found", `go build ./...`, `go vet ./...`, `go test -race -p 1 -count=1 ./...`(실제 PG16 DSN, 51 ok / FAIL 0 / bookmarks 4.150s 로 skip 아님), `plugin-sdk/go` test·vet, `scripts/check-source-sizes.sh`, `docker build --target server-build`.
- 미검증: webapp 단계(webapp 무변경, CI 로그상 이미 통과했던 단계), 전체 이미지 빌드·Playwright e2e. 커밋 `fddc0b8`, 3파일(Dockerfile/go.mod/go.sum)만 변경.
