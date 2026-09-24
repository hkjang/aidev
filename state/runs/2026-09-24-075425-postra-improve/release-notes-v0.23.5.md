# Postra v0.23.5 — IMAP 리터럴 길이 오버플로로 수집이 죽던 문제 수정

IMAP 서버가 선언한 리터럴 길이(`{n}`)를 안전하게 다루는 패치 릴리즈입니다. 표현 범위를 넘는 길이가 수집 작업 전체를 패닉시키던 문제를 고치고, 수용한 리터럴도 선언한 크기를 미리 할당하지 않습니다. 새 설정 키, DB 마이그레이션, 새 필수 환경변수는 없습니다.

## 선언한 길이를 그대로 믿던 문제

- IMAP 어댑터는 서버 리터럴의 길이를 `strconv.Atoi` 로 읽고 **오류를 버렸습니다**(`internal/adapters/imap/client.go`). Go 의 `Atoi` 는 범위를 벗어난 입력에서 해당 타입의 최댓값을 돌려주므로, `{99999999999999999999}` 같은 선언은 오류 없이 `math.MaxInt64` 가 됐습니다.
- 상한(`Sync.MaxMessageBytes`)을 끈 계정에서는 거부 임계값이 꺼지므로 이 길이가 그대로 `make([]byte, n)` 에 들어갔습니다. 실제 TCP 서버로 재현한 결과는 OOM 이 아니라 **패닉**(`makeslice: len out of range`)이며, 해당 계정의 수집 작업이 그 자리에서 죽었습니다. 계정의 접속 호스트는 계정 소유자가 고르는 값이므로 이 길이는 신뢰할 수 없는 입력입니다.
- 길이를 믿는 문제는 범위 초과에만 있지 않았습니다. 파싱에 성공하는 큰 값(예 `{8000000000}`)이면, 서버가 실제로 몇 바이트를 보내든 **선언한 크기만큼 버퍼가 먼저 잡혔습니다.** 23바이트만 보내는 서버가 선언한 1 GiB(실측 1,073,742,152 바이트)를 그대로 할당시켰습니다.

- 길이는 이제 `strconv.ParseInt(…, 10, 64)` 로 읽고, 파싱에 실패하면 **뒤따르는 바이트 수를 알 수 없으므로 드레인하지 않고** 세션을 폐기합니다(`errUnframed` 로 감싼 `abandon`). 다시 맞출 오프셋이 없는 상태에서 계속 읽는 것은 안전하지 않으므로, v0.23.4 가 도입한 다른 프레이밍 붕괴 경로와 같은 처리를 따릅니다. 이후 모든 명령(`exec`·`Idle`)은 즉시 실패합니다.
- 수용한 리터럴은 새 `readLiteral` 이 `bytes.Buffer` + `io.CopyN` 로 1MiB 청크 단위로 받습니다. **선언한 바이트가 아니라 도착한 바이트만큼만** 버퍼가 커지므로, 짧은 응답이나 오지 않는 응답을 임의 크기의 할당으로 증폭시킬 수 없습니다. 선예약은 1MiB 로 묶고, `discardLiteral` 과 마찬가지로 청크마다 명령 데드라인을 갱신해 느리지만 살아 있는 서버가 본문 중간에 끊기지 않게 합니다.

## 검증

- 실제 `net.Listen` 서버(`declaredLiteralServer`)와 실제 `Dialer` 를 쓰는 테스트 2건을 추가했습니다: 범위를 벗어난 길이 → `errUnframed` 반환 및 이후 명령도 계속 실패, 1 GiB 선언·23바이트 전송 → 오류 반환 + `runtime.MemStats.TotalAlloc` 증가분 64MiB 미만. 두 테스트 모두 수정 전 코드에서 실패함을 먼저 확인했습니다.
- 변이 2종으로 각 테스트가 실제로 결함을 잡는지 확인했습니다: `ParseInt` → `Atoi` 되돌림 → 파싱 테스트만 실패, 스트리밍 → `make([]byte, n)` 되돌림 → 할당 테스트만 실패. 확인 후 원복했습니다.
- 기존 IMAP 테스트(`TestIMAPEnumerateAndFetch`·`TestIMAPRejectsOversizeLiteral`·`TestIMAPRefusedLiteralKeepsStreamFramed`·`TestIMAPRefusedLiteralResyncsAtDefaultLimit`·`TestResyncBudgetExceedsRefusalThreshold`·`TestIMAPUndrainableLiteralAbandonsSession`)는 수정 없이 통과합니다.
- `go test -race -count=3 ./internal/adapters/imap/`, 전체 `go test -race`, `go build ./...`, `go vet ./...`
- `make lint`: `gofmt` 빈 출력 + CI 와 같은 플래그의 `gosec v2.28.0` Issues 0
- 계약 검사(`postra-contracts -check`), `git diff --check`
- 프런트엔드 소스 변경이 없어 포함된 `/app` 번들은 v0.23.4 와 동일합니다.

테스트는 로컬 루프백 서버만 사용하며 운영 메일 서버나 운영 데이터에 접근하지 않습니다. 실제 IMAP 서버에서의 동작과 외부 PostgreSQL, 32비트 빌드는 이번에 검증하지 않았습니다.

## 업그레이드 및 호환성 주의사항

DB 마이그레이션, 새 설정 키, 새 필수 환경변수는 없습니다. DB·SecretStore·ObjectStore·KEK를 백업하고 유지한 상태에서 실행 파일 또는 컨테이너 이미지를 교체한 뒤 재시작하세요.

- **`sync.max_message_bytes` 를 0 또는 음수(=상한 없음)로 둔 IMAP 계정이 있다면 이번 릴리즈가 특히 해당됩니다.** 이전 버전에서는 서버가 비정상적인 길이를 선언하는 것만으로 그 계정의 수집이 패닉으로 중단될 수 있었습니다.
- 정상적인 메일 수집 동작과 결과는 달라지지 않습니다. 리터럴을 받는 방식만 바뀌었고, 거부 임계값·드레인 예산·설정 키는 v0.23.4 그대로입니다.
- 읽을 수 없는 길이를 선언하는 서버에서는 세션이 닫히고 그 계정의 남은 메일이 해당 회차에서 `Failed` 로 집계됩니다. 다음 동기화 회차에서 새 세션으로 다시 시도합니다.
- POP3·SMTP 경로, MCP·OAuth·API Key 동작은 v0.23.4 와 같습니다. [v0.23.4 릴리즈 내용](v0.23.4.md)을 참고하세요.

## 배포 파일

- `postra-0.23.5-linux-amd64`: 정적 단일 실행 파일
- `postra-0.23.5.tar.gz`: 폐쇄망 `docker load` 이미지, `postra:0.23.5` (linux/amd64)
- `postra-0.23.5-sbom.cdx.json`: Go CycloneDX SBOM
- `postra-0.23.5-frontend-sbom.cdx.json`: 프런트엔드 CycloneDX SBOM
- `SHA256SUMS.txt`: 배포 파일 SHA-256
