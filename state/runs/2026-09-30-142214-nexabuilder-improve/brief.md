# 과제서 (2026-09-30, 정찰)

- 과제: 수정 과제 — PR #49 의 CI 실패를 증거로 종결(코드·워크플로 원인 아님)하고, 전역 검색(Ctrl+K)이 **휴지통에 넣은 목록·폼·엔티티를 살아 있는 것처럼 계속 돌려주는** `GlobalSearchService.search` 의 soft-delete 누락을 고친다 (가치 3 / 위험 1 / 작업량 S)

## 0. 먼저 — CI 실패 판정 (이것부터 하고, 워크플로는 건드리지 말 것)

이번 브랜치 `auto/2026-09-30-1422` 는 `master@a3ca143` 위의 **빈 브랜치**다. 이 세션에서 직접 실행해 확인했다:

- `git merge-base master HEAD` → `a3ca1438f9ee9135284a782b5e615baeb5eeebf0`
- `git diff --stat master...` → **빈 출력**
- `git diff --stat master... -- .github/ build.gradle.kts settings.gradle.kts gradle/` → **빈 출력**
- `.github/workflows/` = `ci.yml`, `codeql.yml`, `docker-publish.yml`, `release.yml` (master 와 동일, 이 브랜치의 변경 0)

즉 **이 브랜치에는 CI 를 깨뜨릴 코드가 없다.** 같은 서명의 실패가 14회차째다. 원인은 2026-09-30-010003 셰퍼드 회차가 **인증된** `gh api .../jobs` 로 확보한 잡 주석("The job was not started because recent account payments have failed or your spending limit needs to be increased", `steps: []` — 체크아웃조차 실행되지 않음)과 2026-09-30-065545 회차가 측정한 공개/비공개 상관(같은 계정·같은 시간대에 PUBLIC 6곳은 분 단위 success, PRIVATE 2곳만 3~9초 failure)으로 이미 증명돼 있다 = 비공개 저장소 GitHub Actions 과금 차단.

- **미확인**: 이 세션의 `gh` 인증 상태를 확인하지 않았다. PR #49 의 잡 로그 본문도 열어 보지 않았으므로 "PR #49 의 실패 사유" 자체는 **미확인으로 남긴다**. 추정을 사실로 쓰지 말 것.
- **금지**: `.github/workflows/*` 수정, `continue-on-error`, `timeout` 완화, 재시도 추가, `--tests` 로 CI 명령 축소, 재제출·cherry-pick. 워크플로를 느슨하게 만들어 통과시키는 것은 명시적으로 금지다.
- **해야 할 것**: CI 가 돌려야 할 두 명령(`clean test`, `bootJar -x test`)을 **로컬에서 그대로 재현해 통과**를 증거로 남기고, 원장에 '수정 과제' 로 기록한다. 그 뒤 아래 실재 결함을 고친다(지난 6회차와 같은 처리 방식이며 구현자가 전부 채택했다).

## 1. 왜

`GlobalSearchService.search`(`src/main/java/com/nexabuilder/core/search/GlobalSearchService.java:72`, `:81`, `:88`)는 엔티티·폼·목록 후보를 `findByActiveTrue()` 로만 뽑는다. 그런데 이 저장소의 휴지통은 **`active` 를 건드리지 않고 `deleted_at` 만 찍는다** — `UiBuilderController.deleteList:61-64`, `deleteForm:94-97` 이 `row.setDeletedAt(now())` + `save` 만 한다(직접 읽어 확인). 그래서 운영자가 `/admin/trash` 로 옮겨 빌더 목록(`findByDeletedAtIsNullOrderByListIdAsc`, `:30`·`:70`)·런타임·어댑터·내보내기·리포트에서 모두 사라진 정의가 **Ctrl+K 팔레트에서는 살아 있는 항목으로 계속 뜨고**, 하나 고르면 `/admin/list-designer/{listId}` 로 보내진다. 저장소가 6회차에 걸쳐 여섯 경로에 채워 넣은 "휴지통 행은 `/admin/trash` 밖에서 보이지 않는다" 는 계약이 모든 것을 인덱싱하는 이 한 곳에서 깨져 있다.

## 2. 수용 기준

1. 휴지통에 넣은 목록·폼·엔티티가 `GET /api/v1/search?q=<그 이름>` 의 `$.data.lists` / `$.data.forms` / `$.data.entities` 에서 **사라진다**(id 가 결과에 없다). `$.total` 도 그만큼 줄어든다.
2. **같은 픽스처의 살아 있는 형제 행은 계속 검색된다** — 휴지통 행 하나와 살아 있는 행 하나를 같은 접두사로 심고, 휴지통에 넣기 **전에는 둘 다** 결과에 있고 넣은 **뒤에는 살아 있는 쪽만** 남는 것을 같은 테스트에서 연속으로 단언한다(픽스처가 정말 이 경로를 지난다는 증명. 빈 결과만 보면 "쿼리가 애초에 아무것도 못 찾았다" 와 구분되지 않는다).
3. `active=false` 행은 **지금과 똑같이** 제외된다(기존 `findByActiveTrue` 계약 불변). `workflows`/`codes`/`sqls`/`users` 결과는 **한 건도 달라지지 않는다** — 이 네 타입에는 `deletedAt` 필드가 없고 휴지통이 다루지 않는다(`WorkflowDef`/`Code`/`SqlMetadata`/`User` 에 `deletedAt` 이 없음을 grep 으로 확인했다; 휴지통 `switch` 는 `"list"`/`"form"`/`"entity"` 세 가지뿐 — `TrashController:72-78`, `:92-94`).
4. 휴지통 행이 `cap`(per-type limit) 슬롯을 **먹지 않는다** — 휴지통 행 여러 건 + 살아 있는 행 1건을 심고 `limit=1` 로 요청했을 때 살아 있는 행이 나온다(수정 전에는 휴지통 행이 슬롯을 차지해 살아 있는 행이 밀려날 수 있다).
5. 테스트는 **수정 전에 빨갛고** 수정 후 초록이다. 빨간 출력을 원장에 그대로 붙일 것.

## 3. 건드릴 파일 (프로덕션 1개 + 테스트 1개)

- `src/main/java/com/nexabuilder/core/search/GlobalSearchService.java:72`(entities), `:81`(forms), `:88`(lists) — 세 mapper 람다가 휴지통 행에서 `hit(...)` 대신 **`null`** 을 돌려주게 한다. 예: `l -> l.getDeletedAt() == null ? hit("LIST", …) : null`.
  - 이 형태를 쓰는 이유: `filter(...)` 가 이미 `MatchableHit<T> m = mapper.apply(item); if (m == null) continue;`(`:140-141`)로 **null 후보를 건너뛰도록 설계돼 있다**(현재는 절대 발동하지 않는 죽은 가지다). 그래서 `filter` 시그니처·`MatchableHit`·`hit()`·`matches()`·`toMap()`·`cap` 계산(`:70`)·나머지 네 타입을 **한 글자도 바꾸지 않고** 세 람다만 고칠 수 있고, `continue` 가 `out.size()` 를 늘리지 않으므로 수용 기준 4 가 자동으로 성립한다.
  - `getDeletedAt()` 은 세 타입 모두 `String` 이다(`NexaListDef:135`, `NexaFormDef:52`, `DynamicEntity:68` — 직접 확인). **`== null` 만 쓰고 `isBlank()`/`hasText()` 를 넣지 말 것** — 앞선 세 회차(`RuntimeUiController`·`TrashController`·`UiBuilderController`)가 세운 관례다.
  - **리포지토리에 쿼리 메서드를 추가하지 말 것.** `findByActiveTrue()` → `findByActiveTrueAndDeletedAtIsNull()` 같은 교체도 하지 말 것: `NexaListRepository:8`·`NexaFormRepository:8`·`DynamicEntityRepository:14` 의 `findByActiveTrue()` 는 다른 호출자와 공유하는 계약이고, 클래스 javadoc(`:30-35`)이 "in-memory streaming over every active resource" 를 명시하므로 메모리 필터가 이 파일의 관례다.
- `src/test/java/com/nexabuilder/api/GlobalSearchSoftDeleteIntegrationTest.java` — **신규**. 이 저장소에는 전역 검색 테스트가 **하나도 없다**(`src/test` 전체에 `GlobalSearch`/`api/v1/search` 문자열 0건 — 확인). 즉 이번 수정과 회귀 가드가 동시에 이 공백을 메운다.

## 4. 테스트를 어떻게 쓸 것인가 (운영자 지침 반영)

- 컨텍스트 설정은 `src/test/java/com/nexabuilder/api/DataAdapterListSoftDeleteIntegrationTest.java:54-57` 형태를 그대로 복사한다(이 파일을 직접 열어 확인했다 — master 에 존재한다): `@SpringBootTest` / `@AutoConfigureMockMvc` / `@WithMockUser(username = "admin", roles = {"ADMIN"})` 네 줄뿐이고 **`properties = {...}` 가 없다**. **`properties` 금지**(컨텍스트 캐시를 재사용해야 한다 — 이 저장소는 컨텍스트마다 `taskScheduler` 종료로 ~20초를 더 쓴다), **`@DirtiesContext` 금지**.
- `/api/v1/search` 는 인증만 요구하고 역할은 보지 않는다(`GlobalSearchController:40-43` javadoc). 세션 `User` 속성은 이 엔드포인트에 필요 없다(`ScreenPermissionService` 를 쓰지 않는다) — 다만 **미확인**이므로 401/403 이 나오면 `PivotViewIntegrationTest` 의 `sessionAttr("nexabuilder.user", User)` 형태를 더할 것.
- 픽스처는 **실제 `NexaListRepository`·`NexaFormRepository`·`DynamicEntityRepository`** 에 저장한다(목·손수 만든 대역 금지 — 운영자 지침). `active=true` 를 명시하고 이름에 `UUID` 8자 접미사를 붙여 검색어를 그 접미사로 준다(공유 H2 에 다른 테스트의 행이 섞여 있으므로 **절대 개수를 박지 말고** 특정 id 의 유무로 단언할 것). 정리(cleanup)는 이 저장소 관례상 하지 않는다.
- `deleted_at` 은 **직접 SQL UPDATE 가 아니라 프로덕션 경로**로 찍는다. 세 타입의 경로를 소스에서 확인했다(`grep setDeletedAt src/main` 히트는 이 세 곳뿐이다):
  - 목록 → `DELETE /api/v1/admin/lists/{listId}` (`UiBuilderController:56-66`, 200 `$.message == "trashed"`)
  - 폼 → `DELETE /api/v1/admin/forms/{formId}` (`UiBuilderController:89-99`)
  - 엔티티 → `DELETE /api/v1/admin/entities/{entityId}` (`AdminMetadataController:154-171`, 클래스 `@RequestMapping("/api/v1/admin")`). **주의**: `getSystem() == TRUE` 인 엔티티는 400 으로 거부되므로 픽스처 엔티티에 `system` 을 켜지 말 것. 없는 id 는 404 다.
  - 응답 봉투는 `ApiResponse.ok(String)` 오버로드라 `"trashed"` 가 `$.data` 가 아니라 **`$.message`** 에 온다(직전 회차가 `TrashController` 에서 실제로 밟은 함정이다).
- 문자열 `contains` 로 단언하지 말고 `jsonPath("$.data.lists[*].id")` 에 `hasItem(...)` / `not(hasItem(...))` 을 쓸 것.

## 5. 검증 명령 (이 저장소에서 실제로 도는 것)

```
sh ./gradlew --no-daemon test --tests 'com.nexabuilder.api.GlobalSearchSoftDeleteIntegrationTest'
sh ./gradlew --no-daemon clean test        # CI 와 같은 명령. 직전 회차들 기준 5~6분, 612~619건
sh ./gradlew --no-daemon bootJar -x test   # CI 와 같은 명령. 수 초, build/libs/nexabuilder-1.26.0.jar
```

`gradlew` 는 git 모드 100644 라 **`sh ./gradlew`** 로 실행해야 한다. 수정 **전에** 첫 명령을 돌려 빨간 출력을 확보할 것.

## 6. 위험과 피할 것

- **워크플로·빌드 파일 불간섭**: `.github/**`, `build.gradle.kts`, `settings.gradle.kts`, `gradle/**` 를 한 줄도 바꾸지 말 것. 커밋 후 `git diff --stat HEAD~1 -- .github/ build.gradle.kts settings.gradle.kts gradle/` 가 빈 출력이어야 한다.
- **열린 PR 과의 충돌 0 을 직접 확인했다**: `git log --all --not master --name-only -- src/main` 을 돌려 미머지 커밋이 건드린 프로덕션 파일 전부를 열거해 봤다 — `UiBuilderController`(#49), `TrashController`(#48), `RuntimeUiController`(#47), `ListInlineEditController`(#46), `ReportService`(#44·#45), `BulkListController`(#43), `CsvImportController`/`EntityService`/`CsvSchemaInferenceService`(#42), `NexaUiService`(#41), `DataAdapterService`(#40), `ListExportController`(#37·#38·#39), 그 밖에 `Jackson2JsonConfig`·`GlobalExceptionHandler`·템플릿 몇 개. **`GlobalSearchService.java` 와 `core/search/**` 는 히트 0** 이다. 단, 이 과제서가 참조만 하는 `UiBuilderController.deleteList` 는 #49 가 점유 중이므로 **그 파일을 수정하지 말 것**(테스트에서 HTTP 로 호출하는 것은 무해하다).
- `findByActiveTrue()` 를 다른 finder 로 갈아 끼우거나 리포지토리에 메서드를 추가하지 말 것(공유 계약).
- `filter`/`MatchableHit`/`cap` 상한(50)/`q.isEmpty()` 조기 반환/빈 결과 모양(7개 키)을 바꾸지 말 것 — 프런트(`src/main/resources/static/js/nexa-search.js`)가 이 봉투를 읽는다. **응답 봉투와 키 이름은 불변.**
- `typeHints`(`:202-208`)의 `@SuppressWarnings("unused")` 블록을 지우지 말 것(컴파일 경고용 장치다).
- 권한 검사(`ScreenPermissionService`)를 이 엔드포인트에 새로 넣지 말 것 — 별개 계약이고 보호 경로다.
- grep 결과를 증거로 제출하지 말 것. 반드시 **수정 전 빨간 테스트 출력**으로 고정할 것.

## 7. 차선 후보

1. **`README.md`·`agent.md`·`docs/deployment.md`·`docs/roadmap.md` 의 낡은 JDK17 / Spring Boot 3.4.4 / v1.2.x 표기를 Java21 / Boot 4.1.1 / v1.26.0 으로 정렬**(2/1/S) — 14회차 연속 차선 후보. `build.gradle.kts` 가 정본이고 열린 PR 과 충돌 0(위 스캔에 문서 파일 히트 없음). 대체된 옛 가이드는 남기지 말고 덮어쓸 것(정본이 둘이 되지 않게).
2. `MetadataVersionService.restoreList/restoreForm/restoreEntity`(`:334-346`)가 스냅샷을 그대로 `save` 해 **휴지통 이전 스냅샷으로 롤백하면 `deleted_at` 이 null 로 되살아난다**(2/3/S) — 다만 "버전 롤백은 명시적 의사표시" 라는 해석이 가능해 계약 결정이 선행이고, `snapshotCurrent` 가 `Map.of` 를 쓰는 것과도 얽혀 있다. 이번에 코드를 읽었을 뿐 실행으로 재현하지 않았다(**미확인**).
