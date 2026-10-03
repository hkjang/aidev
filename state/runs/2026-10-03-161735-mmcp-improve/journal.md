# 회차 노트 2026-10-03-161735-mmcp-improve — mmcp
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 16:17] base pinned — main@b782c5f
- [러너 16:17] autonomy release — 

## 정찰 노트
- `npm test` 수정(가치 5)은 base 에 그대로 남아 있지만 같은 처방의 PR 이 review-pending 이라 세 번째로 고르지 않았다. `ChannelMembers` 잘림 후보는 호출처 0건(죽은 코드)임을 확인해 기각했고, `client_test.go` 는 폐기된 PR 과 같은 신규 파일이라 차선으로 내렸다. 남은 것이 logbuf 이고, 임시 테스트로 두 결함을 **직접 재현**해서 골랐다.
- 확신 없는 곳: 이 과제는 **잠재 결함**이다 — `WithGroup`/`logger.With` 프로덕션 호출처가 0건이라 오늘 사용자가 겪는 장애가 아니다. 과제서에 숨기지 않고 적었으니 비평자는 가치 3 을 그 전제로 읽을 것. 이전 기록의 "logbuf.go:40 0 나누기" 는 틀렸고 실제 패닉은 39줄 인덱싱이다(재현 로그로 정정).
- 구현자가 조심할 것: ① `h.next.WithAttrs(as)` 에는 키를 고치지 않은 **원본** 을 넘겨야 한다(표준 핸들러가 자기 규약대로 중첩시킨다) ② `web/package.json`·`Makefile`·workflows 를 건드리면 review-pending PR 과 충돌한다 ③ 검증에 `make test` 를 쓰면 과제와 무관하게 `vitest: not found` 로 떨어진다 — 과제서가 지정한 `go test -race ./internal/logbuf/` + `make lint` 로 할 것.
- 프로필은 1일 전 것이 지금 코드와 일치해 새로 쓰지 않았다(`internal/logbuf` 테스트 0개, `internal/mattermost` 테스트 0개, 환경변수 4개 모두 그대로).
- [러너 16:22] scout done — `logbuf.Handler` 의 `WithGroup` 접두사가 그룹을 열기 **전에** 붙인 attr 까지 소급 적용돼 관리 콘솔 로그와 JSON t

## 구현 노트
- 무엇/왜: `Handle()` 이 현재 그룹 접두사를 `h.attrs` **전체**에 붙여 그룹 이전 attr 까지 소급 한정하던 것을, 접두사가 **붙는 시점**에 확정되도록 `WithAttrs` 로 옮겼다(`Handle` 은 레코드 attr 에만 prefix). `h.next.WithAttrs(as)` 에는 원본 `as` 를 그대로 넘긴다. 함께 `add()` 에 `len(b.entries)>0` 가드를 넣어 `New(0)` 패닉을 막았다. 프로덕션 파일 1개(`logbuf.go`) + 신규 `logbuf_test.go`.
- 확신 없는 곳·검증 못 한 것: ① **Postgres 가 없어 `internal/server` 통합 테스트는 skip 됐다**(`ok ... 1.018s`) — 이 변경은 `internal/logbuf` 밖으로 나가지 않지만 `/admin/logs` 응답을 실제로 돌려본 것은 아니다. ② `WithGroup`/`logger.With` 프로덕션 호출처가 0건이라 **오늘 사용자가 겪는 장애가 아닌 잠재 결함**이다(정찰 과제서와 같은 판단). ③ `WithAttrs` 가 `slog.Attr` 를 `range` 로 복사해 호출자 슬라이스를 안 건드린다는 것은 Go 값 의미론에 기댄 추론이고 별도 테스트로 못 박지는 않았다.
- 일부러 안 한 것: `slog.Group(...)` 을 레코드 attr 로 넘길 때 `[]slog.Attr` 가 그대로 `Attrs` 맵에 들어가는 **별개 결함** — 값 타입이 바뀌면 `/admin/logs` JSON 모양이 바뀌어 UI 에 영향이라 과제서 지시대로 범위 밖(ideas.json 에 기록). `logbuf.go:62` 의 중복 복사도 같은 파일이지만 혼동을 피해 제외. `web/package.json`·`Makefile`·workflows 는 review-pending PR 과 충돌하므로 미변경.
- 다음 역할이 조심할 것: `TestZeroCapacityBufferDoesNotPanic` 은 `cancel()` 을 **일부러 defer 하지 않는다** — `add()` 가 `b.mu` 를 쥔 채 패닉하므로 defer 하면 패닉이 10분 테스트 타임아웃으로 둔갑한다(재현 중 실제로 겪었다). 이 줄을 "정리" 하지 말 것. 테스트는 DB 없이 돈다.
- [러너 16:43] brief accepted — 채택 — 과제서의 근거가 현재 코드와 정확히 일치했고(재현 출력까지 글자 단위로 맞았다), 지정한 네 파일(실제로는 `lo
- [러너 16:43] verify failed — 실패한 검증: cd web && npm test --silent (exit 1)
