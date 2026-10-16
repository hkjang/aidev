- 과제: 레거시 리소스 변경(`/api/v1/admin/resources/...`)의 문자열 필드·행 식별자에 든 NUL(U+0000)을 트랜잭션 시작 전에 400 `resource_invalid` 로 거부 (가치 3 / 위험 2 / 작업량 S)
- 왜: `validateLegacyMutation`(internal/server/resource_mutation.go:333)은 rune 길이·필수·`Y|N`·UUID·정수만 보고 **NUL 을 전혀 보지 않는다**(한도 루프 :418-441 은 `utf8.RuneCountInString(text) > limit` 만 검사). 그래서 `{"word":"a\u0000b"}` 같은 입력이 `tx.Begin`(:108)을 지나 `applyLegacyMutation`(:1259)의 varchar/text INSERT·UPDATE 로 내려가고, PostgreSQL 이 NUL 을 담을 수 없어 호출자 입력 오류가 400 이 아니라 500 `resource_update_failed`(:196)로 나간다. 승인(workflow `legacy_data_change`)이 켜진 경우에는 같은 값이 `createApprovalRecordIn`(:167)의 jsonb payload 로 들어가 500 `approval_create_failed`(:169)가 된다. 같은 패키지에 이미 쓰이는 `storableInVarchar`(internal/server/users.go:421)를 같은 자리에서 호출하면, 이 저장소가 users·roles·keys·providers 에서 이미 닫은 계약(입력 NUL = 400)을 레거시 변경 경로에도 똑같이 적용할 수 있다.
- 수용 기준:
  1) 수정 전 red 를 실제 DB·실제 HTTP 로 관찰해 기록한다 — 최소 ① `POST /api/v1/admin/resources/banned-words` 의 `word` 에 NUL, ② `PATCH .../notices/{id}` 의 `title` 에 NUL 이 현재 **500** 임을 로그로 남긴다(기대 코드 `resource_update_failed`. 오류 상세는 22021 `invalid byte sequence for encoding "UTF8": 0x00` 또는 jsonb 의 22P05 중 하나 — **번호는 미확인이므로 red 로그에서 확인해 적을 것**).
  2) 수정 후 같은 요청이 **400 + `"code":"resource_invalid"`** 이고, 레거시 테이블 행 수가 변하지 않으며(create 는 미삽입, patch 는 값 불변) `aiportal.activity_log`/감사에 성공 기록이 남지 않는다.
  3) NUL 이 없는 같은 요청은 그대로 성공한다(create 201, patch 200) — 기존 `exerciseExpandedLegacyHTTP` 의 다른 단정이 하나도 깨지지 않아야 한다.
  4) 기존 한도 계약이 유지된다: 한도 이하 통과 / 한도+1 은 지금과 같은 400 메시지(`… 값은 N자 이하여야 합니다.`). NUL 거부 메시지는 길이 메시지와 구분되는 새 한국어 문장.
  5) 리터럴 역슬래시 6글자 `\u0000`(실제 0x00 아님)은 **거부되지 않고** 저장·round-trip 된다.
  6) 역검증: 추가한 검사만 `false &&` 로 중성화하면 ①② 서브테스트가 수정 전과 같은 500 으로 다시 실패하고, 나머지 레거시 단정은 그대로 통과한다.
- 건드릴 파일 (프로덕션 1개):
  - `internal/server/resource_mutation.go:validateLegacyMutation` — ① 한도 루프(:418-441) 안에서 문자열 캐스팅 성공 후 `!storableInVarchar(text)` 이면 400 으로 반환(길이 검사와 같은 자리, `tx.Begin` 보다 앞). `legacy-admins` 의 `menu_info` 는 :427-433 에서 정규화된 문자열이 `text` 에 담기므로 같은 검사에 자동으로 걸린다. ② `action != "create"` 의 rowID switch(:344-361)에서 문자열 키를 쓰는 `policies`·`app-defaults`·`menus` 의 `rowID` 에도 같은 검사를 추가(숫자 rowID 는 `ParseInt` 가 이미 막는다). **주의: URL 경로의 `%00` 이 chi 핸들러까지 실제로 도달하는지는 미확인** — 구현자가 red 로 확인하고, 라우터가 먼저 404/400 으로 막으면 ②는 빼고 ①만 남길 것.
  - `internal/server/legacy_http_integration_test.go:exerciseExpandedLegacyHTTP` — **기존 fixture 를 재사용**하고 새 통합 테스트 파일·새 스키마 재구축을 만들지 말 것(이 패키지는 이미 26개 파일이 각각 DROP+Migrate+Seed 하며 약 155초다). 기존 헬퍼 `legacyHTTPCreateExpect(…, 400)`(:394)·`legacyHTTPPatch(…, 400)`(:416)를 쓰고, 오류 코드까지 보려면 `chat_request_json_integration_test.go` 의 `assertRejected` 방식처럼 응답 본문의 `error.code` 를 확인하는 작은 테스트 전용 헬퍼를 이 파일에 둘 것.
  - `docs/api.md` — :366-369 의 리소스 변경 표 아래에 "모든 문자열 필드와 문자열 행 식별자는 NUL(U+0000) 을 담을 수 없고 포함되면 400 `resource_invalid`" 한 문장. **새 `vX.Y.Z` 토큰을 넣지 말 것**(`scripts/verify-version.sh` 의 expect_current_v_tokens 가 깨진다). VERSION/CHANGELOG 는 건드리지 않는다.
- 검증 명령 (이 저장소에서 실제로 도는 것):
  1. 전용 폐기 DB: `docker run -d --rm --name ai-admin-legacy-nul-pg -e POSTGRES_PASSWORD=postgres -p 127.0.0.1:55621:5432 postgres:16-alpine` → `export TEST_POSTGRES_DSN='postgres://postgres:postgres@127.0.0.1:55621/postgres?sslmode=disable'` (공유·운영 DB 금지. 통합 테스트가 `ai_admin`·`aiportal` 스키마를 DROP CASCADE 한다. 끝나면 컨테이너 제거)
  2. red/green: `go test -race -count=1 -run 'TestEndToEndAPI$' -v ./internal/server` — 확인함: `internal/server/integration_test.go` 의 유일한 테스트 함수가 `TestEndToEndAPI`(:30)이고 `exerciseExpandedLegacyHTTP` 호출(:456)이 그 안에 있다. `-v` 로 SKIP 아님을 확인(DSN 없으면 SKIP 된다).
  3. 전체: `go test -race -count=1 -v ./...`(internal/server 약 155초, 전체 2분 30초 내외. `TEST_KEYCLOAK_ISSUER` 없으면 OIDC E2E 1건 SKIP — 정상)
  4. `make lint`(gofmt·go vet·verify-version), `go build ./...`, `git diff --check`
  5. `make ci-test` 는 **이 브랜치에 없다**(미머지 PR #46 전용) — 쓰지 말 것.
- 위험과 피할 것:
  - **CI 의 lint+test 동시 실패를 쫓지 말 것.** 2026-10-16 정찰 3회가 이미 lint 세 명령·`go test -race`·콜드 모듈 캐시·web npm 3단계·`docker build` 를 전부 로컬에서 통과시켜 in-repo 재현 후보를 소진했고, 운영자 메모에 **"2026-10-15 이후의 fast lint+test 실패는 Actions 빌링 차단이며 코드 문제가 아니다"** 로 기록돼 있다(이번 회차에 재확인하지 않음 — 미확인). `.github/workflows/**`·`go.mod`·`go.sum` 을 한 줄도 바꾸지 말 것.
  - **미머지 브랜치와 겹치는 자리 금지**: `workflow.go`(PR #44·#45 가 같은 검증 지점을 건드린 상태), `users.go:updatePreferences`(f66d525/auto/2026-10-04-1142), `keys.go:updateKeyScope`(b5146f2, verify-failed), `Makefile`(PR #46). 이번 과제는 `resource_mutation.go` 라서 네 곳 모두와 겹치지 않는다.
  - `storableInVarchar` 를 복제하거나 이름을 바꾸지 말 것. `jsonObjectHasNUL`·`storablePreferenceObject` 같은 미머지 브랜치의 이름을 새로 만들지 말 것.
  - 길이 검사·`TrimSpace` 계약·오류 메시지 문구를 "통일" 하려 들지 말 것. 추가는 NUL 거부 한 가지뿐이고, 통과하던 입력이 새로 거부되면 회귀다.
  - `applyLegacyMutation` 의 SQL·`rejectTruncatedLegacyFields`·`validateLegacyReferences`·동의어 런타임 동기화(`postLegacySynonymGroup`)·승인 실행 경로는 건드리지 말 것.
  - 감사 details 에 요청 원문·NUL 포함 값을 넣지 말 것(식별자만).
- 차선 후보: **MCP `create_approval_request` 제목의 NUL 을 `-32602` 로 거부** (가치 2 / 위험 1 / S) — REST 승인 경로 수정은 미머지 PR #44 에만 있고 `internal/server/mcp.go` 는 trim+rune 300 만 본다(프로필 기록, 이번 회차 mcp.go 미열람 — 구현자가 먼저 열어 확인). `mcp.go` 1파일 + 기존 MCP 통합 테스트 재사용. 1순위가 성립하지 않을 때(예: 레거시 NUL 이 이미 400 으로 떨어져 red 가 안 나올 때)만 고를 것.
