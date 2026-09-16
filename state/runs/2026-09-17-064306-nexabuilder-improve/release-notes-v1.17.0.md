`v1.17.0` 은 테스트 스위트 정리 릴리즈 (#26). 애플리케이션 동작 변경은 없습니다.

## 테스트 (#26)

- **`WorkflowHttpActionTest` 가 메서드마다 컨텍스트를 새로 만들지 않음** (#26). `@DirtiesContext(AFTER_EACH_TEST_METHOD)` 는 "@Async 감사 기록이 컨텍스트 종료와 경합하지 않도록" 붙어 있었지만 실제로는 매번 컨텍스트를 닫는 쪽이 그 경합을 만들었고, 테스트 4개에 컨텍스트 4개를 지어 스위트가 힙 2g 를 요구하게 된 가장 큰 원인이었다 (#11). 대신 각 테스트가 자기 인스턴스의 `WORKFLOW_HTTP` 감사 행이 커밋될 때까지 기다리는 `awaitAudit` 헬퍼를 거치게 해 어느 시점에 컨텍스트가 닫히든 진행 중인 비동기 작업이 없게 함. 흩어져 있던 폴링 루프 2개를 이 헬퍼로 합치고, `writesAuditRowForEveryCall` 은 응답 본문이 `details` 에 담기는지까지 검증. 이 클래스의 컨텍스트 기동 4 → 1, 스위트 전체 26 → 22. `maxHeapSize = "2g"` 는 여유분으로 남기고 `build.gradle.kts` 주석만 현 상태에 맞춤.

**Full Changelog**: https://github.com/hkjang/nexabuilder/compare/v1.16.0...v1.17.0
