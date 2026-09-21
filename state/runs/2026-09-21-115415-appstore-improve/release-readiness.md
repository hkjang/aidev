# v2.11.4 릴리즈 준비

적용 스킬: marketing:product-launch, technology:release-and-deployment. Skill 호출 도구가 없어 headcount 저장소의 원본 SKILL.md와 references/sources.md를 읽음. 별도 강제 반환 schema는 없으며 release.json은 사용자 지정 형식을 따름.

Tier 3: 테스트 회귀 검증 보강으로 릴리즈 노트만 준비. 대상은 SSO/Bootstrap 설치 운영자와 유지보수자. 새 기능이나 사용자 행동 변화가 없고 별도 가격·지원 정책 변경 없음. 외부 홍보 및 메시지 전송 없음. 팀 외부 사용자의 신규 사용 검증은 이 무인 로컬 세션에서 수행하지 못함. 이를 실제 사용자 검증으로 주장하지 않음.

품질 게이트: 정적 검사, 단위·실제 PostgreSQL 통합 테스트, production bundle desktop/mobile E2E. 하나라도 실패하면 릴리즈 중단 및 이번 변경 복원. 로컬 판단 책임자는 릴리즈 에이전트이며 외부 배포는 러너/기존 CI가 담당. 릴리즈 이미지 빌드·load·non-root·내부망 smoke 및 SHA-256은 기존 release.yml의 게시 전 게이트를 사용. 로컬에서 별도의 게시 자산을 만들지 않음.

배포 후 운영자가 로그인 실패·복구 로그인 문의 증가를 확인하고 재현 가능한 신규 회귀 1건이면 확대를 중단하고 v2.11.3 이미지로 복원. schema 변경이 없어 데이터 롤백 불필요. 신규 채택 목표는 해당하지 않으며 이번 변경의 측정 기준은 로그인 구성별 회귀 검증 통과이다. 운영 모니터링 및 실제 Keycloak 검증은 아직 수행하지 않았음.
