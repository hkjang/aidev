# PR 처리기 노트 2026-09-30-041605-nexabuilder-shepherd — nexabuilder PR #41
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-09-30-024234-nexabuilder-improve)
# 회차 노트 2026-09-30-024234-nexabuilder-improve — nexabuilder
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 02:42] base pinned — master@a3ca143
- [러너 02:42] autonomy release — 

## 구현 노트
- 무엇/왜: `NexaUiService.maskRecordFor` 가 atom `fieldName` 의 세 철자{원문·대문자·소문자} 중 겹치는 것을 각각 처리해 **같은 키를 두 번 마스킹**했다. 비멱등인 `phone` 이 `010-1234-5678`→`010-****-5678`→`****-5678` 로 접두사를 잃었다. `LinkedHashSet` 으로 서로 다른 키만 한 번씩 방문. 프로덕션 1파일(+import 1줄), 신규 테스트 1파일 4건. 커밋 cd6d8b9.
- CI: 재추정하지 않고 종결했다. 원인은 이미 **인증된** 증거로 확정 — 010003 회차의 `gh api .../jobs` 잡 주석 "job was not started because recent account payments have failed…", `steps`=[]. 과금 차단이라 저장소에 고칠 것이 없다. 워크플로 파일 불간섭. CI 의 두 명령 로컬 재현: `clean test` 613건/0 fail/0 error/0 skip(6분), `bootJar -x test` 성공(6초).
- **확신 없는 곳·검증 못 한 것**: (1) 이중 마스킹의 **가장 흔한 실제 트리거는 PostgreSQL**(소문자 라벨 + 소문자 fieldName)인데 테스트 DB 가 H2 라 그 조합은 돌려보지 못했다. H2 에서 같은 코드 경로에 닿으려고 **대문자 fieldName**(`MOBILE`)을 썼다 — 동일한 결함이지만 재현 구성이 실사용 중 덜 흔한 쪽일 수 있다. (2) 대문자 fieldName 을 가진 atom 이 실제 배포 데이터에 있는지 확인 못 했다(디자이너가 그렇게 저장하는 경로는 추적 안 함). (3) 고친 뒤에도 마스킹은 폼 레코드 GET 한 곳뿐 — 목록·어댑터·내보내기는 여전히 원문을 내보낸다(ideas.json 신규 4/3/M). 이번 범위 밖.
- 일부러 안 한 것: `PiiMaskingService.maskPhone` 을 멱등으로 만들기(마스킹된 값을 입력으로 받는 것을 정상화하면 진짜 원인을 덮는다). `ListExportController`·`DataAdapterService` 불간섭(PR #37·#38·#39·#40 충돌). 워크플로 완화·재시도(금지).
- 다음 역할 주의: 신규 테스트는 실제 H2 + `SchemaDdlService`/`EntityService`/`NexaFormRepository`/`NexaAtomRepository` + MockMvc 배선이 필요하다(`@DirtiesContext` 없음, 목 없음). 마스킹 여부는 Spring Security 주체가 아니라 **세션 속성 `nexabuilder.user`** 가 정한다 — 이 속성 없이 요청하면 `currentRoleId()` 가 null 을 반환해 마스킹이 걸린다. `gradlew` 는 모드 100644 라 `sh ./gradlew` 로 호출.
- [러너 02:57] verify passed — 검증 1개 통과 (auto)

## 비평 노트
- 확인함: `git checkout master -- NexaUiService.java` 로 수정 전 코드에 새 테스트를 돌려 `phoneMaskKeepsTheCarrierPrefixWhenTheFieldNameMatchesTwoKeyVariants() FAILED` 를 직접 재현했고(4건 중 1건), 수정본에서 4건 전부 통과, 전체 `cleanTest test` 도 5m55s BUILD SUCCESSFUL. 테스트는 실제로 바뀐 경로를 지납니다. **approve / risk low / blocking 없음.**
- 못 본 것: PostgreSQL 조합(소문자 라벨+소문자 fieldName)은 H2 라 재현 불가 — 대문자 fieldName 으로 같은 코드 경로만 짚었습니다. 대문자 fieldName atom 이 실제 배포 데이터에 있는지도 미확인(구현자 자백과 동일).
- 남는 우려 1 (릴리즈 노트): `phone` 출력이 `****-5678` → `010-****-5678` 로 **덜 가려집니다**. 문서화된 정책 모양과 일치하고 `010` 은 식별력이 없지만, 기존 화면 값이 바뀐 것으로 보입니다.
- 남는 우려 2 (경계 조건): 중복 제거가 atom 단위라 한 폼에 `phone`/`PHONE` 두 atom 이 같은 비멱등 정책으로 있으면 같은 키가 여전히 두 번 마스킹됩니다. 드문 구성, 테스트 없음. 레코드 단위로 처리한 키를 기억하는 것이 완전한 해법.
- 다음 회차 최우선: grep 전수 확인 결과 마스킹은 `UiBuilderController.java:242` **한 곳뿐** — 목록·어댑터·내보내기는 원문 PII 를 그대로 내보냅니다(기존 결함, ideas.json 4/3/M). 법무 관점에서 같은 개인정보가 접근 통제 없이 다른 경로로 나가는 상태입니다.
- [러너 03:05] review approved — 리뷰 승인 (risk=low)
- [러너 03:06] pr created — https://github.com/hkjang/nexabuilder/pull/41
- [러너 03:06] ci failed — 성공이 아닌 검사: test + bootJar=failure

## 수리 노트
- **지적(CI 실패)은 형식적으로 맞지만 원인 귀속이 틀렸습니다** — 코드 결함이 아니라 GitHub Actions 과금 차단입니다. 잡 `steps: 0`/3초, 체크런 주석 "The job was not started because recent account payments have failed…". 구현자가 이전 회차 증거로 내린 판단과 같고, 이번에 `gh api` 로 다시 직접 확인했습니다.
- 추가 증거: 09-29 13:01 이후 무관한 5개 커밋의 `ci` 가 전부 같은 주석으로 실패, 그 이전 a3ca143·ee7f786 은 성공. 0421f6d·a36eecc 주석 문구도 직접 대조했습니다. 계정 단위·시간 단위 차단이지 이 브랜치 문제가 아닙니다.
- **고친 것 없음 / 커밋 없음.** CI 두 명령 로컬 재현으로 브랜치가 실제로 green 임을 확인: `cleanTest test` BUILD SUCCESSFUL 5m59s, XML 집계 613건/실패0/오류0/skip0(신규 4건 포함), `bootJar -x test` BUILD SUCCESSFUL 6s + jar 생성. 트리 clean.
- 여전히 확신 없는 곳: 구현·비평 노트의 잔여 우려는 그대로 남습니다 — PostgreSQL(소문자 라벨+소문자 fieldName) 조합 미재현, 같은 폼에 `phone`/`PHONE` 두 atom 이 있으면 여전히 이중 마스킹(레코드 단위 방문 기록이 완전 해법), 마스킹은 `UiBuilderController:242` 한 곳뿐이라 목록·어댑터·내보내기는 원문 PII 노출.
- 다음 역할 주의: 이 체크는 저장소 안에서 통과시킬 수 없습니다. 소유자가 Billing & plans 를 정리하고 재실행해야 초록이 됩니다. 워크플로 완화·재시도는 금지라 손대지 않았습니다.
