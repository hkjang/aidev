- 과제: SSE 재연결 커서 우선순위(Last-Event-ID 대 `?after` 중 큰 값)의 실제 HTTP 회귀 테스트 (가치 3 / 위험 1 / 작업량 M)

- 왜: `streamSimpleRunLogs`(backend/internal/server/simple.go:1171-1174)와 `streamReleaseLogs`(backend/internal/server/releases.go:1492-1495)는 `Last-Event-ID` 헤더와 `?after` 쿼리를 각각 파싱해 **큰 값**을 커서로 고르는데, 이 우선순위를 지나는 테스트가 저장소에 하나도 없다(`grep -rn "Last-Event-ID"` 결과가 프로덕션 2곳과 주석 1곳뿐이고, 기존 통합 테스트 3건은 모두 `?after=0` 으로만 연다 — simple_stream_test.go:76, 113, 144). 이 규칙은 실제 사용자 경로다: `SimpleRunDetailPage.tsx:239` 가 `?after=${lastIdRef.current}` 로 EventSource 를 열고, 브라우저가 스스로 재연결할 때는 **그때의 URL(이제 낡은 after 값)을 그대로 재사용하면서** 최신 `Last-Event-ID` 헤더만 새로 붙이므로, 큰 값을 고르지 않으면 이미 화면에 있는 줄이 다시 내려와 중복된다. 반대로 v0.5.21 의 "다시 연결" 버튼은 저장 로그를 재수집해 커서를 올린 뒤 **새** EventSource 를 열므로 헤더가 없고 `?after` 만 유효하다 — 두 방향이 모두 깨지지 않게 고정해야 한다.

- 수용 기준:
  1) `?after` 만 준 요청(헤더 없음)이 그 id 이후 줄만 보낸다 — 커서 이전 줄이 본문에 없다.
  2) `Last-Event-ID` 헤더만 준 요청(`?after` 없음 또는 `after=0`)이 헤더 id 이후 줄만 보낸다. 이것이 지금 전혀 검증되지 않는 경로다.
  3) 두 값이 다를 때 **큰 쪽**이 이긴다 — 두 방향 모두 확인할 것: (a) 헤더 > after (브라우저 자동 재연결: 낡은 `after=0` + 최신 헤더) 에서 헤더 id 이후만 오고 중복이 없다, (b) after > 헤더 (수동 재연결 뒤 낡은 헤더) 에서 after id 이후만 온다.
  4) 위 네 경우를 프로덕션 커서 코드에서 되돌려(예: 두 값 비교를 지워 헤더만/`after` 만 쓰게) 새 테스트가 실제로 빨개지는 것을 확인하고, 그 실패 출력을 근거로 남길 것. 프로덕션 코드는 되돌림 실험 뒤 반드시 원상 복구한다(이 과제의 커밋에는 테스트만 들어간다).
  5) 전체 모드도 같은 계약임을 한 건으로 덮는다 — `streamReleaseLogs` 에 `Last-Event-ID` 를 준 요청이 그 id 이후만 보낸다. 두 파서가 같은 입력을 같은 값으로 읽는지를 이 한 쌍이 증명한다.

- 건드릴 파일 (프로덕션 0개, 테스트 1개):
  - `backend/internal/server/simple_stream_test.go` — 단순 모드 커서 테스트를 추가한다. `newSimpleStreamFixture(t)`(같은 파일 21행) → `seedSimpleRun(t, s, targetID, runID, batch, "SUCCESS", true)`(simple_batch_test.go:76) → `seedSimpleRunLogs(t, s, runID, 10)`(같은 파일 52행) 로 준비하고, `s.Handler().ServeHTTP(recorder, request)` 로 **실제 라우트**를 탄다(세션 쿠키 이름은 `releasedock_session`, 토큰은 픽스처가 돌려준다). 상태를 `SUCCESS` 로 두면 커서 이후 줄을 모두 보낸 뒤 `event: end` 로 즉시 반환하므로 테스트가 매달리지 않는다. 헤더는 `request.Header.Set("Last-Event-ID", "…")`.
  - 같은 파일에 전체 모드 한 건 — 기존 `TestReleaseLogStreamDrainsFullPagesWithoutWaiting`(144행) 을 그대로 본떠 `newRollbackRetryFixture(t)` + `fixture.server.streamReleaseLogs(recorder, request)` 직접 호출에 `Last-Event-ID` 헤더만 더한다. 전체 모드는 저장소 관례상 라우터를 지나지 않고 핸들러를 직접 부르며, 커서 파싱은 그 핸들러 안에서 일어나므로 이 범위로 충분하다. 테스트를 늘려 파일이 커지면 커서 테스트만 새 파일(`simple_stream_cursor_test.go`)로 빼도 좋다 — 다만 프로덕션 파일은 계속 0개로 둘 것.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - DSN 없이 컴파일·SKIP 확인(이번 정찰에서 실행해 종료 0, 2건 SKIP 확인): `cd backend && go test ./internal/server -run TestSimpleRunLogStream -count=1 -v`
  - **DSN 이 없으면 새 테스트는 SKIP 되어 아무것도 증명하지 못한다.** 과거 회차처럼 도커로 PostgreSQL 16 을 띄워 `TEST_POSTGRES_DSN` 을 채운 뒤 `-v` 로 **SKIP 없이 PASS** 하는 것을 확인할 것: `cd backend && go test ./internal/server -run 'TestSimpleRunLogStream|TestReleaseLogStream' -count=1 -v`
  - 마무리: `cd backend && go test ./... -count=1` (server 패키지에서 SKIP 은 웹 자산이 필요한 `TestEmbeddedPortalServesWithoutADiskRoot` 1건만 남는 것이 직전 회차 기준), `go vet ./...`, 변경 파일 `gofmt -l`.
  - 웹은 건드리지 않으므로 `npm ci`(수 분) 는 불필요하다.

- 위험과 피할 것:
  - **문자열 포함 검사의 접두사 충돌** — `strings.Contains(body, "line 1")` 은 `line 10` 에도 걸린다. 기존 테스트가 이 함정 바로 옆에 있다. `"message":"line 1"` 처럼 JSON 따옴표까지 포함해 비교하거나, 더 낫게 프레임의 `id:` 로 판정할 것.
  - **id 를 1..10 이라고 가정하지 말 것** — `simple_run_logs.id` 는 시퀀스다. 스키마가 테스트마다 격리되므로 지금은 1부터지만, 커서 값은 `SELECT id FROM simple_run_logs WHERE run_id=$1 ORDER BY id` 로 실제 id 를 읽어 쓰는 편이 깨지지 않는다.
  - 스트림 한도는 사용자당 3 **동시** 접속이고 해제는 핸들러 반환 시점이므로, 순차 요청이면 걸리지 않는다. 한 테스트 안에서 스트림을 동시에 열지 말 것.
  - 프로덕션 코드(simple.go / releases.go)를 **고치지 말 것**. 이 과제는 계약을 고정하는 회귀 테스트이며, 커서 로직을 "정리"하려 두 파서를 공용 헬퍼로 합치려 들면 계약이 다른 두 경로를 묶는 과거 교훈에 걸린다.
  - 보호 경로(auth.go/oidc.go/rbac_policy.go, store/migrations, .github/workflows) 와 VERSION·`web/dist` 는 건드리지 않는다.
  - 소스 grep 을 증거로 제출하지 말 것 — 실제 DB 로 돌린 테스트 출력(그리고 수용 기준 4 의 되돌림 실패 출력)이 근거다.

- 차선 후보: 로그 스트림 한도(3/user·64/global) 거절과 해제의 HTTP 통합 테스트 (가치 2 / 위험 2 / M) — 같은 `newSimpleStreamFixture` 위에서 네 번째 접속의 429 `stream_limit` 과 해제 후 재접속을 본다. 1순위와 같은 파일·같은 픽스처라 준비 비용이 겹치지만, 동시성 안정화(스트림 3개를 열어 둔 채 네 번째를 쏘는 타이밍)가 추가 위험이다.
