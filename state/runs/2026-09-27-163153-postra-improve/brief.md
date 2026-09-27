- 과제: POP3 다중행 응답(LIST/UIDL) 누적에 상한 — `.` 종결을 보내지 않는 서버가 수집 워커를 영원히 붙잡고 메모리를 키우지 못하게 (가치 3 / 위험 2 / 작업량 M)

- 왜: `internal/adapters/pop3/client.go:130 readList()` 은 `.` 줄을 만날 때까지 줄을 무제한으로 `lines []string` 에 모으고, 루프 **안** 에서 `s.deadline()`(:133)이 줄마다 `conn.SetDeadline(now+commandTO)` 로 명령 데드라인을 새로 밀어낸다 — 짧은 줄만 1초에 한 줄씩 계속 흘리는 서버 한 대에 대해 루프가 영원히 돌고 `lines` 가 끝없이 자란다. 데드라인이 끝내 주지 못하는 것이 이 결함의 핵심이며, IMAP `exec` 의 **똑같은 모양**(줄 개수 상한 없음 + `readLine` 의 줄마다 데드라인 갱신)은 v0.23.7 에서 이미 고쳤다. POP3 쪽은 남아 있고, 더 나쁜 위치다: `readList` 는 메시지 루프가 아니라 **열거 단계**(`sync.go:198` UIDL → `:201`/`:208` List)에서 불리므로 한 계정의 동기화가 첫 메시지를 처리하기도 전에 통째로 멈춘다.

- 수용 기준:
  1) `.` 을 보내지 않고 짧은 줄만 계속 흘리는 실제 TCP POP3 서버에 대해, 실제 `pop3.Dialer{}.Dial(...)` 로 얻은 세션의 `UIDL`(또는 `List`)이 **명령 타임아웃(60초)보다 훨씬 짧은 시간 안에** 오류로 돌아온다. 오류 메시지에 상한 초과임이 드러난다(예: "multi-line response exceeds %d bytes").
  2) 상한이 트립하면 응답의 나머지가 소켓에 남아 오프셋을 알 수 없으므로 **세션을 폐기**한다(`s.conn.Close()`). 같은 세션의 다음 명령(`List`/`Retrieve`)도 성공하지 않고 오류로 실패한다 — 남은 응답 바이트를 프로토콜로 오해해 읽어 가지 않는다.
  3) 정상 경로가 바이트 단위로 불변임을 테스트가 증명한다: 수천 건 규모의 정상 `UIDL`/`LIST` 응답(번호 + RFC 1939 최대 길이 70자 UID)이 전수 일치로 파싱되고, **그 응답이 상한에서 차지한 실제 바이트 수를 `t.Logf` 로 남기고 상한이 그것의 N배 이상임을 단언**한다(IMAP v0.23.7 가 `enumerateBatch` 2000줄 = 81,994 바이트를 재고 8배 여유를 단언한 방식 그대로). 상한 값은 이 실측 위에서 정한다.
  4) 빈 줄만 무한히 보내는 서버도 같은 상한에 걸린다 — `textproto.ReadLine` 은 CRLF 를 떼고 돌려주므로 `len(line)` 만 세면 빈 줄은 0바이트로 계산돼 상한을 우회한다. IMAP 이 이 정확한 결함으로 한 번 비평을 받아 `responseLineOverhead`(commit 5145007)를 도입했다. 같은 실수를 반복하지 말 것.
  5) 변이 검증: 상한 검사를 지우면 새 테스트만 실패(타임아웃)하고 나머지는 통과함을 확인한 뒤 원복.

- 건드릴 파일 (프로덕션 1개):
  - `internal/adapters/pop3/client.go:130 readList()` — 누적 바이트를 세어 상한 초과 시 `s.conn.Close()` 후 오류 반환. 줄마다 `len(line) + <overhead>` 를 charge 한다. IMAP 선례를 그대로 옮길 것: 상수 2개를 근거 주석과 함께 파일 상단에 추가 — `maxListBytes`(값은 수용 기준 3의 실측 위에서 정하되, POP3 maildrop 은 크다: UIDL 한 줄이 번호+공백+최대 70자 UID ≈ 79바이트 + overhead ≈ 97바이트/메시지이므로 **8 MiB 는 8만 건 남짓에서 정상 maildrop 을 깨뜨릴 수 있다 — 32 MiB(≈34만 건) 이상을 권한다**), `listLineOverhead`(IMAP 의 `responseLineOverhead = 2 + 16` 과 같은 근거: 떼어낸 CRLF 2바이트 + 보관되는 문자열 헤더 16바이트).
  - `internal/adapters/pop3/client_test.go` — **신규 파일. 이 패키지에는 테스트가 하나도 없다**(`go test ./internal/adapters/pop3/` → `[no test files]`, 이번 회차에 실행해 확인). 손으로 만든 대역을 쓰지 말고 `internal/adapters/smtp/client_test.go:107 fakeRelay`(`net.Listen("tcp","127.0.0.1:0")` + `t.Cleanup(ln.Close)`) 와 `internal/adapters/imap/client_test.go` 의 스크립트 서버 패턴을 이식해, 127.0.0.1 에서 `+OK` 인사 → `UIDL` 수신 → 지정된 줄을 흘리는 서버를 세우고 **실제 `pop3.Dialer{}.Dial`** 로 접속한다(`domain.POP3DialOptions{Host:"127.0.0.1", Port:port, Security: domain.SecurityNone}`, 인증은 `Username:""` 로 건너뛸 수 있다 — client.go:78 이 빈 사용자명이면 USER/PASS 를 생략한다).

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `go test -race -count=3 ./internal/adapters/pop3/`
  - `go build ./... && go vet ./...`
  - `go test -race -count=1 ./...` (수 분, `internal/application` 이 가장 김)
  - `make lint` (gofmt -l ./cmd ./internal + gosec v2.28.0 -severity medium -exclude-dir=scripts ./...)
  - `go run ./cmd/postra-contracts -check`
  - `git diff --check`
  - 프런트엔드 미변경이므로 `git status --porcelain internal/transport/spa/assets` 가 비어 있음을 확인(재빌드 불필요).

- 위험과 피할 것:
  - **`retrBody()`(:188)를 건드리지 말 것.** 본문 수신에 `MaxMessageBytes` 상한을 넣는 구현은 미병합 브랜치 `auto/2026-09-21-1520`(04b15be, "fix(pop3): bound message bodies while receiving")에 이미 있고 그 PR #22 가 사람에게 반려된 것인지 **미확인**(샌드박스가 GitHub API 를 거부). 같은 접근을 다시 제출하는 것은 운영자가 명시적으로 금지한 항목이다. 이번 과제는 `readList` 만 다루므로 그 브랜치와 겹치는 줄이 없다 — 이 분리를 유지할 것.
  - **한 줄이 끝없이 이어지는 경우(`\n` 미전송)는 이번 범위 밖.** `textproto.ReadLine` → `readLineSlice` 는 조각을 무제한 `append` 하지만, `s.deadline()` 이 `ReadLine` **앞**에서만 불리므로 그 경로는 60초 명령 데드라인에 묶인다 — 시간이 유한하므로 이번 결함(시간이 무한)보다 엄격히 덜 심각하다. 여기까지 넓히면 `readResponse`/`retrBody` 까지 textproto 를 대체해야 하고 위 브랜치와 충돌한다. 별도 아이디어로 남겼다.
  - `s.deadline()` 을 루프 밖으로 옮겨 해결하려 하지 말 것 — 큰 정상 maildrop 의 UIDL 이 60초를 넘길 수 있어 동작 회귀가 된다. IMAP 이 같은 판단을 했다(데드라인 정책은 그대로 두고 누적에 상한).
  - 폐기 후 동작 확인: `sync.go:198` 은 `UIDL` 실패 시 `List`(:201)로 폴백하고 그것도 실패하면 `providerListFailed` 로 깔끔히 끝난다. 반대로 UIDL 은 성공했고 `:208` 의 List 가 상한에 걸리는 경우 `lerr` 는 **조용히 무시**되고(사이즈 병합만 건너뜀) 세션은 이미 닫혀 있어 이후 모든 `Retrieve` 가 실패해 전건 Failed 로 집계된다. 이것은 기존 `sync.go` 의 설계이므로 **이번에 `sync.go`·`internal/application` 을 고치지 말 것**(별도 보류 아이디어 "세션 폐기 뒤 sync 루프 조기 종료"). 다만 과제 노트에 관찰한 그대로 남길 것.
  - 보호 경로 회피: `internal/application/oidc*.go`, `httpapi/browser_auth.go`, DB migrations, SecretStore/KEK, `.github/workflows`, `internal/transport/spa/assets` — 어느 것도 이번 변경에 필요 없다.
  - 설정 키를 새로 만들지 말 것. 상한은 컴파일 상수로 두는 것이 IMAP 선례와 일치한다.

- 차선 후보: `sync.max_message_bytes` 의 0·음수 = 무제한 규약을 설정 카탈로그·관리자 문서에 명시 (가치 2 / 위험 1 / 작업량 S) — `internal/application/settings_catalog.go:64` 설명과 관리자 문서에 이 규약이 없고, 적힌 곳은 `docs/releases/v0.23.5.md:30` 한 곳뿐이다. 검증 로직은 손대지 말 것(동작 변경). 4~5회 연속 차선으로만 남아 있어, 1순위가 성립하지 않으면 이번에 처리할 만하다.
