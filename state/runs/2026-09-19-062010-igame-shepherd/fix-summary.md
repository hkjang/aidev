# fix-summary

- 문제: `requireMCPAuth`(internal/api/mcp.go)가 `mcpRefusal`을 만나면 `Message`만 클라이언트로 보내고 `Cause`(go-oidc 검증 오류)를 로그에 남기지 않아, mcpoauth.go 주석과 ADMIN_GUIDE의 "서버 로그에 세부 원인이 남습니다"가 거짓이었음. 지적 맞음.
- 수정 (commit 9475a64): `refused && refusal.Cause != nil`이면 `s.logRequestError(r, err)` 호출 — 로그에 `msg="request failed" error="SSO access token is not valid …: oidc: token is expired (…)" path=/mcp` 형태로 메시지+원인이 남음. Cause 없는 거절(audience·ID 토큰·cnf·subject)은 메시지가 이미 전부를 말하므로 로그하지 않음(기존 동작 유지).
- 테스트: `TestMCPOAuthRefusesTokensItCannotTrust`에 slog 버퍼를 붙여 expired/nbf/다른 발급자/위조 발급자/HS256 각각 로그에 실제 go-oidc 원인 문구(`token is expired`, `before the nbf`, `failed to verify signature`, `unexpected signature algorithm`)와 `path=/mcp`가 남는지, Cause 없는 거절은 로그가 비는지 단언. 수정 전 코드로 돌리면 5개 케이스가 실패함을 확인함.
- 검증: `gofmt -l` 없음, `go vet ./...`, `go test -count=1 ./...` 전부 ok.
- 참고: 실패 출력에 보이던 `TestNonPagePathsGetANarrowPolicyAndNoSnippet`의 nil-DB 패닉 스택은 chi Recoverer가 잡아 stderr로 찍는 HEAD 기존 잡음(`-v`로 HEAD에서도 재현, 테스트는 통과)이며 이 지적과 무관해 손대지 않음.
