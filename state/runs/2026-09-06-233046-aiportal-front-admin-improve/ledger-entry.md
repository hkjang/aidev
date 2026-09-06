## 2026-09-06
- 선택: 목록의 날짜 값을 시간대·형식에 흔들리지 않게 표시 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `shared/format.ts`는 단위 테스트가 전혀 없었고 `formatDateTime`이 `new Date(String(value))` 하나로 모든 입력을 처리해, 자료실·공유 앱·정책·콘텐츠 접근·운영 현황 표의 날짜 칸이 두 가지로 어긋났다. (1) 날짜만 있는 `2026-08-25`는 UTC 자정으로 해석돼 KST에서는 없는 시각 "09:00"이 붙고 UTC-8 환경에서는 전날로 밀린다(node로 직접 재현 확인). (2) Java backend가 흔히 문자열로 내려주는 `yyyyMMdd`·`yyyyMMddHHmmss`와 epoch millis는 해석하지 못해 `20260825143005` 같은 원시 숫자가 그대로 표에 남는다. offset이 없는 값은 브라우저 구현에 맡기지 않고 지역 시각으로 직접 만들고, 날짜만 있는 값은 시각 없이 날짜로만 표시하며, `Z`·`+09:00` offset은 그대로 존중하고, 자릿수로 단위를 단정할 수 없는 숫자(10자리 등)나 달력에 없는 날짜(`2026-13-01`)는 조작하지 않고 원본을 남기도록 했다. 이 계약을 admin-v2 ARCHITECTURE.md에 문서화했다. 검증은 `npm run verify`(typecheck + vitest 83개 통과, 기존 70개 + 신규 13개)와 `npm run build`(build → runtime-config 검증 → offline 검사 → integrity manifest 19개) 전체 통과, 그리고 신규 테스트를 KST·UTC·America/Los_Angeles 세 시간대에서 각각 돌려 기대값이 시간대에 의존하지 않는지 확인했다. 커밋 ad2f001.
- 보류 아이디어:
  - `monitoringParser.toPod`이 kubectl 스타일 `restarts`(`"3 (5d ago)"`)에서 NaN을 표에 그대로 노출하는 문제 (가치 2 / 위험 1 / S)
  - CatalogView·ContentAccessView가 route query를 onMounted에서만 읽어 같은 경로의 query 변경(딥링크 재진입)에 반응하지 않는 문제 — 컴포넌트 테스트 환경(jsdom, @vue/test-utils) 선행 정비 필요 (가치 3 / 위험 3 / M)
  - AdvancedPolicyView가 `snapshot.extensions`를 v-model로 직접 변형해 저장 실패 시 화면과 서버 상태가 어긋나는 문제 (가치 2 / 위험 2 / S)
  - PaginationBar가 범위 밖 페이지(총건수 감소)에 머물면 "31–25 / 25"로 표시되고 이전 버튼이 `move()` 가드에 막혀 아무 동작도 하지 않는 문제 (가치 2 / 위험 2 / S)
  - 루트 앱의 `crypto-js` local tarball 의존성 제거로 clean install 복구 — 현재 `npm ci` 자체가 불가해 검증 비용 큼 (가치 4 / 위험 4 / M)
