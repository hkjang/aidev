# 회차 노트 2026-09-26-233213-postra-improve — postra
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 23:32] base pinned — main@06bf78c
- [러너 23:32] autonomy release — 

## 정찰 노트
- IMAP `exec` 의 응답 누적 상한을 골랐다. 지난 세 회차(리터럴 길이 → 리터럴 스트리밍 → 줄 길이)가 닫은 계열의 마지막 구멍이고, 같은 파일 :375-377 주석이 이 어댑터의 무제한 버퍼링을 실제 OOM(파드 재시작) 원인으로 기록해 두었다. 가치 4였던 POP3 상한은 제쳤다 — 구현이 미병합 브랜치 auto/2026-09-21-1520 에 있는데 PR #22 가 **사람에게 반려된 것인지 확인하지 못했고**(샌드박스가 GitHub API curl 을 거부), 확인 없이 재제출하면 운영자 규칙에 정면으로 걸린다.
- 추측으로 적은 것: (1) 상수 제안값 8 MiB·리터럴 64개는 실측이 아니라 `enumerateBatch`=2000 과 정상 FETCH 줄 길이 어림에서 나온 값이다 — 구현자가 정상 경로 테스트를 먼저 쓰고 조정할 것. (2) "수정 전 매달림" 은 코드 읽기(`readLine` 이 줄마다 `s.deadline()` 갱신, exec 루프에 개수 상한 없음)로만 판단했고 실제 TCP 로 재현하지는 않았다 — **미확인**. 재현되지 않으면 차선(문서)으로 갈 것.
- 조심할 것: 리터럴 **바이트 총량**으로 묶으면 `refusalThreshold()==0`(무제한) 계정의 정상 대용량 본문이 깨진다. 개수로 묶어야 한다. `readLine`/`readLineNoReset` 의 데드라인 정책 차이는 의도이므로 합치지 말 것.
- 이번 회차에 실제로 돌린 것: `go test -race -count=1 ./internal/adapters/imap/` → ok 1.731s(테스트 12개). 그 외 전체·프런트·PG·브라우저는 미실행. 코드는 바꾸지 않았다.
- [러너 23:36] scout done — IMAP `exec` 의 응답 누적에 상한을 두어, 태그 완료를 보내지 않는 서버가 메모리를 무제한으로 키우지 못하

## 구현 노트
- 무엇/왜: `exec`(client.go) 가 한 응답의 프로토콜 텍스트(untagged 줄 + 리터럴 `cont`)를 `maxResponseBytes`(8 MiB), 리터럴 **선언 횟수** 를 `maxResponseLiterals`(64) 로 묶고 초과 시 기존 `abandon(errUnframed)`. 태그 완료를 안 보내는 서버가 `readLine` 의 데드라인 갱신 덕에 영원히 돌며 버퍼를 키우던 구멍을 닫았다. 프로덕션 파일 1개(client.go), 커밋 3af8341.
- 확신 없는 곳: (1) 8 MiB 는 *이 저장소의* 정상 최대 응답(enumerateBatch 2000줄 = 실측 81,994 바이트)에서 100배 여유로 잡은 값이다. FETCH 줄에 긴 ENVELOPE/BODYSTRUCTURE 가 실리는 서버를 실물로 본 적은 없다 — 여유가 100배라 안전하다고 보지만 실서버 표본은 없다. (2) 리터럴 상한 64 에 걸릴 수 있는 유일한 후보는 `ListMailboxes`(`LIST "" "*"`, 폴더명을 리터럴로 주는 서버)다. exec 호출 9곳을 전수 확인한 결과 나머지는 `Retrieve`/`Top` 1개·`ensureIndex`/`SELECT`/`STORE`/`FETCH FLAGS` 0개다. 다만 `LIST` 는 지금도 리터럴 폴더명을 읽지 못한다 — exec 가 리터럴 바이트를 `s.literals` 로 빼고 `line` 을 접두부+`cont` 로 재조립하므로 이름이 줄로 돌아오지 않아 `reList` 가 매칭에 실패한다(이번 변경 이전부터 그렇다). 그래서 64 가 새로 깨뜨리는 정상 경로는 없다고 본다. 리터럴 폴더명을 주는 실서버로 검증하지는 못했다(**미확인**). 비평가는 여기를 먼저 볼 것.
- 일부러 안 한 것: 리터럴 **바이트 총량** 은 세지 않았다 — `refusalThreshold()==0`(sync.max_message_bytes<=0) 계정의 정상 대용량 본문이 깨진다. 이 규약을 `TestIMAPLargeBodyWithoutLimitIsNotCapped`(12 MiB > 8 MiB 상한) 로 고정했다. `Idle`/`readLineNoReset` 은 줄을 누적하지 않아 같은 결함이 아니므로 범위 밖.
- 다음 역할 주의: 새 테스트 4개는 DB 불필요, 127.0.0.1 `net.Listen` + 실제 `Dialer` 만 쓴다. 무한 서버 2종은 상한이 없으면 **매달린다** — 상한 검사를 건드린 채 `-timeout` 없이 돌리지 말 것. 미실행: 프런트(npm), 외부 PostgreSQL(POSTRA_TEST_PG), 브라우저/PDF, govulncheck.
- [러너 23:47] brief accepted — 채택 — 근거(줄 개수 상한 없음, `readLine` 의 데드라인 갱신, `enumerateBatch` 는 요청만 묶음, `abandon` 재사용, 리터럴은 바이�
- [러너 23:48] verify passed — 검증 9개 통과 (auto)

## 비평 노트
- 확인: diff 전부(client.go 41줄·테스트 277줄), exec 루프와 readBoundedLine/readLiteral/discardLiteral/abandon, 새 테스트 4개가 새 코드의 에러 문자열을 단언하므로 main 에서는 통과 불가. 실행: gofmt -l 무출력, go vet 통과, `go test -race -count=1 -timeout 240s ./internal/adapters/imap/` ok 5.035s.
- 거절 사유(수리가 먼저 볼 파일: internal/adapters/imap/client.go:253, 같은 방식으로 :301): `len(line)` 은 CRLF 를 떼고 센 값이라 **빈 줄이 0 바이트**다. 빈 줄만 끝없이 보내는 서버에 대해 상한이 절대 걸리지 않고 exec 가 영원히 돌았다 — 임시 스크래치 테스트(실제 Dialer + 127.0.0.1, 실행 후 삭제, 트리 clean)로 70초(=commandTO 60초 초과) 지점에서 HeapAlloc 0.8 MB → 13.97 GB 확인. 커밋이 닫았다고 말하는 바로 그 구멍이다. `+ 2`(또는 줄 개수 상한)로 고칠 수 있다.
- 부수: 줄당 16바이트 string 헤더를 세지 않아 1바이트 줄이면 8 MiB 상한에서 실제 상주 ~134 MiB. 같은 수정으로 함께 닫아야 한다.
- 승인 아니지만 남는 우려: `refusalThreshold()==0` 계정은 여전히 64 × 무제한 바이트(개선이지 상한이 아님) — 릴리즈 노트에 한 줄. docs/releases 에 이번 변경 항목 없음(v0.23.6.md 은 줄 길이 상한만).
- 미확인: 전체 go test, 프런트, POSTRA_TEST_PG, 브라우저/PDF, gosec·govulncheck. 보안/법무 관점 차단 사유는 없다(인증·인가·비밀값·주입·개인정보 표면 무변경).
- [러너 23:56] review rejected — 리뷰 거절: internal/adapters/imap/client.go:253 `respBytes += int64(len(line))` counts the line *after* readBoundedLine has stripped CRLF, so a blank line contributes 0 bytes. A server that answer

## 수리 노트
- 지적은 **맞았다**. 빈 줄만 흘리는 서버용 테스트(`TestIMAPEndlessBlankLinesAbandonSession`, 실제 Dialer + 127.0.0.1)를 먼저 넣어 수정 전 재현했다: 75초 `-timeout` 패닉이 `exec`→`readLine` 루프에서 잡혔고, 60초 commandTO 를 지나서도 계속 돌았다. 부수 지적(줄당 16바이트 string 헤더 미계상)도 맞다.
- 고친 방법: 상수 `responseLineOverhead = 2 + 16`(서버가 실제로 보낸 CRLF + `untagged` 의 string 헤더)을 두 누적 지점(`line`, `cont`)에 함께 부과 — 두 지적을 한 수정으로 닫는다. 이제 빈 줄도 18바이트를 물고, 최악 상주량이 대략 상한(8 MiB)에 묶인다. 정상 경로 가드도 실제 부과량을 재도록 고쳤다(2000줄 = 117,994 바이트, 상한의 1/71). 상한 값·리터럴 개수 64·리터럴 바이트 미계상 규약은 손대지 않았다.
- 검증: `go test -race -count=1 ./internal/adapters/imap/` 13건 통과(새 테스트 0.37초 중단), 전체 `go test -race ./...` 통과, `go build`/`go vet`/`make lint`(gosec Issues 0) 통과. 미실행: 프런트(npm), POSTRA_TEST_PG, 브라우저/PDF, govulncheck.
- 확신 없는 곳: `cont` 에 16바이트 헤더분까지 부과한 것은 약간의 과계상이다(`cont` 는 `line` 에 접합되어 따로 보관되지 않음). 최대 64줄 × 18바이트라 상한 대비 무시할 수준이라 판단해 상수를 하나로 뒀다.
- 릴리즈 역할에게: 비평가가 요청한 "`refusalThreshold()==0` 계정은 64 × 무제한 바이트(개선이지 상한이 아님)" 한 줄은 **다음 버전 노트**에 넣어야 한다. 이미 출시된 `docs/releases/v0.23.6.md` 에 적으면 거짓이 되므로 이번에 건드리지 않았다.
- [러너 00:02] repair done — # 수리 요약 (수리 시도 1)  - 비평가 지적이 맞았다. `respBytes += int64(len(line))` 는 CRLF 가 떼인 길이를 세므로 빈 줄이 0 바이트였고, 빈 줄만 흘리는 서버에

## 비평 노트
- 확인: diff 전부(client.go 59줄 추가·테스트 312줄, 삭제 0), exec 루프의 두 누적 지점(:271 line, :319 cont)과 리터럴 개수 검사(:295), reLit 이 `$` 앵커인 점, abandon/discardLiteral/readLiteral/refusalThreshold/resyncBudget, exec 호출 9곳, Idle(readLineNoReset 은 줄을 누적하지 않아 같은 결함 아님). 수리가 넣은 `responseLineOverhead = 2 + 16` 이 비평의 빈 줄 우회와 string 헤더 미계상을 한 번에 닫는 것을 코드로 확인했다.
- 실행: gofmt -l 무출력, go vet, go build ./..., `go test -race -count=1 ./...` **전체 통과**, imap 패키지 17개 통과(새 테스트 4개 포함, 상한 가드 로그 = 2000줄 81,994B → 부과 117,994B, 상한 8,388,608B), `make lint` gosec Issues 0. 새 테스트는 main 에 없는 에러 문자열을 단언하고 끝나지 않는 서버를 상대하므로 수정 전 통과 불가.
- 승인이지만 남는 우려(다음 회차 후보): 상한이 **바이트에만** 걸려 시간에는 걸리지 않는다 — 59초 간격으로 빈 줄 하나씩 흘리는 서버는 466,034줄을 소진할 때까지 exec 를 붙잡는다(수개월). IMAP 에는 SMTP 와 달리 ctx AfterFunc(Close) 배선이 없다. 이번 커밋의 주장 범위(메모리)는 정확하므로 차단하지 않는다.
- 릴리즈 역할에게: `refusalThreshold()==0` 계정은 여전히 64 × 무제한 리터럴 바이트(개선이지 상한이 아님) — **다음 버전** 노트에 한 줄. 이미 출시된 v0.23.6.md 는 건드리지 말 것. 부수: 리터럴 폴더명을 쓰는 서버에서 폴더 65개 이상이면 ListMailboxes 가 세션 전체를 abandon 한다(이전 동작도 이미 깨져 있었고 실서버 확인 불가 — 현장 보고 시 LIST 한정 완화).
- 미확인: 프런트(npm), POSTRA_TEST_PG, 브라우저/PDF, govulncheck. 보안·법무 차단 사유 없음(인증·인가·비밀값·주입 표면 무변경, 새 에러는 상수만 담아 본문·서버 텍스트를 로그로 흘리지 않음, 개인정보 무변화, 코드 전용이라 revert 로 완전 복구).
- [러너 00:07] review approved — 리뷰 승인 (risk=low)
- [러너 00:07] pr created — https://github.com/hkjang/postra/pull/26
