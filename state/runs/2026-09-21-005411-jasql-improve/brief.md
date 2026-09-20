- 과제: 명시적 숫자 날짜의 달력 유효성을 공통 파서에서 검증 (가치 4 / 위험 1 / 작업량 S)
- 왜: `ParseTimeExpressions`는 숫자 날짜를 정규식으로 추출한 뒤 달력 검증 없이 반환하여 `2025-02-30`도 유효한 기간으로 취급한다. 이를 공통 파서에서 제외하면 문자열 날짜 컬럼에 불가능한 날짜 조건을 생성하는 문제와 DATE 컬럼에서만 조건이 사라지는 불일치를 함께 해결한다.
- 수용 기준:
  1) `2025-02-30`, 평년의 `2025-02-29`, `2025-04-31`, `2025-01-00` 및 이들의 무구분자/점/슬래시 표기는 time_ranges에 포함되지 않는다. 날짜를 다른 날로 보정하지 않으며, 잘못된 날짜와 유효한 날짜가 함께 나오면 유효한 날짜만 유지한다.
  2) `2024-02-29`, `2024.02.29`, `2024/02/29`, `20240229`, 정상 월말은 현재의 expression·YYYYMMDD Start/End를 유지한다. 상대 기간·비교·보고 단위의 기존 동작과 응답 스키마를 바꾸지 않는다. 새 오류 응답/새 확인 질문을 만들지 않고 기존의 '유효한 기간 없음' 처리를 따른다.
  3) 파서 표 테스트와 실제 MCP tools/call 경로 회귀 테스트로 `resolve_time`, `get_schema_context`, `build_sql_skeleton`에서 같은 입력이 일관되게 처리됨을 증명한다. DATE_YYYYMMDD·MONTH_YYYYMM·DATE·TIMESTAMP·DATETIME_YYYYMMDDHH24MISS 컬럼을 가진 TempDir JSON을 `catalog.Load`로 읽어 실제 Server에 연결하고, 잘못된 날짜는 기간/생성 조건/SQL에 없으며 정상 윤일은 실제 조건에 나타남을 확인한다. 정상 입력도 검증하여 빈 응답만 반환하는 수정이 통과하지 못하게 한다. `postJSON`의 실제 `/mcp` 핸들러 또는 `ServeStdio` 왕복을 사용하며 파서 대역이나 소스 문자열 검사는 금지한다.
- 건드릴 파일:
  - `internal/catalog/timeparse.go:ParseTimeExpressions` — 마지막 raw dates 루프에서 구분자를 제거한 `clean`을 기존 `ymd`와 `time.Parse`로 검증한 후에만 add한다. 이미 import된 time을 활용할 수 있다.
  - `internal/catalog/eval_test.go:TestParseTimeExpressions`, `TestRenderTimeConditionBySemanticType` — 기존 표를 참고하여 달력 유효성/윤일/혼합 입력 회귀를 별도 테스트로 추가한다.
  - `internal/mcp/timeparse_test.go` (신규 제안) — `internal/mcp/datasets_test.go:newFixtureServer`, `internal/mcp/server_test.go:postJSON`, `internal/mcp/stdio_test.go:TestServeStdio`의 패턴을 참고해 독립 픽스처와 MCP 통합 테스트를 만든다. 기존 공용 픽스처에는 날짜 컬럼이 없으므로 모든 테스트가 사용하는 픽스처를 바꾸지 말고 새 테스트 전용 JSON을 작성한다.
  - 읽어서 배선 확인할 곳(수정 불필요): `internal/catalog/search.go:yyyyMMddRE`, `SchemaContext`; `internal/catalog/skeleton.go:BuildSQLSkeleton`; `internal/catalog/timeparse.go:ResolveTime`, `RenderTimeCondition`; `internal/catalog/clarify.go:hasTimeRange`; `internal/mcp/server.go:callTool`의 세 도구 case. 모두 공통 파서에 연결되어 있다.
- 검증 명령: 저장소 루트에서 `go test ./internal/catalog ./internal/mcp -run 'Test.*(Time|Calendar)' -count=1` (신규 테스트명에 Time 또는 Calendar 포함), `go build ./...`, `go vet ./...`, `go test ./...`.
- 위험과 피할 것: raw 숫자 날짜의 달력 검증만 좁힌다. `yyyyMMddRE`의 구분자 허용 범위, 한국어 연월 파싱, 최근 N개월/년 의미, RenderTimeCondition의 외부 직접 입력 방어, SQL 가드, auth/session, migrations, workflows, 실데이터는 이번에 변경하지 않는다. 카탈로그를 로드한 뒤 제자리 mutation하지 않는다. 지난 회차 SQL trailing 주석 수정은 기록상 성공이나 현재 e9f3fb2에는 TrimStatement가 없으므로 재구현하거나 끼워 넣지 않는다. Oracle 실 실행과 실사용 발생 빈도는 미확인이다.
- 차선 후보: 개발자 가이드의 Go 버전·의존성·MCP 도구 수 갱신 — 날짜 문제가 착수 시점 코드에서 이미 해결된 경우에만 `docs/development.md`의 Go 1.24+/외부 의존성 0/go.sum 없음/24개 도구 주장을 go.mod·go.sum·stdio_test.go의 42개 단정과 맞춘다. 소스/API 메서드를 확인하고 새 가이드를 복제하지 않는다.

구현 순서 및 추정: 회귀 테스트와 기존 실패 확인 10분 → 공통 파서 검증 5분 → 세 MCP 경로 정상/비정상 테스트 15분 → 전체 검증과 차이 검토 5분, 예비 10분(합계 45분). 세부 날짜 어휘 확장이나 사용자 경고 설계는 예비 시간에도 범위에 넣지 않는다.

정찰 근거: 기준 커밋 e9f3fb2, 작업 트리 코드 변경 없음. `go build ./...`, `go vet ./...`, `go test ./...` 실제 통과(catalog 55.001초, mcp 9.607초, meta 캐시, oracle 0.010초). 다음 read-only 재현도 실행해 structuredContent의 time_ranges.start=20250230 및 BIRTH_DT = '20250230'를 확인했다:

```sh
printf '%s\n' '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"resolve_time","arguments":{"question":"2025-02-30 고객","table":"DWMST.TBIA01A"}}}' | go run ./cmd/jasql-mcp -transport stdio -data ./data/kcb
```

스킬 가용성: 요청된 `pmo:estimating-and-contingency`, `technology:implementation-planning`, `technology:solution-exploration`은 현재 제공된 Skill 도구/스킬 목록에 없고, 기본 로컬 스킬 경로와 추가 경로 검색에서도 찾지 못했다. 원문 절차·반환 형식은 미확인으로, 스킬을 적용했다고 주장하지 않는다. 본 과제서는 사용자 지정 형식으로 대안 평가·실행 순서·작업 추정·예비 시간을 기록한다.
