# 회차 노트 2026-09-29-070145-igame-improve — igame
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 07:01] base pinned — main@7d21df9
- [러너 07:01] autonomy release — 

## 정찰 노트
- 선택: API PG fixture 호출별 스키마 격리. 제품 파일 0개·helper 중심이고 반복된 설정/행 공유를 줄여, 정책 미확정인 재-finish 및 넓은 시간대 오류 전파보다 우선했다.
- 초안을 먼저 저장 후 실DB baseline PASS(API 9.887s/database 1.886s)로 보강. 새 격리 구현은 미검증이며 기존 API SQL의 격리 호환성은 구현자의 전체 PG 회귀로 확정한다.
- 주의: cleanup LIFO, pgcrypto 전용 namespace 보존, 부모 fixture 공유 서브테스트의 델타/복원 유지. auth/migrations/workflows 제품 변경 금지.
- 요청 3개 조직 스킬은 도구 부재로 로컬 SKILL.md를 읽어 적용. 현재 audit-release 타깃 부재 등 이전 프로필 불일치는 profile에 정정; 코드·커밋 변경 없음.
- [러너 07:06] scout done — API PostgreSQL fixture를 호출별 전용 스키마로 격리 (가치 3 / 위험 2 / 작업량 M)

## 구현 노트
- migratedPool 호출별 UUID 스키마, startup search_path, 유한 setup/drop context 및 LIFO cleanup을 구현했다. 제품 파일 0개, 테스트·README 4개 변경.
- 새 TestMigratedPoolIsolation은 같은 setting key의 독립성, pool별 동시 2연결의 스키마/검색 경로, 자식 cleanup 후 형제와 pgcrypto OID/namespace 보존을 검증한다.
- Red: fixtures share schema "public"; 변경 후 PASS, 기존 helper 복원으로 동일 Red 재현 후 수정 복구.
- 실DB PG17: make test-db PASS(API 24.991s/database 1.958s), 격리 테스트 race 3회 PASS(3.903s); 잔여 테스트 스키마 0개. DSN 없는 API/database 테스트, vet, build, diff, release contract PASS.
- 확신 없는 곳·검증 못 한 것: 강제 네트워크 장애/권한 부족 및 교체 연결을 강제로 만드는 별도 주입 테스트는 하지 않았다. 프런트·SDK·원격 CI는 범위 밖이다.
- 기존 preserve 복원과 부모 fixture 내 행 수 델타는 유지했다. 요청한 technology 스킬 3개는 Skill 도구 부재로 로컬 SKILL.md를 직접 읽고 적용했다.
- 다음 역할 주의: IGAME_TEST_DSN 없으면 PG 회귀는 skip된다. 일회용 DB와 애플리케이션 테이블 없는 pgcrypto 전용 스키마를 먼저 준비해야 한다.
- [러너 07:10] brief accepted — 채택 — 현재 helper의 기본 스키마 공유를 실DB Red로 확인했으며 지정된 4개 파일 범위에서 세 수용 기준을 모두 검증했다.
- [러너 07:10] verify passed — 검증 4개 통과 (policy)
