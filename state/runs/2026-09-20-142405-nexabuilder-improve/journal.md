# 회차 노트 2026-09-20-142405-nexabuilder-improve — nexabuilder
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 14:24] base pinned — master@e903a28
- [러너 14:24] autonomy release — 

## 정찰 노트
- 선택 이유: 2026-09-19 내보내기 fix 가 남긴 마지막 soft-delete 구멍(`/builder/lists/{id}/data`)이라 근거·픽스처·테스트 패턴이 전부 갓 검증된 상태고, 이전 정찰이 미뤘던 이유(designer 미리보기 호출자)를 이번에 grep 으로 해소함(runtime 뷰 7개 + export 만 호출). CI 묶음(wrapper-validation 등)은 가치는 같지만 코드 회귀 안전망이 없어 차선으로.
- 확신 없는 곳: `/data` 가 세션 user 없이 200 을 주는지(테스트에서 403 이면 export 테스트의 `adminSessionUser()` 세션 주입으로 대응). 이번 세션에서 gradle 실행이 권한으로 막혀 검증 명령을 직접 돌리지 못함(2026-09-19 회차와 동일 명령).
- 구현자 주의: 예외 타입·메시지·400 매핑을 바꾸지 말 것(export 테스트가 고정). `@DirtiesContext` 금지. 권한 게이트(ScreenPermissionService)를 `/data` 에 끼워 넣지 말 것 — 별도 아이디어로 적어 둠.
- 프로필은 2026-09-19 것이 코드와 일치해 다시 쓰지 않음.
- [러너 14:26] scout done — `POST /api/v1/builder/lists/{listId}/data` 가 휴지통(soft-delete)에 든 목록의 행을 그대로 돌려주는 구멍을 막고, 목�

## 구현 노트
- 무엇/왜: `NexaUiService.requireLiveList(listId)`(public) 를 두고 `listDefinition`·`listData`·`ListExportController.resolveExportable` 이 공유하게 해 `POST /api/v1/builder/lists/{id}/data` 가 휴지통 목록의 행을 주던 구멍을 막음. 예외 타입·"List not found: …" 문구·400 매핑·export 의 권한 판정 순서는 그대로. 커밋 fa42f66.
- 확신 없는 곳·검증 못 한 것: 런타임 뷰 7개 템플릿(kanban/tree/… .html)은 브라우저에서 직접 열어 보지 않았음 — 코드상 `listDefinition` 이 먼저 막으므로 동작 변화는 없어야 하나 화면 확인은 안 함. `ListExportController` 에서 `NexaListRepository` 필드를 뺐는데, 컴파일·통합테스트 5건이 통과했으니 다른 사용처는 없었음(grep 으로도 확인).
- 일부러 하지 않은 것: `GET /api/v1/data/lists/{id}`(`DataAdapterService.queryList`, core/data)에 같은 구멍이 있는 것을 발견했지만 과제서 범위 밖·다른 서비스라 손대지 않고 ideas.json 에 3/2/S 로 올림. `/data` 에 ScreenPermissionService 게이트도 넣지 않음(과제서 지시). `formDefinition` 은 안 건드림.
- 다음 역할이 조심할 것: 새 테스트 `BuilderListDataSoftDeleteIntegrationTest` 는 실제 H2 컨텍스트가 필요(`@SpringBootTest`, `@DirtiesContext` 없음, UUID 접미사 픽스처·정리 안 함). 검증 명령: `sh ./gradlew --no-daemon cleanTest test` → 575개 통과, `sh ./gradlew --no-daemon bootJar -x test` 성공. 빌드 경고(deprecated API, unchecked)는 이전과 동일하고 이번 변경으로 늘지 않음.
- [러너 14:37] brief accepted — 채택 — 근거(listData 가 deletedAt 을 안 봄, 세 곳의 중복 판정, 호출자 범위)가 코드와 전부 일치했고 선택 항목(resolveExportab
