- 과제: 코드 교환이 `RedeemAuthorizationCode` 의 저장소 장애를 `invalid_grant` 로 답하는 것을 500 `server_error` 로 (가치 3 / 위험 2 / 작업량 M)
- 왜: `handleAuthorizationCodeGrant`(`internal/httpserver/oidc.go:517`)가 `RedeemAuthorizationCode` 의 **모든** 오류를 400 `invalid_grant` "authorization code is invalid or expired" 로 합친다 — `authorization_codes` 테이블을 읽지 못한 트랜잭션 오류(Begin·Scan·Exec·Commit)까지 포함해서다. `invalid_grant` 는 RP 라이브러리에게 "이 코드는 죽었다" 는 뜻이고 표준 동작이 로그인을 처음부터 다시 시작하는 것이라, 읽히지도 않은 코드 때문에 사람이 같은 장애로 되돌아간다. 이 자리는 같은 파일 `oidc.go:578-585` 의 주석이 v0.9.95 때 **명시적으로 "아직 안 한 변경"** 으로 지목해 둔 두 곳 중 하나다("the code redemption above answers invalid_grant for every `RedeemAuthorizationCode` failure … Those are separate changes and are still to make").
- 수용 기준:
  1) `authorization_codes` 를 읽을 수 없는 동안 코드 교환이 **500 `server_error`** 로 답하고 본문에 `invalid_grant` 가 없다. 전용 ERROR 로그 한 줄이 남고(`trace_id`·`realm`·`client`·`grant_type=authorization_code`·`error`), `code` 원문·`code_verifier` 는 로그·지표 어디에도 없다.
  2) `resso_token_errors_total{grant_type="authorization_code"}` 가 1 증가한다(이 라벨은 이 지점에서 이미 확정되어 있으므로 정직하다 — `oidc.go:565` 가 같은 라벨을 쓰는 선례).
  3) **호출자 잘못은 400 `invalid_grant` 그대로**: ① 없는 코드 ② 두 번 쓴 코드(`ErrCodeReuse` — 기존 `AUTHORIZATION_CODE_REUSED` 감사·Warn 로그도 그대로) ③ `redirect_uri` 불일치 ④ **PKCE `code_verifier` 불일치**. 네 경우 모두 새 로그 줄 0개·새 지표 증가 0.
  4) 테이블을 되돌린 뒤 **같은 코드가 그대로 200** 으로 교환된다(코드는 소비되지 않았음을 증명 — 이것이 "retry 해도 된다" 는 본문의 근거다).
  5) 새 연동 테스트가 **수정 전 `oidc.go` 에서 실제로 실패**하는 것을 먼저 확인하고 그 출력을 회차 노트에 남긴다.

- 건드릴 파일 (프로덕션 1개):
  - `internal/httpserver/oidc.go:508-516` — `RedeemAuthorizationCode` 에 넘기는 validate 콜백. PKCE 분기(`513`행 `ressooidc.ValidatePKCE`)가 돌려주는 오류를 **센티널로 감싼다**. 같은 파일 위쪽(`realmLookupFailed`·`userLookupFailed` 근처)에 패키지 수준 센티널 하나를 새로 선언:
    ```go
    // errCodeRefused marks a rejection this handler's own validate callback made,
    // so that a store fault RedeemAuthorizationCode returns unchanged can be told
    // apart from a code this handler refused. ValidatePKCE returns a plain error
    // and the mismatch above returns store.ErrNotFound; only this one needed a
    // mark of its own.
    var errCodeRefused = errors.New("the authorization code was refused")
    ```
    콜백: `if pkceErr := ressooidc.ValidatePKCE(...); pkceErr != nil { return fmt.Errorf("%w: %w", errCodeRefused, pkceErr) }; return nil`
  - `internal/httpserver/oidc.go:517-532` — `if err != nil` 블록. 기존 `ErrCodeReuse` 분기(`521`)는 **그대로 두고**, 400 을 쓰는 `530`행 **앞**에 이쪽 장애 분기를 넣는다:
    ```go
    if !errors.Is(err, store.ErrNotFound) && !errors.Is(err, store.ErrCodeReuse) && !errors.Is(err, errCodeRefused) { … 500 … }
    ```
    본문 문안은 아는 것만 말할 것(선례 `oidc.go:540-542`): `the authorization code could not be looked up; it has not been consumed, retry the sign-in after a short delay`.
  - 문서: `docs/operations.md` 의 `resso_token_errors_total` 불릿에 한 문장(이 실패도 이 계열에 세며 500 이고, 찾을 로그 문구는 무엇이고, 없는·만료된·재사용된·PKCE 불일치 코드는 400 으로 남아 세지 않는다). **먼저 그 불릿을 읽고** v0.9.93~95 의 세 문장과 겹치지 않는지 확인할 것. `docs/ADMIN_GUIDE.md:570` 은 이미 `{grant_type=…}` 로 맞으니 손대지 말 것. PDF·캡처 재생성 없음.
  - 테스트: `internal/httpserver/integration_test.go` 에 새 테스트 하나.

- 왜 이 분류식이 맞는가 (실제로 읽고 확인한 것):
  - `store/oidc.go:118-127 scanAuthorizationCodeWithState` 가 `pgx.ErrNoRows` → `store.ErrNotFound` 로 바꾼다. **없는 코드는 `ErrNotFound` 다.**
  - `store/oidc.go:180-182` 만료 코드 → `ErrNotFound`. `oidc.go:509-511` realm/client/redirect_uri 불일치 → `store.ErrNotFound`.
  - `internal/oidc/pkce.go:10-25 ValidatePKCE` 는 **평범한 `errors.New` 넷**을 돌려준다(센티널 없음). 그래서 `!errors.Is(err, store.ErrNotFound)` 만으로 가르면 **PKCE 불일치가 500 이 된다** — 수용 기준 3④가 바로 이 함정을 막는 단언이다. 센티널 래핑(또는 콜백에서 `store.ErrNotFound` 로 감싸기) 없이는 이 과제를 할 수 없다.
  - 남는 raw 오류는 `store/oidc.go` 의 `Pool.Begin`(146)·`Scan`(163)·소비 `Exec`(186)·`Commit`(189), 그리고 재사용 처리 중의 `Exec`/`Commit`(169-175)뿐이다 — 모두 이쪽 장애다.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `eval "$(scripts/test-services.sh)"` — 출력이 비면 준비 실패이므로 eval 하지 말 것. **같은 셸에서** 아래를 돌린다.
  - 실패 재현(수정 전) → `go test -race ./internal/httpserver -run '^TestIntegrationToken' -count=1 -v`
  - `make lint` (golangci-lint + govulncheck + ESLint --max-warnings 0)
  - `make test` — 끝의 `"integration test(s) did not run"` 경고가 **없는지 눈으로 확인**(있으면 SKIP 이고 아무것도 증명되지 않았다). httpserver 약 110~125s.
  - 마무리: `webui/dist/index.html` 복원, `git diff --check`.

- 테스트 작성 요령 (선례를 그대로 쓸 것):
  - 뼈대는 `integration_test.go:1130 TestIntegrationTokenSaysWhenItCouldNotReadTheAccount` 를 복사. 이름 예: `TestIntegrationTokenSaysWhenItCouldNotRedeemTheCode`.
  - 코드 채굴은 `1174-1202` 블록 그대로 — 쿠키 세션 + 실제 `/realms/master/protocol/openid-connect/auth` 302 의 `Location` 에서 `code` 를 뽑는다. **이 블록이 이미 `code_challenge`/`code_verifier` 를 쓴다** → 같은 코드에 틀린 verifier 를 보내 수용 기준 3④를 그대로 검사할 수 있다. 코드는 두 개 이상 채굴할 것(장애용·복구용·PKCE용은 각각 다른 코드여야 한다 — 한 번 쓰면 소비되거나 재사용으로 잡힌다).
  - 장애 주입은 `ALTER TABLE authorization_codes RENAME TO authorization_codes_hidden` + `t.Cleanup` 복구(선례 `1234-1247`). `authorization_codes` 는 Realm 조회·Client 인증·`/auth` 발급이 지나간 **뒤**에만 필요하므로 교환 직전에 숨기면 된다. **단언 전에 즉시 복구**할 것 — 코드 수명이 짧다(`store/oidc.go:152-157` 의 90초 창).
  - 로그는 `lockedBuffer`(`integration_test.go:7287`), 지표는 `New(data, logger, nil, nil)` + `srv.Metrics().WritePrometheus(&sb)`(nil 등록기를 `New` 가 스스로 채우는 프로덕션 배선, `server.go:64-68`·`80`) — v0.9.96 이 이 방식으로 읽었다.

- 위험과 피할 것:
  - **PKCE 불일치를 500 으로 만들지 말 것.** 위에 적은 유일한 실제 함정이다.
  - `ErrCodeReuse` 분기(`oidc.go:521-529`)의 감사·Warn·400 을 한 글자도 바꾸지 말 것. 단, `store/oidc.go:169-175`(재사용 감지 후 family 취소 Exec/Commit 실패)는 raw 오류를 돌려주므로 **새 500 통에 들어간다** — 그 경우 기존에도 `AUTHORIZATION_CODE_REUSED` 감사는 남지 않았다(`errors.Is(ErrCodeReuse)` 가 거짓이므로). 이 회차에서 `store` 에 `ErrFamilyNotRevoked` 같은 것을 새로 만들지 말고(별도 회차), 주석 한 줄로 "이 경로도 여기로 온다" 를 밝히는 선에서 끝낼 것. 테스트로 재현하지 않아도 된다(재현 난이도 높음 — 미확인).
  - `RotateRefreshToken` 쪽(`oidc.go:578-585` 가 지목한 다른 한 곳)은 **이번에 건드리지 말 것**. 확인해 보니 `store/oidc.go:306-312` 가 세션 생존 질의의 오류까지 `ErrNotFound` 로 합쳐 버려 store 변경이 필요하고, 게다가 `InspectRefreshToken` 과 같은 `refresh_tokens` 테이블을 읽어 RENAME 으로는 새 분기에 도달할 수 없다(v0.9.95 분기가 먼저 잡는다). 별도 회차다.
  - `auth.go`·`middleware.go`·`internal/store/migrations`·`.github/workflows` 는 건드리지 않는다. `store/oidc.go` 도 이번에는 읽기만.
  - RENAME 한 테이블을 복구하지 않으면 같은 컨테이너를 쓰는 뒤따르는 테스트가 전멸한다 — `t.Cleanup` 필수.
  - 상태 코드를 추측하지 말 것(과거 교훈). 400→500 외의 응답은 한 글자도 바뀌지 않아야 한다.

- 차선 후보: 기존 Token 연동 테스트 둘(`TestIntegrationTokenSaysWhenItCouldNotReadTheAccount:1130`·`…ReadTheRealm`)이 `New(data, logger, nil, nil)` 로 서버를 세워 v0.9.93·v0.9.94 가 더한 `resso_token_errors_total{grant_type}` 배선을 아무 테스트도 고정하지 않는다 — `srv.Metrics().WritePrometheus` 로 그 둘에 단언을 더한다(가치 2 / 위험 1 / 작업량 S, **프로덕션 파일 0개**).
