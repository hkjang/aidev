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

## 구현 노트
- 직원 화면에 `CSV 내보내기` 를 더했다(커밋 fa4a85f, 프로덕션 2파일: `web/src/lib/employeeExport.ts` 신규, `EmployeesPage.tsx`). 재조회 없이 화면의 `items` 를 담고, 상태·좌석 라벨은 표와 CSV 가 같은 함수를 쓴다(표의 삼항 분기 제거).
- 과제서와 다르게 한 것: 열 이름을 표 머리글이 아니라 같은 화면의 `직원 가져오기` 파서(`internal/app/employees.go:170-177`)에 맞췄다 — 과제서의 "조직"은 파서가 모르는 이름이어서(아는 것은 조직명·조직코드·조직·부서 중 "조직명") 그대로 쓰면 내보낸 파일을 다시 올릴 때 조직이 조용히 버려진다. 같은 이유로 직급=title·직책=position 이다. 비평가는 이 판단을 먼저 봐 달라.
- 과제서의 '미확인'을 확인했다: Playwright 에서 blob URL + 분리된 `<a>` + 즉시 `revokeObjectURL` 다운로드의 **본문이 읽힌다**(`download.path()` → 364바이트, 10행). 그래서 `HistoryPage` 방식을 바꾸지 않았고 E2E 가 머리글과 "표의 행 수 == 파일 데이터 행 수"까지 단정한다.
- 확신 없는 곳: ① 재검증한 적 없는 계약 — 내보낸 파일을 실제로 `직원 가져오기`에 다시 올려 보지는 않았다(직원 데이터를 바꾸므로 E2E 범위 밖). 왕복은 파서 소스를 읽어 맞춘 것이고 실행으로 증명한 것은 아니다. ② 모르는 status 값을 "퇴직"으로 읽는 갈래는 표의 기존 동작을 그대로 옮긴 것이다 — 옳아서가 아니라 화면이 달라지지 않게 하려고 유지했다. ③ `setMessage("n명을 내보냈습니다.")` 는 성공 Alert 를 쓰는데, 브라우저가 다운로드를 막았을 때도 뜬다.
- 일부러 하지 않은 것: 500건 상한 개선(서버 COUNT 필요 — 차선 후보로 남김), `HistoryPage` 의 `<a>` 방식 통일(본문이 읽히므로 바꿀 이유가 없어졌다), guide PNG 재촬영·PDF(관례).
- 다음 역할이 조심할 것: `web/e2e/employee-export.spec.ts` 는 실서버+PostgreSQL+시드가 있어야 돈다(시드 10명 전원 좌석 배정이라는 전제를 두 번째 테스트가 쓴다 — 시드가 바뀌면 그 테스트가 먼저 깨진다). 전체 68건 중 tracking 3·mcp-oauth 4 는 이 환경에서 원래 실패하며(변경 전 이미지에서도 같은 두 spec 이 실패) 내 변경과 무관하다.
- [러너 21:54] brief accepted — 채택 — 과제서가 지목한 자리(머리글 112/121, 표의 삼항 391-397, HistoryPage.exportCsv 163-172, format.ts 헬퍼, 시드 10명 전원 좌석 �
- [러너 21:54] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 거절. 수리가 먼저 볼 파일은 `docs/USER_GUIDE.md:147`(과 재생성할 HTML)과 `web/src/lib/employeeExport.ts` 의 열 목록이다. 구현자가 '확신 없는 곳 ①' 로 남긴 왕복을 실제로 따라가 보니 깨진다: 내보낸 파일에 조직코드가 없어 `saveEmployee`(internal/app/employees.go:109-120)가 `external='import:<조직명>'` 으로 **같은 이름의 새 조직을 만들고** 직원을 그리로 옮긴다(organizations.name 에 UNIQUE 없음). 팀 색이 기본 파랑으로 바뀌고 좌석 지정 구역이 옛 조직을 가리켜 전부 '구역 불일치'가 된다. 조직명 칸을 비우면 소속이 지워져 안전한 길이 없다. 가장 싼 수리는 문서에서 왕복 문장을 빼는 것(프로덕션 코드 무변경), 제대로 고치려면 `/api/v1/organizations` 의 externalId 를 읽어 '조직코드' 열을 더해야 한다.
- 확인한 것: tsc 통과, 새 vitest 11개 통과, `build-docs.py USER_GUIDE` 재생성이 커밋된 HTML 과 동일(즉 `<ol start=14→15>` 는 생성기의 목록 카운터 특성이고 손질이 아니다), 파서 키·직급=title·직책=position 매핑은 employees.go 와 맞다, 표가 items 를 자르지 않으므로 E2E 의 행 수 단언은 성립한다.
- 못 본 것: E2E 실행(실서버·PostgreSQL 없음). 내보낸 파일을 실제로 올려 본 것도 아니라 위 결론은 saveEmployee·migrations.sql 을 읽은 추론이지만, 중복 조직이 생기는 경로는 코드상 분명하다.
- 차단 없음(security/legal): 새로 나가는 요청·새 수집이 없고 화면은 이미 Manager 가드 아래다. 남는 우려는 개인정보 일괄 반출에 감사 기록이 없다는 것(읽기 경로 전반의 기존 상태).
- 다음 회차가 알아야 할 것: USER_GUIDE 3.6 절 번호가 15..19 로 찍히는 생성기 결함이 남아 있다. 그리고 500명 상한 때문에 내보낸 파일을 '전체 명단'이라 부르면 안 된다.
- [러너 21:59] review rejected — 리뷰 거절: docs/USER_GUIDE.md:147 (와 docs/USER_GUIDE.html:180, web/src/lib/employeeExport.ts:13-19) 문서가 약속하는 왕복이 실제로는 조직 데이터를 망가뜨린다. 내보낸 �
- [러너 21:59] pr created — https://github.com/hkjang/seaton/pull/37
