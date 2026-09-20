[`v1.19.0`](https://github.com/hkjang/nexabuilder/releases/tag/v1.19.0)
은 목록 데이터 피드 수정 릴리즈 (#28). v1.18.0 이 내보내기에서 막은
soft-delete 구멍이 `/api/v1/builder/lists/{id}/data` 에는 그대로 남아
있던 것을 막고, 목록 "살아 있음" 판정을 헬퍼 하나로 모았습니다.

### 수정 (#28)

- **휴지통에 든 목록은 `/api/v1/builder/lists/{id}/data` 에서도 "찾을 수
  없음"** (#28). `NexaUiService.listData` 는 `findById` 만 하고
  `deletedAt` 을 보지 않아, 정의 조회(`GET /api/v1/builder/lists/{id}`)·
  런타임 `/app/list/<id>`·내보내기 3종이 모두 "List not found" 로 막는
  목록의 행을 정의 호출을 건너뛴 클라이언트가 `/data` 로는 계속 받을
  수 있었다. 존재 + soft-delete 판정을 public
  `NexaUiService.requireLiveList(listId)` 로 뽑아 `listDefinition`·
  `listData`·`ListExportController.resolveExportable` 세 곳이 공유하게
  함. 예외 타입·`"List not found: …"` 문구·`GlobalExceptionHandler` 의
  400 JSON 매핑·내보내기의 권한 판정 순서는 그대로. `ListExportController`
  는 더 이상 `NexaListRepository` 를 직접 쓰지 않는다.

### 테스트 (#28)

- **`BuilderListDataSoftDeleteIntegrationTest` 신규** (#28). 실제 H2 를
  타는 2건 — 살아 있는 목록의 `/data` 는 200 + `totalCount`, `deleted_at`
  저장 뒤에는 `/data` 와 정의 조회가 같은 400 JSON. 헬퍼 도입 전 단독
  실행에서 soft-delete 케이스만 "expected 400 but was 200" 으로 빨갛던
  것을 확인한 뒤 초록으로 만듦. `cleanTest test` 575 건 (573 + 2) 통과.

