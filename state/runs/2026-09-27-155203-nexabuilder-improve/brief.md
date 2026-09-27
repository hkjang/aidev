- 과제: `POST /api/v1/builder/lists/{listId}/data` 가 `_offset`/`_limit` 를 컬럼 필터로 취급해 런타임 대체 뷰 7종(칸반·차트·피벗·간트·캘린더·지도·트리)이 조용히 빈 화면이 되는 것을 고친다 (가치 4 / 위험 2 / 작업량 S)

- 왜: 런타임 대체 뷰 7개가 모두 이 엔드포인트에 `{ _offset: 0, _limit: N }` 본문을 POST 하는데(`runtime/chart.html:103` `_limit:5000`, `pivot.html:101` 5000, `tree.html:195` 5000, `map.html:146` 1000, `gantt.html:134` 1000, `kanban.html:181` 500, `calendar.html:133` 500), `NexaUiService.listData` 는 그 맵을 그대로 `entityService.list(table, params, 0, pageSize)` 의 **filters** 로 넘긴다. `EntityService.list:180-190` 은 맵의 모든 키로 `WHERE <key> = :<key>` 를 만들고 `sanitizeIdentifier:205-211` 의 정규식 `^[a-zA-Z_][a-zA-Z0-9_]*$` 은 선행 밑줄을 통과시키므로, 엔티티 기반 목록은 존재하지 않는 컬럼 `_offset`/`_limit` 을 WHERE 에 넣어 SQL 문법 오류로 죽는다. 뷰는 `catch (e) { /* fallback */ }` 로 삼켜 행 0건으로 렌더하므로 사용자에게는 "대체 뷰에 데이터가 하나도 안 나온다" 로 보인다. 같은 프로토콜 파라미터를 읽는 다른 경로 `EntityCrudController:53-59` 는 `where.remove("_offset")`/`where.remove("_limit")` 로 명시적으로 걷어내며 주석에 "`_offset` / `_limit` are protocol-level, not column filters" 라고 적어 두었다 — 두 리더의 계약이 어긋난 것이 소스로 확정된다. 덤으로 `listData` 는 offset 을 0 으로 하드코딩하고 limit 을 `list.getPageSize()`(null→100)로 고정하므로, 고친 뒤에는 차트·피벗이 페이지 1장(기본 100행)만 집계하던 문제도 같이 사라진다.

- 수용 기준:
  1) 엔티티 기반 목록에 `{"_offset":0,"_limit":500}` 본문으로 POST 하면 **200** 과 씨딩한 행이 돌아온다. 구현자는 **수정 전에 이 테스트를 먼저 단독 실행해 빨간 것과 실제 예외를 확인**하고(예상: `BadSqlGrammarException`/H2 "Column \"_OFFSET\" not found" → 500. `GlobalExceptionHandler` 에는 `DataAccessException` 계열 핸들러가 없어 400 으로 접히지 않는다 — 2026-09-24 회차에서 `DataIntegrityViolationException`→500 으로 이미 실행 확인됨) 기대값을 실제 동작에 맞춘다.
  2) `_offset`/`_limit` 이 실제 페이징으로 동작한다: 행 3건을 씨딩하고 `{"_limit":2}` → 2건, `{"_offset":2,"_limit":2}` → 1건. `_limit` 이 없거나 0 이하이면 기존 fallback(`list.getPageSize()`, null 이면 **100**)을 그대로 쓴다(이 100 은 `DataAdapterService.parse` 와 공유하는 값이므로 바꾸지 말 것).
  3) 진짜 컬럼 필터는 그대로 동작한다: `{"name":"<씨딩값>","_limit":10}` → 그 행만 돌아온다. 프로토콜 키 2개만 걷어내고 나머지 필터는 손대지 않음을 증명한다.
  4) SQL 기반 목록(`sql_id` 만 있고 `entity_id` 는 null)도 같은 본문으로 200 이다. 이 경로는 `sqlExecutor.executeQuery(sqlId, params)` 로 가고 여분 파라미터가 무해한지는 **미확인**이므로, 정리된 맵을 넘기는 쪽으로 고치고 회귀 테스트로 고정한다. 픽스처는 `SqlBackedListDefinitionIntegrationTest` 의 `SqlMetadata` 씨딩을 그대로 베낀다.
  5) 기존 테스트 592건 전부 통과·0 skip, `bootJar -x test` 성공.

- 건드릴 파일 (프로덕션 1개 + 테스트 1개):
  - `src/main/java/com/nexabuilder/core/ui/NexaUiService.java` — `listData(String listId, Map<String,Object> params)` (약 129~139행). params 의 **복사본**(`new LinkedHashMap<>(params)`)을 만들어 `_offset`/`_limit` 을 꺼내 정수로 해석한 뒤 맵에서 제거하고, 엔티티 경로에 `entityService.list(entity.getTableName(), filters, offset, limit)` 로 넘긴다. SQL 경로(`sqlExecutor.executeQuery`)에도 정리된 맵을 넘긴다. 호출자가 넘긴 맵을 제자리에서 변형하지 말 것(컨트롤러가 `@RequestBody` 맵을 그대로 준다). 키 이름·의미는 `EntityCrudController:53-59` 와 동일하게 `_offset`/`_limit` 만 — 새 이름을 만들지 말 것.
  - `src/test/java/com/nexabuilder/api/BuilderListDataPagingIntegrationTest.java` (신규) — 실제 H2·MockMvc·실제 배선, 목이나 손으로 만든 대역 금지. `BuilderListDataSoftDeleteIntegrationTest`(같은 디렉터리)의 픽스처 모양을 그대로 베낄 것: `@SpringBootTest @AutoConfigureMockMvc @WithMockUser(roles={"ADMIN"})`, `DynamicEntityRepository`/`NexaListRepository`/`EntityService`/`SchemaDdlService` 주입, `UUID` 8자 접미사, 정리 없음.
  - **건드리지 말 것**: `EntityService.list` 의 서명·`sanitizeIdentifier` 정규식(다른 호출자 다수), `RecordSet`(내보내기·Groovy·워크플로가 공유), `runtime/*.html` 7개 템플릿, `DataAdapterService`.

- 검증 명령:
  - 수정 전 빨간 것 확인: `sh ./gradlew --no-daemon test --tests 'com.nexabuilder.api.BuilderListDataPagingIntegrationTest'`
  - 전체: `sh ./gradlew --no-daemon cleanTest test` (수 분, 테스트 힙 2g)
  - 패키징: `sh ./gradlew --no-daemon bootJar -x test`
  - `gradlew` 에 실행 비트가 없으므로 `sh` 경유가 필요하다(이번 회차에도 재확인).

- 위험과 피할 것:
  - `@DirtiesContext` 금지 — 2026-09-17 에 컨텍스트 재생성이 힙·비동기 종료 경합을 악화시킨 전력.
  - `_limit` 상한을 `DataAdapterService.MAX_PAGE_SIZE`(**1000**, `DataAdapterService.java:51` 에서 확인)로 재사용하지 말 것. 차트·피벗·트리가 5000 을 요청하므로 1000 으로 자르면 부분 집계라는 다른 버그가 된다. 상한을 두려면 이 파일 안에 5000 이상의 별도 상수를 두고 이유를 주석으로 적거나, 상한 없이 두고 그 선택을 커밋 메시지에 남길 것.
  - 두 계약을 통합하려 하지 말 것: `/api/v1/data/lists/{id}`(`DataAdapterService.parse`)는 `pagenum`/`pagesize` 라는 다른 파라미터 이름을 쓴다. 이번 회차는 `/builder/lists/{id}/data` 의 `_offset`/`_limit` 만 다루고 두 경로를 합치지 않는다.
  - 템플릿의 `catch (e) { /* fallback */ }` 를 "오류를 표시" 로 바꾸지 말 것 — 파일 7개를 건드리게 되고 범위 밖이다.
  - 보호 경로(auth/session/permission/db/migration/.github/workflows) 에는 닿지 않는다 — 닿게 되면 범위가 틀어진 것이다.
  - 소스 문자열 검사만으로 증거를 삼지 말 것: 위 진단은 수정 전 실패 실행으로 반드시 재현해 노트에 예외 전문을 남길 것.

- 차선 후보: `DataAdapterService.queryList` 의 `sqlId` 백엔드 목록에도 `requireLiveList` 회귀 테스트 추가 — 기존 soft-delete 테스트 전부가 `entityId` 백엔드라 SQL 백엔드는 커버가 없다. 프로덕션 0줄, `SqlBackedListDefinitionIntegrationTest` 의 sqlId 픽스처에 `deleted_at` 만 찍어 400 을 확인하면 끝 (2/1/S).
