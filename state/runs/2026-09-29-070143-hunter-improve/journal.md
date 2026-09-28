# 회차 노트 2026-09-29-070143-hunter-improve — hunter
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 07:01] base pinned — main@f0f7667
- [러너 07:01] autonomy release — 

## 정찰 노트
- 개수·ID·revision·patch/status 순수 계약 검사를 선택: 프로덕션 0파일, DB 없이 재현 가능하며 auth·tracking·목록 연속 변경보다 위험이 낮다. due_date는 차선으로 분리했다.
- 최종 main은 09-28 기록과 다르게 trim 후 wire 길이를 검사한다(b591b40/e7d2bb2). 같은 잘못된 원문 검사 접근을 재도입하지 말 것.
- 실측: 웹 104 PASS/0 FAIL/0 SKIP, Go 담당자 1개+21서브테스트 PASS. 신규 경계 테스트·DB·빌드·원격 CI는 미확인. 초안 작성 후 근거를 반영해 덮어썼다.
- Skill 도구는 없지만 headcount 로컬 스킬 3개를 발견해 읽고 비교·추정·단계별 명령을 반영했다. 실제 HTTP 원자성 증명으로 과장하지 말고 같은 파일의 실제 타입/함수로 검사할 것.
- [러너 07:06] scout done — validateFindingBulk의 개수·ID·revision·patch·상태 계약을 DB 없는 회귀 테스트로 고정 (가치 3 / 위험 1 / 작업량 S

## 구현 노트
- 실제 요청 타입/validateFindingBulk에 37개 독립 사례 추가; 테스트 1파일(+102줄), 프로덕션 0파일, 담당자 fixture와 기존 테스트 보존.
- 기준선 1개+21하위, 중간 2개+38하위, 최종 지정 go test 2개+58하위 PASS/0 FAIL/0 SKIP (0.028s); go vet ./internal/app 종료 0, gofmt -l 빈 출력, git diff --check 종료 0.
- 실패 재현 없음: 기존 계약 회귀 보강으로 처음부터 PASS. 과제서 예외에 따라 인위적 red/프로덕션 결함 주입 안 함.
- 확신 없는 곳·미검증: DB HTTP 권한·현재 상태 전이·원자적 롤백, 전체 Go 스위트·빌드·원격 CI·배포는 검증하지 않음.
- 일부러 하지 않음: due_date 차선·auth·SQL·웹·버전·원본·릴리즈 변경은 범위 밖. 아이디어 기존 항목 유지, 이번 선택만 done.
- 다음 역할: 이 테스트는 DSN 없이 실행하며 skip이 없어야 함; status 허용 목록 검증을 DB 상태 전이 승인 증거로 해석하지 말 것.
- Skill 도구 미제공으로 로컬 headcount technology completion-verification/systematic-debugging/test-driven-development SKILL.md를 읽어 적용; 별도 버그 원인/수정 주장은 없음.
- [러너 07:08] brief accepted — 채택 — 현재 파일에 담당자 공유 벡터만 있어 지정된 순수 계약 테스트 공백을 확인했고 프로덕션 0파일/테스트 1파일 �
- [러너 07:09] verify passed — 검증 9개 통과 (auto)
- [러너 07:09] review missing — 리뷰 결과 없음/손상: missing — 보류
- [러너 07:09] pr created — https://github.com/hkjang/hunter/pull/15
