# PR 처리기 노트 2026-09-19-132229-orbit-shepherd — orbit PR #6
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## 수리 노트
- 지적 5개 모두 코드에서 확인됨(줄 번호는 비평의 389-407 이 실제 233-251 등으로 어긋났을 뿐 내용은 맞음). go-oidc v3.20.0 Provider.Verifier → remoteKeySet() 이 context.Background+client 를 쓰는 것도 소스에서 확인해 WithoutCancel 을 없앴다.
- 고친 방법: request_id 두 곳 추가; oauthProvider 를 락 안 맵 조회·락 밖 discovery·inflight 합치기·15s 실패 캐시로 재작성; 메타데이터 라우트를 `/*` 로; captureLogs 헬퍼 + 전체 라우터 로그 단언 테스트 + discovery 동시성 테스트 3개 + 라우트 테스트 2개. 커밋 486045d.
- 확신 없는 곳 1: 전체 핸들러 로그 테스트는 New() 가 settings 를 DB 에서 읽어야 하므로 mcpoauth_db_test.go 에 있고, CI(ci.yml) 는 ORBIT_TEST_DATABASE_URL 을 주지 않아 CI 에서는 skip 된다. 로컬 Postgres 로는 통과. 테스트 seam 을 프로덕션 코드에 넣지 않으려는 선택이었고, CI 에 Postgres 를 붙이는 것은 이 수리 범위 밖으로 봤다.
- 확신 없는 곳 2: 합쳐진 discovery 는 선두 요청의 ctx 로 돈다. 선두가 취소되면 대기자는 제 ctx 로 다시 시도하도록 했지만(테스트 있음), 선두의 취소가 곧 실패 캐시로 남지 않도록 `ctx.Err()!=nil` 만 기준으로 삼았다 — Client.Timeout 만료는 ctx 와 무관하므로 정상적으로 캐시된다.
- 확신 없는 곳 3: 메타데이터를 normalize 에서 /mcp 로 제한하는 대신 라우트를 넓혔다. 리버스 프록시가 prefix 를 벗기는 배치에서는 프록시도 `/.well-known/oauth-protected-resource/<prefix>/mcp` 를 이 서버로 보내야 한다 — 문서에는 적지 않았다.

## 심사 노트
- 확인: 거절 5건 모두 해소(request_id 두 곳, 락 밖 discovery+inflight+15s 실패 캐시, WithoutCancel 제거, 메타데이터 /* 라우트, 전체 라우터 로그 테스트). 격리 Postgres 로 `go test -race` 전부 통과(TestDB* 8개 포함).
- 변이 검사: 거부 로그에서 request_id 를 빼면 TestDBRefusedTokenIsLoggedWithCauseAndRequestID 가 7 케이스 실패 — 테스트가 변경을 진짜 고정한다.
- 보호 항목: 마이그레이션 추가만·기본 꺼짐, 공개 경로는 RFC 9728 메타데이터뿐(꺼지면 404), aud 는 Host 미사용, external() 게이트 2곳 전부 갱신, 빈 scope 교집합 거부, 로그·응답에 토큰 원문 없음.
- 못 본 것: CI 는 Postgres 가 없어 TestDB* 가 skip 됨(후속으로 CI 에 postgres service 권장); web 은 node_modules 없어 tsc 미실행(CI npm test 에 맡김); 프록시 prefix 배치 시 메타데이터 경로 안내가 문서에 없음.
- 권고: approve / merge, risk medium(인증 경로지만 기본 꺼짐·되돌리기 쉬움).
