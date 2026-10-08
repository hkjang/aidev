# 회차 노트 2026-10-08-104905-ai-admin-improve — ai-admin
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 10:49] base pinned — main@71dad05
- [러너 10:49] autonomy release — 

## 정찰 노트
- chat 후행 JSON/쓰레기가 실제 HTTP 200·upstream 1회로 실행됨을 재현해 EOF 검사를 선택했다(3/1/S, 프로덕션 1파일). 숫자 정밀도 결함은 토큰 호환성 위험이 더 커 분리했다.
- 미확인: 4 MiB 후행 공백 경계의 HTTP 결과, 기본 공급자가 있을 때 최상위 null 결과, 신규 테스트 및 전체 race/lint/build. 기존 관련 3개 테스트는 전용 PostgreSQL에서 SKIP 없이 PASS(1.639초).
- 공용 decodeJSON·UseNumber·auth/migrations/workflows·웹/dist를 건드리지 말 것. raw 본문 테스트는 json.Marshal하는 session.do 대신 실제 HTTP 요청을 사용하고 upstream 0회를 검증한다.
- b5146f2/f66d25c/8e65c33은 main 미포함 별도 브랜치 확인; 재구현하지 않는다. 프로필은 2026-10-07·동일 main@71dad05로 최신이라 새로 쓰지 않았다. 초안을 먼저 저장한 뒤 재현 근거로 덮어썼고 저장소 코드는 변경하지 않았다.
- [러너 10:54] scout done — chat 요청의 첫 JSON 뒤 추가 JSON·쓰레기 문자를 공급자 호출 전에 400으로 거부 (가치 3 / 위험 1 / 작업량 S)
