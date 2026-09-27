# 과제서 — 2026-09-28-043236-ai-admin-improve

- 과제: AI 공급자 `baseUrl`의 `varchar(2000)` DB 상한을 요청 검증 단계에서 400으로 막기 (가치 2 / 위험 1 / 작업량 S)

- 왜: `internal/database/migrations/001_ai_admin.sql:96`의 `base_url varchar(2000)`에 대해 `validateProviderRequest`(`internal/server/providers.go:477-511`)는 `url.Parse`로 scheme/host/user/query/fragment만 보고 **길이를 전혀 보지 않는다** — `name`(160)·`defaultModel`(240)·`availableModels` 각 항목(240)·개수(200)는 모두 rune 상한이 있는데 `baseUrl`만 빠져 있다. 그래서 2001자 이상 base URL은 `createProvider`(`providers.go:144-150`)의 INSERT에서 PostgreSQL `22001 value too long`이 되어 400 `provider_invalid`가 아니라 500 `provider_create_failed`("AI 공급자를 저장하지 못했습니다.")로 나가고, 같은 값이 `updateProvider`(`providers.go:239-` UPDATE)와 승인 워크플로의 `executeApprovedOperation` 경로(`internal/server/workflow.go:677-678`이 같은 `validateProviderRequest`를 호출)에서도 같은 방식으로 늦게 터진다. 이것은 이 저장소가 v1.2.25에서 `updateRole`의 `name` varchar(120)에 대해 이미 고친 것과 **정확히 같은 유형**(요청 검증이 DB 컬럼 상한보다 느슨해 400 대신 500)이고, 고치면 운영자가 "저장 실패"가 아니라 무엇이 잘못됐는지 읽을 수 있는 400 메시지를 받는다.

- 수용 기준:
  1) `baseUrl`이 2001자 이상이면 DB에 닿기 전에 400 `provider_invalid`(`writeError`의 `code`)로 거부되고, 응답 메시지가 길이 문제임을 말한다. 공급자 행은 생성/변경되지 않는다(생성 시 목록에 추가 안 됨, 수정 시 `updatedAt`·`baseUrl` 불변).
  2) 2000자 **정확히**인 `baseUrl`은 여전히 201/200으로 성공하고 저장된 값이 요청과 같다. 상한은 바이트가 아니라 rune 기준이어야 한다(`utf8.RuneCountInString`) — 이 저장소의 관례이고, PostgreSQL `varchar(n)`도 문자 수를 센다. 다만 base URL은 host에 비ASCII가 들어가면 `url.Parse` 뒤 기존 검증을 통과하는지 확인 필요하니, rune 경계 증명은 **path 세그먼트에 한글을 넣은 URL**(예: `http://llm.internal/` + `가`×N)로 하라. 바이트 검사로 바꾸면 그 사례가 실패해야 한다(v1.2.25·v1.2.26에서 쓴 것과 같은 역검증).
  3) 기존 `baseUrl` 검증(scheme 제한, `file://`·query·fragment·userinfo 거부, 후행 `/` trim)과 다른 필드 상한은 회귀하지 않는다. `create`·`update`·승인 실행 세 경로가 같은 한도를 쓴다(= 검증을 `validateProviderRequest` 한 곳에만 추가해 세 호출 지점이 자동으로 공유).

- 건드릴 파일 (프로덕션 1개):
  - `internal/server/providers.go:477` `validateProviderRequest` — `url.Parse` 검증 **뒤/앞 어디든 DB 접근 전**에 `utf8.RuneCountInString(request.BaseURL) > 2000`이면 `errors.New("Base URL은 2000자 이하여야 합니다.")` 반환. `utf8`·`errors`는 이미 import되어 있다. **`normalizeProviderRequest`·`mergeProviderRequest`는 건드리지 말 것** — merge가 빈 `baseUrl`을 현재 값으로 채우는 동작에 의존하는 기존 테스트가 있다.
  - `internal/server/server_test.go:TestValidateProviderRequestCountsCharactersNotBytes`(:50~) — 이미 `name` 160·`defaultModel` 240의 rune 경계를 같은 방식으로 증명하는 순수 단위 테스트다. 여기에 `baseUrl` 2000자 통과 / 2001자 거부 / 한글 path 2000 rune 통과 사례를 추가하라(DB 불필요, 가장 싼 증명).
  - 통합 증명 1개 — `internal/server/role_name_integration_test.go`(v1.2.25의 같은 유형 선례)의 구조를 그대로 베껴 새 파일(예: `provider_base_url_integration_test.go`)을 만들거나, 기존 공급자 생성 셋업이 있는 `internal/server/chat_response_content_type_integration_test.go`의 로그인+공급자 생성 흐름을 재사용하라. 헬퍼는 파일 경계를 넘어 공유된다: `signIn`(`db_error_integration_test.go:100`), `doJSON`/`doJSONWithCookies`(`integration_test.go`). 실제 PostgreSQL → `db.Migrate`/`db.Seed` → `New(...).Handler()`로 `POST /api/v1/providers`(2001자 → 400, DB 행 없음), `PUT /api/v1/providers/{id}`(`expectedUpdatedAt` 필수 — `providers.go:174-181`, 2001자 → 400 + 행 불변)를 확인.
  - `docs/api.md:246` — `contextWindow`/`maxOutputTokens`/`availableModels` 상한을 적어 둔 그 문장에 `baseUrl`은 2000자 이하라는 계약을 한 구절 추가.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - 단위만: `go test -count=1 -run TestValidateProviderRequest ./internal/server/ -v`
  - 전용 폐기 PostgreSQL을 새 포트로 띄우고(과거 회차가 55432·55433·55439·55444·55451·55461·55471·15434를 썼으니 **다른 포트**를 쓸 것):
    `docker run -d --rm -p 55481:5432 -e POSTGRES_PASSWORD=pw -e POSTGRES_DB=aiadmin postgres:16-alpine`
    → `export TEST_POSTGRES_DSN='postgres://postgres:pw@127.0.0.1:55481/aiadmin?sslmode=disable'`
  - `go test -count=1 -run <새 테스트 이름> ./internal/server/ -v` — **SKIP이 아니라 PASS**인지 확인(DSN이 없으면 통합 테스트는 조용히 SKIP된다).
  - 수정 전 FAIL → 수정 후 PASS를 반드시 먼저 보여라(red→green). 그 뒤 `go test -race -count=1 ./...`(internal/server 80~120초), `make lint`(gofmt·go vet·verify-version — VERSION 1.2.29와 일관), `go build ./...`.
  - 웹은 변경하지 않으므로 `npm test`·`npm run build`·`internal/ui/dist` 재빌드는 불필요.

- 위험과 피할 것:
  - `web/src/pages`의 공급자 폼에 `maxLength`를 같이 넣지 말 것 — 웹을 고치면 커밋된 `internal/ui/dist` 재빌드가 따라와 변경 범위가 두 배가 된다(과거 회차의 반복 지적).
  - `workflow.go`는 위험 구역이다. `validateProviderRequest`를 호출하는 `workflow.go:677-678`은 **읽지만 수정하지 말 것**. `tx.Begin` 전후 순서·`errProviderStale` 분기에 손대지 말 것.
  - `internal/auth`·`oidc.go`·`internal/database/migrations`·`.github/workflows`는 이번 과제에서 열 필요가 없다. 마이그레이션의 `varchar(2000)`을 바꾸는 방향은 **금지**(마이그레이션 추가 = 위험 구역).
  - VERSION·CHANGELOG는 릴리즈 단계 전용이니 건드리지 말 것.
  - 2026-09-25의 교훈: 과제서가 "방법"을 못 박은 곳이 수용 기준과 어긋나면 **수용 기준을 따르고 방법을 바꾼 뒤 그 사실을 적어라**. 이번 과제에서 그럴 만한 지점은 기준 2의 "한글 path로 rune 경계 증명" — 만약 기존 `url.Parse` 검증이 비ASCII path URL을 거부한다면(미확인), percent-encoding 없는 ASCII 긴 path로 2000/2001 경계만 증명하고 rune 여부는 `TestValidateProviderRequestCountsCharactersNotBytes`의 순수 단위 테스트에서만 증명하라.
  - 미확인: 2001자 base URL이 실제로 PostgreSQL `22001`을 일으켜 500이 나가는 것을 이 세션에서 **DB로 재현하지는 않았다**(코드·스키마 정적 확인만). 구현자는 먼저 그 500을 실제로 재현해 red를 확보하라. 재현되지 않으면(예: pgx가 다른 오류로 바꾸거나 이미 어딘가에서 막고 있다면) 차선 후보로 넘어가라.

- 차선 후보: **비스트리밍 chat 본문이 중간에 끊겼을 때의 감사 계약을 테스트로 고정** (가치 2 / 위험 1 / 작업량 S — 프로덕션 파일 0개, 테스트만). `relayChatBody`(`providers.go:721-751`)는 stream 플래그와 무관하게 `upstream_read_failed`/`upstream_timeout`/`request_cancelled`를 구분해 돌려주고 `chatCompletions:695-703`이 `panic(http.ErrAbortHandler)`로 응답을 끊는데, 저장소의 truncation 테스트(`chat_truncation_integration_test.go`)는 **스트리밍 경로만** 덮고 v1.2.29가 추가한 비스트리밍 테스트는 정상 본문만 덮는다. 즉 비스트리밍에서 upstream이 `Content-Length`보다 적게 쓰고 끊을 때 감사 `result=failure`·`details.reason=upstream_read_failed`·`complete=false`이고 caller가 완전한 JSON을 받지 못하는지는 미검증이다. `chat_truncation_integration_test.go`의 셋업을 그대로 쓰고 요청에 `"stream": false`를 **명시**하라(생략하면 공급자 `streamDefault`와 `ai.stream_default` 설정이 경로를 결정한다).
