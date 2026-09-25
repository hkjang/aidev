- 과제: 내장 mock 기본 업스트림 URL이 `PII_MASKER_ADDR`의 실제 포트를 따르게 하기 (가치 3 / 위험 2 / 작업량 S)
- 왜: `internal/config/config.go:110-113`은 `PII_MASKER_ENABLE_EMBEDDED_UPSTAGE_MOCK`이 켜져 있고 `PII_MASKER_UPSTAGE_BASE_URL`이 비어 있을 때 기본 업스트림을 `http://localhost:8080/internal/mock/upstage/inference`로 **하드코딩**하는데, mock 핸들러는 `internal/app/app.go:42`에서 **API와 같은 서버**에 마운트되므로 이 URL은 "자기 자신"을 가리키는 값이어야 한다. `PII_MASKER_ADDR=:9090`으로 띄우면 서비스는 자기가 듣지 않는 8080으로 마스킹 요청을 보내고(연결 거부로 모든 마스킹 실패), 그 머신의 8080에 다른 프로세스가 떠 있으면 **업로드한 PII 원문과 인증 토큰이 무관한 로컬 서비스로 전송된다** — `normalizeAllowHosts`가 이때 채우는 기본 allow list는 `localhost` 하나이고 `hostAllowed`는 bare-host 항목에서 포트를 무시하므로(`internal/upstage/client_test.go:136` "port ignored for bare host entry") 차단되지 않는다. 고치면 mock 모드가 포트와 무관하게 동작하고 이 오발송 경로가 사라진다.
- 수용 기준:
  1) mock을 켜고 `PII_MASKER_UPSTAGE_BASE_URL`을 비운 채 `PII_MASKER_ADDR`를 `127.0.0.1:<임의 포트>`로 두고 `config.Load()`를 하면 `cfg.Upstage.BaseURL`이 그 호스트·포트의 `/internal/mock/upstage/inference`를 가리킨다(8080이 아니다).
  2) 주소에 호스트가 없거나(`:9090`) 와일드카드인 경우(`0.0.0.0:7000`, `[::]:7000`) 다이얼 가능한 호스트로 대체된다(`localhost:9090`, `localhost:7000`). `PII_MASKER_ADDR`가 `host:port` 형태가 아니면(SplitHostPort 실패) 지금 동작(`localhost:8080`)을 그대로 유지한다.
  3) `PII_MASKER_UPSTAGE_BASE_URL`을 명시하면 그 값이 그대로 이긴다(mock 여부와 무관, 기존 동작 불변). mock이 꺼져 있을 때의 기본값 `http://localhost:8080/inference`는 이번 범위가 아니다(아래 "위험과 피할 것").
  4) 테스트가 증명할 것: **프로덕션 배선으로 끝까지 도는 것**. 즉 `net.Listen("tcp","127.0.0.1:0")`으로 리스너를 먼저 잡아 `listener.Addr().String()`을 `t.Setenv("PII_MASKER_ADDR", …)`에 넣고 → `config.Load()` → `app.New(cfg)` → `application.Serve(ctx, listener)` 로 띄운 뒤, 그 주소에 PNG를 `POST /v1/mask` 하면 내장 mock을 타고 `status=completed`가 돌아온다. 고치기 전에는 이 테스트가 실패해야 한다(URL이 8080을 가리키므로 업스트림 호출 실패). 실패 메시지가 CI 머신의 8080 점유 여부에 좌우되지 않도록 같은 테스트에서 `cfg.Upstage.BaseURL`이 리스너 포트를 담고 있는지도 함께 단언할 것.
- 건드릴 파일:
  - `internal/config/config.go:107-130 Load` — 지금 `Address`는 `cfg` 리터럴 안(125행)에서 `envOrDefault("PII_MASKER_ADDR", defaultAddress)`로 읽히고 mock 기본 URL은 그보다 앞선 110-113행에서 결정된다. `address := envOrDefault("PII_MASKER_ADDR", defaultAddress)`를 Load 맨 위로 끌어올려 두 곳이 **같은 값**을 읽게 하고(이 저장소의 반복 함정: 같은 값을 읽는 경로가 둘), mock 기본 URL을 새 헬퍼로 조립한다.
  - `internal/config/config.go` — 새 비공개 헬퍼(예: `localMockBaseURL(address string) string` 또는 `dialableHostPort(address string) string`). `net.SplitHostPort` 사용 → host가 `""`·`"0.0.0.0"`·`"::"`이면 `localhost`로 대체, 그 외에는 그대로. 에러면 기존 `localhost:8080` 유지. 왜 자기 자신을 가리켜야 하는지 한 줄 주석(mock이 같은 mux에 마운트된다는 사실)을 남길 것 — 이 저장소의 주석 관례는 영어.
  - `internal/config/config_test.go` — 수용 기준 1~3의 표 테스트. 기존 두 테스트와 같은 스타일로 `t.Setenv("PII_MASKER_STORAGE_DIR", t.TempDir())`를 먼저 두고 `Load()`를 호출한다(`Load`가 `os.MkdirAll`을 한다). `t.Setenv`를 쓰므로 `t.Parallel()` 금지.
  - `internal/app/app_test.go` — 수용 기준 4의 통합 테스트 1개. 기존 헬퍼(381행 근처, `cfg.Server.Address`로 `net.Listen` 후 `application.Serve(ctx, listener)`를 고루틴에 띄우고 `listener.Addr().String()`을 돌려주는 헬퍼)는 `config.Config`를 손으로 만들기 때문에 이번 경로(`config.Load`)를 지나지 않는다. 헬퍼를 고치지 말고 이 테스트 안에서 리스너 → `t.Setenv` → `config.Load()` → `app.New` → `Serve` 순서로 직접 띄울 것. 정리는 `t.Cleanup(cancel)` + `waitForServe`(396행) 관례를 따른다.
  - `README.md` 109행 `PII_MASKER_ENABLE_EMBEDDED_UPSTAGE_MOCK` 항목 근처 또는 139행 docker-compose 문단에 한국어 한 문장 추가(동작 변경 시 README에 한 문장이 이 저장소 관례). `docker-compose.yml:14`는 base URL을 명시하고 `PII_MASKER_ADDR`를 설정하지 않으므로 **수정 불필요** — 건드리지 말 것.
- 검증 명령 (이 저장소에서 실제로 도는 것, 전체 2초 미만):
  - `go test -count=1 ./internal/config ./internal/app`  ← 새 테스트
  - `go test -count=1 ./...`
  - `go vet ./...` / `go build ./...` / `gofmt -l ./cmd ./internal`(무출력) / `git diff --check`
  - `go test -race -count=3 ./internal/app ./internal/config`
  - 고치기 전에 새 테스트가 실제로 실패하는 것을 먼저 확인하고, 고친 뒤 통과를 확인할 것(이 저장소의 채택 회차가 모두 이 순서를 밟았다).
- 위험과 피할 것:
  - **allow-host 검사(`internal/upstage/client.go:464-510`)를 건드리지 말 것.** 이번 변경은 config가 만드는 기본 URL만 바꾸며, `normalizeAllowHosts`는 그 URL의 `Hostname()`을 그대로 기본 allow list로 넣으므로 자동으로 맞는다(`localhost`/`127.0.0.1`). 보호 경로다.
  - **mock이 꺼져 있을 때의 기본 `http://localhost:8080/inference`(115행)는 그대로 둘 것.** 그것은 외부 추론 서버 자리표시자이지 자기 자신이 아니며, 같이 바꾸면 범위가 커지고 의미가 달라진다. 브리프의 "왜"는 mock 경로에만 해당한다.
  - `PII_MASKER_UPSTAGE_BASE_URL`이 설정된 경우의 분기(109-121행)를 재배치하지 말 것 — 명시 값이 이기는 순서가 깨지면 docker-compose 구성이 조용히 달라진다.
  - `0.0.0.0`을 그대로 다이얼 주소로 쓰지 말 것(리눅스에서는 우연히 되지만 이식성이 없다). `localhost`로 대체한다.
  - 포트를 잡았다가 닫고 다시 여는 방식(`listen → close → Setenv → Load → 다시 listen`)은 경합이 난다. **리스너를 열어 둔 채** 그 주소를 `PII_MASKER_ADDR`에 넣고 같은 리스너를 `Serve`에 넘길 것(`App.Serve(ctx, listener)`가 외부 리스너를 받는다 — `internal/app/app.go:88`).
  - `internal/config/config_test.go`의 기존 두 테스트는 수정하지 말고 추가만 할 것.
- 차선 후보: **internal/config `Load()` 경유 환경변수 정규화·기본값 테이블 테스트 (가치 2 / 위험 1 / 작업량 S)** — `config_test.go`는 지금도 70줄, 서버 timeout 3개뿐이고 `normalizeAllowHosts`/`normalizeEndpointURL`/`normalizePIILang`/`normalizePIISchema`/`normalizeAuthMode`/`envInt`/`envNonNegativeInt`/`envBool`이 `Load()` 경유로는 무검증이다(순수 함수 단위 테스트가 아니라 `t.Setenv`+`Load()`로 붙일 것, 병렬 금지). 1순위가 성립하지 않으면 이쪽으로 가되, 1순위를 고친 뒤 남은 시간에 겹쳐 하지는 말 것(같은 파일이라 충돌).
