# 과제서 — 2026-09-26-161550-git-ctx-improve (base main@3c9f5de)

- 과제: `calltrace.Span.Fail` 이 감싸인(wrapped) 데드라인 오류를 `timeout` 이 아니라 `error` 로 기록 (가치 3 / 위험 1 / 작업량 S)

- 왜: `internal/calltrace/calltrace.go:112` 만 `if err == context.DeadlineExceeded` 로 **값 비교**를 한다 — 저장소의 다른 모든 판정은 `errors.Is` 다(`internal/worker/worker.go:182,370`, `internal/mcp/audit.go:120-122`, `internal/app/health.go:151,1051`, `internal/search/service.go:3415`, `internal/source/transport.go:209`). 그래서 **같은 오류 값**이 두 경로에서 다르게 읽힌다: `internal/search/service.go:3326` 은 `sourceErr` 를 `sourceSpan.Fail()` 에 넘기고, 같은 함수의 3415 는 그 `sourceErr` 를 `errors.Is(sourceErr, context.DeadlineExceeded)` 로 판정한다. 실제 원격 검색 오류는 감싸여 온다(`internal/gitlab/client.go:636-641` 이 `c.http.Do(req)` 의 오류를 그대로 반환 → `net/http` 는 `*url.Error` 로 컨텍스트 오류를 감싼다; `internal/bitbucket/v6/client.go:676,883` 도 같은 구조). 결과적으로 사용자 진단문은 "did not finish within the tool timeout" 이라고 말하는데 같은 단계의 감사 기록은 `error` 로 남아, 운영자가 `mcp_call_steps.status` 로 타임아웃을 세거나 걸러낼 수 없다.

- 수용 기준:
  1) `fmt.Errorf("query gitlab: %w", context.DeadlineExceeded)` 와 `&url.Error{Op:"Get", URL:"https://gitlab.example/api", Err: context.DeadlineExceeded}`(= `net/http` 가 실제로 만드는 타입)를 `Span.Fail` 에 넘기면 기록된 `Step.Status == calltrace.StatusTimeout` 이고, `Recorder.Summary()` 가 `"…: timeout"` 을 낸다.
  2) 기존 동작 무변경: bare `context.DeadlineExceeded`(기존 `calltrace_test.go:58`)는 계속 `timeout`, 평범한 오류(`errors.New("boom")`)와 `nil` 은 계속 `error`, nil `*Span`·nil `*Recorder` 는 계속 아무것도 기록하지 않는다. `Step.Detail` 은 감싼 오류의 `err.Error()` 전문이어야 한다(`internal/mcp/dispatch.go:402` 이 300자로 clip 해 저장하므로 잘리기 전 값이 원문이어야 한다).
  3) 테스트가 **수정 전에 실패**하는 것을 직접 확인해 적을 것(수정 전 기대값: `status=error`). 확인 문장을 구현 노트에 붙일 것.

- 건드릴 파일 (프로덕션 1개):
  - `internal/calltrace/calltrace.go:107-120` `(*Span).Fail` — `err == context.DeadlineExceeded` → `errors.Is(err, context.DeadlineExceeded)`, 파일 상단 임포트에 `"errors"` 추가(현재 임포트는 `context`/`sync`/`time` 뿐이다).
  - `internal/calltrace/calltrace_test.go` — 위 케이스를 테이블 테스트로 추가(`TestSummaryAndStepLimit` 의 timeout 블록 확장 또는 새 테스트). 대역 타입을 만들지 말고 실제 `Recorder`/`Span` 과 실제 `*url.Error`·`fmt.Errorf` 래핑만 쓸 것.

- 범위 밖(이번에 하지 말 것):
  - `context.Canceled` 를 새 상태로 구분하는 것 — 새 상태 상수는 `web` 콘솔과 `mcp_call_steps.status` 계약을 건드린다. `internal/mcp/audit.go:120-122` 는 둘을 구분하지만 calltrace 상태 집합(`calltrace.go:22-28`)에는 취소 상태가 없다. 필요하면 다음 회차 아이디어로 남길 것.
  - `calltrace.itoa`(calltrace.go:211) 손대지 않기 — 호출부가 `Candidates > 0` 으로 보호한다.
  - `internal/search`·`internal/mcp` 의 호출부는 한 줄도 바꾸지 말 것. 이 수정은 판정 함수 한 곳으로 끝난다.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `go test -tags sqlite_fts5 -count=1 ./internal/calltrace` — 이번 정찰에서 수정 전 `ok git-ctx/internal/calltrace 0.002s` 실측.
  - `go test -tags sqlite_fts5 -count=1 ./internal/search ./internal/mcp`
  - `go test -tags sqlite_fts5 -race -count=1 ./internal/calltrace`
  - `gofmt -l ./cmd ./internal`(출력이 비어야 함), `go vet ./...`, `go build -tags sqlite_fts5 ./...`
  - 마지막에 `go test -tags sqlite_fts5 ./...`(수 분, `./internal/app` 혼자 ~100초)

- 위험과 피할 것:
  - 보호 경로(auth/migrations/workflows)를 건드리지 않는다. `internal/version`·릴리즈 스크립트도 무관.
  - 상태 문자열 상수(`StatusOK/Empty/Skipped/Error/Timeout`)의 **값**을 바꾸지 말 것 — `mcp_call_steps.status` 에 그대로 저장되고(`dispatch.go:401`) 관리 콘솔이 읽는다. 값 변경은 과거 데이터와 어긋난다.
  - 소스 문자열 grep 을 증거로 제출하지 말 것. 반드시 실제 `Recorder`/`Span` 을 호출한 테스트의 수정 전 실패 출력을 근거로 쓸 것.
  - 미확인: `web/app.js` 가 step status 별로 색·필터를 다르게 주는지는 이번 정찰에서 확인하지 못했다(`grep timeout web/app.js` 는 설정 폼의 `timeoutSeconds` 만 나왔다). 관찰 가능한 효과는 **DB 컬럼 `mcp_call_steps.status` 와 `mcp_calls.trace_summary`** 로만 주장할 것(dispatch.go:387-404 에서 직접 확인).
  - `internal/search` 에는 원격 타임아웃을 만드는 `httptest` 하니스가 없다(grep 결과 0건). end-to-end 재현을 새로 만들려 하면 작업량이 M~L 로 커진다 — 이번 회차 범위 밖.

- 차선 후보: `cutAtRuneBoundary`(internal/search) 와 `runeSafeCut`(internal/mcp) 의 계약을 같은 입력·기대값 테이블 테스트 두 벌로 고정 (가치 2 / 위험 1 / S). 임포트 순환 때문에 의도적으로 중복된 같은 계약의 함수이므로 순수 테스트 추가만 하고 하위 패키지 신설·함수 통합은 하지 말 것.
