- 과제: 이름 없는 공간의 대체 이름이 한국어 UI 에서도 영어 `My Space` 로 뜨고 내려받기 파일 이름에까지 들어간다 (가치 3 / 위험 1 / 작업량 S)
- 왜: `web/src/pages/CanvasPage.tsx:2130` 이 `spaces.find((s) => s.id === activeSpace)?.name || 'My Space'` 로 대체 이름을 **영어 리터럴**로 못 박아 두었는데, 이 값은 캔버스 머리글(`:2276`)과 내려받기 파일 이름 네 곳(`:1966` `umm-${activeName}.md`, `:1990` 차례, `:2024` `.png`, `:2057` `.pdf`), 그리고 자식에게 넘기는 `spaceName={activeName}`(`:3524`)에 그대로 쓰입니다. `web/src/i18n/translate.ts` 가 선언한 계약은 "번역 키는 한국어 원문"이고 사전에는 이미 `web/src/i18n/en.ts:34` `'내 공간': 'My space'` 가 있는데, 이 자리만 그 계약 밖에 있어 한국어 사용자가 영어 이름을 보고 영어 파일 이름을 받습니다. 덧붙여 `?.name` 이 공백만 있는 이름(`'   '`)을 truthy 로 통과시켜 머리글이 비고 파일 이름이 `umm-   .md` 가 됩니다(서버가 공백 이름을 허용하는지는 **미확인** — 아래 수용 기준은 함수 계약으로만 단언하면 됩니다).
- 수용 기준:
  1) 이름 없는(또는 목록에 없는) 활성 공간의 표시 이름이 한국어 로케일에서 `내 공간`, 영어 로케일에서 `My space` 가 된다 — 같은 값이 머리글과 내려받기 파일 이름에 함께 쓰이므로 두 쓰임이 한 값에서 나와야 한다.
  2) 사용자가 지은 공간 이름은 **번역되지 않는다** — 예: 사전에 키로 존재하는 `생각 공간` 이라는 이름의 공간은 영어 로케일에서도 `생각 공간` 으로 남는다(대체 이름만 번역 대상).
  3) 테스트가 증명할 것: (a) 이름이 있으면 그 이름, 이름이 비었거나 공백뿐이면 대체 이름을 고르는 선택 로직, (b) 활성 id 가 `spaces` 에 없을 때도 대체 이름, (c) 실제 `translate`/실제 `en` 사전으로 `translate('내 공간')` 이 ko 에서 `내 공간`, en 에서 `My space` 인 것. 손으로 만든 사전이나 가짜 `t` 를 쓰지 말고 `web/src/i18n/translate.ts` 의 실제 `setLocale`/`translate` 와 실제 `en.ts` 를 쓸 것(`web/src/components/AppearanceMenu.test.tsx` 가 `setLocale` 를 그렇게 씁니다).
- 건드릴 파일:
  - `web/src/lib/space-name.ts` (신규) — 순수 함수 하나. 제안 계약: `spaceDisplayName(spaces, activeSpace, fallback)` → 활성 공간의 `name` 을 `trim()` 해서 비어 있지 않으면 그것, 아니면 `fallback`. **`t`/`translate` 를 이 함수 안에서 부르지 말 것** — 대체 이름을 인자로 받아야 사용자 이름이 사전 키와 겹쳐도 번역되지 않습니다(수용 기준 2). 타입은 실제 타입을 쓸 것 — `web/src/api.ts:34` 의 `export interface Space` 이고 `CanvasPage.tsx:257` 이 `useState<Space[]>([])` 로 들고 있습니다(확인). 테스트도 이 `Space` 를 import 해서 만들 것(손으로 만든 형태 유사 객체 금지).
  - `web/src/pages/CanvasPage.tsx:2130` — `const activeName = spaceDisplayName(spaces, activeSpace, t('내 공간'));` 로 교체. 이 줄에 `t` 가 이미 스코프에 있습니다(바로 아래 `:2135` 가 `t('생각 불러오는 중')`). 다른 줄(1966·1990·2024·2057·2276·3524)은 `activeName` 을 그대로 쓰므로 **손대지 마세요**.
  - `web/src/lib/space-name.test.ts` (신규) — 수용 기준 3.
  - `web/src/i18n/en.ts` — **건드리지 마세요.** `'내 공간'`(:34) 은 이미 있고, `'My Space': 'My Space'`(:14) 는 `AppLayout.tsx:61` 이 아직 쓰므로 지우면 그쪽이 깨집니다.
- 검증 명령 (이 저장소에서 실제로 도는 것):
  - 먼저 설치: `npm ci --prefix web` — 이 체크아웃은 `web/node_modules` 없이 시작하며, 설치 없이는 `npm test` 가 `run-on-supported-node: vitest is not installed …` 로 exit 1 입니다(2026-10-05 회차에서 확인된 동작).
  - `npm --prefix web test` — 기준선은 20파일 / 218시험 통과. 새 시험은 여기에 더해져야 합니다.
  - `npm --prefix web run typecheck`, `npm --prefix web run lint`, `node web/scripts/check-i18n.mjs`(1060키), `npm --prefix web run build`
  - Go 쪽은 이번에 0줄이므로 돌릴 필요 없습니다(돌렸다면 결과를 정직하게 적되, `POSTGRES_DSN` 없이는 DB 통합이 SKIP 입니다).
  - 수정 전 실패를 먼저 보일 것: 새 시험을 넣고 `CanvasPage.tsx` 를 아직 안 고친 상태에서는 선택 로직 시험이 없으니, **프로덕션 파일만 되돌려** 같은 실패가 재현되는지(그리고 복구하면 통과하는지) 확인하세요.
- 위험과 피할 것:
  - **`web/src/components/AppLayout.tsx:61` 의 `label: 'My Space'` 는 이번에 건드리지 마세요.** 같은 영어 리터럴이지만 `:273` 이 `t(label).split(' ')[0]` 로 모바일 탭에서 **첫 단어만** 잘라 쓰므로, 키를 `msg('내 공간')` 로 바꾸면 모바일 탭이 `내` 한 글자가 됩니다. 별도 과제입니다(차선 후보 아님 — 모바일 라벨 설계 결정이 먼저 필요).
  - `activeName` 이 들어가는 `anchor.download` 세 곳(1966·1990·2024)은 PDF(:2057)와 달리 파일 이름 새니타이즈를 하지 않습니다. **이번 과제에서 고치지 마세요** — 네 곳의 새니타이즈 계약 차이는 이 저장소의 기존 보류 아이디어이고, 운영자 규칙 3번(계약이 다른 파서·경로를 무리하게 통합하지 말 것)에 걸립니다.
  - 공백뿐인 이름을 서버가 실제로 허용하는지는 **미확인**입니다. 커밋 메시지나 보고에 "서버가 공백 이름을 허용한다"고 쓰지 말고, 함수 계약(`trim()` 후 비면 대체)으로만 설명하세요.
  - `internal/auth/`·`migrations/`·`.github/workflows/`·`Dockerfile` 은 이번 과제와 무관합니다. 버전(`VERSION`, `web/package.json`)은 올리지 마세요(개선 회차 관례).
  - `CanvasPage.tsx` 는 3500줄이 넘습니다. 2130 한 줄과 import 한 줄만 바꾸고 주변 상태 로직·`listView`·rewind 는 건드리지 마세요.
- 차선 후보: `web/src/lib/edge-vocabulary.ts` 의 `relationLabel`/`originLabel` 이 사전에 없는 서버 값을 받으면 `translate(relation)` 로 **원시 식별자**(`supports` 같은 영문 키)를 그대로 화면에 내보냅니다 — 파일에 적힌 주석("서버의 원시 식별자를 찍던 것을 고쳤다")과 어긋나는 잔여 경로이고, 이 모듈은 테스트 파일이 **없습니다**(`web/src/lib/` 에서 유일). 실제 `EdgeRelation`/`EdgeOrigin` 타입과 실제 `translate` 로 알려진 값 6+6개의 라벨을 못 박고, 알 수 없는 값이 사람이 읽을 수 있는 대체 라벨이 되게 하는 S 과제. (가치 2 / 위험 1 / S)
