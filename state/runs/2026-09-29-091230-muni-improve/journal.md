# 회차 노트 2026-09-29-091230-muni-improve — muni
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 09:12] base pinned — main@cebe102
- [러너 09:12] autonomy release — 

## 구현 노트
- ZIP 내보내기가 정확히 2000건일 때 초과 안내를 내던 오류를 LIMIT+1 조회와 초과 행 제외로 수정했습니다(86476c4). 프로덕션 1파일, 테스트 1파일.
- 실제 DB에 일괄 삽입하고 실제 인증 HTTP 라우트의 ZIP을 읽는 1999/2000/2001 경계 테스트 추가; 수정 전 실패·수정 후 통과·수정 복원 시 같은 실패 확인.
- make test 통과(프런트 297건), Go 전체 재실행 통과(httpapi 239 PASS/SKIP 0), vet·포맷·프런트 빌드·Go 바이너리 빌드 통과.
- 검증 못 한 것: 외부 HWP/HWPX 코퍼스 및 수동 DOCX 출력 테스트 4건 SKIP; Playwright·Docker 이미지 빌드는 미실행. 코드/빌드 경로 변경은 없음.
- 제목 절단·헤더 통합·정렬 동률은 범위 밖이라 미수정. 기존 12개 아이디어와 신규 3개 평가를 ideas.json에 기록했습니다.
- 다음 역할: 새 테스트는 MUNI_TEST_DSN이 필요하며 전용 DB에서 실행하세요. 본 세션의 전용 DB 컨테이너는 검증 뒤 제거했습니다.
- technology 스킬 3개는 Skill 전용 도구가 없어 로컬 headcount SKILL.md로 읽고 적용. 빌드 번들은 제거하고 tracked placeholder 복원; 릴리즈·푸시 없음.
- [러너 09:18] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- approve / low, security·legal 차단 없음. LIMIT+1·초과 행 제외·목록·감사 건수·인증/인가·가역성 확인.
- 전용 PostgreSQL 16에서 경계 1999/2000/2001 및 ZIP 관련 10개 테스트(하위 3개 포함) 통과, SKIP 없음; 원장 실패 증거와 단언 일치. 전용 컨테이너 제거.
- 로컬 main=1045e13은 낡았고 origin/main=회차 base=cebe102; 이번 2파일만 집중 심사, 과거 73파일 전체 재심사·수정 전 실행·테넌트 격리 동적 검증·전체/브라우저/외부 코퍼스 검증은 미실행.
- 기존 webui/dist/index.html 미커밋 변경 보존. 릴리즈는 정확히 2000건의 오안내 수정으로 기술; 정렬 동률·제목 절단 등 기존 보류 사항은 이번 승인 범위 밖.
- [러너 09:20] review approved — 리뷰 승인 (risk=low)
- [러너 09:21] pr created — https://github.com/hkjang/muni/pull/30
- [러너 09:26] ci passed — 검사 2개 모두 success
- [러너 09:26] merge done — 86476c4
- [러너 09:37] release published — v0.50.0
- [러너 09:41] assets verified — v0.50.0 자산 1개 (이전 v0.48.0: 1)
