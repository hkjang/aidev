# 과제서 — 2026-09-29-201332-moyro-improve

- 과제: `fireIncomingWebhook` 의 작성자 멤버십 조회 DB 장애를 403 `creator_not_member` 에서 500 으로 분리 (가치 3 / 위험 2 / 작업량 S)

- 왜: `server/internal/httpapi/handlers.go:3158` 이 `ok, err := h.channels.IsMember(r.Context(), hk.ChannelID, hk.CreatorID)` 뒤에 `if err != nil || !ok` 로 접어, 연결 실패·`channel_members` 테이블 장애·컨텍스트 취소를 403 `api.webhook.incoming.fire.creator_not_member` ("hook creator no longer a channel member") 로 낸다 — 통합 발신자에게는 "이 훅은 이제 못 쓴다"(재시도하지 않는 영구 응답)로 보여 DB 순단 동안의 알림이 조용히 사라진다. **같은 핸들러가 같은 값을 두 번 읽으면서 답이 다르다**: 바로 아래 `Fire` 오류 분기(handlers.go:3168)는 `postcommand.FailurePermissionCheck, postcommand.FailureMembershipCheck` 를 이미 500 `api.webhook.incoming.fire.permission_check` 로 내고, `Fire` → `postcommand.Execute` → `authorize` 가 쓰는 `(ActorID=hook.CreatorID, ChannelID=hook.ChannelID)` 쌍은 선행 검사와 **글자 그대로 같은 쌍**이다(`webhooks/incoming.go:198-208`). 선행 검사만 맞추면 두 경로가 같은 입력에 같은 답을 한다.

- 수용 기준:
  1) `channel_members` 접근이 실패하는 상태에서 `POST /hooks/{hookID}` 가 **500** 과 기존 id `api.webhook.incoming.fire.permission_check` 를 낸다(새 id·새 상태 코드 만들지 않음 — 같은 핸들러가 이미 소유한 id 재사용).
  2) 작성자가 **진짜로** 채널 멤버가 아닌 경우는 기존대로 403 `api.webhook.incoming.fire.creator_not_member` + 메시지 "hook creator no longer a channel member" 가 글자 그대로 유지된다.
  3) 정상 발사는 200 `{"status":"OK"}` 로 유지되고 게시글이 실제로 저장된다(선행 검사 통과 경로 무변경).
  4) 훅 없음/비활성 404, 빈 본문 400, 8KiB 상한 분기는 불변.
  5) 테스트가 증명할 것: **수정 전 코드에서 장애 케이스가 403 으로 실패(RED)** 하고, 수정 후 500 이며, 비회원 403 단언은 수정 전에도 후에도 통과한다(즉 이 변경이 진짜 비회원 거절을 느슨하게 만들지 않았다).

- 건드릴 파일 (프로덕션 1개 + 신규 테스트 1개):
  - `server/internal/httpapi/handlers.go:fireIncomingWebhook` (3158-3162) — `if err != nil || !ok` 를 둘로 가른다. `err != nil` → `writeError(w, 500, "api.webhook.incoming.fire.permission_check", err.Error())`, `!ok` → 기존 403 그대로. `errors` 는 이미 이 파일에서 import·사용 중(3149)이고 새 import 는 필요 없다. `channels.Service.IsMember`(`channels/service.go:1077`)는 `SELECT EXISTS(...)` 라 행이 항상 하나 돌아오므로 `pgx.ErrNoRows` 흡수 분기를 **넣지 말 것** — 모든 오류가 진짜 장애다.
  - `server/internal/httpapi/incoming_webhook_member_errors_postgres_test.go` (신규) — 아래 배선.

- 검증 명령:
  - `cd server && go test -race -p 1 -count=1 ./internal/httpapi/ -run TestIncomingWebhookMemberCheck` (실제 DB 필요: `MOYRO_TEST_POSTGRES_DSN`; 없으면 skip 하므로 `ok` 를 통과로 오인 금지 — 컨테이너 `moyro-pg-improve`, 호스트 포트 55433)
  - 전체: `cd server && go test -race -p 1 ./...`, `go build ./...`, `go vet ./...`
  - `gofmt -l` 은 **수정·신규 파일만** 확인(기존 9파일 드리프트는 그대로 두고 포맷 PR 금지)
  - `bash scripts/check-source-sizes.sh` (handlers.go 가 상한 130000 에 가까우니 반드시 실행)

- 테스트 배선 (손으로 만든 대역 금지 — 프로덕션 배선을 그대로 복사):
  - 기존 `*_postgres_test.go` 들의 격리 스키마 헬퍼 + `store.Migrate` 를 그대로 쓴다.
  - 실제 `rbac.NewPostgres(db.Pool)`, `channels.New(db)`, 실제 `posts` 서비스, `postcommand.New(...)` 에 **`router.go:198-216` 의 `AuthorizeCreate` 클로저를 글자 그대로 복사**해 넣고, `webhooks.NewIncoming(db, pcSvc)` 를 `handlers.incoming` 에 꽂는다(`webhooks/incoming.go:41`). `Fire` 를 가짜 executor 로 대체하지 말 것.
  - 라우트는 `router.go:372` 의 `Post("/hooks/{hookID}", h.fireIncomingWebhook)` chi 패턴을 글자 그대로 복사한다. 이 경로는 `/api/v4` **밖**이고 인증 체인 밖이다(요청에 토큰을 붙이지 말 것).
  - 장애 주입: 같은 격리 스키마 안에서 `DROP TABLE channel_members`. 선행 검사가 여기서 끊기므로 `Fire` 는 애초에 돌지 않는다(결정론적). 훅 행은 `incoming_webhooks` 에 있으므로 `h.incoming.Get` 은 계속 성공한다 — 이것이 404 가 아니라 멤버십 분기까지 도달함을 보장한다.
  - 미확인: 이 패키지의 격리 스키마 헬퍼 정확한 이름(다른 회차 기록은 `newOperationsTestDB` 를 언급). 기존 `team_admin_errors_postgres_test.go` / `upload_stub_limits_postgres_test.go` 를 열어 실제 이름을 확인하고 맞출 것.

- 위험과 피할 것:
  - 새 오류 id 를 만들지 말 것. 500 은 이 핸들러가 이미 쓰는 `api.webhook.incoming.fire.permission_check` 를 재사용한다(기준 1).
  - `Fire` 아래의 `switch postcommand.FailureCodeOf(err)` 분기는 **건드리지 말 것**. 이미 옳다.
  - 8KiB `MaxBytesReader`(3141)는 이번 범위 밖 — 보류 아이디어의 "남은 MaxBytesReader 호출자" 와 합치지 말 것(계약이 경로마다 다르다).
  - `denyGuest*` 의 401 위장, 북마크 4개 핸들러, preferences 400 분리를 이 회차에 끼워 넣지 말 것(각각 별도 과제).
  - 보호 경로 미접촉: `auth`/`session`/`store/migrations`/`.github/workflows` 를 건드리지 않는다. 마이그레이션 없음.
  - grep 결과를 증거로 제출하지 말 것 — RED 실행 출력(403 → 500)을 그대로 붙일 것.
  - 이 라우트는 무인증 공개 경로다. 500 본문에 `err.Error()` 를 넣는 것은 형제 분기(3153)가 이미 하는 방식과 같으니 그대로 따르되, 그 이상으로 내부 정보를 늘리지 말 것.

- 차선 후보: `customprofile` 패키지 단위 테스트 신설 — 빈 맵 `PatchUserValues` 가 tx 전에 no-op 로 반환하는 계약과 null 값 DELETE 계약 (가치 2 / 위험 1 / 작업량 S). 2026-09-26 회차가 이 no-op 전제 위에 장애 주입을 세웠는데 그 전제를 지키는 테스트가 없다.

## 추정 근거 (basis of estimate)
- 분해: 프로덕션 3~5줄(1) / 테스트 배선 복사(2) / 3개 서브테스트(3) / RED 선실행 + 되돌림 확인(1) / 전체 검증 실행(1).
- 방법: 유추 — 2026-09-28 `callerCanAdminTeam`(호출 지점 10곳, M)과 2026-09-27 업로드 스텁(3핸들러, S)의 실적. 이번은 호출 지점 1곳이라 그 사이 아래쪽.
- 범위 제외: 웹 변경 없음, 마이그레이션 없음, 다른 `IsMember` 호출자 무변경.
- 신뢰 구간: 45분 안 완료를 10회 중 8회. 가장 크게 의존하는 가정 = postcommand/webhooks 배선 복사가 30분 안에 컴파일된다는 것. 여기가 깨지면 계획도 여기서 깨진다.
- 컨틴전시: 배선이 예산을 먹으면 정상 200 단언(기준 3)을 마지막으로 미루고 기준 1·2·5 를 먼저 확보할 것 — 이 과제의 값은 장애 분리에 있다.
