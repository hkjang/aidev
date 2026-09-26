# 회차 노트 2026-09-26-161550-git-ctx-improve — git-ctx
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 16:15] base pinned — main@3c9f5de
- [러너 16:15] autonomy release — 

## 정찰 노트
- 선택 이유: 보류 목록은 전부 가치 2 이하이거나 반려 이력 유형(파서 확장)·보호 경로였다. 새로 읽은 internal/calltrace 에서 저장소 유일한 `err == context.DeadlineExceeded`(calltrace.go:112)를 찾았고, 같은 `sourceErr` 를 search/service.go:3326(Fail) 과 3415(errors.Is) 가 다르게 읽는 비대칭을 확인했다 — 지난 회차에 채택된 cache.go 비대칭 근거와 같은 형태이고 프로덕션 파일 1개로 끝난다.
- 확신 있는 것: 임포트·줄 위치, 다른 12곳의 errors.Is 사용, gitlab(client.go:636-641)·bitbucket 클라이언트가 `c.http.Do` 오류를 그대로 반환하는 구조, status 가 mcp_call_steps.status·mcp_calls.trace_summary 로 저장되는 경로(dispatch.go:387-404), 수정 전 `go test -tags sqlite_fts5 ./internal/calltrace` = ok 0.002s.
- 추측·미확인으로 적은 것: web/app.js 가 step status 별로 색·필터를 다르게 주는지 미확인(과제서에 명시). 실제 운영에서 `*url.Error` 가 Fail 까지 도달하는 것은 코드 경로로만 확인했고 실행 재현은 못 했다 — internal/search 에 원격 타임아웃 httptest 하니스가 없다.
- 구현자가 조심할 것: 상태 문자열 상수 값은 절대 바꾸지 말 것(DB 저장 계약). context.Canceled 를 새 상태로 추가하지 말 것 — 별도 아이디어로 분리해 뒀다. 호출부(search·mcp)는 한 줄도 고치지 말 것.
- 프로필은 1일 전 것으로 실질 내용이 현재 코드와 맞아 새로 쓰지 않았다. 기준 커밋만 ad093d9/v0.77.16 → 3c9f5de/v0.77.17 로 올라갔다.
- [러너 16:20] scout done — `calltrace.Span.Fail` 이 감싸인(wrapped) 데드라인 오류를 `timeout` 이 아니라 `error` 로 기록 (가치 3 / 위험 1 / 작업

## 구현 노트
- 무엇·왜: `internal/calltrace/calltrace.go:112` 의 `err == context.DeadlineExceeded` → `errors.Is(...)` (+`errors` 임포트, 사유 주석 4줄). 프로덕션 파일 1개. 감싸인 데드라인이 `mcp_call_steps.status` 에 `error` 로 저장돼 같은 오류를 읽는 진단문(`search/service.go:3415` 의 errors.Is)과 어긋났다.
- 수정 전 실패 직접 확인: `go test -run TestFailClassifiesWrappedDeadlinesAsTimeout ./internal/calltrace` → `wrapped_by_the_caller`·`wrapped_by_net/http` 가 `status="error", want "timeout"` FAIL, bare deadline·`errors.New("boom")`·`nil` 3케이스는 수정 전에도 PASS(동작 무변경 대조군). 대역 타입 없이 실제 `Recorder`/`Span`·실제 `*url.Error`·`fmt.Errorf("%w")` 만 사용.
- 확신 없는 곳·미검증: (1) 실제 운영에서 `*url.Error` 가 `Fail` 까지 도달하는 것은 코드 경로 독해로만 확인했다 — `internal/search` 에 원격 타임아웃 httptest 하니스가 없어 end-to-end 재현은 못 했다(만들면 작업량 M~L). (2) `web/app.js` 가 step status 별로 색·필터를 다르게 주는지 미확인 — 관찰 가능한 효과는 DB 컬럼 `mcp_call_steps.status`(dispatch.go:401)·`mcp_calls.trace_summary`(dispatch.go:387,391)로만 주장한다. (3) 이 변경으로 기존 `error` 행 일부가 앞으로 `timeout` 으로 기록되므로 과거 데이터와 새 데이터의 분포가 달라진다 — 상태 문자열 값 자체는 바꾸지 않았다.
- 일부러 하지 않은 것: `context.Canceled` 새 상태(계약·콘솔 위험, ideas.json 에 분리), `itoa`, 호출부(search·mcp) 한 줄도 안 건드림, 상태 상수 값 불변.
- 다음 역할이 조심할 것: 이 테스트는 DB·네트워크 불필요(순수 패키지 테스트, `t.Parallel()` 도 안 씀). 전체 스위트는 `go test -tags sqlite_fts5 ./...` 로 exit 0(internal/app 99.422s). 커밋 aad7bde 하나뿐이고 릴리즈·버전 파일은 건드리지 않았다.
- [러너 16:24] brief accepted — 채택 — 지정한 줄·임포트·비대칭 근거가 현재 코드와 정확히 맞았고, 과제서가 예측한 두 래핑 케이스만 수정 전 실패�
- [러너 16:26] verify passed — 검증 3개 통과 (auto)

## 비평 노트
- 확인한 것: main 판 calltrace.go 를 /tmp 사본에 넣고 새 테스트를 돌려 수정 전 FAIL 2건(wrapped_by_the_caller·wrapped_by_net/http)·PASS 3건을 직접 재현했다. 소비 경로도 전수 확인 — StatusTimeout 으로 분기하는 제어 흐름은 없고, mcpadmin.go:134 의 실패 집계는 `status NOT IN ('ok','')` 이라 error→timeout 이동에 영향이 없다.
- 구현자가 의심한 두 자리 중 web/app.js 는 해소됐다: 'timeout' 은 새 값이 아니고(service.go:3420·bare deadline 경로가 이미 기록) 콘솔 계약에 새 상태가 들어가지 않는다. http.Client.Timeout 오류도 errors.Is 로 잡힘을 httptest 로 실측했고 internal/source 는 애초에 client Timeout 을 안 쓴다.
- 오히려 근거가 과제서보다 강하다: audit.go:120 errorCode 가 이미 errors.Is 로 error_code='timeout' 을 쓰므로, 이 수정은 같은 호출의 step.status 를 기존 관례에 맞추는 것이다.
- 못 본 것: 원격 타임아웃 end-to-end 재현(internal/search 하니스 없음), 전체 스위트는 이번에 다시 돌리지 않고 calltrace·mcp 두 패키지만 검증했다(gofmt·vet 은 전체).
- 승인 후 남는 우려(릴리즈 노트용): 앞으로 일부 행이 error 대신 timeout 으로 기록돼 과거/신규 데이터 분포가 갈린다. 다음 회차 후보로 context.Canceled 의 동일한 비대칭(status='error' vs error_code='canceled')이 남아 있다.
- [러너 16:28] review approved — 리뷰 승인 (risk=low)
- [러너 16:29] pr created — https://github.com/hkjang/git-ctx/pull/39
