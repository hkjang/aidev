- 과제: 수정 과제 — PR #50 의 CI 실패를 증거로 종결(코드·워크플로 원인 아님)하고, `ListFilterViewController.save` 가 **존재하지 않는/휴지통에 있는 목록**에도 개인 저장 필터 뷰를 만들어 영구 고아 행을 남기는 게이트 누락을 고친다 (가치 3 / 위험 2 / 작업량 S)

## 먼저 — CI 실패(우선 과제)의 처리

**이 브랜치에는 CI 를 깨뜨릴 코드가 없다. 워크플로를 느슨하게 만드는 것은 금지이고, 그럴 필요도 없다.** 이번 세션에서 직접 실행해 확인한 것:

- `git merge-base master HEAD` = `a3ca1438f9ee9135284a782b5e615baeb5eeebf0` (= `master@a3ca143`)
- `git diff --stat master... -- .github/ build.gradle.kts settings.gradle.kts gradle/` → **빈 출력** (워크플로·빌드 설정 변경 0)
- `git log --oneline -3` → `a3ca143` / `ee7f786` / `c446d24`. 빌드·CI 설정을 건드린 최신 커밋은 2026-09-28 의 버전 범프 `c446d24` 이고, 그 위 `ee7f786`(PR #36)은 **CI 를 통과해 머지됐다**.
- `git log master.. --all --name-only -- src/main` → 열린 auto 브랜치 13개 중 `.github/**` 를 건드린 것은 **하나도 없다**.

원인은 저장소 밖이며 앞선 14회차가 같은 서명으로 종결했다: 2026-09-30-010003 셰퍼드 회차가 **인증된** `gh api .../jobs` 로 확보한 잡 주석("The job was not started because recent account payments have failed or your spending limit needs to be increased", `steps: []` — 체크아웃조차 실행되지 않음)과 2026-09-30-065545 회차가 측정한 공개/비공개 상관(같은 계정·같은 시간대에 PUBLIC 6곳은 분 단위 success, PRIVATE 2곳만 3~9초 failure) = **비공개 저장소 GitHub Actions 과금 차단**. 실패 런의 소요 시간(3·4·7·9·39초 vs 성공 런 470~611초)과 `output.title/summary/text` 전부 null, 같은 커밋 `analyze (java)` skipped 이 같은 서명이다.

**구현자가 할 일 / 하지 말 일**
- **하지 말 것**: `.github/workflows/**` 수정, `continue-on-error`·`timeout-minutes` 완화, 테스트 제외(`-x test`, `--tests` 축소), 재시도·재제출·cherry-pick.
- **할 것**: 이 세션의 `gh` 인증 상태를 한 번 확인하고(미인증이면 **PR #50 의 잡 로그 본문은 "미확인" 으로 남길 것** — 추정을 사실로 쓰지 말 것), 위 두 `git diff` 를 **직접 재실행해 빈 출력임을 원장에 기록**한 뒤, CI 가 돌렸어야 할 두 명령(아래 검증 명령)을 로컬에서 재현해 통과를 증거로 남길 것. 커밋 후에도 `git diff --stat HEAD~1 -- .github/ build.gradle.kts settings.gradle.kts gradle/` 가 빈 출력임을 확인할 것.

그 자리를 채우는 실재 결함이 아래 과제다.

## 과제 본문

- 왜: `ListFilterViewController.save:46-81`(이번에 파일 전체 110줄을 열어 확인)은 `@PathVariable String listId` 를 **한 번도 검증하지 않는다**. `viewName` 공백만 거른 뒤 곧바로 `target.setListId(listId)` → `repository.save(target)` 하고, 그 전에 `repository.clearDefaults(listId, user)` 로 쓰기까지 한다. 목록의 존재 여부도 `deleted_at` 도 보지 않으므로 ① `POST /api/v1/lists/bogus-does-not-exist/views` 가 200 + 새 행을 돌려주고 ② 운영자가 휴지통에 넣어 빌더 목록·런타임 8화면·어댑터·내보내기·대량작업·리포트·전역검색에서 **모두 사라진** 목록에도 새 개인화가 계속 쌓인다. `nexa_list_filter_view` 에 소유 목록이 없는 행은 어느 화면에서도 다시 보이지 않고 지울 UI 도 없어 **영구 고아**다(`view_id` 는 IDENTITY, FK 없음 — `NexaListFilterView` 에 `@ForeignKey`/`@ManyToOne` 이 없음을 확인).
- 고치면: 이 저장소가 7회차에 걸쳐 여섯 경로(`NexaUiService.requireLiveList`·`DataAdapterService.queryList`·`ListExportController.resolveExportable`·`ListInlineEditController`·`BulkListController.resolveTarget`·`ReportService.generate`)와 `/app` 런타임 8화면·`TrashController.permanentDelete`·`UiBuilderController.saveList`·`GlobalSearchService` 에 채워 넣은 "휴지통 행은 `/admin/trash` 밖에서 살아 있는 것처럼 동작하지 않는다" 계약이 **쓰기 경로 한 곳 더**에서 성립하고, 잘못된 listId 로 들어온 요청이 조용히 DB 에 쌓이지 않는다.

- 수용 기준:
  1) **없는 listId 로 POST** → HTTP **400** + 본문 `{"success":false,"message":"List not found: <listId>"}`. 그리고 주입한 실제 `NexaListFilterViewRepository.findByListIdAndUserIdOrderByViewIdAsc(listId, "admin")` 이 **빈 리스트**임을 단언(응답 코드만 보지 말 것 — 행이 정말 생기지 않았음을 리포지토리로 확인).
  2) **휴지통 목록으로 POST** → 같은 400 + 같은 문구, 행 생기지 않음. `deleted_at` 은 직접 SQL UPDATE·목·손수 만든 대역이 아니라 **프로덕션 경로 `DELETE /api/v1/admin/lists/{listId}`** 로 찍을 것(200 + `$.message == "trashed"` 를 먼저 단언해 픽스처가 정말 soft-delete 했음을 고정 — `UiBuilderController.deleteList:56-66` 이 `ApiResponse.ok("trashed")` 를 돌려주는 것을 이번에 확인했다).
  3) **회귀 가드 — 수정 전에도 초록이어야 한다**: 살아 있는 목록에서 ㉠ 첫 저장이 200 + 행 1개 생성 ㉡ 같은 `viewName` 재저장이 행을 늘리지 않고 `filterValuesJson` 을 덮어쓴다(`save` 의 (listId,userId,viewName) 업서트 계약) ㉢ `isDefault:true` 저장이 기존 기본값을 내리고 이것만 `true` 가 된다(`clearDefaults` 계약) ㉣ `viewName` 공백이 여전히 400 `"viewName 필수"`. 이 넷이 수정 전에도 초록이어야 "휴지통/없는 목록 경로만 깨져 있었다" 가 실행으로 증명된다.
  4) 휴지통 목록의 **GET(목록 조회)과 DELETE(개별 삭제)는 동작이 변하지 않는다** — 이미 쌓인 고아 뷰를 사용자가 지울 수 있어야 하므로 의도적으로 게이트를 걸지 않는다. 이것도 테스트로 고정할 것(휴지통 목록에 대해 GET 200, 그 목록의 기존 뷰 DELETE 200 + `$.message == "deleted"`).

- 건드릴 파일 (프로덕션 **1개**):
  - `src/main/java/com/nexabuilder/api/controller/ListFilterViewController.java`
    - 클래스에 `private final NexaListRepository listRepository;`(`com.nexabuilder.core.ui.NexaListRepository` — `BulkListController:59` 와 같은 타입·같은 `@RequiredArgsConstructor` 주입 형태) 필드 **하나** 추가. 기존 `repository` 필드명은 그대로 둘 것.
    - `save(...)` **맨 앞**(= 기존 `viewName` 검사보다 **앞**, `currentUser()`·`clearDefaults`·`save` 보다 당연히 앞)에 게이트를 넣는다: `listRepository.findById(listId)` 가 비어 있거나 `getDeletedAt() != null` 이면 `return ResponseEntity.badRequest().body(ApiResponse.error("List not found: " + listId));`
      - 문구는 여섯 경로와 **글자 단위로 같게** (`"List not found: " + listId`).
      - 이 파일의 국지적 관례는 예외를 던지는 것이 아니라 `ResponseEntity.badRequest().body(ApiResponse.error(...))` 직접 반환이다(:52-54 의 `"viewName 필수"` 가 그 형태). `IllegalArgumentException` 을 쓰지 말 것 — HTTP 결과는 같지만 diff 가 커지고 이 파일에 없는 스타일이다.
      - **없는 목록과 휴지통 목록을 구분하지 말 것**(같은 400·같은 문구). 여섯 경로와 동일한 의도된 동작이다.
      - `deletedAt` 은 `NexaListDef:135` 에서 `String` + `@Column(name="deleted_at", length=20)` 이다. 앞선 네 회차 관례대로 **`!= null` 만** 쓰고 `isBlank()`/`StringUtils.hasText()` 를 넣지 말 것.
    - 게이트 아래의 `viewName` 검사·`currentUser()`·업서트 탐색(`findByListIdAndUserIdOrderByViewIdAsc` + `filter(...).findFirst()`)·`regDttm` 기본값·`clearDefaults`/`isDefault` 처리·`repository.save(target)` 는 **한 글자도 바꾸지 말 것**. 구조 재배치(업서트 탐색과 목록 조회를 합치기 등) 금지.
    - `list(...)`·`delete(...)`·`currentUser()`·`now()`·`DTTM` 는 불간섭(수용 기준 4).
  - 테스트 **1개 신규**: `src/test/java/com/nexabuilder/api/ListFilterViewSoftDeleteIntegrationTest.java` (이 컨트롤러에는 기존 테스트가 **없다** — `src/test/java/com/nexabuilder/api/` 에 `*FilterView*` 파일이 없음을 확인했다. 그래서 이번에는 기존 파일에 끼워 넣을 자리가 없고 신규가 맞다.)

- 검증 명령:
  - 좁게(먼저): `sh ./gradlew --no-daemon test --tests 'com.nexabuilder.api.ListFilterViewSoftDeleteIntegrationTest'` — **수정 전에 한 번 돌려 빨간 출력을 원장에 남길 것**(수용 기준 1·2 가 빨갛고 3·4 는 초록이어야 한다).
  - CI 와 같은 두 명령(반드시 둘 다):
    - `sh ./gradlew --no-daemon clean test` — 약 6분. 직전 회차 기준 615건 / failures 0 / errors 0 / skipped 0.
    - `sh ./gradlew --no-daemon bootJar -x test` — `build/libs/nexabuilder-1.26.0.jar` 생성.
  - `gradlew` 는 git 모드 100644 라 **`sh` 를 경유**해야 한다.

- 위험과 피할 것:
  - **열린 PR 충돌 0 을 이번에 직접 확인했다.** `git log master.. --all --name-only --pretty=format:"COMMIT %h %d" -- src/main` 로 열린 브랜치 전부의 프로덕션 파일을 뽑았고 점유 목록은 `GlobalSearchService`(#50/1422) · `UiBuilderController`(1332) · `TrashController`(1222) · `RuntimeUiController`(1112) · `ListInlineEditController`(0932) · `ReportService`(0832/0722) · `BulkListController`(0622) · `CsvImportController`/`EntityService`/`CsvSchemaInferenceService`(0516) · `NexaUiService`(0242) · `DataAdapterService`(0132) · `ListExportController`(0035/2142/openpdf-3) · `Jackson2JsonConfig` · `GlobalExceptionHandler`(webjars-locator) 다. `ListFilterViewController.java` 는 **어디에도 없다**. `NexaListRepository` 도 어느 브랜치에서도 수정되지 않았다 — 그래서 **주입만 하고 리포지토리에 쿼리 메서드를 추가하지 말 것**(`findByIdAndDeletedAtIsNull` 같은 신규 메서드 금지; 공유 계약이다).
  - `NexaListFilterViewRepository` 에 메서드를 추가하지 말 것. `NexaListFilterView` 엔티티에 FK·`@ManyToOne` 을 새로 만들지 말 것(마이그레이션이 필요해지고 위험 구역이다). **`src/main/resources/db/migration/**` 불간섭.**
  - `ScreenPermissionService` 를 이 컨트롤러에 새로 넣지 말 것 — 이 엔드포인트는 원래 세션 `User` 도 권한 서비스도 쓰지 않고 `SecurityContextHolder` 이름만 본다. 권한 검사 부재는 **별개 과제**이고 보호 경로다(`ideas.json` 에 별 항목으로 있다).
  - `AuditService` 로 감사 로그를 새로 추가하지 말 것.
  - `templates/runtime/list.html`(:479 GET / :525 DELETE / :561 POST 가 이 세 엔드포인트를 실제로 호출한다 — 이번에 확인) 은 **건드리지 말 것**. 살아 있는 목록에서는 동작이 완전히 동일하므로 프런트 변경이 필요 없다.
  - **혼란 주의 — 상태 코드 선례가 섞여 있다**: `BulkListController:194` 와 `RuntimeUiController`(8곳)는 `IllegalArgumentException("List not found: " + listId)` → 400 인데 `ListInlineEditController:73` 은 같은 문구로 **404** 를 돌려준다. 이번 과제는 **400** 으로 맞추고(다수 선례 + 이 파일의 `badRequest()` 관례), `ListInlineEditController` 를 "통일" 하려고 손대지 말 것(PR #46 점유이며 별개 계약 결정이다).
  - 테스트 픽스처: `@SpringBootTest @AutoConfigureMockMvc @WithMockUser(username = "admin", roles = "ADMIN")` 에 **`properties` 를 붙이지 말 것**(컨텍스트 캐시 재사용 — `DataAdapterListSoftDeleteIntegrationTest:54-57` 과 같은 설정). **`@DirtiesContext` 금지** (이 저장소는 컨텍스트마다 `taskScheduler` 종료로 ~20초를 쓴다).
  - `currentUser()` 는 `SecurityContextHolder` 의 `auth.getName()` 이므로 `@WithMockUser(username="admin")` 이면 저장되는 `user_id` 는 `"admin"` 이다. 리포지토리 단언에 이 값을 쓸 것.
  - 픽스처 목록은 `NexaListDef` 를 **실제 `NexaListRepository` 에 저장**해 만들 것(listId 는 `"list_lfv_" + UUID 8자` 관례). `NexaListDef` 에 `nullable = false` 컬럼이 **없음을 확인**했으므로 최소 필드(listId/listName/active)만으로 충분하다. **`SchemaDdlService` 로 물리 테이블을 만들 필요가 없다** — 이 컨트롤러는 목록 데이터를 절대 조회하지 않는다. 정리(teardown) 없음이 저장소 관례다.
  - grep 결과를 증거로 쓰지 말 것. "지금 무엇을 돌려주는가" 는 **수정 전 빨간 테스트 출력**으로 고정할 것.

- **미확인 (추측으로 쓰지 말고 실행으로 확인할 것)**:
  1) 수정 전 없는/휴지통 listId 의 POST 가 실제로 **200 + 행 생성**인지. 이번 정찰은 코드만 읽었고 실행하지 않았다(`save` 에 어떤 검증도 없다는 것은 파일 전체를 읽어 확인). 먼저 빨간 테스트로 관측할 것.
  2) `POST/GET/DELETE /api/v1/lists/{listId}/views` 가 `@WithMockUser` 만으로 200 인지, 세션 `nexabuilder.user` 속성이 추가로 필요한지. 이 컨트롤러 코드에는 세션 접근이 **없으므로** `@WithMockUser` 만으로 충분할 것으로 보이지만 **확인하지 않았다**. 만약 401/403 이 나면 `PivotViewIntegrationTest` 형태의 `sessionAttr("nexabuilder.user", User)` 를 더할 것(내보내기 테스트가 쓰는 형태).
  3) 성공 응답 봉투의 어느 키에 무엇이 오는지. **선례 경고**: 이 저장소의 `ApiResponse.ok(String)` 은 `ok(T data)` 가 아니라 **`ok(String message)` 오버로드**로 바인딩되므로 `delete` 의 `"deleted"` 는 `$.data` 가 아니라 **`$.message`** 에 온다(직전 회차들이 `"purged"`/`"trashed"` 에서 실제로 걸렸다). 반면 `save`/`list` 는 객체/리스트를 넘기므로 `$.data` 다. 총 개수 필드는 `$.total` 이 아니라 **`$.totalCount`** 다(`ApiResponse:17`). 단언이 `No value at JSON path ...` 로 빨갛다면 프로덕션 결함이 아니라 단언 오류이니 **프로덕션 응답 형태를 바꾸지 말고 단언을 실제 봉투에 맞출 것**.
  4) PR #50 의 잡 로그 본문 — `gh` 미인증이면 열 수 없다. **미확인으로 남길 것.**

- 차선 후보: **README·`agent.md`·`docs/deployment.md`·`docs/roadmap.md` 의 낡은 JDK17/Boot3.4.4/v1.2.x 표기를 Java21/Boot4.1.1/v1.26.0 으로 정렬** (2/1/S) — 16회차 연속 차선 후보이고 열린 PR 과 충돌 0 이다. `build.gradle.kts` 가 정본이다. 대체된 옛 가이드 문장은 남기지 말고 **덮어쓸 것**(정본이 둘이 되지 않게). 주과제가 성립하지 않으면(예: 위 미확인 1 이 이미 400 이었다면) 즉시 이것으로 갈 것.
