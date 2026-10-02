# 회차 노트 2026-10-02-120731-qurio-improve — qurio
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 12:07] base pinned — main@3f7cf65
- [러너 12:07] autonomy release — 

## 구현 노트
- 무엇/왜: 89b3e76 이 PostgreSQL 게이트에만 붙인 "이름과 `(` 사이에 trivia 만 있어야 호출" 판정을 Oracle 게이트에도 적용했다(헬퍼를 `postgresCallAdjacent`→`callAdjacent` 로 공유, 본문 무변경). 그전까지 Oracle 은 `WHERE amount = (SELECT …)`·`amount * (1 + rate)` 같은 평범한 질의를 허위 `amount()` 사유로 blocked 했다. 커밋 4b29d08, 프로덕션 1파일 +11/-5.
- 확신 없는 곳·검증 못 한 것: **실 Oracle DB 에 붙이지 못했다**(세션에 인스턴스 없음) — Oracle 증명은 전부 게이트 판정 수준이고, 이 변경은 질의를 *통과*시키는 방향이라 최종 문법 심판은 Oracle 서버다. 릴리즈 워크플로의 Oracle 실서비스 단계도 미실행. 비평가는 여기부터 보라.
- 일부러 안 한 것: `oraclePureFunctions`/`oracleParenthesisKeywords` 허용목록, PostgreSQL 경로, `postgresAliasColumnList`(미머지 433a183 과 충돌하므로), `ci.yml` 은 손대지 않았다.
- 다음 역할 주의: 새 단위 테스트는 DB 불필요. `internal/domain/dbexec/policy_test.go` 의 두 단정은 `analyzeExecutionPolicy`(프로덕션 게이트)를 직접 부르며 역시 DB 불필요. 통합 스위트는 폐기 PostgreSQL 17 + 4개 env + `-p=1` 필요(이번엔 127.0.0.1:55502 사용 후 컨테이너 제거). 재검증하려면 148판정 스윕을 base 3f7cf65 의 sqlsafe.go 와 비교하면 된다 — PostgreSQL 74행 diff 0 이 핵심 불변식이다.
