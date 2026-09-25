# 과제서 (2026-09-25, base main@59f6512)

- 과제: 리마인더 생성 두 경로가 DB 장애를 404/403 으로 위장하는 것을 500 으로 분리 (가치 3 / 위험 1 / 작업량 S)

- 왜: `POST /api/v4/posts/{postID}/remind_me`(handlers.go:2164 `createPostReminder`)와 그 공식 모양 별칭 `POST /api/v4/users/{userID}/posts/{postID}/reminder`(compat_wave_handlers.go:1008 `createUserPostReminder`) 가 모두 `p, err := h.posts.Get(...); if err != nil || p == nil { 404 }` 로 되어 있어 연결 실패·테이블 장애·컨텍스트 취소까지 "그런 글은 없다"(404 `api.reminder.create.not_found`)로 답한다. `posts.Service.Get`(internal/posts/service.go:634)은 `scanPost` 가 row.Scan 오류를 그대로 올리므로 없는 글은 `pgx.ErrNoRows` 로 이미 구분 가능하다. 추가로 별칭 쪽만 `isMember, _ := h.channels.IsMember(...)`(compat_wave_handlers.go:1029)로 오류를 버려 DB 장애를 403 "not a channel member" 로 낸다 — 같은 리소스의 본 경로(handlers.go:2182)는 이미 500 `api.reminder.create.member_check` 로 가르고 있어, **같은 값을 읽는 두 경로가 같은 입력에 다르게 답한다**. 고치면 클라이언트가 장애를 재시도 가능한 5xx 로 보고, 두 경로의 계약이 일치한다.

- 수용 기준:
  1) 없는(또는 이미 지워진) postID 로 두 경로를 부르면 **지금과 똑같이** 404 + 본문 id `api.reminder.create.not_found` + 메시지 `post not found`(상태 코드·id·메시지 모두 불변).
  2) `posts` 테이블을 쓸 수 없는 상태(격리 스키마에서 `DROP TABLE posts CASCADE`)에서 두 경로가 500 을 내고 404 를 내지 않는다.
  3) 별칭 경로에서 `channel_members` 를 쓸 수 없는 상태에서 403 이 아니라 500 이 나오고, 본 경로와 같은 오류 id (`api.reminder.create.member_check`) 를 쓴다. 진짜 비멤버는 여전히 403 `api.reminder.create.forbidden` / `not a channel member`.
  4) 정상 경로는 그대로 201 + `reminder_created` WS 이벤트(대상 = 리마인더 소유자)이며, 400(`invalid_body`/`past`)·401·403(admin required) 분기와 `/api/v4` 경로·JSON 모양은 변하지 않는다.
  5) 테스트는 **수정 전 코드에서 실제로 실패**해야 한다(장애 주입 시 404/403 이 나오는 것을 먼저 확인하고 그 사실을 PR 본문에 적을 것). 실제 PostgreSQL·실제 `posts`/`channels`/`reminders` 서비스·실제 핸들러로 증명하고 손으로 만든 대역은 쓰지 말 것.

- 건드릴 파일:
  - `server/internal/httpapi/handlers.go:2164` `createPostReminder` — `h.posts.Get` 오류 분기를 `errors.Is(err, pgx.ErrNoRows) || p == nil` → 404(현행 id·메시지 그대로), 그 외 `err != nil` → 500 `api.reminder.create.app_error`(또는 새로 만들지 말고 기존 id 중 하나를 골라 두 핸들러가 같은 것을 쓰게 할 것). IsMember 분기는 이미 맞으므로 건드리지 말 것.
  - `server/internal/httpapi/compat_wave_handlers.go:1008` `createUserPostReminder` — 위와 같은 `posts.Get` 분기 + `isMember, err := h.channels.IsMember(...)` 로 바꾸고 `err != nil` 이면 500 `api.reminder.create.member_check`(본 경로와 동일 id·형태). `requireUserParamAccess` 는 건드리지 말 것.
  - 신규 `server/internal/httpapi/reminder_errors_postgres_test.go` — 선례: `channel_membership_errors_postgres_test.go`(`newOperationsTestDB(t)` + `store.Migrate` + `seedSidebarHandlerFixture` + `invokeChannelScopeHandler` + `assertMembershipAPIError`)와 `preferences_errors_postgres_test.go`(DROP TABLE 후 500 확인). 핸들러는 `&handlers{posts: …, channels: …, reminders: …, hub: …}` 로 조립하고, 별칭은 actor == target 으로 부르면 `requireUserParamAccess` 가 `h.auth` 없이도 통과한다(handlers.go:236). 필요한 필드 이름은 `handlers` 구조체에서 직접 확인할 것(미확인).

- 검증 명령 (server/ 에서):
  - `go vet ./...`, `go build ./...`, `gofmt -l` (수정·신규 파일)
  - 실제 DB 필수: `MOYRO_TEST_POSTGRES_DSN` 없으면 DB 테스트가 **통째로 skip** 되므로 ok 만 보고 통과라고 하지 말 것. 로컬 `postgres:16-alpine` 을 포트 **55433** 으로 띄울 것(55432 는 점유된 적 있음).
  - `MOYRO_TEST_POSTGRES_DSN=... go test -race -p 1 -count=1 ./internal/httpapi/`
  - 마지막에 `MOYRO_TEST_POSTGRES_DSN=... go test -race -p 1 ./...` (수 분, `-p 1` 유지)
  - 루트에서 `bash scripts/check-source-sizes.sh` (compat_wave_handlers.go 가 상한에 걸리지 않는지)
  - 웹 변경 없음 → webapp 빌드 불필요. (webapp typecheck 가 `@types/node` 때문에 useDraft.test.tsx 에서 TS2345 로 깨지는 알려진 함정이 있으니 Go 전용 변경에서 이를 자기 탓으로 돌리지 말 것.)

- 위험과 피할 것:
  - **새 오류 id 를 만들지 말 것.** 404 본문 id `api.reminder.create.not_found` 와 403 `api.reminder.create.forbidden` 은 그대로 두고 상태 코드만 가른다(선례 ded7e4e, 761c3ae).
  - `reminders.Service.Create`·`ClaimDue`·워커·lease(마이그레이션 000018)는 이번 범위 밖. 마이그레이션 추가 금지.
  - auth/session/oidcauth/store/migrations/.github/workflows 는 건드리지 말 것.
  - `DROP TABLE` 장애 주입은 `newOperationsTestDB` 가 만든 격리 스키마 안에서만 할 것.
  - 같은 파일의 다른 `ok, _ := IsMember` (late.go 북마크 4곳 등)까지 확대하지 말 것 — 범위 부풀리기로 반려된다.
  - `posts.Get` 의 서비스 계약(리턴 타입·nil 의미)은 바꾸지 말 것. 핸들러에서만 분기한다(`pluginhost` 등 다른 호출자 영향 회피).

- 공수 산정 (bottom-up, 근거 명시):
  - 핸들러 2곳 분기 수정 ~5분 · 신규 Postgres 회귀 테스트 작성(선례 파일 복제 + 픽스처) ~20분 · 수정 전 실패 확인(구 코드로 되돌려 1회 실행) ~5분 · `go vet`/패키지 테스트/전체 `-race -p 1 ./...` ~10분 = **약 40분, 한 세션(45분) 안**.
  - 범위에 포함: 위 "건드릴 파일" 3개, 실제 DB 회귀 테스트, 검증 명령 전부. 제외: 마이그레이션, 서비스 계약 변경, 웹, 문서, 다른 `ok, _ :=` 호출부.
  - 신뢰도: 10회 중 8회는 30~50분. 상한 리스크는 전체 `go test -race -p 1 ./...` 가 수 분 걸리는 것과 로컬 Postgres 컨테이너 기동(포트 55433)이며, 둘 다 이전 회차에서 실측된 알려진 변동이다.
  - 컨틴전시(이 과제 안의 알려진 변동) ~10분: 리마인더용 픽스처(users/channels/channel_members/posts)가 `seedSidebarHandlerFixture` 로 부족해 직접 INSERT 를 추가해야 할 경우. 그 이상으로 번지면(예: 픽스처가 다른 테스트와 충돌) **범위를 늘리지 말고 차선 후보로 전환**할 것 — 태스크마다 여유를 덧대 두 번 세지 말 것.

- 차선 후보: **팀 이미지 업로드가 10MB 초과 본문에도 200 + 업로드 감사 로그를 남기는 것 정리** (`compat_wave_handlers_final.go:143` 부근에서 `io.Copy` 오류를 버림, 가치 2 / 위험 1 / 작업량 S). 고른다면 "413 을 내고 감사 로그를 남기지 않는다"까지 테스트로 증명할 것 — 스텁이라 상태 코드만 바꾸면 '효과 없는 정리' 로 몰린다.
