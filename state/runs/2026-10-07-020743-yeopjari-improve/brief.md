- 과제: 어제 추가된 입력 가드 3개(queryInt·isRealDay·isUuid)를 회귀 테스트로 묶는다 (가치 4 / 위험 1 / 작업량 S)
- 왜: `b576143`(2026-10-06)이 GET 116개 경로 퍼징으로 찾은 500들을 `packages/core/src/util/validate.ts` 의 새 함수 `queryInt`·`isRealDay`·`isUuid` 로 막았는데, 이 셋을 검사하는 테스트가 한 줄도 없다(`packages/core/src/__tests__/validate.test.ts` 의 describe 는 string/text/int/uuid and email/composite/v.text with min 0 뿐 — 실제로 열어 확인했다). 호출처는 12개 라우트 파일 17곳(admin/catalog·companies·members·moderation·settings·ops, community, chat, trust, perks, ads, orders, public-pages)이고, 가드가 조용히 느슨해지면 `limit=1.5`·`2026-02-31`·대시 36개 UUID 흉내가 다시 SQL 로 내려가 운영 500 과 운영자 알림 메일로 돌아온다.
- 수용 기준:
  1) `packages/core/src/__tests__/validate.test.ts` 에 `queryInt`·`isRealDay`·`isUuid` 를 각각 검사하는 describe 3개가 추가되고, 기존 테스트는 손대지 않은 채 모두 통과한다. 프로덕션 코드는 바꾸지 않는 것이 기본값이다(아래 2-c 예외만 허용).
  2) 테스트가 **실제 호출처의 인자 모양**으로 호출된다. 추측한 모양이 아니라 아래 조합을 그대로 쓸 것:
     - `queryInt(raw, 50, 1, 200)` (admin/companies.ts:26, members.ts:30, moderation.ts:40, catalog.ts:210)
     - `queryInt(raw, 0, 0, 100_000)` (admin/companies.ts:27, catalog.ts:211)
     - `queryInt(raw, 0, 0, Number.MAX_SAFE_INTEGER)` (chat.ts:152 `after_seq`)
     - `queryInt(raw, 20, 1, 50)` (community.ts:1503 의 `pageInput`, trust.ts:240)
     - `isRealDay` — orders.ts:355~356(유료 광고 시작일), ads.ts:194, perks.ts:259, admin/ops.ts:473·597
     - `isUuid` — community.ts:70, perks.ts:96, public-pages.ts:661
  3) 테스트가 증명해야 하는 것(각 항목이 assertion 하나 이상):
     - `queryInt`: `'abc'`·`'1.5'`·`'NaN'`·`'Infinity'`·`'1e999'`·`null`·`undefined`·`''` 에서 **반환값이 항상 정수이고 `Number.isSafeInteger`를 만족**한다(= `LIMIT NaN`/`LIMIT 1.5` 가 SQL 로 내려갈 수 없다). `'1.9'`→1(trunc, 반올림 아님), `'-5'`→min 으로 clamp, `'99999'`→max 로 clamp, `'1e30'`→`Number.MAX_SAFE_INTEGER` 로 clamp, `'20'`→20.
     - `isRealDay`: `'2026-02-31'`·`'2026-02-29'`(2026 은 평년)·`'2026-13-01'`·`'2026-00-10'`·`'2026-1-01'`·`'20260101'`·`''`·`null`·`undefined`·`'2026-01-01T00:00:00Z'` 는 모두 false, `'2026-02-28'`·`'2024-02-29'`(윤년)·`'2026-12-31'` 는 true. 특히 `'2026-02-31'` 이 false 라는 단정에 "결제 처리 중 `::date` 캐스트가 실패했던 경로(orders.ts:355)" 를 주석 한 줄로 남긴다.
     - `isUuid`: 소문자·대문자 정상 UUID 는 true, 대시 36개(`'-'.repeat(36)`)·길이만 맞는 쓰레기·`'constructor'`·`'__proto__'`·앞뒤 공백 붙은 정상 UUID·중괄호로 감싼 형태·`null`·`undefined` 는 false. (`'constructor'`/`'__proto__'` 는 b576143 커밋 메시지가 지목한 내장 이름 계열이므로 반드시 포함.)
  2-c) 테스트를 쓰다 **실제 결함**을 찾으면(가장 의심스러운 자리: `queryInt` 가 `raw` 의 유한성만 검사하고 `fallback`·`min`·`max` 는 검사하지 않는다 — `community.ts:1503` 은 `fallback`/`max` 를 운영 설정 `ui.list_page_size`/`ui.list_page_max` 에서 받는다) 수정은 `packages/core/src/util/validate.ts` **한 파일 안에서만** 하고, 고치기 전 red 테스트를 먼저 남길 것. 다만 registry.ts:120~127 은 두 설정을 `type:'int', min:5, max:50` 으로 선언하고 있어 **실발생은 확인되지 않았다** — 실발생을 주장하지 말고, 하더라도 "fallback 도 정수로 clamp" 수준의 방어로 그치고 반환 계약(위 3번)은 바꾸지 말 것.
- 건드릴 파일:
  - `packages/core/src/__tests__/validate.test.ts` — 끝에 `describe('queryInt', …)`, `describe('isRealDay', …)`, `describe('isUuid', …)` 추가. import 줄에 `queryInt, isRealDay, isUuid` 를 더한다(현재 파일은 `../util/validate.js` 에서 import 한다 — 확장자 `.js` 관례를 지킬 것).
  - (조건부, 2-c 에 해당할 때만) `packages/core/src/util/validate.ts:47` `queryInt` — fallback 방어 한 줄.
- 검증 명령:
  - 이 작업 트리에 `node_modules` 가 없다(확인함). 먼저 `npm ci`.
  - `npx vitest run packages/core/src/__tests__/validate.test.ts`
  - `npm run check` (= `npm run lint && npm run typecheck && npm run test`; `vitest run` 은 현재 `packages/core/src/__tests__` 26파일)
- 위험과 피할 것:
  - **프로덕션 라우트 파일을 건드리지 말 것.** 17개 호출처 중 하나라도 손대면 파일 수가 터지고(이 저장소 교훈: 파일 10개 이상 변경은 사람 손을 다시 탄다) `limit`/`offset` 의미가 바뀌어 목록 API 가 회귀한다.
  - `validate.ts` 의 기존 `int`/`uuid` 검증기(`v.int`, `v.uuid`)는 다른 계약(필드 에러 수집)을 지킨다. 새 가드와 **통합하려 하지 말 것** — 좁히는 방향으로만.
  - `http/pipeline.ts`(어제 들어간 `%00` 거부, `MAX_JSON_BYTES`)·`db.ts`(재시도 백오프)는 이번 과제 범위가 아니다. 보호 경로(auth/cookies.ts, db/migrations, .github/workflows)는 열지 말 것.
  - 소스 문자열 검사(grep 로 "가드가 있다")를 근거로 삼지 말 것. 실제 함수를 import 해 실행한 결과로만 단정할 것.
  - 중복 금지: `image.test.ts`·`indexnow.test.ts`·`cookies-params.test.ts` 는 이 main(`cd18b14`)에 **없다** — 앞선 3회차 PR 이 아직 머지되지 않았다. 그 세 가지를 다시 만들지 말고, 이번 테스트도 그 파일들과 충돌하지 않게 `validate.test.ts` 안에서만 작업할 것.
- 차선 후보: Node `StaticFiles` 의 실제 파일 기반 HEAD·SPA·경로 격리 테스트 (`apps/server/src/static.ts` 의 `serve`/`shell`/`resolve`, 임시 디렉터리와 실제 `Request` 로 새 테스트 1파일). server 워크스페이스에 추적 테스트가 0개이고 vitest include 는 이미 `apps/**` 를 잡는다. shell fallback 계약과 symlink 정책은 바꾸지 말 것.
