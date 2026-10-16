- 과제: refresh 그랜트가 `RotateRefreshToken` 의 저장소 장애를 400 `invalid_grant` 로 답해 살아 있는 Refresh Token 을 RP 가 버리게 만드는 것을 500 `server_error` 로 가르기 (가치 3 / 위험 2 / 작업량 M)
- 왜: `internal/httpserver/oidc.go:725-746` 은 `RotateRefreshToken` 의 **모든** 오류를(ErrTokenReuse 외) 400 `invalid_grant` "refresh token is invalid or was already used" 로 답한다. 그 오류에는 트랜잭션 Begin/Scan/Exec/Commit 실패와 `internal/store/oidc.go:323-329` 가 `ErrNotFound` 로 합쳐 버리는 **세션 생존 질의의 실패**가 들어 있는데, 그 시점에는 `rotated_at` UPDATE(`store/oidc.go:340`)가 아직 돌지 않았고 트랜잭션은 defer 로 롤백되므로 **호출자가 든 Token 은 그대로 살아 있다**. 그런데 `invalid_grant` 는 "그 Token 은 죽었다" 는 뜻이라 RP 는 그것을 버리고 사람을 로그인으로 보내므로, 몇 초짜리 `sso_sessions` 장애가 살릴 수 있었던 세션을 영구히 가져간다. 이 저장소는 같은 논증을 바로 위 `InspectRefreshToken` 경로(648-688)와 `userLookupFailed` 경로(711-719)에서 이미 500 으로 구현해 놓았고, **648 의 주석 653-657 이 "RotateRefreshToken below … Those are separate changes and are still to make" 라고 이 과제를 직접 지목한다**(실제로 읽어 확인했다).
- 수용 기준:
  1) `sso_sessions` 를 읽을 수 없는 동안 올바른 Refresh Token 으로 보낸 refresh 그랜트가 400 이 아니라 **500 `server_error`** 로 답하고, 본문 description 이 "아직 회전되지도 거절되지도 않았으니 잠시 뒤 재시도하라" 는 뜻을 담는다(기존 686 행 문안과 같은 취지, 단계 구별이 가능한 문구).
  2) 그 실패가 `resso_token_errors_total{grant_type="refresh_token"}` 에 집계되고(기존 계열·라벨 재사용, `metrics.go` 무변경) 서버 로그에 새 ERROR 한 줄(`realm`·`client`·`grant_type`·`error` 포함, **raw refresh_token·Secret 미노출**)이 남는다.
  3) 장애가 걷힌 뒤 **같은 Refresh Token 이 그대로 200 으로 교환된다** — 즉 실패한 시도가 Token 을 회전·폐기하지 않았음을 테스트가 증명한다(이 과제의 핵심 주장).
  4) 평범한 거절은 한 글자도 바뀌지 않는다: 없는/만료된/다른 Realm·Client 의 Token, 끝난 세션(`!sessionActive`), 꺼진 계정 → 여전히 400 `invalid_grant` + 기존 문안. 재사용(`ErrTokenReuse`)은 여전히 400 + `REFRESH_TOKEN_REUSE` 감사.
  5) 기존 테스트 무수정 통과.
- 건드릴 파일 (프로덕션 2개):
  - `internal/store/oidc.go:323-329` `RotateRefreshToken` — `if err := tx.QueryRow(...).Scan(&sessionActive); err != nil || !sessionActive { return ErrNotFound }` 의 통짜 조건을 가른다. 질의 실패는 **원래 오류를 그대로 돌려주고**(새 센티넬을 만들 필요 없음 — 호출부가 "ErrNotFound 도 ErrTokenReuse 도 아닌 것" 으로 판정한다), `!sessionActive` 만 `ErrNotFound` 로 남긴다. 왜 가르는지 주석 한 단락(끝난 세션은 진짜 답, 읽지 못한 것은 아무 판정도 아니라는 것 + 이 시점에 아무것도 써지지 않았다는 것).
  - `internal/httpserver/oidc.go:725-746` `handleRefreshGrant` — `ErrTokenReuse` 분기 **뒤**, 400 을 쓰는 745 행 **앞**에 `if !errors.Is(err, store.ErrTokenReuse) && !errors.Is(err, store.ErrNotFound) { s.metrics.Add(metricTokenErrors, 1, "refresh_token"); s.logger.Error("the refresh token could not be rotated", …); writeOAuthError(w, http.StatusInternalServerError, "server_error", …); return }` 를 둔다. 순서 주의: `ErrFamilyNotRevoked` 포장 오류는 `ErrTokenReuse` 도 함께 감싸므로(`store/oidc.go:312·316`) 반드시 재사용 분기가 먼저 걸려야 하고 새 분기에 들어가서는 안 된다.
  - 같은 파일 `:653-657` 의 주석 — "Those are separate changes and are still to make" 가 이 변경으로 **틀린 말이 되므로 고친다**(남은 것이 `RedeemAuthorizationCode` 쪽뿐이라는 사실로. 코드 교환 500 변경은 사람이 반려했으므로 "하지 않기로 한 것" 으로 적을 것 — 아래 위험 참조).
  - `internal/httpserver/integration_test.go` — 새 연동 테스트 1개(아래).
  - `docs/operations.md` — `resso_token_errors_total` 불릿(84-95 구간, "Refresh Token 자체를 조회하지 못한 경우도 이 계열에 셉니다" 문장 옆)에 **회전 실패도 이 계열·500 이고 로그 문구는 `the refresh token could not be rotated`** 이며 끝난 세션·재사용은 여전히 400 이라는 문장을 더한다.
- 테스트 설계(선례 그대로):
  - 이름 예: `TestIntegrationRefreshSurvivesASessionLookupItCannotRun`. 실제 PostgreSQL + `New(data, logger, nil, metrics)` + `httptest`, 로그는 `lockedBuffer`, Refresh Token 은 실제 `/auth` 302 `Location` 의 코드를 PKCE 로 교환해 받는다(그래야 `refresh_tokens.session_id` 가 non-nil 이고 `store/oidc.go:323` 분기에 들어간다 — 이 분기는 `old.SessionID != nil` 일 때만 돈다).
  - 장애 주입: `ALTER TABLE sso_sessions RENAME TO sso_sessions_hidden` → 단언 **전에** 즉시 복구 + `restored` 플래그 + `t.Cleanup`. **확인한 것**: `InspectRefreshToken`(`store/oidc.go:365-376`)은 `refresh_tokens` 만, `UserByID` 는 `users` 만 읽으므로 이 rename 은 648·711 의 앞선 500 경로에 걸리지 않고 325-327 질의에서 처음 실패한다. **미확인**: PostgreSQL 이 FK 의존이 있는 `sso_sessions` 를 rename 해도 즉시 성공하는지는 이번 회차에 실행하지 않았다(선례는 `users`·`clients` rename 성공). 만약 막히면 대안은 컬럼 rename(`ALTER TABLE sso_sessions RENAME COLUMN last_access TO last_access_moved` — `sessionIsLive` 가 그 컬럼을 쓰는지 먼저 확인) 이다.
  - 단언: ① 은닉 중 500 + `error=server_error` + description ② ERROR 그 문구 정확히 1줄, raw refresh_token·verifier 미노출 ③ `resso_token_errors_total{grant_type="refresh_token"} 1` (부재 단언은 계열 이름이 아니라 **샘플 줄** `resso_token_errors_total{grant_type=` 로 — 등록기가 표본 없는 계열에도 HELP/TYPE 을 출력한다) ④ 복구 후 **같은 Refresh Token** 재시도가 200 + 새 `refresh_token` 발급 ⑤ 그 뒤 로그아웃/세션 종료로 `!sessionActive` 를 만들어 400 `invalid_grant` 가 유지되는지(기준 4) — 세션 종료 수단이 마땅치 않으면 `UPDATE sso_sessions SET ended_at=now()` 같은 상태 조작 대신 기존 로그아웃 Endpoint 를 쓸 것.
- 검증 명령 (저장소 루트, 같은 Bash 셸):
  - `test_env="$(scripts/test-services.sh)" && eval "$test_env" && go test -race ./internal/httpserver -run '^TestIntegrationRefresh' -count=1 -v`
  - `go test -race ./internal/store -count=1` (RotateRefreshToken 기존 테스트 회귀 확인 — 먼저 `grep -rn RotateRefreshToken internal/` 로 다른 호출부가 없는지 볼 것)
  - `go test -race ./internal/httpserver -count=1`
  - `make lint` / `make test` (수분. `make test` 는 `.SHELLFLAGS := -o pipefail -c` 로 첫 go test 상태를 제대로 잡는다 — Makefile:1-11·70-79 확인함)
  - 프로덕션 변경 **전에** 새 테스트를 먼저 돌려 red(400 을 받는 것)를 기록할 것.
- 위험과 피할 것:
  - **반려 선례와 선 긋기**: 사람이 반려한 `5bed9dc`(코드 교환 실패를 500 `server_error` 로) 와 혼동하지 말 것. 이번 것은 **refresh 그랜트**이고, 같은 Endpoint 의 이웃 두 경로(648·711)가 이미 500 으로 구현돼 있어 일관성 쪽이다. `handleAuthorizationCodeGrant`·`RedeemAuthorizationCode` 는 **건드리지 말 것**.
  - `metrics.go` 에 새 계열을 만들지 말 것(기존 `metricTokenErrors` + 기존 라벨값 재사용).
  - 감사 항목을 새로 만들지 말 것 — 이 실패는 아무것도 쓰지 않았으므로 기록할 사건이 없다. 로그+지표로 끝낸다.
  - detail·지표 라벨·로그에 raw `refresh_token`, `code_verifier`, Secret 을 넣지 말 것(운영자 규칙).
  - `migrations/`, `.github/workflows/`, `middleware.go`, `auth.go` 는 건드리지 말 것.
  - 빌드가 `webui/dist/index.html` 을 바꾸면 복원하고 `git status` 에 산출물이 남지 않게 할 것.
  - `scripts/test-services.sh --stop` 은 모든 테스트 컨테이너·인증서를 지운다. 검증 끝에 자동 실행하지 말 것.
  - `make lint` 의 govulncheck 가 구형 Go 로 빌드된 실행 파일이면 Go 1.26 분석에 실패한 선례가 있다(도구를 Go 1.26.7 로 재빌드). npm ci 가 보고하는 devDependencies 취약점 5건은 이번 과제 범위 밖이며 해결했다고 주장하지 말 것.
- 차선 후보: README.md 지표 표의 `resso_client_auth_errors_total` 행 정정 — 아직 "401로 나가는 것은 Secret이 틀린 경우와 같고" 라고 적지만(README:191, 실제로 읽어 확인) 세 Endpoint 는 token 500 / revoke 503 / introspection 200 `active=false` 로 답한다. `docs/operations.md:90` 는 이미 맞게 적혀 있어 README 한 행만 그 사실로 고치면 된다. 문서 1개·코드 0개(가치 2 / 위험 1 / 작업량 S).
