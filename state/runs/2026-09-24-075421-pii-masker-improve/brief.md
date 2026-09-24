# 과제서 (2026-09-24, 기준 커밋 5d43c34)

- 과제: `POST /v1/jobs` 응답에서 아직 결과가 없는 작업의 `download_url` 제거 (가치 3 / 위험 1 / 작업량 S)

- 왜: `internal/httpapi/server.go:187` `handleCreateJob`이 `job.Metadata.Output.DownloadURL`을 조건 없이 채워서, 방금 접수돼 `queued` 상태인 작업의 202 응답에 아직 404를 돌려주는 결과 URL이 들어간다. 같은 값을 읽는 다른 두 경로(`handleGetJob` server.go:209, `handleHistory` server.go:282)는 2026-09-21 수정 이후 `status == "completed" && OutputPath != ""` 를 요구하므로 세 경로의 계약이 어긋나 있고, 내장 Playground는 `renderMetadata`(`internal/httpapi/static/index.html:955`)가 POST 응답의 `download_url`을 그대로 받아 작업 생성 즉시 "파일 다운로드" 링크를 그리므로 사용자는 눌러도 `job_result_not_found` JSON만 받는 죽은 링크를 보게 된다. 드레인 중(`service.CreateJob` → `startJobRunner` false → `markJobInterrupted`, service.go:326~333)에는 `failed`/`job_interrupted` 작업이 202 + `download_url`로 나가는데, 이것은 README 79행의 명시된 계약("비동기 작업은 `failed` 상태에 `download_url` 없이 남습니다")을 직접 위반한다.

- 수용 기준:
  1) 슬롯이 차 있어 `queued`로 남는 작업의 `POST /v1/jobs` 202 응답 JSON에 `output.download_url`이 없다(`omitempty`이므로 키 자체가 없음). 그 작업이 완료된 뒤의 `GET /v1/jobs/{id}` 응답에는 기존과 동일한 URL이 들어온다.
  2) 완료된 작업의 `GET /v1/jobs/{id}`·`GET /v1/history`·`GET/HEAD /v1/jobs/{id}/result` 동작은 변하지 않는다(기존 테스트 무수정 통과).
  3) 새 회귀 테스트가 프로덕션 배선(`app.New` → 실제 리스너 + mock 업스트림)을 지나며, 수정 전에는 "생성 응답에 download_url이 있다"로 실패하고 수정 후 통과한다. 즉 POST·GET 두 경로가 같은 상태에 대해 같은 값을 읽는 것을 end-to-end로 증명한다.

- 건드릴 파일:
  - `internal/httpapi/server.go:164` `handleCreateJob` — 187행의 무조건 대입을 `handleGetJob`(209~212행)과 같은 조건(`job.Metadata.Status == "completed" && job.OutputPath != ""`)으로 맞추고, 아니면 `job.Metadata.Output.DownloadURL = ""`로 비운다. 갓 만든 작업은 항상 후자에 해당하지만, 세 경로가 같은 규칙을 읽게 두는 편이 나중에 규칙이 바뀔 때 또 어긋나지 않는다. 왜 URL을 주지 않는지(결과가 없으므로 404이고, 완료 후 `GET /v1/jobs/{id}`가 준다) 한 줄 주석을 남길 것.
  - `internal/httpapi/integration_test.go` — 새 테스트 1개 추가. 기존 `newGatedUpstream()`(1156행)과 `startAppServerWithUpstream(t, gated, customize)`(1461행)을 그대로 쓰고 `cfg.Limits.MaxConcurrentJobs = 1`로 두 번째 작업을 `queued`에 묶은 뒤, 두 번째 POST 응답에 `Output.DownloadURL == ""`인지 본다. 그 다음 `releaseGate()`로 풀고 `GET /v1/jobs/{id}`를 폴링해 `completed`가 되면 `download_url`이 `/v1/jobs/{id}/result`로 채워지고 실제로 200을 주는지까지 확인한다(폴링 루프는 230~262행의 기존 패턴 그대로).
  - `README.md` — 비동기 작업 문단(37행 근처) 또는 79행 옆에 "`download_url`은 결과 파일이 생긴 완료 작업에만 실린다"는 한 문장. 기존 한국어 서술 톤 유지.

- 검증 명령 (이 저장소에서 실제로 도는 것, 기준 커밋에서 전부 통과 확인함):
  - `go test -count=1 ./internal/httpapi` (0.6초 수준)
  - `go test -count=1 ./...`
  - `go vet ./...` / `go build ./...`
  - `gofmt -l ./cmd ./internal` (무출력이어야 함) / `git diff --check`
  - `go test -race -count=3 ./internal/httpapi ./internal/service`
  - 고치기 전에 새 테스트가 실제로 실패하는 것을 먼저 보고, 고친 뒤 통과를 확인할 것.

- 위험과 피할 것:
  - `handleGetJob`·`handleHistory`의 기존 조건을 건드리지 말 것. 2026-09-21 회차가 세운 계약이며 회귀 테스트(`integration_test.go:1302` 근처의 stale download URL 픽스처, `1742`행의 이력 URL 검증, `internal/jobs/store.go:184`의 복구 시 초기화)가 여기에 걸려 있다.
  - `service.CreateJob`/`markJobInterrupted`/`startJobRunner`(드레인·WaitGroup 경로)는 건드리지 말 것. 이번 변경은 HTTP 표현 계층 한 곳이면 충분하다.
  - `core.FileDescriptor.DownloadURL`은 `omitempty`이므로(`internal/core/types.go:11`) 빈 값이면 키가 사라진다. 테스트는 "빈 문자열"이 아니라 디코드 후 필드가 비어 있는지로 보면 되고, 굳이 원시 JSON에 키가 없는지까지 단언할 필요는 없다.
  - UI(`static/index.html`)는 `renderMetadata`가 `if (metadata.output?.download_url)`로 방어하고 있어 서버가 값을 비워도 추가 수정 없이 링크만 사라진다. 확인했으므로 UI 코드는 손대지 말 것.
  - 보호 경로(auth·migrations·workflows)는 이 저장소에 없고, 이번 변경은 upstage 인증/allow-host에 닿지 않는다.
  - 이 저장소 관례대로 커밋 메시지는 영어 명령문 한 줄.

- 차선 후보: `internal/config` 순수 함수의 `t.Setenv` + `Load()` 경유 테이블 테스트 (가치 2 / 위험 1 / 작업량 S). `config_test.go`는 지금도 서버 timeout 3개뿐이고 `normalizeAllowHosts`/`normalizeEndpointURL`/`normalizePIILang`/`normalizePIISchema`/`envInt`/`envNonNegativeInt`/`envBool`이 무검증이다. `t.Setenv`를 쓰는 테스트는 병렬화하지 말 것.
