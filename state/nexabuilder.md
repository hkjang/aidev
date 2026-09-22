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
## 2026-09-19
- 선택: 목록 내보내기(CSV/XLSX/PDF)가 휴지통(soft-delete)에 든 목록도 내려주는 것을 막고, 내보내기 3종에 통합 테스트를 처음으로 붙인다 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `ListExportController` 의 세 엔드포인트는 `findById` 만 하고 `deletedAt` 을 보지 않아, 런타임 `/app/list/<id>` 가 "List not found" 로 막는 목록의 데이터를 내보내기 URL 로는 계속 받을 수 있었다. 세 곳에 복붙된 조회·권한 블록을 `resolveExportable()` 하나로 모으고 거기서 `NexaUiService.listDefinition` 과 똑같은 `IllegalArgumentException("List not found: …")` 을 던져(응답 스트림을 열기 전) 400 JSON 으로 끝나게 했다. `ListExportIntegrationTest` 5건을 새로 붙였고(실제 H2·ScreenPermissionService·POI·OpenPDF, 목 없음), 컨트롤러를 고치기 전 단독 실행에서 `softDeletedListIsNotExportableInAnyFormat` 만 "expected 400 but was 200" 으로 빨간 것을 확인한 뒤 고쳐 초록. `cleanTest test` 전체 573개(568+5) 통과, `bootJar -x test` 성공. 커밋 3bd46b2.
- 보류 아이디어: `POST /api/v1/builder/lists/{listId}/data`(UiBuilderController→NexaUiService.listData)도 soft-delete 목록의 행을 그대로 돌려줌 — 빌더 미리보기 호출자 확인 후 통일(3/3/M) / CI 에 gradle/actions/wrapper-validation 추가 + docker-publish checkout v6 정렬 + gradlew +x 로 chmod 스텝 제거(3/1/S) / ListExportController 의 OpenPDF 3 deprecated API 정리 — 이번 빌드에서도 경고 확인, 새 테스트가 안전망(2/2/S) / listDefinition·formDefinition·resolveExportable 의 "X not found" 판정을 한 헬퍼로 모으고 404 전용 예외로 전환(2/3/S) / agent.md 낡은 환경 메모 갱신(2/1/S)
- 과제서: 채택 — 과제서의 근거(세 엔드포인트가 deletedAt 을 안 봄, 테스트 0건, 400/403 매핑 경로)가 코드와 전부 일치해 그대로 구현했다.

- 릴리즈: v1.18.0 (2026-09-19, run 2026-09-19-195356-nexabuilder-improve)
## 2026-09-20
- 선택: `POST /api/v1/builder/lists/{listId}/data` 가 휴지통(soft-delete)에 든 목록의 행을 그대로 돌려주는 구멍을 막고, 목록 "살아 있음" 판정을 `NexaUiService.requireLiveList` 헬퍼 하나로 모은다 (가치 3 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: `NexaUiService.listData` 는 `findById` 만 하고 `deletedAt` 을 보지 않아, 정의 조회(`GET /api/v1/builder/lists/{id}`)·런타임 화면·내보내기가 모두 "List not found" 로 막는 목록의 행을 `/data` 로는 계속 받을 수 있었다. 존재+soft-delete 판정을 public `requireLiveList(listId)` 로 뽑아 `listDefinition`·`listData`·`ListExportController.resolveExportable` 세 곳이 공유하게 했고(예외 타입·문구·400 매핑·권한 판정 순서는 그대로, 컨트롤러의 `NexaListRepository` 필드는 제거), `BuilderListDataSoftDeleteIntegrationTest` 2건을 새로 붙였다. 헬퍼 도입 전 단독 실행에서 `softDeletedListIsNotFoundForDataAndDefinition` 만 "expected 400 but was 200" 으로 빨간 것을 확인 → 고쳐 초록 → 수정을 되돌려 다시 빨간 것까지 확인. `cleanTest test` 전체 575개(573+2) 통과, `bootJar -x test` 성공. 커밋 fa42f66.
- 보류 아이디어: `GET /api/v1/data/lists/{id}`(DataAdapterService.queryList, jqxgrid 어댑터·nexa-data.js)도 findById 만 해서 같은 구멍 — requireLiveList 로 통일(3/2/S, 이번에 새로 발견) / CI 에 gradle/actions/wrapper-validation 추가 + docker-publish checkout v6 정렬 + gradlew +x 로 chmod 스텝 3곳 제거(3/1/S) / ListExportController 의 OpenPDF 3 deprecated API 정리 — 이번 빌드에서도 경고(2/2/S) / 폼 쪽 soft-delete 판정도 requireLiveForm(formId, includeInactive) 꼴 헬퍼로(2/2/S) / `/builder/lists/{id}/data` 에 ScreenPermissionService 게이트 없음 — 테스트로 세션 user 없이 200 확인, permission 은 위험 구역이라 별도 회차(3/4/M)
- 과제서: 채택 — 근거(listData 가 deletedAt 을 안 봄, 세 곳의 중복 판정, 호출자 범위)가 코드와 전부 일치했고 선택 항목(resolveExportable 도 헬퍼 사용)까지 포함해 그대로 구현했다. 과제서가 미확인이라 한 "/data 가 세션 user 없이 200 인지"는 테스트로 확인(200, sessionAttr 불필요).

- 릴리즈: v1.19.0 (2026-09-20, run 2026-09-20-142405-nexabuilder-improve)
## 2026-09-22
- 선택: `GET /api/v1/data/lists/{listId}`(jqxgrid 어댑터)의 휴지통 목록 구멍을 `requireLiveList` 로 막고 null `pageSize` NPE(500)를 함께 고친다 (가치 4 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: `DataAdapterService.queryList` 는 `findById` 만 하고 `deletedAt` 을 보지 않아, 정의 조회·`/builder/lists/{id}/data`·내보내기 3종이 모두 "List not found" 로 막는 목록의 행을 이 어댑터 URL 로는 계속 읽을 수 있었고, 같은 줄에서 nullable 한 `page_size` 를 `int` 로 언박싱해 `page_size` 가 빈 목록은 무조건 500 이었다. 판정을 `NexaUiService.requireLiveList` 로 넘겨(이제 `listDefinition`·`listData`·`resolveExportable`·`queryList` 네 경로가 전부 같은 헬퍼 하나를 통과 — grep 으로 확인) 예외 타입·문구·400 매핑을 그대로 맞추고, `parse` 의 두 번째 인자를 `Integer` 로 바꿔 기존 `> 0` 분기를 null 까지 확장해 `listData` 와 같은 100 으로 떨어지게 했다. 신규 `DataAdapterListSoftDeleteIntegrationTest` 3건(실제 H2·MockMvc)을 수정 전에 단독 실행해 null pageSize 는 `expected:<200> but was:<500>`(스택트레이스로 `getPageSize()` 언박싱 NPE at DataAdapterService.java:65 확인), 휴지통 목록은 `expected:<400> but was:<200>` 으로 빨간 것을 먼저 보고 고쳐 초록; `cleanTest test` 전체 578건(575+3) 통과·0 skip, `bootJar -x test` 성공. 커밋 5c65840.
- 보류 아이디어: `DataAdapterService.parse` 의 페이지 입력 음수·오버플로 검증(`pagenum=-1` → subList 음수 인덱스, `pagesize=0` → SQL 백엔드 전체 행) — 계약을 먼저 정할 것(3/2/M) / `/api/v1/data/lists/{id}` 에 ScreenPermissionService 게이트 없음 — permission 은 위험 구역이라 별도 회차(3/4/M) / CI 에 wrapper-validation 추가 + docker-publish checkout v6 정렬 + gradlew 실행 비트(3/1/S) / 폼 쪽 soft-delete 판정을 requireLiveForm 헬퍼로 통일 — 현재 관찰 가능한 버그는 없음(2/2/S) / ListExportController 의 OpenPDF 3 deprecated API 정리 — 이번 빌드에서도 deprecation note 확인(2/2/S)
- 과제서: 채택 — 근거 3가지(`queryList:63-65` 가 deletedAt 미검사, 같은 줄 Integer→int 언박싱, 성공/실패 봉투가 다름)가 코드와 전부 일치했고 지정한 파일·기본값 100·`@DirtiesContext` 금지까지 그대로 따랐다. 다만 과제서가 "휴지통 케이스는 지금 200" 이라 한 것은 그 픽스처(pageSize null)에서는 NPE 가 먼저 터져 500 이었으므로, 휴지통 테스트만 `pageSize(10)` 으로 씨딩해 빨간 이유가 구멍(200+행) 자체가 되도록 바로잡았다.

