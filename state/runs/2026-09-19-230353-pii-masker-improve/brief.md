# 과제서 2026-09-19 — pii-masker

- 과제: `job.json`을 임시 파일 + `os.Rename`으로 원자적으로 기록 (가치 3 / 위험 1 / 작업량 S)

- 왜: `internal/jobs/store.go:217 persistLocked`가 `os.WriteFile`로 `job.json`을 제자리에서 덮어쓰므로(O_TRUNC 후 write), 그 사이에 프로세스가 죽으면(OOM kill, SIGKILL, 디스크 풀) 빈 파일이나 잘린 JSON이 남는다. 2026-09-17 변경(`load()` → `readJobRecord` 실패 시 `os.RemoveAll`) 이후로는 잘린 기록이 "고아"로 판정되어 **job 디렉터리째(이미 써 둔 마스킹 결과 `output_*` 포함) 삭제**되므로, 완료 직전 `Save`에서 죽은 job은 클라이언트에게 `404 job_not_found`로 사라진다. tmp에 쓰고 rename하면 직전 유효 기록(`running`)이 남아 기동 시 `job_interrupted`(재시도 가능)로 정직하게 표시되고, 고아 판정은 "기록을 한 번도 못 쓴 디렉터리"에만 걸리게 된다.

- 수용 기준:
  1) `Store.Create`/`Store.Save`/`load()`의 interrupted 전환 모두가 같은 디렉터리의 임시 파일(예: `job.json.tmp`)에 먼저 쓰고 `os.Rename`으로 `job.json`을 교체한다. 성공 후 job 디렉터리에는 `job.json`만 있고 임시 파일 잔재가 없다.
  2) 임시 파일 쓰기나 rename이 실패하면 `Save`/`Create`는 오류를 돌려주고, 이전에 있던 `job.json`은 바이트 그대로 남아 `readJobRecord`로 읽힌다(직전 상태 보존). 실패한 임시 파일은 best-effort로 지운다.
  3) 크래시 잔재(`job.json.tmp`에 쓰레기, 옆에 정상 `job.json`)가 있는 디렉터리는 `load()`가 고아로 보지 않고 정상 job으로 올리며, 다음 `Save`가 잔재를 덮어써 없앤다. `firstExistingFile`(`input_`/`output_` 접두어만 봄)은 tmp를 입력·출력으로 오인하지 않는다.
  4) 기존 `TestCreateRemovesTheDirectoryWhenTheRecordCannotBePersisted`(`job.json` 자리에 디렉터리를 둠)가 그대로 통과한다 — rename 대상이 디렉터리면 Linux(EISDIR)·Windows 모두 실패하므로 Create는 여전히 오류를 내고 디렉터리를 치워야 한다. 이 테스트는 **수정하지 말 것**.
  5) 테스트가 증명할 것(`internal/jobs/store_test.go`에 추가, 실제 `Store`를 통해서만):
     - `TestSaveLeavesNoTemporaryFileBehind`: `seedJob` 후 `Save` 몇 번 → `os.ReadDir(jobDir)`에 `job.json`, `input_*`, `output_*`만 있음.
     - `TestSaveKeepsThePreviousRecordWhenTheWriteFails`: `seedJob`(status `running`) 후 임시 파일 경로(`job.json.tmp`) 자리에 **디렉터리**를 만들어 tmp 쓰기를 실패시키고 `Save`(status `completed`) 호출 → 오류 반환, 디스크의 `job.json`을 다시 읽으면 status가 `running` 그대로. (Create의 기존 테스트가 rename 실패 케이스를 맡는다.)
     - `TestLoadIgnoresALeftoverTemporaryRecord`: 정상 job 옆에 `job.json.tmp`에 `{"id":"trunc` 같은 잘린 바이트를 심고 `New(root)` 재로드 → job이 로드되고 디렉터리가 삭제되지 않음; 이어서 `Save` 후 tmp가 사라짐.
     - 이전 구현(`os.WriteFile` 직접)으로 임시 되돌려 첫 번째·세 번째 테스트 중 최소 하나가 실제로 실패함을 확인하고 노트에 적을 것(첫 번째는 이전 구현에서도 통과할 수 있음 — 두 번째 테스트가 판별력 있음: 이전 구현은 tmp 디렉터리가 있어도 `job.json`을 그대로 덮어써 `completed`가 된다).

- 건드릴 파일:
  - `internal/jobs/store.go:persistLocked` — `os.WriteFile(job.json)` → `job.json.tmp`에 `os.WriteFile` 후 `os.Rename(tmp, job.json)`; rename 실패 시 `os.Remove(tmp)` best-effort. tmp 이름은 상수(`recordTempName`)로 빼고 `readJobRecord`/`firstExistingFile`이 그 이름을 절대 읽지 않음을 주석으로 못 박을 것. 함수 상단 주석에 "왜 원자적이어야 하는가"(잘린 기록 = 디렉터리 삭제)를 적을 것.
  - `internal/jobs/store.go:load` 주석(140~154행) — "손상된 job.json은 삭제" 문장에 "기록은 rename으로 교체되므로 손상은 한 번도 기록되지 못한 경우에만 생긴다"를 한 줄 보강.
  - `internal/jobs/store_test.go` — 위 테스트 3개. 기존 헬퍼 `seedJob`, `writeOrphanDir` 재사용.
  - `README.md:40` 보존 정책 문단 — 마지막 문장 끝에 "기록은 임시 파일에 쓴 뒤 교체하므로 상태 변경 도중 프로세스가 죽어도 직전 상태가 남습니다" 한 문장 추가.

- 검증 명령 (저장소 루트 `/home/hkjang/.cache/auto-improve-wt/pii-masker`에서, 오늘 기준 전체 `go test ./...` 약 2초):
  - `gofmt -l ./cmd ./internal` (출력 없어야 함)
  - `go vet ./...`
  - `go build ./...`
  - `go test -count=1 ./internal/jobs/...` (신규 테스트 집중)
  - `go test -count=1 ./...`
  - `go test -race -count=3 ./internal/jobs/... ./internal/service/... ./internal/app/...` (플래키 확인; `internal/app` 통합 테스트가 job 라이프사이클을 실제 배선으로 통과함)

- 위험과 피할 것:
  - `fsync`는 넣지 않는다(성능·범위 밖). 목표는 **프로세스 크래시** 원자성이지 전원 차단 내구성이 아니며, README·주석에도 그렇게만 적을 것. 효과 없는 변경을 넣지 말라는 운영자 지침에 따라, rename 외의 "보험" 코드(백업 파일, 재시도 루프)는 추가하지 말 것.
  - Windows에서 `os.Rename`은 기존 파일을 덮어쓰지만 대상이 열려 있으면 공유 위반으로 실패한다. 런타임에 `job.json`을 읽는 곳은 기동 시 `load()`뿐이므로(`Get`/`List`는 메모리 맵) 문제없음 — 그래도 `Save` 실패 시 `runJob`이 `_ = s.jobStore.Save(job)`로 오류를 버리는 현재 동작은 건드리지 말 것(별도 과제).
  - `internal/service/service.go`의 `CreateJob`/`runJob`/`markJobInterrupted` 흐름, `DeleteExpired`, `readJobRecord`의 고아 판정 로직은 바꾸지 말 것. 이번 변경은 `persistLocked` 한 함수와 테스트·문서로 한정.
  - `s.mu` 잠금 구조는 그대로(모든 호출자가 잠금을 잡고 들어옴). 잠금 안에서 rename하므로 같은 job의 tmp가 동시에 두 개 생길 일은 없다.
  - 테스트는 `t.Parallel()` + `t.TempDir()` 관례를 따를 것. 대역·인터페이스 주입 금지(운영자 지침).
  - 미확인: `internal/app` 통합 테스트가 job 디렉터리 내용물을 직접 열거하는지 보지 못했다 — 만약 `os.ReadDir` 결과 개수를 세는 테스트가 있다면 tmp 잔재가 없으므로 영향 없을 것이나 실행해서 확인할 것.

- 차선 후보: `/v1/jobs/{job_id}/result`에 `HEAD` 허용 — `internal/httpapi/server.go:85`의 `.Methods(http.MethodGet)`에 `http.MethodHead` 추가(gorilla/mux는 GET 전용 라우트에 HEAD를 405로 거절). `http.ServeContent`가 HEAD를 이미 처리하므로 핸들러 변경 없음. `internal/httpapi/integration_test.go`의 기존 완료 job 시나리오를 재사용해 HEAD → 200, `Content-Length`/`Content-Disposition`/`Accept-Ranges` 헤더 존재, 본문 0바이트를 확인. `documentResponse` 래퍼가 HEAD에도 `Cache-Control: no-store`를 붙이는지 함께 검사. README 엔드포인트 목록에 HEAD 표기.
