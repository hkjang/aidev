# law.irumx.app 프로필 (2026-10-14)

- 목적: 법무법인마다 자기 브랜드·도메인의 홈페이지를 두고, 공동소송의 모집→신청→검토→위임계약 전자서명→비용 납부→참여 확정→사건 진행까지 한 흐름으로 처리하는 멀티테넌트 SaaS.
- 스택: TypeScript(ESM, Node >= 22) · PostgreSQL 17(스키마 `law`/`law_priv`, 역할 `law_app`, RLS) · React + Mantine · Cloudflare Pages Functions + Hyperdrive + R2 + KV · vitest · playwright · eslint 9(flat config). npm workspaces(`packages/*`, `apps/*`).

## 구조
- `packages/core` — 요청 처리 전부(16,790줄). Node 와 Workers 가 같은 코드를 쓴다.
  - `src/http/pipeline.ts`(504줄) 라우트 실행·오류 로그·`redactSecrets` · `src/env.ts` `loadEnv`(raw 객체 → Env, 순수 함수) · `src/db.ts`(431줄) `elevate`·`identityStatement`
  - `src/routes/` platform·auth·portal·public·ops + `console/`(applications 610줄, site 548줄, domains, matters, campaigns, settings, comms, payments)
  - `src/modules/` payments·documents·forms·workflow·engagements·notify·applications·provision
  - `src/providers/` pg(결제)·sms·identity·domains·dns — **모의 공급자 분기가 여기 모여 있다**
  - `src/site/` 법인 홈페이지 서버 렌더링(render.ts 573줄, blocks.ts) · `src/util/` crypto(429줄)·text·validate · `src/__tests__/` 5개 파일뿐
- `apps/server` Node 어댑터(:8810) · `apps/web` React 앱(console/ops/portal/platform)
- `functions/[[path]]` Pages Function 어댑터(엣지 캐시) · `workers/scheduler` 매분 `/api/v1/internal/cron`
- `db/migrations` 스키마·RLS 정책, `db/tests/rls-probe.sql` · `scripts` 개발 초기화·모의 공급자·e2e·인수시험·UI 흐름·배포 · `engineering` 설계·보안·운영·인수시험 문서

## 빌드·테스트
- `npm run check` = `lint` + `typecheck` + `test`. `npm run test` = `vitest run`(단위, DB 불필요, 빠름)
- `npm run dev:reset` — DB 비우고 마이그레이션 + 모의 공급자(:18910) + 서버(:8810). 아래 시험들의 전제.
- `npm run e2e`(API 한 바퀴) · `npm run acceptance`(AT-01~AT-26, 결과 `data/acceptance.json`) · `npm run ui-flows`(Playwright + axe) · `npm run test:rls`(docker psql) — **전부 DB/브라우저 필요, 오래 걸림**
- `npm run smoke:prod`(읽기 전용) · CI `.github/workflows/check.yml` 이 PostgreSQL 17 서비스로 같은 것을 돌린다. CI 환경값은 `scripts/ci-env.mjs`(`LAW_DEV=true`).
- vitest include: `packages/**/*.test.ts`, `apps/**/*.test.ts`, testTimeout 20초(crypto 때문).

## 관례
- 커밋 메시지 **한국어**, 명사형/평서형 한 줄 요약(예: "엣지 캐시: 플랫폼 첫 화면·404·법인 앱 셸을 DB 왕복 없이"). 릴리즈 커밋은 `vX.Y.Z …`, `VERSION`·`package.json` 동시 갱신.
- 코드 주석은 **영어**, 요구사항 번호(SEC-07, DOM-04, PAY-04/05, AT-12/26, "section 10")를 달아 둔다. 사용자에게 보이는 문자열은 한국어.
- 설정은 `loadEnv(raw)` 한 곳에서만 읽는다(`env.ts`). 운영 값은 `wrangler.toml` 의 `[vars]` + `~/.config/law-irumx/cloud.env`(저장소 밖). `.env.example` 이 개발 설정 전부.
- 마이그레이션은 `db/migrations` 에 번호 파일 + `node scripts/migrate.mjs`. 문서는 `engineering/*.md`.

## 위험 구역
- `packages/core/src/auth/*`, `routes/auth.ts` — 이메일 코드 + 직원 OTP + 법인 사이트로 넘기는 인가 코드/PKCE.
- `db/migrations/*`, `db/tests/rls-probe.sql` — RLS 정책. 공유 Supabase 인스턴스를 옆 서비스와 같이 쓴다. `irumx_gateway` 로 로그인해 트랜잭션마다 `law_app` 으로 전환(`identityStatement`). **배포·마이그레이션이 서비스 역할 로그인을 되살리면 안 된다**(v0.1.2 가 이것 때문에 나왔다).
- `modules/payments.ts`, `routes/console/payments.ts` — 금액·수취인·주문번호를 서버가 정하고, 공급자 응답만 상태를 바꾼다(PAY-06). 멱등 원장. 토스 어댑터는 실거래 미검증.
- `util/crypto.ts`, 전자서명(ECDSA P-256, 계약 스냅샷 해시) — `/.well-known/law-signing-key` 로 공개된다.
- `env.ts:loadEnv` — 여기서 던지면 워커가 기동하지 못한다. 조건을 좁게 유지할 것.

## 자주 깨지는 곳
- 공유 DB 연결: `too many connections for role`, 연결 끊김 재시도(`retryable.test.ts`, `isConnectionLost`). 연결 상한·배분을 건드리는 변경이 반복해서 문제가 됐다.
- 느린 CI 에서의 타이밍(인수시험 AT-21 이 파일 검사 완료를 기다리게 고쳐졌고, 부하 시험 p95 기준을 실행 환경에서 정하게 바뀌었다).
- 같은 종류의 파서·검사가 여러 경로에 있을 때 통합 시도 → 각자 계약이 달라 다른 쪽이 깨진다(운영자 지시: 좁히는 방향만).

## 검증 함정
- 단위 시험(`npm run test`)만 DB 없이 돈다. 나머지는 `npm run dev:reset` 직후의 **빈 DB** 를 전제하므로 순서를 바꾸면 실패한다.
- 본인확인(PASS)·문자·실결제는 모의 공급자로만 시험된다 — 계약 시험이지 실거래 증명이 아니다.
- 모의 공급자는 `LAW_DEV=true`(→ `allowMocks` 기본 true)에 기대고 있다. `LAW_DEV` 를 떼면 키 3종(64자 hex)과 `LAW_SIGNING_KEY` 가 없어 `loadEnv` 가 먼저 던진다.
- 이 세션에서는 샌드박스 제한으로 `npx vitest run` 을 실행하지 못했다 — 기준선 통과 여부는 **미확인**.
