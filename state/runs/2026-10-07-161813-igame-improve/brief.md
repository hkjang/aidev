- 과제: secretbox 의 거부 경로와 AAD 결합을 회귀로 고정 (가치 2 / 위험 1 / 작업량 S)
- 왜: 설치키로 DB 비밀(OIDC client_secret, AI API 키)을 봉인하는 `internal/secretbox` 의 테스트는 `TestRoundTripAndRandomNonce` 하나뿐이고, 왕복과 nonce 무작위성만 본다. `Seal`/`Open` 양쪽에서 AAD `[]byte("igame:v1")` 을 지워도 현재 테스트가 통과하고(두 함수를 같이 바꾸면 왕복은 그대로 성립한다), `Open` 의 네 거부 경로(`v1:` 접두사 없음 / base64 깨짐 / `len(data) < NonceSize` / 변조·다른 키)와 `New` 의 잘못된 키 길이 거부도 전부 미검증이다. 고치면 설치키 암호화 계약이 조용히 약해지는 변경을 테스트가 잡는다.
- 수용 기준:
  1) `internal/secretbox` 에 새 테스트가 추가되고 `go test ./internal/secretbox/... -count=1` 이 PASS 한다. 프로덕션 파일은 0개 변경.
  2) `Open` 이 거부해야 하는 입력 전부에 대해 에러를 돌려주는 것이 단정된다: 빈 문자열, `"v1:"` 단독, 접두사 없음(`"v2:…"`, 접두사 없는 생 base64), base64 아님(`"v1:not base64!!"` 또는 표준 base64 의 `+`/`/` 를 포함한 문자열), nonce 보다 짧은 payload, 유효 ciphertext 의 1비트 변조, 같은 평문을 다른 키로 봉인한 ciphertext. 성공 경로도 함께 단정해 테스트가 "전부 실패" 로 공허하게 녹색이 되지 않게 한다.
  3) AAD 결합이 단정된다 — 소스 문자열 검사가 아니라 crypto 로: ① 테스트가 `crypto/aes`+`cipher.NewGCM` 으로 같은 키의 독립 AEAD 를 만들어 `b.Seal(...)` 의 출력(`"v1:"` 떼고 RawURLEncoding 디코드, 앞 `NonceSize()` 바이트가 nonce)을 AAD `[]byte("igame:v1")` 로 `Open` 해 평문이 나오는 것 → `Seal` 이 정확히 그 AAD 를 쓴다. ② 같은 AEAD 로 **다른** AAD(예: `nil` 과 `[]byte("igame:v2")`)를 써서 ciphertext 를 만들고 `"v1:"+RawURLEncoding` 으로 포장해 `b.Open` 에 넣으면 에러 → `Open` 이 정확히 그 AAD 를 요구한다. 이 두 단정이 있으면 AAD 를 양쪽에서 지우거나 값을 바꾸는 변이가 Red 가 된다.
  4) `New` 가 AES 키 길이가 아닌 입력(빈 슬라이스, 31바이트)에 에러를 돌려주고, 16·24·32바이트에는 성공하는 것이 단정된다.
- 건드릴 파일:
  - `internal/secretbox/secretbox_test.go` — 기존 `TestRoundTripAndRandomNonce` 는 그대로 두고 테이블 기반 거부 테스트, AAD 결합 테스트, `New` 키 길이 테스트를 추가. 한 파일로 끝내도 되고 `secretbox_reject_test.go` 로 나눠도 된다(패키지는 `secretbox` — 내부 필드 `b.aead.NonceSize()` 를 써야 하므로 `secretbox_test` 외부 패키지로 두지 말 것).
  - 프로덕션 파일 없음. `internal/secretbox/secretbox.go` 는 읽기만 하고 수정하지 말 것(에러 문구·`v1:` 접두사·AAD 값은 저장된 기존 데이터의 계약이다).
- 검증 명령:
  - `go test ./internal/secretbox/... -count=1 -v`
  - `go test ./internal/secretbox/ -race -count=3`
  - `gofmt -l .` (무출력), `go vet ./...`, `go build ./...`
  - `go test ./cmd/... ./internal/... ./migrations/...` (DSN 없으면 PG 회귀는 skip 됨 — 이 과제는 DB 불필요)
  - `bash scripts/check-release-contract.sh`
  - 인과 확인(변이 → 원복, 매번 녹색 재확인): M1 `Seal`·`Open` 양쪽에서 AAD 를 `nil` 로 → 기준 3)② 가 Red / M2 `Open` 의 AAD 만 `[]byte("igame:v2")` 로 → 왕복과 3)① 이 Red / M3 `Open` 의 `len(data) < b.aead.NonceSize()` 검사 제거 → 짧은 payload 케이스 Red / M4 접두사 검사 제거 → 접두사 케이스 Red.
- 위험과 피할 것:
  - **프로덕션 코드를 고치지 말 것.** 이것은 테스트 공백 보강이다. 읽다가 `secretbox.go` 를 "개선" 하고 싶어지면(예: 에러 문구 통일, `v2:` 추가, AAD 에 key id 넣기) 하지 말고 `ideas.json` 후보로만 남길 것 — 저장된 ciphertext 의 복호화 계약이고 설치키 회전 경로가 없다(2026-09-29 백서 과제에서 확인됨). 이번에 실제 호출처를 세어 봤다(총 6곳): `Seal` 은 `admin.go:394`(OIDC client_secret)·`admin.go:511`(AI API 키)·`defense.go:1622`·`realmguard.go:1538`(둘 다 `server_proof` 컬럼에 저장), `Open` 은 `auth.go:203`(OIDC client_secret)·`admin.go:456`(AI API 키). 즉 **읽기 경로가 있는 것은 OIDC·AI 두 비밀뿐**이고 `server_proof` 두 곳은 쓰기 전용이다(제품 코드에 Open 호출이 없음을 grep 으로 확인) — 포맷/AAD 를 바꾸면 그 두 비밀이 기존 설치에서 복호화되지 않는다.
  - `internal/api/auth.go`·`admin.go`(`putOIDCSetting`/`putAISetting`)·`apikeys.go` 는 건드리지 말 것. 보호 경로(auth/session/migrations/workflows)를 전혀 지나지 않는 과제이므로 그대로 두면 된다.
  - 1비트 변조 케이스는 **nonce 가 아닌 ciphertext/태그 바이트**를 뒤집을 것. 디코드한 `data` 의 마지막 바이트(GCM 태그)를 `^= 1` 하는 것이 가장 단순하고 결정론적이다. nonce 를 뒤집어도 실패하지만 "변조 감지" 가 아니라 "다른 nonce" 를 검증하는 것이 되어 의도가 흐려진다.
  - 거부 테스트는 에러가 났다는 것만 단정하고 **에러 문구 전체를 문자열 비교하지 말 것**(`unsupported ciphertext` / `invalid ciphertext` / `decrypt secret: %w`). 문구는 계약이 아니고 `%w` 쪽은 crypto 패키지 메시지에 의존한다. 구분이 필요하면 접두사 수준에서만 볼 것.
  - 거부 케이스가 "에러만" 단정하면 함수를 항상 에러 반환으로 바꿔도 통과한다 — 반드시 같은 테스트 안에 성공 왕복 단정을 함께 둘 것(기준 2 의 마지막 문장).
  - base64 깨짐 케이스: `RawURLEncoding` 이라 패딩 `=` 과 `+`/`/` 가 모두 거부된다. 다만 **길이가 맞는 임의 영숫자 문자열은 유효한 base64 로 디코드될 수 있다** — 디코드 실패를 노리는 입력은 `!` 같이 알파벳 밖 문자를 반드시 포함시킬 것.
- 차선 후보: `New` 와 `Open` 만으로는 모자라면 — **페이지된 목록이 유일하지 않은 컬럼으로만 정렬해 OFFSET 경계에서 행이 건너뛰거나 겹칠 수 있음**(`internal/api/admin.go:552`·`:727`, `extended.go:581`, `catalog.go:130` 가 tiebreak 없음; 랭킹은 이미 `score DESC,id`). 이쪽을 고르면 **먼저 동일 `created_at` 행 여러 개로 실제 PG17 Red 를 만들 것** — 못 만들면 버그가 아니라 견고성 과제로 격하하고 한 목록(`listUsers` 하나)만 건드려 파일 수를 2개 안쪽으로 유지할 것. `serviceLocation` 건은 고르지 말 것(프로덕션 6파일 + `playAllowed` 와 계약 갈라짐, 여섯 회차 연속 보류).
