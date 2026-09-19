# 과제서 — 2026-09-19 nexabuilder-improve

- 과제: 목록 내보내기(CSV/XLSX/PDF)가 휴지통(soft-delete)에 든 목록도 내려주는 것을 막고, 내보내기 3종에 통합 테스트를 처음으로 붙인다 (가치 4 / 위험 2 / 작업량 M)

- 왜: `ListExportController` 의 세 엔드포인트(`/api/v1/lists/{listId}/export.csv|.xlsx|.pdf`)는 `listRepository.findById` 로 목록을 찾고 화면 권한만 검사한다. 런타임 화면 `/app/list/<id>` 는 `NexaUiService.listDefinition` 에서 `deletedAt != null` 이면 "List not found" 를 던져 404 처럼 보이게 하는데(NexaUiService.java:50-53, 마이그레이션 027), 내보내기는 그 검사가 없어 휴지통에 버린 목록의 데이터를 URL 하나로 계속 내려받을 수 있다. 같은 값(목록 존재 여부)을 읽는 경로 둘이 다르게 판단하는 상태이며, 이 컨트롤러는 테스트가 한 건도 없다(`src/test/java` 에서 `ListExport`/`export.` 검색 결과 0).

- 수용 기준:
  1) `deletedAt` 이 채워진 목록에 대해 `.csv`, `.xlsx`, `.pdf` 세 요청 모두 `IllegalArgumentException("List not found: …")` → `GlobalExceptionHandler.handleIllegalArgument` 를 타서 400 + `{"success":false,…}` JSON 으로 끝나고 파일 바이트가 한 바이트도 나가지 않는다(`Content-Type` 이 `text/csv`/`application/pdf` 가 아니다).
  2) 살아 있는 목록에 대해서는 세 형식이 지금과 똑같이 동작한다 — CSV: 200, `text/csv;charset=UTF-8`, 본문이 UTF-8 BOM(`﻿`)으로 시작하고 첫 줄이 columnsJson 의 label 순서, 두 번째 줄부터 데이터 행; XLSX: 200, 스프레드시트 content-type, 응답 바이트를 `org.apache.poi.xssf.usermodel.XSSFWorkbook` 으로 열어 0번 시트 0행이 label, 1행이 데이터; PDF: 200, `application/pdf`, 본문이 `%PDF-` 로 시작. 세 응답 모두 `Content-Disposition` 에 `attachment` 와 `filename*=UTF-8''` 이 있다.
  3) 세션에 `nexabuilder.user` 가 없으면(권한 없음) 세 형식 모두 403 + `{"success":false,"message":"Access denied"}` (AccessAuditHandler.java:52-57 의 `/api/` 분기). 이 케이스는 코드 변경 없이 지금도 그렇게 동작하는지 테스트가 증명한다.
  4) 테스트는 `@SpringBootTest + @AutoConfigureMockMvc` 로 실제 H2·실제 `ScreenPermissionService`·실제 OpenPDF/POI 를 통과한다. 목·대역 주입 없음(운영자 규칙).
  5) 기존 스위트(현재 568개)가 그대로 통과한다.

- 건드릴 파일:
  - `src/main/java/com/nexabuilder/api/controller/ListExportController.java`
    - `exportCsv`(75-136행), `exportXlsx`(145-202행), `exportPdf`(210-295행) 세 곳에 복붙된 "findById → 세션 user → canView" 블록을 `private NexaListDef resolveExportable(String listId, HttpSession session)` 한 개로 뽑고, 그 안에 `list.getDeletedAt() != null` 이면 `IllegalArgumentException("List not found: " + listId)` 를 던지는 줄을 추가한다(NexaUiService.listDefinition 의 메시지와 동일하게). 예외 타입·메시지를 바꾸지 말 것 — `handleIllegalArgument` 가 400 JSON 으로 바꿔 주는 계약에 기댄다.
    - 클래스 Javadoc(46-62행)에 `.xlsx`/`.pdf` 엔드포인트가 빠져 있고 "CSV / XLSX export" 라고만 되어 있으니 세 형식과 "휴지통 목록은 404 처럼 400" 을 한 줄로 적는다(선택).
    - 그 외 로직(BOM, stringify/lookup, 컬럼 폴백, 폰트 선택)은 손대지 않는다.
  - `src/test/java/com/nexabuilder/api/ListExportIntegrationTest.java` (신규)
    - 픽스처는 `ListInlineEditIntegrationTest.seed()` 를 그대로 본뜬다: `DynamicEntity` 저장 → `schemaDdlService.ensureTable` → `entityService.insert(tableName, row)` 로 행 1~2개(한글 값 하나 포함, 예: `name="홍길동"`) → `NexaListDef.builder().listId(...).entityId(...).columnsJson([{field:"id",label:"ID"},{field:"name",label:"이름"}])` 저장. 접미사는 UUID 8자로 유일하게.
    - 세션 user 는 `KanbanViewIntegrationTest.adminSessionUser()`(174-182행) 를 복사해 `.sessionAttr("nexabuilder.user", adminSessionUser())` 로 넘긴다. 클래스에 `@WithMockUser(username="admin", roles={"ADMIN"})`.
    - 테스트 메서드(최소): `csvExportHasBomHeaderAndRows`, `xlsxExportOpensWithPoi`, `pdfExportStartsWithPdfMagic`, `softDeletedListIsNotExportableInAnyFormat`(같은 픽스처의 `deletedAt` 을 `"20260919000000"` 같은 문자열로 set 후 `listRepository.save`, 세 URL 순회), `missingSessionUserIs403ForAllFormats`.
    - CSV 검증은 `result.getResponse().getContentAsString(StandardCharsets.UTF_8)` 후 `startsWith("﻿")`, 줄 분리해 `"ID,이름"` 헤더 확인(writeRow 의 인용 규칙은 파일 380행 이후에 있으니 실제 구현을 읽고 기대값을 맞출 것 — 미확인). XLSX 는 `getContentAsByteArray()` → `new XSSFWorkbook(new ByteArrayInputStream(bytes))`. PDF 는 바이트 앞 5개가 `%PDF-`.

- 검증 명령 (저장소 루트에서; 이 워크트리에서는 gradlew 가 100644 라 `sh ./gradlew` 또는 `chmod +x` 후 실행 — 정찰 세션은 실행 권한이 없어 직접 돌리지 못함, 미확인):
  1. `./gradlew --no-daemon test --tests 'com.nexabuilder.api.ListExportIntegrationTest'` — 새 테스트 단독. 먼저 컨트롤러를 고치기 전에 돌려 `softDeletedListIsNotExportableInAnyFormat` 만 빨간 것을 확인(TDD 증거)하고, 고친 뒤 초록.
  2. `./gradlew --no-daemon cleanTest test` — 전체 스위트(직전 회차 기준 568개, 컨텍스트 기동 22회, 몇 분 걸림).
  3. `./gradlew --no-daemon bootJar -x test` — 릴리즈 검증 단계와 동일.

- 위험과 피할 것:
  - `NexaUiService.listData` 에는 deletedAt 검사를 넣지 말 것 — 빌더 API·대시보드 등 다른 호출자가 있을 수 있고(미확인) 이번 과제는 내보내기 경로만 런타임 화면과 같게 맞추는 것이다. 한 경로만 넓히지 말라는 운영자 규칙의 취지대로 "내보내기 3종이 같은 판단을 하게" 만드는 것이 목표.
  - `ScreenPermissionService`, `AccessAuditHandler`, `GlobalExceptionHandler` 는 보호 경로(auth) — 수정하지 않는다. 403/400 매핑은 이미 있는 것을 테스트로 증명만 한다.
  - `@DirtiesContext` 를 붙이지 말 것(2026-09-17 회차가 그것 때문에 힙 2g 문제를 겪고 걷어냄). 픽스처는 유일 접미사로 격리하고 정리는 하지 않아도 된다(다른 통합 테스트도 그렇게 함).
  - PDF 폰트: `chooseFont` 가 CJK 팩 없으면 Helvetica 로 폴백하므로 PDF 본문 안의 한글 글리프를 검사하지 말 것 — `%PDF-` 매직과 content-type 까지만.
  - CSV 응답에서 `getContentAsString()` 을 charset 없이 부르면 BOM 이 깨질 수 있으니 반드시 UTF-8 지정.
  - Jackson: 테스트에서 `ObjectMapper` 는 `com.fasterxml.jackson.databind.ObjectMapper` 를 주입(다른 테스트와 동일). `tools.jackson` 쪽을 쓰지 말 것.
  - 응답 본문을 스트리밍하는 컨트롤러라 예외가 `doc.open()` 이후에 나면 부분 바이트가 새어 나갈 수 있는데, 이번 검사는 스트림을 열기 전에 두므로 해당 없음. 검사 위치를 `response.getOutputStream()` 앞에 둘 것.

- 차선 후보: CI 에 `gradle/actions/wrapper-validation` 스텝 추가 + `docker-publish.yml` 의 `actions/checkout@v4` → `v6` 정렬 + `git update-index --chmod=+x gradlew` 로 워크플로 3곳의 `chmod +x ./gradlew` 스텝 제거 (보류 아이디어 3건을 한 PR 로; 가치 3 / 위험 1 / 작업량 S). 1순위가 성립하지 않을 때(예: export 가 이미 deletedAt 을 다른 계층에서 거르고 있음이 확인될 때) 고를 것. 액션의 최신 메이저 태그는 정찰이 확인하지 못했으니(미확인) 구현자가 GitHub 에서 확인한 뒤 고정할 것.
