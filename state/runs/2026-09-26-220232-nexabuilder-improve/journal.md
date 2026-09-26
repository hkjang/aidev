# 회차 노트 2026-09-26-220232-nexabuilder-improve — nexabuilder
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 22:02] base pinned — master@787aa61
- [러너 22:02] autonomy release — 

## 구현 노트
- 무엇/왜: `NexaUiService.listDefinition` 이 SQL 기반 목록(`sql_id` 만, `entity_id` null)에서 null 을 `entityRepository.findById` 에 넘겨 `InvalidDataAccessApiUsageException` → **500** 이 났다. 엔티티 블록을 `entityMapFor(list)` 로 뽑아 `entityId` 가 있을 때만 조회하고 없으면 빈 맵을 내준다. 프로덕션 파일 1개 + 신규 테스트 1개.
- 확신 없는 곳: (1) 런타임 템플릿에서 빈 `entity` 맵이 실제 브라우저 렌더까지 문제없는지는 **소스 독해로만 확인**했다(`list.html:625/736/918/945/992`, `kanban/calendar/tree/map/gantt` 는 `def.entity && def.entity.keyColumn` 또는 `?.` 로 방어적). 브라우저 실행은 안 했다. (2) `sqlId`·`entityId` 가 둘 다 없는 목록은 이제 정의 조회가 200 + 빈 entity 로 나간다 — 판정은 데이터 경로(`queryList` 의 "has no sqlId nor entityId")에 남겨 뒀고, 이게 옳은 계약인지는 판단이다.
- 일부러 안 한 것: SQL 백엔드의 대문자 컬럼 키(`executeSqlBacked` 가 `lowerCaseKeys` 를 안 지남) 문제는 손대지 않았다 — 응답 키를 바꾸면 기존 SQL 정의 소비자의 컬럼 이름이 바뀌고, 이번 변경과 섞이면 원인 귀속이 안 된다. 다음 회차 후보로 `ideas.json` 에 남겼다. 리스트 디자이너가 SQL 기반 목록을 저장하지 못하는 것(UI 쪽)도 범위 밖.
- 다음 역할 주의: 새 테스트는 H2 가 뜨는 `@SpringBootTest` + MockMvc 라 DB 없이는 못 돈다. `SqlExecutor.resolveSql` 이 `@Cacheable("sqlMetadata")` 라 픽스처마다 UUID 8자 접미사로 새 `sqlId` 를 쓴다 — 고정 id 로 바꾸면 컨텍스트 캐시를 공유하는 다른 테스트와 오염된다. `@DirtiesContext` 는 붙이지 않았다(회차 규칙).
- [러너 22:20] verify passed — 검증 1개 통과 (auto)
