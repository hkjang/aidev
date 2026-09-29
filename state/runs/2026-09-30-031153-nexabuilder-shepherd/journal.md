# PR 처리기 노트 2026-09-30-031153-nexabuilder-shepherd — nexabuilder PR #40
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-09-30-013237-nexabuilder-improve)
# 회차 노트 2026-09-30-013237-nexabuilder-improve — nexabuilder
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 01:32] base pinned — master@a3ca143
- [러너 01:32] autonomy release — 
- [러너 01:38] scout done — 수정 과제 — PR #39 의 CI 실패를 **증거로** 코드 원인에서 배제하고(워크플로 수정 금지), 그 자리를 `DataAda

## 구현 노트
- 무엇을 왜: `DataAdapterService.executeEntityBacked` 가 필터·`sortdatafield` 를 `WHERE`/`ORDER BY` 에 쓰기 전에 그 테이블의 실재 컬럼인지 확인해 아니면 버린다. `entityDataFields` 의 `SELECT * ... WHERE 1=0` + `PreparedStatement.getMetaData()` 프로브를 `probeColumns` 헬퍼로 뽑아 재사용했다. 프로덕션 1파일(커밋 a36eecc) + 신규 테스트 1파일 4건.
- 과제서 정정 2건: 실패 상태는 400 이 아니라 **500** 이었다(`BadSqlGrammarException` 은 `GlobalExceptionHandler` 400 분기에 안 걸린다). 그리고 과제서가 지정한 `DataAdapterIntegrationTest` 에는 `seed()` 픽스처가 없어(시드 USER 엔티티만 쓴다) `DataAdapterListSoftDeleteIntegrationTest` 픽스처 형태로 신규 파일을 만들었다.
- **확신 없는 곳·검증 못 한 것**: (1) CI 실패 로그 본문 — `gh` 미인증 + 비공개 저장소라 4회차째 **미확인**. 코드 원인 배제는 7초 실행시간과 "빌드·CI 파일 미변경" 두 근거뿐이다. (2) jqxGrid 23.x 가 그룹핑 없이 `groupscount` 를 실제로 보내는지 미확인 — 테스트는 "알 수 없는 파라미터 일반" 의 계약으로만 고정했고, 현실적 트리거는 죽은 컬럼 정렬 쪽이다. (3) 프로브 예외 경로(권한·드라이버)는 코드로만 대비했고 테스트로 돌려보지 못했다 — H2 에서 재현할 방법이 없었다. (4) `?sortdatafield=1;DROP` 같은 부적격 값은 이제 `sanitize` 전에 "미존재 컬럼" 으로 조용히 버려진다(전에는 400). 의도한 변화이나 계약 변경이다.
- 일부러 하지 않은 것: `.github/workflows/*`·`build.gradle.kts` 불간섭(완화 금지). `ListExportController` 미개방(PR #37·#38·#39 충돌). `sanitize()`·`RESERVED_PARAMS`·`executeSqlBacked` 불간섭. 알 수 없는 파라미터를 400 으로 거부하는 방향은 채택 안 함(이 URL 의 실패 봉투를 jqxgrid 가 못 읽는다).
- 다음 역할 주의: 신규 테스트는 실제 H2 + `SchemaDdlService`/`EntityService`/`NexaListRepository` + MockMvc 배선이 필요하다(목·대역 없음, `@DirtiesContext` 없음). `gradlew` 는 모드 100644 라 `sh ./gradlew` 로 호출. 검증 실행: `clean test` 613건 통과·0 skip(6분), `bootJar -x test` 성공(6초).
- [러너 01:51] brief accepted — 채택 — 지정한 파일 1개·`probeColumns` 헬퍼 재사용·"검사 대상이 있을 때만 프로브"·"프로브 실패 시 기존 동작"·`sanitize()
- [러너 01:51] verify passed — 검증 1개 통과 (auto)

## 비평 노트
- 확인함: 프로덕션 파일만 master 로 되돌려 신규 테스트 재실행 → 2건 실패(:175 의 status().isOk(), 500 증상 일치)·회귀 2건 통과. 테스트는 진짜로 바뀐 경로를 지난다. 복원 후 워크트리 클린.
- 확인함: filterscount 경로가 parse() 의 식별자 정규식을 안 거치지만, columns==null 일 때만 통과하고 그 경로에서 sanitize() 가 던지므로 인젝션 방어선은 그대로. `/api/v1/data/*` 에 서버측 행 범위 필터를 넣는 호출자가 없어 필터 무시가 권한을 넓히지 않음 → 보안·법무 차단 사유 없음.
- 못 봤음: 전체 테스트(613건) 재실행 안 함(구현 노트의 기록만 신뢰), CI 실패 로그 본문(4회차째 미확인 그대로), 비-H2 드라이버 실제 동작.
- 승인 후 남는 우려: (1) PostgreSQL 에서는 프로브 실패가 readOnly 트랜잭션을 오염시켜 javadoc 이 약속한 '기존 동작 폴백' 이 성립하지 않을 수 있다(실제 트리거 구성은 어려움). (2) 죽은 컬럼으로 필터를 걸면 '필터된 듯 보이는 전체 목록' 이 나온다 — 릴리즈 노트에 적을 것. (3) 필터·정렬 요청마다 프로브 왕복 1회 추가, 캐시 없음.
- [러너 01:54] review approved — 리뷰 승인 (risk=low)
- [러너 01:54] pr created — https://github.com/hkjang/nexabuilder/pull/40
- [러너 01:55] ci failed — 성공이 아닌 검사: test + bootJar=failure
