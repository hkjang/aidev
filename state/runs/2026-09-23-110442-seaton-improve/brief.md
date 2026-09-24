# 정찰 과제서 (2026-09-23, base main@00becef)

- 과제: 좌석맵 상세에서 배정 해제 단추 (가치 4 / 위험 2 / 작업량 S)

- 왜: 서버는 `DELETE /api/v1/seat-assignments/{seatID}`(internal/app/server.go:105 → seats.go:323 `unassignSeat`)를 갖고 있지만 화면에서 이 경로를 부르는 곳이 한 군데도 없다(web/src 전체에서 `method: "DELETE"` 는 MapsPage·SettingsPage·KeysPage·SeatMapPage 의 좌석 삭제 네 곳뿐이고 seat-assignments 는 없음 — 이번 회차에 grep 으로 확인). 그래서 관리자가 한 사람의 자리를 "비우는" 방법이 없다(다른 좌석으로 끌어 옮기는 것만 가능). 게다가 좌석 상세의 `삭제` 단추는 `disabled={Boolean(selected.employeeId)}`(SeatMapPage.tsx:2331)이라 배정된 좌석은 영영 지울 수 없는 막다른 길이 된다. 해제 단추 하나면 두 막힘이 동시에 풀린다.

- 수용 기준:
  1) 좌석맵에서 배정된 좌석을 고르면 상세 패널의 직원 블록에 `배정 해제` 단추가 보이고(관리자·좌석관리자만), 빈 좌석에는 보이지 않는다. `편집 모드`를 켜지 않아도 보인다 — 배정은 배치 편집이 아니며 기존 드래그 배정(`drop`, SeatMapPage.tsx:691)도 `manager` 만 보고 `editMode` 는 보지 않는다.
  2) 확인 창(기존 `removeSeat` 의 `confirm` 과 같은 방식)에서 승인하면 `DELETE /api/v1/seat-assignments/{seat.id}` 가 나가고 204 뒤 화면의 그 좌석이 `빈 좌석`으로 바뀐다. **이때 상세 패널이 닫히면 안 된다** — 지금 쓰이는 `chooseMap(mapId)`는 첫 줄에서 `setSelected(null)`(SeatMapPage.tsx:638)을 하므로 그대로 쓰면 패널이 사라진다. 좌석 목록만 다시 읽고 같은 id 의 좌석으로 `setSelected` 를 갱신하거나, `chooseMap` 뒤에 id 로 다시 고르는 작은 함수를 쓸 것.
  3) 실패(404 `not_assigned` 등)는 기존 `setError` 경로로 사용자에게 보인다.
  4) E2E 가 증명할 것: 실제 서버·DB 를 지나 (a) 해제 전 좌석에 직원 이름이 있고, (b) 화면에서 단추를 눌러 해제한 뒤 `GET /api/v1/seats` 응답에서 그 좌석의 `employeeName`/`employeeId` 가 사라지며, (c) 해제된 좌석은 상세에서 `빈 좌석` 칩과 "배정된 직원이 없습니다"를 보이고, (d) 편집 모드에서 그 좌석의 `삭제` 단추가 더 이상 disabled 가 아니다(=막다른 길이 풀렸다는 증거). 단위 테스트(Vitest)만으로는 배선을 증명하지 못하므로 E2E 가 본체다.
  5) 검증이 끝나면 좌석 배정이 원상 복구되어 다른 spec 이 깨지지 않는다.

- 건드릴 파일:
  - `web/src/pages/SeatMapPage.tsx`
    - 새 핸들러(예: `unassignSeat`) — `removeSeat`(758행) 바로 옆에 같은 모양으로. `api(\`/api/v1/seat-assignments/${selected.id}\`, { method: "DELETE" })` 뒤 좌석 재조회 + 선택 유지.
    - 상세 패널 직원 블록(2337~2366행, `selected.employeeId ? (...)` 안) 끝에 `manager &&` 로 감싼 `배정 해제` 버튼 추가. 아이콘은 이미 import 된 것 중에서 고르거나 `PersonRemoveRounded` 등을 새로 import.
    - 선택 유지용 재조회: `chooseMap` 을 그대로 쓰지 말 것(위 2번). 좌석만 다시 읽는 코드는 `chooseMap` 의 646~650행을 참고.
  - `internal/app/openapi.go:36` 근처 — `"/seat-assignments/{seatID}": {"delete": operation("좌석 배정 해제", true)}` 를 목록에 추가(지금 이 경로와 `/seats/{seatID}` DELETE 가 문서에 빠져 있음. 확인함). **목록 끝이 아니라 `/seat-assignments` 바로 아래**에 넣어도 무방하나, 미머지 브랜치가 목록 끝을 건드리므로 끝줄은 피할 것.
  - `web/e2e/seatmap.spec.ts` 또는 새 `web/e2e/seat-unassign.spec.ts` — 새 테스트 1~2건. **반드시 `helpers.ts` 의 `keepingSeats(page, [직원이름], async () => {...})` 로 감쌀 것.** 이 도우미가 이미 `DELETE /seat-assignments/{id}` + 재배정으로 원상복구를 한다(helpers.ts:86~120). `restoreSeat` 는 좌표만 되돌리므로 배정에는 쓸 수 없다.
  - 문서: `docs/API_AND_MCP.md` 의 좌석 배정 절과 `docs/USER_GUIDE.md` 의 좌석맵 절에 한 줄씩. PDF 는 굽지 말 것(미머지 브랜치와 이진 충돌 — 09-17·09-18 회차 선례).

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `go test ./...` / `go vet ./...` / `gofmt -l internal cmd`
  - `cd web && npm ci && npm run lint && npm test && npm run build`
  - E2E: `docker build -t seaton:e2e .` → PostgreSQL 컨테이너 + 수정본 서버 기동 → `cd web && E2E_BASE_URL=http://127.0.0.1:8080 E2E_USERNAME=admin E2E_PASSWORD=<부트스트랩 비밀번호> npx playwright test e2e/seatmap.spec.ts e2e/seat-unassign.spec.ts e2e/admin.spec.ts e2e/bulk-assign.spec.ts` (앞의 두 개 + 배정을 건드리는 나머지 두 개를 같이 돌려 복구가 되는지 본다). 준비 포함 수 분~10분.
  - 문서 갱신 시 `python3 scripts/build-docs.py USER_GUIDE` (HTML 만).

- 위험과 피할 것:
  - `internal/app/auth.go`(세션·CSRF·권한), `internal/database/migrations.sql`, `.github/workflows` 는 건드리지 말 것. 이번 과제는 서버 핸들러 변경이 **불필요**하다 — `unassignSeat` 는 이미 트랜잭션·`seat_history` 기록·`audit` 까지 한다. 서버를 고치고 싶어지면 그건 범위 이탈이다.
  - `chooseMap` 자체를 고쳐 `setSelected(null)` 을 없애지 말 것 — 도면을 바꿀 때 선택이 남아 있으면 다른 층의 좌석이 상세에 남는다. 새 재조회 경로를 따로 만들 것.
  - E2E 복구 누락이 이 저장소에서 반복된 실패 지점이다(helpers.ts:78~85 의 주석이 실제 사고를 기록하고 있다). `keepingSeats` 밖에서 배정을 바꾸지 말 것.
  - 미머지 브랜치(`auto/2026-09-16-1022` 메일 — seats.go 변경, `auto/2026-09-18-0413` MCP OAuth — auth.go 변경)와의 충돌을 줄이려면 `internal/app/seats.go` 는 손대지 않는 지금 계획을 유지할 것. `openapi.go`·`API_AND_MCP.md` 는 양쪽이 목록 끝을 건드리므로 끝줄 삽입을 피한다.
  - `삭제` 단추의 `disabled` 조건은 그대로 두고(배정된 좌석은 여전히 바로 못 지운다), 해제 후 상태가 갱신되어 자연히 풀리게 할 것.

- 차선 후보: 좌석 상세의 `조직` 행이 지정 구역과 직원 소속을 뒤섞는 것 고치기 — SeatMapPage.tsx:2356 이 `selected.organizationName`(= 좌석에 지정한 구역)을 먼저 보이고 직원 소속(`selectedEmployee?.organizationName`)을 대체값으로 쓴다. 게다가 `selectedEmployee` 는 왼쪽 **검색 결과**(`employees.find(...)`, 2314행 부근)에서만 찾으므로 검색하지 않고 좌석을 고르면 직원 소속은 아예 못 읽는다. 좌석 응답에는 `employeeOrganizationName` 이 이미 실려 온다(types.go:64, `seatSpeech` 가 이미 그 값을 쓴다 — SeatMapPage.tsx:174). `조직` 은 `employeeOrganizationName` 으로, 지정 구역은 `지정 구역` 별도 행으로 나누면 끝. (가치 3 / 위험 1 / 작업량 S. 09-21·09-22 회차에서 과제서는 채택됐으나 스킬 부재로 구현이 중단되어 코드에 그대로 남아 있음 — 이번 회차에 재확인함.)
