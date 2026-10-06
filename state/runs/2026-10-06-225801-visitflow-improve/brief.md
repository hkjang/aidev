- 과제: 「내 API 키」 화면이 받아 보지 못한 정책을 사실처럼 말하고 폐기 실패를 삼키는 것 닫기 (가치 3 / 위험 1 / 작업량 S)

- 왜: `web/src/pages/KeysPage.tsx:51` 의 `policy` 는 하드코딩 초기값 `{allowedScopes:["read","write","mcp"], defaultExpiryDays:90, maxActiveKeys:10}` 을 들고 있고, `load()`(56-59행)가 실패하면 그 추측값이 그대로 남아 122-125행 info Alert 이 「현재 허용 Scope는 read, write, mcp이며 기본 만료는 90일, 활성 키 한도는 10개입니다」를 **사실로 단정**한다(관리자가 `read` 1개·30일·3개로 바꿔 두었어도 그렇게 말한다). 같은 추측값이 `openCreate`(80행)·FormGroup(223행)의 Scope 체크박스를 만들어 서버 `validScopes`(internal/app/keys.go:205)가 거부할 범위를 고르게 하고(`createAPIKey` 59-62행 → 400 `invalid_scopes`), `revoke`(94-98행)는 try/catch 가 없어 되돌릴 수 없는 폐기를 확인까지 받은 뒤 서버가 404 `활성 API 키가 없습니다`(keys.go:199-202)를 줘도 화면이 전혀 바뀌지 않고 unhandled rejection 으로만 남는다.

- 수용 기준:
  1) `/api/v1/api-key-policy` 또는 `/api/v1/api-keys` 조회가 실패하면 info Alert 이 구체적 Scope·일수·한도 **숫자를 말하지 않고**, 정책을 불러오지 못했다는 안내로 바뀐다(기존 error Alert 은 그대로 서버/네트워크 메시지를 보여 준다).
  2) 조회 실패 상태에서 「키 만들기」 대화상자가 추측한 Scope 체크박스 3개를 제시하지 않고, 정책을 모른다는 사실을 말한다. 「생성」이 정책 없이 서버에 400 을 받으러 가지 않는다(버튼 비활성 또는 명시적 안내 — 비활성이면 그 이유가 화면에 읽힌다).
  3) 조회가 **성공**하면 현재 동작이 그대로다: 서버가 준 Scope 만 체크박스로 나오고, Alert 이 서버가 준 숫자를 말하고, 오류 문구는 뜨지 않는다.
  4) 폐기(DELETE)가 실패하면 서버의 한국어 메시지가 기존 `error` Alert 에 뜨고 unhandled rejection 이 남지 않는다. 성공하면 목록이 갱신된다(현재 동작 유지).
  5) 테스트가 증명할 것: 수정 전 번들에서는 조회 실패 화면이 `read, write, mcp`·`90일`·`10개` 를 보여 주고 체크박스 3개가 뜨며 DELETE 실패에 Alert 이 0개라는 것, 수정 후에는 셋 다 반대라는 것 — 실제 `npm run build` 번들·실서버·실제 브라우저로.

- 건드릴 파일:
  - `web/src/pages/KeysPage.tsx` — `load`(56-59행)에 `policyLoaded`(또는 `policyFailed`) 플래그 하나를 더해 성공 시에만 세우고, 122-125행 Alert·`openCreate`(80행)·`openEdit`(81행)·FormGroup(223행)·「생성」 `disabled`(247행)가 **그 한 값만** 읽게 한다. `revoke`(94-98행)를 같은 파일의 `rotate`(82-93행) 꼴로 try/catch 로 감싼다.
  - `web/e2e/visit-flow.spec.ts` — 영구 스펙 2~3개(① 정책 조회 실패 ② 정책 성공 정상 경로 ③ DELETE 실패). `page.route` 로 `**/api/v1/api-key-policy` 를 `abort()`, `**/api/v1/api-keys/*` DELETE 를 `fulfill({status:404, body:{"error":{"code":"not_found","message":"활성 API 키가 없습니다"}}})`.

- 검증 명령:
  - `cd web && npm ci && npm run lint && npm test && npm run build` (lint=tsc -b, test=vitest run. `web/node_modules` 는 지금 **없다**(확인) — `npm ci` 가 먼저다. 직전 회차 기록상 기준 98 passed)
  - `bash scripts/local-e2e.sh` (실제 dist 임베드·전용 DB·실서버·실제 브라우저. 수 분 소요, 인자 미지원)
  - **기준 스펙 수는 기록을 믿지 말고 수정 전 실행으로 직접 받을 것.** `web/e2e/visit-flow.spec.ts` 가 유일한 스펙 파일이고 `grep -c "test("` 가 20인데 직전 회차 기록은 `21 passed` 다 — 한쪽이 어긋나 있다(원인 미확인).
  - `go build ./... && go vet ./... && gofmt -l . && git diff --check` (Go 미변경 확인용)

- 위험과 피할 것:
  - **서버를 고치지 말 것.** `apiKeyPolicy`(keys.go:13-18)가 `getSetting` 오류를 `_` 로 버리고 `strings.Fields("")`→`[]` 를 주는 것과 `validScopes`(keys.go:210-212)가 빈 설정에 `"read write mcp"` 로 **되돌아가는 것**은 같은 설정을 다르게 읽는 두 경로다(확인). 다만 `0005_baseline.sql:450` 이 `'read write mcp'` 를 시드하고 `settings.go:287-298` 이 빈 값 저장을 막으므로 빈 설정은 DB 오류로만 도달한다 — 대역 없이 재현할 수 없으니 **이번 회차에서 건드리지 말고** 아이디어로만 남긴다. 이번 과제는 화면이 자기가 모르는 것을 말하지 않게 하는 것뿐이다.
  - `policy.allowedScopes` 가 **정상적으로 1개뿐일 수 있다**(관리자가 `read` 만 허용). 배열이 짧거나 비었다는 사실을 실패 근거로 삼지 말 것 — 실패는 전용 플래그에서만 나와야 한다(v2.8.17 `referenceFailed` 와 같은 판단).
  - `api.ts` 시그니처·CSRF 경로는 건드리지 않는다(모든 화면 공용). `revoke` 의 `confirm()` 문구·되돌릴 수 없다는 경고는 그대로 둔다.
  - 보호 경로(auth/session/migrations/settings/release·Docker·workflows) 전부 손대지 않는다. `go` 코드 0줄.
  - `web/vite.config.ts:10` 의 `include:["src/**/*.test.ts"]` 때문에 `.tsx` vitest 는 **조용히 0개로 수집**된다. 컴포넌트 테스트를 쓰려 하지 말 것 — 증거는 e2e 로 남긴다(지난 4회차와 동일).
  - 경로는 확인했다: `web/src/App.tsx:92` 의 `<Route path="profile/keys" …>` 로 **가드가 없다** — 로그인 후 `page.goto("/profile/keys")` 로 바로 열린다(AppShell.tsx:172 의 프로필 메뉴 「내 API 키」로도 간다). `web/e2e/` 전체 grep 에서 `api-key`·`API 키`·`KeysPage` 가 **0건**이라 기존 스펙과 locator 충돌이 없다(확인). 단 AppShell:172 메뉴는 직전 회차 프로필 대화상자 스펙이 쓰는 그 메뉴이므로 메뉴를 경유하지 말고 `goto` 를 쓰는 쪽이 안전하다.
  - `KeysPage` 는 `lazy(() => import(...))`(App.tsx:21) 이다 — `page.route` 를 `goto` **전에** 걸고, Suspense 로딩 뒤 대화상자가 뜰 때까지 기다릴 것.
  - 끝나면 `cmd/visitflow/webdist/` 에 추적된 스텁 `index.html` 하나만 남아야 한다. 복원은 local-e2e 의 임시 복사본으로 — `git checkout --` 금지(과거에 미커밋 변경을 잃었다).

- 차선 후보: `web/src/pages/ScannerPage.tsx:21`(재확인 — `useEffect(() => { api<ReferenceData>("/api/v1/reference-data").then((x) => {…}); return () => stopCamera(); }, [])`) 의 `.then(...)` 에 catch 가 없어 로비 선택이 조용히 비고 unhandled rejection 이 남는다(가치 2 / 위험 2 / S). 같은 패턴을 그대로 쓸 수 있으나 같은 파일이 카메라 cleanup·scope 를 들고 있어 위험이 한 단 높다 — 카메라 코드는 건드리지 말 것.
