# 과제서 (정찰, 2026-09-23, base main@5dd0099)

## 0. 우선 과제(오류 대응)에 대한 정찰 판정 — 먼저 읽을 것

이번 회차에 "릴리즈 워크플로가 같은 이유로 두 번 실패했다 / 마지막 회차가 error: interrupted before PR 로 끝났다" 가 자동 배정됐다. 실제 기록을 읽어 확인한 것은 다음과 같다.

- 직전 회차(`2026-09-22-234439-postra-improve`) `stages.json`: `scout: failed — "과제서 없음 — 구현자가 직접 고른다"`(23:48:42) → `resume: closed — "PR 이전 단계에서 중단"`(00:00:23). 그런데 그 회차 디렉터리에는 **완성된 `brief.md` 가 실제로 존재한다**(IMAP 과대 리터럴 과제, 47줄). 즉 실패한 것은 저장소 코드가 아니라 **러너가 정해진 시각에 과제서를 못 본 것**이다.
- 그 앞 릴리즈 실패(`2026-09-21-042417-postra-improve`) `stages.json`: `merge: done` 까지 정상이고 마지막 `release: missing — "릴리즈 결과 없음/손상: missing"`. 이것도 **aidev 러너의 release 에이전트 단계**이지 `.github/workflows/release.yml` 잡이 아니다(그 회차 PR #19 의 `ci: passed — 검사 10개 모두 success`).
- 즉 두 번 반복된 실패 지점은 **러너 파이프라인(aidev)** 이며, 이 저장소 바깥이고 정찰·구현의 쓰기 범위 밖이다. 러너를 고치는 것은 이번 과제가 될 수 없다.
- **미확인**: GitHub Actions `Release` 워크플로가 v0.23.2·v0.23.3 태그에서 실제로 실패했는지는 **확인하지 못했다**. 이 환경에서 네트워크(curl·WebFetch)가 모두 차단돼 Actions API 를 읽을 수 없었다. 로컬에서 확인한 것은 태그 `v0.23.2`(b21804e)·`v0.23.3`(5dd0099) 가 둘 다 main 의 머지 커밋이고 `docs/releases/v0.23.2.md`·`v0.23.3.md` 가 둘 다 트리에 있다는 것뿐이다 — 즉 `release.yml` 의 notes_file 분기는 정상 성립한다. `release.yml` 자체에서 재현 가능한 결함은 찾지 못했다.
- **따라서 구현자에게**: `.github/workflows/` 는 이번에도 건드리지 말 것(근거 없는 워크플로 수정 금지, 느슨하게 만드는 것은 금지). 대신 **직전 회차가 PR 전에 끊겨 끝내 구현되지 않은 과제**를 이어서 끝낸다. 그 과제는 아래와 같고, 근거는 이번 정찰이 코드로 다시 확인했다. 사람이 반려한 적이 없는 신규 과제다.

---

- **과제**: IMAP 어댑터가 과대 리터럴을 거부한 뒤 세션을 프레임 어긋난 채로 남기는 결함 수정 (가치 4 / 위험 2 / 작업량 M)

- **왜**: `internal/adapters/imap/client.go:162-164` 의 OOM 가드는 `{n}` 리터럴이 상한을 넘으면 **n 바이트를 소켓에서 읽지 않고** 그대로 오류를 반환한다(이번에 직접 읽어 확인). 호출자는 `internal/application/sync.go:264` 에서 `continue` 하며 **같은 세션으로 다음 메시지를 계속 가져가므로**, 다음 `exec` 는 이전 메시지의 본문 바이트를 프로토콜 줄로 읽는다. 고치면 과대 메시지 한 통이 그 뒤 수집 전체의 본문을 오염시키지 못한다.

- **도달 경로(이번에 코드로 확인)**: `sync.go:258` 의 사전 스킵은 `maxBytes > 0 && rm.Size > maxBytes` 조건이라 서버가 `RFC822.SIZE` 를 안 주거나 파싱이 안 돼 `rm.Size == 0` 이면 그냥 통과해 `Retrieve`(sync.go:328/353) 에서 가드가 튄다. `runBodyRepair` 경로(`fetchRaw`, sync.go:328)는 크기 사전 검사가 아예 없다. (`maxBytes <= 0` 이면 `s.maxLiteral` 도 0 이라 가드가 꺼지므로 해당 없음.)

- **수용 기준**
  1. 과대 리터럴 거부 **뒤에** 같은 세션에서 `Retrieve(ctx, 2)` 를 호출하면 **2번 메시지의 바이트를 정확히 반환**하거나 **명시적 오류**를 반환한다 — 다른 메시지의 바이트를 절대 반환하지 않는다.
  2. 과대 리터럴 자체는 **여전히 할당하지 않는다**. 기존 `TestIMAPRejectsOversizeLiteral` 이 무수정으로 통과하고, 60초 데드라인까지 매달리지 않는다(아래 함정 1).
  3. 수정 전에는 실패하고 수정 후 통과하는 테스트가 있다. 실제 `net.Listen` 스크립트 서버(기존 `fakeServer`/`oversizeServer` 패턴)로 실제 `Dialer{}.Dial` → `Retrieve` 를 통과시키고, 1번 메시지는 상한 초과 리터럴을 **선언하고 실제로 그만큼 보내되 본문 한 줄이 `{7}` 로 끝나게**, 2번 메시지는 정상 본문을 보낸다. **수정 전 실패를 눈으로 확인하고 그 출력을 회차 노트에 남긴다.**
  4. 기존 `TestIMAPEnumerateAndFetch` / `TestIMAPAuthError` / `internal/application/imap_sync_test.go` 가 무수정으로 통과한다.

- **권장 구현(계약만 지키면 형태는 달라도 됨)**
  - `session`(client.go:106-118)에 `broken bool` 추가. `exec` 진입 시 `broken` 이면 즉시 영구 오류 반환(프레임이 깨진 연결로 더는 명령을 보내지 않는다).
  - 가드가 튀면 **드레인 가능한 크기**(예: 명시적 상한 256MiB 또는 `s.maxLiteral*4`)일 때만 `io.CopyN(io.Discard, s.r, int64(n))` 로 버리고, 오류를 변수에 보관한 채 **태그 완료 줄까지 정상 소비**한 뒤 `untagged, oversizeErr` 를 반환한다 → 세션 재사용 가능.
  - 드레인 상한을 넘으면 드레인하지 않고 `s.broken = true` 로 표시한 뒤 **즉시** 오류 반환(고장·악의 서버가 세션을 붙잡지 못하게).
  - `Close`/`Quit` 는 `broken` 여부와 무관하게 지금처럼 동작할 것.

- **건드릴 파일**
  - `internal/adapters/imap/client.go:106-186` — `session` 구조체에 `broken`, `exec` 의 리터럴 가드 분기와 태그 완료 처리. 그 외 함수는 건드리지 말 것.
  - `internal/adapters/imap/client_test.go` — 새 스크립트 서버(예: `desyncServer`)와 테스트 1~2개 **추가만**. 기존 `fakeServer`/`oversizeServer`/기존 테스트는 수정하지 말 것.

- **검증 명령** (worktree 루트 `/home/hkjang/.cache/auto-improve-wt/postra` 에서)
  - `go test -race -count=3 ./internal/adapters/imap/`
  - `go test -race -count=1 ./internal/application/ -run 'IMAP|Sync'`
  - `gofmt -l ./cmd ./internal` (빈 출력이어야 함)
  - `go build ./... && go vet ./...`
  - `go test -race ./...` (전체는 수 분)
  - `make lint` (lint-format + gosec v2.28.0 medium; Issues 0 유지)
  - `go run ./cmd/postra-contracts -check` (어댑터 내부 변경이라 계약 불변이어야 함)

- **위험과 피할 것**
  - **함정 1(중요)**: 기존 `oversizeServer` 는 4GiB 를 **선언만 하고 아무것도 보내지 않는다**. 무조건 드레인하는 구현을 넣으면 그 테스트가 60초 데드라인까지 매달린다 — 드레인 상한 분기가 반드시 필요하다.
  - **함정 2**: 드레인에 `make([]byte, n)` 을 쓰면 가드의 존재 이유(OOM 방지)가 사라진다. 반드시 `io.CopyN(io.Discard, ...)` 처럼 고정 메모리로.
  - `internal/adapters/pop3/client.go` 는 **절대 건드리지 말 것** — 미병합 브랜치 `origin/auto/2026-09-21-2155`(04b15be, POP3 `MaxMessageBytes`)가 같은 파일을 고치고 있어 충돌한다. 이번 base 의 pop3 에는 그 변경이 없음을 확인했다.
  - `s.maxLiteral == 0`(무제한) 경로의 동작이 바뀌면 안 된다.
  - 보호 경로(`.github/workflows/*`, `application/oidc*.go`, `httpapi/browser_auth.go`, migrations, SecretStore/KEK, `internal/transport/spa/assets`)는 손대지 말 것. 프런트 변경이 없으므로 npm 단계도 불필요.
  - 운영자 규칙: 손으로 만든 대역이 아니라 **실제 TCP·실제 `Dialer{}.Dial`** 로 증명할 것. "실제 동작이 바뀌지 않는 수정" 금지 — 수정 전 실패를 반드시 먼저 재현할 것.

- **차선 후보**
  1. **README govulncheck 로컬 예시를 CI 고정 버전에 정렬** (가치 2 / 위험 1 / 작업량 S) — `README.md:303` 의 `@latest` 와 `.github/workflows/ci.yml:173` 의 `@v1.6.0` 불일치. README 가 "보안 스캐너도 고정 버전으로 설치" 라고 적어 놓고 예시만 `@latest` 라 자기 모순이다. 문서 한 줄만 고치고 워크플로는 건드리지 않는다. 검증: `gofmt -l ./cmd ./internal` + `git diff --check`.
  2. (3순위) `imap.session.readLine` 의 `bufio.Reader.ReadString('\n')` 은 개행 없는 줄을 무한히 버퍼링한다 — 실제 TCP 로 관찰 가능한 실패를 먼저 재현하지 못하면 착수 금지.

- **미확인 사항(추측으로 적지 않음)**
  - `{7}` 로 끝나는 본문 줄이 만드는 **구체적 오염 형태**(엉뚱한 메시지 바이트 반환 vs. 오류)는 코드 독해로 도출했고 이번에도 실제 서버로 재현하지 **않았다**(정찰은 코드를 바꾸지 않는다). 구현자는 수용 기준 3 대로 먼저 재현하고, 형태가 다르면 단언을 실제 관찰에 맞출 것 — "거부 뒤 다른 메시지 바이트 반환 금지" 계약 자체는 유지한다.
  - GitHub Actions `Release` 워크플로의 실제 실패 여부·실패 단계(0절) — 네트워크 차단으로 미확인.
  - 이번 회차에 실행한 명령은 `git log/tag/ls-tree` 읽기와 `grep` 뿐이다. 테스트는 시간·예산 제약으로 돌리지 않았다(직전 회차가 base 동일 `main@5dd0099` 에서 `go test -count=1 ./internal/adapters/imap/` 통과를 기록해 두었다).
