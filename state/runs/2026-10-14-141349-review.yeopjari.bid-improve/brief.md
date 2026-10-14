- 과제: PR #3 의 쿼터 수정을 main 위에 다시 올리면서, 새 e2e '하루 한도' 검사의 벽시계 의존(한국 자정 경계에서 86400초 오차)을 제거한다 (가치 4 / 위험 1 / 작업량 S)

- 왜: `retryAfterSeconds` 의 9시간 오차 수정(PR #3, 브랜치 `auto/2026-10-14-0958`)은 코드가 옳고 로컬 검증까지 끝났는데 **머지되지 않았다** — 이 워크트리의 main@6ea982c 에는 `packages/core/src/__tests__/quota.test.ts` 가 없고 `quota.ts` 도 옛 판이다(`ls packages/core/src/__tests__/`, `git diff --stat main auto/2026-10-14-0958` 로 확인). 그 PR 에 남은 유일한 실제 결함은 새 e2e 검사가 서버 응답 시각이 아니라 `Date.now()` 로 한국 자정 경계를 다시 계산하는 것이어서, 15:00Z 를 수백 ms 걸치는 실행에서 `expected` 가 86400초 튀어 CI 가 거짓 실패한다. 둘을 함께 끝내면 수정이 main 에 들어가고 CI 가 시각 의존 없이 돈다.

- 수용 기준:
  1) `node scripts/e2e.mjs` 의 '하루 한도' 섹션이 서버 응답의 `date` 헤더(= 서버가 `retry-after` 를 계산한 시각)를 기준 시각으로 써서, 실행 시각이 한국 자정을 걸쳐도 `expected` 가 86400초 튀지 않는다. 허용 오차는 ±5초를 **넓히지 말 것**(느슨하게 만드는 것은 금지) — 기준 시각을 바꿔 좁히는 방향으로만 고친다.
  2) `packages/core/src/quota.ts` 의 `nextWindowReset` 과 `retryAfterSeconds`, `packages/core/src/__tests__/quota.test.ts` 8건이 main 위에 그대로 들어간다. `windowStart` 는 손대지 않는다(저장 키 `quota_counters.window_start` 가 바뀌면 배포 시점 진행 중인 카운터가 리셋된다).
  3) 테스트가 증명하는 것: (a) 단위 — 고정 시각에서 일·월·분 창의 다음 경계가 KST 실제 경계와 같다(KST 오전·자정 1분 전·자정 정각·UTC 전날 인스턴트·월 롤오버·12월→1월). (b) e2e — 실제 HTTP 429 의 `retry-after` 헤더가 다음 한국 자정까지의 초와 ±5초 안에서 일치하고, 그 판정이 테스트 실행 시각과 무관하다.

- 건드릴 파일 (프로덕션 1개):
  - `packages/core/src/quota.ts` — `nextWindowReset(window, at)` 추가, `retryAfterSeconds` 가 그것만 쓰게. (PR 브랜치 그대로 가져오면 된다: `git checkout auto/2026-10-14-0958 -- packages/core/src/quota.ts`)
  - `packages/core/src/__tests__/quota.test.ts` — 신규(같은 방식으로 가져오기). 주석의 "최대 9시간" 은 "정확히 9시간" 이 맞다(상수 오차임이 계산으로 확인됨, 이전 비평 지적).
  - `scripts/e2e.mjs` — '하루 한도' 섹션(main@6ea982c 기준 1114행 뒤). `const refusedAt = Date.now()` 를 응답 `date` 헤더 기준으로 바꿀 것: `const refusedAt = Date.parse(overCap.headers.get('date'))` (없으면 `Date.now()` 폴백). `kstToday`/`resetAt`/`expected` 계산식은 그대로 두고 기준 시각만 교체한다.

- 검증 명령 (전부 이 저장소에서 실제로 도는 것):
  1) `npm run check` — eslint + 타입 3개 + 단위(기준선 118건 → 신규 8건 포함 126건)
  2) 로컬 DB 준비: `docker run -d --name review-pg -e POSTGRES_PASSWORD=devpass -e POSTGRES_DB=review -p 55443:5432 postgres:16-alpine` → `node scripts/ci-env.mjs` → `npm run build -w @review/server` → `bash scripts/dev-reset.sh`
  3) `node scripts/e2e.mjs` — 기준선 218 passed / 0 failed, 신규 2건 포함 **220 passed / 0 failed** 여야 한다
  4) 경계 재현(고친 것의 증거): `TZ=UTC` 로 돌려도 같고, 서버/테스트 시각을 15:00Z 직전으로 밀어 돌렸을 때 **고치기 전에는 `expected≈86400+` 로 FAIL, 고친 뒤 PASS** 임을 보일 것. 시각 조작이 어렵다면 최소한 `date` 헤더를 읽는 경로가 실제로 값을 받는다는 것(헤더가 `undefined` 가 아님)을 출력으로 확인하고, 그 사실을 보고에 적을 것.
  5) 화면 변화 없음 → `scripts/a11y.mjs`·`ui-flows.mjs` 는 필수 아님. 돌리면 `npx playwright install --with-deps chromium` 이 먼저 필요.

- 위험과 피할 것:
  - **CI 'check' 의 3초 실패 원인은 아직 미확인이다.** 증거: `started_at 01:14:38Z → completed_at 01:14:41Z`, 실패 단계 이름 비어 있음, 로그 비어 있음, 같은 시각 서로 무관한 5개 저장소의 `check` 가 전부 2~3초 failure(이전 수리 노트 2건). 단계가 하나도 실행되지 않았다는 뜻이고, 그러면 이 diff 는 원인이 아니다. **다만 아무도 확인하지 않은 단서가 하나 남아 있다**: `ci-fe4eaba9edd2.json` 의 `output.annotations_count: 2`. 구현자는 가장 먼저 이것을 읽을 것 —
    `gh api repos/hkjang/review.yeopjari.bid/check-runs/115627512290/annotations`
    (정찰 샌드박스에서는 `gh` 실행이 승인 거부돼 읽지 못했다.) 주석이 워크플로/설정 오류(예: "Failed to initialize container postgres:16-alpine", 워크플로 구문 오류, 러너 라벨 불일치)를 가리키면 **그것이 이번 과제이고 위 과제는 뒤로 미룬다**. 주석이 지출 한도·Actions 장애를 가리키면 코드로 고칠 것이 없으니 위 과제를 그대로 하고, 보고에 주석 원문을 인용할 것.
  - 두 번의 수리 회차가 "고칠 것이 없다, 재시도하면 된다" 로 끝나 두 번 다 verify-failed 가 됐다. **같은 결론만 다시 내지 말 것** — 주석을 읽어 사실을 확정하고, 확정되지 않으면 위 과제로 실제 결함(플레이크)을 없애 CI 재실행에 거짓 실패 요인을 남기지 말 것.
  - `.github/workflows/check.yml` 을 **느슨하게 만들지 말 것**(단계 삭제·`continue-on-error`·타임아웃 늘리기·`|| true` 금지). 주석이 서비스 컨테이너 초기화 실패를 가리킬 때만, 이미지 레지스트리 고정 같은 최소 변경을 근거와 함께.
  - `db/migrations/`·`packages/core/src/auth/*`·`http/pipeline.ts`·`routes/sso.ts` 는 이번 과제와 무관하다 — 건드리지 말 것.
  - `scripts/seed-demo.mjs` 는 호스트가 `127.0.0.1`/`localhost` 가 아니면 거부한다. e2e 는 깨끗한 DB 를 암묵적 전제로 하므로 `dev-reset.sh` 직후에 돌릴 것.

- 차선 후보: `scripts/e2e.mjs` 의 '하루 한도' 검사만 격리해 넣기 — PR #3 을 다시 올리는 경로가 러너 쪽에서 이미 처리돼 main 에 들어와 있다면(구현자가 `git log` 로 확인), `quota.ts`·`quota.test.ts` 는 건드리지 말고 e2e 의 기준 시각 교체 1파일로만 끝낼 것. 그 경우 3순위는 "`util/text.ts` 나머지 순수 함수 단위 테스트 보강(mailboxKey·maskEmail·koreanParticle·excerpt·tidyBody·clientIpFrom, 프로덕션 파일 0개)" [3/1/S].
