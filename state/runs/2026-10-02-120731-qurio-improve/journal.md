# 회차 노트 2026-10-02-120731-qurio-improve — qurio
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 12:07] base pinned — main@3f7cf65
- [러너 12:07] autonomy release — 

## 구현 노트
- 무엇/왜: 89b3e76 이 PostgreSQL 게이트에만 붙인 "이름과 `(` 사이에 trivia 만 있어야 호출" 판정을 Oracle 게이트에도 적용했다(헬퍼를 `postgresCallAdjacent`→`callAdjacent` 로 공유, 본문 무변경). 그전까지 Oracle 은 `WHERE amount = (SELECT …)`·`amount * (1 + rate)` 같은 평범한 질의를 허위 `amount()` 사유로 blocked 했다. 커밋 4b29d08, 프로덕션 1파일 +11/-5.
- 확신 없는 곳·검증 못 한 것: **실 Oracle DB 에 붙이지 못했다**(세션에 인스턴스 없음) — Oracle 증명은 전부 게이트 판정 수준이고, 이 변경은 질의를 *통과*시키는 방향이라 최종 문법 심판은 Oracle 서버다. 릴리즈 워크플로의 Oracle 실서비스 단계도 미실행. 비평가는 여기부터 보라.
- 일부러 안 한 것: `oraclePureFunctions`/`oracleParenthesisKeywords` 허용목록, PostgreSQL 경로, `postgresAliasColumnList`(미머지 433a183 과 충돌하므로), `ci.yml` 은 손대지 않았다.
- 다음 역할 주의: 새 단위 테스트는 DB 불필요. `internal/domain/dbexec/policy_test.go` 의 두 단정은 `analyzeExecutionPolicy`(프로덕션 게이트)를 직접 부르며 역시 DB 불필요. 통합 스위트는 폐기 PostgreSQL 17 + 4개 env + `-p=1` 필요(이번엔 127.0.0.1:55502 사용 후 컨테이너 제거). 재검증하려면 148판정 스윕을 base 3f7cf65 의 sqlsafe.go 와 비교하면 된다 — PostgreSQL 74행 diff 0 이 핵심 불변식이다.
- [러너 12:27] verify passed — 검증 9개 통과 (auto)

## 비평 노트
- 확인함: 원장에 `- 실패 재현:` 줄이 없어 직접 base sqlsafe.go 로 되돌려 돌렸다 — sqlsafe_test.go:479, dbexec/policy_test.go:171 둘 다 `amount()` 사유로 실패한다(가짜 테스트 아님). 74질의×2방언 스윕 base/HEAD diff: **PostgreSQL 0행**, Oracle 13행만 blocked→low 이며 전부 평범한 읽기 전용 식. dbms_lock.sleep·utl_http·dbms_xmlgen·quoted callee·TABLE(dbms_xplan.display())·NEXTVAL·@link 은 blocked 유지. gofmt/vet/`go test ./...` 통과.
- 못 봄: 실 Oracle 서버, 통합 스위트(-tags=integration), 릴리즈 워크플로 Oracle 단계 — 구현자 자백 그대로 남는다. 이 변경은 질의를 *통과*시키는 방향이라 릴리즈에서 Oracle 문법 최종 확인 필요.
- 승인이어도 남는 우려: `callAdjacent`(sqlsafe.go:744)의 `unicode.IsSpace(rune(sql[cursor]))` 가 바이트 캐스팅이라 U+00A0·U+3000·NUL·0x01~0x1f·0x7f 가 trivia 로 안 읽히고 default→false 로 **fail-open** 한다 — `dbms_lock.sleep (10)` 이 이제 Oracle 에서 low. ORA-00911 로 서버에서 죽으니 실행 가능한 우회는 아니고, 89b3e76 이 머지한 PostgreSQL 게이트에 이미 동일하게 존재한다(HEAD 에서 pg_ro=true 재현). **차단하지 않았지만 다음 회차가 두 방언 동시에 ASCII 공백만 허용하도록 좁히는 것이 맞다** — Oracle 단독으로 고치면 또 한쪽만 고치는 패턴이 된다.
- 문구만: sqlsafe.go:718 'both dialects share the same trivia syntax' 는 PostgreSQL 중첩 블록 주석을 무시한 과한 일반화. 그 경우는 스캐너가 '주석이 닫히지 않았습니다' 로 fail-closed 되므로 동작 문제 없음.
- 릴리즈 노트용: Oracle 전용 완화. 그전까지 `WHERE amount = (SELECT …)`·`amount * (1 + rate)`·`CASE WHEN a = (1)` 같은 스칼라 서브쿼리·산술 질의가 허위 함수 호출 사유로 차단됐던 것이 풀린다. PostgreSQL 판정은 비트 단위로 불변.
- [러너 12:32] review approved — 리뷰 승인 (risk=medium)
- [러너 12:32] pr created — https://github.com/hkjang/qurio/pull/31
