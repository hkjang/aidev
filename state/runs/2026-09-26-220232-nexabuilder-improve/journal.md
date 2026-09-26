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

## 비평 노트
- 확인함: diff 전체(프로덕션 1 메서드 + 신규 테스트 1), listDefinition 소비자 전수(서버는 UiBuilderController:215 하나, 프런트는 runtime/*.html 7개 — 모두 `?.`/`&&` 가드, admin 템플릿은 /api/v1/builder/lists/{id} 를 안 부름), SecurityConfig 인가 매처, GlobalExceptionHandler 매핑. 테스트 실행: 신규 4/4, `*List*/*Ui*/*DataAdapter*/*Export*` 36클래스 75테스트 실패 0.
- 판정 approve / risk low / blocking 없음. 보안·법무 모두 차단 사유 없음(새 엔드포인트·의존성·암호·개인정보 없음).
- 못 본 것: pre-fix 실패를 **실제로 실행해 보지는 않았다**(코드 수정 금지). 원장에 `- 실패 재현:` 줄이 없어, 제거된 `entity(list.getEntityId())` 가 무조건 실행 경로였다는 코드 독해로 대신했다. 브라우저 렌더도 소스 독해까지만.
- 남는 우려(릴리즈 노트용): 새 테스트 파일 javadoc(SqlBackedListDefinitionIntegrationTest.java:39)이 "came back **400**" 이라고 적었는데 실제는 **500** 이다(IllegalArgumentException → InvalidDataAccessApiUsageException 변환 → handleGeneral). 주석만 틀렸으니 다음 회차에서 문구 정정.
- 다음 회차 후보: `/api/v1/builder/lists/**`·`/api/v1/data/lists/**` 에 목록 단위 권한 게이트가 없음(이번 변경이 만든 것 아님, anyRequest().authenticated() 뿐) + 구현자가 남긴 SQL 어댑터 대문자 컬럼 키.
- [러너 22:25] review approved — 리뷰 승인 (risk=low)
- [러너 22:25] pr created — https://github.com/hkjang/nexabuilder/pull/32
- [러너 22:34] ci passed — 검사 2개 모두 success
- [러너 22:34] merge done — 3445eba
- [러너 22:53] release published — v1.23.0
- [러너 22:55] assets verified — v1.23.0 자산 1개 (이전 v1.22.0: 1)
