# PR 처리기 노트 2026-09-30-085904-nexabuilder-shepherd — nexabuilder PR #44
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-09-30-072221-nexabuilder-improve)
# 회차 노트 2026-09-30-072221-nexabuilder-improve — nexabuilder
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 07:22] base pinned — master@a3ca143
- [러너 07:22] autonomy release — 

## 정찰 노트
- CI 는 `ci-eaf90787a70a.json` 을 직접 열어 3초 failure·`output` 전부 null·`analyze(java)` skipped 를 재확인했다(8회차 같은 서명, 원인은 인증된 과금 차단 증거로 이미 종결). 고칠 코드가 없어 회차를 채울 실재 결함을 찾았고, 직전 회차가 `BulkListController` 에서 고친 것과 같은 계열을 `listRepository.findById` 전수 grep 으로 훑어 `ReportService.generate:70` 을 찾았다. 리포트는 페이지 없이 전량(MAX_ROWS=1,000,000)을 뜨고 메일로 나가며 스케줄 잡도 같은 서비스를 쓰므로, 6회차 연속 차선이던 문서 버전 정렬(가치 2)보다 값이 높아 이것을 골랐다.
- 제친 후보: `RuntimeUiController` 8군데 동일 누락(껍데기 화면이고 데이터 피드는 이미 막혀 실유출이 없어 보임 + 8군데라 S 아님), 문서 버전 정렬(차선으로 남김), `xlsxToCsv`·`columnsJson` 계열(열린 PR 이 파일 점유).
- 추측으로 적은 것(미확인): ① `/api/v1/admin/**` 에서도 `IllegalArgumentException` 이 400 + `{success,message}` 로 접히는지 — 다른 경로에서는 테스트가 고정하지만 admin 경로에서 직접 실행하진 않았다. ② `recipients` 를 준 경로가 SMTP 없이 400 까지 도달하는지. 둘 다 실제 응답으로 맞추라고 과제서에 적었다.
- 구현자 주의: 테스트는 **기존 `ReportIntegrationTest` 에 추가**할 것 — 이 클래스는 `nexa.report.dir` properties 로 전용 컨텍스트를 쓰므로 새 파일을 만들면 컨텍스트가 하나 더 생겨 종료 지연이 늘어난다. 같은 함수 77-78 행의 엔티티 `deletedAt` 누락은 **이번 범위 밖**(ideas.json 에 분리해 뒀다).
- [러너 07:27] scout failed — 과제서 없음 — 구현자가 직접 고른다

## 구현 노트
- 무엇을 왜: `ReportService.generate` 가 목록의 `deletedAt` 을 안 봐서 휴지통 목록도 전체 행(MAX_ROWS=1,000,000) CSV 로 떠서 디스크에 남기고 `recipients` 가 있으면 메일로 조직 밖에 보냈다(스케줄 `action_type=report` 도 같은 서비스라 휴지통이 정지 수단이 아니었다). `orElseThrow` 직후·분기보다 앞에서 다른 다섯 판독 경로와 글자 단위로 같은 400 `"List not found: {listId}"` 를 던진다. 프로덕션 1파일, 테스트 1파일(기존 클래스에 3건 추가 — 새 컨텍스트 안 늘림). 커밋 a332829.
- **확신 없는 곳·검증 못 한 것**: (1) **CI 로그 본문은 못 봤다** — `gh` 미인증이라 이 세션에서 여는 것이 불가능하다. 종결 근거는 타이밍(3초 vs 정상 470~611초)과 앞선 셰퍼드 회차들의 **인용**이지 이 세션의 재현이 아니다. (2) 과제서 수용 기준 1 의 "리포트 파일이 생기지 않는다" 는 **증거로 쓰지 않았다** — 아래 prune 결함 때문에 결함이 있어도 통과하는 무의미한 단언이라 걷어내고 `generate` 가 던지는지를 서비스 계약으로 단언했다. (3) `generate` 를 고쳤으므로 스케줄 경로(`ScheduledJobManager:102`)도 같이 막힌다고 **추론**했을 뿐 스케줄러를 실제로 돌려 확인하지는 않았다. (4) 권한 없는 사용자 케이스는 없다 — 이 경로는 `/api/v1/admin/**` ADMIN 게이트뿐이라 400/403 구분 문제가 생기지 않는다.
- **이번에 새로 발견한 별개 결함(고치지 않음)**: `prune:170-180` 이 `retain` 개를 남길 때 파일명 역순으로 정렬하는데 파일명 정렬 키가 시각이 아니라 listId 라, 낮게 정렬되는 listId 의 리포트가 **쓰자마자 지워진다**. 빨간 실행에서 `recipients` 경로가 200 이 아니라 **500** 이었던 것이 이 때문이다(`Files.readAllBytes` → `NoSuchFileException`, 본문 `리포트 생성 실패: /tmp/nexa-report-it/report-list_rpt_...csv`). 이번 수정으로 두 경로 다 400 이 되어 증상은 가려지지만 결함은 남아 있다 — `ideas.json` 에 신규 pending(3/2/S)으로 올렸다.
- 일부러 하지 않은 것: `.github/workflows` 무수정(재시도·`continue-on-error`·타임아웃 완화 없음, `git diff --stat master... -- .github/ build.gradle.kts settings.gradle.kts gradle/` 빈 출력 확인). 같은 함수 77-78 행의 엔티티 `deletedAt` 누락은 `requireLiveEntity` 와 계약을 맞춰야 해 다음 회차로(ideas.json). `MAX_ROWS`·`ReportController` 예외 처리·`SqlExecutor`/`RecordSet` 계약 불간섭. 위 prune 결함도 범위 밖으로 뺐다.
- 다음 역할이 조심할 것: 새 테스트 3건은 H2 가 떠야 돌고 정리·`@DirtiesContext` 가 없다. `ReportIntegrationTest` 의 리포트 디렉터리 `/tmp/nexa-report-it` 는 **회차마다 누적**돼 이미 30개(retain 상한)가 차 있다 — 이 디렉터리에 의존하는 단언을 새로 쓰지 말고 `@TempDir` 로 격리할 것. 열린 PR 7건(#37~#43)과 파일이 겹치지 않는다.
- [러너 07:39] verify passed — 검증 1개 통과 (auto)

## 비평 노트
- 가드(`ReportService.java:82-84`)를 임시 제거해 직접 재현했다: 신규 2건이 실패(`tests=6 failures=2`), 되돌리면 6건 전부 초록. 원장의 `- 실패 재현:` 줄과 증상이 일치하고 워크트리는 `git status --porcelain` 빈 출력으로 원복 확인. 판정문·메시지는 `requireLiveList` 외 다섯 판독자와 글자 단위로 동일(`getDeletedAt() != null`, hasText 쓰는 판독자 없음).
- 구현자가 미검증으로 남긴 것 중 (1) admin 경로 400 봉투와 (3) 스케줄 경로를 읽어 확인했다 — `generate`/`generateAndEmail` 의 프로덕션 호출자는 `ReportController:40-41` 과 `ScheduledJobManager:106` 뿐이라 우회 경로가 없고, 후자는 `Exception` 을 잡아 `last_status=error` 로 기록하므로 스케줄러가 깨지지 않는다. 스케줄러 실제 실행과 전체 테스트 스위트는 나도 돌리지 않았다.
- 승인이어도 남는 것: ① `ReportService.java:88-90` 의 **엔티티** 휴지통 미판정(같은 계열 유출, 다음 회차) ② `prune` 의 listId 정렬 결함 — 이번 수정으로 recipients 의 500 증상이 가려지니 잊지 말 것 ③ 휴지통 이전에 생성된 CSV 는 남지만 다운로드 라우트가 없어 공격 경로는 아님(보존 정책만 과제).
- 보안·법무 차단 없음: 새 엔드포인트·권한 확대·신규 개인정보 수집 없음, 마이그레이션/워크플로 무수정, 순수 가드라 revert 로 완전 복귀. 판정 approve / risk low.
- [러너 07:44] review approved — 리뷰 승인 (risk=low)
- [러너 07:44] pr created — https://github.com/hkjang/nexabuilder/pull/44
- [러너 07:44] ci failed — 성공이 아닌 검사: test + bootJar=failure
