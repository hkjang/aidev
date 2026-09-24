# 과제서 (정찰, 2026-09-22, base main@5dd0099)

- **과제**: IMAP 어댑터가 과대 리터럴을 거부한 뒤 연결을 프레임 어긋난 채로 남기는 결함 수정 — 거부 후에도 세션이 다음 FETCH 를 정상 처리하거나 명확히 실패하게 (가치 4 / 위험 2 / 작업량 M)

- **왜**: `internal/adapters/imap/client.go:162-164` 의 OOM 가드는 `{n}` 리터럴이 상한을 넘으면 **n 바이트를 소켓에서 읽지 않고** 바로 오류를 반환한다. 호출자(`internal/application/sync.go:262` `ingestOne` 실패 → `stats.Failed++` → `continue`)는 **같은 세션으로 다음 메시지를 계속 가져가므로**, 그 다음 `exec` 는 이전 메시지의 본문 바이트를 프로토콜 줄로 읽는다. 본문에 `{7}` 처럼 `{숫자}` 로 끝나는 줄이 하나라도 있으면 클라이언트가 그만큼을 또 리터럴로 소비해 프레이밍이 어긋나고, `firstLiteral()` 이 **요청하지 않은 메시지의 바이트를 반환**하거나 엉뚱한 오류를 낸다. 고치면 과대 메시지 한 통이 그 뒤 수집 전체의 본문을 오염시키지 못한다.

- **도달 경로(확인함)**: (a) `runBodyRepair` → `fetchRaw`(sync.go:327)는 크기 사전 검사가 전혀 없다. (b) 일반 수집도 `sync.go:258` 의 사전 스킵은 `rm.Size > maxBytes` 조건이라 서버가 `RFC822.SIZE` 를 안 주거나 `reSize` 가 안 걸려 `rm.Size == 0` 이면 그냥 통과해 `Retrieve` 에서 가드가 튄다. (`maxBytes <= 0` 이면 `s.maxLiteral` 도 0 이라 가드 자체가 꺼지므로 그 경우는 해당 없음.)

- **수용 기준**
  1. 과대 리터럴 거부 **뒤에** 같은 세션에서 `Retrieve(ctx, 2)` 를 호출하면 **2번 메시지의 바이트를 정확히 반환**하거나 **명시적 오류**를 반환한다 — 다른 메시지의 바이트를 절대 반환하지 않는다.
  2. 과대 리터럴 자체는 **여전히 할당하지 않는다**(기존 `TestIMAPRejectsOversizeLiteral` 가 그대로 통과하고, 4GiB 를 실제로 읽으려 하지 않아 60초 데드라인까지 가지 않는다 — 아래 "함정" 참조).
  3. 수정 전에는 실패하고 수정 후 통과하는 테스트가 있다. 테스트는 실제 `net.Listen` 스크립트 서버(기존 `fakeServer`/`oversizeServer` 패턴)로 실제 `Dialer{}.Dial` → `Retrieve` 를 통과시키며, 1번 메시지는 상한 초과 리터럴을 **선언하고 실제로 그만큼 보내되 본문 중 한 줄이 `{7}` 로 끝나게** 하고, 2번 메시지는 정상 본문을 보낸다. 수정 전 `Retrieve(2)` 가 2번 본문과 다른 바이트를 돌려주는 것을 실제로 확인한 뒤(재현 로그를 회차 노트에 남길 것) 수정한다.
  4. 기존 `TestIMAPEnumerateAndFetch` / `TestIMAPAuthError` / `internal/application/imap_sync_test.go` 가 무수정으로 통과한다.

- **권장 구현(이대로 아니어도 되나 계약은 지킬 것)**
  - `session` 에 `broken bool` 추가. `exec` 진입 시 `broken` 이면 즉시 영구 오류 반환(프레임이 깨진 연결로 더 이상 명령을 보내지 않는다).
  - 가드가 튀면: **드레인 가능한 크기**(예: `n <= s.maxLiteral*4` 또는 상수 256MiB 같은 명시적 상한)일 때만 `io.CopyN(io.Discard, s.r, int64(n))` 로 버리고(버퍼 할당 금지), 오류를 `oversize` 변수에 보관한 뒤 **루프를 계속 돌아 태그 완료 줄까지 정상 소비**하고 나서 `untagged, oversize` 를 반환한다 → 세션이 다음 명령에 재사용 가능.
  - 드레인 상한을 넘으면 드레인하지 않고 `s.broken = true` 로 표시한 뒤 **즉시** 오류 반환(악의/고장 서버가 세션을 붙잡지 못하게).
  - `Close`/`Quit` 는 `broken` 여부와 무관하게 지금처럼 동작해야 한다.

- **건드릴 파일**
  - `internal/adapters/imap/client.go:106-186` — `session` 구조체(`broken` 필드), `exec` 의 리터럴 가드 분기와 태그 완료 처리. 그 외 함수는 건드리지 말 것.
  - `internal/adapters/imap/client_test.go` — 새 스크립트 서버(예: `desyncServer`)와 새 테스트 1~2개 추가. 기존 `fakeServer`/`oversizeServer`/기존 테스트는 **수정하지 말고 추가만** 할 것.

- **검증 명령** (worktree 루트 `/home/hkjang/.cache/auto-improve-wt/postra` 에서)
  - `go test -race -count=3 ./internal/adapters/imap/`  (base 에서 `go test -count=1 ./internal/adapters/imap/` 이 0.008s 로 통과함을 이번에 확인)
  - `go test -race -count=1 ./internal/application/ -run 'IMAP|Sync'`
  - `gofmt -l ./cmd ./internal`  (빈 출력이어야 함)
  - `go build ./... && go vet ./...`
  - `go test -race ./...`  (전체는 수 분)
  - `make lint`  (lint-format + gosec v2.28.0 medium; Issues 0 유지)
  - `go run ./cmd/postra-contracts -check`  (어댑터 내부 변경이라 계약은 불변이어야 함)

- **위험과 피할 것**
  - **함정 1(중요)**: 기존 `oversizeServer` 는 4GiB 를 **선언만 하고 아무것도 보내지 않는다**. 무조건 드레인하는 구현을 넣으면 그 테스트가 60초 데드라인까지 매달려 사실상 깨진다 — 드레인 상한 분기가 반드시 필요하다.
  - **함정 2**: 드레인에 `make([]byte, n)` 을 쓰면 가드의 존재 이유(OOM 방지)가 사라진다. 반드시 `io.CopyN(io.Discard, ...)` 처럼 고정 메모리로.
  - `internal/adapters/pop3/client.go` 는 **절대 건드리지 말 것** — 직전 회차 브랜치(`origin/auto/2026-09-21-2155`, POP3 `MaxMessageBytes` 상한)가 아직 main 에 병합되지 않았고 같은 파일·같은 영역이라 충돌한다. (현 base 의 pop3 에는 `MaxMessageBytes` 가 없고 테스트 파일도 없음을 확인함.)
  - `s.maxLiteral == 0`(무제한) 일 때 동작이 바뀌면 안 된다 — 가드가 꺼진 경로는 기존 그대로.
  - 보호 경로(`application/oidc*.go`, `httpapi/browser_auth.go`, migrations, SecretStore/KEK, `.github/workflows`, `internal/transport/spa/assets`)는 이번 과제와 무관하니 손대지 말 것. 프런트 변경이 없으므로 npm 단계도 불필요.
  - 운영자 규칙: 손으로 만든 대역이 아니라 **실제 TCP·실제 `Dialer{}.Dial`** 로 증명할 것. 그리고 "동작이 실제로 바뀌지 않는 수정" 은 넣지 말 것 — 수정 전 실패를 눈으로 확인하고 기록할 것.

- **차선 후보**
  1. **README govulncheck 로컬 예시를 CI 고정 버전에 정렬** (가치 2 / 위험 1 / 작업량 S) — `README.md:303` 은 `go run golang.org/x/vuln/cmd/govulncheck@latest ./...`, `.github/workflows/ci.yml:173` 은 `@v1.6.0`. README:296 이 "보안 스캐너도 고정 버전으로 설치해 재현성을 확보" 라고 적어 놓고 예시만 `@latest` 라 문서가 자기 모순이다. 문서 한 줄만 고치고 워크플로는 건드리지 않는다. 검증: `gofmt -l ./cmd ./internal` 빈 출력 + `git diff --check`.
  2. (3순위, 1·2가 모두 성립하지 않을 때만) `imap.session.readLine` 의 `bufio.Reader.ReadString('\n')` 은 개행 없는 줄을 무한히 버퍼링한다 — 고장/악의 서버가 60초 데드라인 안에 보내는 만큼 메모리를 먹는다. 단, 관찰 가능한 실패를 실제 TCP 로 먼저 재현하지 못하면 착수하지 말 것(효과 없는 변경 금지 규칙).

- **미확인 사항(추측으로 적지 않음)**: `{7}` 로 끝나는 본문 줄이 만드는 구체적인 오염 형태(엉뚱한 메시지 바이트 반환 vs. 오류)는 코드 독해로 도출한 것이며 이번 정찰에서 실제 서버로 재현하지는 **않았다**(정찰은 코드를 바꾸지 않으므로 테스트를 새로 쓰지 않았다). 구현자는 수용 기준 3 대로 **먼저 재현부터** 하고, 재현 형태가 다르면 단언을 실제 관찰에 맞춰 조정할 것 — 단, "거부 뒤 다른 메시지 바이트 반환 금지" 라는 계약 자체는 유지한다.
