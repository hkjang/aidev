# PR #21 fix summary (commit 396312b)

- 문제 1: 메일 링크가 `/admin#approvals`·`#change-sets`·`#alerts` 였는데 admin UI(parseHash)는 `#/…` 만 라우팅함 → `mailPathApprovals`/`mailPathAlerts`=`/admin#/safety`, `mailPathChangeSets`=`/admin#/changesets` 상수로 고치고, 잘못된 링크를 단언하던 `mail_test.go:220` 을 실제 경로(`#/changesets`, `#/safety`) 단언으로 바꿈.
- 문제 2: 헤더 없는 재시도마다 새 approval + 관리자 전원 메일, 발송 고루틴 상한 없음 → `Server.mailApprovals`(`mailThrottle`)로 요청자(UserID→APIKeyID)당 10분에 한 통만 발송(approval 행은 그대로 생성), `mail.Service` 에 `MaxInFlight=64` 슬롯을 두고 초과분은 고루틴을 띄우지 않고 `ErrBusy` 로 ledger 에 failed 기록.
- 문제 3: 승인 메일 문구 "같은 요청을 다시 보내면 이 승인으로 처리" 가 게이트 동작과 불일치 → "`X-Governance-Approval-ID: <id>` 헤더와 함께 재전송하면 처리, 헤더 없이 보내면 새 승인 요청으로 다시 대기" 로 고치고 테스트로 고정.
- 검증: 새 테스트 `TestNotifyCapsDeliveriesInFlightAndRecordsTheOverflow`, `TestMailThrottleAllowsOncePerWindowPerKey`, `TestApprovalRequestsFromOneRequesterMailTheAdminsOncePerWindow`(실제 거버넌스 게이트로 3회 재시도 → approval 3건, 메일은 관리자당 1통) 추가; `go vet ./...`, `go build ./...`, `go test ./...` 통과. 웹 검증은 web 파일 변경이 없고 node_modules 미설치라 생략.
- ADMIN_GUIDE.md 의 해당 표·규칙 문장도 새 동작에 맞춰 갱신함(ADMIN_GUIDE.pdf 는 재생성 도구가 없어 그대로 둠).
