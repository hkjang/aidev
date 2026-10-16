# 수리 요약 — umm PR #173 (커밋 5a3d76c)

- 문제(재현됨): `Dockerfile:9` 이 `golang:1.26.7-alpine` 이고 공식 golang 이미지는 `GOTOOLCHAIN=local` 이라 `go.mod:8` 의 `toolchain go1.26.9` 가 릴리즈 경로에서 무시됐다. `docker build --target go-build` 는 EXIT=0 인데 산출 바이너리는 `go1.26.7`, 그 바이너리에 `govulncheck -mode=binary` 를 돌리니 **14건(표준 라이브러리 12건 포함, `Found in: net/http@go1.26.7` / `Fixed in: @go1.26.9`)** — 지적이 전부 맞았다.
- 고침: `Dockerfile:9` → `golang:1.26.9-alpine` (1줄). `go.mod:5-7` 주석은 "CI 만 고정하고, 릴리즈는 Dockerfile 의 빌더 베이스가 고정하므로 같은 패치로 함께 올려야 한다" 로 사실에 맞게 고쳤다. 앱 코드·시험·CI 워크플로·검증 명령 변경 0건.
- 증거(요청대로 실제 빌드 후 바이너리 검사): 전체 `docker build --build-arg VERSION=0.76.8` EXIT=0 → **최종 런타임 이미지의 `/app/umm` 이 `go version -m` → `go1.26.9`**(fix-release-image-binary.log). 같은 바이너리 `govulncheck -mode=binary` → **14건 → 2건**, 표준 라이브러리 12건 전부 소멸(fix-govulncheck-binary-before/after.log). 남은 2건은 main 에도 있던 미호출 권고 GO-2026-6443(grpc)·GO-2026-5932(openpgp, 수정판 없음).
- 게이트 재실행: `go vet ./...` EXIT=0, `go test ./...` 전부 ok, `go test -race ./internal/intelligence` ok, CI 게이트 `govulncheck ./...` **EXIT=0 / No vulnerabilities found**, `./scripts/check-version.sh` 통과. x/net v0.60.0·x/text v0.42.0 가 산출 바이너리에 실제로 들어간 것도 확인 — 이 PR 의 원래 목적은 그대로다.
- 못 한 것: Go 통합 시험 159건은 `POSTGRES_DSN` 없어 SKIP(통과 아님) — 앱 코드 0줄 변경이라 신호가 없다고 판단. 웹 게이트는 따로 안 돌렸으나 전체 docker build 의 web-build 단계(`npm ci` + `npm run build`)가 통과했다.
