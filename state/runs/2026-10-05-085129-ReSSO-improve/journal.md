# 회차 노트 2026-10-05-085129-ReSSO-improve — ReSSO
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 08:51] base pinned — main@88995a9
- [러너 08:51] autonomy release — 

## 정찰 노트
- 고른 이유: oidc.go:937-952 의 주석이 남은 두 조각(revoke·introspection)을 스스로 지목하고, revoke 쪽은 답이 이미 정해져 있다 — 같은 핸들러가 네 자리에서 쓰는 503 `temporarily_unavailable` 과 그 본문 문안, 그리고 RFC 7009 §2.2.1 의 "토큰은 아직 존재한다고 가정하고 재시도" 가 정확히 이 상태의 뜻이다. introspection 을 제친 것은 RFC 7662 가 "properly authorized" 가 아닌 호출자에게 200 을 주라고 하지 않아 규범 해석이 미확인이고(새 stage 라벨·문서 둘도 따라온다), 차선 후보(token_errors 배선 고정, 프로덕션 0개)를 제친 것은 가치 2 로 다섯 회차 연속 밀려 온 항목이기 때문이다.
- 확인한 것: `s.audit` 시그니처(auth.go:313), revoke 의 503 선례 네 자리와 Realm 분기의 감사 모양(oidc.go:1286-1299), 기존 revoke 테스트 중 **Client 인증에 401 을 요구하는 단언이 없다**는 것, Endpoint 표(11605행)가 Realm 장애를 다루며 revocation 에 이미 503 을 기대하는 것, docs/operations.md:90 이 "revocation 은 아직 401" 이라고 적어 둔 것(이 변경으로 틀린 말이 되므로 고쳐야 한다).
- 추측으로 적은 것(미확인): 감사 detail 의 키 이름(`"error": "authenticate the client: …"`)은 Realm 분기의 형태를 흉내 낸 제안일 뿐 정본이 아니다 — 구현자가 1292행과 `unavailable` 클로저의 실제 문구에 맞출 것. 새 테스트 이름도 제안이다. `clients` RENAME 중에 revoke 의 Realm 조회가 먼저 걸리지 않는지는 실행으로 확인하지 않았다(Realm 은 `realms` 테이블이라 걸릴 이유가 없다고 봤다).
- 조심할 것: 분기는 **`writeClientAuthError` 호출부 앞**에만 두고 헬퍼 자체를 고치지 말 것(세 Endpoint 가 함께 움직인다). introspection 은 손대지 말 것. 끊긴 호출자가 503 을 올리지 못하는지(`ours == false`)를 반드시 단언할 것 — 8f2c73c 가 닫은 구멍이 상태 코드만 바뀐 모양으로 다시 열릴 수 있는 자리다.
- 테스트 실행은 하지 않았다(읽기 예산·비용 한도). 프로필은 HEAD 가 v0.9.96 → v0.9.98 로 바뀌어 새로 썼다.
- [러너 08:56] scout done — Revocation Endpoint 가 판정하지 못한 Client 인증에 401 대신 503 `temporarily_unavailable` 로 답하게 하기 (가치 3 / 위�

## 구현 노트
- `revoke`(oidc.go:1302) 의 `writeClientAuthError` 호출 **앞**에 `errors.Is(err, errClientAuthUndecided)` 분기를 두어 503 `temporarily_unavailable` + `TOKEN_REVOKED`/`FAILURE` 감사(주체 Realm, `revoked=none`, `client_id` 미포함)로 갈랐다. 판정하지 못한 인증은 토큰을 찾아보지도 않았으므로 유출된 토큰이 살아 있고, RFC 7009 §2.2.1 이 바로 그 상태를 가리킨다. 프로덕션 1개(oidc.go) + 문서 1개(docs/operations.md:90) + 테스트 1개. 커밋 231f3f7.
- **확신 없는 곳**: 감사 detail 의 `err.Error()` 는 센티널까지 포함해 `the client authentication could not be decided: <store 오류>` 형태로 들어간다 — 1292행 Realm 분기가 같은 형태를 쓰므로 허용으로 봤으나, pgx 오류 문구에 테이블명이 들어가는 것을 민감하다고 보면 비평가가 판단할 자리다. `stage="secret"`(깨진 digest)로 들어오는 경우는 revoke 경로에서 **실행으로 확인하지 않았다** — 코드상 같은 센티널을 타므로 같은 503 이 나야 하지만 테스트는 `stage="client"` 만 재현한다.
- **일부러 하지 않은 것**: introspection(oidc.go:1171)은 손대지 않았다 — 계약이 200 `active=false` 이고 RFC 7662 가 미인증 호출자에게 200 을 주라고 하는지가 미확인이다. `writeClientAuthError` 자체도 고치지 않았다(세 Endpoint 가 함께 움직인다). `resso_revocation_errors_total` 같은 새 계열도 더하지 않았다(이미 `client_auth_errors{stage}` 가 센다).
- **다음 역할이 조심할 것**: 새 테스트는 실제 PostgreSQL 이 있어야 돈다(`eval "$(scripts/test-services.sh)"` 를 먼저, **같은 셸에서**). 테스트가 `clients` 를 RENAME 하고 또 `ACCESS EXCLUSIVE` 로 잠그므로 `t.Cleanup` 둘을 지우면 뒤따르는 테스트가 전멸한다. 단계 순서도 의미가 있다 — Limiter 검사가 조회 **앞**(oidc.go:812-816)이라 끊긴 호출자 단계는 틀린 Secret 소진 **앞**에 있어야 한다. 빌드가 `webui/dist/index.html` 을 바꾸므로 복원했다(커밋에 없음).
- [러너 09:10] brief accepted — 채택 — 근거(1302-1305 가 모든 실패를 `writeClientAuthError` 로 합치는 것, 1286-1299 Realm 분기가 "감사 항목이 아예 없는 것" 을 �
- [러너 09:10] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 판정 approve(위험 low, 차단 없음). 재현했다: 새 테스트는 HEAD 에서 통과(1.04s), `oidc.go` 만 main 으로 되돌리면 `integration_test.go:4659 … answered 401 … want 503` 로 원장과 같은 증상으로 실패. `go test -race ./internal/httpserver -count=1` ok 124.4s, gofmt·vet 깨끗. 분기 위치(`writeClientAuthError` 앞)·`ours==true` 센티널 게이트·끊긴 호출자 단언·Limiter 무소모·미검증 입력 트레일 배제까지 코드로 확인했다.
- 못 본 것: `make lint`(govulncheck·eslint)와 store·web 테스트는 돌리지 않고 원장 주장을 받았다. `stage="secret"`(깨진 digest)의 503 은 코드상 같은 센티널이지만 테스트가 재현하지 않는다.
- 승인이어도 남는 우려 ①(릴리즈 노트·다음 회차): **docs/operations.md:109 `TOKEN_REVOKED` 표가 따라오지 않았다** — `target_type=realm`·행위자 빈 값 FAILURE 를 `resolve the realm:` 하나로만 설명하는데 이제 `authenticate the client:` 가 같은 모양으로 들어온다. 틀린 말은 아니라 거절하지 않았으나 감사 항목을 읽는 자리는 그 표다.
- 우려 ②: `clients` 장애 중 인증 없는 호출자의 revoke 가 요청마다 감사 행을 쓰고 Limiter 는 깎지 않는다(main 의 Realm 분기와 같은 성질). 우려 ③: 작업 트리의 `webui/dist/index.html` 이 커밋 밖에서 수정돼 있으니 릴리즈가 쓸어 담지 않게 할 것.
- 다음 회차: introspection(oidc.go:1171)만 남았고 RFC 7662 해석 미확인으로 미룬 근거가 주석·원장에 모두 적혀 있다.
- [러너 09:17] review approved — 리뷰 승인 (risk=low)
- [러너 09:17] pr created — https://github.com/hkjang/ReSSO/pull/37
- [러너 09:24] ci passed — 검사 2개 모두 success
- [러너 09:25] merge done — 231f3f7
