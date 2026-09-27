- 과제: `PII_MASKER_MAX_PAGES=0`(페이지 제한 없음)을 설정으로 실제 도달 가능하게 하기 (가치 3 / 위험 1 / 작업량 S)
- 왜: `internal/service/service.go:651`의 `if s.config.Limits.MaxPages > 0 && len(dims) > s.config.Limits.MaxPages` 는 "0이면 페이지 제한 없음"을 명시적으로 구현해 두었는데, 그 값을 만드는 유일한 경로인 `internal/config/config.go:148` 이 `envInt("PII_MASKER_MAX_PAGES", defaultMaxPages)` 를 쓰고 `envInt`(config.go:273-283)는 `parsed <= 0` 이면 조용히 기본값 20으로 되돌린다. 즉 운영자가 `PII_MASKER_MAX_PAGES=0` 을 줘도 아무 경고 없이 20이 적용되고, 프로덕션 코드의 무제한 분기는 설정으로는 절대 도달할 수 없는 죽은 가지다(현재 그 분기를 쓰는 곳은 config를 손으로 만드는 테스트뿐 — `internal/httpapi/integration_test.go:379`, `internal/service/service_test.go`). 고치면 21쪽 이상 PDF를 다루는 운영자가 임의로 큰 숫자를 찍어 넣는 대신 의도를 그대로 표현할 수 있고, `JOB_RETENTION_HOURS=0`(정리 끔)·`SYNC_QUEUE_WAIT_SECONDS=0`(즉시 거절)처럼 이미 `envNonNegativeInt` 로 "0 = 끔"을 지원하는 다른 설정과 계약이 일치한다.
- 수용 기준:
  1) `t.Setenv("PII_MASKER_MAX_PAGES", "0")` 뒤 `config.Load()` 가 `cfg.Limits.MaxPages == 0` 을 준다(현재는 20).
  2) `config.Load()` 로 만든 설정을 그대로 `app.New` 에 넣은 프로덕션 배선에서, 기본 한도(20)를 넘는 페이지 수의 PDF 업로드가 `POST /v1/jobs` 에서 `400 invalid_request`("uploaded pdf exceeds the maximum page count of 20")로 거절되지 않고 접수된다. 손으로 만든 config 구조체가 아니라 반드시 `config.Load()` 를 지나는 경로로 증명할 것 — 이 결함은 config 파서에만 있으므로 config를 직접 채우면 결함이 사라진다.
  3) 회귀 방지: 미설정 → 20, 양수(`"5"`) → 5, 음수(`"-1"`)·파싱 불가(`"abc"`) → 20 으로 폴백하는 것이 테이블 테스트로 고정된다.
  4) 파급 차단: `PII_MASKER_MAX_FILE_SIZE_MB=0`, `PII_MASKER_MAX_CONCURRENT_JOBS=0`, `PII_MASKER_MAX_CONCURRENT_SYNC=0`, `PII_MASKER_DEFAULT_TIMEOUT_SECONDS=0` 은 **여전히** 각자의 기본값으로 폴백한다(0 업로드 상한·0 동시성은 서비스를 못 쓰게 만든다). 이 4개를 `Load()` 경유로 단언하는 케이스를 같이 넣어, 이번 변경이 `envInt` 전체로 새지 않았음을 테스트가 증명한다.
  5) 테스트를 먼저 써서 수정 전 1)·2)가 실패하는 것을 실제로 보고, 수정 뒤 통과하는 것을 확인한다(수정을 임시로 되돌려 다시 실패하는 것까지 보면 더 좋다).
- 건드릴 파일 (프로덕션 1개):
  - `internal/config/config.go:148` — `MaxPages: envInt("PII_MASKER_MAX_PAGES", defaultMaxPages)` 를 `envNonNegativeInt("PII_MASKER_MAX_PAGES", defaultMaxPages)` 로 바꾸고, 왜 이 항목만 0을 허용하는지(= `service.countPages` 의 `MaxPages > 0` 가 무제한을 뜻함) 한 줄 주석을 남긴다. **`envInt` 함수 자체는 절대 고치지 말 것** — 같은 함수를 MaxFileSizeBytes·MaxConcurrentJobs·MaxConcurrentSync·Upstage.Timeout 이 함께 쓰므로 거기서 0이 통과하면 업로드 상한 0바이트, 동시 처리 0개, 타임아웃 0초가 된다.
  - `internal/config/config_test.go` — `Load()` 경유 테이블 테스트 추가. 기존 `TestLoadServerTimeoutRejectsNonPositiveValues`(124행)가 같은 모양의 선례다. `t.Setenv` 를 쓰므로 `t.Parallel()` 금지. `Load()` 는 `PII_MASKER_STORAGE_DIR` 아래 디렉터리를 만드니 `t.Setenv("PII_MASKER_STORAGE_DIR", t.TempDir())` 로 격리할 것(기존 config 테스트들이 어떻게 하는지 먼저 확인).
  - `internal/app/app_test.go` 또는 `internal/httpapi/integration_test.go` — 수용 기준 2)의 통합 테스트 1개. PDF 픽스처는 새로 만들지 말고 `internal/httpapi/integration_test.go:1640` 근처의 손으로 쓰는 PDF 빌더(`%PDF-1.4` + 객체 4개 + xref, `/Type /Pages /Kids [3 0 R] /Count 1`)를 페이지 N개로 일반화해 쓸 것(`Kids` 배열과 `Count`, 객체·xref 오프셋을 함께 늘려야 한다). `api.PageDims` 는 `ValidationRelaxed` 로 읽으므로 최소 구조로 충분하다. 다만 **이 빌더가 21쪽으로 늘렸을 때 pdfcpu 가 실제로 21을 세는지는 미확인** — 먼저 `api.PageDims` 로 길이를 찍어 확인하고, 손으로 늘리는 것이 어긋나면 수용 기준 2)를 `PII_MASKER_MAX_PAGES=0` + 2쪽 PDF + `t.Setenv("PII_MASKER_MAX_PAGES","0")` 대신 `Load()` 결과값 단언(기준 1)과 `internal/service` 레벨 테스트로 낮추고 그 사실을 PR 본문에 적을 것.
  - `README.md` 93-103행 환경변수 목록 — `PII_MASKER_MAX_PAGES` 에 "`0`을 주면 페이지 수 제한을 끕니다" 한 줄. README 41행이 `PII_MASKER_JOB_RETENTION_HOURS`의 `0` 을 같은 식으로 설명하니 문장을 거기에 맞출 것.
- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `go test -count=1 ./internal/config` (실패 재현용, 수정 전 먼저)
  - `go test -count=1 ./...` (전체 2초 미만)
  - `go vet ./...` / `go build ./...` / `gofmt -l ./cmd ./internal`(무출력이어야 함) / `git diff --check`
  - `go test -race -count=3 ./internal/config ./internal/app`
- 위험과 피할 것:
  - `envInt` 공용 함수를 고치는 것이 제일 큰 함정이다. 호출자 5곳이 모두 "0은 말이 안 되는 값"이고 `MaxPages` 만 예외다. 반드시 호출 지점 한 곳만 바꿀 것.
  - `envNonNegativeInt` 는 이미 존재한다(config.go:293). 새 헬퍼를 만들지 말 것.
  - `t.Setenv` 쓰는 테스트에 `t.Parallel()` 을 붙이면 Go 가 패닉한다. `internal/config/config_test.go` 의 기존 테스트들도 병렬이 아니다.
  - 무제한을 켰을 때 거대한 PDF가 렌더링 메모리를 먹는 것은 운영자가 명시적으로 켠 경우뿐이며, `PII_MASKER_MAX_FILE_SIZE_MB` 와 동기/비동기 슬롯 제한이 그대로 남아 있다. 이 점을 커밋 메시지/PR 에 한 줄로 적어 두면 리뷰에서 되묻지 않는다.
  - 보호 경로(upstage 인증·allow-host, service 의 draining/WaitGroup, jobs.load 의 RemoveAll)는 이번 과제와 무관하니 손대지 말 것.
  - 이 저장소의 반복 함정은 "같은 값을 읽는 경로가 둘 이상"이다. `MaxPages` 를 읽는 곳은 `service.countPages`(651) 하나와 `httpapi.handlePublicConfig`(server.go:117, `/v1/config/public` 의 `max_pages`) 둘이다. 무제한일 때 `/v1/config/public` 이 `max_pages: 0` 을 내보내는데 내장 Playground UI(`internal/httpapi/static/index.html`)가 그 값을 어떻게 쓰는지는 **미확인** — 구현 전에 index.html 에서 `max_pages` 를 grep 해 보고, UI가 "0쪽까지 허용"처럼 표시한다면 그 표시까지 이번 변경에 포함하거나(파일 1개 추가) 최소한 PR 에 명시할 것.
- 차선 후보: `internal/config` 의 나머지 환경변수 정규화·기본값을 `Load()` 경유 테이블 테스트로 덮기 (가치 2 / 위험 1 / 작업량 S) — `normalizeAllowHosts`(208행)·`normalizePIILang`·`normalizePIISchema`·`normalizeAuthMode`(233행)·`envBool`(305행)이 `Load()` 를 지나는 검증 없이 남아 있다. 1순위가 성립하지 않으면(예: 0 허용이 바람직하지 않다는 판단) 이쪽을 잡을 것. 특히 `normalizeAllowHosts` 는 `PII_MASKER_ALLOW_HOSTS` 가 설정되면 base URL 의 호스트를 **추가하지 않는다**(config.go:228 `if len(hosts) == 0`) — 이 동작이 의도된 것인지 확인하고 테스트로 못 박는 것만으로도 가치가 있다.
