# 회차 노트 2026-09-24-053432-nexabuilder-improve — nexabuilder
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 05:34] base pinned — master@014ace4
- [러너 05:34] autonomy release — 

## 정찰 노트
- soft-delete 4연작이 끝나 같은 우물이 말랐다. 남은 후보 중 권한 게이트(3/4/M)는 위험 구역, SQL 백엔드 DB 페이징(3/3/M)은 SqlExecutor 조사 선행이라 각각 제쳤고, 이전 두 회차가 "계약을 먼저 정해야 한다"며 미룬 페이지 입력 검증을 골랐다. 이번엔 그 계약을 과제서에서 **클램프**로 못박았다(호출자 4곳 전부 확인 — 음수·0 을 보내는 곳 없음, 성공/실패 봉투가 달라 400 추가가 어댑터를 깨뜨림).
- 추측으로 적은 것: `pagenum=-1` 이 **500** 이라는 예상 — 음수 OFFSET 을 H2 가 거절한다는 가정이다. 코드 경로(`parse` 하한 미검사 → `__offset` 음수 → `LIMIT/OFFSET`, `subList(음수,…)`)는 소스로 확인했지만 실제 응답은 실행하지 못했다(gradlew 가 샌드박스 승인 대상이라 이번 세션에서 테스트를 못 돌림 — 이것도 미확인 항목).
- 구현자가 조심할 것: (1) 반드시 고치기 전에 새 테스트 3건을 먼저 돌려 **실제 실패 모드를 기록**할 것 — 2026-09-22 회차처럼 과제서의 예상 상태코드가 픽스처 때문에 어긋날 수 있다. (2) `parse` 한 곳에서만 고칠 것, `queryList`/`queryEntity` 양쪽이 같은 줄을 지난다. (3) `MAX_PAGE_SIZE` 상한과 null pageSize→100 fallback 은 직전 회차가 방금 고정한 계약이니 건드리지 말 것. (4) `@DirtiesContext` 금지.
- [러너 05:38] scout done — `DataAdapterService.parse` 의 페이지 입력(pagenum 음수 / pagesize 0·음수 / pagenum 오버플로)을 **클램프**로 고쳐 `/api

## 구현 노트
- 무엇/왜: `DataAdapterService.parse` 가 페이지 파라미터의 하한을 안 봐서 생긴 세 구멍을 클램프로 막았다(커밋 7b00675). `parse` 한 곳만 고쳐 `queryList`/`queryEntity` 가 같은 규칙을 쓰고, 오프셋은 `QueryRequest.offset()` 의 long 산술로 두 백엔드가 공유한다.
- 정찰이 미확인이라 한 것 → 실행으로 확인: H2 는 음수 OFFSET 을 거절한다. 고치기 전 `pagenum=-1&pagesize=10` 과 `pagenum=2147483647&pagesize=10` 은 **둘 다 offset -10** 으로 `JdbcSQLDataException: Invalid value "-10" for parameter "result OFFSET"` → `DataIntegrityViolationException` → 500 이었고, `pagesize=0` 은 200 + 빈 Rows 였다. 세 테스트가 빨간 것을 보고 고쳐 초록.
- 확신 없는 곳: (1) 신규 테스트 3건은 전부 `/data/entity/{id}`(엔티티 백엔드) 다 — `/data/lists/{id}` 와 `executeSqlBacked`/`applyPaging` 의 자바 페이징 경로는 **같은 `parse` 를 지나므로 코드상 같은 보호를 받지만 실행으로 확인하지 않았다**(ideas.json 에 후속 항목으로 남김). (2) 삭제한 `applyPaging` 의 `pageSize <= 0` 가드가 도달 불가라는 근거는 "모든 호출자가 `parse` 를 지난다" 는 것뿐이다 — grep 으로 `applyPaging` 호출부가 `executeSqlBacked` 하나임은 확인했다. (3) `queryCodes` 는 페이징을 아예 안 해서 범위 밖이다.
- 일부러 안 한 것: 400 거절(성공/실패 봉투가 달라 jqxgrid 어댑터가 깨진다 — Javadoc 에 근거를 남겼다), `executeSqlBacked` 의 전체 행 읽기를 DB 페이징으로 바꾸기(과제서가 범위 밖이라 명시), 권한 게이트 추가(보호 구역).
- 다음 역할이 조심할 것: 테스트는 시드된 `USER` 엔티티만 쓰고 픽스처를 만들지 않는다(H2 필요, `@DirtiesContext` 없음 — 붙이지 말 것). `pagesize=0` 의 fallback 값은 경로마다 다르다 — entity 경로는 100 고정이지만 lists 경로는 `list.pageSize` 다. 검증: `sh ./gradlew --no-daemon cleanTest test` 전체 **581건 통과·0 skip·0 실패**(직전 578 + 신규 3), `bootJar -x test` 성공.
- [러너 05:48] brief accepted — 채택 — 근거 3가지(`parse` 가 하한 미검사, `applyPaging` 의 `pageSize<=0` 상한 우회, `pageNum*pageSize` int 오버플로)가 코드와 전부 
- [러너 05:48] verify passed — 검증 1개 통과 (auto)

## 비평 노트
- 확인한 것: master 판 `DataAdapterService.java` 로 되돌려 `DataAdapterIntegrationTest` 실행 → 신규 3건 전부 FAILED, HEAD 복원 후 전부 통과(작업 트리 clean). 테스트가 수정 전 코드를 실제로 잡는다.
- 코드로 확인한 것: `parse` 의 fallback 이 항상 >0 이라 pageSize ∈ [1,1000]·pageNum ≥ 0 보장 → `applyPaging` 주석과 일치. `offset()` 최대 ~2.1e12 로 long 안전, `(int)` 캐스트는 `rows.size()` 상한 뒤라 절단 없음. `applyPaging` 호출자는 `executeSqlBacked` 하나(삭제한 가드는 도달 불가).
- 못 본 것: `/api/v1/data/lists/{id}` 의 SQL 백엔드를 실행으로 확인하지 못했다 — 커밋이 가장 무겁게 서술한 `pagesize=0` MAX_PAGE_SIZE 우회가 그 경로인데 테스트 3건은 전부 엔티티 백엔드다. 같은 `parse` 를 지나 코드상은 닫힌다.
- 승인이어도 남는 우려: `executeSqlBacked` 는 여전히 전체 행을 메모리로 읽고 자바에서 자른다(응답만 1000행 제한, 서버측 로드는 그대로). 릴리즈 노트에는 "잘못된 페이지 파라미터가 500 대신 첫 페이지로 클램프된다"는 동작 변화를 적을 것.
- 다음 회차 후속: sqlId 기반 목록으로 `pagesize=0` 회귀 테스트 추가, 그리고 SQL 백엔드의 DB 페이징 전환.
- [러너 05:51] review approved — 리뷰 승인 (risk=low)
- [러너 05:51] pr created — https://github.com/hkjang/nexabuilder/pull/30
- [러너 06:00] ci passed — 검사 2개 모두 success
- [러너 06:00] merge done — 7b00675
- [러너 06:18] release published — v1.21.0
- [러너 06:20] assets verified — v1.21.0 자산 1개 (이전 v1.20.0: 1)
