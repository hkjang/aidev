- 과제: 환경변수 시간 단위 변환의 time.Duration 오버플로 방어 (가치 3 / 위험 2 / 작업량 M)
- 왜: `config.Load()`는 초·시간 설정의 양수 정수를 곱할 때 상한을 검사하지 않아, `PII_MASKER_JOB_RETENTION_HOURS=2562048`이 음수 duration으로 변하고 `StartRetentionSweeper`가 원본 파일 정리를 꺼 버린다. 표현 불가능한 시간 값은 기존 잘못된 입력 처리처럼 각 기본값으로 복귀시켜 보존 정리와 동기 대기 정책을 유지한다.
- 수용 기준:
  1) 대상은 `READ_HEADER_TIMEOUT_SECONDS`(15초), `IDLE_TIMEOUT_SECONDS`(60초), `SHUTDOWN_TIMEOUT_SECONDS`(45초), `DEFAULT_TIMEOUT_SECONDS`(30초), `SYNC_QUEUE_WAIT_SECONDS`(10초), `JOB_RETENTION_HOURS`(24시간)의 PII_MASKER_ 환경변수 여섯 개다. 곱셈 전에 `MaxInt64 / unit`을 넘는지 검사하고 초과 시 각 기본값을 사용한다. 초 최대 9223372036, 시간 최대 2562047은 정확히 유지한다(64비트 환경). int64를 넘는 문자열도 기본값으로 복귀한다.
  2) 미설정/공백/음수/비숫자 기본값, 일반 양수 유지, 앞뒤 공백 제거를 보존한다. 명시적 0은 retention·sync wait에서만 0으로 유지하고 다른 네 timeout에서는 기본값이다. envInt/envNonNegativeInt의 페이지·크기·동시성 호출자 계약은 바꾸지 않는다.
  3) `config.Load()` 경유 테이블 테스트가 여섯 설정의 상한/상한+1/일반값/0/잘못된 입력을 검증한다. 곱한 결과가 음수인지 보는 사후 검사로 때우지 않는다: 양수로 다시 감기는 큰 값(예: 초 18446744074, 시간 5124096)도 기본값이어야 한다. 32비트 Atoi 범위 차이를 고려해 duration 전용 파서는 ParseInt(...,10,64) 등으로 일관성을 확보한다.
  4) 실제 배선의 회귀 테스트: `PII_MASKER_JOB_RETENTION_HOURS=2562048` → `config.Load()` → `app.New()` → 실제 httptest HTTP 서버에서 48시간 된 completed 작업 디렉터리가 삭제되고 GET `/v1/jobs/stale-job`은 404, 최근 작업 파일은 유지되고 GET은 200이다. 설정값을 테스트에서 직접 24시간으로 주입하지 않는다. 명시적 0의 보존은 기존 service 테스트와 Load 테이블로 함께 확인한다.
  5) README 환경 변수 절에 표현 범위를 넘는 시간 설정은 기본값으로 복귀함을 한 문장으로 기록한다. 기존 테스트 전체가 통과한다.
- 건드릴 파일:
  - `internal/config/config.go:Load,envSeconds` — duration 전용 범위 검사 헬퍼를 두고 여섯 경로를 연결. 양수 전용과 0 허용 계약을 명시적으로 구별한다. 프로덕션 변경은 이 파일 1개로 한정.
  - `internal/config/config_test.go` — 기존 TestLoadServerTimeoutOverrides/TestLoadServerTimeoutRejectsNonPositiveValues 양식으로 Load 경유 테이블 추가. t.Setenv + t.TempDir, 병렬화 금지, 관련 환경변수는 명시적으로 비워 외부 환경 영향 차단.
  - `internal/httpapi/integration_test.go` — TestExpiredJobFilesArePurgedOnStartup를 참고해 새 회귀 추가. `startAppServerWithConfig`의 customize 안에서 STORAGE_DIR를 기존 임시 root로, UPSTAGE_BASE_URL을 기존 테스트 upstream으로 설정한 후 Load 결과 전체를 `*cfg`에 대입한다. `seedStoredJobFiles`로 root/jobs 아래 stale/fresh를 만들면 helper가 이후 app.New를 호출한다. `waitForCondition` 후 파일 존재와 두 HTTP 상태를 확인한다. 새 테스트는 t.Parallel 금지. 기존 fixture·helper의 전체 계약 변경 불필요.
  - `README.md:환경 변수` — 초과 입력 정책 한 문장.
- 검증 명령: `go test -count=1 ./internal/config ./internal/httpapi`; `go test -count=1 ./...`; `go vet ./...`; `go build ./...`; `go test -race -count=1 ./internal/config ./internal/httpapi`; `gofmt -l ./cmd ./internal`; `git diff --check`.
- 위험과 피할 것: auth/allow-host/리다이렉트, jobs 삭제·저장 로직, 마이그레이션·워크플로·릴리즈·Docker·vendor 변경 금지. MaxFileSizeBytes 오버플로는 별도 과제로 남긴다. app.orDefault와 upstream.httpClient는 음수 timeout을 이미 보정하므로 모든 timeout이 무력화된다고 주장하지 않는다. 실제 확인된 영향은 retention 정리 비활성화와 sync wait 즉시 거절이다. 하드코딩 config/FakeTask/가짜 Store로 회귀를 증명하지 말 것. 파일 6개 상한 안에서 프로덕션 1개 + 테스트 2개 + 문서 1개로 끝낸다.
- 차선 후보: internal/config 나머지 정규화 함수의 Load 경유 테이블 테스트 — normalizeAllowHosts/PIILang/PIISchema/AuthMode/envBool. 1순위가 현재 HEAD에서 이미 해결돼 재현이 실패할 때만 선택.

근거와 정찰 검증 (main@6badf77)
- 읽은 config.go의 Load 3개 envSeconds와 Upstage.Timeout/SyncQueueWait/JobRetention의 직접 곱셈에 상한 검사가 없다. service.New는 JobRetention/SyncQueueWait를 그대로 보관하고 StartRetentionSweeper/PurgeExpiredJobs는 <=0이면 반환, acquireSyncSlot은 <=0이면 즉시 busy를 반환한다.
- 저장소 수정 없이 Go overlay로 기존 config_test.go에 테스트를 덧붙여 Load를 실행했다. 명령: `go test -overlay=/mnt/c/Users/USER/projects/aidev/state/runs/2026-09-29-100231-pii-masker-improve/assets/duration_overlay.json -run TestScoutDurationOverflowProbe -v ./internal/config`. 실제 FAIL: retention=-2562047h34m33.709551616s, sync_wait=-2562047h47m16.709551616s. overlay 파일은 재현 자료이며 제품에 복사하지 않는다.
- 기존 `go test -count=1 ./...`, `go test -count=1 ./internal/config`, `go vet ./...`, `go build ./...`, `git diff --check` 통과. 새 HTTP 회귀와 race 검증은 구현자가 수행해야 하며 정찰에서는 미실행. Docker/32비트 실행 미확인.

대안 비교와 선택 근거 (solution-exploration)
- 선택: config에서 단위별 상한을 검사하고 기본값으로 복귀. 기존 invalid-input 정책과 같고 여섯 소비 경로에 한 번에 적용 가능하다.
- 소비자마다 음수 보정: 설정이 이미 양수로 감긴 경우를 잡지 못하고 app/service/upstage에 중복 정책을 만든다. 탈락.
- Load 전체를 오류로 종료: 잘못된 설정을 강하게 알리지만 기존 fallback 계약을 바꾸어 배포를 막을 수 있다. 이번 범위에서 탈락.
- 현상 유지/문서만 경고: 발생은 극단 설정에 한정되지만 재현된 무기한 보관을 그대로 남기므로 탈락.
- 가장 큰 가정: 표현 불가능한 설정도 기존 invalid-input처럼 fallback하는 것이 맞다. 일반 사용자가 의도한 명시적 0은 반드시 유지한다.

실행 계획 (implementation-planning; 모두 미착수)
1. config_test.go 경계 회귀와 HTTP 보존 회귀를 작성하고 `go test -count=1 ./internal/config ./internal/httpapi`로 현재 결함에 의한 실패를 확인·기록한다. 이 시점의 실패는 의도된 회귀 증거이며 별도 커밋하지 않는다. 사람 확인 불필요, 실패 원인이 다르면 과제서를 수정한다.
2. config.go의 여섯 duration 경로만 수정하고 같은 명령을 재실행해 모두 통과시킨다. 여기서 빌드 가능한 체크포인트를 만든다. 사람 확인 불필요.
3. README 문장을 추가하고 위 전체 검증 명령을 실행해 결과를 기록한다. 범위 확장은 ideas에만 남긴다. 사람 확인 불필요.

작업량 근거 (estimating-and-contingency)
- bottom-up 기본 작업 27~35분: 경계·HTTP 회귀 12~16분, 공통 변환 및 여섯 배선 10~12분, 문서·검증 5~7분. 알려진 불확실성(환경 격리/비동기 sweeper 대기)에 예비 5~8분을 별도로 더해 총 32~43분을 계획한다. 실측 통계 없는 중간 신뢰의 추정 범위이며 보장/P80 값이 아니다.
- 유사 비교: 9/27 설정 호출부+Load 테스트+실제 app 통합 회차와 구조가 유사하나 당시 소요 시간은 미제공이라 수치 추정 근거로 쓰지 않았다. 이번은 duration 여섯 경로라 S 대신 M이다.
- 관리 예비(예상하지 못한 범위)는 배정하지 않는다. 45분을 넘길 사유가 생기면 범위를 늘리지 말고 원인과 미완료 기준을 기록한다. 기준·가정·예비 분리는 읽은 pmo SKILL.md 절차를 따른다. 외부 문헌 수치를 차용하지 않았다.
- 요청한 세 스킬은 전용 Skill 도구가 없어 `/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/{pmo,technology}/skills/.../SKILL.md`를 직접 읽어 적용했다.
