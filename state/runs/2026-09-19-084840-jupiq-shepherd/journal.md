## 수리 노트

- 네 지적 모두 맞았다(Host→aud 허용, Discovery 실패 미캐시, 계정 조회 오류 원문 유출, request_id 미단언). 틀린 지적은 없었다. 커밋 a1dcba5.
- 고친 방법: 대상 검사는 `AudienceResource()`(설정값만, 비면 aud 비교 생략)로 분리하고 `Resource(r)`는 표시용으로만 둠; `mcpProviderEntry.err`로 30초 negative cache(잠금은 맵에만, 동시 성공 항목은 덮지 않음); 저장소 오류는 `refuse(...)`로 감쌈; 로그 테스트는 `request_id=<X-Request-ID>`가 같은 줄에 있는지 단언.
- 선택 근거: 비평이 제안한 두 안 중 "resource 필수화" 대신 "빈 resource면 audience 목록만"을 택했다 — Keycloak 26 기본(aud=account, azp=클라이언트) 호환 경로가 resource 없이도 성립하고, 기존 설정을 저장한 설치가 갑자기 Inactive로 떨어지지 않는다.
- 확신 없는 곳: (a) 계정 조회 오류 경로(#3)는 저장소를 끊을 수단이 없어 테스트가 없다. (b) 빈 resource일 때 거부 메시지가 Host 값을 매퍼 힌트로 내지 않고 "mcp.oauth.resource를 적으라"고 바뀌었다 — 운영자 안내로는 더 맞다고 보지만 문구 변경이다. (c) docs/ADMIN_GUIDE.pdf는 재생성하지 않아 .md와 어긋난다.

## 심사 노트
- 확인: 거절 3건 모두 고쳐짐 — 대상 검사는 `AudienceResource()`(설정값만)로, Host 유래 값은 Metadata/Challenge 표시용에만 남음; Discovery 실패 30초 negative cache(잠금은 맵에만, JWKS는 go-oidc가 Background+SafeHTTPClient 10s로 가져오고 대기는 요청 ctx를 따름); 저장소 오류는 `refuse()`로 감싸 클라이언트엔 고정 문구만.
- 테스트 실행: `go test -race` auth/api 단위 통과(FakeIDP 실제 기동, skip 아님), Postgres 16 임시 컨테이너로 `-run Integration ./internal/store ./internal/api` 통과(MCP 통합 8 서브테스트 포함). 수정을 되돌려 보니 Host 테스트는 panic(대상 통과→nil Store), negative-cache 테스트는 "cached failure not returned"로 실패 → 회귀를 잡는다.
- 보호 파일 항목: 기본값 꺼짐(Seed ON CONFLICT DO NOTHING, 문서 없으면 off), 공개 경로는 RFC 9728 메타데이터뿐이고 켜진 동안만 200, 계정 자동 생성·정지 계정 되살림 없음(GetOIDCUserBySubject는 조회만), OAuthScopes∩사용자 권한, 빈 scope는 거부, OAuth 주체는 /mcp 밖에서 원문 세션 검사로 떨어짐, 요청 ID는 미들웨어가 헤더에 심어 로그에 실림(테스트가 캡처·단언).
- 못 본 것: 계정 조회 저장소 오류(#3)는 테스트 없음(수리 노트가 밝힘, 코드 경로는 단순). ADMIN_GUIDE.pdf는 a1dcba5의 .md 한 행 변경을 반영하지 않음(과거 001e637에도 같은 전례).
- 권고 merge: 결함 없음. enabled=true + resource·audience 모두 빈 상태는 검증에서 막지 않지만 fail-closed(모든 토큰 거부, 안내 문구)라 결함으로 보지 않음.
