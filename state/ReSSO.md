# ReSSO 자율 개선 기록

## 2026-09-02
- 선택: JWKS Cache-Control이 writeJSON에 덮어써지던 문제 수정 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공 (커밋 35fd2d9)
- 요약: `jwks` 핸들러가 설정한 `Cache-Control: public, max-age=300`을 `writeJSON`이 `no-store`로 무조건 덮어써서, 코드는 캐시 가능하다고 말하는데 응답은 아니라고 말하는 상태였다(직접 재현해 확인). `writeJSON`의 `no-store`를 규칙이 아닌 기본값으로 바꾸고, JWKS의 max-age는 인스턴스 자신의 키 집합 캐시 수명인 `store.SigningKeyTTL`(30초)에서 가져오도록 했다 — 기존 300초는 그 창의 10배라 회전 직후 최대 5분간 새 `kid` 검증이 실패할 수 있어 그대로 되살리지 않았다. 캐시 가능해진 응답의 CORS 헤더가 Origin에 의존하므로 `Vary: Origin`을 허용된 Origin일 때만이 아니라 항상 붙이도록 했다. 검증: 새 테스트 2개(단위 + 연동)가 수정 전 코드에서 실제로 실패함을 확인했고, `go test -race ./...` 전체 통과(연동 테스트 SKIP 0건), `go vet`, `golangci-lint`(0 issues), `govulncheck`(0), `npm run lint`, `npm run test`(22파일/104테스트, 디스크 파일 수와 일치), `npm run build` 모두 통과. 빌드가 만든 `webui/dist/index.html` 해시 변경은 무관하므로 되돌렸다.
- 보류 아이디어:
  - Discovery에 `authorization_response_iss_parameter_supported: true` 추가 — 서버는 이미 인가 응답에 RFC 9207 `iss`를 넣는데 메타데이터로 알리지 않아 RP가 강제 검증을 켜지 못한다 (가치 3 / 위험 1 / S)
  - UserInfo POST에서 form-encoded `access_token` 파라미터 수용 (RFC 6750 §2.2). 현재는 Authorization 헤더만 읽는다 (가치 2 / 위험 1 / S)
  - `authorization` 엔드포인트가 기존 SSO Session 재사용 시 계정 상태를 다시 보지 않아, 토큰 교환에서야 거절될 코드를 발급하는 경로가 있는지 점검 (가치 2 / 위험 2 / M)
  - 잠긴(locked) 계정의 기존 SSO Session이 계속 새 인가 코드를 받는 동작을 의도된 것으로 문서화할지 검토 (가치 2 / 위험 2 / S)
  - `login`에서 `prompt=login` 재인증 시 기존 Session을 정리하지 않아 한 사용자에게 Session이 둘 남는 문제 (가치 2 / 위험 2 / M)
- 릴리즈: v0.9.66 (2026-09-02)

## 2026-09-02 (2회차)
- 선택: Discovery에 `authorization_response_iss_parameter_supported: true` 추가 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공 (커밋 1876209)
- 요약: 서버는 인가 응답 네 갈래(세션 조회 전 오류, 기존 SSO Session에서 발급한 코드, 로그인 폼이 JSON으로 돌려주는 redirect_to, `prompt=none` 미로그인)에 모두 RFC 9207 `iss`를 넣고 있었는데 메타데이터로 알리지 않았다. RFC 9207상 클라이언트는 메타데이터가 약속할 때만 `iss` 없는 응답을 거절하므로, 플래그가 없으면 라이브러리는 계속 `iss` 없는 응답을 받아들인다 — 즉 방어가 꺼진 상태였다. 겸해서 discovery만 `strings.TrimRight(realm.IssuerURL, "/")`로 자기 사본의 끝 슬래시를 떼고 있던 것을 제거해, 발행되는 `issuer`와 모든 응답·토큰의 `iss`가 같은 한 문자열이 되게 했다(store가 v0.1.0부터 입력 시 트림·검증하므로 API로 만든 Realm에는 슬래시가 없다. 그 트림은 불일치를 고쳐주지 못하고 가장 먼저 확인할 곳에서 숨기기만 했다). `docs/compatibility.md`에 한 줄 추가. 검증: 새 연동 테스트가 수정 전 코드에서 실제로 실패함을 확인했고(`discovery advertises <nil>`), `make test` 전체 통과 — `go test -race ./...` 전 패키지 ok, 연동 테스트 SKIP 0건, `go vet`, `golangci-lint`(0 issues), `govulncheck`(0), `npm run lint`, `npm run test`(22파일/104테스트, 디스크 파일 수와 일치), `npm run build`. 빌드가 만든 `webui/dist/index.html` 변경은 되돌렸다.
- 보류 아이디어:
  - UserInfo POST에서 form-encoded `access_token` 파라미터 수용 (RFC 6750 §2.2). 현재는 Authorization 헤더만 읽는다 (가치 2 / 위험 1 / S)
  - `SessionByToken`이 `u.enabled=true`만 보고 `locked_until`은 보지 않아 잠긴 계정의 기존 SSO Session이 계속 새 인가 코드를 받는다 — 의도인지 결정하고 문서화하거나 막을 것 (가치 3 / 위험 2 / M)
  - `login`에서 `prompt=login` 재인증 시 기존 Session을 정리하지 않아 한 사용자에게 Session이 둘 남는 문제 (가치 2 / 위험 2 / M)
  - Discovery에 `prompt_values_supported`·`claim_types_supported` 등 남은 선택 메타데이터 추가 검토 (가치 1 / 위험 1 / S)
  - `scripts/test-services.sh`가 PostgreSQL 포트 충돌 시 컨테이너를 Created 상태로 남기고 다음 실행에서 인증 실패로만 드러난다 — 포트 점유를 감지해 알려주기 (가치 2 / 위험 1 / S)

## 2026-09-03
- 선택: userinfo가 Role을 못 읽었을 때 "Role 없음"으로 답하던 문제 수정 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공 (커밋 c5fb7b4)
- 요약: `roles` 스코프 뒤의 두 조회(`RealmRolesForUser`, `ClientRolesForUser`)가 에러를 `_`로 버려서, 조회 실패가 `realm_access.roles: []` + 200으로 나갔다 — 서비스가 "이 계정은 아무 Role도 없다"고 **단언**하는 것이고, 관리자가 Role을 전부 회수한 경우와 글자 그대로 구별되지 않는다. 같은 스코프의 같은 두 호출이 토큰 발급(`IssueUserTokens`)에서는 이미 요청을 실패시키므로, userinfo만 못 읽은 Role을 에러가 아닌 claim으로 바꾸고 있었다. 이제 500 `server_error`로 거절하고 어느 조회가 실패했는지 로그에 남긴다 — 여기의 다른 거절이 쓰는 401을 일부러 쓰지 않았다. 토큰은 멀쩡한데 invalid_token이라 하면 RP가 멀쩡한 자격증명을 버리고 사용자를 로그아웃시킨다. `/api/v1/me`도 같은 에러를 버리고 있어 함께 고쳤다. 검증: 새 연동 테스트가 두 Role 테이블을 차례로 치우며(테스트마다 전용 스키마라 안전) 수정 전 코드에서 실제로 두 건 모두 실패함을 확인했다(`answered 200 and map[realm_access:map[roles:[]]...]`). `make test` 전체 통과 — `go test -race ./...` 전 패키지 ok, 연동 테스트 SKIP 0건(httpserver 76s / store 77s), `go vet`, `golangci-lint`(0 issues), `govulncheck`(0), `npm run lint`, `npm run test`(22파일/104테스트), `npm run build`. 빌드가 만든 `webui/dist/index.html` 변경은 되돌렸다.
- 보류 아이디어:
  - UserInfo POST에서 form-encoded `access_token` 파라미터 수용 (RFC 6750 §2.2). 현재는 Authorization 헤더만 읽는다 (가치 2 / 위험 1 / S)
  - `SessionByToken`이 `locked_until`을 보지 않는 것은 **의도로 보인다** — 잠금은 무차별 대입 방어이고, 이를 세션 종료로 확장하면 공격자가 남의 비밀번호를 틀리는 것만으로 피해자를 계속 로그아웃시키는 DoS가 된다. 막지 말고 문서화하는 쪽으로 결론낼 것 (가치 2 / 위험 1 / S)
  - `authorization`이 `id_token_hint`·`max_age`를 `AuthorizationRequest`에 저장하지 않아, 로그인 폼을 거친 뒤에는 hint가 지목한 계정과 다른 계정으로 로그인해도 코드가 나간다 (가치 2 / 위험 2 / M)
  - `oidcLogout`이 `id_token_hint`의 토큰 타입(`Extra.Type`)을 확인하지 않아 Access Token도 hint로 받아들인다 (가치 2 / 위험 1 / S)
  - `scripts/test-services.sh`가 PostgreSQL 포트 충돌 시 컨테이너를 Created 상태로 남기고 다음 실행에서 인증 실패로만 드러난다 — 포트 점유를 감지해 알려주기 (가치 2 / 위험 1 / S)
- 릴리즈: v0.9.67 (2026-09-03)

## 2026-09-03 (2회차)
- 선택: RP-initiated logout이 만료된 `id_token_hint`를 거절하던 문제 수정 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공 (커밋 0044bd0)
- 요약: `oidcLogout`이 hint를 `Verify`(만료 검사 포함)로 통과시켜서, 로그아웃 시점에 RP가 들고 있는 **보통 상태인 만료된 ID Token**을 거절했다. 거절이 거절로 보이지도 않았다 — `client`가 nil로 남아 `post_logout_redirect_uri`가 말없이 버려지고 브라우저는 `/login?logged_out=1`에 남았다(세션은 끝나고 쿠키도 지워진 채). RP 입장에선 사람이 그냥 돌아오지 않는다. 만료를 보지 않는 이유는 같은 저장소의 `SubjectFromIDTokenHint`에 이미 적혀 있었고(인가 엔드포인트의 같은 파라미터), RP-Initiated Logout 1.0 §4도 같은 것을 요구한다 — logout만 따르지 않고 있었다. 새 `IDTokenHint`로 서명·issuer는 그대로 확인하고 만료만 빼며, 덤으로 `typ != "ID"`를 확인해 Access Token이 hint로 통하던 것도 막았다(azp·iss·sub가 같아 그대로 통과하고 있었다). 리다이렉트 대상은 여전히 등록 목록과 대조하므로 open redirect가 되지 않고, 꺼진 Client도 여전히 거절된다. 검증: 새 연동 테스트가 수정 전 코드에서 두 건 모두 실제로 실패함을 확인했다(만료 hint → `/login?logged_out=1`, Access Token → 수락). Realm TTL 하한이 60초라 만료 토큰은 Realm 키로 직접 서명했다. `make test` 전체 통과 — `go test -race ./...` 전 패키지 ok, 연동 테스트 SKIP 0건(httpserver 75s / store 76s), `go vet`, `golangci-lint`(0 issues), `govulncheck`(0), `npm run lint`, `npm run test`(22파일/104테스트), `npm run build`. 빌드가 만든 `webui/dist/index.html` 변경은 되돌렸다.
- 보류 아이디어:
  - UserInfo POST에서 form-encoded `access_token` 파라미터 수용 (RFC 6750 §2.2). 현재는 Authorization 헤더만 읽는다 (가치 2 / 위험 1 / S)
  - `authorization`이 `id_token_hint`·`max_age`를 `AuthorizationRequest`에 저장하지 않아, 로그인 폼을 거친 뒤에는 hint가 지목한 계정과 다른 계정으로 로그인해도 코드가 나간다 (가치 3 / 위험 2 / M)
  - `oidcLogout`이 hint의 `sub`를 쿠키 세션의 사용자와 대조하지 않아, 다른 사람의 ID Token을 hint로 줘도 지금 로그인한 사람이 로그아웃된다 (스펙상 SHOULD, CSRF성 성가심 수준) (가치 2 / 위험 2 / M)
  - `SessionByToken`이 `locked_until`을 보지 않는 것은 의도로 보인다 — 막지 말고 문서화하는 쪽으로 결론낼 것 (가치 2 / 위험 1 / S)
  - `scripts/test-services.sh`가 PostgreSQL 포트 충돌 시 컨테이너를 Created 상태로 남기고 다음 실행에서 인증 실패로만 드러난다 — 포트 점유를 감지해 알려주기 (가치 2 / 위험 1 / S)
- 릴리즈: v0.9.68 (2026-09-03)

## 2026-09-03 (3회차)
- 선택: userinfo가 이쪽 장애를 `invalid_token`으로 답하던 문제 수정 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공 (커밋 8ab04c0)
- 요약: userinfo가 claim을 읽기 전에 하는 조회 넷(`realmFromPath`, `Verify`의 폐기 여부 확인, `UserByID`, `SessionAuthTime`)이 "없어서 실패"와 "조회 자체가 실패"를 모두 401 `invalid_token`으로 답하고 있었다. 그 답은 지시다 — RP는 invalid_token을 받으면 자격증명을 버리고 사용자를 로그아웃시키며, 그게 올바른 처리다. 그래서 DB 장애가 이 엔드포인트를 저하시키는 게 아니라 Realm의 모든 토큰을 한꺼번에 무효화했고, 401은 여기서 평범한 응답(만료)이라 요청 카운터·액세스 로그에는 조용한 시간대로 보였다. 같은 파일이 이미 두 번 이 구분을 하고 있었는데(`writeUserInfoUnavailable`의 Role, `recordUnjudgedIntrospection`) 이 넷은 그보다 **앞에서** 돌아 장애가 먼저 여기 닿았고 뒤쪽 배려는 실행되지 않았다. 이제 `store.ErrNotFound`만 401로 두고 나머지는 500 `server_error` + 어느 단계인지 로그에 남긴다. 폐기 여부를 못 읽은 토큰은 `ErrTokenStateUnavailable`(revoke 엔드포인트가 이미 구분하던 것)로 판정 불가 취급한다. 꺼진 Realm은 여전히 401이라 테넌트 정지 동작은 그대로다. 검증: 새 연동 테스트가 네 테이블(`realms`·`revoked_access_tokens`·`users`·`sso_sessions`)을 차례로 RENAME으로 숨기며, 수정 전 코드에서 네 건 모두 401로 실패함을 확인했다(테스트마다 전용 스키마라 안전). `make test` 전체 통과 — `go test -race ./...` 전 패키지 ok(httpserver 94s / store 93s), 연동 테스트 SKIP 0건, `go vet`, `golangci-lint`(0 issues), `govulncheck`(0), `npm run lint`, `npm run test`(22파일/104테스트, 디스크 파일 수와 일치), `npm run build`. 빌드가 만든 `webui/dist/index.html` 변경은 되돌렸다.
- 보류 아이디어:
  - `introspect`의 `Verify` 단계만 `recordUnjudgedIntrospection`을 부르지 않아, 폐기 여부를 못 읽은 경우가 카운터·로그 없이 `active=false`로 나간다 (가치 3 / 위험 1 / S)
  - `authorization`이 `id_token_hint`를 `AuthorizationRequest`에 저장하지 않아, 로그인 폼을 거친 뒤에는 hint가 지목한 계정과 다른 계정으로 로그인해도 코드가 나간다 (컬럼 추가 마이그레이션 필요) (가치 3 / 위험 2 / M)
  - UserInfo POST에서 form-encoded `access_token` 파라미터 수용 (RFC 6750 §2.2). 현재는 Authorization 헤더만 읽는다 (가치 2 / 위험 1 / S)
  - `oidcLogout`이 hint의 `sub`를 쿠키 세션의 사용자와 대조하지 않아, 다른 사람의 ID Token을 hint로 줘도 지금 로그인한 사람이 로그아웃된다 (스펙상 SHOULD) (가치 2 / 위험 2 / M)
  - `scripts/test-services.sh`가 PostgreSQL 포트 충돌 시 컨테이너를 Created 상태로 남기고 다음 실행에서 인증 실패로만 드러난다 — 포트 점유를 감지해 알려주기 (가치 2 / 위험 1 / S)

## 2026-09-03 (4회차)
- 선택: introspect가 답을 못 낸 조회를 세지 않던 문제 수정 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공 (커밋 26d21d5)
- 요약: `recordUnjudgedIntrospection`은 "죽었다고 판정한 토큰"과 "판정하지 못한 토큰"이 둘 다 200 `active=false`로 나가 서비스가 내보내는 모든 신호에서 같은 호출로 보이기 때문에 만들어졌는데, 핸들러 맨 아래 두 조회(`session`·`user`)만 그것을 부르고 있었다. 그 앞에서 도는 조회 셋 — `realmFromPath`, `Verify`의 폐기 여부 확인(`ErrTokenStateUnavailable`), `InspectRefreshToken` — 은 "없다"와 "못 읽었다"를 똑같이 `active=false`로 답했다. 즉 장애는 이 셋에 먼저 닿았고 아래쪽 배려는 실행되지 않았다: store 에러는 그 자리에서 버려지고, 응답은 200이라 요청 카운터에도 정상 호출로 잡히며, 운영 문서가 보라고 지목한 계열은 평평한 채로 모든 Resource Server가 모든 요청을 거절했다. 응답은 일부러 그대로 뒀다(5xx를 받은 RS가 fail-open할 수 있어 fail-closed가 맞는 방향이고, 이 판단은 함수 주석에 이미 적혀 있다). 바뀐 것은 실행되지 못한 조회를 stage 라벨과 함께 세고 로그에 남긴다는 것뿐이며, 없는 Realm·세션·Refresh Token은 진짜 답이므로 여전히 세지 않는다. 덤으로 폐기 여부를 못 읽은 토큰은 Refresh Token 조회로 흘러가지 않고 거기서 멈춘다 — 서명과 클레임이 이미 통과했으니 Refresh Token일 수 없고(불투명 문자열이라 파싱 단계를 넘지 못한다), 그대로 가면 같은 store에 두 번째로 실패할 뿐이었다. 검증: 새 연동 테스트가 세 테이블(`realms`·`revoked_access_tokens`·`refresh_tokens`)을 차례로 RENAME으로 숨기며 수정 전 코드에서 세 건 모두 실패함을 확인했다(`said so nowhere: no resso_introspection_errors_total{stage="realm"} 1` 등). 정상 토큰과 아무 토큰도 아닌 문자열이 카운터를 건드리지 않는 것도 같은 테스트가 확인한다. `make test` 전체 통과 — `go test -race ./...` 전 패키지 ok(httpserver 82s / store 81s), 연동 테스트 SKIP 0건, `go vet`, `golangci-lint`(0 issues), `govulncheck`(0), `npm run lint`, `npm run test`, `npm run build`. 빌드가 만든 `webui/dist/index.html` 변경은 되돌렸다.
- 보류 아이디어:
  - `scripts/test-services.sh`가 이미 떠 있는 컨테이너를 재사용할 때 그 컨테이너의 **실제 매핑 포트**가 아니라 기본값(55439)을 DSN으로 출력한다 — 이번 세션에서 실제로 걸렸다(컨테이너는 55450, 스크립트는 55439를 출력 → `connection refused`). `docker port`로 읽어 출력할 것 (가치 3 / 위험 1 / S)
  - `authorization`이 `id_token_hint`를 `AuthorizationRequest`에 저장하지 않아, 로그인 폼을 거친 뒤에는 hint가 지목한 계정과 다른 계정으로 로그인해도 코드가 나간다 (컬럼 추가 마이그레이션 필요) (가치 3 / 위험 2 / M)
  - UserInfo POST에서 form-encoded `access_token` 파라미터 수용 (RFC 6750 §2.2). 현재는 Authorization 헤더만 읽는다 (가치 2 / 위험 1 / S)
  - `oidcLogout`이 hint의 `sub`를 쿠키 세션의 사용자와 대조하지 않아, 다른 사람의 ID Token을 hint로 줘도 지금 로그인한 사람이 로그아웃된다 (스펙상 SHOULD) (가치 2 / 위험 2 / M)
  - `docs/operations.md`의 `resso_introspection_errors_total` 항목에 stage 라벨 값 목록을 적어, 어느 조회가 멈췄는지 대시보드에서 바로 읽게 하기 (가치 1 / 위험 1 / S)

## 2026-09-04
- 선택: `scripts/test-services.sh`가 재사용 컨테이너의 실제 포트가 아니라 요청한 포트를 출력하던 문제 수정 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공 (커밋 b6d30b9)
- 요약: 스크립트 상단의 세 포트는 **요청**이지 사실이 아니다 — 이번 실행이 만든 컨테이너에만 적용되고, 이전 실행에서 남은 컨테이너는 처음 기동된 포트를 그대로 유지하는데 스크립트의 모든 경로가 기존 컨테이너를 보지도 않고 재사용한다. 그래서 출력된 환경은 요청을, 테스트는 현실을 가리켰다(이번 세션에서 실제로 재현: DSN은 55439, 컨테이너는 55450). 스크립트가 이걸 잡을 수 없었던 이유는 모든 준비 확인이 `docker exec`로 컨테이너 **안에서** 서비스에 닿기 때문이다 — 발행된 포트를 지나는 확인이 하나도 없어서, 아무 데도 닿지 않는 주소가 통합 테스트 예순 개가 한꺼번에 실패하는 것으로 처음 드러났다. 이제 출력 전에 `docker port`로 실제 매핑을 읽고, 출력할 주소를 호스트에서 한 번 열어본다. 같은 독해에서 이웃 둘이 떨어졌다: 포트 충돌로 기동에 실패한 컨테이너는 Created로 남아 다음 실행이 "재사용"하므로 이제 `docker start`로 살리거나 이유를 말하고 멈추며, `--stop` 분기가 부르는 `log`가 그 아래에 정의돼 있어 인증서 디렉터리를 못 지웠다고 말하려던 유일한 경로가 `log: command not found`로 답하던 것도 고쳤다. 검증: 새 Go 테스트 2개가 docker 스텁과 실제 리스너로 스크립트를 그대로 실행해, 수정 전 코드에서 두 건 모두 실제로 실패함을 확인했다(요청 포트 55439/13890/13636을 그대로 출력, 아무도 듣지 않는 포트를 정상 환경으로 출력). `make test` 전체 통과 — `go test -race ./...` 전 패키지 ok(httpserver 82s / store 81s), 연동 테스트 SKIP 0건, `go vet`, `golangci-lint`(0 issues), `govulncheck`(0), `npm run lint`, `npm run test`, `npm run build`. 빌드가 만든 `webui/dist/index.html` 변경은 되돌렸다.
- 보류 아이디어:
  - `authorization`이 `id_token_hint`를 `AuthorizationRequest`에 저장하지 않아, 로그인 폼을 거친 뒤에는 hint가 지목한 계정과 다른 계정으로 로그인해도 코드가 나간다 (컬럼 추가 마이그레이션 필요) (가치 3 / 위험 2 / M)
  - UserInfo POST에서 form-encoded `access_token` 파라미터 수용 (RFC 6750 §2.2). 현재는 Authorization 헤더만 읽는다 (가치 2 / 위험 1 / S)
  - `oidcLogout`이 hint의 `sub`를 쿠키 세션의 사용자와 대조하지 않아, 다른 사람의 ID Token을 hint로 줘도 지금 로그인한 사람이 로그아웃된다 (스펙상 SHOULD) (가치 2 / 위험 2 / M)
  - `docs/operations.md`의 `resso_introspection_errors_total` 항목에 stage 라벨 값 목록을 적어, 어느 조회가 멈췄는지 대시보드에서 바로 읽게 하기 (가치 1 / 위험 1 / S)
  - `SessionByToken`이 `locked_until`을 보지 않는 것은 의도(잠금은 무차별 대입 방어이고 세션 종료로 확장하면 DoS가 된다) — 막지 말고 문서화하는 쪽으로 결론낼 것 (가치 2 / 위험 1 / S)
- 릴리즈: v0.9.69 (2026-09-04)

## 2026-09-04 (2회차)
- 선택: SSO 세션을 **못 읽은 것**을 세션이 **없는 것**으로 답하던 두 곳 수정 (가치 4 / 위험 1 / 작업량 M)
- 결과: 성공 (커밋 bcc6f60)
- 요약: 브라우저의 SSO 세션을 읽는 OIDC 엔드포인트 둘(`authorization`·`oidcLogout`)이 `SessionByToken`의 에러를 "없음"과 같이 취급했고, 그 결과 나가는 두 답이 모두 RP가 **행동으로 옮기는 단언**이었다. (1) `authorization`에서 `prompt=none`의 답인 `login_required`는 "세션이 없다"는 말이고, 조용한 갱신을 하는 RP는 이를 "사용자가 로그아웃했다"로 읽고 **자기 세션도 끝낸다** — 올바른 처리다. 그래서 이쪽 장애가 저하가 아니라 **전 RP 일괄 로그아웃**이 됐다. 바로 한 줄 아래 `SessionAuthenticatedRecently`는 이미 못 돌면 `server_error`를 내는데, 이 조회가 그보다 먼저 돌아 장애가 여기 먼저 닿았다. (2) `oidcLogout`에서는 `endSession` 주석이 "이 서비스가 가진 가장 오해를 부르는 실패"라고 이미 적어둔 그 결과 — 세션은 살아 있고, 거기 묶인 Refresh Token은 계속 갱신되며, 폐기가 곧 통지이므로 back-channel logout도 나가지 않는다 — 가 **한 단계 앞에서** 벌어지는데 감사 기록이 **아예 없었다**. 로그인 안 한 브라우저의 로그아웃과 글자 그대로 같은 모습이었고, 쿠키는 지워지고 RP는 post-logout 페이지로 리다이렉트되어 다 끝난 것처럼 보였다. 이제 `store.ErrNotFound`만 "로그인 안 함"이고, 나머지는 각각 `server_error`와 `PARTIAL` LOGOUT 기록(+ Realm을 적은 로그)이다. 로그아웃 응답은 일부러 그대로 뒀다(`endSession` 위에 이미 적힌 판단 — 그 시점엔 쿠키가 이미 지워졌고 사람이 할 수 있는 게 없다). 기록이 세션을 지목하지 못하는 이유(조회가 실패한 것이므로)와 IP·시각으로 찾으라는 안내를 `docs/operations.md`에, `prompt=none`의 새 구분을 `docs/compatibility.md`에 적었다. 검증: 새 연동 테스트가 `sso_sessions`를 RENAME으로 숨기고(테스트마다 전용 스키마) 수정 전 코드에서 세 건 모두 실제로 실패함을 확인했다(`prompt=none answered error="login_required"`, 인가 요청 `error=""`, `logout ... was not audited at all`). 쿠키 없는 로그아웃이 새 기록을 만들지 않는 것과 정상 상태에서 `prompt=none`이 코드를 받는 것도 같은 테스트가 확인한다. `make test` 전체 통과(exit 0) — `go test -race ./...` 전 패키지 ok(httpserver 85s / store 85s), 연동 테스트 SKIP 0건, `go vet`, `golangci-lint`(0 issues), `govulncheck`(0), `npm run lint`, `npm run test`, `npm run build`. 빌드가 만든 `webui/dist/index.html` 변경은 되돌렸다.
- 보류 아이디어:
  - `requireSession` 미들웨어도 저장소 장애를 401 `authentication_required`로 답해, DB 순단이 콘솔 사용자 전원을 로그인 화면으로 보낸다 — 프런트엔드 처리까지 봐야 해 이번엔 뺐다 (가치 3 / 위험 2 / M)
  - `authorization`이 `id_token_hint`를 `AuthorizationRequest`에 저장하지 않아, 로그인 폼을 거친 뒤에는 hint가 지목한 계정과 다른 계정으로 로그인해도 코드가 나간다 (컬럼 추가 마이그레이션 필요) (가치 3 / 위험 2 / M)
  - UserInfo POST에서 form-encoded `access_token` 파라미터 수용 (RFC 6750 §2.2). 현재는 Authorization 헤더만 읽는다 (가치 2 / 위험 1 / S)
  - `oidcLogout`이 hint의 `sub`를 쿠키 세션의 사용자와 대조하지 않아, 다른 사람의 ID Token을 hint로 줘도 지금 로그인한 사람이 로그아웃된다 (스펙상 SHOULD) (가치 2 / 위험 2 / M)
  - `docs/operations.md`의 `resso_introspection_errors_total` 항목에 stage 라벨 값 목록을 적어, 어느 조회가 멈췄는지 대시보드에서 바로 읽게 하기 (가치 1 / 위험 1 / S)
- 릴리즈: v0.9.70 (2026-09-04)
## 2026-09-05
- 선택: 콘솔의 세션·API Key 미들웨어가 이쪽 장애를 "로그인이 필요합니다"로 답하던 문제 수정 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공 (커밋 b25bc63)
- 요약: 관리 콘솔의 인증된 요청은 전부 같은 조회(`SessionByToken`)에서 시작하는데, `requireSession`·`requireSessionOrAPIKey`가 그 에러를 "세션 없음"과 같이 401 `authentication_required`로 답했다. 콘솔은 그 답을 **보여주지 않고 실행한다** — API 계층이 알리고, auth provider가 캐시된 신원을 지우고, 사람은 "세션이 만료되었습니다"를 달고 로그인 화면에 도착한다. 그래서 `sso_sessions` 장애는 콘솔을 저하시킨 게 아니라 **전원을 동시에 로그아웃시키고 거짓 이유를 붙였다**: 세션은 멀쩡했고, 돌려보낸 로그인 화면도 같은 이유로 실패하고 있었다. 변경 요청은 더 나빴다 — 읽지 못한 쿠키가 쿠키를 안 보낸 것처럼 API Key 분기로 흘러가, 로그인한 사람의 모든 편집에 `browser_session_required`("브라우저 로그인이 필요합니다")가 나갔다. `AuthenticateAPIKey`도 같은 뭉개기를 했고 `/metrics` 수집이 그 경로로 인증하므로, 서비스를 가장 봐야 할 순간에 모니터링은 "키가 폐기됐다"는 답을 받았다. 아무것도 드러나지 않았다: 여기서 401은 로그인 안 한 방문자의 평범한 답이라 요청 카운터·접근 로그에는 한산한 시간대였고, store 에러는 로그 한 줄 없이 버려졌다. 이건 인증된 모든 요청의 **첫 번째** store 조회라, 전면 장애에서 가장 먼저 닿고 그 뒤의 배려(이미 둘을 구분하는 핸들러들, 그 아래 `writeStoreError`)는 실행되지 않는다. 이제 `store.ErrNotFound`만 "로그인 안 함"이고 나머지는 500 `internal_error` + 어느 조회가 멈췄는지 로그. 빈 쿠키와 빈 Bearer는 store에 닿지 않으므로 로그인 안 한 방문자의 답은 그대로고, 세션 테이블 장애에도 API Key 클라이언트는 200을 받는다. 콘솔 쪽은 그 구분을 그대로 따랐다 — 401은 거절이고 500은 아니므로, 신원 조회가 다른 이유로 실패하면 로그인 화면으로 보내지 않고 화면에 머문 채 오류와 `다시 시도`를 보여준다(`ErrorAlert`의 5xx 안내가 이미 맞는 문구다). 검증: 새 연동 테스트가 `sso_sessions`와 `personal_api_keys`를 차례로 RENAME으로 숨기며(테스트마다 전용 스키마), 수정 전 코드에서 다섯 건 모두 실제로 실패함을 확인했다 — GET `/api/v1/me`·PUT `/api/v1/me/profile`·POST `/api/v1/auth/logout`·GET `/api/admin/v1/realms`가 모두 401이었고 API Key도 401이었다. 자격증명 없는 호출이 여전히 401인 것과 세션 테이블 장애 중에도 API Key가 200인 것도 같은 테스트가 확인한다. 콘솔 테스트 2개를 더해 500이 만료로 읽히지 않는 것과 `identityError`가 서는 것을 고정했다. `make test` 전체 통과(exit 0) — `go test -race ./...` 전 패키지 ok(httpserver 84s / store 82s), 연동 테스트 SKIP 0건, `go vet`, `golangci-lint`(0 issues), `govulncheck`(0), `npm run lint`, `npm run test`(22파일/106테스트, 디스크 파일 수와 일치), `npm run build`. 빌드가 만든 `webui/dist/index.html` 변경은 되돌렸다. `docs/operations.md`에 401/500 구분표와 Prometheus 수집 해석을 적었다.
- 보류 아이디어:
  - `login`의 Realm 조회(`RealmByName`·`RealmByID`) 실패가 401 `invalid_credentials`로 나가 사용자에게 "비밀번호가 틀렸다"고 말하고, 로그인 카운터도 움직이지 않는다 — 같은 핸들러의 다른 여섯 조회는 모두 `writeLoginError`(500 + `result="error"`)를 쓰는데 이것만 그 앞에서 돌아 예외다 (가치 4 / 위험 1 / S)
  - `login`의 `AuthorizationRequestByToken` 실패가 400 `expired_request`로 나가, 사용자는 RP에서 흐름을 다시 시작하고 같은 곳에서 다시 실패한다 (가치 3 / 위험 1 / S)
  - `authorization`이 `id_token_hint`를 `AuthorizationRequest`에 저장하지 않아, 로그인 폼을 거친 뒤에는 hint가 지목한 계정과 다른 계정으로 로그인해도 코드가 나간다 (컬럼 추가 마이그레이션 필요) (가치 3 / 위험 2 / M)
  - UserInfo POST에서 form-encoded `access_token` 파라미터 수용 (RFC 6750 §2.2). 현재는 Authorization 헤더만 읽는다 (가치 2 / 위험 1 / S)
  - `oidcLogout`이 hint의 `sub`를 쿠키 세션의 사용자와 대조하지 않아, 다른 사람의 ID Token을 hint로 줘도 지금 로그인한 사람이 로그아웃된다 (스펙상 SHOULD) (가치 2 / 위험 2 / M)

## 2026-09-06
- 선택: login이 이쪽 장애를 "비밀번호가 틀렸다"·"요청이 만료됐다"로 답하던 문제 수정 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공 (커밋 9ba95ab)
- 요약: 로그인 핸들러는 인증할 대상을 갖기 전에 조회를 둘 하는데, 둘 다 "조회 실패"를 "없음"과 같이 취급했고 그 두 답은 모두 사람이 **행동으로 옮기는 지시**였다. (1) `RealmByName`·`RealmByID`는 401 `invalid_credentials`("아이디 또는 비밀번호가 올바르지 않습니다")로 답했다 — 맞는 비밀번호를 다시 치게 하고, 이어서 변형을 하나씩 시도하게 만든다. 같은 핸들러의 나머지 여섯 조회는 모두 `writeLoginError`(500 + `resso_login_attempts_total{result="error"}`)를 쓰는데, 이 조회가 그 여섯보다 **먼저** 돌아 장애가 여기 먼저 닿았고 뒤쪽 배려는 실행되지 않았다: 카운터는 움직이지 않았고, 로그인 경로의 401은 평범한 오타와 요청 카운터·접근 로그에서 구별되지 않으므로 전면 장애가 한산한 시간대로 보였다. 그 카운터가 존재하는 이유가 정확히 이것이다 — 완료되지 못한 시도는 성공도 실패도 아니라서 세지 않으면 시계열이 조용해질 뿐이다. (2) `AuthorizationRequestByToken`은 400 `expired_request`로 답했고, 그러면 사람은 RP에서 흐름을 다시 시작해 새 request token을 받고 같은 조회에서 같은 이유로 다시 실패한다. 이제 `store.ErrNotFound`만 옛 답을 유지한다 — 존재하지 않는 Realm은 계정·테넌트 열거 방지로 401 그대로, 소진·만료·발급된 적 없는 request token은 400 그대로. 나머지는 500 `internal_error` + `result="error"` + 어느 조회가 멈췄는지 로그다. 검증: 새 연동 테스트가 `realms`와 `authorization_requests`를 차례로 RENAME으로 숨기며(테스트마다 전용 스키마) Realm 이름 로그인과 request token 로그인 양쪽을 확인하고, 세 건 모두 카운터에 닿는지 본다. 수정 전 코드에서 네 건 모두 실제로 실패함을 확인했다(401 invalid_credentials 둘, 400 expired_request 하나, 카운터 `result="error"` 0). 존재하지 않는 Realm의 401과 발급된 적 없는 토큰의 400이 그대로인 것, 테이블 복구 후 로그인이 다시 되는 것도 같은 테스트가 고정한다. `make test` 전체 통과(exit 0) — `go test -race ./...` 전 패키지 ok(httpserver 88s / store 85s), 연동 테스트 SKIP 0건, `go vet`, `golangci-lint`(0 issues), `govulncheck`(0), `npm run lint`, `npm run test`(22파일/104테스트, 디스크 파일 수와 일치), `npm run build`. 빌드가 만든 `webui/dist/index.html` 변경은 되돌렸다. `docs/operations.md`의 `result="error"` 경보 항목에 어느 답이 무엇을 뜻하는지 적었다.
- 보류 아이디어: `requireSession`·`AuthenticateAPIKey`의 같은 뭉개기(가치 4/위험 2/M — 다른 브랜치 b25bc63에서 이미 고쳐 병합 대기 중) / `authorization`이 `id_token_hint`·`max_age`를 저장하지 않아 로그인 폼을 거치면 hint가 지목한 계정과 다른 계정으로도 코드가 나간다(가치 3/위험 2/M) / UserInfo POST에서 form-encoded `access_token` 수용, RFC 6750 §2.2(가치 2/위험 1/S) / `oidcLogout`이 hint의 `sub`를 쿠키 세션과 대조하지 않는다(가치 2/위험 2/M) / `docs/operations.md`의 `resso_introspection_errors_total`에 stage 라벨 값 목록 적기(가치 1/위험 1/S)

## 2026-09-06
- 선택: Token 엔드포인트가 이쪽 조회 실패를 invalid_grant로 답하던 문제 수정 (가치 5 / 위험 2 / 작업량 M)
- 결과: 성공 (커밋 7753752)
- 요약: `invalid_grant`는 "다시 해보라"가 아니라 **"들고 있는 Code·Refresh Token은 죽었다"는 단언**이고, RP는 그것을 버리는 것이 올바른 처리다 — Refresh Token이면 그 RP에서 사람이 영구히 로그아웃된다. Token 엔드포인트가 Signing Key에 닿기 전에 하는 조회 여섯(`realmFromPath`, `RedeemAuthorizationCode`, `UserByID` 둘, `InspectRefreshToken`, `RotateRefreshToken`)이 "판정해서 거절한 것"과 "조회를 완료하지 못한 것"을 모두 그 답으로 내보냈다. 그래서 DB 순단이 이 엔드포인트를 저하시킨 게 아니라 **질문받은 세션을 전부 끝냈다**. 근거는 이미 이 파일에 적혀 있었다 — 각 조회 바로 몇 줄 아래 `ErrNoActiveSigningKey` 처리에 "키가 복구되면 끝날 장애가 모든 세션을 데려간다"고 쓰여 있는데, 여섯 조회가 **그보다 앞에서** 돌아 장애가 먼저 닿았고 뒤쪽 배려는 실행되지 않았다. 아무것도 드러나지 않았다: 400 `invalid_grant`는 만료된 Code·회전된 Token으로 하루 종일 나가는 답이라 요청 카운터·접근 로그에는 한산한 시간대로 보였고, store 에러는 그 자리에서 버려졌다. `resso_token_errors_total`이 존재하는 이유가 정확히 이것이다 — 완료되지 못한 요청은 발급도 거절도 아니라서 세지 않으면 시계열이 조용해질 뿐이다. 이제 `store.ErrNotFound`·`ErrCodeReuse`·`ErrTokenReuse`와 핸들러 자신의 거절(다른 Client·redirect_uri로 발급된 코드, 맞지 않는 `code_verifier`)만 400을 유지하고, 나머지는 500 `server_error` + 카운터 + `stage` 로그(`realm`·`authorization_code`·`refresh_token`·`user`·`rotation`)다. PKCE 오류는 평범한 `errors.New`라 store 실패와 타입으로 구분되지 않으므로, 판정하는 자리에서 세우는 플래그로 남겼다. 한 단계 아래 `RotateRefreshToken`도 같은 뭉개기를 하고 있었다 — 세션 생존 확인 쿼리가 **실행되지 못한 것**을 `ErrNotFound`로 답해, `sso_sessions` 장애가 진행 중인 모든 갱신에 "세션이 끝났다"고 말했다. 라벨은 `grantLabel`로 한정했다(grant_type은 호출자가 보내는 폼 필드이고 Realm 조회는 그것을 검사하기 전에 실패하므로, 그대로 쓰면 인증 없는 호출자가 요청마다 시계열을 하나씩 만든다 — 바로 위 `methodLabel`과 같은 이유다). 검증: 새 연동 테스트가 다섯 테이블(`realms`·`refresh_tokens`·`users`·`sso_sessions`·`authorization_codes`)을 차례로 RENAME으로 숨기며 두 grant 여섯 경우를 확인하고, 수정 전 코드에서 **여섯 건 모두 실제로 실패**함을 확인했다(전부 400 invalid_grant, 카운터 0, stage 로그 없음). 발급된 적 없는 Refresh Token의 400과 맞지 않는 `code_verifier`의 400이 카운터를 건드리지 않는 것, 테이블 복구 후 두 grant가 다시 도는 것도 같은 테스트가 고정한다. `make test` 전체 통과(exit 0) — `go test -race ./...` 전 패키지 ok(httpserver 90s / store 86s), 연동 테스트 SKIP 0건, `go vet`, `golangci-lint`(0 issues), `govulncheck`(0), `npm run lint`, `npm run test`(22파일/104테스트, 디스크 파일 수와 일치), `npm run build`. 빌드가 만든 `webui/dist/index.html` 변경은 되돌렸다. `docs/operations.md`의 `resso_token_errors_total` 항목에 stage 값 목록과 400/500 구분을, `docs/compatibility.md`의 Refresh Token 행에 "RP는 `invalid_grant`에서만 자격증명을 버리고 500에는 재시도한다"를 적었다.
- 보류 아이디어: `authenticateOIDCClient`의 store 실패가 401 `invalid_client`로 나갈 뿐 아니라 실패 레이트리밋 버킷까지 채워 장애가 끝난 뒤에도 잠금이 남는다(가치 4/위험 2/M) / `discovery`·`jwks`가 `realmFromPath` 실패를 404 `realm_not_found`로 답해 RP가 "issuer가 없다"로 읽는다(가치 3/위험 1/S) / `authorization`이 `id_token_hint`·`max_age`를 저장하지 않아 로그인 폼을 거치면 hint가 지목한 계정과 다른 계정으로도 코드가 나간다(가치 3/위험 2/M) / `authChallenge`가 없는 요청에 404를 답해 같은 원인에 login의 400 `expired_request`와 화면 문구가 갈린다(가치 2/위험 1/S) / UserInfo POST에서 form-encoded `access_token` 수용, RFC 6750 §2.2(가치 2/위험 1/S)

## 2026-09-07
- 선택: 자격증명을 **대조하지 못한 것**을 **거절한 것**으로 답하던 Client 인증 수정 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공 (커밋 acac754)
- 요약: Token·Introspection·Revocation 셋은 `authenticateOIDCClient` 하나를 공유하는데, 그것이 "조회를 완료하지 못한 것"과 "Secret이 맞지 않은 것"을 똑같이 401 `invalid_client`로 답했다. `invalid_client`는 **호출자의 자격증명에 관한 단언**이라 그 답을 받은 RP에게는 재시도할 것이 없다 — 서버가 방금 그 Secret을 받지 않겠다고 말했기 때문이다. 그래서 `clients` 테이블 장애가 이 셋을 저하시킨 게 아니라 **Realm의 모든 연동에 "너희 자격증명이 거절됐다"고 한꺼번에 통지**했다. 로그인·userinfo·introspection에 이미 그은 그 구분이 여기에는 없었다. **장애가 끝난 뒤까지 남는 것은 그 다음 줄이다**: 실패 버킷 둘(Client별·출발지 주소별)은 Secret 대입을 막으려고 있는 건데, 자격증명을 검사조차 못 한 요청이 그 둘에 실패로 기록됐다. 장애 중 스무 번이면 Client 버킷이 차고, 그때부터는 **맞는 Secret에도 5분 창이 끝날 때까지 429**가 나간다 — 저장소는 이미 복구된 뒤에도. 드러나는 것도 없었다: 여기서 401은 오래된 Secret이 하루 종일 만드는 평범한 답이라 요청 카운터와 접근 로그에는 한산한 시간대로 보였고, store 에러는 그 자리에서 버려졌다. 이제 `store.ErrNotFound`만 옛 답을 유지한다(없는 Client·틀린 Secret·Secret을 보낸 Public Client·비활성 Client는 모두 호출자에 관한 사실이므로 그대로). 나머지는 500 `server_error` + 어느 조회가 멈췄는지 로그이고, 어느 버킷에도 실패를 적지 않는다. 그 결과 `resso_client_auth_failures_total`은 **실제로 대조해서 거절한 것만** 세게 되어 이름이 늘 주장하던 뜻이 됐다. 검증: 새 연동 테스트가 두 조회를 차례로 없애며(테이블 `clients` RENAME → `ClientByIdentifier` 실패, 컬럼 `secret_hash` RENAME → `VerifyClientSecret` 실패) 엔드포인트 셋을 모두 확인하고, 이어서 버킷 하나를 가득 채울 만큼(21회) 장애 중 요청을 보낸 뒤 맞는 Secret으로 토큰을 요청한다. 수정 전 코드에서 **일곱 건 모두 실제로 실패**함을 확인했다(401 invalid_client 여섯, 21번째 요청 429). `make test` 전체 통과(exit 0) — `go test -race ./...` 11개 패키지 ok, FAIL 0, 연동 테스트 SKIP 0건, `go vet`, `golangci-lint`(0 issues), `govulncheck`(0), `npm run lint`, `npm run test`(22파일/104테스트, 디스크 파일 수와 일치), `npm run build`. 빌드가 만든 `webui/dist/index.html` 변경은 되돌렸다. `docs/operations.md`의 `resso_client_auth_failures_total` 항목과 `docs/compatibility.md`의 Client 인증 행에 401/429/500 구분을 적었다.
- 보류 아이디어: `discovery`·`jwks`가 `realmFromPath` 실패를 404 `realm_not_found`로 답해 RP가 "issuer가 없다"로 읽고 캐시한다(가치 3/위험 1/S) / `authorization`이 `id_token_hint`·`max_age`를 저장하지 않아 로그인 폼을 거치면 hint가 지목한 계정과 다른 계정으로도 코드가 나간다(가치 3/위험 2/M) / `authChallenge`가 없는 요청에 404를 답해 같은 원인에 login의 400 `expired_request`와 화면 문구가 갈린다(가치 2/위험 1/S) / UserInfo POST에서 form-encoded `access_token` 수용, RFC 6750 §2.2(가치 2/위험 1/S) / `oidcLogout`이 hint의 `sub`를 쿠키 세션과 대조하지 않는다(가치 2/위험 2/M)

