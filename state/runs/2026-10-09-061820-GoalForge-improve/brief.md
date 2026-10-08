- 과제: MCP activity_report의 since 설명을 실제 지원 문법과 맞추고 공개 프로토콜로 고정한다 (가치 2 / 위험 1 / 작업량 S)
- 왜: `internal/mcp/tools.go:toolDescriptors`는 since 예시로 `7d`를 광고하지만 `activityReport`는 `time.ParseDuration`을 써서 그 입력을 거절한다. 예시를 `168h`로 고치고 실제 tools/list → tools/call 계약을 검증하면 에이전트가 서버 설명대로 호출했다가 실패하는 경로를 없앤다.
- 수용 기준:
  1) 실제 `Server.Serve`의 tools/list 응답에서 activity_report.inputSchema.properties.since.description은 `24h`, `168h`, 기본 24h를 안내하고 `7d`를 지원 예시로 내놓지 않는다. 문장 전체 일치나 소스 문자열 검사는 쓰지 않는다.
  2) 같은 실제 SQLite fixture + Serve의 tools/call로 설명에 명시된 두 예시를 호출하면 RPC error가 없고 result.isError=false이며 content의 text가 유효한 `store.ActivityReport` JSON이다. 입력 생략과 빈 문자열도 기본 24h를 유지한다. report.Since가 각 호출 전후 시각에서 기대 기간을 뺀 구간 안에 있는지 확인해 168h가 실제 일주일을 전달함을 증명한다(고정 시각 일치·sleep 금지).
  3) `7d`, `0h`, `-1h`, 임의 잘못된 문자열은 RPC error가 아니라 result.isError=true로 거절한다. 오류 문장 전체 대신 오류 여부와 positive duration 안내만 확인한다. days 지원이나 파서 정책은 바꾸지 않는다.
  4) 새 회귀 테스트는 수정 전 공개 설명의 7d 때문에 실패하고 설명 수정 후 통과한다. 기존 MCP/전체 테스트도 통과하며 auth·저장소 정책 변경은 없다.
- 건드릴 파일:
  - `internal/mcp/tools.go:toolDescriptors` (현재 73행) — activity_report since 예시의 7d를 168h로 교정. `activityReport` (654행)는 읽되 수정하지 않는다.
  - `internal/mcp/server_test.go:fixture`, `TestStdioProtocolLifecycle` 인접 — `TestActivityReportDurationContract` 테스트 추가. 기존 fixture가 실제 SQLite, New, 프로젝트/목표를 준비하므로 재사용. Serve 출력은 bytes.Buffer 등으로 받아 rpcEnvelope 및 MCP content를 구조적으로 해석한다. 실제 tools/list에서 찾은 설명과 tools/call 결과를 연결한다. 생산 코드 1개, 테스트 1개가 목표.
- 검증 명령:
  - `go test ./internal/mcp -run '^TestActivityReportDurationContract$' -count=1` (구현자가 추가할 테스트)
  - `go test ./internal/mcp -count=1`
  - `go test ./... -count=1`
  - `go vet ./...`; `go build ./...`; `gofmt -l internal/mcp/tools.go internal/mcp/server_test.go`; `git diff --check`
  - 정찰 실측: MCP 테스트 exit 0 (0.548s), 전체 테스트 exit 0 (32개 ok; sqlite 46.017s). 이번 vet/build 별도 명령은 미실시. 전체 출력은 비-verbose여서 skip 수와 외부 PostgreSQL 실행 여부는 미확인.
- 위험과 피할 것: 기간 파서 통합·days/weeks 지원 확장·CLI/API 기간 동작 변경 금지. `internal/mcp/server.go` transport/auth/session, migrations, .github/workflows는 변경 대상이 아니다. store.Activity SQL/집계 개선도 별도 과제다. RPC envelope가 성공이어도 result.isError일 수 있고 content.text에는 다시 JSON이 들어 있으므로 두 층을 확인한다. 테스트용 fake store나 소스 검색만으로 배선을 증명하지 않는다. 현재 root remote.origin.pushurl=DISABLED이므로 전체 테스트의 기존 push 제한을 우회하거나 skip을 늘리지 않는다.
- 차선 후보: HTTP doctor와 MCP project_readiness의 완료 가능성 판정 계약 테스트 (가치 2 / 위험 1 / 작업량 M) — 1순위가 이미 해결된 경우만. `internal/api/setup.go:doctor`와 `internal/mcp/tools.go:projectReadiness`의 FAIL/WARN 결론을 실제 SQLite·HTTP/MCP로 검증하되 환경 진단과 plan runnable 의미를 합치지 않는다. 별도 하네스 세부 설계는 미확인.

확인한 증거와 범위
- 기준 main@c319962. CLI `go run ./cmd/goalforge --db <회차>/assets/scout-mcp.db mcp`에 JSON-RPC를 보냈다. tools/list는 7d를 안내했고 같은 프로세스의 7d 호출은 `error: since must be a positive duration such as 24h, got "7d"` / isError=true였다. 168h·24h·빈 문자열은 실제 보고서 JSON, 0h·-1h는 isError=true였다. CLI 프로세스 exit 0.
- 실제 읽은 보조 위치: `internal/mcp/server.go:Serve`, `internal/store/sqlite/report.go:Activity`, `ActivityReport`. ActivityReport의 Since/Until은 JSON 태그가 없으므로 공개 필드는 대문자다. 출력에 Projects가 null이어도 유효한 빈 보고서다.
- 이전 회차 웹훅 예약 해제는 8f8a0f3이 이미 머지되었으므로 다시 선정하지 않는다. 승인 일일 한도/옛 승인 철회는 권한·감사 정책 설계가 필요하고, 공유 git 하네스 리팩터는 이번의 확실한 설명 오류보다 범위가 크다.

대안 비교와 선택 가정 (solution-exploration)
- 선택: 설명을 기존 문법에 맞춘다. 생산 파일 1개이며 입력·저장 계약을 유지한다. 사용자가 days 문법 자체를 요구한 적이 없다는 것이 핵심 가정이다.
- 대안: MCP에 d/w 파서를 도입한다. 사람에게 편할 수 있으나 다른 기간 경로와 경계값·오버플로 계약을 새로 설계해야 하므로 이번 범위에서 제외한다.
- 대안: 예시를 삭제하거나 현행 유지한다. 작업은 가장 작지만 일주일 조회 안내가 사라지거나 재현된 실패가 남으므로 채택하지 않는다.

실행 순서와 체크포인트 (implementation-planning)
1. [미착수] 기준 확인: `go test ./internal/mcp -count=1`. 통과를 자동 체크포인트로 삼는다. 사람 승인 단계 없음.
2. [미착수] 두 지정 파일에 예시 교정과 계약 테스트를 한 묶음으로 작성. `go test ./internal/mcp -run '^TestActivityReportDurationContract$' -count=1`로 통과 확인. 설명 변경만 잠깐 원복해 같은 테스트의 실패 원인이 7d임을 확인 후 즉시 복구한다. 원복 전후 빌드는 계속 가능해야 한다. 자동 체크포인트이며 실패하면 범위를 넓히지 말고 과제서를 수정한다.
3. [미착수] MCP 및 전체 테스트, vet/build/포맷/diff 검증을 수행하고 실제 결과를 journal에 남긴다. 정찰의 기준선 통과를 구현 검증으로 대체하지 않는다. 구현 완료 여부는 구현자가 갱신한다.

작업량 근거와 여유 (estimating-and-contingency)
- 정찰자의 bottom-up 추정: 재현/하네스 확인 4–6분, 설명+프로토콜 테스트 10–14분, 검증/기록 6–10분 = 기본 20–30분. 테스트 응답 해석·시각 구간 처리의 알려진 불확실성에 5–10분 contingency를 별도로 둬 총 25–40분이다. 보정된 통계는 없으며 45분 안에 끝낼 확신은 정성적으로 높음이다.
- 유사 사례는 10/05의 실제 CLI/MCP 저장 계약 테스트(테스트 파일 3개)이나 소요시간 기록이 없어 숫자 교차추정은 하지 않는다. 이번은 두 파일, 기존 하네스 재사용이라 상대적 S 판단만 보조한다.
- 미지의 추가 범위를 위한 management reserve는 이번 회차에 배정하지 않는다. 파서 확장·새 환경 하네스가 필요해지면 같은 예산 안에 끼워 넣지 말고 별도 pending 아이디어로 분리한다.
- 추정 방법 참고: [GAO Cost Estimating and Assessment Guide](https://www.gao.gov/products/gao-20-195g)의 범위·가정·작업 분해·위험·실측 갱신 원칙. 위 분 단위 값은 GAO의 수치가 아니라 정찰자의 추정이다.
- 요청 스킬은 전용 Skill 도구가 노출되지 않아 로컬 SKILL.md를 직접 읽었다: `/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md`, 같은 plugins 아래 `technology/skills/implementation-planning/SKILL.md`, `technology/skills/solution-exploration/SKILL.md`.
