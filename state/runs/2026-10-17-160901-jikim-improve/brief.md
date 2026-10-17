- 과제: OpenBao `auth/token/create` 가 길이를 알 수 없는(chunked) 요청 본문을 조용히 버려 `ttl`·`display_name` 이 무시되고 잘못된 JSON 도 통과하는 문제 수정 (가치 3 / 위험 2 / 작업량 S)
- 왜: `internal/httpapi/openbao.go:241` 의 게이트가 `if r.ContentLength > 0 && !decodeBaoJSON(w, r, &input)` 라서, `Transfer-Encoding: chunked` 로 온 요청(Go 서버가 `r.ContentLength = -1` 로 채움)은 본문을 **한 바이트도 읽지 않고** 기본값으로 진행한다. 그래서 `{"ttl":"1h"}` 를 chunked 로 보낸 호출자는 1시간 토큰을 요청했는데 **12시간** 토큰을 200 으로 받고(수명 6배 초과 발급), `{"ttl":"99999h"}` 같은 거절 대상도 400 이 아니라 기본 TTL 로 성공하며, 깨진 JSON(`{"ttl":`)도 400 없이 통과한다. 같은 저장소의 다른 세 곳(`resource_handlers.go:521`·`oidc.go:68`·`ai.go:228`)은 모두 `r.ContentLength != 0` 으로 적어 chunked 를 정상 처리하므로, 이 한 줄만 다른 판정을 하고 있다.
- 수용 기준:
  1) `POST /v1/auth/token/create` 에 `ContentLength == -1` 인 요청으로 `{"ttl":"99999h"}` 를 보내면 `400` + `invalid ttl`(기존 Content-Length 있는 요청과 **글자까지 같은** 본문)을 돌려주고, 세션 생성에는 **도달하지 않는다**.
  2) 같은 chunked 요청으로 깨진 JSON(`{"ttl":`)을 보내면 `400` + `invalid request body`(= `decodeBaoJSON` 의 기존 메시지)를 돌려준다. 두 개의 JSON 값(`{} {}`)은 `request body must contain a single JSON value`.
  3) **본문이 없는 요청은 지금처럼 성공해야 한다** — `Content-Length: 0`(현행)뿐 아니라 `ContentLength == -1` + 빈 본문(chunked 로 0바이트)도 기본 12시간 토큰으로 성공한다. 즉 `decodeBaoJSON` 을 그대로 무조건 호출하면 안 된다(빈 스트림에서 `Decode` 가 `io.EOF` 를 내므로 400 이 된다) — 빈 본문을 허용하는 변형 헬퍼가 필요하다.
  4) chunked 로 `{"ttl":"1h","display_name":"ci"}` 를 보내면 응답 `auth.lease_duration == 3600` 이고 세션 생성에 넘어간 이름이 `"ci"` 다(= 본문이 실제로 **읽힌다**). 이 단정은 아래 `sessionCreator` seam 이 있어야 DB 없이 가능하다.
  5) 기존 Content-Length 경로(본문 없음 / 정상 `ttl` / 잘못된 `ttl` / 깨진 JSON)의 판정은 바뀌지 않는다.

- 건드릴 파일:
  - `internal/httpapi/openbao.go:241` (`baoTokenCreate`) — `r.ContentLength > 0 && !decodeBaoJSON(...)` 게이트를 **ContentLength 를 보지 않는** 호출로 교체: `if !decodeOptionalBaoJSON(w, r, &input) { return }`.
  - `internal/httpapi/openbao.go:883-896` (`decodeBaoJSON` 옆) — 빈 본문을 허용하는 헬퍼를 추가. 중복을 피하려면 공용 코어로 뽑는 형태를 권장:
    ```go
    func decodeBaoJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
        return decodeBaoJSONBody(w, r, dst, false)
    }

    // decodeOptionalBaoJSON 은 본문이 아예 없는 요청을 기본값으로 통과시킨다.
    // OpenBao 는 auth/token/create 를 본문 없이도 받으므로 그 계약은 유지하되,
    // 길이를 알 수 없는(chunked) 본문도 반드시 읽어야 한다.
    func decodeOptionalBaoJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
        return decodeBaoJSONBody(w, r, dst, true)
    }

    func decodeBaoJSONBody(w http.ResponseWriter, r *http.Request, dst any, allowEmpty bool) bool {
        r.Body = http.MaxBytesReader(w, r.Body, maxJSONBody)
        decoder := json.NewDecoder(r.Body)
        decoder.UseNumber()
        if err := decoder.Decode(dst); err != nil {
            if allowEmpty && errors.Is(err, io.EOF) {
                return true
            }
            baoError(w, http.StatusBadRequest, "invalid request body")
            return false
        }
        if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
            baoError(w, http.StatusBadRequest, "request body must contain a single JSON value")
            return false
        }
        return true
    }
    ```
    (`json`·`errors`·`io`·`http` 는 이 파일에 이미 import 되어 있다 — `decodeBaoJSON` 이 넷 다 쓴다. 새 import 는 필요 없다.)
  - `internal/httpapi/server.go:28-49`(`Server` 구조체) + `:221-226`(`resolveSession` 바로 아래) — 기존 `resolveSession`/`revokeSessionToken`/`authenticate` 와 **똑같은 모양**의 seam 1개 추가:
    ```go
    sessionCreator func(context.Context, string, string, string, string, time.Duration) (string, model.Session, error)

    func (s *Server) createSession(ctx context.Context, userID, kind, name, createdBy string, ttl time.Duration) (string, model.Session, error) {
        if s.sessionCreator != nil {
            return s.sessionCreator(ctx, userID, kind, name, createdBy, ttl)
        }
        return s.store.CreateSession(ctx, userID, kind, name, createdBy, ttl)
    }
    ```
    그리고 `openbao.go:254` 의 `s.store.CreateSession(...)` 를 `s.createSession(...)` 로 바꾼다. (`store.CreateSession` 의 실제 시그니처는 `internal/store/users.go:52`: `(ctx, userID, kind, name, createdBy string, ttl time.Duration) (string, model.Session, error)`.) **`internal/store/users.go` 는 건드리지 않는다.**
  - 테스트 신규 1파일: `internal/httpapi/openbao_token_create_body_test.go`.
  - 문서: `docs/guides/compatibility.md` — Token 행은 **25행**(`| Token | /v1/auth/token/{lookup-self,create,revoke-self} | 제한 | … create TTL 최대 30일, parent/renew/wrap 의미 미지원 |`)이고 인증 계약 bullet 은 **51-52행**이다(확인). 51-52행 근처 bullet 로 "`auth/token/create` 의 본문은 `Content-Length` 가 없는(chunked) 요청에서도 읽히며, 본문이 아예 없는 요청만 기본 TTL(12시간)로 통과합니다" 를 1문장 추가. 표의 25행은 손대지 않아도 된다.

- 정찰이 실제로 실행해 본 재현 (임시 테스트로 확인 후 삭제, 작업 트리는 clean):
  ```
  === RUN   .../content_length_invalid_ttl
      OK status=400 body={"errors":["invalid ttl"]}
  === RUN   .../chunked_invalid_ttl
      PANIC (body reached store.CreateSession with nil store): runtime error: invalid memory address or nil pointer dereference
  === RUN   .../chunked_broken_json
      PANIC (body reached store.CreateSession with nil store): runtime error: invalid memory address or nil pointer dereference
  --- FAIL (0.141s) — content_length_invalid_ttl 만 PASS
  ```
  즉 **Content-Length 가 있는 똑같은 본문은 400 `invalid ttl` 로 거절되는데, `ContentLength = -1` 로 바꾼 같은 본문은 `ttl` 검사도 JSON 검사도 통과해 세션 생성까지 내려간다**. 이 하네스(`quietServer` + `sessionResolver` 주입 + `http.NewServeMux()` + `server.openBaoRoutes(mux)` + `X-Vault-Token` 헤더)가 store 없이 실제 라우트와 `withBaoAuth` 를 통과하는 것도 같은 실행으로 확인했다.

- 검증 명령:
  - 하네스: `quietServer()`(auth_outage_test.go:16) + `server.sessionResolver = func(...) (model.Session, error)` 로 admin 세션을 주입하고, `mux := http.NewServeMux(); server.openBaoRoutes(mux)` 로 **실제 라우트 + `withBaoAuth`** 를 통과시킨다(`openBaoRoutes` 는 핸들러만 등록하므로 store 가 nil 이어도 구성된다. `withBaoAuth` 는 `s.resolveSession` 을 쓰고 그것이 seam 을 존중한다 — openbao.go:54, server.go:221-226). 토큰 헤더는 `X-Vault-Token`(`tokenRequest` auth_outage_test.go:20 참고). `sessionCreator` 에는 호출 **카운터**와 받은 `name`·`ttl` 을 기록하는 함수를 주입해, 거절 케이스에서 **0회 호출**을 단정한다.
  - chunked 재현: `request := httptest.NewRequest("POST", "/v1/auth/token/create", strings.NewReader(body))` 뒤에 `request.ContentLength = -1` 을 **명시적으로** 세팅한다(이 저장소의 기존 선례: `internal/httpapi/tracking_test.go:332`). `httptest.NewRequest` 는 `*strings.Reader` 본문에 길이를 채우므로 이 한 줄이 꼭 필요하다. target 에 **공백을 넣지 말 것**(`malformed HTTP version` panic).
  - `go test ./internal/httpapi/ -run 'TokenCreate|BaoToken' -count=1 -v`
  - `go test ./internal/httpapi/ -run 'TokenCreate|Bao|Transit|Session' -count=1 -race -count=5`
  - `go test ./... -count=1` · `go vet ./...` · `gofmt -l internal/httpapi/openbao.go internal/httpapi/server.go internal/httpapi/openbao_token_create_body_test.go`
  - `bash scripts/verify.sh` (exit 0, 프런트 59/59 기대)

- 위험과 피할 것:
  - **수정 전 증상은 400 도 200 도 아니라 nil store panic 이다(정찰에서 실행 확인).** `Server.store` 가 nil 인 테스트에서 거절 케이스가 수정 전에 store 까지 내려가면 `s.store.CreateSession` 에서 `invalid memory address or nil pointer dereference` 로 panic 한다(bare mux 에는 `recoverer` 가 없다). `sessionCreator` seam 을 **먼저** 넣고 거기에 기록 함수를 주입하면 수정 전 증상이 "400 대신 200 + 카운터 1회" 로 깔끔하게 나온다 — 그렇게 쓰는 것을 권장한다(2026-10-15 transit 회차에서 과제서가 예상한 수정 전 본문이 실제와 달랐던 자리와 같은 함정). **실제로 관찰된 수정 전 응답을 원장에 적을 것.**
  - 빈 본문 계약(수용 기준 3)을 깨지 말 것. `r.ContentLength != 0` 로만 바꾸면 chunked 는 고쳐지지만 **빈 chunked 본문이 400** 이 된다(`Decode` 가 `io.EOF`). 그래서 `allowEmpty` 변형이 필요하다.
  - `decodeBaoJSON` 의 기존 두 메시지(`invalid request body`, `request body must contain a single JSON value`)와 `maxJSONBody` 상한·`UseNumber`·단일 JSON 검사를 바꾸지 말 것. `DisallowUnknownFields` 를 추가하지 말 것(보류 아이디어에서 호환성 계약 변경으로 기각된 사안).
  - `openbao.go:241` 외의 다른 `ContentLength` 호출부(`resource_handlers.go:521`·`oidc.go:68`·`ai.go:228`)는 이미 `!= 0` 이라 chunked 를 읽는다 — **이번 회차에 함께 손대지 말 것**(auth/OIDC/AI 는 위험 구역이고 결함도 없다).
  - `withBaoAuth`·`baoSessionFailure`·권한 검사(`session.User.Role != "admin" && != "manager"`) 순서를 바꾸지 말 것. 권한 거절은 본문 디코드 **전**에 남아야 한다(현재 openbao.go:233-236).
  - `scripts/version.sh`·CHANGELOG 는 이 회차에서 올리지 않는다.

- 차선 후보: 단건 transit 과 batch 의 계약 동등성을 한 테이블로 고정하는 회귀 테스트 (2/1/S) — `openbao_transit_missing_field_test.go`·`openbao_transit_unsupported_param_test.go`·`openbao_transit_batch_test.go` 에 흩어진 같은 판정(미지원 파라미터·필드 누락·빈 문자열)을 같은 본문을 단건/`batch_input` 두 형태로 보내 메시지가 글자까지 같은지 단정하는 한 표로 묶는다. 프로덕션 변경이 없다. 하네스: `transitBatchRequest` openbao_transit_batch_test.go:18, `decodeBatchResults` :25, `allowAllTransit` :37, `decodeBaoData` openbao_transit_missing_field_test.go:14.
