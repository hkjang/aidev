- 과제: 내 제출 기록 카드가 "아직 열려 있는 지난주" 를 말한다 (가치 3 / 위험 2 / 작업량 M)

- 왜: `myParticipation`(internal/app/participation.go:67-79)의 격자 질의는 `WHERE … AND `+deadlinePassed 로 **마감이 지난 주만** 읽고, 열려 있는 주는 121-127행의 두 번째 질의가 **이번 주(`current`) 하나만** 본다. 기본 규칙(`workflow.deadline_days=7`, `deadline_hour=24`)에서 지난주의 마감은 `지난주 시작 + 7일 + 24시간` = **이번 주 화요일 0시**이므로, 월요일 하루 종일 지난주는 아직 열려 있는데 그 주는 `owed`·`filed`·`streak`·`lastMissed`(마감이 지난 주만) 에도 없고 `thisWeek*`(이번 주만) 에도 없다 — 즉 응답 어디에도 없다. 그래서 지난주를 안 낸 사람이 월요일에 카드에서 보는 문장은 "이번 주(…)는 아직입니다. 마감 전까지는 기록이 끊기지 않습니다." 뿐이고, **오늘 안에 내면 기록이 지켜지는 지난주가 있다는 사실을 볼 길이 없다**. 고치면 카드가 아직 살릴 수 있는 주를 이름으로 말한다.

- 수용 기준:
  1) `GET /api/v1/me/participation` 응답에 `openArrears`(문자열, `omitempty`)가 생긴다 — **이번 주보다 앞서면서, 아직 마감이 지나지 않았고, 그 사람에게 귀속되며(`expectedFromWeek` 통과), 제출이 없는 주 가운데 가장 오래된 주의 시작일**. 해당하는 주가 없으면 필드가 아예 없다.
  2) 기존 숫자가 그대로다 — `streak`·`best`·`owed`·`filed`·`window`·`thisWeekStart`·`thisWeekFiled`·`lastMissed` 의 값이 이 변경 전후로 같다(열린 주는 여전히 `owed`/`filed`/`streak` 에 들지 않는다 — 들면 월요일마다 모두의 기록이 끊긴다).
  3) 대시보드 `나의 제출 기록` 카드가 `openArrears` 가 있을 때만 그 주를 가리키는 문장을 한 줄 더 보여 준다. 없으면 문구가 늘지 않는다(상시 문구 금지).
  4) 시험이 증명하는 것: **고치기 전 코드에서 먼저 실패**하는 것을 확인할 것. 지난주가 열려 있는 규칙에서 (a) 안 낸 열린 지난주가 `openArrears` 로 나오고, (b) 그 주를 내면 `openArrears` 가 **사라지며**, (c) 같은 응답의 `streak` 가 (a)·(b) 양쪽에서 변하지 않는다.

- 건드릴 파일 (제품 4 + 시험 1 + 문서 2):
  - `internal/app/participation.go:26-44 participationView` — `OpenArrears string \`json:"openArrears,omitempty"\`` 필드 1개 추가. 주석에 "열린 주는 여전히 세지 않는다, 이름만 말한다" 를 남길 것.
  - `internal/app/participation.go:67-79 myParticipation` 격자 질의 — `WHERE` 에서 `AND `+deadlinePassed 를 빼고, **SELECT 목록 세 번째 컬럼으로** `deadlinePassed` 를 그대로 넣어 `closed bool` 로 스캔한다(`week.day::date >= `+expectedFromWeek 는 `WHERE` 에 그대로 둔다). 새 질의·새 플레이스홀더를 만들지 말 것 — `$3·$4·$5` 는 이미 timezone·days·hour 로 바인딩돼 있고 `deadlinePassed = deadlinePassedFor("week.day")`(internal/app/admin_analytics.go:333) 를 그대로 재사용하면 관리자 미제출 명단과 같은 판정을 쓴다.
  - `internal/app/participation.go:86-111` 스캔 루프 — `closed` 를 먼저 보고, **`!closed` 인 행은 `view.Owed++` 이전에 `continue`** 한다(그 앞에서 `running`·`counting` 도 건드리지 않아야 2)번이 지켜진다). `continue` 하기 전, `!filed && !day.Equal(current)` 이면 `view.OpenArrears = day.Format(dateLayout)` 로 **덮어쓴다** — `ORDER BY week.day DESC` 이므로 마지막에 남는 값이 가장 오래된 열린 주다. `current` 를 빼는 이유는 그 주가 이미 `thisWeek*` 로 답해지기 때문(같은 주를 두 필드가 다르게 말하면 안 된다).
  - `internal/app/participation.go:121-127` 두 번째 질의는 **손대지 말 것**. 이 질의에는 `expectedFromWeek` 필터가 없어 격자와 범위가 다르다 — 격자에서 `thisWeekFiled` 를 뽑아 대체하면 이번 주에 가입한 사람의 답이 바뀐다.
  - `frontend/src/types.ts:431-434 ParticipationRecord` — `openArrears?: string` 추가.
  - `frontend/src/pages/DashboardPage.tsx:45-48` — `record.lastMissed` 문장 **앞에** `{record.openArrears && \` 지난주(${record.openArrears})는 아직 마감 전입니다 — 지금 내면 기록에 들어갑니다.\`}` 를 넣는다. 39행의 카드 노출 조건 `record && record.owed > 0` 은 그대로 둘 것(바꾸면 숫자가 전부 0인 카드가 신규 사용자에게 뜬다).
  - `docs/openapi.yaml:123-137` `/me/participation` — 이 경로는 `$ref: Success` 만 쓰고 스키마 본문이 없으므로 **description 에 `openArrears` 한 문장만** 더한다(135-137행의 `lastMissed` 설명 뒤). 새 스키마·새 응답을 만들지 말 것.
  - `docs/USER_GUIDE.md:146-152` 3.5.1 — 150행("아직 열려 있는 이번 주는…") 뒤에 아직 마감 전인 지난주를 가리켜 준다는 한 문장. HTML 재생성은 `python3 scripts/render-docs.py USER_GUIDE` 만 돌릴 것(다른 문서 재생성 금지).

- 시험 (기존 파일에 덧붙일 것, 새 시험 함수를 만들지 말 것):
  - `internal/app/participation_test.go:38 participationRecordScenario` 안. 이 시험은 이미 `t.Run("last week still open", … "13")`(24행)로 `workflow.deadline_days=13` 을 걸어 **오늘이 무슨 요일이든 지난주가 열려 있는 경로**를 밟는다 — 월요일을 기다릴 필요가 없다.
  - 이미 있는 `closed := closedWeekStart(now, server.app.deadlineRule(server.ctx()))`(50행)와 `current`(49행)로 기대값을 만든다:
    `wantOpen := ""; if oldest := closed.AddDate(0,0,7); oldest.Before(current) { wantOpen = oldest.Format(dateLayout) }`
    시나리오가 내는 주는 `week(1)=closed`·`week(2)`·`week(3)` 와 `current` 뿐이므로 그 사이의 열린 주들은 비어 있다.
  - 첫 `record := read()`(89행 부근) 뒤에 `record.OpenArrears != wantOpen` 를 검사하고, `wantOpen != ""` 이면 `file(wantOpen)` 뒤 다시 읽어 `OpenArrears == ""` 이고 `Streak` 가 그대로인 것을 검사한다. 기본 규칙 서브테스트에서는 오늘이 월요일이 아니면 `wantOpen == ""` 이라 조용히 지나간다 — 정상이다.
  - `// guards: myParticipation` 주석은 이미 20행에 있다.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `gofmt -l internal/app` (빈 출력), `go build ./...`, `go vet ./...`
  - `go test ./internal/app -run TestMyRecordCountsTheWeeksTheArrearsListCounts -count=1 -v` — **고치기 전에 먼저 돌려 `openArrears` 검사가 실패하는 것을 확인할 것**(특히 `last week still open` 서브테스트).
  - `go test ./... -count=1` (실제 DB 로 150초 안팎)
  - `python3 scripts/openapi-check.py`, `python3 scripts/paging-check.py`, `python3 scripts/guard-check.py --changed main`
  - `npm --prefix frontend ci && npm --prefix frontend run lint && npm --prefix frontend test && npm --prefix frontend run build`
  - `python3 scripts/render-docs.py USER_GUIDE`

- 위험과 피할 것:
  - **DB 환경 미확인**: 이번 정찰 세션은 `go build`·`docker ps`·환경변수 조회가 모두 권한으로 막혀 실행하지 못했다. `WEEKLY_TEST_POSTGRES_DSN`(프로필: 127.0.0.1:15434, weeklytest)이 실제로 살아 있는지는 **미확인**이다. 죽어 있으면 `pgvector/pgvector:pg16` 으로 컨테이너를 올릴 것 — `postgres:16` 으로는 `report_item_embeddings` 관련 2건이 환경적으로 실패한다(2026-09-23 회차 기록).
  - **공유 시험 DB 의 잔재**로 마이그레이션 개수 시험이 환경적으로 실패할 수 있다. 실패하면 detached 워크트리로 HEAD 를 꺼내 같은 컨테이너에서 돌려 내 변경과 무관함을 증명할 것.
  - 보호 경로(`auth.go`·`crypto.go`·`internal/app/migrations/`·`.github/workflows/`)는 전혀 건드리지 않는다. 마이그레이션 없음, 설정 키 추가 없음.
  - `admin_analytics.go` 의 `deadlinePassed`·`expectedFromWeek`·`weekIsOwed` 를 **고치지 말 것** — 관리자 미제출 명단·제출률이 같은 조각을 읽는다. 참여 카드를 위해 조각을 바꾸면 관리자 화면 숫자가 같이 움직인다(과거에 제출률 109.1% 를 낸 자리다).
  - 마감 시각을 Go 에서 다시 계산해 응답에 싣지 말 것(예: "화요일 0시까지"). `deadlineRule.instant` 와 SQL `deadlinePassedFor` 가 각각 days+hours 를 더하는 두 경로이고, 운영자 지침이 정면으로 걸린다. 이번 과제는 **주 시작일 문자열 하나**로 끝낸다.
  - 응답에 `total` 이나 배열 목록을 더하지 말 것 — `paging-check.py` 가 그 화면에 offset 경로를 요구한다. 스칼라 한 개(`openArrears`)는 지난 회차의 `truncated?: boolean` 과 같은 모양이라 요구가 생기지 않을 것으로 본다(이번 회차에 실행으로는 미확인).
  - `mutation-check`·`authz-check` 는 소스를 제자리에서 고쳐 쓴다 — 커밋 **뒤에**, 혼자, `--test TestMyRecordCountsTheWeeksTheArrearsListCounts --budget 480` 로만 돌릴 것.
  - 대시보드에는 SettingsTab 과 마찬가지로 렌더링 시험이 없다. 수용 기준 3)은 `tsc -b`·`build` 와 JSX 위치 확인까지만 가능하다 — 렌더 시험으로 증명했다고 쓰지 말 것.

- 차선 후보: **README.md:47 · docs/MCP.md:3 · AdminPage.tsx 의 MCP SSO 도움말** 세 곳에 남은 "읽기 전용" 서술을 `internal/app/mcpwrite.go` 의 `mcpMayWrite`·`mcp:write`(본인 보고서 쓰기, d5ee67a) 기준으로 정정 (가치 2 / 위험 1 / 작업량 S). 도구 이름과 범위를 소스에서 확인해 적고, 대체된 옛 서술은 남기지 말 것.
