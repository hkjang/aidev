# fix-summary (commit 0cb4bd2)

- 문제 1: `mcpAuthenticate`가 SSO 토큰 거부(`mcpOAuthRefusal`)를 401로 내보낼 때 로그 호출이 없어 go-oidc 원인(서명/발급자/만료/nbf)·audience 거부 사유가 어디에도 남지 않았음(지적 맞음).
- 수정: `server.go` `errors.As(err,&refusal)` 분기에서 `s.logger.Warn("mcp oauth token refused", "error", refusal.detail(), "request_id", middleware.GetReqID(...), "path", ...)` 호출. `mcp_oauth.go`에 `detail()` 추가 — cause를 `cleanOIDCText`+`truncateRunes(400)`로 정리(iss 등 공격자 입력 인용 대비).
- 테스트: 통합 테스트의 `io.Discard`를 mutex 보호 버퍼(`serverLog`)+`slog.NewTextHandler`로 바꾸고, `refusedAndLogged` 헬퍼로 만료(`token is expired`)·nbf(`before the nbf`)·다른 issuer(`issued by a different provider`)·타 키 서명(`failed to verify signature`)·대상 거부(`audience [account] / azp`) 각각에서 로그 라인에 원인 문자열과 `request_id=<X-Request-Id>`가 있고 원인이 클라이언트 본문엔 없음을 단언. 로그 호출을 지우고 돌리면 5곳에서 실패함을 확인.
- 문제 2: `throttledTransport` 단위 테스트 부재 → `TestThrottledTransportLetsOneRequestThroughPerInterval` 추가(가짜 RoundTripper: 첫 요청 즉시 통과, 두 번째는 interval 이후 base 도달; 대기 중 ctx 취소 시 base 미호출 + `context.Canceled` 반환). security.md 문장은 유지.
- 검증: `make lint` 통과, Docker postgres:16 DSN으로 `go test -race -count=1 ./...` 전부 통과(internal/server 72s, 통합 테스트 실제 실행).
