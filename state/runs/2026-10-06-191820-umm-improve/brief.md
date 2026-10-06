# 과제서 (2026-10-06, base main@84daebc / VERSION 0.76.6)

- 과제: 같은 공간을 네 가지로 내려받으면 파일 이름의 기준이 서로 다르다 — 네 곳 중 PDF 만 새니타이즈한다 (가치 3 / 위험 1 / 작업량 S)

- 왜: `web/src/pages/CanvasPage.tsx` 의 내려받기 네 곳이 모두 같은 한 값 `activeName`(`:2131` — 지난 회차에 `spaceDisplayName()` 으로 모인 사용자가 지은 공간 이름)을 파일 이름에 쓰는데, 네 곳 중 `downloadPDF`(`:2058`)만 `activeName.replace(/[\\/:*?"<>|]/g, '-')` 로 금지 문자를 바꾸고 `downloadMarkdown`(`:1967` `.md`)·`downloadOutline`(`:1991` `-차례.md`)·`downloadImage`(`:2025` `.png`) 세 곳은 날것으로 쓴다. 공간 이름이 `2026/Q4 기획` 이면 PDF 만 `umm-2026-Q4 기획.pdf` 로 결정적으로 떨어지고 나머지 셋은 브라우저가 알아서 고친 이름이 되어(크롬은 `/`·`:` 를 `_` 로 바꾼다 — 이 동작은 **미확인**, 아래 수용 기준은 브라우저에 의존하지 않게 적었다), 같은 공간에서 나온 네 파일이 서로 다른 기준의 이름을 갖는다. 고치면 파일 이름 결정이 네 곳에 흩어진 네 계약에서 한 함수의 한 계약으로 모이고, 지난 회차가 `spaceDisplayName` 으로 시작한 "한 값에서 머리글과 파일 이름이 함께 나온다" 를 끝까지 잇는다.

- 수용 기준:
  1. 새 순수 모듈 `web/src/lib/download-name.ts` 의 `downloadFileName(name, extension, suffix?)` 하나가 네 곳의 파일 이름을 만들고, 네 곳 모두 금지 문자 `\ / : * ? " < > |` 를 `-` 로 바꾼 **같은 기준**(base name 이 글자 그대로 같음)을 쓴다. `CanvasPage.tsx` 안에 남는 인라인 `replace(...)` 나 인라인 템플릿 파일 이름은 0건.
  2. PDF 경로의 **지금 출력이 바뀌지 않는다**: 이미 맞게 돌던 유일한 곳이므로 문자 집합을 넓히거나 줄이지 않는다(개행·제어문자는 이 과제에서 다루지 않는다 — 아이디어로 남긴다). 확장자 앞 `umm-` 접두사와 차례의 `-${t('차례')}` 접미사 위치도 그대로.
  3. 테스트(`web/src/lib/download-name.test.ts`)가 증명할 것:
     - 같은 입력 하나(예: `2026/Q4: 기획*안`)에 대해 `.md`·차례 `.md`·`.png`·`.pdf` **네 가지 호출의 base name 이 서로 같다** — 이것이 이번 결함이다. 수정 전에는 세 곳이 날것이라 이 단언이 떨어져야 한다.
     - 금지 문자 9종(`\ / : * ? " < > |`)이 각각 `-` 로 바뀐다.
     - 금지 문자가 없는 이름(`생각 공간`, `My space`)과 공백·한글은 **그대로 남는다**(통제 시험 — "전부 거절" 로 통과하는 것이 아니라는 증거).
     - `suffix` 가 주어지면 `umm-<safe>-<suffix>.md`, 없으면 `umm-<safe>.<ext>` 꼴.
     - 손으로 만든 대역·가짜 `t` 금지. 접미사는 호출부가 `t('차례')` 로 넘기는 값이므로 테스트는 실제 `web/src/i18n/translate.ts` 의 `translate('차례')` 를 넘겨 쓸 수 있다(실제 `en.ts` 사용).

- 건드릴 파일 (프로덕션 2개):
  - `web/src/lib/download-name.ts` — **신규**. `downloadFileName(name: string, extension: string, suffix?: string): string`. 본문은 `umm-` + `name.replace(/[\\/:*?"<>|]/g, '-')` + (`suffix` 있으면 `-${suffix}`) + `.${extension}`. `space-name.ts` 와 같은 문체의 doc comment 로 "왜 Go 쪽과 통합하지 않는가" 를 한 문단 적을 것.
  - `web/src/pages/CanvasPage.tsx` — `downloadMarkdown`(`:1967`), `downloadOutline`(`:1991`), `downloadImage`(`:2025`), `downloadPDF`(`:2058`) 네 줄만 교체. import 는 `:114` 근처 기존 `../lib/...` 묶음에 추가. `activeName` 의 정의(`:2131`)·캔버스 머리글(`:2277`)·자식 prop(`:3525`)은 **건드리지 말 것**.
  - `web/src/lib/download-name.test.ts` — 신규(테스트).

- 검증 명령 (이 저장소에서 실제로 도는 것. **`npm ci --prefix web` 선행이 필수** — 없으면 `run-on-supported-node: vitest is not installed …` 로 exit 1 이고 이것은 제품 결함이 아니다):
  ```
  npm ci --prefix web
  npm --prefix web test --silent          # 기준선 21파일 / 225시험 (17e1741 기준). 새 시험만큼 늘어야 한다
  npm --prefix web run typecheck
  npm --prefix web run lint               # 기존 파일에 경고가 이미 있다. 새 파일 2개는 무경고여야 한다
  node web/scripts/check-i18n.mjs         # 1060키
  npm --prefix web run build
  scripts/check-version.sh
  ```
  Go 는 0줄이므로 돌리지 않는다(돌렸다고 쓰지 말 것). 실패 재현은 `download-name.ts` 에 수정 전 식(날것 `umm-${name}.md`)을 글자 그대로 넣어 "네 base name 이 같다" 단언이 떨어지는 것을 먼저 찍고, 되돌려 통과시킬 것.

- 위험과 피할 것:
  - **Go 쪽 네 함수와 통합하지 말 것** — `internal/httpapi` 의 `safeFilename`·`dispositionSafe`·`attachmentDisposition`·`handoffFilename` 은 계약이 서로 다르고, 운영자 규칙 3번 후반("계약이 다른 파서를 통합하려 하지 말 것")과 프로필의 "자주 깨지는 곳" 에 명시돼 있다. 이번 과제는 **웹 쪽 네 곳끼리만** 맞춘다. Go 파일 0개.
  - `content_disposition.go` 주석 정리(보류 아이디어)를 끌어오지 말 것 — Go 를 열면 과제가 커지고 검증에 DSN 이 필요해진다.
  - `web/scripts/check-i18n.mjs` 는 `src/` 의 비-테스트 `.ts`/`.tsx` 에서 `t()`/`translate()`/`msg()` 밖의 한글 리터럴을 **거부**한다(주석은 공백으로 지워 통과). 따라서 `download-name.ts` 본문에 한글 리터럴을 넣지 말 것 — 한글 doc comment 는 안전하고, `.test.ts` 는 검사 대상에서 빠지므로 테스트의 한글은 자유다.
  - `spaceDisplayName` 이 이미 `trim()` 을 하므로 **trim 을 다시 넣지 말 것**(같은 값을 두 번 처리하는 것은 운영자 규칙 7번의 패턴). 빈 이름 대체도 거기서 끝났다.
  - 보호 경로 비해당: `internal/auth/`·`migrations/`·`.github/workflows/`·`Dockerfile` 전부 건드리지 않는다. 따라서 `docker build` 확인은 필요 없다(운영자 규칙 4번 비해당).
  - 버전을 올리지 말 것(개선 회차 관례). 커밋에 Co-Authored-By/Claude 트레일러를 넣지 말 것.
  - `dist/` 산출물이 생기면 커밋하지 말 것(`git status` 로 확인 — `dist/` 는 ignore 되어 있다).

- 차선 후보: **`edge-vocabulary.ts` 의 알려진 라벨 12개를 실제 타입·실제 `translate` 로 못 박기** (가치 2 / 위험 1 / S). `web/src/lib/` 에서 유일하게 `.test.ts` 가 없는 모듈이고, `relationLabel`/`originLabel` 의 `?? relation` 폴백이 사전에 없는 값을 받으면 서버의 원시 식별자(`supports`)를 화면에 내보낸다. 다만 **지금 그 폴백은 도달 불가**다 — `migrations/010_memory_graph.sql:55-60` 의 CHECK 제약이 relation/origin 을 각각 정확히 그 6값으로 묶어 두었으므로(직접 읽어 확인), 이 과제는 "버그 수정" 이 아니라 **테스트 공백 보강 + 드리프트 방어**(7번째 relation 을 DB·Go 에 넣고 프런트 사전을 잊으면 원시 식별자가 화면에 뜨는 것)로 정직하게 적어야 한다. 시험은 `web/src/api.ts:86,89` 의 실제 `EdgeRelation`/`EdgeOrigin` 과 실제 `setLocale`/`translate`/`en.ts` 로 ko·en 양쪽 12개 라벨을 단언할 것. 1순위가 성립하지 않을 때만 고를 것.
