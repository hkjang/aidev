# 회차 노트 2026-10-08-210855-dataworks-improve — dataworks
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 21:08] base pinned — main@39f9d99
- [러너 21:08] autonomy release — 

## 정찰 노트
- 선택: factory 목록의 두 조회 오류 전달(4/1/S). 실제 UI 호출과 SQLite+HTTP의 세 거짓 200을 확인해 정책 미확인 graph/상세·웹 도구 복구 후보보다 좁고 확실하다.
- 초안을 먼저 저장한 뒤 재현으로 개정. 상품 2건·아이디어 5건·각 복구 본문 동일; 기존 store/proxy 테스트 PASS(14.812초/39.645초). 코드·커밋 무변경.
- 미확인: PostgreSQL·브라우저·전체 CI/웹 실행 및 실제 사용자 트래픽 빈도. 오류 코드는 같은 핸들러의 기존 products_failed 재사용을 선택했다.
- 주의: 아이디어는 id 전용 뷰로 dashboard 성공을 유지해야 분기 누락을 잡는다. data_products 전체 장애·store/auth/런처 변경은 제외. 12개 보류 유지+신규 2개+완료 1개를 ideas.json에 기록했다.
- [러너 21:16] scout done — 레거시 factory 상품 목록 GET의 아이디어·대시보드 조회 실패를 500으로 전달 (가치 4 / 위험 1 / 작업량 S)

## 구현 노트
- handleFactoryProducts의 아이디어·대시보드 오류를 기존 500/server_error/products_failed로 즉시 반환; 거짓 성공/부분 응답 차단. 커밋 d0f5c75, 프로덕션 1+테스트 1+문서 1파일.
- 실제 SQLite+NewServer.Routes 신규 테스트: 세 장애 red→green→프로덕션 원복 시 동일 red 확인; 매 cleanup 복구 후 전체 본문 동일, 빈 배열/0 KPI·status trim·순서/내용·아이디어 50개 제한 유지.
- 검증: 신규/기존 dashboard·store/proxy·전체 Go 테스트 PASS, build/vet/diff 검사 exit 0, API 감사 550/612·누락 0. 로그는 assets/test-{red,green,reverted,dashboard,store-proxy}.log 및 go-*.log.
- 확신 없는 곳·검증 못 한 것: PostgreSQL 장애·실제 브라우저 오류 표시·웹 lint/test/build 미실행; 실제 사용자 트래픽 빈도 미확인.
- 일부러 제외: store SQL/마이그레이션·auth·웹/런처/lockfile·상세 GET·graph·publish gate·릴리즈. 이번 두 호출부 분기로 범위 한정.
- 다음 역할 주의: 아이디어는 id 전용 뷰로 COUNT 성공을 유지해야 오류 분기를 분리 검증한다. 테스트의 rename/뷰/시각 고정 SQL은 t.TempDir SQLite에만 실행; 테스트를 병렬화하지 말 것.
- [러너 21:22] brief accepted — 채택 — HEAD 39f9d99와 두 오류 무시 지점이 일치했고 지정된 세 SQLite 장애 레버로 수정 전 실패·수정 후 500·각 스키마 복�
- [러너 21:23] verify passed — 검증 8개 통과 (auto)

## 비평 노트
- approve / low / blocking=[]: 이번 두 오류 분기·문서·테스트에서 실제 결함 없음; 코드 무변경.
- 기준 주의: 로컬 main=baa3415는 오래됨. pinned base 및 origin/main=39f9d99 → d0f5c75의 3파일 정밀 검토; 누적 73파일 전체 재승인 아님.
- 실제 SQLite/HTTP 테스트의 독립 장애 레버·복구·정상 계약 확인, red/reverted 로그의 세 거짓 200 확인; store/proxy 재실행 PASS(15.500초/42.850초).
- 세 부서 스킬 로컬 적용; 인가·데이터 처리·가역성 변경 위험 없음. PostgreSQL·브라우저·웹/전체 CI 미실행; 릴리즈 시 목록 장애의 500 계약을 유지할 것.
- [러너 21:26] review approved — 리뷰 승인 (risk=low)
- [러너 21:26] pr created — https://github.com/hkjang/dataworks/pull/39
- [러너 21:29] ci passed — 검사 2개 모두 success
- [러너 21:29] merge done — d0f5c75
- [러너 21:29] release missing — 릴리즈 결과 없음/손상: missing
