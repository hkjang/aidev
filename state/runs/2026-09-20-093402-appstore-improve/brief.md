# 과제서 (2026-09-20)

- 과제: silent SSO 가 거절된 뒤(`/login?sso=none`) 로그인 화면에 "자동 로그인하지 않은 이유" 한 줄 안내 (가치 3 / 위험 1 / 작업량 S)
- 왜: 자동 로그인(Silent SSO)이 켜진 설치에서 Keycloak 세션이 없으면 콜백이 `/login?sso=none(&returnTo=…)` 으로 보내는데(`internal/httpapi/auth_handlers.go:124 silentRefusalPath`), `LoginPage`(`web/src/pages/auth-pages.tsx:17`)는 `useSearchParams` 로 `returnTo` 만 읽고 `sso` 는 읽지 않아 사용자는 "왜 자동으로 안 들어갔지" 를 알 길이 없고 ADMIN_GUIDE 4.x 장애 대응 표(539행)가 "주소에 `sso=none` 이 붙어 있는지 보라" 고 관리자에게만 알려 준다. 안내 한 줄이면 사용자가 곧바로 "회사 계정으로 SSO 로그인" 을 누르면 된다는 것을 알고, 관리자가 주소창을 읽어 달라고 할 일이 없어진다.
- 수용 기준:
  1) `/login?sso=none` 으로 열면 SSO 버튼 위(또는 바로 아래)에 `notice` 스타일의 한 줄 — 예: "회사 계정 세션이 없어 자동으로 로그인하지 않았습니다. 아래 버튼으로 로그인하세요." — 가 보인다. `role="status"` 로 두어 스크린리더가 읽는다.
  2) `/login` (sso 없음), `/login?returnTo=%2Fsubmit`, `/login?sso=` 등 `sso=none` 이 아닌 경우에는 화면이 바이트 동일하게 그대로다(기존 `login.png` 캡처가 변하지 않아야 함 — 캡처 route 는 `web/e2e/visual.spec.ts:126` 이 `/login?returnTo=%2Fsubmit` 만 쓴다).
  3) `sso=none` 이 있을 때 SSO 버튼의 `href` 와 bootstrap 로그인 뒤 `navigate(returnTo)` 는 전과 같이 `returnTo` 를 그대로 쓴다(안내는 표시만 하고 흐름을 바꾸지 않는다). 즉 `/login?sso=none&returnTo=%2Fmy%2Fapps` 에서 SSO 링크는 `/api/v1/auth/oidc/login?returnTo=%2Fmy%2Fapps`.
  4) 테스트가 증명할 것: (a) `sso=none` 이면 안내 텍스트가 렌더링되고 (b) 없으면 렌더링되지 않으며 (c) `sso=none` 이 있어도 SSO 링크 href 가 returnTo 만 담는다. `web/src/pages/` 에 `auth-pages.test.tsx` 가 아직 없으므로 새로 만들되 `personal-pages.test.tsx` 의 렌더 헬퍼(QueryClient·MemoryRouter·auth provider mock 방식)를 그대로 따를 것 — 그 파일을 먼저 열어 어떤 provider 를 감싸는지 확인.
- 건드릴 파일:
  - `web/src/pages/auth-pages.tsx:LoginPage` — `const ssoRefused = params.get("sso") === "none";` 를 두고 `{oidcEnabled && ssoRefused && (<div className="notice mb-5" role="status">…</div>)}` 를 SSO 버튼(109행) 바로 위에 추가. `oidcEnabled` 가 아니면(설정이 꺼진 뒤 옛 주소로 들어온 경우) 안내를 내지 않는다.
  - `web/src/pages/auth-pages.test.tsx` — 새 파일, 위 (a)(b)(c) 3건. `api.publicConfig` 를 `oidcEnabled: true, oidcConfigured: true, oidcAutoLogin: true` 로 mock.
  - `docs/USER_GUIDE.md` — 로그인 절(1.x 또는 2.x, "회사 계정으로 SSO 로그인" 을 설명하는 자리)에 "자동 로그인이 켜져 있어도 회사 계정 세션이 없으면 이 안내와 함께 로그인 화면이 뜬다" 한 문장. 가이드를 바꾸면 관례대로 `node /mnt/c/Users/USER/projects/aidev/tools/guide/md2pdf.mjs` 로 PDF 재생성(직전 회차 방식: 표지 버전은 현재 v2.11.2 유지). ADMIN_GUIDE 539행 장애 대응 표의 "확인" 칸에 "로그인 화면에 '자동으로 로그인하지 않았습니다' 안내가 떠 있는지" 를 덧붙이면 좋으나 선택.
  - 건드리지 말 것: `web/src/features/auth/silent-sso.ts`(규칙은 이미 `sso=none` 을 보고 재시도를 막음 — 그대로), `internal/httpapi/auth_handlers.go`, `internal/auth/*`, `web/src/app/providers.tsx`, `internal/webui/dist`.
- 검증 명령 (저장소 루트에서, 모두 exit 0 이어야 함):
  - `npm --prefix web test` (현재 55 건 + 새 3 건)
  - `npm --prefix web run lint`
  - `npx --prefix web prettier --check web/src` (직전 회차는 `prettier --check src` 를 web 안에서 돌림)
  - `npm --prefix web run build && ./scripts/check-offline-assets.sh web/dist`
  - `./scripts/check-docs.sh` · `./scripts/check-env-contract.sh`
  - Go 는 변경이 없으므로 `go vet ./... && go build ./cmd/server` 만.
  - Playwright(`npm --prefix web run test:e2e`, Chromium 필요)는 login 캡처 route 가 `sso` 를 안 붙이므로 화면이 안 변한다 — 돌릴 시간이 있으면 visual desktop 만; 캡처 파일은 바이트 동일해야 하고 달라지면(렌더링 흔들림) HEAD 로 되돌릴 것.
- 위험과 피할 것:
  - 검증 하드닝(입력 길이·제어문자·에러 메시지 다듬기) 류 PR 은 사람이 두 번 반려했다(2026-09-08/10) — 이 과제는 사용자 안내 추가이지 검증이 아니지만, "덤으로" `sso` 값 검증이나 `returnTo` 검증 강화를 얹지 말 것.
  - 안내 문구를 `sso` 값 외의 조건(예: sessionStorage 의 ATTEMPTED 플래그)으로 켜지 말 것 — 같은 값을 읽는 경로가 둘이 되면 `silent-sso.ts` 와 어긋난다. 오직 주소의 `sso=none` 만 본다.
  - `params.get("sso")` 로 `error` 값도 프런트 규칙에 있으나 서버는 `none` 만 만든다(`auth_handlers.go:125`) — `error` 용 문구를 추측으로 만들지 말 것(미확인 경로).
  - 캡처 manifest(`check-docs.sh`)는 새 라우트에만 캡처를 요구한다 — 새 라우트 없음. 캡처를 건드리지 않는 것이 기본.
  - `notice` 클래스는 같은 파일 138·181행에서 이미 쓰는 스타일이라 CSS 추가 불필요.
- 차선 후보: 소유자 카드의 즐겨찾기 하트가 게시되지 않은 앱에서 조용한 no-op — `web/src/features/apps/app-card.tsx:14 publiclyViewable` 이 이미 있고 32행 `viewable` 이 계산되므로 하트 버튼(83행 부근)을 `viewable` 일 때만 렌더링하고 `app-card.test.tsx`/`personal-pages.test.tsx` 에 "검토 대기 앱 카드에 즐겨찾기 버튼이 없다" 1건 추가(가치 2 / 위험 1 / S). 공개 카탈로그(`showStatus` 아님)는 변하지 않아야 한다.
