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
