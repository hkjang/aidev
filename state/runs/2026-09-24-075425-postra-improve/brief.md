# 과제서 (정찰, 2026-09-24, base main@c2adc61 / v0.23.4)

- 과제: IMAP 리터럴 길이를 안전하게 파싱하고, 선언된 크기를 미리 할당하지 않기 (가치 4 / 위험 2 / 작업량 S)

## 왜
`internal/adapters/imap/client.go:194` 의 `n, _ := strconv.Atoi(m[1])` 는 오류를 버린다. Go 문서상 범위를 넘는 입력에서 `Atoi` 는 `ErrRange` 와 함께 **해당 타입의 최대값**을 돌려주므로, 상한이 꺼진 계정(`refusalThreshold()==0`, 즉 `sync.max_message_bytes <= 0`)에서는 서버가 `{99999999999999999999}` 를 선언하는 것만으로 `client.go:206` 의 `make([]byte, n)` 가 `MaxInt` 크기 할당을 시도한다(패닉 또는 거대 할당). 계정 호스트는 사용자가 정하므로 이 입력은 신뢰할 수 없는 서버에서 온다 — 같은 프로세스를 공유하는 다른 계정의 수집까지 같이 죽거나(패닉은 `sync.go:79/143/347` 의 `recover` 가 잡아 잡 실패로 끝나지만) 메모리를 삼킨다.
더구나 지금은 **실제로 보낸 바이트가 아니라 선언한 바이트**로 버퍼를 먼저 잡기 때문에, 상한이 켜져 있어도 "선언만 크게 하고 보내지 않는" 서버가 할당을 증폭시킬 수 있다. 길이를 64비트로 제대로 읽고, 파싱 실패는 오프셋을 알 수 없으므로 세션을 폐기하며, 본문은 실제 도착한 만큼만 버퍼에 담게 하면 이 증폭 경로가 사라진다.

## 수용 기준
1. `maxBytes = 0`(무제한)으로 연 실제 TCP 세션에서 서버가 `{99999999999999999999}` 같은 int64 범위 밖 길이를 선언하면, 어댑터가 패닉이나 기가바이트급 할당 없이 `errUnframed` 를 감싼 오류를 돌려주고, 같은 세션의 이후 명령도 모두 `errUnframed` 로 실패한다(`s.abandon` 경로 재사용).
2. 선언 크기 선할당 제거: 정상 리터럴 본문이 바이트 단위로 동일하게 돌아오고(기존 `TestIMAPEnumerateAndFetch` 무수정 통과), "n 을 선언하고 그보다 적게 보낸 뒤 끊는" 서버에 대해 선언값 크기의 버퍼를 미리 잡지 않는다.
3. 기존 거부·드레인·세션 폐기 동작 불변: `TestIMAPRejectsOversizeLiteral`, `TestIMAPRefusedLiteralKeepsStreamFramed`, `TestIMAPRefusedLiteralResyncsAtDefaultLimit`, `TestResyncBudgetExceedsRefusalThreshold`, `TestIMAPUndrainableLiteralAbandonsSession` 이 모두 무수정으로 통과.
4. 변이 검증: 길이 파싱을 `strconv.Atoi` 로 되돌리면 새 테스트가(그리고 그 테스트만) 실패함을 실행으로 확인한 뒤 원복.

## 건드릴 파일
- `internal/adapters/imap/client.go:exec` (189-218행 루프) — `strconv.Atoi` 대신 `strconv.ParseInt(m[1], 10, 64)`. 오류(범위/구문)면 몇 바이트가 뒤따르는지 알 수 없으므로 드레인하지 말고 `s.abandon(fmt.Errorf("%w: ...", errUnframed, ...))` 로 즉시 폐기. 성공하면 기존 `refusalThreshold()` 비교는 `int64` 끼리 그대로.
- `internal/adapters/imap/client.go:exec` 수용 분기 — `make([]byte, n)` + `io.ReadFull` 를 `bytes.Buffer` + `io.CopyN(&b, s.r, n)` 로. 선할당이 필요하면 `b.Grow` 를 작은 상한(예: `drainChunk`)으로만. `s.literals = append(s.literals, b.String())` 로 기존 계약(문자열 슬라이스) 유지.
- (선택) 같은 분기에서 `discardLiteral` 처럼 `drainChunk` 마다 `s.deadline()` 갱신. 느린 서버가 명령 타임아웃에 걸리지 않게 되는 개선이지만, 기존 타임아웃 테스트가 있으면 그 기대를 깨지 말 것.
- `internal/adapters/imap/client_test.go` — 기존 `oversizeServer`/`refusedLiteralServer` 옆에 "int64 범위 밖 길이를 선언하는 서버" 헬퍼를 추가하고 `dialMax(t, addr, 0)` 으로 연결하는 테스트 1~2개. 기존 헬퍼 시그니처는 바꾸지 말고 새로 추가할 것.

## 검증 명령
```
go test -race -count=3 ./internal/adapters/imap/     # 기준선 1.095s (이번 정찰에서 실행, 통과)
go build ./... && go vet ./...
go test -race -count=1 ./...
make lint                                            # gofmt -l + gosec v2.28.0 (Issues 0 유지)
go run ./cmd/postra-contracts -check
git diff --check
```
프런트엔드 미변경이면 `npm` 단계와 `internal/transport/spa/assets` 재생성은 불필요(`git status --porcelain internal/transport/spa/assets` 가 비어 있는지만 확인).

## 위험과 피할 것
- **설정 키를 새로 만들지 말 것.** `sync.max_message_bytes` 의 기본값(50MiB, `config.go:278`)·`literalMargin`·`resyncDrainFactor`·`resyncBudget` 상수는 건드리지 않는다. 이번 과제는 "길이를 어떻게 읽고 어떻게 담는가" 만이다.
- `sync.go` 의 실패 집계·조기 종료는 범위 밖(보류 아이디어로 남아 있음). `internal/application` 은 손대지 않는다.
- 보호 경로(auth/session/OIDC/migrations/SecretStore/workflows/spa assets) 는 이번 변경과 무관하므로 전혀 열지 말 것.
- 과거 교훈: **실제 동작이 바뀌지 않는 수정은 넣지 말 것.** 새 테스트가 수정 전 코드에서 확실히 실패하는지(수용 기준 4) 먼저 확인하고 나서 구현을 확정할 것.
- 과거 교훈: 손으로 만든 대역이 아니라 **실제 TCP 서버 + 실제 `Dialer`** 로 증명할 것. 이 패키지에는 이미 그 패턴(`fakeServer`/`oversizeServer`/`dialMax`)이 있으니 그대로 따를 것.
- 함정: `oversizeServer` 는 4GiB 를 선언만 하고 보내지 않는다. 새 테스트 서버도 선언 뒤 바이트를 보내지 않는다면, 구현이 드레인을 시도하는 순간 명령 타임아웃(기본 60초)까지 매달린다 — 파싱 실패는 드레인하지 말고 즉시 폐기해야 하는 이유다.
- 미확인: `make([]byte, math.MaxInt)` 가 이 환경에서 패닉인지 OOM 인지 **직접 실행으로 확인하지 못했다**(샌드박스가 스크래치 실행을 거부). 구현자는 수정 전 상태에서 한 번 재현해 어느 쪽인지 기록할 것. 32비트 빌드(`int` 32비트)에서의 추가 오차도 미확인.

## 추정 근거 (basis of estimate)
분해: 파싱 교체 ~15분 / 스트리밍 버퍼 교체 ~15분 / 새 서버 헬퍼 + 테스트 2개 ~25분 / 변이 검증 ~10분 / 전체 검증 명령 ~15분. 가정: 기존 테스트 헬퍼를 재사용하고 `client.go` 의 나머지 구조는 그대로 둔다, `go test -race ./...` 가 캐시를 상당 부분 재사용한다. 범위: **60~80분**(10회 중 8회). 선택 사항인 데드라인 갱신을 포함하면 상단. 한 세션(45분) 을 넘길 수 있는 쪽이므로, 시간이 부족하면 수용 기준 1·3·4(파싱)만 먼저 끝내고 기준 2(스트리밍)는 다음 회차로 넘길 것 — 두 변경은 서로 독립이다.

## 차선 후보
**README govulncheck 예시를 CI 고정 버전(v1.6.0)에 정렬** (가치 2 / 위험 1 / 작업량 S) — `README.md:303` 의 `@latest` 와 `.github/workflows/ci.yml` 의 `@v1.6.0` 불일치. 문서만 수정, 코드·워크플로 무변경. 1순위가 수정 전 재현에 실패하면(= 관찰 가능한 결함이 없으면) 이쪽으로 전환할 것.
