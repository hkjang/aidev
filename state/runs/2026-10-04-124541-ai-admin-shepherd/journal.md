# PR 처리기 노트 2026-10-04-124541-ai-admin-shepherd — ai-admin PR #38
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-04-114232-ai-admin-improve)
# 회차 노트 2026-10-04-114232-ai-admin-improve — ai-admin
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 11:42] base pinned — main@c49f39e
- [러너 11:42] autonomy release — 

## 구현 노트
- `updatePreferences`(`internal/server/users.go`)가 `sidebarState`·`aiDefaults` 를 검증 없이 jsonb 에 넣어 `\u0000` 은 500(PostgreSQL 22P05)·비객체는 200 저장·`null`/비객체 `sidebarState` 는 저장된 화면 상태를 조용히 `{}` 로 지웠다. `locale`·`timezone` 검사와 같은 자리에서 400 `sidebar_state_invalid`·`ai_defaults_invalid` 로 거부한다. 프로덕션 파일 1개.
- 확신 없는 곳: ① `null` 을 "보내지 않았다" 로 읽는 쪽으로 정했다(같은 요청 구조체의 nullable 필드와 일치, 비우려면 `{}`). 기존 동작은 `{}` 로 비우기였으니 계약 변경이다 — `web/src` 는 antd 폼이 항상 객체를 보내므로 영향이 없다고 판단했고(`PersonalPages.tsx:72-88`·`types/api.ts:52-54` 확인) 웹 테스트는 돌리지 않았다(변경 없음). ② 이미 배포된 DB 에 `ai_defaults` 가 JSON `null`/비객체로 저장된 행이 있으면 조회 응답이 그대로 비객체를 돌려준다 — 마이그레이션으로 정리하지 않았다. ③ `\u0000` 외에 jsonb 가 거부하는 입력은 없다고 보았으나 전수로 확인한 것은 아니다.
- 일부러 하지 않은 것: `aiRaw` 는 요청 원문을 그대로 저장한다(map 재직렬화를 하면 큰 정수가 float64 를 거쳐 정밀도를 잃는다). `varchar` 쪽 NUL 결함(`displayName` 등)은 호출부가 여럿이라 이번에 묶지 않고 다음 회차 후보로 적었다. VERSION·CHANGELOG·`internal/ui/dist` 는 손대지 않았다.
- 다음 역할이 조심할 것: 새 통합 테스트는 `TEST_POSTGRES_DSN` 이 있어야 돌고 없으면 조용히 SKIP 한다(전용 폐기 `postgres:16-alpine` 포트 55521 로 검증, `DROP SCHEMA` 를 하므로 공유 DB 금지). 단위 테스트 `TestStorablePreferenceObjectAcceptsOnlyStorableObjects` 는 DB 없이 돈다. 역검증은 `jsonObjectHasNUL` 호출을 원문 substring 검사로 바꾸면 리터럴 escape 사례가 FAIL 하는 것으로 했다.
- [러너 11:52] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인한 것: 전용 폐기 `postgres:16-alpine`(새 포트 55539)로 `TEST_POSTGRES_DSN` 을 주고 새 통합 테스트 12개 서브테스트 전부 PASS(SKIP 아님) 확인. 원장의 실패 재현 출력이 이 변경이 고치는 증상과 일치하는 것도 확인. `docs/api.md` 문장·`web/src` 호출부(`PersonalPages.tsx:75-76`·`types/api.ts:52-54`)·`getPreferences` 읽기 쪽도 읽음.
- **거절 사유(수리가 가장 먼저 볼 파일: `internal/server/users.go:552` 와 `:606`)**: 검증은 디코딩된 값을, 저장은 원문 바이트를 본다. 브랜치 HEAD 를 빌드해 실제 서버로 재현 — `PATCH /api/v1/preferences {"theme":"light","aiDefaults":{"note":"\ud800"}}` → **500 `preferences_update_failed`**, 같은 요청의 `theme` 은 저장되지 않음(원문 0x80 바이트도 동일). Go 디코더가 짝 없는 surrogate·깨진 UTF-8 을 U+FFFD 로 바꿔 검증을 통과시키고, `:552` 가 원문을 jsonb 에 넣어 22P05/22021 을 맞는다. `sidebarState` 는 재직렬화되어 같은 입력에 200 — 두 필드 계약이 갈라진다.
- 수리 방향: `aiRaw` 원문 패스스루를 `json.Decoder`+`UseNumber()` 재직렬화로 바꾸면 큰 정수 정밀도를 잃지 않고 바이트가 정규화된다(구현 노트가 든 패스스루 이유가 해소됨). 또는 원문에서 `\u0000`·짝 없는 surrogate escape·비 UTF-8 을 함께 거부. 테스트에 `{"note":"\ud800"}` 케이스 추가 필수.
- 못 본 것: `make lint`·`go build`·웹 테스트·전체 통합 스위트는 돌리지 않았다(`verify.txt` 의 `internal/server 0.191s` 는 DSN 없이 돈 결과이므로 통합은 그 실행에서 SKIP — 게이트를 통합 통과로 읽지 말 것).
- 남는 우려(수리 후에도): ① `null` 의미를 '비우기'→'보내지 않음'으로 바꾼 계약 변경은 릴리즈 노트에 적을 것. ② 기존 DB 에 `ai_defaults` 가 비객체로 저장된 행은 그대로 돌려주고 재저장된다(마이그레이션 없음 — 다음 회차 후보). 보안·법무 차단 사유는 없다.
- [러너 11:57] review rejected — 리뷰 거절: internal/server/users.go:552 aiDefaults 는 검증과 저장이 서로 다른 표현을 본다. storablePreferenceObject(:606,:611)는 json.Unmarshal 로 '디코딩된' 값을 검사하

## 수리 노트
- 비평은 전부 맞았다. 전용 폐기 postgres(포트 55557)로 브랜치 HEAD 에서 재현 — `aiDefaults:{"note":"\ud800"}` 과 원문 0x80 이 500 `preferences_update_failed`, 같은 요청의 `theme` 미저장, `sidebarState` 는 같은 입력에 200(계약 갈림). 틀린 지적은 없었다. 구현 노트의 '확신 없는 곳 ③'이 정확히 그 자리였다.
- 고친 방법: `storablePreferenceObject` 가 `json.Decoder`+`UseNumber()` 로 디코딩한 객체를 함께 돌려주고(뒤에 남은 토큰도 거절), `sidebarState`·`aiDefaults` 둘 다 검증한 그 값을 재직렬화해 저장한다. 원문 패스스루 이유(큰 정수 정밀도)는 `json.Number` 로 해소 — `UseNumber()` 를 지우면 큰 정수 테스트 2곳이 FAIL 하는 것으로 역검증했다.
- 테스트: 통합에 `\ud800`·`\udc00`·0x80·sidebarState 동일 입력·큰 정수 보존 5개 서브테스트, 단위에 재직렬화 바이트 단언(기대 바이트까지 못 박음)을 추가. 통합은 Go 리터럴로 못 적는 본문을 보내려 로컬 `patchRaw` 헬퍼로 원문을 직접 쓴다.
- 확신 없는 곳: 짝 없는 surrogate 를 400 으로 거절하지 않고 U+FFFD 로 정규화하는 쪽을 택했다(sidebarState 기존 동작과 일치, jsonb 가 담을 수 있는 값이므로). 거절이 더 정직하다고 보면 두 필드를 함께 바꿔야 한다 — 중재가 판단할 지점.
- `make lint`·`go build`·`go test -race ./...` 전체(실 DB, SKIP 아님)·`npm test` 모두 PASS 출력 확인. `internal/ui/dist`·VERSION·CHANGELOG 는 손대지 않았다.
- [러너 12:06] repair done — # 수리 요약 (ff9ee06)  - 비평이 맞았다. `storablePreferenceObject` 는 디코딩된 값을 검사했는데 `:552` 는 `request.AIDefaults` 원문을 jsonb 에 넣었다. Go 디코더가 짝 

## 비평 노트 (2회차 · 머지 심사)
- 확인한 것: 전용 폐기 `postgres:16-alpine`(새 포트 55601, 검사 후 삭제)로 새 테스트 전부 PASS(SKIP 아님, 통합 17개 서브테스트 포함), `make lint`·`go build ./...` PASS. 수리가 고친 surrogate·0x80 경로는 실제로 고쳐졌고 테스트가 그 경로를 지난다(`users.go`만 main 으로 되돌리면 그 케이스가 FAIL). 못 본 것: `npm test`·웹 빌드·전체 통합 스위트는 돌리지 않았다.
- **거절 사유(수리가 가장 먼저 볼 파일: `internal/server/users.go:636` `storablePreferenceObject`, `:654` `jsonObjectHasNUL`)**: 수리가 넣은 `UseNumber()` 가 jsonb 가 담을 수 없는 지수의 숫자를 그대로 통과시킨다. HEAD 재현 — `{"theme":"light","sidebarState":{"w":1e1000000}}` → **500 `preferences_update_failed`**, 같은 요청의 `theme` 미저장. 같은 입력이 **main 에서는 200**(`objectValue` 의 float64 overflow 로 `{}` 가 되던 자리) — sidebarState 는 이 PR 이 만든 200→500 회귀이고, 하필 이 PR 이 없애려던 그 증상이다. `aiDefaults` 는 `1e1000000`·`1e-1000000` 둘 다 500(미수정). PG 확인: `'{"a":1e1000000}'::jsonb` → 22003 `value overflows numeric format`(`1e1000` 은 통과 — numeric 한계가 경계).
- 설명 불일치도 함께: 함수 이름·`storable` 반환값·doc 주석·`preference_validation_test.go:62` 의 "jsonb column이 객체로 담을 수 있는 것 뿐"·`docs/api.md:97` 의 "거부된 요청은 … 다른 설정도 변경하지 않습니다" 가 9바이트 입력으로 반증된다. 고칠 자리는 이미 전수 순회하는 `jsonObjectHasNUL` 안(`json.Number` 를 "볼 필요 없다" 고 적은 바로 그 case), 테스트는 두 필드 양쪽에 `1e1000000`·`1e-1000000` 추가.
- 남는 우려(수리 후에도): ① `null` 의미 변경('비우기'→'유지', 비우려면 `{}`)은 릴리즈 노트 필수. ② 기존 DB 의 비객체 `ai_defaults` 행 정리 마이그레이션 없음. ③ `PATCH /api/v1/preferences` 는 API 키 scope 검사가 없다(GET 은 `profile:read`, `server.go:137-141`) — 자기 `user_id` 만 쓰므로 차단 아님, main 에서도 동일.
- 보안·법무 차단 사유 없음(`blocking: []`): 새 엔드포인트·권한 확대·비밀값·외부 요청·의존성 없음, 쓰기 대상은 호출자 자신의 `user_id` 고정, 개인정보 신규 수집·전송 없음, 입력은 전부 pgx 바인딩 파라미터. risk=medium(쓰기 경로 회귀).
- [러너 12:10] review rejected — 리뷰 거절: internal/server/users.go:636 storablePreferenceObject 가 UseNumber() 로 숫자 literal 을 그대로 보존해 재직렬화하면서, jsonb 가 담을 수 없는 지수 범위의 숫�
- [러너 12:10] pr created — https://github.com/hkjang/ai-admin/pull/38

## 수리 노트
- 비평은 전부 맞았다(틀린 지적 없음). 포트 55611 전용 폐기 postgres 로 HEAD 재현 — `sidebarState`/`aiDefaults` 의 `1e1000000`·`1e-1000000` 이 500 `preferences_update_failed` 이고 같은 요청의 `theme` 미저장. main 에서 200 이던 `sidebarState` 가 이 PR 로 500 이 된 것도 그대로 확인했다.
- 고친 방법: `jsonObjectHasNUL` → `storableJSONValue`(이름·주석을 하는 일과 맞춤)에 `json.Number` case 추가, `storableJSONNumber` 가 literal 에서 최상위 유효숫자 자리·scale 을 구해 numeric 한계(앞 131072·뒤 16383)와 견준다. 한계·경계는 PG 16 에 `'{"a":<literal>}'::jsonb` 를 직접 넣어 확인한 값이다(`0.1e131072` 통과·`0e-1000000` 거부 같은 비직관 사례 포함).
- 역검증: `storableJSONNumber` 를 `return true` 로 되돌리면 새 통합 6개가 다시 500 으로 FAIL. 통합 23개 서브테스트 전부 PASS(SKIP 아님), `make lint`·`go build ./...`·`go test -race ./...`(실 DB, internal/server 135s)·`npm ci && npm test`(81) 모두 PASS 출력 확인.
- 확신 없는 곳: ① 한계를 PG 16 의 numeric 상한(131072/16383)에 맞췄다 — 상한이 다른 PG 버전이 있으면 그 쪽이 기준이 된다(현재 지원 대상은 16+). ② 지수가 int 범위를 넘는 literal 은 거절 대신 폭을 눌러 판정하는데(`min/max`), 눌러도 판정이 같다는 근거는 "mantissa 자릿수가 literal 길이를 넘지 못한다" 이고 단위 테스트로만 못 박았다.
- 남는 우려는 비평 2회차의 ①(null 의미 변경 릴리즈 노트)·②(비객체 `ai_defaults` 행 정리 마이그레이션 없음)·③(PATCH scope 검사 없음, main 동일) 그대로이고 이번 수리 범위 밖이다.

## 심사 노트
- 확인한 것(전용 폐기 postgres:16-alpine 포트 55641, 검사 후 삭제): 새 테스트 전부 PASS(SKIP 아님, 통합 23개), `make lint`·`go build ./...`·`go test -count=1 ./...`(실 DSN) 전 패키지 PASS. 역검증 — `storableJSONNumber` 를 `return true` 로 되돌리면 숫자 통합 6개가 500 으로 FAIL(파일 원복, git clean). 수리가 적은 numeric 경계 12개를 PG 16 에 직접 넣어 전부 일치 확인.
- 거절 사유: 쓰기 경로에만 `UseNumber()` 를 넣어 읽기 경로(`objectValue` users.go:759, `decodeRaw` auth_handlers.go:141)와 계약이 갈라졌다. HEAD 를 빌드한 실제 서버로 재현 — `{"sidebarState":{"w":1e1000,"keep":"yes"}}` 200 저장 뒤 `{"theme":"light"}` PATCH 가 저장된 값을 `{"w":null}` 로 파괴하고, GET 은 `aiDefaults:null`·`123456789012345680000` 을 돌려준다. docs/api.md:97 의 "조회 응답과 같은 JSON 객체"·"숫자는 보낸 자릿수를 잃지 않습니다" 가 반증된다.
- 권고 근거: 운영자 지시 "같은 값을 읽는 파서·경로가 둘 이상이면 한쪽만 고치지 말고 두 경로가 같은 입력을 같은 값으로 읽는지 end-to-end 로 확인" 에 그대로 걸린다. 다만 방향 자체의 문제는 아니고 읽기 쪽을 같이 맞추거나 받는 범위를 좁히면 되므로 `fix`(사람 결정 불필요, 마이그레이션 없음 → revert 가능).
- 못 본 것: `npm test`·web 빌드·`-race`(web·동시성 변경 없어 생략), 2MB 본문에 `1e131071` 수만 개를 담아 jsonb 전개로 부풀리는 자원 경로(머신 영향 우려로 미실행 — 검증하지 않은 우려, main 의 aiDefaults 패스스루도 동일).
- 보안·법무 차단 없음(`blocking: []`): 새 경로·권한 확대·비밀값·외부 요청·의존성·마이그레이션 없음, 쓰기 대상은 호출자 자신의 `user_id`, 입력은 전부 pgx 바인딩. 남는 우려 3건(null 의미 변경 릴리즈 노트·비객체 `ai_defaults` 행 마이그레이션·PATCH scope 미검사)은 범위 밖으로 그대로 남긴다.
