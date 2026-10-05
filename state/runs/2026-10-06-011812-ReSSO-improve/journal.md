# 회차 노트 2026-10-06-011812-ReSSO-improve — ReSSO
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 01:18] base pinned — main@da27380
- [러너 01:18] autonomy release — 

## 정찰 노트
- 골랐다: Introspection 의 미판정 Client 인증 → 200 `active=false` + `introspection_errors{stage="client_auth"}`. v0.9.97 이 Limiter·로그를 가르고 token(v0.9.98)·revoke(v0.9.99)가 응답을 가른 뒤 **남은 마지막 Endpoint**이고, oidc.go:937-959 주석이 이 하나만 지목한다. 호출부 앞 분기 하나·프로덕션 파일 1개여서 한 세션에 들어간다.
- 제친 것: `token_errors` 배선 테스트(일곱 회차 연속 보류, 값 2 — 차선으로만 남겼다), `RotateRefreshToken`/`ErrFamilyNotRevoked`(둘 다 store 변경 또는 장애 주입 수단 설계가 먼저여서 45분에 안 들어간다), npm audit(lockfile 회차). CSP 와 `engines` 는 이번에 기각했다.
- 확신 없는 곳: **RFC 7662 §2.2/§2.3 원문을 읽지 않았다.** 200 이 옳다는 근거로 쓴 것은 oidc.go:941 이 introspection 의 fault 계약을 스스로 "200 active=false" 로 적어 둔 것과, 본문이 모르는 Token 에 주는 답과 한 글자도 같다는 점뿐이다. 원문이 "properly authorized 아닌 호출자는 401" 로 읽히면 과제서대로 중단하고 차선으로 갈 것.
- 확인한 것(추측 아님): 꺼진 Client·public Client 는 `authenticateOIDCClient` 가 `err == nil` 로 돌려주므로(oidc.go:843-845) `integration_test.go:3298` 의 401 루프는 그대로 통과한다 — 분기는 센티널만 보면 된다.
- 조심할 것: `recordUnjudgedIntrospection` 을 쓰면 ERROR 가 두 줄이 된다(지표만 직접 더할 것). 끊긴 호출자 단계는 틀린 Secret 소진 **앞**에 둘 것(Limiter 검사가 조회 앞, oidc.go:812-816). `docs/operations.md:90` 의 "introspection 만 401" 은 **고쳐야** 하는 문장이다.
- [러너 01:23] scout done — Introspection Endpoint 가 판정하지 못한 Client 인증에 401 대신 200 `{"active":false}` + `resso_introspection_errors_total{stage=

## 구현 노트
- 무엇을 왜: `introspect`(oidc.go:1178) 호출부 **앞**에 `errors.Is(err, errClientAuthUndecided)` 분기 하나 — `s.metrics.Add(metricIntrospectionErrors, 1, "client_auth")` + 200 `{"active":false}`. 판정하지 못한 인증에 401 을 주면 Resource Server 가 맞았던 자격증명을 폐기한다. 프로덕션 파일 1개(oidc.go), 테스트 1개, 문서 1개. 커밋 bf6d1bd.
- **정찰이 미확인으로 남긴 RFC 7662 는 원문으로 확인했다**(§2.3 전문을 rfc-editor·datatracker 두 경로로 받아 대조). §2.3 의 401 은 "its credentials are **invalid**" 조건에 걸린 것이고, 인증이 확립되지 않은 호출자에게 401 을 요구하는 MUST 는 §2.3 에 없다. §2.1 의 MUST 는 token scanning 방지가 목적이고 본문이 상수라 scanning 할 것이 없다. 중단 조건 불성립 → 과제서대로 구현. 논증을 §번호와 함께 분기 주석에 남겼다.
- 확신 없는 곳: 위 규범 판단은 원문 인용에 근거하지만 **해석**이다 — 비평가가 다시 볼 자리는 거기 하나다. 그 외에 검증 못 한 것은 없다.
- 일부러 안 한 것: `recordUnjudgedIntrospection` 사용(ERROR 가 두 줄이 된다 — 테스트가 그 줄 0 을 단언한다), `metrics.go`·`registerMetrics` 수정(기존 계열에 라벨값 하나), `README.md:193`(계열 불변), ADMIN_GUIDE·PDF·캡처, `uuid.Parse` 두 분기(도달 가능성 미확인 — ideas.json 에 남김).
- 과제서가 고치라 한 주석 셋 외에 **넷째**가 있었다: `errClientAuthUndecided` 의 doc(oidc.go:782)이 "only the token endpoint's is implemented so far" 라고 적고 있어 함께 고쳤다.
- 다음 역할이 조심할 것: 새 테스트는 **실제 PostgreSQL 이 있어야 돈다**(`eval "$(scripts/test-services.sh)"` 뒤 같은 셸). `clients` 를 RENAME 하고 `LOCK TABLE clients IN ACCESS EXCLUSIVE MODE` 를 잡으므로 복구·해제 둘 다 `t.Cleanup` 에 걸려 있다 — 중간에 죽이면 컨테이너를 공유하는 다른 테스트가 함께 넘어진다. `make test` 는 `webui/dist/index.html` 을 바꾸므로 커밋 전 복원(이번에도 했다).
- [러너 01:37] brief accepted — 채택 — 근거(1178-1182 가 모든 실패를 `writeClientAuthError` 로 합치는 것, 937-959 주석이 이 Endpoint 하나만 남았다고 스스로 지�
- [러너 01:37] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 직접 돌렸다: 새 테스트 PASS(0.99s), 패키지 전체 ok 117.580s, `go vet`·`gofmt -l` 무출력. 원장의 실패 재현 단언(5066행 401 vs 200)이 실제 테스트와 일치하고 main 의 1178-1182 가 모든 실패를 `writeClientAuthError` 로 합치므로 수정 전 통과가 불가능 — 테스트가 바뀐 경로를 실제로 지난다.
- 구현자가 유일한 미확인으로 남긴 RFC 7662 해석을 대조했다. §2.3 의 401 은 "credentials are invalid" 조건부이고 미확립 인증에 대한 MUST 는 없다. 본문이 1172·1231·1291 과 동일한 상수인 것도 확인 — 규범 위반 없음. 센티널은 `ours==true` 에서만 감싸져 끊긴 호출자(8f2c73c)가 지표를 올리지 못하는 것도 코드로 확인.
- 못 본 것: `stage="secret"`(digest 깨짐) 분기는 읽기로만 봤다 — 장애 주입이 clients RENAME 뿐이라 테스트가 짚지 않는다. `make lint`·`make test` 전체는 재실행하지 않고 원장 기록 + 내 패키지 실행으로 갈음했다.
- 승인이어도 남는 우려(릴리즈 노트): 이제 clients 장애 중 Resource Server 는 401 신호를 완전히 잃고 "모든 Token 이 죽었다"만 본다 — 세 Endpoint 중 응답만으로 장애를 알 수 없는 유일한 곳이 되었다. 간헐 실패가 나면 먼저 볼 자리는 테스트 (5)의 `pg_locks` 폴링(server 와 테스트가 Pool 을 공유한다).
- 작업 트리에 커밋되지 않은 `webui/dist/index.html`(verify 의 빌드 산출물) 이 남아 있다 — PR 내용과 무관하지만 릴리즈 단계가 되돌려야 한다.
- [러너 01:43] review approved — 리뷰 승인 (risk=low)
- [러너 01:43] pr created — https://github.com/hkjang/ReSSO/pull/38
- [러너 01:52] ci passed — 검사 2개 모두 success
- [러너 01:52] merge done — bf6d1bd
