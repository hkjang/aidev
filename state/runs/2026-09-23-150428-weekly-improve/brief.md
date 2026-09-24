# 정찰 과제서 (2026-09-23, main@0253313 / v0.306.0)

- **과제: 관리자 메일 발송 기록을 서버에서 상태·종류로 좁혀, 최신 50건 뒤에 묻힌 실패를 찾게 한다 (가치 3 / 위험 1 / 작업량 M)**

- **왜**: `internal/app/mail.go:adminMailDeliveries` 는 14일 창으로만 거른 뒤 `ORDER BY m.created_at DESC, m.id DESC LIMIT 50`(`mailDeliveryLimit = 50`) 을 하고, `frontend/src/pages/AdminPage.tsx:SettingsTab` 은 `useEffect` 에서 `/api/v1/admin/mail/deliveries` 를 인자 없이 한 번만 불러 그대로 표로 그린다 — 그래서 발송이 활발한 곳에서는 최근 50건이 전부 `SENT` 로 차고, 바로 위 현황 문단이 "실패 3건" 이라고 말하는 그 세 건이 표에는 한 줄도 없다. 상태·종류를 **LIMIT 앞에서** 서버가 거르면, 관리자가 카드 안에서 `실패만 보기` 로 그 세 건의 수신자·사유·시각까지 바로 읽는다.

- **수용 기준**:
  1. `GET /api/v1/admin/mail/deliveries?status=FAILED` 가 같은 14일 창 안에서 **최신 `SENT` 50건보다 오래된 `FAILED` 행을 돌려준다** — 즉 필터가 `LIMIT` 앞에 적용된다(프런트에서 50건을 받아 거르는 것은 이 과제가 아니다). `kind=TEAM_REMINDER` 도 같은 방식으로 동작하고, 둘을 함께 주면 AND 로 좁힌다.
  2. 알 수 없는 값(`status=OOPS`, `kind=NOPE`)은 **조용히 무시하지 않고** 400 + 기존 오류 봉투(`writeError`, 예: `INVALID_STATUS` / `INVALID_KIND`)로 거절한다. 인자를 비우거나 아예 주지 않으면 지금과 **완전히 같은 응답**(필터 없음)이다. 권한은 그대로 ADMIN 전용이고 USER 는 403.
  3. 관리자 화면의 `주간보고 메일 발송` 카드 표 위에 상태·종류 선택칸이 생기고, 바꾸면 **서버를 다시 부른다**(클라이언트 필터 금지). 현황 문단(`mailHealth`)의 숫자는 필터와 무관하게 창 전체를 계속 말한다.
  4. 새 시험이 증명해야 할 것: 같은 하네스 DB에 `SENT` 51건 이상 + 그보다 **오래된** `FAILED` 1건을 넣고 → 무필터 응답에는 그 `FAILED` 가 **없고**, `?status=FAILED` 응답에는 **있다**. 이 시험이 고치기 전 코드에서 실패하는 것을 먼저 확인할 것(2026-09-20 회차와 같은 순서). 잘못된 값 400 과 USER 403 도 같은 시험에서 확인한다.

- **건드릴 파일**:
  - `internal/app/mail.go:adminMailDeliveries` — `r.URL.Query()` 에서 `status`·`kind` 를 읽어 **허용 목록**(status: `QUEUED|SENT|FAILED`, kind: `REPORT|TEAM_REMINDER|SCHEDULE_REMINDER`)으로 검증하고, `WHERE m.created_at > …` 뒤에 `AND m.status = $n` / `AND m.kind = $n` 을 `LIMIT` 앞에 붙인다. 문자열 이어붙이기 대신 인자 번호로 넘길 것.
  - `internal/app/mail.go:mailDeliveryList` — 적용된 필터를 응답에 되비출 거면 `status`·`kind` 문자열 필드만 더한다. **`total` 은 더하지 말 것**(아래 위험 참조).
  - `internal/app/mailfilter_test.go` (신규) — 위 수용 기준 4. 기존 `TestTheOperatorSeesEveryQueueTheRelayTouches`(`mail_test.go` 계열)의 fixture 방식(세 큐 표에 직접 INSERT + `newTestServer`)을 그대로 따른다. 제품 함수에 `// guards` 주석 관례를 지킬 것.
  - `docs/openapi.yaml` — `/admin/mail/deliveries` 의 `get` 에 `parameters`(status, kind, enum, 선택) 와 "필터는 서버가 50건을 자르기 **전에** 적용한다" 는 문장.
  - `frontend/src/pages/AdminPage.tsx:SettingsTab` — `mailDeliveries` 를 부르는 `useEffect` 의 의존성에 필터 state 두 개를 넣고 쿼리스트링을 붙인다. 표 위에 `<select>` 둘(라벨: 상태/종류, 기본 `전체`). 기존 `mailDeliveryKindLabels`·`mailDeliveryStatusLabels` 를 재사용.
  - `frontend/src/types.ts` — 필요하면 `MailDeliveryList` 에 필드 추가만. `MailDeliveryKind` 는 이미 있다.
  - `docs/ADMIN_GUIDE.md` 3.7 — 표에서 실패만 골라 보는 법 한 문장. 고쳤으면 `python3 scripts/render-docs.py ADMIN_GUIDE` 로 HTML 만 다시 만든다(다른 문서는 건드리지 말 것).

- **검증 명령** (이 저장소에서 실제로 도는 것):
  - **먼저 DB를 띄울 것.** 이 워크트리에는 `WEEKLY_TEST_POSTGRES_DSN` 이 **이미 설정돼 있는데**(`user=postgres database=weeklytest`, `127.0.0.1:15434`) 그 포트에 아무것도 없어서, 오늘 `go test ./...` 는 **skip 이 아니라 수백 건 FAIL** 한다(`build the harness template: … dial tcp 127.0.0.1:15434: connect: connection refused`). 로컬에 `postgres:16` 이미지가 있다(`pgvector/pgvector:pg16` 은 **없다** — 받거나, 없이 가면 벡터 시험은 건너뛴다). 예: `docker run -d --name weeklypg -p 15434:5432 -e POSTGRES_PASSWORD=<DSN의 비밀번호> -e POSTGRES_DB=weeklytest postgres:16` — DSN 의 정확한 전문·비밀번호는 **미확인**이니 `printenv WEEKLY_TEST_POSTGRES_DSN` 으로 먼저 읽고 맞출 것. DB를 못 띄우면 `env -u WEEKLY_TEST_POSTGRES_DSN go test ./... -count=1` 로 비-DB 경로만 돌고(2.9초), **그 통과는 이 과제의 증거가 아니다**(핵심 시험이 skip 된다) — 반드시 그렇게 적을 것.
  - `gofmt -l internal/app`, `go build ./...`, `go vet ./...`, `go test ./... -count=1` (실제 DB로 약 145~160초)
  - `go test ./internal/app -run <새 시험 이름> -count=1` (고치기 전/후 각각)
  - `python3 scripts/openapi-check.py` (오늘 119개 기준), `python3 scripts/paging-check.py`, `python3 scripts/guard-check.py --changed main`
  - `npm --prefix frontend run lint`, `npm --prefix frontend test`, `npm --prefix frontend run build`
  - 커밋 **뒤에만**: `python3 scripts/mutation-check.py --test <새 시험 이름> --budget 480`

- **위험과 피할 것**:
  - **`total` 을 응답에 더하지 말 것.** `scripts/paging-check.py` 는 `total` 을 받는 화면마다 요청에 `offset` 을 싣거나 허용 목록에 사유를 적으라고 요구하는데, 이 카드에는 쪽 넘김이 없다 — `total` 을 더하는 순간 검사가 깨지고 과제가 L 로 부푼다. 쪽 넘김은 별도 회차.
  - **필터를 `mailDeliveriesUnion` 안이나 `adminMailHealth` 에 넣지 말 것.** 현황 숫자·마지막 사유는 창 전체를 말해야 하고, 둘이 같은 창을 읽는다는 2026-09-20 의 계약을 깨면 "실패 3건" 과 표가 다시 어긋난다. 같은 값을 읽는 두 경로를 한쪽만 넓히지 말라는 운영자 지침이 정확히 이 자리다.
  - **건드리지 말 것**: `internal/app/migrations/`(출하 체크섬), `auth.go`·OIDC·MCP OAuth, `crypto.go`, `.github/workflows/`. 이 과제에 마이그레이션은 **필요 없다**.
  - `scripts/mutation-check.py` 와 `scripts/authz-check.py` 는 **소스를 제자리에서 고쳐 쓴다**. 빌드·시험과 **절대 병행하지 말고**, 커밋 **전에** 돌리지 말 것(변이가 커밋에 들어간 기록이 있다). authz-check 는 40분 이상이라 이 회차에서는 돌리지 않는다.
  - 화면 필터를 만들 때 서버를 다시 부르지 않고 `items` 를 `.filter()` 하면 **과제가 아무것도 고치지 않는다**(50건 안에 없는 행은 여전히 없다). 효과 없는 변경은 반려 사유다.
  - 45분 상한: 착수 25분 안에 백엔드+시험을 첫 커밋으로 넣고, 프런트·문서를 둘째 커밋으로 하는 2026-09-20 의 분할을 그대로 쓸 것.

- **차선 후보**: **README·MCP 도입부의 "읽기 전용" 설명 정정 (가치 2 / 위험 1 / 작업량 S)** — `internal/app/mcpwrite.go`(`mcpMayWrite`, `mcp:write`)가 d5ee67a 로 들어왔는데 `docs/MCP.md:3` 은 아직 "인증된 **읽기 전용** MCP 도구를 제공한다", `README.md:47` 은 "**읽기 전용** Streamable HTTP MCP 서버" 라고 말한다(같은 MCP.md 의 SSO 절 37행은 이미 `mcp:write` 를 정확히 설명하고 있어 한 문서 안에서 서로 어긋난다). 두 도입부 문장과 `docs/MCP.md:11` 의 "키에는 `mcp:read` 범위가 필요하다" 를 소스(`mcpwrite.go`, `mcp.go:205`)에 맞춰 고치는 문서 과제. DB가 끝내 없어서 1순위의 통합 시험을 증명할 수 없을 때만 고를 것.

---

## 추정 근거 (basis of estimate)
- 분해: 백엔드 질의·검증 ~15분 / 시험(fixture 51+1행) ~15분 / openapi·가이드 ~5분 / 프런트 select 2개 + 재조회 ~10분 = 45분. 여기에 DB 컨테이너 확보 5~10분이 **앞에** 붙는다.
- 범위: DB가 이미 살아 있으면 35~50분(열에 여덟), DB를 새로 띄워야 하면 45~60분 — 45분 상한에 아슬아슬하므로 프런트·가이드를 둘째 커밋으로 떼어 낸 것이 컨틴전시다.
- 포함: 서버 필터·시험·openapi·프런트·가이드 한 문장. **제외**: 쪽 넘김(offset), `total`, 실패 재시도 단추, 필터 상태의 URL 영속화.
- 가장 크게 기대는 가정: **PostgreSQL 컨테이너를 띄울 수 있다.** 이것이 틀리면 수용 기준 1·4를 증명할 수 없고, 그때는 차선(문서 과제)으로 간다.
