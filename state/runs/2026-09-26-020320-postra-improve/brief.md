- 과제: IMAP 프로토콜 줄 읽기에 길이 상한을 두어, 개행을 보내지 않는 서버가 클라이언트 메모리를 무제한으로 키우지 못하게 하기 (가치 3 / 위험 2 / 작업량 M)

- 왜: `internal/adapters/imap/client.go:153 readLine` 과 `:492 readLineNoReset` 이 둘 다 `s.r.ReadString('\n')` 을 상한 없이 호출한다(두 곳이 raw ReadString 을 쓰는 유일한 지점, 이번에 직접 열어 확인). 리터럴 본문 경로는 v0.23.4~v0.23.5 에서 닫혔지만(`readLiteral` 이 선언 크기가 아니라 도착 바이트만 버퍼) 프로토콜 줄 자체는 여전히 무제한이라, `\n` 을 한 번도 보내지 않고 바이트만 흘리는 서버 하나가 `ReadString` 의 누적 버퍼를 데드라인까지 키운다. 특히 IDLE 경로는 `client.go:550` 이 읽기 직전에 **28분** 데드라인을 걸어 두므로(`readLineNoReset` 은 의도적으로 데드라인을 갱신하지 않는다) 최악의 창이 명령 데드라인 60초가 아니라 28분이다. 고치면 "계정 소유자가 고른 신뢰할 수 없는 서버" 입력에 대해 메모리가 무제한으로 늘어나는 마지막 경로가 닫힌다.

- 수용 기준:
  1) 실제 `net.Listen` 서버가 `\n` 없이 바이트를 계속 흘릴 때, `Dial`/`exec` 경로가 명령 데드라인까지 버티지 않고 "줄 길이 상한 초과" 오류로 실패한다. 수정 전에는 같은 테스트가 데드라인까지 매달리거나 누적 할당이 상한을 크게 넘는다 — **구현자는 수정 전 실패를 먼저 재현하고 그 관찰값을 기록할 것**.
  2) 상한 초과 뒤 스트림 오프셋을 알 수 없으므로 세션을 폐기한다: 같은 세션에 대한 이후 명령이 모두 `errUnframed` 로 실패한다(`errors.Is(err, errUnframed)` 단언). 기존 `abandon`(`client.go:312`)을 그대로 쓴다 — 새 실패 모드를 만들지 말 것.
  3) 정상 동작 회귀 없음: 기존 IMAP 테스트 9개(`TestIMAPEnumerateAndFetch`, `TestIMAPAuthError`, `TestIMAPRejectsOversizeLiteral`, `TestIMAPRefusedLiteralKeepsStreamFramed`, `TestIMAPRefusedLiteralResyncsAtDefaultLimit`, `TestResyncBudgetExceedsRefusalThreshold`, `TestIMAPUndrainableLiteralAbandonsSession`, `TestIMAPOutOfRangeLiteralAbandonsSession`, `TestIMAPLiteralBufferTracksDeliveredBytes`)가 **무수정** 통과하고, 상한 미만의 긴 정상 줄은 온전히 반환된다(끝에서 잘리지 않음 — 반환 문자열 길이·내용 단언).
  4) 변이 검증: 상한 검사를 무효화하면 새 테스트만 실패하고 기존 테스트는 통과함을 확인한 뒤 원복.

- 건드릴 파일:
  - `internal/adapters/imap/client.go` — 새 상수 `maxLineBytes`(권장 1<<20, 근거 주석 필수: 메시지 본문은 `{n}` 리터럴로 오고 이 경로가 읽는 것은 프로토콜 줄뿐이라 1 MiB 는 넉넉하다) 추가. `readLine`(153)과 `readLineNoReset`(492)이 공통 헬퍼를 쓰게 한다. 헬퍼는 `s.r.ReadSlice('\n')` 을 돌려 `bufio.ErrBufferFull` 일 때 누적하고(기본 bufio 크기 4096), 누적이 `maxLineBytes` 를 넘으면 `s.abandon(fmt.Errorf("%w: protocol line exceeds %d bytes", errUnframed, maxLineBytes))` 를 반환한다. 데드라인 정책은 지금과 동일하게 유지 — `readLine` 만 진입 시 `s.deadline()`, `readLineNoReset` 은 갱신하지 않음(이 구분이 IDLE 의 28분 창을 지탱하므로 통합하지 말 것).
  - `internal/adapters/imap/client_test.go` — 기존 `fakeServer`/`declaredLiteralServer`(327) 와 같은 방식의 `net.Listen("tcp","127.0.0.1:0")` 스크립트 서버를 하나 더 추가해, 그리팅 뒤(또는 LOGIN 응답 자리에서) `\n` 없는 바이트를 계속 써 보낸다. 실제 `Dialer`+`dial`/`dialMax`(411/416) 로 접속할 것. 서버 고루틴이 테스트 종료 후에도 쓰기를 계속하지 않도록 `t.Cleanup` 으로 종료 신호를 줄 것.

- 검증 명령:
  - `go test -race -count=3 ./internal/adapters/imap/`  (기준선: 이번 정찰에서 `-count=1` 이 ok 1.083s 로 통과)
  - `go build ./... && go vet ./...`
  - `go test -race -count=1 ./...`  (전체 수 분, application 패키지가 가장 김)
  - `make lint`  (gofmt -l ./cmd ./internal + gosec v2.28.0 -severity medium -exclude-dir=scripts ./...)
  - `go run ./cmd/postra-contracts -check`
  - 새 테스트에 `-timeout 30s` 를 명시해, 수정 전 "매달림" 이 타임아웃으로 드러나게 할 것.

- 위험과 피할 것:
  - `literalMargin`(133) / `resyncDrainFactor`(144) / `drainChunk`(148) 상수를 건드리지 말 것 — v0.23.4 에서 드레인 도달성이 이 값들에 걸려 한 번 깨졌다.
  - `readLine` 과 `readLineNoReset` 을 "중복이니까" 하나로 합치지 말 것. 데드라인 갱신 여부가 다른 것이 의도이고(주석 489~491), 합치면 IDLE 이 60초마다 끊긴다.
  - IDLE 의 `defer` 드레인 루프(537~546)는 `derr != nil` 에서 빠져나오므로 abandon 이 커넥션을 닫아도 안전하다. 그래도 abandon 뒤 `DONE` 쓰기가 닫힌 소켓에 가는 경로를 한 번 눈으로 확인할 것.
  - `internal/adapters/pop3/` 는 건드리지 말 것 — 브랜치 `auto/2026-09-21-1520`(04b15be)이 같은 파일을 고치며 아직 병합되지 않았다(HEAD 의 조상이 아님, 이번에 `git merge-base --is-ancestor` 로 확인).
  - 보호 경로(auth/oidc, migrations, SecretStore, `.github/workflows`, `internal/transport/spa/assets`)는 이번 과제와 무관하므로 손대지 말 것. 프런트 미변경이면 npm 단계 불필요.
  - 상한 값을 정할 때 `internal/application` 이나 설정 카탈로그에 새 키를 추가하지 말 것 — 상수로 충분하고, 새 설정 키는 SQLite/PostgreSQL 양쪽 스키마·문서까지 번진다.

- 작업량 근거(basis of estimate): 유사 추정 — 직전 회차의 v0.23.5(같은 파일, 같은 픽스처, 파싱+스트리밍 버퍼 교체)가 S~M 로 한 세션에 끝났다. 분해: 헬퍼 1개 + 호출부 2곳 수정(~30줄), 스크립트 서버 1개 + 테스트 2개(~80줄), 수정 전 재현 + 변이 1회, 지정 패키지 race 3회 + 전체 테스트. 범위 30~45분, 8/10 신뢰. **미포함(제외 명시)**: 프런트·PostgreSQL·브라우저 검사, POP3, 설정 키·문서 추가. 컨틴전시는 "수정 전 재현이 안 될 때 차선 후보로 전환" 한 줄로 갈음하고 작업 추정치 안에 패딩을 넣지 않았다.

- 차선 후보: README govulncheck 로컬 예시를 CI 고정 버전에 정렬 — `README.md:303` 이 `@latest`, `.github/workflows/ci.yml` 이 `@v1.6.0` 로 어긋난다(문서만 수정, 코드·워크플로 무변경, 가치 2 / 위험 1 / S). 1순위가 "수정 전 실패" 를 재현하지 못하면 이것으로 전환할 것.
