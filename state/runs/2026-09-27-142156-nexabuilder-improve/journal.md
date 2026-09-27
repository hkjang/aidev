# 회차 노트 2026-09-27-142156-nexabuilder-improve — nexabuilder
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 14:22] base pinned — master@54a83f1
- [러너 14:22] autonomy release — 

## 정찰 노트
- 보류 아이디어 1순위(SQL 백엔드 컬럼 키 대소문자)를 골랐다. 권한 게이트(3/4/M)는 permission 보호 구역이라 8회 연속 미뤘고, executeSqlBacked 의 자바 페이징(3/3/M)은 SqlExecutor 페이징 API 설계가 선행돼야 하며 이번 과제와 같은 메서드를 건드리니 순서상 뒤다. 디자이너 SQL 분기는 JS 가 커서 M 초과 위험.
- 결정적 근거는 저장소가 이미 같은 문제를 두 곳에서 방어한다는 것이다 — ListExportController.lookup:366-371 이 exact→UPPER→lower 로 찾고 NexaUiService.maskRecordFor:860-862 이 "JDBC 드라이버는 컬럼 케이스가 다르다" 주석과 함께 세 변형을 치환한다. 즉 내보내기는 SQL 기반 목록에서도 값을 찾는데 어댑터만 보정이 없다. 기존 lowerCaseKeys 헬퍼 재사용으로 프로덕션 파일 1개.
- 추측으로 적은 것: H2 가 이 픽스처에서 실제로 대문자 라벨(ID/NAME)을 주는지 실행 미확인 — 정찰 단계에서 gradlew 가 샌드박스 승인에 막혔다(프로필 예고대로). 정황 3개는 과제서에 적었다. 런타임 화면의 빈 셀도 브라우저 확인 없음, 서버 응답 키 불일치까지만 확인.
- 구현자가 조심할 것: 수용 기준 4) 대로 **먼저 빨간 것을 보고** 고칠 것. 소문자 키 검증이 처음부터 초록이면 이 과제는 성립하지 않으니 차선(applySort 대소문자 무시)이나 3순위(sqlId soft-delete 테스트)로 갈 것. SqlExecutor.executeQuery 는 절대 건드리지 말 것(listData·내보내기·Groovy 공유). lowerCaseKeys 는 applySort 앞에 와야 정렬이 고쳐진다.
- [러너 14:26] scout done — SQL 기반 목록의 jqxgrid 어댑터 응답이 DB 라벨 대소문자를 그대로 내보내 그리드 셀이 비고 정렬이 조용히 �

## 구현 노트
- 무엇/왜: `DataAdapterService.executeSqlBacked` 가 `SqlExecutor` 원본 행을 그대로 `Rows` 로 내보내던 것을 기존 `lowerCaseKeys` 헬퍼에 통과시켰다(프로덕션 1파일, 한 줄 + 주석). 엔티티 백엔드가 이미 하던 것과 같은 정규화다.
- **과제서 진단이 틀린 부분**: 정렬은 "조용한 no-op" 이 아니라 **500** 이었다. `applySort` 가 `rows.sort(...)` 로 제자리 정렬하는데 `RecordSet.getRows():77` 은 `Collections.unmodifiableList` 를 준다 → `UnsupportedOperationException`. 즉 SQL 기반 목록은 헤더 클릭 한 번에 그리드가 죽었다. 한 줄 수정이 키 소문자화와 가변 복사본 확보를 동시에 하므로 범위는 그대로 두고 테스트 기대값만 실제 동작에 맞췄다.
- 확신 없는 곳: 따옴표 별칭(`AS "userName"`) SQL 정의가 있으면 이 URL 의 키가 `username` 으로 바뀌는 계약 변화가 실재한다. migrations grep 으로 저장소 안엔 없음을 확인했으나 **런타임에 사용자가 만든 `nexa_sql` 정의는 확인할 방법이 없다** — 커밋 메시지에 명시. 런타임 화면이 실제로 셀을 그리는지 브라우저 확인은 안 했다(서버 응답 키까지만).
- 일부러 안 한 것: `SqlExecutor`/`RecordSet` 미수정(`listData`·내보내기 3종·Groovy 가 공유). `POST /builder/lists/{id}/data` 의 같은 대문자 키 문제도 범위 밖으로 남김(ideas.json pending).
- 다음 역할 주의: 새 테스트 2건은 실제 H2·MockMvc 로 돌며 목이 없다(DB 필요). `applySort` 는 제자리 정렬이라 호출자가 소유한 리스트를 넘겨야 한다 — 주석으로 명시해 뒀다.
- [러너 14:36] brief accepted — 채택 — 지정한 파일·헬퍼 재사용·호출 순서(`lowerCaseKeys` → `total` → `applySort` → `applyPaging`)를 그대로 따랐고 수용 기준
- [러너 14:36] verify passed — 검증 1개 통과 (auto)

## 비평 노트
- 확인한 것: diff 전체, `DataAdapterService.executeSqlBacked`/`applySort`/`applyPaging`/`lowerCaseKeys` 실제 소스, `RecordSet.getRows():77`(unmodifiableList 확인), 원장 `- 실패 재현:` 두 출력이 픽스처 SQL(따옴표 없는 `SELECT id, name`)과 증상이 일치함, `applySort` 호출자가 165 줄 하나뿐임(grep), 저장소 내 따옴표 별칭 부재, `templates/runtime/list.html` 의 `c.field` → `dataField` 바인딩. 판정 approve / risk low / blocking 없음.
- 못 본 것: gradlew 재실행(원장의 592건 통과를 그대로 신뢰), 브라우저 렌더 확인, 런타임 사용자가 만든 `nexa_sql` 정의.
- 승인이어도 남는 우려 1 — 릴리즈 노트에 **계약 변화**를 적을 것: 따옴표 별칭(`AS "userName"`) + camelCase `columnsJson.field` 조합을 쓰는 기존 SQL 목록은 이 URL 에서 새로 셀이 빈다. 더 안전한 형태는 같은 메서드 149-152 줄이 파라미터에 쓰는 비파괴 방식(원본 키 유지 + 소문자 변형 추가)이었다.
- 우려 2 — `POST /builder/lists/{id}/data` 는 여전히 대문자 키라 이제 두 엔드포인트의 키 규약이 엇갈린다(수정 전엔 일관되게 깨짐). 다음 회차 우선.
- 우려 3(사소, 거절 아님) — `SqlBackedListDefinitionIntegrationTest.java:206-210` javadoc 과 프로덕션 주석 158 줄이 아직 "silent no-op" 이라는 정정된 진단을 담고 있다. 커밋 메시지는 정확하니 다음 손대는 사람이 주석만 실제 원인(500)으로 맞추면 된다.
- [러너 14:39] review approved — 리뷰 승인 (risk=low)
- [러너 14:39] pr created — https://github.com/hkjang/nexabuilder/pull/33
- [러너 14:48] ci passed — 검사 2개 모두 success
- [러너 14:48] merge done — 4988e27
- [러너 15:07] release published — v1.24.0
- [러너 15:08] assets verified — v1.24.0 자산 1개 (이전 v1.23.0: 1)
