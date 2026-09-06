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
