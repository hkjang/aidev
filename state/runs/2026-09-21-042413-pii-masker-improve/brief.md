- 과제: 재기동 후 미완료 job의 잔여 결과 파일 다운로드 차단 (가치 4 / 위험 2 / 작업량 M)
- 왜: `internal/jobs/store.go:Store.load`는 모든 상태에서 `output_*`를 `OutputPath`로 복구한 뒤 queued/running을 failed로 바꾸며, HTTP 조회·이력·결과 핸들러는 상태가 아니라 경로 유무만 보므로 실패한 작업의 잔여 파일을 정상 다운로드로 공개한다. 완료 기록이 확인된 작업만 결과를 제공하게 하면 결과 쓰기 도중 중단되거나 완료 상태 영속에 실패한 작업의 불완전한 파일을 클라이언트가 정상 결과로 받는 일을 막을 수 있다.
- 수용 기준:
  1) 디스크에 queued/running job.json과 output_*가 함께 남은 상태에서 기동하면 기존대로 failed/job_interrupted가 되고 UpdatedAt은 유지되며, 작업 조회와 이력에 download_url이 없고 결과 GET·HEAD는 404를 반환한다. 이미 failed인 기록과 잔여 output_*의 조합도 동일하게 다운로드를 막는다.
  2) completed 기록과 결과가 있으면 조회·이력의 다운로드 URL, GET 본문, HEAD 크기·무본문, Range 동작이 유지된다. 결과 파일이 없으면 기존 404를 유지한다. 차단 응답도 no-store/nosniff를 유지하고, 잔여 파일·입력·기록은 기존 보존 정책에 맡기며 즉시 삭제하지 않는다.
  3) 실제 디스크 fixture → app.New → jobs.New/Store.load → 실제 HTTP 리스너를 통과하는 통합 테스트로 위의 상태별 조회·이력·GET·HEAD를 검증한다. running→failed 전환 직후뿐 아니라 두 번째 로드에서도 차단을 확인하고, 기존 completed 다운로드 테스트를 함께 통과시킨다. 저장소 단위 테스트는 미완료 OutputPath 비복구와 시간 유지, 완료 OutputPath 복구를 증명한다.
- 건드릴 파일:
  - `internal/jobs/store.go:Store.load` — 입력 복구는 유지하고 OutputPath는 completed인 기록에서만 복구한다. 미완료 레코드의 Metadata.Output.DownloadURL도 비워 오래된 저장 메타데이터가 URL을 되살리지 못하게 한다. 중단 표시·UpdatedAt·persistLocked·삭제 조건은 바꾸지 않는다.
  - `internal/httpapi/server.go:handleGetJobResult, handleGetJob, handleHistory` — 결과 공개 조건을 completed && OutputPath != ""로 일치시킨다. 결과 핸들러는 불충족 시 기존 job_result_not_found 404 경로를 사용한다. 조회·이력은 불충족 시 기존 DownloadURL도 비운다. 필요하면 작은 공통 조건 헬퍼를 쓰되 큰 리팩터는 하지 않는다. handleCreateJob의 기존 202 응답 계약은 이번 범위 밖이다.
  - `internal/jobs/store_test.go` — 기존 `seedJob`, `TestLoadKeepsTheTimestampOfInterruptedJobs`, `TestLoadPersistsInterruptedJobs`, `TestDeleteExpiredIsAppliedToReloadedJobs`를 참고해 새 회귀 테스트 추가.
  - `internal/httpapi/integration_test.go` — `startAppServerWithConfig`의 customize 콜백은 app.New 이전에 호출되므로 여기서 저장 디렉터리를 준비한다. `seedStoredJobFiles`는 completed를 고정하므로 상태 지정 helper를 새로 만들거나 기존 호출과 호환되게 확장한다. `fetchHistory`, `TestJobResultAnswersHeadProbes`, `TestJobResultServesRangeRequests`를 재사용/참고한다.
  - `README.md` — 재기동으로 실패한 작업은 잔여 결과 파일이 있어도 다운로드하지 않는다는 문장을 종료/보존 정책에 추가.
- 검증 명령: `go test -count=1 ./internal/jobs ./internal/httpapi`; `go test -count=1 ./...`; `go vet ./...`; `go build ./...`; `go test -race -count=3 ./internal/jobs ./internal/httpapi` (마지막 race 명령은 구현 후 실행할 권장 명령, 정찰에서는 미실행). 수정 파일에 gofmt 적용. 가능하면 수정 전 새 테스트 실패 → 수정 후 통과를 확인한다.
- 위험과 피할 것: 결과 파일이 존재한다는 이유만으로 completed로 승격하지 말 것. 파일 쓰기 원자화·fsync·Save 실패 정책·재개·옵션 영속·보존 기간 변경을 묶지 말 것. load의 고아 삭제 조건 및 auth/allow-host, 의존성, workflows는 건드리지 않는다. 잔여 파일은 검증 fixture로만 준비하고 실제 프로세스 강제 종료 타이밍에 기대는 flaky 테스트는 만들지 않는다. GET만 막고 조회/이력/HEAD를 빠뜨리지 말 것. 이번 문제는 불완전 파일 공개이며 원문 개인정보 유출이 실제 발생했다는 주장은 하지 않는다.
- 차선 후보: internal/config의 Load 경유 환경변수 정규화·기본값 테스트 보강 (가치 2 / 위험 1 / 작업량 S) — 1순위가 이미 다른 변경으로 해결된 경우에만 선택; t.Setenv를 쓰는 테스트에는 t.Parallel을 붙이지 않는다.

근거와 실행 결과:
- 기준 HEAD e2528ae. `Store.writeFile`은 os.WriteFile로 output_*를 직접 기록하고 `Service.runJob`은 결과 저장 다음에 completed 기록을 Save한다. 크래시/쓰기 실패에서 잔여 파일이 생길 가능성은 이 순서로 판단했으며, 실제 디스크 가득 참이나 SIGKILL 장애는 주입하지 않았다.
- 정찰은 실행 중인 러너 대신 재기동 시점 디스크 상태를 합성하고 `go run ./cmd/pii-masker`로 실제 서버를 시작했다. running 기록 + `PARTIAL-SYNTHETIC-OUTPUT` 파일에서 조회/이력은 failed와 download_url을 함께 반환하고, 결과 GET 200으로 해당 바이트를 반환, HEAD 200을 확인했다. 증거: `assets/restart-probe/observed.json` (이 과제서와 같은 run 디렉터리 기준).
- 정찰 검증: `go test -count=1 ./...`, `go vet ./...`, `go build ./...` 모두 통과. 저장소 수정·커밋 없음.
- 실행 계획/견적: 회귀 fixture 및 실패 확인 12분 → 복구·HTTP 조건 수정 10분 → 전체/race 검증 10분 → 문서/검토 5분, 예비 8분 = 총 45분. 새 정책 논쟁이나 저장 형식 변경이 필요하면 범위를 늘리지 말고 차선으로 전환한다.
- 스킬 가용성: 요청된 pmo:estimating-and-contingency, technology:implementation-planning, technology:solution-exploration은 이 세션의 도구 목록에 Skill/skills.list/skills.read가 없고 로컬 .codex/.claude 및 aidev/cache 검색에서도 해당 SKILL.md를 찾지 못했다. 따라서 해당 스킬의 절차·반환 형식은 미확인; 사용자 정찰 절차와 위의 대안 비교·견적·예비 시간을 적용했다.
