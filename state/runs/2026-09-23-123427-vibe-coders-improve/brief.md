# 과제서 — 2026-09-23-123427-vibe-coders-improve

기준 HEAD: `d67267b` (worktree `/home/hkjang/.cache/auto-improve-wt/vibe-coders`, 작업 트리 깨끗).

- **과제**: `text2sql.work_mem` 값을 저장 시점에 검증하고, 형식에 맞지 않으면 pgx 실행 트랜잭션에 `SET LOCAL work_mem` 을 보내지 않는다 (가치 3 / 위험 2 / 작업량 S)

- **왜**: `internal/proxy/text2sql_handler.go:1201` 은 `SET LOCAL work_mem = '` + 작은따옴표만 제거한 값 + `'` 를 READ ONLY 트랜잭션에 보내고 그 결과를 `_, _ =` 로 버린다. `text2sql.work_mem` 은 `internal/proxy/admin_settings.go:158` 에서 `Type: stString` 이고 `validate` 가 없어 관리자가 `64M`·`64 MB`·`large` 같은 값을 그대로 저장할 수 있고, `config.go:557` 의 `TEXT2SQL_WORK_MEM` 환경변수도 검증 없이 같은 자리로 들어온다. 값이 PostgreSQL 이 모르는 형식이면 `SET LOCAL` 이 오류를 내는데 그 오류는 버려지고, 같은 트랜잭션 안의 실제 쿼리가 대신 실패한다(PostgreSQL 은 트랜잭션 안의 오류 뒤 이후 명령을 거부한다 — **이 동작은 이 환경에 PostgreSQL 이 없어 실행으로 확인하지 못했다. 문서 지식이며 구현자는 검증 불가 영역으로 다룰 것**). 고치면 잘못된 값이 저장 단계에서 이유와 함께 거부되고, 이미 저장돼 있거나 환경변수로 들어온 잘못된 값도 실행 경로를 망가뜨리지 않는다.

- **수용 기준**:
  1. 관리자 설정 저장 경로(`POST /admin/settings/validate` 와 일괄 저장)에서 `text2sql.work_mem` 에 `64M`, `64 MB`, `large`, `64MB; DROP` 같은 값을 넣으면 400 으로 거부되고 메시지가 허용 형식(예: `64MB`)을 말한다. 빈 문자열은 여전히 허용된다(기본값 = 미설정).
  2. `64kB`·`64MB`·`1GB` 같은 정상 값은 지금처럼 저장되고 `effectiveSettingValue` 로 되읽힌다(기존 저장 값이 있는 설치를 읽기만으로 깨뜨리지 않는다 — 검증은 저장 시점에만 건다).
  3. 실행 경로에서, 형식을 만족하지 않는 `workMem` 은 `SET LOCAL work_mem` 을 **보내지 않는다**(문자열 이어붙이기 자체를 하지 않는다). 형식을 만족하는 값은 지금과 같은 SQL 을 만든다.
  4. 테스트가 증명할 것: (a) 설정 저장 경로는 실제 `NewServer` 라우트 + SQLite 저장소를 통과하는 HTTP 요청으로 거부/허용을 보여 준다(손으로 만든 `settingDef` 를 직접 호출하는 단위 테스트만으로 끝내지 말 것). (b) 실행 경로는 `workMem` 판정 헬퍼에 대한 표 기반 테스트로 허용/거부 목록을 고정하고, 거부 값에서 SQL 이 만들어지지 않음을 보인다. (c) 수정 전 코드에서 (a) 의 거부 케이스가 200 으로 통과하던 것을 먼저 확인하고 기록한다.

- **건드릴 파일**:
  - `internal/proxy/admin_settings.go` — `chIdent` 클로저(현재 113행)가 있는 자리에 같은 모양의 `memSize` 클로저를 추가하고, 158행 `{Key: "text2sql.work_mem", …}` 에 `validate: memSize` 를 붙인다. 형식은 `^\d+(kB|MB|GB|TB)$` 정도로 좁게 잡고 빈 문자열은 통과시킨다(`validateSettingValue` 는 빈 값을 따로 건너뛰지 않으므로 클로저 안에서 `v == ""` 를 먼저 통과시켜야 한다 — 현재 1003행 `validateSettingValue` 확인함).
  - `internal/proxy/text2sql_handler.go` — `executeReadOnlyQuery` 의 `case "pgx"` 안 `if workMem != ""` 분기(현재 1200–1202행). 판정을 같은 규칙의 작은 헬퍼(예: `validWorkMem(string) bool` 또는 `workMemSetting(string) (string, bool)`)로 빼고, 두 자리(설정 검증·실행)가 **같은 규칙을 같게** 읽도록 한 곳에 둔다. `executeReadOnlyQuery` 의 시그니처와 호출처는 바꾸지 말 것.
  - 테스트: `internal/proxy/` 에 새 `_test.go` 하나(예: `text2sql_workmem_test.go`). 기존 `admin_settings_test.go`·`settings_bulk_test.go` 가 HTTP 로 설정을 저장하는 방식을 먼저 읽고 그 패턴을 그대로 쓸 것.
  - 문서: `README.md:893` 과 `docs/ADMIN_GUIDE.md:157` 의 `TEXT2SQL_WORK_MEM` 줄에 허용 형식 한 마디를 더한다(둘 다 실제로 존재하는 줄임을 확인함).

- **검증 명령** (worktree 루트에서):
  - `gofmt -l internal/proxy`
  - `go build ./...`
  - `go vet ./...`
  - `go test ./internal/proxy -run 'WorkMem|Setting' -count=1`
  - `go test ./... -count=1` (직전 회차 기록 기준 약 80초)
  - `go test -race ./internal/proxy -run 'WorkMem' -count=1`
  - `go run ./cmd/api-surface-audit` (설정 키 계약을 건드렸는지 확인)
  - 프런트는 이 과제에서 건드리지 않으므로 `web` 검증은 러너가 도는 것 외에 추가로 요구하지 않는다.

- **위험과 피할 것**:
  - **같이 건드리지 말 것**: `executeReadOnlyQuery` 첫 줄의 고정 `context.WithTimeout(ctx, 30*time.Second)`. 09-20 회차가 여기에 `execQueryDeadline` 헬퍼를 넣었지만 **그 커밋은 이 HEAD 의 조상이 아니어서 코드에 없다**(1184행에서 30s 고정 확인). 같은 함수라고 해서 이번 diff 에 타임아웃 수정을 섞지 말 것 — 별도 회차 과제로 둔다.
  - 검증을 너무 넓게 잡으면 지금 정상 동작 중인 설치의 값을 저장 시점에 거부할 수 있다. `kB/MB/GB/TB` 와 순수 정수(바이트 아님 — PostgreSQL 은 단위 없는 숫자를 kB 로 읽으므로 허용할지 판단해 문서에 적을 것)만 좁게 다루고, 기존에 **저장된** 값은 읽기 경로에서 거부하지 말 것(수용 기준 2).
  - 환경변수 경로(`config.go:557`)에서 프로세스를 죽이거나 기동을 막지 말 것. 잘못된 값은 "적용하지 않음" 으로 처리하고, 남긴다면 `slog` 경고 한 줄까지만.
  - 보호 경로 회피: `internal/proxy/auth.go`·`keycloak*.go`·`pipeline.go`, `internal/store` 마이그레이션, `.github/workflows` 는 건드리지 않는다. 이 과제는 그 어느 것도 필요로 하지 않는다.
  - 증거로 삼지 말 것: 소스 문자열 검사, 손으로 만든 가짜 `*sql.DB`. PostgreSQL 이 없으므로 pgx 분기의 실제 실행은 확인할 수 없다 — 과제서에 **미확인** 으로 남기고 "PostgreSQL 에서 고쳐졌다" 고 주장하지 말 것.
  - 작은따옴표 제거가 이미 문자열 리터럴 탈출을 막고 있어 **이것을 SQL 인젝션 수정으로 광고하지 말 것**. 실제로 고치는 것은 "잘못된 값이 조용히 트랜잭션을 망가뜨리는 것" 과 "관리자가 그 사실을 알 길이 없는 것" 이다.

- **차선 후보**: `docs/APP_UI_ROADMAP.md:28` 의 캐시 설명 정정 (가치 2 / 위험 1 / 작업량 S). 문서는 "`index.html` 은 항상 `Cache-Control: no-cache`" 라고 적었으나, 추적이 켜진 요청에서 `internal/appui/handler.go:147` 의 `serveTrackedIndex` 는 `no-store` 를 보내고 validator 를 붙이지 않는다(`appCacheControl = "no-cache"` 는 26행, `no-store` 는 147행에서 확인). `internal/appui/tracking_test.go:80` 이 이미 `no-store` 를 고정하고 있으므로 문서만 코드에 맞추면 된다. 1순위가 성립하지 않을 때만 고를 것.
