## 2026-09-22
- 선택: useAppList의 비배열 sidemenuAppList 캐시 가드 (가치 3 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: `commonStorage`의 `sidemenuAppList`가 객체/문자열이면 `|| []`가 truthy를 걸러내지 못해 `getFormattedAppList`의 `list.map`에서 `TypeError: list.map is not a function`이 났고, 같은 값을 읽는 형제 경로(`src/utils/appList.js:77`)만 `toAppArray`로 방어돼 두 경로의 동작이 갈려 있었다. `fetchTopApps`와 `useAppList.updateList` 두 진입 경로를 기존 `toAppArray` 재사용으로 바꾸고(새 헬퍼 없음), 비배열·null·미설정 입력과 정상 배열의 매핑/limit 회귀를 함께 고정하는 `tests/unit/useAppList.spec.js`(9케이스)를 추가했다. 검증: 수정 전 새 스펙 3개 실패(Red, 원인 로그 `useAppList.js:13`) → 수정 후 `npm test` 20파일 433테스트 통과(기준 19/424), 가드를 한 곳씩 되돌릴 때마다 각각 2개씩 실패함을 확인해 인과를 증명했고, `npm run build:dev`도 통과 후 `dist/`를 지웠다.
- 보류 아이디어:
  - [수정 과제] 릴리즈 버전 결정 입력 복구: pending — release-prompt.md:21,22(2·3항)이 '이전 증가 패턴'을 요구하는데 태그 0개·릴리즈 커밋 0개이고, :24(5항)의 `skipped`는 '버전 파일도 없음'을 요구하는데 `package.json:3`에 `"version": "0.0.0"`이 있어 released도 skipped도 불가능한 구조적 교착이다. 관례 신설은 5항·AGENTS.md·docs/RELEASE.md:3,83이 모두 금지하므로 저장소 안에 합법적 수단이 없다. 이번 회차에 버전 파일·태그·CHANGELOG를 건드리지 않았다.
  - 릴리즈 절차 교착 자체를 사람에게 에스컬레이션: pending — 수정 지점이 외부 aidev 절차이고 사람의 결정이 필요하다.
  - globalLoading 병렬 요청 참조 카운트: pending — 호출 짝 감사와 실제 병렬 요청 재현이 선행되어야 한다.
  - README 문제 해결 절의 존재하지 않는 `npm run build`: pending — 3순위 문서 정합성 수정으로 남긴다(실제 스크립트는 `build:dev`/`build:core` 등 모드별).
- 과제서: 차선 — 진입 조건 1)이 정한 대로 승인된 증가 단위·태그 형식·커밋 양식·노트 위치가 어디에도 출처와 함께 없어 1순위가 성립하지 않으므로, 과제서가 '이번 회차의 기본 실행 대상'으로 지정한 차선을 구현했다.
