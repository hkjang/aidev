# PR 처리기 노트 2026-09-20-140035-madi-shepherd — madi PR #9
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-09-20-130359-madi-improve)
# 회차 노트 2026-09-20-130359-madi-improve — madi
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 13:04] base pinned — main@fd2c3b5
- [러너 13:04] autonomy release — 

## 정찰 노트
- 선택 이유: base 가 main@fd2c3b5 로 09-17~19 의 모든 PR(?sso=error, MCP OAuth, SAML return_to, .gitkeep, grpc/apk) 이 미병합이라 그 위에 얹는 후보는 전부 착수 불가·충돌 위험. OIDC 링크 return_to 는 main 에 서버 기반(oidcStart 75행)이 이미 있고 App.tsx:402 한 줄만 비어 있어 독립적으로 완결되며 pending PR 과 겹치는 파일이 없다.
- 확신 없는 곳: (1) `git cherry-pick e8e0960` 이 fd2c3b5 위에 충돌 없이 적용되는지 미확인(부모가 main 이라 될 것으로 추정). (2) 비관리자가 `/admin` return_to 로 떨어졌을 때 화면 동작 미확인 → `/app` 접두사만 허용하도록 좁혔다. (3) DB 통합 테스트는 컨테이너가 필요 — 못 띄우면 노트에 미확인으로 남기라고 적었다.
- 구현자 주의: 선행 cherry-pick 없이는 러너의 `go build ./...` 가 09-19 두 회차와 똑같이 실패한다. go.mod·apk lock·워크플로는 pending PR 영역이라 무변경. `tests/docs.mjs` 는 돌리지 말 것. 검증은 `-run` 패턴으로 좁게.
- 프로필은 09-19 것이 현 코드와 일치해 다시 쓰지 않았다(단, "base 가 main 이고 미병합 PR 9개" 는 이 노트가 대신 적는다).
- [러너 13:07] scout done — 로그인 화면 "회사 계정으로 로그인"(OIDC) 링크에 마지막 경로를 `return_to` 로 전달해 SSO 로그인 뒤 원래 보�

## 구현 노트
- 무엇/왜: 로그인 화면 OIDC 링크가 `/api/v1/auth/oidc/start` 고정이라 SSO 사용자는 항상 `/app` 으로 떨어졌다. 순수 함수 `oidcLoginStartUrl`(silentSso.ts) 이 localStorage `madi.last_path` 가 같은 출처 `/app/…`·`/app?…` 일 때만 `return_to` 를 붙이고 App.tsx Login 링크가 이를 쓴다. 서버 무변경. 커밋 5dc7f95, 선행 cherry-pick 3be8100(e8e0960, 충돌 없음).
- 확신 없는 곳: (1) 실제 DOM 의 href 는 브라우저로 안 봤다 — tsc 통과 + 순수 함수 node 테스트 + 서버 Go 테스트만. (2) `/app?view=x` 처럼 `/app` 뒤 쿼리만 있는 경로도 부착하도록 했는데 App 이 실제로 그런 경로를 저장하는지는 미확인(저장 코드는 pathname+search). (3) `lastPath()` 는 Login 렌더마다 localStorage 를 읽는다 — 렌더 비용은 무시할 수준으로 봤다.
- 일부러 하지 않은 것: `/admin/*` 는 return_to 로 보내지 않음(로그인 전 role 미상, 비관리자가 /admin 에 떨어졌을 때 화면 동작 미확인 — 관리자 복귀 손실은 의도된 축소). `/app` 자체는 효과 없는 `return_to=%2Fapp` 이라 미부착. SAML 링크(IdentityLogin.tsx:25)는 main 의 samlStart 가 return_to 를 안 읽어 무변경. 서버 저장 `preferences.last_path` 는 로그인 전 못 읽으므로 미사용. go.mod·apk lock·워크플로 무변경(pending PR 영역).
- 다음 역할 주의: Go 케이스는 `MADI_TEST_POSTGRES_DSN` 이 있어야 돈다(비DB 실행 시 skip). 서버가 이미 지원하는 동작이라 새 Go 검사는 구현 전에도 통과한다 — 대신 기대 Location 을 틀리게 바꿔 실패함을 확인해 검사가 살아 있음을 증명했다. App.tsx prettier 경고는 변경 전과 동일한 기존 드리프트. CI 의 govulncheck·Build service image 실패는 이 PR 무관(pending PR 이 고침). `tests/docs.mjs` 는 돌리지 않았다.
- [러너 13:12] brief accepted — 채택 — 파일·행 근거(App.tsx:402 href 고정, oidcStart 75행의 return_to, 테스트 388행 근처의 비조용+safe 공백, e8e0960 충돌 없음)가
- [러너 13:13] verify passed — 검증 5개 통과 (auto)
- [러너 13:13] pr created — https://github.com/hkjang/madi/pull/9
- [러너 13:13] guard held — web/src/auth/silentSso.ts 

## 심사 노트
- 확인: node tests/silent-sso.mjs PASS + 변이 2건(접두 검사 넓히기·인코딩 제거)에서 실패 확인; 임시 postgres:17 컨테이너로 TestPostgresOIDCSilentLogin PASS(1.75s); go build/vet ./..., npm ci+tsc -b+vite build 통과, .gitkeep 재생성·트리 깨끗.
- 확인: 클라이언트는 /app/·/app? 접두의 같은 출처 경로만 보내고 서버 oidcReturnTo 가 start·callback 양쪽에서 재검증 — 열린 리다이렉트 아님. 기본값·공개 경로·권한·마이그레이션 변경 없음. prompt=none 미부착.
- 확인: 3be8100 은 이미 두 번 승인된 e8e0960 의 cherry-pick(반려 이력 없음), pending PR 과 동일 패치라 머지 충돌 없음.
- 못 본 것: 실제 브라우저 DOM href, 전체 -race 스위트, tests/docs.mjs.
- 권고 merge / risk medium(oidc 링크를 건드리지만 서버 무변경·되돌리기 쉬움). 차단 소견 없음.
