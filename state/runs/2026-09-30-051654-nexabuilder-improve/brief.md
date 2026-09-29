- 과제: 수정 과제 — CI 실패를 증거로 종결(코드·워크플로 원인 아님)하고, CSV 로 자동 생성한 엔티티에 `/commit` 이 한 행도 넣지 못하는 결함을 고친다 (가치 4 / 위험 2 / 작업량 S)

## 0. 먼저 — CI 판정 (이 부분은 코드를 고치지 않는다)

**PR #41 의 CI 실패는 이 diff 의 결함이 아니다. 워크플로 파일도 건드리지 마라.**

이번 세션이 직접 읽은 증거(`/mnt/c/Users/USER/projects/aidev/state/runs/2026-09-30-024234-nexabuilder-improve/ci-cd6d8b98be9a.json`):
- `test + bootJar` (check-run 109547499168, head_sha `cd6d8b98be9a…`, PR #41): `started_at 2026-09-29T18:06:12Z` → `completed_at 18:06:15Z` = **3초** 만에 `failure`.
- 같은 커밋의 `analyze (java)` 는 `conclusion: "skipped"`, `annotations_count: 0`.
- `output.title/summary/text` 는 전부 `null` (러너가 로그 본문을 남기지 않는다).

3초는 `actions/checkout` 도 끝나지 않는 시간이다. 그리고 2026-09-30-010003 셰퍼드 회차가 **인증된** `gh api .../jobs` 로 확보한 잡 주석이 원인을 못 박아 두었다: *"The job was not started because recent account payments have failed or your spending limit needs to be increased"*, `steps` 는 **빈 배열**. 즉 GitHub Actions **과금 차단**이고 저장소 코드·워크플로와 무관하다. 같은 저장소의 성공 런은 470~611초, 앞선 실패들은 4초·7초·39초로 동일 서명이다.

이번 회차 브랜치(`auto/2026-09-30-0516` @ `a3ca143`)는 master 팁과 같고, 빌드·CI 설정을 건드린 최신 커밋은 2026-09-28 의 버전 범프 `c446d24` 다. 고칠 코드가 없다.

**너(구현자)가 할 일:**
1. `.github/workflows/*` 를 **한 줄도** 바꾸지 마라. 재시도·`continue-on-error`·타임아웃 완화·`if: always()` 전부 금지다.
2. CI 가 돌렸어야 할 두 명령을 로컬에서 그대로 실행해 통과를 기록해라 (아래 "검증 명령").
3. 회차 요약에 위 증거(3초 / analyze skipped / 인증된 과금 차단 주석)를 적고, 그 자리를 아래 실재 결함 수정으로 채워라.
4. `gh` 는 이 환경에서 미인증이다(`gh auth status` → not logged in) — 로그 본문을 다시 열려고 시간을 쓰지 마라. 6회차째 같은 결론이다.

## 1. 본 과제 — CSV 자동 엔티티의 `/commit` 이 모든 행을 떨어뜨린다

- 왜: `POST /api/v1/admin/import/infer-entity` 는 항상 `keyColumn = "id"` 를 돌려주고(`CsvSchemaInferenceService.infer:89`, `String pk = "id";`), `POST /api/v1/admin/import/create-from-csv` → `SchemaDdlService.ensureTable:78` 이 그 컬럼을 `id VARCHAR(255) NOT NULL` + `PRIMARY KEY (id)` 로 만든다. 그런데 뒤이어 도는 `POST /api/v1/admin/import/commit` 은 매핑된 CSV 컬럼만 담아 `EntityService.insert(tableName, data)` 를 부르고, `insert:59-90` 은 `reg_dttm` 만 채울 뿐 **키 값을 생성하지 않는다**. CSV 에 `id` 컬럼이 없으면(=일반적인 스프레드시트) 모든 행이 NOT NULL 위반으로 죽어 응답은 HTTP 200 + `inserted: 0` + 행별 error 만 돌아온다. "스프레드시트를 올리면 한 번에 동작하는 엔티티"라는 이 화면의 존재 이유가 마지막 한 걸음에서 무너진다.
- `CsvSchemaInferenceService` 의 javadoc 주석(86-88행: "The PK is a synthetic id column **the runtime fills on insert**")은 사실이 아니다. 이 문장도 같이 바로잡아라.
- 이 계약이 원래 어떻게 지켜지는지: 다른 호출자는 **자기가** 키를 넣는다 — `WorkflowPortalService:279` 의 `row.put(entity.getKeyColumn(), key)`, `LegacyApiController:78-79`, `NexaUiService:666-680`(키가 비면 `string.key` 아톰의 시퀀스로 생성하고, 아톰이 없으면 `Key value is required` 로 거절). 즉 **키 채우기는 호출자 책임**이고, CSV 임포트 경로만 그 책임을 빼먹었다. 그러니 `EntityService.insert` 를 고치는 게 아니라 CSV 임포트 경로에서 채워야 한다.

- 수용 기준:
  1. `POST /api/v1/admin/import/create-from-csv` 로 `keyColumn="id"` 엔티티를 만든 직후 `POST /api/v1/admin/import/commit` 에 CSV 데이터 행을 넘기면, 응답이 `inserted == 행 수`, `errors` 가 빈 배열이고, `SELECT COUNT(*) FROM <table>` 이 그 행 수와 같다. (현재는 `inserted: 0`, `errors` 에 행마다 NOT NULL 위반)
  2. 생성된 키가 실제로 각 행에 들어가 있고 행마다 서로 다르다 — `SELECT id FROM <table>` 의 값이 전부 non-null 이고 중복이 없다.
  3. **운영자가 CSV 컬럼을 키 컬럼에 매핑한 경우 그 값이 그대로 저장된다** — 자동 생성이 사용자 값을 덮어쓰지 않는다(`putIfAbsent` 의미). 이 케이스를 별도 테스트로 고정해라.
  4. 키 컬럼이 없는(=이미 다른 모양의) 엔티티로 `/commit` 할 때 기존 동작이 그대로다 — 즉 테이블에 그 컬럼이 없으면 아무것도 넣지 않는다. 회귀 가드로 남겨라.
  5. 신규 테스트 중 최소 1건이 **수정 전에 빨갛고** 최소 1건(기준 3 또는 4)이 **수정 전에도 초록**임을 실행 로그로 보여라. 빨간 로그의 근본 원인 문구(H2 의 NULL not allowed for column "ID" 계열)를 요약에 인용해라.

- 건드릴 파일 (프로덕션 **2개** 이내):
  - `src/main/java/com/nexabuilder/api/controller/CsvImportController.java:commit` — 행 루프에서 `data` 를 만든 뒤, 엔티티의 `keyColumn` 이 **비어 있을 때만** 생성한 키를 `putIfAbsent` 로 넣는다. 조건 셋을 전부 지켜라: (a) `entity.getKeyColumn()` 이 null/blank 가 아니고 `IDENT` 패턴을 통과하며, (b) 그 컬럼이 실제 테이블에 존재하고, (c) `data` 에 그 키(원문/대문자 어느 철자로도)가 아직 없을 때. 대소문자 판정은 `data.keySet()` 을 `equalsIgnoreCase` 로 훑어라 — `EntityService` 는 대문자 컬럼을 왕복시키므로 원문 비교만 하면 사용자 값을 덮어쓸 수 있다(수용 기준 3 이 여기서 깨진다).
  - 키 생성은 **새 방식을 발명하지 말고** 저장소 관례를 따라라: `UUID.randomUUID().toString().replace("-", "").substring(0, N)` (예: `AiAuditLogger:50`, `FileController:47`, `PasswordPolicyService:126`). 길이는 `VARCHAR(255)` 안이면 되고 32자 이하를 권한다. 접두사를 붙일지는 자유지만 붙인다면 한 곳에만 상수로 둬라.
  - 컬럼 존재 확인은 **이미 있는 것을 재사용**해라 — `EntityService:213` 의 `hasColumn(String tableName, String columnName)` 이 정확히 그 일을 하고 대소문자 세 철자(원문/대문자/소문자)를 이미 훑는다(`insert:61` 이 `reg_dttm` 판정에 쓰는 것). 다만 **`private` 이다**(213행). `public` 으로 바꾸는 것이 가장 작은 변경이고 `insert`/`update` 의 동작은 그대로다. 컨트롤러에서 `DatabaseMetaData` 를 새로 열지 마라 — 세 철자 처리를 다시 구현하면 계약이 갈라진다. (이 경우 프로덕션 파일은 3개가 된다 — 여전히 허용 범위다.) **`EntityService.insert` 자체의 동작은 바꾸지 마라** — 폼·워크플로·Groovy·레거시 API 가 전부 공유하는 계약이다(운영자 지침: 공유 계약으로 범위를 넓히지 말 것).
  - `src/main/java/com/nexabuilder/core/entity/CsvSchemaInferenceService.java:86-88` — 거짓이 된 주석("the runtime fills on insert")을 실제 동작으로 고쳐 쓴다. **`infer` 의 로직은 건드리지 마라**(기존 테스트 `inferenceDetectsCanonicalTypes` 가 `keyColumn == "id"` 를 고정하고 있다).
  - 테스트: `src/test/java/com/nexabuilder/api/CsvAutoEntityIntegrationTest.java` — 기존 4건(`inferenceDetectsCanonicalTypes`, `inferEntityEndpointReturnsTypedSchema`, `createFromCsvPersistsEntityAndMaterialisesTable`, `createFromCsvRejectsMalformedIdentifiers`)에 신규를 덧붙인다. 픽스처 형태를 그대로 따라라: `UUID` 8자 접미사로 `entityId`/`tableName` 을 만들고, `MockMvc` + 실제 H2 + 실제 `SchemaDdlService`·`EntityService` 로 HTTP 경로를 탄다. **목·대역·직접 주입 금지**, `@DirtiesContext` 금지(컨텍스트 재생성이 힙·종료 경합을 부른다). 클래스에 `@WithMockUser(roles="ADMIN")` 이 이미 걸려 있다.

- 검증 명령 (이 저장소에서 실제로 도는 형태. **이번 정찰 세션은 Bash 승인을 못 받아 직접 실행하지 못했다 — 미실행**. 다만 같은 형태를 직전 회차들이 성공적으로 사용했다):
  - 단일 클래스: `sh ./gradlew --no-daemon test --tests 'com.nexabuilder.api.CsvAutoEntityIntegrationTest'` (기존 4건 통과가 기준선, 직전 회차 기준 단일 클래스 ~25초)
  - CI 재현 1: `sh ./gradlew --no-daemon clean test` (전체 ~6분, 직전 회차 613건 / 실패0 / 오류0 / skip0)
  - CI 재현 2: `sh ./gradlew --no-daemon bootJar -x test` (~6초)
  - `gradlew` 는 git 모드 100644 라 반드시 `sh` 를 경유해라.

- 위험과 피할 것:
  - **열린 PR 과의 충돌**: #37·#38·#39 는 `ListExportController.java`, #40 은 `DataAdapterService.java`, #41 은 `NexaUiService.java` 를 바꾼 채 과금 차단으로 막혀 있다. 이번 과제의 파일(`CsvImportController`, `CsvSchemaInferenceService`, `CsvAutoEntityIntegrationTest`)은 **그 다섯과 전부 겹치지 않는다**. 이 세 파일 밖으로 나가지 마라. 재제출·cherry-pick 도 금지다.
  - `EntityService.insert`·`SchemaDdlService.ensureTable`·`CsvSchemaInferenceService.infer` 의 **동작**은 건드리지 않는다. 특히 `ensureTable` 은 이미 PK 와 필드의 중복을 정상 처리한다(`ensureTable:63-71,80,94` — CSV 에 `ID` 헤더가 있어도 컬럼이 두 번 생기지 않는다). 여기에 손대지 마라.
  - `/commit` 의 "매핑이 비면 스킵" 계약(`commit:275`)과 "매핑된 컬럼이 없음" 에러(`commit:280-283`)를 바꾸지 마라. 키를 자동으로 넣는 것 때문에 `data` 가 절대 비지 않게 되면 그 에러 분기가 죽는다 — **키는 매핑된 값이 하나라도 있을 때만 넣어라**(`data.isEmpty()` 판정 **뒤에** 넣는 것이 가장 안전하다).
  - 보호 경로 미접촉: `.github/workflows`, `core/auth`, `infra/security`, `resources/db/migration` 전부 이번 과제 밖이다.
  - 과거 교훈: 같은 값을 읽는 경로가 둘 이상일 때 한쪽만 고치면 안 된다. 여기서는 키 철자(원문/대문자) 두 가지가 그 자리다 — 수용 기준 3 이 그것을 잡는다.

- 차선 후보: **README·agent.md·docs/deployment.md·docs/roadmap.md 의 낡은 JDK17/Boot3.4.4/v1.2.x 표기를 현재 값(Java21 toolchain, Boot 4.1.1, version 1.26.0 — `build.gradle.kts` 기준)으로 정렬한다** (가치 2 / 위험 1 / 작업량 S). 문서만 건드려 열린 PR 5건과 충돌이 없다. 대체된 옛 가이드는 남기지 말고 갈아치워라(운영자 지침). 단, 위 주과제가 성립하지 않을 때만(= `/commit` 이 이미 키를 채운다는 것을 실행으로 확인했을 때만) 이쪽으로 가라.
