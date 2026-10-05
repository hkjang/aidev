# 회차 노트 2026-10-05-100721-ai-admin-improve — ai-admin
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 10:07] base pinned — main@c49f39e
- [러너 10:07] autonomy release — 

## 정찰 노트
- 고른 이유: display_name varchar(190) 에 쓰는 **두 핸들러**(updateProfile·updateUser)가 같은 NUL 구멍을 공유해 "같은 컬럼을 쓰는 경로를 한쪽만 고치지 않는다" 를 한 파일 안에서 만족시킬 수 있다. 레거시 행 변경(resource_mutation.go:402-417)은 14개 리소스 상한이 이미 스키마와 맞아 기각했고, 열한 회차 차선인 chat reason 테스트는 프로덕션 결함을 안 고쳐 차선으로 남겼다.
- 확신 없는 곳: PostgreSQL 이 varchar 파라미터의 NUL 을 22021 로 거부한다는 **전제를 이번에 재현하지 못했다** — 이 환경에서 `docker run` 이 권한으로 막혔다(이미지는 있다). 구현자는 반드시 red(500)를 먼저 찍고 시작하고, 500 이 아니면 과제를 버리고 차선으로 갈 것.
- 조심할 것: 브랜치 auto/2026-10-04-1142(f66d25c 외 3)가 **같은 users.go 의 updatePreferences 를 고친 채 main 에 미머지**다. updatePreferences 와 그 브랜치 헬퍼 이름(jsonObjectHasNUL·storablePreferenceObject)은 건드리지도 같은 이름으로 만들지도 말 것. 2026-10-01 의 updateKeyScope 과제도 verify-failed 로 main 에 없다(결함 잔존, 재시도 보류).
- 라우트 정정: 개인 설정은 `/api/v1/preferences` 가 아니라 `PATCH /api/v1/profile/preferences`(server.go:141)다. 프로필은 `PATCH /api/v1/profile`(server.go:139), 사용자 수정은 `PATCH /api/v1/users/{id}`(server.go:130).
- [러너 10:13] scout done — `display_name varchar(190)`·`email varchar(320)` 에 담을 수 없는 NUL(U+0000) 을 프로필·사용자 수정 두 경로 모두에서 

## 구현 노트
- 무엇·왜: `updateProfile`(displayName·email)·`updateUser`(displayName) 세 검증 자리에 `storableInVarchar`(= NUL 없음) 판정을 더했다. `app_user.display_name varchar(190)`·`email varchar(320)` 은 NUL 을 담을 수 없어 호출자 입력 오류가 400 이 아니라 UPDATE 에서 터져 500 으로 나갔다. 오류 코드는 핸들러별 기존 코드(`profile_invalid`/`email_invalid`/`name_invalid`)를 그대로 썼다. 프로덕션 1개(`internal/server/users.go`) + 신규 통합 테스트 1개 + `docs/api.md` 두 문장.
- 확신 없는 곳·검증 못 한 것: (1) `storableInVarchar` 는 `users.go` 안에 있지만 패키지 전역 이름이다 — 같은 패키지 다른 파일에서 쓰려는 후속 변경이 생기면 위치를 다시 볼 것. 미머지 브랜치 `auto/2026-10-04-1142` 의 `jsonObjectHasNUL`·`storablePreferenceObject` 와는 이름이 겹치지 않음을 확인했고 `updatePreferences` 계열 함수는 한 줄도 건드리지 않았으나, 그 브랜치와의 실제 머지는 해 보지 않았다. (2) `email` 의 NUL 판정은 `value != ""` 가 참일 때만 돈다(기존 길이·`@` 검사와 같은 괄호 안). NUL 만 담긴 이메일은 `TrimSpace` 가 지우지 않아 `value != ""` 가 참이므로 거부되지만, 이 경우는 테스트하지 않았다. (3) `web/src`·`npm test`·`make build` 는 웹 변경이 없어 실행하지 않았다.
- 일부러 하지 않은 것: 오류 코드 통일(핸들러마다 다른 코드가 이미 계약이고 `docs/api.md` 가 적고 있다), rune→byte 길이 기준 변경, 다른 varchar 쓰기 경로(`updateRole`·`createKey`·provider)로의 확대 — 그 경로들이 실제로 500 이 되는지 확인하지 않았으므로 `ideas.json` 에 다음 회차 후보로만 적었다. 마이그레이션·VERSION·CHANGELOG·`internal/ui/dist` 는 손대지 않았다.
- 다음 역할이 조심할 것: `internal/server/profile_nul_integration_test.go` 는 `TEST_POSTGRES_DSN` 이 없으면 **조용히 SKIP** 되고, 돌 때는 `DROP SCHEMA ai_admin/aiportal CASCADE` 를 하므로 전용 폐기 DB 가 아니면 절대 돌리지 말 것. 이번 회차는 `postgres:16-alpine` 을 포트 **55531** 로 새로 띄워 썼다(컨테이너 `ai-admin-1005-pg` 가 남아 있으니 정리 가능). 검증 결과: `go test -race -count=1 ./...` 전체 통과(internal/server 148.3s), `make lint`·`go build ./...` 통과, 신규 테스트 11개 서브테스트 `-v` 로 PASS(SKIP 아님) 확인, 역검증(헬퍼를 `return true` 로 되돌리면 NUL 7건만 다시 FAIL)까지 하고 복구했다.
- [러너 10:22] brief accepted — 채택 — 결함·코드 위치(`users.go:88-113`·`:413-446`)·`varchar(190)`/`varchar(320)`·라우트(`server.go:130`/`:139`)·재사용 셋업(`profile_unk
- [러너 10:22] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인함: main 을 detached worktree 로 꺼내 신규 테스트만 복사해 돌려 **red 를 직접 재현**(NUL 7건이 :99 에서 500 profile_update_failed/user_update_failed, 회귀 4건 PASS) → HEAD 에서 11건 전부 PASS(SKIP 아님). 전용 DB 55531 로 `go test -count=1 ./...` 전체 ok, gofmt·go vet·verify-version 통과. 검증 3자리 모두 tx.Begin/Exec 앞이라 부분 쓰기 없음. 러너 verify.json 은 DSN 없이 돌아 통합이 SKIP 이었다 — 이번에 메웠다.
- 못 봄: `-race` 재실행(ledger 의 148.3s 주장은 신뢰), web(변경 없음), 미머지 브랜치 auto/2026-10-04-1142 와의 실제 머지.
- 승인이어도 남는 우려: ① NUL 거부가 `profile_invalid` 의 "1~190자" 메시지로 나가 원인을 길이로 오해하게 한다(계약은 일치, 메시지 세분화는 다음 회차). ② `oidc.go:684·741` 은 IdP claim 의 display_name·email 을 NUL 가드 없이 같은 column 에 써서 로그인 500 이 잔존 — 호출자 경로는 둘 다 막혔으므로 범위 이탈 아님. ③ 머지 후 users.go 에 NUL 헬퍼 두 개가 공존한다.
- 릴리즈 노트: 프로덕션 변경 1파일, 마이그레이션·외부 상태 없음 — revert 로 완전 복구. 정리 대기 컨테이너 `ai-admin-1005-pg`(55531).
- [러너 10:26] review approved — 리뷰 승인 (risk=low)
- [러너 10:26] pr created — https://github.com/hkjang/ai-admin/pull/39
- [러너 10:33] ci passed — 검사 2개 모두 success
- [러너 10:33] merge done — addbbf7
