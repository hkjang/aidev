# 과제서 — 2026-10-03 (orbit, base main@735c26b / VERSION 0.7.7)

- 과제: `GET /api/v1/personal/export` 가 섹션 중간 실패 시 **파싱조차 불가능한 미완결 JSON** 을 200 으로 내보내는 것을, 배열과 최상위 객체를 닫고 `"complete":false` 로 끝내 진단 가능하게 만들기 (가치 3 / 위험 1 / 작업량 S)

- 왜: `internal/server/export.go:75~87` 의 섹션 루프는 `section.write` 가 오류를 내면 `slog.Error` 를 찍고 **그 자리에서 `return`** 한다. 열어 둔 배열(`newJSONArray` 가 `,"people":[` 를 이미 써 버렸다)도, 최상위 객체(58줄의 `{`)도 닫히지 않으므로 사용자가 받는 파일은 `{"exported_at":"…","service_version":"0.7.7","user":{…},"people":[` 로 끊긴 **JSON 이 아닌 바이트열**이다. 78~79줄 주석은 "아래 complete 표시를 남기지 않는 것으로 온전하지 않다를 알린다" 고 선언하지만, 그 신호는 파서가 돌아야 읽을 수 있는 것이어서 실제로는 전달되지 않는다 — `jq`·`json.load`·브라우저 전부 "unexpected end of input" 만 낸다. 배열을 닫고 `,"complete":false}` 로 끝내면 주석이 의도한 계약이 실제로 성립하고, 사용자는 어느 섹션까지 받았는지(그 뒤 섹션 키가 아예 없다)를 파싱해서 알 수 있다. "관계는 당신의 것" 이라는 이 모듈의 선언(43~44줄)에서 내보내기 파일의 진단 가능성은 핵심이다.

- 수용 기준:
  1) `people` 섹션이 행 스트림 중간에 깨진 상태로 `exportData` 를 부르면, 응답 본문이 **`json.Unmarshal` 로 파싱되고** `complete` 가 `false` 다. (지금은 같은 입력에서 `json.Unmarshal` 이 `unexpected end of JSON input` 으로 실패한다 — 이것이 red 기준이다.)
  2) 같은 응답에서 이미 성공한 부분은 그대로 읽힌다: `exported_at`·`service_version`·`user.id` 가 들어 있고, 실패한 섹션 뒤의 섹션 키(`interactions`/`memories`/`links`)는 **아예 없다**(= 어디서 끊겼는지 파싱으로 알 수 있다). 상태 코드는 헤더가 이미 나갔으므로 **200 그대로**다 — 200 을 바꾸려 들지 말 것.
  3) 정상 경로 회귀: 아무것도 깨지 않은 상태에서는 본문이 파싱되고 `complete` 가 `true` 이며 `people`·`interactions`·`memories`·`links` 네 키가 모두 배열로 있다. 이 하위 시험은 **고치기 전에도 PASS** 여야 한다(가드가 정상 경로를 바꾸지 않았다는 기준).
  4) `gofmt -l internal/server` 무출력, `go vet ./...`, `go test -race -count=1 ./...` 가 DSN 없음/있음 양쪽에서 초록(DSN 없으면 새 DB 시험 SKIP).

- 건드릴 파일 (프로덕션 1개 + 테스트 1개):
  - `internal/server/export.go:75~87` `exportData` 의 섹션 루프 — 오류 분기에서 `return` 하기 전에 `array.close()` 를 부르고 `fmt.Fprint(w, ",\"complete\":false}")` 를 쓴다. 78~79줄 주석을 지금 코드가 실제로 하는 일로 고쳐 쓰고(왜 상태 코드를 못 바꾸는지 + 왜 닫아야 하는지), 89줄의 성공 주석은 그대로 둔다. 새 오류 코드·헬퍼·`writeError` 는 쓰지 않는다(헤더가 이미 나갔다).
    - 선택(권장): 어느 섹션이 끊겼는지도 담으려면 `,"complete":false,"failed_section":%q}` 로 `section.name` 을 함께 쓴다. 추가 전용(additive) 필드이고 `complete===true` 를 보는 기존 소비자를 깨지 않는다. 범위를 좁히고 싶으면 생략해도 수용 기준은 모두 만족한다.
  - `internal/server/export_db_test.go` (신규) — 실제 postgres 로 프로덕션 핸들러 `(&Server{store: st, version: "test"}).exportData` 를 직접 부른다. **대역(fake) 금지** — 운영자 규칙이다.

- 테스트를 어떻게 세우는가 (실제로 열어 확인한 것만):
  - `openTestStore(t)` (`timetravel_db_test.go:23`) — `ORBIT_TEST_DATABASE_URL` 없으면 `t.Skip`. `store.Open(ctx,dsn,nil)` 이므로 **Vault 는 nil** 이다.
  - **핵심**: 행이 하나라도 스캔되면 `exportPeople` 이 `s.exportKey → dataKeyVersion`(`data.go:52`) → `s.store.Vault.UnwrapKey` 로 내려가 nil Vault 에서 터진다. 그러므로 두 하위 시험 모두 **스캔되는 행이 0 이어야** 한다 — 아래 두 상황 모두 자연히 그렇게 된다:
    - 정상 경로 하위 시험: **사람이 0명인 별도 사용자**(`emptyUser := seedUser(t, st)`)로 부른다 → 네 섹션 모두 0행 → 키가 필요 없다 → `complete:true`. (`exportPeople` 은 `WHERE p.user_id=$1` 이므로 같은 DB 의 다른 사용자 행은 섞이지 않는다.)
    - 실패 경로 하위 시험: 두 번째 사용자(`boomUser`) 아래에 `seedNamedPerson(t, st, boomUser, "boom-"+id.New()[:8])`(`mcp_db_test.go:18`) + `seedRelationship` 을 심은 **뒤에** `breakPeopleRowStream(t, st)`(`mcp_db_test.go:39` — **이미 있다. 새로 쓰지 말고 재사용**) 로 `people` 을 동명 뷰로 가린다. `display_name::int` 가 행마다 평가되어 22P02 가 실행 중에 나고, `exportPeople` 질의의 `ORDER BY p.created_at` 이 블로킹 Sort 라 postgres 는 DataRow 를 **하나도 보내기 전에** ErrorResponse 를 보낸다 → `rows.Next()` 는 조용히 false, 오류는 `rows.Err()` 에만 남고(125줄이 그것을 리턴한다) 루프 본문은 한 번도 안 돌아 키 조회에 닿지 않는다.
  - **순서를 반드시 지킬 것 — 씨앗 먼저, 뷰 나중.** `breakPeopleRowStream` 이 씌우는 뷰는 select 목록에 `CASE` 식이 있어 postgres 가 자동 갱신 가능 뷰로 보지 않는다 → 뷰를 씌운 뒤 `INSERT INTO people` 은 실패한다. `mcp_db_test.go:95~129` 가 쓴 순서(사람·관계 심기 → 정상 경로 하위 시험 → `breakPeopleRowStream` → 실패 경로 하위 시험)를 그대로 따를 것.
  - **놓치기 쉬운 것**: `exportPeople` 질의는 `FROM people p JOIN relationships r ON r.person_id=p.id` 다. `seedRelationship(t, st, boomUser, personID)`(`timetravel_db_test.go:72`) 를 **반드시 함께** 심어야 그 사람이 조인 결과에 들어오고 뷰의 CASE 가 평가되어 스트림이 깨진다. 이것을 빠뜨리면 조인이 0행이 되어 오류가 나지 않고 시험이 초록으로 거짓 통과한다(= 실패 경로 하위 시험이 안 빨개지면 가장 먼저 의심할 자리).
  - 권장 골격(한 함수 + 두 하위 시험):
    ```go
    st := openTestStore(t)
    emptyUser := seedUser(t, st)            // 사람 0명 — nil Vault 를 건드리지 않는다
    boomUser := seedUser(t, st)
    p := seedNamedPerson(t, st, boomUser, "boom-"+id.New()[:8])
    seedRelationship(t, st, boomUser, p)
    t.Run("정상 내보내기는 파싱되고 complete=true", …)  // emptyUser — 고치기 전에도 PASS
    breakPeopleRowStream(t, st)
    t.Run("끊긴 섹션도 파싱되고 complete=false", …)      // boomUser — 고치기 전엔 FAIL
    ```
  - 요청 배선: `httptest.NewRequest(http.MethodGet, "/api/v1/personal/export", nil)` + `req.WithContext(context.WithValue(req.Context(), userContextKey, User{ID: userID, Username: "…", Role: "user"}))`. `exportData` 는 `userFromContext` 만 쓰고 chi URL 파라미터는 쓰지 않으므로 라우터를 세울 필요가 없다(`server.go:71` 이 `p.Get("/export", s.exportData)` 로 매는 것은 확인했다).
  - `s.audit`(`server.go:176`)은 성공 경로 끝에서 `audit_logs` 에 INSERT 한다. 실제 store 라 그냥 돈다. 실패 경로에서는 호출되지 않는다(= 지금도, 고친 뒤에도 그래야 한다 — 끊긴 내보내기를 성공으로 감사 기록하지 않는다). 시험 사용자 행은 `seedUser` 의 `t.Cleanup` 이 지우고 FK `ON DELETE CASCADE` 가 나머지를 따라간다.
  - `t.Parallel()` 금지 — `breakPeopleRowStream` 이 `people` 테이블 DDL 을 바꾼다. 뷰 원복은 그 헬퍼의 `t.Cleanup` 이 한다. 원복 확인 시험(`mcp_db_test.go:157 TestMCPSearchPeopleProbeRestoresSchema`)이 이미 있지만 그것은 go test 의 선언 순서에 의존하므로, 새 파일에도 같은 모양의 짧은 `pg_class` 확인 시험을 하나 더 두는 것을 권한다(`relkind='r'` + `people_probe` 없음).
  - `httptest.NewRecorder()` 는 `http.Flusher` 를 구현하므로 84~86줄의 Flush 분기가 실제로 돈다 — 별도 래퍼가 필요하지 않다.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  ```
  gofmt -l internal/server                 # 무출력
  go vet ./...
  go build ./...
  go test -race -count=1 ./...             # DSN 없이 — 새 DB 시험은 SKIP
  docker run -d --name orbit-pg-1003 -e POSTGRES_PASSWORD=orbit -e POSTGRES_DB=orbit \
    -p 127.0.0.1:55641:5432 postgres:16-alpine
  ORBIT_TEST_DATABASE_URL='postgres://postgres:orbit@127.0.0.1:55641/orbit?sslmode=disable' \
    go test -race -count=1 -v ./internal/server -run 'TestExport'
  ORBIT_TEST_DATABASE_URL='…' go test -race -count=1 ./...   # 전체도 초록
  ```
  포트 55641 은 미사용(이미 쓴 포트 55433·55439·15434·55471·55481·55491·55521·55537·55603·55617·49761 회피). 공유·운영 DB 절대 금지.
  **red→green 고정**: 고치기 전에 먼저 돌려 수용 기준 1 하위 시험이 `unexpected end of JSON input` 으로 빨개지는 것을 확인하고, 고친 뒤 초록, 그리고 `array.close()` + `complete:false` 두 줄만 되돌려 다시 빨개지는 것까지 확인해 인과를 고정할 것.

- 위험과 피할 것:
  - **상태 코드를 바꾸지 말 것.** `w.WriteHeader` 가 이미 암시적으로 200 으로 나갔다(58줄의 첫 `Fprintf`). 500 으로 바꾸려는 시도는 `superfluous WriteHeader` 경고만 낳고 본문은 여전히 섞인다.
  - **섹션 함수 네 개(`exportPeople`/`exportInteractions`/`exportMemories`/`exportLinks`)와 SQL·`jsonArray` 의 시그니처를 건드리지 말 것.** 이들은 이미 `rows.Err()` 를 리턴하고 있고(125·158·192·216줄) 이번 결함과 무관하다. `jsonArray.close()` 가 쓰기 오류를 무시하는 것도 그대로 둔다(클라이언트가 끊긴 경우 더 쓸 의미가 없다).
  - 59~64줄의 `user` 인코딩 실패 분기도 똑같이 미완결 본문을 남기지만, 그것은 **쓰기 오류**(클라이언트가 이미 끊김)라 더 쓰는 것이 무의미하다 — 이번 범위에 넣지 말고 손대지 말 것. 범위는 "섹션 루프의 질의 실패" 하나다.
  - `internal/store/migrations`·`auth.go`·`throttle.go`·`internal/secure`·`.github/workflows` 는 열지 않는다. `openapi.go:31` 의 `/personal/export` 항목은 `operation()` 한 줄이고 응답 스키마를 담지 않으므로 바꿀 것이 없다 — openapi 확장 리팩터 금지(프로필 "위험 구역").
  - docs: `docs/*.md` 에 `/personal/export` 설명이 **없음을 grep 으로 확인**했다. 문서 변경은 불필요하며, 쓰고 싶다면 `docs/API.md` 에 한 단락만 더하고 소스에서 경로·메서드(`GET /api/v1/personal/export`)를 확인한 대로 적을 것.
  - 과거 교훈: 뷰 주입 시험은 원복을 빠뜨리면 같은 컨테이너의 다른 DB 시험을 전부 깬다. `t.Cleanup` 등록을 DDL **직후**에 두는 기존 패턴을 그대로 따를 것.

- 기준선(이번 정찰이 실제로 돌림): `go test -count=1 ./...` — 전부 `ok`(server 0.017s, config·secure·scripts ok, DSN 없음). DB 시험은 돌리지 않았다(DSN 미제공) — 위 실패 경로의 postgres 거동은 `mcp_db_test.go` 가 같은 기법으로 v0.7.7 에서 실증한 것에 기대고 있고, **export 질의에서의 재현은 미확인**이다.

- 차선 후보: `internal/server/ai.go:306~312 safeAIError`(실제로 열어 확인했다: `message := err.Error()` → `if len(message) > 300 { message = message[:300] }`) 의 `message[:300]` 이 바이트 슬라이스라 한국어(룬 3바이트) 제공자 오류 꼬리를 깨진 룬으로 잘라 SSE `error` 메시지 끝이 U+FFFD 로 나가는 것을, 룬 경계에서 자르도록 고치기 (가치 2 / 위험 1 / 작업량 S — DB 불필요, 순수 함수 단위 시험으로 증명 가능). 1순위가 성립하지 않을 때(예: `breakPeopleRowStream` 재사용이 export 질의에서 뜻대로 동작하지 않을 때) 이쪽으로 갈 것.
