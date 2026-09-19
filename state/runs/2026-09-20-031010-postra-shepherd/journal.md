# PR 처리기 노트 2026-09-20-031010-postra-shepherd — postra PR #16
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-09-19-230358-postra-improve)
# 회차 노트 2026-09-19-230358-postra-improve — postra
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 23:04] base pinned — main@49edf02
- [러너 23:04] autonomy release — 

## 정찰 노트
- 선택 이유: main(49edf02) 에 notifymail·handoff 가 여전히 없어(grep 0건) 그쪽 보류 항목 6개는 손댈 수 없고, 남은 독립 후보 중 smtp 어댑터 테스트 공백(`[no test files]` 직접 확인)이 가치가 가장 높고 `client.go` 를 안 건드려 릴리즈 위험이 0 에 가깝다. gofmt CI 검사는 이번 회차에 로컬 gofmt 상태를 확인하지 못해(권한 제한) 차선으로 뒀다.
- 확신 없는 곳: `domain.SecretHandle.Zero` 의 파일명·정확한 `Security` 상수명은 grep 으로 확인하라고 적었을 뿐 열어 보지 않았다. net/smtp 의 PlainAuth 가 127.0.0.1 평문에서 허용된다는 것과 LOGIN 챌린지가 base64 디코드돼 `Next` 로 온다는 것은 표준 라이브러리 지식이며 이 저장소에서 실행해 보지 않았다.
- 구현자 주의: 대역이 아니라 실제 `net/smtp.Client` ↔ TCP 스크립트 서버로 증명할 것(운영자 규칙). 테스트가 실패하면 어댑터를 고치지 말고 시나리오를 빼고 결함을 노트에 적을 것. 실제 STARTTLS 핸드셰이크는 선택 사항.
- [러너 23:07] scout done — smtp 어댑터 단위/통합 테스트 추가 — net.Listen 스크립트 서버로 AUTH·오류 분류·STARTTLS 미광고·auto 무인증 

## 구현 노트
- `internal/adapters/smtp/client_test.go` 신규(테스트 파일만, `client.go` 무수정). `fakeRelay` 스크립트 릴레이(127.0.0.1 net.Listen, 명령 줄을 TLS 레그 여부와 함께 수집, `waitConn` 으로 서버 goroutine 종료 뒤 단언) 위에서 실제 `Client{}.Send`/`TestConnection` 이 실제 net/smtp 로 왕복 — 과제서 a~i 전부 + 실제 STARTTLS/implicit TLS(자체서명 ecdsa 헬퍼 `selfSigned`) + auth=none·MAIL 4xx·DATA 끊김·연결 거부 진단·classify 표.
- 증명: 기존 코드 덮개라 첫 실행에 전부 초록이었으므로 client.go 변이 5개(4xx/5xx 반전·auto 폴백 제거·AuthError 임시화·STARTTLS 검사 제거·Zero 제거)로 각각 해당 테스트만 red 인 것을 확인하고 원복(`git status` 에 테스트 파일만 남음). `-race -count=5` 통과, 전체 `go build && go vet && go test -race ./...` exit 0, `gofmt -l` 빈 출력.
- 확신 없는 곳: (1) `recorded` 의 순서 단언은 서버가 응답을 쓰기 전에 mutex 로 기록하고 클라이언트가 그 응답을 받은 뒤 `waitConn` 까지 기다리므로 안전하다고 보지만, CI 의 느린 러너에서 5초 conn deadline 이 짧을 가능성은 낮아도 0 은 아님(로컬은 전체 1.2s). (2) `TestTestConnection/connection_refused` 는 닫은 포트를 다른 프로세스가 그 사이 잡으면 이론상 오탐 — 실무상 무시할 수준.
- 일부러 하지 않은 것: CRAM-MD5 분기와 "no supported AUTH mechanism" 분기(인트라넷에서 드묾, ideas 에 남김); `Send` 의 ctx 가 MAIL/RCPT/DATA 단계에 전파되지 않는 문제는 client.go 수정이라 범위 밖 — ideas 에 실패 테스트 먼저 만들라고 적음. 어댑터 결함은 발견 못 함.
- 다음 역할 주의: 이 테스트는 DB·네트워크 외부 의존이 없고 127.0.0.1 TCP 만 씀. `selfSigned` 는 매 호출 P-256 키를 만들며 ~50ms. 릴리즈 노트에는 "테스트만 추가, 동작 변화 없음" 으로 적으면 됨.
- [러너 23:12] brief accepted — 채택 — 근거(smtp 테스트 전무·client.go 무수정·127.0.0.1 Host·LOGIN base64 챌린지)가 모두 코드와 맞았고 선택 사항이던 실제 T
- [러너 23:13] verify passed — 검증 9개 통과 (auto)

## 비평 노트
- 확인: diff 는 `internal/adapters/smtp/client_test.go` 신규 1파일뿐(client.go 무수정, 범위 이탈 없음). `gofmt -l` 빈 출력, `go vet` 통과, `go test -race -count=3` 1.1s 초록. 구현자가 의심한 두 곳은 실제 결함 아님 — `SecretHandle` 이 포인터라 Zero 단언은 유효, 순서 단언은 서버가 응답 쓰기 전 mutex 기록 + `waitConn` 이라 안전.
- 직접 변이 2개(classify 4xx/5xx 반전 → TestSendClassifiesReplies·TestClassify red, STARTTLS 미광고 검사 제거 → TestSendSTARTTLS red)로 테스트가 대상을 실제로 고정함을 재확인하고 원복(`git status` clean).
- 못 본 것: CRAM-MD5·"no supported AUTH mechanism" 분기는 여전히 미덮개(구현자가 의도적으로 제외). CI 느린 러너에서의 5초 데드라인은 로컬로만 판단.
- 보안·법무: 루프백 전용, 인증서·키는 런타임 생성(커밋된 키 재료 없음), 자격 증명은 더미 문자열 — 차단 사유 없음.
- 사소한 관찰(비차단): `serve` 의 defer 순서상 `finished` 신호가 `conn.Close()` 보다 먼저 나가나 기록은 이미 끝난 뒤라 무해. 릴리즈 노트엔 "테스트만 추가, 동작 변화 없음" 으로 적으면 됨.
- [러너 23:15] review approved — 리뷰 승인 (risk=low)
- [러너 23:15] pr created — https://github.com/hkjang/postra/pull/16
- [러너 23:20] ci failed — 성공이 아닌 검사: gosec (medium+)=failure

## 수리 노트
- 지적(gosec 실패)은 사실이나 이 변경의 결함이 아님: 로컬 재현 결과 2건 모두 base main@49edf02 의 `mcp_oauth_proxy.go:47`(G101)·`oauth_proxy_http.go:381`(G705) 이고 PR diff 는 `client_test.go` 1파일(gosec 은 테스트 파일 미스캔).
- main 은 이미 3f9eb61 로 두 줄에 `#nosec` 을 달아 고쳤음 — 그 두 파일만 바꿔 재스캔하면 Issues 0·exit 0 확인 후 원복(트리 clean).
- 이 PR 의 테스트 파일은 PR #17(bf4e5a9, 이미 main) 과 바이트 동일 → PR #16 은 #17 에 완전히 대체됨. 커밋 없이 종료, 닫기 권고(무관 파일 수정·빈 PR 생성 회피).
- 확신 없는 곳: 없음. 다만 `gosec@v2.28.0` 을 로컬에 새로 설치했더니 `-version` 이 dev 로 찍혀 CI 와 같은 바이너리인지는 결과(동일 2건)로만 판단.
