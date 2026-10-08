# PR 처리기 노트 2026-10-08-154945-moyro-shepherd — moyro PR #38
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-08-143828-moyro-improve)
# 회차 노트 2026-10-08-143828-moyro-improve — moyro
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 14:38] base pinned — main@879082e
- [러너 14:38] autonomy release — 

## 정찰 노트
- 선택: 실제 NewRouter/WebSocket/DB를 통과하는 수동 DND·away 보존 테스트. 기존 audience 테스트의 직접 handlers/hub.Register 배선 공백을 메우며, 상태코드·본문 파서 수정 반복을 피한다. 프로덕션 0파일, 테스트 1파일, 40분+5분 예비.
- 후보 15개(기존 12+신규 3)를 재평가했다. 외부 러너 로그 접근은 저장소 범위 밖이라 rejected; 나머지는 pending. 북마크 reorder 주석/구현 불일치는 외부 계약 미확인이라 보류했다.
- 확인: 비DB httpapi/ws 및 소스 크기 검사 통과; 실제 PostgreSQL에서 기존 NewRouter·presence 테스트 3.161s PASS/skip 0. 새 lifecycle 시나리오 자체와 전체 race/웹/릴리즈는 미실행이며 현재 제품 결함을 주장하지 않는다.
- 주의: 비동기 콜백 완료를 ClientCount/sleep으로 추측하지 말고 PUT 이벤트를 소비한 뒤 lifecycle 이벤트를 관찰한다. 회사 스킬 3개 미발견을 brief에 기록. 코드·커밋 변경 없이 초안→최종 brief 및 9일 된 profile을 갱신했다.
- [러너 14:44] scout done — 실제 WebSocket 연결·해제가 수동 DND/away 상태를 보존하는 통합 회귀 테스트 (가치 3 / 위험 1 / 작업량 M)

## 구현 노트
- `c86e957`: 신규 presence_lifecycle_postgres_test.go 1파일(261줄), 프로덕션 변경 0파일. 실제 라우터·auth·HTTP·WebSocket·격리 DB로 수동 DND/away 재접속 보존과 자동 상태 대조군을 검증한다.
- 각 PUT 이벤트를 먼저 소비하고 연결/해제 이벤트 후 payload·본인 GET·DB를 확인한다. 한 reader와 읽기 deadline을 쓰며 ClientCount/고정 sleep/콜백 대체가 없다. 마지막 관찰자 teardown만 DB offline을 폴링하고 hubDone을 기다린 후 DB cleanup이 실행된다.
- 검증: 지정 focused `-race -count=1 -v` 10.999s PASS/skip 0(신규 5.30s), 실제 DSN 전체 `go test -race -p 1 -count=1 ./...` 50패키지 ok/FAIL 0/exit 0(httpapi 71.134s); build/vet/소스 크기/gofmt/diff 통과. 로그는 이 회차 폴더에 저장했다.
- 확신 없는 곳·미검증: 외부 플러그인 아카이브 4개 미제공으로 기존 TestMattermostArchivesInstallRunAndSurviveRestart는 skip(별도 verbose 확인). 웹·릴리즈, 다중 탭·급속 재접속·custom status·guest 범위는 이번에 검증하지 않았다.
- 의도적으로 하지 않음: 정상 동작의 테스트 공백 과제라 인위적 RED·프로덕션 수리·공통 헬퍼 리팩터 없음. 정찰의 후보 15개를 보존하고 선택 항목만 done으로 갱신했다.
- 스킬 정정: 전용 Skill 도구는 없지만 로컬 headcount marketplace의 technology completion-verification/systematic-debugging/test-driven-development SKILL.md 3개를 찾아 읽었다. 실패 재현이 없는 이유와 검증 범위를 원장에 명시했다.
- 다음 역할 주의: MOYRO_TEST_POSTGRES_DSN이 없으면 공통 DB 헬퍼가 skip한다. DSN/토큰을 출력하지 말고 실제 PostgreSQL로 실행할 것. 푸시·태그·릴리즈는 수행하지 않았다.
- [러너 14:54] brief accepted — 채택 — 명시된 실제 배선과 테스트 공백이 현재 코드에 존재했고, 테스트 1파일·프로덕션 0파일로 수용 기준을 충족했�
- [러너 14:55] verify passed — 검증 2개 통과 (policy)
- [러너 14:55] review missing — 리뷰 결과 없음/손상: missing — 보류
- [러너 14:55] pr created — https://github.com/hkjang/moyro/pull/38
