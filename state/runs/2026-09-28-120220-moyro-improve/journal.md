# 회차 노트 2026-09-28-120220-moyro-improve — moyro
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 12:02] base pinned — main@89c6d67
- [러너 12:02] autonomy release — 

## 정찰 노트
- 고른 이유: 이 저장소에서 5회 연속 채택·머지된 유형(같은 리소스의 형제 경로가 같은 장애에 다른 답을 하는 것)의 팀 쪽 마지막 덩어리다. `listChannelViews` 는 이미 IsMember 오류를 500 으로 내는데 팀 관리자 판정 10곳은 전부 403 이다. 결정적으로 **장애 주입이 결정론적**이다 — `DROP TABLE team_members CASCADE` 면 `users` 가 살아 있어 `HasRole` 만 정상 동작하므로 한 요청에서 깨지는 곳이 관리자 판정 하나뿐이 된다.
- 제친 후보: patchPost 되읽기(5회차째 주입 수단 없음 — 또 태우지 않는다), denyGuestMutation 의 401 위장(401→500 은 클라이언트 재시도 동작을 바꿔 위험 3), 북마크 IsMember(관리자 예외 설계 선행), MaxBytesReader 나머지(경로별 계약이 달라 일괄 금지).
- 추측으로 적은 것(미확인): `compat_wave_handlers_final.go` 의 `errors`/`pgx` import 유무를 확인하지 않았다(compat_wave_handlers.go 는 둘 다 있음을 확인). 테스트를 이 회차에 실제로 돌려보지 않았다(DSN 없는 환경 + 예산) — RED 확인은 구현자 몫이다. `seedSidebarHandlerFixture` 가 `team-main`/`user-a` 를 만드는 것은 upload_stub_limits_postgres_test.go:46-55 의 사용 방식에서 읽은 것이고 함수 본문은 열지 않았다.
- 구현자가 조심할 것: `final.go:93` 의 단축평가(`uid != caller && !h.callerCanAdminTeam(...)`) — 자기 탈퇴가 DB 를 건드리지 않는 성질을 잃지 말 것. 오류 메시지가 호출 지점마다 "team admin required"/"team_admin required" 로 갈리는데 통일하면 응답 보존 관례를 깬다. `auth.HasRole`/`teams.IsTeamAdmin` 서비스 시그니처는 건드리지 말 것.
- 프로필: 6일 전 것이 현재 코드와 어긋나지 않아 새로 쓰지 않았다(파일 크기·미머지 브랜치·검증 함정 모두 이번 확인과 일치).
- [러너 12:07] scout done — `callerCanAdminTeam` 이 팀 관리자 판정의 DB 장애를 403 "team admin required" 로 위장하는 것을 500 으로 분리 (가치 3 

## 구현 노트
- `callerCanAdminTeam` 이 `auth.HasRole`/`teams.IsTeamAdmin` 오류를 `ok, _ :=` 로 버려 DB 장애가 팀 관리 10개 엔드포인트에서 403(영구 응답)으로 보였다. 헬퍼를 `(bool, error)` 로 바꾸고 `pgx.ErrNoRows` 만 흡수, 새 `requireTeamAdmin` 게이트가 호출 지점 10곳의 응답을 쓴다(장애 500, 그 외는 기존 403 + 호출 지점별 메시지 그대로). 프로덕션 2파일 + 신규 테스트 1파일.
- 확신 없는 곳: `router.go:756` 과 `:1054` 가 `POST /teams/{teamID}/invite/email` 을 둘 다 등록한다 — 어느 쪽이 실제로 도는지 확인하지 않았다. 스텁(`inviteTeamMembersFromBody`)이 이긴다면 `compat_wave_handlers.go:1317` 의 게이트는 죽은 코드일 수 있다(이번 변경이 만든 문제는 아니다). 테스트가 직접 커버한 것은 4개 경로(setTeamMemberRoles / removeTeamMember / uploadTeamImage / inviteGuestsByEmail)이고 나머지 6곳은 같은 헬퍼를 지나므로 빌드로만 확인했다. `updateTeamFull`/`patchTeam`/`updateTeamPrivacy`/`regenerateTeamInviteID`/`inviteTeamMembersByEmail` 은 HTTP 로 실행해 보지 않았다.
- 일부러 하지 않은 것: 차선 후보(`uploadTeamImage` 의 `denyGuestMutation` 누락)는 별개 동작 변경이라 뺐다 — ideas.json 에 pending 으로 남겼고, 이번 테스트 배선을 그대로 재사용하면 싸다. 오류 메시지 두 변형("team admin required" / "team_admin required")은 통일하지 않았다.
- 다음 역할이 조심할 것: `team_admin_errors_postgres_test.go` 는 `MOYRO_TEST_POSTGRES_DSN` 이 있어야 돈다(없으면 skip). 로컬 컨테이너는 `moyro-pg-improve`, `postgres://moyro:moyro@127.0.0.1:55433/moyro`. 테스트는 격리 스키마 안에서 `DROP TABLE team_members CASCADE` 를 하므로 서브테스트 순서(정상 → 403 → 자기탈퇴 → 감사 확인 → DROP → 장애)를 바꾸면 깨진다.
- [러너 12:19] brief accepted — 채택 — 과제서의 근거(헬퍼의 두 `ok, _ :=`, 호출 지점 10곳, 두 서비스가 Scan 오류를 그대로 올림, `listChannelViews` 선례, `DROP
- [러너 12:20] verify passed — 검증 2개 통과 (policy)

## 비평 노트
- 확인한 것: RED/GREEN 을 실제 DB(`moyro-pg-improve`, 55433)로 직접 재현했다 — main 의 두 프로덕션 파일로 되돌리면 4개 서브테스트가 원장에 적힌 그 줄(`status = 403, want 500`)로 실패하고, HEAD 에서는 통과한다. `go test -count=1 ./internal/httpapi ./internal/teams ./internal/auth` 전부 DSN 있는 상태로 통과(httpapi 39s — 러너의 policy 검증은 DSN 이 없어 이 테스트를 skip 했다). `HasRole`/`IsTeamAdmin` 은 부재를 `pgx.ErrNoRows` 로만 알리므로 흡수 범위가 정확하고, 권한이 넓어지는 방향의 변화는 없다(장애는 fail-closed 500). `listChannelViews` 선례·주석·커밋 메시지 모두 코드와 일치. `check-source-sizes.sh` 통과.
- 못 본 것: 9개 호출 지점 중 HTTP 로 실행된 것은 4개뿐이다(나머지는 같은 헬퍼 한 줄을 지나므로 위험은 낮다). 웹 클라이언트가 이 4xx→5xx 전환을 어떻게 표시하는지는 보지 않았다.
- 승인이어도 남는 우려: (1) 릴리즈 노트에 "팀 관리 10개 엔드포인트가 DB 장애 시 403 대신 500 을 준다 — 클라이언트 재시도 동작이 바뀐다" 를 적어야 한다. (2) `router.go:756`/`:1054` 의 `POST /teams/{teamID}/invite/email` 중복 등록은 이번 변경과 무관하게 미해결이다 — 나중 등록(`inviteTeamMembersFromBody` 스텁)이 이기면 `compat_wave_handlers.go:1317` 의 게이트는 도달 불가다. 다음 회차의 독립 과제로 적합. (3) `compat_wave_handlers_final.go:98` 등이 500 본문에 `err.Error()` 를 그대로 싣는다(DB 오류 문자열 노출) — 이번 변경은 일반 문구를 쓰므로 새 노출은 없지만 형제 경로에 남아 있다.
- [러너 12:24] review approved — 리뷰 승인 (risk=low)
- [러너 12:24] pr created — https://github.com/hkjang/moyro/pull/28
- [러너 12:36] ci passed — 검사 3개 모두 success
- [러너 12:36] merge done — dcb9487
- [러너 12:55] release published — v0.2.41
- [러너 13:10] assets verified — v0.2.41 자산 1개 (이전 v0.2.40: 1)
