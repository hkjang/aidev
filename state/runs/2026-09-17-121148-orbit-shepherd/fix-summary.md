# PR #1 fix summary

- 문제: 조용한 SSO 시도(prompt=none)가 `oidcStart`의 Discovery 실패나 콜백의 코드 교환 실패·id_token 검증 실패·provisioning_disabled·account_disabled에 들어가면 그대로 JSON 오류(500/401/403)를 내서, auto_login이 켜진 상태에서 Keycloak 장애·미등록·비활성 사용자는 화면을 열 때마다 로그인 폼 대신 원문 JSON을 봤다. 지적 두 건 모두 코드로 확인함(auth.go 옛 237–240, 345–380).
- 고침: `oidcFailure(w, r, silent, status, code, message, err)` 헬퍼를 두어 `silent`면 `/login?sso=error`로 302 리다이렉트(경고 로그 남김), 아니면 기존처럼 JSON writeError/internalError. 시작(Discovery·state 봉인)과 콜백(설정·Discovery·교환·검증·nonce·provisioning·status·세션 발급)의 모든 실패 자리가 이 헬퍼를 거친다. `sso=error`는 프론트 `shouldAttemptSilentSso`가 이미 재시도 억제 표시로 읽고 LoginPage가 안내를 띄우므로 프론트 변경 없음.
- 테스트를 위해 `oidcStart`/`oidcCallback`을 설정 로딩 껍데기와 본체 `beginOIDC`/`completeOIDC`로 나눠 DB 없이 본체를 직접 호출할 수 있게 했고, 죽은 issuer·토큰 교환을 거절하는 가짜 제공자로 4개 테스트 추가(옛 동작으로 되돌리면 4개 모두 실패함을 확인).
- 검증: `go build ./... && go vet ./... && go test -race ./...` 통과, `web`에서 `npm run test -- --run`(104 passed) 및 `npm run build` 통과.
- 커밋 539b441 (rebase/amend 없음, push 안 함). 변경 파일: internal/server/auth.go, internal/server/auth_test.go.
