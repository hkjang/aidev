# 회차 노트 2026-10-14-090829-biz.irumx.app-improve — biz.irumx.app
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 09:08] base pinned — main@9b864f8
- [러너 09:08] autonomy release — 

## 정찰 노트
- 고른 이유: v.day 시험 보강(차선)은 프로덕션 결함이 아니라 계약 고정뿐이고 직전 두 회차가 이미 시험 추가였다. validate.ts 를 읽는 중 v.money/v.int/v.decimal 이 문자열을 Number() 로 바로 변환하는 것을 보았고, 이 검증기들이 finance.ts 의 공급가·수금액·차변/대변(불변 원장 쓰기 경로)에 쓰이므로 가치가 더 높다고 판단했다. 수정은 허용 문자열을 좁히는 방향 한 가지뿐이라 운영자 지시(파서는 좁히는 쪽으로만)와도 맞는다.
- 확신 없는 곳: 이 워크트리에서 node 실행이 허용되지 않아 `Number('0x10') === 16` 을 직접 돌려 확인하지 못했다 — ECMAScript 변환 규칙에 따른 판단이다. 과제서 수용 기준 (1) 이 사전 실패로 재현을 먼저 요구하게 해 뒀다.
- 또 하나 추측: 금액 문자열이 실제로 어느 클라이언트에서 오는지(Mantine NumberInput 이 숫자를 보내는지 문자열인지)와 AI/MCP 도구 인자 경로는 확인하지 않았다. 호출부는 읽기만 하라고 적었다.
- 구현자가 조심할 것: v.int 에 쉼표 제거를 새로 넣지 말 것(money/decimal 만 제거한다 — 넓히면 경로별로 같은 값이 달라진다). v.day·v.string·v.text 는 건드리지 말고, 파일은 validate.ts + domain.test.ts 두 개로 끝낼 것.
- 프로필은 0일 전 것이고 이번에 읽은 validate.ts·text.ts·time.ts·시험 배치와 어긋나지 않아 다시 쓰지 않았다.
- [러너 09:12] scout done — 금액·정수·소수 입력이 '0x10'·'1e3' 같은 비십진 리터럴을 조용히 받는 것 차단 (가치 4 / 위험 2 / 작업량 S)

## 구현 노트
- 바꾼 것: `packages/core/src/util/validate.ts` 에 `DECIMAL_RE` + `asNumber(value, stripSeparators)` 를 추가하고 `v.money`/`v.int`/`v.decimal` 의 문자열→수 변환 한 줄씩만 교체했다. 이유: `Number('0x10')===16`·`Number('1e3')===1000` 이라 공급가·수금액·차변/대변에 비십진 리터럴이 검증 오류 없이 금액으로 들어갔다. 리터럴이 아니면 **원래 문자열을 반환**해 기존 분기가 기존 한국어 문구로 거절한다 — 새 메시지·새 분기 0개.
- 확신 없는 곳·검증 못 한 것: (a) `npm run e2e`/`ui`/`screens`/`a11y` 는 PostgreSQL·fake OAuth·Playwright 가 필요해 돌리지 않았다 — 실제 라우트를 HTTP 로 통과시킨 증거는 없고, `parse`/`parseOne` 진입점까지만 증명했다. (b) 화면(`apps/web`)이 금액 입력을 보내기 전에 자체 전처리를 하는지는 읽어 확인하지 않았다 — 화면에서 정상 입력이 막히는 회귀는 없다고 믿지만 브라우저로 보지 않았다. (c) `'+1000'`·`'.5'` 같은 부호·앞점 형식은 전후 모두 통과하도록 정규식에 `[+-]?`/`\.\d+` 를 넣었는데, 이 형식이 실제로 들어오는 경로가 있는지는 미확인(기존 동작 유지가 목적).
- 일부러 하지 않은 것: `v.int` 에 쉼표 제거를 넣지 않았다(money 와 달리 지금도 `'1,000'` 을 거절 — 같은 값이 경로마다 다르게 읽히지 않도록 넓히지 않음). 같은 파일의 `v.day`·`v.string`·`v.text`·`v.instant`·`v.bool` 은 손대지 않았다(차선 후보·신규 아이디어로 `ideas.json` 에 적음). 라우트·마이그레이션은 읽기만 했다.
- 다음 역할이 조심할 것: 새 시험은 DB·브라우저 없이 `npm test -- packages/core/src/__tests__/domain.test.ts` 로 돈다(59개). 가드 유효성은 `validate.ts:65` 의 `!DECIMAL_RE.test(cleaned)` 를 지우면 15개가 다시 실패하는 것으로 확인했다. `decimal` 의 동점 반올림은 `Math.round` 가 +∞ 로 깨뜨려 음수에서 비대칭인데(이번 범위 밖) 시험 기대값은 동점을 피한 `-12.346`→`-12.35` 로 적었다.
- [러너 09:17] brief accepted — 채택 — 지정된 수용 기준 (1) 의 사전 실패로 결함이 실제 `parse` 반환값/예외로 재현됐고, 권장 형태(`DECIMAL_RE` + `asNumber`)�
- [러너 09:17] verify passed — 검증 6개 통과 (auto)

## 비평 노트
- 확인함(실행): main 의 validate.ts 로 되돌려 새 시험 15개가 실제로 실패하는 것을 직접 재현했고(0x10→16·1e3→1000 통과), 복원 후 `npm run check` 전체 통과(eslint+tsc×3+vitest 138/8). 시험은 바뀐 경로를 진짜 지난다. 호출부(v.money/v.int/v.decimal 전부)와 웹 입력(MoneyInput→amt 가 number 반환)도 읽어 회귀 없음 확인. 판정: approve / risk low / blocking 없음.
- 못 봄: e2e·ui·screens·a11y(PostgreSQL·fake OAuth·Playwright 필요) — 실제 HTTP 라우트 통과 증거는 없고 parse/parseOne 진입점까지만 실행으로 확인.
- 릴리즈 노트에 적을 것: 커밋 메시지에 없는 행동 변화 1건 — 예전엔 money/decimal 에 ',' 만 오면 쉼표 제거 후 Number('')===0 이라 **0원으로 조용히 통과**했는데 이제 거절된다(같은 결함을 더 막는 방향이라 승인에 영향 없음).
- 다음 회차 후보: `apps/web/src/routes/w/finance/parts.tsx:29` 의 `toAmount` 가 여전히 맨 `Number()` 라 '0x10'→16 으로 읽는다. 공격 경로는 없으나(웹은 숫자를 보냄) 같은 결함의 사본이다 — 관례대로 통합하지 말고 그 자리에서 좁힐 것.
- 서버 측 다른 Number() 경로는 안전함을 확인: sales.ts 의 vat_rate 는 `v.oneOf` 허용목록, w.ts·finance.ts 의 limit/offset/min 은 Math.min/Math.max 로 범위 고정.
- [러너 09:20] review approved — 리뷰 승인 (risk=low)
- [러너 09:20] pr created — https://github.com/hkjang/biz.irumx.app/pull/4
- [러너 09:30] ci passed — 검사 1개 모두 success
- [러너 09:30] merge done — 04601ff
- [러너 09:37] release published — 0.1.4
