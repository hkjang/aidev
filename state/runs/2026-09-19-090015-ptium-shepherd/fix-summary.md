# fix-summary — PR #25 (commit 1e18588)

- 문제: 두 로그 경로(server.go:457 의 /mcp 토큰 거부 WARN, authenticator.go:84-87 의 "switched on but cannot take tokens" WARN)를 고정하는 테스트가 없었음 — 지적 둘 다 맞음.
- httpapi: `TestARefusedTokensCauseIsLoggedUnderTheRequestIDTheClientGot` 추가. handoff_test.go 패턴(`slog.NewTextHandler(&lines, nil)`)으로 Bearer a.b.c 를 /mcp 에 보내고, `msg="authentication failed"` 인 **그 한 줄**에 `request_id=<body.requestId>`·`path=/mcp`·`error="sso token refused: audience"` 가 있음을 단언 (버퍼 전체 검색은 "http request" 줄의 request_id 에 속아 mutation 을 못 잡아 줄 단위로 좁힘). 원문 cause 가 body 에 안 나감도 단언.
- mcpoauth: fixture 에 `logs bytes.Buffer` + `Logger: slog.New(slog.NewTextHandler(&f.logs, nil))`. 스위치 off 로 토큰 보낸 뒤 경고 없음, `Policy{Enabled:true}` 로 실제 토큰 보내 Refusal(cause "no resource identifier") 과 경고+`resource identifier` 가 로그에 남음을 단언.
- 검증: 두 로그 줄을 각각 제거하는 mutation 에서 새 테스트가 실패, 복원 후 통과. `go test -race`(두 패키지)·`go test ./...`·`go vet ./...`·gofmt 모두 통과. 소스 코드는 변경 없음, 테스트 파일 2개만.
