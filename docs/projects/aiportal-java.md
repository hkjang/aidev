---
title: "aiportal-java — 자율 개선 이력"
description: "aiportal-java: 자율 개선 회차 10회, 릴리즈 0건. 최근 릴리즈 없음."
last_modified_at: 2026-09-06 00:17:21 +0900
---
{% raw %}
<script type="application/ld+json">
{
 "@context": "https://schema.org",
 "@type": "SoftwareSourceCode",
 "name": "aiportal-java",
 "codeRepository": "https://github.com/hkjang/aiportal-java",
 "url": "https://hkjang.github.io/aidev/projects/aiportal-java/",
 "description": "aiportal-java: 자율 개선 회차 10회, 릴리즈 0건. 최근 릴리즈 없음.",
 "inLanguage": "ko",
 "maintainer": {
  "@type": "Person",
  "name": "hkjang",
  "url": "https://github.com/hkjang"
 },
 "dateModified": "2026-09-06T00:17:21+09:00"
}
</script>

# aiportal-java

<p class="tldr"><strong>요약.</strong> aiportal-java: 자율 개선 회차 10회, 릴리즈 0건. 최근 릴리즈 없음. <span class="pill pill-merged" title="14일: 릴리즈 0, 실패 0, 경고 0, 회귀 0">건강 B</span> <span class="meta">14일: 릴리즈 0, 실패 0, 경고 0, 회귀 0</span></p>

<ul class="stats"><li><b>10</b><span>회차</span></li><li><b>1</b><span>프로젝트</span></li><li><b>0</b><span>배포 준비 완료</span></li><li><b>0</b><span>릴리즈 진행 중</span></li><li><b>9</b><span>병합 완료</span></li><li><b>0</b><span>검토 대기</span></li><li><b>0</b><span>검증 실패</span></li><li><b>0</b><span>변경 없음</span></li><li><b>1</b><span>실행 오류</span></li></ul>

## 현황

<dl class="kv">
<dt>저장소</dt><dd><a href="https://github.com/hkjang/aiportal-java">https://github.com/hkjang/aiportal-java</a></dd>
<dt>마지막 회차</dt><dd>2026-09-05 18:50 KST — <span class="pill pill-other">• 기타</span> hold: budget</dd>
<dt>최근 릴리즈</dt><dd>skipped — skipped</dd>
<dt>사유</dt><dd>릴리즈 이력이 전혀 없는 저장소. git tag 0개(전체 이력 23커밋), GitHub Release 0개(gh 인증 정상, release list 비어 있음), CHANGELOG/RELEASE/VERSION 파일 및 릴리즈 노트 양식 없음, docs/ 25개 문서에도 릴리즈 절차 없음. 유일한 버전 문자열인 build.gradle:9 version=&#x27;0.0.1-SNAPSHOT&#x27; 은 Spring Initializr 기본값으로 first commit 이후 한 번도 변경된 적 없음. .github/workflows 부재이며 .gitlab-ci.yml 은 전적으로 브랜치 트리거(main/develop -&gt; gradlew clean build -&gt; app.war 복사 -&gt; k8s rollout/podman restart)로 태그에 반응하는 워크플로나 버전별 산출물이 없음. deploy/offline/change-package.sh 는 폐쇄망 GitLab clone 으로 변경분을 옮기는 base-&gt;target diff 전송 도구로 릴리즈 패키징이 아님. 태그 형식/버전 체계/노트 위치를 새로 정하는 것은 사람의 판단이라 아무것도 만들지 않음. 저장소 변경 없음(커밋·태그 없음, worktree clean).</dd>
</dl>

## 회차 이력

<div class="table-wrap"><table class="rt" data-filter="1"><thead><tr><th>일시</th><th class="primary">프로젝트</th><th>결과</th></tr></thead><tbody><tr data-status="other"><td data-label="일시">2026-09-05 18:50</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-java/">aiportal-java</a></td><td data-label="결과"><span class="pill pill-failed" title="outcome=error">실행 오류</span> hold: budget</td></tr><tr data-status="merged"><td data-label="일시">2026-09-05 00:51</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-java/">aiportal-java</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> merged <a href="https://github.com/hkjang/aiportal-java/pull/9">PR #9</a>, release skipped</td></tr><tr data-status="merged"><td data-label="일시">2026-09-04 05:56</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-java/">aiportal-java</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> merged <a href="https://github.com/hkjang/aiportal-java/pull/8">PR #8</a>, release skipped</td></tr><tr data-status="merged"><td data-label="일시">2026-09-03 22:15</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-java/">aiportal-java</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> merged <a href="https://github.com/hkjang/aiportal-java/pull/7">PR #7</a>, release skipped</td></tr><tr data-status="merged"><td data-label="일시">2026-09-03 13:56</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-java/">aiportal-java</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> merged <a href="https://github.com/hkjang/aiportal-java/pull/6">PR #6</a>, release skipped</td></tr><tr data-status="merged"><td data-label="일시">2026-09-03 07:56</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-java/">aiportal-java</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> merged <a href="https://github.com/hkjang/aiportal-java/pull/5">PR #5</a>, release skipped</td></tr><tr data-status="merged"><td data-label="일시">2026-09-03 07:46</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-java/">aiportal-java</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> merged <a href="https://github.com/hkjang/aiportal-java/pull/4">PR #4</a>, release skipped</td></tr><tr data-status="merged"><td data-label="일시">2026-09-03 01:55</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-java/">aiportal-java</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> merged <a href="https://github.com/hkjang/aiportal-java/pull/3">PR #3</a>, release skipped</td></tr><tr data-status="merged"><td data-label="일시">2026-09-02 19:35</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-java/">aiportal-java</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> merged <a href="https://github.com/hkjang/aiportal-java/pull/2">PR #2</a>, release skipped</td></tr><tr data-status="merged"><td data-label="일시">2026-09-02 13:39</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-java/">aiportal-java</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> merged <a href="https://github.com/hkjang/aiportal-java/pull/1">PR #1</a>, release skipped</td></tr></tbody></table></div>

## 원장 (에이전트가 남긴 기록)

## 2026-09-02
- 선택: 인증정보 로그 마스킹 (SEC-002) (가치 5 / 위험 1 / 작업량 M)
- 결과: 성공
- 요약: `JwtAuthenticationFilter`가 모든 요청 헤더와 `access_token`을 INFO로 평문 기록하고 있었고, `AuthServiceImpl`(SSO), 배치 job의 adminToken, 토큰 발급 API 응답 본문에도 같은 누출이 있어 `LogMaskUtil`(민감 헤더 판별 / 값 전체 마스킹 / JSON token·password 필드 마스킹)을 추가하고 7개 파일의 로그 호출에 적용했습니다. 로드맵 단계 0의 P0 항목이며 위험이 낮고 한 세션에 끝낼 수 있어 선택했습니다. 검증은 `LogMaskUtilTest`(5) + logback ListAppender 기반 `JwtAuthenticationFilterLoggingTest`(1)를 추가한 뒤 `./gradlew check build` 실행으로 했고 28개 테스트 전부 통과했습니다. 커밋 36950f8.
- 보류 아이디어:
  - SEC-004: `/sample`, `/batch/sample` 공개 경로를 운영 프로필에서 비활성화하거나 관리자 권한으로 제한 (가치 4 / 위험 3 / 작업량 M) — 운영 배포 영향 확인 필요
  - `ControllerLogAspect.logControllerCud`가 `getAuthentication()` null 일 때 NPE 발생 가능 (가치 3 / 위험 1 / 작업량 S)
  - `JwtAuthenticationFilter`의 CORS 허용 Origin 30여 개 하드코딩을 설정(yaml)으로 외부화 (가치 3 / 위험 3 / 작업량 M)
  - `getDataMap`이 null 반환 시 호출부에서 `dataMap.get("active")` NPE — 명시적 예외 처리로 정리 (가치 3 / 위험 2 / 작업량 S)
  - 공통 util(`ValidUtil`, `PagingUtil`, `ApiCallUtil`) 단위 테스트 공백 보강 (가치 2 / 위험 1 / 작업량 S)

### 환경 메모
- 이 머신에는 JRE만 설치되어 있어(`/usr/lib/jvm/java-21-openjdk-amd64`에 javac 없음) Gradle toolchain이 실패합니다. `~/.gradle/jdks/jdk-21.0.12.1+1`(Temurin 21)을 내려받아 `JAVA_HOME`으로 지정해야 빌드가 됩니다.
- `./gradlew`에 실행 권한이 없어 `sh gradlew ...` 로 실행해야 합니다.

## 2026-09-02
- 선택: 샘플/수동 batch API 익명 공개 차단 (SEC-004) (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `/sample/**`, `/batch/sample/**` 가 `SecurityConfig` permitAll 과 `JwtAuthenticationFilter.shouldNotFilter` 양쪽에서 열려 있어 배치 수동 실행(`/batch/sample/daily`), Athena 프로젝트 삭제, 사용자 초대 같은 상태 변경 기능을 익명으로 호출할 수 있었습니다. `SampleApiProperties`(`security.sample-api.enabled`, 기본 false, env `SAMPLE_API_ENABLED`)를 추가해 두 지점의 경로 판정을 한 곳으로 모으고, 상태 변경 sample 컨트롤러 4종에 `@ConditionalOnProperty` 를 붙여 기본적으로 빈 등록 자체를 막았습니다. SSO 검증에 쓰이는 읽기 전용 `/sample/jwks` 는 설정과 무관하게 열어 두어 운영 영향을 없앴습니다. 검증은 `SampleApiAccessTest`(7) 추가 후 `sh gradlew check build` 실행으로 했고 13개 테스트 클래스 전부 통과했습니다. 커밋 1b39744.
- 보류 아이디어:
  - `ControllerLogAspect.logControllerCud`가 `getAuthentication()` null 일 때 NPE 발생 가능 (가치 3 / 위험 1 / 작업량 S)
  - `JwtAuthenticationFilter`의 CORS 허용 Origin 30여 개 하드코딩을 설정(yaml)으로 외부화 (가치 3 / 위험 3 / 작업량 M)
  - `getDataMap`이 null 반환 시 `dataMap.get("active")` NPE — 명시적 예외 처리로 정리 (가치 3 / 위험 2 / 작업량 S)
  - 공통 util(`ValidUtil`, `PagingUtil`, `ApiCallUtil`) 단위 테스트 공백 보강 (가치 2 / 위험 1 / 작업량 S)
  - SEC-003: SSO 토큰 교환의 `is_skip_jwt=true` 운영 우회 제거 (가치 5 / 위험 4 / 작업량 M) — IAM 담당자 계약 확인 필요라 자율 진행 부적합

## 2026-09-03
- 선택: 페이징 파라미터 하한 미보정으로 인한 목록 조회 오류 수정 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `PageVo.pageNum` 기본값이 0 이라 `pageNum` 을 생략한 요청에서 `getOffset()` 이 `-pageSize` 를 반환해 `OFFSET` 을 쓰는 20여 개 매퍼 쿼리가 PostgreSQL "OFFSET must not be negative" 로 실패하고, `PagingUtil.paginate()` 는 음수 `fromIndex` 로 `subList` IndexOutOfBoundsException 을 던졌습니다(예: `GET /app/chat/list`, `/tools` OCR·STT 목록). 이미 `QuestionLogServiceImpl`, `StorageBoxServiceImpl`, `ToolsServiceImpl`, `AppStatisticsServiceImpl` 4곳에 같은 보정이 개별 복사돼 있어 중앙화가 맞다고 판단했습니다. `getOffset()` 과 `paginate()` 에서 pageNum/pageSize 를 1 이상으로 보정하고 long 연산으로 오버플로를 막았으며, `ToolsMapper` 의 인라인 `((#{pageNum} - 1) * #{pageSize})` 2건을 다른 매퍼와 동일하게 `#{offset}` 로 통일했습니다. `AthenaServiceImpl` 의 `getPageNum() != 0` 센티널 의미를 깨지 않으려고 필드 기본값과 getter 는 건드리지 않았습니다. 검증은 `PageVoTest`(5) + `PagingUtilTest`(7) 추가 후 `sh gradlew check build` 실행으로 했고 47개 테스트 전부 통과했습니다. 커밋 44052f5.
- 보류 아이디어:
  - `ControllerLogAspect.logControllerCud`가 `getAuthentication()` null 일 때 NPE — @AfterReturning 이라 성공 응답이 500 으로 바뀜 (가치 3 / 위험 1 / 작업량 S)
  - `getDataMap`이 null 반환 시 `dataMap.get("active")` NPE — 명시적 예외 처리로 정리 (가치 3 / 위험 2 / 작업량 S)
  - `JwtAuthenticationFilter`의 CORS 허용 Origin 30여 개 하드코딩을 설정(yaml)으로 외부화 (가치 3 / 위험 3 / 작업량 M)
  - `LIMIT #{pageSize}` 에 음수/0 pageSize 가 그대로 전달되는 경로 보정 (가치 2 / 위험 2 / 작업량 S) — setter 보정 시 pageSize 복사 경로 영향 확인 필요
  - `ValidUtil`, `ApiCallUtil` 단위 테스트 공백 보강 (가치 2 / 위험 1 / 작업량 S)

## 2026-09-03
- 선택: 감사 로그 실패가 성공한 CUD 요청을 500 으로 뒤집는 문제 수정 (가치 4 / 위험 1 / 작업량 M)
- 결과: 성공
- 요약: `ControllerLogAspect.logControllerCud` 는 `@AfterReturning` 이라 컨트롤러가 성공하고 트랜잭션이 커밋된 뒤 실행되는데 본문 전체에 예외 처리가 없어, activity_log insert 실패·관리자 계정 조회 실패·인증정보 부재 NPE 가 그대로 전파되면 `GlobalExceptionHandler` 가 500 을 반환해 클라이언트가 이미 성공한 생성/수정/삭제를 실패로 인식하고 재시도하게 됩니다. 감사 로그 기록 전체를 try/catch 로 감싸고, `currentRequestAttributes()` 를 `getRequestAttributes()` + null 체크로 바꿨습니다. 또 `getAuthentication().getPrincipal().equals("anonymousUser")` 후 캐스팅하는 동일 패턴이 `ControllerLogAspect`, `GlobalExceptionHandler`, `AuditUtil` 3곳에 복사돼 있어 `CurrentUserUtil` 로 모으고 authentication null / principal null / CustomUserDetails 아닌 principal 을 모두 null 로 처리해 각 호출부의 기존 fallback(관리자 계정 조회 또는 BusinessException)을 타게 했습니다. `AuditUtil` 이 principal 객체 전체를 INFO 로 남기던 로그는 userId 만 DEBUG 로 축소했습니다. 검증은 `CurrentUserUtilTest`(5) + `AuditUtilTest`(4) + `ControllerLogAspectTest`(5) 추가 후 `sh gradlew check build` 실행으로 했고 17개 테스트 클래스 61개 테스트 전부 통과했습니다. 커밋 c115e68.
- 보류 아이디어:
  - `JwtAuthenticationFilter` 80행 `(boolean) dataMap.get("active")` — Athena 장애로 `getDataMap` 이 null 을 반환하면 NPE, 명시적 인증 실패로 정리 (가치 3 / 위험 2 / 작업량 S)
  - `JwtAuthenticationFilter` 의 CORS 허용 Origin 30여 개 하드코딩을 설정(yaml)으로 외부화 (가치 3 / 위험 3 / 작업량 M)
  - `LIMIT #{pageSize}` 에 음수/0 pageSize 가 그대로 전달되는 경로 보정 (가치 2 / 위험 2 / 작업량 S)
  - `ValidUtil`, `ApiCallUtil` 단위 테스트 공백 보강 (가치 2 / 위험 1 / 작업량 S)
  - `AuditUtil.getUserInfo()` 외 `TransferManagementServiceImpl` 63행도 `CurrentUserUtil` 로 통일 (가치 2 / 위험 1 / 작업량 S)

## 2026-09-03
- 선택: IAM introspect 응답 형식 미검증으로 인한 필터 500 오류 수정 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `JwtAuthenticationFilter` 80행이 `(boolean) dataMap.get("active")` 로 IAM introspect 응답을 검증 없이 꺼내고 있어, `getDataMap` 이 null 을 반환하는 경로(`jsonToVo` 파싱 실패 → null, 또는 code 200 인데 data 없음)에서 NPE 가, `active`/`ext`/`email` 타입이 다르면 ClassCastException 이 필터 밖으로 전파됐습니다. 필터에서 새어 나간 예외는 `@ControllerAdvice` 인 `GlobalExceptionHandler` 가 잡지 못해 CORS 헤더도 에러 본문도 없는 컨테이너 500 이 나가고, 브라우저는 재로그인 안내 대신 CORS 오류만 보게 됩니다. 네 가지 형식 오류를 모두 `AthenaJwtException(ATHENA_JWT_PARSING)` 으로 바꿔 필터의 기존 인증 실패 응답 경로(상태 400 + JSON 본문 + CORS 헤더)를 타게 했고, `ext` 는 있는데 `email` 이 없어 인증 없이 체인을 통과시키던 경로도 명시적 실패로 정리했습니다. `ext` 가 null 이면 M2M 으로 취급하는 기존 동작은 유지했습니다. 검증은 `JwtAuthenticationFilterIntrospectTest`(8: 형식 오류 6 + 정상 사용자 인증 / M2M 인증 회귀 가드) 추가 후 `sh gradlew check build` 실행으로 했고 18개 테스트 클래스 69개 테스트 전부 통과했습니다. 커밋 e373af9.
- 보류 아이디어:
  - `JwtAuthenticationFilter` 의 CORS 허용 Origin 30여 개 하드코딩을 설정(yaml)으로 외부화 (가치 3 / 위험 3 / 작업량 M) — catch 블록 안에서 요청마다 Set.of 를 새로 만드는 문제도 함께 해소됨
  - 필터 내 `BusinessException("사용자 정보가 없습니다.")` 도 catch 되지 않아 500 + CORS 헤더 누락 (가치 3 / 위험 2 / 작업량 S) — 상태코드 변경이라 프론트 영향 확인 필요
  - `LIMIT #{pageSize}` 에 음수/0 pageSize 가 그대로 전달되는 경로 보정 (가치 2 / 위험 2 / 작업량 S)
  - `ValidUtil`, `ApiCallUtil` 단위 테스트 공백 보강 (가치 2 / 위험 1 / 작업량 S)
  - `TransferManagementServiceImpl` 63행도 `CurrentUserUtil` 로 통일 (가치 2 / 위험 1 / 작업량 S)

## 2026-09-03
- 선택: 업로드 파일명 경로 조작 및 확장자 없는 파일명 500 오류 수정 (가치 5 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `MultipartFile#getOriginalFilename()` 은 클라이언트가 보낸 값을 그대로 돌려주는데 6개 업로드 경로(`FileServiceImpl.upload/ocrUpload/uploadDrm`, `ToolsServiceImpl.parse/parse2/decFileDrm`, `AthenaServiceImpl.decFileDrm`, `SampleDrmServiceImpl.decFileDrm`)가 이 값을 `Path#resolve` / `File#createTempFile` 에 검증 없이 넘겨 `../../` 이 포함된 파일명으로 업로드 기준 디렉토리 밖에 파일을 쓸 수 있었고, `FileServiceImpl` 의 두 업로드 메소드는 `substring(lastIndexOf("."))` 로 확장자를 뽑아 확장자 없는 파일명(예: `README`)에 StringIndexOutOfBoundsException → 500 이 났습니다. 이미 `ToolsServiceImpl` 에 private 으로 `safeOriginalFilename`/`fileExtension` 이 있었지만 STT 경로에서만 쓰여, 이를 `FileNameUtil`(`safeFileName` / `extension` / `storedFileName`)로 옮겨 모든 업로드 경로가 같은 규칙을 쓰도록 통일했습니다. 덤으로 `ocrUpload` 가 저장은 원본 파일명으로 하면서 `realName` 에는 `savedFilename` 을 기록해 이후 다운로드·삭제가 항상 실패하던 문제도 고쳤습니다. 검증은 `FileNameUtilTest`(10) + `FileServiceImplTest`(3, @TempDir 로 실제 저장 위치 확인) 추가 후 `sh gradlew check build` 실행으로 했고 20개 테스트 클래스 82개 테스트 전부 통과했습니다. 커밋 eba85d7.
- 보류 아이디어:
  - `JwtAuthenticationFilter` 의 CORS 허용 Origin 30여 개 하드코딩을 설정(yaml)으로 외부화 (가치 3 / 위험 3 / 작업량 M)
  - `FileController.down` 이 `fileService.get(fileId)` null 일 때 NPE, `uploadDrm` 에 `System.out.println` 디버그 잔존 (가치 2 / 위험 1 / 작업량 S)
  - `FileServiceImpl.ocrUpload` 의 하드코딩 경로 `E:\KCB\doc\ocr` — 호출부가 없는 사실상 죽은 코드라 설정화 또는 제거 판단 필요 (가치 2 / 위험 2 / 작업량 S)
  - `LIMIT #{pageSize}` 에 음수/0 pageSize 가 그대로 전달되는 경로 보정 (가치 2 / 위험 2 / 작업량 S)
  - 필터 내 `BusinessException("사용자 정보가 없습니다.")` 도 catch 되지 않아 500 + CORS 헤더 누락 (가치 3 / 위험 2 / 작업량 S)

## 2026-09-03
- 선택: 파일 다운로드 500 오류 및 삭제 후 잔존 파일 정보 수정 (가치 4 / 위험 1 / 작업량 M)
- 결과: 성공
- 요약: `ResponseHandController.downloadFile` 이 `MediaTypeFactory.getMediaType(realName).orElseThrow(IllegalArgumentException)` 로 되어 있어 확장자를 모르는 저장 파일(hwp/hwpx, 그리고 직전 세션에서 허용하게 된 확장자 없는 업로드 → UUID 만 남는 저장명)의 다운로드가 404 대신 500 으로 실패했고, `FileController.down` 은 `fileService.get(fileId)` 가 null 을 반환하는 삭제된 파일 아이디에 NPE 로 500 을 냈습니다. 같은 클래스의 `downFile` 이 이미 `CUSTOM_MEDIA_TYPES(hwp) → MediaTypeFactory → octet-stream` 폴백을 갖고 있어 이를 `resolveMediaType` 으로 추출해 두 경로가 같은 규칙을 쓰도록 통일하고, MIME 결정을 `Files.newInputStream` 이전으로 옮겨 예외 시 열린 스트림이 남지 않게 했습니다. Content-Length 는 DB 값 대신 실제 파일 크기를 쓰고, `fileName` 이 null 이면 저장명으로 대체합니다. 추가로 `FileServiceImpl.delete` 가 `Files.deleteIfExists` 가 false 일 때 DB 행을 남겨 목록에 계속 노출되고 재삭제가 영원히 실패하던 문제(게시판/자료실 첨부 교체 경로에서 발생)와 `ToolsServiceImpl.deleteImgHistory` 의 이력/파일 정보 null NPE 를 고쳤고, `FileController.uploadDrm` 의 디버그 `System.out.println` 을 제거했습니다. 검증은 `ResponseHandControllerDownloadTest`(8) + `FileServiceImplTest` 삭제 케이스(4) 추가 후 `sh gradlew check build` 실행으로 했고 21개 테스트 클래스 94개 테스트 전부 통과했습니다. 커밋 cfa5c48.
- 보류 아이디어:
  - `JwtAuthenticationFilter` 의 CORS 허용 Origin 30여 개 하드코딩을 설정(yaml)으로 외부화 (가치 3 / 위험 3 / 작업량 M)
  - 필터 내 `BusinessException("사용자 정보가 없습니다.")` 도 catch 되지 않아 500 + CORS 헤더 누락 (가치 3 / 위험 2 / 작업량 S)
  - `FileServiceImpl.ocrUpload` 의 하드코딩 경로 `E:\KCB\doc\ocr` — 호출부가 없는 사실상 죽은 코드라 설정화 또는 제거 판단 필요 (가치 2 / 위험 2 / 작업량 S)
  - `LIMIT #{pageSize}` 에 음수/0 pageSize 가 그대로 전달되는 경로 보정 (가치 2 / 위험 2 / 작업량 S)
  - `ValidUtil`, `ApiCallUtil` 단위 테스트 공백 보강 (가치 2 / 위험 1 / 작업량 S)

## 2026-09-04
- 선택: 페이지 크기 하한 미보정으로 인한 목록 조회 500 오류 수정 (가치 4 / 위험 1 / 작업량 M)
- 결과: 성공
- 요약: 이전 세션에서 `getOffset()` 만 보정했고 `LIMIT #{pageSize}` 는 그대로 남아 있어, 20개 매퍼 쿼리(게시판·자료실·권한검색·통계·앱관리·로그·툴 목록 등)에서 `?pageSize=-1` 이 PostgreSQL "LIMIT must not be negative" 로 500 을 냈고 `pageSize=0` 은 totalCount 가 있는데도 항상 빈 목록을 돌려주었습니다. `PageVo.getLimit()` 을 추가해 1 이상으로 보정하고 매퍼 20곳을 `#{limit}` 으로 통일했으며, `DeptStatisticsController` 등 4개 컨트롤러가 전체 내려받기 용도로 `setPageSize(Integer.MAX_VALUE)` 를 쓰고 있어 기존 세 서비스의 100 상한을 전역으로 올리지는 않고 하한만 보정했습니다. 별도 패턴이던 `AppDirectMapper` 의 `LIMIT #{params.size} OFFSET #{params.page} * #{params.size}` 도 `getLimit()`/`getOffset()` 으로 옮겨 null(빈 문자열 바인딩)·음수 page/size 를 함께 막았습니다. 검증은 `PageVoTest` 4건 추가 + `AppDirectSearchParamsTest`(6) + 매퍼 XML 의 LIMIT/OFFSET 바인딩이 보정된 프로퍼티만 쓰는지 확인하는 `MapperLimitBindingTest`(3) 추가 후 `sh gradlew check build` 실행으로 했고 23개 테스트 클래스 107개 테스트 전부 통과했습니다. 커밋 60ac379.
- 보류 아이디어:
  - `JwtAuthenticationFilter` 의 CORS 허용 Origin 30여 개 하드코딩을 설정(yaml)으로 외부화 (가치 3 / 위험 3 / 작업량 M)
  - 필터 내 `BusinessException("사용자 정보가 없습니다.")` 도 catch 되지 않아 500 + CORS 헤더 누락 (가치 3 / 위험 2 / 작업량 S)
  - `AthenaServiceImpl` 1629·2294행 등 Athena 응답의 `get(0)` / `getEmail().split("@")[0]` 무검증 접근 — 빈 배열·null 이메일에 NPE (가치 3 / 위험 2 / 작업량 S)
  - `FileServiceImpl.ocrUpload` 의 하드코딩 경로 `E:\KCB\doc\ocr` — 호출부가 없는 사실상 죽은 코드라 설정화 또는 제거 판단 필요 (가치 2 / 위험 2 / 작업량 S)
  - `ValidUtil`, `ApiCallUtil` 단위 테스트 공백 보강 (가치 2 / 위험 1 / 작업량 S)

## 2026-09-05
- 선택: 인증 필터 예외가 컨테이너 500 으로 새어 나가는 문제 수정 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `JwtAuthenticationFilter` 는 `AthenaJwtException` 만 catch 하고 있어, IAM introspect 가 active=true 로 응답했지만 포털 DB 에 사용자 행이 없을 때 던지는 `BusinessException("사용자 정보가 없습니다.")` 과 `userInfoService.getUserInfoByEmail` 이 실패할 때의 예외가 필터 밖으로 전파됐습니다. 필터는 `@ControllerAdvice` 인 `GlobalExceptionHandler` 의 처리 대상이 아니라 이 경우 클라이언트가 약속된 `ResponseDto{code,message}` 대신 컨테이너 기본 오류 응답을 받아, IAM 계정은 있지만 포털에 미등록된 사용자가 원인을 알 수 없는 500 을 보게 됩니다. 인증 로직을 `authenticate()` 로 분리하고 `filterChain.doFilter` 를 try 밖으로 옮겨 인증 단계 예외만 응답으로 변환하도록 했습니다(컨트롤러에서 올라온 예외는 삼키지 않음). `BusinessException` 은 `GlobalExceptionHandler` 와 동일한 형식으로, 그 외 예외는 `INTERNAL_SERVER_ERROR` 형식으로 내려줍니다. 덤으로 catch 블록 안에서 요청마다 새로 만들던 30여 개 Origin `Set.of` 와 `ObjectMapper` 를 static 상수로 올리고 응답 기록을 `writeErrorResponse()` 로 추출했으며, `Origin` 헤더는 항상 스킴을 포함해 절대 매칭될 수 없던 `"sso.kcb4u.com"` 항목을 `WebConfig` 와 같은 `"https://sso.kcb4u.com"` 으로 고쳤습니다. 검증은 `JwtAuthenticationFilterErrorResponseTest`(6: 사용자 미등록 / 조회 실패 / CORS 허용·비허용 Origin / sso Origin / 다운스트림 예외 비삼킴 회귀 가드) 추가 후 `sh gradlew check build` 실행으로 했고 24개 테스트 클래스 113개 테스트 전부 통과했습니다. 커밋 ff2d4a6.
- 보류 아이디어:
  - `JwtAuthenticationFilter` 와 `WebConfig` 의 허용 Origin 목록 이중 관리를 설정(yaml)으로 일원화 — 현재 두 목록이 서로 다르게 드리프트 중 (가치 3 / 위험 3 / 작업량 M)
  - `AthenaServiceImpl` 1629·2294행, `UserInfoAthenaServiceImpl` 215·362행의 JsonNode `.get(0)` 무검증 접근 — 빈 items 배열에 NPE (가치 3 / 위험 2 / 작업량 S)
  - `AthenaServiceImpl` 1790행 `userInfoVo.getEmail().split("@")[0]` — email null 에 NPE (가치 3 / 위험 1 / 작업량 S)
  - `FileServiceImpl.ocrUpload` 의 하드코딩 경로 `E:\KCB\doc\ocr` — 호출부가 없는 사실상 죽은 코드라 설정화 또는 제거 판단 필요 (가치 2 / 위험 2 / 작업량 S)
  - `ValidUtil`, `ApiCallUtil` 단위 테스트 공백 보강 (가치 2 / 위험 1 / 작업량 S)


[← 대시보드](https://hkjang.github.io/aidev/) · [교훈 모음](https://hkjang.github.io/aidev/lessons/)

{% endraw %}
