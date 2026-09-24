- 과제: 저장 결과 조회·삭제 실패의 HTTP 상태를 원인과 맞추기 (없는 결과·권한 없음 404 / 잘못된 ID 400 / 저장소 장애·손상 저장본 500) (가치 3 / 위험 1 / 작업량 S)
- 왜: `internal/server/resultsave.go:95` 는 `Service.Get` 의 **모든** 실패를 404 로, `:113` 은 `Service.Delete` 의 **모든** 실패를 400 으로 낸다 — 메타 DB 장애(`GetSavedResult` 의 `fmt.Errorf("저장 결과 데이터 조회: %w", err)`, `DeleteSavedResult` 의 `fmt.Errorf("저장 결과 삭제: %w", err)`)와 손상 저장본(`internal/resultsave/service.go:112` 의 "저장본이 손상됐을 수 있습니다")까지 '없는 결과'·'잘못된 요청'으로 나간다. 고치면 메타 DB가 죽었을 때 모니터링이 5xx 로 집계해 운영자가 장애를 보게 되고, 사용자는 "없는 결과"라는 틀린 안내 대신 서버 문제임을 안다.
- 수용 기준:
  1) `GET /api/v1/saved-results/{id}` — 없는/남의 결과는 그대로 404(`DF_SAVE_NOT_FOUND`), `id` 가 0·음수·숫자 아님이면 그대로 400(`DF_SAVE_INVALID_REQUEST`), 저장소가 그 밖의 오류를 내면(메타 DB 장애) **500**, 저장본 JSON 이 깨졌으면 **500** 이고 본문에 "손상" 안내가 남는다.
  2) `DELETE /api/v1/saved-results/{id}` — 없는/남의 결과·이미 지운 결과는 **404**(`DF_SAVE_NOT_FOUND`, 지금은 400), `id` 가 잘못되면 400 유지, 저장소 오류는 **500**.
  3) 테스트는 실제 라우터(`New(logger, options)`)에 httptest 로 요청해 위 네 가지(404 / 400 / 500-저장소장애 / 500-손상저장본)가 각각 나오는지 증명한다. 구현 전에 돌려 500 케이스 2개(그리고 삭제 404)가 **실패하는 것**을 먼저 확인할 것. 어떤 경우에도 응답 본문에 DB 드라이버 원문(`Error 1452`, `write tcp …`)이 없어야 한다.
- 건드릴 파일:
  - `internal/resultsave/service.go` — 패키지 수준 센티널 `var ErrNotFound = errors.New("저장된 결과를 찾을 수 없거나 권한이 없습니다")` (필요하면 `ErrInvalidID` 도) 추가. `Get`/`Delete` 의 `id <= 0` 은 기존 문구를 유지하되 센티널로 감싸고, 손상 저장본 오류는 지금처럼 그대로 올린다(센티널 아님 → 500 으로 떨어짐).
  - `internal/store/mariadb/resultsave_repository.go:85 GetSavedResult`, `:101 DeleteSavedResult` — 지금 `errors.New("저장된 결과를 찾을 수 없거나 권한이 없습니다")` 로 만드는 두 자리(sql.ErrNoRows 분기, RowsAffected()==0 분기)를 `resultsave.ErrNotFound` 로 바꾼다. 이 파일은 이미 `resultsave.Saved` 를 쓰므로 순환 import 없음(확인함).
  - `internal/server/resultsave.go:81 getSavedResultHandler`, `:101 deleteSavedResultHandler` — `errors.Is(err, resultsave.ErrNotFound)` → 404 `DF_SAVE_NOT_FOUND`, `errors.Is(err, resultsave.ErrInvalidID)` → 400 `DF_SAVE_INVALID_REQUEST`, 그 밖 → 500 (Get 은 `DF_SAVE_GET_FAILED`, Delete 는 `DF_SAVE_DELETE_FAILED` — 목록 쪽 `DF_SAVE_LIST_FAILED` 500 과 같은 꼴). 상세 문구는 계속 `err.Error()` 로 넘겨도 된다: `writeProblem`(`internal/server/http.go:1682`)이 한 곳에서 `safeProblemDetail` 로 드라이버 원문을 거른다(확인함).
  - 새 테스트: `internal/server/resultsave_test.go`. 세션은 `internal/server/loginpage_test.go:17 loginPageOptions`/`:22 sessionCookie` 패턴(`Options{Ready: true, Sessions: auth.NewSessions(time.Hour, nil)}` + `options.Sessions.Create`)을 그대로 쓰고, `options.ResultSave = resultsave.NewService(<테스트용 Repository>, nil)` 로 **실제 핸들러 배선**을 통과시킨다(`resultsave.Repository` 는 exported 인터페이스, `internal/resultsave/service.go:49`). Repository 대역은 저장소 경계이지 핸들러 대역이 아니므로 허용 — 핸들러·라우터·writeProblem 은 실물을 지난다.
- 검증 명령:
  - `cd /home/hkjang/.cache/auto-improve-wt/DartFly && go test -race ./internal/server/... ./internal/resultsave/... ./internal/store/...` (구현 전 red 확인 → 구현 후 green)
  - `go test -race ./...` · `go vet ./...` · `gofmt -l .` (전체, 이번 정찰에서 대상 2개 패키지는 baseline green 확인함)
  - 화면 동작이 바뀌지 않는 변경이지만 `/saved` 를 만지면 `DF_SMOKE_REQUIRE_BROWSER=1 bash test/smoke/run.sh` (수분, Docker+Chromium)
- 위험과 피할 것:
  - `internal/auth`·세션·SSO·`internal/store/mariadb/migrations`·`.github/workflows` 는 건드리지 말 것.
  - `ErrNotFound` 문구를 바꾸면 화면 안내가 바뀐다 — `internal/webui/js/saved.js:43,108,166` 은 상태코드를 보지 않고 `cause.message` 를 그대로 띄운다(확인함). 문구는 현행 그대로 유지할 것.
  - 목록(`listSavedResultsHandler`)은 이미 500 이다. 같은 것을 또 고치지 말 것. `Save` 경로(`writeExecutionProblem`)도 건드리지 말 것 — 실행 오류 분류는 별도 체계다.
  - 드라이버 원문 노출은 `safeProblemDetail`(커밋 4a806e3)로 이미 해결됐다. 그 자리를 다시 손대지 말고, 테스트로 회귀만 못 박을 것.
  - `internal/server/history.go:55`·`http.go:1659` 에도 같은 "모든 실패 404" 꼴이 있지만 **이번 범위 밖**이다. 범위를 넓히지 말 것.
- 차선 후보: 로그인 폼 429(속도 제한) 응답의 Retry-After 를 카운트다운으로 보여 주고 그 동안 제출 단추 비활성 (가치 2 / 위험 1 / 작업량 S) — 서버 리미터는 `internal/server/http.go:1695 logins` 로 존재함. `internal/webui/js/login.js` 가 Retry-After 를 읽는지는 **미확인**이니 먼저 열어 볼 것.
