# 회차 노트 2026-09-21-195440-muni-improve — muni
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 19:54] base pinned — main@1547faa
- [러너 19:54] autonomy release — 

## 정찰 노트
- DOCX 제목 중복은 실제 writer/reader와 신규 import 분기의 근거가 연결되고 기존 제거 헬퍼로 해결 가능하여, 보호 경로 e2e·메일 캠페인 충돌·서버 캡처보다 작고 확실하다.
- HEAD 1547faa에 지난 HTML/ZIP 개선이 없어도 재수행하지 않는다. 실제 DOCX HTTP 실패 재현은 미확인; 구현자가 writer 출력으로 먼저 증명해야 한다.
- go test ./...·placeholder 통과, DSN 미설정으로 live 검증 없음. docxImport 자체·끼워 넣기·HTML 정책을 바꾸지 말고 전용 DB에서 추가 테스트 SKIP 여부 확인.
- 확대 검색으로 headcount/plugins의 요청된 세 SKILL.md를 찾아 적용했다(전용 Skill 도구는 없음). 과제서에 대안·추정 범위/예비·검증 체크포인트 보완; 기존 12개 후보와 새 후보 2개·기록 승계 2개를 기록했다.
- [러너 19:59] scout done — DOCX 신규 가져오기에서 확정 제목과 같은 첫 H1만 본문에서 제거하기 (가치 3 / 위험 1 / 작업량 S)
