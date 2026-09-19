## 수리 노트

- 맞았던 지적: 세 가지 모두 맞았다. Host 기반 aud 통과는 임시 테스트로 재현했고(refusal=nil, Scopes=[read mcp]), 뮤텍스 안 discovery·WithoutCancel·DefaultClient 무타임아웃·음성 캐시 없음, 거절 로그의 request_id 부재 모두 코드에서 확인했다. 틀린 지적은 없었다.
- 고친 방법: 리소스 식별자·aud 허용값·메타데이터 resource·도전 헤더 URL 을 전부 `cfg.Resource` 설정값에서만 만들고 `*http.Request` 를 검증기 시그니처에서 제거; 켤 때 Resource 필수(validate)·비면 꺼짐(active). discovery 는 뮤텍스 밖 goroutine + Background/10s 타임아웃 클라이언트 + 30s 음성 캐시 + 요청 ctx 취소 존중. `refuseMCPToken` 두 분기에 `request_id`. 테스트는 Host 조작 거부·음성 캐시 호출 횟수·뮤텍스 비점유·request_id 를 각각 고정했고, UI/e2e/문서의 "비우면 요청 주소로" 문구를 서버와 맞췄다.
- 지적과 다른 선택: "Resource 또는 Audiences 중 하나 필수" 대신 **Resource 필수**로 갔다. Audiences 만으로는 RFC 9728 필수 필드 `resource` 와 metadata URL 을 Host 없이 만들 수 없어서다 — 허용값을 설정에서만 만든다는 규칙은 그대로 만족한다.
- 확신 없는 곳: (1) e2e(`web/e2e/mcp-oauth.spec.ts`)는 Docker 가 없어 로컬에서 못 돌렸다 — 추가한 noResource 케이스와 기존 단언 순서는 서버 validate 순서에 맞췄지만 실행으로 확인한 것은 아니다. (2) `mcpOAuthHTTPClient` 를 패키지 전역 변수로 두었다(테스트에서 바꿀 필요는 없었음). (3) 음성 캐시 TTL 30s·discovery 타임아웃 10s 는 상수로만 두고 설정으로 열지 않았다.

## 심사 노트
- 확인한 것: `go vet`, `go test ./...`, `go test -race`(MCP 테스트 전부) 통과. active() 에서 Resource 검사를 지우는 변이로 TestProtectedResourceMetadata·TestHostHeaderDoesNotBecomeAcceptedAudience 가 실제로 실패함을 확인(변이 후 원복). verifyMCPAccessToken 에 *http.Request 가 없어 Host 기반 허용값이 구조적으로 불가능. go-oidc v3.15 소스에서 RemoteKeySet 이 WithoutCancel(ctx)+ClientContext 의 10s 클라이언트를 쓰는 것을 확인 — 코드 주석과 일치.
- 확인한 것: 마이그레이션은 ON CONFLICT DO NOTHING 추가만, 기본 `mcp.oauth.enabled=false`; 설정 저장은 단일 트랜잭션(전부 아니면 무); SettingsPage 삭제 줄은 라벨 삼항식 재들여쓰기뿐; username 도출 규칙이 웹 OIDC 로그인(auth.go:416-422)과 동일; 로그 캡처는 기존 slog TextHandler 패턴에 버퍼만 바꾼 것.
- 못 본 것: e2e(`web/e2e/mcp-oauth.spec.ts`)는 이 세션에도 Playwright/node_modules 가 없어 실행 못 함 — CI 의 e2e 잡이 PR 에서 돌린다. 단언 순서는 서버 validate 순서(issuer→mcp→resource)와 맞음을 읽어서 확인.
- 참고(결함 아님): account_not_registered 401 에 error="invalid_token" 도전이 붙어 클라이언트가 재로그인을 한 번 더 시도할 수 있음(MCP SDK 는 1회 재시도 후 오류). SQL 로 mcp.oauth.scopes 를 비우면 빈 범위=세션 동등 권한이지만 본인 계정 범위를 넘지 않음.
- 권고 근거: 이전 거절 3건이 모두 코드+테스트로 고정됐고, 캠페인 규칙(Host 허용값·뮤텍스 밖 discovery·request_id·빈 교집합 거부·게이트 전수 검색·전부 아니면 무·삭제 줄) 전부 충족.
