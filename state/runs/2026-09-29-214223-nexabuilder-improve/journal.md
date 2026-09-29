# 회차 노트 2026-09-29-214223-nexabuilder-improve — nexabuilder
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 21:42] base pinned — master@a3ca143
- [러너 21:42] autonomy release — 

## 정찰 노트
- 내보내기 3종이 같은 fallback 블록을 복제하고 있어 한 곳만 고치면 계약이 어긋나는 구조라, "같은 값을 읽는 경로가 둘 이상이면 함께 고친다" 는 이 저장소의 반복 교훈과 맞아 골랐다. 차트·피벗 catch 삼킴(M, 브라우저 검증 비용)과 5000행 잘림 안내(M, 설계 선행)는 한 세션 범위를 넘어 제쳤고, ScreenPermission 게이트는 보호 경로라 계속 보류.
- 확신 없는 곳: all-hidden 목록의 **실제 HTTP 응답을 이번 세션에 돌려 보지 못했다**(예산). 진단은 `resolveColumns`(:339)와 세 fallback(:114/:155/:242)을 직접 읽은 정적 근거뿐이다. 리스트 디자이너 UI 가 전 컬럼 Hidden 저장을 막는지도 미확인 — 저장소 시드로 테스트하면 우회된다.
- 구현자가 조심할 것: fallback 자체를 지우면 `columnsJson` 없는 목록(SQL 기반 다수)이 빈 파일이 된다 — 수용 기준 2 가 그 회귀 가드다. `createSafeSheetName`(:173)과 `_limit` 주입(:381)은 최근 2회차 머지분이니 건드리지 말 것.
- 빨간 테스트를 먼저 볼 것. 수정 전에도 CSV 가 이미 비어 있다면 과제가 성립하지 않으니 차선(휴지통 SQL 목록 `requireLiveList` 회귀 테스트)으로 전환.
- [러너 21:46] scout done — 모든 컬럼이 `hidden` 인 목록을 내보낼 때 CSV/XLSX/PDF 가 숨긴 컬럼을 첫 행 키 fallback 으로 다시 노출하는 것�

## 구현 노트
- 무엇/왜: all-hidden 목록의 내보내기가 `SELECT *` 로 새는 것을 막았다. `resolveColumns` 가 `List<ColumnSpec>` 대신 `ColumnPlan(visible, configured)` 를 반환하고, 세 엔드포인트에 복제됐던 첫 행 키 fallback 을 `applyFirstRowFallback` 헬퍼 하나로 합쳐 `configured` 면 건너뛴다. 프로덕션 파일 1개(`ListExportController.java`), 테스트 1개, 신규 6건.
- 확신 없는 곳: (1) `configured` 의 정의를 "columnsJson 이 비어 있지 않은 JSON 배열로 파싱됨" 으로 못 박았으므로 **파싱 실패는 여전히 조용히 전 컬럼 fallback** 한다 — 수용 기준 2 가 "지금과 똑같이" 를 요구해서 의도적으로 유지했지만 그 자체가 별개 결함일 수 있다(ideas.json 에 신규 항목으로 적었다). (2) 리스트 디자이너 UI 가 전 컬럼 Hidden 저장을 실제로 허용하는지는 브라우저로 확인하지 않았다 — 테스트는 리포지토리 시드로 만들었고 저장소·서비스 레벨에는 막는 검증이 없다. (3) 컬럼이 빈 XLSX 는 완전히 빈 시트를 내보낸다 — POI 로 다시 열리는 것은 검증했지만 Excel 실물로는 확인 못 했다. (4) PDF 는 기존 `columns.isEmpty()` 분기를 타므로 "행: N" 문구는 행 수를 그대로 찍는다(행 수 자체는 숨긴 값이 아니라 그대로 뒀다).
- 일부러 안 한 것: fallback 제거(컬럼 설정 없는 목록의 유일한 컬럼 소스), `createSafeSheetName` 호출과 `_limit` 주입(최근 2회차 머지분) 손대기, `NexaUiService`/`RecordSet` 등 공유 계약 확장, CSV 에 "숨김" 안내 문구 추가(계약 신설이라 범위 밖).
- 다음 역할 주의: 이 테스트는 실제 H2·POI·MockMvc 로 돌며 컨텍스트를 공유한다(`@DirtiesContext` 없음). `csvExportWithoutColumnsJsonStillFallsBackToFirstRowKeys` 는 헤더를 소문자로 정규화해 비교한다 — 드라이버가 `ID,NAME` 을 주기 때문이며 H2 외 DB 로 바꾸면 이 단언이 먼저 흔들린다. 검증: 단일 클래스 19건 통과, `cleanTest test` 615건·0 fail·0 skip(5분36초), `bootJar -x test` 성공. 커밋 0421f6d.
- [러너 21:58] brief accepted — 채택 — 진단(세 곳에 복제된 fallback 이 hidden 필터 결과의 빈 리스트를 `SELECT *` 로 되살림)과 지정한 설계(`ColumnPlan` record, 
- [러너 21:58] verify passed — 검증 1개 통과 (auto)

## 비평 노트
- 확인: `git diff master...HEAD` 전체(프로덕션 1·테스트 1), `resolveColumns`/`applyFirstRowFallback` 의 6개 분기(null·blank·[]·파싱실패·부분숨김·전부숨김), 세 엔드포인트의 fallback 호출 위치, `runtime/list.html:809` 의 화면 측 hidden 계약. 실제 실행: `ListExportIntegrationTest` 19건 통과/실패0(18초).
- 원장에 `- 실패 재현:` 줄이 없어 diff 로 직접 확인함 — master 는 all-hidden 에서 첫 행 키로 헤더+데이터를 쓰므로 `isEqualTo(BOM)` 은 수정 전 확정 실패. 테스트는 바뀐 경로를 실제로 지난다. 회귀 가드 3건(no columnsJson·`[]`·부분숨김)도 계약의 반대쪽을 잡는다.
- 못 본 것: 전체 스위트(단일 클래스만 실행), 리스트 디자이너 UI 가 전 컬럼 Hidden 저장을 허용하는지(브라우저 미확인 — 구현자 의심 (2) 그대로 남음), Excel 실물에서 빈 시트 열기(구현자 의심 (3)).
- 남는 우려: `configured` 가 '모든 항목의 field 가 blank' 인 깨진 설정에서도 fallback 을 끈다(도달 경로는 안 보임). columnsJson 이 없는 목록의 `SELECT *` 내보내기는 기존 위험으로 그대로 남는다 — 민감 필드 개념이 이 경로엔 없다.
- 판정 approve / risk low / blocking 없음. 보안·법무 차단 사유 없음(노출을 좁히는 변경, 개인정보·의존성 변화 없음). 릴리즈 노트에 "all-hidden 목록은 이제 빈 파일" 을 반드시 넣을 것.
- [러너 22:01] review approved — 리뷰 승인 (risk=low)
- [러너 22:01] pr created — https://github.com/hkjang/nexabuilder/pull/37
- [러너 22:02] ci failed — 성공이 아닌 검사: test + bootJar=failure
