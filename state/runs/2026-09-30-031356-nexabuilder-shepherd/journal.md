# PR 처리기 노트 2026-09-30-031356-nexabuilder-shepherd — nexabuilder PR #41
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
- **틀린 지적**: "CI 실패 = 이 변경의 결함". 잡이 3초 만에 끝나고 실패 로그가 비어 있어 스텝이 하나도 실행되지 않았습니다(결제 차단, 직전 회차의 인증된 `gh api` 증거와 일치). 이 워크트리에는 `gh` 인증이 없어 API 재확인은 못 했고, 잡 시간·빈 로그·로컬 전체 통과의 정황으로 판정했습니다.
- **맞은 지적**: 없음 — 비평가는 approve 했고 blocking 도 없었습니다. 그래서 **아무것도 고치지 않았고 커밋도 만들지 않았습니다**(규칙: 코드가 이미 맞으면 근거만 남긴다).
- **직접 재현한 것**: CI 두 명령을 cd6d8b9 그대로 — `cleanTest test` 613건/0 fail/0 error/0 skip(5m58s), `bootJar -x test` 성공(6s). 프로덕션 파일만 origin/master 로 되돌리면 `phoneMaskKeepsTheCarrierPrefix...() FAILED`, 복구 후 워킹트리 clean. 테스트는 실제로 바뀐 경로를 지납니다.
- **여전히 확신 없는 곳**: PostgreSQL(소문자 라벨+소문자 fieldName) 조합은 H2 라 재현 불가; 한 폼에 `phone`/`PHONE` 두 atom 이 같은 비멱등 정책으로 있으면 이중 마스킹이 남습니다(비평 노트 우려 2, 범위 밖). 필요한 조치는 결제 복구 후 CI **재실행**뿐입니다.
