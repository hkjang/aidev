# 과제서 (2026-10-03, mmcp)

- 과제: `internal/logbuf` 에 첫 테스트 — 링 버퍼 래핑·Snapshot 순서·Subscribe 해제 계약·slog.Handler 배선 (가치 3 / 위험 1 / 작업량 S)

- 왜: 관리 콘솔의 로그 뷰어(`/admin/logs`)와 SSE 라이브 테일(`admin_api.go:1523 adminLogStream`)이 전적으로 이 129줄 패키지에 얹혀 있는데 테스트가 0개다(이 base 에서 `go test ./internal/logbuf/` → `[no test files]` 확인). 링 버퍼의 래핑·순서와 구독 해제 후 송신 금지는 눈으로 읽어서는 회귀를 못 잡는 종류라, 테스트를 박아 두면 앞으로 이 파일을 건드릴 때 관리자 로그가 조용히 잘리거나 SSE 가 블로킹되는 사고를 막는다.

- 수용 기준:
  1) `go test -race -count=1 ./internal/logbuf/` 가 통과하고 `[no test files]` 가 아니다.
  2) 링 버퍼: 용량 N 에 N 미만 / 정확히 N / N 초과로 기록했을 때 `Snapshot()` 이 **오래된 것 → 최신 순**으로 최신 N 개만 돌려준다는 것을 증명한다. `Seq` 가 1 부터 빠짐없이 증가하고 래핑 후에도 이어진다는 것(폐기된 것의 Seq 는 사라짐)도 포함.
  3) `Snapshot()` 이 돌려준 슬라이스를 호출자가 수정해도 내부 상태가 오염되지 않는다(`logbuf.go:62` 가 복사본을 돌려주는 계약).
  4) `Subscribe()`: 구독 중에는 새 엔트리가 채널로 오고, 돌려받은 해제 함수를 부른 뒤에는 **더 이상 오지 않는다**. 해제 후 로그를 많이 써도 블로킹·패닉이 없다.
  5) 느린 구독자: 채널 버퍼(256)를 채운 뒤 더 기록해도 `add()` 가 블로킹하지 않고 그냥 버린다(`logbuf.go:47`). 타임아웃 없이 테스트가 끝나는 것으로 증명.
  6) 배선: 손으로 만든 대역이 아니라 **실제 `slog.New(logbuf.NewHandler(buf, w, level))`** 로 검증한다. `*slog.LevelVar` 를 WARN 으로 올리면 INFO 가 버퍼에도 `w` 에도 안 들어가고, 레벨을 런타임에 내리면 다시 들어온다(`Enabled`). `w` 로 넣은 `bytes.Buffer` 에는 JSON 한 줄이 같이 찍힌다(tee 가 살아 있음). `Attrs` 의 `error` 값이 문자열로 평탄화된다(`logbuf.go:109-111`).

- 건드릴 파일:
  - `internal/logbuf/logbuf_test.go` (신규, `package logbuf`) — 위 전부. 프로덕션 코드는 **건드리지 않는다**.
  - 그 외 파일 변경 없음. 파일 1개짜리 과제다.

- 검증 명령:
  - `go test -race -count=1 ./internal/logbuf/` (수 초)
  - `gofmt -l .` 출력이 비어 있어야 함 — 이 저장소 lint 의 첫 관문이고 신규 파일에서 가장 흔한 탈락 사유다
  - `make lint` (= gofmt + `go vet ./...` + `./scripts/verify-version.sh`)
  - 마무리로 `go test ./...` (Postgres 없으면 `internal/server` 통합 테스트는 조용히 skip 된다. 이 과제는 `internal/server` 를 안 건드리므로 `TEST_POSTGRES_DSN` 없이도 충분하다)

- 위험과 피할 것:
  - **프로덕션 코드를 고치지 말 것.** 읽은 범위에서 `logbuf.go` 의 링 버퍼·Subscribe 는 정확하다. 이 과제는 순수 커버리지다. 테스트를 쓰다 "고칠 거리"가 보여도 이번 회차 범위 밖이다(아래 두 건은 알고 남긴 것).
  - **`WithAttrs` → `WithGroup` 순서 조합을 테스트로 못 박지 말 것.** `logbuf.go:96-106` 은 `h.attrs` 를 평탄한 리스트로 쌓고 **모든** attr 에 현재 그룹 접두사를 붙이므로, 그룹 이전에 붙인 attr 까지 접두사를 받는다 — slog 규약(그룹은 그 *이후* attr 만 한정)과 어긋나고, 같은 레코드를 tee 하는 JSON 핸들러(`h.next`)는 중첩 객체로 다르게 쓴다. **코드를 읽고 판단한 것이고 실행으로는 확인하지 않았다(미확인).** 프로덕션에 `WithGroup`/`logger.With(...)` 호출자는 **없다**(`grep` 이 아니라 전체 .go 에서 호출처를 찾아 0건 확인). 테스트는 `WithAttrs` 단독, `WithGroup` 단독까지만 덮고, 뒤섞인 순서는 건드리지 말 것 — 지금 동작을 테스트로 못 박으면 나중에 올바르게 고칠 때 그 테스트가 방해가 된다.
  - **`New(0)` 을 테스트하지 말 것.** `logbuf.go:40` 의 `% len(b.entries)` 가 0 나누기로 패닉한다. 호출자는 `main.go:61` 의 `New(5000)` 과 `integration_test.go:209` 의 `New(100)` 뿐이라 도달 불가능하다. 패닉을 테스트로 못 박으면 나중에 가드를 넣을 때 깨진다.
  - **보호 경로를 건드리지 말 것** — `internal/server/oauth.go`·`oidc.go`·`auth.go`·`identity.go`·`tools.go approve()`·`internal/store/migrations`·`.github/workflows/`·`VERSION`.
  - **지금 떠 있는 다른 작업과 충돌하지 말 것.** 이 base(`b782c5f`)에는 2026-10-02 회차의 두 변경이 **아직 없다** — `internal/mattermost/client_test.go` 는 존재하지 않고(`[no test files]` 확인), `web/package.json:10` 도 여전히 `"test": "vitest run"` 이다. 그 둘(mattermost 테스트, web npm 스크립트)은 다른 회차가 들고 있으니 이번에 손대지 말 것.
  - 경합 테스트에서 `time.Sleep` 으로 동기화하지 말 것 — 채널 수신에 타임아웃을 건 `select` 를 쓰면 `-race` 와 CI 부하에서도 안 흔들린다.

- 작업량 근거(S, 25~45분): 신규 파일 1개·프로덕션 변경 0건이고, 비교 대상은 2026-10-02 의 `internal/mattermost/client_test.go`(같은 성격의 "테스트 0개인 작은 패키지에 첫 테스트", S 로 실측돼 성공)다. 포함: 테스트 작성 + `-race` 실행 + `make lint`. 제외(이번 회차 밖): 프로덕션 코드 수정, `internal/server` 통합 테스트, Postgres 기동. 가장 흔들릴 지점은 구독 해제·느린 구독자 테스트의 동기화이고, 그게 길어지면 수용 기준 5 를 빼고 1~4·6 만으로 끝내도 과제는 성립한다.

- 차선 후보: **`internal/mattermost` 의 남은 경로 테스트** — `FileContent` 의 `*[]byte` 우회(JSON 디코드를 건너뛰는 분기), `PostList.Ordered()` 가 `Posts` 에 없는 id 를 조용히 버리는 것, `New()` 가 깨진 CA PEM 을 거부하는지. 반드시 `httptest.Server` + 실제 `*mattermost.Client` 로 배선할 것(이 저장소 관례는 `integration_test.go:30 fakeMM` 이다). **단, 이 패키지는 2026-10-02 회차의 `client_test.go` 가 머지 대기 중이라 파일 충돌 위험이 있다** — 1순위를 고를 수 없을 때만, 그리고 현재 `main` 에 `client_test.go` 가 들어왔는지 먼저 확인하고 착수할 것.
