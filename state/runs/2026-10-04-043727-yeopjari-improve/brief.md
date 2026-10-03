- 과제: 클라이언트가 보낸 퍼센트 인코딩 때문에 요청이 500 이 되고 운영자가 오류를 지울 수 없는 문제 (가치 4 / 위험 1 / 작업량 S)
- 왜: `readCookies` 가 쿠키 값을 `decodeURIComponent` 로 보호 없이 풀기 때문에, `Cookie: yj_s=%` 처럼 깨진 퍼센트 인코딩이 오면 URIError 가 `handleRequest` 의 바깥 catch 까지 올라가 500 응답 + 운영자 오류 알림 메일이 된다(쿠키는 누구나 임의로 넣을 수 있고, 비로그인 공개 경로까지 같이 죽는다). 같은 계열로 `admin/errors/:signature` 는 라우터가 이미 디코딩한 파라미터를 한 번 더 디코딩하기 때문에, 경로에 한글·비ASCII 가 들어간 오류 서명은 삭제 요청이 영원히 "그런 오류 기록이 없습니다" 로 끝나고 알림 스로틀 행이 남는다.
- 수용 기준:
  1) `Cookie: yj_s=%` 또는 `yj_s=%E0%A4%A` 가 붙은 요청에서 `readCookies` 가 던지지 않고, 그 깨진 쿠키만 없는 것으로 취급하며 같은 헤더의 정상 쿠키(`yj_csrf=t`)는 그대로 읽힌다. 결과적으로 그 요청은 "세션 없음"(익명)으로 처리되어 500·운영자 알림이 발생하지 않는다.
  2) `DELETE /api/v1/admin/errors/:signature` 가 웹이 보내는 인코딩(`encodeURIComponent(signature)`, apps/web/src/routes/admin/Errors.tsx:59)과 왕복으로 맞는다. 즉 `GET /api/v1/companies/%ED%9C%B4 internal` 같은 리터럴 `%` 를 품은 서명도 그 행이 실제로 지워진다.
  3) 테스트가 증명할 것: (a) 프로덕션 `Request` 객체(전역 Request, 손으로 만든 대역 아님)를 `readCookies` 에 넣어 깨진 인코딩에서 throw 가 없고 정상 쿠키가 보존됨, (b) 프로덕션 `Router` 로 `/api/v1/admin/errors/:signature` 패턴을 실제 등록·match 해서 `encodeURIComponent(signature)` 로 만든 경로가 `ctx.params.signature === signature` 로 도착함 — 즉 핸들러의 두 번째 디코딩이 불필요하고 해롭다는 것을 라우터 쪽에서 보여줄 것.
- 건드릴 파일:
  - `packages/core/src/auth/cookies.ts:readCookies` — 값 디코딩을 try/catch 로 감싸 실패한 쿠키는 건너뛴다(한 쿠키의 실패가 같은 헤더의 다른 쿠키를 버리지 않게). 참고로 `packages/core/src/http/router.ts:97-103` 가 이미 같은 위험을 try/catch 로 막고 있으니 그 방식과 어긋나지 않게 맞출 것.
  - `packages/core/src/routes/admin/ops.ts:518` (`router.delete('/api/v1/admin/errors/:signature')`) — `decodeURIComponent(ctx.params.signature ?? '')` 의 중복 디코딩 제거(라우터가 이미 세그먼트를 디코딩함: router.ts:100). `if (!signature) throw notFound(...)` 와 감사 로그 인자는 그대로.
  - `packages/core/src/__tests__/` 에 테스트 추가 — 새 파일(예: `cookies-params.test.ts`)이 깔끔하다. 기존 `registry-router.test.ts` 에 라우터 왕복 케이스를 덧붙여도 된다.
  - (프로덕션 파일 2개로 끝낼 것. 아래 "위험" 의 파일들로 번지면 과제를 쪼갠다.)
- 검증 명령:
  - 워크트리에 node_modules 가 없다(확인: `npm test` 가 `sh: 1: vitest: not found`). 먼저 `npm ci`.
  - `npx vitest run packages/core/src/__tests__` (전체는 `npm test`)
  - `npm run lint && npm run typecheck`
  - DB·서버가 필요한 `scripts/e2e.mjs` 는 이 변경에 필요 없다(실행 안 했음 — 미확인).
- 위험과 피할 것:
  - `cookies.ts` 는 auth 디렉터리지만 이 변경은 순수 파서 3줄이다. `auth/session.ts`·`auth/principal.ts`·`http/pipeline.ts` 의 세션 판정 로직은 건드리지 말 것. 정상 인코딩 값(`abc%20def` → `abc def`)의 해석이 바뀌면 로그인·CSRF 가 깨진다.
  - `db.ts:64,82-83` 와 `mail/smtp.ts:30-31` 에도 보호 없는 `decodeURIComponent` 가 있지만 입력이 운영자 설정값(DATABASE_URL·SMTP URL)이고 공격면이 다르다. 이번에는 건드리지 말 것(별 아이디어로 남겨 뒀다).
  - 서명 왕복은 "같은 값을 읽는 경로가 둘" 인 전형이다. 서버만 고치고 끝내지 말고 웹(`Errors.tsx:59` 의 단일 `encodeURIComponent`)이 보내는 것과 라우터가 읽는 것이 같은 값이 되는지 끝에서 끝까지 맞춰 볼 것. 웹 쪽을 이중 인코딩으로 바꾸는 방향은 택하지 말 것(라우터 계약이 정본).
  - 마이그레이션·`.github/workflows` 는 손대지 않는다.
- 차선 후보: 연봉 집계의 최소 표본 억제와 "내 연봉을 먼저 등록해야 남의 것을 본다" 규칙에 대한 서버측 회귀 테스트 (DB 필요, 작업량 M). 그다음은 `media/image.ts` 의 EXIF·GPS 제거 재인코딩 테스트.
