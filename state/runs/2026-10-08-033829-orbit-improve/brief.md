# 과제서 — 2026-10-08

- 과제: REST 두 핸들러(`createInteraction`·`createMemory`)가 사람 조회의 **DB 장애를 "사람을 찾을 수 없습니다"** 로 덮는 것을 고치고, 형제 핸들러 일곱 곳에 이미 있는 `looksLikeUUID` 가드를 이 두 곳에도 깔기 (가치 3 / 위험 2 / 작업량 S)

- 왜: `internal/server/data.go:505` 와 `internal/server/workflow.go:75` 는 `SELECT EXISTS(...)` 의 결과를 `if err != nil || !exists` 한 묶음으로 보고 400/404 를 내보낸다 — DB 가 끊기거나 질의가 실패해도 호출자에게는 "사람을 찾을 수 없습니다."(404) / "관계 인물을 확인해 주세요."(400) 로 나가고, `internalError` 를 지나지 않으므로 **서버 로그에도 원인이 한 줄도 남지 않는다**(respond.go:44~48 의 `slog.Error` 가 이 경로를 타지 않는다). v0.7.10 이 MCP 쪽 같은 자리(`mcp.go:337~342`)를 이미 두 줄로 갈라 놨고 그 주석이 이유를 그대로 적어 두었는데, REST 의 마지막 두 곳만 남았다(`grep -n 'err != nil || !exists' internal/server/*.go` 결과가 정확히 이 둘). 함께 `personID` 모양 가드도 깐다 — `{personID}` 를 받는 다른 핸들러 다섯 곳(data.go:182·357·418·665·697)과 본문 `person_id` 를 받는 두 곳(data.go:720, workflow.go:137)은 모두 `looksLikeUUID` 로 DB 앞에서 걸러내는데, 이 두 핸들러만 빠져 있어 모양이 어긋난 값이 uuid 컬럼까지 내려가 22P02 를 만든다. 가드가 없으면 오류 분기를 가르는 순간 그 22P02 가 500 으로 바뀌어 **호출자 실수가 내부 오류로 승격되는 회귀**가 생기므로, 두 변경은 한 묶음이어야 한다.

- 수용 기준:
  1) `POST /api/v1/people/{personID}/interactions` 에 모양이 어긋난 `personID`(`not-a-uuid`, 하이픈 없는 32자 16진수, `{...}` 로 감싼 형태, `1' OR '1'='1`)를 주면 **DB 에 닿지 않고** `404 not_found` + 기존 메시지 "사람을 찾을 수 없습니다." 로 끝난다. 상태 코드·오류 코드·메시지는 지금과 **한 글자도 달라지지 않는다**(형제 핸들러 data.go:182 와 같은 리터럴).
  2) `POST /api/v1/memories` 본문의 `person_id` 가 모양이 어긋나면 **DB 에 닿지 않고** `400 invalid_person` + 기존 메시지 "관계 인물을 확인해 주세요." 로 끝난다. 역시 지금 응답과 바이트가 같다. 빈 `person_id`("사람 없는 기억")는 지금처럼 통과해 DB 까지 간다.
  3) 두 핸들러에서 `exists` 조회의 `err` 와 `!exists` 가 갈라져, 조회가 실패하면 `internalError(w, r, err)` 로 500 `internal_error` 가 나가고 `slog.Error` 에 원인이 남는다. `!exists` 일 때만 지금과 같은 404 / 400 이 나간다.
  4) 테스트가 증명할 것: (a) 모양이 어긋난 네 입력 × 두 핸들러가 `callWithoutStore` 에서 `panicked == false` 이고(= DB 앞에서 끝났다) 기준 1·2 의 상태·코드로 끝난다, (b) **회귀 기준** — 소문자 uuid·대문자 uuid 는 두 핸들러 모두 `panicked == true`(= 가드를 지나 DB 까지 갔다), `createMemory` 의 빈 `person_id` 도 `panicked == true`, (c) 기존 `createInteraction` 의 입력 검증(유형·미래 시각)이 가드보다 **먼저** 돌아 모양이 어긋난 personID 라도 잘못된 `kind` 는 여전히 `400 validation_error` 로 끝난다 — 지금 순서가 그러하므로 바꾸면 안 된다.
  5) `go test -race -count=1 ./...` 가 DSN 없이 초록이고, 새 하위 시험은 CI 에서 실제로 돈다(DB 불필요).

- 건드릴 파일 (프로덕션 2개):
  - `internal/server/data.go:createInteraction` — `validateInteractionInput` 호출 **뒤**, `var exists bool` **앞**에 `if !looksLikeUUID(personID) { writeError(w, 404, "not_found", "사람을 찾을 수 없습니다."); return }`. 그리고 505줄의 `if err := ...Scan(&exists); err != nil || !exists {` 를 `if err := ...Scan(&exists); err != nil { internalError(w, r, err); return }` + `if !exists { writeError(w, 404, "not_found", "사람을 찾을 수 없습니다."); return }` 로 가른다. SQL·인자·순서는 그대로. 왜 갈랐는지 한국어 주석 한두 줄(`mcp.go:334~336` 주석의 어투를 따를 것).
  - `internal/server/workflow.go:createMemory` — `if in.PersonID != ""` 블록(75줄) 안에서 같은 모양으로. 모양 가드는 그 블록 **안**(즉 `in.PersonID != ""` 일 때만)에 두어 빈 값의 뜻("사람 없는 기억")을 건드리지 않는다: `if !looksLikeUUID(in.PersonID) { writeError(w, 400, "invalid_person", "관계 인물을 확인해 주세요."); return }`. 오류 분기는 `internalError(w, r, err)`.
  - `internal/server/data_test.go` — 기존 헬퍼 `personRequest(t, method, target, body, params)`(102~119줄)와 `assertAPIError`(121~)를 그대로 재사용. `callWithoutStore` 는 같은 패키지 `workflow_test.go:17` 에 있으니 그대로 부를 수 있다. 본문은 가드에 닿도록 유효해야 한다 — `{"kind":"meeting","summary":"x"}`(`occurred_at` 생략 시 now 로 채워지고 `weight` 는 1 로 보정된다).
  - `internal/server/workflow_test.go` — 기존 `TestListMemoriesRejectsMalformedPersonIDBeforeDB`(34줄)·`TestListMemoriesLetsWellFormedPersonIDReachDB`(68줄)의 표 구조와 네 입력 목록을 그대로 복사해 `createMemory` 용으로. 요청은 `httptest.NewRequest(POST, "/api/v1/memories", body)` + `context.WithValue(..., userContextKey, User{ID:"u1"})`(`memoriesRequest`(28줄)가 하는 것과 같은 모양, 다만 POST 본문).

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `gofmt -l internal/server` — 출력이 없어야 한다
  - `go vet ./...` / `go build ./...`
  - `go test -race -count=1 ./internal/server -run 'Interaction|Memor'` — 새 시험과 기존 교류·기억 시험을 함께
  - `go test -race -count=1 ./...` — 전체(DSN 없으면 DB 시험 SKIP)
  - 인과 고정: 모양 가드 두 줄만 지우면 기준 1·2 의 하위 시험이 빨개지고(패닉 또는 500), `err != nil` 분기만 되돌리면 — DB 없이는 증명할 수 없으므로 — 최소한 "되돌린 코드에서 기준 1·2 가 여전히 초록이어서 가드가 독립적임" 을 확인하고, 기준 3 은 코드로만 남긴다(**DSN 이 있으면** `ORBIT_TEST_DATABASE_URL=... go test -race -count=1 ./internal/server -run '<새 DB 시험>'` 로 `mcp_db_test.go:breakPeopleUserIDLookup` 방식의 뷰 주입을 재사용해 500 을 실측할 것 — 2026-10-05 회차가 `display_name` 을 깨뜨리면 플래너가 가지치기해 `SELECT EXISTS` 가 멀쩡히 성공한다는 것을 실측했으므로, 반드시 질의가 읽는 `user_id` 를 깨뜨려야 한다. 이것은 **선택**이고, 하지 않으면 기준 3 은 "실측 미확인" 으로 남긴다).

- 위험과 피할 것:
  - **메시지·상태 코드를 "개선" 하지 말 것.** 기준 1·2 는 기존 응답과 바이트가 같아야 한다. 모양이 어긋난 personID 에 404 대신 400 을 주고 싶어도 주지 말 것 — 지금 22P02 가 404 로 나가고 있고 형제 핸들러 다섯 곳이 404 리터럴을 쓴다.
  - **새 헬퍼를 만들어 기존 네 자리를 묶는 리팩터 금지.** `looksLikeUUID` 는 시그니처 그대로 호출만 한다. `writeError`/`internalError` 도 그대로.
  - `validateInteractionInput`·`decodeJSON`·`normalizeTags`·`activeDataKey`·`approvalSettings`·`recalculateRelationship`·`searchMemories`·`queryMemoriesLimit` 은 한 글자도 건드리지 말 것 — REST·MCP·AI 가 공유한다.
  - 가드를 `validateInteractionInput` **앞**으로 올리지 말 것. 지금은 입력 검증이 먼저이고 기준 4(c) 가 그 순서를 고정한다.
  - 보호 경로(`auth.go`·`throttle.go`·`internal/secure`·`internal/store/migrations`·`.github/workflows`)는 열지 말 것. `auth.go:466` 의 `EXISTS` 는 이미 `err != nil` 을 따로 보고 있어 손댈 것이 없다.
  - 과거 교훈: 2026-09-24 의 `memory_count`·`orbitAt` 포함 규칙 계열은 verify 실패 전력이 있다 — 이번 과제와 무관하니 끌어들이지 말 것. 성능(limit 내리기, 집계)도 이번 범위 밖.
  - 미확인으로 남기는 것: 기준 3 의 500 응답을 **실제 DB 장애로 실측하지 않았다**(정찰은 코드 경로만 읽었다). `mcp.go:337` 의 같은 분리가 2026-10-05 에 실 DB 로 증명된 것이 근거다.

- 차선 후보: `docs/API.md` 에 v0.7.8 의 `/personal/export` `complete:false`·`failed_section` 계약 문서화 (2/1/S) — 코드 변경 0. 다섯 회차 연속 차선이다. `docs/API.md`(62줄)에는 `/personal/export` 설명이 아예 없음을 이번에 직접 확인했다. 본문 모양은 `export.go:58~90` 과 `export_db_test.go` 의 실제 출력에서, 경로·메서드는 `server.go` 라우팅에서 옮겨 적을 것. `docs/cru-manual.md` 는 실제와 다르다는 기록이 있어 근거로 쓰지 말 것.
