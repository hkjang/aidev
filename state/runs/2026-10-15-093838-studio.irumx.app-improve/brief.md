# 과제서 — 2026-10-15-093838-studio.irumx.app-improve

- **과제**: 작품 상세(`/work/:slug`) 공유 미리보기를 영상 카드로 — `og:type`·`og:video`·`og:image:alt`·`twitter:card` 를 서버가 쪽마다 채운다 (가치 4 / 위험 2 / 작업량 S)

- **왜**: `shell.html` 의 `og:type` 은 모든 쪽에서 고정값 `website` 이고 `og:video`·`twitter:*`·`og:image:alt` 자리는 아예 없다(`shell.html:12~19` 확인). 그래서 어제 공개한 대표 콘셉트 영상(`/work/irumx-brand-space`)을 카카오톡·X·페이스북에 공유하면 영상 작품이 아니라 일반 웹사이트 링크로 보이고, 포스터 이미지에 대체 설명도 붙지 않는다. 영상 스튜디오 사이트의 유입 경로가 사실상 링크 공유인데 가장 보여줄 것(영상)이 미리보기에서 빠져 있다. `pageMeta` 가 이미 작품 행을 읽어 제목·설명·`og:image` 를 넣고 있으므로(`src/worker/services/seo.ts:46~96`) 같은 자리에 규칙 네 개를 더하는 일이다.

- **수용 기준**:
  1. 공개된 작품(영상 있음) 상세 HTML 응답에 `<meta property="og:type" content="video.other">`, `<meta property="og:video" content="<origin>/media/portfolio/<itemId>/video">`, `<meta property="og:video:type" content="video/mp4">`, `<meta property="og:image:alt" content="<altText 또는 summary>">`, `<meta name="twitter:card" content="summary_large_image">` 가 들어 있다. URL 은 `env.APP_ORIGIN` 기준 절대 주소다(기존 `og:image`·`canonical` 과 같은 방식).
  2. 고정 쪽(`/`, `/pricing` 등)과 비공개 쪽(`/app`, `/admin`)에서는 `og:type` 이 `website` 로 남고, **`og:video`·`og:video:type` 태그가 응답 HTML 에 아예 없다**(빈 값으로 남지 않는다 — `HTMLRewriter` 의 `element.remove()` 사용). 영상이 없는 작품(포스터만)도 `og:video` 가 없어야 한다.
  3. `npm run build` 가 통과한다 — `scripts/verify-build.mjs:41` 의 메타 자리 확인 목록에 새 자리(`property="og:video"`, `name="twitter:card"`)를 더해, 앞으로 셸에서 자리가 빠지면 빌드가 멈춘다.
  4. api 시험이 위 1)·2)를 HTTP 응답 본문으로 증명한다(소스 문자열 검사가 아니라 `wrangler dev` 를 거친 실제 응답).

- **건드릴 파일**(프로덕션 3 + 시험 1):
  - `shell.html` — `<head>` 에 자리 추가: `<meta property="og:type" content="website" />`(이미 있음, 그대로) 아래에 `<meta property="og:video" content="" />`, `<meta property="og:video:type" content="video/mp4" />`, `<meta property="og:image:alt" content="" />`, `<meta name="twitter:card" content="summary" />`, `<meta name="twitter:title" content="" />`, `<meta name="twitter:description" content="" />`, `<meta name="twitter:image" content="" />`. **인라인 `<script>` 는 절대 넣지 말 것**(아래 위험 참고).
  - `src/worker/services/seo.ts:`
    - `pageMeta()` — 작품 분기(`if (item) { … }`)에서 `videoUrl`(있을 때만), `imageAlt = item.altText || item.summary`, `ogType = 'video.other'` 를 지역 변수로 잡는다. 고정 쪽·비공개 쪽은 `ogType = 'website'`, `videoUrl = null`.
    - `apply()` — `.on('meta[property="og:type"]', …setAttribute('content', ogType))`, `.on('meta[property="og:video"], meta[property="og:video:type"]', { element: (e) => videoUrl ? (e.getAttribute('property') === 'og:video' && e.setAttribute('content', videoUrl)) : e.remove() })` 형태로 영상 없으면 **제거**. `og:image:alt` 도 값 없으면 `e.remove()`. `twitter:card` 는 `og:image` 가 기본 og.png 가 아니거나 작품 쪽일 때 `summary_large_image`, 그밖엔 `summary`. `twitter:title`·`twitter:description`·`twitter:image` 는 `title`·`description`·`image` 와 같은 값으로 채운다.
    - 값에 `"`·`<` 가 들어가도 `setAttribute` 가 처리한다(기존 `og:title` 과 동일 경로) — 별도 `esc()` 를 쓰지 말 것. `esc()` 는 `noscript#seo` 의 `{ html: true }` 경로 전용이다.
  - `scripts/verify-build.mjs:41` — 메타 자리 확인 배열에 `'property="og:video"'`, `'name="twitter:card"'` 추가.
  - `tests/api-portfolio.spec.ts:25` 의 기존 시험(`자체 제작은 … 공개된다(AT-03)`)에 이어 붙이거나 새 `tests/api-seo.spec.ts` 를 만든다. 기존 시험이 이미 포스터+영상 있는 `self` 작품을 공개하고 `const html = await (await pub.get('/work/${slug}')).text()` 로 `<title>` 을 확인하므로(46행), 그 자리에서 `expect(html).toContain('content="video.other"')`, `expect(html).toMatch(/og:video" content="http:\/\/127\.0\.0\.1:8848\/media\/portfolio\/[a-z0-9]+\/video"/)` 를 더하면 된다. 고정 쪽 쪽은 `tests/api-isolation.spec.ts:138` 이 이미 `/pricing` HTML 을 받아 `canonical` 을 보고 있으니 같은 응답에 `expect(pricing).not.toContain('og:video')` 를 더하는 것이 가장 싸다.

- **검증 명령**(이 저장소에서 실제로 도는 것):
  ```bash
  npm run build            # 글꼴 → tsc 3종 → vite build → verify-build (수용 기준 3)
  npm run test:api         # playwright --project=api (wrangler dev :8848 + 가짜 Resend :8849 를 Playwright 가 띄운다)
  ```
  `npm run build` 를 먼저 돌려야 한다 — `playwright.config.ts` 의 webServer 가 `dist` 를 `wrangler dev` 로 띄운다. 전체는 `npm test`(unit 26 · api 38 · desktop 10 · mobile 2 = 76개가 기준선).

- **위험과 피할 것**:
  - **`shell.html` 에 인라인 `<script>`(JSON-LD 포함) 를 넣으면 빌드가 멈춘다.** `scripts/verify-build.mjs:36-37` 이 `src=` 없는 `<script>` 를 0개로 단정하고(`shell.html: 인라인 스크립트 N개`), `vite.config.ts` 주석대로 CSP 가 `script-src 'self'` 만 허용한다. JSON-LD 구조화 데이터는 이번 과제에 넣지 말 것 — 그 가드를 느슨하게 하는 변경이 되므로 별도 회차로 미뤘다(ideas.json 참고).
  - `HTMLRewriter` 는 **셸에 이미 있는 요소만** 고친다. 새 메타를 `shell.html` 에 넣지 않고 `seo.ts` 만 고치면 아무 일도 일어나지 않는다. 반대로 셸에만 넣고 `seo.ts` 에서 제거 처리를 빼면 모든 쪽에 빈 `og:video` 가 남아 미리보기가 더 나빠진다 — 수용 기준 2)가 이 자리다.
  - **최종 출력(HTTP 응답 HTML)까지 확인할 것.** `vite build` 가 셸을 거쳐 `dist/client/shell.html` 로 나가고 Worker 가 `/shell` 을 받아 다시 쓰므로, `shell.html` 소스만 보고 끝내면 효과를 못 본다. `npm run test:api` 응답 본문으로 증명한다.
  - 건드리지 말 것: `migrations/`(특히 `0001_guards.sql` 트리거), `src/worker/auth.ts`·`access.ts`, `src/worker/internal/`(화면 번들에 들어가면 빌드 실패), `scripts/verify-build.mjs` 의 `FORBIDDEN` 목록·인라인 스크립트 가드, `src/worker/services/portfolio.ts` 의 공개 사본 복사·철회 로직.
  - `sitemap()` 은 이미 작품을 넣고 있다(`seo.ts:92~104`, `tests/api-portfolio.spec.ts:47` 이 확인). 손대지 말 것.

- **차선 후보**: `scripts/portfolio-import.mjs` 의 SQL 조립에서 `meta.sort ?? 0` 가 `q()` 를 거치지 않고 그대로 들어간다(119행 부근 `portfolio_items` INSERT). 운영자가 작품 JSON 에 `"sort": "1"` 처럼 문자열이나 임의 문자열을 쓰면 SQL 이 깨지거나 주입된다. `q(Number(meta.sort ?? 0))` 가 아니라 `Number.isInteger` 검사 후 숫자로 확정하고(아니면 `fail()`), 같은 파일의 입력 검사 블록(31~37행)에 `sort`·`featured` 타입 확인을 더하는 S 과제. 단일 파일 + `content/portfolio/irumx-brand-space.json` 으로 수동 확인만 가능해(이 스크립트는 자동 시험이 없다) 1순위로 두지 않았다.
