# 과제서 (2026-10-01, base main@f1de053)

- 과제: 결과 개수 집계와 절단 경계가 `#### ` 로 쓰인 search-code 히트를 보게 하기 — `sectionCount`·`cutAtBoundary` (가치 3 / 위험 2 / 작업량 S)

- 왜: `internal/mcp/format.go:118` 의 `formatCodeSearch` 는 소스 히트를 `#### ` 로 쓰는데, 같은 텍스트를 읽는 `internal/mcp/budget.go` 의 `sectionCount`(74행)·`cutAtBoundary`(96행)는 `\n### ` 만 본다. 그래서 (a) 감사 열 `mcp_calls.result_count`(dispatch.go:375 → 391-393)가 히트 수가 아니라 구조 제목 수를 기록해 관리 콘솔·`/me` 가 "결과 3건" 으로 표시하고, (b) 절단 공지가 "2 of 2 result sections are included" 라고 거짓을 말하며, (c) 절단 경계가 히트 중간에 떨어져 마지막 히트가 제목만 남고 스니펫·`Source:` 인용이 사라진다. 세 가지가 모두 `#### ` 하나를 못 보는 같은 원인이고, 고치면 운영자가 보는 숫자와 에이전트가 받는 공지가 실제 히트 수와 일치한다.

- 측정한 현상 (이 회차에 임시 프로브 테스트로 실측, 프로브는 삭제했고 작업 트리는 깨끗함):
  - `fixture(t)` + 실제 `Server.ServeHTTP` `tools/call` `search-code {"query":"GPU"}` → 히트 2건, `mcp_calls.result_count=3`, `sectionCount(text)=3`(= `### Repository Matches` / `### Source Matches (2)` / `### Notes`).
  - 같은 fixture 에 `document_chunks` 60행을 더 넣고 `search-code {"query":"GPU","limit":50,"maxBytes":4000}` → 50건 중 19건이 전송됐는데 `result_count=3`, 공지는 `- 2 of 2 result sections are included.` 였고 바로 아래 Notes 는 `- index: 50 match(es)…` 라 서로 모순. 본문 끝은 `#### /kcb/clustara · docs/gpu-25.md` 로 끝나 스니펫과 `Source:` 줄이 없는 반쪽 히트였다(= `clampResponse` 주석이 약속한 "the last entry is whole" 위반).

- 수용 기준:
  1) 실제 배선(대역 타입 없이 `fixture(t)` + `Server.ServeHTTP` JSON-RPC `tools/call`)으로 `search-code` 를 불러, 히트가 N건일 때 `mcp_calls.result_count == N` 이다(현재 3). 절단되지 않는 작은 호출과 절단되는 큰 호출 두 경우 모두.
  2) 절단된 `search-code` 응답의 `- X of Y result sections are included.` 에서 Y 가 전체 히트 수(위 재현에서 50), X 가 실제로 본문에 남은 `#### ` 히트 수(위 재현에서 19)와 일치한다.
  3) 절단된 `search-code` 응답 본문의 마지막 히트가 온전하다 — 본문의 마지막 `#### ` 블록이 자기 `Source: ` 인용 줄을 포함하고, 본문(`### Truncated` 앞)의 마지막 비어있지 않은 줄이 `#L<숫자>-L<숫자>` 로 끝난다.
  4) 회귀 방지 대조군: 같은 fixture 로 `read-file`(현재 `result_count=0` 실측)과 `find-symbol`/`get-symbol-context` 같은 `### ` 전용 포매터 한 개를 같이 불러 `result_count` 가 수정 전과 같음을 단언한다. 즉 변경의 영향 범위가 `#### ` 를 쓰는 경로로 한정된다.
  5) 기존 `internal/mcp/truncation_test.go` 3건과 `internal/mcp/fence_test.go` 가 그대로 통과한다(수정 금지).
  6) 수정 전 실패를 직접 돌려 확인하고, 실패 출력(기대/실제 숫자가 보이는 줄)을 노트에 남긴다.

- 건드릴 파일 (프로덕션 1개):
  - `internal/mcp/budget.go:sectionCount` — 결과 항목을 "가장 깊은 제목 수준" 으로 세게 한다. 즉 `\n#### ` 가 하나 이상이면 그 개수를, 없으면 지금처럼 `\n### ` 개수를, 그것도 없으면 `\n- ` 개수를 돌려준다. 기존 두 단 사다리에 한 단을 앞에 붙이는 형태로, `### ` 전용 포매터의 반환값은 바뀌지 않는다. 주석에 "포매터가 히트에 `#### ` 를 쓴다"는 근거(format.go:118)를 적는다.
  - `internal/mcp/budget.go:cutAtBoundary` — 경계 후보 사다리 맨 앞에 `strings.LastIndex(window, "\n#### ")` 를 추가한다. 기존 `enough(at)` 조건을 그대로 쓰면 `#### ` 가 없는 텍스트에서는 `-1` 이라 즉시 통과되어 동작이 바뀌지 않는다.
  - 테스트: `internal/mcp/` 에 새 파일 1개(예: `result_count_test.go`). 기존 테스트 파일은 고치지 않는다.
  - `internal/mcp/format.go` 는 건드리지 않는다 — 제목 수준을 바꾸면 모든 에이전트가 받는 렌더링이 바뀌고, 그래도 개수는 구조 제목 2개만큼 틀린 채 남는다.

- 검증 명령:
  - `go test -tags sqlite_fts5 -count=1 ./internal/mcp` (이 회차 실측 0.02~0.9s)
  - `go test -tags sqlite_fts5 -race -count=1 ./internal/mcp`
  - `gofmt -l ./cmd ./internal` (출력이 비어야 함), `go vet ./...`, `go build -tags sqlite_fts5 ./...`
  - `go test -tags sqlite_fts5 ./...` (전체, `./internal/app` 혼자 ~100초)
  - 새 테스트에 `t.Parallel()` 을 쓰지 말 것 — mcp fixture 는 공유 in-memory SQLite 이름을 쓴다.

- 위험과 피할 것:
  - `sectionCount` 는 감사 열 `result_count` 에 직접 들어가므로 수치 계약이다. 변경 범위를 `#### ` 를 쓰는 경로로 한정했음을 수용 기준 4로 관측 가능하게 고정할 것. `internal/app/mcpadmin.go`·`web/app.js` 는 열을 그대로 읽어 표시만 하므로 건드리지 않는다.
  - 히트 0건인 search-code 응답에는 `#### ` 가 없어 지금과 같은 값(구조 제목 수)이 나온다. 이번 범위에서 그걸 0으로 만들려 하지 말 것(빈 응답은 `empty` 분기라 `results` 가 애초에 0으로 기록된다 — dispatch.go:364-370 확인).
  - 코드 펜스 **안**의 마크다운 제목까지 제외하는 일반화(파일 본문의 `### `/`#### ` 를 결과로 세지 않기)는 하지 말 것. 보류 아이디어로 남겨 둔 별건이고, `closeOpenFence` 의 펜스 스캐너와 얽혀 S 를 벗어난다.
  - 보호 경로 금지: `internal/auth`, `internal/app` 인증/세션, `internal/store` migration, `.github/workflows`, `scripts/release.sh`, `internal/version` — 이번 과제는 이 중 아무것도 필요 없다.
  - 과거 교훈: 손으로 만든 문자열만으로 증명하지 말 것. 기존 `truncation_test.go:57` 의 `TestASectionedAnswerStillEndsOnASection` 은 히트를 `### ` 로 쓰는 **프로덕션에 없는 모양**을 합성해 통과하고 있다 — 그것이 이 결함이 여태 안 보인 이유다. 반드시 실제 `formatCodeSearch` 출력을 HTTP 왕복으로 받아 단언할 것.
  - 포매터와 카운터 두 경로가 같은 입력을 같은 값으로 읽는지 end-to-end 로 확인할 것(운영자 반복 지시). 한쪽만 고치고 끝내지 말 것.

- 차선 후보: 추적된 디버그 산출물 `c.txt`·`server.log` 를 삭제하고 `.gitignore` 에 등록 (가치 2 / 위험 1 / S). 두 파일은 현재도 저장소 루트에 추적되어 있다(이 회차 `ls` 로 확인). 비밀값 유무는 미확인이므로 지우기 전에 내용을 확인할 것.
