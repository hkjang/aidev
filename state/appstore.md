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
