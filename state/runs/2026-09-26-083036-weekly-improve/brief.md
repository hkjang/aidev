- 과제: 관리자 메일 발송 기록 표가 50건에서 잘렸다는 사실을 말한다 (가치 3 / 위험 1 / 작업량 S)
- 왜: `adminMailDeliveries` 는 14일 창의 행을 `LIMIT 50`(`mail.go` 의 `mailDeliveryLimit`) 으로 자르는데 응답에도 화면에도 잘렸다는 표시가 없어, 표가 가득 찬 화면은 "이것이 최근 14일 전부" 로 읽힌다 — 지난 회차가 넣은 종류·상태 필터로 좁힌 뒤에도 그 조건의 행이 50건을 넘으면 같은 오독이 남고, 운영자는 더 좁혀야 한다는 것을 알 길이 없다. 개수가 아니라 `truncated` 불리언 하나를 더하고 화면이 그것을 문장으로 말하면, 잘렸을 때만 잘렸다고 알린다.
- 수용 기준:
  1) 창 안(필터 적용 후)의 행이 50건을 넘으면 응답에 `truncated: true` 이고 `items` 는 정확히 50건이다 — 51번째 행이 `items` 로 새지 않는다.
  2) 50건 이하이면 `truncated` 가 응답에 없다(`omitempty`). 좁힌 결과가 1건인 `?status=FAILED` 응답에는 `truncated` 가 없어야 한다.
  3) 관리자 화면의 발송 기록 표 아래 문장이 `truncated` 일 때만 나타난다.
  4) 새 주장이 **고치기 전 코드에서 실패한다**(무필터 응답에 `truncated` 가 없음). 이것을 먼저 확인하고 나서 고칠 것.
  5) 응답에 총건수(`total`)는 없다 — 아래 위험 참조.
- 건드릴 파일:
  - `internal/app/mail.go:mailDeliveryList` — `Truncated bool \`json:"truncated,omitempty"\`` 를 `Items` 앞에 추가. 필드 이름에 `total`·`count` 를 쓰지 말 것.
  - `internal/app/mail.go:adminMailDeliveries` — 현재 `args = append(args, mailDeliveryLimit)` 로 `LIMIT $N` 을 만든다. 이것을 `mailDeliveryLimit+1` 로 보내고, 스캔 루프 뒤(또는 루프 안에서 `len(list.Items) == mailDeliveryLimit` 일 때 `Truncated=true` 로 두고 break) 51번째 행을 `Items` 에 넣지 말 것. `rows.Err()` 검사는 유지. COUNT 질의를 새로 넣지 말 것 — 한 번의 질의로 끝난다.
  - `internal/app/mail_test.go:TestTheOperatorCanPickTheFailuresOutOfAGoodWeek` (1145행) — **새 시험을 만들지 말고 이것을 늘릴 것.** 이 시험은 이미 창 안에 57건(SENT 55 + FAILED 1 + QUEUED 1)을 넣고 `read("")` 가 정확히 `mailDeliveryLimit` 건임을 주장한다(1203행 부근). 그 자리에 무필터 응답의 `truncated == true` 를, 그리고 이미 있는 `?status=FAILED`(1건) 응답에 `truncated` 가 **없음**을 더하면 양쪽 방향이 한 시험에서 증명된다. `read` 헬퍼가 `data map[string]any` 를 이미 돌려주므로 `data["truncated"]` 로 본다.
  - `frontend/src/types.ts:401` — `MailDeliveryList` 에 `truncated?: boolean` 추가(한 줄 인터페이스).
  - `frontend/src/pages/AdminPage.tsx` — `주간보고 메일 발송` 카드의 발송 기록 `<div className="table-wrap">…</div>` 블록 **바로 뒤**에 `{group.title === '주간보고 메일 발송' && mailDeliveries?.truncated && <p className="setting-help">최근 {mailDeliveryLimit}건만 보여 줍니다 — 종류·상태로 좁혀 보세요.</p>}` 를 붙인다. SettingsTab 의 JSX 는 매우 긴 한 줄이므로 표 블록의 닫는 `}` 위치를 정확히 찾을 것. 50 은 화면에 하드코딩하지 말고 `mailDeliveries.items.length` 를 쓰는 편이 안전하다.
  - `docs/openapi.yaml` — `/admin/mail/deliveries` 응답 `properties`(1714행 부근, `days`·`status`·`kind`·`items` 가 있는 곳)에 `truncated: { type: boolean, description: 창 안의 행이 더 있어 잘렸음. 잘리지 않으면 생략 }`. 설명 문단(1684~1694행)에도 한 줄.
  - `docs/ADMIN_GUIDE.md` 3.7 — 표가 최근 50건까지만 보여 주며 넘으면 화면이 그렇게 말한다는 한 문장. 고쳤으면 `python3 scripts/render-docs.py ADMIN_GUIDE` (ADMIN_GUIDE 만; 다른 문서 재생성 금지).
- 검증 명령:
  - `gofmt -l internal/app` (빈 출력) · `go build ./...` · `go vet ./...`
  - `go test ./internal/app -run TestTheOperatorCanPickTheFailuresOutOfAGoodWeek -count=1` — 고치기 전 실패, 고친 뒤 통과를 둘 다 볼 것 (실제 DB 필요, 아래 함정)
  - `go test ./... -count=1` (실제 DB 로 약 150초)
  - `python3 scripts/openapi-check.py` · `python3 scripts/paging-check.py` · `python3 scripts/guard-check.py --changed main`
  - `npm --prefix frontend run lint && npm --prefix frontend test && npm --prefix frontend run build` (vitest 160개)
- 위험과 피할 것:
  - **`total` 개수를 더하지 말 것.** `scripts/paging-check.py:paged_types`(51~67행) 를 이번에 읽어 확인했다: 그 검사는 TS 의 `interface`/`type` 본문에 `\btotal\s*:` 가 있고 배열 칸이나 `items` 가 있을 때만 그 형을 "잘릴 수 있는 목록" 으로 보고, 그 화면에 `offset` 또는 `ALLOWED` 사유를 요구한다. `truncated?: boolean` 은 그 정규식에 걸리지 않으므로(`truncated` 안에 `total` 이 없다) offset 요구가 생기지 않는다 — 이 과제가 offset 없이 성립하는 근거가 이것이다. 단 `paging-check.py` 를 이번 세션에서 **실행하지는 못했다(권한 거부, 미확인)** — 구현자가 한 번 돌려 확인할 것. 만약 걸리면 응답은 그대로 두고 화면에 고정 문구만 다는 축소판으로 내려갈 것.
  - `mailDeliveriesUnion` 과 `mailHealthDays`(14일 창)는 카드의 숫자와 아래 표가 공유한다 — 창·UNION·집계 질의는 건드리지 말 것. 이 과제는 **자르기만** 다룬다.
  - `mailDeliveryFilter` 의 400 `INVALID_FILTER` 계약과 응답의 `status`·`kind` 되돌림은 그대로 유지. `ORDER BY m.created_at DESC, m.id DESC` 도 바꾸지 말 것(별건의 보류 아이디어다).
  - 보호 경로 금지: `internal/app/auth.go`·OIDC·MCP OAuth, `crypto.go`, `internal/app/migrations/`(출하 체크섬), `.github/workflows/`. 이 과제는 마이그레이션이 필요 없다.
  - 검증 함정: 이 워크트리의 `WEEKLY_TEST_POSTGRES_DSN` 은 127.0.0.1:15434 를 가리키고 거기에 아무것도 떠 있지 않을 수 있다 — 그러면 DB 시험이 skip 이 아니라 수백 건 FAIL 한다(`build the harness template: … connection refused`). **`pgvector/pgvector:pg16`** 으로 컨테이너를 띄울 것: `postgres:16` 에서는 `report_item_embeddings does not exist` 로 2건이 내 변경과 무관하게 실패한다(2026-09-23 회차에서 detached 워크트리로 입증됨).
  - `mutation-check`·`authz-check` 는 소스를 제자리에서 고쳐 쓴다 — 커밋 전·빌드와 병행 금지. `authz-check` 는 40분 이상이므로 이 45분 회차에서는 돌리지 말 것. `mutation-check` 를 돌린다면 커밋 **뒤** `--test TestTheOperatorCanPickTheFailuresOutOfAGoodWeek --budget 480` 으로 혼자, 끝난 뒤 트리가 깨끗한지 확인할 것.
  - 45분 한도: 백엔드+시험+openapi 를 먼저 커밋하고(목표 20분), 프런트+가이드를 둘째 커밋으로. 커밋 메시지는 한국어 `feat:`.
- 차선 후보: `README.md:47` · `docs/MCP.md:3` 의 'MCP 는 읽기 전용' 설명을 `internal/app/mcpwrite.go` 의 `mcp:write`(본인 보고서 쓰기, d5ee67a) 에 맞게 정정 (2/1/S) — 문서만 고치는 변경이라 값은 낮지만 사실과 어긋난다. 고칠 때 도구 이름·범위를 소스에서 확인하고 대체된 낡은 서술을 남기지 말 것. `AdminPage.tsx` 의 MCP OAuth 도움말에도 '개인 키와 같은 읽기 전용 규칙' 이라는 문구가 있어(이번에 확인) 같은 정정 대상인지 함께 볼 것.
