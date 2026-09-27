- 과제: SQL 기반 목록의 jqxgrid 어댑터 응답이 DB 라벨 대소문자를 그대로 내보내 그리드 셀이 비고 정렬이 조용히 무시되는 것을 고친다 (가치 3 / 위험 2 / 작업량 S)

- 왜: 같은 URL `GET /api/v1/data/lists/{id}` 의 두 백엔드가 응답 키 규약이 다르다 — `DataAdapterService.executeEntityBacked:199` 는 `lowerCaseKeys(rows)` 를 거치는데 `executeSqlBacked:153-158` 은 `sqlExecutor.executeQuery` 의 원본 행을 그대로 `Rows` 로 내보낸다. jqxgrid 는 `columnsJson` 의 소문자 `field` 를 `datafield` 로 쓰므로 H2/Oracle 처럼 라벨을 대문자로 주는 드라이버에서는 키가 어긋나 셀이 비고, 같은 이유로 `applySort:208` 의 `a.get(field)`(field = jqxgrid 가 보내는 소문자 `sortdatafield`)가 전부 null 이 되어 정렬이 예외 없이 no-op 가 된다. 이 저장소는 이미 "행 키의 대소문자는 드라이버마다 다르다" 를 두 곳에서 방어하고 있다(`ListExportController.lookup:366-371` 이 exact→UPPER→lower 로 찾고, `NexaUiService.maskRecordFor:860-862` 이 세 변형을 모두 치환하며 주석까지 달려 있다) — 즉 내보내기 3종은 SQL 기반 목록에서도 값을 찾아내는데 어댑터만 보정이 없다. **같은 값을 읽는 N 개 경로 중 하나만 규약에서 벗어나 있는** 이 저장소의 반복 패턴이고, 기존 헬퍼 하나를 통과시키면 끝난다.

- 수용 기준:
  1) `GET /api/v1/data/lists/{sqlBackedListId}` 의 `Rows[0]` 이 소문자 키(`id`, `name`)를 갖는다 — 즉 `jsonPath("$.Rows[0].name")` 이 값으로 존재한다.
  2) 같은 URL 에 `?sortdatafield=name&sortorder=desc` 를 주면 실제로 역순으로 온다(`Rows[0].name` 이 `row 1 …`). 수정 전에는 정렬이 무시되어 SQL 의 `ORDER BY id` 순서(`row 0 …`)가 그대로 나온다.
  3) 엔티티 백엔드 응답(`/data/lists/{entityBackedListId}`, `/data/entity/{id}`)과 `totalCount` 는 지금과 동일 — 기존 테스트가 회귀 없이 통과한다.
  4) **먼저 실패를 보고 고칠 것**: 새 테스트를 수정 전에 단독 실행해 1)·2) 가 빨간 것을 눈으로 확인한 로그를 회차 노트에 남긴다. (아래 "미확인" 참고 — H2 가 이 픽스처에서 실제로 대문자 라벨을 주는지는 이번 정찰에서 실행으로 확인하지 못했다. 만약 1) 이 처음부터 초록이면 이 과제는 성립하지 않으니 차선 후보로 넘어갈 것.)

- 건드릴 파일 (프로덕션 1개):
  - `src/main/java/com/nexabuilder/core/data/DataAdapterService.java:executeSqlBacked` — `var rs = sqlExecutor.executeQuery(...)` 로 받은 `rs.getRows()` 를 **기존 `lowerCaseKeys(rows)` 헬퍼(238-246줄)에 통과시킨다**. 새 헬퍼를 만들지 말 것, 엔티티 쪽이 쓰는 그 메서드를 그대로 재사용. 호출 순서는 `lowerCaseKeys` → `total = rows.size()` → `applySort` → `applyPaging` (정렬이 소문자 키를 보게 하려면 반드시 `applySort` 앞이어야 한다). `total` 은 페이징·정렬 전에 세는 현재 동작을 유지.
  - `src/test/java/com/nexabuilder/api/SqlBackedListDefinitionIntegrationTest.java` — 기존 픽스처(`seedSqlBackedList()`: `SELECT id, name FROM t_sqllist_<sfx> ORDER BY id`, 행 2개 `row 0 <sfx>` / `row 1 <sfx>`, `pageSize(10)`)를 그대로 재사용해 테스트 2건 추가: 소문자 키 검증, `sortdatafield=name&sortorder=desc` 역순 검증. 이 픽스처는 이미 실제 `SqlMetadata`/`SqlExecutor` 배선으로 도는 것이 2026-09-26 회차에 확인됐다(목 없음).

- 검증 명령:
  - 단독(수정 전 빨간 것 확인용, 그 다음 초록): `sh ./gradlew --no-daemon test --tests 'com.nexabuilder.api.SqlBackedListDefinitionIntegrationTest'`
  - 어댑터 회귀: `sh ./gradlew --no-daemon test --tests 'com.nexabuilder.api.DataAdapter*'`
  - 전체(수 분): `sh ./gradlew --no-daemon cleanTest test` — 직전 회차 590건 기준이므로 590+추가분·0 skip
  - 패키징: `sh ./gradlew --no-daemon bootJar -x test`
  - `gradlew` 는 실행 비트가 없으니 반드시 `sh ./gradlew` 로 실행. (정찰 단계에서는 gradlew 실행이 샌드박스 승인 대상이라 이번에 못 돌렸다.)

- 위험과 피할 것:
  - **`SqlExecutor.executeQuery` 를 고치지 말 것.** `NexaUiService.listData`·내보내기 3종·Groovy 플러그인이 같은 `RecordSet` 을 공유하므로 거기서 소문자화하면 블라스트 반경이 크고, 내보내기는 이미 `lookup` 으로 보정하고 있어 이득도 없다. 변경은 어댑터 경로 안에서 끝낼 것.
  - 따옴표로 대소문자를 고정한 별칭(`SELECT name AS "userName"`)을 쓰는 기존 SQL 정의는 이 URL 에서 키가 `username` 으로 바뀐다. 저장소 안에 그런 정의가 있는지는 **미확인**. 커밋 메시지에 이 계약 변화를 적을 것. Postgres/MySQL 은 따옴표 없는 식별자를 이미 소문자로 주므로 그 환경에서는 no-op 다.
  - `POST /api/v1/builder/lists/{id}/data`(→ `NexaUiService.listData`)도 SQL 기반 목록에서는 원본 키를 그대로 내보낸다. **이번 회차 범위 밖** — 그 응답의 프런트 소비자를 확인하지 않았고(미확인), 건드리면 `RecordSet` 계약과 파일 수가 늘어난다. 보류 아이디어로 남겨 뒀다.
  - `@DirtiesContext` 를 붙이지 말 것(2026-09-17 교훈: 컨텍스트 재생성으로 시간·힙·비동기 종료 경합 악화).
  - 성공 응답 봉투 `{Rows,totalCount}` 를 바꾸지 말 것. 이 URL 은 실패 시에만 `{success,message}` 라서 새 에러 경로를 넣으면 jqxgrid 어댑터가 파싱하지 못한다. 이번 과제는 400 을 새로 만들 이유가 없다.
  - 보호 경로(auth/session/permission/migrations/.github/workflows)는 건드리지 않는다. 이번 과제는 그 어느 것도 필요 없다.
  - 테스트 픽스처 관례: UUID 8자 접미사, 정리 없음(기존 파일 그대로).

- 미확인 (추측으로 적은 것):
  - H2 가 이 픽스처의 `SELECT id, name` 에 대해 `getColumnLabel` 을 `ID`/`NAME` 으로 주는지를 **실행으로 확인하지 못했다**(정찰 단계에서 gradlew 가 샌드박스에 막혔다). 근거는 저장소 안의 정황 3개다: `executeEntityBacked` 가 `jdbcTemplate.queryForList` 결과에 `lowerCaseKeys` 를 부르는 것(원본이 소문자가 아니라는 뜻), `entityDataFields:130` 이 `md.getColumnLabel(i).toLowerCase()` 를 하는 것, 그리고 `lookup`/`maskRecordFor` 의 대소문자 방어와 그 주석. 구현자는 수용 기준 4) 대로 먼저 빨간 것을 확인해 이 가정을 검증할 것.
  - 런타임 목록 화면이 실제로 빈 셀을 그리는지는 브라우저로 확인하지 않았다. 서버 응답 키 불일치까지만 확인했다.

- 차선 후보: **`applySort` 의 필드 조회만 대소문자 무시로 좁히기** — `lowerCaseKeys` 로 응답 키 전체를 바꾸는 것이 위 "따옴표 별칭" 위험 때문에 부담스럽다고 판단되거나 수용 기준 1) 이 처음부터 초록이면, `applySort:203-221` 안에서 `sortField` 를 행의 키집합과 대소문자 무시로 매칭해 실제 키를 한 번 해석한 뒤 `a.get(resolved)` 를 쓰도록 좁힌다(응답 키는 불변). 수용 기준 2)·3) 만 남기면 되고 파일도 같은 1개다.
- 3순위: `DataAdapterService.queryList` 의 sqlId 백엔드 목록에도 `requireLiveList` 회귀 테스트 추가(2/1/S) — 프로덕션 코드 0줄, 테스트만. 위 두 개가 모두 성립하지 않을 때의 안전한 착지점.
