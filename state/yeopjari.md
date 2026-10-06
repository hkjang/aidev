## 2026-10-04
- 선택: 클라이언트 퍼센트 인코딩 하드닝 — 쿠키 파서 500 과 오류 서명 이중 디코딩 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `readCookies` 의 보호 없는 `decodeURIComponent` 를 쿠키 단위 try/catch 로 감쌌다(읽을 수 없는 쿠키만 없는 것으로 취급 — 같은 헤더의 CSRF 토큰을 함께 잃지 않게). `Cookie: yj_s=%` 하나로 URIError 가 pipeline 바깥 catch 까지 올라가 비로그인 공개 경로까지 500 + 운영자 알림이 되던 경로가 막혔다. 같은 계열로 `admin/errors/:signature` 핸들러의 중복 디코딩을 없앴다(라우터가 이미 세그먼트를 디코딩하므로, 리터럴 `%` 를 품은 비ASCII 서명이 지워지지 않던 문제). 프로덕션 파일 2개, 테스트 1개. 검증: `npx vitest run packages/core/src/__tests__` (9 파일 118건 통과), `npm test`, `npm run lint`, `npm run typecheck` 모두 통과.
- 실패 재현: `FAIL packages/core/src/__tests__/cookies-params.test.ts > readCookies > drops only the broken cookie and keeps the rest of the header` / `URIError: URI malformed ❯ Module.readCookies packages/core/src/auth/cookies.ts:39:27` (고치기 전 3건 실패 / 6건 통과)
- 보류 아이디어: PR 에서 도는 CI 워크플로가 없다(lint/typecheck/test) — 보호 경로라 사람 승인 회차용 [5/3/M] · 연봉 최소 표본 억제와 선등록 규칙의 서버측 회귀 테스트(DB 필요) [4/2/M] · `media/image.ts` EXIF·GPS 제거 재인코딩 테스트 [4/2/M] · `http/pipeline.ts` CSRF·본문 크기 한계 단위 테스트(DB 를 띄우는 형태로만) [3/2/M] · `db.ts`·`mail/smtp.ts` 의 보호 없는 `decodeURIComponent`(입력이 운영자 설정값이라 공격면이 다름) [2/1/S]
- 과제서: 채택 — 근거가 지금 코드와 정확히 맞았다(cookies.ts:39, ops.ts:518, router.ts:97-103 선례까지). 다만 수용 기준 3(b) 는 라우터 왕복으로만 증명 가능해 핸들러 자체를 지나는 red 테스트는 만들지 못했다(이유는 구현 노트).

## 2026-10-05
- 선택: 사진 정제의 EXIF·GPS 제거를 실제 JPEG 회귀 테스트로 보장 (가치 4 / 위험 1 / 작업량 M)
- 결과: 성공
- 요약: image.test.ts 1파일에 실제 jpeg-js.encode/decode로 만든 합성 JPEG의 TIFF GPS IFD·위경도 RATIONAL·COM 존재를 입증하고, 실제 sanitise의 축소 유무 × full/thumb 4경우에서 정상 디코딩·예상 크기·메타데이터 제거를 검증하는 총 7건을 추가했다. JPEG 세그먼트 길이를 따라 SOS 이전 헤더를 검사하며 EXIF 원본 및 COM만 있는 원본을 각각 거부하는 대조 assertion을 포함했고 프로덕션 코드는 바꾸지 않았다. Node v22.23.1에서 npm ci 성공 후 npm test -- packages/core/src/__tests__/image.test.ts(입력 검증 1건, 완성 후 7건), npm run check(린트·3 workspace 타입 검사·17파일 180건), npm run build 모두 exit 0; 커밋 14f79f7.
- 실패 재현: 못 함 — 기존 sanitise가 이미 메타데이터를 제거하므로 프로덕션 결함의 red→green은 없으며 입력 검증 첫 실행부터 통과했다. EXIF APP1 must be removed / COM must be removed assertion이 메타데이터가 남은 실제 JPEG를 거부함을 각각 toThrow로 확인했다(이 대조군을 기존 코드의 실패 재현으로 주장하지 않음).
- 보류 아이디어: Node StaticFiles 실제 파일 기반 HEAD·SPA·경로 격리 테스트 [3/1/S] — 이번 차선 후보, 현재 shell fallback 계약 유지.
- 보류 아이디어: 사진 입력 바이트·형식 한계의 AppError 회귀 테스트 [3/1/S] — 실제 JPEG와 손상 입력으로 별도 검증.
- 보류 아이디어: 연봉 RLS E2E 쿼리·결과 파싱 실패를 성공 처리하지 않게 한다 [4/2/M] — 격리 PostgreSQL 필요.
- 보류 아이디어: http/pipeline.ts CSRF·본문 크기 한계 테스트 [3/2/M] — 실제 DB·createApp 배선 필요, 대역 금지.
- 과제서: 채택 — 현재 sanitise와 mediaRoutes가 정찰 근거와 일치하고 지정된 테스트 1파일만으로 수용 기준을 충족했다.

## 2026-10-05
- 선택: IndexNow 알림이 publicUrl 설정 오류에 트랜잭션 뒤에서 터지지 않게 하고, 키·요청 본문 계약을 테스트로 묶는다 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `pingIndexNowStatus` 의 `new URL(env.publicUrl)` 이 try 밖에 있어 스킴 없는 설정값(`yeopjari.bid`) 하나로 TypeError 가 호출자까지 올라갔다 — `jobs/disclosures.ts:296` 은 트랜잭션 커밋 뒤에 부르므로 글을 다 올린 아침 공시 작업이 실패로 기록되고 `/api/v1/internal/indexnow` 는 500 이 된다. 파싱을 try 로 감싸 `error publicUrl 설정이 주소가 아닙니다` 를 돌려주고(dev/offline/빈 목록은 그대로 `skipped`, `sent`·`http n`·`error …` 문자열은 손대지 않음 — `ops.ts` 가 그대로 노출한다), 프로덕션 파일은 `seo/indexnow.ts` 1개만 고쳤다. 새 테스트 19건은 스텁한 `globalThis.fetch` 가 실제로 받은 본문·헤더를 읽어 assert 한다(실제 네트워크 호출 없음): 잘못된 publicUrl 3값에서 비던짐·fetch 미호출, 키가 `public-pages.ts` 의 `/^[0-9a-f]{32}\.txt$/` 게이트 통과·같은 env 결정성·tokenSign 다르면 키도 다름, host/key/keyLocation/urlList·중복 제거·1000개 상한, `user-agent` 헤더(c730b67 회귀), `sent`/`http 429`/80자 절단 오류. 검증: `npx vitest run …/indexnow.test.ts` (19/19), `npm run check` = lint + 3 워크스페이스 typecheck + `vitest run` 28파일 247건 모두 exit 0. 커밋 507ea8a.
- 실패 재현: `FAIL packages/core/src/__tests__/indexnow.test.ts > pingIndexNowStatus 설정 오류 > publicUrl 이 "yeopjari.bid" 면 pingIndexNow 는 false 다` / `Caused by: TypeError: Invalid URL ❯ pingIndexNowStatus packages/core/src/seo/indexnow.ts:31:18` (고치기 전 7건 실패 / 12건 통과 — 계약 테스트 12건은 기존 동작을 그대로 고정하므로 처음부터 통과)
- 보류 아이디어: PR 에서 도는 CI 워크플로가 없다(lint/typecheck/test) — 보호·릴리즈 경로라 사람 승인 회차용 [5/3/M] · 연봉 RLS E2E 가 쿼리·결과 파싱 실패를 성공 처리하지 않게(격리 PostgreSQL 필요) [4/2/M] · Node StaticFiles 실제 파일 기반 HEAD·SPA·경로 격리 테스트(차선 후보 3회 연속, server 워크스페이스에 추적 테스트 0) [3/1/S] · 사진 입력 바이트·형식 한계의 AppError 회귀 테스트 [3/1/S] · http/pipeline.ts CSRF·본문 크기 한계 단위 테스트(실제 DB·createApp 배선 필요) [3/2/M]
- 과제서: 채택 — 근거가 지금 코드와 정확히 맞았다(indexnow.ts:31 의 try 밖 `new URL`, disclosures.ts:296 의 커밋 뒤 호출, public-pages.ts:775 의 키 게이트, ops.ts:477 의 문자열 노출까지). 다만 프로필이 말한 cookies-params.test.ts 는 이 main 에 없다(2026-10-04 회차 미머지).

## 2026-10-07
- 선택: 어제 추가된 입력 가드 3개(queryInt·isRealDay·isUuid)를 회귀 테스트로 묶는다 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `b576143` 이 GET 116경로 퍼징으로 찾은 500 들을 `util/validate.ts` 의 `queryInt`·`isRealDay`·`isUuid` 로 막았는데 이 셋을 검사하는 테스트가 0건이어서, `validate.test.ts` 에 describe 3개(12건)를 추가했다. 호출처를 grep 으로 다시 확인해(queryInt 21곳·isRealDay 7곳·isUuid 6곳 — 과제서가 센 17곳의 상위집합) 실제 인자 조합 네 가지((50,1,200)·(0,0,100_000)·(0,0,MAX_SAFE_INTEGER)·(20,1,50))만 썼고, junk 12입력 × 4조합에서 반환값이 항상 `Number.isSafeInteger` 임을 단정한다. 프로덕션 코드는 바꾸지 않았다 — 2-c 가 의심한 `fallback` 미검사는 `registry.ts:120~127` 이 `ui.list_page_size`/`ui.list_page_max` 를 `type:'int', min:5, max:50` 으로 선언해 비정수 유입 경로를 찾지 못했고, 실발생 없는 방어는 넣지 않았다. 검증: `npx vitest run …/validate.test.ts`(29건 통과), `npm run check`(lint + 3 워크스페이스 typecheck + 28파일 244건) 모두 exit 0. 커밋 `df2be97`, 변경 파일 1개(테스트).
- 실패 재현: 못 함 — 기존 가드가 이미 올바르므로 프로덕션 결함의 red→green 이 없다. 대신 테스트가 가드 코드를 실제로 지나는지 돌연변이로 증명했다: 가드를 b576143 이전의 느슨한 형태(`queryInt` 유한성 검사 제거·`isRealDay` 정규식만·`isUuid` `length === 36`)로 되돌리면 `AssertionError: abc -> NaN: expected false to be true` / `expected 1.9 to be 1` / `expected NaN to be 50` / `expected true to be false`(×2) 로 5건 실패(24 통과), UUID 정규식의 `^$` 앵커만 떼면 2건 실패(27 통과). 이후 `validate.ts` 를 복구해 `git diff` 가 비어 있음을 확인했다. 이 돌연변이 실패를 현재 코드의 결함으로 주장하지 않는다.
- 보류 아이디어: PR 에서 도는 CI 워크플로가 없다 (lint/typecheck/test) — 보호·릴리즈 경로라 사람 승인 회차용 [5/3/M] · 연봉 RLS E2E 가 쿼리·결과 파싱 실패를 성공 처리하지 않게 한다(격리 PostgreSQL 필요) [4/2/M] · 쿠키 파서의 보호 없는 decodeURIComponent — 2026-10-04 회차 PR 이 아직 미머지라 중복 구현 위험 [4/1/S] · Node StaticFiles 의 실제 파일 기반 HEAD·SPA·경로 격리 테스트(차선 후보 5회 연속, server 워크스페이스 추적 테스트 0) [3/1/S] · normalisePath·redactSecrets 가 client-errors.ts 를 통해 처음으로 외부 입력을 받는다 — 계약 테스트 [3/1/S]
- 과제서: 채택 — 근거가 지금 코드와 정확히 맞았다(validate.test.ts 의 describe 6개에 새 가드가 없고, 지정된 네 가지 인자 조합과 isRealDay/isUuid 호출처가 모두 실재했다). 2-c 의 `fallback` 결함은 과제서의 유보대로 실발생을 확인하지 못해 프로덕션 변경 없이 끝냈다.

