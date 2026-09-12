## [0.2.12] - 2026-09-13

### 추가

- Keycloak에 이미 로그인한 사람이 jikim을 열면 로그인 화면 없이 바로 본 화면으로 들어가는 **자동 로그인(조용한 SSO)** 추가. OIDC 설정에 `auto_login`(boolean, 기본값 꺼짐, 타입 검증)을 두고 관리 화면 OIDC 탭의 **자동 로그인(조용한 SSO)** 스위치로 켭니다. 켜지 않은 설치에서는 아무것도 달라지지 않습니다
- `GET /api/v1/oidc/login?prompt=none[&return_to=…]`가 이 설정이 켜졌을 때만 `prompt=none` 인가 요청을 보내고, 꺼져 있으면 `?prompt=none`이 붙어도 조용히 평범한 로그인으로 바꿉니다. 조용한 시도인지와 `return_to`는 봉인된 state 쿠키에 기록하므로 주소를 고쳐 흐름을 바꿀 수 없습니다. 콜백은 조용한 시도(쿠키 열기 성공·state 일치·10분 이내)가 제공자 오류를 받았을 때만 `/login?sso=none[&return_to=…]`으로 보내고, 그 밖의 오류는 기존 `/oidc/callback?error=…`를 유지합니다
- `return_to`는 `/`로 시작하고 `//`·`/\`로 시작하지 않는 같은 오리진 경로만 받으며 `/login`과 `/oidc/callback`은 돌아갈 자리로 받지 않습니다. 인가 코드 교환과 수동 로그인 뒤에도 같은 자리로 돌아갑니다
- 공개 설정 `GET /api/v1/settings/public`에 `oidc_auto_login` 추가. OIDC 자체가 꺼져 있으면 항상 `false`입니다
- 프런트엔드는 게스트가 보호된 화면을 열 때 공개 설정을 읽어 최상위 이동(`window.location.assign`)으로 한 번만 조용한 로그인을 시도합니다. 무한 루프는 `sessionStorage`의 **한 탭 세션에 한 번** 표시, 스스로 로그아웃한 뒤 다시 로그인할 때까지 억제, 콜백이 남기는 `?sso=none` 표시의 세 겹으로 막고, `sessionStorage`를 읽지 못하면 "이미 시도했다"로 취급해 막히는 쪽으로 실패합니다. 콜백·로그인·오류 화면과 `/api/*`·`/v1/*`·`/mcp`·probe·`/momento/*` 경로에서는 시도하지 않습니다

### 문서

- 관리자 가이드 3.3절에 `auto_login` 설정과 동작 순서, 루프 방지, `return_to` 검증 규칙과 켜기 전 확인할 Keycloak 세션 정책을 적고, 사용자 가이드 로그인 절에 자동 로그인이 켜졌을 때의 화면 흐름을 추가했습니다. 두 가이드 PDF를 같은 변환기로 다시 구웠습니다

**Full Changelog**: https://github.com/hkjang/jikim/compare/v0.2.11...v0.2.12
