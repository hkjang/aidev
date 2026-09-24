# 정찰 과제서 (2026-09-24)

- 과제: `DataAdapterService.parse` 의 페이지 입력(pagenum 음수 / pagesize 0·음수 / pagenum 오버플로)을 **클램프**로 고쳐 `/api/v1/data/*` 가 500 과 "무한 페이지" 로 빠지지 않게 한다 (가치 3 / 위험 2 / 작업량 S)

- 왜: `parse` 는 `pagenum`/`pagesize` 의 하한을 전혀 검사하지 않는다 — `q.pageNum = intOr(params.get("pagenum"), 0)` 은 `-1` 을 그대로 받고, `q.pageSize = Math.min(intOr(...), MAX_PAGE_SIZE)` 는 상한만 본다. 그 값이 `executeEntityBacked` 에서 `params.addValue("__offset", req.pageNum * req.pageSize)` 로 `LIMIT :__limit OFFSET :__offset` 에 들어가고(`SqlPagination.limitOffset`), `executeSqlBacked` 쪽에서는 `applyPaging` 의 `rows.subList(from, to)` 에 들어간다. 그래서 ① `pagenum=-1` 은 음수 OFFSET/음수 subList 인덱스로 500, ② `pagesize=0` 은 엔티티 백엔드에서 `LIMIT 0` 으로 조용히 빈 Rows, SQL 백엔드에서는 `applyPaging` 의 `if (req.pageSize <= 0) return rows;` 때문에 **MAX_PAGE_SIZE 상한을 통째로 우회해 전체 행**을 돌려준다, ③ `pagenum` 이 크면 `pageNum * pageSize` 가 int 오버플로로 엉뚱한(음수일 수 있는) 오프셋이 된다. 한 줄짜리 클램프로 세 경로가 모두 같은 규칙을 따르게 되고, 인증만 통과하면 누구나 쿼리 문자열 하나로 500 이나 전체 테이블 읽기를 유발할 수 있는 입구가 닫힌다.

- 계약 결정(이전 회차가 미뤘던 것 — 이번엔 이걸로 간다): **400 으로 거절하지 말고 클램프한다.**
  - `pagenum < 0` → `0`
  - `pagesize <= 0` → 기존 fallback(`list.pageSize` 유효하면 그것, 아니면 100), 그 뒤 기존대로 `MAX_PAGE_SIZE(1000)` 상한 적용
  - 오프셋은 `long` 으로 계산해 int 오버플로를 없앤다
  - 근거: (a) 실제 호출자 중 음수·0 을 보내는 곳이 없다 — `src/main/resources` 전체에서 `pagenum`/`pagesize` 를 만드는 곳은 `static/js/nexa-data.js:101`(`if (spec.pageSize) src.pagesize = spec.pageSize;` — 0 이면 아예 안 붙음), `templates/admin/entity.html:549`, `templates/admin/system-console.html:986`(둘 다 `pagenum=0&pagesize=50` 고정), `templates/runtime/list.html:838`(jqxgrid 가 생성) 뿐이다. (b) 이 URL 은 성공 시 `{Rows,totalCount}`, 실패 시 `{success,message}` 로 **봉투가 달라서** 새 400 경로를 늘리면 jqxgrid 어댑터가 파싱하지 못한다(`DataAdapterController` 클래스 주석에 이 설계가 적혀 있음). (c) 400 으로 바꾸면 `pagesize=0` 을 "전부 달라" 로 쓰는 외부 호출자가 있을 경우 깨지지만, 클램프는 페이지 한 장을 돌려주므로 실패하지 않는다.

- 수용 기준:
  1) `GET /api/v1/data/entity/USER?pagenum=-1&pagesize=10` 이 200 이고 `pagenum=0` 과 같은 Rows(첫 페이지)를 돌려준다. **고치기 전에 먼저 돌려 빨간 것을 확인하고 실제 실패 모드(상태코드/예외)를 커밋 메시지나 회차 요약에 적을 것** — 과제서는 이것을 500 으로 예상하지만 H2 의 음수 OFFSET 거절 여부를 이번 정찰에서 실행으로 확인하지 못했다(미확인).
  2) `GET /api/v1/data/entity/USER?pagenum=0&pagesize=0` 과 `...&pagesize=-5` 가 200 이고 **빈 Rows 가 아니라** fallback 크기(100)의 첫 페이지를 돌려준다. `totalCount` 는 두 경우 모두 정상 호출과 같다.
  3) `GET /api/v1/data/entity/USER?pagenum=2147483647&pagesize=10` 이 200 이고 `Rows` 가 빈 배열, `totalCount` 는 정상값이다(오프셋이 int 오버플로로 음수가 되지 않는다는 증명).
  4) 기존 `DataAdapterIntegrationTest` 5건과 `DataAdapterListSoftDeleteIntegrationTest` 3건이 그대로 통과한다 — 특히 `explicitPageSizeIsHonored`(pageSize 1 → Rows 1, totalCount 2)와 `liveListServesRowsWithNullPageSize`(null → 100) 가 변하지 않아야 한다. 클램프가 기존 fallback 분기를 덮어쓰지 않았다는 증거다.

- 건드릴 파일:
  - `src/main/java/com/nexabuilder/core/data/DataAdapterService.java:parse(Map,Integer)` — `q.pageNum` 에 `Math.max(0, …)`, `q.pageSize` 는 `int requested = intOr(params.get("pagesize"), fallback); q.pageSize = Math.min(requested > 0 ? requested : fallback, MAX_PAGE_SIZE);` 꼴로. 기존 `fallback` 계산(`defaultPageSize != null && defaultPageSize > 0 ? defaultPageSize : 100`)과 `MAX_PAGE_SIZE` 상한은 그대로 유지. 메서드 Javadoc 에 왜 거절이 아니라 클램프인지 한 줄 추가.
  - 같은 파일 `executeEntityBacked` — `params.addValue("__offset", req.pageNum * req.pageSize)` 를 `long` 산술로(`(long) req.pageNum * req.pageSize`). 오프셋 계산을 `applyPaging` 과 공유하려면 `QueryRequest` 에 `long offset()` 를 두는 것도 좋다(선택).
  - 같은 파일 `applyPaging` — `int from = Math.min(req.pageNum * req.pageSize, rows.size())` 도 long 으로 계산한 뒤 `(int)` 로 좁힌다. `if (req.pageSize <= 0) return rows;` 가드는 클램프 뒤로는 도달 불가가 되므로 지우거나, 남긴다면 "parse 가 이미 보장한다" 는 주석을 붙일 것(죽은 조건을 말없이 남기지 말 것).
  - `src/test/java/com/nexabuilder/api/DataAdapterIntegrationTest.java` — 수용 기준 1~3 의 테스트 3건 추가. 이 클래스는 **시드된 `USER` 엔티티만 쓰고 픽스처를 만들지 않으므로** 새 테스트 클래스를 만들 필요가 없다(`@SpringBootTest @AutoConfigureMockMvc @WithMockUser(roles="ADMIN")` 이미 붙어 있음). `@DirtiesContext` 는 절대 붙이지 말 것 — 컨텍스트 캐시가 깨진다(2026-09-17 회차 교훈).

- 검증 명령:
  - 대상만: `sh ./gradlew --no-daemon test --tests 'com.nexabuilder.api.DataAdapter*'`
  - 전체: `sh ./gradlew --no-daemon cleanTest test` (수 분, 직전 회차 기준 578건)
  - 패키징: `sh ./gradlew --no-daemon bootJar -x test`
  - (주의: 이번 정찰 세션에서는 이 명령들을 실행하지 못했다 — 샌드박스 승인 필요. 명령 자체는 직전 3회차에서 쓰인 것 그대로다.)

- 위험과 피할 것:
  - `queryList` 와 `queryEntity` 가 같은 `parse` 를 쓴다. 수용 기준의 테스트는 `/data/entity/{id}` 로 검증하지만 `/data/lists/{id}` 도 같은 줄을 지난다 — 한쪽만 고치는 일이 생길 수 없게 **반드시 `parse` 한 곳에서** 고치고, 컨트롤러나 호출부에서 값을 미리 보정하지 말 것(같은 값을 읽는 경로가 여럿일 때 한쪽만 넓히지 말라는 운영자 규칙).
  - `MAX_PAGE_SIZE` 상한과 null `page_size` → 100 fallback 은 2026-09-22 회차에서 방금 고친 계약이다. 그 동작을 바꾸지 말 것(기존 테스트 2건이 이것을 고정하고 있다).
  - `requireLiveList` 게이트(soft-delete)는 `parse` 보다 앞에 있다. 순서를 바꾸지 말 것 — 휴지통 목록은 페이지 파라미터를 보기 전에 400 이어야 한다.
  - 보호 경로 금지: `core/auth`·`core/session`·`infra/security`·`core/permission`·`resources/db/migration`·`.github/workflows` 는 이번 과제와 무관하니 열지 말 것.
  - `executeSqlBacked` 가 전체 행을 메모리로 읽는 구조 자체(`sqlExecutor.executeQuery` → 자바 페이징)는 이번 범위 밖이다. `pagesize<=0` 우회를 막는 것까지만 하고 DB 페이징으로 바꾸지 말 것.

- 차선 후보: `DataAdapterService.queryList` 가 `sqlId` 백엔드 목록에서도 `requireLiveList` 를 타는지 회귀 테스트 추가 (2/1/S) — 현재 신규 테스트 3건은 모두 `entityId` 백엔드다. 코드상 판정이 백엔드 분기보다 앞(queryList 첫 줄)이라 막히는 것이 맞지만 실행으로 확인된 적이 없다. `SqlMetadata` 픽스처가 필요하다.
