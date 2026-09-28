- 과제: `callerCanAdminTeam` 이 팀 관리자 판정의 DB 장애를 403 "team admin required" 로 위장하는 것을 500 으로 분리 (가치 3 / 위험 2 / 작업량 M)
- 왜: `internal/httpapi/compat_wave_handlers.go:216` 의 `callerCanAdminTeam` 이 `h.auth.HasRole(ctx, uid, "system_admin")` 과 `h.teams.IsTeamAdmin(ctx, teamID, uid)` 의 오류를 둘 다 `ok, _ :=` 로 버려 false 로 접는다. 두 서비스는 실제로 오류를 올린다(`auth/auth.go:1205-1210` 은 `SELECT roles FROM users ... .Scan` 오류를, `teams/service.go:100-107` 은 `SELECT roles FROM team_members ... .Scan` 오류를 그대로 반환) — 그래서 연결 실패·`team_members` 테이블 장애·컨텍스트 취소가 팀 관리 10개 엔드포인트에서 403 "권한 없음"(클라이언트가 재시도하지 않는 영구 응답)으로 보이고, 팀 관리자는 자기 권한이 박탈된 것처럼 느낀다. 이 저장소는 이미 같은 결정을 형제 경로에서 내렸다 — `compat_wave_handlers_final.go:190-196`(listChannelViews)은 `h.channels.IsMember` 오류를 `500 "api.context.permissions.app_error" / "failed to check channel membership"` 으로 내며, 2026-09-21 회차가 채널 경로 3곳에 같은 분리를 넣어 머지됐다. 팀 쪽만 남았다.
- 수용 기준:
  1) 헬퍼가 `(bool, error)` 를 돌려주고, **`pgx.ErrNoRows` 는 오류가 아니라 "관리자 아님"(false, nil)** 으로 접힌다 — 비회원(`team_members` 행 없음)·삭제된 사용자(`users.delete_at != 0`)는 지금과 똑같이 **403** 을 받는다(fail-closed 유지).
  2) 그 밖의 진짜 저장소 장애는 호출 지점 10곳 전부에서 `500` + 기존 오류 id `api.context.permissions.app_error` + 고정 메시지(예: `failed to check team admin permissions`)로 조기 반환되고, **감사 기록(`h.audit.LogAsync`)·쓰기·브로드캐스트보다 앞에서** 끊긴다.
  3) 정상 경로 응답은 완전 불변: team_admin/system_admin 은 200(또는 201), 권한 없는 멤버는 403 + 기존 메시지 문자열("team admin required" / "team_admin required" — 호출 지점별로 다르니 각각 그대로 둔다), 401 분기와 `denyGuestMutation` 순서도 그대로.
  4) 새 오류 id·새 상태 코드·마이그레이션·서비스 계약 변경 없음. `auth.HasRole` / `teams.IsTeamAdmin` 의 시그니처는 건드리지 않는다(다른 호출자 무영향).
  5) 테스트가 증명할 것: 수정 전 코드에서 **403 이 나오던 장애 요청이 실패로 뜨고**(RED 먼저 확인), 수정 후 500 이 되며, 같은 테스트의 비회원·비관리자 케이스는 수정 전후 모두 403 으로 통과한다. 장애 케이스에서 해당 `audit_logs` 행이 생기지 않는 것도 단언한다(스텁 경로).
- 건드릴 파일 (프로덕션 2개 + 테스트 1개):
  - `internal/httpapi/compat_wave_handlers.go:216 callerCanAdminTeam` — `(bool, error)` 로 바꾸고 `errors.Is(err, pgx.ErrNoRows)` 는 `(false, nil)` 로 흡수. **이 파일은 `errors` 와 `github.com/jackc/pgx/v5` 를 이미 import 한다**(3-22행 확인) — 새 import 불필요.
  - 같은 파일 호출 지점 5곳: `:138 updateTeamFull`(→ `patchTeam:169` 이 이 핸들러를 그대로 호출하므로 자동 적용), `:187 updateTeamPrivacy`, `:717 setTeamMemberRoles`, `:1248`(invite id 재발급), `:1292 inviteTeamMembersByEmail`.
  - `internal/httpapi/compat_wave_handlers_final.go` 호출 지점 4곳: `:93 removeTeamMember`(주의 — `uid != caller && !h.callerCanAdminTeam(...)` 이라 **자기 자신을 지우는 경우는 헬퍼를 아예 부르지 않아야 한다**. 단축평가를 풀 때 순서를 지켜라), `:120 deleteTeamImage`, `:135 uploadTeamImage`, `:1148 inviteGuestsByEmail`. 이 파일에 `errors`/`pgx` import 가 있는지 확인하고 없으면 추가.
  - 신규 `internal/httpapi/team_admin_errors_postgres_test.go`.
- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `cd server && go build ./...` / `go vet ./...` / `gofmt -l internal/httpapi`(수정·신규 파일만 clean 확인 — 기존 9파일 드리프트는 무관)
  - `MOYRO_TEST_POSTGRES_DSN='postgres://...@127.0.0.1:55433/...' go test -race -p 1 -count=1 ./internal/httpapi/` — **DSN 없이 돌리면 DB 테스트가 통째로 skip 되어 ok 만 보고 통과라고 하면 안 된다.** 로컬 컨테이너 포트는 55433(55432 는 점유 이력).
  - 전체: `MOYRO_TEST_POSTGRES_DSN=... go test -race -p 1 ./...` (수 분, `-p 1` 유지)
  - 루트에서 `bash scripts/check-source-sizes.sh` — 현재 `compat_wave_handlers.go` 58658/64000, `compat_wave_handlers_final.go` 72812/77000(헤드룸 4.2KB). 웹 변경 없으므로 webapp 은 손대지 말 것.
- 테스트 배선과 장애 주입 (2026-09-27 의 `upload_stub_limits_postgres_test.go:39-74` 를 그대로 베끼면 된다 — 실제 확인함):
  - `db := newOperationsTestDB(t)`(임의 스키마 격리) → `store.Migrate(ctx, db)` → `seedSidebarHandlerFixture(t, ctx, db)` 가 `team-main` / `user-a` 를 만든다. `UPDATE team_members SET roles='team_user team_admin' WHERE team_id='team-main' AND user_id='user-a'` 로 관리자를 만들고, 별도 `INSERT INTO users ... roles='system_user'` 로 비관리자·비회원 배우를 만든다.
  - 핸들러는 `&handlers{auth: auth.New(db, []byte("...32bytes..."), time.Hour, nil), teams: teams.New(db), audit: audit.New(db, slog.Default())}`. **roles 는 콤마가 아니라 공백 구분**(`auth.splitRoles`).
  - 라우트 패턴은 `router.go` 에서 글자 그대로 복사하고, 테스트 미들웨어가 `userIDKey` 컨텍스트에 배우를 직접 넣는다(세션 발급 불필요).
  - 대상 경로는 **`hub` 가 필요 없는 것**을 고르는 편이 싸다: `POST /teams/{teamID}/invite-guests/email`(`inviteGuestsByEmail`, `denyGuestMutation` 도 없음), `POST /teams/{teamID}/image`, `DELETE /teams/{teamID}/members/{userID}`. `updateTeamFull` 계열은 `h.hub.Broadcast` 를 타므로 넣으려면 실행 중 `ws.Hub` 를 붙여야 한다 — 기준 2 를 위해 필수는 아니다.
  - 장애 주입(격리 스키마 안에서만): **`DROP TABLE team_members CASCADE`** → `HasRole` 은 `users` 가 살아 있어 정상적으로 false 를 주고 `IsTeamAdmin` 만 깨지므로, 한 요청에서 깨지는 곳이 관리자 판정 하나뿐이 된다(결정론적). 추가로 `ALTER TABLE users DROP COLUMN roles` 는 `HasRole` 의 Scan 만 깨뜨리는데, 이 경로는 `denyGuestMutation`(→ `auth.UserByID`)이 없는 `inviteGuestsByEmail` 에서만 써라.
  - ErrNoRows 가드: `team_members` 행이 없는 배우(비회원)와 `users.delete_at != 0` 인 배우는 **403 그대로** 여야 한다 — 이 두 단언이 기준 1 을 지킨다.
- 위험과 피할 것:
  - 헬퍼 시그니처를 바꾸면 호출 지점 10곳을 모두 고쳐야 한다. `compat_wave_handlers_final.go:93` 의 단축평가(`uid != caller &&`)를 풀면서 **자기 탈퇴(`uid == caller`)가 DB 를 건드리지 않는 성질을 잃지 말 것.** 잃으면 자기 탈퇴가 team_members 장애 때 500 이 되는데, 그건 이 과제가 의도한 변화가 아니다(그래도 틀린 건 아니니 판단하되, 기준 3 의 "정상 경로 불변" 을 우선).
  - `auth.HasRole` / `teams.IsTeamAdmin` 자체를 고치지 말 것 — 다른 호출자(`requireUserParamAccess` 의 fail-closed HasRole 등)가 ErrNoRows 를 false 로 보는 것에 의존한다.
  - 보호 경로 회피: `internal/auth`, `internal/oidcauth`, `store/migrations`, `.github/workflows` 는 건드리지 않는다. 이 과제는 httpapi 안에서 끝난다.
  - 오류 메시지 문자열이 호출 지점마다 다르다("team admin required" vs "team_admin required"). **통일하지 말 것** — 기존 응답 본문 보존이 이 저장소의 최우선 관례다.
  - `grep` 결과를 증거로 제출하지 말 것. 반드시 실제 PostgreSQL + 실제 `auth`/`teams`/`audit` 서비스 + 실제 핸들러로 RED→GREEN 을 보여줄 것(손으로 만든 대역 금지).
  - 웹(`webapp/`)은 무관하다. `webapp/tsconfig.json` 의 @types/node 문제로 typecheck 가 깨져도 이 변경 탓이 아니다.
- 차선 후보: `uploadTeamImage` 에 `denyGuestMutation("api.team.image.guest_forbidden")` 추가 — 형제 `deleteTeamImage`(final.go:117)는 게스트를 막는데 `uploadTeamImage`(:134)는 `callerCanAdminTeam` 만 본다. 이번에 확인한 바: 게스트는 `users.roles` 에 `system_guest`(auth.go:1239)로 표시되고 팀 관리자 여부는 `team_members.roles`(teams/service.go:100)로 별도 판정되므로 **한 사용자가 system_guest 이면서 team_admin 일 수 있다** → 비대칭은 실재한다. 스텁이라 저장되는 것은 없지만 감사 행 + 200 이 남는다(가치 2 / 위험 2 / 작업량 S). 1순위와 같은 파일·같은 테스트 파일에서 함께 증명할 수 있다.
