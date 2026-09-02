## 2026-09-02
- 선택: 관리자 서버 로그 검색이 화면에 보이는 필드를 찾지 못하던 문제 수정 + `internal/logging` 첫 테스트 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공 — PR https://github.com/hkjang/AgentHub/pull/1 (main 머지됨)
- 요약: 운영 화면은 로그 한 줄에 message와 structured fields를 함께 출력하는데 서버의 `Ring.Entries`는 message와 source만 훑고 있어, 화면에 보이는 agent 이름·에러 문자열을 그대로 입력하면 결과가 0건이었습니다. `matches()`로 필드 키·값까지 검색하도록 고치고, 저장소에서 유일하게 테스트가 0개였던 `internal/logging`에 링 랩어라운드·limit·레벨 필터·검색·capture 핸들러·동시성 테스트 9개를 추가했습니다. 검증: go vet, go test -race ./cmd/... ./internal/..., web npm ci+lint+build 모두 통과.
- 보류 아이디어:
  - dlp.notPhone의 앞 두 조건이 죽은 코드이고 0으로 시작하는 실제 계좌번호를 전부 놓침 (3/2/S)
  - korean.EndsInConsonant가 괄호·따옴표로 끝나는 값에서 조사를 잘못 고름 (2/2/S)
  - captureHandler.WithGroup이 그룹 이름을 버려 서로 다른 그룹의 같은 키가 충돌 (2/1/S)
  - runInfoCommand가 version 외 인자를 run()으로 흘려보내 `agenthub --help`가 DB 오류로 실패 (2/1/S)
- 릴리즈: v0.226.0 (2026-09-02)

## 2026-09-02 (2차)
- 선택: DLP 계좌번호 검출기가 0으로 시작하는 계좌번호를 전부 놓치던 문제 수정 (가치 4 / 위험 2 / 작업량 S)
- 결과: 성공 — 커밋 4e240c5 (auto/2026-09-02-1730)
- 요약: `notPhone`이 계좌와 전화번호를 "0으로 시작하지 않을 것"으로 갈랐는데, 앞 두 조건(`!HasPrefix("01")`, `!HasPrefix("02")`)은 뒤의 `!HasPrefix("0")`에 이미 먹혀 죽은 코드였고 살아있던 조건은 틀린 판정이었습니다 — 기업은행·우체국·상당수 국민은행 계좌가 0으로 시작하므로 계좌번호를 '차단'으로 설정한 사이트가 그 은행들만 조용히 통과시키고 있었습니다(찾지 못한 것은 감사 로그에 아무 흔적도 남지 않는 실패). 체크섬이 없는 두 값을 실제로 가르는 것은 자릿수 묶음이라, 한국 전화번호 모양(0으로 시작하는 2~4자리 지역번호 - 3~4자리 국번 - 정확히 4자리, 세 묶음)일 때만 계좌에서 제외하도록 고쳤습니다. `internal/dlp`는 런타임 base 이미지 소스라 BASE_VERSION도 0.16.0으로 올렸습니다(5곳). 검증: 0으로 시작하는 계좌 3종·전화번호 4종·날짜 리터럴 테스트 추가, go vet, go test -race ./cmd/... ./internal/..., web npm ci+lint+build, `scripts/release-catalog-images.sh check-versions` 모두 통과.
- 보류 아이디어:
  - 사업자등록번호가 계좌번호로도 중복 집계됨(220-81-62517이 두 등급으로 보고되어 운영자가 읽는 건수가 부풀려짐) (3/2/S)
  - korean.EndsInConsonant가 괄호·따옴표로 끝나는 값에서 조사를 잘못 고름 (2/2/S)
  - captureHandler.WithGroup이 그룹 이름을 버려 서로 다른 그룹의 같은 키가 충돌 (2/1/S)
  - runInfoCommand가 version 외 인자를 run()으로 흘려보내 `agenthub --help`가 DB 오류로 실패 (2/1/S)
- 릴리즈: v0.227.0 (2026-09-02)
