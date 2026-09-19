# 과제서 — 2026-09-20 releasedock

- 과제: 로그 한도에서 잘리는 마지막 줄을 UTF-8 문자 경계에서 자르기 (가치 2 / 위험 1 / 작업량 S)

- 왜: `simpleRunLogger.append`(backend/internal/server/simple.go:832-850)가 명령 예산이 바닥나는 순간 `payload = payload[:allowed]` 로 바이트 단위로 자르므로, 한글처럼 여러 바이트 문자를 출력하는 스크립트는 저장된 마지막 줄이 깨진 글자(읽을 때 U+FFFD `�`)로 끝난다. `payload` 컬럼은 BYTEA 라 저장은 되지만 SSE(simple.go:1181 `string(payload)`)·목록(simple.go:1283)·내려받기(simple.go:1303 부근) 세 경로가 모두 이 바이트를 그대로 읽어 화면·복사·파일 전부에 깨진 글자가 남는다. 쓰는 지점 한 곳에서 rune 경계까지 되감으면 세 읽기 경로가 동시에 고쳐진다(값을 읽는 경로가 여럿이지만 원천이 하나이므로 한쪽만 고치는 문제가 없다).

- 수용 기준:
  1) 명령 예산의 남은 바이트가 한 문자의 중간에 떨어질 때, 저장된 마지막 줄은 그 문자 앞에서 끝나고 `utf8.Valid` 가 참이다(완전한 문자만 남는다). ASCII 만 있는 줄은 지금과 똑같이 정확히 `allowed` 바이트에서 잘린다.
  2) 되감은 만큼 덜 저장했어도 `simple_runs.log_bytes` 는 **실제로 저장한 바이트 수**를 더한다(지금도 `len(payload)` 를 잘린 뒤에 세므로 그대로 두면 됨), 그리고 `로그 저장 한도에 도달하여…` 안내는 여전히 정확히 한 번 붙는다(`exhausted` 판정은 `take` 가 이미 내렸으므로 되감기가 그것을 바꾸지 않아야 함).
  3) 테스트가 증명할 것: (a) 순수 단위 — 되감기 헬퍼가 `"가나다"` 를 4·5 바이트에서 자르면 `"가"` 로, 6 바이트면 `"가나"` 로, 3 바이트 ASCII 는 그대로, 0 은 빈 값으로, 그리고 애초에 잘못된 바이트(0xFF 등)가 섞인 입력은 최대 3 바이트만 되감고 무한히 뒤로 가지 않는다는 것. (b) 스키마 격리 통합 — `simpleRunLogger{budget: logBudget{command: N, system: ...}}` 로 예산을 작게 잡고 `write("stdout", []byte("이미지 로드\n"))`+`flush()` 뒤 `storedLogLines` 로 읽었을 때 마지막 stdout 줄이 완전한 문자로 끝나고(`utf8.ValidString`), `storedLogBytes` 가 그 줄 길이와 같으며, 뒤에 system 안내 행이 한 번만 있다는 것. 기존 `simple_log_test.go`·`simple_logger_test.go` 의 테스트는 그대로 통과해야 한다. 고치기 전 코드로 되돌려 (b) 가 실제로 실패하는 것(깨진 바이트로 끝남)도 확인해 회차 노트에 적을 것.

- 건드릴 파일:
  - `backend/internal/server/simple.go:append` — `payload = payload[:allowed]` 직전/대신, `allowed < len(payload)` 일 때 `for allowed > 0 && !utf8.RuneStart(payload[allowed]) { allowed-- }` 꼴로 rune 시작까지 되감기(`unicode/utf8` import). `RuneStart` 는 연속 바이트(10xxxxxx)만 거짓이므로 최대 3 바이트만 뒤로 간다. 작은 헬퍼 함수(예: `trimToRuneBoundary(payload []byte, allowed int) []byte`)로 빼면 (a) 테스트가 DB 없이 돈다. `take` 의 예산 회계(`*remaining -= charge`)는 건드리지 말 것 — 되감아 남는 1~3 바이트는 그냥 버린다(예산은 이미 0 이고 다음 줄은 `store=false` 로 버려지므로 실질 차이 없음).
  - `backend/internal/server/simple_log_test.go` — (a) 순수 단위 테스트 추가(기존 `TestLogBudget*` 옆).
  - `backend/internal/server/simple_logger_test.go` — (b) 통합 테스트 추가(`newSimpleBatchFixture`·`seedSimpleRun`·`storedLogLines`·`storedLogBytes` 헬퍼 재사용, `TestSimpleRunLoggerChargesTheCapNotice` 가 같은 모양이니 그 옆에).
  - `docs/simple-mode.md:322` 한도 절 — "한도에 걸려 잘리는 줄은 글자 중간에서 끊지 않는다" 한 항목 추가(선택, 짧게).

- 검증 명령 (이 저장소 CI 와 같은 방식, 도커 PostgreSQL 16 을 세션 전용으로 띄워 DSN 을 채움):
  - `cd backend && gofmt -l . && go vet ./...`
  - `cd backend && TEST_POSTGRES_DSN='postgres://postgres:releasedock-ci-password@127.0.0.1:5432/postgres?sslmode=disable' go test ./internal/server/ -run 'LogBudget|SimpleRunLogger|RuneBoundary' -v`
  - `cd backend && TEST_POSTGRES_DSN=... go test ./...` 와 `cd runner && go test ./...`
  - 웹은 손대지 않으므로 `cd web && npm test -- --run` 은 회귀 확인용으로만.
  - 이 정찰 세션에서는 권한 제한으로 `go vet`·`go test` 를 직접 돌리지 못했다(미확인). 이 main(v0.5.15)에는 `make vet` 타깃이 없으니 위 명령을 직접 쓸 것.

- 위험과 피할 것:
  - `take` 의 예산 회계·`exhausted` 판정·빈 줄 저장 규칙(2026-09-10 회차)을 바꾸지 말 것. 되감기는 `append` 안에서 잘라 낼 바이트 수만 줄이는 것으로 끝낸다.
  - `write` 의 64 KiB 강제 flush(simple.go:875)는 별개 경로 — 한 줄이 64 KiB 를 넘으면 청크 경계에서 두 행으로 갈릴 수 있으나 바이트는 보존되므로 이번 범위 밖. 손대지 말 것.
  - 읽기 경로 세 곳(SSE·목록·내려받기)에 `strings.ToValidUTF8` 같은 보정을 추가하지 말 것 — 원천을 고치면 되고, 읽기 쪽만 고치면 경로마다 다르게 보일 위험이 생긴다(운영자 규칙).
  - 마이그레이션·auth·workflows 는 건드릴 이유가 없다. main 에 없는 `make vet`(auto/2026-09-17-0853)·MCP OAuth(auto/2026-09-18-1213) 브랜치의 파일을 끌어오지 말 것 — 그 PR 들의 머지 여부는 미확인.
  - VERSION 은 릴리즈 세션의 몫. `web/dist` 를 커밋하지 말 것.
  - 효과 없는 변경 금지(운영자 규칙): 한글 출력에서 실제 `�` 가 사라지는 것을 통합 테스트로 보여야 한다.

- 차선 후보: 진행 중 실행의 SSE 로그 중복 제거를 O(n²) 에서 마지막 id 비교로 (가치 2 / 위험 1 / S) — `web/src/pages/simple/SimpleRunDetailPage.tsx:191` 의 `current.some(...)` 을 `current.length === 0 || current[current.length-1].id < parsed.id` 로. 서버는 `id>$2 ORDER BY id`(simple.go:1168) 로만 보내므로 안전. 이 경우 `vitest` 단위 테스트를 `web/src` 의 기존 테스트 관례로 추가하고 `npm test -- --run` 과 `npx tsc -b --noEmit` 으로 검증.
