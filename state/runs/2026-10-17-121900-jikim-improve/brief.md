# 정찰 과제서 (2026-10-17, base main@69b62df / v0.2.39)

- 과제: OpenBao `auth/token/create` 가 길이를 알 수 없는(chunked) 요청 본문을 조용히 버려 `ttl`·`display_name` 을 무시하고 기본 12시간 토큰을 발급하는 문제 수정 (가치 3 / 위험 2 / 작업량 S)

- 왜: `baoTokenCreate`(internal/httpapi/openbao.go:231-262)의 본문 게이트가 `if r.ContentLength > 0 && !decodeBaoJSON(w, r, &input)`(openbao.go:241)이다. Go 의 net/http 는 chunked transfer-encoding 과 길이를 알 수 없는 요청에 `r.ContentLength = -1` 을 넣으므로 이 조건이 거짓이 되어 **본문 전체가 파싱되지 않는다** — 호출자가 `{"ttl":"1h"}` 를 보내도 `input.TTL == ""` 가 되어 기본값 12시간 토큰이 발급되고(openbao.go:243-252), `{"ttl":"nonsense"}`·`{"ttl":"-5h"}`·`{"ttl":"40d"}` 같은 잘못된 값은 `400 invalid ttl` 검증(openbao.go:246-250)과 30일 상한을 **건너뛴 채** 12시간 토큰이 된다. 요청보다 수명이 긴 자격증명이 조용히 발급되는 것이므로 "조용한 무시" 중 실제 피해가 있는 쪽이다. 같은 저장소의 다른 선택적 본문 게이트는 모두 `!= 0` 을 쓴다(resource_handlers.go:521, ai.go:228, oidc.go:68) — 이 한 곳만 `> 0` 이다.

- 수용 기준:
  1) `POST /v1/auth/token/create` 에 `ContentLength == -1`(chunked) 로 `{"ttl":"1h","display_name":"ci"}` 를 보내면 세션 생성에 **1시간**과 `"ci"` 가 전달된다(수정 전: 12시간·빈 문자열).
  2) 같은 chunked 경로로 `{"ttl":"nonsense"}` 를 보내면 `400` + `invalid ttl` 이고 세션 생성이 **불리지 않는다**(주입 seam 안의 호출 카운터로 단정).
  3) 기존 계약이 그대로다: 본문 없음(`ContentLength == 0`) → 기본 12시간 성공 / `Content-Length` 가 있는 `{"ttl":"1h"}` → 1시간 / 길이가 있는 깨진 JSON → `400 invalid request body` / **빈** chunked 본문(ContentLength -1, 0바이트) → 지금처럼 기본 12시간 성공(여기서 400 이 되면 안 된다) / `admin`·`manager` 아닌 역할 → `403 permission denied`.
  4) 테스트는 "본문이 실제로 파싱되어 ttl 이 반영된다" 와 "잘못된 ttl 이 store 에 닿기 전에 거절된다" 를 각각 증명한다.

- 건드릴 파일 (프로덕션 2개):
  - `internal/httpapi/openbao.go:241 baoTokenCreate` — 게이트를 `r.ContentLength != 0` 로 바꾸고, 길이 미지(-1)인데 본문이 비어 있는 경우를 성공으로 넘기기 위해 **선택적 본문 전용 디코더**를 쓴다. `decodeBaoJSON`(openbao.go:883-896)은 **손대지 말 것** — transit/kv 핸들러는 빈 본문이 `400` 이어야 한다. 같은 파일에 형제 헬퍼를 추가하는 형태를 권한다:
    `func decodeBaoOptionalJSON(w http.ResponseWriter, r *http.Request, dst any) bool` — `http.MaxBytesReader(w, r.Body, maxJSONBody)` + `UseNumber()` 로 `decodeBaoJSON` 과 동일하게 구성하고, **첫** `Decode` 가 `io.EOF` 면 `true`(본문 없음)로 반환, 그 밖의 오류는 `baoError(w, 400, "invalid request body")`, 성공 후에는 `decodeBaoJSON` 과 같은 단일 JSON 확인(`"request body must contain a single JSON value"`)을 유지한다.
  - `internal/httpapi/server.go:19-51`(Server 구조체) + `server.go:220-232`(`sessionResolver`/`sessionRevoker` 의 nil-이면-store 헬퍼 패턴) — 테스트에서 ttl 전달을 관찰할 수 있도록 `sessionCreator func(context.Context, string, string, string, string, time.Duration) (string, model.Session, error)` seam 과 그 헬퍼(`createSession`)를 **그 패턴 그대로** 추가하고, `baoTokenCreate` 가 `s.store.CreateSession(...)` 대신 헬퍼를 부르게 한다. 실제 시그니처는 `internal/store/users.go:52 func (s *Store) CreateSession(ctx, userID, kind, name, createdBy string, ttl time.Duration) (string, model.Session, error)` 이고 호출 인자는 `(..., session.User.ID, "openbao", input.DisplayName, session.User.ID, ttl)` 다 — `"openbao"` kind 문자열은 토큰 접두사(`hvs.`)를 결정하므로 바꾸지 말 것.
    - seam 을 새로 만들고 싶지 않다면 **대안**: 수용 기준 2)만 `&Server{}`(nil store) 로 핸들러를 직접 불러 `400 invalid ttl` 로 증명한다(수정 전에는 nil store 까지 내려가 panic 한다 — 테스트가 깨지는 방식이 보기 나쁘고 패키지 전체 실행이 중단되므로 seam 쪽을 권한다). 기준 1)은 seam 없이는 증명할 수 없다.
  - 신규 테스트 `internal/httpapi/openbao_token_create_test.go` — 하네스는 transit 테스트 관례를 따른다(`openbao_transit_batch_test.go:18 transitBatchRequest` 참고): `httptest.NewRequest(http.MethodPost, "/v1/auth/token/create", strings.NewReader(body))` + `request.WithContext(context.WithValue(request.Context(), sessionKey, model.Session{User: model.User{ID: "user-1", Username: "hong", Role: "manager"}, ExpiresAt: …}))` 로 만들고 `server.baoTokenCreate(rec, request)` 를 **직접** 부른다(`withBaoAuth` 는 store 가 필요하다). chunked 는 요청을 만든 뒤 `request.ContentLength = -1` 로 만든다 — 이 저장소에 이미 같은 기법의 전례가 있다(`tracking_test.go:332`).
  - 선택: `docs/guides/compatibility.md` 의 token create 설명에 "선택적 본문은 길이를 알 수 없는(chunked) 요청에서도 파싱된다" 를 한 줄 추가(해당 줄이 있는지 먼저 확인하고, 없으면 문서는 건드리지 말 것).

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `go test ./internal/httpapi/ -run 'OpenBaoToken|Transit' -count=1 -v` (패키지 전체가 1초 안쪽)
  - `go test ./internal/httpapi/ -race -count=5`
  - `go test ./... -count=1` (PostgreSQL DSN 없으면 lifecycle 테스트는 skip)
  - `go vet ./...` / `gofmt -l internal/httpapi/openbao.go internal/httpapi/server.go`
  - `bash scripts/verify.sh` (프런트 59/59 포함, 시간 소요). `make release-check` 는 돌리지 말 것.

- 위험과 피할 것:
  - `withBaoAuth`·`auth_handlers.go`·`oidc*.go`·`store/users.go` 의 세션 발급 로직·`migrations/` 는 **건드리지 말 것**(인증 계약). 이번 변경은 "본문을 읽는지" 와 seam 추가뿐이다.
  - 기본 12시간, 30일 상한, `parsed <= 0` 거절, 403 역할 판정, 응답 envelope 키(`client_token`·`accessor`·`lease_duration` 등)를 바꾸지 말 것.
  - `decodeBaoJSON` 을 공유 수정하면 transit·kv 의 "빈 본문은 400" 계약이 깨진다 — 반드시 **새 헬퍼**로 분리할 것.
  - `httptest.NewRequest` 의 target 에 공백이 있으면 `malformed HTTP version` 으로 panic 한다.
  - 중간 값·소스 문자열 검사로 끝내지 말고 **핸들러가 최종적으로 전달한 ttl 값**(seam 인자)과 **응답 상태·본문**을 단정할 것. seam 을 주입하는 것이므로 "불리지 않았다" 는 카운터로 증명할 것.
  - 릴리즈 파일(`scripts/version.sh`·`CHANGELOG.md`)은 이 회차에서 올리지 않는다.
  - 미확인: 실제 OpenBao/Vault 가 chunked token-create 본문을 어떻게 다루는지는 오프라인에서 확인하지 못했다(이 과제는 "같은 서버 안의 다른 게이트들과 일치시키고 호출자가 보낸 값을 버리지 않는다" 를 근거로 한다). 수정 전 12시간 폴백도 **코드 독해**이고 실행 재현은 아직 하지 않았다 — 구현자가 먼저 실패하는 테스트로 재현한 뒤 고칠 것.

- 차선 후보: 단건 transit 과 batch 의 계약 동등성을 한 테이블로 고정하는 회귀 테스트 (2/1/S) — 같은 본문을 단건/`batch_input` 두 형태로 보내 미지원 파라미터·필드 누락·빈 문자열 판정 메시지가 **글자까지** 같은지 단정한다. 프로덕션 변경이 없다. 하네스: `transitBatchRequest`(openbao_transit_batch_test.go:18)·`decodeBatchResults`(:25)·`allowAllTransit`(:37)·`decodeBaoData`(openbao_transit_missing_field_test.go:14).
