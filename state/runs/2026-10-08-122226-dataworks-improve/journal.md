# 회차 노트 2026-10-08-122226-dataworks-improve — dataworks
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 12:22] base pinned — main@a528bcd
- [러너 12:22] autonomy release — 

## 정찰 노트
- 선택: FactoryDashboard 집계 오류 전파. 실제 서버에서 리스크·PoC 테이블 장애와 AVG 단독 실패가 200으로 숨겨짐을 재현했고, 프로덕션 1파일로 끝나 web 설치·정책 변경 후보보다 우선했다.
- 기존 store/proxy 테스트 PASS(18.387s/48.600s). 초안을 먼저 쓰고 실행 증거와 네 라우트 오류 코드를 반영해 덮어썼다; 상세 응답은 scout-reproduction.json.
- 미확인: 신규 정상 수치/반올림 회귀 테스트와 PostgreSQL 장애 재현, 전체 Go·웹 검증. 시간 추정 32–43분은 통계적 보장이 아니다.
- 주의: data_products 장애의 home/analytics 500은 기존 형제 조회 때문이라 거짓 양성; factory/dashboard·funnel을 검사. legacy factory/products 오류 무시는 별도 후보. 스키마·auth·CI·lock 변경 금지.
- [러너 12:29] scout done — FactoryDashboard의 집계 조회 실패를 정상 KPI 0건으로 숨기지 않기 (가치 4 / 위험 1 / 작업량 S)
