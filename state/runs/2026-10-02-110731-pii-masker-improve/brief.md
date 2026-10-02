- 과제: `PII_MASKER_MAX_FILE_SIZE_MB`의 MiB 곱셈 오버플로 방어 (가치 3 / 위험 1 / 작업량 S)
- 왜: `internal/config/config.go:148`이 `int64(envInt("PII_MASKER_MAX_FILE_SIZE_MB", 50)) * 1024 * 1024`로 검사 없이 곱한다. MB 값이 `math.MaxInt64/(1024*1024)` = 8796093022207을 넘으면 `MaxFileSizeBytes`가 음수로 감싼다(`8796093022208 * 1048576`은 정확히 2^63이라 `MinInt64`가 된다 — 산술로 확인, 실행 확인은 샌드박스 제약으로 미수행). 그 음수는 업로드 경로 세 곳이 모두 읽으므로 **빈 파일이 아닌 모든 업로드가 거부된다**: `readProcessInput`(server.go:315)의 `http.MaxBytesReader(w, body, 음수)`, 같은 함수 330행의 `io.LimitReader(file, 음수+1)`(0바이트만 읽혀 "uploaded file is empty"), 그리고 `service.validateAttachment`(service.go:628)의 `attachment.Size > 음수`가 항상 참. `/v1/config/public`의 `max_file_size_bytes`(server.go:116)도 음수로 노출된다. **미확인: 셋 중 어디서 먼저 끊겨 어떤 상태코드가 나오는지**(Go의 `MaxBytesReader`가 음수 n을 0으로 보는지 무제한으로 보는지 이번에 소스 확인 못 함) — 그래서 수용 기준은 "수정 후 200"으로 두고, 구현자는 수정 전 실제 응답을 실행해서 기록하라. 2026-09-29(001f6f7)에 같은 종류의 결함을 시간 설정 여섯 개에 대해 고치면서 `MaxFileSizeBytes`의 MB 곱셈은 범위 밖으로 남겨 뒀고(그 회차 노트의 "크기 오버플로는 미해결 후보"), 이번에 그 하나를 닫는다. 고치면 잘못된 값이 조용히 서비스를 전면 거부 상태로 만들지 않고 기본값 50MB로 복귀한다.
- 수용 기준:
  1) `PII_MASKER_MAX_FILE_SIZE_MB=8796093022208`(또는 더 큰 값)으로 `config.Load()`를 지나면 `cfg.Limits.MaxFileSizeBytes`가 기본값 `50*1024*1024`가 되고, 어떤 입력에서도 결과가 0 이하가 되지 않는다.
  2) 기존 계약 불변: 미설정→50MiB, `"1"`→1MiB, `"8796093022207"`(허용 최대)→`8796093022207*1024*1024`(양수), `"0"`·`"-1"`·`"abc"`·빈 문자열→50MiB. 다른 정수 설정(`MAX_PAGES`, `MAX_CONCURRENT_JOBS`, `MAX_CONCURRENT_SYNC`)의 동작은 그대로다.
  3) 테스트가 증명할 것: (a) `Load()` 경유 테이블 테스트가 경계값 8796093022207/8796093022208을 양쪽에서 단언한다. (b) 프로덕션 배선을 지나는 HTTP 회귀 1건 — 거대한 MB 값을 준 서버가 작은 PNG `POST /v1/mask`를 200으로 처리한다(수정 전 응답은 실행해서 확인·기록할 것: 413 또는 400이 예상되고 둘 다 결함의 증거다). (c) 수정만 되돌리면 두 테스트가 다시 실패한다.
- 건드릴 파일 (프로덕션 1개):
  - `internal/config/config.go:148` — 호출 지점만 교체. 새 헬퍼 `envMebibytes(key string, fallbackMB int64) int64`를 `envDuration`(config.go:298) 바로 아래에 더해 `strconv.ParseInt(value, 10, 64)`로 읽고 `err != nil || parsed <= 0 || parsed > math.MaxInt64/(1024*1024)`면 `fallbackMB * 1024 * 1024`를 돌려준다. `math`는 이미 import되어 있다(001f6f7이 넣음). 왜 곱셈 전에 상한을 보는지 `envDuration` 주석과 같은 톤으로 한 줄 남긴다.
  - `internal/config/config_test.go` — `Load()` 경유 테이블 테스트 추가(기준 2의 7개 입력 + 경계 2개). `t.Setenv` 사용 테스트는 `t.Parallel()` 금지, 외부 환경 영향 차단을 위해 읽는 변수는 명시적으로 비운다(기존 테스트들이 쓰는 방식 참고).
  - `internal/app/app_test.go` — HTTP 회귀 1건. `app_test.go:509-560`의 `TestUnlimitedMaxPagesAcceptsADocumentOverTheDefaultLimit`를 그대로 본떠라: `net.Listen("tcp","127.0.0.1:0")` → `t.Setenv(STORAGE_DIR/ENABLE_EMBEDDED_UPSTAGE_MOCK=true/UPSTAGE_BASE_URL=""/ADDR=<리스너 주소>/MAX_FILE_SIZE_MB="8796093022208")` → `config.Load()` → `app.New` → `Serve(ctx, listener)` → 그 주소로 작은 PNG `POST /v1/mask`. `cfg.Limits.MaxFileSizeBytes > 0`도 함께 단언해 실패 원인이 분명하게 남게 한다.
  - `README.md` — 환경변수 문단에 한 문장(이 저장소의 관례).
- 검증 명령 (이 저장소에서 실제로 도는 것, 이번 정찰에서 `go test -count=1 ./...` 통과 확인):
  - `go test -count=1 ./internal/config ./internal/app`
  - `go test -count=1 ./...`
  - `go vet ./...` · `go build ./...` · `gofmt -l ./cmd ./internal`(무출력) · `git diff --check`
  - `go test -race -count=1 ./internal/config ./internal/app`
  - `GOARCH=386 go test -count=1 ./internal/config` — 001f6f7이 쓴 검증. `int64` 파싱으로 바꾸면 32비트에서도 같은 값을 읽어야 한다(`envInt`는 `Atoi`라 32비트에서 큰 값이 조용히 기본값으로 떨어졌다).
- 위험과 피할 것:
  - **`envInt` 함수 자체를 고치지 마라.** 2026-09-27 회차에서 같은 상황을 "호출 지점만 교체"로 처리해 파급을 막았다. `MAX_CONCURRENT_JOBS`/`MAX_CONCURRENT_SYNC`는 곱셈이 없어 이번 범위 밖이다.
  - `uploadBodyHeadroomBytes`(server.go:29,315)의 `+64KiB`는 건드리지 말 것. 상한을 `MaxInt64/(1024*1024)`로 두면 bodyLimit 덧셈도 오버플로하지 않는다(여유 1MiB-64KiB 이상 남음) — 그래도 상한을 더 낮추고 싶다면 기존 통과 테스트(`integration_test.go:355,400,423,448`의 작은 limit)는 영향받지 않는지 확인하라.
  - 보호 경로(auth/migrations/workflows) 없음. `docker-compose.yml`·`Dockerfile`은 이 변수를 설정하지 않으므로 손대지 않는다(미확인: compose가 다른 값을 넣는지는 grep 한 번으로 확인하라).
  - 재선정 금지(이미 해결됨): UTF-8 절단, created_at 보존, 미완료 `download_url`, mock 기본 URL, HTTPS 강등, `MAX_PAGES=0`, 시간 설정 오버플로.
- 차선 후보: `internal/config`의 나머지 정규화 함수(`normalizeAllowHosts`·`normalizePIILang`·`normalizePIISchema`·`normalizeAuthMode`·`envBool`)를 `Load()` 경유 테이블 테스트로 덮기 (가치 2 / 위험 1 / 작업량 S) — 프로덕션 변경 0, 테스트 파일 1개. README가 설명하는 "명시 allow list 우선" 계약이 실제로 유지되는지 확인하는 값이 있다.
