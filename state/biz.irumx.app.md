## 2026-10-07
- 선택: Markdown 내부 링크의 역슬래시 우회 차단 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: safeHref의 내부 경로 분기에서 원문 역슬래시를 거절해 외부 목적지로 해석되는 경로가 속성 없는 내부 anchor로 출력되지 않도록 했다. 실제 Markdown export를 createElement + renderToStaticMarkup으로 실행하는 13개 시험에서 슬래시·역슬래시 변형의 span 표시, 정상 내부 경로 및 http(s) 속성 보존, javascript:/data: 거절을 검증했다. npm ci 성공(ESLint 지원 종료 경고 1건), 지정 시험 13개·npm run check(린트/세 workspace 타입 검사/전체 54개 시험)·npm run build(웹/서버) 모두 통과했으며 커밋 c1f9a2b에 프로덕션 1파일+시험 1파일만 포함했다.
- 실패 재현: `Tests  4 failed | 9 passed (13)`
  `Received: "<div class="bz-prose"><p><a href="/\example.org/path">링크</a></p></div>"`
- 보류 아이디어:
  - 원장 CSV 수식 중립화와 인용 처리의 API 회귀 시험 (가치 4 / 위험 2 / 작업량 M): DB 및 fixture 준비가 필요해 기존 pending 유지.
  - 달력 날짜 검증의 윤년·월말 회귀 시험 보강 (가치 3 / 위험 1 / 작업량 S): 1순위 결함을 재현하여 차선은 보류.
  - 클립보드 복사 폴백 예외 시 textarea 정리 (가치 2 / 위험 1 / 작업량 S): 브라우저 예외 재현과 실패 반환 계약 확인을 다음 회차로 유지.
  - 상대 시간 표기의 분·시·일 경계 확인과 회귀 시험 (가치 2 / 위험 1 / 작업량 S): 제품의 반올림 의도 확인이 필요하여 보류.
- 과제서: 채택 — 지정된 실제 컴포넌트 출력 시험에서 결함을 재현했고 내부 경로 허용 조건만 좁혀 모든 수용 기준을 만족했다.
- 원인 검증: Node URL이 원문 /\example.org/path를 https://example.org/path로 해석함을 확인했다. 추가 조건만 임시 제거하자 동일 4개 시험이 다시 실패했고 복원 후 13개 통과; 시험 구성 오류나 다른 파서 변경에 따른 효과가 아님을 확인했다.
- 한계: 실제 브라우저 클릭·DB/OAuth e2e·원격 CI는 실행하지 않았다. 이번 검증 경계는 공유 컴포넌트의 최종 HTML이며, 저장 문서에 의도된 역슬래시 링크가 존재하는지는 미확인이다. percent-decode·URL 정규화·문법 확대·다른 파서 통합 및 의존성 변경은 하지 않았다.
- 스킬: 전용 Skill 도구가 없어 technology의 completion-verification, systematic-debugging, test-driven-development SKILL.md를 로컬 파일로 읽고 실패→최소 수정→통과→조건 제거 재실패→복원 통과 및 전체 검증 절차를 적용했다.

## 2026-10-14
- 선택: 짧은 금액 표기 krwShort 의 1억 경계 오표기 수정 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `apps/web/src/lib/format.ts` 의 `krwShort` 가 억/만 분기를 반올림 전 원값으로 판정해 99,995,000~99,999,999원이 "10,000만" 으로 나오던 것을, 반올림된 만 값(`man`) 기준 판정으로 바꿨다. 음수는 `Math.round` 가 동점을 +∞ 로 깨뜨려 비대칭(+99,995,000→"1억" vs -99,995,000→"-9,999만")이 남으므로 크기를 반올림하고 부호를 다시 붙이는 형태로 구현했다. 신규 시험 19개가 수정 전 7개 실패 → 수정 후 전원 통과하고, 되돌리면 다시 7개 실패함을 확인했으며 `npm run check`(ESLint + 세 workspace tsc + 전체 82개 시험)와 `npm run build` 가 통과했다. 호출부(Opportunities.tsx)는 읽기만 했고 프로덕션 변경은 1파일이다.
- 실패 재현: `AssertionError: expected '10,000만' to be '1억' // Object.is equality`
  `AssertionError: expected '-10,000만' to be '-1억' // Object.is equality` / `Tests  7 failed | 12 passed (19)`
- 보류 아이디어: 달력 날짜 검증(v.day)의 윤년·월말 회귀 시험 보강 (3/1/S, 차선 — 프로덕션 변경 없이 계약만 고정, 다음 회차 유력); 원장 CSV 수식 중립화의 API 회귀 시험 (3/2/M — csvCell 은 비공개라 DB·원장 fixture e2e 가 필요해 비용이 큼); 클립보드 복사 폴백 예외 시 textarea 정리 (2/1/S — vitest environment 가 node 라 DOM 시험 환경이 없음); 상대 시간 표기 relative 의 분·시·일 경계와 미래 시각 처리 (2/1/S — 제품의 반올림 의도 미확인, 이번에 같은 파일이지만 일부러 건드리지 않음); 프로젝트 손익 조회의 from/to 역순 검증 누락 (2/2/S — period() 로 교체하면 기본값 의미가 달라져 API e2e 필요).
- 과제서: 채택 — 지정된 경계값에서 결함을 실제 `krwShort` 반환값으로 재현했고 권장 형태대로 고쳤다. 다만 권장 형태 그대로는 음수 동점에서 비대칭이 남아 크기 반올림 + 부호 재부착으로 조정했다(수용 기준은 모두 충족).

- 릴리즈: 0.1.2 (2026-10-14, run 2026-10-14-072624-biz.irumx.app-improve)
## 2026-10-14
- 선택: 로그인 복귀 경로 가드 safeNext 의 회귀 시험과 두 사본 일치 고정 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: 로그인 후 돌아갈 경로를 거르는 `safeNext` 가 서버(`packages/core/src/routes/sso.ts:57`)와 화면(`apps/web/src/lib/nav.ts:2`)에 같은 한 줄로 중복돼 있는데 양쪽 모두 시험이 없었다. 외부 출처(`//evil.com`, 앞 공백 붙은 `  //evil.com`, `/\evil.com`, `/\/evil.com`, `https://evil.com/x`, `javascript:alert(1)`, `evil.com`, `relative/path`, `''`, `null`, 길이 321)와 통과 경로(`/`, 쿼리·프래그먼트 포함 경로, `  /w/acme  `→`/w/acme`, 길이 299 경계)를 15개 케이스 표로 못 박고, 같은 표를 두 시험 파일에 복제해 두 사본이 동일한 결과를 내는 것을 보였다(화면 사본은 `undefined` 입력 1건 추가 — 시그니처가 다름). 프로덕션 변경은 0파일이고, `npm run check`(ESLint + 세 workspace tsc + 전체 113개 시험)와 `npm run build` 가 통과했다.
- 실패 재현: 결함 수정이 아니라 시험 추가라 사전 실패가 없다 — 대신 가드 변이로 시험이 실제로 가드를 보는지 증명했다. 서버 사본에서 `!next.startsWith('//')` 제거 → `AssertionError: expected '//evil.com' to be '/' // Object.is equality` / `Tests  2 failed | 48 passed (50)` (domain.test.ts 만 실패, nav.test.ts 통과 — 사본 독립 확인). 화면 사본에서 같은 조건 제거 → `FAIL apps/web/src/lib/nav.test.ts > safeNext > maps "//evil.com" to "/"` / `Tests  2 failed | 48 passed (50)`. 화면 사본에서 `!next.startsWith('/\\')` 제거 → `FAIL ... maps "/\\evil.com" to "/"`, `maps "/\\/evil.com" to "/"` / `Tests  2 failed | 14 passed (16)`. 세 변이 모두 `git checkout --` 로 되돌린 뒤 전원 통과를 재확인했다.
- 보류 아이디어: 달력 날짜 검증 v.day 의 윤년·월말·범위 회귀 시험 보강 (3/1/S — 차선 후보 그대로, 프로덕션 변경 없이 계약만 고정, 다음 회차 유력); 원장 CSV 수식 중립화와 인용 처리의 API 회귀 시험 (3/2/M — csvCell 이 비공개라 DB·원장 fixture e2e 필요); 클립보드 복사 폴백 예외 시 textarea 정리 (2/1/S — vitest environment 가 node 라 DOM 증거를 만들 수 없음); 상대 시간 표기 relative 의 분·시·일 경계와 미래 시각 처리 (2/1/S — 제품의 반올림 의도 미확인); 프로젝트 손익 조회의 from/to 역순 검증 누락 (2/2/S — period() 로 바꾸면 기본값 의미가 달라져 API e2e 필요); krwShort 의 10억 이상 '10.0억' 표기 (2/1/S — 조·십억 표기 정책 미확인).
- 과제서: 채택 — 지정된 두 파일에 지정된 케이스 표를 그대로 넣고 변이 시험으로 가드를 보고 있음을 증명했다. 탭 문자 케이스 제외와 두 사본 통합 금지도 지시대로 지켰다.

- 릴리즈: 0.1.3 (2026-10-14, run 2026-10-14-081817-biz.irumx.app-improve)
## 2026-10-14
- 선택: 금액·정수·소수 입력이 '0x10'·'1e3' 같은 비십진 리터럴을 조용히 받는 것 차단 (가치 4 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: `packages/core/src/util/validate.ts` 의 `v.money`/`v.int`/`v.decimal` 이 문자열을 `Number()` 로 바로 바꿔 `'0x10'`→16, `'1e3'`→1000, `'0b101'`→5, `'0o17'`→15 이 금액·수량으로 조용히 통과하던 것을, 십진 리터럴 정규식(`DECIMAL_RE`)을 통과한 문자열만 변환하는 도움 함수 `asNumber` 로 좁혔다. 리터럴이 아니면 원래 문자열을 그대로 돌려줘 기존 `typeof n !== 'number'` 분기가 기존 한국어 문구(`… 원 단위 정수여야 합니다.` / `… 정수여야 합니다.` / `… 숫자여야 합니다.`)로 거절한다 — 새 메시지·새 분기 없음. 쉼표·공백 제거 규칙은 경로별로 그대로 유지했다(money·decimal 은 제거, int 는 trim 만). 라우트가 실제로 쓰는 진입점 `parse`/`parseOne` 과 실제 검증기 구성을 지나는 시험 24개가 수정 전 15개 실패 → 수정 후 전원 통과하고, `npm run check`(ESLint + 세 workspace tsc + 전체 138개 시험)와 `npm run build` 가 통과했다. 프로덕션 변경은 1파일(+시험 1파일)이다.
- 실패 재현: `FAIL packages/core/src/__tests__/domain.test.ts > 입력 검증: 금액·정수·소수 > money 가 "0x10" 를 거절한다` / `AssertionError: expected AssertionError: 0x10 이 금액으로 통과했다 { …(3) } to be an instance of AppError` / `Tests  16 failed | 43 passed (59)` (16건 중 1건은 decimal 동점 반올림에 대한 내 기대값 오류였고 `-12.345`→`-12.346` 으로 고쳤다 — 기존 동작은 손대지 않았다. 나머지 15건이 결함.)
- 보류 아이디어: 달력 날짜 검증 v.day 의 윤년·월말·범위 회귀 시험 보강 (3/1/S — 차선 후보 그대로, 프로덕션 변경 0파일, 다음 회차 유력); 원장 CSV 수식 중립화와 인용 처리의 API 회귀 시험 (3/2/M — csvCell 이 비공개라 DB·원장 fixture e2e 필요); v.string 은 UTF-16 단위·v.text 는 코드포인트로 길이를 세어 같은 '자' 가 필드마다 다름 (2/2/S — 어느 쪽이 제품의 '자' 인지 docs/requirements 확인이 먼저); 프로젝트 손익 조회의 from/to 역순 검증 누락 (2/2/S — period() 로 바꾸면 기본값 의미가 달라져 API e2e 필요); 클립보드 복사 폴백 예외 시 textarea 정리 (2/1/S — vitest environment 가 node 라 DOM 증거를 만들 수 없음).
- 과제서: 채택 — 지정된 수용 기준 (1) 의 사전 실패로 결함이 실제 `parse` 반환값/예외로 재현됐고, 권장 형태(`DECIMAL_RE` + `asNumber`)와 호출 규칙(money·decimal `true`, int `false`)을 그대로 따랐다.

