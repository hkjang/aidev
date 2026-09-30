# PR 처리기 노트 2026-09-30-095942-nexabuilder-shepherd — nexabuilder PR #45
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-09-30-083215-nexabuilder-improve)
# 회차 노트 2026-09-30-083215-nexabuilder-improve — nexabuilder
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 08:32] base pinned — master@a3ca143
- [러너 08:32] autonomy release — 

## 정찰 노트
- 우선 과제(PR #44 CI 실패)는 9회차째 같은 서명이라 코드 원인이 없다 — 이번 베이스에서 `.github/`·`build.gradle.kts`·`gradle/` diff 가 빈 출력임을 직접 확인했고, 워크플로 완화는 금지라 실재 결함으로 회차를 채우도록 과제서를 썼다.
- 고른 것은 보류 1순위였던 `ReportService.prune` 의 파일명 정렬 결함이다 — 직전 회차 빨간 실행에서 실제 500 응답 본문까지 관측된 유일한 항목이라 추측이 섞이지 않는다. 엔티티 soft-delete 후보들은 이번에 읽어 보니 `EntityCrudController.resolve:129-140`·`LookupController:115·151` 이 이미 막고 있어 `rejected` 로 닫았고, 8회차 연속 차선인 문서 정렬은 가치 2 라 다시 차선으로 남겼다.
- 추측으로 적은 것(구현자가 확인할 것): ① `nexa.report.dir` 공유 디렉터리에 데코이 30개를 심으면 수정 전 두 테스트가 반드시 빨갛다고 적었지만 **실행해 확인하지는 않았다** — 검증 명령 1)로 먼저 빨간 것을 볼 것. ② `Files.setLastModifiedTime` 으로 과거 시각을 박는 것이 이 파일시스템(WSL)에서 유효하다고 가정했다.
- 조심할 것: PR #44 가 `ReportService.generate` 와 `ReportIntegrationTest` 를 점유하므로 `list()`/`prune()`/파일 끝 헬퍼로만 한정하고, `retain` 상향이나 `prune` 제거로 통과시키지 말 것(수용 기준 2가 가드). 새 테스트 클래스를 만들면 스프링 컨텍스트가 하나 늘어난다.
- [러너 08:38] scout done — 수정 과제 — PR #44 의 CI 실패를 증거로 종결(코드·워크플로 원인 아님)하고, `ReportService.prune` 이 **파일명 

## 구현 노트
- `ReportService.prune`/`list` 가 복제해 쓰던 파일명 역순 비교자를 수정 시각 내림차순 `NEWEST_FIRST` 로 교체했다 — 파일명이 `report-<listId>-<stamp>.csv` 라 정렬 키가 시각이 아니라 listId 였고, 알파벳순으로 더 높은 listId 의 리포트가 retain(30)개 있으면 방금 쓴 파일이 자기 자신을 지워 recipients 실행이 500 이 됐다. 프로덕션 1파일, 테스트 1파일(기존 클래스에 3건 추가).
- **확신 없는 곳**: (1) mtime 동시각 타이는 기존과 같은 파일명 역순으로 고정했지만, 초 단위 해상도 파일시스템에서 같은 초에 여러 listId 가 생성될 때 '어느 것을 지킬지' 는 여전히 listId 순이다 — 근본 해결은 파일명에 초 이상의 해상도를 넣는 별개 과제(ideas.json pending). (2) `list()` 의 정렬 변경이 관리 화면(`admin/report*.html`)의 기대 순서와 맞는지는 **HTTP/서비스 계약만 확인했고 화면을 띄워 보지는 않았다** — 컬럼이 `modifiedAt` 을 그대로 보여주므로 이제 표시값과 순서가 일치한다는 판단.
- **일부러 하지 않은 것**: `generate()` 불간섭(PR #44 가 첫 블록을 점유 — 텍스트 충돌 회피). 엔티티 `deletedAt` 누락도 같은 이유로 제외. `retain` 기본값·`prune` 호출·`STAMP`·`safe()`·`ReportController`·`EmailService` 는 손대지 않았다. `.github/**`·`build.gradle.kts`·`settings.gradle.kts`·`gradle/` 는 한 줄도 안 바꿨다(`git diff --stat` 로 빈 출력 확인) — CI 실패는 비공개 저장소 Actions 과금 차단이라 저장소 밖 원인이고, 이 세션 `gh` 는 미인증이라 잡 로그 본문은 미확인으로 남긴다.
- **다음 역할이 조심할 것**: 신규 3건은 실제 파일시스템(`nexa.report.dir` = `${java.io.tmpdir}/nexa-report-it`)에 쓴다. 이 디렉터리는 회차마다 **누적**되므로 데코이는 `try/finally` 로 치우지만, 앞선 회차가 남긴 실제 리포트가 많이 쌓여 있어도 테스트는 성립한다(데코이 `retain` 개가 수정 전 정렬에서 항상 상위를 차지하므로). `pruneStillCapsTheDirectoryAtRetain` 은 **수정 전에도 초록인 가드**다 — prune 제거식 "해결" 을 막으려고 넣었으니 지우지 말 것.
- [러너 08:49] brief accepted — 채택 — 지정한 파일 2개·`prune()`/`list()` 두 경로 동시 수정·`NEWEST_FIRST` + `modifiedMillis` 헬퍼 2개만 추가(공개 API 불변)·`reta
- [러너 08:49] verify passed — 검증 1개 통과 (auto)

## 비평 노트
- 확인한 것: `NEWEST_FIRST` 체이닝(`reversed()` 가 1차 키에만 적용, 파일명 타이브레이크 이중역전 없음, `Long.MIN_VALUE` 가 맨 뒤 = 먼저 삭제 — 주석과 일치), 수정 전 빨강의 논리적 성립(`zzzz`(0x7A) > `sample-list`(0x73) → 새 파일이 정확히 index 30 == retain), 원장 `- 실패 재현:` 출력이 이 증상과 일치함. HEAD 에서 `cleanTest test --tests ReportIntegrationTest` 를 직접 실행 → tests=6/failures=0 (17초). `git diff --stat` 2파일, `.github/**`·build 파일 무변경.
- 못 본 것: 전체 605건과 `bootJar`(범위가 report 2파일·공개 서명 불변이라 생략), `admin/report*.html` 실제 화면.
- 승인이어도 남는 우려 ①: 배포 후 첫 prune 이 구 코드가 지키던 집합과 **다른 집합**을 지우고 revert 로 복구되지 않는다(보존 개수 30 은 불변, CSV 는 재생성 가능 → 차단 아님). **릴리즈 노트에 넣을 것.** ②: `/api/v1/admin/reports` 응답 정렬이 파일명순→수정시각순으로 바뀐다(커밋 본문에 명시돼 있어 설명-동작 불일치는 아님).
- 다음 회차: 같은 밀리초 mtime 타이는 여전히 listId 가 승자를 결정한다. 근본 원인은 `STAMP=yyyyMMddHHmmss` 의 초 해상도이므로 stamp 해상도 상향을 별개 과제로(ideas.json pending).
- 보안/법무 차단 없음: 인가·비밀값·마이그레이션·의존성 무변경이고 `SecurityConfig:196` 의 `/api/v1/admin/**`→`hasRole("ADMIN")` 이 ReportController 를 덮는 것까지 확인. 참고(기존 사항): prune 은 개수 기준이라 연령 기준 보존기간을 집행하지 않음 — 목록 CSV 에 개인정보가 담길 수 있음.
- [러너 08:53] review approved — 리뷰 승인 (risk=low)
- [러너 08:53] pr created — https://github.com/hkjang/nexabuilder/pull/45
- [러너 08:53] ci failed — 성공이 아닌 검사: test + bootJar=failure

## 수리 노트
- 틀린 지적: CI `test + bootJar` 실패는 이 변경 탓이 아니다. 커밋 354f3c0 체크런 원본에서 3초 failure·`output` 전부 null·짝 잡 0초 skipped 를 다시 확인했고(잡이 스텝을 못 돌았다 = 과금 차단), HEAD 에서 CI 명령을 그대로 재현해 `test` 612건 전부 통과·`bootJar -x test` 성공을 얻었다. **고친 것 없음, 커밋 없음.**
- 맞았던 지적: 없음. 다만 전체 스위트 1·2회차에서 `SessionActivityIntegrationTest.listForReturnsNewestFirst` 1건이 빨갛게 나왔다 — 앞선 회차들이 "전체 초록" 으로만 적고 넘어간 부분이라 이번에 끝까지 파고들었다.
- 그 실패의 원인을 측정으로 확정했다: 이 WSL2 호스트 벽시계가 ~29초마다 ~1.1초 **뒤로** 당겨진다(sleep(1.1) 전후 초 해상도 스탬프 170쌍 중 6쌍 동일). `last_seen_at` 이 타이가 되고 `findByUserIdOrderByLastSeenAtDesc` 에 타이브레이크가 없어 순서가 임의가 된다. 코드 결함이 아니라 환경 + 테스트의 시계 의존.
- 인과 배제: `origin/master`(a3ca143) 별도 워크트리에서 같은 명령 2회 → 둘 다 초록(609건), 브랜치 3회차도 초록. 이 테스트 파일들은 브랜치에서 무변경이라 규칙대로 손대지 않았다.
- 여전히 확신 없는 곳: 1·2회 연속 실패는 3.5% 확률치고 잦다 — 시계 보정이 정확히 주기적이라 스위트 실행 시점과 위상이 맞았다고 본다(추정). GitHub 러너에는 이 시계 거동이 없으므로 CI 판정에는 영향이 없다고 보지만 CI 가 돌지 않아 확인은 못 했다. 다음 회차 과제 후보: `SessionActivityIntegrationTest` 의 초 해상도 시계 의존 제거.
