# 회차 노트 2026-09-21-005406-igame-improve — igame
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 00:54] base pinned — main@f8be3ea
- [러너 00:54] autonomy release — 

## 정찰 노트
- 선택: Migrate의 실제 DB 체크섬·원자성·재실행 테스트. 정책 변경/보호 경로 수정 없이 확인된 공백을 메우며, audit DX보다 기동 안전성 가치가 높다.
- 미확인: 전용 schema+010 이력 INSERT 실패 trigger 방식은 읽은 SQL에 근거한 제안이며 실제 PG 실행·소요 시간은 구현자가 확인해야 한다.
- 주의: shared public 이력과 SQL 파일은 수정하지 말고, 새 test-db 패키지 간 격리·pgcrypto·cleanup을 확인. 45분 예산과 차선은 brief에 명시했다.
- 기본 Go/계약 검사 통과(PG skip); 조직 스킬 3개는 검색했으나 부재. 현재 메일/MCP OAuth 미머지를 프로필에 바로잡았고, 잘못된 전제의 보류 아이디어는 rejected 처리했다.
- [러너 00:58] scout done — Migrate의 체크섬 거부·DDL 롤백·재실행 안전성을 실제 PostgreSQL로 검증 (가치 4 / 위험 1 / 작업량 M)

## 구현 노트
- 1ac69f4: PostgreSQL Migrate 회귀 테스트 3개, make test-db API/database 연결, README 격리·권한·확장 준비 절차 추가.
- 체크섬/시간/seed 재실행 불변, 변조 거부·복구, 010 이력 INSERT 오류의 DDL/이력 롤백·재시도를 실제 DB로 확인.
- 초기 fixture는 public 제외 시 digest를 찾지 못해 실패; pgcrypto를 확장 전용 스키마에 준비하고 검색 경로에 포함해 해결. loader/SQL 수정 없음.
- 검증: Go test/vet/build·릴리즈 계약, PG17 신규/전체 test-db·race count=3 통과. 연결 실패는 FAIL, DSN 누락 가드 유지, 잔여 스키마 0·확장 보존 확인.
- 확신 없는 곳·검증 못 한 것: PostgreSQL 17 외 버전·비관리자 계정 권한 조합·프런트엔드 검사는 미실행. 현재 CI는 DSN을 제공하지 않아 이 PG 테스트를 자동 실행하지 않음.
- 일부러 제외: workflows/auth/SQL/프로덕션 로더·버전·릴리즈 변경. 조직 스킬 3개는 현재 도구/로컬 경로에 없어 미적용.
- 다음 역할 주의: 일회용 DB의 pgcrypto를 public 아닌 확장 전용 스키마에 미리 설치하고 README의 search_path 포함 DSN 사용; API fixture는 기본 스키마에 실제 데이터를 씀.
- [러너 01:03] brief accepted — 채택 — 현재 Migrate 및 010 SQL이 과제서와 일치하며 실제 PostgreSQL 격리 fixture를 준비해 세 수용 계약을 모두 검증했다.
- [러너 01:04] verify passed — 검증 4개 통과 (policy)

## 비평 노트
- approve / low: diff·로그·과제서·구현 노트와 실제 Migrate/SQL, 새 단언·격리·cleanup·README를 대조했고 실제 결함은 발견하지 못함.
- PG17 전체 make test-db 및 비관리자 TestMigrate -race -count=3 통과; 잔여 스키마 0·pgcrypto 보존 확인.
- CI의 DSN 미제공으로 회귀 테스트 자동 실행 공백은 남음; PG17 외 버전·프런트엔드·원격 CI 미검증.
- 조직 스킬 3개는 도구/로컬 경로에 없어 미적용; 사용자 기준으로 검토했으며 security/legal 차단 없음.
- [러너 01:05] review approved — 리뷰 승인 (risk=low)
- [러너 01:06] pr created — https://github.com/hkjang/igame/pull/23
