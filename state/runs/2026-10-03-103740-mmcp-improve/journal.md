# 회차 노트 2026-10-03-103740-mmcp-improve — mmcp
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 10:37] base pinned — main@b782c5f
- [러너 10:37] autonomy release — 

## 정찰 노트
- `internal/logbuf` 테스트를 골랐다. 이 base(b782c5f)에는 2026-10-02 두 회차의 변경이 **둘 다 없다** — `internal/mattermost` 와 `web/package.json` 은 그 PR 들이 들고 있어 건드리면 충돌하므로, 겹치지 않으면서 테스트 0개인 곳이 logbuf 하나뿐이었다.
- 읽어 본 범위(logbuf.go 전체, admin_api.go:1490-1567 의 Snapshot/Subscribe 호출부)에서 링 버퍼 로직은 정확했다. 그래서 버그 수정이 아니라 순수 커버리지 과제이고, 가치를 3 으로 솔직히 매겼다.
- 추측으로 적은 것: `WithAttrs`→`WithGroup` 순서에서 버퍼와 JSON tee 의 출력이 갈린다는 판단은 **코드 판독만 했고 실행 확인은 안 했다**. 그래서 이번 과제에서 고치라고 하지 않고 "테스트로 못 박지 말 것"으로 격리했다(지금 동작을 고정하면 나중에 못 고친다). 프로덕션에 `WithGroup`/`logger.With` 호출자가 0건인 것은 확인했다.
- 구현자가 조심할 것: `gofmt -l .` 이 lint 첫 관문이라 신규 테스트 파일 포맷을 반드시 맞출 것. 구독 테스트는 `time.Sleep` 대신 타임아웃 건 `select` 로 — `-race` CI 에서 흔들린다.
- [러너 10:41] scout done — `internal/logbuf` 에 첫 테스트 — 링 버퍼 래핑·Snapshot 순서·Subscribe 해제 계약·slog.Handler 배선 (가치 3 / 위험 

## 구현 노트
- 무엇을/왜: `internal/logbuf/logbuf_test.go` 신규 1개만 추가(프로덕션 변경 0건, 커밋 `f8f7aab`). 관리 콘솔 로그 뷰어와 SSE 라이브 테일이 전부 이 129줄에 얹혀 있는데 테스트가 0개였다. 핸들러는 `main.go:61-62` 와 같은 실제 `slog.New(logbuf.NewHandler(buf, w, level))` 로 배선했다.
- 뮤테이션으로 이빨 확인: 고칠 결함이 없는 커버리지 과제라 "고치기 전 실패" 대신 프로덕션을 9가지로 일부러 깨뜨려 해당 테스트가 잡는지 확인하고 전부 되돌렸다(`git diff` 빈 출력 확인). 8건 적중 — 출력은 ledger 의 `실패 재현` 에 그대로 있다.
- **확신 없는 곳**: 뮤테이션 1건이 안 잡혔고 그게 발견이다 — `logbuf.go:62` 의 복사를 `return out` 으로 바꿔도 통과한다. `out` 이 nil 에 append 해서 만들어져 이미 새 배열이라 62줄은 중복이다. 진짜 앨리어싱 뮤테이션(`return b.entries[:b.next]`)으로 기준 3 이 유효함을 재확인하고, 테스트 주석의 잘못된 귀속을 고쳤다. 이 중복은 `ideas.json` 에 [1/1/S] 로 남겼다(이번에 고치지 않음 — 프로덕션 변경 금지).
- **검증 못 한 것**: `internal/server` 통합 테스트는 **돌지 않았다**(`ok 1.016s` = Postgres 없어 skip). 로컬에 뜬 Postgres 컨테이너는 다른 프로젝트 것(`bbmcp-test-pg`, `yeopjari-pg`)뿐이고 mmcp 통합 테스트는 DSN 에 마이그레이션을 거니 다른 세션 DB 를 건드리지 않았다. 이 과제는 `internal/server` 를 안 건드려 영향 없다.
- 일부러 안 한 것: ① `WithAttrs`→`WithGroup` 혼합 순서(지금 동작을 못 박으면 나중에 올바르게 고칠 때 방해 — 테스트 주석에 이유를 적어 뒀다) ② `New(0)` 패닉 ③ `Entry.Attrs` 의 깊은 복사(Snapshot 은 얕은 복사다 — 테스트 주석에 명시).
- 다음 역할이 조심할 것: 이 테스트는 DB·네트워크 없이 돈다(`go test -race -count=1 ./internal/logbuf/`, ~1s). 느린 구독자 테스트는 "끝나는 것" 자체가 논블로킹 증거라서, 혹시 멈추면 타임아웃까지 간다(블로킹 뮤테이션에서 실제로 `test timed out after 25s` 를 봤다). 동기화에 `time.Sleep` 을 쓴 곳은 없다.
- [러너 10:49] brief accepted — 채택 — 과제서의 근거가 현재 코드와 정확히 일치했다(`[no test files]`, `WithGroup`/`logger.With` 프로덕션 호출처 0건, `admin_api.g
- [러너 10:49] verify failed — 실패한 검증: cd web && npm test --silent (exit 1)
