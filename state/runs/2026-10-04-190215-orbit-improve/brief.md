- 과제: AI 스트림이 제공자 오류로 실패할 때 원인이 서버 로그에 한 줄도 남지 않는 것을 고치고, 그 용도로 쓰려고 만들었으나 호출자가 0인 `safeAIError` 를 실제로 쓰면서 룬 경계 버그까지 닫기 (가치 3 / 위험 1 / 작업량 S)
- 왜: `internal/server/ai.go:86~90` 의 `streamAI` 는 `proxyAIStream` 이 돌려준 `err` 를 변수에 받고도 버린 채 사용자에게 고정 문구 SSE(`"AI 응답을 완료하지 못했습니다."`)만 보내고 `return` 한다 — 제공자 401/429/500, 잘못된 엔드포인트, 타임아웃, 끊긴 스트림이 전부 같은 한 문장으로 보이고 **서버 로그에도 아무것도 남지 않아** 운영자가 "AI 가 안 된다" 는 신고를 받아도 원인을 알 방법이 없다(이 저장소는 `internalError`(respond.go:47)·`export.go:85` 에서 실패를 `slog.Error` 로 남기는 관례가 이미 있다 — 이 경로만 비어 있다). 같은 파일 `ai.go:306~312` 의 `safeAIError` 는 정확히 이 용도로 작성됐는데 저장소 전체에서 호출자가 0이고(`grep -rn safeAIError --include=*.go .` → 정의 1줄뿐, 확인함), 그 안의 `message[:300]` 은 바이트 슬라이스라 한국어 제공자 오류가 300바이트에서 잘리면 룬이 쪼개져 꼬리가 U+FFFD 로 깨진다. 한 자리를 배선하면 진단 가능성이 생기고 죽은 함수가 되살아나며 룬 버그가 같이 닫힌다.
- 수용 기준:
  1) `proxyAIStream` 이 오류를 돌려주면 `slog.Error` 한 줄이 남는다. `export.go:85` 의 형식을 따라 식별자 + 원인만 담는다(예: `slog.Error("AI 스트림이 중간에 끊겼습니다", "user", u.ID, "model", settings.Model, "error", safeAIError(err))`). 사용자에게 가는 응답은 **바이트 단위로 지금과 동일**해야 한다 — 상태코드 200, `event: meta` → `event: error` + 같은 문구, `done` 없음. 새 오류 코드·새 사용자 문구를 만들지 말 것.
  2) 클라이언트가 스트림을 끊어서(탭 닫기·`context canceled`) 생긴 오류는 ERROR 로 남기지 않는다 — `respond.go:44` 의 `if errors.Is(err, r.Context().Err()) { return }` 선례를 그대로 쓴다. 이것 없이는 정상적인 사용자 이탈마다 오탐 ERROR 가 쌓인다.
  3) `safeAIError` 가 300바이트에서 자를 때 UTF-8 룬을 쪼개지 않는다. 한글(룬 3바이트)로 300바이트 경계를 정확히 걸치게 만든 입력에서 반환값이 `utf8.ValidString` 을 통과하고 `�` 를 포함하지 않으며, 300바이트 이하 입력은 **한 바이트도 바뀌지 않고** 그대로 반환된다.
  4) 테스트가 증명할 것: (a) 실제 `httptest` 제공자 서버가 500 + 본문을 돌려주는 상황에서 프로덕션 핸들러 `(&Server{store: st}).streamAI` 를 불러, 캡처한 로그 레코드가 1개 이상이고 제공자 상태코드(`500`)와 `u.ID` 가 그 안에 있다 — 고치기 전에는 레코드 0개로 빨개진다. (b) 같은 실행에서 제공자가 정상 delta 를 주는 경로는 로그 레코드가 0개이고 SSE 바이트가 `ai_db_test.go:116` 의 기존 기대값과 같다(회귀 기준). (c) DB 없이 도는 `safeAIError` 순수 함수 시험이 수용 기준 3을 덮는다.
- 건드릴 파일:
  - `internal/server/ai.go:86~90` `streamAI` 의 `if err := s.proxyAIStream(...)` 분기 — 사용자 SSE 를 보내기 전/후 어디든 좋으나 `sendSSE`·`flusher.Flush()`·`return` 구조는 유지하고 `slog.Error` 한 줄 + 수용 기준 2의 조기 `return` 가드만 더한다. `import "log/slog"` 가 ai.go 에 아직 없으므로 추가 필요(현재 import 블록은 ai.go:3~18, `log/slog` 없음 — 확인함).
  - `internal/server/ai.go:306~312` `safeAIError` — 300바이트 초과 시 룬 경계까지 물러나 자른다. 표준 라이브러리만(`unicode/utf8`). 시그니처 `func safeAIError(err error) string` 는 바꾸지 말 것.
  - `internal/server/ai_db_test.go` — 신규 `TestStreamAIProviderFailureIsLogged`. 기존 자산을 그대로 재사용한다: `openTestStore`(timetravel_db_test.go), `preserveAISetting`(ai_db_test.go:24, AI 설정 행은 전역이므로 **필수**), `seedUser`/`seedPerson`/`seedRelationship`(timetravel_db_test.go:41~88), 설정 행 쓰기 패턴(ai_db_test.go:99~103 의 `INSERT ... ON CONFLICT ... encrypted_value=''` — 빈 키 + 기억 0건이면 nil Vault 로도 맥락 조회가 된다), 요청 배선(ai_db_test.go:105~110 의 `userContextKey` 컨텍스트 주입 + `httptest.NewRecorder()`; recorder 는 `http.Flusher` 를 구현한다). `t.Parallel()` 금지.
  - `internal/server/ai_test.go` — `safeAIError` 단위 시험(DB 불필요, CI 에서 실제로 돈다).
  - 프로덕션 파일은 `ai.go` 하나다.
- 검증 명령:
  - `gofmt -l internal/server`(무출력) · `go vet ./...` · `go build ./...`
  - `go test -race -count=1 ./...` (DSN 없음 — 새 DB 시험은 SKIP, `safeAIError` 시험은 돈다)
  - 격리 postgres 로:
    ```
    docker run -d --rm --name orbit-probe -e POSTGRES_PASSWORD=orbit -p 55659:5432 postgres:16-alpine
    ORBIT_TEST_DATABASE_URL='postgres://postgres:orbit@127.0.0.1:55659/postgres?sslmode=disable' \
      go test -race -count=1 -v ./internal/server -run 'TestStreamAI|TestSafeAIError'
    ORBIT_TEST_DATABASE_URL='...' go test -race -count=1 ./...
    ```
    포트 55659 는 프로필의 기사용 목록(55433·55439·15434·55471·55481·55491·55521·55537·55603·55617·55641·49761)과 겹치지 않는다. 공유/운영 DB 금지.
  - 인과 고정: `slog.Error` 한 줄을 지운 빌드에서 새 시험이 빨개지고, 되살리면 초록인 것을 확인해 보고할 것.
- 위험과 피할 것:
  - **`s.audit(...)` 호출(ai.go:94)은 건드리지 말 것.** 제공자 오류 원문을 감사 테이블에 넣으면 감사 출력이 그 원문을 다시 내보낸다 — 감사에는 식별자만 간다는 운영자 방침에 걸린다. 이번 변경은 `slog` 한 줄까지다(`internalError`·`export.go:85` 가 raw err 를 slog 에 넘기는 것이 이 저장소의 선례다).
  - 사용자에게 가는 문구·상태코드·SSE 바이트열을 바꾸지 말 것. `ai_db_test.go:116` 의 기대 바이트열이 깨지면 그건 회귀다.
  - `proxyAIStream` 본문(ai.go:206~283)·`extractDelta`·`sendSSE`·`relationshipContext`·SQL·`AISettings` 는 건드리지 말 것. SSE 파서 개선은 이번 과제가 아니다(쪼개진 조각을 섞지 말 것).
  - 전역 기본 로거를 시험에서 바꾸면(`slog.SetDefault`) 반드시 `t.Cleanup` 으로 원복하고 병렬 실행하지 말 것 — AI 설정 행과 같은 이유로 전역 상태다.
  - 보호 경로 회피: `auth.go`·`throttle.go`·`internal/secure`·`internal/store/migrations`·`.github/workflows` 는 열지 않는다. 이번 과제는 마이그레이션 0건.
  - 과거 교훈: 2026-09-24 의 orbitAt 계열, 2026-09-21 의 OrbitPage 조기 반환은 verify 실패·no-change 로 끝났다 — 이번 과제는 그 경로와 무관하다.
  - 미확인: 로그 캡처 방식(`slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))`)으로 핸들러가 남긴 레코드를 실제로 읽을 수 있는지 이번 정찰에서 실행해 보지 못했다. 성립하지 않으면 `slog.NewTextHandler` 또는 작은 `slog.Handler` 구현으로 바꿀 것(표준 라이브러리 범위 안에서 해결 가능한 문제다). 또 `httptest.ResponseRecorder` 가 `http.Flusher` 를 만족한다는 것은 기존 시험이 200 SSE 경로를 통과하는 것으로 간접 확인했을 뿐 직접 단정은 하지 않는다.
- 차선 후보: `mcp.go` 의 나머지 세 도구(`mcp.go:170·175·183·202·250`)가 uuid 모양 아닌 `person_id` 를 uuid 컬럼에 그대로 넘겨 22P02 가 `pgx.ErrNoRows` 로 분류되지 않아 "대상을 찾을 수 없습니다" 대신 일반 메시지가 나가는 메시지 충실도 문제 — 기존 `looksLikeUUID` 재사용, `mcp_db_test.go` 재사용, 가치 2 / 위험 2 / 작업량 S.
