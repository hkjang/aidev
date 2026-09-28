- 과제: uuid 모양이 아닌 `person_id` 로 `GET /api/v1/memories/` 와 `POST /api/v1/ai/stream` 이 500 을 내는 것을 DB 앞에서 400 으로 (가치 3 / 위험 1 / 작업량 S)
- 왜: 지난 회차가 `data.go` 여섯 핸들러에 `looksLikeUUID` 가드를 붙였지만 같은 값을 uuid 컬럼에 넘기는 **다른 두 REST 경로**는 그대로 남았다 — `queryMemoriesLimit`(workflow.go:187)은 `m.person_id=NULLIF($2,'')::uuid` 로 캐스팅하고, `relationshipContext`(ai.go:85)는 `p.id=$1` 에 넘긴다. 둘 다 postgres `22P02` 가 그대로 `internalError` 로 감싸여 500 이 되므로, 호출자(API 키·웹)는 "내가 잘못 보냈다"를 알 수 없고 서버 로그에는 사용자 실수가 내부 오류로 쌓인다.
- 수용 기준:
  1) `GET /api/v1/memories/?person_id=not-a-uuid` 가 **400 `validation_error`** 를 돌려준다(500 아님). `person_id` 가 비어 있거나 정상 uuid 일 때의 동작·응답 모양(`{"memories":…,"count":…}`)은 그대로다.
  2) `POST /api/v1/ai/stream` 에 본문 `person_id` 가 uuid 모양이 아니면 **400 `validation_error`** 를 돌려준다. `person_id` 가 비어 있으면(전체 관계 요약 경로) 지금과 똑같이 진행한다.
  3) 두 경로 모두 **DB 에 닿기 전에** 끝난다는 것을 테스트가 증명한다 — `&Server{store: nil}` 로 실제 핸들러(`s.listMemories`, `s.streamAI`)를 불러, 가드가 있으면 400 이고 가드를 되돌리면 nil 포인터 패닉으로 빨개지는 것을 실패 재현으로 먼저 확인한다(`data_test.go:20` 의 `req.WithContext(context.WithValue(req.Context(), userContextKey, User{ID: "u1"}))` 패턴 그대로).
  4) 정상 uuid 는 거부되지 않는다 — 대문자 16진수 uuid 로도 가드를 통과하는 하위 시험을 둔다(2026-09-27 회차가 확정한 판단: postgres 가 같은 id 로 읽으므로 거부하지 않는다). 이미 있는 `TestLooksLikeUUID`(data_test.go:215)를 다시 쓰지 말고 **두 핸들러를 통해** 확인할 것.
- 건드릴 파일 (프로덕션 2개):
  - `internal/server/workflow.go:listMemories`(130~141) — `personID := r.URL.Query().Get("person_id")` 다음에 `if personID != "" && !looksLikeUUID(personID) { writeError(w, 400, "validation_error", …) ; return }`. `searchMemories`/`queryMemoriesLimit` 의 시그니처와 SQL 은 건드리지 말 것 — `queryMemories` 는 MCP·AI 와 공유한다.
  - `internal/server/ai.go:streamAI`(18~55) — 프롬프트 검증(29~32줄) **직후**, `s.readSetting`(35줄) **앞에** 같은 가드. 이 자리여야 수용 기준 3 의 nil-store 시험이 성립한다(설정 읽기가 먼저 돌면 그 지점에서 패닉한다).
  - 새 `internal/server/workflow_test.go`(또는 `ai_test.go`) — 이 패키지에 `workflow_test.go`·`ai_test.go` 는 아직 없다. 두 핸들러 시험을 한 파일에 모아도 된다.
  - 오류 코드·메시지는 새로 만들지 말고 기존 `validation_error` 를 쓸 것(2026-09-27 회차의 판단과 같게).
- 검증 명령:
  - `cd /home/hkjang/.cache/auto-improve-wt/orbit`
  - `gofmt -l internal/server` (출력 없어야 함)
  - `go vet ./...`
  - `go test -race -count=1 -v ./internal/server -run 'Memor|AI|Stream|Person'`
  - `go test -race -count=1 ./...` (DB 없이 전부 초록. DB 시험은 `ORBIT_TEST_DATABASE_URL` 없으면 SKIP)
  - DB 로 더 확인하고 싶으면(선택): docker `postgres:16-alpine` 을 띄워 `ORBIT_TEST_DATABASE_URL='postgres://…' go test -count=1 -v ./internal/server -run TestOrbit` — **포트 55433·55439·15434·55471·55481·55491·55521 은 다른 세션이 쓸 수 있으니 피할 것**. 공유·운영 DB 금지.
- 위험과 피할 것:
  - **`createMemory`(workflow.go:73~79)는 이미 400 이다** — `err != nil || !exists` 라서 `22P02` 도 `invalid_person` 400 으로 떨어진다. 여기는 건드리지 말 것(고치면 기존 오류 코드가 바뀐다). 확인함.
  - **`streamAI` 의 오류 우선순위가 한 가지 바뀐다**: AI 가 비활성(`503 ai_disabled`)이고 동시에 `person_id` 가 깨졌으면 지금은 503, 고치면 400 이다. 입력 검증을 DB 앞에 두는 이 저장소의 기존 관례(`validateInteractionInput`, data.go 여섯 가드)와 맞으므로 의도된 변경이고, 주석으로 그 이유를 남길 것. 이 한 줄이 비평에서 걸릴 수 있는 유일한 자리다.
  - **`POST /ai/stream` 에 정상 uuid 지만 없는 사람을 주면 지금도 500 이다**(`relationshipContext` 가 `pgx.ErrNoRows` 를 그대로 올리고 ai.go:52 이 `internalError`). 이번 범위 밖 — 손대면 파일이 늘고 SSE 오류 계약까지 얽힌다. 보류 아이디어로만 남기고 이번에 고치지 말 것.
  - **MCP(`mcp.go` 170·202·219)는 이번 범위 밖**. 거기는 이미 `isError:true` 로 곱게 떨어지므로 500 이 아니고, JSON-RPC 봉투의 오류 매핑 확인이 먼저다.
  - 보호 경로(`auth.go`·`throttle.go`·`internal/secure`·`internal/store/migrations`·`.github/workflows`)는 건드리지 않는다. 이번 과제는 그 어느 것도 필요 없다.
  - 마이그레이션·SQL 문자열 변경 없음. `docs/` 변경은 필요 없다(두 경로의 400 은 openapi.go 의 `operation()` 이 이미 "400 잘못된 요청" 으로 선언해 둔다 — openapi.go:43 확인함).
- 차선 후보: `openapi.go:openAPI` 핸들러 렌더를 httptest 로 고정 (가치 2 / 위험 1 / S) — 이 핸들러를 부르는 테스트가 여전히 0개다. 200·유효 JSON·`paths` 키 10개·각 operation 의 `summary`/`description`/`responses` 를 고정. DB 불필요, 프로덕션 코드 무변경. 단 다섯 회차 연속 차선이었다는 점은 가치가 낮다는 신호다.
