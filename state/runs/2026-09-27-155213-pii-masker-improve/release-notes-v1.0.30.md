## `PII_MASKER_MAX_PAGES=0`(페이지 제한 없음)이 설정으로는 도달할 수 없던 문제 수정

`service.countPages`는 `MaxPages > 0`일 때만 페이지 수를 검사해 "0이면 페이지 제한 없음"을 이미 구현해 두었습니다. 그런데 그 값을 만드는 유일한 경로인 `config.Load`가 `PII_MASKER_MAX_PAGES`를 `envInt`로 읽었고, `envInt`는 `parsed <= 0`이면 조용히 기본값으로 되돌립니다. 그래서 운영자가 `PII_MASKER_MAX_PAGES=0`을 주어도 경고 없이 기본값 20이 적용되었고, 프로덕션에서 무제한 분기는 어떤 설정으로도 도달할 수 없었습니다.

- `PII_MASKER_MAX_PAGES` 호출 지점만 기존 헬퍼 `envNonNegativeInt`로 바꿨습니다. 이 헬퍼는 `JOB_RETENTION_HOURS`·`SYNC_QUEUE_WAIT_SECONDS`가 이미 "0 = 끔"으로 쓰고 있던 것이고, 왜 이 항목만 0을 허용하는지 코드에 남겼습니다.
- `envInt` 자체는 그대로 두었습니다. 나머지 4개 호출자(`MAX_FILE_SIZE_MB`·`MAX_CONCURRENT_JOBS`·`MAX_CONCURRENT_SYNC`·`DEFAULT_TIMEOUT_SECONDS`)는 0이 곧 서비스 중단이므로 계속 기본값으로 폴백해야 하며, 그 폴백이 유지되는 것을 `Load()`를 지나는 테스트로 못 박았습니다.
- 내장 Playground가 `/v1/config/public`의 `max_pages`를 그대로 찍어, 무제한으로 뜬 서버가 "최대 페이지 0"으로 읽히던 표시를 "제한 없음"으로 고쳤습니다. `max_pages`를 읽는 경로는 `service.countPages`와 이 화면 둘뿐이고 둘 다 처리했습니다.
- 검증은 `Load()` 경유 테이블 테스트(미설정→20, `"0"`→0, `"5"`→5, `"-1"`·`"abc"`→20)와, 프로덕션 배선을 끝까지 지나는 통합 테스트로 했습니다. 후자는 `PII_MASKER_MAX_PAGES=0`으로 `config.Load`→`app.New`→`Serve`를 올린 뒤 21쪽 PDF를 `POST /v1/jobs`로 올려 `202`와 `input.pages == 21`을 확인합니다. 두 테스트가 수정 없이는 실제로 실패하는 것까지 확인했습니다.

설정을 주지 않았을 때의 동작은 바뀌지 않습니다(기본값 20 유지). 페이지 제한을 끄려면 `PII_MASKER_MAX_PAGES=0`을 명시해야 하며, 이때는 업로드된 문서의 쪽수만큼 렌더 비용이 들어가므로 신뢰할 수 있는 입력에만 쓰는 것을 권합니다.
