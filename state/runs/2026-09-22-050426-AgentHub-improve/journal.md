# 회차 노트 2026-09-22-050426-AgentHub-improve — AgentHub
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 05:04] base pinned — main@d42cc59
- [러너 05:04] autonomy release — 

## 정찰 노트
- guide-shots 백업 실패 후 쓰기 지속을 원본 블록 vm 실행으로 재현하여 선택. runtime/base·auth·workflow 변경 없이 설정 손실을 차단해 다른 후보보다 가치/위험이 유리하다.
- 기존 Node 57건·Go API/runtime-proxy·guide-shots 구문 검사 통과. DB/브라우저 실물 검증은 미확인이며 새 테스트는 구현자가 추가한다.
- 네 API 값의 복원 형식을 섞지 말고 설정 누락 시 seed 전체 중단. 복원 오류 수집·CSP 기록 전체 삭제는 후속으로 남긴다.
- 요청된 세 부서 스킬은 도구/로컬 검색에서 미발견이라 형식 미확인. 프로필의 이미 해결된 Langflow 항목을 갱신했으며 코드는 변경하지 않았다.
