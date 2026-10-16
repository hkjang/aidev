# 과제서 (2026-10-16, base main@306196e, v0.2.38)

- 과제: Transit decrypt 가 비정규 버전 토큰(`vault:v01:`·`vault:v+1:`·`vault:v001:`)을 거절하지 않고 저장소·복호화까지 내려보내는 문제 수정 (가치 3 / 위험 1 / 작업량 S)
- 왜: `internal/store/transit.go:47` 의 `strconv.Atoi(strings.TrimPrefix(parts[1], "v"))` 는 부호(`+1`)와 선행 0(`01`·`001`)을 통과시키고, 그 뒤 AAD 를 **파싱된 정수**로 재조립(`transit:%s:v%d`, :66)하므로 AEAD 도 통과한다 — 즉 `vault:v01:<payload>` 가 `vault:v1:<payload>` 와 똑같이 평문을 돌려준다(버전 토큰 텍스트가 사실상 가변이 되고, 같은 암호문에 여러 표기가 생긴다). compatibility.md:120 은 ciphertext 형식을 `vault:vN:...` 로 고정하고 136 행은 "ciphertext 형식" 오류를 `400` 으로 약속하는데, 이 세 표기는 형식 검증을 그냥 통과한다. 고치면 jikim 의 ciphertext 표기가 OpenBao 와 같이 단일 정규형으로 고정된다.
- 수용 기준:
  1) `store.TransitDecrypt` 가 `vault:v01:…`·`vault:v001:…`·`vault:v+1:…`·`vault:v+01:…` 를 **DB 에 닿기 전에** `store.ErrInvalid` 로 거절한다.
  2) 정규 토큰 `vault:v1:…`·`vault:v12:…` 는 지금처럼 저장소 조회로 진행한다(과잉 거절 없음). 기존 거절 케이스(`vault:v0:`·`vault:v-1:`·`vault:vx:`·`vault:v:`·`vault` 접두사 불일치·`len(payload)<=12`·비 base64)는 그대로 `ErrInvalid` 로 남는다.
  3) 테스트가 증명할 것: 비정규 토큰이 "저장소까지 내려가지 않는다" 를 `errors.Is(err, ErrInvalid)` 로 단정한다(아래 검증 방법이 DB 접근 오류와 `ErrInvalid` 를 분리해 준다). 정규 토큰에 대해서는 `!errors.Is(err, ErrInvalid)` 를 함께 단정해 과잉 거절이 없음을 고정한다.
- 건드릴 파일 (프로덕션 1개 + 테스트 1개 + 문서 1줄):
  - `internal/store/transit.go:42-54` — `TransitDecrypt` 의 형식 검증만. 버전 토큰 파싱을 정규 10진수로 좁힌다. 권장 형태: 이 파일 안에 순수 헬퍼를 두고(`func transitCiphertextKeyVersion(token string) (int, bool)` — 인자는 `parts[1]`), `v` 접두사 확인 후 남은 문자열이 비어 있지 않고 **모든 문자가 `'0'`–`'9'`** 이며 **선행 0 이 없을 때**(즉 `len>1 && s[0]=='0'` 이면 거절)만 `strconv.Atoi` 로 변환하고, `version < 1` 은 지금처럼 거절. 이렇게 하면 `+`/`-`/공백/빈 문자열/선행 0 이 한곳에서 걸러진다.
  - `internal/httpapi/openbao.go:871 transitCiphertextVersion` — **건드리지 말 것.** 이 함수는 `baoTransitEncrypt`(:695)와 `baoTransitEncryptBatch`(:726)에서 **jikim 이 방금 만든** ciphertext 만 파싱한다(클라이언트 입력 경로가 아님). 두 파서를 통합하지 말고 store 쪽만 좁힌다.
  - `internal/store/transit_ciphertext_test.go` (신규) — `package store` 내부 테스트. 기존 순수 헬퍼 단위 테스트 관례는 `internal/store/logic_test.go`(`package store`, `TestValidateSecretPath` 등).
  - `docs/guides/compatibility.md` 136 행 부근 — Transit 오류 계약에 "버전 토큰은 `v` + 선행 0 없는 10진수만 유효하며 `vault:v01:`·`vault:v+1:` 같은 비정규 표기는 `400`" 1문장 추가. (이 저장소는 계약 변화를 여기 적는 관례다 — v0.2.37·v0.2.38 회차 모두 compatibility.md 를 함께 고쳤다.)
- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `go test ./internal/store/ -run 'Transit' -count=1 -v` — 패키지가 1초 안쪽. 아래 재현 테스트는 0.14초였다.
  - `go test ./internal/store/ -race -count=5`
  - `go test ./... -count=1` · `go vet ./...` · `gofmt -l internal/store/transit.go`
  - `bash scripts/verify.sh` (시간 소요; exit 0 과 `검증 완료: jikim v0.2.38` 확인)
  - `JIKIM_TEST_POSTGRES_DSN` 은 **필요 없다**(아래 참조). 설정돼 있지 않으면 `secrets_openbao_test.go` 통합 테스트는 skip 된다.
- 재현 방법 — **정찰에서 실행으로 확인함**. `package store` 내부 테스트에서 **닿지 않는 DSN** 의 lazy pool 을 만들어 `&Store{pool: p}` 로 `TransitDecrypt` 를 부르면, 비정규 토큰은 DB 까지 내려가 **연결 오류**(= `ErrInvalid` 아님)로 돌아오고 이미 거절되는 토큰은 `ErrInvalid` 로 돌아온다. 그대로 쓸 수 있는 골격:

  ```go
  cfg, err := pgxpool.ParseConfig("postgres://u:p@127.0.0.1:1/none")
  // err 처리
  cfg.MinConns = 0
  pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
  // err 처리; defer pool.Close()
  s := &Store{pool: pool}
  payload := base64.RawStdEncoding.EncodeToString(make([]byte, 32)) // len(payload)>12 를 통과시키기 위함
  ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
  defer cancel()
  _, err = s.TransitDecrypt(ctx, "k", "vault:v01:"+payload)
  // 수정 전: errors.Is(err, ErrInvalid) == false (연결 거부) / 수정 후: true
  ```

  정찰에서 찍은 실제 출력(수정 전):

  ```
  token=v1    isErrInvalid=false err=failed to connect to `user=u database=none`: 127.0.0.1:1 … connect: connection refused
  token=v01   isErrInvalid=false err=failed to connect to `user=u database=none`: …   ← 결함
  token=v+1   isErrInvalid=false err=failed to connect to `user=u database=none`: …   ← 결함
  token=v001  isErrInvalid=false err=failed to connect to `user=u database=none`: …   ← 결함
  token=v0    isErrInvalid=true  err=요청 값이 올바르지 않습니다
  token=v-1   isErrInvalid=true  err=요청 값이 올바르지 않습니다
  token=vx    isErrInvalid=true  err=요청 값이 올바르지 않습니다
  token=v     isErrInvalid=true  err=요청 값이 올바르지 않습니다
  ```

  `strconv.Atoi` 쪽도 실행으로 확인했다: `"+1"`→1, `"01"`→1, `"001"`→1, `"+01"`→1 (모두 `err=nil`), `"-1"`→-1, `" 1"`·`"1 "`·`""`→ syntax error. 즉 지금 막히는 것은 공백·빈 문자열·음수뿐이고 부호 `+` 와 선행 0 이 구멍이다.
- 위험과 피할 것:
  - `&store.Store{}`(nil pool)로 `TransitDecrypt` 를 부르면 DB 접근 지점에서 **panic** 한다 — 재현 근거로 쓰지 말 것. 위의 lazy pool 방식을 쓰라(`MinConns = 0` 을 꼭 넣고, 포트 1 이라 즉시 refuse 되어 테스트가 매달리지 않는다 — 정찰에서 0.139s).
  - `httpapi` 의 transit 테스트는 `transitEncryptor`/`transitDecryptor` seam 을 주입하므로 **store 의 파서를 타지 않는다** — httpapi 쪽에서 이 결함을 재현하려 하지 말 것(2026-10-15 회차에 같은 함정으로 예상 400 이 200 으로 나왔다).
  - `TransitEncrypt` 의 출력 형식(`transit.go:39` 의 `fmt.Sprintf("vault:v%d:%s", …)`)과 AAD 문자열(`transit:%s:v%d`, :34·:66)은 바꾸지 말 것 — 이미 저장된 암호문이 복호화 불가가 된다. 이번 변경은 **입력 수용 범위만** 좁힌다.
  - `transitKey`·`RotateTransitKey`·`cleanTransitPermissions` 는 건드리지 않는다. 보호 경로(`auth_handlers.go`·`oidc*.go`·`mcp_oauth.go`·`store/users.go`·`migrations/`·릴리즈 workflow)와 `settings`·`webhook` 은 이번 회차 범위 밖.
  - 릴리즈 버전·CHANGELOG 는 별도 작업이다 — `scripts/version.sh` 를 올리지 말 것.
- 차선 후보: 단건 transit 과 batch 의 계약 동등성을 한 테이블로 고정하는 회귀 테스트 (2/1/S — 프로덕션 변경 없음. 지금 `openbao_transit_missing_field_test.go`·`openbao_transit_unsupported_param_test.go`·`openbao_transit_batch_test.go` 에 같은 판정이 흩어져 있어 한쪽만 바뀌어도 통과한다. 같은 본문을 단건 형태와 `batch_input` 형태로 각각 보내 "메시지 문자열이 글자까지 같다" 를 단정하는 표 하나로 묶는다. 하네스: `transitBatchRequest` openbao_transit_batch_test.go:18, `decodeBatchResults` :25, `allowAllTransit` :37, `decodeBaoData` openbao_transit_missing_field_test.go:14)
