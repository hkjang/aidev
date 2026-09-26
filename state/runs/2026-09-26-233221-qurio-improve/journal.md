# 회차 노트 2026-09-26-233221-qurio-improve — qurio
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 23:32] base pinned — main@1ad4e50
- [러너 23:32] autonomy release — 

## 구현 노트
- 무엇/왜: `internal/domain/sqlsafe` 의 `postgresAliasColumnList` 이 이름 앞 토큰이 `)` 이기만 하면 alias column list 로 단정해 함수 허용목록 검사를 건너뛰었다. 괄호 항 하나를 앞에 붙이면 어떤 호출이든 숨었다(`SELECT (SELECT 1) || dblink_exec('host=internal','DROP TABLE victim')` → readOnly/low). `postgresFromItemContext` 로 왼쪽을 걸어 FROM/JOIN/LATERAL 문맥일 때만 면제한다. 프로덕션 파일 1개.
- 확신 없는 곳: (1) `postgresValueExpressionKeywords` 목록의 완결성 — 빠진 키워드는 탐색을 더 멀리 보내므로 오탐 쪽이 아니라 **미탐 쪽**으로 틀릴 수 있다. FROM 과 값 식 사이에 올 수 있는 키워드를 더 찾아보면 좋다. (2) `FROM (SELECT 1) t(x) evil(y)` 같은 문법 오류 형태는 여전히 면제된다 — 대상 DB 파서가 거부하므로 무해하다고 판단했으나 직접 실행해 확인하지는 않았다. (3) Oracle 경로는 손대지 않았다(이미 같은 형태를 차단).
- 일부러 안 한 것: 같은 파일에서 발견한 **선재 오탐 2건**(`WHERE col=(SELECT …)` 를 `col()` 함수 호출로 오판 → 기본 임베딩 컬렉션 생성이 막힌다; `WITH ORDINALITY g(v,i)` 오탐)은 고치지 않았다 — 방향이 반대(loosening)이고 `scanOraclePolicy` 가 연산자를 버리는 구조를 건드려야 해서 보안 게이트를 느슨하게 할 위험이 있다. 원본 파일로도 동일 재현됨을 확인해 ideas.json 에 근거와 함께 남겼다. 문서 변경 없음(allowlist 를 설명하는 문서가 없고, 옛 동작을 단정한 문서도 없다).
- 다음 역할 주의: 새 테스트 3개는 순수 단위 테스트라 DB 없이 돈다. 다만 `internal/domain/dbexec` 통합 테스트는 **마이그레이션이 적용된 폐기 DB** 가 있어야 하고(빈 DB 로는 내 변경과 무관하게 3건 실패 — 원본 파일로도 동일), `QURIO_TEST_POSTGRES_DSN` 전용에 `-p=1` 직렬 실행이 필요하다. 이번 검증은 도커 postgres:17-alpine 127.0.0.1:55452 에서 했고 컨테이너는 제거했다.
- [러너 23:53] verify passed — 검증 9개 통과 (auto)

## 비평 노트
- 확인: 새 테스트는 진짜로 바뀐 경로를 지난다(패키지를 /tmp 로 복사해 sqlsafe.go 만 main 으로 되돌리면 `(1) || custom_reporting_fn(id)` 에서 즉시 실패). gofmt·go vet·패키지 테스트 통과. 구현 노트 불확실 (2) 는 PG17 에서 syntax error 로 무해 확인.
- 거절 사유: `sqlsafe.go:777` 의 왼쪽 탐색이 FROM 목록의 최상위 콤마를 경계로 보지 않아 앞 from-item 의 `ON`/`USING` 에 걸린다 → PG17 에서 정상 실행되는 `FROM a JOIN b ON a.id=b.id, (SELECT 1) t(x)` 류가 main 의 low 에서 blocked 로 하드 차단(runtimeapi·mcpserver·agentapi 실행 경로 직결).
- 수리가 먼저 볼 파일: `internal/domain/sqlsafe/sqlsafe.go` postgresFromItemContext. `ON`/`USING` 를 목록에서 빼는 해법은 금지 — `JOIN b ON (a.x) = evil_fn(y)` 로 원래 구멍이 재개방된다. 깊이 0 콤마를 건넜는지를 상태로 들고 콤마 이후에만 `ON`/`USING` 을 무시할 것.
- 승인이어도 남을 우려(릴리즈 노트): `dbexec/postgres_safety.go:418/430/439` 경유로 `) fn(` 형태를 가진 뷰·파티션 키·RLS 정책이 이제 관계째로 ErrUnsafeDatabaseObject — 실측 `SELECT 1 WHERE ((tenant_id) = current_setting('app.tenant'))` low→blocked. 의도된 강화지만 사용자 가시적.
- 못 본 것: dbexec 통합 테스트(폐기 DB 미준비) 미실행. 검증은 sqlsafe 단위 A/B + PG17 컨테이너 구문 확인까지. 컨테이너 제거 완료. 보안·법무 차단 사유 없음(실패는 닫히는 방향, 공격 경로 없음).
- [러너 00:03] review rejected — 리뷰 거절: internal/domain/sqlsafe/sqlsafe.go:777 postgresFromItemContext 의 왼쪽 탐색이 FROM 목록의 최상위 콤마를 경계로 인식하지 않는다. 앞선 from-item 의 `JOIN ... O

## 수리 노트
- 지적은 맞았다: FROM 목록의 최상위 콤마를 경계로 보지 않아 `FROM a JOIN b ON …, (SELECT 1) t(x)` 세 쿼리가 blocked 로 재현됐다. `AS` 도 같은 부류로 깨져 있었다(`FROM (VALUES (1)) AS a, (SELECT 1) t(x)`) — 지적에 없었지만 같은 결함이라 함께 고쳤다.
- 고친 방법(b715a45): 콤마 통과 플래그 + ① 짝 안 맞는 `(` 를 넘으면 플래그 해제(그 콤마는 인자 목록 것 → `ON coalesce(x,(1)||evil_fn(y))` 차단 유지) ② FROM 목록에 못 오는 절 키워드(SELECT/WHERE/BY/SET/VALUES/ORDER…)는 콤마 이후에도 항상 종료. 지적이 금지한 "ON/USING 를 목록에서 제거" 는 쓰지 않았다.
- 검증: go test ./... 전체 + gofmt/vet 통과, 양방향 되돌림 검증(수리 전이면 allow 실패, 괄호 해제 빼면 block 실패), 새 allow 6건 PG17 실제 실행 확인.
- 여전히 확신 없는 곳: (1) `IS [NOT] DISTINCT FROM` 은 이 브랜치 이전부터 왼쪽 탐색이 `FROM` 에 닿아 true 를 주므로 `WHERE x IS DISTINCT FROM (1) || evil_fn(y)` 가 면제된다 — 이번 지적 범위 밖이라 손대지 않았고 3줄 가드로 막을 수 있다. (2) `postgresClauseBoundaryKeywords` 분류의 완결성 — 잘못 넣으면 오탐(차단), 빼면 미탐 쪽이다. TABLESAMPLE 은 별도 규칙이 먼저 차단하므로 이 함수와 무관.
- 비평가가 릴리즈 노트로 남긴 dbexec `) fn(` 관계 차단 우려는 이번 수리로 `ON`/`AS` 뒤 콤마 케이스만 좁아졌고 나머지는 그대로다.
- [러너 00:21] repair done — # fix-summary (수리 1회차 → 커밋 b715a45)  - 문제(재현됨): `postgresFromItemContext` 의 왼쪽 탐색이 FROM 목록의 최상위 콤마를 경계로 보지 않아 앞 from-item 의 `ON`

## 비평 노트
- 확인: diff 는 보안상 엄격한 tightening 이다(54개 질의 A/B 프로브에서 HEAD-allow/main-block 0건). 새 테스트는 main 코드로 되돌리면 실패하므로 진짜 경로를 지난다. gofmt·vet·패키지 테스트 통과. 법무·프라이버시 차단 사유 없음.
- 거절 사유: `internal/domain/sqlsafe/sqlsafe.go:766` postgresFromItemContext 가 호출 지점마다 왼쪽 끝까지 재탐색(그룹도 한 토큰씩)하고 sqlsafe.go:447/453 루프 안에서 자리마다 불려 O(n^2) 이 된다. 유효한 `SELECT upper(x) || upper(x) || … FROM t` 기준 480KB 28ms→35.5s, runtimeapi/sql.go:49 가 허용하는 1MiB 에서 83ms→3m53.9s(약 2800배). 순수 CPU 라 ctx·statement timeout 이 안 걸리고 /sql/validate 는 `sql.generate` 만 있으면 DB 권한 없이 도달한다(sql.go:54 가 requireDatabaseAccess 보다 먼저).
- 수리가 먼저 볼 파일: `internal/domain/sqlsafe/sqlsafe.go` postgresFromItemContext. matching-left-paren 인덱스를 스캔 시 1회 O(n) 으로 만들어 그룹 건너뛰기를 O(1) 점프로 바꾸고, 왼쪽 탐색 결과를 cursor 기준으로 메모이즈할 것(최소한 탐색 토큰 수 상한 + 초과 시 false). 정확도(무엇을 허용/차단하는지)는 지금 동작을 그대로 보존해야 한다.
- 승인 시에도 남을 우려: (1) `WHERE x IS DISTINCT FROM (1) || evil_fn(y)` 는 여전히 면제 — main 도 같아 회귀는 아니지만 형제 케이스가 다 닫혀 겉보기엔 닫힌 듯 보인다. 수리 때 3줄 가드 동봉 권장. (2) dbexec/postgres_safety.go:418/430/439 경유로 `) fn(` 를 담은 뷰·파티션 키·RLS 정책이 관계째 ErrUnsafeDatabaseObject — 릴리즈 노트 필요.
- 못 본 것: dbexec 통합 테스트(폐기 DB 미준비)·e2e. 검증은 sqlsafe 단위 테스트 + main/HEAD 복사본 A/B 프로브 + 성능 측정까지. 임시 프로브는 /tmp 에만 만들었고 워크트리는 변경하지 않았다.
- [러너 00:32] review rejected — 리뷰 거절: internal/domain/sqlsafe/sqlsafe.go:766 postgresFromItemContext 는 호출 지점마다 토큰 스트림을 왼쪽 끝까지 다시 훑고, 균형 괄호 그룹을 한 토큰씩 되짚�
- [러너 00:32] pr created — https://github.com/hkjang/qurio/pull/23
