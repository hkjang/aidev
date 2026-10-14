# 수정 과제 (fix-round) — PR #4 의 CI 실패

> 이 세션에서 **아무것도 실행할 수 없었다**: `npm ci`/`npm run check` 는 샌드박스가 거부(`This command requires approval`, 비대화형이라 승인 불가), `gh auth status` 도 거부. 워크트리에 `node_modules` 없음. 따라서 아래 진단은 **코드·커밋 메시지·워크플로를 읽어서 세운 가설**이고, 실패 로그로 확인한 것이 아니다(미확인). 구현자는 ①번 분기 판정을 **먼저** 하라.

- 과제: PR #4 (`auto/2026-10-14-2333`, 커밋 ea36d85/4213cea) 의 `check` 실패를 분기 판정하고, 저장소 안의 원인일 때 `db-breaker.test.ts` 의 **노트북 속도에 맞춘 벽시계·연결횟수 단언**만 좁게 고치기 (가치 4 / 위험 2 / 작업량 S)
- 왜: 직전 회차가 DB 차단기(16파일, `packages/core/src/__tests__/db-breaker.test.ts` 신규 232줄)를 넣고 CI 가 두 번 같은 이유로 실패했다. 저장소 안의 유일한 새 CI 위험은 이 테스트 파일에 든 **실제 소켓 + 실제 벽시계** 단언들(`dropped > 3`, `< 50ms`, `< 200ms`, `< 3s`, `< 5s`)이며, 2코어 공유 러너에서 10개 테스트 파일과 동시에 돌면 노트북에서만 성립하는 상수들이다. 고치면 "빨리 실패·시도 안 함" 이라는 **증명 내용은 그대로 두고** 하드웨어 속도 가정만 제거해 게이트가 다시 믿을 수 있게 된다.

## ① 먼저 할 분기 판정 (10분 안)
1. `gh run list --branch auto/2026-10-14-2333 --limit 5` / `gh api repos/hkjang/workkit.irumx.app/actions/runs/<id>/jobs` 로 **잡 주석과 `steps` 배열**을 본다.
2. 주석이 *"The job was not started because recent account payments have failed or your spending limit needs to be increased."* 거나 `steps: []`·소요 2~3초·`output` 전부 null 이면 → **원인은 저장소 밖(Actions 결제·지출 한도)**. `.github/workflows/check.yml` 을 **한 줄도 건드리지 말고**(과거 2회 교훈), 로컬에서 CI 순서를 재현해 green 을 증명한 뒤 원장에 그렇게 기록하고 끝낸다. 아래 ②는 하지 않는다.
3. 실패 단계가 **`Lint, types, unit tests`** 이고 로그에 `db-breaker.test.ts` 가 보이면 → ② 를 한다. 실패 단계가 다른 것(e2e / ui-flows / a11y / Kits)이면 ② 를 버리고 **그 로그가 가리키는 한 곳만** 좁게 고친다(아래 "차선" 참고).

## ② 저장소 안의 원인일 때: 건드릴 파일
- `packages/core/src/__tests__/db-breaker.test.ts` — **이 한 파일만.** 프로덕션 코드(`db.ts`, `errors.ts`, `respond.ts`, `vendor/postgres/*`)는 건드리지 않는다.
  - :111 `expect(dropped).toBeGreaterThan(3)` — "끊긴 연결은 블립으로 한 번 더 시도한다" 를 증명하려는 것. 러너가 느리면 2.5초 예산이 먼저 끝나 `3` 으로 떨어진다. 시도 횟수 대신 **`dropped % 3 === 0 && dropped >= 3`** 처럼 "드라이버가 연결당 3회로 포기했다" 를 보고, 재시도 여부는 러너 속도에 걸리지 않는 단언으로 옮기거나 이 줄을 `toBeGreaterThanOrEqual(3)` 로 두고 재시도 증명은 별도 단위 테스트(예산 함수)로 옮긴다.
  - :125 `expect(Date.now() - t).toBeLessThan(50)` 과 :229 `< 200` — 둘 다 **바로 윗줄의 `expect(dropped).toBe(tried)`(= 연결을 아예 시도하지 않았다)가 이미 증명**한다. 벽시계 상수는 증명에 기여하지 않으므로 지우거나, 첫 실패 소요시간 대비 **상대 비교**(`fast * 10 < slow`)로 바꾼다. 이것은 느슨하게 만드는 것이 아니라 하드웨어 가정을 빼는 것이다 — 행동 단언(`db_unavailable`, 503, `retryAfter > 0`, `location` 없음, 점검 화면 문구)은 하나도 지우지 말 것.
  - :112 `< 5_000`, :133 `< 4_000`, :160 `< 3_000`, :212 `< 5_000` — 이것들은 제품 요구("몇 초 안에 포기")라서 **그대로 두는 것이 원칙**이다. 로그가 바로 이 줄에서 실패했을 때만, 2.5초 예산 + 연결 대기값으로 설명되는 최소 여유까지만 올리고(예: 5_000 → 8_000) 왜 그 값인지 주석으로 남긴다.
- 테스트 파일 상단 주석에 "무엇을 증명하는가" 가 이미 있으니 그 계약을 바꾸지 말 것.

## 수용 기준
1) `gh` 로 읽은 실패 단계·주석을 원장에 **그대로 인용**한다(결제 차단이면 그 문장, 테스트 실패면 실패한 단언 줄).
2) 결제 차단이면 `.github/workflows/**` diff 0줄이고, 로컬 재현(아래 검증 명령 전부)이 통과한다.
3) 저장소 안 원인이면 `db-breaker.test.ts` 가 **부하 아래서도** 통과한다 — 바꾼 뒤 `npx vitest run` 전체를 동시에 돌려(단일 파일만이 아니라) 통과를 보이고, 차단기 코드를 한 줄 꺾었을 때(예: `db.ts` 의 조기 실패를 끄기) 여전히 **빨간불이 되는 것**을 보여 증명력이 남아 있음을 확인한다.
4) 프로덕션 파일·워크플로 변경 없음(또는 있다면 왜 불가피했는지 원장에 적는다).

## 검증 명령 (이 저장소에서 실제로 도는 것, CI 순서)
```
npm ci --no-audit --fund=false
npx vitest run packages/core/src/__tests__/db-breaker.test.ts
npm run check                 # lint + typecheck 4개 패키지 + vitest 11파일 (135 passed 기대)
npm run build
npm run kits && cp dist-kits/manifest.json /tmp/m1.json && npm run kits && cmp /tmp/m1.json dist-kits/manifest.json && npm run kits:test
node scripts/ci-env.mjs
```
`e2e`/`ui`/`a11y` 는 Docker PostgreSQL + 가짜 서버(:18997/:18998) + Playwright 가 필요하고, 이 워크트리에서는 공유 dev DB(:55447)와 충돌해 키트 다운로드 7개가 환경 때문에 깨진다(회차 기록 확인). 돌릴 거면 **전용 포트의 DB 컨테이너**를 띄워 격리할 것.

## 위험과 피할 것
- **워크플로를 느슨하게 하지 말 것**: 검사 항목 삭제·`continue-on-error`·`timeout` 늘리기·단계 건너뛰기 전부 금지(운영자 반복 지시).
- `packages/core/vendor/postgres/**` 는 건드리지 말 것. `vendor-postgres.test.ts` 가 upstream 과 줄 단위로 비교하고, PATCH 블록은 `PATCHED.md` 와 쌍이다. (참고로 읽다 본 것 — 미확인 의심거리: `src/connection.js` 의 PATCH 5 가 `errored(...)` 를 부른 **뒤에도** `if (initial) return reconnect()` 로 떨어진다. `errored()` 가 `initial` 을 비우지 않으면 오류를 올리면서 재연결도 계속한다. 이번 과제에서 고치지 말고, 사실이면 다음 회차 후보로 올릴 것.)
- `auth/*`, `domain/payments.ts`, `db/` 마이그레이션은 이번 과제와 무관 — 열지 말 것.
- 테스트의 한국어 단언 문구·`dropped`/`tried` 계측 구조는 유지. 실제 소켓을 가짜 객체로 바꾸지 말 것(운영자 지시: 손으로 주입한 대역은 배선 결함을 못 본다).

## 차선 후보
- **실패 단계가 `Accessibility` 또는 `Purchase flow through the screens` 일 때**: 커밋 ea36d85 의 확인 줄은 `npm run check` / `e2e 155` / `tools-flows 55` 만 적고 **`ui-flows`·`a11y` 는 적지 않았다** — 작성자가 돌리지 않은 유일한 단계들이고, 같은 커밋이 `apps/web/src/styles/global.css`(+21, `.wk-degraded` 고정 배너), `App.tsx`(배너 삽입), `Shell.tsx`(푸터 `v{config.version}` → 조건부)를 바꿨다. 배너는 `degraded` 일 때만 렌더되므로 CI(정상 DB)에서는 안 보이는 것이 맞지만, 푸터 문구 변화(`© 옆자리 워크킷` 뒤 `· v…` 가 빠질 수 있음)를 `ui-flows.mjs`/`a11y.mjs` 가 문자열로 보고 있으면 거기서 깨진다. 단, `scripts/` 전체를 `옆자리 워크킷|config.version|v${` 로 grep 해 보니 `ui-flows.mjs`·`a11y.mjs` 는 푸터 버전 문구를 보지 않았다(히트는 `smoke-prod.mjs:53`, `import-kits.mjs:150`, `deploy-pages.sh:68` 뿐) — 그러니 이 가설은 **약하다**. 실패 로그가 실제로 이 단계를 가리킬 때만 그 로그가 지목한 한 곳을 고치고, 검사 항목은 빼지 말 것.
