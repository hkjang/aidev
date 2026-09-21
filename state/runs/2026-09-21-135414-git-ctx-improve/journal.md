# 회차 노트 2026-09-21-135414-git-ctx-improve — git-ctx
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 13:54] base pinned — main@cf3b598
- [러너 13:54] autonomy release — 

## 정찰 노트
- YAML 리스트 형제 필드 과마스킹을 선택: 실제 name·port 소실/빈 본문 허위 finding을 재현했고 인증 변경·반려 파서 확장·미재현 flake보다 근거와 범위가 명확하다.
- 원본 복사 probe 및 contentsecurity/indexer/search/mcp 테스트 exit 0; 전체·race·외부 서비스 미검증. YAML 라이브러리 대조는 미실행이며 키 열 계산은 구현 제안이다.
- Revision 현재 edeca363cffe; 경계 계산만 바꾸고 지문을 놓치지 말 것. 이전 성공이나 main 미반영인 curl/변수/OAuth 작업을 다시 묶지 말 것.
- 기존 후보 12개 유지·재평가, 이전 노트 후보 1개와 신규 2개 포함 총 15개 pending. 요청한 세 스킬·Skill 도구 부재로 반환 형식 미확인.
- [러너 13:58] scout done — YAML 리스트 블록 스칼라가 형제 필드까지 과마스킹하지 않게 수정 (가치 3 / 위험 2 / 작업량 M)
