# 회차 노트 2026-09-22-000433-ptium-improve — ptium
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 00:04] base pinned — main@905a9c5
- [러너 00:04] autonomy release — 

## 정찰 노트
- 생략 XLSX 행의 출처 수정 선정: onScreen의 index 기록을 기존 placement로 바로잡으면 되어 메일 동시성·인증 e2e보다 국소적이다.
- 신규 실행 재현은 미실시; 코드로 경로 확인. 무효 r의 배열 index 대체는 제안 계약이며 실제 비정상 파일은 미확인이다.
- gridOf에 빈 행을 채우지 말고 숫자 파서·숨김 배선·rangeOf 시작 A1은 유지. 전체 Go race와 vet 통과, DB·웹 검증 미실시.
- 요청 스킬 3종은 도구/로컬에 없어 미적용. 기록의 MCP OAuth가 현 HEAD에는 없음을 프로필·아이디어에 반영했다.
- [러너 00:09] scout done — XLSX에서 생략된 행의 실제 r 좌표를 슬라이드 출처에 반영 (가치 3 / 위험 2 / 작업량 S)

## 구현 노트
- 3c3493f: onScreen의 placement.rows에 실제 r을 전달해 생략 행의 슬라이드 출처를 바로잡음. TrimSpace/Atoi 뒤 1..1048576만 수용하고 나머지는 기존 배열 index로 대체.
- ZIP→Read→ParseSource 24개 사례: 차트/표의 좌표·라벨·수치·순서, 숨김+빈 행, 17개 본문 행의 3장 출처, 최대/무효 r. 수정 전 출처 실패 확인 후 전부 통과.
- go test ./internal/docs → go test -race ./... → go vet ./... 및 git diff --check 통과. 기존 테스트 기대값 변경 없음.
- 확신 없는 곳·검증 못 한 것: 실제 Excel/LibreOffice 출력의 비정상 r 사례 미확인. PTIUM_TEST_DSN 미설정으로 DB 연결 테스트 생략; 웹은 변경 없어 미실행.
- gridOf 빈 행 확장·셀 참조 기반 행 추론·정렬/중복 처리·숫자 파서·범위 시작 A1·버전/릴리즈는 범위 밖이라 그대로 둠.
- 요청 technology:completion-verification/systematic-debugging/test-driven-development는 도구와 로컬 검색에서 없어 미적용. 프롬프트의 TDD/검증 절차는 수행.
- 다음 역할: 무효 r은 마지막 정상 r이 아니라 XML 배열 index로 대체하는 계약이며, 큰 r은 배치 좌표에만 반영한다. 기존 아이디어와 정찰의 신규 2개 후보를 보존하고 선택 항목만 done 처리.
- [러너 00:12] brief accepted — 채택 — 배열 index만 출처에 기록하는 결함이 현재 코드와 신규 ZIP 기반 실패 재현에서 확인되어 지정된 placement 매핑만 �
- [러너 00:12] verify passed — 검증 9개 통과 (auto)
- [러너 00:12] review missing — 리뷰 결과 없음/손상: missing — 보류
- [러너 00:12] pr created — https://github.com/hkjang/ptium/pull/29
