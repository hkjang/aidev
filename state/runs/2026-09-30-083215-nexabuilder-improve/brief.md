- 과제: 수정 과제 — PR #44 의 CI 실패를 증거로 종결(코드·워크플로 원인 아님)하고, `ReportService.prune` 이 **파일명 역순**으로 정리해 방금 쓴 리포트를 즉시 지우고 메일 발송을 500 으로 죽이는 결함을 고친다 (가치 3 / 위험 2 / 작업량 S)
- 왜: `prune(Path)` 이 `Comparator.comparing(파일명).reversed()` 로 정렬해 앞 `retain`(기본 30)개만 남기는데, 파일명이 `report-<safe(listId)>-<yyyyMMddHHmmss>.csv` 라 정렬 키가 시각이 아니라 **listId** 다 — 알파벳순으로 낮은 listId 의 리포트는 디렉터리에 더 높게 정렬되는 파일이 30개만 있어도 **쓰는 즉시 자기 자신이 지워진다**. 이러면 `/api/v1/admin/reports` 목록에 안 나오고, `recipients` 를 준 실행은 `generateAndEmail:98` 의 `Files.readAllBytes` 가 `NoSuchFileException` → `ReportController.run` 의 `catch (IOException)` 에서 **500 `리포트 생성 실패: <경로>`** 가 되어 스냅샷 메일이 조용히 안 나간다(직전 회차의 빨간 실행에서 실제로 관측된 응답 본문). 정렬 키를 수정 시각으로 바꾸면 리포트는 보존되고 정리는 그대로 동작한다.

- 수용 기준:
  1) 디렉터리에 **더 높게 정렬되지만 더 오래된** 파일이 `retain` 개 있는 상태에서 `reportService.generate("sample-list")` 를 호출하면 반환된 `ReportResult.path()` 의 파일이 **실제로 존재한다**(수정 전에는 존재하지 않는다).
  2) 정리 기능은 죽지 않았다 — 같은 실행 뒤 디렉터리의 `report-*.csv` 개수가 `retain` 이하다(이 단언은 수정 전에도 초록이어야 한다 = prune 을 지우는 식의 "해결" 을 막는 가드).
  3) 같은 상태에서 `POST /api/v1/admin/reports/run` 에 `recipients` 를 주면 **200 + `$.success=true`** 다(수정 전에는 500 `리포트 생성 실패: …`). SMTP 가 없어도 200 이어야 한다 — `EmailService.sendWithAttachment:109-116` 은 `cfg.host` 가 비면 네트워크를 타지 않고 `false` 를 돌려주고, 설정돼 있어도 예외는 `generateAndEmail` 의 `catch (Exception)` 이 `log.warn` 으로 접는다.
  4) `.github/workflows/**` 와 `build.gradle.kts`·`settings.gradle.kts`·`gradle/` 는 **한 줄도 바뀌지 않는다**(`git diff --stat master... -- .github/ build.gradle.kts settings.gradle.kts gradle/` 가 빈 출력).

- 건드릴 파일 (프로덕션 1개 + 테스트 1개):
  - `src/main/java/com/nexabuilder/core/report/ReportService.java`
    - `prune(Path dir)` (파일 끝, 약 163-178행) — `.sorted(Comparator.comparing((Path p) -> p.getFileName().toString()).reversed())` 를 **수정 시각 내림차순** 비교자로 교체. `for (int i = retain; i < files.size(); i++)` 루프와 `retain <= 0` 조기 반환은 그대로.
    - `list()` (약 111-133행) — **같은 비교자**를 쓰도록 교체. 같은 잘못된 정렬 키가 두 곳에 복제돼 있고(운영자 지침: 같은 값을 읽는 경로를 한쪽만 고치지 말 것), 관리 화면은 이 순서를 '최신순' 으로 읽는다. `ReportFile` 레코드 모양과 `list()` 의 필터·예외 처리는 그대로.
    - 새 private 헬퍼 2개만 추가(공개 API 변경 금지):
      ```java
      private static long modifiedMillis(Path p) {
          try { return Files.getLastModifiedTime(p).toMillis(); } catch (IOException e) { return Long.MIN_VALUE; }
      }
      private static final Comparator<Path> NEWEST_FIRST =
              Comparator.comparingLong(ReportService::modifiedMillis).reversed()
                      .thenComparing(Comparator.comparing((Path p) -> p.getFileName().toString()).reversed());
      ```
      `Comparator`·`Files`·`Path`·`IOException` 은 이미 import 돼 있다. mtime 을 못 읽는 파일은 `Long.MIN_VALUE` 로 맨 뒤(= 먼저 정리)로 간다 — 의도이므로 한 줄 주석으로 남길 것. 초 단위 해상도 파일시스템에서 mtime 이 같을 때의 순서는 기존과 같은 파일명 역순으로 고정된다.
  - `src/test/java/com/nexabuilder/api/ReportIntegrationTest.java` — **기존 클래스에 추가**(신규 클래스 금지: 이 클래스는 `@SpringBootTest(properties = "nexa.report.dir=${java.io.tmpdir}/nexa-report-it")` 로 전용 컨텍스트를 이미 쓰므로 새 파일은 스프링 컨텍스트를 하나 늘린다). 기존 3건(`generatesCsvForSampleListAndLists`, `serviceWritesNonEmptyCsv`, `rejectsMissingListId`)은 건드리지 말 것.
    - `@Value("${nexa.report.dir}") private String reportDir;` 와 `@Value("${nexa.report.retain:30}") private int retain;` 를 필드로 주입.
    - 공용 픽스처 헬퍼: `private List<Path> seedHigherSortingOlderReports(Path dir, int n)` — `report-zzzz<i>-20260101000000.csv` 이름으로 실제 파일을 `Files.write` 하고(`report-` 시작 + `.csv` 끝이어야 `prune` 의 필터에 걸린다), 각각 `Files.setLastModifiedTime(p, FileTime.from(Instant.now().minusSeconds(3600 + i)))` 로 **과거 시각**을 박는다. `"zzzz"` 는 `"sample-list"` 보다 문자열이 크므로 수정 전 정렬에서 전부 새 파일보다 위에 온다. `n = retain` 으로 부르면 디렉터리에 이미 쌓인 파일과 무관하게 새 파일의 인덱스가 `retain` 이상임이 보장된다.
    - 테스트 3건(이름은 그대로 쓸 것): `freshReportSurvivesPruneWhenOlderFilesSortHigher()`(수용 기준 1), `pruneStillCapsTheDirectoryAtRetain()`(수용 기준 2 — 수정 전에도 초록), `reportRunWithRecipientsSucceedsAfterPrune()`(수용 기준 3, MockMvc 로 실제 HTTP 경로).
    - 각 테스트는 `try/finally` 로 심은 데코이를 `Files.deleteIfExists` 로 치울 것 — `nexa.report.dir` 는 회차마다 **누적되는 공유 디렉터리**다(그래서 실제 운영에서 이 버그가 터졌다).

- 검증 명령 (이 저장소에서 실제로 도는 것):
  1) 수정 전에 빨간 것 확인: `sh ./gradlew --no-daemon test --tests 'com.nexabuilder.api.ReportIntegrationTest'` → 6건 중 신규 2건(`freshReportSurvives…`, `reportRunWithRecipients…`)이 실패해야 한다. 실패 문구를 회차 기록에 그대로 남길 것.
  2) 수정 후 같은 명령으로 6건 전부 초록.
  3) CI 와 같은 두 명령: `sh ./gradlew --no-daemon clean test` (약 6분, 직전 회차 612~615건 / 실패0) 와 `sh ./gradlew --no-daemon bootJar -x test` (약 6초).
  - `gradlew` 는 모드 100644 라 반드시 `sh ./gradlew` 로 실행.

- 위험과 피할 것:
  - **CI 는 저장소 밖 원인이다 — 워크플로를 완화하지 말 것.** 9회차째 같은 서명이다: 성공 런 22건은 `test + bootJar` 가 470~611초인데 실패 런은 3·4·7·9·39초로 `actions/checkout` 도 끝나지 않는 시간이고, `output.title/summary/text` 는 전부 null, 같은 커밋의 `analyze (java)` 는 skipped 다. 원인은 2026-09-30-010003 셰퍼드 회차가 **인증된** `gh api .../jobs` 로 확보한 주석("The job was not started because recent account payments have failed or your spending limit needs to be increased", `steps: []`)과 2026-09-30-065545 회차가 직접 측정한 공개/비공개 상관(같은 계정·같은 시간대에 PUBLIC 6곳은 분 단위 success, PRIVATE 2곳만 3~9초 failure)으로 증명돼 있다. **이 베이스(`master@a3ca143`)에서 `git diff --stat master... -- .github/ build.gradle.kts settings.gradle.kts gradle/` 가 빈 출력임을 이번 정찰이 직접 확인했다.** 이 세션 환경의 `gh` 는 미인증이라 **잡 로그 본문은 미확인으로 남길 것** — 재추정에 시간을 쓰지 말고 위 증거를 인용해 종결하고, 로컬에서 위 검증 명령 3)으로 통과를 재현하라. 재시도·`continue-on-error`·타임아웃 완화·PR 재제출·cherry-pick 전부 금지.
  - **`retain` 기본값 상향·`prune` 호출 제거·`nexa.report.retain` 설정 완화로 "해결" 하지 말 것.** 수용 기준 2가 그 경로를 막는다.
  - `generate()` 는 건드리지 말 것 — **PR #44 가 같은 파일의 `generate()` 첫 블록(목록 soft-delete 게이트)을 점유**하고 있다. 이번 변경은 `list()`/`prune()`/파일 끝 헬퍼로 한정해 텍스트 충돌을 피하라. `ReportIntegrationTest` 도 PR #44 가 3건을 더한 상태이므로 클래스 끝에 덧붙일 것. 두 변경이 만나면 **둘 다 살리는 것이 정답**이다.
  - `MAX_ROWS`, `toCsv`/`csvCell`/`safe`, `ReportController`, `ScheduledJobManager`, `EmailService`, `NexaListRepository` 는 건드리지 말 것.
  - `@DirtiesContext` 금지(프로필의 반복 실패 지점), 목·손수 만든 대역 금지 — 실제 파일시스템·실제 `ReportService` 빈·MockMvc 로만 증명할 것.
  - 이번 정찰이 확인한 것(같은 계열이라 헷갈리기 쉬움): **`EntityCrudController.resolve:129-140` 과 `LookupController.entitySearch:115`/`entityResolve:151` 은 이미 엔티티 `deletedAt` 을 막고 있다** — 여기에 손대지 말 것. `ReportService.generate:77` 의 엔티티 `deletedAt` 누락은 여전히 남아 있으나 PR #44 의 hunk 와 붙어 있어 이번 범위에서 제외한다.

- 차선 후보: README·`agent.md`·`docs/deployment.md`·`docs/roadmap.md` 의 낡은 JDK17/Boot3.4.4/v1.2.x 표기를 `build.gradle.kts` 기준(Java21 toolchain / Spring Boot 4.1.1 / v1.26.0)으로 정렬 — 8회차 연속 차선으로 남은 항목이고 열린 PR 8건과 파일이 전혀 겹치지 않는다. 대체된 옛 가이드는 남기지 말고 덮어쓸 것(운영자 지침). 문서만 바꾸므로 검증은 `sh ./gradlew --no-daemon bootJar -x test` 로 충분하다.
