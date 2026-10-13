- 과제: 짧은 금액 표기 krwShort 의 1억 경계 오표기 수정 (가치 3 / 위험 1 / 작업량 S)
- 왜: `apps/web/src/lib/format.ts` 의 `krwShort` 는 억 분기를 **반올림 전 원값**으로 판정하고 만 분기에서 `Math.round(n / 10_000)` 를 하므로, 99,995,000~99,999,999원이 "1억" 이 아니라 **"10,000만"** 으로 나온다(음수도 대칭으로 "-10,000만"). 짧게 읽히도록 만든 표기가 경계에서 다섯 자리 만 단위를 내보내 자릿수를 세어야 읽히고, 영업 파이프라인 요약(`apps/web/src/routes/w/sales/Opportunities.tsx:191` 의 `{p.count}건 · {krwShort(p.amount)}`)은 단계 합계라 1억 근처를 실제로 지나간다.
- 수용 기준:
  1) `krwShort(99_999_999)` 와 `krwShort(99_995_000)` 가 억 표기("1억")로 나오고, `krwShort(-99_999_999)` 도 대칭으로 나온다. 어떤 입력에서도 만 단위 숫자가 10,000 이상으로 찍히지 않는다.
  2) 기존 표기가 그대로다: `krwShort(120_000_000)`="1.2억", `krwShort(100_000_000)`="1억", `krwShort(34_000_000)`="3,400만", `krwShort(12_000)`="1만", `krwShort(9_999)`="9,999원", `krwShort(0)`="0원", `krwShort(null)`="0원", `krwShort(-34_000_000)`="-3,400만".
  3) 새 시험 파일이 (1)의 경계값에서 수정 전 실패하고 수정 후 통과한다 — 구현자는 수정 전 실패 출력을 기록에 남길 것. 소스 문자열 검사가 아니라 실제 `krwShort` 호출 결과로 판정한다.
- 건드릴 파일 (프로덕션 1 + 시험 1):
  - `apps/web/src/lib/format.ts:krwShort` — 억/만 분기 판정을 **반올림된 만 값** 기준으로 바꾼다. 권장 형태: `const man = Math.round(n / 10_000)` 를 먼저 구하고 `Math.abs(man) >= 10_000` 이면 `(man / 10_000).toFixed(man % 10_000 === 0 ? 0 : 1)` + `억`, 그 아래면 `man.toLocaleString('ko-KR')` + `만`, 만 미만은 지금처럼 `krw(n)`. (이 형태를 확인해 두었다: 123,400,000→"1.2억", 100,000,000→"1억", 99,999,999→"1억", 99,994,999→"9,999만" — 다른 형태로 해도 수용 기준만 지키면 된다.)
  - `apps/web/src/lib/format.test.ts` (신규) — 위 경계값·회귀값 시험. 같은 디렉터리에 시험 파일이 아직 없으므로 새로 만든다. vitest include 는 `apps/**/*.test.ts`, environment 는 `node` (`vitest.config.ts`) 이므로 DOM 없이 순수 함수만 불러 쓸 것.
- 검증 명령:
  - `npm ci --no-audit --fund=false` (이 워크트리에 node_modules 가 없으면 먼저)
  - `npm test -- apps/web/src/lib/format.test.ts` (수정 전 실패 → 수정 후 통과)
  - `npm run check` (ESLint + 세 workspace 타입 검사 + 전체 vitest)
  - `npm run build`
- 위험과 피할 것:
  - `krw`·`relative`·`date`·`shortDate`·`addDays` 는 건드리지 말 것. 같은 파일에 있지만 각각 다른 계약이고, `relative` 의 분/시/일 반올림 경계는 제품 의도 미확인이라 이번 범위가 아니다(보류 아이디어로 남겨 둔다).
  - `apps/web/src/routes/w/sales/Opportunities.tsx` 는 읽기만 하고 수정하지 말 것 — 호출부 변경 없이 표기만 고쳐야 한다.
  - auth/session, `db/migrations`, `.github/workflows`, 공유 DB·배포 스크립트는 접근하지 않는다.
  - 과거 교훈대로 **최종 출력까지** 바뀌는지 확인할 것: 중간 계산만 고치고 `toFixed`/`toLocaleString` 단계에서 효과가 삼켜지지 않는지 실제 반환 문자열로 본다.
  - `toLocaleString('ko-KR')` 결과가 CI 의 Node ICU 에 따라 달라질 수 있다 — 시험은 "3,400만" 처럼 쉼표를 기대하는데, 만약 로케일 데이터가 없는 Node 에서 깨지면 기대값을 `krwShort` 결과끼리 비교하는 대신 숫자 자리수 패턴으로 완화하지 말고, 먼저 `node -e "console.log((34000).toLocaleString('ko-KR'))"` 로 환경을 확인하고 기록할 것. (이 워크트리의 Node 에서 실제 확인은 하지 않았다 — 미확인)
- 차선 후보: 달력 날짜 검증(v.day)의 윤년·월말 회귀 시험 보강 — `packages/core/src/util/validate.ts` 의 `v.day` 는 `new Date(\`${value}T00:00:00Z\`)` ISO 왕복과 2000~2100 연도 범위를 검사하는데 `packages/core/src/__tests__/domain.test.ts` 에 해당 경계 시험이 없다. `parseOne(v.day(...), ...)` 로 2024-02-29 정상 / 2023-02-29·2026-04-31 거절 / 2026-1-1 형식 거절 / 1999-12-31·2101-01-01 범위 거절 계약을 고정한다(프로덕션 변경 없음).
