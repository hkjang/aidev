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

## 구현 노트
- c835720: FactoryDashboard 10개 집계의 오류를 반환하도록 수정; SQL·순서·부분 구조체·반올림 유지. 프로덕션 1 + 테스트 2 + 문서 1파일.
- 실제 SQLite/Routes 회귀 테스트: 12개 HTTP 단언이 수정 전 200으로 실패, 수정 후 500/기존 코드/KPI 부재 확인; 원본 재역전에서도 동일 실패(test-red.log/test-reverted.log).
- 빈 DB·장애 cleanup 후 네 경로의 200/0 복귀, 여섯 상태별 수·리스크 >=70·pending PoC·양수 AVG 반올림 모두 검증.
- 신규/기존 store·proxy 테스트, 전체 build/vet/test, API 감사(550/612, 누락 0), diff 검사 PASS; 상세 로그는 이 회차 폴더.
- 미검증: 실제 PostgreSQL 장애 레버, 웹/npm·브라우저·Pages 실행. SQL과 웹을 변경하지 않아 이번 검증 범위에서 제외.
- 일부러 제외: legacy factory/products의 오류 무시(별도 pending); 스키마·auth·CI·의존성·릴리즈 변경 없음.
- 다음 역할 주의: 테이블/뷰 장애 테스트는 임시 SQLite에서만 순차 실행하며 rename 직후 cleanup 등록. 요청한 세 technology 스킬은 Skill 도구 부재로 로컬 SKILL.md를 읽어 적용.
- [러너 12:37] brief accepted — 채택 — HEAD a528bcd와 대상 코드가 일치했고 지정된 SQLite/HTTP 장애 레버로 수용 기준의 실패·복구·정상 계산을 실행 확인�
- [러너 12:38] verify passed — 검증 8개 통과 (auto)

## 비평 노트
- approve / low / blocking 없음. 오래된 main diff·log 확인 후 원장 고정 base a528bcd→c835720의 4파일 심사; 코드 수정 없음.
- 10개 집계 오류·네 HTTP 호출부·정상 계산·격리 DB cleanup·문서 일치 및 수정 전 12개 실패 로그 확인; store/proxy 독립 재실행 PASS(15.175s/39.961s).
- 신규 개인정보·권한·입력·의존성·스키마 변경과 보안/법무 차단 사유 없음; 요청한 세 스킬은 Skill 도구 부재로 로컬 SKILL.md 적용.
- 미검증: 실제 PostgreSQL·웹/브라우저/Pages; 전체 Go 검증은 기존 로그 확인. legacy admin_factory.go:110 오류 무시는 후속 과제이며 이번 해결로 홍보하지 말 것.
- [러너 12:40] review approved — 리뷰 승인 (risk=low)
- [러너 12:40] pr created — https://github.com/hkjang/dataworks/pull/38
