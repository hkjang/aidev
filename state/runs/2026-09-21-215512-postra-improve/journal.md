# 회차 노트 2026-09-21-215512-postra-improve — postra
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 21:55] base pinned — main@5dd0099
- [러너 21:55] autonomy release — 

## 정찰 노트
- POP3 MaxMessageBytes 무시를 선택: 실제 메모리 보호 공백이라 일반 테스트 추가·README 버전 정렬보다 가치가 높고 auth/workflows를 피할 수 있음(4/2/M, 35분+예비10분).
- app.go:dialInbound → pop3 Dial/retrBody → sync.go:ingestOne 순서로 정적 확인; 과대 본문 실행 재현과 메모리 수치는 미확인, 구현자는 작은 상한의 실제 TCP 실패 테스트부터 작성.
- 긴 행을 다 읽은 뒤 검사·DotReader의 LF 변환·초과 시 무한 drain을 피할 것. 실제 application 배선과 정상 메일 해시/중복 유지 검증 필수, 개별 실패를 JobFailed로 바꾸지 말 것.
- 기존 pending 재평가 및 신규 2개 포함 ideas.json 작성, 현재 v0.23.3/lint 반영 profile 갱신. 전체 Go race·포맷·계약 검사 통과, git status 깨끗함. 부서 스킬/Skill 도구 미발견이며 원문 적용 주장 없음; SMTP timeout·notifymail/handoff 재적용 금지.
- [러너 22:00] scout done — POP3 본문 수신에 MaxMessageBytes 상한 적용 — 긴 단일 행까지 제한하고 실제 TCP·동기화 배선으로 검증 (가치 
