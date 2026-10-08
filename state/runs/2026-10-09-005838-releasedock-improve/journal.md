# 회차 노트 2026-10-09-005838-releasedock-improve — releasedock
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 00:58] base pinned — main@da054c9
- [러너 00:58] autonomy release — 

## 정찰 노트
- 다른 실행의 로그·커서가 섞일 수 있는 SimpleRunDetailPage의 id 경계를 선택; 공용 helper 개명보다 사용자 가치가 크고 DB/성능 후보보다 검증 준비가 작다. 프로덕션 1+테스트 1파일로 한정.
- 근거는 실제 App 라우트, useAsync, loadStoredLogs와 스트림 effect에서 확인; 새 결함을 실행한 실패 테스트는 미작성이라 구현자가 지연 응답으로 먼저 고정해야 한다.
- key는 실행 id에만 묶고 내부 컴포넌트는 모듈 수준에 둔다. 같은 id 새로 고침 경쟁·AbortSignal·공통 훅 변경은 제외; 기존 재연결/커서 테스트 유지.
- 세 headcount SKILL.md를 로컬에서 읽어 비교·추정·검증 체크포인트를 brief에 반영; 작업 트리 무수정. 산출물 안의 무수정 웹 복사본에서 149건/13파일과 tsc 통과; 새 결함 재현·build·DB 검증은 미실행.
- [러너 01:07] scout done — 단순 실행 상세에서 다른 실행으로 이동할 때 로그·커서·지연 응답을 실행 ID별로 격리하기 (가치 4 / 위�
- [러너 01:07] brief unstated — 구현자가 과제서 판정을 적지 않음
- [러너 01:07] improve no-change — 커밋 없음
