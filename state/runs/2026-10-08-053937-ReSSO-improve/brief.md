- 과제: 코드 재사용 감사의 actor 가 `users` 테이블 장애로 조용히 비는 것을 막기 — `handleAuthorizationCodeGrant` 의 `UserByID` 오류 버림(`oidc.go:560`)을 `userLookupFailed` 로 가르기 (가치 3 / 위험 1 / 작업량 S)
- 왜: `internal/httpserver/oidc.go:560` 이 `affected, _ := s.store.UserByID(...)` 로 오류를 버리므로, `users` 를 읽지 못하는 동안 재사용이 감지되면 `AUTHORIZATION_CODE_REUSED` 항목의 `actor_name` 이 빈 칸으로 기록되고 그 사실이 로그·지표 어디에도 남지 않는다. 형제 경로가 **바로 이 결함을 스스로 적어 고쳤다** — `oidc.go:706-711` 의 주석("the actor column was blank and a search by account could never return it")이 `REFERESH_TOKEN_REUSE` 쪽에서 같은 문제를 지목하고, 거기서는 `userLookupFailed`(690-698)를 **회전 전에** 지나므로 actor 가 보장된다. 코드 재사용 쪽만 그 보장이 없다: 토큰이 유출된 그 순간의 유일한 기록이 "누구의 것인지" 를 잃고, 운영 가이드가 안내하는 "계정으로 감사 검색" 이 그 항목을 영원히 못 찾는다.
- 수용 기준:
  1) `users` 를 읽을 수 없는 동안 코드가 재사용되면 (a) `AUTHORIZATION_CODE_REUSED` 감사 항목은 **여전히 기록되고**(잃지 않는다) (b) 그 detail 에 actor 를 해결하지 못했다는 표시가 들어가고 (c) ERROR 로그와 `resso_token_errors_total{grant_type="authorization_code"}` 가 그 조회 실패를 말한다.
  2) **HTTP 응답은 한 글자도 바뀌지 않는다**: 400 `invalid_grant` + `authorization code is invalid or expired`. (기각된 `5bed9dc` 와 겹치지 않도록 테스트가 본문 문안까지 단언할 것.)
  3) `users` 가 정상일 때의 기존 동작은 그대로다: 평범한 재사용은 `actor_name=<username>`, detail 에 새 키 **부재**, WARN `authorization code replayed` 1줄, ERROR 0줄, `token_errors` 무증가. (`TestIntegrationCodeReuseIsRecordedEvenWhenItsRevocationFails` 의 기존 단언이 전부 통과해야 한다.)
  4) 계정이 **정말 삭제된 경우**(`store.ErrNotFound`)는 이쪽 장애가 아니므로 새 신호를 내지 않는다 — `userLookupFailed` 가 `ErrNotFound` 를 false 로 거르는 것을 그대로 쓴다.
  5) 테스트가 증명해야 하는 것: 고치기 전에는 (1b)·(1c) 가 없고 감사 항목의 actor 칸이 비는 것, 고친 뒤에는 둘이 생기면서 (2)·(3) 이 그대로인 것.
- 건드릴 파일 (프로덕션 **1개**):
  - `internal/httpserver/oidc.go:560` — `handleAuthorizationCodeGrant` 의 재사용 분기. `affected, _ := s.store.UserByID(...)` 를 `affected, actorErr := s.store.UserByID(...)` 로 바꾸고, 바로 아래에서
    `if s.userLookupFailed(r, realm.Name, "authorization_code", actorErr) { detail["actor_resolved"] = false }` 를 `detail` 선언(561) 뒤에 둔다. `detail` 은 이미 `map[string]any{"session_id": …}` 로 만들어져 있으므로 키 하나를 더하는 것이 전부다. `affected.Username` 은 그대로 쓴다(장애 시 빈 문자열이고, 그것이 비었다는 사실을 detail 이 말한다).
  - 주석: 558-559 의 "the name is resolved here even though the grant is already lost" 가 **오류를 버리는 것을 정당화하지 않는다**는 점을 분명히 하도록 한 문단 — 왜 응답은 바꾸지 않는지(코드는 이미 소진돼 재시도할 것이 없다), 왜 `ErrNotFound` 는 신호를 내지 않는지(삭제된 계정은 진짜 답이다), 왜 `userLookupFailed` 의 ERROR 줄이 재사용 WARN/ERROR 줄과 **다른 사실**을 말하므로 "같은 사건 두 줄" 이 아닌지. 이 저장소의 주석 밀도를 따를 것.
  - `internal/httpserver/integration_test.go` — `TestIntegrationCodeReuseIsRecordedEvenWhenItsRevocationFails`(**2201행**, 확인) 을 **고치지 말고**, 그 옆에 새 테스트 `TestIntegrationCodeReuseNamesWhoseCodeLeakedEvenWhenTheAccountCannotBeRead` 를 추가. 그 테스트의 `/auth` 302 코드 채굴(PKCE 포함)·`lockedBuffer`·`restored` 플래그·`t.Cleanup` 패턴을 그대로 베낄 것. 지표를 읽으려면 `New(data, logger, nil, metrics)` 로 등록기를 넘길 것(2201 테스트는 `nil` 을 넘기므로 복사만으로는 수용 기준 1(c)를 못 읽는다).
  - 문서 `docs/operations.md` — 두 줄 다 **고치는** 쪽이다(열어서 확인했다):
    - `:104` `AUTHORIZATION_CODE_REUSED` 행: 지금 "기록된 계정과 Client를 확인하고" 라고 쓰여 있는데, `users` 장애 중에는 **그 계정 칸이 빈다**. `actor_resolved=false` 가 상세에 있으면 actor 칸이 빈 이유가 계정 조회 실패이고 `session_id` 와 서버 로그 `the account named in the grant could not be looked up` 으로 추적하라는 문장을 더할 것.
    - `:87` `resso_token_errors_total` 불릿: 지금 "**계정 조회 실패는 이 계열에 셉니다** … 이 역시 400 `invalid_grant`가 아니라 500 `server_error`로 답합니다" 로 **단정**한다. 이 변경으로 그 단정이 조건부로 틀려진다 — 재사용 분기의 계정 조회 실패는 이 계열에 세지만 응답은 그대로 400 `invalid_grant` 다(코드가 이미 소진돼 재시도할 것이 없으므로). 그 예외를 한 문장으로 적을 것.
    - README 지표 표는 계열이 늘지 않으므로 손대지 않는다. ADMIN_GUIDE·PDF·캡처 재생성 없음.
- 검증 명령:
  - `eval "$(scripts/test-services.sh)"` 를 먼저, **같은 셸에서**(출력이 비면 준비 실패이므로 eval 하지 말 것). `/tmp/resso-test-certs/ca.crt` 가 없으면 `scripts/test-services.sh --stop` 후 재실행(2026-10-07 회차가 여기서 깨졌다).
  - 실패 재현(프로덕션 변경 **전**): `go test -race ./internal/httpserver -run '^TestIntegrationCodeReuseNamesWhoseCodeLeakedEvenWhenTheAccountCannotBeRead$' -count=1 -v`
  - `go test -race ./internal/httpserver -run '^TestIntegrationCodeReuse' -count=1 -v` (기존 테스트 동반 통과 확인)
  - `go test -race ./internal/httpserver -count=1` (약 120~140s), `go test -race ./internal/store -count=1` (약 80~95s)
  - `make lint` (0 issues 여야 함), `make test` (exit 0; 끝의 `"integration test(s) did not run"` 매치가 Makefile 레시피 에코 1행뿐 = SKIP 0 인지 눈으로 확인)
  - 마무리: `webui/dist/index.html` 복원, `git diff --check`, `git status` 에 빌드 산출물 없음.
- 위험과 피할 것:
  - **응답을 바꾸지 말 것.** 이 분기의 400 `invalid_grant` 를 500 으로 바꾸는 접근(`5bed9dc`)은 사람이 받지 않았고 재제출 금지다. 이 과제는 **감사·로그·지표만** 바꾼다. 테스트가 본문 문안을 단언해 그 선을 지킨다.
  - **장애 주입 순서**: 코드는 `users` 를 치우기 **전에** `/auth` 302 의 `Location` 에서 채굴해야 하고(로그인이 `users` 를 읽는다), 첫 교환도 치우기 전에 끝내야 한다(`oidc.go:584` 가 `users` 를 읽고 성공해야 200 이 난다). 순서: 로그인 → 코드 채굴 → 첫 교환 200 → `ALTER TABLE users RENAME TO users_hidden` → 같은 코드 재사용 → **단언 전 즉시 복구** + `restored` 플래그 + `t.Cleanup`.
  - **감사 insert 가 살아남는지가 이 과제의 첫 확인 사항**: `migrations/001_initial.sql:206` 이 `actor_id uuid REFERENCES users(id) ON DELETE SET NULL` 이다. 테이블 RENAME 은 FK 를 OID 로 따라가므로 행이 `users_hidden` 에 그대로 있어 insert 가 통과할 것으로 **판단했으나 실행으로 확인하지 않았다 — 미확인**. 수용 기준 1(a)가 바로 그 확인이다. 만약 insert 가 깨지면(`s.audit`=`auth.go:313` 이 Warn `write audit event failed` 를 남긴다) 장애를 더 좁게 — `ALTER TABLE users RENAME COLUMN <컬럼> TO <…>_moved` 로 — 바꿀 것. `UserByID`(`internal/store/users.go:37`)는 `"SELECT "+userColumns+" FROM users WHERE id=$1"` 이므로 `userColumns` 목록 중 아무 컬럼 하나를 옮기면 그 SELECT 가 깨진다(같은 상수를 쓰는 다른 조회도 함께 깨지는 점은 주의). 어느 쪽이든 `t.Cleanup` 복구 필수.
  - detail 키 이름은 `actor_resolved:false` 를 권하되(87cb26a 의 `tokens_revoked:false`·`family_revoked:false` 와 같은 꼴), 구현자가 더 나은 이름을 고르면 그래도 된다. **오류 원문을 detail 에 넣지 말 것** — 운영자 규칙이고 87cb26a 가 같은 판단으로 사유를 로그에만 두었다.
  - `userLookupFailed` 를 고치지 말 것. 세 호출부(585·691·새 자리)가 공유하고 앞의 둘은 응답까지 바꾸는 경로다. 여기서는 **반환값을 응답 분기로 쓰지 않고 신호용으로만** 쓴다 — 그것이 이 과제의 전부다.
  - `auth.go`·`middleware.go`·`migrations`·`.github/workflows` 는 건드리지 않는다. `store/oidc.go` 도 이번에는 건드리지 않는다(87cb26a 가 방금 고친 자리다).
  - 로그 줄이 둘(재사용 WARN/ERROR + `userLookupFailed` ERROR)이 되는 것은 **의도**다 — 서로 다른 사실이고, 87cb26a 가 피한 것은 *같은* 사실을 두 severity 로 쓰는 것이었다. 테스트가 각 줄을 정확히 1줄로 세어 중복이 아님을 고정할 것.
- 차선 후보: `UpdateUser` 가 계정을 끄면 세션을 함께 끝낸다는 계약을 연동 테스트로 고정 — `internal/store/users.go:302` 가 `EndSessionsOfDisabledUsers`(`internal/store/sessions.go:314`)를 부른다 (2/1/S, 프로덕션 파일 0개). **이번 정찰에서 확인했다: `EndSessionsOfDisabledUsers` 를 언급하는 테스트 파일은 하나도 없다**(`internal/` 전체에서 참조가 `sessions.go:302,314`·`users.go:302`·`ldap_federation.go:287,364,704` 뿐) — 즉 이 계약은 지금 어떤 테스트도 고정하지 않으므로 그대로 쓸 수 있다. `internal/store` 연동 테스트에서 계정을 끄고 `sso_sessions` 가 끝났는지 단언하는 것이 전부다.
