# 회차 노트 2026-09-21-180425-kanpic-improve — kanpic
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 18:04] base pinned — main@742871c
- [러너 18:04] autonomy release — 

## 정찰 노트
- 인코딩 검증 누락은 실제 HTTPS→Resolve에서 손상 표 성공을 재현했고, 숫자 공용 픽스처·출처 API보다 직접 효과와 작은 변경 범위가 분명해 선택했다.
- #VALUE!는 기존 parseCSV 오류 코드에 맞춘 설계 판단이며 실제 브라우저/DB까지의 영향은 미확인이다. Go 전체 테스트·vet·build 통과; 웹/DB/E2E 미실행.
- ToUTF8 뒤에서만 검사하고 UTF-16 복구·캐시·WEBSERVICE·SSRF·숫자·XLSX를 건드리지 말 것. 증거 overlay는 회차 폴더에만 있으며 저장소 변경 없음.
- Skill 도구는 없어 headcount 형제 저장소의 요청한 세 SKILL.md를 직접 읽어 적용했다. brief에 대안·추정 근거·예비·검토 지점을 기록했다.
- [러너 18:10] scout done — IMPORTDATA가 업로드처럼 잘못된 UTF-8 입력을 거절한다 (가치 3 / 위험 1 / 작업량 S)
