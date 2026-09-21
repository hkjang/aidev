# 회차 노트 2026-09-21-215509-pii-masker-improve — pii-masker
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 21:55] base pinned — main@3487570
- [러너 21:55] autonomy release — 

## 정찰 노트
- UTF-8 진단 절단을 선택: 실제 HTTP에서 한글 손상 재현, 공통 함수의 작은 수정으로 세 관찰 경로 개선. 설정 테스트보다 출력 효과가 명확하고 슬롯/재개/영속 변경보다 위험이 낮다.
- 전체 test/vet/build 통과. 이전 결과 다운로드 차단은 완료로 갱신하고 오래된 프로필을 교정했다. 기존 9개 후보 유지, 신규 2개와 완료 3개를 기록했다.
- 미확인: Windows 실행·race·이미 잘못된 UTF-8 입력의 처리 정책. 요청 스킬 3종은 도구/로컬 검색에서 미발견으로 절차·형식 미확인.
- 주의: JSON 유효성만 검사하면 U+FFFD 손상을 놓친다. 오류 detail과 성공 debug.body를 실제 app.New 배선으로 검증하고 RawBody/인증/8MB 상한은 바꾸지 말 것.
- [러너 21:59] scout done — 업스트림 진단 문자열을 UTF-8 경계에서 절단 (가치 2 / 위험 1 / 작업량 S)

## 구현 노트
- 7511062: truncateString 절단 위치를 UTF-8 문자 시작까지 후퇴시켜 기존 바이트 상한·말줄임·TrimSpace·무제한 의미를 유지했다.
- 단위 경계 테이블과 실제 app.New/HTTP 통합 500 연결 점검·500 mask·200 mask debug에서 수정 전 실패, 수정 후 통과를 확인했다.
- 오류 detail 279바이트, 성공 debug.body 16,382바이트의 정확한 접두부+말줄임 및 U+FFFD 부재를 검사하며 completed/applied_regions=0/원본 PNG 파일 파트도 확인한다.
- 검증: 지정 패키지·전체 go test -count=1, go vet ./..., go build ./..., gofmt 무출력, git diff --check 통과.
- 확신 없는 곳·검증 못 한 것: Windows·race 미실행; 요청 technology 스킬 3종 및 Skill 도구 미발견으로 절차·반환 형식 미확인.
- 일부러 하지 않은 것: 원래 잘못된 UTF-8/읽기 한도 절단의 교정, 오류 debug 전달 개선, RawBody/ParsePayload·인증·라우팅·저장 계약 변경은 범위 밖.
- 다음 역할 주의: engine.debug.response는 JSON 문자열이므로 재디코딩한 body를 검사해야 하며 JSON 유효성만으로는 손상을 찾지 못한다.
- [러너 22:03] brief accepted — 채택 — 바이트 인덱스 절단이 현재 코드에 남아 있었고 실제 HTTP 세 경로의 새 회귀 테스트로 진단 손상을 재현했다.
- [러너 22:03] verify passed — 검증 3개 통과 (policy)
