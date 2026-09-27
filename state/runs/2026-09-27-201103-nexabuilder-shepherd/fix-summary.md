# fix-summary (2026-09-27 수리)

- 문제 1·2 (`_limit` 무제한, export GET 경로): `NexaUiService`에 `public static final int MAX_FEED_ROWS = 5000` 을 두고 fallback 계산 **뒤에** `limit = Math.min(limit, MAX_FEED_ROWS)` 를 넣었다. 클램프가 `listData` 안에 있으므로 `POST /builder/lists/{id}/data` 와 `GET /lists/{id}/export.{csv,xlsx,pdf}` 두 리더가 한 번에 닫힌다.
- 문제 3 (SQL 분기 계약): `sqlExecutor.executeQuery` 에 `_offset`/`_limit` 를 **정규화·클램프된 값으로 되돌려** 넘긴다. `NamedParameterJdbcTemplate` 은 `:_offset`/`:_limit` 를 파라미터로 파싱하므로(밑줄은 구분자가 아님) 자체 페이징 SQL 본문이 "No value supplied" 로 깨지던 것이 사라지고, 클램프도 그 경로에 적용된다.
- 테스트 4건 추가: `protocolLimitIsClampedToMaxFeedRows`, `theLargestRuntimeViewRequestIsNotClamped`(5000 은 온전히 통과), `selfPagingSqlBodyStillBindsTheProtocolParameters`(BuilderListDataPagingIntegrationTest), `exportQueryStringLimitIsClampedToMaxFeedRows`(ListExportIntegrationTest — export 경로 첫 페이징 테스트). 상한은 상수를 그대로 참조해 단언하므로 값이 드리프트할 수 없다. 대량 픽스처는 H2 `SYSTEM_RANGE` 단일 INSERT.
- 되돌려 확인: 두 수정을 제거하면 정확히 이 3개 테스트만 실패했다 — 클램프 제거 시 피드 5025행/CSV 5026줄(전량 반환), SQL 파라미터 제거 시 자체 페이징 목록이 500. 지적 3건 모두 실제였다.
- 검증: `sh ./gradlew --no-daemon cleanTest test` → **602 tests, 0 failures, 0 skipped** (BUILD SUCCESSFUL 5m28s), `sh ./gradlew --no-daemon bootJar -x test` → BUILD SUCCESSFUL. 커밋 `1b265b2` (브랜치 위 새 커밋, push 안 함).
