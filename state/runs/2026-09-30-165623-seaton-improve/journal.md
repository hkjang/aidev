# 회차 노트 2026-09-30-165623-seaton-improve — seaton
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 16:56] base pinned — main@fa0dba2
- [러너 16:56] autonomy release — 

## 정찰 노트
- 왜 이것인가: 보류 후보 중 유일하게 **관측 가능한 결함**이면서 프로덕션 1파일로 끝난다. `EmployeesPage.tsx:60-73` 의 `load` 에 순번 검사가 없고 `HistoryPage.tsx:73-101` 에는 있다 — 같은 저장소 안의 정답을 옮기는 일이다. rows.Err(3/2/M)은 DB 오류 하네스가 없어 또 탈락, total·더 보기(2/2/M)는 501명 시드가 필요, E2E README(2/1/S)는 문서만이라 차선으로 내렸다. guide-shots 는 릴리즈 경로라 rejected 로 굳혔다.
- 추측으로 적은 것: ① `page.route` 1500ms 지연이 응답 역전을 안정적으로 만드는지 실제로 돌려보지 않았다(값은 조정 가능) ② `route.abort("failed")` 때 `api` 헬퍼가 내는 문구를 확인하지 않았다 — 그래서 배너를 문구가 아니라 `getByRole("alert")` 존부로 단정하라고 적었다 ③ 이번 회차는 `npm ci` 를 돌리지 않았다(node_modules 없음), Go 테스트도 미실행.
- 구현자가 조심할 것: 순번은 `load` **안에서만** 올릴 것(호출부 두 곳 `useEffect:74`·`applyFilters:86` 에 복제하면 이 저장소의 반복 실패 패턴이 된다). `upload` 의 `await load()` 기본 인수를 깨지 말 것. `web/package.json` 에 jsdom·testing-library 가 없으니 컴포넌트 테스트용 의존성을 새로 넣지 말고, 붉은-먼저 증거는 변경 전 이미지(`fa0dba2`) E2E 로 낼 것. `HistoryPage` 통합·lib 헬퍼 추출은 다음 회차로 남겼다.
- 프로필: 1일 전 것이 지금 코드와 맞아(특히 "EmployeesPage load 에 순번 보호 없음" 기재) 새로 쓰지 않았다.
- [러너 17:00] scout done — 직원 목록 조회의 늦은 응답이 최신 필터 결과를 덮지 않게 하고, 재조회 시 낡은 오류 배너를 지우기 (가�
- [러너 17:08] brief unstated — 구현자가 과제서 판정을 적지 않음
- [러너 17:08] improve no-change — 커밋 없음
