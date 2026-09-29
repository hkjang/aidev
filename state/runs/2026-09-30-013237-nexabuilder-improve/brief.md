- 과제: 수정 과제 — PR #39 의 CI 실패를 **증거로** 코드 원인에서 배제하고(워크플로 수정 금지), 그 자리를 `DataAdapterService` 의 실재 결함으로 채운다: 그리드가 보내는 알 수 없는 파라미터·죽은 컬럼 이름이 SQL 식별자로 그대로 들어가 어댑터 URL 이 파싱 불가능한 400 으로 죽는다 (가치 4 / 위험 2 / 작업량 S)

- 왜: PR #39 의 `test + bootJar` 는 `started_at 2026-09-29T15:54:23Z` → `completed_at 15:54:30Z`, **7초** 만에 failure 다(`state/runs/2026-09-29-003529-nexabuilder-improve/ci-297a248d3235.json:153-163`). 같은 저장소의 성공 22건은 470~611초이고, 앞선 두 실패는 39초·4초였다 — 7초는 `actions/checkout` 도 못 끝내는 시간이라 자바 코드 변경이 죽일 수 있는 구간이 아니다. 더 결정적으로, **이번 세 실패 브랜치는 전부 `master@a3ca143` 기반이고 `build.gradle.kts`·`gradle/`·`.github/` 를 한 줄도 건드리지 않았으며**(`git log -3 -- .github/workflows/ build.gradle.kts settings.gradle.kts gradle/` → 최신이 2026-09-28 의 버전 범프 `c446d24`), 그 직전 `ee7f786`(같은 파일 `ListExportController` 를 고친 PR #36)은 **CI 를 통과해 머지됐다**. 즉 고칠 코드가 없다. 그래서 회차를 비우지 않기 위해 다른 파일의 실재 결함을 함께 싣는다: `DataAdapterService.parse` 가 예약어가 아닌 모든 쿼리 파라미터를 컬럼 필터로 승격시키고(`:296-304`), `executeEntityBacked` 가 그것과 `sortdatafield` 를 `sanitize()` 만 거쳐 `WHERE`/`ORDER BY` 에 문자열로 끼워 넣는다(`:176-202`). 존재하지 않는 컬럼 이름은 `sanitize` 의 정규식(`^[a-zA-Z_][a-zA-Z0-9_]*$`)을 통과하므로 `BadSqlGrammarException` → `{success,message}` 400 이 되고, 이 URL 을 읽는 jqxgrid 어댑터는 `{Rows,totalCount}` 만 파싱할 수 있어 그리드가 통째로 죽는다. 같은 파일 `:264-271` 의 주석이 이미 "이 URL 은 실패 봉투가 달라서 새 400 경로를 만들면 안 되고, 범위를 벗어난 페이징 파라미터는 **거부가 아니라 클램프**한다" 는 설계를 못 박아 뒀다 — 필터·정렬만 그 규칙에서 빠져 있다.

- 수용 기준:
  1) **CI 판정(코드 아님)**: 워크플로 파일을 한 줄도 바꾸지 않은 채, `sh ./gradlew --no-daemon clean test` 와 `sh ./gradlew --no-daemon bootJar -x test` 가 이 브랜치에서 통과하는 것을 실행 로그로 남긴다. CI 로그 본문은 이 세션의 권한으로 볼 수 없다(`gh` 미인증 + 비공개 저장소) — 못 본 것은 "미확인" 으로 적을 것. **워크플로를 완화하거나 재시도/`continue-on-error` 를 넣는 변경은 금지.** 회차 노트에 위의 시간·베이스 근거를 그대로 적는다.
  2) 엔티티 기반 목록에 대해 `GET /api/v1/data/lists/{listId}?groupscount=0` 이 **200 + `{Rows,totalCount}`** 를 돌려준다(현재는 400). 알 수 없는 파라미터는 필터로 승격되지 않고 **조용히 무시**된다 — `totalCount` 는 파라미터가 없을 때와 같아야 한다.
  3) 같은 목록에 대해 `?sortdatafield=no_such_column&sortorder=asc` 가 **200** 을 돌려주고 행 순서는 정렬 없는 응답과 같다(현재는 `ORDER BY no_such_column` → 400). 리스트 정의의 `columnsJson` 에 남은 죽은 컬럼을 헤더 클릭했을 때의 실제 경로다.
  4) **회귀 가드(반드시 유지)**: 실존 컬럼으로 거는 필터(`?name=홍길동`, `s_name=…`)와 실존 컬럼 정렬(`sortdatafield=name&sortorder=desc`)은 **수정 전과 동일하게** 동작한다. 최소 1건은 수정 전에도 초록이어야 하고, 그 사실을 회차 노트에 적는다.
  5) 테스트는 목·대역 없이 실제 H2 + `SchemaDdlService` + `EntityService` + `NexaListRepository` + MockMvc 배선으로 짠다(운영자 지침: 손으로 만든 대역으로 결함을 증명하지 말 것). 수정 전에 신규 테스트를 단독 실행해 **빨간 것을 먼저 확인**하고, 그 출력(상태 코드와 `Column "..." not found` 원인)을 회차 노트의 "실패 재현" 에 붙인다.

- 건드릴 파일 (프로덕션 1개):
  - `src/main/java/com/nexabuilder/core/data/DataAdapterService.java:170 executeEntityBacked` — `req.filters` 를 `WHERE` 로 만들기 전에, 그리고 `req.sortField` 를 `ORDER BY` 에 쓰기 전에 **그 테이블에 실재하는 컬럼인지 확인하고 아니면 버린다**. 컬럼 집합은 같은 파일 `entityDataFields:116-140` 이 이미 쓰는 `SELECT * FROM <t> WHERE 1=0` + `PreparedStatement.getMetaData()` 프로브(`getColumnLabel(i).toLowerCase(Locale.ROOT)`)를 **private 헬퍼로 뽑아 재사용**할 것 — 새 메타데이터 조회 방식을 발명하지 말고, 드라이버가 대문자 라벨을 주는 문제도 그 소문자 정규화가 이미 흡수한다(비교는 양쪽 `toLowerCase(Locale.ROOT)`).
  - 프로브는 검사할 대상이 있을 때만(필터가 비어 있지 않거나 `sortField` 가 있을 때만) 돌려 요청당 추가 쿼리를 최소화한다. 프로브가 예외를 던지면(권한·드라이버) **기존 동작 그대로** 통과시켜 이번 변경이 새 실패를 만들지 않게 한다.
  - `sanitize()` 는 그대로 둘 것 — 여전히 SQL 인젝션 방어선이고, 이번 변경은 그 앞단에서 "존재하지 않는 컬럼" 만 걸러낸다.
  - `executeSqlBacked:144` 와 `RESERVED_PARAMS:52-55` 는 건드리지 않는다. SQL 기반 목록은 여분 파라미터가 명명 파라미터로 들어가 무해함이 2026-09-27 회차에 실행으로 확인됐고(`sqlBackedListAcceptsProtocolKeys`), `applySort` 는 모르는 필드에 이미 no-op 다. 예약어 목록에 키를 하나씩 더하는 방식은 다음 키에서 또 깨지므로 채택하지 말 것.
  - `src/test/java/com/nexabuilder/api/DataAdapterIntegrationTest.java` — 신규 4건(수용 기준 2·3·4)을 기존 픽스처(UUID8 접미사, `seed()` 패턴) 위에 붙인다. `@DirtiesContext` 금지(컨텍스트 재생성이 테스트 JVM 종료를 더 늘린다).

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - 단독: `sh ./gradlew --no-daemon test --tests 'com.nexabuilder.api.DataAdapterIntegrationTest'`
  - 전체: `sh ./gradlew --no-daemon clean test` (직전 회차 기준 612~618건, 약 6분, 0 skip 이어야 함)
  - 패키징: `sh ./gradlew --no-daemon bootJar -x test` (CI 와 같은 명령, 수 초)
  - `gradlew` 는 모드 100644 라 반드시 `sh ./gradlew` 로 호출한다.

- 위험과 피할 것:
  - **`.github/workflows/*` 는 한 줄도 바꾸지 말 것.** 완화 금지는 명시 규칙이고, 워크플로 강화(wrapper-validation 추가 등)도 사람이 지켜보는 회차 몫이다.
  - **`ListExportController.java` 는 이번 회차에 열지 말 것.** 열려 있는 PR #37·#38·#39 가 전부 이 파일을 바꾸고 있어 충돌한다. 그래서 이번 과제를 다른 파일로 골랐다. 같은 이유로 PR #37/#38/#39 의 cherry-pick 재제출도 하지 말 것(세 번 반복된 접근).
  - `core/sql`(`SqlExecutor`·`RecordSet`)·`core/entity` 의 공유 계약, `core/auth`·`permission`, `resources/db/migration` 은 범위 밖.
  - 알 수 없는 파라미터를 **400 으로 거부하는 방향**으로 고치지 말 것 — 이 URL 의 실패 봉투는 `{success,message}` 라 jqxgrid 가 못 읽는다(같은 파일 `:264-271` 주석의 확정된 설계).
  - 미확인으로 남는 것(추측으로 적지 말 것): jqxGrid 23.x 가 그룹핑 없이도 `groupscount` 를 실제로 보내는지는 **확인하지 못했다**(`runtime/list.html` 에 `groupable` 설정은 없음). 수용 기준 2 는 "알 수 없는 파라미터 일반" 의 계약을 고정하는 것으로 충분하며, 결함의 현실적 트리거는 수용 기준 3(정의에 남은 죽은 컬럼으로 정렬)이다. 브라우저가 보내는 파라미터 목록을 근거로 적지 말 것.
  - CI 가 이번에도 실패할 가능성이 높다(원인이 저장소 밖). 실패해도 **로컬 검증 결과와 위 시간 근거를 회차 노트에 정확히 적고**, CI 를 통과시키려고 워크플로나 테스트를 손대지 말 것.

- 차선 후보: README(JDK17/Boot3.4.4 배지)·`agent.md`(Windows 고정 경로)·`docs/deployment.md`(JDK17 Docker 예시) 의 낡은 표기를 `build.gradle.kts`·CI 에서 확인한 Java21/Boot4.1.1 로 정렬 (2/1/S). 문서만 건드려 충돌이 없고, 옛 가이드는 남기지 말고 대체할 것.
