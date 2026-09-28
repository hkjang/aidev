# 회차 노트 2026-09-28-174206-postra-improve — postra
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 17:42] base pinned — main@66a8915
- [러너 17:42] autonomy release — 

## 정찰 노트
- 1순위였던 POP3 `retrBody` 상한(가치 4)은 이번에도 뺐다 — 미병합 04b15be/PR #22 소유이고 `gh` 가 샌드박스에 막혀 3회 연속 반려 여부를 확인하지 못했다. 대신 아직 아무도 건드리지 않은 STLS/STARTTLS 업그레이드 경계를 골랐다: 두 인바운드 어댑터가 업그레이드 후 리더를 새로 만들어 주입된 평문을 조용히 버리므로(RFC 의 MUST discard 는 만족) 주입 시도가 어디에도 남지 않는다. 프로덕션 파일 2개로 끝난다.
- 추측으로 적은 것: `textproto.Conn` 의 임베드된 `Reader.R *bufio.Reader` 접근 — `go doc`/GOROOT 조회가 샌드박스에 막혀 소스로 확인하지 못했다. 컴파일로 먼저 확인하고, 막히면 과제서의 대안(Dial 에서 bufio 핸들 직접 보관)으로 갈 것. imap/client_test.go 의 기존 헬퍼 이름도 프로필에서 옮겨 적은 것이라 미확인.
- 구현자가 조심할 것: 오탐이 곧 메일 수집 중단이다 — 정상 업그레이드 경로 테스트를 반드시 같이 낼 것. SMTP 는 net/smtp 라 같은 검사를 못 넣으니 "SMTP 도 고쳤다" 고 쓰지 말 것. 탐지는 버퍼 도달분만 보는 best-effort 이므로 주석·릴리즈 노트에서 과장하지 말 것.
- 제친 후보: 문서 단독(max_message_bytes 규약·govulncheck 핀)은 6~7회 연속 차선으로만 남은 저가치 항목이라 또 미뤘고, sync 루프 조기 종료는 도메인 치명-오류 신호가 없어 위험 3 이라 뺐다(다만 이번에 sync.go:274 가 전건 실패에도 JobSucceeded 를 보고한다는 사용자 가시 오보를 확인해 가치를 3 으로 올려 두었다).
- 프로필은 1일 전 것이고 이번에 읽은 코드(pop3 client.go, imap Dial, sync.go, app.go dialInbound)와 어긋나지 않아 새로 쓰지 않았다.
- [러너 17:48] scout done — STLS/STARTTLS 핸드셰이크 직전에 서버가 미리 보낸 평문 바이트를 감지해 업그레이드를 거부 (가치 3 / 위험 2

## 구현 노트
- POP3 `Dial` 의 STLS 블록과 IMAP `Dial` 의 STARTTLS 블록에, 업그레이드 승인 응답과 `tls.Client` 사이에서 리더 버퍼 잔량(`s.text.R.Buffered()` / `s.r.Buffered()`)을 보고 0 이 아니면 `conn.Close()` 후 dial 을 실패시키는 검사를 넣었다. TLS 는 클라이언트 우선이므로 그 바이트는 규약 위반(STARTTLS command injection)이고, 이전에는 리더 교체와 함께 조용히 버려져 dial 이 그냥 성공했다.
- 확신 없는 곳: (1) 주입 탐지는 버퍼 도달분만 보는 best-effort 다 — 서버가 승인 응답과 주입을 **다른 TCP 세그먼트로 늦게** 보내면 검사를 통과한다. 주석에 명시했고 테스트 픽스처는 한 번의 `io.WriteString` 으로 같이 보내 결정적으로 만들었다. 릴리즈 노트에 "STARTTLS 주입을 막는다" 로 쓰지 말 것 — "핸드셰이크 전 서버 바이트를 감지해 거부한다" 가 정확하다. (2) 실서버 중 승인 응답 뒤 핸드셰이크 전에 배너/CAPA 를 미리 보내는 비표준 구현이 있으면 이 검사가 오탐이 된다 — 그런 서버를 확인하지는 못했다(TLS 규약상 불가능한 동작이라 판단).
- 일부러 하지 않은 것: SMTP(`internal/adapters/smtp/client.go:67` 은 `net/smtp` 의 `c.StartTLS` 라 리더 버퍼에 접근할 수 없다 — 범위 밖, 고치지 않았다). `retrBody`·`readList`·`maxListBytes`·IMAP 프레이밍 상수·데드라인 정책·설정 키·문서·워크플로도 건드리지 않았다.
- 다음 역할이 조심할 것: 새 테스트 4개는 외부 의존이 없다(로컬 `net.Listen` + 즉석 생성 자가서명 인증서). 픽스처만 `InsecureSkipVerify: true` 를 쓰고 프로덕션 기본값은 그대로다. `selfSigned` 는 pop3·imap 테스트 패키지에 각각 복사됐다(smtp 에서 이식, 패키지 경계 때문). 정상 업그레이드 테스트 2개가 오탐 방지 장치이므로 지우지 말 것.
- 미실행: 외부 PostgreSQL(POSTRA_TEST_PG)·브라우저/PDF·프런트 빌드·govulncheck. 프런트는 미변경이라 `git status --porcelain internal/transport/spa/assets` 가 비어 있다.
- [러너 17:59] brief accepted — 채택 — 근거(두 `Dial` 이 업그레이드 후 리더를 새로 만들어 주입을 삼킨다, 검사 위치, `s.text.R`/`s.r` 핸들, smtp `selfSigned` �
- [러너 18:00] verify passed — 검증 9개 통과 (auto)

## 비평 노트
- 판정 approve(risk low). 원장에 `- 실패 재현:` 줄이 없어 직접 재현했다: 프로덕션 2파일만 main 으로 되돌리면 새 테스트 2개가 "Dial accepted a server that injected plaintext..." 로 FAIL 하고, 복원 후 통과한다 — 테스트가 정말 바뀐 경로를 지난다. 워크트리는 clean 으로 되돌렸다.
- 검사 위치를 소비 경로로 확인했다: imap `exec` 는 태그 완료 줄에서 곧바로 return 하고 pop3 `readResponse` 는 한 줄만 읽으므로 `Buffered()>0` 는 서버가 핸드셰이크 전에 보낸 바이트만 뜻한다 — 클라이언트 자신의 선반입에서 오는 오탐 경로는 코드에서 찾지 못했다. 오탐 결과도 확인: AuthError 가 아니므로 계정 비활성화(sync.go:184) 를 타지 않고 JobFailed + 진단으로 끝난다. 에러 문구에 주입 내용·비밀값 없음.
- 실행: gofmt, vet, `go build ./...`, `go test -race ./internal/adapters/...` 전부 통과, 새 테스트 `-count=30` 플레이크 없음. 못 본 것: 전체 `go test ./...`, 외부 PG, 브라우저/PDF, 프런트, govulncheck, gosec(네트워크 — smtp 에 동일 패턴 선례 있고 gosec 기본값은 _test.go 미스캔).
- 승인이어도 남는 우려: (1) 비규격 서버용 예외 경로가 없어 그런 서버를 만나면 그 계정 수집이 영구 실패하고 끌 방법이 없다(규약 위반 서버라 수용 가능하나 첫 사용자 보고가 이 모양일 수 있다); (2) 릴리즈 노트는 "주입을 막는다" 가 아니라 "핸드셰이크 전 서버 바이트를 감지해 거부한다" 로 쓸 것 — 탐지는 버퍼 도달분만 보는 best-effort 이고 SMTP(net/smtp) 는 이번에 보호되지 않았다.
- 스타일만(차단 아님): `selfSigned` 3중 복제, imap `dialStartTLS` 의 에러 무시, 주석의 CVE-2011-0411 은 Postfix/SMTP 건이라 약간 어긋남.
- [러너 18:05] review approved — 리뷰 승인 (risk=low)
- [러너 18:05] pr created — https://github.com/hkjang/postra/pull/28
- [러너 18:11] ci passed — 검사 10개 모두 success
- [러너 18:11] merge done — 33f8950
