# 회차 노트 2026-09-27-213157-seaton-improve — seaton
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 21:31] base pinned — main@d1fb490
- [러너 21:31] autonomy release — 

## 정찰 노트
- 직원 화면 CSV 내보내기를 골랐다: 프로덕션 2파일, 서버 무관, 이미 있는 `format.ts`(toCSV/safeFileName/localDateStamp)와 `HistoryPage.exportCsv` 선례를 그대로 쓴다. 좌석맵 URL 확장(필터 4종)은 직렬화 규칙을 새로 정해야 해서 M, 목록 핸들러 rows.Err() 는 파일 10개 + DB 테스트 하네스 부재로 또 제쳤다. 직원 500건 상한은 `EmployeesPage.tsx:423` 에 이미 안내가 있어 가치를 2로 낮추고 차선으로 돌렸다.
- 추측으로 적은 것(과제서에 미확인으로 표시): Playwright 에서 blob URL + 분리된 `<a>` 클릭 + 즉시 `revokeObjectURL` 다운로드의 **본문**을 읽을 수 있는지. `web/e2e` 전체에 `waitForEvent("download")` 가 없어 선례가 없다. 내용 단정은 vitest 로 돌리고 E2E 는 발생+파일명까지만 잡는 길을 함께 적어 뒀다.
- 확인한 것: 라우트는 `/admin/employees`(App.tsx:71), 시드 직원 10명은 전원 active·전원 좌석 배정(seed.mjs:19-30,152-161)이라 '미배정' 행은 E2E 로 만들 수 없다 — 대신 배정상태=미배정 필터로 빈 목록/disabled 를 증명하게 했다.
- 구현자가 조심할 것: 상태 라벨(재직/휴직/퇴직)을 CSV 쪽에서 새로 짓지 말 것 — 이 저장소가 반복해 깨진 자리다. 표의 `-` 자리 표시를 CSV 에 그대로 넣으면 `csvCell` 의 수식 방어가 `'-` 로 바꾼다. USER_GUIDE 147행 캡션이 머리글 단추를 나열하므로 문장도 함께 고치고 PNG 재촬영은 하지 말 것.
- [러너 21:38] scout done — 직원 화면에서 조회 결과를 CSV 로 내보내기 (가치 3 / 위험 1 / 작업량 S)
