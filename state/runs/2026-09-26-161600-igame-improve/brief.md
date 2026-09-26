- 과제: 관리자 설정 PUT 이 `{"value": null}` 을 받아들여 설정을 조용히 초기화하는 것을 막기 (가치 3 / 위험 1 / 작업량 S)
- 왜: `putSetting`(internal/api/admin.go:106)은 `json.Valid` 와 키별 `validateSetting`(admin.go:160~) 만 통과시키는데, encoding/json 문서대로 JSON 리터럴 `null` 을 구조체에 Unmarshal 하면 오류 없이 아무 일도 하지 않으므로(문서상 계약. 이번 정찰에서 실제 실행으로 재확인하려 했으나 샌드박스가 `go run` 을 막아 **미실행** — 구현자가 Red 단계에서 실측할 것) 7개 편집 가능 설정 전부가 `null` 을 "유효" 로 통과시켜 `system_settings.value` 에 jsonb `null` 을 쓴다. 그 결과 `play_policy`·`privacy`·`service` 같은 설정이 오류 없이 제로값(=기본값)으로 되돌아가고 GET/목록은 `null` 을 내보낸다 — 설정을 지우는 쓰기는 400 으로 거부돼야 한다.
- 수용 기준:
  0) 전제 실측(먼저 할 것): 수정 전 상태에서 `{"value": null}` 이 통과해 `SELECT jsonb_typeof(value) FROM system_settings WHERE key='service'` 가 `null` 이 되는지 실제 PG 로 확인한다. 만약 pgx/DB 단계에서 이미 거부된다면(미확인) 이 과제는 성립하지 않으므로 차선 후보로 갈 것.
  1) `PUT /api/v1/admin/settings/{key}` 에 `{"value": null}` (7개 키 중 최소 `service`·`play_policy`) 을 보내면 400 `invalid_setting` 이고, DB 의 기존 `system_settings.value` 와 `updated_at` 이 변하지 않으며 감사 로그에 `setting.update` 가 남지 않는다.
  2) 정상 오브젝트 값(예: `{"value":{"timezone":"UTC"}}`)은 종전대로 200 이고 저장·`invalidateSetting`·감사가 그대로 동작한다. 배열·숫자·문자열·불리언 값도 400 `invalid_setting` 이어야 한다(현재도 대부분 키별 검사에서 걸리지만 계약을 명시적으로).
  3) 테스트는 수정 전에 실패해야 한다 — 수정 전 `{"value": null}` 이 200 이고 행이 jsonb `null` (`jsonb_typeof(value)='null'`) 로 바뀌는 것을 실패 메시지로 증명한 뒤, 수정 후 400 + 행 불변을 증명한다. 진짜 `Router()` 를 httptest 로 띄우고 실제 관리자 세션으로 호출할 것(수동 대역 금지).
- 건드릴 파일:
  - `internal/api/admin.go:putSetting` — `json.Valid` 검사 직후, `validateSetting` 호출 앞에 "값은 JSON 오브젝트여야 한다" 검사를 추가(예: `bytes.TrimSpace(wrapper.Value)` 의 첫 바이트가 `{` 인지, 또는 `json.Unmarshal` 로 `map[string]any` 대상 + nil 맵 거부). 에러 코드는 기존 `invalid_setting` 을 재사용하고 새 코드·새 문자열 계약을 만들지 말 것.
  - `internal/api/settings_put_pg_test.go` (신규) — `IGAME_TEST_DSN` 규약. 기존 헬퍼를 그대로 재사용할 것: `migratedPool(t)`(admin_pg_test.go:27), `insertTestUser(t,pool,"admin")`(:46), `insertTestSession(t,pool,userID)`(:59) → 세션 쿠키. 실제 `Router()` 를 httptest 로 띄워 `PUT /api/v1/admin/settings/service` 를 호출한다. 이 fixture 는 **기본 스키마를 공유**하고 `system_settings` 는 키가 전역이라 다른 테스트와 겹칠 수 있으므로, 대상 키의 원래 값·updated_at 을 먼저 읽어 두고 `t.Cleanup` 에서 복원할 것.
  - `docs/api.md:68` — `GET/PUT /api/v1/admin/settings/{key}` 행(또는 그 아래 설정 문단)에 "PUT 의 `value` 는 JSON 오브젝트여야 하며 `null`·배열·스칼라는 400 `invalid_setting`" 계약 한 줄.
- 검증 명령:
  - `gofmt -l .`, `go vet ./...`, `go build ./...`
  - `go test ./cmd/... ./internal/... ./migrations/...`
  - 실제 PG: `docker run -d --rm -e POSTGRES_PASSWORD=igame -p 55432:5432 postgres:17-alpine` 뒤 README 절차대로 pgcrypto 를 전용 확장 스키마에 설치하고 `make test-db DSN='postgres://postgres:igame@127.0.0.1:55432/postgres?search_path=public,ext'`
  - 새 테스트만: `go test ./internal/api/ -run TestPutSetting -race -count=3 -v`
  - `bash scripts/check-release-contract.sh`
  - `scripts/smoke-settings.sh` 를 **읽어서** 그 왕복(조회 응답을 그대로 PUT) 이 오브젝트 값만 쓰는지 확인할 것(docs/api.md:96 이 이 스크립트를 릴리즈 게이트로 언급한다). 실행에는 기동된 서버가 필요하므로 최소한 정적 확인은 필수.
- 위험과 피할 것:
  - `oidc`·`ai` 키는 `putSetting` 앞에서 `putOIDCSetting`/`putAISetting` 로 분기하며 **래퍼 없이 본문 전체를 타입 구조체로 받고 다시 마셜**하므로 애초에 `null` 을 저장할 수 없다(확인함). 계약이 다른 이 두 경로를 통합하거나 같이 고치지 말 것 — auth 인접 경로다.
  - `editableSettings` 7개(`service, approval, privacy, play_policy, api_keys, tracking, mcp`) 는 모두 `validateSetting` 에서 구조체로 읽히는 오브젝트다(확인함). 오브젝트 강제가 기존 정상 값을 깨지 않는지 기존 설정 테스트로 확인할 것.
  - `setting_not_editable`·`invalid_setting`·`invalid_json` 등 기존 에러 코드 문자열은 다른 테스트와 프런트엔드가 보고 있다 — 바꾸지 말 것.
  - migrations/*.sql(체크섬 불변), .github/workflows, auth.go/apikeys.go 는 건드리지 말 것. 프런트엔드(web/)·SDK 는 이번 범위 밖(npm 검증 불필요).
  - CI 의 `make test` 는 `IGAME_TEST_DSN` 이 없어 PG 테스트를 skip 한다 — 로컬 도커로 반드시 돌려서 결과를 보고할 것. grep 결과나 "문자열이 있다" 를 증거로 제출하지 말 것.
- 차선 후보: `startGameSession` 의 `metadata: null` 이 `client_info`(NOT NULL DEFAULT '{}') 에 jsonb `null` 로 저장되는 것 막기 — `catalog.go:206-213` 은 map 대상 Unmarshal 이라 리터럴 `null` 이 통과하고 `in.Metadata` 원문이 그대로 INSERT 된다(catalog.go:292, $4). `finishGameSession`(catalog.go:496-500)은 이미 `*map[string]any` 로 같은 입력을 거부하므로 두 경로가 같은 값을 다르게 읽는 상태다. 기존 `invalid_metadata` 문자열은 그대로 쓰고, `client_info` 를 읽는 코드는 현재 없음(확인함 — 피해는 작다).
