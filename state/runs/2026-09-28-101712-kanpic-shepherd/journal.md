# PR 처리기 노트 2026-09-28-101712-kanpic-shepherd — kanpic PR #35
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-09-28-092238-kanpic-improve)
# 회차 노트 2026-09-28-092238-kanpic-improve — kanpic
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 09:22] base pinned — main@6265307
- [러너 09:22] autonomy release — 

## 정찰 노트
- 왜 이것을 골랐나: 보류 목록에서 가치 3 인 유일한 S 급이고, 커버리지(35.3%)와 grep 으로 `internal/mail/service.go` 테스트 0건을 실제로 확인했다. 차선인 ADMIN_GUIDE 수정(가치 2)은 세 회차 연속 밀렸는데 PDF 재생성 부담 때문이며 이번에도 같은 이유로 2순위다. 파일·클립보드 숫자 계약 계열은 지난 네 회차가 다 먹었고 남은 것은 계약을 먼저 정해야 하는 M 이라 제외했다.
- 확신 없는 곳: (1) 취소된 컨텍스트에서 `Notify` 가 실제로 알림을 빠뜨리는지 **돌려 보지 않았다** — 과제서에 빨강/초록 두 갈래를 다 적었으니 구현자가 테스트로 판정할 것. (2) pgx 가 취소된 ctx 에서 즉시 오류를 내는지 코드로 재확인하지 않았다(미확인). (3) `fakeRelay.body` 가 필드 하나라 다수 수신자 본문 비교가 안 된다는 것은 mail_test.go:112 를 읽고 판단했다.
- 구현자가 조심할 것: `pool` 은 nil 로 두되 `Deliveries` 는 부르지 말 것(nil 패닉). `SetSender` 로 대역을 끼우지 말고 실제 `Deliver`+`fakeRelay` 로 갈 것. `Notify` 가 고루틴을 띄우므로 `-race` 필수이고, 실패 재시도(2초 sleep)를 타는 경로는 테스트하지 말 것.
- [러너 09:27] scout done — 알림 메일의 "누가 받는가" 계약을 실제 relay 로 못 박고, 요청이 끊겨도 알림이 조용히 사라지지 않게 한�

## 구현 노트
- 무엇을 왜: `Notify` 가 요청 컨텍스트로 설정 읽기·수신자 조회·발송 기록을 해서, 클라이언트가 떠나면 pgx 가 곧바로 실패하고 알림이 조용히 사라졌다 → `WithTimeout(WithoutCancel(ctx), 10s)`. 새 `internal/mail/service_test.go` 가 프로덕션 `Deliver`+in-process relay 의 `RCPT TO` 줄로 수신자 7행·게이트 4가지를 고정한다. 프로덕션 파일 1개.
- 확신 없는 곳: (1) 10초 시한은 내가 고른 값이다 — `SendNow` 의 `config.Timeout+5s` 와 달리 설정에서 오지 않는다. (2) 취소 빨강은 테스트의 settings/directory 가 `ctx.Err()` 를 먼저 보게 해서 잡았다. 그 전제(pgx 는 취소된 ctx 에서 질의 전에 실패)는 `puddle/v2@v2.2.2 pool.go:338-344` 를 읽어 확인했지만 **실제 Postgres 로 돌려 보지는 않았다**. (3) `Deliveries`(nil 풀 패닉)는 여전히 아무 테스트도 지나지 않는다 — 통합 몫.
- 일부러 하지 않은 것: `SetSender` 로 전송 대체(운영자 규칙), 전송 실패·재시도 경로(2초 sleep — mail_test.go 가 `Deliver` 수준에서 이미 덮는다), 본문 비교(`fakeRelay.body` 는 필드 하나라 마지막 DATA 만 남는다), `mail.go`·`message.go`·`config.go`·TLS/AUTH·문서(그래서 PDF 재생성 없음).
- 다음 역할이 조심할 것: 새 테스트는 DB 없이 돈다(`pool=nil`) 대신 127.0.0.1 로 listen 하므로 루프백이 막힌 곳에서는 못 돈다. `Notify` 가 고루틴을 띄우니 `-race` 로 돌릴 것(`-count=3 -race` 까지 통과). `expectNothingSent` 는 200ms 를 자므로 그 세 테스트는 각 0.2초다.
- [러너 09:49] brief accepted — 채택 — 과제서가 두 갈래로 써 둔 것 중 '빨강' 갈래가 그대로 맞았고(취소된 요청에서 알림이 나가지 않았다), 건드릴 �
- [러너 09:49] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인한 것: main 의 service.go 를 `go test -overlay` 로 끼워 넣어 빨강/초록을 직접 봤다 — 취소 테스트만 `RCPT TO=[]` 로 실패하고 현재 코드에서는 `-race -count=1` 전부 통과(vet·gofmt 깨끗, 작업 트리 손대지 않음). 원장에 `- 실패 재현:` 줄이 없어 대신 재현했다. mail.Notify 의 6개 호출처를 모두 열어 봤다.
- 거절 사유: 커밋 메시지와 service.go:74-75 주석이 첫 예로 든 **칸 저장** 경로가 안 고쳐졌다. `internal/httpapi/mail.go:145` 의 `SheetWatchRules(r.Context(), …)` 가 취소된 컨텍스트에서 실패해 Notify 에 닿지도 못한다. `sharing.go:215-217` 의 접근 요청도 `workbookAudience(r.Context())` 가 nil 을 돌려줘 `mail.go:65` 에서 멈춘다. service.go 안의 WithoutCancel 로는 둘 다 구제되지 않는다.
- 수리가 먼저 볼 파일: `internal/httpapi/mail.go`(notifyWatchers)와 `internal/httpapi/sharing.go`(createAccessRequest). 둘을 `automations.go:249` 처럼 경계에서 취소를 떼든, 커밋 메시지·주석을 실제로 덮는 경로로 좁히든 하나면 된다 — 후자도 유효한 수리다.
- 못 본 것·남는 우려: 실제 Postgres 로 취소 갈래를 돌려 보지는 않았다(전제는 이번 변경과 동일). comments.go 경로는 이번 변경으로 메일이 나가기 시작하는데 취소 시 `book.Title` 이 비어 제목이 `[kanpic]  B2에 새 댓글` 이 된다. `waitForRecipients` 는 want 개를 보면 곧바로 돌아와 수신자 7행이 읽히는 것보다 느슨하다. 보안·법무 차단 사유는 없다.
- [러너 09:55] review rejected — 리뷰 거절: internal/httpapi/mail.go:145 커밋 메시지와 service.go:74-75 주석이 첫 예로 든 '칸 저장' 경로는 여전히 알림이 사라진다. notifyWatchers 는 publishCells 가 �
- [러너 09:55] pr created — https://github.com/hkjang/kanpic/pull/35

## 수리 노트
- 지적은 모두 맞았다. 취소된 요청 컨텍스트에서 `SheetWatchRules`(mail.go:145)와 `workbookAudience`(sharing.go:215) 가 먼저 실패해 `mail.Notify` 에 닿지도 못하는 것을 테스트로 재현했다 — 칸 저장 0통, 댓글은 언급 메일만 나갔다. 틀린 지적은 없었다.
- 비평이 세지 않은 같은 결함이 둘 더 있었다: `putWorkbookShare`(sharing.go:135, GetWorkbook 이 공유 알림을 막는다)와 `decideAccessRequest` 의 제목 조회.
- 고친 방법은 (1)번 갈래다. `notifyContext` = `WithTimeout(WithoutCancel(ctx), 10s)` 를 httpapi 에 두고 알림 채비 6곳에 적용. 시한 10초는 mail.Notify 와 맞췄다(설정에서 오지 않는다 — 둘 다 그렇다).
- 새 테스트는 `cancelAwareRepository` 로 pgx 의 "취소된 ctx 는 질의 전에 실패" 를 MemoryRepository 위에 되살린다. 전송은 `SetSender` 로 잡는다 — SMTP 와이어는 internal/mail 쪽 테스트가 이미 덮으므로 여기서 볼 것은 수신자뿐이다.
- 확신 없는 곳: 실제 Postgres 로 취소 갈래를 안 돌려 봤다(전제는 puddle pool.go:338-344 독해 그대로). 취소를 떼면 클라이언트가 떠난 뒤에도 핸들러가 저장소 읽기를 몇 번 더 하는데, 이는 automations.go 가 이미 택한 값이다.

## 심사 노트
- 확인한 것: 이전 거절이 짚은 세 경로를 직접 빨강/초록으로 봤다 — `git archive origin/main` 사본에 새 테스트만 얹으니 `TestNotifyWatchersSendsAfterRequestCancel`(칸 저장)·`TestWorkbookAudienceSurvivesRequestCancel`(접근 요청)·`TestNotifyCommentMailSendsAfterRequestCancel`·`TestNotifySendsEvenWhenTheRequestWasCancelled` 넷이 실패하고, 브랜치에서는 `go test -race -count=1 ./internal/httpapi ./internal/mail` 과 `go test ./...`·`go build ./...`·vet·gofmt 가 모두 깨끗하다.
- 전제도 소스로 확인했다: `pgxpool/pool.go:747-748` Query→Acquire, `pgxpool/pool.go:598` Acquire→puddle `Pool.Acquire` 첫머리 `select { case <-ctx.Done(): return nil, ctx.Err() }`. 취소된 요청 컨텍스트로 부른 저장소 읽기는 질의에 닿기 전에 실패한다 — 테스트 대역이 흉내 낸 것이 실제 동작이다.
- 알림 채비 6곳(mail.go:98,176 · comments.go:168 · sharing.go:135,217,253)이 모두 `notifyContext` 를 지나고, `notifyMail`→`mail.Notify` 는 제 안에서 다시 WithoutCancel 하므로 상위의 `defer cancel()` 이 전송을 끊지 않는다. 마이그레이션·엔드포인트·권한·의존성 변화 없음(revert 로 되돌아온다).
- 못 본 것: 실제 PostgreSQL 로 취소 갈래를 돌리지 못했다(`internal/integration` 에 이 경로의 통합 테스트는 없다). 대역이 틀렸다면 이 변경은 무해한 무효과이지 해가 아니다. 웹·E2E 는 무관해 돌리지 않았다.
- 권고 근거: 승인·머지. 남는 것은 차단 아닌 note 둘 — 클라이언트가 떠난 뒤에도 핸들러가 최대 10초 저장소를 읽는다(automations.go:249 의 30초 선례와 같은 값), 그리고 4e4d1d5 의 커밋 본문만 읽으면 httpapi 쪽 수리(8afb6a5)가 안 보이니 릴리즈 노트는 두 커밋을 함께 적을 것.
