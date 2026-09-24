# 과제서 (2026-09-23, base main@5120fb9 / v0.77.15)

- 과제: search 쪽 바이트 절단(ReadFile 192KiB, ExportContext 200000B)이 UTF-8 글자를 반으로 자르지 않게 수정 (가치 3 / 위험 1 / 작업량 S)

- 왜: 지난 회차(1dd6363)가 MCP 쪽 `cutAtBoundary`만 글자 경계로 고쳤고, 같은 텍스트를 만들어 내는 상류 경로인 `internal/search/service.go`의 두 바이트 절단은 그대로 남아 있다 — `body = body[:readFileByteBudget]`(service.go:2321)과 `result = result[:200000]`(service.go:1185)은 멀티바이트 문자 중간에서 잘라 invalid UTF-8을 만든다. 이 값은 REST 플레이그라운드(`internal/app/playground.go:164` `testReadFile` → `jsonOut`, `:608` export)로 그대로 JSON 직렬화되고, Go의 `encoding/json`은 유효하지 않은 바이트를 U+FFFD(`�`)로 바꾸므로 한국어·일본어·중국어 파일의 마지막 글자가 깨져서 에이전트에게 전달된다.

- 수용 기준:
  1) 192KiB를 넘는 한글(3바이트) 본문을 `Service.ReadFile`로 읽으면 반환된 `FileContent.Content`가 `utf8.ValidString` 을 통과하고, 원문의 접두부와 정확히 일치한다(잘린 부분 앞까지 문자 단위로 보존, 잘린 마지막 글자는 통째로 빠진다).
  2) `ExportContext`의 200000바이트 절단도 같은 성질을 만족하고, 뒤에 붙는 `[Export truncated at the platform safety limit.]` 공지는 그대로 남는다.
  3) 회귀 테스트가 **수정 전에 실패**하는 것을 직접 확인해 기록할 것. 테스트는 실제 `store.Open("sqlite", …)` + 실제 `Service.ReadFile`(기존 `internal/search/service_test.go`의 `TestReadFileServesIndexedAndUnindexedFiles` 패턴: `repositories`/`repository_permissions`/`repository_files` 인서트 + `SetSourceLoader`로 큰 파일을 돌려주는 fake source)를 통과해야 하고, 손으로 만든 문자열을 헬퍼 함수에 직접 넣는 단위 테스트만으로 증거를 삼지 말 것. 최소 한 건은 JSON 왕복(`json.Marshal` 결과에 `�` 없음)까지 검사한다.
  4) 절단이 일어나지 않는 입력(예산 이하), ASCII만 있는 입력, 예산 경계 ±몇 바이트(잘린 자리가 1·2·3바이트 문자 중간에 걸리는 모든 경우)에서 기존 동작과 `Truncated` 플래그·`truncated: returned lines …` 진단이 그대로인지 확인한다.

- 건드릴 파일:
  - `internal/search/service.go:2320-2323` — `ReadFile` 안의 `if len(body) > readFileByteBudget { body = body[:readFileByteBudget]; out.Truncated = true }` 를 글자 경계까지 물러나는 절단으로. `out.Truncated` 설정과 진단 문구는 그대로 둘 것.
  - `internal/search/service.go:1184-1186` — `ExportContext` 의 `result = result[:200000] + "\n\n[Export truncated …]"` 도 같은 헬퍼로.
  - `internal/search` 에 작은 헬퍼 하나(예: `cutAtRuneBoundary(value string, limit int) string`, `unicode/utf8` 사용). `internal/mcp/audit.go:137` 의 `runeSafeCut` 를 **임포트해서 재사용하지 말 것** — 의존 방향이 mcp → search 이므로 반대 임포트는 순환이 된다. 같은 계약(바이트 한도 이하의 가장 긴 유효 접두부)을 search 안에 따로 두고 주석으로 이유를 적을 것.
  - 테스트: `internal/search/service_test.go`(또는 같은 패키지의 새 `*_test.go`)에 ReadFile·ExportContext 회귀 2건.
  - 선택(여유 있으면): `internal/app` 의 플레이그라운드 read-file 라우트를 통한 HTTP 왕복 1건 — 실제 사용자 노출 경로가 REST JSON이므로 여기서 U+FFFD 부재를 확인하면 가장 강한 증거다. 다만 `indexedApp` 하네스는 백그라운드 인덱서를 기다리고(`testing.Short()` 에서 skip) 인덱서에는 `MaxFileBytes`(`internal/indexer/indexer.go:30`) 상한이 있어 200KB 파일이 색인되지 않을 수 있다 — 그 경우 live read(remote origin) 경로로 유도하거나, 이 항목을 빼고 1)~4)만 충족해도 된다.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `go test -tags sqlite_fts5 -count=1 ./internal/search`
  - `go test -tags sqlite_fts5 -count=1 ./internal/mcp ./internal/app`(read-file 포맷·예산 회귀 확인. `internal/app` 은 통과해도 ~100초 걸린다)
  - `go test -tags sqlite_fts5 -race -count=1 ./internal/search`
  - `gofmt -l ./cmd ./internal`(출력 없어야 함), `go vet ./...`, `go build -tags sqlite_fts5 ./...`
  - 마지막에 `go test -tags sqlite_fts5 ./...`(수 분)

- 위험과 피할 것:
  - **경계 산수(정찰이 손으로 계산, 런타임 미확인)**: `// note\n`(8B) 뒤에 34바이트짜리 한글 줄이 반복되는 본문에서 192KiB(196608) 절단점은 줄 내부 오프셋 12 — `주`(바이트 10-12)의 중간이라 invalid UTF-8이 된다. 구현자는 이 계산을 믿지 말고 **수정 전 실패**를 실제로 재현해 기록할 것. 재현되지 않으면 입력을 바꿔(ASCII 1~2자 접두, 이모지 4바이트 혼합) 경계에 걸리게 만들 것.
  - MCP 기본 응답 예산은 24KiB(`internal/mcp/budget.go` `DefaultResponseBytes`)라 MCP read-file 경로에서는 192KiB 꼬리가 `clampResponse` 에 다시 잘려 손상이 가려질 수 있다. "MCP에서 재현이 안 된다"를 "버그가 없다"로 읽지 말 것 — 노출 경로는 REST 플레이그라운드와 예산을 올린 MCP 툴(`MaxResponseBytes` 256KiB까지 가능)이다.
  - `internal/contentsecurity`(마스킹 규칙)·`internal/auth`·`internal/store` 마이그레이션·`.github/workflows`·`internal/version` 은 건드리지 말 것. 이번 수정은 색인 내용이 아니라 응답 절단이므로 `contentsecurity.Revision()` 이나 재색인과는 무관하다 — Revision 을 바꾸지 말 것.
  - 예산 상수(`readFileLineBudget`, `readFileByteBudget`, 200000)를 바꾸지 말 것. 값이 아니라 자르는 자리만 고치는 과제다.
  - 과거 교훈: 실제 출력이 바뀌지 않는 수정은 반려 사유다. "utf8.ValidString 만 true" 가 아니라 **JSON 직렬화 결과에 U+FFFD 가 사라지는 것**을 증거로 남길 것.
  - `internal/search/service.go` 는 크다(4000줄+). 지정된 두 자리와 새 헬퍼 외에는 손대지 말 것.

- 차선 후보: `mcp.cacheKey` 가 호출자의 ACL 슬라이스를 제자리 정렬하는 문제 — `internal/mcp/cache.go:31-33` 의 `principals := principalACLs(p); sort.Strings(principals)` 는 `internal/mcp/args.go:167` `principalACLs` 가 비관리자에서 `p.ACLPrincipals` 백킹 배열을 그대로 돌려줄 때 호출자의 슬라이스를 정렬해 버린다(`search.WithUnrestricted` 가 원본을 반환하는 경로 — 구현자가 먼저 확인할 것). `principals := append([]string(nil), principalACLs(p)...)` 한 줄 복사 + 정렬 전후 호출자 슬라이스 순서를 확인하는 테스트. 실제 오동작 재현이 안 되면 3순위로 넘어갈 것(3순위: `clampResponse` 의 `responseNoticeBytes = 320` 예약이 실제 공지보다 짧아 최종 응답이 예산을 수십 바이트 넘기는지 재현·수정).
