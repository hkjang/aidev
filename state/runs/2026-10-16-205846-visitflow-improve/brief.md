- 과제: 관리 Dashboard 의 「운영 지표」가 조회 실패를 「대기열 0건·잠긴 계정 0건·활성 세션 0개」로 단정하는 것 닫기 (가치 3 / 위험 1 / 작업량 S)
- 왜: `web/src/pages/AdminPage.tsx:29` 의 `api<OperationalMetrics>("/api/v1/admin/metrics").then(setMetrics).catch(() => setMetrics(null))` 는 실패를 **어디에도 알리지 않고** `null` 로 삼키고, 같은 `Dashboard` 함수(30행 「운영 지표」 카드)는 `metrics?.queueBacklog ?? 0` 꼴로 여섯 값을 전부 `0`/`v-` 로 렌더링한다 — 지표가 끊긴 장애 상태가 화면에서는 "알림 대기열 0건 · 대기열 최장 지연 0분 · 잠긴 계정·주소 0건 · 활성 세션 0개 · 활성 API 키 0개" 라는 **완벽히 건강한 시스템**으로 보인다(이 화면을 보고 있는 관리자 자신의 세션이 있으므로 "활성 세션 0개" 는 성립 자체가 불가능한 값이다). 운영자가 모니터링 공백을 정상으로 오판하는 것을 막고, 최근 5회(roster·profile·keys·scanner)와 같은 "실패를 사실로 단정하지 않는다" 규칙을 이 화면에도 적용한다.

- 수용 기준:
  1) `/api/v1/admin/metrics` 응답이 실패하면(차단·500 모두) 「운영 지표」 카드가 **숫자를 말하지 않고**(0 도 `v-` 도 아님) 한국어로 지표를 불러오지 못했다고 알린다. 원인 문구를 덧붙일 때는 한국어 문장이 먼저, 원인은 괄호로 뒤에 둔다(네트워크 실패의 `e.message` 는 브라우저의 영어 "Failed to fetch" — v2.8.16/17 선례).
  2) 조회 중 / 실패 / 정상이 서로 구별된다. 특히 **서버가 정상 200 으로 `queueBacklog: 0` 등 실제 0 을 준 경우에는 실패 문구가 절대 뜨지 않는다**(정상 0 과 실패를 같은 화면으로 만들지 말 것 — 지금 결함의 본질이 그 혼동이다).
  3) 실패 후 `page.on("pageerror")` 에 미처리 rejection 이 없다(지금은 `.catch` 가 있으므로 rejection 은 없다 — 이 단정은 수정이 catch 를 깨지 않았음을 고정하는 가드다).
  4) `/api/v1/admin/dashboard`(27행, 이미 `setError` 로 상단 error Alert 에 보고한다) 경로의 기존 동작은 바뀌지 않는다.
  5) 영구 Playwright 스펙이 수정 전 번들에서 실패하고(위 1·2) 수정 후 통과한다. 정상 경로 스펙은 수정 전에도 통과해야 한다(선택자 유효성 + 수용 기준 2 의 영구 가드).

- 건드릴 파일:
  - `web/src/pages/AdminPage.tsx` : `Dashboard` 함수(26~30행)만. `metrics` 상태를 "0 을 말할 수 없는" 모양으로 바꾼다 — 권장: `const [metrics, setMetrics] = useState<{ state: "loading" } | { state: "failed"; message: string } | { state: "ready"; value: OperationalMetrics }>({ state: "loading" })` 처럼 판별 가능한 한 값으로 두고, 「운영 지표」 카드의 여섯 행이 **그 한 값만** 읽게 한다. v2.8.18(KeysPage)에서 플래그 대신 `| null` 타입으로 가서 `tsc` 가 모든 독자에게 처리를 강제하게 한 것이 효과가 있었다 — 여기서도 `?? 0` 를 타입으로 불가능하게 만드는 쪽을 택하라. 실패 시 카드 본문은 라벨을 남기고 값을 `—` 로 두거나 숫자 목록을 아예 `Alert severity="warning"` 으로 대체한다(둘 중 하나로 일관되게).
  - (선택) 한 값 → 표시 행 변환만 순수 함수로 뽑아 `web/src/adminMetrics.ts` + `web/src/adminMetrics.test.ts` 로 vitest 가 덮게 해도 좋다. `web/vite.config.ts:10` 이 `src/**/*.test.ts` 만 수집하므로 `.tsx` 에 테스트를 두면 **조용히 0개로 수집된다**. 프로덕션 파일은 그래도 2개 이내.
  - `web/e2e/visit-flow.spec.ts` : 스펙 2개(실패 1 + 정상 1) 추가. 기존 `login(page)` 헬퍼 사용 후 `await page.goto("/admin/dashboard")`(`App.tsx:89` 의 `admin/:section`; `AdminPage` 는 `section ?? "dashboard"`). **기존 스펙 중 `/admin/dashboard` 를 여는 것은 없다**(admin 스펙은 `/admin/resources`·`/admin/notification-settings` 만 연다) — locator 충돌 없음. 실패 재현은 `page.route("**/api/v1/admin/metrics", (r) => r.abort())` 로 하고, **`/api/v1/admin/dashboard` 는 차단하지 말 것**(상단 error Alert 가 생겨 단정이 섞인다). `sw.js` 는 roster 외 `/api/` 를 캐시하지 않으므로 `serviceWorkers: "block"` 은 불필요(v2.8.16·17 에서 확인됨). 라우트는 `login` **뒤에** 설치한다.
  - (선택) `docs/ADMIN_GUIDE.md` 한 문장. **PDF 재생성은 하지 말 것**(이 저장소 관례).

- 검증 명령:
  - `cd web && npm ci && npm run lint && npm test && npm run build` (lint=`tsc -b`, test=vitest; 기준 98 passed — 단위 테스트를 더하면 그만큼 늘어야 한다)
  - `bash scripts/local-e2e.sh` (실제 dist 임베드·새 PostgreSQL·실서버·실제 Chrome. 기준 26 passed. 수정 **전** 번들로 먼저 돌려 새 스펙이 실패하는 것을 받아 두고, 수정 후 통과를 확인하라. 수분 소요)
  - `go build ./... && go vet ./... && gofmt -l .`(빈 출력) `&& go test ./... -count=1`(DSN 미설정이면 DB 통합 SKIP — 서버 미변경 확인용), `git diff --check`
  - 끝난 뒤 `git status --porcelain` 이 의도한 파일만 보이고 `cmd/visitflow/webdist/` 에 추적된 스텁 `index.html` 하나만 남아야 한다. 복원은 local-e2e 의 임시 복사본으로 — **`git checkout --` 를 쓰지 말 것**(과거에 미커밋 작업을 잃은 기록).

- 위험과 피할 것:
  - 서버(`internal/app/metrics.go:139 adminMetrics`, `metricsSnapshot`)·`web/src/api.ts`·`web/public/sw.js` 는 0줄 건드리지 말 것. 서버는 실패 시 이미 `notFoundOrServer` 로 제대로 답한다 — 이번 결함은 **화면이 그 답을 버리는 것**뿐이다.
  - `Dashboard` 상단의 네 개 `MetricCard`(오늘 방문자·현재 체류·승인 대기·알림 실패)는 `counts` 를 읽고 실패 시 상단 error Alert 가 뜨므로 이번 범위 밖이다. 같이 고치려 들면 파일·행이 불필요하게 커진다.
  - `AdminPage.tsx` 는 `VisitAdmin`·`Resources`·`Statistics`·`Audit`·`Operations`·`APIPage` 를 한 파일에 담고 있다. **`Dashboard` 밖으로 손을 내밀지 말 것**(특히 `Resources` 의 `resourceEndpoints` 주석과 `erase`/watchlist 경로 — 감사·개인정보 파기 경로다).
  - 보호 경로(auth/session/migrations/workflows)는 건드릴 일이 없다.
  - 실패 근거로 "값이 0 이다" 를 절대 쓰지 말 것 — 신규 설치에서 `queueBacklog`·`lockedAccounts` 는 정상적으로 0 이다. 실패는 전용 상태에서만 나와야 한다(v2.8.18 의 `policy.allowedScopes` 교훈과 같은 자리).
  - `OperationalMetrics`(`web/src/types.ts:212`)에는 `notifications: Record<string, number>` 등 카드가 쓰지 않는 필드도 있다. 타입을 바꾸지 말 것.

- 차선 후보: 만료된 API 키의 「수정」·「회전」 버튼이 계속 활성으로 보인다 (`web/src/pages/KeysPage.tsx:195/200` 의 `disabled` 는 `revokedAt` 만 보지만 서버 `internal/app/keys.go` 의 update/rotate 는 `expires_at > now()` 를 요구한다 — 가치 2 / 위험 2 / S). 실제 만료 시각을 만드는 e2e 설계가 미확인이라 2순위.
