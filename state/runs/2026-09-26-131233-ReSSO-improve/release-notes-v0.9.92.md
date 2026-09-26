## v0.9.92

**CORS 검사가 조회 실패 때문에 Header를 빼고 통과시킨 것을, 이제 서버 로그에 남깁니다.** 프로토콜 Endpoint 앞단의 `oidcCORS`는 Realm 조회와 Origin 등록 여부 조회 두 가지를 하는데, 어느 쪽이 오류를 내든 그 오류를 그대로 버렸습니다. 그래서 **저장소가 답하지 못한 경우와 아무도 등록하지 않은 Origin이 바이트 단위로 같은 결과**였습니다 — CORS Header 없이 다음 Handler로 넘어가고, 그 Handler는 200을 답합니다. RP 쪽에는 원인이 드러나지 않는 CORS 실패만 보이므로 운영자는 멀쩡한 자기 설정을 다시 읽고, 이쪽에는 읽을 것이 없으며 접근 로그마저 200으로 건강해 보였습니다. `realmFromPath`를 부르는 다른 자리들은 실패를 저마다 기록하거나(Discovery · JWKS · 인가 · Revocation · 로그아웃의 `realmLookupFailed`, UserInfo와 Introspection의 전용 경로) 적어도 호출자에게 다른 답을 돌려주는데, 이 미들웨어만 **기록도 하지 않고 응답도 구별되지 않았습니다.**

### 수정

- **Realm 조회가 `ErrNotFound`가 아닌 오류를 내면** Discovery · JWKS · 인가 · Revocation · 로그아웃이 이미 쓰는 `realmLookupFailed`를 이 미들웨어의 이름(`endpoint=cors`)으로 남깁니다. 실제로 없거나 꺼진 Realm은 남기지 않습니다 — 뒤의 Handler가 404나 401로 답하며 그쪽에서 말하므로, 남기면 한 사실에 두 줄이 됩니다.
- **Origin 등록 여부 조회(`WebOriginAllowed`)가 오류를 내면** 전용 한 줄 `a CORS origin could not be checked against the Realm's registered ones`(level `ERROR`)을 남깁니다. 담기는 값은 `trace_id` · `realm` · `error` 셋이며, **Origin 원문은 넣지 않습니다** — 검증되지 않은 입력이고, 이 줄의 목적은 어느 Realm의 조회가 멈췄는지를 말하는 것입니다. `realm`은 경로에서 오므로 `realmLookupFailed`가 이미 쓰는 것과 같은 출처입니다.
- **단순히 등록되지 않은 Origin에는 여전히 아무것도 남기지 않습니다. 이 침묵은 빠뜨린 것이 아니라 계약이고, 테스트가 단언합니다.** `Origin`은 호출자가 정하는 Header이고 인증 없이 이 미들웨어에 닿으며, 이 미들웨어는 프로토콜이 제공하는 모든 경로에 걸려 있습니다 — Origin마다 한 줄을 남기면 누구나 Header 하나를 바꿔 바깥에서 채울 수 있는 로그가 됩니다. 저장소가 답하지 못하는 것은 그런 식으로 닿을 수 없습니다: 이 서비스 쪽 장애이고, 모든 호출자에게 같은 오류이며, 저장소가 회복되면 멎습니다.
- **응답은 한 글자도 바뀌지 않습니다.** 두 실패 모두 이전처럼 Header를 붙이지 않고 그대로 통과합니다(fail closed) — 답하지 못한 저장소가 Token 응답을 읽을 수 있는 대상을 넓히는 일은 없어야 하기 때문입니다. 상태 코드, 본문, `Vary: Origin`, 허용된 경로에서 붙는 Header 네 개와 그 값이 모두 그대로입니다.
- `docs/operations.md`의 «`clients` 조회가 멈추면 찾을 로그» 표를 두 줄에서 세 줄로 늘려 이 줄을 더했습니다. 증상이 «등록되지 않은 Origin»과 구별되지 않으므로, RP의 Origin 설정을 의심하기 전에 이 줄부터 확인하라고 적었습니다.

### 확인

- 새 연동 테스트 `TestIntegrationCORSSaysWhenItCouldNotCheckAnOrigin` — 실제 PostgreSQL과 프로덕션 Handler로 다섯 경우를 봅니다. (a) 등록된 Origin은 200과 Header 넷을 받고 조용합니다 (b) 등록되지 않은 Origin은 침묵하며 상태가 (a)와 같습니다 (c) `clients`를 RENAME해 조회를 세운 뒤 등록된 Origin으로 부르면, Header는 빠지되 상태는 (a)와 같고(fail closed) 전용 한 줄이 `realm=master`로 남으며 Origin 원문은 없고 `Vary`는 유지됩니다 (d) 없는 Realm에는 `endpoint=cors`가 없고 (e) `realms`를 RENAME하면 `endpoint=cors`가 있습니다. 키 세트를 고른 것은 이 미들웨어 아래에서 `clients`도 세션도 읽지 않는 가장 싼 경로라 (c)의 RENAME이 Origin 검사 외에는 아무것도 깨뜨리지 않기 때문입니다.
- **수정 전 `middleware.go`에서 (c)와 (e)의 단언 3개가 실제로 실패**함을 먼저 확인했습니다. 그때에도 (a) · (b) · (d)와 (c)의 fail-closed 단언은 통과했는데, 이것이 응답이 바뀌지 않았다는 증거입니다. 그 실패 로그에 Handler의 `endpoint=jwks` 줄만 있고 미들웨어의 줄이 없는 것이 «침묵한 것은 미들웨어였다»는 증거입니다.
- 릴리즈 준비에서 `make lint`, `make test`(Go `-race` 전 패키지 · 연동 SKIP 0 · `go vet` · 콘솔 테스트 · 빌드), `make build VERSION=v0.9.92`, `git diff --check`가 통과했습니다.

### Upgrade notes

관측 정보만 늘어나며 **CORS의 동작은 달라지지 않습니다** — 허용 판정 규칙, 붙는 Header 넷과 그 값, `Vary: Origin`, 조회가 실패했을 때 허용으로 기울지 않는 fail-closed 규칙이 모두 이전과 같습니다. 마이그레이션도 설정 변경도 없고 이전 `v0.9.91` 이미지로 롤백할 수 있습니다(되돌리면 조회 실패가 다시 무음이 될 뿐입니다).

로그를 기계로 읽는다면 **`ERROR` 두 종류가 새로 나타날 수 있습니다**: `endpoint=cors`가 붙은 기존 Realm 조회 실패 줄과, 새 메시지 `a CORS origin could not be checked against the Realm's registered ones`입니다. 둘 다 감사 이벤트가 아니라 서버 로그에만 남고, 요청 하나에 최대 한 줄입니다. 정상 운영에서는 나오지 않는 줄이므로 **0이 아니면 알림을 걸 만합니다** — 이 줄이 보이는 동안 그 Realm의 브라우저 RP는 전부 CORS가 막힌 상태입니다.

등록되지 않은 Origin에 대해서는 **의도적으로 아무 줄도 남지 않습니다.** "CORS가 막힌다"는 문의를 받았는데 이 줄이 하나도 없다면 저장소는 정상이라는 뜻이니, 그때는 해당 Client의 `web_origins` 등록을 보세요.

