## 2026-09-02
- 선택: 관리자 서버 로그 검색이 화면에 보이는 필드를 찾지 못하던 문제 수정 + `internal/logging` 첫 테스트 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공 — PR https://github.com/hkjang/AgentHub/pull/1 (main 머지됨)
- 요약: 운영 화면은 로그 한 줄에 message와 structured fields를 함께 출력하는데 서버의 `Ring.Entries`는 message와 source만 훑고 있어, 화면에 보이는 agent 이름·에러 문자열을 그대로 입력하면 결과가 0건이었습니다. `matches()`로 필드 키·값까지 검색하도록 고치고, 저장소에서 유일하게 테스트가 0개였던 `internal/logging`에 링 랩어라운드·limit·레벨 필터·검색·capture 핸들러·동시성 테스트 9개를 추가했습니다. 검증: go vet, go test -race ./cmd/... ./internal/..., web npm ci+lint+build 모두 통과.
- 보류 아이디어:
  - dlp.notPhone의 앞 두 조건이 죽은 코드이고 0으로 시작하는 실제 계좌번호를 전부 놓침 (3/2/S)
  - korean.EndsInConsonant가 괄호·따옴표로 끝나는 값에서 조사를 잘못 고름 (2/2/S)
  - captureHandler.WithGroup이 그룹 이름을 버려 서로 다른 그룹의 같은 키가 충돌 (2/1/S)
  - runInfoCommand가 version 외 인자를 run()으로 흘려보내 `agenthub --help`가 DB 오류로 실패 (2/1/S)
