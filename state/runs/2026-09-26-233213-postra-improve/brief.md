- 과제: IMAP `exec` 의 응답 누적에 상한을 두어, 태그 완료를 보내지 않는 서버가 메모리를 무제한으로 키우지 못하게 한다 (가치 3 / 위험 2 / 작업량 M)
- 왜: v0.23.6 이 줄 **하나**의 길이를 `maxLineBytes`(1 MiB)로 묶었지만, `internal/adapters/imap/client.go:219` 의 `exec` 루프는 줄 **개수**에 상한이 없고 매 줄마다 `readLine()`(:162-165)이 `s.deadline()` 으로 명령 데드라인을 갱신하므로, 짧은 untagged 줄만 계속 흘리고 태그 완료를 보내지 않는 서버에 대해 루프가 영원히 돌며 `untagged`(:214)와 `s.literals`(:257)가 끝없이 자란다. 같은 파일 :375-377 주석이 "단일 FETCH 1:N 의 전체 메타데이터 버퍼링이 OOM(파드 재시작)의 주요 원인이었다" 고 적어 두었듯 이 어댑터의 무제한 버퍼링은 이미 실제로 파드를 죽인 적이 있고, `enumerateBatch` 는 **요청** 크기만 묶을 뿐 서버 **응답** 크기는 묶지 못한다. 상한을 두면 이 계열(리터럴 길이 → 리터럴 스트리밍 → 줄 길이)의 마지막 구멍이 닫힌다.
- 수용 기준:
  1) 실제 `net.Listen` 스크립트 서버 + 실제 `Dialer`(기존 `dial`/`dialMax` 헬퍼)로, 태그 완료를 절대 보내지 않고 짧은 untagged 줄을 계속 흘리는 서버에 대해 `exec` 경로(예: `UIDL`/`List` → `ensureIndex` 의 FETCH, 또는 `ListMailboxes`)가 **수 초 안에** 오류로 끝난다. 수정 전에는 같은 테스트가 매달리는 것(명령 데드라인 60초 이상 또는 `-timeout` 초과)을 먼저 확인해 기록할 것.
  2) 반환 오류가 `errors.Is(err, errUnframed)` 이고, 같은 세션의 다음 명령도 `errUnframed` 로 즉시 실패한다(`s.broken` 경로 — 기존 `abandon` 재사용, 새 실패 모드를 만들지 말 것).
  3) 리터럴 개수도 묶인다: `FETCH 1 (BODY.PEEK[])` 한 번에 작은 리터럴을 수없이 돌려주는 서버가 `s.literals` 를 무제한으로 키우지 못한다(줄 바이트 상한만으로는 못 막는다 — 리터럴 한 개당 줄 텍스트는 30바이트 남짓이다).
  4) 정상 경로 회귀 없음: 기존 IMAP 테스트 12개가 무수정 통과하고, `enumerateBatch`(2000) 만큼의 `* n FETCH (UID .. RFC822.SIZE ..)` 줄을 돌려주는 정상 응답과 상한이 꺼진 계정(`dialMax(...,0)`)의 정상 대용량 본문 1개 수신이 상한에 걸리지 않는다.
  5) 변이 검증: 새 상한 검사를 지우면 새 테스트만 실패하고 나머지는 통과함을 확인한 뒤 원복.
- 건드릴 파일 (프로덕션 2개 — 그 이상으로 번지면 쪼갤 것):
  - `internal/adapters/imap/client.go` — 상단 상수 블록(:130-157, `literalMargin`/`resyncDrainFactor`/`drainChunk`/`maxLineBytes` 옆)에 근거 주석을 단 새 상수 2개 추가: 한 응답의 untagged 프로토콜 텍스트 총량 상한(제안 8 MiB — `enumerateBatch`=2000줄 × 최악 1 MiB/줄 이론값보다 작지만, 실제 FETCH 메타데이터 줄은 100바이트 안팎이라 2000줄 ≈ 200 KB 로 40배 여유. 값은 구현자가 실측으로 조정하되 근거를 주석에 남길 것)과 한 응답이 받을 수 있는 리터럴 **개수** 상한(제안 64 — `Retrieve`/`Top` 은 `firstLiteral`(:445) 로 1개만 쓰고 `ensureIndex` 는 리터럴을 쓰지 않으므로 정상값은 사실상 1).
  - `internal/adapters/imap/client.go:203 exec` — 루프 안에서 누적 바이트(`untagged` 에 담는 줄과 리터럴 루프의 `cont` 증가분)와 리터럴 개수를 세고, 상한을 넘으면 `return nil, s.abandon(fmt.Errorf("%w: ...", errUnframed, ...))`. **리터럴 바이트 총량으로 묶지 말 것** — `refusalThreshold()`(:278)가 0(무제한)인 계정에서 정상 대용량 본문을 깨뜨린다.
  - `internal/adapters/imap/client_test.go` — 기존 관례대로 손으로 만든 대역이 아니라 `net.Listen` 스크립트 서버 + 실제 `Dialer` 로 테스트 3개: 무한 untagged, 리터럴 개수 폭주, 정상 2000줄 배치 무회귀. 베낄 만한 기존 헬퍼(이번 회차에 실제로 열어 확인): `fakeServer`(:25), `longLineServer(t, untagged)`(:530 — 한 줄을 조립해 흘리는 스크립트 서버, 무한 줄 버전으로 고치기 쉬움), `declaredLiteralServer`(:328), `dial`(:608)/`dialMax`(:613), 할당 측정은 `TestIMAPLiteralBufferTracksDeliveredBytes`(:392)의 `runtime.MemStats.TotalAlloc` 패턴.
  - 문서·설정 키·`sync.go`·`internal/application`·`pop3`·워크플로는 건드리지 말 것.
- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `go test -race -count=3 ./internal/adapters/imap/`
  - `go build ./... && go vet ./...`
  - `go test -race -count=1 ./...` (수 분, application 패키지가 가장 김)
  - `make lint` (gofmt -l ./cmd ./internal + gosec v2.28.0 -severity medium -exclude-dir=scripts ./...)
  - `go run ./cmd/postra-contracts -check`, `git diff --check`, `git status --porcelain internal/transport/spa/assets`(비어 있어야 함 — 프런트 미변경)
- 위험과 피할 것:
  - 상한을 너무 낮게 잡아 정상 대용량 메일박스 열거(`ensureIndex` 의 2000줄 배치)를 깨뜨리는 것이 이 과제의 유일한 실제 위험이다. 수용 기준 4의 정상 응답 테스트를 **먼저** 쓰고 상한을 정할 것.
  - `literalMargin`/`resyncDrainFactor`/`drainChunk`/`maxLineBytes` 값은 건드리지 말 것 — 드레인 도달성과 줄 상한이 이 값들의 관계에 걸려 있다(:133-157 주석).
  - `readLine`(:162)과 `readLineNoReset`(:532)의 데드라인 정책 차이는 **의도**다(주석 :527-531). 합치면 IDLE 의 28분 창이 60초로 줄어든다. 이번 변경은 `exec` 안에서만 세고, 두 함수의 시그니처·정책을 바꾸지 말 것.
  - `Idle`(:542)은 줄을 누적하지 않고 28분 **절대** 데드라인(갱신 없음)에서 돈다 — 같은 결함이 아니다. 이번 범위에 넣지 말 것.
  - 거부된 리터럴 경로(`discardLiteral`)와 `refused` 의 "이 명령만 실패시키고 프레임은 유지" 계약을 깨지 말 것. 새 상한은 프레임을 복구할 수 없는 경우이므로 `refused` 가 아니라 `abandon` 이 맞다.
  - 보호 경로(auth/session/migrations/.github/workflows/spa/assets)는 이번 과제와 무관하다.
- 차선 후보: `sync.max_message_bytes` 의 0·음수 = 무제한 규약을 문서화 (가치 2 / 위험 1 / S) — 어댑터 `refusalThreshold()`(client.go:278)는 `<=0` 에서 0(=상한 없음)을 돌려주는데, 이 규약을 적어 둔 곳은 `docs/releases/v0.23.5.md:30` 한 곳뿐이고 설정 카탈로그 설명(`internal/application/settings_catalog.go:64` "메일 한 건 최대 크기(bytes)")에도 관리자 문서에도 없다(이번 회차에 `grep -rn max_message_bytes` 로 확인). 문서·카탈로그 문자열만 고치는 저위험 항목. 설정 검증 로직을 바꾸지는 말 것(동작 변경이 되고 위험이 올라간다). 1순위가 성립하지 않을 때(= 수정 전 매달림을 실제 TCP 로 재현하지 못했을 때)만 고를 것.
  - **고르지 말 것**: POP3 본문 상한(미병합 브랜치 `auto/2026-09-21-1520`/04b15be) — 이번 회차에 PR #22 가 사람에게 반려된 것인지 확인하지 못했다(네트워크 접근 거부, **미확인**). 확인 없이 같은 접근을 다시 제출하면 운영자 규칙(사람이 반려한 접근 재제출 금지)에 걸릴 수 있다.
