# 수리 요약 — ReSSO PR #43 (커밋 dcfd26f)

- 문제: lint·verify 둘 다 go1.26.7 로 고정된 툴체인 하나가 원인. 로컬 재현 — `govulncheck ./cmd/... ./internal/... ./webui/...` 가 stdlib 10건(GO-2026-6603/6605/6607~6613/6617, 전부 `Found in …@go1.26.7 / Fixed in …@go1.26.9`), Trivy 는 같은 CVE 를 `stdlib v1.26.7 → 1.26.9` 로 보고.
- 고친 것: `go.mod` `toolchain go1.26.9`, `Dockerfile` `golang:1.26.9-alpine` 두 줄. 저장소의 `TestToolchainPinsAgreeAcrossTheBuild` 가 요구하는 대로 함께 올렸다. refresh 회전 변경(8d3b77a)은 손대지 않았다.
- 검증: govulncheck 0건·golangci-lint 0 issues·`go vet` OK·gofmt clean·`go test -race ./...` 전부 ok(SKIP 0, 실제 PG 55440)·`docker build` 후 `trivy image --exit-code 1 --ignore-unfixed --severity HIGH,CRITICAL` 취약점 0·exit 0.
- 못 돌린 것: `npm run lint`/`npm test` (이 워크트리에 web/node_modules 없음). 변경이 go.mod·Dockerfile 두 줄이라 프런트엔드에 영향 없음.
- 주의: 이 툴체인 상향은 PR 의 본래 목적과 무관하지만 CI 실패의 유일한 원인이었고, main 에도 같은 실패가 있는 선반영 수정이다.
