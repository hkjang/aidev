## v0.18.7

- **조용한 SSO(silent SSO)** — 관리자가 `auth.oidc.auto_login`(기본 꺼짐)을 켜면 로그인 화면이 OIDC `prompt=none` 으로 한 번 조용히 로그인을 시도합니다. 이미 IdP 에 로그인한 사용자는 로그인 화면을 거치지 않고 받은편지함으로 갑니다. 탭당 1회 시도·로그아웃 표시·주소의 `sso=` 표시로 IdP↔앱 되돌이표를 막고, `return_to` 는 앱 내부 경로(`/` 로 시작, `//` 제외)만 허용합니다. ADMIN_GUIDE 에 문서화.
- **로그인 리다이렉트 경고 정리** — `return_to` 오픈 리다이렉트 경고를 근거와 함께 닫았습니다(동작 변화 없음).
- **빌드 정보·OCI 라벨** — `postra version` 이 버전·커밋·빌드 시각을 함께 알리고, 릴리즈 워크플로가 이를 링크 시 새깁니다. 오프라인 이미지에 `org.opencontainers.image.*` 라벨이 붙습니다.

**Full Changelog**: https://github.com/hkjang/postra/compare/v0.18.6...v0.18.7
