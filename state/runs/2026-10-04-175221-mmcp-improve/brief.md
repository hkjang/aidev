- 과제: `BOOTSTRAP_ADMIN_PASSWORD` 가 72바이트를 넘으면 `mmcp reset-password` 가 `password_hash=''` 를 쓰고 "password reset for ..." 를 출력한다 — 깨어진 글래스 경로가 조용히 관리자를 잠근다 (가치 4 / 위험 2 / 작업량 S)

- 왜: `bcrypt.GenerateFromPassword` 는 72바이트 초과 비밀번호에 `(nil, ErrPasswordTooLong)` 를 돌려주는데(`golang.org/x/crypto@v0.57.0/bcrypt/bcrypt.go:96-97` 에서 확인), `cmd/mmcp/main.go:146` 이 그 에러를 `_` 로 버리고 `string(nil)` = `""` 를 그대로 `UPDATE users SET password_hash=$2, failed_logins=0, locked_until=NULL, status='active'` 에 넣은 뒤 exit 0 으로 성공을 보고한다. 같은 환경변수를 읽는 다른 경로(`internal/server/users.go:169 bootstrapAdmin`)는 `hashPassword` 의 에러를 **올바르게** 반환해 서버를 띄우지 않으므로, 두 경로가 같은 입력을 전혀 다르게 처리한다 — 비밀번호를 잃어버린 운영자가 쓰는 유일한 복구 수단이 조용히 계정을 더 망가뜨린다(`users.go:102` 의 `HasPassword = passwordHash != nil && *passwordHash != ""` 가 false 가 되고, `auth.go:172` 의 `CompareHashAndPassword([]byte(""), ...)` 는 항상 실패한다 — 인증 우회는 아니고 **잠금**이다).

- 수용 기준:
  1) `internal/config.Load()` 가 `BOOTSTRAP_ADMIN_PASSWORD` 73바이트 이상을 거부하고, 다른 필수 변수 누락과 **함께** 수집된 하나의 에러 문자열로 돌려준다(현재 `Load` 는 `problems` 를 모아 `strings.Join(problems, "; ")` 로 반환한다 — 이 관례를 깨지 말 것). 메시지는 `internal/server/users.go:128 validatePassword` 가 이미 쓰는 한국어 문구와 같은 기준(72바이트)을 말해야 한다.
  2) `cmd/mmcp/main.go:resetAdmin()` 이 해시 에러를 더 이상 버리지 않는다 — 에러면 `UPDATE` 를 **실행하지 않고** stderr 에 쓰고 1 을 반환한다. 어떤 경로로도 `password_hash` 에 빈 문자열이 들어가지 않는다.
  3) 신규 `internal/config/config_test.go` 가 Postgres 없이 돌며 증명한다: (a) 73바이트 비밀번호가 거부되고 72바이트는 통과한다(경계 양쪽), (b) 8바이트 미만이 여전히 거부된다, (c) 필수 변수 3개를 모두 비우면 에러 하나에 3건이 다 들어 있다, (d) `POSTGRES_DSN` 이 없으면 `DATABASE_URL` 로 넘어가고 공백이 트림된다(`firstEnv`), (e) `ENCRYPTION_KEY` 는 **실제** `crypto.ParseKeyring` 로 검증된다(손으로 만든 대역 금지 — 유효한 키 하나와 쓰레기 값 하나), (f) `Config.String()` 이 `BootstrapPassword` 를 담지 않는다(로그 유출 회귀 방지 — `main.go` 는 `cfg.String()` 대신 필드를 직접 찍지만 메서드는 존재하므로 계약을 못 박아 둘 것).

- 건드릴 파일 (프로덕션 2개 + 신규 테스트 1개):
  - `internal/config/config.go:Load` — `len(password) < 8` 검사 바로 뒤에 72바이트 상한을 추가. `len()` 은 **바이트**여야 한다(bcrypt 의 한계가 바이트이고 `validatePassword` 도 `len(pw) > 72` 로 바이트를 센다 — `utf8.RuneCountInString` 으로 바꾸지 말 것).
  - `cmd/mmcp/main.go:146 resetAdmin` — `hash, _ := bcrypt.GenerateFromPassword(...)` 의 `_` 를 실제 처리로. `internal/server/users.go:119 hashPassword` 와 같은 모양으로 맞추되, `internal/server` 는 별 패키지이고 `hashPassword` 는 비공개이므로 **import 하려 하지 말고** 이 자리에서 에러만 확인할 것.
  - `internal/config/config_test.go` (신규) — `t.Setenv` 으로 환경변수를 격리하고(`t.Setenv` 는 `t.Parallel` 과 함께 쓸 수 없다), 패키지는 `config`(내부 테스트)로 두면 `firstEnv` 도 직접 볼 수 있다. 유효한 `ENCRYPTION_KEY` 는 `crypto.ParseKeyring` 이 받는 형식을 **먼저 `internal/crypto/crypto.go` 와 `internal/crypto/crypto_test.go` 에서 확인하고** 쓸 것(32바이트 base64/hex 또는 32자 이상 패스프레이즈라고 `config.go:38` 의 메시지가 말하지만, 실제 수용 형식은 `ParseKeyring` 이 정본이다 — `crypto_test.go` 에 쓰인 값을 재사용하는 것이 가장 안전하다). 이번 회차에 `internal/crypto/crypto.go:44-67 ParseKeyring` 을 읽어 확인한 것: 쉼표로 쪼개 여러 키를 받고, 빈 조각은 건너뛰고, 각 조각을 `decodeKey` 에 넘기며, 유효한 키가 하나도 없으면 `"no key material"` 에러를 낸다. `decodeKey` 의 수용 형식 자체는 열어 보지 않았다(미확인) — 테스트 값을 고르기 전에 그 함수를 볼 것.

- 검증 명령:
  - `go test -race -count=1 ./internal/config/` — 새 테스트. Postgres 불필요.
  - `go test -race -count=1 ./...` — 회귀. `internal/server` 통합 테스트는 `TEST_POSTGRES_DSN` 없으면 skip 되고 그래도 `ok` 로 나온다(이 과제는 Postgres 를 요구하지 않는다).
  - `gofmt -l .` 가 빈 출력이어야 하고 `make lint`(gofmt + `go vet ./...` + `./scripts/verify-version.sh`) exit 0.
  - `go build ./...` — `cmd/mmcp` 를 건드리므로 반드시. (`make build` 는 UI 빌드를 선행해 오래 걸린다 — 이 과제엔 불필요.)
  - 뮤테이션으로 테스트가 이빨을 갖는지 확인할 것: 새 72바이트 가드를 지우면 `internal/config` 테스트가 떨어져야 한다.

- 위험과 피할 것:
  - **되돌릴 수 없는 회귀 가능성 하나를 알고 들어갈 것**: 오늘은 관리자 계정이 **이미 존재하면** `bootstrapAdmin`(`users.go:166-178`) 이 해싱을 건너뛰므로 73바이트 이상 `BOOTSTRAP_ADMIN_PASSWORD` 로도 서버가 뜬다. `Load()` 에서 거부하면 그런 설치는 다음 부팅에 **기동 실패**한다. 이것이 맞는 거래라고 보는 근거: 같은 값으로 첫 부팅을 하면 오늘도 `bootstrapAdmin` 이 에러를 반환해 기동이 막히고(두 경로가 이미 갈려 있다), 관리 콘솔의 모든 비밀번호는 `validatePassword` 가 같은 72바이트 상한을 강제한다. 에러 메시지에 **무엇을 어떻게 고쳐야 하는지**(72바이트 이하로 줄여 재시작) 를 적어 운영자가 막히지 않게 할 것.
  - `internal/server/auth.go`·`oidc.go`·`keys.go`·`oauth.go` 는 **열지 말 것**(보호 경로). `users.go:validatePassword`·`hashPassword` 도 바꾸지 말 것 — 이미 옳다. 이 과제는 그 규칙을 config 쪽에 맞추는 일이다.
  - `internal/store/migrations` 와 릴리즈 경로(`VERSION`, `scripts/package-offline.sh`, `.github/workflows/release.yml`)는 건드리지 않는다. `VERSION` 은 1.0.1 이고 `verify-version.sh` 가 여러 파일의 일치를 강제하므로 버전 문자열에 손대면 lint 가 떨어진다.
  - `web/` 는 건드리지 말 것. 이 워크트리에서 `cd web && npm test` 는 `web/package.json:10` 의 `"vitest run"` 때문에 깨끗한 체크아웃에서 실패한다(base `b782c5f` 에서 미수정 확인). `make test` 는 web 을 부르므로 **검증에 `make test` 를 쓰지 말고** 위의 `go test` + `make lint` 를 쓸 것.
  - 교훈: 손으로 만든 대역으로 증명하지 말 것 — `crypto.ParseKeyring` 과 `bcrypt` 는 실제 타입/실제 함수로 통과시킬 것. 그리고 `cfg.BootstrapPassword` 를 로그·에러 메시지에 **절대** 찍지 말 것(길이만 말할 것).

- 차선 후보: `logbuf.Handler.Handle` 이 레코드 attr 로 넘어온 `slog.Group(...)` 을 평탄화하지 않아 `[]slog.Attr` 가 `/admin/logs` JSON 에 날것으로 나간다 [3/3/S] — `logbuf.go` 의 `a.Value.Resolve().Any()` 가 `KindGroup` 을 그대로 맵에 넣는다(이전 회차가 코드로 확인, 이번 회차 미재확인). 고치려면 `error` 평탄화 루프처럼 재귀 평탄화하고 `web/src/pages/admin/Logs.tsx:16 attrsInline` 의 `typeof v === 'object'` 분기가 여전히 맞는지 함께 볼 것. 단, 프로덕션 호출처가 0건으로 보여(미확인) 가치가 1순위보다 낮다.
