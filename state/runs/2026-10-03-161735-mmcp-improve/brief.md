- 과제: `logbuf.Handler` 의 `WithGroup` 접두사가 그룹을 열기 **전에** 붙인 attr 까지 소급 적용돼 관리 콘솔 로그와 JSON tee 가 같은 레코드를 다르게 보여준다 (+ `New(0)` 0나누기 가드) (가치 3 / 위험 2 / 작업량 S)

- 왜: `internal/logbuf/logbuf.go:94 Handle()` 이 `strings.Join(h.groups,".")` 하나를 `h.attrs` **전체**와 레코드 attr 에 똑같이 붙인다. 그런데 `h.attrs` 는 `WithAttrs` 가 평탄하게 쌓은 리스트라 그룹을 열기 전에 붙인 attr 까지 접두사를 먹는다 — slog 규약은 그룹이 '이후' attr 만 한정하는 것이고, 같은 레코드를 받는 `h.next`(표준 `slog.JSONHandler`)는 규약대로 쓴다. 즉 **같은 값을 읽는 경로가 둘인데 답이 다르다**(프로필의 알려진 위험 패턴). 고치면 `/admin/logs` 뷰어·SSE 라이브 테일과 stdout JSON 이 같은 키를 보여주고, 129줄 패키지에 첫 테스트가 들어온다.

- **이번 정찰에서 실제로 재현했다** (base `b782c5f`, 임시 테스트를 넣고 돌린 뒤 삭제 — `git status --porcelain` 빈 출력 확인):
  - `lg.With(slog.String("outer","o")).WithGroup("mm").Info("m", slog.String("user_id","u1"))` →
    `buffer attrs = map[mm.outer:o mm.user_id:u1]`
    `tee json = {... ,"msg":"m","outer":"o","mm":{"user_id":"u1"}}`
    → 버퍼는 `outer` 를 `mm` 안으로 잘못 넣고, tee 는 밖에 둔다. 두 경로가 갈린다.
  - `New(0)` + 로그 1건 → `panic: runtime error: index out of range [0] with length 0` at **`logbuf.go:39`** (`b.entries[b.next] = e`). 40줄 모듈로가 아니라 **39줄 인덱싱에서 먼저 터진다** — 가드는 39줄 앞에 와야 한다.
  - **다만 솔직히 적는다: 지금 프로덕션에 `WithGroup`/`logger.With` 호출처가 0건이고(`grep -rn "WithGroup\|logger\.With\|\.With(" cmd/ internal/ --include=*.go` 에서 `logbuf.go` 와 테스트를 빼면 결과 없음) `New()` 호출처는 `main.go:61 New(5000)` 과 `integration_test.go:209 New(100)` 뿐이다.** 즉 둘 다 **잠재 결함**이고 오늘 사용자가 겪는 장애가 아니다. 가치를 3 으로 둔 이유도 이것이다 — 회귀 위험이 사실상 0 이고(깨질 호출처가 없다), 관리 콘솔 로그가 전적으로 얹힌 패키지에 테스트가 0개라는 쪽이 본 가치다.

- 수용 기준:
  1) `slog.New(logbuf.NewHandler(buf, w, lv))` 로 만든 로거에서 `logger.With(slog.String("outer","o")).WithGroup("mm").Info("m", slog.String("user_id","u1"))` 를 쓰면 `Snapshot()[0].Attrs` 가 `{"outer":"o","mm.user_id":"u1"}` 이다(현재는 `{"mm.outer":"o","mm.user_id":"u1"}`). 그룹 밖 attr 에 접두사가 붙지 않는 것이 핵심.
  2) 같은 호출의 `w`(JSON tee) 한 줄을 `json.Unmarshal` 해서 **top-level `outer` 가 있고** 중첩 `mm` 객체 안에 `user_id` 가 있음을 확인 — 두 경로가 같은 attr 을 같은 소속으로 읽는다는 교차 검증(기준 1 만으로는 tee 와의 일치를 증명하지 못한다).
  3) 중첩 그룹(`WithGroup("a").With(...).WithGroup("b").Info(...)`)에서 각 attr 이 **붙은 시점의** 그룹 경로만 갖는다(`a.x`, `a.b.y`). 그룹을 전혀 안 쓰면 키가 그대로다(회귀 없음).
  4) `logbuf.New(0)` 에 로그를 써도 패닉하지 않는다(현재 `logbuf.go:40` 의 `% len(b.entries)` 가 0나누기로 패닉). `Snapshot()` 은 빈 슬라이스, `Capacity()==0`, `Subscribe()` 구독자에게는 여전히 전달된다.
  5) `go test -race -count=1 ./internal/logbuf/` 통과, `gofmt -l .` 빈 출력.

- 건드릴 파일:
  - `internal/logbuf/logbuf.go:117 WithAttrs` — attr 을 **받는 시점에** 현재 `h.groups` 로 키를 한정해서 저장하도록(`prefix := strings.Join(h.groups,".")`, `for _, a := range as { if prefix != "" { a.Key = prefix+"."+a.Key }; ... }`). `range` 는 구조체를 복사하므로 호출자의 `as` 원소를 건드리지 않는다. `c.next = h.next.WithAttrs(as)` 는 **원본 `as`** 를 그대로 넘겨야 한다(표준 핸들러가 자기 규약대로 처리).
  - `internal/logbuf/logbuf.go:94 Handle` — `h.attrs` 는 이미 한정됐으니 접두사 없이 넣고, `r.Attrs(...)` 쪽만 `prefix` 를 붙이도록 `put` 을 두 갈래로 나눈다(또는 `put(a, prefix)` 꼴).
  - `internal/logbuf/logbuf.go:35 add` — `b.seq++` / `e.Seq = b.seq` 뒤, **39줄의 `b.entries[b.next] = e` 앞에** `if len(b.entries) > 0 { ... }` 가드를 둔다(구독자 전달 루프와 `b.seq++` 는 가드 밖에 유지). 재현 시 패닉이 39줄에서 났으므로 40줄만 고치면 안 된다.
  - `internal/logbuf/logbuf_test.go` (신규) — 위 기준. 핸들러는 전부 `main.go:61-62` 와 같은 실제 `slog.New(logbuf.NewHandler(buf, &bytes.Buffer{}, lv))` 로 배선할 것. 손으로 만든 `slog.Handler` 대역은 이 저장소 관례가 아니다.
  - **이 네 개로 끝낼 것.** 프로덕션 파일은 `logbuf.go` 한 개다.

- 검증 명령:
  - `go test -race -count=1 ./internal/logbuf/` (수초)
  - `gofmt -l .` → 빈 출력, `go vet ./...`
  - `go test -race -count=1 ./...` (Postgres 없으면 `internal/server` 통합 테스트는 조용히 skip — 그래도 `logbuf` 는 돈다)
  - `make lint` (`gofmt` + `go vet` + `./scripts/verify-version.sh`)
  - **`cd web && npm test` 는 이 과제와 무관하게 깨끗한 체크아웃에서 실패한다**(`web/package.json:10` 이 `"vitest run"` 바레 호출) — 이미 별 회차의 미머지 PR 이 고치는 중이니 **건드리지 말고**, `make test` 대신 위 Go 명령으로 검증하라.

- 위험과 피할 것:
  - `internal/logbuf/logbuf_test.go` 는 2026-10-03 회차(verify-failed 로 폐기)가 같은 경로에 파일을 만들었다. 지금 base `b782c5f` 에는 **없음을 확인했다**(`ls internal/logbuf/` → `logbuf.go` 뿐). 그래도 그 PR 이 살아나면 같은 파일에서 충돌하니, 테스트는 위 기준 5개에 집중해 짧게 쓸 것(커버리지 전수 조사는 이번 과제가 아니다).
  - `web/package.json`·`Makefile`·`.github/workflows/*` 를 건드리지 말 것 — `npm test` 수정 PR 이 review-pending 이라 충돌한다.
  - `Handler.Handle` 의 **에러 평탄화 루프(108-112)와 `h.buf.add` → `h.next.Handle` 순서는 그대로 둘 것**. `/admin/logs` 응답이 `Attrs` 를 그대로 JSON 으로 내보내므로 값 타입을 바꾸면 UI 가 바뀐다.
  - `slog.Group(...)` 을 **레코드 attr 로** 넘기면 `a.Value.Resolve().Any()` 가 `[]slog.Attr` 를 그대로 맵에 넣는 별개의 결함이 있다. 유혹적이지만 **이번 범위 밖**(값 타입이 바뀌어 UI 영향). 발견만 적고 고치지 말 것.
  - `internal/server/*`(특히 `oauth.go`·`oidc.go`·`identity.go`·`tools.go approve()`)와 `internal/store/migrations` 는 손대지 말 것. 이 과제는 `internal/logbuf` 밖으로 나갈 이유가 없다.
  - 커밋 메시지는 영어 Conventional Commits(`fix: ...`), 코드 주석은 영어, 사용자에게 보이는 문자열은 한국어 — 이 저장소 관례.

- 차선 후보: `internal/mattermost/client.go` 의 남은 경로에 첫 테스트 — `FileContent` 의 `*[]byte` 우회, `PostList.Ordered()` 가 `Posts` 에 없는 id 를 조용히 버리는 것, `New()` 의 깨진 CA PEM 거부. 전부 `httptest.Server` + 실제 `*mattermost.Client` 로 배선(`internal/server/integration_test.go:30 fakeMM` 관례). 단 `client_test.go` 도 미머지 PR 이 있어 충돌 위험이 1순위보다 높다.
