# PR #3 fix summary (commit 6de62c2)

- 재현: 테스트로 확인 — 익명 `DELETE /momento/api/admin/users/1` 이 수집기에 `DELETE /api/admin/users/1` 로 도달(200)했고, `PUT /api/settings {"tracking_allowed_hosts":"'unsafe-inline' * data: x;default-src"}` 가 200 으로 저장됐습니다.
- 프록시(internal/mcp/tracking.go): `path.Clean(TrimPrefix(path, /momento))` 가 `/tracker.js`(GET/HEAD) 또는 tracker.js 가 data-endpoint 에서 만드는 수집 끝점 `/collect/v1/events`(POST, contract-version 1 — Momento sdk `collectorURL()` 기준)와 정확히 일치할 때만 넘기고 나머지는 404. base path 결합(pr.SetURL)은 그대로 두고, 상수는 `tracking.ProxyTrackerPath`/`ProxyCollectPath` 로 두어 스니펫과 같은 값을 씁니다.
- 허용 출처(internal/tracking/tracking.go + internal/meta/settings.go): `tracking.ValidateAllowedHosts` 를 `SetAllowedHosts` 의 Validate 로 달아 토큰마다 `https?://host[:port]` 또는 `https?://*.host` 만 통과(url.Parse 로 Host 검사, 경로·쿼리·userinfo 거부), 따옴표 키워드·단독 `*`·`data:`/`blob:` 등 스킴 키워드·`;`·공백 포함 토큰은 ErrInvalidSetting(400). 이미 저장된 값도 헤더에 못 실리도록 `PolicySources` 가 출처 형식이 아닌 토큰을 건너뜁니다.
- 테스트: `TestTrackingMomentoProxyEndToEnd` 에 허용 목록 밖 경로/메서드 10종이 404 이고 수집기에 닿지 않음 + base path(`…/base/`) 유지 확인, `TestTrackingSettingsValidation` 에 거부 케이스 20종과 `https://*.x.example.com` 등 통과·CSP 반영 확인, `internal/tracking` 에 `TestValidateAllowedHosts`/`TestPolicySourcesDropNonOrigins` 추가.
- 검증: `go build ./...`, `go vet`(수정 패키지), `go test ./...` 전부 통과. 문서(admin_guide)의 "/momento/* 전달" 표현은 허용 부분집합을 넘긴다는 뜻으로 여전히 맞아 손대지 않았습니다.
