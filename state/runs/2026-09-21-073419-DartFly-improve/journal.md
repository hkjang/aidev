# 회차 노트 2026-09-21-073419-DartFly-improve — DartFly
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 07:34] base pinned — main@f09cfc0
- [러너 07:34] autonomy release — 

## 정찰 노트
- TSV 두 경로의 실제 출력 결함을 선택: 페이지 이동보다 범위가 작고 인증·DB 스키마를 피하면서 문자열 보호와 숫자 보존을 함께 바로잡는다.
- 코드상 saved.js 보호 누락·app.js 숫자 문자열화 확인; 실제 브라우저/Excel 재현은 미확인. 전체 go test -race ./... 통과(일부 캐시).
- 구현자는 기존 spreadsheetSafe 계약과 saved.js 상세 경합 방지를 유지하고 실제 서버·MariaDB·브라우저 클립보드로 두 경로를 검증할 것; 대역/소스 검사만으로 완료하지 말 것.
- 회사 스킬 3개·Skill 도구를 찾지 못해 적용 불가. 큰 정수 후보는 신규 저장 전 문자열 정규화를 확인해 기각했으며, 미병합 기능은 계속 보류했다.
- [러너 07:38] scout done — 저장 결과·워크스페이스 TSV 복사의 수식 방지와 숫자 타입 처리 일치 (가치 4 / 위험 1 / 작업량 M)

## 구현 노트
- e6bd782: dialect.js에 타입 판정 후 문자열화하는 tsvCell을 추가하고 app.js/saved.js의 TSV 헤더·값에 적용했습니다. spreadsheetSafe 계약과 상세 선택·삭제 상태 로직은 유지했습니다.
- 실제 원본 바이너리에서 두 클릭 경로의 TSV 실패를 확인한 뒤 수정 후 동일 회귀가 통과했습니다(smoke-red.log / smoke-green.log). API·DOM·클립보드는 대역으로 교체하지 않았습니다.
- 검증: JS 지정 3파일(상세/삭제 경합 11개 포함), go test -race ./...(38패키지), go vet ./..., 전체 필수 브라우저 스모크(빌드·31페이지·SQL 실행/저장·실제 클립보드·정렬/필터·JSON/Markdown), git diff --check 통과.
- 확신 없는 곳·검증 못 한 것: 실제 Excel의 수식 실행은 미검증입니다. MariaDB TRUE는 숫자 1이므로 JS boolean은 helper 테스트로 보완했고, MariaDB가 제거하는 별칭 선행 공백도 helper 테스트로 확인했습니다.
- 스킬 제약: 요청한 technology:completion-verification/systematic-debugging/test-driven-development와 Skill/skills.list/read 도구를 제공 목록·로컬 경로에서 찾지 못했습니다. 사용자 지정 재현→수정→검증 절차로 진행했습니다.
- 일부러 하지 않은 것: CSV 환경변수·저장 JSON·인증/SSO·마이그레이션·새 자산 등록·릴리즈는 이번 TSV 범위 밖이라 변경하지 않았습니다.
- 다음 역할 주의: 회귀는 Docker MariaDB와 Python Playwright Chromium이 필요합니다. 이번 환경의 Chromium 미설치는 설치로 해결했고, SQL 헤더의 탭/CR/LF는 MariaDB 문자열 별칭으로 생성합니다.
- [러너 07:47] brief accepted — 채택 — 실제 클릭에서 보호 누락·숫자 과보호·헤더 구조 손상을 재현했고, 두 화면에 같은 변환을 적용한 뒤 같은 회귀
- [러너 07:47] verify passed — 검증 3개 통과 (auto)

## 비평 노트
- 판정 approve / risk low / blocking 없음: 변경 7파일의 타입·경계값·클릭 배선·경합·범위·되돌리기를 확인했고 신규 결함은 찾지 못했습니다.
- JS 13개 및 diff --check 통과; 새 스모크 단언과 기존 red/green 로그에서 수정 전 결함 검출을 확인했습니다.
- 직접 스모크는 빌드·서버 검사 통과 후 Chromium 실행 파일 부재로 브라우저 단계 실패: 독립 브라우저 재검증은 미완료입니다.
- 실제 Excel 수식 해석은 미검증이며 boolean·선행 공백 헤더는 helper 검증만 있으므로 다음 회차에서 참고하세요.
- [러너 07:49] review approved — 리뷰 승인 (risk=low)
- [러너 07:49] pr created — https://github.com/hkjang/DartFly/pull/11
- [러너 07:54] ci passed — 검사 3개 모두 success
- [러너 07:54] merge done — e6bd782
- [러너 07:55] release missing — 릴리즈 결과 없음/손상: missing
