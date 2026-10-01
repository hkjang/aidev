- 과제: `fireIncomingWebhook` 의 `FailureSave`(진짜 게시글 저장 장애)를 400 `post_failed` 에서 500 으로 분리 (가치 3 / 위험 1 / 작업량 S)

- 왜: `server/internal/httpapi/handlers.go:3174` 의 Fire 오류 switch 가 `default:` 에서 400 `api.webhook.incoming.fire.post_failed` 를 내는데, 이 default 안에 `postcommand.FailureSave`(= `posts` INSERT 가 실패한 진짜 저장소 장애)와 `empty text`(= `webhooks.Fire` 가 `errors.New` 로만 내는 진짜 클라이언트 오류)가 같이 들어온다. 통합 발신자는 400 을 "이 요청은 영구히 잘못됐다" 로 읽고 재시도를 멈추므로 DB 순단 동안의 알림이 조용히 사라진다. **같은 `postcommand.Execute` 를 부르는 형제 어댑터 `createPost`(handlers.go:1623 switch)는 이미 같은 실패를 `default: → 500 api.post.create.save.app_error` 로 낸다** — 두 어댑터가 같은 실패 객체를 다르게 읽고 있고, 이번 과제는 그 둘을 일치시키는 것이다.

- 수용 기준:
  1) `posts` 테이블이 사라진 상태에서 `POST /hooks/{hookID}` 에 정상 페이로드를 보내면 **500** 이 나온다(현재는 200 아닌 400). 응답 id 는 같은 핸들러가 이미 쓰는 500 id `api.webhook.incoming.fire.app_error` 를 **재사용**한다(새 id·새 상태 코드 금지).
  2) `{"text":"   "}`(공백만) 은 수정 후에도 **400 `api.webhook.incoming.fire.post_failed`** 로 남는다 — `empty text` 는 `FailureCode` 가 없어 `FailureCodeOf` 가 `""` 를 돌려주므로 `default:` 분기를 400 그대로 두면 자동으로 보존된다. 이 서브테스트가 "switch 를 통째로 500 으로 바꾸지 않았다" 를 증명한다.
  3) 기존 계약 무변경을 같은 실행에서 재확인: 정상 발사 200 + `posts` 행 저장, 작성자 비회원 403 `creator_not_member`, 멤버십 조회 장애 500 `permission_check`, 없는 훅 404 `not_found`, 빈 본문 400 `invalid_body`.
  4) 테스트가 **프로덕션 배선**으로 돈다: 실제 PostgreSQL 격리 스키마 + `store.Migrate` + 실제 `rbac.NewPostgres`/`channels.New`/`posts.New`/`webhooks.NewIncoming(db, pcSvc)` + `router.go:198-216` 의 `AuthorizeCreate` 클로저 복사. 손으로 만든 대역(가짜 executor) 금지 — 기존 파일 `server/internal/httpapi/incoming_webhook_member_errors_postgres_test.go:44-145` 가 이 배선을 이미 글자 그대로 갖고 있으니 그대로 복사해 쓸 것.
  5) 수정 전에 먼저 돌려 RED 를 관찰하고(저장 장애 서브테스트만 `status = 400, want 500` 으로 실패), 수정 후 새 `case` 한 줄을 되돌리면 그 서브테스트만 다시 깨지는 것을 확인한다.

- 건드릴 파일:
  - `server/internal/httpapi/handlers.go` — `fireIncomingWebhook` 의 switch(3174~3182)에 `case postcommand.FailureSave:` 를 추가하고 `writeError(w, 500, "api.webhook.incoming.fire.app_error", err.Error())`. `default:` 는 400 `post_failed` 그대로 둘 것. 왜 이렇게 갈랐는지 짧은 주석(이 저장소 관례: 위 멤버십 분기에 이미 같은 형태의 주석이 있다).
  - `server/internal/httpapi/incoming_webhook_save_fault_postgres_test.go` (신규) — `TestIncomingWebhookSaveFaultIsNotBadRequest`. **반드시 별도 Test 함수 + 별도 `newOperationsTestDB(t)`** 로 만들 것: 기존 파일의 테스트는 중간에 `DROP TABLE channel_members` 를 하므로 같은 함수 안에 끼우면 멤버십 분기에서 먼저 끊긴다.
  - 프로덕션 파일 1개 + 신규 테스트 1개. 그 이상 늘리지 말 것.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - 테스트 DB: 컨테이너 `moyro-pg-improve` 가 **Exited 상태로 존재**한다(2일 전 정지). `docker start moyro-pg-improve` 로 되살리고 `docker port moyro-pg-improve` 로 호스트 포트를 확인한 뒤 `MOYRO_TEST_POSTGRES_DSN` 을 주고 돌릴 것. 이번 정찰 세션에서는 `docker start` 가 권한 승인에 막혀 **실행하지 못했다(미확인)** — 포트가 55433 이라는 이전 회차 기록은 재확인 못 했다. 접속 비밀은 출력·문서·커밋에 복사하지 않는다.
  - `cd server && go test -count=1 -run TestIncomingWebhook ./internal/httpapi/` — DSN 없으면 **skip 되므로 `ok` 를 통과로 오인하지 말 것**. `-v` 로 skip 여부를 눈으로 확인할 것.
  - `cd server && go test -race -p 1 -count=1 ./internal/httpapi/` (DSN 주고, 40~55초)
  - `cd server && go test -race -p 1 ./...` (DSN 주고, 수 분)
  - `cd server && go build ./... && go vet ./...`
  - `gofmt -l` 은 수정·신규 2파일만 확인할 것(`server/internal` 전체는 기존 9파일 드리프트를 출력한다 — 독립 포맷 PR 금지).
  - `bash scripts/check-source-sizes.sh` — handlers.go 는 124880/130000 이라 여유가 크지 않다. 주석을 길게 쓰지 말 것.
  - 이번 정찰 세션 실측: DSN 없이 `go test -count=1 ./internal/httpapi/` → `ok … 0.172s` (컴파일·비DB 테스트 통과).

- 위험과 피할 것:
  - **같은 switch 의 다른 분기를 손대지 말 것.** `FailurePluginRejected` 가 400 으로 나가는 것(형제 `createPost` 는 403), `FailureInvalidRoot` 가 400 default 로 나가는 것(웹훅 Command 는 `RootID` 를 세팅하지 않아 실제로는 도달 불가)은 **이번 범위 밖**이다. 보류 아이디어로 남겼다.
  - `webhooks.Fire`(`server/internal/webhooks/incoming.go:180-182`)의 `errors.New("empty text")` 를 `Failure` 로 감싸려 하지 말 것 — 그 순간 `FailureCodeOf` 가 바뀌어 400 계약이 깨지고 범위가 서비스 패키지로 번진다.
  - 장애 주입: `posts` 테이블을 격리 스키마 안에서 치우는 것이 결정론적이다. Fire 경로가 `posts` 를 건드리는 지점은 `postcommand.Service.create` 의 INSERT **하나뿐**임을 확인했다 — `incoming.Get` 은 `incoming_webhooks`, `channels.IsMember` 는 `channel_members`, `AuthorizeCreate`(router.go:198-216)는 channels/users/rbac 만 읽고, `RootID` 가 비어 있으므로 `posts.Get` 선행 조회는 돌지 않는다. **`DROP TABLE posts` 가 다른 테이블의 FK 때문에 거부될지는 미확인** — 거부되면 `DROP TABLE posts CASCADE` 또는 FK 를 건드리지 않는 `ALTER TABLE posts RENAME TO posts_hidden` 로 바꿀 것(둘 다 INSERT 를 "relation does not exist" 로 실패시킨다). 격리 스키마 밖으로 새지 않는지는 `newOperationsTestDB`(native_operations_postgres_test.go:86)의 search_path 설정을 먼저 읽어 확인할 것.
  - 보호 경로 무관: auth·session·migrations·`.github/workflows` 를 건드리지 않는다. 웹·마이그레이션·서비스 시그니처 변경 없음.
  - `post_failed` 문자열은 저장소 전체에서 handlers.go:3180 **한 곳에만** 존재한다(webapp·docs·테스트에 소비자 없음 — 이번 세션에서 grep 으로 확인). 그래도 2)번 기준으로 400 을 회귀에 고정해 둘 것.

- 차선 후보: **customprofile 패키지 단위 테스트 — 빈 맵 `PatchUserValues` no-op / null DELETE 계약 (가치 2 / 위험 1 / 작업량 S)**. 2026-09-26 회차의 장애 주입이 "빈 맵이면 tx 를 열기 전에 반환" 이라는 전제 위에 서 있는데 그 전제를 지키는 테스트가 없어, 누가 구현을 바꾸면 그 회귀가 조용히 무의미해진다. 프로덕션 코드 변경 0 파일. (패키지에 테스트 파일이 있는지는 이번에도 **미확인** — 없다면 `server/internal/customprofile/service_test.go` 신규.)
