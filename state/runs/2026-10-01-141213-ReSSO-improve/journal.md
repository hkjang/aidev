# 회차 노트 2026-10-01-141213-ReSSO-improve — ReSSO
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 14:12] base pinned — main@8604422
- [러너 14:12] autonomy release — 

## 정찰 노트
- 코드가 스스로 남긴 지목을 골랐다: oidc.go:578-585 주석이 "아직 안 한 변경" 둘을 명시하는데, RotateRefreshToken 쪽은 store 변경(306-312의 ErrNotFound 합침)이 필요하고 RENAME 으로는 v0.9.95 분기가 먼저 잡혀 새 분기에 도달조차 못 한다 — 그래서 장애 주입이 깨끗한 RedeemAuthorizationCode 쪽을 택했다. 차선(지표 배선 고정)은 프로덕션 가치가 낮아 밀었다.
- 확인한 것: 없는·만료된·불일치 코드는 모두 store.ErrNotFound 로 모인다(store/oidc.go:118-127·180-182, oidc.go:509-511). ValidatePKCE 는 센티널 없는 errors.New 넷(pkce.go:10-25) — 래핑 없이 가르면 PKCE 불일치가 500 이 된다.
- 추측으로 적은 것: 센티널 이름·본문 문안은 제안이다. store/oidc.go:169-175(재사용 처리 중 Exec/Commit 실패)가 새 500 통에 들어가는 것은 코드로 확인했으나 테스트 재현은 미확인 — 주석으로만 밝히고 범위를 넓히지 말 것.
- 구현자가 조심할 것: 코드는 한 번 쓰면 소비되므로 장애용·복구용·PKCE용 코드를 각각 따로 채굴할 것. authorization_codes RENAME 은 단언 전 즉시 복구(90초 수명 + 공유 컨테이너).
- 기각: 비화면 경로 CSP 좁히기 — 10회 연속 보류였고 직전 정찰의 권고대로 rejected 로 내렸다.
- [러너 14:17] scout done — 코드 교환이 `RedeemAuthorizationCode` 의 저장소 장애를 `invalid_grant` 로 답하는 것을 500 `server_error` 로 (가치 3 / �

## 구현 노트
- 바꾼 것: `oidc.go` 하나. 코드 교환의 400 `invalid_grant` 앞에 `!ErrNotFound && !ErrCodeReuse && !errCodeRefused` 분기(500 `server_error` + `resso_token_errors_total{grant_type="authorization_code"}` + ERROR 한 줄)를 넣고, validate 콜백이 `ValidatePKCE` 오류를 새 센티널 `errCodeRefused` 로 감싸게 했다. `oidc.go:578-585` 의 "아직 안 한 변경" 주석도 이쪽이 끝났으므로 함께 고쳤다(안 고치면 코드가 거짓말을 한다).
- 확신 없는 곳: ① `store/oidc.go:169-175`(재사용 감지 뒤 family 취소 Exec/Commit 실패)가 새 500 통에 들어가는 것은 **코드만 읽어 확인**했고 테스트로 재현하지 않았다 — 주석 한 줄로만 밝혔다. ② `make test` 의 프런트엔드 단계가 실패한다. **내 변경과 무관하다**: 프런트엔드 파일을 하나도 건드리지 않았고(`git status` 가 Go 2개 + docs 1개만 보인다), 깨끗한 `web/` 에서 `npx vitest run` 만 돌려도 29개 파일 중 26개가 `TypeError: webidl.util.markAsUncloneable is not a function`(jsdom → undici 8.10.0)으로 워커 시작부터 깨진다. `--pool=threads` 도 같고, `/usr/bin/node` 25.9.0 으로 돌리면 26개 파일/142개 테스트가 통과하고 3개 파일(CommandPalette·realms)의 19개가 실패한다 — 어느 Node 로도 깨끗하지 않다. 조사 항목으로 `ideas.json` 에 넣었고 lockfile 은 손대지 않았다.
- 일부러 안 한 것: `store/oidc.go` 는 읽기만 했다(`ErrFamilyNotRevoked` 신설은 별도 회차). `RotateRefreshToken` 쪽은 과제서 지시대로 건드리지 않았다. 재사용 분기의 감사·Warn·400 은 한 글자도 안 바꿨다(새 테스트 (d)가 그것을 단언한다 — 이 경로는 지금까지 어떤 테스트도 지나지 않았다).
- 다음 역할이 조심할 것: 새 테스트는 PostgreSQL 이 있어야 돈다 — `eval "$(scripts/test-services.sh)"` 를 **같은 셸에서**. `authorization_codes` 를 RENAME 하므로 `t.Cleanup` 과 단언 전 즉시 복구가 필수이고(코드 수명 90초), 실패한 채 내버려두면 이 스키마의 `/auth` 가 전멸한다. 지표는 `New(data, logger, nil, nil)` + `srv.Metrics()` 로 읽는다. Go 쪽 검증은 끝났다: `go test -race ./...` 13개 패키지 ok(httpserver 137.0s), SKIP 경고 0, `make lint` 0 issues, `npm run build`·`go build` 성공, `webui/dist/index.html` 복원.
- [러너 14:31] brief accepted — 채택 — 근거(517행이 `ErrNotFound` 구분 없이 400, 578-585 주석이 스스로 지목, `ValidatePKCE` 가 센티널 없는 `errors.New` 넷, 없는·�
- [러너 14:31] verify failed — 실패한 검증: cd web && npm test --silent (exit 1)
