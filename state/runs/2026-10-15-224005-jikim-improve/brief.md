- 과제: OpenBao Transit 단건 encrypt·decrypt 가 `plaintext`·`ciphertext` **필드 누락**을 빈 문자열로 처리하는 문제 수정 (가치 3 / 위험 1 / 작업량 S)
- 왜: `baoTransitEncrypt`(internal/httpapi/openbao.go:653-663)의 입력 구조체가 `Plaintext string` 이라 필드가 **아예 없는** 요청(`{}`)과 `""` 인 요청을 구분하지 못해, `{}` 로도 빈 평문을 암호화한 **200 + ciphertext** 를 돌려준다. 같은 파일 batch 경로는 `baoTransitBatchItem.Plaintext *string`(openbao.go:592-594)로 받아 누락을 `missing plaintext to encrypt` 400 으로 거절하므로(baoTransitEncryptBatch 의 `case item.Plaintext == nil:`), 같은 엔드포인트가 단건과 batch 에서 다르게 답한다. 고치면 단건이 batch 와 같은 계약을 쓰고(OpenBao 도 필드가 없으면 `missing plaintext to encrypt`, 명시된 `""` 는 정상 암호화), 누락 요청이 store/DB 까지 내려가지 않는다.
  - 참고(디코드 경로 확인): 누락 시 `""` 가 `s.encryptTransit`→`store.TransitEncrypt`(internal/store/transit.go:22)로 내려가고 `base64.StdEncoding.DecodeString("")` 는 성공하므로 빈 평문이 **실제로 암호화된다**. decrypt 쪽은 `Ciphertext string` 이 `""` 로 내려가 `store.TransitDecrypt`(transit.go:43-46)가 `ErrInvalid: ciphertext 형식` 을 돌려 상태는 이미 400 이지만, 메시지가 batch 의 `missing ciphertext to decrypt` 와 다르고 무의미한 store 호출이 한 번 난다.
- 수용 기준:
  1) `POST /v1/transit/encrypt/{key}` 에 `{}`(또는 `{"reference":"a"}` 처럼 plaintext 없는 본문)를 주면 **400** 과 `missing plaintext to encrypt` 를 돌려주고, 주입한 `transitEncryptor` 가 **한 번도 호출되지 않는다**(호출 카운터로 단정).
  2) `{"plaintext":""}` 는 **지금처럼 200** 이고 `data.ciphertext`·`data.key_version` 이 그대로 나온다(명시된 빈 평문은 유효 입력 — OpenBao 와 동일). `{"plaintext":"aGk="}` 정상 경로도 200 유지.
  3) `POST /v1/transit/decrypt/{key}` 에 `{}` 를 주면 400 과 `missing ciphertext to decrypt` 를 돌려주고 주입한 `transitDecryptor` 가 호출되지 않는다. `{"ciphertext":"vault:v1:a"}` 정상 경로는 그대로.
  4) `{"batch_input":[{"plaintext":"aGk="}]}` 처럼 **top-level 필드가 없는 batch 요청**이 계속 200/batch_results 를 돌려준다(새 nil 검사가 batch 분기보다 **뒤**에 있어야 한다 — 이 순서를 틀리면 모든 batch 요청이 400 이 된다).
  5) 테스트가 증명할 것: 수정 전 1)·3)이 실패(encrypt 는 `status=200`, decrypt 는 본문이 `ciphertext 형식`)하고 2)·4)는 수정 전후 모두 통과한다. 수정 줄만 되돌려 같은 서브테스트가 같은 메시지로 다시 실패하는지 확인하라.
- 건드릴 파일:
  - `internal/httpapi/openbao.go:653` `baoTransitEncrypt` — 익명 입력 구조체의 `Plaintext string` → `Plaintext *string`, `input.BatchInput != nil` 분기 **다음에** `if input.Plaintext == nil { baoError(w, http.StatusBadRequest, "missing plaintext to encrypt"); return }` 추가, 672행 호출을 `*input.Plaintext` 로. (이 함수 안에서 `input.Plaintext` 를 읽는 자리는 672행 하나뿐임을 확인했다.)
  - `internal/httpapi/openbao.go:725` `baoTransitDecrypt` — 같은 방식으로 `Ciphertext *string` + batch 분기 뒤 `missing ciphertext to decrypt` 400 + 744행 호출을 `*input.Ciphertext` 로. (읽는 자리는 744행 하나뿐.)
  - 테스트: `internal/httpapi/openbao_transit_errors_test.go` 에 서브테스트 추가(또는 같은 패키지 신규 파일 1개). 기존 하네스를 그대로 써라 — `transitBatchRequest(target, key, body)`(openbao_transit_batch_test.go:18), `allowAllTransit()`(동 38행), `decodeBatchResults(t, body)`(동 25행), `&Server{transitAuthorizer: allowAllTransit(), auditRecorder: func(context.Context, model.AuditEvent) error { return nil }, transitEncryptor: …, transitDecryptor: …}`. 이 파일의 관례는 `server.baoTransitEncrypt(recorder, transitBatchRequest(...))` 로 핸들러를 직접 부르는 것이다(`withBaoAuth` 토큰 미들웨어는 store 가 필요해 통과시키지 않는다) — 관례를 바꾸지 말고 그대로 따르고, decrypt 성공 경로는 `recordSensitiveDisclosure` 가 `auditRecorder` 를 타므로 반드시 주입하라.
  - 프로덕션 파일 1개 + 테스트 파일 1개로 끝낸다.
- 검증 명령:
  - `go test ./internal/httpapi/ -run 'Transit' -count=1 -v`
  - `go test ./internal/httpapi/ -run 'Transit' -race -count=5`
  - `go test ./... -count=1` · `go vet ./...` · `gofmt -l internal/httpapi/openbao.go <테스트파일>`(출력 없음)
  - `bash scripts/verify.sh`(시간 소요, exit 0 과 `검증 완료: jikim v0.2.36` 확인)
- 위험과 피할 것:
  - **batch 분기 순서**가 유일한 실질 위험이다(수용 기준 4). `docs/guides/compatibility.md:123` 이 "batch_input 이 있으면 단건 필드는 무시한다" 를 계약으로 적어 두었으므로 nil 검사를 batch 분기 앞에 두면 문서화된 계약을 깬다.
  - `baoTransitBatchItem`·`unsupportedParameter`·`baoTransitBatchStatus`·`baoTransitFailure`·`transitCiphertextVersion` 은 건드리지 마라. 단건 경로가 `context`·`nonce` 같은 미지원 파라미터를 조용히 무시하는 것은 **별개의 더 큰 과제**다(아이디어 파일에 남겼다) — 이번에 끼워 넣지 말 것.
  - `store/transit.go` 는 손대지 마라. 이번 수정은 HTTP 입력 계약만 좁히는 것이고, store 의 ciphertext 파서를 함께 고치면(교훈: 여러 경로의 같은 파서를 통합하지 말 것) 범위가 PostgreSQL 통합 테스트로 번진다.
  - 감사(audit) 메시지·`recordSensitiveDisclosure` 순서·에러 envelope 형식은 그대로 둔다. 버전/CHANGELOG 는 릴리즈 단계 몫이다.
  - 400 응답은 `baoError` 로만 쓰고 입력 원문(평문·ciphertext)을 본문이나 감사 details 에 넣지 마라.
- 차선 후보: Transit ciphertext 버전 토큰의 비정규 표기 거절 — `store.TransitDecrypt`(internal/store/transit.go:47)이 `strconv.Atoi(TrimPrefix(parts[1],"v"))` 를 쓰므로 `vault:v01:`·`vault:v+1:` 이 통과하고 AAD 를 파싱된 정수로 재조립해 AEAD 까지 통과, 200 + 평문이 나온다(OpenBao 는 거절). 3/1/M — `Store` 에 seam 이 없어 200→400 을 정직하게 보려면 `JIKIM_TEST_POSTGRES_DSN` 통합 테스트(docker 사용 가능, 전례 internal/store/secrets_openbao_test.go·2026-10-08 회차)가 필요하다. 1순위가 성립하지 않을 때만 고르고, nil pool `&store.Store{}` 로 panic 을 재현 근거로 쓰지는 마라.
