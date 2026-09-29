- 과제: PR #37 (`auto/2026-09-29-2142` @ `0421f6d`) 의 CI `test + bootJar` 실패를 원인까지 파고들어 고친다 (가치 5 / 위험 2 / 작업량 S)
- 왜: 직전 회차의 all-hidden 내보내기 수정은 로컬에서 `cleanTest test` 615건 전부 통과했는데 GitHub Actions 의 `test + bootJar` 잡만 failure 로 끝나 PR #37 이 열린 채 막혀 있다. 이 로컬/CI 차이를 특정해 고치지 않으면 이번 회차도 같은 자리에서 멈춘다.

## 확인된 사실 (이번 정찰이 실제로 읽은 것)
- 실패한 검사: check-run `109418505248` "test + bootJar", `conclusion: failure`, `annotations_count: 2`. 같은 커밋의 `analyze (java)`(CodeQL)는 `skipped`. 출처: `/mnt/c/Users/USER/projects/aidev/state/runs/2026-09-29-214223-nexabuilder-improve/ci-0421f6d69b84.json`.
- 워크플로 실행 URL: https://github.com/hkjang/nexabuilder/actions/runs/36572142192/job/109418505248
- **잡이 39초 만에 끝났다** — `started_at 2026-09-29T13:01:41Z` → `completed_at 13:02:20Z`. 러너 `stages.json` 도 pr created 22:01:38 KST → ci failed 22:02:38 KST 로 같은 창을 가리킨다. 이 저장소의 전체 `test` 는 로컬 기준 5분36초이므로 **테스트 단계까지 가지도 못하고 죽었다**는 뜻이다. (주의: 같은 JSON 의 CodeQL 항목은 `completed_at` 이 `started_at` 보다 1초 빠르다 — 타임스탬프가 완벽히 신뢰할 만하지는 않다. 그래서 수용 기준 1이 로그 확인이다.)
- 실패 워크플로는 `release.yml` 이 **아니다**. `release.yml` 은 `on: push: tags: v*.*.*` 라 PR 에서는 돌지 않는다. 실패한 것은 `.github/workflows/ci.yml` 의 단일 잡 `test`(표시명 `test + bootJar`)다.
- `ci.yml` 단계 순서: `actions/checkout@v6` → `actions/setup-java@v5`(temurin 21, `cache: gradle`) → `chmod +x ./gradlew` → `./gradlew --no-daemon test` → `./gradlew --no-daemon bootJar -x test` → (실패 시) `actions/upload-artifact@v7` 로 `build/reports/tests/test/`, `build/test-results/test/` 업로드.
- `gradle/wrapper/gradle-wrapper.properties`: `gradle-8.14.3-bin.zip`, `networkTimeout=10000`, `validateDistributionUrl=true`. 저장소에 **`gradle.properties` 가 없다**(Gradle JVM 힙 미지정). `build.gradle.kts:146-158` 의 `maxHeapSize = "2g"` 는 test JVM 에만 적용된다.
- PR 의 diff 는 프로덕션 1 + 테스트 1, 총 2파일뿐이다: `ListExportController.java`(+238/-61 중 대부분), `ListExportIntegrationTest.java`. 프로덕션 변경은 `resolveColumns` 가 `List<ColumnSpec>` 대신 새 `private record ColumnPlan(List<ColumnSpec> visible, boolean configured)` 를 반환하고, 세 엔드포인트의 복제된 첫 행 키 fallback 을 `applyFirstRowFallback(plan, rs)` 헬퍼 하나로 합친 것이다. 워크플로·빌드 스크립트·의존성은 **건드리지 않았다**.
- 미확인: 실패 단계의 로그와 2건의 annotation 본문. 이번 정찰 환경에서는 `gh` 호출과 워크트리 `git checkout` 이 권한으로 막혀 **로컬 재현을 못 했다**. 아래 원인 후보는 전부 가설이며, 수용 기준 1을 먼저 끝내고 가설을 버릴 것.

## 원인 후보 (확률 높은 순, 전부 미검증 가설)
1. **컴파일 실패** — 39초와 `annotations_count: 2`(javac 오류 2건이면 annotation 2건)에 가장 잘 맞는다. 직전 회차는 `cleanTest test` 로 검증했는데 `cleanTest` 는 **테스트 결과만 지우고 컴파일 산출물은 지우지 않는다**. Lombok 애너테이션 처리 + Gradle 증분 컴파일이 남긴 낡은 클래스가 로컬에서만 통과시켰을 수 있다. → 로그에 `error:` 가 있으면 여기.
2. **Gradle 구성/의존성 해석 실패(네트워크)** — `networkTimeout=10000` 이라 `services.gradle.org` 또는 Maven Central 이 10초 안에 응답하지 않으면 초 단위로 죽는다. 전형적으로 annotation 은 "Process completed with exit code 1" 1건 + 경고 1건. → 이 경우면 **재실행으로 통과하는 플레이크**이므로 코드가 아니라 워크플로의 재시도/타임아웃 여유(느슨하게 만드는 것이 아니라 견고하게)를 다뤄야 한다. 판정 전에 반드시 로그로 확인할 것.
3. **`actions/setup-java@v5` 의 gradle 캐시 복원 실패 / 디스크 부족** — 보통 경고로 끝나므로 낮음.
4. **진짜 테스트 실패** — 39초 안에 615건이 돌 수 없으므로 가장 낮다. 다만 타임스탬프가 완전히 믿을 만하지는 않으니 로그에 `> Task :test FAILED` 와 클래스명이 찍혀 있으면 이쪽이다. 그 경우 CI 에만 있는 차이(기본 로케일·타임존·파일 인코딩)를 의심할 것 — 신규 CSV 단언은 BOM(`﻿`)과 헤더 문자열을 그대로 비교하고, `csvExportWithoutColumnsJsonStillFallsBackToFirstRowKeys` 는 헤더를 소문자로 정규화해 비교한다(H2 가 `ID,NAME` 을 주기 때문).

- 수용 기준:
  1) **실패 로그를 실제로 읽고 실패 단계와 예외/오류 문구를 과제 기록에 그대로 인용한다.** 다음 명령이 바로 쓸 수 있다 (앞의 두 개가 가장 빠름):
     - `gh api repos/hkjang/nexabuilder/check-runs/109418505248/annotations`
     - `gh run view 36572142192 --log-failed`
     - `gh run view --job 109418505248 --log`
     로그를 못 가져오면 2)의 clean 재현으로 대체하되, "로그 미확인" 을 명시할 것.
  2) **로컬에서 CI 와 같은 조건으로 재현한다**: `sh ./gradlew --no-daemon clean test`. `cleanTest` 가 아니라 **`clean`** 이어야 한다 — 후보 1을 배제하는 유일한 방법이고, 직전 회차가 하지 않은 것이 정확히 이것이다.
  3) 원인을 고친 뒤 `sh ./gradlew --no-daemon clean test` 와 `sh ./gradlew --no-daemon bootJar -x test` 가 통과한다. 테스트 건수는 615건 이상(0 fail / 0 skip)이어야 한다.
  4) 후보 2(플레이크)로 판명되면: 워크플로를 **완화하지 말고**(단계 삭제·`continue-on-error`·`--continue`·테스트 제외 금지) 재현 가능한 근거를 적고, 이번 회차 과제를 차선 후보로 전환할 것. 통과시키기 위한 완화는 즉시 반려 사유다.

- 건드릴 파일 (로그로 원인을 특정한 뒤 결정. 프로덕션 3개를 넘기지 말 것):
  - `src/main/java/com/nexabuilder/api/controller/ListExportController.java` — 후보 1이면 `resolveColumns`(≈:339) / `applyFirstRowFallback` / `record ColumnPlan`(파일 끝) 주변의 컴파일 오류.
  - `src/test/java/com/nexabuilder/api/ListExportIntegrationTest.java` — 후보 1/4면 신규 6건(`csvExportOfAllHiddenColumnsCarriesNoData`, `xlsxExportOfAllHiddenColumnsCarriesNoData` 등)의 컴파일·단언.
  - `.github/workflows/ci.yml` — **후보 2로 확정된 경우에만**, 그리고 완화가 아닌 방향으로만.

- 검증 명령 (이 저장소에서 실제로 도는 것. `gradlew` 는 모드 100644 라 `sh` 경유 필요):
  - `sh ./gradlew --no-daemon test --tests 'com.nexabuilder.api.ListExportIntegrationTest'` — 단일 클래스 19건, 약 20초.
  - `sh ./gradlew --no-daemon clean test` — 전체, 6분+ (clean 이라 직전 회차보다 오래 걸린다). **이번 과제의 핵심 명령.**
  - `sh ./gradlew --no-daemon bootJar -x test` — CI 패키징 단계, 수 초.

- 위험과 피할 것:
  - `.github/workflows/*` 는 보호 경로다. 워크플로를 느슨하게 해 통과시키는 것은 이번 과제에서 명시적으로 금지다. 운영자 지시: "릴리즈·빌드 경로를 건드리는 변경은 릴리즈까지 통과하는 것을 확인할 것."
  - **`applyFirstRowFallback` 자체를 지우지 말 것** — `columnsJson` 이 없는 목록(SQL 기반 다수)이 빈 파일이 된다. PR #37 의 회귀 가드 3건(`columnsJson` null · `[]` · 부분 hidden)이 그 계약을 잡고 있다.
  - `WorkbookUtil.createSafeSheetName`(XLSX 시트명, `ee7f786`) 과 `readQueryParams` 의 `MAX_FEED_ROWS` 주입(`ff96bd8`) 은 최근 2회차 머지분이다. 건드리지 말 것.
  - `ListExportIntegrationTest` 에 `@DirtiesContext` 를 추가하지 말 것 — 컨텍스트 재생성이 이 저장소에서 반복적으로 힙/종료 경합을 일으켰다.
  - 근본 원인을 모른 채 "혹시 몰라서" 여러 곳을 동시에 고치지 말 것. 원인 1개 → 수정 1개.

- 차선 후보: **SQL 기반 목록(`sqlId` 백엔드)의 `requireLiveList` 회귀 테스트** (2/1/S) — 후보 2(CI 플레이크)로 판명돼 프로덕션 수정이 성립하지 않을 때 고를 것. `DataAdapterService.queryList` 의 sqlId 분기에 `deleted_at` 이 찍힌 목록 사례가 없어 `GET /api/v1/data/lists/{id}` 와 `POST /api/v1/builder/lists/{id}/data` 의 400 계약이 고정돼 있지 않다. 기존 `SqlBackedListDefinitionIntegrationTest`(6건)의 `SqlMetadata`/`SqlExecutor` 픽스처를 재사용하면 프로덕션 0줄 · 테스트 파일 1개로 끝난다.
