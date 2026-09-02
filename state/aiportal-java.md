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
