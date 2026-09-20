# 과제서 — 2026-09-20 qurio

- 과제: 로컬(아이디·비밀번호) 로그인 뒤에도 `location.state.from` 딥링크로 복귀 + App.test.tsx 딥링크 케이스 추가 (가치 2 / 위험 1 / 작업량 S)
- 왜: `RequireAuth`(web/src/App.tsx:55)가 로그인이 필요했던 딥링크를 `state={{ from: returnTo }}`로 넘기고 LoginPage.tsx:47이 이미 `safeReturnTo(...)`로 계산해 SSO 버튼(214행)에만 쓰고 있는데, 로컬 로그인 성공 경로(LoginPage.tsx:81)는 `navigate(mustChangePassword ? '/password-renew' : '/workspace')`로 고정되어 있어 로컬 계정 사용자는 북마크·공유 링크로 들어와도 매번 /workspace 로 떨어진다. 한 줄만 바꾸면 SSO 와 로컬이 같은 규칙으로 복귀하고, `web/src/App.test.tsx` 의 기존 로그인 하네스에 딥링크 케이스가 없는 공백도 함께 메운다.
- 수용 기준:
  1) `/login` 에 `state.from = '/admin/mail?tab=deliveries'` 로 들어와 로컬 로그인에 성공하면 `/admin/mail?tab=deliveries` 로 이동한다(`replace: true` 유지).
  2) `state.from` 이 없거나 `safeReturnTo` 가 거부하는 값(`//evil.example`, `https://…`, 빈 문자열)이면 전과 같이 `/workspace` 로 이동한다.
  3) 응답의 `user.mustChangePassword === true` 면 `from` 이 있어도 `/password-renew` 로 이동한다(비밀번호 강제 변경 우선 — App.tsx:57 `RequireAuth` 와 같은 규칙).
  4) 로그인 실패(401)면 이동하지 않고 오류 문구가 보인다(기존 동작 회귀 없음).
  5) 테스트는 **기존 `web/src/App.test.tsx` 에 케이스를 더한다**(새 파일 대신). 이 파일에 이미 두 하네스가 있다(확인): `renderApp(initialEntry)`(20~26행, 실제 `<App/>` 전체 — `PublicOnly`/`RequireAuth` 포함)와 `renderLogin()`(45~58행, `<LoginPage/>` + `/workspace`·`/password-renew` 프로브 라우트), `mockPublicApi(config)`(28~43행)와 85~110행의 로컬 로그인 케이스(placeholder `'admin 또는 name@company.com'`·`'비밀번호 입력'`, 버튼 `'로그인'`, `/auth/login` fetch 응답). 위 1)~4) 를 **실제 `AuthProvider` + `apiClient` 를 거쳐(`vi.spyOn(globalThis, 'fetch')` 로 `/auth/login`·`/config`·`/version` 응답만 흉내)** 증명한다 — `useAuth`/`useNavigate` 를 통째로 모킹하지 말 것(운영자 규칙: 대역이 아니라 프로덕션 배선을 통과).
     - 1)은 `renderApp('<보호된 딥링크>')` 로 세션 없이 시작해(`oidcEnabled:false, oidcAutoLogin` 꺼짐 → 140~158행 silent SSO 케이스처럼 로그인 화면이 뜸) 로그인 폼을 제출하고 그 딥링크 화면의 식별 가능한 텍스트에 도달하는지로 판정한다 — 이렇게 하면 `PublicOnly` 상호작용까지 실제 라우터로 증명된다. 딥링크는 App 라우트 중 세션만 있으면 렌더되는 것으로 고른다(어느 라우트가 API 호출 없이 안정적으로 식별 텍스트를 내는지는 미확인 — `/workspace` 하위 등에서 구현자가 고를 것; 쿼리 문자열이 보존되는지도 같이 검사).
     - 2)~4)는 `renderLogin()` 을 `initialEntries={[{ pathname: '/login', state: { from } }]}` 를 받도록 확장(또는 인자 추가)하고 `*` 프로브 라우트(현재 `location.pathname + location.search` 를 찍음)를 더해 판정한다.
  6) `npm --prefix web run lint`(tsc)·`vitest run` 전체가 통과하고 기존 27+ 파일 테스트 수가 줄지 않는다.
  7) 임베디드 SPA(`internal/webui/dist`)를 재빌드해 바이너리에 반영한다(Makefile `build` 절차 그대로: `npm --prefix web run build` → `rm -rf internal/webui/dist && mkdir -p internal/webui/dist && cp -R web/dist/. internal/webui/dist/`). `git status` 로 dist 변경이 커밋에 포함됐는지 확인.
- 건드릴 파일:
  - `web/src/pages/LoginPage.tsx:submit`(81행) — `'/workspace'` 를 이미 계산된 `returnTo`(47행) 로 교체. `mustChangePassword` 분기와 `{ replace: true }` 는 그대로. 47행 주석을 "로컬 로그인도 같은 곳으로 돌아간다" 취지로 한 줄 보강.
  - `web/src/App.test.tsx` — 케이스 추가(위 5; 새 LoginPage.test.tsx 는 만들지 않음). `MemoryRouter initialEntries={[{ pathname: '/login', state: { from } }]}` 형태로 state 를 넣는다. 폼은 `label="아이디 또는 이메일"`(TextInput, 172행)·`label="비밀번호"`(PasswordInput, 182행)·`type="submit"` 버튼(201행)으로 찾는다. LoginPage 는 마운트 시 `apiClient.config()`·`apiClient.version()` 을 부르므로(64~70행, `Promise.allSettled`) fetch 스파이에서 그 두 경로는 200 JSON(`{ oidcEnabled: false }`, `{ version: '0.0.0' }`) 또는 404 를 돌려주면 된다 — allSettled 라 실패해도 렌더는 깨지지 않는다.
  - `internal/webui/dist/**` — 재빌드 산출물(수동 편집 금지).
- 검증 명령:
  - `npm --prefix web ci` (이 워크트리에는 `web/node_modules` 가 없음 — ls 로 확인, 먼저 설치)
  - `npm --prefix web test -- --run src/App.test.tsx` (해당 파일만 빠르게) → 수정 전 1)이 실패(red)하는 것을 먼저 확인
  - `npm --prefix web run lint && npm --prefix web run typecheck && npm --prefix web test -- --run`
  - `make build` 또는 위 dist 복사 절차 뒤 `go build ./...`(Go 코드는 안 바뀌므로 `go test ./...` 는 선택)
- 위험과 피할 것:
  - `web/src/App.tsx` 의 `PublicOnly`/`RequireAuth`, `web/src/lib/silentSso.ts`(`safeReturnTo` 규칙), `AuthContext`, 서버 `internal/httpapi/auth_handlers.go` 는 손대지 말 것. 특히 auth_handlers.go 는 지난 회차 PR(98602d8, 아직 main 에 없음, review-pending)이 같은 함수를 고치고 있어 충돌한다.
  - LoginPage 는 App.tsx:96 `PublicOnly` 안에 있다. `login()` 이 세션을 세팅하고 곧바로 `navigate()` 가 호출되므로 재렌더 시점에는 이미 딥링크 라우트로 옮겨가 `PublicOnly` 의 `/workspace` 리다이렉트를 타지 않아야 하지만, 이 상호작용은 **미확인** — 테스트의 라우트 트리에 `PublicOnly` 와 같은 모양의 래퍼(세션 있으면 `/workspace` 로 Navigate)를 `/login` 위에 한 겹 더 두고 1)이 여전히 통과하는지 확인하면 안전하다. 통과하지 않으면 원인을 노트에 적고 래퍼 없이 두지 말 것(그 경우 과제는 성립하지 않으므로 차선으로).
  - `/login?sso=none`(조용한 SSO 실패 뒤 복귀, auth_handlers.go:181)은 서버가 `return_to` 를 버려 state.from 이 없다 — 이번 범위 밖(서버·보호 경로 수정 필요). 넓히지 말 것.
  - `useNavigate`·`useAuth` 를 `vi.mock` 으로 갈아 끼우거나 소스 문자열을 검사하는 테스트는 반려 사유.
- 차선 후보: `gofmt -l` 검사를 `make lint` 에 추가(`test -z "$(gofmt -l cmd internal)"`) 하고 `.github/workflows/ci.yml` 의 `go vet` 단계 옆에 같은 한 줄 추가 — 지난 회차에 `gofmt -l cmd internal` 드리프트 0건 확인됨(이번 회차는 미실행). 워크플로는 보호 경로이니 CI 는 한 줄만.
