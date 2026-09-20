# 과제서 2026-09-20 — madi (base main@fd2c3b5)

- 과제: 로그인 화면 "회사 계정으로 로그인"(OIDC) 링크에 마지막 경로를 `return_to` 로 전달해 SSO 로그인 뒤 원래 보던 화면으로 복귀 (가치 3 / 위험 1 / 작업량 S)

- 왜: 비밀번호 로그인은 `App.tsx:1622-1636` 에서 `preferences.last_path`/`localStorage["madi.last_path"]` 로 원래 화면에 복귀하지만, OIDC 링크(`App.tsx:402`)는 `href="/api/v1/auth/oidc/start"` 고정이라 서버 기본값 `/app` 으로만 떨어진다(서버 `oidcStart` 는 `integrations_oidc.go:75` 에서 silent 여부와 무관하게 `return_to` 를 이미 받아 `oidc_attempts.return_to` 에 저장하고 콜백이 그 경로로 302 한다 — 기반이 전부 있고 링크만 안 쓴다). 링크에 안전한 `/app…` 경로를 붙이면 SSO 사용자도 문서 링크를 눌러 로그인한 뒤 그 문서로 돌아간다. 서버 코드는 손대지 않는다.

- **선행 조건(반드시 첫 커밋)**: 이 base 에는 `web/dist/.gitkeep` 이 없어 러너 검증의 `go build ./...`(npm 빌드보다 먼저 돎)가 `web/embed.go:7 pattern all:dist: no matching files found` 로 실패한다 — 09-19 두 회차가 이 이유로 verify-failed. `git cherry-pick e8e0960`(build: track web/dist/.gitkeep…, `.gitignore`·`web/dist/.gitkeep`·`web/vite.config.ts`·README 4파일) 을 먼저 적용하고 `git clean -xfd web/dist && go build ./...` 가 exit 0 인지 확인할 것. 이 커밋은 review-pending PR(auto/2026-09-19-2134)에도 같은 내용으로 들어 있어 병합 시 동일 변경으로 충돌 없이 합쳐진다(반려된 접근 아님). govulncheck(grpc GO-2026-6348)·apk lock 드리프트로 CI 두 단계가 실패하는 것은 PR 무관 외부 드리프트로 같은 pending PR 이 고치므로 **이번 회차에서 go.mod·deploy/runtime-apk.lock 은 건드리지 말 것**(중복 PR 방지). 회차 노트에 CI 실패 시 그 두 단계 이름을 적어 두면 심사가 알아본다.

- 수용 기준:
  1) `web/src/auth/silentSso.ts` 에 순수 함수 `oidcLoginStartUrl(lastPath: string | null | undefined): string` 추가 — `lastPath` 가 `/app` 또는 `/app/` 로 시작(`safeReturnTo` 규칙 포함: `/` 시작, `//` 제외)하면 `/api/v1/auth/oidc/start?return_to=<encodeURIComponent(lastPath)>`, 그 외(빈 값·null·`/admin/...`·`//evil`·`https://…`·`/login`)는 `return_to` 없이 `/api/v1/auth/oidc/start` 를 돌려준다(효과 없는 `return_to=%2Fapp` 을 붙이지 않는다). `prompt=none` 은 절대 붙이지 않는다(자동 로그인 함수 `silentSsoStartUrl` 과 별개, sessionStorage 표시도 쓰지 않음).
  2) `App.tsx:402` 의 `href` 가 `oidcLoginStartUrl(localStorage.getItem("madi.last_path"))` 를 쓴다(localStorage 접근 예외는 try/catch 로 null 취급 — `silentSso.ts` 의 `browserEnv`/저장소 예외 처리 관례 참고). `IdentityLogin.tsx:25` 의 SAML 링크는 **건드리지 않는다**(main 의 `samlStart` 는 return_to 를 읽지 않아 붙여도 효과 없음 — 83239b9 는 미병합).
  3) `tests/silent-sso.mjs` 에 `oidcLoginStartUrl` 검사 추가: `/app/documents/abc?x=1` → 인코딩된 return_to 포함, `null`/`""`/`/admin/settings`/`//evil.test`/`https://evil.test/app`/`/login` → 정확히 `/api/v1/auth/oidc/start`, 어떤 입력에도 `prompt=none` 미포함. `node --test tests/silent-sso.mjs` 통과.
  4) 서버 끝단 증명: `internal/server/integrations_oidc_test.go` 의 `TestPostgresOIDCSilentLogin` 에 **prompt=none 없이** `?return_to=%2Fapp%2Fdocuments%2Fdeep` 로 start → 콜백 302 Location 이 `/app/documents/deep` 인 경우 1건 추가(388행은 비조용+unsafe 만, 343·357·376행은 silent 만 검사하므로 지금은 비조용+safe 조합이 비어 있다). 클라이언트가 붙이는 값과 서버가 읽는 값이 같은 규칙(`/` 시작, `//` 제외)으로 끝까지 흐르는지 이 한 건이 증명한다.
  5) `docs/admin-guide.md` 자동 로그인/OIDC 절에 "로그인 화면의 회사 계정 버튼은 마지막으로 보던 `/app` 경로로 복귀한다" 한 줄 추가 후 `node scripts/build-docs.mjs` 로 manuals HTML 재생성(다른 HTML diff 가 생기면 되돌릴 것). `tests/docs.mjs` 는 돌리지 말 것(Chromium 없을 수 있고 스크린샷 바이너리 diff 를 만든다).

- 건드릴 파일:
  - `web/src/auth/silentSso.ts` — `safeReturnTo` 아래에 `oidcLoginStartUrl` 추가(기존 함수 무수정).
  - `web/src/App.tsx:402` — `Login` 컴포넌트의 OIDC `<a href>` 만 교체; import 는 파일 상단 기존 `./auth/silentSso` import 에 이름 추가.
  - `tests/silent-sso.mjs` — 검사 블록 추가(기존 형식: `assert.equal`, 파일 끝의 return_to 절 근처).
  - `internal/server/integrations_oidc_test.go` — `TestPostgresOIDCSilentLogin` 안 388행 근처에 비조용+safe 케이스 1건(테스트 헬퍼 `start(query)` 재사용).
  - `docs/admin-guide.md` + `docs/manuals/*.html`(생성물).

- 검증 명령(좁게 — 전체 스위트·-race 전체·브라우저 시험 금지, 러너 예산 교훈):
  - `git clean -xfd web/dist && go build ./... && go vet ./internal/server` (선행 조건 확인)
  - `cd web && npm ci && npm run build` (tsc 포함; 빌드 뒤 `git status --short web/dist` 가 빈 출력)
  - `node --test tests/silent-sso.mjs`
  - `npx prettier --check web/src/auth/silentSso.ts web/src/App.tsx` (App.tsx 는 기존 경고 수치와 같으면 통과로 본다 — 09-17 기록)
  - 임시 PostgreSQL 17 컨테이너 뒤 `MADI_TEST_POSTGRES_DSN=… go test -count=1 -run 'TestPostgresOIDCSilentLogin|TestPostgresOIDCCodePKCENonceAndVerifiedIdentity' ./internal/server` (DB 를 못 띄우면 그 사실을 회차 노트에 "미확인" 으로 적고 1~3 만으로 마감)
  - `gofmt -l internal/server` 빈 출력, `node scripts/build-docs.mjs`

- 위험과 피할 것:
  - `oidcReturnTo`(서버)·`safeReturnTo`·`silentSsoStartUrl`·`shouldAttemptSilentSso` 는 수정 금지 — 자동 로그인 루프 가드가 여기에 걸려 있다.
  - `/admin` 경로는 return_to 로 보내지 않는다(로그인 전에는 role 을 모르고, 비관리자가 `/admin` 에 떨어졌을 때의 화면 동작은 미확인). 관리자 복귀 손실은 의도된 축소이며 노트에 적을 것.
  - `.github/workflows/*`, `go.mod`, `deploy/runtime-apk.lock`, `scripts/licenses.mjs` 무변경.
  - `tests/docs.mjs` 실행으로 생기는 `docs/assets/product-*` 스크린샷 변경은 커밋하지 말 것.
  - 커밋 메시지는 영어 conventional(`feat: carry last /app path into OIDC login link`), 선행 cherry-pick 은 별도 커밋으로 유지.

- 차선 후보: 러너 검증 예산·`web/dist/.gitkeep` 자리표시자·apk lock 드리프트·`-run` 패턴 검증 관례를 `CLAUDE.md` 로 명문화 (가치 3 / 위험 1 / S) — 코드 무변경, 선행 조건(e8e0960 cherry-pick)은 동일하게 필요. 내용은 09-19 세 회차 기록(전체 스위트 금지, go.mod 바꾸면 `scripts/licenses.mjs` 재생성, lock 바꾸면 image-smoke 69/52 카운트, `tests/docs.mjs` 스크린샷 되돌리기)을 소스에서 확인해 옮길 것.
