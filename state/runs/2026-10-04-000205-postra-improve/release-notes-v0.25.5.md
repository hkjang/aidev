# Postra v0.25.5 — 크기 선별이 꺼졌다는 사실을 남깁니다

POP3 동기화는 중복 판정 기준으로 `UIDL` 을 쓰고, `LIST` 는 **메일 크기를 알아내려고만** 부릅니다. 그런데 그 `LIST` 의 오류가 통째로 버려지고 있었습니다. `UIDL` 은 답하면서 `LIST` 는 거부하는 서버에서는 모든 메일 크기가 `0` 으로 남아 **내려받기 전 크기 선별이 아무 흔적 없이 무력화**됐고, `sync.max_message_bytes` 를 걸어 둔 운영자에게는 제한이 그냥 무시되는 것처럼 보였습니다. 이번 릴리즈는 그 사실을 경고 한 줄로 드러냅니다. **동작은 한 줄도 바뀌지 않습니다.** 설정 키, 환경변수, DB 마이그레이션, 화면 변경도 **없습니다**.

**`LIST` 에 정상 응답하는 서버(대부분)는 아무 변화가 없습니다.** 로그 한 줄이 새로 나타나는 것은 `UIDL` 성공 + `LIST` 실패라는 조합에서만입니다.

## 무엇이 조용했나

- `runSync` 는 `UIDL` 로 체크포인트를 잡은 뒤 `if listed, lerr := sess.List(ctx); lerr == nil { ... }` 로 크기를 병합합니다. `else` 가 없어 `lerr` 은 그대로 버려졌습니다.
- 크기가 전부 `0` 이면 내려받기 전 선별 조건(`maxBytes > 0 && rm.Size > maxBytes`)은 **절대 참이 되지 않습니다.** 상한을 넘는 메일도 전부 받은 다음 `ingestOne` 이 사후에 거부합니다.
- 결과는 **조용한 성능·대역 손실**입니다. 데이터가 틀어지지는 않습니다(사후 거부가 받아내므로). 그러나 상한을 넘는 메일을 받지 않으려고 설정한 운영자는 그 설정이 왜 듣지 않는지 알 방법이 없었습니다 — 작업은 `succeeded` 로 끝나고, 어디에도 이유가 없었습니다.

## 고친 내용

- 버려지던 그 자리에 `slog.Warn` 한 줄을 넣었습니다: `sync: LIST failed; message sizes unknown and the pre-fetch size screen is off for this sync`. 계정 ID, `providerDiagnostic(lerr)` 의 고정 문장, 그리고 `domain.ClassifyInbound(lerr)` 의 분류 레이블을 함께 남깁니다.
- 새 관용구가 아닙니다. 같은 파일의 `syncSentFolder` 가 "동기화를 실패시키지 않는 실패" 에 이미 쓰고 있는 `slog.Warn` + `providerDiagnostic` 짝을, 빠져 있던 그 한 자리에 적용했습니다.
- `providerDiagnostic` 은 서버 원문을 **절대 섞지 않는** 고정 문장 집합입니다. 그래서 분류 정보도 함께 사라지므로, 레이블만 반환하는 `domain.ClassifyInbound` 를 `"class"` 키로 덧붙였습니다. POP3 의 `-ERR` 응답은 `"class":"rejected"` 로 나옵니다.
- **동작은 그대로 둡니다.** `LIST` 거부는 메일을 다 받아낸 동기화를 실패시킬 이유가 아닙니다. 작업은 여전히 성공으로 끝나고, 사후 oversize 정책·`rawReadLimit`·`maxBytes > 0` 규약은 건드리지 않았습니다. `domain.SyncDiagnostic` 에도 필드를 더하지 않았으므로 API 계약과 화면은 그대로입니다.
- 프로덕션 변경은 `internal/application/sync.go` **한 파일, 2줄**(근거 주석 5줄 별도)입니다.

## 검증

- 되돌려 실패를 먼저 확인했습니다: `slog.Warn` 호출을 `_ = lerr` 로 돌리면 `LIST_refused` 하위 테스트만 실패하고(`log mentions "pre-fetch size screen" = false, want true`, 로그는 빈 상태) `LIST_answered` 는 통과합니다. 같은 실행에서 `status=succeeded`·메일 2건 저장 단언은 모두 통과했으므로, 결함은 "수집이 안 된다" 가 아니라 정확히 "흔적이 없다" 입니다.
- 테스트는 손으로 만든 대역을 쓰지 않습니다. 루프백 TCP 메일드롭 + **프로덕션 `pop3.Dialer`** + 실제 `CreateAccount` → `StartSync` → `Search` 로 양쪽 모양을 모두 단언합니다. `LIST` 거부: 경고 1줄 + `"class":"rejected"` + 메일 2건 저장 + `status=succeeded`. `LIST` 정상: **그 경고 없음** + 동일하게 2건 저장 + 성공. 양쪽 모두 로그에 서버 원문(`LIST not available`)이 **없음**을 단언합니다.
- 경고는 `StartSync` 의 비동기 워커에서 나오므로 로그 포착에는 mutex 로 감싼 writer 를 썼습니다(`bytes.Buffer` 직접 사용은 `-race` 에서 경합이 됩니다).
- `go test -race -count=1 -timeout=900s ./...` 25개 패키지 실패 0건
- `go build ./...`, `go vet ./...`, `gofmt` 무지적, `go mod tidy` 무변경, canonical API 계약 검사(`postra-contracts -check`)
- `gosec -severity medium` — 141개 파일 지적 0건
- `npm ci && npm run build` 후 `internal/transport/spa/assets` 변화 없음 — 태그에 담긴 `/app` 번들이 최신입니다(릴리즈 워크플로가 같은 검사를 다시 합니다)

## 업그레이드 및 호환성 주의사항

DB 마이그레이션, 새 설정 키, 새 필수 환경변수가 없습니다. 실행 파일 또는 컨테이너 이미지를 교체한 뒤 재시작하면 끝입니다. 되돌릴 때도 v0.25.4 로 그대로 내려갈 수 있습니다.

- 바뀌는 것은 **로그 한 줄**뿐입니다. 수집 결과, 저장된 메일, 작업 상태, API 응답, 화면은 영향이 없습니다.
- 로그를 자동 수집하고 있다면 `WARN` 레벨에 `sync: LIST failed; ...` 가 새로 나타날 수 있습니다. **경고가 보이는 계정은 크기 상한이 사실상 걸리지 않고 있다**는 뜻이므로, 그 서버에 대해서는 상한을 넘는 메일도 일단 내려받은 뒤 거부된다는 점을 감안하세요.
- 이 경고에는 서버가 보낸 원문이 들어가지 않습니다. 기존 `providerDiagnostic` 의 고정 문장과 `class` 레이블만 남습니다.
- v0.25.4 의 크기 제한 없는 계정 본문 잘림 수정, v0.25.3 의 POP3 `fetch` 진단 교정, v0.25.2 의 OpenTelemetry v1.45.0, v0.25.1 의 "계정당 인바운드 연결 하나"와 `mail.idle_enabled`, v0.25.0 의 동기화 실패 진단은 바뀌지 않았습니다.

## 배포 파일

- `postra-0.25.5-linux-amd64`: 정적 단일 실행 파일
- `postra-0.25.5.tar.gz`: 폐쇄망 `docker load` 이미지, `postra:0.25.5` (linux/amd64)
- `postra-0.25.5-sbom.cdx.json`: Go CycloneDX SBOM
- `postra-0.25.5-frontend-sbom.cdx.json`: 프런트엔드 CycloneDX SBOM
- `SHA256SUMS.txt`: 배포 파일 SHA-256
