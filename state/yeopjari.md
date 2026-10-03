## 2026-10-04
- 선택: 클라이언트 퍼센트 인코딩 하드닝 — 쿠키 파서 500 과 오류 서명 이중 디코딩 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `readCookies` 의 보호 없는 `decodeURIComponent` 를 쿠키 단위 try/catch 로 감쌌다(읽을 수 없는 쿠키만 없는 것으로 취급 — 같은 헤더의 CSRF 토큰을 함께 잃지 않게). `Cookie: yj_s=%` 하나로 URIError 가 pipeline 바깥 catch 까지 올라가 비로그인 공개 경로까지 500 + 운영자 알림이 되던 경로가 막혔다. 같은 계열로 `admin/errors/:signature` 핸들러의 중복 디코딩을 없앴다(라우터가 이미 세그먼트를 디코딩하므로, 리터럴 `%` 를 품은 비ASCII 서명이 지워지지 않던 문제). 프로덕션 파일 2개, 테스트 1개. 검증: `npx vitest run packages/core/src/__tests__` (9 파일 118건 통과), `npm test`, `npm run lint`, `npm run typecheck` 모두 통과.
- 실패 재현: `FAIL packages/core/src/__tests__/cookies-params.test.ts > readCookies > drops only the broken cookie and keeps the rest of the header` / `URIError: URI malformed ❯ Module.readCookies packages/core/src/auth/cookies.ts:39:27` (고치기 전 3건 실패 / 6건 통과)
- 보류 아이디어: PR 에서 도는 CI 워크플로가 없다(lint/typecheck/test) — 보호 경로라 사람 승인 회차용 [5/3/M] · 연봉 최소 표본 억제와 선등록 규칙의 서버측 회귀 테스트(DB 필요) [4/2/M] · `media/image.ts` EXIF·GPS 제거 재인코딩 테스트 [4/2/M] · `http/pipeline.ts` CSRF·본문 크기 한계 단위 테스트(DB 를 띄우는 형태로만) [3/2/M] · `db.ts`·`mail/smtp.ts` 의 보호 없는 `decodeURIComponent`(입력이 운영자 설정값이라 공격면이 다름) [2/1/S]
- 과제서: 채택 — 근거가 지금 코드와 정확히 맞았다(cookies.ts:39, ops.ts:518, router.ts:97-103 선례까지). 다만 수용 기준 3(b) 는 라우터 왕복으로만 증명 가능해 핸들러 자체를 지나는 red 테스트는 만들지 못했다(이유는 구현 노트).

