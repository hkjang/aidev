# 회차 노트 2026-09-27-023231-relio-improve — relio
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 02:32] base pinned — main@39fe52c
- [러너 02:32] autonomy low-risk — 롤백 PR 

## 정찰 노트
- 골랐다: pgx 연결 실패(`*pgconn.ConnectError`)가 REST 는 400+DSN 조각, MCP 는 `return message` 로 원문을 내보내는 것을 둘 다 500/일반 문장으로. 지난 회차 SQLSTATE 분류(18c83f2)가 남긴 구멍이고 같은 자리·같은 어휘라 위험이 낮으며, 두 문을 함께 고치는 것이 운영자 규칙(한쪽만 고치지 말 것)에 맞다. 감사 q/metadata·기간 필터·Cursor 후보는 실 DB 또는 사용자 계약 결정이 먼저라 제쳤고, 차선으로 둔 AST 대조 테스트는 가치 2 라 뒤로 미뤘다.
- 추측으로 적은 것: 실제 오류의 동적 타입이 `*pgconn.ConnectError` 라는 것 — 소스(`pgconn.go:154/160`, `pgxpool/pool.go:237-240`)로만 확인했고 실제 오류 객체는 못 봤다. 과제서에 "첫 테스트에서 `%T` 로 찍어 확인" 을 수용 기준 앞에 넣었다.
- 구현자가 조심할 것: 새 case 를 `isPg` **뒤**에 둘 것(ConnectError 가 PgError 28P01 을 감싸는 인증 실패는 로그에 sqlstate 를 남겨야 함). `pgErrorVerdict`·`sanitizeToolError` 의 SQLSTATE→문장 표는 한 글자도 건드리지 말 것. `internal/mcp` → `internal/platform/database` 새 import 는 순환이 없으나 `go build ./...` 로 확인할 것.
- 프로필은 3일 전 것이 지금 코드와 어긋나지 않아 새로 쓰지 않았다(migrations 마지막 017, 빌드·테스트 명령 동일 재확인). base main@39fe52c, 작업 트리 깨끗.
- 이번 회차는 테스트를 돌리지 않았다(예산). 과제서의 검증 명령은 지난 회차들이 실제로 돌린 것들이다.
- [러너 02:40] scout done — PostgreSQL 에 **닿지 못한** 오류(`failed to connect to \`user=… database=…\``)를 REST·MCP 두 문 모두에서 원문 없이 �

## 구현 노트
- `*pgconn.ConnectError`(PostgreSQL 에 닿지 못함, SQLSTATE 없음)가 REST 에서 400+DSN 조각, MCP 에서 `return message` 로 원문이 나가던 것을 새 `database.Unreachable(err)` 하나로 두 문 모두 일반 서버 오류로 접었다. 프로덕션 3파일, SQLSTATE→문장 표 두 개는 무변경.
- 정찰이 미확인으로 남긴 동적 타입은 probe 로 확인했다 — 정확히 `*pgconn.ConnectError`, `errors.As` 참, 사슬에 `PgError` 없음. 문자열 폴백은 쓰지 않았다.
- **확신 없는 곳**: (1) SASL 인증 실패(ConnectError 가 PgError 28P01 을 감싸는 경우)에서 `case isPg` 가 먼저 잡아 로그에 sqlstate 가 남는다는 것은 switch 순서로 구조적으로 보장되지만 **실제 PostgreSQL 로 재현하지 않았다**(28P01 도 pgErrorVerdict 기본값 → 500 이므로 응답은 어느 쪽이든 같고 새는 것도 없다). (2) 이름 해석 실패·dial timeout 은 connection refused 로만 재현했다 — 셋 다 같은 `&ConnectError{}` 생성자를 지나므로 같은 타입이지만 앞의 두 경로를 직접 실행하진 않았다.
- **검증 못 한 것**: 실제 PostgreSQL 컨테이너로 요청 경로 end-to-end(지난 두 회차가 하던 것). 이번 결함은 순수 오류 분류이고 두 테스트가 실제 풀·실제 함수를 지나므로 컨테이너 없이 재현됐다. 프런트·마이그레이션 무변경이라 web 빌드는 돌리지 않았다.
- **일부러 하지 않은 것**: `context.DeadlineExceeded`·`net.Error`(연결은 됐지만 질의가 끊긴 경우)는 여전히 400+원문이다 — 클라이언트 취소와 서버측 한도 초과를 구분할지 결정이 필요해 ideas.json 에 남겼다. 영어 substring 분류를 센티널로 바꾸는 작업, `*pgconn.ParseConfigError`(기동 경로 전용)도 범위 밖.
- **다음 역할이 조심할 것**: 두 새 테스트는 DB 가 필요 없다 — 127.0.0.1 의 임의 포트를 잡았다 닫고 그 주소로 질의해 refused 를 받는다. 다만 **localhost 로의 TCP 연결을 막는 샌드박스에서는 오류 타입이 달라져 깨질 수 있다**(이 환경에서는 `go test`·`go test -race` 모두 통과). `internal/mcp/results_test.go` 의 `captureDefaultLogger` 는 `slog.SetDefault` 를 건드리므로 이 패키지에 `t.Parallel()` 테스트를 추가하지 말 것.
- [러너 02:48] brief accepted — 채택 — 과제서가 지목한 자리와 근거(`server.go:714` 의 case 순서, `results.go:98` 의 `return message`, `pgconn/errors.go` 가 싣는 DSN 조�
- [러너 02:48] verify passed — 검증 9개 통과 (auto)

## 비평 노트
- **approve, risk low, 차단 없음.** red/green 을 직접 재현했다 — `server.go:743` 의 case 를 `false &&` 로 끄면 400 + 본문에 `failed to connect to \`user=relio database=relio\`: 127.0.0.1:33215`, `results.go:79` 에서 `|| database.Unreachable(err)` 를 지우면 모델 문장에 같은 원문. 두 테스트는 실제로 바뀐 분기를 지나고 증상이 커밋 메시지와 일치한다. 복원 후 build·vet·`gofmt -l`·`go test ./...`(실패 0)·`-race`(mcp/server/database) 통과.
- 구현 노트의 '확신 없는 곳' 둘을 소스로 닫았다: `pgconn/pgconn.go:158/164/171` 이 ConnectError 생성 지점 전부라 DNS·dial timeout·AfterConnect 도 같은 타입이고, perDialConnectError·errors.Join 이 모두 Unwrap 을 주므로 28P01 은 `isPg` 가 먼저 잡는다(양쪽 다 500). 멱등성 주장도 `server.go:694` 의 `StatusCode < 500` 로 확인.
- **보안·법무 소견 없음.** `pgconn/errors.go:67-75` 가 `User`·`Database` 만 찍고 **비밀번호는 안 싣는다** — 원문을 로그로 옮기는 것이 비밀값을 새로 새게 하지 않는다. 신규 엔드포인트·인가·의존성·암호 비교 없음, 개인정보 신규 처리 없음, 노출은 오히려 줄어든다.
- **못 본 것**: 실제 PostgreSQL 컨테이너 end-to-end, 프런트(무변경). ADMIN_GUIDE 미변경은 형제 커밋 18c83f2 선례와 일치해 결함으로 보지 않았다.
- **남는 우려(다음 회차·릴리즈 노트)**: 연결된 **뒤** 끊긴 경우(`context.DeadlineExceeded`·`*net.OpError` `write tcp 10.0.0.2:5432: …`)는 여전히 400+원문으로 내부 host:port 를 실을 수 있다. 또 두 테스트는 루프백 TCP 가 refused 로 돌아와야 하므로 블랙홀 샌드박스 CI 에서 무관한 이유로 깨진다.
- [러너 02:54] review approved — 리뷰 승인 (risk=low)
- [러너 02:54] pr created — https://github.com/hkjang/relio/pull/35
- [러너 02:58] ci passed — 검사 2개 모두 success
- [러너 02:58] merge done — f5064f6
- [러너 02:58] release skipped — 자율화 단계 low-risk — 릴리즈는 사람이
