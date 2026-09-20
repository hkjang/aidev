# 회차 노트 2026-09-20-232418-hunter-improve — hunter
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 23:24] base pinned — main@96f7414
- [러너 23:24] autonomy release — 

## 정찰 노트
- CSV 두 경로의 보호 불일치를 선택: 실제 보고서 링크→exportReport→csvSafe 배선 확인, auth·워크플로 변경 없이 좁은 HTTP 회귀로 검증 가능.
- 메일·추적 스니펫·handoff 상한은 성공 기록과 달리 main@96f7414에 없어 중복 선정하지 않음. OIDC 공통 벡터는 의도된 차이가 많아 후순위.
- 서버 HTTP/DB와 실제 스프레드시트 실행은 미확인; 구현자는 먼저 HTTP 실패를 재현하고 JS/Go 공백 집합 차이·NUL 저장 불가에 주의.
- 웹92개 통과(Node22, CI는26); 코드 변경·커밋 없음. 요청된 회사 스킬/Skill 도구는 검색했으나 제공되지 않아 미적용.
- [러너 23:28] scout done — 서버 CSV 보고서의 공백·제어 문자 뒤 수식 접두사 보호를 목록 CSV와 일치시키기 (가치 4 / 위험 1 / 작업량 

## 구현 노트
- 74f07c7: csvSafe만 수정하여 JS 공백+C0 뒤 수식 보호를 목록 CSV와 일치시켰다. 원문을 잘라 저장하지 않는다.
- 공유 JSON 83사례를 실제 Go/TS 함수로 비교; PostgreSQL17 격리 스키마 HTTP로 수정 전 실패→수정 후 race 통과(2개 테스트, skip 없음).
- CSV BOM·열·다운로드 헤더·한글·팀 격리·미인증401·JSON 원문 보존 검증. Node26.9 웹93개, Go1.26.7 vet/build, PentAGI312 해시, diff 검사 통과.
- 확신 없는 곳·검증 못 한 것: 실제 Excel/LibreOffice 실행, 전체 Go 스위트, CI·배포. CRLF 디코딩 정규화는 이번 사례 밖.
- 일부러 하지 않은 것: UI·권한·조회·감사·JSON·버전 변경. 지정 범위 밖이며 웹 보호는 기준 동작 그대로 유지.
- 다음 역할 주의: HTTP 테스트에는 HUNTER_TEST_DSN이 필요하며 없으면 skip된다. NUL·빈 제목은 함수 사례만 검증. 임시 DB 컨테이너와 빌드 실행 파일은 제거했다.
- Skill 도구/technology 3종은 미제공·검색 실패. 로컬 superpowers systematic-debugging/test-driven-development/verification-before-completion을 대체 적용; 원래 회사 스킬 형식은 미확인.
- [러너 23:33] brief accepted — 채택 — 현재 csvSafe와 웹 판정의 차이가 과제서와 일치했고 실제 서버 HTTP 실패로 확정했다.
- [러너 23:33] verify passed — 검증 9개 통과 (auto)

## 비평 노트
- approve / low / blocking 없음: 74f07c7의 4파일, CSV 실제 호출·권한 경로·공유 벡터·원문 보존·변경 범위·revert 가능성을 검토했고 실제 결함을 찾지 못했다.
- 독립 검증: Go1.26.7 공유 벡터83개, Node22.23.1 관련 웹5개, diff 검사 통과; 기존 함수는 공백 뒤 수식 기대값을 만족하지 못함을 코드로 확인.
- HTTP/DB 테스트는 DSN 미설정으로 skip; 실제 Excel/LibreOffice·CRLF 왕복·전체 Go·CI/배포 미검증이며 릴리즈에 실증 범위를 과장하지 말 것.
- 요청 회사 스킬3종/Skill 도구는 제공 도구·로컬 검색에서 찾지 못해 미적용; 사용자 지정 심사 형식 적용.
- [러너 23:35] review approved — 리뷰 승인 (risk=low)
- [러너 23:35] pr created — https://github.com/hkjang/hunter/pull/8
