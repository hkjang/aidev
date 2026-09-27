# 회차 노트 2026-09-27-155203-nexabuilder-improve — nexabuilder
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 15:52] base pinned — master@7f168ac
- [러너 15:52] autonomy release — 

## 정찰 노트
- 고른 이유: 대체 뷰 7종(칸반·차트·피벗·간트·캘린더·지도·트리)이 전부 죽어 있는 사용자 가시 버그이고, 원인이 `NexaUiService.listData` 한 곳이며 같은 파라미터를 올바로 처리하는 대조 구현(`EntityCrudController:53-59`, 주석까지 있음)이 저장소 안에 있다. 보류 1순위였던 "listData 대문자 키" 는 소비자 7개가 전부 `valueOf` 로 대소문자를 흡수하는 것을 확인해 기각했고(관찰 가능한 효과 없음), permission 게이트·디자이너 SQL 분기·CI 는 보호 구역/M+ 라 또 미뤘다.
- 추측으로 적은 것: 실패 모드를 실행으로 재현하지 못했다(정찰은 테스트를 못 쓴다). `_offset` 이 `sanitizeIdentifier` 정규식을 통과해 WHERE 에 들어가는 것까지는 소스로 확실하지만, H2 가 내는 예외 타입·메시지와 최종 상태코드 500 은 추론이다. SQL 기반 목록(`sqlExecutor.executeQuery`)이 여분 파라미터로 깨지는지도 미확인 — 수용 기준 4로 넘겼다.
- 구현자가 조심할 것: `_limit` 상한에 `MAX_PAGE_SIZE`(1000)를 재사용하면 5000 을 요청하는 차트·피벗·트리가 부분 집계로 바뀌어 다른 버그가 된다. `@RequestBody` 맵을 제자리에서 변형하지 말고 복사할 것. `EntityService.list`·`RecordSet`·템플릿 7개는 건드리지 말 것 — 프로덕션 파일은 `NexaUiService.java` 1개로 끝난다.
- [러너 15:57] scout done — `POST /api/v1/builder/lists/{listId}/data` 가 `_offset`/`_limit` 를 컬럼 필터로 취급해 런타임 대체 뷰 7종(칸반·차트·�

## 구현 노트
- 무엇을/왜: `NexaUiService.listData` 가 `@RequestBody` 맵을 그대로 `EntityService.list` 의 filters 로 넘겨 `WHERE _offset = ? AND _limit = ?` 를 만들던 것을, 복사본에서 `_offset`/`_limit` 만 걷어내 실제 offset/limit 으로 넘기게 고쳤다(`EntityCrudController:53-59` 와 같은 계약). 프로덕션 파일 1개, 신규 테스트 1개(6건).
- 확신 없는 곳: (1) `_limit` 에 상한을 두지 않았다 — 인증된 호출자가 `_limit: 10000000` 을 보내면 그만큼의 LIMIT 가 나간다(수정 전에는 항상 page_size 로 묶여 있었다). 차트·피벗·트리가 5000 을 요청하므로 1000 캡은 부분 집계 버그가 되고, 5000+ 임의 상수는 테스트로 고정할 수 없어 무제한 + 주석 쪽을 택했다. 이 트레이드오프는 재검토 대상이다. (2) 오프셋 테스트는 `SELECT *` 에 ORDER BY 가 없어 행 **개수**만 단정한다(2건/1건) — 어느 행이 오는지는 단정하지 않았다. (3) `_offset`/`_limit` 에 숫자가 아닌 값(`"abc"`)이 오면 기존 `intOr` 헬퍼가 조용히 fallback 으로 떨어진다 — 테스트 없음.
- 일부러 하지 않은 것: `/api/v1/data/lists/{id}` 의 `pagenum`/`pagesize` 계약과 합치지 않음(다른 URL·다른 이름). `runtime/*.html` 7개의 `catch (e)` 는 손대지 않음(범위 밖, 다음 회차 후보로 원장에 남김). `EntityService.list` 서명·`sanitizeIdentifier` 정규식·`RecordSet` 불변.
- 다음 역할이 조심할 것: 새 테스트는 H2 + 전체 Spring 컨텍스트 + MockMvc 가 필요하다(`sh ./gradlew --no-daemon test --tests 'com.nexabuilder.api.BuilderListDataPagingIntegrationTest'`, gradlew 실행 비트 없어 `sh` 필수). 픽스처는 정리하지 않는다(관례). `sqlBackedListAcceptsProtocolKeys` 는 수정 전에도 초록이었던 회귀 가드다.
- [러너 16:08] brief accepted — 채택 — 진단(같은 파라미터를 읽는 두 리더의 계약 불일치, `sanitizeIdentifier` 가 선행 밑줄 통과, 500 으로 접힘)이 실행으�
- [러너 16:08] verify passed — 검증 1개 통과 (auto)
