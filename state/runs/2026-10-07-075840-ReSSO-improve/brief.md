- 과제: 코드 재사용 감지의 폐기 실패가 `AUTHORIZATION_CODE_REUSED` 감사 항목을 통째로 잃는 것을 막기 — `RotateRefreshToken` 이 이미 쓰는 `ErrFamilyNotRevoked` 관용구를 `RedeemAuthorizationCode` 에도 적용 (가치 3 / 위험 2 / 작업량 M)
- 왜: `store/oidc.go:166-177` 은 재사용을 감지한 뒤 그 응답(세션·Client 범위 refresh token 폐기)이 실패하면 `return AuthorizationCode{}, err` 로 **raw 오류**를 돌려주고, `httpserver/oidc.go:554` 의 `errors.Is(err, store.ErrCodeReuse)` 가 거짓이 되어 감사 `AUTHORIZATION_CODE_REUSED`·Warn `authorization code replayed` 가 **둘 다 사라진다** — 코드가 유출됐고 그 대응이 실패했는데 운영자에게는 흔한 400 `invalid_grant` 하나만 보인다(응답은 어차피 같다). 같은 파일의 형제 경로 `RotateRefreshToken`(295·299)은 이 상황에 `fmt.Errorf("%w: %w: %v", ErrTokenReuse, ErrFamilyNotRevoked, err)` 로 감싸고 `httpserver/oidc.go:687-705` 가 그것을 읽어 감사를 남기고 `family_revoked:false` 를 적는다 — 이 과제는 **그 관용구를 코드 경로에 그대로 옮기는 것**이다.
- 수용 기준:
  1) 재사용 분기의 폐기 `Exec`(169-170) 또는 `Commit`(173)이 실패해도 호출자가 받는 오류에 `errors.Is(err, store.ErrCodeReuse)` 가 **참**이고, `errors.Is(err, store.ErrFamilyNotRevoked)` 로 "폐기가 안 됐다" 를 구분할 수 있다.
  2) 그 경우에도 `AUTHORIZATION_CODE_REUSED` 감사 1건이 남고(target `client`, `actor` = 코드 주인 계정), detail 에 기존 `session_id` 와 함께 폐기가 안 됐다는 표시(`tokens_revoked:false`)가 있으며, ERROR 한 줄이 그 사실을 사유와 함께 남긴다. **HTTP 응답은 400 `invalid_grant` "authorization code is invalid or expired" 로 한 글자도 바뀌지 않는다.**
  3) 폐기가 **성공한** 평범한 재사용은 지금과 완전히 동일하다: 감사 1건, Warn `authorization code replayed`, detail 에 `tokens_revoked` 키 없음(또는 true), 400.
  4) 연동 테스트가 프로덕션 배선(`New(data, logger, nil, nil)` + httptest + 실제 `/auth` 302 `Location` 에서 코드 채굴)으로 ①②를 증명한다. **장애 주입은 테이블 은닉이 아니라 컬럼 이동이다**: `ALTER TABLE refresh_tokens RENAME COLUMN revoked_at TO revoked_at_moved` — 이 저장소가 바로 이 목적(refresh token 폐기만 실패시키기)으로 이미 쓰는 수단이고 선례가 같은 파일에 넷 있다(`integration_test.go:5847-5862`·5959·6076·6206, `restored` 플래그 + `restore()` + `t.Cleanup(restore)` 꼴). 재사용 분기의 `UPDATE … SET revoked_at=COALESCE(revoked_at,now())` 만 깨지고 코드 조회(`authorization_codes`)·감사 기록·첫 교환의 refresh token `INSERT`(컬럼을 명시 열거하며 `revoked_at` 을 포함하지 않는다)는 그대로 돈다 — 테이블을 통째로 숨기는 것보다 훨씬 좁다. **첫 교환이 성공한 뒤**에 걸고 단언 전 즉시 복구.
  5) `go test -race ./internal/store -count=1` 의 기존 코드 재사용 테스트(`internal/store/integration_test.go:1931`, `:132`)가 그대로 통과한다 — 폐기가 되는 경로의 반환값은 바뀌지 않는다.
- 건드릴 파일 (프로덕션 2개):
  - `internal/store/oidc.go:166-177` — `RedeemAuthorizationCode` 재사용 분기. 171·174 의 `return AuthorizationCode{}, err` 를 `return code, fmt.Errorf("%w: %w: %v", ErrCodeReuse, ErrFamilyNotRevoked, err)` 로. **`code` 를 꼭 함께 돌려줘야 한다**(지금은 zero value 를 돌려준다): 호출부가 `code.UserID`·`code.SessionID.String()` 을 쓰므로 zero value 면 nil `*uuid.UUID` 역참조로 panic 한다. 176 의 `return code, ErrCodeReuse` 는 손대지 않는다.
  - `internal/store/oidc.go:230-234` — `ErrFamilyNotRevoked` doc. 지금은 "accompanies ErrTokenReuse" 라고만 적혀 있다. 두 재사용 센티널 양쪽에 붙으며, 코드 재사용에서 남는 것은 family 가 아니라 **그 세션·Client 의 refresh token** 이라는 것을 한 문장 더한다. (차선: `ErrTokensNotRevoked` 같은 새 센티널을 두는 것 — 이름이 더 정확하지만 같은 개념의 어휘가 둘이 되고 `errors.Is` 관용구를 복제한다. 재사용을 권한다.)
  - `internal/httpserver/oidc.go:550-564` — `handleAuthorizationCodeGrant`. `ErrCodeReuse` 분기 안에서 `errors.Is(err, store.ErrFamilyNotRevoked)` 일 때 detail 에 `"tokens_revoked": false` 를 더하고 ERROR 한 줄(`s.logger.Error`, 메시지는 687-705 의 `refresh token reuse detected but its family was not revoked` 와 같은 꼴, 필드는 `trace_id`·`realm`·`client`·`error`)을 남긴다. **551-553 주석("the store has already revoked what it could")이 이 변경으로 틀린 말이 되므로 함께 고친다.** `code` 원문·`code_verifier` 는 로그·감사 어디에도 넣지 않는다.
  - `internal/httpserver/integration_test.go` — 새 테스트 1개. 코드 채굴 선례는 PKCE 포함해 1174-1202 부근, 로그 단언은 `lockedBuffer`(7287 부근).
  - 문서: `docs/operations.md` 의 `AUTHORIZATION_CODE_REUSED` 설명이 있으면 "폐기가 실패한 경우" 한 문장. 없으면 손대지 말 것(미확인 — 먼저 grep 할 것).
- 검증 명령:
  - `eval "$(scripts/test-services.sh)"` 를 먼저, **같은 셸에서** 아래를 돌린다(출력이 비면 준비 실패이므로 eval 하지 말 것).
  - `go test -race ./internal/httpserver -run '^TestIntegration' -count=1`
  - `go test -race ./internal/store -count=1` (기준 2·5)
  - `make lint` (golangci-lint + govulncheck + eslint, 0 issues)
  - `make test` (exit 0, 끝의 `"integration test(s) did not run"` 경고가 Makefile 레시피 에코 1행뿐임을 눈으로 확인 = SKIP 0)
  - 끝나고 `webui/dist/index.html` 복원, `git diff --check`.
- 위험과 피할 것:
  - **2026-10-01 의 `5bed9dc` 접근(코드 교환의 저장소 장애를 500 `server_error` 로)을 다시 올리지 말 것.** 사람이 받지 않아 main 에 없다. 이 과제는 **응답을 한 글자도 바꾸지 않는다**(400 유지) — 500 이나 새 `errCodeRefused` 센티널로 번지면 그 기각된 접근의 재제출이 된다. `oidc.go:563` 의 400 줄과 612-618 주석 문단은 그대로 둔다(그 주석은 "아직 안 한 변경" 을 적고 있고 이 과제는 거기 적힌 것이 아니다).
  - `ValidatePKCE` 는 센티널 없는 `errors.New` 넷을 돌려준다 — validate 콜백은 **건드리지 말 것**. 이번 변경은 validate 앞의 재사용 분기만 본다.
  - 테이블 통째 은닉(`RENAME TO refresh_tokens_hidden`, 선례 1429·10036)은 쓰지 말 것 — 첫 교환의 refresh token 발급까지 같이 깨뜨려 테스트 순서가 취약해진다. 컬럼 이동이 더 좁다(수용 기준 4).
  - 감사가 **프로덕션 배선으로** 남는 것을 증명할 것. `AUTHORIZATION_CODE_REUSED` 를 단언하는 httpserver 연동 테스트는 **지금 하나도 없다**(grep 결과 `oidc.go:558` 한 곳뿐) — 그래서 수용 기준 3(폐기 성공하는 평범한 재사용)도 같은 테스트에서 함께 고정해 두는 것이 값이 크다.
  - 보호 경로를 피한다: `internal/store/migrations`·`auth.go`·`middleware.go`·`.github/workflows` 는 건드리지 않는다.
  - `metrics.go` 는 건드리지 않는다(새 계열 없음). 감사와 로그로 충분하다 — v0.9.99·v0.9.100 이 "같은 사건에 로그 두 줄" 을 피한 판단과 달리 여기서는 `clientAuthUndecided` 같은 선행 로그가 없으므로 ERROR 한 줄이 중복이 아니다.
  - gofmt 정렬로 lint 첫 실행이 깨지는 것이 이 저장소의 상습 지점이다.
- 차선 후보: `store/oidc.go:306-312` 가 세션 생존 질의 오류(`err != nil || !sessionActive`)를 `ErrNotFound` 로 합치는 것을 가르기 — 질의 오류와 "세션이 끝났다" 를 분리하고 refresh 그랜트가 전자에 500 으로 답하게 한다. `sso_sessions` RENAME 이 그 분기에 닿는지 확인이 선행이고(미확인), 이쪽이 성립하지 않을 때만 고를 것.
