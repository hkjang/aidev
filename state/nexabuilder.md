## 2026-09-13
- 선택: [수정 과제] JDK 21 이 없는 머신에서도 Gradle 툴체인을 내려받아 릴리즈 빌드가 되게 한다 (가치 5 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: 릴리즈 검증 단계의 명령(`./gradlew --no-daemon bootJar -x test`)을 그대로 돌려 실패를 재현했다 — build.gradle.kts 가 Java 21 툴체인을 고정하는데 실행기에는 javac 없는 JRE 21 만 있어 "Cannot find a Java installation … Toolchain download repositories have not been configured" 로 멈춘다. 코드 문제가 아니라 툴체인 리졸버 부재라 settings.gradle.kts 에 `org.gradle.toolchains.foojay-resolver-convention` 1.0.0 을 붙여 Gradle 이 Temurin 21 을 `$GRADLE_USER_HOME/jdks` 에 한 번 내려받아 재사용하게 했고(setup-java 가 있는 CI 와 Temurin 이미지 Dockerfile 에서는 내려받지 않음), 워크플로는 손대지 않았다. JAVA_HOME 을 지운 채 bootJar 와 `cleanTest test`(563개 통과)를 재실행해 확인했고, 문서·워크플로 스텝 이름에 남아 있던 JDK 17 표기를 21 로 맞췄다. 커밋 4174eaf.
- 보류 아이디어: Spring Boot 4.1 이전(dependabot #5 재오픈 → 플러그인 적용 확인 → Framework 7 / jakarta / 프로퍼티 이름 변경 추종; 메이저라 별도 회차, L) / CI 에 gradle wrapper 검증 액션 추가(공급망 보안, S) / docker-publish.yml 의 actions/checkout@v4 를 다른 워크플로와 같이 v6 로 맞추기(S) / ListExportController 의 OpenPDF 3 deprecated API 정리(S) / WorkflowHttpActionTest 의 @DirtiesContext(AFTER_EACH_TEST_METHOD) 를 줄여 테스트 메모리·시간 절감(M)

- 릴리즈: v1.16.0 (2026-09-14, run 2026-09-14-123211-nexabuilder-release)
## 2026-09-17
- 선택: WorkflowHttpActionTest 의 @DirtiesContext(AFTER_EACH_TEST_METHOD) 제거로 테스트 컨텍스트 재생성 4회 → 0회 (가치 3 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: 그 어노테이션은 "@Async 감사 기록이 컨텍스트 종료와 경합하지 않도록" 붙어 있었지만 실제로는 매 메서드마다 컨텍스트를 닫는 것이 경합을 만드는 쪽이었고, 테스트 4개에 컨텍스트 4개를 지어 스위트가 힙 2g 를 요구하게 된 가장 큰 원인이었다(#11). 어노테이션을 빼고 각 테스트가 자기 인스턴스의 WORKFLOW_HTTP 감사 행이 커밋될 때까지 기다리는 awaitAudit 헬퍼를 거치게 해 어느 시점에 컨텍스트가 닫히든 진행 중인 비동기 작업이 없게 했으며, 흩어진 폴링 루프 2개를 합치고 응답 본문이 details 에 담기는지 검증을 더했다. 단독 실행으로 컨텍스트 기동 4→1 을 확인한 뒤 `cleanTest test` 전체 568개 통과, 스위트 전체 컨텍스트 기동 26→22(이 클래스는 캐시 재사용). maxHeapSize 2g 는 여유분으로 남기고 build.gradle.kts 주석만 현 상태로 고쳤다. 커밋 7a40a00.
- 보류 아이디어: CI 에 gradle/actions/wrapper-validation 추가(공급망 보안, 3/1/S) / docker-publish.yml 의 actions/checkout@v4 를 v6 로 맞추기(2/1/S) / ListExportController 의 OpenPDF 3 deprecated API 정리(2/2/S) / docs/operations/deployment.md 의 nexabuilder-1.2.0.jar 표기를 버전 무관 표현으로(1/1/S) / gradlew 의 git 파일 모드를 +x 로 고쳐 워크플로 3곳의 chmod 스텝 제거(2/1/S) / Jackson2JsonConfig 의 MappingJackson2HttpMessageConverter [removal] 경고 — Jackson 3 이전과 묶이는 L 작업(3/4/L). Spring Boot 4.1 이전은 #16 으로 이미 완료.

- 릴리즈: v1.17.0 (2026-09-17, run 2026-09-17-064306-nexabuilder-improve)
