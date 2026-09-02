# aiportal-java 자율 개선 기록

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
