---
title: "aiportal-java — 자율 개선 이력"
description: "aiportal-java: 자율 개선 회차 12회, 릴리즈 0건. 최근 릴리즈 없음."
last_modified_at: 2026-09-06 19:57:49 +0900
---
{% raw %}
<script type="application/ld+json">
{
 "@context": "https://schema.org",
 "@type": "SoftwareSourceCode",
 "name": "aiportal-java",
 "codeRepository": "https://github.com/hkjang/aiportal-java",
 "url": "https://hkjang.github.io/aidev/projects/aiportal-java/",
 "description": "aiportal-java: 자율 개선 회차 12회, 릴리즈 0건. 최근 릴리즈 없음.",
 "inLanguage": "ko",
 "maintainer": {
  "@type": "Person",
  "name": "hkjang",
  "url": "https://github.com/hkjang"
 },
 "dateModified": "2026-09-06T19:57:49+09:00"
}
</script>

# aiportal-java

<p class="tldr"><strong>요약.</strong> aiportal-java: 자율 개선 회차 12회, 릴리즈 0건. 최근 릴리즈 없음. <span class="pill pill-merged" title="14일: 릴리즈 0, 실패 0, 경고 0, 회귀 0">건강 B</span> <span class="meta">14일: 릴리즈 0, 실패 0, 경고 0, 회귀 0</span></p>

<ul class="stats"><li><b>12</b><span>회차</span></li><li><b>1</b><span>프로젝트</span></li><li><b>0</b><span>배포 준비 완료</span></li><li><b>0</b><span>릴리즈 진행 중</span></li><li><b>9</b><span>병합 완료</span></li><li><b>0</b><span>검토 대기</span></li><li><b>3</b><span>검증 실패</span></li><li><b>0</b><span>변경 없음</span></li><li><b>0</b><span>실행 오류</span></li><li><b>$7.68</b><span>비용</span></li><li><b>24분</b><span>에이전트 시간</span></li></ul>

## 현황

<dl class="kv">
<dt>저장소</dt><dd><a href="https://github.com/hkjang/aiportal-java">https://github.com/hkjang/aiportal-java</a></dd>
<dt>마지막 회차</dt><dd>2026-09-06 12:37 KST — <span class="pill pill-other">• 기타</span> verify failed: 실행한 검증 명령이 없음 — 정책(verify) 또는 자동 감지 필요</dd>
<dt>최근 릴리즈</dt><dd>skipped — skipped</dd>
<dt>사유</dt><dd>릴리즈 이력이 전혀 없는 저장소. git tag 0개(전체 이력 23커밋), GitHub Release 0개(gh 인증 정상, release list 비어 있음), CHANGELOG/RELEASE/VERSION 파일 및 릴리즈 노트 양식 없음, docs/ 25개 문서에도 릴리즈 절차 없음. 유일한 버전 문자열인 build.gradle:9 version=&#x27;0.0.1-SNAPSHOT&#x27; 은 Spring Initializr 기본값으로 first commit 이후 한 번도 변경된 적 없음. .github/workflows 부재이며 .gitlab-ci.yml 은 전적으로 브랜치 트리거(main/develop -&gt; gradlew clean build -&gt; app.war 복사 -&gt; k8s rollout/podman restart)로 태그에 반응하는 워크플로나 버전별 산출물이 없음. deploy/offline/change-package.sh 는 폐쇄망 GitLab clone 으로 변경분을 옮기는 base-&gt;target diff 전송 도구로 릴리즈 패키징이 아님. 태그 형식/버전 체계/노트 위치를 새로 정하는 것은 사람의 판단이라 아무것도 만들지 않음. 저장소 변경 없음(커밋·태그 없음, worktree clean).</dd>
</dl>

## 회차 이력

<div class="table-wrap"><table class="rt" data-filter="1"><thead><tr><th>일시</th><th class="primary">프로젝트</th><th>결과</th></tr></thead><tbody><tr data-status="other"><td data-label="일시">2026-09-06 12:37</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-java/">aiportal-java</a></td><td data-label="결과"><span class="pill pill-failed" title="outcome=verify-failed">검증 실패</span> verify failed: 실행한 검증 명령이 없음 — 정책(verify) 또는 자동 감지 필요<div class="meta">5파일 <span style="color:var(--good)">+411</span>/<span style="color:var(--bad)">−17</span> · 테스트 2 — fix(athena): 응답 형식이 어긋난 Athena/IAM 응답에서 발생하는 NPE 수정</div></td></tr><tr data-status="other"><td data-label="일시">2026-09-06 12:27</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-java/">aiportal-java</a></td><td data-label="결과"><span class="pill pill-failed" title="outcome=verify-failed">검증 실패</span> verify failed: 실행한 검증 명령이 없음 — 정책(verify) 또는 자동 감지 필요<div class="meta">11파일 <span style="color:var(--good)">+351</span>/<span style="color:var(--bad)">−80</span> · 테스트 6 — refactor(cors): 허용 Origin 목록 이중 관리로 인한 인증 실패 응답 CORS 누락 수정</div></td></tr><tr data-status="other"><td data-label="일시">2026-09-06 02:19</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-java/">aiportal-java</a></td><td data-label="결과"><span class="pill pill-failed" title="outcome=verify-failed">검증 실패</span> verify failed: 실행한 검증 명령이 없음 — 정책(verify) 또는 자동 감지 필요<div class="meta">3파일 <span style="color:var(--good)">+225</span>/<span style="color:var(--bad)">−0</span> · 테스트 1 — fix(error): 잘못된 요청 파라미터가 400 대신 500 으로 응답되는 문제 수정</div></td></tr><tr data-status="merged"><td data-label="일시">2026-09-05 00:51</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-java/">aiportal-java</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> merged <a href="https://github.com/hkjang/aiportal-java/pull/9">PR #9</a>, release skipped</td></tr><tr data-status="merged"><td data-label="일시">2026-09-04 05:56</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-java/">aiportal-java</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> merged <a href="https://github.com/hkjang/aiportal-java/pull/8">PR #8</a>, release skipped</td></tr><tr data-status="merged"><td data-label="일시">2026-09-03 22:15</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-java/">aiportal-java</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> merged <a href="https://github.com/hkjang/aiportal-java/pull/7">PR #7</a>, release skipped</td></tr><tr data-status="merged"><td data-label="일시">2026-09-03 13:56</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-java/">aiportal-java</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> merged <a href="https://github.com/hkjang/aiportal-java/pull/6">PR #6</a>, release skipped</td></tr><tr data-status="merged"><td data-label="일시">2026-09-03 07:56</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-java/">aiportal-java</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> merged <a href="https://github.com/hkjang/aiportal-java/pull/5">PR #5</a>, release skipped</td></tr><tr data-status="merged"><td data-label="일시">2026-09-03 07:46</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-java/">aiportal-java</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> merged <a href="https://github.com/hkjang/aiportal-java/pull/4">PR #4</a>, release skipped</td></tr><tr data-status="merged"><td data-label="일시">2026-09-03 01:55</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-java/">aiportal-java</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> merged <a href="https://github.com/hkjang/aiportal-java/pull/3">PR #3</a>, release skipped</td></tr><tr data-status="merged"><td data-label="일시">2026-09-02 19:35</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-java/">aiportal-java</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> merged <a href="https://github.com/hkjang/aiportal-java/pull/2">PR #2</a>, release skipped</td></tr><tr data-status="merged"><td data-label="일시">2026-09-02 13:39</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-java/">aiportal-java</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> merged <a href="https://github.com/hkjang/aiportal-java/pull/1">PR #1</a>, release skipped</td></tr></tbody></table></div>

## 비용·사용량

<div class="table-wrap"><table class="rt"><caption class="meta">최근 30세션</caption><thead><tr><th>시각</th><th class="primary">프로젝트</th><th>단계</th><th class="num">시간</th><th class="num">턴</th><th class="num">비용</th><th class="num">토큰 입력/출력</th><th>종료</th></tr></thead><tbody><tr data-status="other"><td data-label="시각">12:37</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-java/">aiportal-java</a></td><td data-label="단계">개선</td><td data-label="시간" class="num">7분</td><td data-label="턴" class="num">48</td><td data-label="비용" class="num">$2.75</td><td data-label="토큰 입력/출력" class="num">2.3M / 33K</td><td data-label="종료">success</td></tr><tr data-status="other"><td data-label="시각">12:27</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-java/">aiportal-java</a></td><td data-label="단계">개선</td><td data-label="시간" class="num">7분</td><td data-label="턴" class="num">39</td><td data-label="비용" class="num">$2.32</td><td data-label="토큰 입력/출력" class="num">1.6M / 33K</td><td data-label="종료">success</td></tr><tr data-status="other"><td data-label="시각">02:19</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-java/">aiportal-java</a></td><td data-label="단계">개선</td><td data-label="시간" class="num">9분</td><td data-label="턴" class="num">55</td><td data-label="비용" class="num">$2.61</td><td data-label="토큰 입력/출력" class="num">2.5M / 28K</td><td data-label="종료">success</td></tr></tbody></table></div>

## 아이디어 백로그 — 대기 6 / 전체 8

<div class="table-wrap"><table class="rt"><caption class="meta">에이전트가 회차마다 재평가한다. 가치 높고 위험 낮은 대기 항목이 다음 회차 후보다.</caption><thead><tr><th class="primary">아이디어</th><th>가치/위험/크기</th><th>상태</th><th>메모</th><th>갱신</th></tr></thead><tbody><tr data-status="nochange"><td data-label="아이디어" class="primary">컨트롤러가 @RequestParam user_id 를 인증 사용자와 대조 없이 신뢰 (IDOR)</td><td data-label="가치/위험/크기">5/4/L</td><td data-label="상태">대기</td><td data-label="메모">AppController·ToolsController·WidgetController 등 10여 곳. ChatController 는 2026-09-02 세션에서 principal 로 덮어쓰도록 고쳐졌다. 프론트 영향 범위가 넓어 엔드포인트를 나눠 여러 세션에 걸쳐 진행할 것</td><td data-label="갱신">2026-09-06</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">인증 필터의 요청 헤더 전량 INFO 로깅 축소</td><td data-label="가치/위험/크기">3/2/S</td><td data-label="상태">대기</td><td data-label="메모">JwtAuthenticationFilter 가 요청마다 모든 헤더와 profile 을 INFO 로 남긴다. LogMaskUtil 로 마스킹은 하지만 요청량 대비 로그량이 크고 운영 로그에서 실제 오류를 가린다. DEBUG 로 낮추거나 필요한 헤더만 남기는 방안</td><td data-label="갱신">2026-09-06</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">남은 외부 응답 무검증 접근에 JsonNodeUtil 확대 적용</td><td data-label="가치/위험/크기">3/2/M</td><td data-label="상태">대기</td><td data-label="메모">이번에 유틸을 만들었으므로 getAthenaCallApi 반환값의 (JsonNode) 무검증 캐스팅, PythonApiCallUtil/RerankModelCallUtil 응답 파싱 등 나머지 경로에도 같은 방식을 적용할 수 있다</td><td data-label="갱신">2026-09-06</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">ValidUtil, ApiCallUtil 단위 테스트 공백 보강</td><td data-label="가치/위험/크기">2/1/S</td><td data-label="상태">대기</td><td data-label="메모">common/util 아래 16개 중 6개만 테스트 존재 (이번에 JsonNodeUtilTest 추가)</td><td data-label="갱신">2026-09-06</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">CORS 허용 메서드·헤더도 CorsProperties 로 옮기고 필터 오류 응답에 Vary: Origin 추가</td><td data-label="가치/위험/크기">2/2/S</td><td data-label="상태">대기</td><td data-label="메모">이전 세션에서 Origin 만 설정화했다. WebConfig 의 allowedMethods/allowedHeaders/exposedHeaders 는 아직 하드코딩이고, 필터가 직접 쓰는 오류 응답에는 Vary: Origin 이 없어 캐시가 잘못된 CORS 헤더를 재사용할 여지가 있다</td><td data-label="갱신">2026-09-06</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">FileServiceImpl.ocrUpload 의 하드코딩 경로 E:\KCB\doc\ocr 설정화 또는 제거</td><td data-label="가치/위험/크기">2/2/S</td><td data-label="상태">대기</td><td data-label="메모">호출부가 없는 사실상 죽은 코드. 제거 쪽이 깔끔하나 외부 호출 여부 확인 필요</td><td data-label="갱신">2026-09-06</td></tr><tr data-status="released"><td data-label="아이디어" class="primary">AthenaServiceImpl 1790행 userInfoVo.getEmail().split(&quot;@&quot;)[0] — email null 에 NPE</td><td data-label="가치/위험/크기">3/1/S</td><td data-label="상태">완료</td><td data-label="메모">checkMyProject 에서 email 이 null/공백이면 userInfoVo == null 과 동일하게 false 를 반환하고 경고 로그를 남기도록 수정. AthenaServiceImplResponseGuardTest 로 회귀 가드. 커밋 602dfc3</td><td data-label="갱신">2026-09-06</td></tr><tr data-status="released"><td data-label="아이디어" class="primary">AthenaServiceImpl 무검증 JsonNode 체인 접근으로 인한 NPE</td><td data-label="가치/위험/크기">3/2/S</td><td data-label="상태">완료</td><td data-label="메모">JsonNodeUtil(경로 탐색/필수값 검증/배열 첫 요소)을 추가해 AthenaServiceImpl 1629·2294·2545·2963~2967·3164행과 UserInfoAthenaServiceImpl 215행에 적용. total&gt;0 인데 items 가 빈 응답, config 에 knowledge_id 가 없는 응답에서 NPE 대신 폴백 또는 이름이 담긴 IllegalArgumentException. (ObjectNode) 캐스팅의 ClassCastException 도 함께 차단. 커밋 602dfc3</td><td data-label="갱신">2026-09-06</td></tr></tbody></table></div>

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
## 2026-09-06
- 선택: 잘못된 요청 파라미터가 400 대신 500 으로 응답되는 문제 수정 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `GlobalExceptionHandler` 의 `@ExceptionHandler(Exception.class)` 는 Spring 의 `DefaultHandlerExceptionResolver` 보다 먼저 동작하기 때문에, Spring 이 원래 400 으로 내려 주던 클라이언트 입력 오류(`?limit=abc` 같은 파라미터 타입 불일치, 필수 파라미터 누락, 깨진 JSON 본문, 필수 업로드 항목 누락)까지 전부 삼켜서 `C002 서버 내부 오류가 발생했습니다`(500) 로 응답했습니다. 호출자는 자기 요청이 잘못된 것인지 서버가 죽은 것인지 구분할 수 없고, 이 요청들이 activity_log 에 서버 오류로 쌓여 실제 장애를 가립니다(컨트롤러 45개 `@RequestParam`, 44개 `@RequestBody` 가 모두 해당). 핸들러 인자 처리 단계에서 발생하는 `MethodArgumentTypeMismatchException`·`MissingServletRequestParameterException`·`MissingServletRequestPartException`·`HttpMessageNotReadableException`·`BindException` 을 400(C001) 로, `HttpMediaTypeNotSupportedException` 을 415(신규 `ErrorCode.UNSUPPORTED_MEDIA_TYPE`, C003) 로 매핑하고 어떤 파라미터가 문제인지 알려 주는 메시지를 붙였습니다(예외 원문에는 내부 타입·패키지명이 섞여 있어 그대로 노출하지 않음). 매핑 단계에서 던져지는 예외(405·404 등)는 handler 가 null 이라 `basePackages` 로 제한된 이 advice 의 대상이 아니어서 범위에서 제외했고, 기존 `MethodArgumentNotValidException` 핸들러는 `BindException` 보다 구체적이라 그대로 우선합니다. 검증은 `GlobalExceptionHandlerInvalidRequestTest`(8: 타입 불일치 / 필수 파라미터 누락 / 깨진 JSON / 필수 업로드 항목 누락 / 415 / 정상 요청 / 서버 예외 500 유지 / BusinessException 400 유지 회귀 가드) 를 standalone MockMvc 로 추가한 뒤 `sh gradlew check build` 실행으로 했고 25개 테스트 클래스 121개 테스트 전부 통과했습니다. 커밋 5cb50b3. (참고: 이 세션 환경에는 JDK 가 없고 JRE 만 있어 Temurin 21 을 `~/jdks` 에 내려받아 `JAVA_HOME` 을 지정해 빌드했습니다. 저장소에는 아무 변경도 하지 않았습니다.)
- 보류 아이디어:
  - `JwtAuthenticationFilter` 와 `WebConfig` 의 허용 Origin 목록 이중 관리를 설정(yaml)으로 일원화 — 현재 두 목록이 서로 다르게 드리프트 중 (가치 3 / 위험 3 / 작업량 M)
  - `AthenaServiceImpl` 2294행 `root.get("data").get("data").get("documents").get(0)` 등 무검증 JsonNode 체인 — 응답 형식이 다르면 NPE (가치 3 / 위험 2 / 작업량 S)
  - `AthenaServiceImpl` 1790행 `userInfoVo.getEmail().split("@")[0]` — email null 에 NPE (가치 3 / 위험 1 / 작업량 S)
  - `AppController`·`ToolsController`·`WidgetController` 등이 `@RequestParam user_id` 를 그대로 신뢰 — 인증 사용자와 대조하지 않아 타인 데이터 접근 가능(IDOR). 프론트 영향 범위가 넓어 별도 세션 필요 (가치 5 / 위험 4 / 작업량 L)
  - `FileServiceImpl.ocrUpload` 의 하드코딩 경로 `E:\KCB\doc\ocr` — 호출부가 없는 사실상 죽은 코드라 설정화 또는 제거 판단 필요 (가치 2 / 위험 2 / 작업량 S)

## 2026-09-06
- 선택: 허용 Origin 목록 이중 관리로 인한 인증 실패 응답 CORS 헤더 누락 수정 (가치 3 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: CORS 헤더는 두 단계에서 붙는다 — 정상 응답은 MVC 의 `WebConfig`, 인증 실패처럼 필터가 직접 기록하는 응답은 `JwtAuthenticationFilter` 다. 두 곳이 30여 개 Origin 을 각각 하드코딩하고 있어 목록이 서로 다르게 드리프트했고, 필터 쪽에만 `https://ai-portal-front-local.kubagents.koreacb.com:8080` 과 `https://daiportal-java.kubagents-ofc.koreacb.com` 이 빠져 있어 그 두 프론트는 인증 실패 시 401/500 본문 대신 상태코드도 메시지도 읽을 수 없는 CORS 오류를 보게 됩니다(직전 세션의 필터 오류 응답 개선이 이 두 Origin 에서는 무효). 목록을 `application.yaml` 의 `security.cors.allowed-origins` 한 곳으로 옮기고 새 `CorsProperties`(@ConfigurationProperties, 병합 결과 캐시)를 두 곳이 함께 읽도록 했습니다. 실제 CORS 정책인 `WebConfig` 목록을 기준으로 삼았고(중복 1건 제거), 필터에만 있던 `http://localhost:8080` 은 정상 요청에서는 어차피 차단되던 값이라 운영 정책을 넓히지 않기 위해 제외하는 대신 재빌드 없이 덧붙일 수 있는 `CORS_ADDITIONAL_ALLOWED_ORIGINS` 환경변수를 추가하고 `ENVIRONMENT_VARIABLES_GUIDE.md` 에 문서화했습니다. 검증은 `CorsPropertiesTest`(6: yaml 이 기존 두 목록의 34개 Origin 을 모두 포함하는지 / 모든 항목이 스킴 포함·경로 없음 / 매칭·null 안전 / 환경변수 병합·공백 무시 / 캐시 갱신 / WebConfig 등록 내용) 와 `JwtAuthenticationFilterErrorResponseTest` 에 드리프트 회귀 테스트 1건을 추가하고, 기존 필터 테스트 4곳이 실제 `application.yaml` 목록을 바인딩해 쓰도록 `CorsPropertiesFixture` 를 만든 뒤 `sh gradlew check build` 를 실행해 했고 25개 테스트 클래스 120개 테스트 전부 통과했습니다. 커밋 acc1904. (참고: 이 세션 환경에도 JDK 가 없어 Temurin 21 을 `$HOME/jdks` 에 내려받아 빌드했습니다. 저장소에는 아무 변경도 하지 않았습니다.)
- 보류 아이디어:
  - `AthenaServiceImpl` 2294행 `root.get("data").get("data").get("documents").get(0)` 등 무검증 JsonNode 체인 — 응답 형식이 다르면 NPE (가치 3 / 위험 2 / 작업량 S)
  - `AthenaServiceImpl` 1790행 `userInfoVo.getEmail().split("@")[0]` — email null 에 NPE (가치 3 / 위험 1 / 작업량 S)
  - `AppController`·`ToolsController`·`WidgetController` 등이 `@RequestParam user_id` 를 그대로 신뢰 — 인증 사용자와 대조하지 않아 타인 데이터 접근 가능(IDOR). 엔드포인트를 나눠 여러 세션에 걸쳐 진행 필요 (가치 5 / 위험 4 / 작업량 L)
  - `FileServiceImpl.ocrUpload` 의 하드코딩 경로 `E:\KCB\doc\ocr` — 호출부가 없는 사실상 죽은 코드라 설정화 또는 제거 판단 필요 (가치 2 / 위험 2 / 작업량 S)
  - `ValidUtil`, `ApiCallUtil` 단위 테스트 공백 보강 — common/util 15개 중 5개만 테스트 존재 (가치 2 / 위험 1 / 작업량 S)

## 2026-09-06
- 선택: Athena/IAM 응답 무검증 JsonNode 체인 접근 및 email null NPE 수정 (가치 3 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: 외부 Athena/IAM 응답을 `root.get("data").get("data").get("documents").get(0).get("id")` 처럼 무검증으로 체인 접근하던 곳들이, 응답 형식이 조금만 달라도 NPE 로 끝나 클라이언트에는 "서버 내부 오류"(500) 로만 보이고 로그에도 어떤 값이 비어 있었는지 남지 않았습니다. 특히 `getDirectoryId`(1629행)와 `UserInfoAthenaServiceImpl.createProject`(215행)는 `total > 0` 인데 `items` 가 빈 응답에서, `detailApp`(2963~2966행)은 `config` 에 `knowledge_id` 등이 없을 때 터졌는데 — 후자는 바로 아래에 `validateOrThrow(param.get("knowledge_id").toString(), "지식저장소 ID")` 라는 검증 의도가 이미 있었지만 그 전에 NPE 가 나서 의도한 메시지가 한 번도 나올 수 없었습니다. 공통 유틸 `JsonNodeUtil`(경로 탐색 / 필수값 검증 / 배열 첫 요소, 없으면 기존 `validateOrThrow` 와 같은 `"%s(이)가 없습니다."` 형식의 IllegalArgumentException)을 추가해 7개 호출부에 적용했고, 복구 가능한 곳(`getDirectoryId` → 디렉토리 생성 경로, `checkMyProject` 의 email null → false)은 예외 대신 기존 폴백 경로를 타게 했습니다. `(ObjectNode) node.path("data").path("application")` 의 ClassCastException 가능성과 `UserInfoAthenaServiceImpl` 의 예외 메시지에 원인이 문자열 리터럴로 박혀 있던 문제(`"... {}, e.getMessage()"`)도 함께 고쳤습니다. 검증은 `JsonNodeUtilTest`(9) + `AthenaServiceImplResponseGuardTest`(8: items 불일치 / items 누락 / 정상 / email null / 사용자 없음 / data 없는 배포 이력 / 정상 배포 이력) 추가 후 `sh gradlew check build` 실행으로 했고 26개 테스트 클래스 130개 테스트 전부 통과했습니다. 커밋 602dfc3. (참고: 이 세션 환경에도 JDK 가 없어 Temurin 21 을 `$HOME/jdks` 에 내려받아 빌드했습니다. 저장소에는 아무 변경도 하지 않았습니다.)
- 보류 아이디어:
  - `AppController`·`ToolsController`·`WidgetController` 등이 `@RequestParam user_id` 를 그대로 신뢰 — 인증 사용자와 대조하지 않아 타인 데이터 접근 가능(IDOR). 엔드포인트를 나눠 여러 세션에 걸쳐 진행 필요 (가치 5 / 위험 4 / 작업량 L)
  - 인증 필터의 요청 헤더 전량 INFO 로깅 축소 — 요청마다 모든 헤더를 남겨 운영 로그에서 실제 오류를 가림 (가치 3 / 위험 2 / 작업량 S)
  - `AthenaServiceImpl` 에 남은 무검증 외부 응답 접근 확대 적용 — 이번에 유틸을 만들었으므로 `getAthenaCallApi` 반환값 `(JsonNode)` 캐스팅 등 나머지 경로에도 적용 (가치 3 / 위험 2 / 작업량 M)
  - CORS 허용 메서드·헤더도 `CorsProperties` 로 옮기고 필터 오류 응답에 `Vary: Origin` 추가 (가치 2 / 위험 2 / 작업량 S)
  - `FileServiceImpl.ocrUpload` 의 하드코딩 경로 `E:\KCB\doc\ocr` — 호출부가 없는 사실상 죽은 코드라 설정화 또는 제거 판단 필요 (가치 2 / 위험 2 / 작업량 S)


[← 대시보드](https://hkjang.github.io/aidev/) · [교훈 모음](https://hkjang.github.io/aidev/lessons/)

{% endraw %}
