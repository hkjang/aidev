# PR 처리기 노트 2026-09-28-183826-qurio-shepherd — qurio PR #27
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-09-28-174216-qurio-improve)
# 회차 노트 2026-09-28-174216-qurio-improve — qurio
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 17:42] base pinned — main@d5eaf8d
- [러너 17:42] autonomy release — 
- [러너 17:51] scout timeout — 단계 제한 시간 초과
- [러너 17:51] scout done — sqlsafe PostgreSQL 게이트가 `WHERE a.b = evil_fn(...)` 처럼 **점 표기 좌변 뒤의 함수 호출을 alias 컬럼 목록으로 오�

## 구현 노트
- 무엇/왜: `postgresAliasColumnList` 마지막 분기(sqlsafe.go:770)가 `word . word 이름(` 을 전부 alias 컬럼 목록으로 단정해 FROM 문맥 없이 면제했다. 연산자가 토큰으로 안 나오므로 `WHERE a.b = dblink_exec(...)` 이 같은 형태가 되어 허용목록을 통째로 건너뛰고 readOnly/low 로 통과했다. 760행 `)` 분기가 이미 쓰는 `index.fromItemContext(nameIndex-1)` 를 붙였다(프로덕션 +6/-2, 파일 1개). 커밋 e1af783.
- 확신 없는 곳/검증 못 한 것: (1) Oracle 실서비스 경로는 돌려보지 못했다 — 도커 PostgreSQL 17 만 띄웠다. 다만 이 분기는 PostgreSQL 전용이고 스윕에서 Oracle 판정 57건이 바이트 단위로 동일했다. (2) `make test-web`·`make test-e2e`·`make build` 는 돌리지 않았다(SPA·릴리즈 경로를 건드리지 않았다). (3) `fromItemContext` 의 memo 는 `(cursor, crossedFromItemComma)` 상태에 걸리는데, 내 호출이 새 시작점을 추가하므로 캐시 상호작용은 기존 `)` 분기와 같은 성질이라고 보았다 — 기존 성능 회귀 테스트(sqlsafe_test.go:364 부근)가 통과하는 것으로만 확인했다.
- 일부러 안 한 것: `AS` 분기(763행)는 같은 냄새가 나지만 악용 경로가 미확인이라 손대지 않았다(ideas.json 에 조사 과제로 남김). `WITH ORDINALITY g(v,i)` 선재 오탐도 범위 밖 — 스윕에서 수정 전/후 판정이 동일함을 확인했다. 허용목록·Oracle 경로·`postgresCallAdjacent`·CTE 분기는 한 줄도 안 건드렸다.
- 다음 역할이 조심할 것: 통합 테스트는 DB 가 있어야 돈다. 55432/55433/55439 가 점유돼 55472 를 썼다(`POSTGRES_DSN`·`QURIO_INTEGRATION_DSN`·`QURIO_TEST_POSTGRES_DSN` 셋 다 같은 폐기 DB, `cmd/qurio` 마이그레이션 테스트로 bootstrap 후 `-race -p=1`). 컨테이너는 제거했다. 재현하려면 `-tags=integration ./...` 를 깨끗한 DB 에서 돌릴 것.
- [러너 18:00] brief accepted — 채택 — 과제서가 지목한 행(770-771 의 마지막 `return`, 760 의 `)` 분기, 869 의 `fromItemContext`, 456 의 `continue`)이 모두 코드와 정
- [러너 18:02] verify passed — 검증 9개 통과 (auto)

## 비평 노트
- 확인: main 코드 + HEAD 테스트로 실패 재현이 원장 출력과 동일함을 직접 재현(`dblink_exec` readOnly/low 통과) → 새 테스트는 수정 전 코드에서 실제로 실패한다. 공격 4건 blocked 로 뒤집힘, 정상 쿼리 23건·대조 2건 판정 불변, gofmt·vet·`go test ./...` 통과.
- 구현자 미검증 3건 해소: Oracle 은 호출자가 postgres 분기(456) 하나뿐이라 코드상 무영향; SPA·릴리즈 경로 미접촉; memo 는 점표기 호출 20000개로 pre/post 둘 다 선형(21.9→27.9ms) — 2차 폭발 없음.
- 승인이어도 남는 우려(다음 회차 최우선): `SELECT g FROM public.a x, LATERAL unnest(string_to_array(x.n || dblink_exec('h','DROP TABLE v'), ',')) g` 가 여전히 readOnly/low 로 통과하며 `RequireReadOnlyDialect` 도 nil. 원인은 `fromItemContext`(sqlsafe.go:909-910)가 좌괄호에서 `crossedFromItemComma` 를 지워 FROM 절 함수 인자 안의 호출이 바깥 FROM/LATERAL 문맥을 빌리는 것. PostgreSQL 17 에서 실제 실행·파일 내용 반환까지 확인(컨테이너 제거). **main 과 판정이 동일한 선재 결함이라 이번 PR 의 차단 사유는 아니다** — 이 PR 은 좁히기만 한다.
- 못 본 것: 통합 테스트·`make test-web`·`make test-e2e`·`make build` 는 돌리지 않았다(diff 가 Go 파일 1개+테스트뿐). Oracle 실서비스도 미기동(호출 그래프로 대체).
- 릴리즈 노트: 사용자 영향은 "점 표기 좌변 뒤 함수 호출이 이제 허용목록을 거친다" 뿐. 기존에 통과했던 `a.b = 사용자함수(...)` 쿼리는 이제 차단되니 릴리즈 노트에 명시할 것.
- [러너 18:14] review approved — 리뷰 승인 (risk=low)
- [러너 18:14] pr created — https://github.com/hkjang/qurio/pull/27
- [러너 18:33] ci timeout — 제한 시간 안에 CI 완료를 확인하지 못함

## 심사 노트
- 확인: main 코드 + HEAD 테스트로 새 거부 테스트가 실제로 실패함을 재현(`public.events.name = dblink_exec(...)` 가 readOnly/low 통과) → 테스트가 변경을 진짜 검증한다. 스윕 18건을 main/HEAD 비교해 16건 판정 동일, 의도한 2건만 뒤집힘(pg_sleep·lo_export readOnly/medium → blocked). main 에서 `lo_export` 가 RequireReadOnlyDialect 를 통과했던 것이 이 PR 이 닫는 실제 구멍이다.
- 확인: 정상 FROM 별칭 컬럼 목록(콤마 목록·JOIN·ONLY·CROSS JOIN·따옴표 별칭·CTE 내부)과 허용목록 함수 판정 전부 불변. 점표기 호출 2000/8000/20000 개 스케일링이 main 과 동일(선형, 30 vs 34ms). gofmt·go vet ./...·go test ./... 전체 통과, `go vet -tags=integration ./...` 컴파일 통과.
- 못 본 것: 실 DB 통합 테스트·Oracle 실서비스·make test-web/test-e2e/make build 미실행. 대신 게이트를 참조하는 테스트 2개가 모두 integration 태그 없이 통과함과, 저장소 SQL 픽스처에 `x.y = fn(` 형태가 없음을 확인해 대체했다. Go 파일 1개만 바뀌고 릴리즈·빌드·SPA 경로 미접촉.
- 근거: 프로덕션 변경은 sqlsafe.go:766-775 의 +6/-2 뿐 — 760 행 `)` 분기가 이미 쓰는 `fromItemContext(nameIndex-1)` 를 같은 계약으로 한 항 추가했다. 마이그레이션·auth·의존성 미접촉이라 revert 로 완전히 되돌아온다. 면제를 넓히지 않고 좁히기만 하므로 보안·법무 차단 소견 없음.
- 선재 결함(차단 사유 아님): `FROM public.a x, LATERAL unnest(string_to_array(x.n || dblink_exec(...), ',')) g` 는 main·HEAD 판정이 동일하게 readOnly/low — 원인 sqlsafe.go:909-910 의 좌괄호에서 `crossedFromItemComma` 리셋. 다음 회차 최우선. 권고 merge, risk=medium(SQL 정책 게이트).
