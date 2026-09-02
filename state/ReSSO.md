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
