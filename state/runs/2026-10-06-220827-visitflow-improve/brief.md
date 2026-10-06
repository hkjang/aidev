- 과제: 프로필 대화상자가 기준정보 조회 실패를 「부서·대리 담당자 없음」으로 단정하고 메일 알림 토글 저장 실패를 되돌리지 않는 것 닫기 (가치 3 / 위험 1 / 작업량 S)
- 왜: `web/src/components/AppShell.tsx:110` 이 `if (!reference) setReference(await api<ReferenceData>("/api/v1/reference-data"))` 를 try/catch 없이 호출한다 — 대화상자는 바로 앞 줄(`:109` 의 `setProfileOpen(true)`)에서 이미 열려 있으므로, 이 조회가 실패하면 「소속 부서」는 `미지정` 하나, 「대리 담당자」는 `지정 안 함` 하나만 남은 화면이 **정상처럼** 보이고(`:157`·`:159` 가 `reference?.departments` / `reference?.hosts ?? []` 를 읽는다) 오류는 `:152` 의 `void openProfile()` 때문에 unhandled rejection 으로 사라지며, `:111` 의 메일 알림 조회까지 아예 실행되지 않아 「메일 알림」 섹션이 통째로 사라진다. 같은 파일 `saveMailPrefs`(`:113-116`)는 PUT 전에 `setMailPrefs(next)` 로 화면을 먼저 바꾸고 실패 시 오류만 띄우므로, 스위치·체크박스는 **서버에 저장되지 않은 상태를 저장된 것처럼** 계속 보여 준다. 고치면 이 대화상자가 서버의 답을 그대로 전한다(v2.8.15 알림 재발송·v2.8.16 대피 명단과 같은 계열의 결함).
- 수용 기준:
  1) `/api/v1/reference-data` 가 실패하면 프로필 대화상자에 한국어 오류(`기준정보를 불러오지 못했습니다` — `AdminPage.tsx:68` 과 같은 문구 권장)가 보이고, 부서·대리 담당자 칸이 「선택지를 불러오지 못했습니다」임을 사용자가 알 수 있다. **빈 목록을 정상으로 단정하지 않는다.**
  2) 그 실패에서도 `/api/v1/profile/notifications` 조회는 실행되어 메일 알림 섹션이 뜬다(현재는 건너뛴다).
  3) 실패를 나타내는 표시는 **조회 실패 플래그**에서 나와야 하고 배열이 빈 것에서 나오면 안 된다 — `ReferenceData.hosts` 는 `types.ts:83` 에서 optional 이라 대리 담당자 목록이 **정상적으로** 비어 있을 수 있다(혼자인 테넌트). 정상 빈 목록에는 오류 문구가 뜨지 않아야 한다.
  4) `PUT /api/v1/profile/notifications` 가 실패하면 스위치/체크박스가 **이전 값으로 되돌아가고** 기존 `profileError` Alert 에 서버 메시지가 보인다.
  5) 성공 경로는 그대로다: 부서 목록·대리 담당자 목록이 뜨고, 토글 저장이 되고, 「저장」이 프로필을 저장한다. 테스트는 수정 전 번들에서 실패하고 수정 후 통과하는 짝으로 증명한다.
- 건드릴 파일:
  - `web/src/components/AppShell.tsx` — `openProfile`(104행): `:110` 을 try/catch 로 감싸고 실패를 상태로 남긴다(`setProfileError` 재사용 또는 전용 `referenceFailed` state 하나). `:111` 의 메일 알림 조회를 참조 데이터 성공 여부와 무관하게 실행되는 위치로 옮긴다(가장 작은 변경: 메일 조회 줄을 try 블록 **앞**으로 이동). 두 select 의 `helperText`/안내가 그 한 값만 읽게 한다. `saveMailPrefs`(113행): 진입에서 이전 값을 잡아 두고 `catch` 에서 `setMailPrefs(previous)` 로 롤백한다.
  - `web/e2e/visit-flow.spec.ts` — 영구 스펙 1~2개. `page.route("**/api/v1/reference-data", (r) => r.abort())` 로 ①을, `page.route` 로 `PUT **/api/v1/profile/notifications` 만 `fulfill({status:500, body: JSON.stringify({error:{code:"x",message:"…"}})})` 해서 ④를 고정한다(GET 은 통과시켜야 섹션이 뜬다 — `request.method()` 로 구분).
  - 프로덕션 파일 1개. 이 범위를 넘기지 말 것.
- 검증 명령:
  - `cd web && npm run lint && npm test && npm run build` (lint=`tsc -b`, test=`vitest run`, 기준 98개)
  - `bash scripts/local-e2e.sh` (실제 dist 임베드 + 새 PostgreSQL + 실서버 + 실제 브라우저. 기준 18 passed. 수분 소요)
  - `go build ./... && go vet ./... && gofmt -l . && git diff --check` (서버 미변경 확인용)
- 위험과 피할 것:
  - Go 서버·`web/src/api.ts`·`web/src/auth.tsx`(세션/OIDC)·마이그레이션·워크플로는 건드리지 말 것. 이 결함은 전부 화면 쪽이다.
  - `web/vite.config.ts:10` 의 `include: ["src/**/*.test.ts"]` 때문에 `.tsx` 단위 테스트는 **조용히 0개로 수집**되고 testing-library 도 없다 — React state 배선 증거는 반드시 local-e2e(실서버+실브라우저)로 남긴다. 손으로 만든 대역을 증거로 쓰지 말 것.
  - MUI select 의 `helperText`·required 별표·중복 버튼 이름 때문에 locator 는 실제 DOM 으로 확인할 것. 「대리 담당자」 select 에는 이미 `helperText` 가 있으니 문구를 덮지 말고 합치거나 Alert 로 분리한다.
  - `reference` 는 `if (!reference)` 가드로 재시도되므로, 실패 후 대화상자를 닫고 다시 열면 재조회된다 — 롤백·오류 state 를 대화상자 열 때 초기화해 옛 오류가 남지 않게 할 것.
  - 다른 호출부(`ScannerPage.tsx:21` 도 catch 없음, `AdminPage.tsx:136`)는 **이번에 건드리지 말 것**. 파일 수가 늘면 사람 손을 다시 탄다.
  - 미확인: ① 수정 전 번들에서 `route.abort()` 가 실제로 unhandled rejection 과 빈 select 를 그대로 재현하는지는 브라우저로 확인하지 않았다(코드 경로상 성립하지만 실측 전) ② 기존 18개 e2e 중 프로필 대화상자를 여는 스펙이 있는지 확인하지 않았다 — 있으면 그 스펙과 locator 가 충돌하지 않는지 볼 것.
- 차선 후보: `KeysPage.tsx:57` 의 `Promise.all(...).then(...)` 에 `.catch` 가 없어 API 키·정책 조회 실패가 조용히 사라진다 (가치 2 / 위험 1 / S, 프로덕션 1파일, 같은 유형).
