# PR 처리기 노트 2026-09-20-115314-jasql-shepherd — jasql PR #1
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-09-20-110405-jasql-improve)
# 회차 노트 2026-09-20-110405-jasql-improve — jasql
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 11:04] base pinned — main@e9f3fb2
- [러너 11:04] autonomy release — 

## 정찰 노트
- 선택 이유: v0.31.0의 silent SSO·DB 프로파일 경로는 oauth_test/dbprofile_mcp_test로 이미 촘촘히 덮여 있어 손댈 곳이 없었고, 대신 `WrapLimit`(sqlguard.go)이 `SELECT … -- 주석`/`…; -- 주석`에서 닫는 괄호를 주석에 먹이거나 `;`를 남기는 실제 버그를 찾음 — 검증은 통과하고 실행만 깨지는 검증/실행 불일치라 가치가 높고 범위가 작다.
- 확신 없는 곳: 실제 Oracle에서 ORA 오류로 재현하지 못함(stub 드라이버) — 문자열 분석으로 판단. `stripSQL` 마스킹 길이로 원문을 자르는 구현 방식은 제안일 뿐 검증 안 함.
- 구현자 주의: `ValidateReadOnlySQL`의 정책 판정과 `stripSQL` 마스킹은 건드리지 말 것; 헬퍼는 문장 **끝**의 주석·`;`만 제거. 소스 grep을 테스트 증거로 쓰지 말고 헬퍼·WrapLimit 표 테스트 + allowed 목록 확장으로 두 경로 일치를 잠글 것. 버전 범프는 하지 말 것.
- [러너 11:07] scout done — 실행 경로의 SQL 정리(trailing 주석·세미콜론)를 한 헬퍼로 모아 `WrapLimit`/`Count`/`Explain`이 깨진 SQL을 만들지 

## 구현 노트
- 무엇/왜: `internal/oracle.TrimStatement` 추가 — 문장 끝의 공백·`;`·line/block 주석을 한 번의 전방 스캔(stripSQL과 같은 토큰 규칙)으로 제거. `WrapLimit`·`Count`·`ExplainPlan` 세 실행 경로가 모두 이 헬퍼를 쓰고, WrapLimit/Count는 본문을 `(\n본문\n)`으로 감싼다. 검증이 통과시키는 `… -- note`/`…; -- note`가 실행에서 ORA-00907/00911로 깨지던 불일치 해소. 커밋 1507e00.
- 확신 없는 곳·검증 못 한 것: (1) 실제 Oracle 실행 불가(stub 드라이버, `m.db()`가 const `driverAvailable`로 즉시 실패) — `Count`/`ExplainPlan`이 헬퍼를 쓰는 것은 코드 변경 자체로만 보장되고 전송 SQL을 잡는 테스트는 없음. `WrapLimit`은 표 테스트로 잠금. (2) `EXPLAIN PLAN … FOR <본문>` 뒤에 개행을 두지 않았는데, 헬퍼가 trailing 주석을 제거하므로 필요 없다고 판단 — 본문 끝이 `-- x`로 끝나는 경우가 남는지는 TestTrimStatement로만 확인.
- 일부러 하지 않은 것: 과제서가 제안한 "stripSQL 마스크 길이로 원문 자르기"는 `SELECT 'x'`(끝이 리터럴)에서 리터럴이 잘림(마스크가 리터럴 공백/주석 공백을 구분 못 함) — 전용 스캐너로 대체하고 그 케이스를 테스트에 넣음. `stripSQL`·`ValidateReadOnlySQL` 판정·`internal/mcp/tuning.go`·`catalog/validate.go`의 TrimSuffix는 손대지 않음. 기존 `oracle_test.go`의 gofmt 미정렬 줄(HEAD부터 그랬음)은 diff 최소화를 위해 그대로 둠.
- 다음 역할 주의: `WrapLimit` 출력 형식이 `SELECT * FROM (\n…\n) WHERE ROWNUM <= N`으로 바뀜(개행 추가) — 이 문자열을 로그/문서에서 정확히 비교하는 곳은 저장소 내 없음(grep으로 확인: oracle_test.go만). 전체 `go test ./...` 통과(mcp 8.8s).
- [러너 11:11] brief accepted — 채택 — 과제서의 근거(세 경로의 TrimSuffix 복제, 검증/실행 불일치)가 코드와 정확히 일치했고 수용 기준 1~3을 그대로 구�
- [러너 11:11] verify passed — 검증 3개 통과 (auto)
- [러너 11:11] review missing — 리뷰 결과 없음/손상: missing — 보류
- [러너 11:11] pr created — https://github.com/hkjang/jasql/pull/1

## 심사 노트
- 확인한 것: diff 4파일 전부 읽음. TrimStatement 토큰 규칙이 stripSQL과 1:1로 같아 검증/실행 불일치가 실제로 닫힘; 실행 문자열은 원문 접두라 정책이 넓어지지 않음. go build/vet, oracle+mcp 테스트 통과. 새 테스트는 구 코드에서 실패(형식·`-- note)` 잔존)/컴파일 불가 — 변경을 진짜 잠금.
- 못 본 것: 실 Oracle 실행(stub 드라이버). Count/Explain이 헬퍼를 쓰는 것은 코드 읽기로만 확인.
- 엣지: 큰따옴표 식별자 안의 `--`/`'`는 stripSQL·TrimStatement 둘 다 못 봄(`"x--y"` 잘림, `"it's" …;` 세미콜론 잔존). 구 코드가 우연히 통과시키던 케이스이나 KCB 카탈로그엔 없고, 고치려면 sqlguard 토크나이저 확장(보호 구역)이라 노트로만.
- 권고 근거: 범위 이탈 없음(gofmt 미정렬은 base부터), 인증·권한·데이터 무관, 문자열 조립만 바뀌어 revert 즉시 복구 — approve/merge, risk low.
