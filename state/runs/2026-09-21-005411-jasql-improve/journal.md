# 회차 노트 2026-09-21-005411-jasql-improve — jasql
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 00:54] base pinned — main@e9f3fb2
- [러너 00:54] autonomy release — 

## 정찰 노트
- 불가능한 날짜 2025-02-30이 실제 MCP resolve_time에서 문자열 날짜 조건으로 출력됨을 재현해 공통 파서 검증을 선택; 정규식 최적화·포맷보다 사용자 출력 개선이 명확하고 보호 경로를 피한다.
- go build/vet/test 전부 통과(catalog 55초); Oracle 실 실행·실사용 빈도는 미확인. 세 MCP 경로 정상/비정상 회귀 테스트와 TempDir JSON 로드를 요구한다.
- 이전 SQL 정리 개선은 기록상 성공이나 pinned e9f3fb2에 미반영이므로 반복하지 않는다. 실데이터 참조 오류 5건도 이번 범위에서 제외한다.
- 요청된 세 부서 스킬은 도구 목록/로컬 검색에서 찾지 못해 원문 절차·반환 형식 미확인; 사용자 형식으로 산출물을 작성했다. 구현은 숫자 날짜 검증만 좁히고 구분자·상대 기간·인증을 건드리지 않는다.
- [러너 00:58] scout done — 명시적 숫자 날짜의 달력 유효성을 공통 파서에서 검증 (가치 4 / 위험 1 / 작업량 S)

## 구현 노트
- 3b9fcb1: 공통 파서 raw 숫자 날짜 루프에 time.Parse 달력 검증 3줄을 추가해 불가능한 날짜 조건을 제거했다.
- 파서 표 테스트와 실제 /mcp tools/call 540개 조합(5타입×4표기×9입력×3도구); TempDir JSON/catalog.Load 사용, 대역·로드 후 mutation 없음.
- 수정 전 잘못된 range/조건/SQL 실패 확인(calendar-red.log), 수정 후 지정 부분 테스트 및 build/vet/전체 test 통과(catalog 55.844초, mcp 9.826초).
- 확신 없는 곳·검증 못 한 것: Oracle 실 실행·실사용 빈도 미확인; PostgreSQL 실 DB 검증을 주장하지 않음(전체 실행 meta 캐시).
- technology:completion-verification/systematic-debugging/test-driven-development와 Skill 도구는 검색상 부재; 로컬 superpowers 디버깅·TDD 스킬 대체 참고, 부서 원문 형식 적용 미주장.
- 구분자 정규식·상대 기간·렌더러 직접 입력 방어·인증·실데이터·릴리즈·전회 SQL 정리는 범위 밖으로 변경하지 않았다.
- 다음 역할: 정상 윤일의 정확한 SQL 조건도 단정하므로 빈 응답으로 통과할 수 없음; 실 Oracle/PostgreSQL 검증에는 별도 환경이 필요하다.
- [러너 01:02] brief accepted — 채택 — 현재 코드의 검증 없는 raw 날짜 추가 및 세 도구의 공통 파서 배선이 정찰 근거와 일치했다.
- [러너 01:03] verify passed — 검증 3개 통과 (auto)

## 비평 노트
- 판정 approve / risk low / blocking 없음: main diff·로그, 변경 파일 전체와 공통 파서 호출 경로 확인; 실제 머지 차단 결함 없음.
- 날짜 부분 테스트(-count=1) 및 diff --check 통과; 정상 윤일 SQL·비정상 날짜·혼합 입력 단언 확인, base의 무검증 추가가 새 단언과 충돌함을 소스로 확인.
- 인증·권한·의존성·실데이터·외부 상태 변경 없음; 코드 revert로 복구 가능.
- Oracle/PostgreSQL 실 DB·실사용 빈도 및 전체 build/vet/test 재실행은 미검증; 릴리즈에는 불가능한 날짜를 무시하여 날짜 조건이 생략되는 동작을 명시할 것.
- [러너 01:04] review approved — 리뷰 승인 (risk=low)
- [러너 01:05] pr created — https://github.com/hkjang/jasql/pull/2
- [러너 01:09] ci no-ci — 이 커밋에 검사가 없음 (정책 allow_merge_without_ci 가 없으면 차단)
