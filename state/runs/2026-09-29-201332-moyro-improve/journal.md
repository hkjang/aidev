# 회차 노트 2026-09-29-201332-moyro-improve — moyro
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 20:13] base pinned — main@a6d5106
- [러너 20:13] autonomy release — 

## 정찰 노트
- 웹훅 멤버십 과제를 고른 이유: 같은 핸들러가 같은 (CreatorID, ChannelID) 쌍을 두 번 읽는데 선행 검사는 403, Fire 아래 분기는 이미 500 `...permission_check` 를 낸다 — 새 오류 id 없이 두 경로를 일치시키는 프로덕션 1파일 과제다. inviteGuestsByEmail(어제 팀이미지와 같은 유형이라 연속 선택 가치 낮음), 북마크 4핸들러·preferences(정책 결정 선행), patchPost 되읽기(주입 수단 6회차 미해결), router 중복 라우트(승자 라우트 재현 선행)를 이 이유로 제쳤다.
- 확인한 것: handlers.go:3158-3162, 3168의 500 분기, webhooks/incoming.go:198-208 의 ActorID=hook.CreatorID, postcommand/service.go:243-264 authorize, router.go:198-216 AuthorizeCreate 배선, router.go:372 무인증 `/hooks/{hookID}`, channels/service.go:1077 SELECT EXISTS(→ ErrNoRows 없음).
- 추측으로 적은 것(미확인): httpapi 격리 스키마 테스트 헬퍼의 정확한 이름, 그리고 이번 회차에 테스트를 실제로 돌려 보지 않았다는 점(코드 변경·테스트 실행 없음). 구현자는 기존 `team_admin_errors_postgres_test.go` 배선을 먼저 열어 맞출 것.
- 구현자 주의: `pgx.ErrNoRows` 흡수 분기를 넣지 말 것(EXISTS는 항상 행을 낸다), Fire 아래 switch 무변경, 8KiB 상한·404·400 분기 무변경, handlers.go 크기 상한 검사 실행.
- 프로필은 0일 전 작성본이 이번에 열어 본 코드와 어긋나지 않아 새로 쓰지 않았다.
- [러너 20:17] scout done — `fireIncomingWebhook` 의 작성자 멤버십 조회 DB 장애를 403 `creator_not_member` 에서 500 으로 분리 (가치 3 / 위험 2 / �

## 구현 노트
- `handlers.go:fireIncomingWebhook` 의 `if err != nil || !ok` 를 둘로 갈랐다 — 조회 장애는 500 `api.webhook.incoming.fire.permission_check`(바로 아래 `Fire` 분기가 이미 소유한 id 재사용), `!ok` 는 기존 403·기존 메시지 그대로. 새 id·새 상태 코드·마이그레이션 없음. 커밋 94db712.
- 확신 없는 곳: 없다고 말하지 않겠다 — 500 본문에 `err.Error()` 를 싣는데, 이 라우트는 무인증 공개 경로다. 형제 분기(3153)가 이미 같은 방식이라 따랐지만, 이 저장소가 공개 경로의 오류 문구 노출 범위를 정한 적은 없다. 비평가가 먼저 볼 곳으로 지목한다.
- 검증 못 한 것: 웹앱은 전혀 돌리지 않았다(Go 전용 변경). 컨텍스트 취소·연결 끊김은 직접 재현하지 않았고 `DROP TABLE channel_members` 한 가지 장애로만 증명했다 — 같은 `err != nil` 경로라 동치로 봤다.
- 일부러 하지 않은 것: `Fire` 아래 `switch FailureCodeOf` 무변경(이미 옳다), 8KiB `MaxBytesReader`·404·400 분기 무변경, `pgx.ErrNoRows` 흡수 분기 미추가(`IsMember` 가 `SELECT EXISTS` 라 행이 항상 온다). 읽다 발견한 "FailureSave 가 400 post_failed 로 뭉뚱그려진다" 는 ideas.json 에 새 항목으로만 적고 손대지 않았다.
- 다음 역할 주의: 신규 `incoming_webhook_member_errors_postgres_test.go` 는 `MOYRO_TEST_POSTGRES_DSN` 이 없으면 **skip** 한다 — `ok` 를 통과로 오인하지 말 것(컨테이너 `moyro-pg-improve`, 호스트 55433). 서브테스트 순서에 의존한다: `DROP TABLE channel_members` 앞의 세 건이 먼저 돌아야 하므로 `t.Parallel()` 을 넣거나 순서를 바꾸면 깨진다.
- 실행한 검증: DSN 을 준 `go test -race -p 1 ./...` 전 패키지 실패 0건, `go build ./...`, `go vet ./...`, `gofmt -l`(2파일 clean), `git diff --check`, `bash scripts/check-source-sizes.sh`(handlers.go 124880/130000).
- [러너 20:27] brief accepted — 채택 — 과제서의 근거(`if err != nil || !ok`, 3168 의 500 분기가 같은 (CreatorID, ChannelID) 쌍을 읽는 것, `SELECT EXISTS` 라 ErrNoRows 없
- [러너 20:27] verify passed — 검증 2개 통과 (policy)
