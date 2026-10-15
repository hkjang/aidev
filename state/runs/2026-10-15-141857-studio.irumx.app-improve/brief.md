# 과제서 — 2026-10-15-141857-studio.irumx.app-improve

- 과제: 해석되지 않는 Range 헤더에 416 대신 전체(200)로 응답 — 공개 영상이 통째로 재생 불가가 되는 자리 막기 (가치 3 / 위험 1 / 작업량 S)

- 왜: `src/worker/services/parseRange()` 는 `/^bytes=(\d*)-(\d*)$/` 에 걸리지 않는 모든 Range 헤더를 `'invalid'` 로 돌려주고, `serveObject()` 가 그것을 **416 Range Not Satisfiable** 로 내보낸다. 그래서 여러 구간 요청(`bytes=0-1,5-6`), 모르는 단위(`items=0-5`), 깨진 값(`bytes=abc`)이 오면 본문을 하나도 주지 않고 거절한다 — 같은 함수가 공개 작품 영상(`/media/portfolio/:id/video`, 홈 대표 영상)과 시안 재생·최종본 내려받기를 모두 태우므로, 그런 헤더를 보내는 클라이언트(내려받기 관리자·일부 프록시·curl `--range a-b,c-d`)에서는 재생이 아예 안 된다. 416 은 **문법은 맞지만 자원 범위를 벗어난** 요청에만 쓰는 응답이고(nginx·R2 를 포함한 일반적인 서버 동작), 해석할 수 없는 Range 는 헤더가 없는 것처럼 무시하고 200 전체를 주는 쪽이 맞다. 고치면 "범위를 모르면 아무것도 못 받는" 실패가 "조금 더 받아서 재생된다" 로 바뀐다.

- 수용 기준:
  1) 해석 불가 헤더(`bytes=0-1,5-6`, `bytes=abc`, `items=0-5`, `bytes=-`, `bytes=0--1`)는 `parseRange` 가 `null` 을 돌려주고, HTTP 로는 **200 + 전체 Content-Length + Content-Range 없음 + `Accept-Ranges: bytes`** 가 나온다.
  2) 범위를 벗어난 **문법이 맞는** 요청은 종전대로 416 이다: `bytes=1000-`(size 1000), `bytes=-0`, `bytes=500-400` → `'invalid'`, HTTP 416 + `Content-Range: bytes */<size>`.
  3) 정상 범위 동작은 그대로다: `bytes=0-99`→{0,100}, `bytes=900-`→{900,100}, `bytes=-100`→{900,100}, `bytes=0-0`→{0,1}, 끝 초과(`bytes=0-99999`, size 1000)→{0,1000}. 기존 206 시험(`tests/api-portfolio.spec.ts:44`, `tests/api-flow.spec.ts:131`)이 계속 통과한다.
  4) 시험이 증명할 것: 고친 코드를 되돌리면 (1) 의 단위 시험과 HTTP 시험이 **416 을 받아 실패**한다(구현자가 실제로 되돌려 확인하고 실패 출력을 남길 것).

- 건드릴 파일 (프로덕션 1개 + 시험 2개):
  - `src/worker/services/files.ts:139 parseRange` — 지금 `if (!m || (!m[1] && !m[2])) return 'invalid';` 를 `return null` 로 바꾸고(= 헤더 무시), `'invalid'` 는 **문법이 맞는데 만족시킬 수 없는 경우**(마지막 `if (start > end || start >= size)`)만 남긴다. `bytes=-0` 은 지금도 start=size, end=size-1 로 그 분기에 걸려 `'invalid'` 가 되므로 그대로 둔다. 주석(138행)도 "해석 못 하면 무시(null), 범위를 벗어나면 416('invalid')" 로 고칠 것. `serveObject` 는 이미 `range === null` 이면 전체를 주므로 **건드리지 않는다**.
  - `tests/unit-media.spec.ts:60 '범위 요청 해석'` — 위 1)·2)·3) 경계를 이 시험에 덧붙인다(새 파일 만들지 말 것).
  - `tests/api-portfolio.spec.ts:44` 부근(공개 작품 영상 `mine.videoUrl` 로 이미 206 을 찌르는 자리) — 같은 URL에 ① `Range: bytes=0-1,5-6` → 200 이고 `content-range` 헤더가 없음 ② `Range: bytes=99999999-` → 416 이고 `content-range` 가 `bytes */` 로 시작 을 더한다. 이것이 프로덕션 배선(public 라우트 → serveObject → R2)을 통과하는 증거다.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `npm run build` — 마지막에 "빌드 점검 통과" 가 떠야 한다. **느리다(글꼴 서브셋 + tsc 3종 + vite + verify-build)**
  - `npm test` — Playwright 78개(종전 통과 수). 빠르게 보려면 `npm run test:unit` → `npm run test:api`. **어느 쪽이든 `npm run build` 가 먼저 필요하다**(webServer 가 `dist` 를 `wrangler dev :8848` 로 띄운다)
  - 타입만: `npm run check`

- 위험과 피할 것:
  - **반환 타입(`{offset,length} | null | 'invalid'`)을 바꾸지 말 것.** 분류만 옮기는 변경이다. `serveObject` 의 헤더 조립·`Accept-Ranges`·`Content-Disposition`·CSP 는 손대지 않는다.
  - 여러 구간 요청을 `multipart/byteranges` 로 **구현하려 하지 말 것**(첫 구간만 206 으로 주는 것도 금지 — 클라이언트가 multipart 를 기대한다). 이번 회차는 "무시하고 전체" 까지만.
  - 호출자는 `uploads.ts:215`, `deliverables.ts:176`·`:417`, `public.ts:143` 네 곳인데 전부 `serveObject` 를 거치므로 **라우트 파일은 건드릴 필요가 없다**. 건드리게 되면 과제가 어긋난 것이다.
  - 보호 경로를 피할 것: `src/worker/auth.ts`·`access.ts`, `migrations/`, `.github/workflows/`, `src/worker/services/payments.ts`, `src/worker/internal/`.
  - 커밋 메시지·주석은 **한국어**(명사형 제목 + `- ` 불릿). 새 의존성 금지.
  - 미확인: 실제 브라우저·플레이어가 이 사이트에 여러 구간 Range 를 보내는 빈도는 측정하지 않았다(코드 경로상 거절된다는 것만 확인). 그래서 가치를 3 으로 잡았다 — 범위를 더 넓히지 말 것.

- 차선 후보: **`seo.ts`·`shell.html` 짝 고정 시험** (가치 2 / 위험 1 / 작업량 S) — `/`·`/work/<slug>`·`/app` 각각의 셸 HTML 을 HTTP 로 받아 `robots` 가 private 경로에서 `noindex`, `canonical` 이 색인 가능한 쪽만 자기 주소, `noscript#seo` 가 `esc()` 를 거친 `<h1>`·`<p>` 를 담는 것을 고정한다(`tests/api-portfolio.spec.ts` 또는 `tests/ui-public.spec.ts` 안). 프로덕션 코드는 건드리지 않는 시험 전용 회차다.
