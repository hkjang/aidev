# fix-summary

- 문제: `testMail`(internal/app/mail.go)이 `sendCtx`(timeout+5s)로 SMTP 발송 후 **같은 sendCtx**를 `mailFinish`에 넘기고 오류를 버렸음. 릴레이가 TCP만 받고 응답하지 않으면 `sendSMTPNotification`은 정확히 sendCtx 만료 시점에 반환하므로 `mailFinish`의 DB Exec가 즉시 실패 → 행이 `status='sending'`, `body_encrypted=''`로 남고 lease 만료 후 `mailDeliver`가 재청구해 "(본문을 읽지 못했습니다)" 시험 메일을 한 통 더 보냄. 지적이 맞음.
- 재현: 응답 없는 TCP 리스너를 릴레이로 두고 `POST /api/admin/mail/test`를 호출하는 `TestMailTestSendAgainstSilentRelayIsRecordedOnce` 추가 → 수정 전 행이 `status:sending`으로 남아 실패 확인.
- 수정: `recordCtx := context.WithoutCancel(r.Context())`를 만들고 sendCtx는 그 위에 timeout을 얹어 파생, `mailFinish`에는 만료되지 않는 `recordCtx`를 전달. 오류는 버리지 않고 `slog.Error`로 남김(id만, 비밀값 없음).
- 검증: `go vet ./...` 통과, `go test -race ./...`(HUNTER_TEST_DSN 로컬 PG) 통과. 새 테스트는 행이 `failed`/본문 삭제로 닫히고 lease 만료 후에도 `mailDeliver`가 아무것도 집지 않음을 단언.
