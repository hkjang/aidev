# appstore 자율 개선 기록

## 2026-09-02
- 선택: 로그인 returnTo의 open redirect 우회 차단 (가치 5 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `auth.SafeReturnTo`와 프런트엔드 `safeReturnTo`가 선행 슬래시 하나만 검사해서 `/\evil.test`, `/<TAB>/evil.test` 같은 값이 통과했고, 브라우저가 backslash를 slash로 정규화하고 tab·CR·LF를 제거하면서 `//evil.test`(protocol-relative)로 해석되어 OIDC 로그인 후 외부 리디렉션이 가능했다. 양쪽 모두 backslash와 제어문자를 거부하고 Go 쪽은 `url.Parse`로 scheme·host·userinfo가 없는지 재확인하도록 고쳤으며, 프런트엔드 guard는 `web/src/lib/utils.ts`로 옮겨 단위 테스트를 붙였다. 검증: `go test -race`(전체 패키지), `npm test`, `npm run lint`, prettier check, `npm run build` + `check-offline-assets.sh` / `check-env-contract.sh` / `check-docs.sh` 모두 통과. 커밋 53a7c5d.
- 보류 아이디어:
  - `AccessLog`의 `responseRecorder`가 `http.Flusher`·`http.Hijacker`를 직접 구현하지 않음(현재는 `Unwrap()` + `http.NewResponseController` 덕에 SSE 동작). 가치 2 / 위험 2 / S
  - `internal/httpapi` 커버리지 5.5% — DB 없이 테스트 가능한 `SPAHandler`, middleware, `DecodeJSON`, `parseID` 등에 단위 테스트 추가. 가치 3 / 위험 1 / M
  - `httpapi.validHTTPURL` 강화(호스트의 공백·제어문자 거부, IDN/포트 검증). 가치 2 / 위험 2 / S
  - rate limiter의 `Retry-After: 60` 고정값을 현재 분 윈도우 잔여 시간으로 계산. 가치 2 / 위험 1 / S
  - `httpapi.decodeSetting`이 Unmarshal 실패 시 부분 변형된 fallback을 반환할 수 있음 — 임시 값으로 디코드 후 성공 시에만 대입. 가치 2 / 위험 1 / S
- 릴리즈: v2.1.1 (2026-09-02)

## 2026-09-02
- 선택: 앱·사용자 검색의 LIKE 메타문자 이스케이프 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `store.ListApps`와 `store.ListUsers`가 검색어를 그대로 `ILIKE '%' || $n || '%'` 패턴에 넣어 사용자가 입력한 `%`, `_`, `\`가 와일드카드로 해석됐다. `%` 한 글자만 검색하면 전체 목록이, `100%`나 `a_b`를 검색하면 무관한 행이 함께 반환됐다(공개 카탈로그 검색, 관리자 앱·사용자 검색, MCP search tool 모두 영향). `internal/store/repository.go`에 `likePattern` 헬퍼를 추가해 메타문자를 이스케이프하고 각 `ILIKE` 절에 `ESCAPE '\'`를 명시했으며, 앱 검색은 `plainto_tsquery`용 원문 파라미터와 ILIKE용 패턴 파라미터를 분리했다. 검증: `likePattern` 단위 테스트 추가 + `go test ./...`(전체 통과), `go test -race ./internal/store/... ./internal/httpapi/...`, `go vet`, `gofmt -l`, `check-env-contract.sh`, `check-docs.sh` 통과. 통합 테스트(DSN 필요, CI에서는 skip)에도 와일드카드 검색이 0건을 반환하는지 확인하는 단언을 추가. 커밋 96e2daf.
- 보류 아이디어:
  - `internal/httpapi` 커버리지 5.5% — DB 없이 테스트 가능한 `SPAHandler`, middleware, `DecodeJSON`, `parseID` 등에 단위 테스트 추가. 가치 3 / 위험 1 / M
  - rate limiter의 `Retry-After: 60` 고정값을 현재 분 윈도우 잔여 시간으로 계산. 가치 2 / 위험 1 / S
  - `httpapi.decodeSetting`이 Unmarshal 실패 시 부분 변형된 fallback을 반환할 수 있음 — 임시 값으로 디코드 후 성공 시에만 대입. 가치 2 / 위험 1 / S
  - `httpapi.validHTTPURL` 강화(호스트의 공백·제어문자 거부, 포트 검증). 가치 2 / 위험 2 / S
  - `ai.sanitizeProviderError`가 512 byte로 자를 때 UTF-8 rune 경계를 무시해 한국어 provider 오류 메시지 끝이 깨짐. 가치 2 / 위험 1 / S
- 릴리즈: v2.1.2 (2026-09-02)

## 2026-09-03
- 선택: maxOutputTokens 미설정 provider의 `max_tokens: 0` 전송 버그 수정 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `store.validateAIProvider`와 `httpapi.minimumPositive`는 `maxOutputTokens = 0`을 "AppStore 측 출력 제한 없음"으로 취급하는데, `ai.Stream`은 그 값을 그대로 upstream payload의 `"max_tokens"`에 넣어 OpenAI 호환 서버가 거부하는 `max_tokens: 0`을 보냈다(해당 provider로는 모든 AI 요청이 HTTP 400). `input.MaxTokens > 0`일 때만 필드를 포함하도록 고쳤고, 함께 미뤄 뒀던 `sanitizeProviderError`의 512 byte 절단이 UTF-8 rune 경계를 깨뜨리던 문제도 마지막 불완전 rune을 잘라내도록 수정했다. 검증: payload를 캡처하는 httptest 기반 테이블 테스트와 rune 경계 테스트를 추가해 수정 전 코드에서 두 테스트가 모두 실패하는 것을 확인한 뒤 `go test -race`(전체 패키지), `go vet ./...`, `gofmt -l`, `go build ./cmd/server`, `check-env-contract.sh`, `check-docs.sh` 통과. Frontend는 변경하지 않아 npm 검사는 생략. 커밋 4450a58.
- 보류 아이디어:
  - `internal/httpapi` 커버리지 5.5% — DB 없이 테스트 가능한 `SPAHandler`, middleware, `DecodeJSON`, `parseID` 등에 단위 테스트 추가. 가치 3 / 위험 1 / M
  - `ai.consumeOpenAIStream`이 SSE 다중 `data:` 줄을 이어 붙이지 않아 한 이벤트를 여러 줄로 쪼개 보내는 provider에서 JSON 디코드 실패. 가치 3 / 위험 2 / M
  - rate limiter의 `Retry-After: 60` 고정값을 현재 분 윈도우 잔여 시간으로 계산. 가치 2 / 위험 1 / S
  - `httpapi.validHTTPURL` 강화(호스트의 공백·제어문자 거부, 포트 검증). 가치 2 / 위험 2 / S
  - `httpapi.SPAHandler.fmtInt`는 `strconv.Itoa` 재구현 — 표준 라이브러리로 교체. 가치 1 / 위험 1 / S
- 릴리즈: v2.1.3 (2026-09-03)
