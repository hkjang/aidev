- 과제: OpenBao Transit **단건** encrypt·decrypt 가 미지원 파라미터(`context`·`nonce`·`associated_data`·`key_version`)를 조용히 무시하는 문제 수정 (가치 3 / 위험 2 / 작업량 S)
- 왜: batch 경로는 `baoTransitBatchItem.unsupportedParameter()`(openbao.go:620-633)로 이 네 파라미터를 per-item `error` 로 거절하고, 그 의도가 코드 주석(openbao.go:585-589)에 "the remaining OpenBao parameters are reported as a per-item error instead of being ignored so a client never receives a result that silently dropped the derivation context or key version it asked for" 라고 적혀 있다. 그런데 단건 경로의 입력 구조체(openbao.go:655-658·734-737)에는 이 네 필드가 아예 없고 `decodeBaoJSON`(openbao.go:868-881)이 미지의 필드를 허용하므로, `{"plaintext":"aGk=","context":"Y3R4"}` 는 context 가 버려진 채 **200 + ciphertext** 를 받는다 — 같은 서버의 두 경로가 정반대 계약을 갖고, 파생 컨텍스트를 요청한 호출자가 그보다 약한 암호문을 성공으로 받는다.
- 수용 기준:
  1) 단건 `POST /v1/transit/encrypt/{key}` 에 `context`·`nonce`·`associated_data` 중 하나가 **비지 않은 값**으로, 또는 `key_version` 이 **0 이 아닌 값**으로 오면 `400` 과 batch 와 **같은 문자열** `unsupported transit parameter: <이름>` 을 돌려주고, 주입한 `transitEncryptor` 가 **호출되지 않는다**(카운터로 단정).
  2) 단건 decrypt 도 같은 네 파라미터에 같은 `400`·같은 메시지를 돌려주고 `transitDecryptor` 가 호출되지 않는다.
  3) 기존 계약이 그대로다: ① `batch_input` 이 있으면 단건 필드(이 네 개 포함)는 무시된다(compatibility.md:125) ② 빈 문자열 `context`/`nonce`/`associated_data` 와 `key_version: 0` 은 "요청하지 않음" 으로 보아 정상 처리된다 ③ 필드 누락 400(`missing plaintext to encrypt`·`missing ciphertext to decrypt`)과 정상 평문·정상 ciphertext 왕복은 그대로다. 테스트는 수정 전 미지원 파라미터 케이스만 200→실패하고 기존 계약 케이스는 처음부터 통과해야 한다.
- 건드릴 파일 (프로덕션 2개 + 테스트 1개):
  - `internal/httpapi/openbao.go:655-658` (`baoTransitEncrypt` 의 익명 입력 구조체) — 권장 형태:
    `var input struct { baoTransitBatchItem; BatchInput *[]baoTransitBatchItem \`json:"batch_input"\` }`
    encoding/json 은 익명 임베딩 구조체의 필드를 평평하게 취급하므로 `plaintext`·`context`·`nonce`·`associated_data`·`key_version` 이 모두 디코드되고 `input.Plaintext`·`input.unsupportedParameter()` 가 promote 된다(같은 패키지이므로 비공개 타입 임베딩 가능). 임베딩이 싫으면 네 필드를 명시적으로 추가하고 `baoTransitBatchItem{...}.unsupportedParameter()` 를 불러도 된다 — 메시지만 batch 와 같으면 된다.
  - `internal/httpapi/openbao.go:668-676` — `if input.BatchInput != nil { … return }` 분기(668-671) **뒤**, `if input.Plaintext == nil`(676) **앞**에
    `if name := input.unsupportedParameter(); name != "" { baoError(w, http.StatusBadRequest, "unsupported transit parameter: "+name); return }`
    를 넣는다. 순서(unsupported → missing)는 batch 루프의 `switch`(openbao.go:703-708)와 같아야 `{"context":"Y3R4"}`(plaintext 없음)가 두 경로에서 같은 메시지를 낸다.
  - `internal/httpapi/openbao.go:734-737` + `747-753` (`baoTransitDecrypt`) — 같은 두 변경을 `Ciphertext` 쪽에 적용(batch 분기 747-750 뒤, 753 의 nil 검사 앞).
  - `internal/httpapi/openbao_transit_unsupported_param_test.go` (신규) — 아래 하네스로 서브테스트 작성.
  - `docs/guides/compatibility.md` — ① 125행 직전(`단건 요청에 plaintext·ciphertext 필드가 아예 없으면 …` 문단, 123행) 뒤에 단건도 네 파라미터를 `400` 으로 거절한다는 한 문장을 추가하고, ② 131행 bullet `- context, nonce, associated_data, 0이 아닌 key_version은 구현하지 않으므로 조용히 무시하지 않고 해당 항목을 error로 처리합니다.` 는 batch 절 안에 있으므로 그대로 두거나 "batch 는 항목별 `error`, 단건은 `400`" 으로 다듬는다. 26행 표의 `… 지정과 rewrap, sign, HMAC … 미구현` 은 수정 불필요(미구현이라는 사실은 그대로다).
- 검증 명령:
  - `go test ./internal/httpapi/ -run 'Transit' -count=1 -v`  ← 이 패턴이 기존 transit 테스트를 전부 포함한다(정찰 시점 baseline `ok … 0.137s`)
  - `go test ./internal/httpapi/ -run 'Transit' -race -count=5`
  - `go test ./... -count=1`
  - `go vet ./...` / `gofmt -l internal/httpapi/openbao.go internal/httpapi/openbao_transit_unsupported_param_test.go` (출력 없어야 함)
  - `bash scripts/verify.sh` (느림, 마지막에 한 번)
- 위험과 피할 것:
  - **`withBaoAuth` 를 타는 통합 테스트를 쓰지 말 것** — store 가 필요하다. 이 파일의 관례대로 핸들러를 직접 부른다. 하네스는 전부 이미 있다: `transitBatchRequest(target, key, body)`(openbao_transit_batch_test.go:18), `allowAllTransit()`(:37), `decodeBaoData(t, body)`(openbao_transit_missing_field_test.go:14). 서버는 `&Server{transitAuthorizer: allowAllTransit(), auditRecorder: func(context.Context, model.AuditEvent) error { return nil }, transitEncryptor: …, transitDecryptor: …}` 로 만든다(openbao_transit_missing_field_test.go:37-44 가 그대로 쓸 수 있는 본보기).
  - 수용 기준 1·2 의 "호출되지 않는다" 는 주입 seam 안에서 **카운터를 올려** 단정할 것 — 이전 회차에서 seam 주입 때문에 결함이 400 이 아니라 200 으로 나타난 전례가 있다. 최종 상태 코드와 본문 문자열도 함께 단정한다.
  - **검사를 `BatchInput != nil` 분기 앞으로 올리지 말 것** — compatibility.md:125 의 "batch_input 이 있으면 단건 필드는 무시" 계약이 깨진다. 기존 계약 테스트에 `{"batch_input":[{"plaintext":"aGk="}],"context":"Y3R4"}` 가 여전히 200 인 케이스를 넣어 이 경계를 고정하라.
  - 빈 문자열/0 을 거절로 바꾸지 말 것 — `unsupportedParameter()` 가 이미 `*Context != ""`·`*KeyVersion != 0` 으로 걸러낸다. batch 와 동작이 갈라지면 안 된다.
  - 임베딩을 쓰면 encrypt 단건이 `ciphertext`·`reference` 필드도 디코드하게 되지만 **관찰 가능한 변화는 없다**(지금도 `decodeBaoJSON` 이 미지의 필드로 받아 무시한다). 이 김에 `DisallowUnknownFields` 를 켜지 말 것 — 별도 호환성 계약 변경이다.
  - `recordSensitiveDisclosure` 실패 시 503 으로 평문을 withhold 하는 decrypt 의 순서(openbao.go:768-772)를 건드리지 말 것.
  - auth·migrations·workflows·settings·프런트는 건드리지 않는다. 버전 bump·CHANGELOG 는 이 과제 범위가 아니다.
  - **미확인**: 수정 전 단건 `context` 요청이 실제로 200 을 돌려준다는 것은 코드 독해(입력 구조체에 필드 없음 + `decodeBaoJSON` 이 미지의 필드 허용)로 단정한 것이며, 정찰에서 실행 재현하지는 않았다. 구현자는 테스트를 먼저 써서 수정 전 실패를 눈으로 확인하라.
- 차선 후보: Transit ciphertext 버전 토큰이 비정규 표기(`vault:v01:`·`vault:v+1:`)를 받아들이는 문제 (3/1/M) — `internal/store/transit.go:47` 의 `strconv.Atoi(strings.TrimPrefix(parts[1], "v"))` 가 부호·선행 0 을 통과시키고 AAD 를 파싱된 정수로 재조립해 AEAD 도 통과하므로 OpenBao 가 거절하는 ciphertext 에 200+평문을 준다. `Store` 에 seam 이 없어 200→400 을 정직하게 보려면 `JIKIM_TEST_POSTGRES_DSN` 통합 테스트가 필요하다(전례 `internal/store/secrets_openbao_test.go`, docker 사용 가능). `&store.Store{}`(nil pool)로 부르면 panic 하므로 재현 근거로 쓰지 말 것.
