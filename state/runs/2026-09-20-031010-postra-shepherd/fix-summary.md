# fix-summary — PR #16 gosec 실패

- 문제: gosec 실패는 이 PR 의 결함이 아님. 로컬에서 CI 와 같은 명령(`gosec -severity medium -exclude-dir=scripts ./...`)을 돌려 재현 — 지적 2건은 `internal/application/mcp_oauth_proxy.go:47`(G101)·`internal/transport/mcpserver/oauth_proxy_http.go:381`(G705) 로 모두 base main@49edf02 의 코드이며, PR diff 는 `internal/adapters/smtp/client_test.go` 1파일뿐(gosec 은 `_test.go` 를 스캔하지 않음).
- main 은 이미 3f9eb61 `fix(ci): annotate gosec false positives in the MCP OAuth proxy` 로 두 줄에 `#nosec` 을 달아 고쳤고, 그 두 파일만 origin/main 것으로 바꿔 다시 스캔하면 Issues 0·exit 0 임을 확인(이후 원복, 트리 clean).
- 게다가 이 PR 의 테스트 파일은 이미 PR #17(bf4e5a9) 로 main 에 머지된 파일과 바이트 단위로 동일(`git diff origin/main HEAD -- client_test.go` 빈 출력)이라 PR #16 은 #17 에 완전히 대체됨.
- 조치: 커밋 없음. 무관한 파일(OAuth 프록시)에 nosec 을 다시 얹거나 main 을 머지해 빈 PR 을 만드는 대신, PR #16 은 #17 로 superseded 로 닫기를 권고. (굳이 초록을 원하면 origin/main 머지 커밋 하나로 충분 — diff 가 비게 되므로 의미 없음.)
