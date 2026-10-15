# studio.irumx.app (이룸필름) 프로필 (2026-10-15)

- 목적: 이룸엑스의 홍보영상 제작 스튜디오 — 공개 사이트(작품·서비스·가격·과정·의뢰)와 의뢰→견적→결제→기획 승인→제작→시안 검수→납품 업무 흐름, 운영(상품·작품 CMS·원가 원장·출시 잠금)을 Cloudflare Worker 하나로 제공.

- 스택: TypeScript. 화면 React 19 + Vite 8 + react-router 8 + TanStack Query, CSS 변수·Pretendard 부분 글꼴. API Cloudflare Workers + Hono + zod. 데이터 D1 + Drizzle. 파일 비공개 R2(Worker 경유만). 인증 Better Auth 이메일 OTP(쿠키 접두사 `irumfilm`). 비동기 D1 outbox + Cron(5분) + Queues. 결제 PaymentProvider 어댑터(토스=미검증·잠금, fake=비운영). 메일 Resend 어댑터.

- 구조:
  - `src/client/` — `pages/`(공개·의뢰·고객), `admin/`(운영), `components/`(marketing.tsx 에 공개 사이트 부품: VideoPoster·WorkGrid·PricingCards), `lib/`, `styles/`
  - `src/shared/` — 화면·서버 공용, **공개해도 되는 것만**(상태·권한 표, 의뢰서 스키마, 공개 카탈로그 JSON)
  - `src/worker/internal/` — 서버 전용 원가 정책·가격 산식(정확한 유리수). **화면 번들에 들어가면 빌드 실패**
  - `src/worker/routes/` — public·me·projects·quotes·payments·storyboards·uploads·deliverables·jobs·admin·admin-catalog·admin-costs
  - `src/worker/services/` — outbox·mail·audit·payments(상태 기계)·files·portfolio(공개 사본 복사·철회)·seo(앱 셸 메타·사이트맵)·cron·settings·serialize(공개 필드 화이트리스트)
  - `src/worker/lib/mp4.ts` — MP4 상자 구조 판독(길이·해상도·트랙). 서버와 `scripts/portfolio-import.mjs` 가 **같은 판독기**를 공유
  - `migrations/` — 덧붙이기만. `0000_init` 스키마, `0001_guards` 잠금 트리거, `0002_seed` 초기 상품·원가 정책 v1·설정
  - `scripts/` — verify-build(공개 번들 금칙어·메타 자리 검사), db-local, admin, deploy, portfolio-import, subset-font, make-og, gen-seed
  - `content/portfolio/<slug>.json` — 명령으로 올리는 작품 메타(미디어 파일은 저장소 밖, `.gitignore`)
  - `shell.html` — 앱 셸 하나. Worker 가 쪽마다 `HTMLRewriter` 로 제목·설명·OG·canonical·`noscript#seo` 를 채운다

- 빌드·테스트:
  - `npm run build` — 글꼴 서브셋 → `tsc` 3종(app/worker/node) → `vite build` → `scripts/verify-build.mjs`. **느리다**
  - `npm test` — Playwright 76개(unit 26 · api 38 · desktop 10 · mobile 2). `npm run test:unit` / `test:api` / `test:ui` 로 쪼갤 수 있다
  - **`npm test` 전에 `npm run build` 가 필요하다** — webServer 가 `dist` 를 `wrangler dev :8848` 로 띄우고, 로컬 D1(`.wrangler/test-state`)을 비운 뒤 `ops@irumfilm.test`·`maker@irumfilm.test` 를 만든다. 가짜 Resend `:8849` 도 함께
  - `npx playwright install chromium` 1회. Node 22.12+
  - `npm run check` 로 타입만, `SHOTS=1 npx playwright test --project=shots` 로 캡처

- 관례: 커밋 메시지·주석·문서·식별자 설명이 **전부 한국어**(명사형 제목 + `- ` 불릿 본문 + `Co-Authored-By`). 공개 응답은 `services/serialize.ts` 화이트리스트를 거친다. 마이그레이션은 덧붙이기만(drizzle-kit generate). 문서는 `docs/`(assumptions·operations·acceptance·requirements), 인수 기준은 AT-01~AT-28 로 번호를 달고 시험 제목에 `(AT-03)` 처럼 적는다.

- 위험 구역:
  - `src/worker/auth.ts`, `src/worker/access.ts` — 요청마다 프로젝트 권한 확인. 화면에서 버튼을 숨기는 것으로 끝나지 않는지 api 시험이 직접 찌른다
  - `migrations/0001_guards.sql` — 출시 잠금·작품 공개 권리/동의 트리거(`locked:portfolio_consent`, `portfolio_publish_requires_rights`). DB 가 최후 방어선이므로 코드 쪽만 고치고 끝내지 말 것
  - `src/worker/internal/` — 원가·공급사 정보. 공개 번들에 새면 `verify-build` 의 `FORBIDDEN` 목록이 빌드를 멈춘다(AT-02)
  - `src/worker/services/payments.ts` — 시간 초과·5xx·처리 중은 실패가 아니라 `unknown`, Cron 이 재조정. 실패로 접으면 돈이 틀어진다
  - `src/worker/services/portfolio.ts` — 동의 범위가 줄면 공개 사본(R2)까지 지운다

- 자주 깨지는 곳: (이 저장소의 회차 기록이 아직 없어 미확인) 구조상 걸리기 쉬운 자리 — `shell.html` 과 `seo.ts` 의 짝(한쪽만 고치면 효과 없음), `verify-build.mjs` 의 단정들(인라인 스크립트 0개·메타 자리·금칙어), `src/shared/` 에 서버 전용 값을 넣는 것.

- 검증 함정:
  - `shell.html` 에 **`src=` 없는 `<script>` 를 넣으면 `verify-build` 가 멈춘다**(CSP `script-src 'self'`). JSON-LD 도 여기 걸린다
  - `HTMLRewriter` 는 셸에 **이미 있는 요소만** 고친다. 새 메타는 `shell.html` 에 자리를 먼저 만들고, 해당 없는 쪽에서는 `element.remove()` 로 빈 태그를 지워야 한다
  - `setAttribute` 경로는 이스케이프가 자동, `noscript#seo` 의 `setInnerContent(..., { html: true })` 경로만 `esc()` 가 필요하다 — 섞으면 안 된다
  - 시험하지 않은 것: 실제 토스페이먼츠(테스트 키 포함), 운영 로그인 메일 실제 도착, 실제 제작 공급사, D1/R2 복구, 실제 H.264 재생(시험 영상은 상자 구조만 갖춘 합성 MP4), Queues 실제 재전달
  - `scripts/portfolio-import.mjs` 는 자동 시험이 없다(수동 확인만). `--remote` 는 `CLOUDFLARE_API_TOKEN`·`CLOUDFLARE_ACCOUNT_ID` 필요
