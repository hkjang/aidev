# 과제서 (2026-10-05, run 2026-10-05-085134-Vendra-improve, base main@a07d57b)

- 과제: 대시보드의 `activeContractValue` 가 계약 금액 문의 권한(`contract.amount.read`)을 읽게 하기 — 지금은 그 권한을 받은 역할에 ₩0 을 답한다 (가치 4 / 위험 2 / 작업량 S)

- 왜: `business_objects.amount` 를 여는 권한은 이 저장소 네 곳에서 전부 `<type>.amount.read` 다 — `objects.go:350`(목록·상세 redaction 이 없으면 `o.Amount = nil`), `objects.go:91`(금액 정렬 허용), MCP `get_expiring_contracts`(`integrations.go:662` 의 `hasPermission(p, "contract.amount.read")`), AI 컨텍스트의 만료 계약(`integrations.go:200`, 같은 식). 그런데 대시보드 KPI 의 `showContractValue`(`analytics.go:27`)는 `spend.read`/`analytics.read`/`*` 만 받고 `contract.amount.read` 라는 이름을 모른다. 그래서 `contract.read`+`contract.amount.read`+`dashboard.read` 를 받은 역할은 **같은 세션에서** `GET /api/v1/contracts/<id>` 는 실제 금액을, `GET /api/v1/dashboard` 는 `kpis.activeContractValue 0` 을 읽는다. 0 은 「가려졌다」가 아니라 「활성 계약 금액이 0원」이라는 허구의 사실이고, 화면에서는 그 카드 옆에 실제 금액을 답하는 계약 목록으로 가는 링크가 붙어 있다. 2026-10-04 회차(b50d3f3)가 `suppliers.annual_spend` 에서 고친 것과 같은 결함의 거울상이며, 그 회차가 "business_objects.amount 는 자기 문을 유지한다"며 **일부러 범위에서 뺀 자리**다(`suppliers.go:352-356` 주석이 그 사실을 남겨 두었다).

- 수용 기준:
  1) `["supplier.read","dashboard.read","contract.read","contract.amount.read"]` 역할로 `GET /api/v1/dashboard` 를 부르면 `kpis.activeContractValue` 가 활성/승인 계약 금액 합계를 답한다.
  2) 같은 역할로 `GET /api/v1/contracts/<id>` 가 답하는 `amount` 와 대시보드가 답하는 합계가 **서로 모순되지 않는다**(한쪽만 0 이 아니다). 테스트는 두 응답 본문을 같은 세션에서 읽어 이 모순으로 실패해야 한다.
  3) `spend.read` 단독·`analytics.read` 단독·`*` 경로는 **고치기 전과 똑같이** 금액을 답한다(넓히기만, 좁히기 없음).
  4) `["supplier.read","dashboard.read"]`(금액 권한 없음)과 `["supplier.read","dashboard.read","supplier.financial.read"]` 는 그대로 `activeContractValue 0` 이다 — `supplier.financial.read` 는 `suppliers.annual_spend` 의 문이고 이 컬럼의 문이 아니다. 기존 `TestDashboardReadsTheSameSupplierSpendDoor` 가 이 두 행을 이미 고정하고 있으니 **그 테스트가 계속 통과해야 한다**.
  5) `kpis.annualSpend`·`topSuppliers[].annualSpend` 는 한 자리도 바뀌지 않는다(두 플래그를 다시 하나로 합치지 말 것).

- 건드릴 파일 (프로덕션 1개 + 주석 1개 + 신규 테스트 1개):
  - `internal/httpapi/analytics.go:27` — `showContractValue` 에 `hasPermission(p, "contract.amount.read")` 를 **더한다**. 같은 줄의 `|| hasPermission(p, "*")` 는 제거한다: `permissionMatches`(`context.go:96-100`)가 `got == "*"` 를 먼저 보므로 `"*"` 를 가진 principal 은 `hasPermission(p,"spend.read")` 에서 이미 true 다 — b50d3f3 이 네 자리에서 같은 중복을 뺀 선례이고, 믿지 말고 `["*"]` 행으로 단정할 것(기존 테이블에 그 행이 있다).
  - `internal/httpapi/suppliers.go:352-356` — `canReadSupplierSpend` 주석의 "activeContractValue 는 business_objects.amount 로, 자기 읽기를 유지한다 — analytics.go 의 dashboard 를 보라" 문장이 이번 변경으로 낡는다. 두 문이 **여전히 다른 문**이라는 사실은 유지하되, 계약 금액 쪽도 자기 이름(`contract.amount.read`)을 읽게 됐다는 것으로 고쳐 쓸 것. 이 저장소 문체(왜 이렇게 되어 있나 / 고치지 않으면 무엇이 잘못 전달되나) 유지.
  - 신규 통합 테스트 파일(예: `internal/httpapi/dashboard_contract_amount_integration_test.go`) — 기존 테스트 파일은 **한 글자도 고치지 말 것**(이 저장소 관례: 추가만). 하네스는 그대로 재사용 가능하다: `newScopeWorld(t)`(`datascope_test.go`), `w.myContract`(`datascope_test.go:92`, object_type=contract), `grantOnly(t, w, permissions)`(`mcp_spend_permission_integration_test.go:235` — 픽스처의 `spend.*` 를 **교체**해야 차이가 보인다), `doRequest(t, w.handler, "GET", path, w.deptToken)`(`login_integration_test.go:391`). 시드: `UPDATE business_objects SET amount=$2,status='active' WHERE id=$1` — 대시보드 쿼리가 `status IN('active','approved')` 로 필터하므로 status 를 반드시 같이 쓸 것(`mcp_spend_permission_integration_test.go:147-150` 이 그 모양).

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - DB 필요. 전용 `postgres:16-alpine` 컨테이너 + 전용 DB 3개 + 세 DSN(`VENDRA_TEST_DSN`, `VENDRA_TEST_MIGRATE_DSN`, `VENDRA_TEST_UPGRADE_DSN`). DSN 이 없으면 통합 테스트는 **skip** 하고 초록을 내므로 초록을 증거로 쓰지 말 것.
  - 고치기 **전에 먼저**: `go test ./internal/httpapi/ -run 'TestDashboard' -count=1 -v` → 새 테스트가 실제 응답 본문으로 실패하고, 기존 `TestDashboardReadsTheSameSupplierSpendDoor` 는 **통과**(대조군)해야 한다. SKIP 이면 DSN 이 안 걸린 것이다.
  - 고친 뒤: 같은 명령 전부 PASS, 이어서 `go test ./internal/httpapi/ -run 'TestDashboard|TestSpend|TestMCP|TestObject|TestContract' -count=1`.
  - 전체: `go test ./internal/... ./cmd/... -count=1`, `gofmt -l internal cmd`(무출력), `go vet ./internal/... ./cmd/...`.
  - 커밋 전 `gate.py secrets` 를 diff 에 먼저(테스트에 긴 리터럴 쓰지 말 것 — 비밀정보 게이트가 회차를 통째로 떨군다).
  - 참고(기존 SKIP): `VENDRA_PERF` 2건, `failclosed_test.go` 2건(superuser), `TestFirstSupplierInAnEmptyRegisterIsAccepted` 1건은 환경성이며 이 과제와 무관하다.

- 위험과 피할 것:
  - 권한 게이트는 위험 구역이다. **더하기만** 할 것. 기존 `spend.read`/`analytics.read` 를 빼고 `contract.amount.read` 만 남기면 지금 금액을 보는 역할이 조용히 깨진다(수용 기준 3 이 그것을 잡는다).
  - `contract.read` 를 **함께 요구하도록 좁히지 말 것**. 형제 자리(`canReadSupplierSpend`)도 `supplier.read` 를 함께 요구하지 않는다 — 이번 회차에서 그 판단을 새로 내리지 말고 모양을 맞출 것.
  - `canReadSupplierSpend` 를 재사용하거나 `supplier.financial.read` 를 이 식에 넣지 말 것 — 다른 컬럼이고, 그렇게 합치는 것이 애초의 결함이었다.
  - SQL·스키마·migrations·`.github/workflows`·인증/세션/OIDC 는 건드리지 말 것. `objects.go:350`·`objects.go:91`·`integrations.go:200`·`integrations.go:662` 는 **이미 맞으므로 손대지 말 것**(이번 변경은 대시보드 한 자리를 그들에게 맞추는 것이다).
  - `recommend_suppliers.category`(`integrations.go:~670` 의 `stringValue`)와 `supplierIDArg`, `numberArg`/`riskCeilingArg` 호출 지점은 **미병합 브랜치 `auto/2026-10-01-1412`(fcce3d7)가 고치는 줄**이다(2026-10-05 확인: `git merge-base --is-ancestor fcce3d7 HEAD` → NOT_MERGED). 이번 회차에서 건드리면 충돌한다.
  - `t.Cleanup` 안의 DB 작업은 `context.Background()` 를 쓸 것(취소된 ctx 는 조용히 no-op). 픽스처 병렬 실행 금지, 정리는 `SC-` 접두사.

- 차선 후보: **`analyze_spend` 가 `supplier.financial.read` 역할에게 도구 자체를 거부하는 것**(가치 2 / 위험 2 / 작업량 S) — `mcpCall` 의 `required` 맵(`integrations.go:476-482`)이 `analyze_spend`→`spend.read` 를 요구하므로, `supplier.financial.read` 만 받은 역할은 `search_suppliers`·`get_supplier` 에서는 `annualSpend` 실측값을 받으면서 같은 컬럼을 집계하는 `analyze_spend` 에는 "Insufficient permission for analyze_spend" 를 받는다. 거부는 거짓말이 아니므로(0 을 답하는 것과 다르다) 가치는 낮다 — 1순위가 성립하지 않을 때만. 그 아래 3순위는 `eslint.config.js` 의 `ignores` 에 `coverage` 를 더하는 한 줄(2/1/S).

## 이번 정찰이 확인한 것 / 추측한 것
- 확인(코드로): `analytics.go:27`·`:56` 의 두 줄, `objects.go:350-351`·`:91`, `integrations.go:200`·`:662`, `context.go:96-100` 의 `permissionMatches`, `auth.go:410` 이 `<objectType>.amount.read` 를 카탈로그에 생성하는 것, `web/src/status.ts:148` 이 관리자 화면에 그 이름을 내주는 것, `app.go:190` 의 `require("dashboard.read", a.dashboard)`, 기존 `TestDashboardReadsTheSameSupplierSpendDoor` 가 `activeContractValue` 를 다섯 행으로 이미 고정하고 있는 것(`contract.amount.read` 행은 **없다**), `git merge-base` 로 fcce3d7 미병합.
- 미확인: 이번 세션에서 **테스트를 실행하지 않았다**(DB 컨테이너 미기동, `go build` 도 돌리지 않음 — 승인 거부). 그래서 「고치기 전에 새 테스트가 실패한다」는 구현자가 직접 확인해야 하는 예측이다. 웹 쪽(`web/src` 가 `activeContractValue` ₩0 카드를 어떻게 렌더하는지)도 미확인 — 수용 기준은 API 응답만으로 세웠다.
