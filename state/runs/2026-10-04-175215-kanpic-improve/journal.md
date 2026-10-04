# 회차 노트 2026-10-04-175215-kanpic-improve — kanpic
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 17:52] base pinned — main@2613cf7
- [러너 17:52] autonomy release —

## 정찰 노트
- mail 4회·automation 2회로 포화된 패키지를 피해 테스트가 없는 `internal/observability/logs.go` 를 열었고, 거기서 콘솔과 DB 가 같은 레코드를 다르게 읽는 자리를 찾았다: `attr.Value.Any()` + `encoding/json` 이 error 를 `{}` 로 굳히는데 slog 의 JSONHandler 는 `err.Error()` 를 적는다. 프로덕션 28곳이 `"error", err` 를 넘기므로 관찰되는 결함이고, delimited(이미 17사례로 덮임)·문서 #N/A(PDF 동반)·cron 부호(저장된 일정 파괴 위험)를 제쳤다.
- 재현은 임시 테스트(`zz_probe_test.go`)로 프로덕션 `persistentHandler` 를 `slog.New` 에 꽂아 실제로 돌렸고 출력은 과제서에 그대로 옮겼다; 파일은 지웠고 `git status` 는 깨끗하다. 스크래치 사본은 `scratch/probe.go`.
- 추측으로 적은 것: `newPersistentHandler` 추출은 테스트 편의를 위한 제안일 뿐 꼭 그 모양일 필요는 없다. `go test ./...` 전체와 웹·DB·E2E 는 이번에 돌리지 않았다(관찰 패키지만 실행). 프로필(2026-09-29)은 지금 코드와 어긋나는 데가 없어 새로 쓰지 않았다.
- 구현자가 조심할 것: `run()` 을 테스트에서 돌리면 `pool=nil` 역참조로 패닉한다(확인). `WithAttrs`/`WithGroup` 의 속성 유실은 같은 파일에서 눈에 띄지만 프로덕션에 `logger.With(` 호출자가 없어 증명할 자리가 없으니 이번에 끌어오지 말 것 — 아이디어로 남겼다.
- 범위는 프로덕션 1파일(`logs.go`)+테스트 1파일. SQL·마이그레이션·CSV 내보내기·새 로깅 값 추가는 모두 범위 밖으로 못 박았다.
- [러너 17:57] scout done — 시스템 로그 DB·감사 내보내기가 콘솔과 같은 값을 읽게 한다 — 지금 `"error", err` 는 `{}` 로 저장된다 (가�

## 구현 노트
- `internal/observability/logs.go` 의 `Handle` 이 콘솔 JSONHandler 와 같은 규칙으로 `attributes` 를 만들게 했다(`addAttribute`/`attributeValue`: Resolve → group 중첩 맵 → json.Marshaler 아닌 error 는 `Error()`). 그래야 `"error", err` 가 DB·CSV 에도 원인 문자열로 남는다. 프로덕션 1파일, 테스트 1파일 신규, 커밋 7e5904d.
- 확신 없는 곳: 콘솔과의 동치는 `log/slog` 의 `appendAttr`/`appendJSONValue` 규칙을 읽고 옮긴 것이라 Go 가 그 규칙을 바꾸면 갈릴 수 있다 — 다만 테스트가 실제 JSONHandler 출력과 비교하므로 그때 빨강이 된다.
- `time.Time` 속성은 콘솔(밀리초)과 DB(나노초)의 표현이 여전히 다르다. 일부러 두었고(저장 쪽이 더 정밀) 테스트는 이 키를 콘솔 비교에서 빼고 `time.Time` 보존만 단언한다 — 주석에 이유를 적었다.
- 검증 못 한 것: 실제 Postgres 에 들어간 행은 보지 못했다(`run()` 은 pool 이 필요하고 이 세션에 DB 가 없다). `json.Marshal(record.attributes)` 까지만 증명했고 그 뒤 INSERT 경로는 바뀌지 않았다. 웹·DB 통합·E2E 미실행(Go 서버 안쪽 변경).
- 일부러 하지 않은 것: `WithAttrs`/`WithGroup` 의 handler 수준 속성 유실(프로덕션에 `logger.With(` 호출자가 없어 증명할 자리가 없다), 옛 `{}` 행 백필, SQL·마이그레이션·CSV 내보내기, 새 로깅 값 추가.
- 다음 역할이 조심할 것: 테스트에서 `handler.run()` 을 돌리면 `pool=nil` 역참조로 패닉한다. `closeFn`/`close(handler.queue)` 는 한 번만(`once` 공유). 테스트는 DB 없이 돈다 — `go test ./internal/observability -run TestPersisted -v`.
- [러너 18:03] brief accepted — 채택 — 과제서의 재현이 지금 코드와 정확히 맞았고(`"error", err` → `{}`, group → `[{Key,Value:{}}]`), 건드릴 파일 2개(프로덕�
- [러너 18:03] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 판정 approve(low). 원장에 `- 실패 재현:` 줄이 없어 직접 세웠다: main(2613cf7) 분리 worktree 에 새 `logs_test.go` + `newPersistentHandler` 껍데기만 넣어 돌리니 두 테스트 모두 실패하고 출력이 과제서 증상과 일치했다(`"error": DB={}, 콘솔="상위: 연결 거부"`, `g=[{Key,Value:{}}]`, `flat=<nil>`). 테스트는 바뀐 경로를 실제로 지난다; 임시 worktree 제거, 대상 트리 clean.
- `addAttribute`/`attributeValue` 를 Go `log/slog` 의 `appendAttr`·`appendJSONValue` 와 한 줄씩 대조해 일치 확인(Marshaler 우선 → error→`Error()` → `json.Marshal`, 빈 그룹·키없는 그룹 인라인·빈 속성 생략). 빈 속성 검사가 그룹 분기 뒤에 있으나 빈 Value 의 Kind 가 `KindAny` 라 순서가 결과를 바꾸지 않는다. go build·vet·gofmt 깨끗, `go test ./...` 전체 통과, observability `-race` 통과.
- 검토 부서 차단 없음. 신규 엔드포인트·권한 변경 없고 조회/CSV/퍼지는 모두 `requireAdmin` 뒤(platform.go:152,211,258). CSV 수식 주입 확인 — attributes 열은 항상 `{` 로 시작하는 JSON 객체라 새 error 문자열이 셀을 수식으로 열 수 없다. 비밀값: AI 키는 헤더로만(gateway.go:505), external 전체 URL 은 이미 `"url"` 로 평문 저장돼 있었다(fetcher.go:367) — 새 노출 없음. 개인정보는 새 수집이 아니라 콘솔에 이미 적던 값의 영구화이고 삭제 수단(PurgeBefore)은 있다.
- 못 본 것: 실제 Postgres 행(INSERT 경로 미변경이라 생략), 웹 lint/test/build, DB 통합, E2E. 남는 우려 둘 — ① `WithAttrs`/`WithGroup` 의 handler 수준 속성은 여전히 DB 에서 빠지거나 평평해진다(프로덕션 `logger.With(` 호출자 0개 확인, 누가 하나 추가하면 커밋 메시지 주장이 거짓이 됨 → 다음 회차 후보). ② `run()` 의 `data, _ := json.Marshal` 이 에러를 버려 +Inf/NaN 하나로 attributes 전체가 NULL 이 된다(기존 결함, 이번 변경은 오히려 이 경우를 줄인다).
- 릴리즈 노트에 쓸 것: "시스템 로그의 `error` 속성이 DB·CSV 에도 원인 문자열로 남는다(이전 `{}`), 그룹 속성은 중첩 객체로." 옛 `{}` 행은 백필하지 않았으니 과거 기록에는 원인이 없다. 마이그레이션·스키마·외부 상태 변경이 없어 revert 로 완전히 돌아온다.
- [러너 18:08] review approved — 리뷰 승인 (risk=low)
- [러너 18:08] pr created — https://github.com/hkjang/kanpic/pull/40
