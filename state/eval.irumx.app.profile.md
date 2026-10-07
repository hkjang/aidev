# 이룸평가(eval.irumx.app) 프로필 (2026-10-07)

- 목적: 업무용 AI 에이전트를 고객 업무에 투입해도 되는지, 같은 업무 사례를 변경 전후로 같은 초기 상태에서 반복 실행해 결과·도구·승인·권한·최종 상태를 검사하고 배포 판단과 근거를 남기는 서비스. v0.2.0, main@202c416.
- 스택: TypeScript. 화면 React 19 + Vite 8 + React Router + TanStack Query(정적 소개·안내는 HTML). API Cloudflare Workers + Hono. DB D1 + Drizzle. R2(원문·보고서), Queues(`irumx-eval-trials` + DLQ), Cron 1분. 인증 Better Auth 이메일 OTP(초대제). 모델 Workers AI.
- 구조:
  - `src/worker/routes/*` — Hono 라우트(요청마다 권한 확인, 기본 거부). `src/worker/index.ts` 가 미들웨어(보안 헤더·gzip·세션·속도 제한)·`queue()`·`scheduled()`.
  - `src/worker/domain/*` — 실행 엔진(`engine.ts` 임대·세대 가드), 판정기(`graders.ts`), 배포 판단(`gates.ts`), 역검증(`validation.ts`), 모의 업무 시스템(`sandbox/`), 실행 틀(`harness.ts`).
  - `src/worker/services/*` — 실험 계획·시작(`experiments.ts`), 예산 예약·정산(`usage.ts`), 대상·프로젝트·감사·메일.
  - `src/worker/fixtures/inquiry-pack.ts` — 검수팩 사례 40개(정상 15·권한/승인 10·장애 10·주입 5).
  - `src/client/pages/*` — 화면. `src/shared/*` — 사례 형식·도메인 타입.
  - `docs/` — architecture · spec-coverage(설계 대응표, 무엇이 🟡/⬜ 인지 **여기부터 보라**) · operations · deploy · design.
- 빌드·테스트:
  - `npm run build` — 글꼴 서브셋 → tsc 3개 프로젝트 → `scripts/check-pack.ts`(판정기 역검증 40개) → vite build → `scripts/verify-build.mjs`. **수 분 걸린다.**
  - `npm test` — 로컬 D1 비우고 `wrangler dev`(:8860) + 가짜 Resend(:8861)·모델(:8862)·HTTP 대상(:8863) 위에서 Playwright. 프로젝트: `unit`/`api`/`desktop`/`mobile`. `workers: 1`, 전체 timeout 240s. **빌드 결과를 띄우므로 build 가 먼저.**
  - 단위만: `npx playwright test --project=unit` (브라우저 없이 순수 계산 — 그래도 webServer 는 뜬다).
  - `npm run shots`(캡처·가로 넘침), `npm run backup-drill`, `npm run deploy`(운영 확인 30개).
- 관례: 커밋 메시지·주석·문서·오류 메시지 모두 **한국어**, 영어 약어 최소. 커밋 제목은 "무엇을 했는지 · 점으로 구분" 형식. 마이그레이션은 `drizzle-kit generate`(`migrations/`, 불변 기록 트리거는 `0001_guards.sql`). 설정은 `wrangler.jsonc`. 문서는 `docs/`, 설계 원문 `docs/spec/eval-service-spec.md`, 구현 상태는 `docs/spec-coverage.md` 에서 관리.
- 위험 구역: `src/worker/auth.ts`·Better Auth 쿠키(`irumx-eval.*`), `migrations/*`(불변 기록 트리거), `src/worker/security/{crypto,egress}.ts`(대상 주소 검사·키 암호화), `src/worker/services/usage.ts`(예산 한도 — 과금), `scripts/deploy.mjs`·`verify-build.mjs`(client/agent 과 자원이 섞이지 않는지 확인). D1 batch 의 `changedGuardSql` 가드(0행 변경을 오류로 바꿈)를 깨면 낙관적 동시성이 조용히 무너진다.
- 자주 깨지는 곳: (회차 기록 없음 — 이번이 첫 정찰) 다만 구조상 주의: D1 문장당 바인딩 100개 한도(배치 쪼개기 코드가 여러 곳), 시험이 업체 공간 한도를 바꾼 뒤 되돌리지 않으면 뒤 시험이 전부 깨진다, 판정기·사례를 고치면 `check-pack`(빌드)에서 막힌다.
- 검증 함정: CI 설정이 **없다**(`.github` 없음) — 검증은 로컬 `npm run build && npm test` 뿐. 빌드를 다시 했으면 떠 있는 `wrangler dev`(:8860)를 끄고 다시 띄워야 새 자산이 보인다. 실제 Workers AI·Resend 를 부르는 시험은 `LIVE`/`--model` 플래그에서만 돈다(과금). `reuseExistingServer: true` 라 옛 서버가 떠 있으면 옛 코드를 시험한다.
