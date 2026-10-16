- 과제: MCP `search_suppliers`·`compare_suppliers` 가 **가려진 연간지출을 `annualSpend: 0`** 으로 답하는 것 고치기 — 같은 파일의 형제 도구들은 같은 가림을 `null` 로 답한다 (가치 3 / 위험 2 / 작업량 S)

- 왜: `search_suppliers`(`integrations.go:542`)와 `compare_suppliers`(`:591`)의 SELECT 는 `CASE WHEN $5 THEN annual_spend ELSE 0 END` 로, 권한이 없으면 **0 을 하나의 숫자로** 내보낸다. 그런데 같은 열을 같은 플래그(`canReadSupplierSpend`)로 가리는 형제 둘은 `CASE WHEN $N THEN annual_spend END` → **null** 로 답한다(`recommend_suppliers` `:695`, AI 컨텍스트 `:194`). `suppliers.annual_spend` 는 `001_initial.sql:63` 에서 `numeric(20,2) NOT NULL DEFAULT 0` 이므로 이 열에 NULL 은 **가림으로만** 생긴다 — 즉 null 은 "가려짐", 0 은 "실제로 0원" 을 정확히 구분할 수 있는데 이 두 도구만 그 구분을 버린다.
  도구 결과는 모델 컨텍스트에 그대로 떨어지고 모델에게는 대조할 두 번째 출처가 없으므로, `{"name":"SC-MINE","annualSpend":0}` 은 "거래액이 0원인 업체" 라는 **허구의 사실**로 사람에게 전달된다. 이 저장소는 대시보드 KPI 에서 같은 거짓을 같은 이유로 이미 고쳤고(ae0eaff), 기존 테스트 파일이 이 결함을 **문장으로 이미 적어 두었다**: `mcp_spend_permission_integration_test.go:21-24` "A zero is not a withheld number, it is a number … relayed to the person who asked as a supplier the company spends nothing with", `:221-222` "a column left at its default zero cannot show the difference between a figure withheld and a figure that is really zero, **which is the whole subject**". 지난 회차는 권한 문(door)만 맞췄고 0/null 쪽 절반이 남았다.

- 수용 기준:
  1) 금액 권한이 **없는** 역할이 `search_suppliers`·`compare_suppliers` 를 부르면 각 행의 `annualSpend` 가 **`null`** 이다(키는 남고 값만 null — `recommend_suppliers` 와 같은 모양). 응답 본문으로 확인한다.
  2) 금액 권한이 **있는** 역할에게는 실제 숫자가 그대로 나온다. 그리고 `annual_spend` 가 **진짜 0** 인 공급업체는 **`0`** 으로 나온다 — 고치기 전에는 "가려진 지출" 과 "지출 0원" 이 같은 `0` 이었고, 고친 뒤에는 둘이 다른 값이다. 이 "0 ≠ null" 대조가 이 과제의 핵심 증거다.
  3) `recommend_suppliers` 는 한 글자도 바뀌지 않고, 같은 역할·같은 공급업체에 대해 세 도구가 **같은 답**(null 또는 같은 숫자)을 준다.
  4) 신규 테스트가 고치기 전에 **실제 응답 본문으로 실패**하고(실패 로그를 `assets/` 에 남길 것) 고친 뒤 통과한다. 인과는 쿼리 한 곳만 `ELSE 0` 으로 되돌려 같은 테스트만 다시 실패하는 것으로 확인하고 원복한다(파일 통째 revert 금지 — 빌드 에러는 증거가 아니다).
  5) 기존 테스트 파일은 **한 글자도** 고치지 않는다(추가만). `TestSupplierSpendDoorOpensTheSameOnEveryTool`·`TestEverySupplierSpendPermissionIsReadTheSameWay`·`TestMCPSupplierIDToolsStillAnswerWellFormedCalls` 가 대조군으로 그대로 통과해야 한다 — 통과한다고 예상하는 근거: 두 테스트가 쓰는 `spendByName`(`:246-259`)은 `m["annualSpend"].(float64)` 를 `, _` 로 읽어 null 을 0 으로 받고, 헬퍼 주석(`:243-245`)이 "a row that omits the key or carries null reads as zero here" 라고 그 의도를 명시한다. **그러므로 기존 테스트는 이 변경을 증명하지 못한다 — 신규 테스트는 `map[string]any` 로 받아 `값이 nil 인지`를 직접 봐야 한다**(`.(float64)` 로 읽으면 결함이 구조적으로 안 보인다).

- 건드릴 파일 (프로덕션 1개):
  - `internal/httpapi/integrations.go:542` `runMCPTool` / `case "search_suppliers"` — SELECT 의 `CASE WHEN $5 THEN annual_spend ELSE 0 END` → `CASE WHEN $5 THEN annual_spend END`.
  - `internal/httpapi/integrations.go:591` 같은 함수 / `case "compare_suppliers"` — 같은 치환(파라미터는 `$5`).
  - `internal/httpapi/integrations.go:701-714` `supplierSummaryRows` — `var spend float64` → `var spend *float64`, `rows.Scan(..., &spend)`, 맵에 `"annualSpend": spend`. **이 함수 안에 이미 `grade *string`·`score *float64` 가 같은 이유로 포인터**이므로 새 관례를 들여오는 것이 아니다. nil `*float64` 는 `encoding/json` 이 `null` 로 마셜한다. 이 함수는 위 두 case 에서만 호출된다(`:547`, `:596` — 전부).
  - 주석: 왜 이 열의 null 이 "가림" 만 뜻하는지(`NOT NULL DEFAULT 0`)와 형제 도구와 같은 모양을 쓴다는 것을 한두 줄로 남길 것.
  - 신규 테스트 1파일(예: `internal/httpapi/mcp_spend_null_integration_test.go`). 재사용 가능한 하네스: `newScopeWorld`, `grantOnly`(테이블 각 행 **안에서** 호출할 것 — 루프 밖에서 부르면 엉뚱한 403 으로 실패한다), `seedSecondOwnSupplier`, `seedAnnualSpend(t,w,id,0)` 으로 진짜 0 을 심기, `callMCPTool`, `toolRows`(`mcptools_test.go:26`, `structuredContent` 만 읽는다).

- 검증 명령 (전용 컨테이너·전용 DB·DSN 3개 — DSN 없이 통과한 Go 테스트는 DB 검증이 아니다):
  - `docker run -d --name vendra-1017-improve-pg -e POSTGRES_HOST_AUTH_METHOD=trust -p 127.0.0.1:55501:5432 postgres:16-alpine` 후 DB 3개 생성, `VENDRA_TEST_DSN`/`VENDRA_TEST_MIGRATE_DSN`/`VENDRA_TEST_UPGRADE_DSN`(뒤 둘은 각각 빈 DB) 설정. 정리는 `docker rm -f vendra-1017-improve-pg` — **패턴 기반 pkill 금지**(실행 중인 셸을 죽인다).
  - 집중: `go test ./internal/httpapi/ -run 'TestMCP|TestSupplierSpendDoor|TestEverySupplierSpendPermission' -count=1 -v` → **SKIP 0** 을 이름까지 확인.
  - 전체: `go test ./internal/... ./cmd/... -count=1` (httpapi 약 30초). 기존 SKIP 5건은 환경성이다(`VENDRA_PERF` 2, `failclosed_test.go` 2, `TestFirstSupplierInAnEmptyRegisterIsAccepted` 1) — 그 이상이면 조사.
  - `gofmt -l internal cmd`(무출력), `go vet ./internal/... ./cmd/...`, 그리고 커밋 전 `gate.py secrets` 를 diff 에 돌릴 것(12자 이상 리터럴이 password/secret/token 옆에 있으면 러너가 전체를 떨어뜨린다).
  - 웹은 손대지 않으므로 `npm test` 불필요. 혹시 돌린다면 `/mnt/c` 에서는 Vitest 가 못 돌고 PATH 의 떠돌이 node 가 jsdom 스위트를 통째로 죽인다 — 원인을 lockfile 로 돌리지 말 것.

- 위험과 피할 것:
  - **REST 와 웹은 건드리지 않는다.** `redactSupplier`(`suppliers.go:366-368`)는 `s.AnnualSpend = 0` 을 유지한다. 웹의 `types.ts:25,60` 과 `Suppliers.tsx`·`Dashboard.tsx`·`SupplierNetwork.tsx` 가 `annualSpend: number` 로 읽으므로 REST 를 null 로 바꾸면 화면이 깨진다. 이번 변경 경로(`supplierSummaryRows`)는 MCP 전용이라 웹 타입 변경이 **필요 없다** — 이것이 이 과제를 S 로 유지하는 조건이다.
  - `analyze_spend`(`:668`, 지출을 가리지 않고 도구 자체를 `spend.read` 로 막는다)와 대시보드 KPI(`analytics.go:42-43`)는 범위 밖이다. 두 문은 여전히 두 문이다 — 공유 플래그로 합치지 말 것.
  - 운영자 지시: **같은 종류의 코드가 여러 경로에 있으면 통합하지 말고 좁히는 방향으로만** 고칠 것. 이 과제는 MCP 두 도구를 MCP 형제와 맞추는 것이고, REST/대시보드/AI 컨텍스트는 각자 계약을 유지한다.
  - 감사 기록에 값을 넘기지 말 것(`:513` 은 `arguments` 를 그대로 details 에 넣는다 — **그 줄을 건드리지 말고**, 새 코드에서 금액을 details 로 보내지도 말 것).
  - 보호 경로를 건드릴 필요가 없다: 인증/세션/`migrations`/`.github/workflows` 모두 무변경.
  - 미병합 `011c5a3`(duplicate submit 방지, `web/src/pages/Objects.tsx`)이 HEAD 조상이 아니다. 이번 과제는 Go 전용이라 충돌하지 않는다 — **웹 과제로 바꾸지 말 것.**
  - `jsonb_build_object` 가 null 값을 키째로 유지한다는 점에 의존한다(`recommend_suppliers` 가 이미 그렇게 동작한다는 것이 근거). 신규 테스트의 기준 1 은 `recommend_suppliers` 의 실제 응답과 **나란히** 비교해 이 가정을 같은 실행에서 고정할 것.

- 차선 후보: **저장된 보기 조회 실패를 화면이 말하고 재시도하게 하기** — `web/src/pages/Objects.tsx` `ObjectList.loadViews`(≈:267)가 실패를 `setViews([])` 로 삼켜, 저장한 보기가 **없어진 것처럼** 보이고 이유도 재시도도 없다. 28572eb 가 목록 조회 실패에 쓴 `{key,message}` + 한 줄 배너 모양을 그대로 베낄 수 있다. 단 같은 파일에 미병합 `011c5a3` 이 있으므로 `NewObject` 쪽은 한 줄도 건드리지 말 것.
