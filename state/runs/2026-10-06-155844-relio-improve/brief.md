- 과제: VOC 기록 `eventType` 값 목록을 저장소의 기존 관용(`voice.CauseEvidenceLevels` 식 공유 var)으로 한 곳에 모으고, 그 목록이 `customer_voice_events.event_type` CHECK 의 부분집합임을 계약 테스트로 고정 (가치 3 / 위험 1 / 작업량 S)

- 왜: `{CUSTOMER_CONTACT, COMMENT, ESCALATED}` 가 지금 **두 곳에 따로 하드코딩**돼 있다 — MCP 도구 스키마가 공표하는 `internal/mcp/server.go:683` 의 인라인 `[]string{...}` 과, REST·MCP 두 호출자가 공통으로 지나가는 집행 지점 `internal/voice/service.go:821` 의 삼중 `!=` 비교. 한쪽만 손대면 `go build`·`go vet`·기존 테스트가 전부 green 인 채로 MCP 에이전트가 공표된 값을 보내고 `invalid eventType` 을 받는다(또는 공표되지 않은 값이 조용히 통과한다). 같은 패키지 안에 **이미 올바른 관용이 있다**: `internal/voice/workspace.go:61-66` 의 `CauseEvidenceLevels`/`KnowledgeStatuses` 는 공개 `[]string` + `setOf()` 사설 집합 한 쌍이고, `internal/mcp/voice_tools.go:128` 과 `internal/server/voice_workspace.go:21` 이 그 **같은 var 를 참조**한다. eventType 만 그 관용에서 빠져 있다. 고치면 공표와 집행이 컴파일 시점에 같은 값이 되고(테스트가 아니라 타입 시스템이 지킨다), 남는 유일한 손 목록인 DB CHECK 와의 관계는 계약 테스트가 지킨다.

- 수용 기준:
  1) `internal/voice/workspace.go` 의 기존 var 블록에 `CommentEventTypes = []string{"CUSTOMER_CONTACT", "COMMENT", "ESCALATED"}` 와 `commentEventTypes = setOf(CommentEventTypes)` 가 추가되고, `internal/voice/service.go:821` 의 삼중 `!=` 가 `if !commentEventTypes[eventType]` 로 바뀐다. 오류 문장 `"invalid eventType"` 은 **한 글자도 바꾸지 말 것**(`internal/server/voice.go:86` → `s.serviceError` → `server.go` 의 영어 substring 분류가 이 문장을 읽는다; 분류 동작 변화 금지).
  2) `internal/mcp/server.go:683` 의 인라인 `[]string{"CUSTOMER_CONTACT", "COMMENT", "ESCALATED"}` 가 `voice.CommentEventTypes` 로 바뀐다(해당 파일은 `internal/voice` 를 **이미 import** 한다 — server.go:24). 한국어 설명 문자열은 그대로 둔다.
  3) 새 계약 테스트가 `CommentEventTypes` ⊆ `customer_voice_events.event_type` 의 CHECK 값 집합임을 증명한다. 집합 **동일이 아니라 부분집합**이다 — CHECK 는 `CREATED`·`STATUS_CHANGE`·`ASSIGNED`·`RESOLVED`·`REOPENED`·`SATISFACTION`·`KNOWLEDGE_REVIEW` 를 포함하는 의도된 상위집합이고(서비스가 상태 전이·지식 검토 때 스스로 쓰는 값들), 이 7개는 사용자가 보낼 수 없다. 실패 시 어느 값이 CHECK 에 없는지와 CHECK 자리(`file:line`)를 찍을 것.
  4) 섭동으로 red 를 **눈으로 확인**할 것: ① `CommentEventTypes` 에 `"BOGUS"` 추가 → 기준 3 테스트가 red(되돌림). ② `migrations/016_voice_workspaces.sql:56` 의 값 목록에서 `'ESCALATED'` 삭제 → 같은 테스트가 red, `git checkout -- migrations/` 로 되돌리고 `git status` 로 migrations 가 깨끗함을 확인. ③ `internal/voice/service.go` 에서 `commentEventTypes` 를 `causeEvidence` 로 오타 → 기준 5 의 거부 테스트가 red(되돌림).
  5) `internal/voice` 안의 in-package 테스트가 **실제 프로덕션 함수**로 거부 경로를 증명한다: `(&Service{}).Comment(context.Background(), p, "v-1", "BOGUS", "note", crm.RequestMeta{})` 가 `invalid eventType` 을 돌려준다. 손으로 만든 대역을 쓰지 말 것 — 실제 `*voice.Service` 와 실제 `*auth.Principal` 을 쓴다.

- 건드릴 파일 (프로덕션 3 + 테스트 2, 전부 작다):
  - `internal/voice/workspace.go:61-66` — 기존 `var (...)` 블록에 두 줄 추가(`CommentEventTypes`, `commentEventTypes`). 주석 한 줄로 "MCP 스키마와 집행이 같은 목록을 본다"는 뜻을 적을 것. 추가 뒤 `gofmt` 로 정렬 재조정.
  - `internal/voice/service.go:818-822` — `Service.Comment` 의 `eventType != ... && ... && ...` 를 집합 조회로. `strings.ToUpper`(820)는 그대로 유지(대소문자 관용이 기존 동작이다). 836줄 `if eventType == "CUSTOMER_CONTACT"` 의 SLA 분기는 **건드리지 말 것**.
  - `internal/mcp/server.go:683` — `record_voice_response` 스키마의 인라인 리터럴 → `voice.CommentEventTypes`.
  - `internal/api/enum_contract_test.go` (끝에 새 테스트 함수 추가, 현재 404줄) — 기존 `migrationCheckConstraints(t) (map[column]checkConstraint, int)` 과 `column{table, name}`·`sortedSet()` 을 **그대로 재사용**해 `column{table: "customer_voice_events", name: "event_type"}` 을 조회하고 부분집합을 확인한다. `package api` 는 지금 `migrations` 만 import 하지만 `internal/voice` 를 추가해도 **순환이 없다**(확인: `internal/api` 의 프로덕션 import 는 `platform/version` 하나뿐, `internal/voice` 는 `auth`·`crm`·`audit`·`mail`·`platform/*` 만 쓴다). 파일 수준 상수 `knownEnumQueries = 26`·`knownCheckConstraints = 42`·`knownMappedEnums = 13` 은 **바뀌지 않아야 한다**(바뀌면 뭔가 잘못 건드린 것).
  - `internal/voice/service_test.go` (끝에 새 테스트 함수 추가) — 기준 5. 기존 테스트(`TestTransitionsRequireAPathThroughHandling` 등)와 같은 in-package 스타일.

- 검증 명령:
  - `go test ./internal/voice/ ./internal/mcp/ ./internal/api/ -count=1 -v` (베이스라인 3개 전부 `ok` 인 것을 이번 정찰에서 실측했다)
  - `go test ./...` — 23 ok / FAIL 0 이어야 한다
  - `go test -race ./internal/voice/ ./internal/api/`
  - `go build ./...` ; `go vet ./...` ; `gofmt -l internal/voice internal/mcp internal/api` (무출력)
  - `./scripts/check-env-contract.sh` ; `./scripts/check-static-assets.sh`
  - `git diff --check` 무출력 ; 최종 `git status --short` 가 의도한 5파일만
  - `make test` 는 **돌리지 않아도 된다** — 이번 변경은 Go 전용이고 `npm ci` 가 네트워크를 탄다.

- 위험과 피할 것:
  - **함정 1 (가장 중요) — `auth.Principal` 의 권한은 비공개 `perm` 맵으로 판정된다.** `internal/auth/service.go:77` 의 `Has` 는 `p.Permissions` 를 보지 않고 `p.perm` 을 본다(`perm` 은 소문자 비공개 필드, `service.go:45`, `service.go:408` 에서만 채워진다). 그래서 `package voice` 테스트에서 `&auth.Principal{Permissions: []string{"voice:write"}}` 를 만들면 `Require` 가 **실패**하고 `permission voice:write is required` 가 돌아와 테스트가 엉뚱한 이유로 통과/실패한다. 쓸 것은 `&auth.Principal{IsBootstrap: true}` 다 — `Has`(77줄)가 `IsBootstrap` 를 먼저 보고 `keyAllows`(82줄)는 `KeyID == ""` 일 때 true 를 돌려준다. 실제 부트스트랩 관리자가 쓰는 실제 필드이므로 대역이 아니다.
  - **함정 2 — 유효한 eventType 으로 `Comment` 를 호출하지 말 것.** `Comment` 의 순서는 `auth.Require` → note 공백 검사 → `strings.ToUpper` → eventType 검사 → `s.Get(...)` 이다(`service.go:818-826`). **잘못된** eventType 은 `s.Get` 전에 돌아오므로 `Service{}`(DB nil)로 안전하지만, **유효한** eventType 은 `s.Get` 까지 내려가 nil `*pgxpool.Pool` 로 패닉한다. 즉 기준 5 는 거부 경로만 증명할 수 있다. 유효 값 쪽은 `commentEventTypes` 집합 멤버십을 직접 확인하는 것으로 충분하고, "수락된다"고 주장하지 말 것(폐기용 PostgreSQL 없이는 증명 불가).
  - note 는 공백이 아닌 문자열을 넘길 것 — 그러지 않으면 eventType 검사 전에 `note is required` 로 돌아온다.
  - `migrations/` 와 `.github/workflows/` 는 **최종 diff 에서 0줄**이어야 한다(섭동 ②는 반드시 되돌릴 것). 2026-09-06 롤백·자율화 강등이 워크플로 경로에서 났다.
  - `internal/mcp/server.go` 는 긴 줄 파일이다 — 683줄 **그 한 줄의 리터럴만** 바꾸고 파일 전체를 재포매팅하지 말 것(프로필의 명시적 금지 사항).
  - **기본값 두 개는 서로 다르지만 이번에 통일하지 말 것.** MCP `record_voice_response` 는 eventType 생략 시 `CUSTOMER_CONTACT`(`internal/mcp/server.go:1021-1023`), REST `POST /voices/{id}/events` 는 `COMMENT`(`internal/server/voice.go:82-84`). `CUSTOMER_CONTACT` 는 `first_responded_at` 를 채워 최초 응답 SLA 를 충족시키므로(`service.go:836`) 의미가 다르다. 이번 정찰 판단으로는 **각 엔드포인트 이름·문서와 일치하는 의도된 차이**다(MCP 도구 설명이 "기본 CUSTOMER_CONTACT" 를 명시해 공표하고, REST 쪽은 일반 "이벤트 추가" 엔드포인트다). 제품 의도 판단이 필요한 동작 변경이므로 이번 과제 범위 밖 — 두 기본값을 건드리지 말고 그대로 두라.
  - `docs/api-mcp.md:96` 이 같은 세 값을 **산문으로** 네 번째로 적어 둔다. 오늘 내용은 맞으므로 손대지 말고, 코드에서 문서를 생성하려 하지도 말 것.
  - `voice.CommentEventTypes` 를 JSON 응답에 새로 노출하지 말 것(`internal/server/voice_workspace.go:21` 처럼 메타 엔드포인트에 끼워 넣는 것은 OpenAPI 계약 변경이라 별 과제다).
  - 프로덕션 코드는 바뀌지만 **동작은 바뀌지 않아야 한다** — 받아들여지는 값 집합·오류 문장·SLA 분기·대소문자 관용 모두 전과 동일. 수락/거부 집합이 달라지면 되돌릴 것.

- 차선 후보: **감사 목록 기간 필터(from/to) 의 서버 쪽만** — `internal/api/parameters.go:269-275` 가 `GET /admin/audit` 에 `q`·`channel`·`resource`·`action`·`limit` 만 공표하고 `from`/`to` 가 없어 기간으로 감사 기록을 좁힐 수 없다. 서버·OpenAPI 쪽만 먼저 하고 `AdminPages.tsx` 는 다음 회차로 미뤄 파일 수를 6개 안에 유지할 것. 날짜 파라미터는 정수 그물에도 enum 그물에도 걸리지 않으므로 계약 테스트를 함께 둬야 한다. (3순위: MCP `voice_tools.go` 가 이미 `voice.CauseEvidenceLevels` 를 참조하는 것과 같이, `voice.KnowledgeStatuses` 를 참조하지 않는 손 목록이 남아 있는지 확인 — 이번 정찰에서 미확인.)
