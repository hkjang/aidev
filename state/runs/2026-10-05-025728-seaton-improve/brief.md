- 과제: 직원 가져오기·단건 저장으로 퇴직 처리한 직원의 좌석을 인사 동기화와 같은 방식으로 해제한다 (가치 3 / 위험 2 / 작업량 M)
- 왜: `runEmployeeSync`(internal/app/sync.go:139-146)는 status 가 `retired` 가 된 직원의 `seat_assignments` 를 종료하고 좌석을 `available` 로 돌리고 `seat_history` 에 `퇴직자 자동 좌석 해제`/`hr_sync` 를 남기는데, 파일 가져오기(`importEmployees`→`saveEmployee`)와 단건 저장(`upsertEmployee`→`saveEmployee`)은 `employees.status` 만 바꾼다. 그래서 파일 한 장으로 퇴직 처리하면 퇴직자가 좌석을 계속 점유해 대시보드의 "퇴직자 좌석"(dashboard.go:103 `retiredAssignments`, actionCountKeys 에 들어가 처리필요 배지에도 합산)이 올라가고 좌석맵에는 퇴직자가 앉아 있으며, 같은 사건(퇴직)을 두 입력 경로가 다르게 처리한다 — 이 저장소가 반복해서 고쳐 온 어긋남이다.
- 설계 결정(정찰이 정함): **해제한다.** 같은 status 전이를 인사 동기화는 해제하고 파일은 두는 상태가 근거 없고, 퇴직자 좌석은 이미 대시보드가 "처리필요"로 세고 있다. 다만 `seat_history` 에 반드시 기록을 남겨 누가·왜 비었는지 추적되게 하고, 좌석 자체를 지우거나 다른 사람에게 재배정하지는 않는다(되돌릴 수 있는 변경만).

- 수용 기준:
  1) 재직상태 칸이 `퇴직`(또는 `retired`)인 행을 직원 화면의 **직원 가져오기**로 올리면 그 직원의 열린 `seat_assignments` 가 `ended_at` 으로 닫히고, 그 좌석의 `seats.status` 가 `available` 이 되고, `GET /api/v1/employees` 가 그 직원에 대해 `seatId=null`·`seatNo=""` 를 돌려준다.
  2) `seat_history` 에 그 해제 기록이 한 건 남고 이력 화면에서 사람이 읽을 수 있는 "방식" 라벨로 보인다(원문 `employee_import` 가 그대로 노출되지 않는다 — `web/src/lib/format.ts` 의 `SOURCE_LABELS` 주석이 그 규칙을 못박아 두었다).
  3) 대시보드 `retiredAssignments` 가 그만큼 줄어든다(파일로 퇴직 처리한 직원이 거기 남지 않는다).
  4) 멱등·무해: 좌석이 없는 직원을 퇴직으로 올려도 오류가 나지 않고 이력도 남지 않으며, 이미 퇴직인 직원을 다시 올려도 이력이 늘지 않는다. `active`/`leave` 로 올린 행은 좌석을 건드리지 않는다(기존 가져오기 E2E 가 좌석을 그대로 두는지 함께 확인).
  5) 부분 성공 계약 유지: 한 행의 좌석 해제 실패가 `success` 로 보고된 다른 행을 되돌리지 않고, 실패 사유에는 pgx 원문이 아니라 사람 말이 뜬다(`userMessage(err, "저장하지 못했습니다")` 규칙 유지).

- 건드릴 파일 (프로덕션 3개):
  - `internal/app/sync.go:139-146` — 퇴직자 좌석 해제 루프를 `employees.go` 의 공유 함수로 뽑아 `tx` 를 넘겨 부르게 바꾼다. 동작은 그대로(`err == nil` 일 때만 좌석·이력을 쓰는 지금 모양이 "좌석 없음"을 자연스럽게 건너뛴다).
  - `internal/app/employees.go` — ① 새 함수 `releaseRetiredSeat(ctx, q, employeeID, actorID, source) error`(또는 같은 뜻의 이름). `*pgxpool.Pool` 과 `pgx.Tx` 를 모두 받으려면 이 파일에 작은 인터페이스를 두면 된다: `Exec(context.Context, string, ...any) (pgconn.CommandTag, error)` + `QueryRow(context.Context, string, ...any) pgx.Row` (`github.com/jackc/pgx/v5/pgconn` 는 pgx 모듈에 들어 있어 go.mod 추가가 필요 없다 — 확인할 것). ② `saveEmployee`(198-238)가 INSERT 성공 뒤 `in.Status == "retired"` 면 그 함수를 부른다. ③ `saveEmployee` 에 `source string` 인자를 더해 `importEmployees`(304)는 `"employee_import"`, `upsertEmployee`(251)는 `"manual"` 을 넘긴다. 호출자는 이 둘뿐이다(`grep -n "saveEmployee" internal/app` 로 재확인).
  - `web/src/lib/format.ts:14-20` — `SOURCE_LABELS` 에 `employee_import: "직원 가져오기"` 한 줄. 이 표가 이력 화면의 "방식" 선택 목록을 만들므로 빠뜨리면 그 기록을 골라 볼 수 없다.
  - 테스트/문서: `web/e2e/employee-import.spec.ts`(새 spec), 필요하면 `web/src/lib/format.test.ts`, `internal/app/employees_test.go`, `docs/USER_GUIDE.md` 3.4 절 한 문장 + `python3 scripts/build-docs.py USER_GUIDE` 로 HTML 만 재생성(PDF 는 관례대로 생략).

- 구현 순서(각 단계 끝에서 빌드가 서 있어야 한다):
  1) `releaseRetiredSeat` 추가 + `sync.go` 가 그것을 쓰게 바꾼다 → `go build ./... && go test ./...` 로 기존 동작이 그대로인지 본다.
  2) `saveEmployee` 에 호출과 `source` 인자를 더한다 → 다시 빌드·테스트.
  3) `format.ts` 라벨 한 줄 + 프런트 체크.
  4) E2E: **변경 전 이미지에서 붉은 것을 먼저 보고** 수정본에서 통과시킨다.

- 검증 명령 (이 저장소에서 실제로 도는 것, 이번 정찰에서 Go 쪽은 실행해 녹색 확인):
  - `cd /home/hkjang/.cache/auto-improve-wt/seaton && go build ./... && go vet ./... && go test ./...` / `gofmt -l .` (출력 없어야 함)
  - `cd web && npm ci && npm test && npm run lint && npm run build` (lint = `tsc -b`; 이 체크아웃에는 node_modules 가 없어 `npm ci` 가 먼저 필요하다)
  - 실서버 E2E: `docker build --build-arg VERSION=e2e -t seaton:e2e .` → 전용 bridge 네트워크 + `postgres:16-alpine` + 앱 컨테이너를 `-p 127.0.0.1:<빈포트>:8080` 로 띄우고 `/readyz` 로 준비 확인(`/api/v1/health` 를 기다리지 말 것) → `cd web && E2E_BASE_URL=http://127.0.0.1:<포트> E2E_USERNAME=admin E2E_PASSWORD=ci-e2e-password-123 npx playwright test employee-import employee-export employee-filter dashboard` 식으로 직원·대시보드 관련만 먼저. 역검증은 변경 전 HEAD(50768e3)를 `git archive` 로 **새로** 구운 이미지(`seaton:e2e-before-1005`)에서.

- 위험과 피할 것:
  - **E2E 가 시드 직원의 좌석을 실제로 비운다.** `employee-import.spec.ts` 의 `keepingEmployee`(79-100)는 `POST /api/v1/employees` 로 직원 열만 되돌리고 **좌석 배정은 되돌리지 않는다** — 그대로 두면 시드 10명 전원 배정을 전제하는 `employee-export.spec.ts`(표 10행·좌석 칸)와 대시보드 spec 이 엉뚱한 이유로 깨진다. 끝에 `POST /api/v1/seat-assignments {employeeId, seatId, source:"manual"}`(seats.go:260-285, `X-CSRF-Token` 필요)로 다시 붙이는 복구를 반드시 넣고, spec 끝에서 그 직원의 `seatNo` 가 원래 값인지 단정할 것. `helpers.restoreSeat` 는 좌표만 복구하니 쓸 수 없다.
  - **시드는 재실행으로 복구되지 않는다.** `web/e2e/seed.mjs` 는 도면이 이미 있으면 `repair()` 로 빠져 직원을 다시 POST 하지 않는다. 손으로 curl 검증을 했으면 DB 컨테이너를 새로 만들고 시드할 것.
  - `migrations.sql` 은 건드리지 말 것. `seat_history.source` 에는 CHECK 가 없어(migrations.sql:165 `source text NOT NULL`) 새 값 추가에 마이그레이션이 필요 없다 — 이번 정찰에서 확인했다.
  - `auth.go`·`mcpoauth.go`·`internal/tracking`·`.github/workflows` 는 범위 밖.
  - `saveEmployee` 는 트랜잭션 안이 아니다(가져오기는 행마다 독립이 계약). 좌석 해제 세 문장이 중간에 깨지면 "배정은 닫혔는데 좌석이 occupied" 가 남을 수 있다. 가장 작은 처방은 **해제 세 문장만** 짧은 트랜잭션으로 묶는 것이다(직원 INSERT 까지 함께 묶으면 부분 성공 계약과 `importEmployees` 전체를 다시 설계해야 하므로 하지 말 것).
  - `upsertEmployee` 경로까지 닫는 것은 지난 회차(`normalizeEmployeeStatus` 를 `saveEmployee` 로 옮긴 판단)의 연장이다. 다만 `keepingEmployee` 의 복구 POST 는 status 를 `active` 로 되돌리므로 거기서 해제가 또 일어나지는 않는다.
  - **미확인**: ① `performAssignment` 가 "퇴직 상태 직원" 또는 "방금 비운 좌석" 에 대해 거부하는 조건이 있는지 읽지 않았다(복구 POST 가 409 로 조용히 실패하면 뒤 spec 이 깨진다 — 복구 응답 코드를 단정할 것). ② 대시보드 spec 이 `retiredAssignments` 숫자를 문자열로 단정하는지 확인하지 않았다. ③ 이번 회차에 프런트 테스트·E2E 는 실행하지 않았다(Go 만 녹색 확인).

- 차선 후보: 가져오기 실패 행 사유에 어느 열이 비었는지 적기 — `importEmployees`(employees.go:298-300)의 `fail("사번/이름 누락")` 은 사번과 이름 중 무엇이 빈지 가리지 않는다. `normalizeEmployeeStatus`(147)가 세운 "어느 값이 문제인지 적는다" 꼴로 맞추면 된다. 프로덕션 1파일. 착수 전에 `grep -rn "사번/이름 누락" web docs internal` 로 기존 E2E·문서가 그 문장을 단정하는지 먼저 확인할 것.
