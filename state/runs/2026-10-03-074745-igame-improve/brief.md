- 과제: OIDC 로그인/콜백이 "설정을 읽을 수 없음"을 "SSO 미설정"으로 바꿔 보고하고 로그도 남기지 않는 것을 고치기 (가치 3 / 위험 2 / 작업량 S)

- 왜: `internal/api/auth.go:212` 의 `oidcLogin` 과 `auth.go:302` 의 `oidcCallback` 은 `cfg, err := s.loadOIDCSetting(...)` 의 `err != nil` 을 `!cfg.Enabled` 와 같은 분기에 넣어 `writeError(w, 404/400, "oidc_disabled", "OIDC login is not configured")` 로 답한다. 그런데 `loadOIDCSetting`(auth.go:195-209)이 에러를 내는 두 경로는 모두 **전사 로그인 전면 장애**다 — ① `s.setting(ctx,"oidc",&cfg)` 가 일시적 DB 실패를 올린 경우(cache.go:37-57 의 `default:` 분기는 실패를 캐시하지 않고 그대로 반환한다) ② `s.Secrets.Open(cfg.ClientSecret)` 가 설치키가 바뀌었거나 저장된 ciphertext 가 손상돼 실패한 경우(secretbox.go:36-48 의 세 거부 경로).
  그 결과 사용자는 `web/src/api/errorMessages.ts:26` 의 "사내 SSO가 설정되어 있지 않습니다" 를 보고, 운영자는 바로 아래 `oidcProvider` 실패가 `s.serverError(..., 502, "oidc_discovery_failed", ..., err)` 로 로그를 남기는 것과 달리 **아무 로그도** 받지 못한다(`writeError` 는 err 를 받지 않는다). 셸이 없는 런타임 이미지(status.go:35-39 주석)에서 이 오진은 장애 복구를 직접 늦춘다. 같은 저장소에 올바른 선례가 이미 있다: `playAllowed`(catalog.go:429-434)는 `pgx.ErrNoRows` 만 "미설정" 으로 읽고 나머지 에러는 거부로 올린다.

- 수용 기준:
  1) `oidc` 설정 읽기가 `pgx.ErrNoRows` **가 아닌** 에러로 실패하거나 `s.Secrets.Open` 이 실패하면 `GET /api/v1/auth/oidc/login` 이 `503` 과 새 코드 `oidc_unavailable` 을 돌려주고, 응답 본문이 "미설정" 이라고 말하지 않는다. 같은 조건에서 `GET /api/v1/auth/oidc/callback` 도 `503 oidc_unavailable` 이다(현재 400).
  2) 그 경로가 `s.serverError`(또는 동등한 로깅 경로)를 지나 원인 err 가 서버 로그에 남는다. 테스트는 `Server.Logger`(실제 필드명은 소스에서 확인할 것 — 미확인) 를 버퍼로 바꿔 로그 한 줄에 원인이 들어갔음을 확인한다.
  3) **회귀 금지**: 설정을 정상적으로 읽었고 `enabled=false` 이거나 `issuer`/`client_id` 가 빈 경우에는 지금과 똑같이 login 이 `404 oidc_disabled`, callback 이 `400 oidc_disabled` 다. `migrations/001_initial.sql:239` 가 `('oidc','{"enabled":false,...}')` 를 seed 하므로 이것이 갓 설치한 상태의 정상 응답이며, 이 코드/상태 쌍이 바뀌면 프런트 문구가 어긋난다.
  4) `pgx.ErrNoRows`(행이 없음)도 "미설정" 으로 남는다 — `oidc_disabled`. `errors.Is` 로 분기할 것.
  5) 테스트가 증명해야 하는 것: 손으로 만든 대역이 아니라 **실제** `secretbox.New` 로 만든 두 개의 서로 다른 키로 Seal/Open 불일치를 만들고, **실제** 설정 캐시(`storeSetting`)와 **실제** `Router()`/핸들러를 지나 위 세 분기(복호화 실패 / DB 실패 / 정상 disabled)가 서로 다른 상태코드·코드를 낸다는 것.

- 건드릴 파일 (프로덕션 2개):
  - `internal/api/auth.go:oidcLogin`(212-216 근처) — `err != nil` 을 분리해 `errors.Is(err, pgx.ErrNoRows)` 는 기존 404 `oidc_disabled` 로, 그 외는 `s.serverError(w, r, 503, "oidc_unavailable", "OIDC 설정을 읽을 수 없습니다", err)` 로. `cfg.Enabled`/`Issuer`/`ClientID` 검사는 그대로 둘 것.
  - `internal/api/auth.go:oidcCallback`(302-306 근처) — 같은 분리. 단 콜백은 앞선 `DELETE FROM oidc_flows ... RETURNING`(auth.go:297)이 이미 state 를 소모했으므로 **쿼리 순서를 바꾸지 말 것**. 상태 소모 뒤의 응답 코드만 바꾼다.
  - `web/src/api/errorMessages.ts` — `oidc_unavailable` 한국어 문구 1줄 추가(예: "사내 SSO 설정을 읽을 수 없습니다. 잠시 후 다시 시도하거나 운영자에게 문의하세요."). `oidc_disabled` 문구는 그대로.
  - `internal/api/oidc_unavailable_test.go` (신규) — 아래 방식. `internal/api/play_policy_test.go:17-27`(`playPolicyServer` 의 `storeSetting` + `settingEntry{raw:..., expires:...}`)과 `:140-147`(포트 1 을 향한 `pgxpool.New` 로 DB 실패 재현)을 그대로 빌려 쓸 것 — DB 없이 돈다.
  - (선택) `docs/api.md` 의 OIDC 로그인 항목에 코드 한 줄. 과제 범위가 넘치면 생략.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `gofmt -l .` (무출력), `go vet ./...`, `go build ./...`
  - `go test ./internal/api/ -run 'TestOIDC' -count=1 -v`
  - `go test ./cmd/... ./internal/... ./migrations/...`
  - `go test ./internal/api/ -run 'TestOIDC' -race -count=3`
  - `bash scripts/check-release-contract.sh` (v0.7.26 확인)
  - 프런트 파일을 고쳤으므로 `make web-build` 도 한 번 돌릴 것(errorMessages.ts 는 타입 검사 대상).
  - 실DB 는 필요 없다. 그래도 한 번 돌리려면 README 절차의 일회용 PG17 로 `make test-db DSN='postgres://igame:igame@127.0.0.1:15432/igame?sslmode=disable&search_path=public,igame_test_extensions'`.

- 인과 확인(구현자가 반드시 할 것): 고친 분기를 한 줄씩 되돌려(예: `errors.Is` 분기를 다시 `err != nil` 하나로 합치기) 신규 테스트가 Red 가 되는지 보고 원복 후 Green 을 확인할 것. grep 으로 문자열이 있다는 증거는 제출하지 말 것.

- 위험과 피할 것:
  - `auth.go` 는 위험 구역이다. **세션 쿠키·토큰 교환·ID 토큰 검증·`oidc_flows` 쿼리·`randomToken`·`requireRole` 은 한 줄도 건드리지 말 것.** 이번 변경은 두 군데의 **에러 분기 응답**에 한정된다.
  - 오류 코드 문자열은 프런트·테스트 계약이다. `oidc_disabled` 를 **고치지 말고**, 새 코드 `oidc_unavailable` 을 추가하는 방향으로만 갈 것. 먼저 `rg -n 'oidc_disabled' web/ internal/ docs/ sdk/` 로 현재 사용처(확인된 것: `web/src/api/errorMessages.ts:26`, `auth.go:214`, `auth.go:304` — 테스트 사용처는 없음)를 다시 확인하고, 기존 테스트가 404/400 을 기대하면 그 기대를 **깨지 않게** 둘 것.
  - `migrations/` 와 `.github/workflows/` 는 손대지 말 것(체크섬·릴리즈 게이트).
  - `loadAISetting`(admin.go:438)과 `ai.go:16` 도 똑같은 모양이지만 **이번 회차에 함께 고치지 말 것** — 파일 수가 늘고 AI 설정의 응답 계약은 별개다. 아이디어로 남긴다.
  - `loadOIDCSetting` 의 반환값 주의: Open 실패 시 `cfg.ClientSecret` 에는 **봉인된 ciphertext 가 그대로 남아** 돌아온다(auth.go:201-206). 에러를 무시하는 호출자를 새로 만들지 말 것. 로그·감사에 이 값이나 평문 secret 을 절대 넘기지 말 것 — err 메시지만 넘긴다(`secretbox.go:46` 의 `"decrypt secret: %w"` 는 평문을 담지 않음을 확인했다).
  - 확인됨: `func (s *Server) serverError(w http.ResponseWriter, r *http.Request, status int, code, message string, cause error)`(api.go:808) 이고 내부에서 `s.logRequestError`(api.go:813) 를 호출한다. 로거는 `Server.Log *slog.Logger`(api.go:43)이며 `s.Log == nil` 이면 조용히 넘어간다 — 수용 기준 2) 를 보려면 테스트에서 `s.Log = slog.New(slog.NewTextHandler(&buf, nil))` 를 꼭 넣을 것(미설정이면 로그가 비어 있는 것이 정상이라 오판한다).
  - 기존 테스트 주의: `internal/api/silent_sso_test.go:43` 과 `silent_sso_pg_test.go:56` 이 `/api/v1/auth/oidc/login` 을 호출한다. 두 파일이 어떤 상태코드를 기대하는지 먼저 읽고, 정상 disabled 경로가 그대로 404 로 남는지 확인할 것(이 두 파일을 수정해야 한다면 과제 전제가 틀렸다는 뜻이므로 멈추고 재검토).

- 차선 후보: secretbox 의 거부 경로 회귀 보강 — `internal/secretbox/secretbox_test.go` 가 왕복과 nonce 차이 2개 단정뿐이고, `Open` 의 세 거부 경로(`v1:` 접두사 없음 / base64 깨짐 / `len(data) < NonceSize` / ciphertext 1비트 변조 / 다른 키로 Open)가 전부 미검증이다. 실제 `secretbox.New`·`Seal`·`Open` 으로 DB 없이 검증 가능하고 파일 1개(테스트)만 늘며, 설치키가 암호화하는 것이 OIDC client secret 과 AI API 키라는 점에서 값이 있다. 명령: `go test ./internal/secretbox/... -count=1 -v`. 단 결함 수정이 아니라 테스트 공백 보강이므로 Red 는 없다 — 그 사실을 보고에 적을 것.
