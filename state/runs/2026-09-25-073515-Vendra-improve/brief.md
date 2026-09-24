- 과제: recommend_suppliers 가 설명에 적힌 「최대 위험 등급」을 실제 인자로 받게 하고, minScore 를 문자열로 받았을 때 조용히 무필터가 되지 않게 하기 (가치 3 / 위험 2 / 작업량 M)
- 왜: `internal/httpapi/integrations.go:423` 의 도구 설명은 "품목, 최소 점수, 최대 위험 등급으로 공급업체 후보를 추천합니다" 이고 `docs/USER_GUIDE.md:297` 도 같은 말을 하는데, inputSchema 의 properties 는 `category`/`minScore`/`limit` 뿐이라 위험 등급 인자가 존재하지 않고 SQL 은 `risk_level NOT IN('CRITICAL')` 로 고정되어 있다 — 모델이 설명을 믿고 등급 상한을 보내면 인자는 말없이 버려지고 HIGH 업체가 그대로 섞여 돌아온다(지난 두 회차의 「공개한 limit 이 적용되지 않음」·「인자 실수를 권한 거부로 답함」과 같은 계열의 계약 불일치다). 같은 분기의 `minScore, _ := args["minScore"].(float64)` (`:608`) 는 `"80"` 처럼 문자열로 오면 0 이 되어 `COALESCE(score,0)>=0` — 필터가 통째로 사라진 채 200 으로 응답한다.
- 수용 기준:
  1) `recommend_suppliers` 의 inputSchema 에 위험 등급 상한 인자가 있고(이름은 `maxRisk` 권장, `"type":"string"` + `"enum"` 은 `riskGrades` 에서 끌어 쓰되 `CRITICAL` 은 넣지 않음 — 지금도 절대 추천하지 않는 등급이다), 그 값을 주면 응답에 그 등급보다 높은 업체가 한 건도 없다. `riskGrades` (`internal/httpapi/rows.go:593`, `LOW MEDIUM HIGH CRITICAL` 순서)가 순위의 정본이며 문자열 비교로 순서를 지어내지 말 것.
  2) `maxRisk` 를 주지 않은 호출의 결과 집합은 수정 전과 정확히 같다. 즉 기존 `risk_level NOT IN('CRITICAL')` 는 그대로 두고 인자가 있을 때만 `AND risk_level = ANY($n::text[])` 같은 조건을 **덧붙인다** — 기본 경로를 화이트리스트로 바꾸면 어휘 밖 값(빈 문자열·레거시)을 가진 업체가 조용히 사라진다.
  3) 등급이 어휘에 없는 값(`"매우높음"`, `"low"` 같은 소문자)이면 조용히 무시하지 말고 `mcpToolError` 로 그 값을 들어 거절한다. `supplierIDArg` (`:684`)가 세운 방식 — 모델이 고칠 수 있는 말로 답하기 — 을 따른다.
  4) `minScore` 가 숫자가 아닌 값(문자열 `"80"`, `true`, 객체)으로 오면 0 으로 떨어지지 않는다. 계약은 둘 중 하나를 골라 주석에 적을 것: (a) `mcpToolError("minScore must be a number, not text: %q")` 로 거절 — `supplierIDArg` 와 같은 방향이라 권장, (b) 숫자로 읽히는 문자열은 파싱. **조용한 0 만은 남기지 말 것.**
  5) 테스트가 증명할 것: (i) 수정 전에 먼저 돌려 실패를 볼 것 — `maxRisk` 를 준 호출이 HIGH 업체를 돌려주고, `minScore:"80"` 이 점수 낮은 업체까지 돌려준다. (ii) 수정 후 `maxRisk:"MEDIUM"` 은 LOW·MEDIUM 만, `maxRisk:"LOW"` 는 LOW 만, 인자 없음은 수정 전과 같은 집합(CRITICAL 만 제외). (iii) 어휘 밖 등급과 비숫자 minScore 가 `isError` 로 그 값을 말한다. (iv) 기존 limit 상한·스코프(`department`/`own`)·소프트 삭제 제외·`structuredContent` 와 `content[0].text` 일치가 그대로 유지된다.
- 건드릴 파일:
  - `internal/httpapi/integrations.go:423` — `mcpTools` 의 `recommend_suppliers` inputSchema 에 등급 인자 추가. 설명 문자열은 이미 맞는 말을 하고 있으므로 건드릴 필요 없다.
  - `internal/httpapi/integrations.go:606-609` — `runMCPTool` 의 `recommend_suppliers` 분기. SQL 의 번호 자리 표시자가 `$1..$7` 로 꽉 차 있으니 인자를 더하면 번호를 다시 세야 한다(`showSpend` 가 `$6`, `limit` 이 `$7`). `orgInScope("organization_id","$3","$4")` 의 문자열 인자도 같이 어긋나기 쉬운 자리다.
  - `internal/httpapi/integrations.go:716-729` 부근 — `intNumber` 옆에 숫자/등급 인자 헬퍼를 두고 왜 그렇게 답하는지 주석을 남긴다(이 파일의 관례가 그렇다).
  - 새 테스트 파일(예: `internal/httpapi/mcp_recommend_integration_test.go`). 기존 파일에 끼워 넣기보다 새 파일이 낫다.
- 검증 명령:
  - `go test ./internal/httpapi/ -run TestMCP -count=1` — DSN 없이도 컴파일·스키마 테스트가 돈다(이번 정찰에서 `TestMCPToolSchemasNameTheSupplierConsistently`·`TestMCPToolErrorsStaySeparateFromInternalFaults` 0.047s 통과 확인).
  - 실제 증명은 DB 가 필요하다: `docker run` 으로 `postgres:16-alpine` 를 띄워 전용 DB 세 개를 만들고 `VENDRA_TEST_DSN`·`VENDRA_TEST_MIGRATE_DSN`·`VENDRA_TEST_UPGRADE_DSN`(뒤 둘은 빈 DB)을 걸어 `go test ./internal/... ./cmd/... -count=1` 전체. 새 테스트가 SKIP 되지 않았는지 `-v` 로 확인할 것.
  - `gofmt -l internal cmd` 무결, `go vet ./internal/... ./cmd/...` 통과.
- 테스트 하네스(헤매지 말 것): `newScopeWorld(t)` 가 부서/own 토큰과 `w.handler` 를 준다. `callMCPTool(t, w, token, tool, argsJSON)` (`internal/httpapi/mcptools_test.go:13`) 로 실제 세션 쿠키를 붙여 `POST /mcp` 를 때리고, `toolRows(t, rec)` 로 `structuredContent` 를 읽는다. `internal/httpapi/mcp_limits_integration_test.go:13-105` 가 `SC-LIMIT-` 접두사로 105건을 시드하고 구조화 응답과 `content[0].text` 를 `reflect.DeepEqual` 로 맞춰 보는 본보기다 — 시드 접두사는 `SC-` 로 시작해야 `wipe` 가 치운다. `t.Cleanup` 안의 DB 작업은 `context.Background()` 를 쓸 것(취소된 ctx 는 조용히 아무 것도 안 한다). 이 fixture 는 병렬 실행 금지.
- 위험과 피할 것:
  - 기본 동작(인자 없음)을 바꾸지 말 것. 수용 기준 2) 가 이 회차의 가장 큰 회귀 위험이다.
  - `risk_level` 어휘를 `riskGrades` 말고 다른 데서 새로 적지 말 것. 소문자/한국어 표기를 새로 허용하면 REST 쪽 `riskGradeField` 검증과 어긋난다.
  - 다른 MCP 도구(`search_suppliers`·`analyze_spend`·`get_expiring_contracts`)의 기존 상한·`days 180/3650` 정책·`analyze_spend` 의 share 분모는 손대지 말 것 — share 는 접근 가능한 전체 지출 기준이어야 한다는 결정이 이미 서 있다.
  - 보호 경로(auth/세션/OIDC·`internal/db/migrations`·`.github/workflows`)와 프런트(`web/src`)는 이번 과제에서 건드릴 이유가 없다.
  - 비밀정보 게이트: 테스트에 12자 이상 리터럴을 `password`/`secret`/`token` 근처에 두지 말 것. 커밋 전 러너의 `gate.py secrets` 를 diff 에 돌릴 것.
  - 문서: `docs/USER_GUIDE.md:297` 은 이미 「최대 위험 등급」을 적고 있으므로 수정이 아니라 **이제 사실이 되는** 쪽이다. 굳이 고치지 말 것.
- 차선 후보: 사용자 가이드 4.6 의 MCP 도구표(`docs/USER_GUIDE.md` 열한 줄)를 실제 `mcpTools`/`tools/list` 응답과 양방향으로 묶는 가드 테스트 (가치 2 / 위험 1 / 작업량 S). `mcpTools` 가 패키지 변수라 DB 없이 돌고, `internal/httpapi/guide_docs_test.go` 가 같은 방식(그림·환경 변수 양방향 대조 + 「긁은 수가 너무 적으면 실패」 자기 검증)의 본보기다.
