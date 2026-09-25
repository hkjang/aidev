# 수리 요약 — sqlon PR #13 (커밋 ae5b989)

- 지적은 맞았다: `cacheKey(profile, sql, maxRows)` 가 `opts.Binds` 를 빼고 키를 만들어, 같은 SQL·다른 바인드 요청이 60초 동안 한 캐시 항목을 공유했다. 비동기(fresh=true)는 읽기만 건너뛰고 put 은 그대로라 남의 바인드 결과가 동기 요청에 반환됐다.
- 재현(RED): 표준 빌드에 없는 드라이버 이름 `godror` 로 바인드 값을 그대로 행으로 돌려주는 테스트용 드라이버를 등록하고 oracle 타입 프로파일을 만들어 실제 HTTP 로 확인 — `binds B got binds A's cached rows: echo="A" cached=true`, `binds D got the async job's binds-C rows: echo="C" cached=true`.
- 수정: `cacheKey(profile, sql, maxRows, binds)` 가 `json.Marshal(binds)` 의 SHA-256 을 키에 덧붙이고 `(key, ok)` 를 반환한다. get(:141)·put(:159) 이 같은 키를 쓰고, 인코딩 불가한 바인드는 ok=false 로 캐시를 아예 건너뛴다(잘못된 히트보다 미스가 안전).
- 검증(GREEN): `go test ./... -count=1` 전 패키지 ok(mcp 4.561s), `go vet ./...`·`go vet -tags integration ./test/integration` 통과, `go test ./internal/mcp -race` 통과. 같은 바인드 반복은 여전히 `cached:true` 이고 비동기가 넣은 항목을 같은 바인드의 동기 요청이 읽는 것까지 단언한다.
- 손대지 않은 것: 캐시 TTL·엔트리 수 정책, 비동기/동기 요청 구조체, CHANGELOG 는 CRLF 혼합이라 파이썬으로 3줄만 바이트 삽입(`git diff --numstat` = 3/0).
