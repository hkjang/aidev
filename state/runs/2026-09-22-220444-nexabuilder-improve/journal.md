# 회차 노트 2026-09-22-220444-nexabuilder-improve — nexabuilder
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 22:04] base pinned — master@ff270c2
- [러너 22:04] autonomy release — 

## 정찰 노트
- `DataAdapterService.queryList` 를 골랐다. 남은 soft-delete 구멍 중 유일하게 master 에 실재하고(63-65줄 직접 확인), 같은 줄에 `Integer pageSize` → `int` 언박싱 NPE 까지 붙어 있어 한 테스트로 둘을 증명할 수 있다. 권한 게이트(M·보호 구역)와 404 전환(기존 400 계약 파기)은 제쳤다.
- 2026-09-21 이 같은 대상을 골라 no-change 로 끝났지만 사람이 반려한 게 아니라 구현이 돌지 않은 것이고 코드는 그대로다. 범위를 NPE 까지 넓혀 관찰 가능한 결과를 늘렸다.
- 미확인: 승인 제약으로 이번엔 테스트를 한 번도 실행하지 못했다. "픽스처 pageSize 가 null → 지금 500" 은 소스 추론이며 실행 확인이 아니다. 구현자는 반드시 수정 전 빨강을 먼저 눈으로 볼 것.
- 조심할 것: `/api/v1/data/*` 는 성공 시 `{Rows,totalCount}`, 실패 시에만 `{success,message}` 봉투다. 단언을 섞으면 테스트가 엉뚱하게 초록이 된다.
- 조심할 것: `DataAdapterService` → `NexaUiService` 주입이 막히면 2줄을 복제하지 말고 판정을 별도 컴포넌트로 뽑을 것(한쪽만 넓히면 같은 값을 두 경로가 다르게 읽는다).
- [러너 22:08] scout done — `GET /api/v1/data/lists/{listId}` (jqxgrid 어댑터)가 ① 휴지통 목록의 행을 그대로 돌려주는 구멍을 막고 ② `pageSiz

## 구현 노트
- 과제서 그대로 채택. `queryList` 를 `uiService.requireLiveList` 로 바꿔 휴지통 목록 구멍을 막고(이제 listDefinition/listData/resolveExportable/queryList 네 경로가 같은 헬퍼 하나), `parse` 두 번째 인자를 `Integer` 로 넓혀 null `page_size` NPE(500)를 없앴다. 빈 순환은 없었고 컨텍스트는 그대로 뜬다. 커밋 5c65840.
- 과제서와 달랐던 점: 휴지통 케이스는 pageSize 가 null 이면 NPE 가 먼저 터져 200 이 아니라 500 이었다. 그래서 그 테스트만 `pageSize(10)` 으로 씨딩해 빨간 이유가 구멍(`expected:<400> but was:<200>`) 자체가 되게 했다. 두 빨강의 원인이 서로 달라 수정 2개를 개별 귀속할 수 있다.
- 확신 없는 곳: 신규 테스트 3건은 전부 **entityId 백엔드** 목록이다. `sqlId` 백엔드 목록도 판정이 분기보다 앞이라 코드상 막히지만 실행으로 확인하지 않았다. 또 `@WithMockUser(ADMIN)` 이라 이 URL 의 권한 공백(별개 보류 항목)은 전혀 덮지 않는다.
- 일부러 안 한 것: `pagenum=-1` / `pagesize=0` 하한 검증(차선 후보 — 계약 선택이 선행돼야 함), 권한 게이트, DB 기본값 50 과의 정렬(기존 폴백 100 을 유지해 두 경로가 같은 값을 같게 읽게 둠), `NexaListDef` 의 `@Builder.Default`.
- 다음 역할이 조심할 것: 이 테스트는 H2 스프링 컨텍스트가 필요하고(`sh ./gradlew` — gradlew 는 100644), `@DirtiesContext` 는 붙이지 않았다. 성공 응답은 `{Rows,totalCount}`, 오류만 `{success,message}` 봉투이므로 단언을 섞으면 엉뚱하게 초록이 된다.
- 검증: 단독 3건(수정 전 500/200 빨강 → 수정 후 초록), `cleanTest test` 578건 통과·0 skip·0 실패, `bootJar -x test` 성공.
- [러너 22:20] brief accepted — 채택 — 근거 3가지(`queryList:63-65` 가 deletedAt 미검사, 같은 줄 Integer→int 언박싱, 성공/실패 봉투가 다름)가 코드와 전부 일
- [러너 22:20] verify passed — 검증 1개 통과 (auto)

## 비평 노트
- 확인함: 수정 전 코드로 되돌려 재실행 → 2건 빨강(500 NPE `getPageSize() is null`, 400 기대에 200 = 휴지통 행 유출), 수정 후 3건 초록(17초). 실험 후 파일 복원·트리 clean. `explicitPageSizeIsHonored` 는 수정 전에도 통과하는 회귀 가드일 뿐이다.
- 확인함: `requireLiveList` 는 deletedAt 만 보므로 active=false 동작은 안 바뀐다. `queryList` 호출자는 DataAdapterController 하나, URL 소비자는 nexa-data.js 런타임 그리드뿐 — 휴지통 목록을 일부러 보여주던 화면 없음. 복원은 deletedAt=null 이라 다시 열린다.
- 못 본 것: sqlId 백엔드는 실행 검증 없이 소스로만 확인(게이트가 분기보다 앞). 전체 테스트 스위트는 재측정하지 않고 구현 노트의 578건 기록을 신뢰했다.
- 승인이어도 남는 우려(릴리즈 노트용): `/api/v1/data/lists/{listId}` 는 SecurityConfig 의 `anyRequest().authenticated()` 만 걸려 목록 단위 권한이 없다(기존 IDOR, 이번 diff 가 만든 것 아님). 그리드 데이터는 PII 마스킹을 타지 않는다(마스킹은 폼 레코드 전용). pagesize 하한 검증도 여전히 없음.
- 판정: approve / risk low / blocking 없음. 마이그레이션·외부 상태 변경 없어 revert 로 완전 복구.
- [러너 22:24] review approved — 리뷰 승인 (risk=low)
- [러너 22:24] pr created — https://github.com/hkjang/nexabuilder/pull/29
- [러너 22:33] ci passed — 검사 2개 모두 success
- [러너 22:33] merge done — 5c65840
