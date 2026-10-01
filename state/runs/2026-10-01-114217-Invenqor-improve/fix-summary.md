# fix-summary

비평가의 지적 3건은 모두 사실이었고, 실제 postgres:17-alpine 으로 재현했다 — `urn:uuid:<id>` 는 assetID·source_ids 양쪽에서 **500 INTERNAL_ERROR**, `{<실제 id>}` 와 하이픈 없는 32자는 **201 로 split 이 실제 수행**(자산 생성·source 이동·audit 기록)되는데 SQLite 폴백은 같은 입력 모두 400 이었다.
원인은 `uuid.Parse` 를 모양 검사기로 쓴 것 — Parse 는 urn:·중괄호·하이픈 없는·대문자 형식을 모두 통과시키고, PostgreSQL 의 uuid 입력은 첫 형식만 22P02 로 거절하고 나머지는 조용히 정규화해 실제 행에 매칭한다.
수정: `canonicalUUID`(assets.go:648) 가 openapi 의 `format: uuid` 가 선언하는 36자 하이픈 형식만 받고(`len==36` + Parse) `parsed.String()` 으로 소문자 정규화해 돌려주며, splitAsset 은 이 정규형만 SQL·`asset_changes.after_json`·audit 에 넘긴다 — 구현자가 남긴 '대문자가 글자 그대로 저장된다' 문제도 같이 닫혔다.
테스트: 지적대로 Parse 가 받아들이는 형식을 **실제 존재하는** id 에 적용한 표 기반 테스트 2건(urn:·중괄호·하이픈 없는·not-a-uuid × assetID/source_ids = 8 서브테스트)과 대문자 정규화 1건을 추가했다. 고치기 전 이 테스트들은 두 방언에서 7건 실패했고, 기존 테스트는 하나도 지우거나 느슨하게 하지 않았다.
검증: `go test ./...`·`go vet ./...`·`gofmt -l .`·`go build` 가 SQLite 에서 통과하고, 같은 전체 스위트가 실제 PostgreSQL DSN 으로도 통과한다(httpapi 58s, 전체 패키지 green).
