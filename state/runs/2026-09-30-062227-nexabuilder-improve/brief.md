- 과제: 수정 과제 — PR #42 의 CI 실패를 증거로 종결(코드·워크플로 원인 아님)하고, **휴지통에 넣은 목록으로도 대량 삭제·수정이 그대로 실행되는** `BulkListController.resolveTarget` 의 soft-delete 누락을 고친다 (가치 4 / 위험 2 / 작업량 S)

- 왜: PR #42 의 CI 는 7회차째 같은 서명으로 죽었고 원인은 저장소 밖(GitHub Actions 과금 차단)이라 고칠 코드가 없다 — 이번 회차는 그것을 증거로 종결하고 그 자리를 실재 결함으로 채운다. `BulkListController:193` 의 `listRepository.findById(listId)` 는 목록의 `deletedAt` 을 **전혀 보지 않는다**. 같은 저장소의 다른 목록 판독 경로 다섯 곳 — `NexaUiService.requireLiveList`(정의·데이터), `DataAdapterService.queryList`, `ListExportController.resolveExportable`, `ListInlineEditController:72`(`list.getDeletedAt() != null` → 404) — 은 전부 거부하는데, **가장 파괴적인 두 엔드포인트**(`POST /api/v1/lists/{listId}/bulk/delete`, `/bulk/update`)만 통과시킨다. 즉 운영자가 목록을 휴지통에 넣어 화면·그리드·내보내기에서 사라진 뒤에도 그 listId 를 아는 호출자는 밑의 테이블 행을 계속 지우고 고칠 수 있다.

- 수용 기준:
  1) **CI 종결(코드 원인 아님).** `ci-a7d58cb02962.json`(이번 회차 run 디렉터리 옆 `2026-09-30-051654-nexabuilder-improve/`)의 `test + bootJar`(check-run 109615593167)는 `started_at 2026-09-29T20:55:07Z` → `completed_at 20:55:16Z`, **9초** 만에 failure 이고 같은 커밋의 `analyze (java)` 는 `conclusion: skipped`, `output.title/summary/text` 는 전부 null 이다. 성공 런은 470~611초, 앞선 실패 6건은 3·4·7·39초 — 9초는 `actions/checkout` 도 끝나지 않는 시간이다. 이번 브랜치에서 `git diff --stat master... -- .github/ build.gradle.kts settings.gradle.kts gradle/` 는 **빈 출력**이고(이번 정찰이 실제로 실행해 확인함), 빌드·CI 설정을 건드린 최신 커밋은 2026-09-28 의 버전 범프 `c446d24` 로 그 위의 `ee7f786`(PR #36)은 CI 를 통과해 머지됐다. 2026-09-30-010003 셰퍼드 회차가 **인증된** `gh api .../jobs` 로 "The job was not started because recent account payments have failed or your spending limit needs to be increased" + `steps: []` 를 이미 확보해 두었다. 따라서 **`.github/workflows` 를 한 줄도 바꾸지 않고**(재시도·`continue-on-error`·타임아웃 완화 금지) 이 조사를 종결한다. `gh` 는 이 환경에서 미인증이라 로그 본문을 새로 여는 것은 **불가능 — 미확인으로 남길 것**. 7회차째 재추정에 시간을 쓰지 말 것.
  2) `deletedAt` 이 설정된 목록에 `POST /api/v1/lists/{listId}/bulk/delete` 를 보내면 **400 + `{"success":false,"message":"List not found: {listId}"}`** 가 오고, **밑의 테이블 행이 한 건도 지워지지 않는다**(요청 뒤 `entityService.findByKey` 또는 `/api/v1/data/lists/...`(살아 있는 별도 목록)로 행 존재를 직접 단언할 것 — 응답만 보고 끝내지 말 것). `/bulk/update` 도 같은 응답이고 컬럼 값이 그대로다.
  3) 회귀 가드: `deletedAt` 이 null 인 **살아 있는** 목록의 `/bulk/delete` 는 여전히 200 + `affected: 1`, `/bulk/update` 는 200 + `affected: 1` 이고 값이 실제로 바뀐다. 이 두 건은 **수정 전에도 초록**이어야 한다(그래야 "휴지통 경로만 깨져 있었다" 가 증명된다).
  4) 테스트가 증명해야 하는 것: 실패 메시지가 다른 다섯 판독 경로와 **글자 단위로 같다**(`"List not found: " + listId`). `DataAdapterListSoftDeleteIntegrationTest:160` 이 이미 같은 문구를 고정하고 있으니 그것을 따를 것.

- 건드릴 파일 (프로덕션 **1개**):
  - `src/main/java/com/nexabuilder/api/controller/BulkListController.java:191 resolveTarget` — `listRepository.findById(listId).orElseThrow(...)` **직후**에 `if (list.getDeletedAt() != null) throw new IllegalArgumentException("List not found: " + listId);` 를 넣는다. **권한 검사보다 앞**에 두어 휴지통 목록이 없는 목록과 구분되지 않게 한다(존재 여부는 `orElseThrow` 가 이미 같은 방식으로 노출하므로 새 누출이 아니다). `IllegalArgumentException` 은 `GlobalExceptionHandler:92-95` 가 400 + `ApiResponse.error(message)` 로 접는다(이번 정찰이 소스에서 확인).
  - (같은 메서드, 여유가 있으면) `list.getEntityId()` 가 null/blank 인 SQL 기반 목록 — 현재 `entityRepository.findById(null)` 로 들어간다. `ListInlineEditController:75-81` 과 같은 취지로 명시적인 `IllegalArgumentException` 으로 접을 것. **단 "현재 무엇이 오는가" 는 미확인이다** — 먼저 빨간 테스트로 현재 상태 코드를 확인하고, 이미 400 으로 접히고 있으면 이 항목은 **버리고** 수용 기준 2·3 만으로 회차를 닫을 것(추측으로 코드를 넣지 말 것).
  - `src/test/java/com/nexabuilder/api/BulkListSoftDeleteIntegrationTest.java` — **신규**. `BulkListController` 에는 테스트가 **하나도 없다**(이번 정찰이 `src/test/java/com/nexabuilder/api/` 에서 `bulk` 로 찾은 것은 `BulkUnlockIntegrationTest`·`MenuPermissionBulk*` 뿐으로 전부 다른 컨트롤러다).

- 검증 명령 (이 저장소에서 실제로 도는 것. `gradlew` 는 mode 100644 라 `sh` 경유):
  - 단일 클래스: `sh ./gradlew --no-daemon test --tests 'com.nexabuilder.api.BulkListSoftDeleteIntegrationTest'`
  - 전체(CI 와 같은 조건): `sh ./gradlew --no-daemon clean test` — 약 6분, 직전 회차 612건 통과
  - 패키징: `sh ./gradlew --no-daemon bootJar -x test` — 약 6초
  - 집계는 `build/test-results/test/*.xml` 의 `tests/failures/errors/skipped` 로 확인할 것(로그 요약만 믿지 말 것).

- 픽스처 (헤매지 않도록 — 목·대역 금지, 실제 배선으로만):
  - 템플릿은 `src/test/java/com/nexabuilder/api/DataAdapterListSoftDeleteIntegrationTest.java:70-114 seedList` 를 그대로 베낄 것: `@SpringBootTest @AutoConfigureMockMvc @WithMockUser(username="admin", roles={"ADMIN"})`, `UUID.randomUUID().toString().replace("-","").substring(0,8)` 접미사, `DynamicEntity.builder()...keyColumn("id")` → `entityRepository.save` → `schemaDdlService.ensureTable(entity)` → `entityService.insert(tableName, row)` → `NexaListDef.builder()...entityId(entityId).columnsJson(...)` → `listRepository.save`. **정리(cleanup) 없음**, `@DirtiesContext` **금지**(컨텍스트 재생성이 테스트 JVM 종료를 더 느리게 만든다).
  - 휴지통 만들기: `list.setDeletedAt("20260922000000")` 후 `listRepository.save(list)` — `deletedAt` 은 `yyyyMMddHHmmss` **문자열**이다(`DataAdapterListSoftDeleteIntegrationTest:150-152` 와 동일).
  - **세션 사용자가 반드시 필요하다.** `BulkListController:195` 는 `session.getAttribute("nexabuilder.user")` 로 `User` 를 읽고 `hasExplicitOrAdmin` 이 null 이면 `AccessDeniedException` 을 던진다. `@WithMockUser` 만으로는 부족하다. `PivotViewIntegrationTest:100-104` 의 형태를 쓸 것: `User u = new User(); u.setUserId("admin"); u.setUserName("admin"); u.setAdmin(true); u.setActive(true); u.setRoleId("ADMIN");` → `post(url).sessionAttr("nexabuilder.user", u)`.
  - 본문: `/bulk/delete` 는 `{"keys":["r0_<sfx>"]}`, `/bulk/update` 는 `{"keys":["r0_<sfx>"],"values":{"name":"changed"}}`, `contentType(MediaType.APPLICATION_JSON)`.

- 위험과 피할 것:
  - **열린 PR 6건과 파일이 겹치면 안 된다** — #37·#38·#39 = `ListExportController`, #40 = `DataAdapterService`, #41 = `NexaUiService`, #42 = `CsvImportController`·`EntityService`·`CsvSchemaInferenceService`. `BulkListController` 는 여섯 건 모두를 비껴간다. **막힌 PR 의 재제출·cherry-pick 금지**(운영자 지침).
  - `hasExplicitOrAdmin`·`ScreenPermissionService`·`EntityService.delete/update`·`AuditService` 를 **건드리지 말 것**. 권한 정책은 보호 경로이고, `EntityService` 는 PR #42 가 이미 만지고 있으며 Groovy·워크플로가 공유하는 계약이다.
  - `NexaListRepository` 에 `findByListIdAndDeletedAtIsNull` 류의 새 쿼리 메서드를 **추가하지 말 것** — 리포지터리 계약을 넓히지 않고 컨트롤러 안에서 `getDeletedAt()` 만 보면 끝난다(다른 다섯 경로도 그렇게 한다).
  - 감사 로그에 `values` 원문이 그대로 들어가는 것(`BulkListController:157`)은 이번 과제가 **아니다** — PII 계약 설계가 선행이고 범위를 넘는다. 건드리지 말 것.
  - 워크플로 완화·재시도·타임아웃 변경 절대 금지. `.github/` 는 한 줄도 바꾸지 말 것.
  - 회귀 가드(수용 기준 3)를 **먼저 초록으로** 확인하고 나서 수정하라 — 수정 전에 그것이 빨갛다면 픽스처가 권한/세션에서 걸린 것이고 결함 증명이 아니다.

- 차선 후보: README·agent.md·deployment.md·docs/roadmap.md 의 낡은 JDK17/Boot3.4.4/v1.2.x 표기를 `build.gradle.kts` 기준(Java21 toolchain, Boot 4.1.1, version 1.26.0)으로 정렬 (2/1/S). 문서만 건드려 열린 PR 6건과 충돌이 없다. 대체된 옛 가이드는 남기지 말고 덮어쓸 것(운영자 지침). 문서 변경이라도 `sh ./gradlew --no-daemon clean test` 는 돌려 초록을 확인할 것.
