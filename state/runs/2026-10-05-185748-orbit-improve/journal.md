# 회차 노트 2026-10-05-185748-orbit-improve — orbit
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 18:57] base pinned — main@f73bbe9
- [러너 18:57] autonomy release — 

## 정찰 노트
- 보류 목록의 1순위 후보(mcpCreateMemory 의 title/content 메시지 충실도)를 코드로 재확인해 그대로 골랐다. 차선인 docs/API.md 문서화는 세 회차 연속 쓰이지 않아 가치 신호가 약하고, list_memories 의 limit·orbitAt 성능 계열은 실측이 없어 "측정 없는 성능 변경 금지" 에 걸려 또 제쳤다.
- 확인한 것: mcp.go:292~297 의 errors.New 와 239~246 의 매핑이 지금 코드와 일치, 기존 create_memory 시험 케이스는 전부 title·content 를 넘기므로(mcp_test.go:71·176~177) 새 가드가 회귀를 내지 않음, tools/list 는 이미 required:[title,content], `go test -count=1 ./...` 기준선 초록.
- 추측으로 남긴 것(실행하지 않음): store nil 로도 빨강이 보인다는 것은 코드 경로를 읽어 판단했다 — mcpCreateMemory 가 첫 DB 접촉(mcp.go:301·308) 앞에서 반환하므로 패닉 없이 일반 메시지가 나올 것이다. 구현자는 이것을 먼저 빨강으로 확인하고 시작할 것.
- 조심할 것: 새 가드를 person_id 가드 뒤에 둘 것(앞이면 수용 기준 3 이 깨진다). mcpCreateMemory 안의 기존 검사는 사전조건이므로 지우지 말 것. isError 네 자리를 공통 헬퍼로 묶는 리팩터는 지난 회차가 의도적으로 남겨 둔 것이니 하지 말 것.
- 프로필은 0일 전이고 이번에 읽은 mcp.go·mcp_test.go·workflow_test.go 와 어긋나는 데가 없어 새로 쓰지 않았다.
- [러너 19:01] scout done — MCP `orbit_create_memory` 의 title/content 누락이 "요청을 처리하지 못했습니다." 로 나가는 것을, DB 앞에서 무엇이

## 구현 노트
- 무엇을 왜: `mcpPersonIDError` 바로 뒤에 `mcpMemoryFieldError(title, content)` 가드를 넣어, `orbit_create_memory` 의 title/content 누락이 "요청을 처리하지 못했습니다." 대신 빠진 필드 이름을 담은 한국어 메시지로 나간다(HTTP 200 + `result.isError:true`, 기존 봉투 그대로). 프로덕션 1파일(`internal/server/mcp.go`), 테스트 1파일, 커밋 1개(92495ef).
- 확신 없는 곳·검증 못 한 것: 메시지 문구 자체는 취향 판단이다 — 시험은 "빠진 필드 이름이 들어 있고 채워 보낸 필드는 들어 있지 않다" 만 고정하므로 문구를 다듬어도 깨지지 않는다. DB 가 있는 경로는 이번에 전혀 확인하지 않았다(필요가 없다 — 가드가 DB 앞에서 끝난다). `mcpCreateMemory` 의 정상 경로가 실제 postgres 에서 전과 같은지는 기존 `mcp_db_test.go` 에 맡겼고 DSN 으로 돌리지 않았다. 다만 가드는 값을 읽고 조기 반환만 하므로 `mcpCreateMemory` 가 받는 인자는 코드상 그대로다(`normalizeTags`·`TrimSpace` 손대지 않음).
- 일부러 하지 않은 것: `mcpCreateMemory` 안의 `title=="" || content==""` 는 사전조건이라 지우지 않고 주석만 남겼다(이제 도달하지 않는 방어선). isError 리터럴이 이제 다섯 자리로 늘었지만 공통 헬퍼로 묶지 않았다 — 지난 두 회차가 같은 판단으로 남겼고 ideas.json 에 `rejected` 로 못 박았다. `tools/list` 스키마(이미 `required:[title,content]`)·공유 질의·`orbit_search_people` arm 은 건드리지 않았다.
- 다음 역할이 조심할 것: 새 시험 13개(핸들러 6 + 순수 함수 6 + 순서 1)는 **DB 가 필요 없고 CI 에서 실제로 돈다**. `TestMCPPersonIDGuardRunsBeforeMemoryFieldGuard` 는 고치기 전에도 PASS 였다 — 새 가드를 person_id 가드 뒤에 두었다는 회귀 기준이므로, 순서를 바꾸면 이것만 빨개진다. 인과는 두 갈래로 따로 고정했다: 가드 블록만 지우면 핸들러 6개 빨강, 헬퍼의 `TrimSpace` 만 지우면 공백 케이스 3개 + 순수 함수 1개 빨강.
- [러너 19:06] brief accepted — 채택 — 지정한 자리(mcp.go 의 `errors.New` 와 오류 매핑, `mcpPersonIDError` 의 모양과 위치, 기존 create_memory 시험 케이스가 전부 
- [러너 19:06] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인한 것: 가드 블록만 떼면 새 핸들러 하위 시험 6개가 원장의 `- 실패 재현:` 과 글자까지 같은 출력으로 빨개지고, 두 가드 순서를 뒤집으면 `TestMCPPersonIDGuardRunsBeforeMemoryFieldGuard` 만 빨개진다 — 시험이 실제로 바뀐 경로를 지난다. `gofmt -l internal/server`(무출력)·`go vet`·`go test -race -count=1 ./...` 초록.
- 구현자가 비워 둔 자리(DB 경로 미확인)를 메웠다: 격리 postgres:16-alpine(포트 55723, 사용 후 제거)에 DSN 을 주고 `./internal/server` 전체 PASS — `TestMCPCreateMemorySeparatesLookupFailureFromMissingPerson` 과 create_memory arm 이 SKIP 아닌 PASS. 정상 경로가 전과 같다는 주장이 실측으로 확정됐다. 프로필 기사용 포트에 55723 추가 권함.
- 못 본 것: 웹(`npm`) 쪽은 변경이 없어 돌리지 않았다. 판정은 approve, 차단 없음(스코프 검사가 여전히 가드보다 앞, 메시지가 입력값을 반사하지 않음, 새 개인정보 수집 없음).
- 다음 회차가 알아야 할 것 ①: `mcp.go:234` 의 `_ = json.Unmarshal` 때문에 `"title": 123` 같은 타입 오류는 모든 인자가 영값이 되어 이제 "title과 content가 비어 있습니다." 가 나간다 — content 를 보냈는데도 그렇다. 전보다 나쁘지는 않으나 후보감이다.
- 다음 회차가 알아야 할 것 ②: `mcp_test.go:64`·`mcp_db_test.go:228` 의 "title·content 가 없으면 person_id 에 닿기 전에 끝난다" 주석은 새 순서 시험과 반대다(v0.7.10 부터 이미 틀린 문장, 이번 diff 가 건드린 줄 아님). 릴리즈 노트는 봉투·상태코드 변화 없음을 명시하면 된다.
- [러너 19:10] review approved — 리뷰 승인 (risk=low)
- [러너 19:10] pr created — https://github.com/hkjang/orbit/pull/21
- [러너 19:12] ci passed — 검사 1개 모두 success
- [러너 19:12] merge done — 92495ef
- [러너 19:18] release published — v0.7.11
