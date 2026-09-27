# 회차 노트 2026-09-27-163201-qurio-improve — qurio
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 16:32] base pinned — main@2128ef7
- [러너 16:32] autonomy release — 
- [러너 16:39] scout done — sqlsafe PostgreSQL 게이트가 `col=(SELECT …)` 같은 스칼라 서브쿼리를 함수 호출로 오판해 차단하는 오탐 수정 (�

## 구현 노트
- 89b3e76: sqlsafe 가 `version=(SELECT …)` 를 `version()` 호출로 오판해 차단하던 것을 고쳤다. 비인용 word·leftParen 토큰에 소스 오프셋만 채우고(토큰 종류 신설 없음 = 기존 인접 가정 무변경) 새 헬퍼 `postgresCallAdjacent` 로 이름과 `(` 사이에 trivia 만 있을 때에만 호출로 판정한다. 허용목록은 한 줄도 안 건드렸다. 프로덕션 파일 1개.
- **확신 없는 곳**: (1) `unicode.IsSpace(rune(sql[cursor]))` 는 바이트를 룬으로 캐스팅하므로 U+0085/U+00A0 의 UTF-8 연속 바이트를 공백으로 볼 여지를 따졌다 — 선행 바이트(0xC2/0xC3)가 먼저 default 로 떨어져 `false`(=호출 아님)를 돌려주니 차단이 풀리는 방향은 아니고, PostgreSQL 은 NBSP 를 공백으로 안 보므로 그 문장은 애초에 실행 불가다. 기존 `oracleOuterJoinEnd` 와 같은 패턴이라 맞췄지만 비평가가 먼저 볼 만한 곳이다. (2) `int32` 오프셋은 2GB 초과 SQL 에서 음수로 넘치는데 `start < 0` 가드로 fail-closed 처리했다(선재 패턴).
- **검증 못 한 것**: Oracle 실 DB 는 없어서 안 돌렸다 — 대신 Oracle 방언 판정 85건이 수정 전후 바이트 단위로 동일함을 스윕으로 보였다(`oracleReadOnlyViolations` 는 `.start` 를 전혀 읽지 않는다). SPA·웹 테스트는 무관해서 안 돌렸다.
- **일부러 하지 않은 것**: `WITH ORDINALITY g(v,i)` 오탐 — 이 수정으로 안 고쳐지는 것을 스윕에서 재확인했고(`g` 와 `(` 가 실제로 인접) diff 를 키우지 않으려 별건으로 남겼다. 허용목록에 VERSION 추가도 금지대로 하지 않았다.
- **다음 역할이 조심할 것**: 새 테스트 3건은 DB 없이 돈다(`go test ./internal/domain/sqlsafe ./internal/legacyapi`). 통합 테스트는 폐기 PG17 + 세 env(`POSTGRES_DSN`·`QURIO_INTEGRATION_DSN`·`QURIO_TEST_POSTGRES_DSN`) 필요. **`internal/platformapi` 통합 2건은 같은 DB 에 `./...` 를 두 번 돌리면 잔존 행 때문에 실패한다 — 내 변경과 무관함을 원본 파일 대조로 확인했고 깨끗한 DB 에서는 30 패키지 전부 통과한다. 재실행 시 DB 를 새로 만들 것.**
- [러너 16:58] brief accepted — 채택 — 과제서가 지목한 행(450-470·1074·1094·1157, `.start` 를 읽는 곳이 443행 한 곳뿐, 419행의 `sql` 이 스캔 대상과 동일)이 �
- [러너 17:01] verify passed — 검증 9개 통과 (auto)

## 비평 노트
- 판정 approve(risk medium). 테스트 진정성은 main 의 sqlsafe.go 로 갈아끼워 새 테스트 2건이 실제 red 임을 재현해 확인했고, legacyapi 테스트는 프로덕션과 같은 상수·방언을 쓴다. 이 diff 는 단방향 permissive 라 "새로 통과하는 위험 호출" 만 위험한데, 이름↔`(` 사이 1바이트 전수 + 2바이트 1156건 스윕으로 HEAD vs main 차이를 뽑아 보니 새로 허용되는 것은 PostgreSQL 에서 구문 오류이거나 식별자에 흡수되는 문자뿐이었다 — 이 변경이 만드는 실행 가능한 우회는 없다. 중첩 주석 fail-closed 도 실측했다.
- **다음 회차 최우선(선재 결함, 이번 머지 사유 아님)**: CR 로 끝나는 `--` 주석이 게이트를 통째로 우회한다. `sqlsafe.go:1053-1056` 이 `\n` 만 주석 종료로 보는데 PostgreSQL 은 CR 도 종료로 본다(폐기 PG 17.11 에서 `SELECT abs--x\r(-1)` → `1` 실측). 그래서 `SELECT dblink_exec--x\r('host=h dbname=d','DROP TABLE victim')` 가 readonly=true/risk=low 로 통과하고 `nextval`·`query_to_xml`·`pg_catalog.pg_terminate_backend` 도 같다. main 에도 동일하게 존재함을 확인했다. 수리 시 스캐너와 새 헬퍼(`sqlsafe.go:740` 블록 내부 루프) **두 곳**을 함께 고쳐야 한다 — 이번 변경으로 같은 규칙이 두 곳으로 늘었다.
- 못 본 것: `-tags=integration` 통합 테스트(원장의 폐기 PG17 2회 통과를 재현하지 않았다), Oracle 실 DB, SPA/웹. Oracle 무영향은 `.start` 독자가 443·724·725 세 곳뿐이고 Oracle 경로가 읽지 않음을 확인해 코드로 대체했다. 돌린 것: `go test ./...` 전부 통과, `go vet ./...`, `gofmt -l` 무출력, 프로브 파일 제거 후 worktree 깨끗.
- [러너 17:08] review approved — 리뷰 승인 (risk=medium)
- [러너 17:08] pr created — https://github.com/hkjang/qurio/pull/24
- [러너 17:27] ci timeout — 제한 시간 안에 CI 완료를 확인하지 못함
