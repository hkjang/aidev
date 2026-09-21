# 회차 노트 2026-09-21-115415-appstore-improve — appstore
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 11:54] base pinned — main@a858bf4
- [러너 11:54] autonomy release — 

## 구현 전 검토
- 기준 a858bf4, 작업 트리 깨끗함. README·최근 30개 커밋·CI·테스트 구성·관련 docs와 현재 구현을 확인. 저장소 CLAUDE.md/AGENTS.md/로드맵 및 TODO/FIXME 없음. 목적은 오프라인 사내 앱 카탈로그이고 Go/chi/pgx와 React/Vite/TanStack Query를 사용한다.
- Skill 호출 도구는 없었음. /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/{completion-verification,systematic-debugging,test-driven-development}/SKILL.md 원본을 읽음.
- 기존 보류 12건을 현재 파일과 대조하고 신규 2건을 ideas.json에 채점. 선행 MCP OAuth·메일 코드 없음, 정책 미확인 검토 단계 변경은 보류, PR #26의 미병합 기능은 재구현하지 않음.
- 선택은 LoginPage 기본 3상태 테스트 공백(2/1/S). 제품 버그를 재현한 수정이 아니라 정상 동작을 지키는 테스트 과제이다. 따라서 기존 코드에서 새 테스트는 처음부터 통과했으며 이를 TDD red로 주장하지 않는다. 프로덕션 코드 변경은 없고 회귀 감지 능력은 일시적 mutation 실패→원본 복원 통과로 확인한다.
- 실제 LoginPage/AuthProvider/TanStack Query/API 함수를 사용하고 fetch/HTTP 응답만 대체. 모듈 대역·가짜 런타임 객체·소스 문자열 단언은 없음. 세션과 설정 조회 완료를 기다려 임시 폼을 잘못 검증하는 초기 테스트 허점을 수정함.
- mutation: 복구 토글 제거 시 새 테스트 1건 실패, bootstrapAvailable 무시 시 새 테스트 4건 실패, oidcEnabled 무시 시 새 테스트 1건과 기존 테스트 1건 실패. 로그는 impl-mutation-*.log. 모든 프로덕션 코드를 원본으로 복원함.
