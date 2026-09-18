# fix-summary (ea4644a, PR #125)

- 문제 1(로그 미고정): `mcpChallenge` 의 `slog.Warn("mcp oauth token refused", "detail", …)` 과 `fromOAuthToken` 의 `reason` Warn 을 어느 테스트도 읽지 않았다. → `captureLog`/`refusalLogged` 헬퍼(documentserve_test.go 의 SetDefault 패턴)를 두고 `TestOnlyATokenMintedForThisServerOpensMCP` 거부 표의 모든 행(만료 "token is expired"·발급자 "different provider"·HS256/none "unexpected signature algorithm"·nbf "before the nbf"·다른 realm·typ·cnf·sub)에서 `msg="mcp oauth token refused"` + 라이브러리 원문 + 응답의 `X-Request-ID` 와 같은 `request_id=` 를 단언. `TestARefusedMCPClientIsToldWhereToSignIn` 은 스위치 켜짐+Issuer 비움에서 `reason="mcp.oauth.enabled is on but oidc.issuer is empty"` 를, 스위치 꺼짐에서는 로그 없음을 단언.
- 확인: 두 Warn 을 각각 지운 변이체로 돌려 각각 10건/1건 실패 → 원복 후 통과.
- 문제 2(문서≠코드): ADMIN_GUIDE.md 거부 표의 `로그인이 필요합니다` 행을 "스위치 켜져 있는데 Issuer 비어 있을 때만 로그가 남고, 꺼진 배포는 조용히 평소 401" 로 좁힘(코드 변경 없음). PDF 는 공용 md2pdf 로 재생성.
- 문제 3(alg=none 미검증): `signJWT` 에 `none` 분기(서명 없음 `a.b.`) 추가, 거부 표에 "빈 서명"(looksLikeJWT 탈락 → 평소 401, 로그 없음)·"가짜 서명"(go-jose 가 `unexpected signature algorithm "none"` 로 거부 → 401 "서명", 로그 있음) 두 행 추가. 문서의 `HS*`·`none` 문장 유지.
- 검증: gofmt/vet 깨끗, 세 DSN(docker postgres:16-alpine) 으로 `go test ./internal/... ./cmd/... -count=1` 전부 ok, gate.py secrets clean.
