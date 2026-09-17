# PR #28 수정 요약

- 문제 1·2: `Service.deliver`/`SendNow` 가 발송용 컨텍스트를 `complete → store.Update` 에도 그대로 넘겨, 응답 없는 릴레이(연결 마감 2*Timeout 까지 매달림)에서는 컨텍스트가 먼저 만료돼 기록이 `context deadline exceeded` 로 실패하고 행이 영원히 `queued` 로 남았다. 컨텍스트를 거부하는 저장소 + 컨텍스트가 끝날 때까지 매달리는 sender 로 재현(`TestStalledRelayOutcomeIsStillRecorded`, 수정 전 `Status:queued Attempts:0`).
- 수정: `complete()` 가 `context.WithoutCancel(ctx)` 위에 별도 `recordTimeout`(10s) 을 두고 기록하도록 바꿔 두 호출 경로를 한 곳에서 고침. 발송 여유(15s)는 `sendGrace` 변수로 빼 테스트가 줄일 수 있게 함(기본 동작 동일).
- 문제 3: 만료 임박 메일이 '키를 회전하면 새 키가 발급' 된다고 안내했지만 `keys.go` 회전은 만료일을 그대로 복사해 회전해도 만료를 벗어나지 못하고 같은 키 이름으로 메일이 반복됨. 회전 동작은 이 PR 밖의 기존 코드라 손대지 않고, 안내 문구를 '새 키를 발급해 연동에 넣고 옛 키는 폐기, 회전은 만료일을 이어받는다' 로 바꿈(`TestAPIKeysExpiringDoesNotRecommendRotation`).
- 검증: `go test ./...`, `go vet ./...`, `gofmt -l` 모두 통과. 커밋 `82caf4d` (변경 파일: internal/mail/service.go, message.go, mail_test.go).
