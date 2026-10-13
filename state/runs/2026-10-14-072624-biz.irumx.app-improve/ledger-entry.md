## 2026-10-14
- 선택: 짧은 금액 표기 krwShort 의 1억 경계 오표기 수정 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `apps/web/src/lib/format.ts` 의 `krwShort` 가 억/만 분기를 반올림 전 원값으로 판정해 99,995,000~99,999,999원이 "10,000만" 으로 나오던 것을, 반올림된 만 값(`man`) 기준 판정으로 바꿨다. 음수는 `Math.round` 가 동점을 +∞ 로 깨뜨려 비대칭(+99,995,000→"1억" vs -99,995,000→"-9,999만")이 남으므로 크기를 반올림하고 부호를 다시 붙이는 형태로 구현했다. 신규 시험 19개가 수정 전 7개 실패 → 수정 후 전원 통과하고, 되돌리면 다시 7개 실패함을 확인했으며 `npm run check`(ESLint + 세 workspace tsc + 전체 82개 시험)와 `npm run build` 가 통과했다. 호출부(Opportunities.tsx)는 읽기만 했고 프로덕션 변경은 1파일이다.
- 실패 재현: `AssertionError: expected '10,000만' to be '1억' // Object.is equality`
  `AssertionError: expected '-10,000만' to be '-1억' // Object.is equality` / `Tests  7 failed | 12 passed (19)`
- 보류 아이디어: 달력 날짜 검증(v.day)의 윤년·월말 회귀 시험 보강 (3/1/S, 차선 — 프로덕션 변경 없이 계약만 고정, 다음 회차 유력); 원장 CSV 수식 중립화의 API 회귀 시험 (3/2/M — csvCell 은 비공개라 DB·원장 fixture e2e 가 필요해 비용이 큼); 클립보드 복사 폴백 예외 시 textarea 정리 (2/1/S — vitest environment 가 node 라 DOM 시험 환경이 없음); 상대 시간 표기 relative 의 분·시·일 경계와 미래 시각 처리 (2/1/S — 제품의 반올림 의도 미확인, 이번에 같은 파일이지만 일부러 건드리지 않음); 프로젝트 손익 조회의 from/to 역순 검증 누락 (2/2/S — period() 로 교체하면 기본값 의미가 달라져 API e2e 필요).
- 과제서: 채택 — 지정된 경계값에서 결함을 실제 `krwShort` 반환값으로 재현했고 권장 형태대로 고쳤다. 다만 권장 형태 그대로는 음수 동점에서 비대칭이 남아 크기 반올림 + 부호 재부착으로 조정했다(수용 기준은 모두 충족).
