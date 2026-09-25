# 회차 노트 2026-09-25-192100-moyro-improve — moyro
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 19:21] base pinned — main@59f6512
- [러너 19:21] autonomy release — 

## 정찰 노트
- 고른 이유: 리마인더 생성 두 경로가 `posts.Get` 오류를 전부 404 로 접고 별칭만 `IsMember` 오류까지 403 으로 접어, 같은 리소스의 두 경로가 같은 입력에 다르게 답한다 — 운영자가 반복해 말한 "두 경로가 같은 값을 같게 읽는지" 에 정확히 걸리고, ded7e4e/761c3ae 로 이미 두 번 채택된 국소 패턴이라 위험이 낮다. 북마크 late.go IsMember(권한 예외 설계 선행, M), preferences upsert 400 분리(센티널 필요, M), 메일·e2e(선행 조건·브라우저 비용)는 제쳤다.
- 확인한 것: compat_wave_handlers.go:1029 `isMember, _ :=`, handlers.go:2182 는 이미 500, posts/service.go:634→scanPost:129 가 ErrNoRows 를 그대로 올림, router.go:546/741 두 경로 배선, handlers 구조체에 posts/channels/reminders/hub/auth 존재, requireUserParamAccess 는 actor==target 이면 auth nil 이어도 통과, uploadTeamImage 의 `_, _ = io.Copy`(차선 근거). `go test -count=1 ./internal/httpapi/` 통과(단 DSN 미설정이라 DB 테스트는 전부 skip — 통과로 읽지 말 것).
- 추측(미확인): 500 에 쓸 오류 id 를 `api.reminder.create.app_error` 로 제안했으나 구현자가 두 경로에서 같은 id 를 쓰기만 하면 다른 선택도 무방하다. `gofmt -l` 과 전체 `go vet`/실제 DB 테스트는 이번에 못 돌렸다.
- 구현자 주의: 새 오류 id·새 마이그레이션 만들지 말 것, 404 본문 id·메시지 유지, 장애 주입은 격리 스키마 안의 DROP TABLE 로만, late.go 북마크 등 다른 `ok, _ :=` 로 범위를 넓히지 말 것.
- 프로필은 2026-09-22 판이 지금 코드와 어긋나지 않아 새로 쓰지 않았다(base 만 main@59f6512 로 이동, v0.2.36).
- [러너 19:24] scout done — 리마인더 생성 두 경로가 DB 장애를 404/403 으로 위장하는 것을 500 으로 분리 (가치 3 / 위험 1 / 작업량 S)

## 구현 노트
- 무엇/왜: 리마인더 생성 본 경로(handlers.go:2164)와 별칭(compat_wave_handlers.go:1008)이 `posts.Get` 의 모든 오류를 404 로 접던 것을 `errors.Is(err, pgx.ErrNoRows) || (err == nil && p == nil)` 만 404(본문 id·메시지 불변), 나머지는 500 `api.reminder.create.app_error` 로 갈랐고, 별칭의 `isMember, _ :=` 를 본 경로와 같은 500 `api.reminder.create.member_check` 로 맞췄다. 커밋 b807fa5.
- 확신 없는 곳·검증 못 한 것: (1) `posts.Get` 의 주석은 "Returns nil,nil if missing" 이라고 하지만 실제로는 `pgx.ErrNoRows` 를 올린다 — 주석은 건드리지 않고 두 조건을 모두 404 로 받았다(계약 변경 회피). (2) 컨텍스트 취소(`context.Canceled`)도 500 으로 가는데, 이건 본 경로 IsMember 가 이미 하던 것과 같은 취급이라 의도적이다. (3) 로컬 postgres 는 기존 컨테이너 `moyro-pg-improve`(55433, postgres:16-alpine)를 재사용했다 — 새로 띄우려 했으나 포트가 이미 그 컨테이너에 잡혀 있었다. (4) webapp 은 Go 전용 변경이라 아예 돌리지 않았다.
- 일부러 하지 않은 것: late.go 북마크 4곳 등 다른 `ok, _ := IsMember` 로 확대하지 않음(범위 부풀리기), 새 오류 id·마이그레이션 없음, `posts.Service.Get` 시그니처·nil 의미 불변(pluginhost 등 다른 호출자 무영향).
- 다음 역할 주의: 신규 `internal/httpapi/reminder_errors_postgres_test.go` 는 `MOYRO_TEST_POSTGRES_DSN` 없으면 통째로 skip 된다 — ok 만 보고 통과로 읽지 말 것. 장애 주입은 `newOperationsTestDB` 가 만든 격리 스키마 안의 `DROP TABLE posts/channel_members CASCADE` 뿐이다. 수정 전 코드에서 404(본 경로 저장소 장애)·403(별칭 멤버십 장애)으로 실제 실패하는 것을 먼저 확인했고, 404 분기를 무력화하면 404 단언만 깨지는 것도 확인했다.
- [러너 19:33] brief accepted — 채택 — 과제서의 근거(두 핸들러의 `err != nil || p == nil` → 404, 별칭만 `isMember, _ :=`, `scanPost` 가 ErrNoRows 를 그대로 올림, han
- [러너 19:33] verify passed — 검증 2개 통과 (policy)

## 비평 노트
- 확인: main@59f6512 임시 worktree 에 신규 테스트만 얹어 red 재현(404/403 → want 500), HEAD 에서 green, 실 DB(PG16:55433)로 `go test -race ./internal/httpapi/` 전체 ok, `go vet` 통과. 오류 경로 fail-closed 확인(IsMember err → 500 return, 통과 없음) → 권한 확대 없음.
- 구현 노트 3건 의심 모두 검증됨: posts.Get 주석("nil,nil")은 실제 ErrNoRows 와 여전히 어긋나지만 두 조건 모두 404 로 받아 동작은 정확 — 주석 정정은 다음 회차 몫. context.Canceled→500 은 main 의 member_check 와 동일 취급.
- 못 본 것: webapp(Go 전용 변경), 전체 `go test ./...`(httpapi 만 돌림), PG15.
- 승인이어도 남는 우려: `api.reminder.create.app_error` 가 한 핸들러에서 조회 실패/삽입 실패 두 원인을 덮어 클라이언트가 구분 못 함. `err.Error()` 가 원시 PG 오류를 본문에 싣는 것은 저장소 전반 472곳의 관례라 이번 차단 사유는 아니나 전역 정리 과제로 남는다. `gofmt -l` 이 잡는 native_activity.go 는 main 부터 있던 것 — 이 PR 탓 아님.
- 릴리즈 노트: 리마인더 생성 두 경로에서 DB 장애가 404/403 대신 500 으로 보고된다(404 본문 id·메시지와 403 본문은 불변, 상태 코드만 분리). 새 오류 id·마이그레이션 없음.
