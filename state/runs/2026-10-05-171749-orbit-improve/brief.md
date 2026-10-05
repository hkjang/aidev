- 과제: MCP `tools/call` 의 세 도구가 uuid 모양이 아닌(또는 빈) `person_id` 를 uuid 컬럼에 그대로 넘겨, 외부 에이전트가 고칠 수 없는 일반 메시지만 받는 것을 DB 앞에서 구체적 메시지로 막기 (가치 3 / 위험 2 / 작업량 S)
- 왜: `orbit_get_relationship`·`orbit_list_memories`·`orbit_create_memory` 는 `args.PersonID` 를 검사 없이 uuid 컬럼(`mcp.go:178` 의 `p.id=$2`, `workflow.go:195` 의 `NULLIF($2,'')::uuid`, `mcp.go:258` 의 `WHERE id=$1`)에 넘기므로, 에이전트가 id 대신 사람 이름을 보내면 postgres `22P02` 가 되고 `pgx.ErrNoRows` 로 분류되지 않아 `mcp.go:232~239` 에서 "요청을 처리하지 못했습니다." 한 문장만 돌아간다 — 두 스키마(`mcp.go:116·117`)에는 `format: uuid` 도 설명도 없어 애초에 모양을 알려주지도 않는다. 무엇이 잘못됐는지 말해 주면 호출한 모델이 `orbit_search_people` 로 id 를 먼저 찾는 식으로 스스로 고칠 수 있고, REST 쪽이 2026-09-27·09-28 두 회차에 `looksLikeUUID` 로 이미 닫은 구멍이 MCP 표면에만 남아 있는 상태가 끝난다.
- 수용 기준:
  1) `tools/call` 로 세 도구(`orbit_get_relationship`·`orbit_list_memories`·`orbit_create_memory`)에 모양이 어긋난 `person_id`(`not-a-uuid`, 하이픈 없는 32자 16진수, `{…}` 로 감싼 형태, `1' OR '1'='1`)를 주면 HTTP 200 JSON-RPC 봉투 + `result.isError == true` + `person_id` 가 uuid 모양이어야 함을 말하는 한국어 메시지가 돌아오고, **질의에 닿지 않는다**.
  2) `orbit_get_relationship` 에 `person_id` 가 비었거나 없으면(이 도구는 `required: ["person_id"]`) 같은 방식으로 "person_id 가 필요하다" 는 구체적 isError 가 돌아오고 질의에 닿지 않는다. 지금은 `p.id=''` 가 역시 22P02 로 일반 메시지가 된다.
  3) 회귀 기준 — 정상 uuid(소문자 **와 대문자** 16진수 둘 다: 대문자는 postgres 가 같은 id 로 읽으므로 거부하면 지금 찾아지던 것이 사라진다, 2026-09-28 판단)는 세 도구 모두 가드를 지나 DB 까지 가고, `orbit_list_memories`·`orbit_create_memory` 의 **빈** `person_id`(사람을 가리지 않는 전체 목록 / 사람 없는 기억)는 지금처럼 그대로 통과한다.
  4) `mcpCreateMemory`(mcp.go:256~259)가 "조회가 실패했다" 와 "그런 사람이 없다" 를 구분한다 — `Scan` 이 오류를 내면 그 오류를 그대로 돌려(일반 메시지), `!exists` 일 때만 `pgx.ErrNoRows` 를 돌려 "대상을 찾을 수 없습니다." 가 되게 한다. 지금은 `err != nil || !exists` 가 한 묶음이라 DB 장애도 "대상을 찾을 수 없습니다." 로 보이고 원인이 사라진다.
  5) `tools/list` 의 `orbit_list_memories`·`orbit_create_memory` 스키마에서 `person_id` 가 `{"type":"string","format":"uuid","description":"…"}` 가 된다(`orbit_get_relationship` 은 이미 `format: uuid`). 도구 이름·`required` 배열·다른 속성은 바꾸지 않는다.
  6) 시험은 DB 없이 CI 에서 돈다 — `workflow_test.go:17` 의 `callWithoutStore`(`&Server{}` 로 불러 패닉 여부로 "DB 에 닿았는가" 를 판정하는 기존 헬퍼)를 그대로 재사용하고, 요청 본문은 `mcp_db_test.go:63` 의 `callSearchPeople` 과 같은 모양(JSON-RPC `tools/call` + `userContextKey` 에 `User{ID:…}`, `authContextKey` 는 넣지 않음 — `requestHasScope` 는 authInfo 가 없으면 통과시킨다)으로 만든다.
- 건드릴 파일:
  - `internal/server/mcp.go:mcpCall` — `orbit_get_relationship`(169~173)·`orbit_list_memories`(201~206)·`orbit_create_memory`(219~227) 세 arm 의 `json.Unmarshal` **직후**, 첫 질의 앞에 `looksLikeUUID`(data.go:73, 시그니처 그대로) 기반 가드를 넣고 `s.rpcWrite(w, req.ID, <isError 맵>, nil); return`. 빈 값 허용 여부가 도구마다 다르므로(기준 2·3) 작은 헬퍼 하나(예: `mcpPersonIDError(personID string, required bool) string`, 문제없으면 `""`)로 세 곳이 같은 판단을 쓰게 할 것. isError 맵은 `mcp.go:133`(권한 거부)의 리터럴과 같은 모양을 쓰고, **기존 세 곳(133·229·237)을 헬퍼로 바꿔 쓰는 리팩터는 하지 말 것**.
  - `internal/server/mcp.go:mcpCreateMemory` (249~) — 기준 4 의 두 줄 분리. `topics`/`title`/`content` 처리와 그 뒤 INSERT·승인 경로는 건드리지 않는다.
  - `internal/server/mcp.go:mcpTools` (116·117) — 기준 5 의 스키마 두 줄.
  - `internal/server/mcp_test.go` (신규) — DB 불필요 시험. 기준 1~3 을 세 도구 × 입력 표로 고정하고, 정상/빈 경로가 **고치기 전에도** PASS 였음을 남길 것(가드가 정상 입력을 삼키지 않는다는 회귀 기준).
- 검증 명령:
  - `gofmt -l internal/server` (출력 없어야 함)
  - `go vet ./...` · `go build ./...`
  - `go test -race -count=1 -v ./internal/server -run 'MCP'` — 신규 시험 포함. 고치기 전 빨강 → 고친 뒤 초록 → 가드 한 줄만 되돌려 다시 빨강까지 확인해 인과를 고정할 것.
  - `go test -race -count=1 ./...` — 정찰이 이번에 실제로 돌린 것은 `-race` 없는 `go test -count=1 ./...` 이며 DSN 없이 전부 `ok`(server/config/secure/scripts, DB 시험은 SKIP). `-race` 판은 미확인이나 CI 가 `-race` 로 돌리므로 구현 검증은 `-race` 로 할 것.
  - (선택) 실 DB 로 22P02 를 눈으로 확인하려면 격리 `postgres:16-alpine` 를 **처음 쓰는 포트**로 띄우고 `ORBIT_TEST_DATABASE_URL='postgres://…' go test -race -count=1 -v ./internal/server -run 'MCP'`. 기사용 포트 금지: 55433·55439·15434·55471·55481·55491·55521·55537·55603·55617·55641·55659·49761. 공유/운영 DB 금지.
- 위험과 피할 것:
  - `queryMemories`/`queryMemoriesLimit`(workflow.go:190·194)·`personConnections`(ai.go:201)·`escapeLike` 의 시그니처와 SQL 은 한 글자도 바꾸지 말 것 — REST·AI·MCP 가 공유한다. 가드는 **호출자 쪽(mcpCall)** 에만 둔다.
  - `orbit_search_people` arm 의 `rows.Err()` 가드(mcp.go:160~167, v0.7.7)와 `mcp.go:232~239` 의 기존 오류 매핑·`requestHasScope`(244~247)는 건드리지 않는다. 기존 일반 메시지 문구 "요청을 처리하지 못했습니다."·"대상을 찾을 수 없습니다." 도 그대로 둔다(mcp_db_test.go:148 이 전자를 문자열로 고정하고 있다).
  - MCP 오류는 HTTP 200 + `result.isError:true` 가 저장소 계약이다. JSON-RPC `error`(-32602)로 바꾸지 말 것 — 봉투가 바뀌면 `mcp_db_test.go` 의 파싱이 깨지고, 모델이 못 보는 자리로 메시지가 옮겨간다.
  - `auth.go`·`throttle.go`·`internal/secure`·`internal/store/migrations`·`.github/workflows` 는 열지 않는다(보호 경로). 마이그레이션 불필요.
  - 대문자 uuid·빈 person_id 를 함께 거부하면 **지금 되던 기능이 사라진다** — 기준 3 이 그 선을 지킨다. 과거 회차가 이 지점을 명시적으로 판단했으니 되풀이하지 말 것.
  - 프로덕션은 `mcp.go` 한 파일로 끝낸다(파일 1 + 시험 1). 다른 uuid 검증 자리를 찾아 넓히지 말 것.
- 차선 후보: `docs/API.md` 에 v0.7.8 이 만든 `GET /api/v1/personal/export` 의 `complete:false`·`failed_section` 계약을 문서화 (가치 2 / 위험 1 / 작업량 S). 코드 변경 0. 경로·메서드는 `internal/server/server.go` 라우팅에서 확인한 대로 적고, 본문 모양은 `internal/server/export.go:58~90` 과 `export_db_test.go` 의 실제 출력에서 옮겨 적을 것.
