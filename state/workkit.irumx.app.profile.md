# 옆자리 워크킷 (workkit.irumx.app) 프로필 (2026-10-14)

- 목적: 가입 없이 브라우저 안에서만 도는 직장인 실무 도구함(주간보고 정리기·엑셀/CSV 대조기·업무 인수인계 작성기) + 업무 자동화 키트 상점. 운영 이룸엑스. 현재 v0.3.0, 수익화는 꺼 둠.
- 스택: TypeScript(ESM, Node 22) · npm workspaces · React(apps/web, Mantine 계열) · Cloudflare Pages + Function + Hyperdrive → Supabase PostgreSQL · R2 · Cloudflare Worker 스케줄러 · vitest · Playwright · eslint 9 / typescript-eslint.
- 구조:
  - `packages/core` — 서버 도메인 전부(라우터·auth/session·orders/payments/refunds·settings·mail·audit·db). 라우트는 `src/routes/*`.
  - `packages/tools` — **브라우저에서 도는 순수 로직**: `table/`(csv.ts, xlsx-read.ts, xlsx-write.ts, types.ts), `compare/`(engine.ts, report.ts), `weekly/model.ts`, `handover/model.ts`, `doc/docx.ts`, `project/`. 시험은 `src/__tests__/`.
  - `apps/server` — 개발·운영 서버 진입점. `apps/web` — React 화면(`src/tools/*` 가 도구 UI, `compare.worker.ts` 가 워커).
  - `functions/[[path]].ts` — Cloudflare Pages Function. `workers/scheduler` — 예약 작업.
  - `db/` 마이그레이션, `scripts/` 운영·시험 스크립트, `docs/` 문서 11종, `seed-kits/`·`fixtures/`.
- 빌드·테스트:
  - `npm ci` (자동 개선 워크트리에는 node_modules 가 없다 — 안 하면 `vitest: not found`)
  - `npm run check` = lint + typecheck + `vitest run` (빠름, 1차 관문)
  - `npx vitest run <파일>` 로 한 파일만 돌릴 수 있다
  - `npm run build` / `npm run e2e`(PostgreSQL 16 + 가짜 옆자리 + 가짜 토스 필요) / `npm run tools`·`ui`·`a11y`(Playwright Chromium) — **느리고 외부 의존**
  - `npm run routes` 로 `docs/api-routes.md`·`openapi.yaml` 재생성(단위 시험이 어긋남을 잡음)
- 관례: 커밋 메시지는 **한국어 한 줄**, `영역: 무엇을 했는가` 꼴(`파일 대조: …`, `공유 DB 가드: …`). 주석·문서도 한국어, 다만 소스 상단 파일 설명 주석은 영어. 설정은 `.env.example` 에 환경별로. 마이그레이션은 `scripts/migrate.mjs` + `db/`.
- 위험 구역: `packages/core/src/auth/*`(세션·쿠키·스코프), `packages/core/src/domain/payments.ts`·`refunds.ts`·`settlement.ts`(결제 상태기계, `docs/payment-state-machine.md`), `db/` 마이그레이션 — **공유 Supabase 를 쓰고 서비스 역할이 NOLOGIN** 이라 로그인 역할을 건드리는 마이그레이션은 금지(커밋 8abcd8a·63dee8d). `.github/workflows/check.yml`.
- 자주 깨지는 곳: 같은 종류의 파서가 여러 경로에 흩어져 있다(`table/csv.ts`, `table/xlsx-read.ts`, `compare.worker.ts`) — 통합하지 말고 한쪽만 좁게 고칠 것. 공유 DB 연결 풀(irumx_gateway → workkit_app 전환)은 과거에 두 번 손봤다.
- 검증 함정: 워크트리에 node_modules 없음. `npm run e2e`·`tools`·`ui`·`a11y` 는 Docker PostgreSQL·Playwright·가짜 서버(:18997 옆자리, :18998 토스, :8797 앱)가 떠 있어야 한다 — CI 에서는 돌지만 로컬에서 준비 없이 돌리면 그냥 실패한다. 단위 시험만으로 끝나는 과제를 고르면 가장 안전하다. 키트 zip 은 결정적이어야 해서 CI 가 두 번 빌드해 manifest 를 비교한다.
