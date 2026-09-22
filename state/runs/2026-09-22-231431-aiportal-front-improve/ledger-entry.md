## 2026-09-22
- 선택: README·docs 의 존재하지 않는 빌드/lint 명령과 끊긴 문서 링크 정정 (가치 2 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: 과제서의 기본 실행 대상(globalLoading 참조 카운트)은 과제서가 스스로 정한 중단 조건에 걸려 차선으로 전환했다 — 31개 호출 파일을 전수 감사한 결과 `src/views/ChatStorage/ChatStorageDetail.vue:886-902 openLawViewer` 가 `startLoading()` 만 하고 어떤 경로에서도 `stopLoading()` 을 부르지 않으며, `src/views/Support/Ocr/SupportOcr.vue:203-223` 은 `stopLoading()` 이 `setTimeout` 안에만 있어 catch 경로에서 누락된다. 이 상태로 카운터를 넣으면 카운터가 0 으로 돌아오지 못해 스피너가 영구히 안 꺼지는, 현재 조기 종료보다 나쁜 회귀가 난다. 대신 차선 후보를 구현해 README 문제 해결절·docs/06 의 `npm run build`/`npm run lint`, docs/01·docs/06 의 `npm run build:prod` 를 실제 스크립트(`build:dev`, `npm test`, 모드별 `build:core`/`build:ofc`)로 바꾸고, docs/09 문서 표의 끊긴 링크 6건과 README 의 docs/07 경로 오타를 정리했다. 검증: package.json scripts 와 실제 파일 존재를 대조하는 python3 스크립트로 수정 전 끊긴 링크 5건·없는 스크립트 6건을 확인하고 수정 후 끊긴 링크 0건(남은 `build:prod` 1건은 docs/06:424 의 "GitLab CI 예제" 가상 블록으로 의도적 제외), `npm test` 19파일 424테스트 통과, `npm run build:dev` 통과 후 `dist/` 삭제.
- 보류 아이디어: [수정 과제] 릴리즈 버전 결정 입력 복구: pending — release-prompt.md:21,22 는 없는 증가 패턴을 요구하고 :24 의 skipped 는 package.json:3 의 "0.0.0" 때문에 불가한 교착. 진입 조건(승인된 증가 단위·태그 형식·커밋 양식·노트 위치) 미충족이라 버전 파일·태그·CHANGELOG 미변경.
  - 릴리즈 절차 교착을 사람에게 에스컬레이션: pending — 수정 지점이 외부 aidev 절차라 저장소 안에 수단이 없다.
  - globalLoading 호출부 누수 2건 선행 수정(ChatStorageDetail.vue:887, SupportOcr.vue:204): pending — 신규. 참조 카운트보다 먼저 고쳐야 하며, .vue 는 현재 vitest 설정(SFC 플러그인 없음)으로 테스트 불가라 증명 수단 확보가 선행 과제다.
  - globalLoading 병렬 요청 참조 카운트: pending — 위 누수 2건이 남아 있는 한 넣으면 안 된다(스피너 영구 유지).
- 과제서: 차선 — 과제서가 명시한 중단 조건("`finally` 없이 start 하는 호출부를 하나라도 발견하면 범위를 줄여 보고하라")에 해당하는 누수 2건을 실물로 확인해 지정된 차선 후보를 구현했다.
