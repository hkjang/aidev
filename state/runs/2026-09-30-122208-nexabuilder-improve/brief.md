# 과제서 — 2026-09-30-122208-nexabuilder-improve

## 먼저: 우선 과제(CI 실패)에 대한 판정

- **이 브랜치에는 고칠 빌드·워크플로 코드가 없다.** 이번 세션에서 직접 실행해 확인했다:
  - `git diff --stat master...` → **빈 출력**
  - `git diff --stat master... -- .github/ build.gradle.kts settings.gradle.kts gradle/` → **빈 출력**
  - 베이스는 `master@a3ca143`. 빌드·CI 설정을 건드린 최신 커밋은 2026-09-28 의 버전 범프 `c446d24` 이고, 그 위 `ee7f786`(PR #36)은 CI 를 통과해 머지됐다.
- 원인은 **12회차째 같은 서명**이며 저장소 밖이다. 2026-09-30-010003 셰퍼드 회차가 **인증된** `gh api .../jobs` 로 확보한 잡 주석("The job was not started because recent account payments have failed or your spending limit needs to be increased", `steps: []` — 체크아웃조차 실행되지 않음)과 2026-09-30-065545 회차가 측정한 공개/비공개 상관(같은 계정·같은 시간대에 PUBLIC 6곳 분 단위 success, PRIVATE 2곳만 3~9초 failure)으로 이미 증명돼 있다 = 비공개 저장소 GitHub Actions 과금 차단. PR #47 의 런 로그 본문은 이 세션에서 **미확인**(`gh` 미인증).
- 따라서 구현자는 **`.github/workflows` 를 한 줄도 건드리지 말 것.** 재시도·`continue-on-error`·타임아웃 완화·`--tests` 축소·재제출·cherry-pick 금지(워크플로 완화는 명시 금지 사항). 대신 **CI 가 돌렸어야 할 두 명령을 로컬에서 그대로 재현해 통과를 확인**하고 그 출력을 원장에 남긴다(아래 검증 명령).
- 그 자리를 채울 실재 결함이 아래 과제다.

---

- 과제: 휴지통 하드삭제 엔드포인트가 **휴지통에 없는(살아 있는) 목록·폼·엔티티 정의를 영구 삭제**한다 — `TrashController.permanentDelete` 의 `deleted_at` 게이트 누락 (가치 4 / 위험 2 / 작업량 S)

- 왜: `DELETE /api/v1/admin/trash/{type}/{id}` 는 `listRepository.deleteById(id)` / `formRepository.deleteById(id)` / `entityRepository.deleteById(id)` 를 **아무 조건 없이** 호출한다(`TrashController.java:88-99`, 이번에 파일 전체를 읽어 확인). 같은 클래스 javadoc(:16-29)이 "Each DELETE on a designer artifact stamps a `deleted_at` timestamp instead of issuing a physical DELETE so accidents stay reversible" 를 약속하고 이 엔드포인트를 "Hard delete — drop the row permanently. **No undo.**"(:87) 로 규정하는데, 정작 "휴지통에 있는 것만 비운다" 는 전제를 **코드가 어디에서도 검사하지 않는다**. 즉 `/admin/trash` UI 는 휴지통 목록의 행만 이 URL 로 보내지만(`templates/admin/trash.html:160` 확인), 그 URL 을 아는 호출자는 **운영 중인 살아 있는 목록 정의를 되돌릴 수 없게 지울 수 있다** — 이 저장소가 6회차에 걸쳐 목록 soft-delete 를 여섯 경로(`NexaUiService.requireLiveList`·`DataAdapterService.queryList`·`ListExportController.resolveExportable`·`ListInlineEditController`·`BulkListController.resolveTarget`·`ReportService.generate`)에 채워 넣어 지킨 안전망을, 이 한 엔드포인트가 통째로 우회한다. 게이트를 넣으면 하드삭제가 "휴지통을 비우는 동작" 이라는 자기 계약대로만 동작한다.

- 수용 기준:
  1) **살아 있는**(`deletedAt == null`) 목록에 `DELETE /api/v1/admin/trash/list/{listId}` → 200 이 아니라 **400** + `{"success":false,"message":…}`(`ApiResponse.error`) 이고, 호출 뒤 `listRepository.findById(listId)` 가 **여전히 존재**한다(행이 지워지지 않았음을 리포지토리로 직접 단언 — 응답 코드만 보지 말 것).
  2) `form`·`entity` 두 타입도 같다: 살아 있는 `NexaFormDef` / `DynamicEntity` 는 400 이고 `formRepository.findById` / `entityRepository.findById` 가 존재한다.
  3) **회귀 가드**: `deletedAt` 이 찍힌(휴지통에 있는) 목록·폼·엔티티는 **그대로 200 + `"purged"`** 이고 호출 뒤 `findById` 가 **비어 있다**(정말 하드삭제된다 = 게이트를 넣어 기능을 죽이지 않았다는 증명). 이 가드가 수정 전에도 초록이어야 하고, 수정 후에도 초록이어야 한다.
  4) `DELETE /api/v1/admin/trash/bogus/{id}`(알 수 없는 타입)는 **기존과 같이** 400 `Unknown type: bogus`.
  5) 존재하지 않는 id → 200 `"purged"` 가 아니라 **404** + `Row not found: {type}/{id}`(`restore` 가 이미 쓰는 문구·상태와 같게). *주의: 현재 Spring Data JPA 의 `deleteById` 가 없는 id 에 예외를 던지는지 no-op 인지는 이 세션에서 **실행으로 확인하지 않았다(미확인)**. 존재 게이트를 넣으면 어느 쪽이든 404 로 수렴하므로 구현에는 영향이 없지만, "수정 전 현재 동작" 을 원장에 적을 때는 추정으로 쓰지 말고 빨간 테스트 출력으로 관측할 것.*

- 건드릴 파일 (프로덕션 **1개**):
  - `src/main/java/com/nexabuilder/api/controller/TrashController.java:88-99` — `permanentDelete(String type, String id)`. 조건 없는 `deleteById` 3분기를 **타입별 `findById` → 판정 → 삭제** 로 바꾼다. 판정 순서: ① 없음 → 404 `Row not found: {type}/{id}` ② `getDeletedAt() == null` → 400(휴지통에 없음을 알리는 메시지) ③ 그 외 → `delete(row)` 후 200 `"purged"`. 알 수 없는 타입은 기존 `default` 그대로 400 `Unknown type: {type}`.
    - `deletedAt` 은 세 엔티티 모두 `String` + `@Column(name="deleted_at", length=20)` 이다 — 이번에 세 파일을 직접 확인했다(`NexaListDef.java:134-135`, `NexaFormDef.java:51-52`, `DynamicEntity.java:67-68`). 그러므로 **`!= null` 만** 쓰고 `isBlank()` 같은 추가 판정을 넣지 말 것(직전 회차 `RuntimeUiController` 와 같은 관례).
    - 같은 파일 `restore`(:64-85)의 `findById(...).map(...).orElse(false)` 형태와 **국지적 관례를 맞출 것**. 이 파일은 이미 세 타입을 세 분기로 늘어놓고 있으므로 제네릭 헬퍼를 발명하지 말고 같은 모양을 유지하되, 응답 문구·상태 매핑은 switch 밖에서 한 번만 만든다(세 갈래 결과가 필요하면 파일 안의 private enum 하나까지는 허용).
    - `deleteById` → `delete(row)` 로 바뀌는 것은 의도된 것이다(이미 조회한 행을 지운다). 리포지토리에 쿼리 메서드를 **추가하지 말 것**.
  - 테스트 **1개 신규**: `src/test/java/com/nexabuilder/api/TrashPurgeIntegrationTest.java` — `@SpringBootTest @AutoConfigureMockMvc @WithMockUser(username="admin", roles={"ADMIN"})` + `MockMvc`. 이 조합은 `DataAdapterListSoftDeleteIntegrationTest`(:56-58 확인)와 **동일한 컨텍스트 설정**이라 Spring 컨텍스트 캐시를 재사용한다(properties 를 새로 주지 말 것 — `ReportIntegrationTest` 처럼 `properties` 를 주면 컨텍스트가 하나 늘고 종료마다 ~20초를 더 쓴다). `@DirtiesContext` 금지.
    - 픽스처는 `DataAdapterListSoftDeleteIntegrationTest.seedList`(:69~) 형태를 그대로 따를 것: `UUID.randomUUID().toString().replace("-","").substring(0,8)` 8자 접미사, 실제 `DynamicEntityRepository`·`NexaListRepository`(폼은 `NexaFormRepository`)로 저장, 정리 없음. `deletedAt` 은 **실제 리포지토리에 `setDeletedAt(LocalDateTime.now().format(ofPattern("yyyyMMddHHmmss")))` 로 찍을 것**(직접 SQL UPDATE·목·손수 만든 대역 금지 — 운영자 지침).
    - `SchemaDdlService` 로 물리 테이블을 만들 필요는 없다(이 엔드포인트는 정의 행만 다룬다). 만들지 않아도 되는 것을 만들지 말 것.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  1) 먼저 **빨간 것을 보고** 그 출력을 원장에 남긴다 — `sh ./gradlew --no-daemon test --tests 'com.nexabuilder.api.TrashPurgeIntegrationTest'`
  2) 수정 후 같은 명령 → `tests=N skipped=0 failures=0 errors=0`
  3) CI 가 돌렸어야 할 두 명령 재현: `sh ./gradlew --no-daemon clean test` (5~6분, 직전 회차 619건/실패0) 와 `sh ./gradlew --no-daemon bootJar -x test` (약 6초, `build/libs/nexabuilder-1.26.0.jar`)
  - `gradlew` 는 git 모드 100644 라 **`sh ./gradlew`** 로 부를 것.

- 위험과 피할 것:
  - **`.github/workflows` 를 건드리지 말 것.** 워크플로를 느슨하게 해 통과시키는 것은 금지이며, 이번 실패는 저장소 밖 원인이다(위 판정).
  - **열린 PR 이 점유한 파일을 건드리지 말 것.** #37·#38·#39=`ListExportController`, #40=`DataAdapterService`, #41=`NexaUiService`, #42=`CsvImportController`/`EntityService`/`CsvSchemaInferenceService`, #43=`BulkListController`, #44·#45=`ReportService`, #46=`ListInlineEditController`, **#47=`web/controller/RuntimeUiController`**. `TrashController` 는 이 중 어디에도 없다(그래서 이 과제를 골랐다).
  - **`restore`(:64-85)·`trashed`(:41-61)·`toRow`·`firstNonBlank` 를 건드리지 말 것.** 특히 `restore` 가 살아 있는 행에도 200 을 주는 것은 **별개 사안이고 이번 범위 밖**이다(무해하고 계약이 모호하다).
  - **엔티티 하드삭제가 물리 테이블을 남기는 문제(고아 테이블)에 손대지 말 것** — `SchemaDdlService` DDL 경로는 위험 구역이고 이번 과제와 계약이 다르다.
  - `templates/admin/trash.html` 을 바꿀 필요가 없다. UI 는 휴지통 목록의 행만 이 URL 로 보내므로 게이트를 넣어도 **정상 사용 동작이 변하지 않는다**(:148·:160 확인). 화면을 같이 고치려 하지 말 것.
  - `ScreenPermissionService`·`AuditService`·`GlobalExceptionHandler` 불간섭. 이 컨트롤러는 `ResponseEntity` 를 직접 만들므로 예외 핸들러 경로를 타지 않는다 — `IllegalArgumentException` 을 던지는 형태로 바꾸지 말고 `restore` 와 같이 `ResponseEntity.status(...).body(ApiResponse.error(...))` 로 반환할 것.
  - 감사 로그를 새로 추가하지 말 것(운영자 지침: 감사에 넘기는 값은 식별자만 — 새 계약을 여는 것은 이번 범위 밖).
  - 문자열 grep 을 증거로 제출하지 말 것. 실패 재현은 **빨간 테스트 출력**으로, 삭제 여부는 **리포지토리 조회**로 단언할 것.

- 차선 후보: README·`agent.md`·`docs/deployment.md`·`docs/roadmap.md` 의 낡은 JDK17 / Spring Boot 3.4.4 / v1.2.x 표기를 실제(`build.gradle.kts` 의 Java21 toolchain·Boot 4.1.1·version 1.26.0)로 정렬 — 11회차 연속 차선으로 남아 있고 열린 PR 과 충돌 0, 문서만 건드린다. 대체된 옛 가이드는 남기지 말고 덮어쓸 것(운영자 지침). 검증은 `sh ./gradlew --no-daemon bootJar -x test` 로 충분.
