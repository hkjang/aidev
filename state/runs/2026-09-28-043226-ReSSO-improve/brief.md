- 과제: refresh 그랜트가 `refresh_tokens` 조회 장애를 `invalid_grant`로 답하는 것을 500 `server_error`로 (가치 3 / 위험 2 / 작업량 S)

- 왜: `handleRefreshGrant`(`internal/httpserver/oidc.go:576-580`)가 `InspectRefreshToken`의 **모든** 오류를 400 `invalid_grant` "refresh token is invalid or expired"로 합친다. `refresh_tokens`를 읽지 못한 것과 "그런 토큰 없음"이 같은 답이 되어, RP는 장애가 끝나면 그대로 살았을 refresh token을 버리고 사람을 다시 로그인시킨다. 이 저장소가 세 회차 연속 고쳐 온 바로 그 부류이고, **`InspectRefreshToken` 호출자 셋 중 이곳만 구분하지 않는다** — introspection(1016행)은 `recordUnjudgedIntrospection(r, "refresh_token", inspectErr)`로, revocation(1106-1109행)은 `inspectErr != nil && !errors.Is(inspectErr, store.ErrNotFound)` → 503 `temporarily_unavailable`로 이미 갈랐다. 게다가 이 자리는 그 판정의 근거가 가장 강한 자리다: 576행은 이 그랜트의 **첫** 저장소 호출이라 회전(`RotateRefreshToken`)은커녕 아무것도 건드려지지 않았고, 호출자가 쥔 토큰은 장애 내내 그대로 살아 있다. 같은 논지가 이미 **두 줄 아래 597행**(`userLookupFailed`, v0.9.94)과 아래 `ErrNoActiveSigningKey` 분기에 있고, 이 Endpoint에서 남은 마지막 자리다.

- 수용 기준:
  1. `refresh_tokens`를 읽을 수 없을 때(= `store.ErrNotFound`가 **아닌** 오류) `POST …/protocol/openid-connect/token` `grant_type=refresh_token`이 **500 `server_error`**로 답하고, 본문에 `invalid_grant`가 **없다**. 본문 설명은 토큰이 검증됐다고 주장하지 말고 "건드려지지 않았다"만 말할 것 — 권장 문안: `the refresh token could not be looked up; it has not been rotated or rejected, retry after a short delay` (597-601행의 문안 및 주석 밀도를 그대로 따를 것).
  2. 그 실패가 **한 줄의 `ERROR` 로그**로 남는다 — `trace_id`·`realm`(=`realm.Name`)·`client`(=`client.ClientID`)·`grant_type`(=`refresh_token`)·`error`. **`raw` 토큰 값은 로그·감사·지표에 절대 넣지 말 것**(식별자만; `inspected`는 실패했으므로 읽지 말 것).
  3. `resso_token_errors_total{grant_type="refresh_token"}`이 1 증가한다. 여기서는 그랜트 타입을 이미 읽었으므로 정직한 라벨이 있다(`userLookupFailed`가 oidc.go:94에서 같은 판단을 하고, Realm 조회 실패는 그렇지 못해 일부러 세지 않았다 — 475행 주변 주석). 라벨을 새로 추가하지 말 것.
  4. **없는 토큰·쓰레기 값·만료·다른 Realm/Client의 토큰은 400 `invalid_grant` "refresh token is invalid or expired"를 한 글자도 바꾸지 않고, 새 로그 줄도 지표도 0이다.** 577행의 `inspected.RealmID != realm.ID || inspected.ClientID != client.ID || inspected.UserID == nil` 불일치 분기와 `err != nil`(= `ErrNotFound`) 경로는 그대로 남긴다. 새 판정은 577행 **앞**에 두고, 그 뒤 기존 조건식은 손대지 않는 것이 가장 작은 diff다.
  5. 테스트가 증명할 것: **테이블을 복구한 뒤 같은 refresh token이 그대로 200을 받는다**(이 과제의 전부 — 장애가 세션을 죽이지 않았다는 증거). 그리고 수정 전 `oidc.go`에서 새 테스트의 단언이 **실제로 실패**하는 것을 먼저 확인해 회차 노트에 실패 출력을 붙일 것.

- 건드릴 파일 (프로덕션 1개):
  - `internal/httpserver/oidc.go:576-580` — `handleRefreshGrant`의 `InspectRefreshToken` 오류 처리를 갈라 `errors.Is(err, store.ErrNotFound)`가 아닌 오류에만 500 + 로그 + 지표. 호출자가 한 곳뿐이므로 **새 헬퍼를 만들지 말고 인라인 분기**로 둘 것(`realmLookupFailed`/`userLookupFailed`는 호출자가 여럿이어서 헬퍼가 됐다). 다만 "왜 이 답인가"를 담은 주석은 이 저장소의 밀도대로 길게 쓸 것 — 597-601행과 46-92행이 본보기.
  - `internal/httpserver/integration_test.go` — 새 연동 테스트 `TestIntegrationTokenSaysWhenItCouldNotLookUpTheRefreshToken`(이름은 자유). 본보기는 **`TestIntegrationTokenSaysWhenItCouldNotReadTheAccount`(1130행)** — 그대로 베낄 수 있다: `openHTTPIntegrationStore(t)` → `Bootstrap` → `EnsureActiveSigningKey` → `RealmByName("master")` → `CreateClient`(grant_types에 `refresh_token`) → `CreateUser` → `CreateSession` → **`ressooidc.Service{Store: data}.IssueUserTokens(...)`로 진짜 refresh token 발급**(반드시 RENAME **전에**) → 로그는 `lockedBuffer`(정의 `integration_test.go:7287`) → 지표를 단언하려면 `metrics := observability.NewRegistry()`를 만들어 `New(data, logger, nil, metrics)`로 넘기고 `metrics.WritePrometheus(&exported)`로 읽을 것(선례 `integration_test.go:2199·2229`; 1130행 테스트는 `New(data, logger, nil, nil)`이라 등록기를 못 읽는다).
  - 선택: `docs/operations.md`의 `resso_token_errors_total` 불릿에 한 문장(이 실패도 이 계열에 세며 찾는 로그 문구). 지난 두 회차가 같은 자리를 이미 손봤으므로 **문장이 중복되지 않는지 먼저 읽을 것**. PDF·캡처 재생성은 하지 말 것.

- 장애 재현 방법(확인된 것): `ALTER TABLE refresh_tokens RENAME TO refresh_tokens_hidden` + `t.Cleanup`으로 되돌리기. 선례 다수(profile의 2898·3700·7144·8092, 그리고 `users`/`realms`를 쓴 직전 두 회차). **`refresh_tokens`를 참조하는 외래키는 자기 참조 하나뿐**(`internal/store/migrations/001_initial.sql:168` `parent_id uuid REFERENCES refresh_tokens(id)`)이라 RENAME이 함께 따라가고 다른 테이블은 걸리지 않는다 — 직접 확인함. 토큰 Endpoint가 576행에 닿기까지 읽는 것은 `realms`·`clients`·`rate_limits`뿐이라 부수 실패가 없다.

- 테스트가 검사할 경우(권장 네 가지):
  - (a) 테이블 은닉 중: 500 `server_error`, 본문에 `invalid_grant` 없음, 새 로그 줄 정확히 1개(`grant_type=refresh_token`), `resso_token_errors_total{grant_type="refresh_token"} 1`.
  - (b) `t.Cleanup` 아니라 테스트 안에서 이름을 되돌린 뒤: **같은 raw refresh token으로 200** + 새 access token 발급.
  - (c) 테이블 정상 + 존재하지 않는 refresh token 값: 400 `invalid_grant` "refresh token is invalid or expired", 새 로그 줄 0, 지표 증가 0.
  - (d) 다른 Client가 발급받은 refresh token으로 교환: 여전히 400 `invalid_grant`, 새 로그 줄 0(불일치 분기 무변경의 증거).

- 검증 명령:
  - `eval "$(scripts/test-services.sh)"` — 출력이 비면 준비 실패이므로 eval 하지 말 것. **같은 셸에서** 아래를 돌린다.
  - 단일: `go test -race ./internal/httpserver -run '^TestIntegrationTokenSays' -count=1 -v` (SKIP 0 확인. 이 접두어로 직전 두 회차의 형제 테스트도 같이 돈다 — 회귀 확인에 좋다.)
  - 전체: `make lint` (0 issues), `make test` (exit 0, 끝에 `integration test(s) did not run` 경고가 **없어야** SKIP 0. httpserver 약 110~120s).
  - `webui/dist/index.html`이 빌드로 바뀌면 되돌리고 `git diff --check`.

- 위험과 피할 것:
  - **577행의 불일치 조건식과 `ErrNotFound` 경로를 재구성하지 말 것.** `InspectRefreshToken`은 두 번째 반환값 `active`를 576행에서 `_`로 버리고 있는데(만료·회전·취소 판정은 뒤의 `RotateRefreshToken`이 한다) **이것은 이번 과제가 아니다.** 눈에 걸려도 건드리지 말 것 — 만료 토큰의 현재 답을 바꾸면 계약 변경이 된다.
  - introspection(1016행)·revocation(1106행)은 이미 갈라져 있으니 **손대지 말 것**. 세 경로를 하나의 헬퍼로 통합하려 하지 말 것(답이 각각 200/503/500으로 계약이 다르다 — 운영자 규칙: 계약이 다른 파서·경로를 통합하지 말 것).
  - 감사(`audit`) 항목은 더하지 말 것. 이 실패 시점에 사용자·세션이 특정되지 않았고, 직전 두 회차도 이 자리에는 로그+지표만 남겼다.
  - RENAME cleanup 누락 = 뒤따르는 테스트 전멸. `t.Cleanup`으로 반드시 되돌릴 것.
  - 보호 경로(auth.go, middleware.go, `internal/store/migrations`, `.github/workflows`)는 건드릴 일이 없다.
  - grep 결과를 증거로 제출하지 말 것 — 수정 전 실패 출력과 수정 후 PASS 출력이 증거다.

- 차선 후보: **UserInfo 거절 카운터 `resso_userinfo_errors_total`** — `writeUserInfoUnavailable`(oidc.go:855 부근)의 stage 여섯을 로그로만 남긴다. `errors_total` 계열(token·introspection·authorization·silent)에서 userinfo만 없다. `metrics.go`에 `registry.Counter(...)` 한 줄 + 호출 한 줄 + 연동 테스트. 라벨은 코드에 이미 있는 고정 stage 집합만(미검증 입력 금지).
