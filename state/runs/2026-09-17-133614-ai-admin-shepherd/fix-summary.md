# PR #23 fix summary (commit f9ddcc1)

- 재현: ctx 를 존중하는 store 로 응답 없는 릴레이(Accept 후 침묵)+5s ctx+수신자 2명을 돌리면 r2 는 INSERT 만 되고 UPDATE 가 ctx 만료로 실패해 queued 로 남았고, `loginAuth.Start(ServerInfo{Name:"relay.internal",TLS:false})` 는 err=nil(PlainAuth 는 "unencrypted connection" 거부)이라 지적 모두 사실이었음.
- service.go: 기록 INSERT/UPDATE 는 `WithoutCancel`+10s 의 별도 ctx, 발송은 수신자별 `WithoutCancel`+`(2×timeout)×시도+대기+5s` 예산의 별도 ctx 로 분리하고 수신자 전원을 먼저 기록; Notify 는 설정/주소 조회조차 못 한 경우에만 error 를 돌려줌.
- server/mail.go: 스윕은 프로세스 ctx 로 돌고 claim/조회만 1분 예산, 소유자별 Notify 가 시도되지 못했거나(ctx 만료·조회 실패) 셧다운으로 남은 소유자가 있으면 `expiry_notified_at` 을 NULL 로 되돌려 다음 스윕이 다시 경고.
- mail.go: 자격증명은 TLS 위에서만 보내고 `mail.security=none` 을 명시한 경우에만 평문 허용 — PLAIN/LOGIN 모두 같은 규칙(자체 plainAuth 로 localhost 예외 제거), auto+STARTTLS 없음이면 두 경로 모두 ErrInvalid 안내; ADMIN_GUIDE 에 규칙 기재.
- 테스트: 응답 없는 릴레이에서 두 수신자 모두 failed(queued 아님) 단언, relay.internal 이름의 LOGIN/PLAIN 전용·STARTTLS 없음 가짜 릴레이에서 AUTH·비밀번호가 나가지 않음 단언, 중단된 스윕이 claim 을 되돌리고 다음 스윕이 마저 경고함을 통합 테스트로 단언(각각 수정 전 코드에서 실패 확인). gofmt/go vet/`go test -race ./...`(임시 Postgres 로 통합 테스트 포함) 통과; web 은 미변경이라 npm test 생략.
