# 과제서 (2026-09-20)

- 과제: `POST /api/v1/builder/lists/{listId}/data` 가 휴지통(soft-delete)에 든 목록의 행을 그대로 돌려주는 구멍을 막고, 목록 "살아 있음" 판정을 `NexaUiService` 헬퍼 하나로 모은다 (가치 3 / 위험 2 / 작업량 S)
- 왜: 2026-09-19 에 내보내기 3종(`ListExportController.resolveExportable`)은 막았지만, 같은 데이터를 주는 `UiBuilderController.listData` → `NexaUiService.listData`(`NexaUiService.java:87-95`)는 여전히 `findById` 만 하고 `deletedAt` 을 안 본다. 런타임 `/app/list/<id>` 와 `GET /api/v1/builder/lists/{id}`(→`listDefinition`, :47-55)는 "List not found" 로 막히는데 `/data` 만 열려 있어 규칙이 엇갈리고, `listDefinition`·`listData`·`resolveExportable` 세 곳이 각각 같은 예외를 만들고 있어 규칙을 바꿀 때 놓치기 쉽다.
- 수용 기준:
  1) `deleted_at` 이 설정된 목록에 `POST /api/v1/builder/lists/{id}/data` 를 치면 200 + 행 대신 400 JSON(`IllegalArgumentException("List not found: …")` → `GlobalExceptionHandler`)이 온다. 살아 있는 목록은 기존대로 200 + `totalCount`.
  2) `NexaUiService` 에 `NexaListDef requireLiveList(String listId)`(이름은 자유) 헬퍼가 생기고, `listDefinition` 과 `listData` 가 둘 다 그 헬퍼를 쓴다(예외 메시지는 기존 `"List not found: " + listId` 그대로 — `ListExportIntegrationTest.softDeletedListIsNotExportableInAnyFormat` 과 런타임 화면이 이 문구/400 에 기대고 있음).
  3) 통합 테스트가 증명할 것: (a) 살아 있는 목록의 `/data` 는 200, (b) 같은 목록에 `setDeletedAt("2026…")` 저장 후 `/data` 는 400, (c) 기존 `BuilderApiIntegrationTest.listFormAndAtomMetadataDriveRuntimeApis`(:209-214, `sample-list` 의 `/data` 200) 가 그대로 초록. 헬퍼 도입 전에 (b)만 빨간 것을 먼저 확인하고 고칠 것(TDD).
- 건드릴 파일:
  - `src/main/java/com/nexabuilder/core/ui/NexaUiService.java:47-55` `listDefinition` — `findById + deletedAt` 판정을 새 private/package 헬퍼 `requireLiveList(listId)` 로 뽑아내고 기존 주석("Soft-deleted lists (see migration 027)…")을 헬퍼로 옮긴다.
  - `src/main/java/com/nexabuilder/core/ui/NexaUiService.java:87-89` `listData` — `findById(...).orElseThrow(...)` 를 `requireLiveList(listId)` 로 바꾼다. `sqlId` 분기·`entityService.list` 호출은 그대로.
  - (선택, 헬퍼를 public 으로 둘 경우) `src/main/java/com/nexabuilder/api/controller/ListExportController.java` `resolveExportable` — 자체 `findById`+`deletedAt` 블록을 `uiService.requireLiveList(listId)` 호출로 바꿔도 됨. 권한 판정(`ScreenPermissionService`) 순서는 바꾸지 말 것. 시간이 없으면 생략해도 수용 기준은 충족.
  - `src/test/java/com/nexabuilder/api/ListExportIntegrationTest.java` 옆에 새 클래스 `src/test/java/com/nexabuilder/api/BuilderListDataSoftDeleteIntegrationTest.java` (또는 `ListExportIntegrationTest` 에 테스트 1건 추가 — 픽스처 `seed()`(:73-118)가 엔티티+테이블+행+목록을 이미 만들어 주므로 재사용이 가장 싸다). `@SpringBootTest + @AutoConfigureMockMvc + @WithMockUser(username="admin", roles={"ADMIN"})`. `/data` 는 세션 user 를 안 보므로 `sessionAttr("nexabuilder.user", …)` 불필요(미확인 — 403 이 나오면 export 테스트처럼 `adminSessionUser()` 를 세션에 넣을 것).
- 검증 명령 (저장소 루트, `gradlew` 가 100644 라 `sh` 필요):
  - `sh ./gradlew --no-daemon test --tests 'com.nexabuilder.api.ListExportIntegrationTest' --tests 'com.nexabuilder.api.BuilderApiIntegrationTest'` (새 테스트를 별도 클래스로 두면 그 클래스도 `--tests` 에 추가)
  - 마지막에 `sh ./gradlew --no-daemon cleanTest test` (전체 573개 + 신규, 수 분) 와 `sh ./gradlew --no-daemon bootJar -x test` (릴리즈 검증과 동일).
  - 이번 정찰 세션에서는 권한 문제로 gradle 을 실행하지 못했음(미실행). 명령 자체는 2026-09-19 회차에서 그대로 돌았던 것.
- 위험과 피할 것:
  - `@DirtiesContext` 붙이지 말 것(프로필의 "자주 깨지는 곳"). 픽스처는 UUID 접미사로 격리하고 정리하지 않는 기존 관례를 따를 것.
  - `GlobalExceptionHandler` 의 400 매핑, 예외 타입, 메시지 문구를 바꾸지 말 것 — 404 전환은 별도 회차(보류 아이디어).
  - `formDefinition(formId, includeInactive)` 의 `includeInactive` 분기(designer 가 휴지통 폼을 편집하는 경로)는 건드리지 말 것. 목록 쪽 `/data` 호출자는 런타임 뷰 7개(`templates/runtime/{kanban,tree,calendar,pivot,gantt,map,chart}.html`)와 `ListExportController` 뿐이고 designer(`static/js`, builder 템플릿)에서는 호출하지 않는 것을 grep 으로 확인함 — 런타임 뷰는 모두 `listDefinition` 을 먼저 타므로 이미 휴지통 목록을 열 수 없다. 따라서 designer 미리보기 깨짐 걱정은 해소됨.
  - `UiBuilderController.listData` 에는 `ScreenPermissionService` 호출이 없다(런타임 화면·내보내기는 있음). 이번 과제에서 권한 게이트를 추가하지 말 것 — 런타임 뷰 7개가 세션 user 없이도 동작하는지 미확인이라 위험이 다른 급.
  - `core/permission`, `infra/security`, 마이그레이션은 건드리지 않는다.
- 차선 후보: CI 에 `gradle/actions/wrapper-validation` 추가 + `docker-publish.yml` 의 `actions/checkout@v4` → `v6` 정렬 + `gradlew` git 모드 `100755` 로 바꿔 ci/codeql/release 의 `chmod +x` 스텝 3곳 제거 (3/1/S, 2026-09-20 재확인: 셋 다 여전히 그 상태). 액션의 최신 메이저 태그는 미확인이니 워크플로 파일에 넣기 전에 `gh api repos/gradle/actions/releases/latest` 로 볼 것. 코드·테스트 변경 없음이라 검증은 `bootJar -x test` + YAML 문법(`actionlint` 없으면 `python3 -c "import yaml,sys;[yaml.safe_load(open(f)) for f in sys.argv[1:]]" .github/workflows/*.yml`).
