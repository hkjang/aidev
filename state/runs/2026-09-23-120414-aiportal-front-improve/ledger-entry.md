## 2026-09-23
- 선택: 전역 로딩 스피너 계약과 법률 문서 링크 클릭 경로 테스트 고정 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: 과제서 B(ChatStorageDetail.vue:887 고아 startLoading() 제거)는 근거가 지금 코드와 맞지 않아 구현하지 않았다 — `openLawViewer` 의 유일한 호출부 452행이 `handleContentClick` 의 try 안에 있고 finally(476-478)가 `stopLoading()` 을 부르므로 887행은 누수가 아니라 이미 true 인 상태를 다시 켜는 중복 호출이다(과제서는 호출부를 확인하지 않았다). 증명: 공유 모드(chat-share, 네트워크 없이 route.params.data 로 메시지 복원)로 실제 컴포넌트를 마운트해 법률 링크를 실제 DOM 클릭으로 통과시킨 뒤, ① finally 의 stopLoading() 만 제거하면 단정이 실패하고(테스트가 누수를 실제로 잡는다) ② 887행만 제거하면 2 테스트 모두 그대로 통과함(관측 가능한 동작 변화 0)을 각각 확인했다. 운영자 지시 "출력·동작이 바뀌지 않는 수정은 넣지 말 것" 에 따라 소스는 건드리지 않고 차선 후보인 계약 spec 을 구현했다. 검증: `npm ci` → `npm test` 기준선 21파일 444테스트 → 23파일 452테스트 통과, globalLoading 구현 변이 2건(문구 초기화 제거 / singleton 해제)으로 새 spec 이 각각 1건·2건 실패함을 확인해 무딘 테스트가 아님을 증명, `npm run build:dev` 통과 후 `dist/` 삭제.
- 보류 아이디어: [수정 과제] 릴리즈 버전 결정 입력 복구 — pending(9회째). HEAD 595c0c5 에서 재확인: `.github` 없음, git tag 0개, package.json:3 = 0.0.0/private:true. 진입 조건 1·2(승인된 증가 단위·태그 형식·커밋 양식·노트 위치)가 출처와 함께 인계되지 않아 버전 파일·태그·CHANGELOG·원격 전송을 일체 하지 않았고 판정을 skipped/released 로 낮추지 않았다.
  - 릴리즈 절차 교착을 사람에게 에스컬레이션 — pending; 수정 지점이 외부 aidev 절차라 저장소 안에 합법적 수단이 없다.
  - SupportOcr.vue:205 의 ocrParse 를 await 하지 않아 try/catch 가 비동기 실패를 못 잡음 — pending; stopLoading 누락 건과 묶어 함께 고쳐야 무효 변경이 되지 않는다.
  - globalLoading 참조 카운트 — pending; SupportOcr 누수가 남아 있고 loading.vue 의 60초 자동 해제가 카운터를 되돌리지 않아 선행 조건 미충족. 이번 spec 은 참조 카운트를 계약으로 고정하지 않았다.
  - ChatStorageDetail.vue:887 중복 startLoading() 제거 — rejected(누수 아님으로 확정). 동작 변화가 없어 단독 수정 대상이 아니며, 이번 회차 spec 이 실제 동작을 고정했다.
- 과제서: 차선 — B 의 근거(고아 startLoading → 60초 오버레이)가 호출부 확인 누락으로 지금 코드와 맞지 않음을 마운트 테스트와 되돌림 실험으로 확정하고, 과제서가 지정한 차선 후보(globalLoading 계약 spec)를 구현했다.
