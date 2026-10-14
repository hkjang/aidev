## 2026-10-06
- (원장 항목에 비밀/내부 정보 의심 문자열이 있어 비공개 기록으로 옮김 — run 2026-10-06-191811-review.yeopjari.bid-improve)

- 릴리즈: 0.3.0 (2026-10-06, run 2026-10-06-191811-review.yeopjari.bid-improve)
## 2026-10-14
- 선택: 리뷰 쓰기의 파트너 이름란이 띄어쓰기를 삼키던 것 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: Write.tsx 의 파트너 이름 TextInput 이 `value={form.partner_name.trim()}` 로 제어돼, 한 글자씩 입력하면 공백을 넣은 순간 React 가 DOM 값을 되돌려 "메가존 클라우드" 가 "메가존클라우드" 로 붙어 버렸다. 같은 칸을 비우면 표시 조건(`form.partner_name ?`)이 거짓이 되어 입력란 자체가 사라지는 문제도 함께 있었다. 화면 전용 `partner_by_name` 플래그로 표시 여부를 선택에 묶고 값은 그대로 보여 주게 고쳤다(프로덕션 파일 1개). 검증: scripts/ui-flows.mjs(실제 크로미움, 실제 컴포넌트)에 검사 2건 추가 — 고치기 전 2건 실패, 고친 뒤 19 passed / 0 problem. 더해서 `npm run check`(lint·타입·단위 118건), `node scripts/e2e.mjs`(218 passed / 0 failed, dev-reset 직후), `node scripts/a11y.mjs`(심각 위반 0건) 전부 통과. 화면이 보내는 본문을 실제 API 로 왕복시켜 공백 있는 이름이 DB 까지 그대로 저장되는 것도 확인했다.
- 실패 재현: `FAIL  파트너 이름: 띄어쓰기가 있는 이름을 그대로 입력 :: value="메가존클라우드"` / `FAIL  파트너 이름: 다 지워도 입력란이 남아 다시 쓸 수 있음`
- 보류 아이디어: ① util/text.ts 나머지 순수 함수(mailboxKey·maskEmail·koreanParticle·excerpt·tidyBody) 단위 테스트 보강 [3/1/S] ② 일·월 쿼터의 window_start 가 KST 날짜를 UTC 자정으로 라벨링해 Retry-After 가 9시간 과장됨 [2/2/S, 신규] ③ clientIpFrom 이 cf-connecting-ip 빈 문자열에서 폴백하지 않음 [2/1/S] ④ e2e 상태 의존 검사 멱등성 — 이번 회차 관측으로는 dev-reset 직후 218/0 이라 "더러운 DB 에서만 실패" 가 확정됨 [3/2/M] ⑤ domain/products.ts 의 slugify 가 어디서도 쓰이지 않는 죽은 코드(한글 이름은 빈 슬러그가 됨) [1/1/S, 신규]
- 과제서: 기각 — 정찰 과제서가 없었음(journal.md 에 러너 2줄뿐), 1~4단계를 직접 수행했다.

- 릴리즈: 0.7.4 (2026-10-14, run 2026-10-14-081827-review.yeopjari.bid-improve)
## 2026-10-14
- 선택: 하루 한도 429 의 Retry-After 가 최대 9시간 과장되는 것 — 재시도 시점을 KST 실제 경계로 계산 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `quota.ts` 의 `windowStart` 는 일·월 창을 "KST 날짜의 UTC 자정"으로 라벨링하는 저장 키인데, `retryAfterSeconds` 가 그 라벨을 실제 창 경계로 믿어 429 의 `retry-after` 가 언제나 정확히 9시간 길게 나갔다(시각에 무관한 상수 오차임을 계산으로 확인). 실제 경계를 구하는 `nextWindowReset(window, at)` 를 새로 두고(day = `kstDayStart+24h`, month = 다음 KST 월 1일 자정, minute 은 동일 결과) `retryAfterSeconds` 가 그것만 쓰게 했다. `windowStart` 는 손대지 않아 `quota_counters.window_start` 키가 그대로이고 배포 시점 진행 중인 카운터가 리셋되지 않는다(diff 확인). 검증: 새 단위 테스트 8건(`packages/core/src/__tests__/quota.test.ts`, 고정 시각 — KST 오전·자정 1분 전·자정 정각·UTC 전날 인스턴트·월 롤오버·12월→1월·분 창), `npm run check` 통과(eslint + 타입 3개 + 126 tests), `node scripts/e2e.mjs` 220 passed / 0 failed(dev-reset 직후, 기준선 218 + 신규 2). e2e 에 실제 HTTP 429 의 `retry-after` 헤더를 검사하는 검사를 추가해 프로덕션 배선까지 통과하는 것으로 증명했다.
- 실패 재현: 단위 — `AssertionError: expected 82800 to be 50400` (2026-10-14T01:00Z 일 한도), `expected 32460 to be 60` (KST 자정 1분 전), 7 failed / 1 passed. e2e — `FAIL  retry-after 가 다음 한국 자정까지 :: retry-after=82351 expected≈49951` (고치기 전 코드로 빌드·재시작해서 실제 서버 응답 헤더로 재현, 219 passed / 1 failed).
- 보류 아이디어: ① util/text.ts 나머지 순수 함수(mailboxKey·maskEmail·koreanParticle·excerpt·tidyBody·clientIpFrom) 단위 테스트 보강 [3/1/S] ② 429 안내에 재시도 가능 시각 표시 — 이제 헤더 값이 정확해져 선행 조건이 풀렸다 [2/2/S] ③ purgeOldCounters 의 `interval '3 days'` 가 월 창 카운터를 매달 4일째 지우는 잠복 함정(지금 month 호출자 0건) [2/1/S] ④ e2e 가 깨끗한 DB 를 암묵적 전제로 하는 것 [3/2/M] ⑤ clientIpFrom 이 cf-connecting-ip 빈 문자열에서 폴백하지 않음 [2/1/S]
- 과제서: 채택 — 근거·수용 기준·손계산 수치(50400/82800)가 모두 코드와 일치했고 범위도 과제서대로 프로덕션 파일 1개로 끝났다. 다만 "e2e 에 review.submit 일 한도 회귀가 있다"는 서술은 사실이 아니었다(e2e 에 쿼터 한도 검사가 없었다) — 그래서 그 검사를 이번에 추가했다.

