- 과제: 모든 컬럼이 `hidden` 인 목록을 내보낼 때 CSV/XLSX/PDF 가 숨긴 컬럼을 첫 행 키 fallback 으로 다시 노출하는 것을 막는다 (가치 3 / 위험 2 / 작업량 S)

- 왜: `ListExportController.resolveColumns`(`ListExportController.java:339`) 는 `columnsJson` 에서 `hidden:true` 컬럼을 건너뛰는데, 건너뛴 결과가 빈 리스트가 되면 `exportCsv`/`exportXlsx`/`exportPdf` 세 곳에 각각 복제된 fallback 블록이 `rs.getRows().get(0).keySet()` 으로 **행의 모든 키**를 컬럼으로 다시 세운다(각각 `:114`, `:155`, `:242` 의 `if (columns.isEmpty() && !rs.getRows().isEmpty())`). 즉 관리자가 리스트 디자이너의 Hidden 체크박스(`admin/list-designer.html:435-436`, `:791`)로 전 컬럼을 숨긴 목록은 내보내기 파일에서 `SELECT *` 결과가 그대로 나온다 — 화면에서는 숨겼는데 파일로는 나가는, 방향이 반대인 구멍이다. fallback 자체는 "컬럼 설정이 아예 없는 목록" 을 위한 의도된 동작이므로(주석 `:86-89`) 그 경우는 유지하고, "설정은 있는데 전부 숨김" 만 구분하면 된다.

- 수용 기준:
  1) `columnsJson` 의 모든 항목이 `hidden:true` 인 목록에 대해 `GET /api/v1/lists/{id}/export.csv` 가 200 을 주되 본문에 데이터 컬럼 값(`id` 값, `이름` 값)이 하나도 들어 있지 않다. 같은 목록의 `.xlsx` 는 열리되 시트에 데이터 셀이 없고, `.pdf` 는 200 + PDF magic(`%PDF`) 이며 기존 `columns.isEmpty()` 경로의 "표시할 데이터가 없습니다." 로 떨어진다.
  2) `columnsJson` 이 null/blank 이거나 `[]` 이거나 파싱 실패인 목록은 **지금과 똑같이** 첫 행의 모든 키로 fallback 해 데이터가 나간다(회귀 가드 — 이 동작을 없애면 안 된다).
  3) 일부만 `hidden:true` 인 목록은 보이는 컬럼만 나가고 숨긴 컬럼은 안 나간다(이미 동작하지만 계약을 못으로 박는다).
  4) 기존 `ListExportIntegrationTest` 13건이 그대로 초록.
  - 테스트가 증명해야 하는 것: 수정 **전에** 1) 의 CSV/XLSX 단언이 빨간 것(숨긴 컬럼 값이 파일에 들어 있음)을 먼저 확인하고, 2)·3) 은 수정 전에도 초록이어야 한다. 빨간 로그를 회차 기록에 남길 것.

- 건드릴 파일 (프로덕션 1개 + 테스트 1개):
  - `src/main/java/com/nexabuilder/api/controller/ListExportController.java`
    - `resolveColumns(NexaListDef)` — 반환을 `List<ColumnSpec>` 대신 "보이는 컬럼 + 컬럼 설정이 존재했는가" 를 함께 나르는 private record(예: `ColumnPlan(List<ColumnSpec> visible, boolean configured)`)로 바꾼다. `configured=true` 의 정의는 **`columnsJson` 이 비어 있지 않은 JSON 배열로 파싱됐다**(항목이 1개 이상) — hidden 여부와 무관. 파싱 실패(`catch`)와 `null`/blank/`[]` 는 `configured=false`.
    - 세 엔드포인트에 복제된 fallback 블록 3개를 private 헬퍼 하나(예: `applyFirstRowFallback(ColumnPlan plan, RecordSet rs)`)로 합치고, `plan.configured()` 면 fallback 을 건너뛴다. 세 경로가 같은 규칙을 쓰게 하는 것이 이 과제의 핵심 — 한 곳만 고치지 말 것.
    - `exportCsv` — fallback 후 컬럼이 비면 헤더 행과 데이터 행을 쓰지 않는다(BOM 만 나가게). 지금은 `writeRow(w, [])` 가 빈 줄을 행 수만큼 찍는다.
    - `exportXlsx` — 컬럼이 비면 헤더/데이터 `createRow` 를 돌리지 않는다(빈 시트). `WorkbookUtil.createSafeSheetName` 호출(`:173`)은 건드리지 말 것 — 직전 회차(커밋 ee7f786)에서 막 고친 자리다.
    - `exportPdf` — `if (columns.isEmpty())` → "표시할 데이터가 없습니다." 분기(`:247-251`)가 이미 있으니 fallback 억제만 하면 되고 추가 수정 불필요.
  - `src/test/java/com/nexabuilder/api/ListExportIntegrationTest.java` — `seed(Integer pageSize, int rowCount, String requestedListId)` 계열에 `columnsJson` 을 넘길 수 있는 오버로드를 하나 더 얹고(기존 `seed()`→`seed(null,1)` 위임 패턴을 그대로 따라 기존 13건의 기대값은 손대지 않는다), 신규 4~5건 추가: all-hidden CSV/XLSX/PDF, `columnsJson=null` fallback 유지, 부분 hidden. 실제 `NexaListRepository`·`EntityService`·`SchemaDdlService`·MockMvc·POI 로 검증하고 목·대역 금지.

- 검증 명령:
  1. `sh ./gradlew --no-daemon test --tests 'com.nexabuilder.api.ListExportIntegrationTest'` — 이 클래스만. 직전 회차 실측 약 25초(9건 기준).
  2. `sh ./gradlew --no-daemon cleanTest test` — 전체. 직전 회차 609건 통과, 약 6분.
  3. `sh ./gradlew --no-daemon bootJar -x test` — 패키징(직전 회차 6초).
  - `gradlew` 는 git 모드 100644 라 반드시 `sh ./gradlew` 로 실행.

- 위험과 피할 것:
  - **fallback 자체를 지우지 말 것.** `columnsJson` 이 없는 목록(샘플·SQL 기반 목록 다수)은 fallback 이 유일한 컬럼 소스다. 수용 기준 2 가 이것을 지킨다.
  - `NexaUiService.listData`·`readQueryParams`·`MAX_FEED_ROWS`·`resolveExportable`(권한/soft-delete) 는 이번 범위 밖. 특히 `_limit` 주입(`:381`)과 시트 이름 변환은 최근 2회차에 머지된 자리라 회귀시키면 바로 드러난다.
  - `NexaUiService`·`DataAdapterService`·`RecordSet` 등 공유 계약으로 범위를 넓히지 말 것. 프로덕션 파일은 `ListExportController.java` 한 개로 끝난다.
  - `@DirtiesContext` 금지(과거 회차에서 컨텍스트 재생성·힙 문제).
  - 테스트 픽스처는 UUID 8자 접미사·정리 없음 관례를 따를 것.
  - 미확인(정찰이 실행으로 재현하지 못한 것): all-hidden 목록의 실제 HTTP 응답은 이번 세션에서 돌려 보지 못했다. 위 진단은 `resolveColumns` 와 세 fallback 블록을 직접 읽은 결과이며, 구현자는 **빨간 테스트를 먼저 만들어 실제 파일에 숨긴 컬럼이 들어가는지 확인**한 뒤 고칠 것. 만약 수정 전에도 CSV 가 이미 비어 있다면 이 과제는 성립하지 않으니 아래 차선으로 갈 것.
  - all-hidden 이 리스트 디자이너 UI 로 저장 가능한지(전 컬럼 Hidden 저장 시 검증이 막는지)는 미확인 — 저장소 레벨(`NexaListRepository.save`)에서는 막는 것이 없으므로 테스트는 리포지토리 시드로 만들면 된다.

- 차선 후보: `DataAdapterService.queryList` 의 sqlId 백엔드 목록에도 `requireLiveList` 회귀 테스트를 붙인다 (가치 2 / 위험 1 / 작업량 S, 프로덕션 0줄). `SqlBackedListDefinitionIntegrationTest`(6건)에 `deleted_at` 이 찍힌 SQL 기반 목록 사례가 없어, `GET /api/v1/data/lists/{id}` 와 `POST /api/v1/builder/lists/{id}/data` 가 휴지통 SQL 목록에 대해 400 을 주는지 고정돼 있지 않다. 기존 픽스처를 재사용하므로 테스트 파일 1개로 끝난다.
