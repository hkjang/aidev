# 회차 노트 2026-09-20-093402-appstore-improve — appstore
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 09:34] base pinned — main@a1a7719
- [러너 09:34] autonomy release — 

## 정찰 노트
- 고른 이유: 보류 목록의 1순위 후보(silent SSO 거절 안내)를 코드로 재확인했다 — `auth-pages.tsx:LoginPage` 가 `sso` 를 안 읽는 것이 사실이고, 서버(`auth_handlers.go:124`)는 `sso=none` 만 만든다. 사용자 눈에 보이는 결함이면서 Go·auth·migrations 를 건드리지 않고 S 로 끝나며, MCP OAuth 계열 3건은 PR #23 이 아직 main 에 없어(로그에 #21·#22·#24 만) 여전히 막혀 있다. branding 오류 메시지 항목은 검증 하드닝 반려 전례와 같은 결이라 rejected 로 내렸다.
- 확신 없는 곳: `personal-pages.test.tsx` 의 렌더 헬퍼가 auth provider 를 어떻게 mock 하는지 열어 보지 않았다(과제서에 "그 파일을 먼저 열라" 고 적음). USER_GUIDE 의 로그인 절 번호도 미확인. `prettier --check` 경로는 직전 회차 기록에서 옮긴 것.
- 조심할 것: 안내는 오직 주소의 `sso=none` 만 보고 켠다(sessionStorage 플래그를 같이 보면 `silent-sso.ts` 와 두 경로가 됨). `sso=error` 문구를 추측으로 만들지 말 것. `login.png` 캡처 route 는 `sso` 를 안 붙이므로 캡처는 손대지 않는다. 검증 강화(sso/returnTo 값 검사)를 덤으로 얹지 말 것 — 반려 전례.
- 프로필은 1 일 전 것이 지금 코드와 맞아 다시 쓰지 않았다.
- [러너 09:36] scout done — silent SSO 가 거절된 뒤(`/login?sso=none`) 로그인 화면에 "자동 로그인하지 않은 이유" 한 줄 안내 (가치 3 / 위험
