비평 지적은 전부 맞았다. 재현(실제 `Server.ServeHTTP` 왕복): 청크 하나가 ```bash 를 열고 닫지 않으면 `search-semantic` 의 `### ` 12개가 `result_count=1`, `find-runbook` 9개가 1로 기록되고, `maxBytes:2500` 컷이 스니펫·`Source:` 없는 히트 헤딩에서 끊기며 공지가 "1 of 1" 이라 거짓말을 했다.

원인은 펜스 인식을 답변 텍스트에서 추측한 것이다. 수리: `fencesContent(tool)` 로 **도구 이름**에 묶었다 — `contentFence`(format.go:207)로 내용을 감싸는 read-file·get-symbol-context 만 펜스를 건너뛰고, 나머지(semantic·runbook·change-request·code-search)는 main 과 같은 `strings.Count("\n### ")`·`LastIndex("\n### ")` 를 그대로 쓴다. `sectionCount`·`cutAtBoundary`·`clampResponse` 가 `tool` 을 받고 dispatch.go:375 가 넘긴다.

테스트: `internal/mcp/unfenced_content_test.go` 신규 — 생prose 포매터 2개(search-semantic·find-runbook)의 감사 카운트와, 잘린 semantic 답변이 스니펫·citation 까지 갖춘 히트에서 끝나고 공지의 두 숫자가 실제와 맞는지. `fencesContent` 를 `return true`(반려된 동작)로 되돌리면 네 지적 모두 그대로 실패한다(확인함).

검증: `go build -tags sqlite_fts5 ./...`, `go vet ./...`, `gofmt -l ./cmd ./internal`(빈 출력), `go test -tags sqlite_fts5 -count=1 ./...` 전부 통과. 기존 단언은 하나도 느슨해지지 않았고, 직접 호출 테스트 7곳에 `tool` 인자만 추가했다.
