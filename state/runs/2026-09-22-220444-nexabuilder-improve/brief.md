# 과제서 (2026-09-22 정찰)

- 과제: `GET /api/v1/data/lists/{listId}` (jqxgrid 어댑터)가 ① 휴지통 목록의 행을 그대로 돌려주는 구멍을 막고 ② `pageSize` 가 null 인 목록에서 NPE(500) 로 터지는 것을 고친다 (가치 4 / 위험 2 / 작업량 S)

- 왜: `DataAdapterService.queryList` (src/main/java/com/nexabuilder/core/data/DataAdapterService.java:62-64) 는 `listRepository.findById` 만 하고 `deletedAt` 을 보지 않는다. 정의 조회·런타임 화면·`/builder/lists/{id}/data`(fa42f66)·내보내기 3종(3bd46b2)은 모두 `NexaUiService.requireLiveList` 로 "List not found" 를 내는데, 이 어댑터 URL 하나만 남아 있어 운영자가 휴지통에 넣은 목록의 행을 계속 읽을 수 있다. 같은 줄(:65)에서 `parse(requestParams, list.getPageSize())` 가 `Integer` 를 `int` 파라미터로 언박싱하므로 `page_size` 가 null 인 목록은 이 엔드포인트에서 무조건 NullPointerException → 500 이다(`NexaUiService.listData` 는 :110 에서 `getPageSize() == null ? 100` 으로 이미 막고 있다 — 같은 값을 두 경로가 다르게 읽는 상태).

- 수용 기준:
  1) 살아 있는 목록에 `GET /api/v1/data/lists/{listId}` 를 하면 `pageSize` 가 null 이어도 200 과 `{"Rows":[...],"totalCount":1}` 이 온다(지금은 500).
  2) `deleted_at` 이 찍힌 목록에 같은 요청을 하면 400 JSON 과 `message == "List not found: <listId>"` 로 `GET /api/v1/builder/lists/{id}` · `POST /api/v1/builder/lists/{id}/data` 와 글자까지 같은 응답이 온다(지금은 200 + 행).
  3) 새 통합 테스트가 위 두 가지를 실제 H2·실제 컨트롤러·MockMvc 로 증명한다. 목이나 서비스 직접 호출이 아니라 HTTP 경로를 탄다. 수정 전에 단독 실행해 빨간 것(1은 500, 2는 200)을 먼저 확인하고, 고친 뒤 초록, 되돌려 다시 빨간 것까지 확인할 것.

- 건드릴 파일:
  - `src/main/java/com/nexabuilder/core/data/DataAdapterService.java`
    - `queryList` — `listRepository.findById(...).orElseThrow(...)` 를 `uiService.requireLiveList(listId)` 로 교체. 예외 타입(`IllegalArgumentException`)·문구(`"List not found: " + listId`)는 헬퍼가 이미 동일하게 던지므로 그대로 두면 400 매핑이 유지된다.
    - 필드 — `NexaUiService uiService` 를 `@RequiredArgsConstructor` 생성자 주입으로 추가. `NexaListRepository listRepository` 가 다른 곳에서 안 쓰이면 필드·import 제거(3bd46b2 에서 `ListExportController` 를 그렇게 정리했다).
    - `parse(Map<String,String> params, int defaultPageSize)` → 두 번째 인자를 `Integer` 로 바꾸고 내부에서 `defaultPageSize != null && defaultPageSize > 0 ? defaultPageSize : 100` 로 판정. `queryEntity` 의 `parse(params, 100)` 호출은 그대로 컴파일된다. (기존 `defaultPageSize > 0 ? defaultPageSize : 100` 분기를 null 까지 확장하는 것이지 새 정책이 아니다.)
  - `src/test/java/com/nexabuilder/api/DataAdapterListSoftDeleteIntegrationTest.java` (신규)
    - `src/test/java/com/nexabuilder/api/BuilderListDataSoftDeleteIntegrationTest.java` 의 `seedList()` 를 그대로 본떠 쓴다(UUID 8자 접미사, `DynamicEntityRepository.save` → `SchemaDdlService.ensureTable` → `EntityService.insert` → `NexaListDef.builder()...build()` → `NexaListRepository.save`, 정리 없음). **그 픽스처는 `.pageSize(...)` 를 안 부르므로 pageSize 가 null 이다 — 그래서 이 테스트가 NPE 도 같이 증명한다. 일부러 null 로 두고, 명시적으로 pageSize 를 넣은 케이스를 하나 더 두면 회귀 폭이 넓어진다.**
    - 테스트 3개 권장: `liveListServesRowsWithNullPageSize`(200, `$.totalCount == 1`), `softDeletedListIsNotFoundForAdapter`(400, `$.success == false`, `$.message` 일치), `explicitPageSizeIsHonored`(`pagesize` 파라미터 또는 `.pageSize(1)` 로 200).
    - 클래스 어노테이션은 기존과 동일: `@SpringBootTest @AutoConfigureMockMvc @WithMockUser(username="admin", roles={"ADMIN"})`. `@DirtiesContext` 는 절대 붙이지 말 것(7a40a00 교훈 — 컨텍스트 재생성이 힙·시간·비동기 종료 경합을 악화시킨다).

- 검증 명령:
  - 단독: `sh ./gradlew --no-daemon test --tests 'com.nexabuilder.api.DataAdapterListSoftDeleteIntegrationTest' --tests 'com.nexabuilder.api.DataAdapterIntegrationTest'`
  - 전체: `sh ./gradlew --no-daemon cleanTest test` (수 분, 이전 회차 기준 575건 → 이번 신규 3건 추가 예상)
  - 패키징: `sh ./gradlew --no-daemon bootJar -x test`
  - (주의: `gradlew` 는 100644 이므로 반드시 `sh ./gradlew`. 이번 정찰에서는 승인 제약으로 테스트를 실행하지 못했다 — **미확인**. 위 명령 자체는 이전 3회차에서 쓰인 것이다.)

- 위험과 피할 것:
  - **응답 형식 차이**: `/api/v1/data/*` 는 `ApiResponse` 봉투를 쓰지 않고 `{Rows, totalCount}` 를 그대로 돌려준다(`DataAdapterController` 주석). 하지만 예외는 `GlobalExceptionHandler` 를 타므로 **오류일 때만** `{success,message}` 봉투다. 테스트의 200 단언은 `$.Rows`/`$.totalCount`, 400 단언은 `$.success`/`$.message` 로 써야 한다. 섞지 말 것.
  - **빈 순환**: `DataAdapterService` → `NexaUiService` 주입. 확인한 범위에서 `NexaUiService` 는 `core/data` 를 전혀 참조하지 않고 `DataAdapterService` 는 `DataAdapterController` 한 곳에서만 쓰이므로 순환은 없을 것으로 본다. 그래도 컨텍스트 기동이 깨지면 대안은 `requireLiveList` 를 `NexaUiService` 에 그대로 두고 `DataAdapterService` 안에서 같은 2줄을 복제하는 것이 **아니라**, 판정을 `core/ui` 쪽 작은 컴포넌트로 뽑는 것이다(같은 값을 두 경로가 다르게 읽는 상태를 다시 만들지 말 것).
  - **`deleted_at` 문자열 형식**: `yyyyMMddHHmmss` 문자열이다(`UiBuilderController.deleteList` 가 `now()` 로 찍음). 테스트에서도 `"20260922000000"` 같은 문자열로 세팅할 것.
  - **권한 게이트를 이번에 넣지 말 것**: `/api/v1/data/lists/{id}` 에 `ScreenPermissionService` 가 없다는 건 별개 문제(보류 아이디어 3/4/M). `core/permission`·`core/auth`·`infra/security`·`db/migration`·`.github/workflows` 는 이번 과제에서 건드리지 않는다.
  - **null pageSize 기본값을 50 으로 바꾸지 말 것**: DB 기본값은 50(migration 007)이지만 `NexaUiService.listData` 와 `parse` 의 기존 폴백은 둘 다 100 이다. 100 을 유지해 두 경로가 같은 값을 같게 읽게 한다. DB 마이그레이션이나 `NexaListDef` 에 `@Builder.Default` 를 넣는 방향은 금지(기존 행 의미가 바뀐다).
  - **과거 교훈**: 같은 판정을 하는 경로가 여럿이면 한쪽만 넓히지 말 것 — 이번 수정 뒤 `listDefinition` / `listData` / `resolveExportable` / `queryList` 네 곳이 전부 `requireLiveList` 하나를 통과하는지 확인하고 커밋 메시지에 그 사실을 남길 것.

- 차선 후보: `DataAdapterService.parse` 의 페이지 입력 음수·오버플로 검증(`pagenum=-1` → `applyPaging` 의 `subList` 음수 인덱스 / SQL 음수 offset, `pagesize=0` → SQL 백엔드에서 전체 행 반환). 1순위의 빈 순환이 실제로 막히면 이쪽을 잡는다. 단 계약(400 으로 거절 vs 0 으로 보정)을 먼저 정하고 테스트로 못 박을 것.
