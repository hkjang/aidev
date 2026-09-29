# 과제서 — 2026-09-30-072221-nexabuilder-improve

- **과제**: 수정 과제 — PR #43 의 CI 실패를 증거로 종결(코드·워크플로 원인 아님)하고, **휴지통에 넣은 목록으로도 리포트 CSV 가 전량 생성되고 메일 첨부로 밖으로 나가는** `ReportService.generate` 의 soft-delete 누락을 고친다 (가치 4 / 위험 2 / 작업량 S)

- **왜**: `ReportService.generate:69-71`(실제로 열어 확인)의 `lists.findById(listId).orElseThrow(...)` 는 목록의 `deletedAt` 을 **전혀 보지 않는다**. 같은 값을 읽는 다른 다섯 경로 — `NexaUiService.requireLiveList:60-63`(정의·데이터), `DataAdapterService.queryList`, `ListExportController.resolveExportable`, `ListInlineEditController:71-72`, 직전 회차에 고친 `BulkListController:193` — 는 전부 400 `List not found: {listId}` 로 거부하는데, 리포트 경로만 통과한다. 그래서 운영자가 목록을 휴지통에 넣어 화면·그리드·내보내기에서 전부 사라진 뒤에도 `POST /api/v1/admin/reports/run` 은 밑 테이블의 **전체 행(MAX_ROWS = 1,000,000, 페이지 없음)** 을 CSV 로 떠서 `nexa.report.dir` 에 파일로 남기고, `recipients` 가 있으면 `generateAndEmail:95-111` 이 그걸 메일 첨부로 조직 밖에 보낸다. 더 나쁜 것은 스케줄 경로다 — `ScheduledJobManager:102` 의 `case "report"` 가 같은 서비스를 부르므로, `action_type=report` 잡이 걸린 목록을 휴지통에 넣어도 스냅샷 메일은 **계속 나간다**(휴지통에 넣는 것이 정지 수단이 아니게 된다).

- **CI 판정(먼저 이것부터, 코드 수정 전)**: PR #43(`auto/2026-09-30-0622` @ `eaf90787a70a`)의 실패는 저장소의 코드도 워크플로도 원인이 아니다. 이번 정찰이 `2026-09-30-062227-nexabuilder-improve/ci-eaf90787a70a.json` 을 직접 열어 확인했다:
  - `test + bootJar`(check-run 109633534678): `started_at 2026-09-29T21:42:33Z` → `completed_at 21:42:36Z` = **3초** 만에 `conclusion: failure`, `output.title/summary/text` 전부 `null`.
  - 같은 커밋 `analyze (java)`(109633536544): `conclusion: skipped`, `annotations_count: 0`, `started_at == completed_at`.
  - 3초는 `actions/checkout` 도 끝나지 않는 시간이다(이 저장소 성공 런 22건은 470~611초, 앞선 실패 7건은 3·4·7·9·39초로 같은 서명).
  - 원인은 이미 **인증된 직접 증거**로 증명돼 있다 — 2026-09-30-010003 셰퍼드 회차가 인증된 `gh api .../jobs` 로 잡 주석 "The job was not started because recent account payments have failed or your spending limit needs to be increased" 와 `steps: []`(체크아웃조차 실행 안 됨)를 확보했다. GitHub Actions **과금 차단**이다.
  - 이 세션/구현 세션은 `gh` 미인증일 가능성이 높다. 로그 본문을 새로 못 열면 **"미확인" 이라고 적고 종결**하라. 8회차째 같은 서명을 재추정하는 데 시간을 쓰지 말 것.
  - **`.github/workflows` 는 한 줄도 건드리지 말 것.** 재시도·`continue-on-error`·타임아웃 완화·`-x test` 추가 전부 금지(워크플로 완화로 통과시키는 것은 금지 규칙).
  - 대신 CI 가 돌렸어야 할 두 명령을 로컬에서 그대로 재현해 통과를 보고하라(아래 검증 명령).

- **수용 기준**:
  1. 휴지통에 넣은 목록(`deleted_at` 이 set)으로 `POST /api/v1/admin/reports/run {"listId":"<id>"}` 가 **400** + `{"success":false,"message":"List not found: <listId>"}` 를 돌려주고, `reportService.list()` 에 그 목록의 새 파일이 생기지 않는다.
  2. `recipients` 를 함께 준 경우(=`generateAndEmail` 경로)도 같은 400 이며 **메일 시도 자체가 없다**(= `generate` 에서 던지므로 `generateAndEmail:96` 다음 줄로 넘어가지 않는다. SMTP 를 세우지 않으려면 이 케이스는 400 단언까지만 하면 충분하다).
  3. **살아 있는 목록은 그대로 동작한다** — 기존 `ReportIntegrationTest` 3건(`generatesCsvForSampleListAndLists`, `serviceWritesNonEmptyCsv`, `rejectsMissingListId`)이 그대로 초록이고, 새로 시드한 살아 있는 목록도 200 + `rows >= 1`.
  4. 수정 **전에** 새 테스트를 돌려 빨간 것을 확인하고 그 출력을 회차 노트에 남긴다. 예상: `Status expected:<400> but was:<200>`. 회귀 가드(수용 기준 3의 살아 있는 목록)는 수정 전에도 초록이어야 한다 — 그래야 "휴지통 경로만 깨져 있었다" 가 증명된다.

- **건드릴 파일** (프로덕션 **1개**):
  - `src/main/java/com/nexabuilder/core/report/ReportService.java:generate(String)` — `orElseThrow` **직후**, `sqlId`/`entityId` 분기보다 **앞**에서 `if (list.getDeletedAt() != null) { throw new IllegalArgumentException("List not found: " + listId); }`. 문구는 다른 다섯 경로와 **글자 단위로 동일**하게(휴지통 목록과 없는 목록을 구분하지 않는다 — 존재 여부는 기존 `orElseThrow` 가 이미 같은 방식으로 노출하므로 새 누출이 아니다). `getDeletedAt()` 은 `NexaListDef` 의 **String**(`yyyyMMddHHmmss`)이므로 `!= null` 로 판정한다(`NexaUiService:62` 와 같은 형태). 헬퍼를 새로 만들지 말고 이 한 블록만.
  - `src/test/java/com/nexabuilder/api/ReportIntegrationTest.java` — **기존 파일에 추가**(새 파일 금지). 이 클래스는 `@SpringBootTest(properties = "nexa.report.dir=${java.io.tmpdir}/nexa-report-it")` 로 전용 컨텍스트를 이미 쓰고 있어, 같은 클래스에 넣으면 **컨텍스트가 늘지 않는다**(다른 `properties` 로 새 파일을 만들면 컨텍스트가 하나 더 생겨 종료 지연이 늘어난다). `@Autowired` 로 `ObjectMapper`·`DynamicEntityRepository`·`NexaListRepository`·`EntityService`·`SchemaDdlService` 를 추가하고, 시드는 `DataAdapterListSoftDeleteIntegrationTest.seedList:70-114`(실제로 열어 확인) 형태를 그대로 따라라 — `UUID.randomUUID().toString().replace("-","").substring(0,8)` 접미사, `DynamicEntity.builder()` → `entityRepository.save` → `schemaDdlService.ensureTable(entity)` → `entityService.insert(tableName, row)` → `NexaListDef.builder()...entityId(entityId).build()` → `listRepository.save`, 정리 없음. 휴지통 처리는 같은 파일 141-152 행처럼 `list.setDeletedAt("20260922000000"); listRepository.save(list);`.
  - 목·대역·`FakeXxx` 금지(실제 H2·`SchemaDdlService`·`EntityService`·MockMvc). `@DirtiesContext` 금지.

- **검증 명령** (이 저장소에서 실제로 도는 것. `gradlew` 는 모드 100644 라 `sh` 경유):
  - 한 클래스만: `sh ./gradlew --no-daemon test --tests 'com.nexabuilder.api.ReportIntegrationTest'`
  - 전체(CI 와 같은 조건, 약 6분): `sh ./gradlew --no-daemon clean test` — 직전 회차 master 기준 615건 통과/0 fail/0 skip. 결과는 `build/test-results/test/*.xml` 의 `tests/failures/errors/skipped` 합으로 집계해 보고할 것.
  - 패키징: `sh ./gradlew --no-daemon bootJar -x test` (약 6초, `build/libs/nexabuilder-1.26.0.jar`)

- **위험과 피할 것**:
  - **`.github/workflows`·`build.gradle.kts`·`settings.gradle.kts`·`gradle/` 불간섭.** 이번 브랜치에서 `git diff --stat master... -- .github/ build.gradle.kts settings.gradle.kts gradle/` 가 빈 출력임을 직접 확인하고 보고에 쓸 것(정찰 시점 워크트리는 master 팁 `a3ca143` 과 같아 diff 자체가 비어 있었다).
  - **열린 PR 과 파일 충돌 금지** — #37·#38·#39=`ListExportController`, #40=`DataAdapterService`, #41=`NexaUiService`, #42=`CsvImportController`/`EntityService`/`CsvSchemaInferenceService`, #43=`BulkListController`. 이번 과제의 `ReportService`·`ReportIntegrationTest` 는 **다섯 건 모두와 겹치지 않는다**. 위 파일들을 "같이 정리" 하지 말 것.
  - **범위 밖(이번에 건드리지 말 것)**: (a) 같은 함수 77-78 행의 `entities.findById(list.getEntityId())` 도 엔티티의 `deletedAt` 을 보지 않는다 — 같은 계열의 별개 결함이고 `NexaUiService.requireLiveEntity` 와 계약을 맞춰야 하므로 **다음 회차**로 넘겨라(ideas.json 에 올려 뒀다). (b) `ReportController` 의 예외 처리(지금 `IOException` 만 잡고 나머지는 `GlobalExceptionHandler` 로 흘린다) — 그 흐름이 400 을 만들어 주므로 손대지 말 것. (c) `ScheduledJobManager` — `generate` 를 고치면 스케줄 경로도 같이 막히므로 별도 수정이 필요 없다. (d) `MAX_ROWS` 상한 변경 금지. (e) `SqlExecutor`/`RecordSet` 계약 불간섭(Groovy·워크플로가 공유한다).
  - 과거 교훈: 손으로 만든 대역으로 증명하지 말 것, 소스 문자열 grep 을 증거로 제출하지 말 것, 같은 값을 읽는 경로가 둘 이상이면 문구·상태코드를 실제 HTTP 로 맞춰 확인할 것.
  - 미확인으로 남긴 것(추측하지 말고 실행으로 확인하라): ① `GlobalExceptionHandler` 가 `IllegalArgumentException` 을 400 + `ApiResponse.error` 로 접는다는 것은 `DataAdapterListSoftDeleteIntegrationTest:156-160` 이 같은 문구로 이미 고정하고 있어 계약은 확실하지만, **`/api/v1/admin/**` 경로에서도 같은 봉투인지는 이번에 직접 실행해 확인하지 않았다** — 테스트에서 실제 응답으로 맞춰라. ② `recipients` 를 준 경로가 SMTP 없이도 400 까지 도달하는지(=`generate` 가 먼저 던지므로 그럴 것으로 보이지만 미확인).

- **차선 후보**: README·`agent.md`·`docs/deployment.md`·`docs/roadmap.md` 의 낡은 JDK17/Boot3.4.4/v1.2.x 표기를 Java21/Boot4.1.1/v1.26.0 으로 정렬 (가치 2 / 위험 1 / 작업량 S). 문서만 건드려 열린 PR 6건과 충돌이 없다. 값은 `build.gradle.kts` 에서 확인하고, 대체된 옛 가이드는 남기지 말고 덮어쓸 것(운영자 지침). 6회차 연속 차선 후보다 — 주과제가 성립하지 않을 때만 고르되, 이번엔 주과제가 성립할 것으로 본다.
