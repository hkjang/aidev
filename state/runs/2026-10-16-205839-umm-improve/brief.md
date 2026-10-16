# 과제서 — run 2026-10-16-205839-umm-improve (umm)

베이스: `main@745dcf3`, `VERSION`=0.76.8. 이 정찰에서 직접 열어 확인한 것만 적었고, 확인하지 못한 것은 **미확인**이라고 표시했습니다.

- 과제: `edge-vocabulary.ts` 의 연결 어휘 12개와 `relationOptions` 를 실제 타입·실제 `translate`·실제 `en.ts` 로 못 박기 (가치 2 / 위험 1 / 작업량 S)

- 왜: `web/src/lib/` 의 모듈 8개 중 **`edge-vocabulary.ts` 하나만 인접 테스트가 없습니다**(`ls web/src/lib` 로 확인 — `browser-storage`·`download-name`·`handoff`·`markdown-import`·`optimistic-write`·`silent-sso`·`space-name` 은 전부 `.test.ts` 가 있고 `edge-vocabulary.ts` 만 없음). 그리고 이 모듈에는 지금 아무것도 잡아 주지 않는 드리프트 구멍이 하나 있습니다: `relationLabels`(`:8`)·`originLabels`(`:17`)는 `Record<EdgeRelation, string>`·`Record<EdgeOrigin, string>` 이라 타입에 멤버가 늘면 **typecheck 가 막아 주지만**, 캔버스 피커가 실제로 읽는 `relationOptions: EdgeRelation[]`(`:31`)은 그냥 배열이라 `EdgeRelation`(`web/src/api.ts:86`)에 일곱 번째 관계가 추가되어도 **아무 검사도 깨지지 않고 피커에서만 조용히 빠집니다**. 시험 파일 안에 손으로 적은 exhaustive `Record<EdgeRelation, …>` 기대값을 두면 그 순간부터 `npm run typecheck` 가 그 추가를 가리키게 되고(아래 2번), 동시에 11개 한국어 라벨의 **영어 문구**가 고정됩니다(`check-i18n.mjs` 는 키의 **존재**만 보고 문구는 보지 않습니다 — `web/scripts/check-i18n.mjs` 의 "Every key the app asks for must exist in each non-default dictionary" 주석으로 확인).

- 수용 기준:
  1) 새 시험이 **실제** `web/src/i18n/translate.ts` 의 `setLocale`/`translate`(대역·가짜 `t` 금지, 실제 `en.ts` 사전이 `translate.ts:32` 에서 자동 등록됨)로 ko·en 양쪽을 돌려, `relationLabel` 6개 + `originLabel` 6개가 각 로케일에서 기대 문자열을 낸다. en 기대값은 `web/src/i18n/en.ts:982-991` 에서 그대로 읽어 적을 것: `연결됨`→`Related`, `뒷받침함`→`Supports`, `상충함`→`Contradicts`, `구체화함`→`Refines`, `확장됨`→`Expands`, `이어짐`→`Follows`, `직접 연결`→`Drawn by hand`, `에이전트`→`Agent`, `Dream 확장`→`Dream development`, `자동 추천`→`Suggested`, 그리고 `가져오기`→`Import`(이 키는 같은 블록이 아니라 `en.ts:353` 에 있음).
  2) **컴파일 시점 드리프트 망**: 시험 파일에 `Record<EdgeRelation, string>` 과 `Record<EdgeOrigin, string>` 형태의 기대값 객체를 **리터럴로** 선언하고(`satisfies` 가 아니라 타입 애너테이션으로 — 누락 멤버를 오류로 만들기 위해), `relationOptions` 가 그 키 집합과 **같은 멤버**임을 단언한다. `api.ts` 의 두 유니온에 멤버를 하나 더하면 `npm --prefix web run typecheck` 가 **실패해야 한다**(`web/tsconfig.app.json` 의 `"include": ["src"]` 로 시험 파일도 typecheck 대상임을 확인했음). 구현자는 이것을 추측하지 말고 `api.ts:86` 에 임시 멤버를 하나 더해 typecheck 가 **실제로 빨개지는지** 찍고 되돌릴 것.
  3) `originLabels.dream = msg('Dream')` 은 en.ts 에 `Dream` 키가 **없는 것이 정상**이다(`grep "  Dream:" web/src/i18n/en.ts` → 0건). 한글이 없는 키라 `check-i18n` 대상이 아니고, `translate` 의 폴백(`translate.ts:67` 의 `?? source`)이 영어에서도 `Dream` 을 그대로 내는 것이 맞는 동작이다. 시험은 이것을 **버그로 적지 말고 의도된 계약으로 단언**할 것.

- 건드릴 파일:
  - `web/src/lib/edge-vocabulary.test.ts` — **신규, 이 과제의 유일한 파일**. `relationLabel`·`originLabel`·`relationOptions`·`isInferred` 를 `../lib/edge-vocabulary` 가 아니라 `./edge-vocabulary` 에서 import. 로케일은 모듈 전역 상태이므로(`translate.ts:34` 의 `activeLocale`) `afterEach(() => setLocale('ko'))` 로 반드시 되돌릴 것 — 안 되돌리면 같은 파일 뒤쪽 시험이 영어로 돌아 거짓 통과/실패가 난다.
  - 프로덕션 파일 **0개**. `edge-vocabulary.ts`·`en.ts`·`api.ts`·`translate.ts`·`BacklinkRow.tsx`·`CanvasPage.tsx` 는 **한 글자도 바꾸지 않는다**.

- 검증 명령 (이 저장소에서 실제로 도는 것, 순서대로):
  1) `npm ci --prefix web` — **필수 선행**. 이 정찰 시점 `web/node_modules` 는 없음(`ls -d web/node_modules` → No such file). 루트 `npm ci` 는 web 의존성을 설치하지 않는다.
  2) `npm --prefix web test --silent` — vitest. `web/vitest.config.ts` 의 `include: ['src/**/*.test.{ts,tsx}', 'scripts/**/*.test.mjs']`, `environment: 'jsdom'`, `globals: true`. 기준선 파일/시험 수는 이 정찰에서 **미측정**(설치가 없어 돌리지 않았음) — 구현자가 시험 추가 **전에** 한 번 돌려 기준선을 찍고, 추가 후 그 수만큼 늘었음을 보일 것.
  3) `npm --prefix web run typecheck`
  4) `npm --prefix web run lint` — `lint` 는 `oxlint src && prettier --check src` 이므로 새 파일이 `src/lib/` 안에 있어 **이 명령에 포함된다**(2026-10-09 회차처럼 별도 prettier 경로를 쓸 필요 없음). 기존 `src/` 파일의 oxlint 경고 약 50건은 남는 것이 정상이고, 새 파일을 가리키는 경고가 0건이어야 한다.
  5) `node web/scripts/check-i18n.mjs` — 1060키 통과 유지(이 변경으로 새 한국어 리터럴이 늘지 않아야 한다; 시험 파일은 `walk()` 가 `.test.tsx?` 를 제외하므로 애초에 검사 대상이 아님 — `check-i18n.mjs:20` 확인).
  6) `make test-web` — 게이트 8개(`Makefile:22-30`). `audit` 는 네트워크가 필요하고 커밋이 아니라 레지스트리가 판정을 바꾼다.
  - Go 파일 0개이므로 `make test-go`·`go test`·`docker build` 는 **돌리지 않는다**. DB·E2E 도 범위 밖.

- 위험과 피할 것:
  - **보호 경로 전부 회피**: `internal/auth`, `migrations/`, `.github/workflows/`, `Dockerfile`, `VERSION`, 락파일, `web/package.json`. 이 과제는 그 어느 것도 건드리지 않는다.
  - `relationLabel`/`originLabel` 의 `?? relation` 폴백을 **버그로 주장하지 말 것**. `migrations/010_memory_graph.sql` 의 CHECK 제약이 값을 제한하므로 서버에서 오는 값으로는 도달 불가다(이 정찰에서 마이그레이션 파일은 **재확인하지 않았음** — 과거 회차 기록에 근거). 폴백을 "없는 관계를 넣으면 id 가 그대로 나온다" 로 단언하고 싶다면 타입을 캐스팅해 호출하는 시험이 되는데, 그러면 2번의 exhaustive 망과 의미가 충돌한다. 폴백 시험은 **넣지 말 것**.
  - **`en.ts` 의 문구를 "더 나은 번역" 으로 고치려 하지 말 것.** 이번 과제는 지금 문구를 고정하는 일이다.
  - **여러 경로의 같은 파서/사전을 통합하려 하지 말 것**(운영자 규칙 3번). `relationLabels` 와 `en.ts` 가 같은 문자열을 각자 갖는 것은 i18n 설계(한국어 원문 키)의 결과이고 정상이다.
  - **소스 문자열 검사를 근거로 삼지 말 것**(운영자 규칙 5번). "en.ts 에 키가 있다" 를 `grep` 으로 증명하지 말고 `setLocale('en')` 뒤 `relationLabel('supports') === 'Supports'` 라는 **실행 결과**로 증명할 것.
  - 과거 교훈: 다운로드 파일 이름 통합(PR #167)은 머지 뒤 되돌려졌다. 같은 접근(여러 호출부를 한 모듈로 모으는 리팩터)을 이 과제에 끌어들이지 말 것.
  - 2026-10-06 `make test-go` 확장(fa91965)은 **지금 main 에 없다**(`Makefile:7-9` 는 여전히 두 줄). 완료됐다고 가정하지 말고, 그 과제를 이번에 다시 집지도 말 것(verify-failed 전력).

- 차선 후보: **`format` 스크립트에도 설치 가드 한 줄** (가치 2 / 위험 1 / S) — `web/package.json:20` 의 `"format": "prettier --write src"` 에는 `preformat` 이 없어 미설치 체크아웃에서 `sh: 1: prettier: not found` / exit 127 로 죽는다(이 정찰에서 **실행 확인은 하지 않았음**; `predev`/`prelint`/`pree2e` 등 다른 7개 훅은 실제로 존재함을 `package.json:10-27` 에서 읽어 확인). 고침은 `web/package.json` 에 `"preformat": "node scripts/require-installed.mjs prettier"` 한 줄 + `web/scripts/require-installed.test.mjs` 에 fixture 시험 추가. 단 `format` 은 **소스를 쓰는 명령**이므로 시험은 실제 체크아웃이 아니라 기존 fixture 방식(임시 manifest 복사)으로만 돌릴 것.
