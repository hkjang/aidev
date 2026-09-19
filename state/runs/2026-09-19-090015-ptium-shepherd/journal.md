## 수리 노트
- 지적 두 건 모두 맞았음: ssoServer() 의 DiscardHandler, fixture 의 nil Logger 로 어느 로그 줄도 테스트에 잡히지 않았음. 동작 자체는 맞아서 소스는 건드리지 않고 테스트 파일 2개만 고침 (커밋 1e18588).
- 고친 방법: httpapi 는 새 테스트 1개(strings.Builder 로거, /mcp 에 Bearer a.b.c), mcpoauth 는 fixture 에 bytes.Buffer 로거를 넣고 기존 switched-off 테스트의 Enabled=true 분기에서 실제 토큰을 보냄.
- 배운 것: 버퍼 전체에서 `request_id=` 를 찾으면 "http request" 접근 로그 줄이 대신 맞아 mutation 이 통과함 → `msg="authentication failed"` 줄만 뽑아 그 줄에서 단언하도록 좁힘. mutation 으로 두 테스트 모두 실패→복원 후 통과 확인.
- 확신 없는 곳: 없음. 단, `$JOURNAL_FILE` 이 비어 있고 run 디렉터리에 저널이 없어 이 노트를 journal.md 로 새로 만듦.

## 심사 노트
- 확인: 거절 사유 2건 모두 고쳐짐 — httpapi 테스트는 실제 Handler() 를 통과해 `msg="authentication failed"` 줄에서 request_id(=body.requestId)·path·원문 cause 를 단언하고, mcpoauth 픽스처는 bytes.Buffer 로거로 on/off 양쪽 경고 유무를 단언. 소스는 안 건드림(테스트 파일 2개만).
- 변이 검사 7건(request_id 제거, 경고 제거, aud 규칙 무력화, disabled 무시, azp 검사 제거, SSO scope 게이트 제거, cause 를 body 로 누출) 모두 테스트 실패 → 테스트가 실제로 물고 있음. go build/vet, -race 3패키지 통과.
- 보호 파일: 기본값 off(mcp.oauth.enabled=false), 새 공개 경로는 RFC 9728 메타데이터뿐이며 off 면 404, aud 허용값은 설정/PUBLIC_BASE_URL 만(Host 미사용), 계정 생성·role 승격·정지 계정 부활 없음(GetUserBySubject 조회만), 비밀값 누출 없음, 마이그레이션은 설정 기본값 추가만, JWKS fetch 는 refreshMu 뒤 별도 잠금이라 캐시 히트가 막히지 않음.
- 못 본 것: 실제 Keycloak 상대로는 안 돌림(fakeIdP 의 RSA 서명 토큰으로 대신). 동작 변화 하나 — OIDC 켜진 배포에서 웹 로그인 토큰을 /mcp 에 직접 쓰던 경우(무제한 scope 였음)는 이제 SSO 를 켜고 클라이언트를 허용 대상에 적어야 함; 세션·키는 그대로.
- 권고 merge: 실제 결함 없음. 사소한 드리프트(web grantableScopes 가 서버가 받는 api_keys:manage/mcp:use 를 빼고 있음, 토큰 scope 와 관리자 목록이 안 겹치면 관리자 목록 전체를 줌 — 천장은 넘지 않음)는 결함 아님.
