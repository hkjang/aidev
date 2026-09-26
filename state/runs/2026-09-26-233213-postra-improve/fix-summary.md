# 수리 요약 (수리 시도 1)

- 비평가 지적이 맞았다. `respBytes += int64(len(line))` 는 CRLF 가 떼인 길이를 세므로 빈 줄이 0 바이트였고, 빈 줄만 흘리는 서버에 대해 `maxResponseBytes` 가 영원히 걸리지 않았다. 새 테스트 `TestIMAPEndlessBlankLinesAbandonSession`(실제 Dialer + 127.0.0.1) 로 수정 전 재현 — 75초 `-timeout` 에서 `exec`→`readLine` 루프 중 패닉, 즉 60초 commandTO 를 지나서도 계속 돌았다.
- 고친 방법: 상수 `responseLineOverhead = 2 + 16`(서버가 실제로 보낸 CRLF + `untagged` 에 남는 string 헤더)을 도입해 줄 단위 누적 두 곳(client.go:249 의 `line`, :301 의 `cont`)에 함께 부과했다. 빈 줄 우회와 부수 지적(1바이트 줄 × 8 MiB = 실상주 ~134 MiB)을 한 수정으로 닫는다 — 이제 최악의 상주량이 대략 상한값에 묶인다. 상한 값(8 MiB)·리터럴 개수 상한(64)·리터럴 바이트를 세지 않는 규약은 그대로다.
- 검증: 수정 후 `go test -race -count=1 ./internal/adapters/imap/` 통과(13건, 새 테스트는 0.37초에 `errUnframed`/"response exceeds" 로 중단). `go build ./...`, `go vet ./...`, `make lint`(gofmt 무출력, gosec Issues 0) 통과. 전체 `go test -race ./...` 는 백그라운드 완료 후 커밋 직전 확인.
- 정상 경로 가드도 새 회계에 맞췄다: `TestIMAPFullEnumerateBatchStaysUnderResponseBound` 가 이제 실제 부과량(2000줄 = 81,994 + 36,000 = **117,994** 바이트, 상한의 1/71)을 재고 8배 여유를 단언한다. 테스트를 지우거나 느슨하게 한 곳은 없다.
- 남긴 것: 릴리즈 노트. 이 변경은 아직 미출시라 이미 나간 `docs/releases/v0.23.6.md` 에 적으면 거짓이 되므로, 비평가가 요청한 "`refusalThreshold()==0` 계정은 64 × 무제한 바이트(개선이지 상한이 아님)" 한 줄은 릴리즈 역할이 다음 버전 노트에 넣어야 한다.
