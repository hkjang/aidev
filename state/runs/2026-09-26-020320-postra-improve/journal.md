# 회차 노트 2026-09-26-020320-postra-improve — postra
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 02:03] base pinned — main@7441f8c
- [러너 02:03] autonomy release — 

## 정찰 노트
- IMAP 줄 길이 상한을 골랐다: 리터럴 경로(v0.23.4~v0.23.5)와 같은 무제한 버퍼링 결함의 마지막 잔여분이고, `readLine`/`readLineNoReset` 두 곳이 raw `ReadString('\n')` 을 쓰는 유일한 지점임을 직접 확인했으며 실제 TCP 테스트 픽스처가 이미 있다. POP3 상한(가치 4)이 더 높아 보였지만 그 구현은 미병합 브랜치 04b15be 에 이미 있어 새로 짜면 충돌한다 — 기각이 아니라 "PR 병합" 과제로 남겼다.
- 미확인/추측: (1) 수정 전 실패를 직접 재현하지 않았다(코드 없는 정찰이라 읽기로만 판단) — 구현자는 착수 전에 개행 없는 서버로 매달림/할당 증가를 먼저 관찰할 것. (2) 상한값 1 MiB 는 "본문은 리터럴로 온다" 는 추론에 기댄 권장치이고 실측 근거는 없다.
- 조심할 것: `readLine` 과 `readLineNoReset` 의 데드라인 정책 차이는 의도다(IDLE 28분 창). 중복처럼 보여도 합치지 말 것.
- 프로필 정정: 이전 프로필의 "POP3 에 MaxMessageBytes 상한 적용됨(PR #21)" 은 틀렸다. `merge-base --is-ancestor` 로 04b15be 가 HEAD 조상이 아님을 확인했고 pop3/client.go 에 상한 코드가 없다. profile.md 를 새로 썼다.
- [러너 02:07] scout done — IMAP 프로토콜 줄 읽기에 길이 상한을 두어, 개행을 보내지 않는 서버가 클라이언트 메모리를 무제한으로 �

## 구현 노트
- 무엇/왜: `readLine`·`readLineNoReset` 이 공통 `readBoundedLine` 을 쓰게 하고, `ReadSlice` 조각 누적이 새 상수 `maxLineBytes`(1 MiB)를 넘으면 기존 `abandon(errUnframed)` 로 세션을 폐기한다(커밋 a7f6c3e). 수정 전 재현: 실제 TCP 서버에서 FETCH 한 번이 60초 명령 데드라인 내내 매달리며 TotalAlloc +596,768,608 바이트, 종료 오류는 errUnframed 가 아닌 날 `i/o timeout`.
- 확신 없는 곳: (1) 1 MiB 값 자체 — 이 경로는 프로토콜 텍스트만 읽는다는 근거로 정했고 실제 대형 IMAP 서버의 최장 응답 줄(거대한 BODYSTRUCTURE/LIST)을 실측하진 못했다. 넘치면 해당 세션이 폐기된다. (2) `ReadSlice` 반환 슬라이스는 다음 읽기가 덮어쓰므로 즉시 `append` 로 복사하는데, 조각 누적 경로는 4 KiB 초과 줄에서만 돌고 그건 새 테스트 2개(1 MiB 정상 줄·무한 줄)로만 커버된다 — 기존 9개 테스트는 짧은 줄이라 이 경로를 타지 않는다.
- 검증 못 한 것: 프런트(미변경, assets 드리프트 0)·외부 PostgreSQL(POSTRA_TEST_PG 없음)·브라우저/PDF·CI 전체. 실제 IMAP 서버(Dovecot/Gmail) 대상 수동 확인 없음.
- 일부러 안 한 것: `readLine`/`readLineNoReset` 통합(데드라인 정책 차이가 의도, IDLE 28분 창) · `literalMargin`/`resyncDrainFactor`/`drainChunk` 변경 · pop3(미병합 브랜치 04b15be 와 같은 파일) · 새 설정 키·문서(상수로 충분) · 차선 후보(README govulncheck)는 1순위가 재현됐으므로 미착수.
- 다음 역할이 조심할 것: `unterminatedLineServer` 는 4 KiB/ms 로 스로틀한 뒤 클라이언트가 끊을 때까지 쓴다 — 상한 검사를 없애면 이 테스트는 실패가 아니라 **매달림**이므로 `-timeout` 없이 돌리지 말 것(변이 검증도 타임아웃으로 드러난다). `TestIMAPLongLineAtBoundIsReturnedWhole` 의 pad 길이는 `maxLineBytes` 에서 CRLF 2바이트까지 빼 경계에 정확히 맞춘 값이라, 상수를 바꾸면 자동으로 따라가지만 경계 의미(정확히 상한 = 통과)를 유지할 것.
- [러너 02:32] brief accepted — 채택 — 근거(두 곳이 raw `ReadString` 을 쓰는 유일한 지점, IDLE 의 28분 창, `abandon` 재사용)가 모두 코드와 맞았고 수정 전 실
- [러너 02:33] verify passed — 검증 9개 통과 (auto)

## 비평 노트
- 확인함: main 의 client.go 로 되돌려 새 테스트를 돌렸고 TestIMAPUnterminatedLineAbandonsSession 이 60.08s 매달린 뒤 i/o timeout 으로 실패 — 수정을 진짜로 검증한다. LongLineAtBound 는 수정 전에도 통과하지만 `>` 를 `>=` 로 바꾸면 깨지는 경계 핀이라 허수 아님. build·vet·gofmt·`go test ./...` 전부 통과, 새 테스트 `-race -count=5` 안정.
- 못 본 것: gosec(바이너리 없음)·프런트(web/ 미변경)·PostgreSQL·브라우저/PDF·실제 Dovecot/Gmail. 승인이지만 이 셋은 "미확인".
- 승인이어도 남는 우려 ①: client.go:228-264 exec 의 리터럴 연속 루프가 조립하는 `line` 은 여전히 무제한이고, 줄마다 readLine 이 60초 데드라인을 갱신하므로 `{0}` 리터럴 + ~1 MiB 연속 줄을 번갈아 보내는 서버는 데드라인 없이 무한히 키울 수 있다. 기존 결함이지만 커밋 제목이 다 닫힌 것처럼 읽힌다 — 다음 회차 1순위 후보.
- 우려 ②: client.go:170-173 주석은 "나머지가 아직 전선에 있다" 고 하지만 LF 로 정상 종료된 과대 줄은 LF 까지 소비된 뒤 걸리므로 그 경우 스트림은 프레임이 맞다. 동작(보수적 폐기)은 안전, 근거 문장만 과장.
- 릴리즈 단계가 알 것: 이 브랜치에 docs/releases 노트와 버전 범프가 없다(v0.23.4·v0.23.5 는 fix + docs(release) 쌍이었다) — v0.23.6 노트를 새로 만들어야 한다. 상한 1 MiB 는 어댑터가 내는 전 명령(SELECT/FETCH/LIST/STORE/EXPUNGE, SEARCH·BODYSTRUCTURE 미사용)으로 안전함을 확인했다.
- [러너 02:40] review approved — 리뷰 승인 (risk=low)
- [러너 02:40] pr created — https://github.com/hkjang/postra/pull/25
- [러너 02:45] ci passed — 검사 10개 모두 success
- [러너 02:45] merge done — a7f6c3e
- [러너 02:56] release published — v0.23.6
- [러너 02:58] assets verified — v0.23.6 자산 5개 (이전 v0.23.5: 5)
