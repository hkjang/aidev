- 과제: 로그인 뒤 원래 보려던 화면으로 돌아가기 (깊은 링크 returnTo) (가치 3 / 위험 2 / 작업량 S)
- 왜: 세션이 없을 때 `Protected`(`web/src/App.tsx:15-18`)가 `<Navigate to="/login" replace />` 로만 보내서 원래 주소(예: 링크로 공유받은 `/admin/maps`)가 그 자리에서 사라지고, 로그인에 성공해도 `submit` 의 `navigate("/")`(`web/src/pages/LoginPage.tsx:41`)와 `if (user) return <Navigate to="/" replace />`(같은 파일 34행) 때문에 항상 좌석맵 첫 화면으로 떨어진다. 로그인 화면의 SSO 단추도 `href="/api/v1/auth/oidc/start"`(170행)라 returnTo 없이 출발한다. 서버(`internal/app/auth.go:299-301` 의 start, `442-445` 의 callback)와 조용한 SSO(`web/src/lib/silentSso.ts:103` 의 `silentSsoStartUrl`)는 이미 `returnTo` 를 주고받고 검증까지 하므로, **서버를 건드리지 않고 화면 쪽 두 군데만 이으면** 깊은 링크가 로그인을 건너 살아남는다.
- 수용 기준:
  1) 로그아웃 상태에서 `/admin/maps` 같은 보호 경로를 열면 주소가 `/login?returnTo=%2Fadmin%2Fmaps` 가 되고, 로컬 계정으로 로그인하면 `/` 가 아니라 `/admin/maps` 로 간다. 쿼리스트링이 있는 깊은 링크(`/admin/maps?floor=3`)도 쿼리째 보존된다.
  2) `/` 를 열어 로그인 화면으로 밀려난 경우와 `/login` 을 직접 연 경우에는 `returnTo` 가 붙지 않고(주소가 그냥 `/login`), 로그인 뒤 기존대로 `/` 로 간다 — `web/e2e/helpers.ts:9-14` 의 `login()` 이 그대로 동작해야 한다.
  3) `config.oidcEnabled` 일 때 SSO 단추의 `href` 가 `/api/v1/auth/oidc/start?returnTo=<encode(깊은 링크)>` 이고, returnTo 가 없으면 지금처럼 파라미터 없는 `/api/v1/auth/oidc/start` 이다.
  4) 밖으로 나가는 값(`//evil.example`, `https://evil.example`, `/\evil`)은 기존 `safeReturnTo`(`web/src/lib/silentSso.ts:89-95`)를 지나 `/` 로 접히고, 열린 리다이렉트가 생기지 않는다.
  5) 테스트가 증명할 것 — (vitest) 주소를 **쓰는 쪽(App)과 읽는 쪽(LoginPage)이 같은 헬퍼를 쓴다**는 것을 왕복(round-trip)으로: 깊은 링크 → 로그인 주소 → 되읽은 returnTo 가 원래 값과 같고, 위험한 값은 `/` 가 된다. (E2E) 로그아웃 상태로 `/admin/maps` 진입 → 로컬 로그인 → `/admin/maps` 도착까지 실제 서버·브라우저로.
- 건드릴 파일:
  - `web/src/lib/silentSso.ts` — 이미 있는 `safeReturnTo` 를 재사용해 헬퍼 두 개를 같은 파일에 더한다: `loginPathFor(pathname, search)`(→ `/login` 또는 `/login?returnTo=…`, 기본 경로 `/` 는 파라미터 없이)와 `returnToFrom(search)`(→ `safeReturnTo` 를 통과한 값, 없으면 `/`). **문자열을 두 파일에 각각 짓지 말 것** — 지금 seaton 에서 같은 값을 두 곳이 따로 읽어 어긋난 전례가 있다(도면 aria-label 과 상세의 '구역 불일치' 문구). 이름이 silentSso 와 안 맞으면 `web/src/lib/loginReturn.ts` 로 새로 빼도 좋으나, `safeReturnTo` 는 한 곳에만 두고 import 로 쓸 것(복제 금지).
  - `web/src/App.tsx:15-18` `Protected` — `useLocation()` 으로 현재 위치를 읽어 `<Navigate to={loginPathFor(location.pathname, location.search)} replace />`. `Manager`(19-30행)의 `Navigate to="/"` 는 권한 부족이지 미로그인이 아니므로 건드리지 말 것.
  - `web/src/pages/LoginPage.tsx` — 이미 `useLocation`/`useNavigate` 를 import 해 두었다(14행). 34행 `if (user) return <Navigate to="/" replace />`, 41행 `navigate("/")`, 170행 SSO `href` 세 자리를 `returnToFrom(location.search)` 로 계산한 하나의 값으로 바꾼다. `queryError`(48행)가 이미 `location.search` 를 파싱하므로 그 옆에 둘 것.
  - `web/src/lib/silentSso.test.ts` (또는 새 헬퍼 파일의 짝 테스트) — 위 5)의 왕복·거절 케이스.
  - `web/e2e/login.spec.ts` — 깊은 링크 케이스 1건 추가. 끝에서 `helpers.ts` 의 기존 흐름을 깨지 않도록 새 테스트 안에서만 `page.goto("/admin/maps")` 로 시작할 것.
  - (선택) `docs/USER_GUIDE.md` 한 줄. 고치면 `python3 scripts/build-docs.py USER_GUIDE` 로 HTML 만 다시 굽고 **PDF 는 굽지 말 것**(미머지 브랜치와 이진 충돌 — 09-17 이후 회차의 관례).
- 검증 명령:
  - `cd web && npm ci && npm run lint && npm test && npm run build` (lint 는 tsc 검사다)
  - `cd /home/hkjang/.cache/auto-improve-wt/seaton && go vet ./... && go test ./... && gofmt -l .` (서버를 안 건드려도 무출력 확인)
  - E2E: `docker build -t seaton:e2e .` 로 수정본 이미지를 굽고 PostgreSQL 16 과 함께 띄운 뒤 `cd web && E2E_BASE_URL=http://127.0.0.1:<port> E2E_USERNAME=admin E2E_PASSWORD=<부트스트랩값> npx playwright test e2e/login.spec.ts e2e/seatmap.spec.ts` (준비 포함 수 분). **역검증**: 화면 변경 전 이미지에서 새 spec 이 실제로 빨갛게 되는 것을 먼저 볼 것 — 이 저장소의 최근 성공 회차(09-23, 09-24)가 모두 그렇게 했다.
- 위험과 피할 것:
  - `internal/app/auth.go` 는 **손대지 말 것**(보호 경로: 세션·CSRF·OIDC). start 와 callback 이 각각 `safeReturnTo` 로 이미 검증하므로 서버 변경은 필요 없다. 서버까지 고쳐야 할 것 같으면 그건 설계가 틀어진 신호다.
  - 열린 리다이렉트가 이 과제의 유일한 보안 위험이다. 클라이언트 검증은 UX 용일 뿐이고 실제 방어는 서버에 있다 — 그래도 클라이언트에서 직접 `window.location.assign(returnTo)` 같은 것을 하지 말고 React Router `navigate` 로만 옮길 것.
  - `web/src/auth.tsx:71-73` 의 조용한 SSO(`shouldAttemptSilentSso` → `beginSilentSso`)는 이미 `window.location.pathname + search` 를 returnTo 로 넘긴다. 같은 헬퍼로 바꾸려다 기존 동작(`prompt=none`, 시도 플래그)을 바꾸지 말 것 — 이 경로는 **읽기만 하고 그대로 둔다**.
  - `helpers.ts:14` 의 `waitForURL((url) => !url.pathname.startsWith("/login"))` 때문에, 로그인 뒤 주소에 `/login` 이 남으면 모든 E2E 가 멎는다. 수용 기준 2)를 먼저 확인할 것.
  - 미머지 브랜치 `auto/2026-09-16-1022`(메일)·`auto/2026-09-18-0413`(MCP OAuth)이 남아 있다. 둘 다 `App.tsx`·`LoginPage.tsx`·`silentSso.ts` 를 바꾸는지는 **미확인** — 충돌이 보이면 문서 파일(ADMIN_GUIDE/PDF)만 피하면 된다.
  - 45분 안에 끝나는 크기다. 로그인 실패 재시도·세션 만료 후 자동 재로그인 같은 인접 주제로 넓히지 말 것(보류 아이디어에 이미 위험 3 으로 적혀 있다).
- 차선 후보: 목록 핸들러가 `rows.Err()` 와 `Scan` 오류를 삼켜 부분 목록을 200 으로 돌려주는 것 고치기 — `internal/app/seats.go:36` 은 `if rows.Scan(...) == nil` 이라 스캔이 실패한 좌석이 조용히 사라지고, 루프 뒤 `rows.Err()` 검사가 없어 연결이 끊기면 잘린 목록이 200 으로 나간다. 같은 꼴이 `maps.go:27,72,126`, `employees.go:22,78,293`, `keys.go:28`, `detection.go:468`, `dashboard.go:317`, `mcp.go:113,127,153` 에 있고, 제대로 된 예는 `grid.go:150-158`(Scan 오류 반환 + `rows.Err()`)이다. 1순위가 성립하지 않으면 seats·maps·employees 세 파일만 `grid.go` 꼴로 맞추고 500 으로 올릴 것. **주의**: 이 저장소의 Go 테스트에는 DB 하네스가 없어(`internal/app/*_test.go` 는 analyzer·auth·mcpoauth·tracking·vlm 뿐) 단위 테스트로 스캔 실패를 증명할 수 없다 — 손으로 만든 대역을 쓰지 말고 실제 PostgreSQL 로 확인하거나, 증명이 안 되면 범위를 줄일 것.
