# 과제서 — 2026-10-06 (client.irumx.app / 이룸검수)

- 과제: 붙여넣기 분석기 — 메일 안에 섞인 카톡 한 줄이 입력 전체를 가로채 앞부분을 통째로 버리는 문제 (가치 4 / 위험 2 / 작업량 S)

- 왜: `src/shared/kakao.ts`의 `parseConversation` 은 `parseKakao` 를 **가장 먼저** 돌리고, 카톡 머리줄이 **한 줄이라도** 있으면(`parseKakao`: `if (hits === 0) return null`) 형식을 `kakao` 로 확정한다. 그런데 `parseKakao` 는 첫 머리줄이 나오기 전의 줄을 "내보내기 머리말" 로 보고 **아무 말 없이 버린다**(`src/shared/kakao.ts:135-136`, `if (cur) ... 아니면 버림`). 그래서 고객이 "메일 본문 + 아래에 고객이 카톡으로 보낸 한 줄" 을 함께 붙여 넣으면 메일 본문의 요청 목록이 화면에서 사라진다 — 요청 접수의 유일한 입구(`POST /api/w/:wid/p/:pid/requests/import/parse`, `src/worker/routes/requests.ts:163-171`)에서 생기는 조용한 자료 손실이다.
  고치면 카톡 내보내기가 아닌 섞인 붙여넣기는 `list`/`paragraphs` 로 떨어져 **아무 줄도 버리지 않는다**(카톡처럼 보이던 줄도 문단·항목으로 남는다).

- 수용 기준:
  1) 아래 입력을 `parseConversation` 에 넣으면 `format !== 'kakao'` 이고, 메일 본문의 글머리표 항목("배너 교체")이 `messages` 안에 남는다 — 지금 코드는 `format === 'kakao'`, `messages.length === 1`, 앞 세 줄이 사라진다.
     ```
     안녕하세요, 아래 수정 부탁드립니다.
     - 메인 배너 교체
     - 푸터 전화번호 삭제
     고객이 카톡으로 보낸 내용도 함께 붙입니다:
     2026. 10. 5. 오후 3:12, 김고객 : 로고 크게 해주세요
     감사합니다.
     ```
  2) 진짜 카톡 내보내기는 그대로 `kakao` 로 읽힌다 — `tests/kakao.spec.ts` 의 기존 9개(PC 내보내기 머리말 2줄 '…님과 카카오톡 대화' / '저장한 날짜 : …' 가 앞에 붙은 것, 안드로이드·iOS·24시간, **카톡 한 줄만 붙여 넣은 것**(`tests/kakao.spec.ts:58-61`), 목록·문단·자르기·제목)가 **손대지 않고** 통과한다.
  3) 새 테스트가 "버려지는 줄이 없다" 를 증명한다: 입력의 요청 문장(위 2개 항목 + 카톡 한 줄의 내용)이 모두 어느 메시지의 `text` 안에 들어 있는지 확인한다. 머리말(`카카오톡 대화`/`저장한 날짜`)만 있는 진짜 내보내기는 여전히 머리말이 메시지로 새어 나오지 않는다.

- 건드릴 파일 (2개):
  - `src/shared/kakao.ts:94 parseKakao` — 첫 머리줄 이전의 **비어 있지 않은 줄 수**를 세고, 그 줄이 카톡 내보내기 머리말(새 상수 예: `EXPORT_HEADER = /(카카오톡 대화|저장한 날짜\s*:)/`)·`PC_DATE`·`MOBILE_DATE`·`MOBILE_SYSTEM` 이 아니면 "깨끗한 내보내기가 아님" 으로 보고 `null` 을 돌려 `parseConversation` 이 `stripMail`→`parseList`→`parseParagraphs` 로 떨어지게 한다. 머리줄이 전혀 없거나(`hits === 0`) 지금 통과하는 경우의 동작은 바꾸지 말 것.
  - `tests/kakao.spec.ts` — 수용 기준 1·3 의 테스트를 `메일·그 밖의 글` describe 에 추가. 기존 단정은 수정하지 말 것(수정해야 한다면 그 자체가 회귀 신호다).
  - 주석도 함께 고칠 것: `src/shared/kakao.ts:135` 의 "첫 메시지 앞(내보내기 머리말)은 버린다" 는 새 규칙을 설명해야 한다.

- 검증 명령:
  - `npm ci` (이 워크트리는 `node_modules` 가 **비어 있다** — 설치가 먼저다)
  - `npm run check` — `tsc -p tsconfig.app.json/worker/node --noEmit` 세 개. 분석기는 세 쪽 모두에서 쓰이므로 타입은 이걸로 본다.
  - `npm run build && npx playwright test --project=unit` — kakao 9개 + guard 1개. **주의**: `playwright.config.ts` 의 `webServer` 는 프로젝트를 골라도 떠서, `node scripts/db-local.mjs` + `npx wrangler dev --port 8798` 과 가짜 Resend(8799)가 먼저 올라오고 `dist/` 가 있어야 한다. 그래서 `npm run build` 가 선행 조건이다.
  - 전체는 `npm test`(46개, 오래 걸림). 최소한 `--project=unit` 과 `--project=api` 는 돌릴 것.
  - **미확인**: 이 정찰 세션에서는 `npm ci` 실행이 허용되지 않아 위 명령을 직접 돌려 보지 못했다. 명령 자체는 `package.json` scripts·`playwright.config.ts`·`README.md` 에서 확인한 것이다. 설치나 `wrangler dev` 가 이 환경에서 뜨지 않으면 `npm run check` + `--project=unit` 까지로 범위를 줄이고 그 사실을 보고할 것.

- 위험과 피할 것:
  - **분류기 문턱을 비율로 만들지 말 것.** "카톡 머리줄 수 / 전체 줄 수" 같은 비율 규칙은 여러 줄 메시지가 많은 정상 대화(머리줄 3개·이어지는 줄 15개)를 카톡이 아니라고 오판한다. 판단 근거는 **버려지는 줄(첫 머리줄 이전)** 하나로 좁힐 것.
  - `PC_LINE`·`MOBILE_LINE`·`MOBILE_SYSTEM`·`HINT`·`ACK` 정규식은 건드리지 말 것 — `suggested` 판정이 함께 바뀌면 기존 단정 9개가 같이 흔들려 원인 분리가 안 된다. 이번 변경은 **형식 판정(어느 파서를 쓰는가)만**이다.
  - 같은 값을 읽는 경로가 둘이다: 서버의 `parseConversation`(`src/worker/routes/requests.ts:170`)과 화면의 `suggestTitle`(`src/client/pages/requests.tsx:212,257`). 서버가 돌려준 `messages[].text` 로 화면이 제목을 만들므로, 형식이 `list`/`paragraphs` 로 바뀌면 `author`·`at` 이 빈 문자열이 된다 — `requests.tsx` 의 붙여넣기 화면이 빈 `author`/`at` 을 이미 다루는지 확인하고(목록·문단 형식은 지금도 그렇게 돌아간다) 화면 쪽은 바꿀 필요가 없음을 확인만 할 것.
  - 보호 경로(`src/worker/auth.ts`, `src/worker/access.ts`, `migrations/`, `src/worker/services/approvals.ts`)와 `wrangler.jsonc` 는 건드리지 말 것. 이 과제는 승인·권한·DB 와 무관하다.
  - `MAX_INPUT`/`MAX_MESSAGES` 와 `truncated` 동작을 바꾸지 말 것(기존 테스트 `너무 긴 입력은 자르고 알린다` 가 300개·`truncated===true` 를 잰다).

- 차선 후보: **`MAX_INPUT` 경계에서 서버가 자르는 길이 자체가 죽은 코드인 점을 맞추기** — `src/worker/routes/requests.ts:169` 가 `text.length > MAX_INPUT` 을 400 으로 먼저 거절하므로, `normalize()` 의 `truncated` 는 API 경로에서는 절대 켜지지 않는다(테스트는 `MAX_MESSAGES` 쪽만 증명한다). 라우트에서 자르고 `truncated` 를 켜 줄지, 아니면 거절을 정본으로 두고 `normalize` 의 자르기와 `ParseResult.truncated` 주석을 "메시지 수 초과만" 으로 정리할지 **한쪽을 정해** 두 경로가 같은 입력을 같은 값으로 읽게 만든다(파일 2개: `src/worker/routes/requests.ts`, `src/shared/kakao.ts` + 테스트).
