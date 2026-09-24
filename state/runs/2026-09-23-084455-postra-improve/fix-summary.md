# fix-summary (수리 시도 2)

- 비평가 지적대로 `resyncDrainLimit = 32MiB` 는 거절 임계값(MaxMessageBytes+1MiB=51MiB, 기본값)보다 작아 드레인 경로가 출하 기본값에서 죽은 코드였다. 재현: 예산만 32MiB 로 되돌린 상태에서 새 기본값 테스트가 `54525952-byte literal exceeds the 33554432-byte resync budget` 로 실패함을 확인.
- 고침: 예산을 거절 임계값에서 유도(`refusalThreshold() = maxLiteral + 1MiB`, `resyncBudget() = threshold * 4`, 오버플로 시 MaxInt64 포화). 기본값이면 51MiB 초과 리터럴을 204MiB 까지 드레인하므로 거절된 본문은 항상 드레인 가능하게 시작하고, abandon 은 '길이가 거짓이라 바이트가 오지 않는' 경우의 폴백으로만 남는다(4GiB 테스트 그대로 통과).
- 테스트: `TestIMAPRefusedLiteralResyncsAtDefaultLimit` 이 `config.Default().Sync.MaxMessageBytes`(50MiB)로 52MiB 본문을 실제 소켓으로 흘려보내 드레인 후 다음 메시지 정상 수집을 단언(서버는 본문을 스트리밍해 메모리 미보유). `TestResyncBudgetExceedsRefusalThreshold` 가 모든 설정에서 예산 > 임계값 불변식과 오버플로 포화를 고정.
- 검증: `go build ./...`, `go vet ./...`, `go test -race ./internal/...` 전부 통과(imap 7개 테스트 PASS), `make lint`(gofmt + gosec, issues 0) 통과. 프런트·PostgreSQL·브라우저 검사는 이번 변경과 무관해 미실행.
- 남은 범위: sync.go 는 손대지 않았다 — 드레인이 살아나 기본 설정의 스톨 원인은 사라졌고, 거짓 길이(abandon) 서버는 어떤 클라이언트도 스트림을 읽고 넘어갈 수 없어 세션 종료가 불가피하다(루프 조기 종료는 별건).
