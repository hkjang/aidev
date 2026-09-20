# 과제서 — jasql 2026-09-20

- 과제: 실행 경로의 SQL 정리(trailing 주석·세미콜론)를 한 헬퍼로 모아 `WrapLimit`/`Count`/`Explain`이 깨진 SQL을 만들지 않게 하기 (가치 4 / 위험 2 / 작업량 S)

- 왜: `internal/oracle/sqlguard.go:WrapLimit`은 `strings.TrimSuffix(";")` 뒤에 같은 줄에 `)`를 붙이므로, LLM이 흔히 내는 `SELECT … FROM T -- 설명` 또는 `SELECT … FROM T; -- 설명` 형태는 `ValidateReadOnlySQL`(주석 마스킹 후 검사)을 통과하고도 실행 시 `SELECT * FROM (SELECT … -- 설명) WHERE ROWNUM <= 101`(닫는 괄호가 주석에 먹힘) 또는 `(… ; -- 설명)`(세미콜론 잔류)이 되어 ORA-00907/ORA-00911로 실패한다. 같은 `TrimSuffix(";")` 패턴이 `manager.go:Count`(`SELECT COUNT(*) FROM (`+trimmed+`)`)와 `explain.go`(`EXPLAIN PLAN … FOR `+cleanSQL)에도 복제돼 있어 세 경로가 같은 값을 다르게 다룬다. 고치면 검증을 통과한 SQL은 실행도 되고, 세 경로가 같은 정리 규칙을 공유한다.

- 수용 기준:
  1) `internal/oracle`에 `TrimStatement(sql string) string`(이름은 자유) 헬퍼를 추가: 앞뒤 공백, 끝에 붙은 line 주석(`-- …`)·block 주석(`/* … */`), trailing `;`를 **반복적으로** 제거한다(`… ; -- x`, `… -- a\n-- b`, `… /* c */ ;` 모두 본문만 남김). 문자열 리터럴 안의 `--`/`;`는 건드리지 않는다(기존 `stripSQL`의 마스킹 결과로 위치를 찾으면 안전).
  2) `WrapLimit`은 이 헬퍼를 쓰고, 출력은 `SELECT * FROM (\n<본문>\n) WHERE ROWNUM <= N` 형태로 본문 뒤에 개행을 두어(이중 안전장치) 남은 line 주석이 있어도 `)`가 살아남는다. `manager.go:Count`와 `explain.go`의 `cleanSQL`도 같은 헬퍼로 교체한다. `WrapLimit(sql, 0)`(Metadata 경로)도 자연히 같은 규칙.
  3) 테스트가 증명할 것: `oracle_test.go`의 `TestWrapLimit`를 표 테스트로 확장 — (a) 기존 케이스 `SELECT A FROM T ORDER BY A;` (기대 문자열은 새 개행 형식으로 갱신), (b) `SELECT A FROM T -- note`, (c) `SELECT A FROM T; -- note`, (d) `SELECT A FROM T /* note */;`, (e) `SELECT 'a;--b' FROM T` — 각각 결과에 `;`가 없고 `)` 앞에 `--` 주석이 같은 줄에 남지 않으며 리터럴은 보존됨을 확인. 헬퍼 자체의 표 테스트도 추가. `ValidateReadOnlySQL`의 allowed 목록에 `SELECT 1 FROM DUAL; -- ok`를 추가해 "검증 통과 ⇒ 실행 문자열도 정상"의 두 경로가 같은 입력을 같게 읽는지 함께 잠근다.

- 건드릴 파일:
  - `internal/oracle/sqlguard.go` : `WrapLimit` — 헬퍼 사용 + 개행 형식; 새 헬퍼 `TrimStatement` 추가(`stripSQL` 재사용 가능: 마스킹된 문자열에서 끝의 공백/`;`를 잘라 낸 길이로 원문을 자르면 리터럴 안전 — 단 `stripSQL`은 주석을 공백으로 치환하므로 TrimSpace 후 길이 기준 자르기가 그대로 통한다).
  - `internal/oracle/manager.go:Count` (약 349행) — `strings.TrimSuffix(strings.TrimSpace(sqlText), ";")` → 헬퍼.
  - `internal/oracle/explain.go` (약 76행 `cleanSQL`) — 동일 교체.
  - `internal/oracle/oracle_test.go:TestWrapLimit`, `TestValidateReadOnlySQL` — 위 3).
  - (선택, 같은 규칙이므로) `internal/mcp/tuning.go:59`는 catalog 정적 검증용이라 실행 SQL이 아님 — **이번엔 건드리지 않음**.

- 검증 명령:
  - `go build ./... && go vet ./...`
  - `go test ./internal/oracle/ -run 'TestWrapLimit|TestValidateReadOnlySQL' -v`
  - `go test ./internal/oracle/ ./internal/mcp/` (mcp는 ~9초; 전체 `go test ./...`는 catalog 골든셋 때문에 ~60초)

- 위험과 피할 것:
  - 실제 Oracle 실행은 이 환경에서 불가(stub 드라이버 빌드) — `Execute`/`Count`를 통한 end-to-end는 미확인. 헬퍼 단위 테스트 + 세 호출부가 모두 헬퍼를 쓰는지 코드로 확인하는 데 그친다는 점을 PR 설명에 명시할 것. 소스 문자열 grep을 테스트 증거로 삼지 말 것.
  - `ValidateReadOnlySQL`의 검사 로직(거부 키워드, 다중 문장 판정)은 바꾸지 말 것 — 정책 경계다. 정리 규칙을 "넓히는" 것이 아니라 검증이 이미 허용하는 입력을 실행 경로가 같게 읽게 하는 것.
  - `stripSQL`의 마스킹 동작 자체를 바꾸지 말 것(검증 결과가 달라진다).
  - `explain.go`의 EXPLAIN PLAN은 DDL성 — 헬퍼가 본문 끝의 `;`만 제거하는지 확인(문장 중간의 `;`는 검증 단계에서 이미 거부됨).
  - `internal/catalog/validate.go:1002`, `template_compile.go`의 TrimSuffix는 다른 계약(정적 검증·CTE 조립)이므로 손대지 말 것.
  - 버전 범프·README 갱신은 하지 않는다(릴리즈 단계 담당).

- 차선 후보: `ValidateReadOnlySQL`이 호출마다 거부 키워드 ~25개의 `regexp.MustCompile`을 반복 컴파일함(`sqlguard.go:53`) — 내장 키워드 정규식을 패키지 변수로 사전 컴파일하고 profile extraDenied만 호출 시 컴파일. 동작 변화 없이 실행 경로 비용만 줄이는 S 작업. 벤치마크(`go test -bench`)로 전후를 남길 것.
