# 회차 노트 2026-09-28-194213-seaton-improve — seaton
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 19:42] base pinned — main@7502f82
- [러너 19:42] autonomy release — 

## 정찰 노트
- 조직 필터를 골랐다: 서버가 이미 `organizationId` 를 받는데 화면만 안 보내는 확정된 공백이라 프로덕션 2파일로 닫히고, 시드가 3조직·10명이라 E2E 로 곧장 증명된다. `rows.Err()` 삼킴(3/2/M)은 파일 10개 안팎이라 6파일 규칙에, 500건 total 은 서버 COUNT+501명이 필요해 45분에 걸린다.
- 확인한 것: employees.go:62,69(서버가 org 를 거름), EmployeesPage.tsx:58-61(화면이 안 보냄), server.go:82(조직 GET 은 인증만), seed.mjs:14-30(개발6·영업2·인사2), UsersPage.tsx:260 ↔ admin.spec.ts:150(MUI Select 의 inputProps aria-label 이 getByRole 로 실제 잡힘).
- 추측으로 적은 것: 조직이 0개인 설치에서 Select 가 어떻게 보이는지는 확인하지 않았다(과제서에 미확인으로 표기). 코드는 실행하지 않았고 테스트도 돌리지 않았다.
- 구현자가 조심할 것: 기존 두 Select 에 aria-label 을 더할 때 **보이는 텍스트를 바꾸면** employee-export.spec.ts:52-55 의 `filter({hasText:"전체 배정상태"})` 가 멎는다. 그리고 주소 반영(?organizationId=)·total 은 범위 밖으로 명시했으니 끌어들이지 말 것.
- 프로필은 기준이 v1.4.8 로 낡아 있어 v1.4.11(7502f82) 기준으로 새로 썼다(씨드 구성·MUI Select 접근 이름·employees 쿼리 계약을 새로 담음).
- [러너 19:46] scout done — 직원 화면에 조직 필터 노출 (가치 3 / 위험 1 / 작업량 S)
