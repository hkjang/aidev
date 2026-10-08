# 회차 노트 2026-10-09-000833-muni-improve — muni
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 00:08] base pinned — main@9d781ef
- [러너 00:08] autonomy release — 

## 정찰 노트
- 임의 ZIP 파일/폴더 충돌 후보에서 고정 목록.md 충돌만 분리했다. 프로덕션 1파일의 기존 예약 로직으로 해결 가능해 Windows 문자 처리·e2e·중복 폴더 API 계약보다 위험이 낮다.
- 실제 코드의 빈 claimed·고정 목록.md·허용 폴더명을 확인했고 합성 ZIP Linux 추출 오류를 재현했다. 실제 HTTP 회귀 및 Windows/macOS 추출은 미확인이므로 구현자가 live 실패부터 확인한다.
- 선택 테스트 PASS 7/SKIP 2, placeholder 통과. 전용 DB와 자기 ID cleanup을 사용하고 map 순서·기존 문서 예약·휴지통 분리를 보존한다.
- 세 요청 스킬은 전용 Skill 도구가 없어 로컬 SKILL.md를 직접 읽어 적용했다. 10/08 HWPX 수정은 pinned base에 없지만 이미 수행된 과제라 재선택하지 않았다.
- [러너 00:14] scout done — 워크스페이스 ZIP의 루트 목록.md 폴더가 고정 안내 파일과 충돌하지 않게 하기 (가치 3 / 위험 1 / 작업량 S)
