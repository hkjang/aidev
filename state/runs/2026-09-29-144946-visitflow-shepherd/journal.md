# PR 처리기 노트 2026-09-29-144946-visitflow-shepherd — visitflow PR #28
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-09-29-123231-visitflow-improve)
# 회차 노트 2026-09-29-123231-visitflow-improve — visitflow
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 12:32] base pinned — main@9b66607
- [러너 12:32] autonomy release — 

## 정찰 노트
- 선택: QR 스캔·로비 현황 기준 정보 실패 안내/재시도. 현 코드에서 결함이 보이고 VisitFormPage 선례가 있어, 재현 전인 응답 역전·미머지 메일/MCP 후보보다 범위와 위험이 작다.
- 초안을 먼저 저장한 뒤 두 프로덕션 파일·실제 E2E 중심으로 보완했다. 정찰 Go 전체 PASS이나 DSN 없어 통합 SKIP; 화면 실패→복구·scope fixture·45분 내 환경 준비는 미확인이다.
- 주의: Scanner의 빈 lobbyId가 전체 스캔을 막는다는 과거 설명은 부정확. QR 버튼 조건·카메라 정리·Lobby SSE를 보존하고 refError를 작업 오류와 분리할 것.
- 세 headcount 스킬은 전용 Skill 도구 부재로 로컬 원문을 읽어 적용했다. 과제서에 대안·단계별 증명·자동 점검·30~40분 조건부 추정과 예비를 명시했으며 저장소 코드/커밋은 변경하지 않았다.
- [러너 12:37] scout done — QR 스캔·로비 현황에서 기준 정보 로드 실패를 알리고 재시도 제공 (가치 3 / 위험 1 / 작업량 M)

## 구현 노트
- Scanner/Lobby에 별도 기준 정보 오류·로딩 상태와 재시도 Alert를 추가했다. 프로덕션 2개 + E2E 1개 + 가이드 1개 파일만 변경.
- 수정 전 실제 서버에서 복구 버튼 부재로 4개 E2E 실패 및 Scanner pageerror 관찰 → 집중 4개 통과 → 수정 전 바이너리 재기동 시 두 화면 재실패 → 최종 전체 14개 통과(45.1s). 로그: assets/recovery/red.log, reverted-red.log, e2e-final.log.
- 실제 API로 siteScope 제한 계정·로비 0/1/2개를 만들고 성공은 route.continue(). 공용 오류 닫기, 재실패, 지연 중 disabled·3회 추가 클릭/요청 수 3건, QR·badge·result, 검색·탭·빈 필터 값·목록 보존 및 SSE 후 현재 필터 재조회까지 검증했다.
- npm ci/lint/test(60개)/build, DB DSN 지정 go test ./... -count=1(app 54.516s), go vet/build, git diff --check 통과. 임베드 원본 복원, UI/테스트 산출물 제거, 전용 서버·PostgreSQL 컨테이너 종료 완료.
- 검증 못 한 것: 실제 카메라 하드웨어는 사용하지 않았고 기존 stopCamera cleanup을 보존했다. PDF·릴리즈·서버/인증 변경은 과제 범위 밖이라 하지 않았다.
- 다음 역할 주의: E2E는 실제 dist 임베드 서버·PostgreSQL·Chromium이 필요하다. 중간에 브라우저 실행 파일 소실로 전체 실행이 시작 실패해 assets/recovery/browsers에 재설치 후 최종 통과했다. MUI 로비 select는 선택 뒤 접근성 이름에 값이 붙고 빈 값은 텍스트가 비므로 DOM 값으로 검증한다.
- 커밋: 8dc4180 fix(web): recover reference data in scanner and lobby pages. push·릴리즈 미실행.
- [러너 12:53] brief accepted — 채택 — 현재 코드와 실제 실패가 과제서의 원인과 일치했고, 네 파일 범위에서 수용 기준을 실제 서버 브라우저로 검증�
- [러너 12:53] verify passed — 검증 7개 통과 (auto)
- [러너 12:54] review missing — 리뷰 결과 없음/손상: missing — 보류
- [러너 12:54] pr created — https://github.com/hkjang/visitflow/pull/28
