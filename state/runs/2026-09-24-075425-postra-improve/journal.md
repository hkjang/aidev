# 회차 노트 2026-09-24-075425-postra-improve — postra
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 07:54] base pinned — main@c2adc61
- [러너 07:54] autonomy release — 

## 정찰 노트
- 보류 1순위였던 IMAP 리터럴 길이 오버플로를 골랐다. 지난 회차가 같은 파일의 프레이밍을 이미 고쳐 두어 문맥이 살아 있고, 실제 TCP 테스트 픽스처(oversizeServer/dialMax)가 그대로 재사용된다. POP3 계열 두 건은 PR #22 미병합으로 충돌 위험이 있어 제쳤고, workflows·auth 항목은 보호 경로라 계속 후순위다.
- 범위를 넓혀 "선언 크기 선할당 제거"까지 묶었다 — 파싱만 고치면 상한이 꺼진 계정에서 파싱에 성공하는 거대 길이(예 {8000000000})가 그대로 남기 때문이다. 두 변경은 독립이라 시간이 모자라면 파싱만 먼저 끝내도 된다고 과제서에 적었다.
- 추측으로 적은 것: `make([]byte, math.MaxInt)` 가 패닉인지 OOM 인지 직접 실행으로 확인하지 못했다(샌드박스가 스크래치 go run 을 거부). Go 문서상 Atoi 는 범위 초과에서 최대값을 돌려준다는 점까지만 확실하다. 32비트 빌드 영향도 미확인.
- 구현자가 조심할 것: 파싱 실패는 드레인하지 말고 즉시 abandon 할 것(바이트를 안 보내는 서버에 60초 매달린다). 그리고 수정 전 코드에서 새 테스트가 실제로 실패하는지부터 확인할 것 — 효과 없는 변경은 반려 사유다.
- 이번에 실행: `go test -race -count=1 ./internal/adapters/imap/` 통과(1.095s, 기준선). 그 외 전체 검증·프런트·외부 PG 는 미실행.
- [러너 07:59] scout done — IMAP 리터럴 길이를 안전하게 파싱하고, 선언된 크기를 미리 할당하지 않기 (가치 4 / 위험 2 / 작업량 S)

## 구현 노트
- 무엇/왜: IMAP 리터럴 길이를 `strconv.ParseInt(...,10,64)` 로 읽고, 범위/구문 오류는 뒤따르는 바이트 수를 알 수 없으므로 드레인 없이 `abandon(errUnframed)`. 수용 리터럴은 새 `readLiteral` 이 `bytes.Buffer`+`io.CopyN` 로 drainChunk 단위(청크마다 deadline 갱신)로 받아 선언 바이트가 아니라 도착 바이트만큼만 버퍼를 잡는다. 커밋 89d4076.
- 정찰의 미확인 항목 해소: 수정 전 `make([]byte, math.MaxInt64)` 는 이 환경에서 **패닉**(`makeslice: len out of range`)이다 — OOM 아님. 실제 TCP 서버로 재현 확인.
- 확신 없는 곳: (a) 32비트 빌드는 여전히 미검증 — `b.Grow(int(reserve))` 는 reserve<=drainChunk(1MiB) 라 안전하지만 32비트에서 전체 경로를 돌려보지는 않았다. (b) `TestIMAPLiteralBufferTracksDeliveredBytes` 는 `runtime.MemStats.TotalAlloc` 델타(<64MiB, 실측 차이는 1GiB 대 수 KiB)로 판정하므로 프로세스 전역 지표에 의존한다 — 이 패키지 테스트는 `t.Parallel()` 을 쓰지 않아 지금은 안정적이나, 누가 이 패키지에 병렬 테스트를 넣으면 흔들릴 수 있다. (c) 부분 수신(선언보다 적게 보내고 끊김)은 기존과 같이 오류만 반환하고 세션을 abandon 하지 않는다 — 서버가 이미 끊은 상황이라 그대로 뒀다.
- 일부러 안 한 것: 설정 키 신설·상수(literalMargin/resyncDrainFactor/drainChunk) 변경 없음. `sync.go` 의 실패 집계·조기 종료와 `internal/application` 은 범위 밖. SELECT/FETCH 의 나머지 `Atoi` 오류 무시(exists·seq)는 관찰 가능한 실패를 재현하지 못해 손대지 않았다(효과 없는 변경 금지).
- 다음 역할이 조심할 것: 새 테스트는 DB·네트워크 외부 의존이 없고 127.0.0.1 `net.Listen` 만 쓴다. 검증 실행분 — `go test -race -count=3 ./internal/adapters/imap/`(ok 1.178s), `go test -race -count=1 ./...`, `go build`/`go vet ./...`, `make lint`(gosec Issues 0), `postra-contracts -check`, `git diff --check` 전부 통과. 프런트·외부 PostgreSQL·브라우저 검사는 미실행(프런트 미변경, assets 드리프트 0).
- [러너 08:05] brief accepted — 채택 — 근거(Atoi 오류 무시, 상한 꺼진 계정, 선언 크기 선할당)가 모두 코드와 맞았고 실제 TCP 로 수정 전 패닉을 재현했
- [러너 08:05] verify passed — 검증 9개 통과 (auto)

## 비평 노트
- 확인함: main 의 client.go 를 임시로 되돌려 새 테스트 2건이 수정 전에 실제로 깨지는지 직접 실행 — panic(makeslice: len out of range, client.go:206) 과 1,073,814,112 바이트 할당으로 각각 실패, 변경 후 통과. 워크트리는 원상복구(clean). build/vet/gofmt/`-race -count=3` imap/`-race ./...` 전체/contracts -check 모두 통과. 프런트·외부 PG·브라우저·gosec 은 미실행(프런트 미변경).
- 승인. 범위 이탈 없고 커밋 메시지·주석과 동작이 일치하며, 원격 서버가 유발하던 프로세스 panic 을 없애는 순 개선이라 보안·법무 차단 사유 없음. 코드 전용이라 revert 로 완전히 복구된다.
- 남는 우려 1(다음 회차 과제 후보): readLiteral(client.go:266) 이 1MiB 청크마다 deadline 을 갱신해 본문 하나의 읽기 시간이 더 이상 CommandTimeoutSec 하나로 묶이지 않는다. 기본 50MiB 에서 ~51×timeout, sync.max_message_bytes<=0 이면 시간 상한 없음 — discardLiteral 의 resyncBudget 에 해당하는 상한이 없고 IMAP 은 dial 이후 ctx AfterFunc(Close) 가 없어 sync.go:150 세마포어 슬롯을 그동안 점유한다.
- 남는 우려 2: 부분 수신 실패는 abandon 없이 오류만 반환해 s.broken 이 비고 스트림이 미지의 오프셋에 남는다(main 과 동일한 기존 동작, 이번 결함 아님). 그리고 새 테스트는 프로세스 전역 TotalAlloc 델타로 판정하므로 이 패키지에 t.Parallel() 이 들어오면 흔들린다.
- 릴리즈 노트에 적을 것: 원격 IMAP 서버가 선언한 리터럴 길이로 인한 프로세스 중단·대용량 선할당 제거(v0.23.4 리터럴 프레이밍 수정의 후속).
- [러너 08:10] review approved — 리뷰 승인 (risk=low)
- [러너 08:10] pr created — https://github.com/hkjang/postra/pull/24
- [러너 08:16] ci passed — 검사 10개 모두 success
- [러너 08:16] merge done — 89d4076
- [러너 08:24] release published — v0.23.5
- [러너 08:27] assets verified — v0.23.5 자산 5개 (이전 v0.23.4: 5)
