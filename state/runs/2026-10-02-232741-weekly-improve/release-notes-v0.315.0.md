# v0.315.0 — IPv6 로 접속한 사용자가 로그인하고, 그 로그인 실패가 속도 제한에 세어집니다

## 주소를 잘못 읽어 세 가지가 조용히 어긋났습니다

`remoteHost`(`internal/app/auth.go`)는 접속자의 주소를 `strings.Split(r.RemoteAddr, ":")[0]` 으로 읽었습니다. net/http 가 넘기는 IPv6 주소는 `"[::1]:54321"` 형태라, 첫 조각은 주소가 아니라 **`"["`** 였습니다.

이 값을 받는 곳은 셋이고 모두 `inet` 컬럼입니다 — `audit_logs.ip_address`, `user_sessions.ip_address`, `login_attempts.ip_address`. `"["` 는 `inet` 으로 변환되지 않으므로 세 쓰기가 모두 실패하는데, **세 실패가 각각 다르게 조용했습니다**:

- **비밀번호가 맞으면 로그인이 안 됐습니다.** `issueSession` 의 INSERT 가 변환에서 실패해 **`500 SESSION_ERROR`** 를 돌려줬습니다. 자격 증명은 옳은데 세션만 만들어지지 않습니다.
- **비밀번호가 틀리면 그 실패가 세어지지 않았습니다.** `recordLoginFailure` 의 행이 `login_attempts` 에 들어가지 않아 **`auth.max_login_attempts` 가 영원히 도달하지 않았습니다.** 그 설정이 막으려는 바로 그 공격 — 비밀번호 추측 — 이 IPv6 경로에서는 횟수 제한 없이 가능했습니다.
- **감사 기록은 로그만 남기고 버려졌습니다.**

바이너리는 `Addr: ":8080"`(`cmd/weekly`)으로 붙고 Linux 에서 그것은 IPv6 연결도 받습니다. 그래서 이것은 특이한 설정이 아니라 **듀얼 스택 망의 기본 동작**에서 그대로 재현됩니다.

## 주소로 읽히는 것만 주소로 씁니다

- `net.SplitHostPort` 로 포트를 떼고 `net.ParseIP` 로 읽습니다. 주소로 읽히지 않으면 **`""`** 를 돌려줍니다 — 주소가 아닌 문자열을 `inet` 컬럼에 건네는 것은 주소가 없는 것보다 나쁘기 때문입니다(쓰기 자체가 실패합니다).
- `netip.ParseAddr` 대신 `net.ParseIP` 를 씁니다. 영역이 붙은 `fe80::1%eth0` 는 `inet` 컬럼이 받지 않는데 `netip` 은 그것을 주소로 넘겨주기 때문입니다.
- `X-Forwarded-For` 는 **그것이 주소일 때만** 이기고, 아니면 연결 자체의 주소가 남습니다. 이전에는 헤더 값을 그대로 썼으므로, 누구나 보낼 수 있는 헤더에 쓰레기값을 넣으면 `login_attempts` 쓰기가 실패해 **자기 실패 횟수를 지울 수 있었습니다.** 같은 결함의 두 번째 입구를 함께 막았습니다. 프록시가 헤더를 정상적으로 덮어쓰는 배치에서는 동작이 달라지지 않습니다.
- `ip.String()` 로 정규화하므로 IPv4-mapped 주소(`::ffff:192.0.2.1`)는 `192.0.2.1` 로 기록됩니다. 쓰기와 조회(`host(ip_address) = $1`)가 같은 함수를 지나므로 서로 일치합니다.
- 응답 코드·문구·필드와 OpenAPI 는 바꾸지 않았고, 프런트엔드는 한 줄도 건드리지 않았습니다.

## 검증

구현 단계에서 새 시험 3개(`internal/app/clientaddress_test.go`)를 **실제 IPv6 리스너 위에서** 돌렸습니다 — `net.Listen("tcp", "[::1]:0")` 에 프로덕션 `app.Handler()` 를 올리고 실제 HTTP 클라이언트로 POST 합니다. `RemoteAddr` 을 손으로 채워 넣지 않고 net/http 가 채우게 한 것이 요점입니다. 고치기 전 코드에서 셋 모두 실패하는 것을 먼저 확인했습니다:

```
clientaddress_test.go:118: a correct password from ::1 answered 500 {"success":false,…,"error":{"code":"SESSION_ERROR",…}}
clientaddress_test.go:104: the rate limiter counts 0 failures from ::1, so the limit never arrives there
```

위조 `X-Forwarded-For` 시험도 `left 0 failures on record` 로 함께 실패했습니다. 고친 뒤 셋 모두 통과하며, **고친 뒤 옛 한 줄을 되돌려 같은 실패가 다시 나는 것까지** 확인했습니다. 비평 단계에서는 `main` 을 별도 작업 트리로 꺼내 새 시험 파일만 얹어 같은 실패를 독립적으로 재현했습니다.

릴리즈 단계에서도 실제 PostgreSQL(pgvector/pgvector:pg16) 위에서 `go test ./... -count=1` 전체를 통과했습니다. `go build ./...`·`go vet ./...`·`gofmt`, 버전·OpenAPI·쪽넘김·모달 검사, 프런트엔드 시험과 타입 검사, `scripts/build.sh` 를 통과했습니다. 로드맵 HTML·PDF 와 두 안내서 HTML 을 기존 스크립트로 다시 만들었습니다.

검증하지 않은 것을 밝힙니다. 새 시험은 **DB 와 IPv6 루프백이 둘 다 있어야** 돌고, 없으면 `t.Skipf` 로 건너뛰므로 그런 환경에서는 통과처럼 보입니다 — 이번 회차에서는 둘 다 있는 환경에서 실제로 실행됐습니다. `parseClientHost` 의 대괄호 벗기기는 브래킷이 붙은 `X-Forwarded-For` 를 가정한 것이며 **실제 프록시 출력으로는 확인하지 않았습니다.** 이미 저장된 행과 새 정규화 표기의 비교도 확인 범위 밖입니다. `gh` 를 사용할 수 없는 세션이므로 원격 이력을 보는 advisory `release-check.sh` 는 실행하지 않았습니다.

## 업그레이드

마이그레이션·새 환경변수·설정 변경은 없습니다. IPv4 로만 접속하는 배치에서는 달라지는 것이 없고, 프록시 뒤에서 `X-Forwarded-For` 를 정상적으로 받는 배치에서도 기록되는 주소는 이전과 같습니다. 기존과 같이 PostgreSQL 을 백업한 뒤 v0.315.0 이미지를 올립니다.

이 릴리즈는 **지난 기록을 고치지 않습니다.** 결함이 있던 동안 버려진 감사 기록과 `login_attempts` 행은 되돌아오지 않습니다. 따라서 IPv6 경로로 진행 중이던 비밀번호 추측은 이 업그레이드 시점부터 0 에서 세어집니다 — 그 기간의 접근을 조사해야 한다면 애플리케이션 로그를 보십시오.

배포 담당자는 배포 후 ① IPv6 주소로 접속한 사용자가 맞는 비밀번호로 로그인되는지, ② 그 로그인이 `user_sessions`·`audit_logs` 에 **읽을 수 있는 주소**로 남는지, ③ 틀린 비밀번호를 `auth.max_login_attempts` 만큼 반복했을 때 실제로 차단에 도달하는지를 확인합니다. ①이 실패하거나 주소 컬럼이 비어 들어오면 확대를 중단하고 v0.314.0 이미지로 되돌립니다 — 되돌리면 IPv6 로그인 불가와 속도 제한 무력화도 함께 돌아옵니다. 배포 다음 날까지 `SESSION_ERROR` 문의와 로그인 관련 문의를 확인하며, **IPv6 사용자의 `SESSION_ERROR` 0건**을 확인 기준으로 삼습니다.
