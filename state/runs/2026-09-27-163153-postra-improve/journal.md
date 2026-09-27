# 회차 노트 2026-09-27-163153-postra-improve — postra
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 16:31] base pinned — main@89aecc9
- [러너 16:31] autonomy release — 

## 정찰 노트
- POP3 `readList`(client.go:130) 를 골랐다: 루프 **안** 의 `deadline()`(:133) 때문에 IMAP v0.23.7 과 똑같이 "데드라인이 끝내 주지 못하는" 무한 루프이고, 열거 단계(sync.go:198/201/208)라 한 계정 동기화가 통째로 멈춘다. IMAP 쪽 같은 결함은 이미 다 닫혀서 5회 연속 IMAP 회차를 피할 수 있었고, 프로덕션 파일은 1개다.
- 제친 후보: POP3 본문 MaxMessageBytes(가치 4지만 미병합 04b15be·PR #22 반려 여부 미확인 — 운영자 금지 항목), 문서 2건(가치 2, 차선으로 남김), IMAP Atoi(재현 불가).
- 확신 없는 곳(추측): (1) 권장 상한 값 32 MiB 는 실측이 아니라 RFC 1939 의 UID 최대 70자 계산(≈97바이트/메시지)에서 나온 추정이다 — 구현자가 수용 기준 3 처럼 실측하고 그 위에서 정할 것. 8 MiB 를 그대로 베끼면 대형 maildrop 이 깨진다. (2) PR #22 의 사람 반려 여부는 이번에도 확인 못 했다(GitHub API 미호출) = 미확인.
- 구현자가 조심할 것: 빈 줄은 `textproto.ReadLine` 이 CRLF 를 떼어 `len(line)==0` 이므로 바이트만 세면 상한을 우회한다 — IMAP 이 이 정확한 결함으로 비평을 받고 `responseLineOverhead`(5145007) 를 도입했다. 그리고 `retrBody`·`s.deadline()` 위치·`sync.go` 는 건드리지 말 것.
- 실행해 본 것: `go test -race -count=1 ./internal/adapters/pop3/ ./internal/adapters/imap/` → pop3 `[no test files]`, imap ok 4.460s. 전체·프런트·외부 PG·gosec 은 미실행.
- [러너 16:37] scout done — POP3 다중행 응답(LIST/UIDL) 누적에 상한 — `.` 종결을 보내지 않는 서버가 수집 워커를 영원히 붙잡고 메모�

## 구현 노트
- `pop3/client.go` 의 `readList` 에 누적 상한을 넣었다(fb1f2e6, 프로덕션 파일 1개): 줄마다 `len(line)+listLineOverhead`(2+16) 를 `maxListBytes`(32 MiB) 에 charge 하고 초과 시 `s.conn.Close()` + "pop3 multi-line response exceeds %d bytes". 수정 전엔 루프 **안** `deadline()` 때문에 90초(=60초 명령 데드라인 초과)까지 매달리는 것을 실제 TCP 로 재현했다.
- 32 MiB 는 정찰의 추정(97바이트/메시지)이 아니라 실측으로 정했다 — 신규 테스트가 최대 길이(70자) UID 5000건 = charged 463,893 바이트, **메시지당 92 바이트**, 상한이 364,722건 수용임을 `t.Logf` 로 남기고 8배 여유 + 30만 건 수용을 단언한다.
- 확신 없는 곳: (1) 32 MiB 가 "충분히 큰" 상한이라는 판단은 RFC 1939 최대 UID 길이 기준 메시지 수(≈364k)에 근거한 것이고, 그보다 큰 실제 maildrop 을 본 적은 없다 — 값 자체는 정책 선택이다. (2) `s.conn.Close()` 만으로 세션을 폐기했고 IMAP 처럼 `broken` 센티넬은 두지 않았다: 이후 명령은 닫힌 소켓 오류로 실패하며(테스트로 `List`·`Retrieve` 둘 다 확인) 오류 메시지가 "use of closed network connection" 이라 상한이 원인임을 드러내지 않는다. (3) 외부 PostgreSQL·브라우저/PDF·프런트엔드 검사는 실행하지 않았다(프런트 미변경, assets 드리프트 0).
- 일부러 안 한 것: `retrBody()`(미병합 04b15be 소유, PR #22 반려 여부 미확인이라 운영자 금지 항목), `s.deadline()` 위치 이동(큰 maildrop 회귀), `sync.go`/`internal/application`, 설정 키, 단일 행 길이 상한(deadline 이 앞에서 걸려 시간은 유한 — 별도 아이디어로 남겼다).
- 관찰만 하고 남긴 것: `sync.go:208` 의 `List` 가 상한에 걸리면 `lerr` 가 조용히 무시되고 세션은 이미 닫혀 있어 이후 `Retrieve` 전건이 Failed 로 집계된다. 기존 `sync.go` 설계이므로 이번에 고치지 않았다(ideas.json "세션 폐기 뒤 sync 루프 조기 종료").
- 다음 역할이 조심할 것: `internal/adapters/pop3/client_test.go` 는 이 패키지의 첫 테스트 파일이고 127.0.0.1 루프백 리스너만 쓴다(DB·외부 서비스 불필요). 상한 검사를 지우면 endless 테스트 2개가 **타임아웃으로** 실패하므로 `-timeout` 없이 돌리면 10분을 기다린다 — 변이 검증 때는 `-timeout 60s` 를 붙일 것.
- [러너 16:50] brief accepted — 채택 — 근거(줄 개수 상한 없음, 루프 **안** 의 `deadline()`, 열거 단계라 더 나쁜 위치, 빈 줄 우회, 픽스처 이식처)가 모두 
- [러너 16:51] verify passed — 검증 9개 통과 (auto)

## 비평 노트
- 판정 approve(risk low, blocking 없음). 읽기만 한 게 아니라 **변이로 검증**했다: `charged`+상한 블록을 지우면 endless 테스트 2개가 `-timeout 60s` 에서 타임아웃 실패, `listLineOverhead=0` 으로 바꾸면 빈 줄 테스트만 매달린다(22바이트 줄 테스트는 통과) — 테스트가 상한과 빈 줄 우회를 각각 실제로 붙잡고 있다. 무변이 실행은 -race 3개 통과 3.18s.
- 직접 실행: `go build ./...`, `go vet`, `gofmt -l ./cmd ./internal`(clean), `go test -count=1 ./internal/...`(23개 패키지 전부 ok, application 35.7s). **미실행: gosec/make lint-security, govulncheck, web/ 툴체인, 외부 PostgreSQL, 브라우저·PDF.** 프런트 미변경이라 assets 드리프트는 없다.
- 구현자의 자기 의심 3건을 확인했다: (a) 32 MiB 는 t.Logf 실측(92 B/메시지, 364,722건) 위에서 고른 값이고 주석 산술도 맞다 — 정책 선택으로 수용; (b) `broken` 센티넬 부재로 후속 명령이 "use of closed network connection" 만 남기는 것은 사실이나 진단 품질 문제이지 결함은 아님(notes); (c) sync.go 소비자 경로를 직접 읽어 **데이터 유실 없음**을 확인했다(DeleteAfterFetch 부재, sync.go:31). UIDL 상한 → List 폴백 실패 → JobFailed/providerListFailed 로 깨끗이 끝난다.
- 승인이어도 남는 우려: 이 상한은 **시간이 아니라 바이트** 상한이다. 루프 안 `deadline()` 이 그대로라 60초 미만 간격으로 한 줄씩 흘리는 서버는 1.86M 줄을 채울 때까지 붙잡을 수 있다 — 커밋 제목의 "cannot hold the session for ever" 는 과장이다(IMAP v0.23.7 과 동일한 성질이라 거절 사유는 아님). **릴리즈 노트는 타임아웃 수정이 아니라 "한 열거 응답의 바이트 상한" 으로 쓰고, `retrBody()` 는 여전히 무제한임을 명시할 것.** 단일 행 상한(IMAP `maxLineBytes` 대응물)도 POP3 에는 아직 없어서 `maxListBytes` 주석의 "caps how much readList may accumulate" 는 실제보다 넓게 말한다 — 다음 회차 아이디어.
- [러너 16:57] review approved — 리뷰 승인 (risk=low)
- [러너 16:57] pr created — https://github.com/hkjang/postra/pull/27
- [러너 17:03] ci passed — 검사 10개 모두 success
- [러너 17:03] merge done — fb1f2e6
- [러너 17:13] release published — v0.23.8
- [러너 17:15] assets verified — v0.23.8 자산 5개 (이전 v0.23.7: 5)
