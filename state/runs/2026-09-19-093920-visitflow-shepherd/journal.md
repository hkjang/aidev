# journal — 2026-09-19 visitflow-shepherd (PR #18)

(앞선 역할의 저널이 없어 수리 에이전트가 새로 만듦. `$JOURNAL_FILE` 미설정.)

## 수리 노트
- 두 지적 모두 맞았음. Host/X-Forwarded-Host fallback → 위조 호스트용 토큰이 200, 그리고 `oidc.NewProvider`가 `oauthMu` 안에서 실행돼 동시 요청 4개가 모두 한 stall(6s)을 기다리는 것을 scratch 테스트로 재현.
- 고친 방법: `mcpOAuthConfig.Resource`를 설정(`mcp.oauth.resource` → `general.base_url`+`/mcp`)으로만 계산하고 `active()`에 포함, 저장 시 둘 다 비면 `mcp_oauth_incomplete`; `mcpResource(r)` 제거·`mcpOAuthInfo`/`mcpOAuthPrincipal`에서 `r` 제거, `mcpChallenge`는 `(ctx, w, path)`. 디스커버리는 per-issuer in-flight 채널 + 별도 goroutine(`context.Background()`+10s 클라이언트), 호출자는 `ctx.Done()`과 select. 커밋 715a242.
- 테스트 2개 추가(호스트 위조 거부 / 디스커버리 비직렬화·취소). 락을 다시 잡는 변형으로 후자가 실제로 실패(10s)하는 것 확인. `go test ./...`, `go vet`, `npm test`, `npm run build` 통과.
- 확신 없는 곳: `protectedResourceMetadata`는 `http.HandlerFunc` 서명상 `r`을 유지(컨텍스트만 사용) — 지적의 "drop r" 문구를 글자 그대로는 못 따름. 취소된 요청은 `oidc_discovery_failed` 401로 답하는데 클라이언트는 이미 떠난 뒤라 실질 영향 없음. `docs/ADMIN_GUIDE.pdf`는 재생성하지 않음.

## 심사 노트
- 확인: throwaway postgres:16 으로 `go test ./...` 전체·MCP 테스트 `-race`·`go vet` 통과. 잠금 안에서 discovery 하도록 되돌리면 TestMCPOAuthDiscoveryIsNotSerialized… 가 실제로 실패(9.99s 대기)함을 mutation 으로 확인. Host fallback 제거·enable 거부·위조 X-Forwarded-Host 401·메타데이터 404 를 진짜 핸들러로 확인. 마이그레이션은 INSERT … ON CONFLICT DO NOTHING, 기본 false.
- 거절(fix) 사유 1건: TestMCPOAuthRefusesBadTokensAndLogsWhichCheckFailed 가 로그의 원문 오류는 단언하지만 request_id 는 단언하지 않음. 스크래치 테스트로 request_id 가 실제로 비어 있지 않음(chi RequestID)은 확인했으므로 코드 수정 없이 테스트에 단언 한 줄만 더하면 됨(캠페인 규칙 2).
- 못 본 것: docs/ADMIN_GUIDE.pdf 내용(압축 스트림, pdftotext 없음) — c10f7b5 에서 만든 PDF 라 Host fallback 서술이 남아 있을 것으로 추정, 재생성 권고.
- 권고 근거: 프로덕션 결함은 못 찾았고 캠페인이 명시한 테스트 고정 규칙만 미충족 → reject/fix, risk medium(인증).
