## 2026-09-29
- 선택: make test에 기존 프런트 회귀 테스트를 포함하기 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: Makefile의 기존 Go test/vet 및 npm ci/typecheck 순서를 유지하고 끝에 npm test를 연결했으며 README 개발 절에 검증 범위 한 문장을 추가했다. Go 1.26.7·Node 25.0.0(24+ 충족)에서 make test exit 0, 기존 프런트 4파일 27/27 통과; 임시 실패 테스트를 넣자 27 pass/1 fail과 make exit 2를 확인하고 삭제 뒤 npm --prefix web test 27/27 재통과했다. git diff --check 통과, 최종 변경은 Makefile·README 두 파일이며 커밋 ed0352e 뒤 작업 트리가 깨끗하다.
- 실패 재현: 수정 전 run 디렉터리에 작성한 check_make_test.py로 실제 make -n test 출력을 검사하여 `AssertionError: make test omits frontend regression tests` (exit 1)를 확인했다. 수정 후 같은 검사 통과, npm test 연결을 잠시 제거하자 같은 실패가 다시 발생했고 즉시 복원했다.
- 보류 아이디어: 연결 성립 후 단절(net.OpError·EOF) 오류 정제 (가치 3 / 위험 2 / S) — 실 DB 동적 타입 확인 필요.
- 보류 아이디어: serviceError substring을 센티널 오류로 점진 변경 (가치 3 / 위험 3 / M) — 패키지별 별도 회차.
- 보류 아이디어: 릴리즈 태그 검증의 npm test 포함 (가치 3 / 위험 2 / S) — 보호 워크플로는 이번 범위 밖.
- 보류 아이디어: VOC 로드맵 현재 상태 명시 (가치 2 / 위험 1 / S) — 1순위 성립으로 차선 미선택.
- 과제서: 채택 — 실제 Makefile의 npm test 누락과 기존 테스트 4파일이 근거와 일치하여 지정된 두 파일만 변경했다.
