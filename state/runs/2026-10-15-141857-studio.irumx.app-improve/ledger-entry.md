## 2026-10-15
- 선택: 해석되지 않는 Range 헤더에 416 대신 전체(200)로 응답 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `src/worker/services/parseRange()` 가 `/^bytes=(\d*)-(\d*)$/` 에 걸리지 않는 모든 Range 를 `'invalid'` 로 돌려주고 `serveObject()` 가 그것을 416 으로 내보내, 여러 구간(`bytes=0-1,5-6`)·모르는 단위(`items=0-5`)·깨진 값을 보내는 클라이언트는 공개 작품 영상·홈 대표 영상·시안·최종본의 본문을 하나도 받지 못했다. 해석 불가는 헤더가 없는 것처럼 `null`(=200 전체)로 분류하고, `'invalid'`(416)는 문법이 맞는데 만족시킬 수 없는 경우(start>end · start>=size)만 남기도록 한 줄을 좁게 바꿨다(프로덕션 1파일, 반환 타입·`serveObject` 헤더 조립은 그대로). 검증은 `npm run build`("빌드 점검 통과")와 `npm test` **76개 전부 통과(58.3s)**, 그리고 고치기 전에 단위·HTTP 시험이 416 으로 실패하는 것을 먼저 확인했다.
- 실패 재현: `✘ [unit] tests/unit-media.spec.ts:60 범위 요청 해석 — ... Expected: null / Received: "invalid"` (`parseRange('bytes=0-1,5-6', 1000)`), 그리고 프로덕션 배선을 지나는 HTTP 시험 `✘ [api] tests/api-portfolio.spec.ts:25 ...(AT-03) Expected: 200 / Received: 416` (공개 영상 `mine.videoUrl` + `Range: bytes=0-1,5-6`).
- 보류 아이디어: ① `seo.ts`·`shell.html` 짝 고정 시험(private 경로 noindex · canonical · `noscript#seo` 의 `esc()`) ② `scripts/portfolio-import.mjs` 드라이런 + 자동 시험 ③ 길이 판정을 mvhd 외에 트랙 mdhd 와 교차 확인 ④ `topLevel` 의 최상위 상자 64개 제한 ⑤ 공유 미리보기용 `twitter:card`·`og:image:alt`(shell.html·seo.ts·verify-build 3파일이 함께 걸린다)
- 과제서: 채택 — 과제서의 근거가 지금 코드와 정확히 맞았고(파일·행 번호 포함), 수용 기준 1~4 를 그대로 구현·검증했다. 다만 종전 통과 수는 78 이 아니라 **76** 이었다(이 base `b00d53c` 에 MP4 회전 회차가 들어 있지 않다).
