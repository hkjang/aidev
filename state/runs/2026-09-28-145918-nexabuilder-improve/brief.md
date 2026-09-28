# 과제서 — 2026-09-28-145918-nexabuilder-improve

- 과제: CSV/XLSX/PDF 내보내기가 그리드 한 페이지(기본 100행)만 내려주고 조용히 잘리는 것을 고친다 (가치 4 / 위험 2 / 작업량 S)

- 왜: 런타임 목록의 "현재 필터 결과 CSV로 내보내기" 버튼(`templates/runtime/list.html:1039 downloadExport`)은 `filterValues()` 만 쿼리스트링에 담고 `_offset`/`_limit` 를 보내지 않는데, 세 내보내기 엔드포인트가 그 쿼리스트링을 그대로 `NexaUiService.listData` 로 넘기기 때문에 `_limit` 이 없어 `listData:174-179` 의 fallback(`list.page_size`, null 이면 100)으로 떨어진다. 즉 5,000행 목록을 내보내도 파일에는 첫 100행(또는 목록의 page_size 만큼)만 들어가고, 사용자에게는 잘렸다는 표시가 전혀 없다 — 보고용 파일이 조용히 틀린 데이터가 되는 것이라 데이터 손실급이다. 고치면 내보내기가 필터 조건에 맞는 전체 행(상한 `NexaUiService.MAX_FEED_ROWS`=5000)을 담는다.

- 수용 기준:
  1) `page_size` 가 작게(예: 5) 설정된 목록에 12행이 있을 때 `GET /api/v1/lists/{listId}/export.csv` 응답의 데이터 행이 12줄이다(헤더 제외). **수정 전에는 5줄**이므로 테스트를 먼저 돌려 빨간 것을 확인할 것.
  2) 같은 목록의 `export.xlsx` 도 같은 12행을 담는다 — 세 엔드포인트가 `readQueryParams` 하나를 공유한다는 것이 관찰로 확인돼야 한다(PDF 는 바이트 검증이 어려우니 생략 가능).
  3) 쿼리스트링에 `_limit` 을 명시하면(`?_limit=3`) 그 값이 그대로 존중돼 3행만 나온다 — 기본값 주입이 명시 요청을 덮어쓰지 않는다.
  4) 기존 필터 계약이 유지된다: `?name=홍길동` 같은 컬럼 필터는 그대로 동작하고(기존 `ListExportIntegrationTest` 5건 초록), 휴지통 목록은 여전히 400 이다.

- 건드릴 파일 (프로덕션 1개):
  - `src/main/java/com/nexabuilder/api/controller/ListExportController.java:readQueryParams` (현재 355-361 근처) — 쿼리스트링 맵을 만든 뒤 `_limit` 키가 없을 때만 `NexaUiService.MAX_FEED_ROWS`(public static, 5000)를 넣는다. 세 엔드포인트(`exportCsv:96`, `exportXlsx:151`, PDF)가 모두 이 메서드 하나를 지나므로 여기 한 곳만 고치면 세 경로가 같은 값을 읽는다 — 호출부 세 곳에 각각 넣지 말 것(같은 값을 읽는 리더가 갈라지는 것이 이 저장소에서 6회차 연속 반복된 실패 패턴이다). `MAX_FEED_ROWS` 를 복제한 새 상수를 만들지 말고 `NexaUiService.MAX_FEED_ROWS` 를 직접 참조할 것.
  - 메서드 javadoc(현재 "Servlet query params → Map<String, Object>, single values only.")에 "내보내기는 페이징 UI 가 없으므로 `_limit` 기본값을 피드 상한으로 둔다, 호출자가 명시하면 존중한다" 를 남길 것.
  - `src/test/java/com/nexabuilder/api/ListExportIntegrationTest.java` — 기존 `seed()` 는 1행·`pageSize` null 이라 이 결함을 못 잡는다. `pageSize` 와 행 수를 받는 오버로드(예: `seed(int pageSize, int rowCount)`)를 추가하고 기존 `seed()` 는 그것에 위임해 기존 5건의 기대값을 바꾸지 말 것. 신규 테스트 3건(기준 1·2·3)을 같은 파일에 추가한다. `@DirtiesContext` 는 붙이지 말 것.

- 검증 명령:
  - 수정 전 빨간 것 확인: `sh ./gradlew --no-daemon test --tests 'com.nexabuilder.api.ListExportIntegrationTest'`
  - 전체: `sh ./gradlew --no-daemon cleanTest test` (수 분, 현재 598건 통과 기준 → 601건)
  - 패키징: `sh ./gradlew --no-daemon bootJar -x test`
  - `gradlew` 가 100644 라 `sh ./gradlew` 로 실행해야 한다(chmod 하지 말 것 — 작업 트리 변경이 러너에 버려지고 CI 는 자체 chmod 스텝을 쓴다).

- 위험과 피할 것:
  - `NexaUiService.listData` 를 건드리지 말 것. `_limit` 의 fallback(`page_size`→100)은 `POST /api/v1/builder/lists/{id}/data` 의 그리드 페이징 계약이고, 거기 기본값을 5000 으로 올리면 런타임 그리드가 매 요청마다 전체 표를 읽는다. **내보내기 쪽 기본값만 바꾼다.**
  - `MAX_FEED_ROWS`(5000) 상한 자체를 올리거나 없애지 말 것. 1b265b2 가 의도적으로 넣은 전표 읽기 방어이고, `EntityService.list` 가 모든 행을 `List<Map>` 으로 적재하므로 상한 제거는 OOM 경로를 다시 연다. 5000행 초과 목록의 내보내기는 여전히 잘리는데, 그것을 이번 회차에 해결하려 하지 말 것(스트리밍 페이지네이션은 별도 L 과제다). 다만 잘린 사실을 사용자가 알 수 있게 하는 것까지 넣고 싶다면 응답 헤더 한 줄(예: `X-Nexa-Export-Truncated: true`)로 제한하고, 그 헤더도 테스트로 고정할 것 — 템플릿 JS 는 건드리지 말 것.
  - PDF 경로는 `PdfPTable` 을 메모리에 다 쌓으므로 5000행에서 느려질 수 있다. 테스트 픽스처를 5000행으로 만들지 말 것(수용 기준 1 은 `pageSize=5` + 12행으로 충분히 증명된다).
  - `readQueryParams` 는 `_offset` 은 건드리지 않는다(기본 0 이 이미 올바르다). `_limit` 만 다룰 것.
  - 보호 경로 금지: `core/auth`·`core/session`·`infra/security`·`core/permission`, `db/migration`, `.github/workflows`. `resolveExportable` 의 권한/휴지통 판정 순서는 그대로 둘 것.
  - `RecordSet.getRows()` 는 `unmodifiableList` 다(2026-09-27 에 여기서 500 이 났다). 내보내기 코드에서 행 리스트를 정렬·수정하지 말 것.
  - 미확인: `list.page_size` 가 실제 운영 목록에서 보통 어떤 값인지(픽스처 기준으로만 확인), PDF 5000행의 실제 소요 시간.

- 차선 후보: `DataAdapterService.queryList` 의 `sqlId` 백엔드 목록에도 `requireLiveList` 회귀 테스트 추가 (가치 2 / 위험 1 / 작업량 S, 프로덕션 0줄). 기존 soft-delete 테스트 4건이 전부 `entityId` 백엔드라 SQL 백엔드 커버가 0 이다. `SqlBackedListDefinitionIntegrationTest` 의 `sqlId` 픽스처를 재사용해 `deleted_at` 만 찍고 `GET /api/v1/data/lists/{id}` 가 400 인지 확인한다.
