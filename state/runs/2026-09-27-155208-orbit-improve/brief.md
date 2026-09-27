- 과제: URL/본문으로 들어온 사람 id 가 uuid 모양이 아닐 때 `data.go` 의 여섯 핸들러가 500 을 내는 것을 DB 앞에서 404/400 으로 돌려보내기 (가치 3 / 위험 1 / 작업량 S)
- 왜: `people.id` 는 uuid 컬럼(`internal/store/migrations/001_init.sql:74`)인데 REST 핸들러들은 `chi.URLParam(r,"personID")` 값을 검사 없이 그대로 파라미터로 넘긴다. uuid 모양이 아닌 값이 오면 postgres 가 캐스팅 오류를 내고, 이 핸들러들은 그 오류를 `internalError` 로 감싸 **500** 을 낸다 — 같은 파일이 "없는 사람" 에는 404, 잘못된 본문에는 400 을 주는데 오직 이 입력만 서버 오류로 보인다. 호출자(API 키·웹)는 자기 요청이 잘못됐는지 서버가 고장났는지 구분할 수 없고, 서버 로그에는 사용자가 만든 문자열마다 에러가 쌓인다.
- 수용 기준:
  1) `GET|PUT|DELETE /api/v1/people/{personID}`, `GET|POST /api/v1/people/{personID}/links`, `DELETE /api/v1/people/{personID}/links/{linkID}` 에 uuid 모양이 아닌 id 를 주면 DB 를 타기 전에 **404 `not_found`** 로 끝난다(경로 파라미터는 "그런 자원이 없다" 가 맞는 답이고, 기존 404 메시지·코드를 그대로 쓴다 — 새 오류 코드를 만들지 말 것).
  2) `POST /api/v1/people/{personID}/links` 의 **본문** `person_id` 가 uuid 모양이 아니면 **400 `validation_error`** 로 끝난다(같은 핸들러의 kind·자기연결·strength 검증과 같은 취급).
  3) 정상 uuid 일 때의 동작(404/200/201/204, 응답 본문과 키)은 하나도 바뀌지 않는다.
  4) 테스트가 `&Server{store: nil}` 로 만든 서버에 각 핸들러를 직접 불러 **DB 앞에서 끝나는 것**을 증명한다 — 검증을 빠뜨리면 nil store 에 닿아 패닉으로 빨개진다(`data_test.go:16 TestCreateInteractionRejectsFutureOccurredAt` 이 이미 쓰는 방식이고, 경로 파라미터는 `chi.NewRouteContext()` + `rctx.URLParams.Add("personID", …)` 를 요청 컨텍스트에 넣어 준다). 그리고 모양 검사 자체는 순수 함수 단위 테스트로 표(정상 uuid·빈 문자열·대문자·중괄호 감싼 것·길이 틀린 것·하이픈 없는 것)를 고정한다. **DB 가 필요 없으므로 CI 에서도 실제로 돈다.**
- 건드릴 파일 (프로덕션 1개):
  - `internal/server/data.go` — 파일 수준에 순수 함수 하나(예: `looksLikeUUID(s string) bool`, `internal/id/id.New()` 가 만드는 `8-4-4-4-12` 소문자 16진수 모양을 기준으로; 표준 라이브러리만 쓰고 새 의존성을 넣지 말 것)를 더하고, 아래 여섯 곳의 맨 위에서 가드한다:
    - `getPerson`(141~148: `QueryRow … WHERE p.user_id=$1 AND p.id=$2` → `errors.Is(pgx.ErrNoRows)` 는 캐스팅 오류를 잡지 못해 500)
    - `updatePerson`(312, 351)
    - `deletePerson`(369~372)
    - `listPersonLinks`(612~613)
    - `createPersonLink`(638) — 경로 `personID` 는 404, 본문 `in.PersonID` 는 400. 본문 검증은 이미 있는 검증들 사이(`in.PersonID == personID` 자기연결 검사 근처)에 두어 `normalizeLink`(603)·`SELECT count(*) … id IN ($2,$3)`(663) 앞에서 끝나게 한다.
    - `deletePersonLink`(679~681) — `linkID` 도 uuid 컬럼인지 `internal/store/migrations/004_person_links.sql` 을 열어 한 번 확인하고(다른 표는 전부 `id uuid PRIMARY KEY`), uuid면 같은 가드를 적용.
  - `internal/server/data_test.go` — 위 4)의 핸들러 테스트와 순수 함수 표 테스트.
- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `gofmt -l internal/server` (출력 없어야 함)
  - `go vet ./...`
  - `go test -race -count=1 -v ./internal/server -run 'Person|Link|UUID'`
  - `go test -race ./...`
  - (선택) 고치기 **전에** 먼저 새 테스트를 돌려 빨개지는 것을 확인할 것 — 가드가 없으면 nil store 패닉이 난다. 이것이 "500 이 난다" 의 대리 증명이 아니라 "검증이 DB 앞에서 끝난다" 의 직접 증명이다.
- 위험과 피할 것:
  - **미확인(이 세션에서 실행하지 못함)**: "postgres 가 uuid 컬럼 비교에서 잘못된 문자열에 `22P02` 오류를 내고 0행이 아니다" 는 스키마(`id uuid`)와 핸들러의 `internalError` 배선을 읽어 추론한 것이고, 실제 DSN 으로 500 을 재현하지는 않았다(docker 실행이 이 세션에서 승인되지 않았다). 다행히 수용 기준 1~4 는 전부 DB 없이 증명되므로 구현이 이 추론에 매달리지 않는다. 확인하고 싶으면 `ORBIT_TEST_DATABASE_URL` 을 주고 실제 핸들러를 한 번 불러 보면 되지만, 포트 55433·55439·15434·55471·55481·55491 은 다른 세션이 쓸 수 있으니 피할 것.
  - `createInteraction`(443~453)은 **고치지 말 것** — 그 자리는 `if err := …Scan(&exists); err != nil || !exists` 라 캐스팅 오류도 이미 404 로 떨어진다. 손대면 멀쩡한 경로를 흔든다.
  - `internal/server/mcp.go`(170·195·213 의 `args.PersonID`)는 **이번 범위 밖**이다 — 응답 봉투가 JSON-RPC 로 달라 오류 매핑이 REST 와 같지 않다(그 매핑을 확인하지 않았다). 같은 헬퍼를 쓰는 후속 과제로 남기고, 이번 회차에 끌어들여 파일 수를 늘리지 말 것.
  - `normalizeLink`(603)·`orbitLinks`(698)·`getOrbit`·`timetravel.go` 는 현재/과거 두 경로가 공유한다 — 시그니처·질의를 바꾸지 말 것.
  - 보호 경로 금지: `auth.go`·`throttle.go`·`internal/secure`·`internal/store/migrations`(새 마이그레이션 불필요)·`.github/workflows`.
  - 오류 코드·메시지 문자열을 새로 만들지 말 것. 기존 `"사람을 찾을 수 없습니다."`(getPerson/updatePerson/deletePerson/createPersonLink)와 `"연결을 찾을 수 없습니다."`(deletePersonLink) 를 그대로 쓴다. `setAnchor` 의 메시지는 `"인물을 찾을 수 없습니다."` 로 다르다 — 통일하려 들지 말고 각 자리의 것을 쓴다.
  - 주석은 이 저장소 관례대로 한국어로 "왜 이렇게 했는가"(경로 파라미터는 404, 본문 필드는 400 으로 가른 이유)를 남길 것.
- 차선 후보: `openapi.go:openAPI` 핸들러 렌더를 테스트로 고정 — 이 핸들러를 부르는 테스트가 0개다. httptest 로 불러 200·유효 JSON·`paths` 키 10개·각 operation 의 summary/description/responses 만 고정하면 된다. DB 불필요, 프로덕션 코드 무변경.
