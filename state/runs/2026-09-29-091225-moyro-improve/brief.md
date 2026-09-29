- 과제: 팀 이미지 업로드에 형제 DELETE와 동일한 게스트 차단 적용 (가치 2 / 위험 2 / 작업량 S)
- 왜: `uploadTeamImage`는 팀 관리자 여부만 확인하여 `system_guest`이면서 `team_admin`인 사용자가 200과 `team.image.upload` 감사 기록을 만들 수 있는 반면, `deleteTeamImage`는 같은 사용자를 차단한다. 기존 게스트 가드를 업로드에도 적용하면 팀 이미지 변경의 권한 계약이 일치하고 제한된 게스트의 성공 응답·감사 기록을 막는다.
- 수용 기준:
  1) `POST /api/v4/teams/{teamID}/image`의 게스트+팀관리자는 403, id `api.team.image.guest_forbidden`, 메시지 `guest access is restricted to invited channels`; 같은 배우의 DELETE도 동일하다. 게스트의 본문은 읽지 않고 `team.image.upload` 감사 행을 남기지 않는다.
  2) 일반 팀관리자와 시스템관리자는 상한 이내 200 `{"status":"OK"}`와 올바른 target 감사 기록을 유지한다. 일반 비관리자·비회원은 기존 403 권한 오류, 일반 팀관리자의 10MiB 초과는 기존 413을 유지한다.
  3) 실제 PostgreSQL 격리 스키마 + 실제 auth/teams/audit 서비스로 게스트와 팀관리자 역할이 겹치는 fixture를 구성한다. 변경 전 게스트 POST 200으로 RED를 확인하고, 변경 후 403과 감사 미기록을 증명한다. 본문 미소비는 읽기 횟수를 세는 요청 Body로 확인 가능하며 서비스 대역은 필요 없다.
  4) 기존 `TestTeamAdminCheckFaultIsNotForbidden`의 upload 항목에 `guestGated: true`를 반영한다. 그 테스트가 해당 배우를 건너뛰므로 신규 테스트에서 삭제된 사용자 POST가 선행 가드의 401 `api.team.image.guest_forbidden` / `active user session required`임을 명시적으로 단언한다. 살아 있는 사용자에 대한 `DROP TABLE team_members` 장애는 여전히 500 `api.context.permissions.app_error`이어야 한다.
  5) `TestUploadStubsRefuseOversizedBodies`, `TestTeamAdminCheckFaultIsNotForbidden`, 신규 게스트 회귀가 모두 실제 DB에서 실행되어 통과한다. 감사 미기록은 즉시 SELECT 한 번으로 끝내지 말고 아래 기존 관찰 방식을 따른다.
- 건드릴 파일:
  - `server/internal/httpapi/compat_wave_handlers_final.go:uploadTeamImage` — 함수 첫머리에 형제 `deleteTeamImage`와 같은 `denyGuestMutation(w, r, "api.team.image.guest_forbidden")` 및 return. 기존 requireTeamAdmin, drainCappedBody, 감사 순서·상한을 보존한다. 프로덕션 파일은 이 1개만.
  - `server/internal/httpapi/team_admin_errors_postgres_test.go:TestTeamAdminCheckFaultIsNotForbidden` — upload route의 guestGated 표시와 관련 설명만 갱신. 다른 경로·장애 단언을 약화시키지 않는다.
  - 신규 `server/internal/httpapi/team_image_guest_postgres_test.go:TestTeamImageGuestMutationPostgres` — 위 권한 교집합/본문/감사/삭제 사용자 회귀.
- 검증 명령:
  - server/에서 유효한 `MOYRO_TEST_POSTGRES_DSN`을 설정한 뒤 `go test -count=1 -v ./internal/httpapi -run 'TestTeamImageGuestMutationPostgres|TestTeamAdminCheckFaultIsNotForbidden|TestUploadStubsRefuseOversizedBodies'`.
  - 이어서 server/에서 `go test -race -p 1 -count=1 ./internal/httpapi`, `go build ./...`, `go vet ./...`.
  - 루트에서 `bash scripts/check-source-sizes.sh`; 바꾼 Go 파일만 `gofmt -l`로 확인.
  - 정찰 실행 결과: DSN 없는 `go test -count=1 ./internal/httpapi` 통과(0.167초, DB 테스트 skip). 기존 컨테이너 `moyro-pg-improve`의 접속 정보를 환경변수로 전달하고 55433 포트의 실제 DB에서 `go test -count=1 -v ./internal/httpapi -run 'TestTeamAdminCheckFaultIsNotForbidden|TestUploadStubsRefuseOversizedBodies'` 통과(8.748초, skip 없음). 소스 크기 검사 통과(final.go 72662/77000 bytes). race/build/vet/전체 패키지/신규 RED는 정찰에서 실행하지 않았다.
- 위험과 피할 것: auth·session·guest_access.go 자체, migrations, workflows, 실제 이미지 저장, 다른 초대 경로를 함께 고치지 않는다. 이 변경은 기존 가드 계약을 따르므로 삭제 사용자 403→401이 의도된 부수 효과이며, UserByID DB 오류도 현재 가드가 401로 내는 문제는 별도 후보로 남긴다. 이를 이번에 500으로 바꾸면 넓은 호출 범위가 섞인다. 역할 문자열은 공백 구분이다. 게스트+팀관리자 상태는 서로 다른 roles 컬럼으로 표현 가능함을 확인했지만 정상 UI에서 생성되는 경로와 인증 미들웨어를 포함한 신규 HTTP 재현은 미확인이다. 실제 이미지 바이트가 저장되는 보안 사고라고 과장하지 말 것(현재는 스텁).
- 차선 후보: customprofile.PatchUserValues 빈 맵 no-op / null DELETE 계약 고정 — 첫 후보가 이미 해결되었거나 유효하지 않을 때만 채택. `server/internal/customprofile/service.go:PatchUserValues`는 빈 맵이면 DB 접근 전 nil을 반환하고 null은 해당 사용자/필드 행을 삭제한다. 신규 서비스 테스트에서 빈 맵 no-op, 저장 후 null 삭제, 타 사용자 값 유지 확인; `go test -count=1 ./internal/customprofile` 사용, DB가 필요한 경우 격리 스키마를 마련한다.

구현 배선과 작업량:
- 읽은 선례: `native_operations_postgres_test.go:newOperationsTestDB`는 DSN 미설정 시 skip, 임의 search_path 스키마 생성 및 cleanup을 소유한다. `team_admin_errors_postgres_test.go`의 `seedSidebarHandlerFixture`, `auth.New`, `teams.New`, `audit.New`, `serveTeamAdminRequest`를 재사용한다. 새 테스트에는 팀 이미지에 불필요한 hub 배선을 추가하지 않아도 된다.
- `users.roles='system_guest'`, 해당 `team_members.roles='team_user team_admin'`로 배우를 구성한다. `auth.User.IsGuest`는 users.roles, `teams.Service.IsTeamAdmin`은 team_members.roles를 별도로 읽는다. chi 패턴은 읽은 `router.go:903-904`의 DELETE/POST `/teams/{teamID}/image`를 사용한다.
- `audit.Service.LogAsync`는 goroutine 안에서 3초 timeout을 쓴다. `upload_stub_limits_postgres_test.go:waitForAuditRow`로 정상 감사 행이 관찰되는 양성 대조를 두고, `team_admin_errors_postgres_test.go`처럼 거절 배우의 감사 행이 약 3.2초 관찰 동안 0임을 확인한다. 기존 관찰 방식도 스케줄링을 수학적으로 보장하는 barrier는 아니므로 403 및 본문 미소비 단언과 함께 사용한다.
- 예상 30분 + 예비 15분: fixture/RED 10분, 가드 및 기존 테스트 정합 5분, DB 회귀·build/vet 15분, 환경/감사 관찰 문제에 예비 15분. 장애 주입 수단이 여러 회차 미해결인 patchPost 후보는 이번에 재선택하지 않는다.
- 스킬 제한: 요청된 pmo:estimating-and-contingency, technology:implementation-planning, technology:solution-exploration은 현재 callable Skill 도구·로컬 스킬 검색·리소스 카탈로그에서 발견되지 않았다. 로드 성공이나 반환 형식 준수는 주장하지 않으며, 사용자 지정 과제서 형식과 독립적인 추정/차선 계획으로 작성했다.
