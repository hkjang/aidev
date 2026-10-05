- 과제: 남은 varchar/jsonb 쓰기 경로(역할 이름·API 키 이름·공급자 이름/모델)의 NUL(U+0000) 을 저장 전 400 으로 막기 (가치 2 / 위험 1 / 작업량 S)
- 왜: 2026-10-05 회차가 `storableInVarchar`(`internal/server/users.go:414-421`)로 `app_user.display_name`·`email` 두 경로만 닫았고, 같은 꼴의 500 이 남은 세 자리가 그대로다 — `updateRole` 의 `name`(`users.go:272-282`, `ai_admin.role.name varchar(120)`), `createKey` 의 `name`(`keys.go:111-121`, `ai_admin.api_key.name varchar(160)`), `validateProviderRequest` 의 `name`/`defaultModel`/`availableModels`(`providers.go:477-505`, `ai_provider.name varchar(160)`·`default_model varchar(240)`·`available_models jsonb`). 세 자리 모두 `strings.TrimSpace` + `utf8.RuneCountInString` 만 보고 NUL 을 통과시키므로 호출자 입력 오류가 400 이 아니라 INSERT/UPDATE 에서 터져 500 이 되고(역할은 `users.go:305` 의 UPDATE 에서 500 `role_update_failed`, 키는 `keys.go:174` 에서 500 `key_create_failed`, 공급자는 500 `provider_create_failed`/`provider_update_failed` — 세 코드 모두 소스에서 확인, 단 실제 500 재현은 구현자가 red 로 찍을 것), 고치면 세 경로가 이미 길이·빈값을 거부하는 **같은 자리에서 같은 400** 을 돌려준다.
- 수용 기준:
  1) `PATCH /api/v1/roles/{id}` 에 `{"name":"역할\u0000"}`(선행·후행·단독 NUL 각각)이 400 `name_invalid` 이고, 그때 `ai_admin.role.name`·`updated_at` 이 불변임을 SELECT 로 확인한다.
  2) `POST /api/v1/keys` 에 NUL 이 든 `name` 이 400 `key_invalid` 이고, `ai_admin.api_key` 행 수가 늘지 않는다.
  3) `POST /api/v1/ai/providers` 와 `PATCH /api/v1/ai/providers/{id}` 에서 NUL 이 든 `name`·`defaultModel`·`availableModels[i]` 가 각각 400 `provider_invalid` 이고, 공급자 행 수·`name`·`default_model`·`available_models`·`updated_at` 이 불변이다.
  4) 회귀: 여섯 글자 리터럴 `\\u0000`(백슬래시+u0000 문자열)은 계속 200/201 로 저장되고 조회에서 그대로 돌아온다 — 판정은 디코딩된 값의 rune 이지 원문 substring 이 아니다. 기존 상한 회귀(역할 120자 통과/121자 400, 키 160자 통과/161자 400, 공급자 160자·240자 경계)도 같은 테스트에서 함께 고정한다.
  5) 수정 전에 각 사례가 **실제로 500** 임을 red 로 먼저 찍고 로그에 남긴다(추정 금지 — 어느 경로가 500 이 아니면 그 사례는 과제에서 빼고 그 사실을 적을 것).
- 건드릴 파일 (프로덕션 3개):
  - `internal/server/users.go:updateRole` — 272-282 의 `if value == "" || utf8.RuneCountInString(value) > 120` 조건에 `|| !storableInVarchar(value)` 를 더한다. 기존 주석 스타일대로 "varchar 는 길이와 무관하게 NUL 을 담을 수 없다" 근거를 한 줄 남긴다. 400 코드는 기존 `name_invalid` 유지.
  - `internal/server/keys.go:createKey` — 118 의 복합 조건에 `|| !storableInVarchar(request.Name)` 를 더한다. 400 코드는 기존 `key_invalid` 유지. `storableInVarchar` 는 같은 `server` 패키지(`users.go`)에 있으므로 **옮기지 말고** 그대로 호출한다(옮기면 diff 가 커지고 `server.go` 를 건드리게 된다).
  - `internal/server/providers.go:validateProviderRequest` — `requireName` 분기(478)에 `name`, 494 의 `defaultModel`, 500-505 의 `availableModels` 루프에 각각 NUL 검사를 더한다. 이 함수 하나가 생성·수정·승인 실행 세 경로를 모두 덮는다(2026-09-28 baseUrl 회차에서 확인된 구조). `available_models` 는 varchar 가 아니라 **jsonb** 이므로 주석은 "jsonb 도 `\u0000` 을 담을 수 없다(PostgreSQL 22P05)" 로 적고 `storableInVarchar` 라는 이름이 오해를 주면 호출부에 근거 주석을 남긴다(헬퍼 이름 변경·신규 헬퍼 추가는 하지 말 것 — 이름 변경은 `users.go` 의 기존 세 호출부까지 번진다).
  - 테스트: 신규 `internal/server/varchar_nul_write_integration_test.go` 1개로 세 경로를 함께 고정하는 것을 권한다. 셋업은 `internal/server/profile_nul_integration_test.go`(2026-10-05 회차, main 에 있음) 를 그대로 베끼고 **포트만 새로** 쓴다(과거 사용: 15434·55432·55433·55439·55444·55451·55461·55471·55481·55491·55501·55511·55521·55531 → 55541 권장). 헬퍼는 `signIn`(`db_error_integration_test.go:100`)·`sessionCredentials.do`·`doJSON`(`integration_test.go`) 을 쓴다. 역할·공급자 쪽 사례 작성은 기존 `role_name_integration_test.go`·`provider_base_url_integration_test.go` 의 요청 형태(특히 `PATCH /api/v1/ai/providers/{id}` 는 `expectedUpdatedAt` 필수)를 참고한다.
  - 문서: `docs/api.md` 의 역할·키·AI 공급자 절에 각각 한 문장(“NUL(U+0000) 이 포함된 값은 400 … 으로 거부합니다.”). 이번 회차에 감사 CSV 절은 건드리지 말 것(별 과제).
- 검증 명령:
  - red/green: `TEST_POSTGRES_DSN=... go test -count=1 -run TestVarcharNULWrites ./internal/server/ -v` (SKIP 이 아니라 PASS 임을 `-v` 로 확인)
  - 전체: `TEST_POSTGRES_DSN=... go test -race -count=1 ./...` (internal/server 약 130~150초)
  - `make lint` (gofmt -l·go vet·scripts/verify-version.sh), `go build ./...`
  - 전용 폐기 DB: `docker run --rm -d -p 55541:5432 -e POSTGRES_PASSWORD=... postgres:16-alpine` — **이 환경에서 `docker run` 이 샌드박스에 막힐 수 있다(2026-10-05 정찰에서 거부됨, 구현 환경에서는 동작했다). 먼저 `docker run` 가능 여부를 확인하고, 불가하면 이미 떠 있는 테스트용 PostgreSQL 을 찾아 전용 DB 를 만들어 쓸 것.** 통합 테스트가 `DROP SCHEMA ai_admin/aiportal CASCADE` 를 하므로 운영·공유 DB 금지.
  - 웹 변경이 없으므로 `npm test`·`make build`(`internal/ui/dist` 재빌드)는 하지 않는다. VERSION·CHANGELOG 는 릴리즈 단계 전용이니 건드리지 않는다.
- 위험과 피할 것:
  - `internal/server/workflow.go:392`(승인 실행이 저장된 payload 의 키 이름을 다시 검증하는 자리)는 **건드리지 말 것** — `workflow.go` 는 위험 구역이고, `createKey` 의 검증이 핸들러 맨 앞(118)이라 payload 생성(`keys.go:162`)보다 앞선다. `grep -rn '"api_key.create"' internal/server/*.go` 로 프로덕션 생성자가 `keys.go:162` 하나뿐임을 정찰이 확인했다(나머지 hit 은 `workflow.go:375` 소비자, `keys.go:177` 감사, `integration_test.go:573` 테스트).
  - `storableInVarchar` 의 이름·구현(`strings.ContainsRune(value, 0)`)을 바꾸지 말 것. `updateProfile`/`updateUser` 의 기존 세 호출부와 `profile_nul_integration_test.go` 가 걸려 있다.
  - `updateKeyScope` 의 `name` trim·160자 결함(2026-10-01 `b5146f2` 가 verify-failed 로 main 에 없음)은 **이번 과제와 묶지 말 것** — 같은 파일(`keys.go`)이지만 실패 이력이 다른 별개 과제다.
  - 미머지 브랜치 `auto/2026-10-04-1142`(`updatePreferences` jsonb 검증, 헬퍼 `jsonObjectHasNUL`·`storablePreferenceObject`)는 main 에 **없음을 이번에 grep 으로 확인**했다. 그 헬퍼 이름을 쓰거나 `updatePreferences` 를 함께 고치지 말 것(그 브랜치와 충돌한다).
  - 보호 경로(`internal/auth`·`auth_handlers.go`·`oidc.go`·`internal/database/migrations`·`.github/workflows`·`internal/ui/dist`)는 전혀 건드리지 않는다. 마이그레이션 추가 금지.
  - 길이 상한은 바이트가 아니라 rune 이라는 저장소 관례를 유지한다. 오류 코드를 통일하려 하지 말고 각 핸들러의 기존 코드를 그대로 쓸 것.
  - 세 경로 중 하나라도 red 가 500 으로 재현되지 않으면 그 자리는 빼고 나머지만 제출할 것(파일 수를 줄이는 편이 낫다).
- 차선 후보: 비스트리밍 chat 본문이 중간에 끊겼을 때의 감사 `reason` 계약을 테스트로 고정 — `relayChatBody`(`providers.go:726-`)의 `client_write_failed`/`upstream_read_failed`/`upstream_timeout`/`request_cancelled` 네 사유가 `result=failure`·`details.reason`·`complete=false` 로 남는지를 `chat_truncation_integration_test.go` 셋업을 재사용해 못박는다(프로덕션 파일 0개, `"stream":false` 를 명시해야 공급자 `streamDefault` 에 끌려가지 않는다).
