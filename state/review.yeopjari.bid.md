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

