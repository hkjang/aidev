# 과제서 — 2026-09-30-093231-nexabuilder-improve

- 과제: 수정 과제 — PR #45 의 CI 실패를 증거로 종결(코드·워크플로 원인 아님)하고, 인라인 편집이 **값을 비우거나 빈 칸을 채울 때 감사 로그가 old/new 를 통째로 잃는** `ListInlineEditController` 의 `Map.of` NPE 를 고친다 (가치 3 / 위험 1 / 작업량 S)

## 왜
`ListInlineEditController.patchCell:129-133` 이 감사 `details` 를 `Map.of("listId",…, "column",…, "old", oldValue, "new", req.value())` 로 만든다. `Map.of` 는 **null 값에 NPE 를 던진다**(JDK 계약). `oldValue` 는 원래 비어 있던 셀에서 null 이고 `req.value()` 는 셀을 비우는 편집(`{"column":"name","value":null}`)에서 null 이다. 그래서 두 경우 모두 바로 아래 `catch (Exception e)` 가 잡아 `details` 가 축약 fallback `{"listId":"…","column":"…"}` 으로 떨어지고, **파괴적인 편집(값 삭제)일수록 감사 로그에 무엇이 지워졌는지가 남지 않는다**. 이 클래스의 javadoc(:34-35)은 "always carry the list id + column + old/new value" 라고 약속하므로 코드가 자기 계약을 어긴다. 고치면 값 삭제·최초 입력이 감사 뷰어에서 `'홍' → (비움)` 으로 보인다.

CI 쪽: 이번 브랜치 `auto/2026-09-30-0932` 는 `master@a3ca143` 와 diff 가 **없고**(이번 세션에서 `git diff --stat master...` 빈 출력 직접 확인), `git diff --stat master... -- .github/ build.gradle.kts settings.gradle.kts gradle/` 도 **빈 출력**이며 빌드·CI 설정을 건드린 최신 커밋은 2026-09-28 의 버전 범프 `c446d24` 다(`git log -1 -- .github/ …` 로 확인). 원인은 2026-09-30-010003 셰퍼드 회차가 인증된 `gh api .../jobs` 로 확보한 잡 주석 — "The job was not started because recent account payments have failed or your spending limit needs to be increased", `steps: []` — 즉 비공개 저장소 GitHub Actions 과금 차단이다. **고칠 코드가 없다.** 워크플로는 한 줄도 건드리지 말 것.

## 수용 기준
1. `PATCH /api/v1/data/list/{listId}/{key}` 에 `{"column":"name","value":null}` 을 보내면 200 이고, 행의 `name` 이 실제로 NULL 이 되며, 그 편집의 `nexa_audit.details` 가 JSON 으로 파싱되어 `listId`·`column` 뿐 아니라 **`new` 키를 담고 그 값이 null** 이고 `old` 가 `"before"` 다. (현재는 `details` 가 `{"listId":…,"column":…}` 뿐이라 `old`/`new` 키 자체가 없다.)
2. 원래 NULL 이던 셀을 채우는 편집도 `details` 에 `old`(null) 와 `new`("x") 를 남긴다.
3. 기존 정상 경로(둘 다 non-null)의 `details` 는 **모양이 바뀌지 않는다** — `listId`/`column`/`old`/`new` 네 키가 그대로다. 이 가드는 수정 전에도 초록이어야 하며, 그래야 "null 이 섞인 경로만 깨져 있었다" 가 실행으로 증명된다.
4. 기존 4건(`patchEditableCellUpdatesRowAndWritesAudit`, `patchNonEditableColumnReturns403`, `patchPrimaryKeyIsRejected`, `patchMalformedColumnNameIsRejected`)이 전부 그대로 초록.

## 건드릴 파일 (프로덕션 1개 + 테스트 1개)
- `src/main/java/com/nexabuilder/api/controller/ListInlineEditController.java:126-143` — `patchCell` 의 감사 블록. `Map.of(...)` 를 null 을 허용하는 `LinkedHashMap` 조립(이미 `java.util.LinkedHashMap` 이 :16 에 import 돼 있다)으로 바꾼다. 삽입 순서는 기존과 같이 `listId`, `column`, `old`, `new`. **`objectMapper.writeValueAsString` 호출과 바깥/안쪽 두 `catch`, fallback 문자열, `auditService.record(...)` 인자는 그대로 둔다** — 감사 실패가 업데이트를 되돌리면 안 된다는 기존 계약을 유지해야 한다.
- `src/test/java/com/nexabuilder/api/ListInlineEditIntegrationTest.java` — 기존 파일에 추가. **새 테스트 클래스를 만들지 말 것**(Spring 컨텍스트가 하나 늘고, 이 저장소는 이미 컨텍스트마다 `taskScheduler` 종료로 ~20초씩 쓴다). 기존 `seed()` 픽스처(`Fixture(entityId, tableName, listId, rowId)`, `name` 만 `editable:true`)를 그대로 재사용한다.
  - 감사 행 읽기는 기존 테스트가 이미 쓰는 `jdbc` 로: `SELECT details FROM nexa_audit WHERE entity_id = ? AND entity_key = ? AND action = 'INLINE_EDIT' ORDER BY audit_id DESC` 의 첫 행(`AuditLog.details` 는 `@Column(name="details")`, PK 는 `audit_id` — 엔티티에서 확인). `objectMapper.readValue(details, Map.class)` 로 파싱해 `containsKey("new")` / `get("old")` 를 단언하라. **문자열 `contains` 로 단언하지 말 것**(JSON 키 유무를 증명하지 못한다).
  - 값이 NULL 인지 확인할 때는 기존 테스트와 같이 `after.get("name")` 이 null 이면 `after.get("NAME")` 도 보는 대소문자 fallback 을 쓸 것. 다만 "둘 다 null" 은 "컬럼이 없음" 과 구분이 안 되므로, NULL 확인은 `jdbc.queryForObject("SELECT COUNT(*) FROM " + f.tableName() + " WHERE id = ? AND name IS NULL", Integer.class, f.rowId())` 로 하라.

## 검증 명령
```
sh ./gradlew --no-daemon test --tests 'com.nexabuilder.api.ListInlineEditIntegrationTest'
sh ./gradlew --no-daemon clean test
sh ./gradlew --no-daemon bootJar -x test
```
`gradlew` 는 모드 100644 라 `sh` 경유가 필요하다. 세 명령 모두 직전 회차들에서 실제로 돌았다(전체 `clean test` 약 6분, 612~615건). **이번 정찰 세션에서는 Gradle 실행 권한이 거부돼 미실행 — 구현자가 반드시 직접 돌릴 것.** 수정 전에 신규 테스트를 먼저 돌려 빨간 것을 확인하고(TDD), 그 출력(정확한 단언 실패 문구)을 회차 기록에 남길 것.

## 위험과 피할 것
- **워크플로 완화 금지**: `.github/workflows` 를 한 줄도 바꾸지 말 것. 재시도·`continue-on-error`·타임아웃 완화·재제출·cherry-pick 전부 금지. CI 는 과금 차단이라 로컬 재현으로만 증명한다. `gh` 는 이 환경에서 미인증이라 **잡 로그 본문은 미확인으로 남길 것** — 추정을 사실로 쓰지 말 것.
- **열린 PR 과의 충돌**: #37·#38·#39 = `ListExportController`, #40 = `DataAdapterService`, #41 = `NexaUiService`, #42 = `CsvImportController`/`EntityService`/`CsvSchemaInferenceService`, #43 = `BulkListController`, #44·#45 = `ReportService`. 이번 과제의 `ListInlineEditController` 는 **어느 것과도 겹치지 않는다**. 그래서 이 파일을 골랐다 — 다른 파일로 넓히지 말 것.
- **범위 밖(하지 말 것)**: ① :112 의 `req.column().toUpperCase()` 에 `Locale.ROOT` 를 더하는 것 — 옳지만 별개 회차다. ② 감사 `details` 에 값 원문을 담는 것 자체의 PII 문제(운영자 지침: 감사에는 식별자만) — 정책이 atom(formId)에 붙어 있어 계약 설계가 선행이고 `BulkListController:157` 과 같은 계열이다, 이번에 손대면 과제가 M 이상으로 커진다. ③ 이 엔드포인트에 `ScreenPermissionService` 권한 검사가 없는 것 — 보호 경로이고 별도 회차다. ④ `AuditService`·`AuditChainService`·`EntityService.update`·`isEditable` 불간섭.
- **`@DirtiesContext` 금지**(컨텍스트 재생성이 힙·종료 경합을 만든다). **목·손으로 만든 대역 금지** — 실제 H2·`SchemaDdlService`·`EntityService`·`MockMvc`·실제 `AuditService` 배선으로만 증명할 것(운영자 지침).
- 공유 H2 에 누적되므로 기존 관례대로 `UUID` 8자 접미사로 id·테이블명을 유일하게 만들 것(`seed()` 가 이미 그렇게 한다).
- `Map.of` → `LinkedHashMap` 이 유일한 실질 변경이어야 한다. 내부 `catch` 를 지우지 말 것 — Jackson 이 다른 이유로 던질 수 있고 fallback 은 여전히 필요하다.

## 차선 후보
`ReportService.generate:77-78` 의 `entities.findById(list.getEntityId())` 가 엔티티의 `deletedAt` 을 보지 않는다 — `NexaUiService.requireLiveEntity` 는 검사한다. 단, **PR #44·#45 가 같은 파일(`ReportService.java`)을 점유**하므로 충돌 위험이 있다. 그것도 막히면: README·`agent.md`·`docs/deployment.md`·`docs/roadmap.md` 의 낡은 JDK17/Boot3.4.4/v1.2.x 표기를 Java21/Boot4.1.1/v1.26.0 으로 정렬(문서만, 충돌 0, 9회차 연속 차선 후보). 대체된 옛 가이드는 남기지 말고 덮어쓸 것.
