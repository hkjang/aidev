- 과제: 클라이언트 JSON 이 그대로 jsonb 컬럼에 들어가는 세 경로(`metadata`·telemetry `data`·업적 `metadata`)에 오브젝트 검증 추가 (가치 3 / 위험 2 / 작업량 M)
- 왜: 09-23(`finish` 의 `result`)과 09-26(관리자 설정 PUT)에서 "클라이언트 JSON → jsonb 컬럼" 경로에 오브젝트 강제를 넣었는데, 같은 패턴의 세 경로는 아직 비-오브젝트를 저장한다 — `startGameSession` 은 리터럴 `null` 만 새어 들어가고(map 대상 Unmarshal), `submitTelemetry` 와 `unlockAchievement` 는 **검증이 전혀 없어** 배열·숫자·문자열·불리언·null 이 모두 저장된다. 저장된 값은 뒤에서 읽힌다(`realmguard_admin.go:1012` 의 `data->>$3` 집계, `content.go:405` 가 업적 `metadata` 를 응답에 그대로 반환) — 지금 막으면 세 엔드포인트의 jsonb 계약이 기존 두 경로와 같아진다.
- 수용 기준:
  1) `POST /api/v1/games/{slug}/sessions` 에 `{"metadata": null}` → 400 `invalid_metadata`, 세션 행 미생성. 배열·숫자·문자열·불리언도 400 `invalid_metadata`(기존 동작, 이번에 테스트로 고정). 본문 생략·`{}`·정상 오브젝트는 기존대로 성공하고 `jsonb_typeof(client_info)='object'`.
  2) `POST /api/v1/telemetry` 에 `data` 가 `null`·배열·숫자·문자열·불리언 → 400 `invalid_telemetry`(메시지는 "telemetry data must be a JSON object" 같은 새 문장 가능, **코드 문자열은 기존 `invalid_telemetry` 재사용**), `game_telemetry` 행 미생성. `data` 생략·`{}`·오브젝트는 기존대로 202/200.
  3) `POST /api/v1/achievements/unlock`(핸들러 content.go:443~) 에 `metadata` 가 비-오브젝트 → 400 `invalid_achievement_unlock`, `user_achievements` 행 미생성. 오브젝트·생략은 기존대로 성공.
  4) 테스트가 수정 전에는 위 세 경로에서 성공 응답 + 저장된 행의 `jsonb_typeof(...)` 가 `null`/`array` 등으로 남는 것을 Red 실패 메시지로 찍고, 수정 후 400 + 행 부재로 바뀌어야 한다(수정을 무력화하면 다시 Red 가 되는지 확인).
- 건드릴 파일 (프로덕션 2개):
  - `internal/api/catalog.go`:`startGameSession`(197~213) — `metadata` 언마셜을 `*map[string]any` 포인터 대상으로 바꾸고 nil 이면 기존 `invalid_metadata` 로 거부. 이후 `requestedRealmGuardVersionID`/`requestedDefenseVersionID` 에 넘기는 map 은 역참조한 값으로(빈 map 보장). `$4` 로 들어가는 `in.Metadata` 원문 사용 방식은 그대로 둔다.
  - `internal/api/catalog.go`:`submitTelemetry`(637~) — `len(in.Data)==0 → "{}"` 기본값과 64KiB 검사 사이(또는 직후)에 오브젝트 검사 한 개. RealmGuard/Defense 분기보다 앞이므로 모든 게임 종류를 한 곳에서 덮는다. 같은 파일에 작은 헬퍼(예: `jsonObject(raw json.RawMessage) bool` — `bytes.TrimSpace` 첫 바이트가 `{` 인지, 09-26 `putSetting` 과 동형)를 두고 세 곳에서 재사용할 것.
  - `internal/api/content.go`:업적 unlock 핸들러(459~460 의 기본값 처리 직후) — 같은 헬퍼로 검사, 기존 `invalid_achievement_unlock` 코드 재사용. DB 조회(`SELECT a.id ...`)보다 앞에 두어 거부된 요청이 아무것도 건드리지 않게 한다.
  - `internal/api/jsonb_object_pg_test.go`(신규) — 실제 `Router()` + 기존 헬퍼 `migratedPool`(admin_pg_test.go:27)·`insertTestUser`(:46)·`insertTestSession`(:59) 로 세 엔드포인트 왕복. 세션/토큰은 실제 `POST /games/{slug}/sessions` 로 만들고(session_finish_pg_test.go 가 본보기), 저장 여부는 `jsonb_typeof` 와 `count(*)` 로 확인.
  - `docs/api.md` — 30행(`POST /api/v1/games/{gameId}/sessions`), 33행(telemetry), 업적 unlock 행에 31행(`result` 계약)과 같은 문장 한 줄씩.
- 검증 명령: `gofmt -l .`(무출력), `go vet ./...`, `go build ./...`, `go test ./cmd/... ./internal/... ./migrations/...`, `make test-db DSN='postgres://…'`(api+database), `go test ./internal/api/ -run TestJSONB -race -count=3`, `bash scripts/check-release-contract.sh`. 프런트·SDK 변경이 없으므로 npm 은 범위 밖.
- 위험과 피할 것:
  - 에러 **코드** 문자열(`invalid_metadata`·`invalid_telemetry`·`invalid_achievement_unlock`)은 프런트·테스트가 본다 — 새 코드를 만들지 말고 재사용만. 새 설정 변수·새 마이그레이션 금지(migrations/*.sql 체크섬 불변).
  - 호환성: 확인한 것 — SDK 타입은 `metadata?: Record<string, unknown>`(sdk/gamehub-js/src/index.ts:16,40)과 `payload?: Record<string, unknown>`(:58)이라 1st-party 클라이언트는 항상 오브젝트를 보낸다. 기존 Go 테스트 중 비-오브젝트 `data`/`metadata` 를 보내는 것은 grep 으로 0건. **미확인**: 이미 비-오브젝트가 저장된 운영 DB 행의 복구(이번 범위 아님 — 아이디어 목록에 별도 항목으로 둠)와, `data->>key` 집계가 비-오브젝트 행에서 에러인지 NULL 인지(이 세션에서 PG 컨테이너 기동이 샌드박스에 막혀 실측 못 했다. 과제 성립에는 영향 없지만 "집계가 500 이 된다" 고 단정해 쓰지 말 것).
  - RealmGuard/Defense 버전 핀 파싱, attestation/replay 경로(`decodeDefenseTelemetry` 는 이미 422 로 안전), auth·세션 발급·apikeys 는 손대지 말 것.
  - `migratedPool` 은 기본 스키마를 공유하므로 고유 tag/slug 접두사로 격리하고, 전역 행을 바꾸면 `t.Cleanup` 으로 복원할 것. `IGAME_TEST_DSN` 없으면 skip, 있는데 연결 실패면 실패.
  - 세 경로를 한 회차에 다 넣지 못하겠으면 `startGameSession` + `submitTelemetry`(둘 다 catalog.go)만 끝내고 업적은 다음 회차로 넘겨라 — 절반 상태로 커밋하지 말 것.
- 차선 후보: `submitTelemetry` 의 `data` 오브젝트 검증만 단독으로(catalog.go 1개 + 테스트 1개 + docs 1줄, 가치 3 / 위험 1 / 작업량 S) — 읽는 쪽(admin 집계)이 있어 세 경로 중 가치가 가장 높다.
