# 수리 요약 (efafce8)

- 지적은 맞았다. `analytics.go:14` 의 `showAmounts` 한 플래그가 **두 열**을 덮고 있었다 — `suppliers.annual_spend`(KPI `annualSpend` + `topSuppliers[].annualSpend`)와 `business_objects.amount`(`activeContractValue`). 그래서 `supplier.financial.read` 만 가진 역할은 대시보드에서 ₩0, /suppliers 에서 실제 금액을 읽었다.
- 비평가 경고대로 통째 치환하지 않고 **플래그를 둘로 쪼갰다**: `showSpend = canReadSupplierSpend(p)` 가 annual_spend 두 자리를, `showContractValue` 가 옛 목록(`spend.read || analytics.read || "*"`)을 그대로 유지해 계약 금액은 넓어지지 않는다.
- `suppliers.go` 주석에 대시보드 두 자리를 열거에 넣고, 「이것은 suppliers.annual_spend 뿐이며 activeContractValue 는 자기 문을 유지한다」는 범위 문장을 추가해 커밋 제목의 "every surface" 주장을 코드와 일치시켰다.
- 재현·검증: 새 `TestDashboardReadsTheSameSupplierSpendDoor`(REST `/api/v1/dashboard`, 권한 5벌)가 수정 전 `supplier.financial.read` 한 벌에서 3줄 실패 → 수정 후 통과. 전체 DB 통합 `go test ./internal/... ./cmd/...` 전부 ok(httpapi 29.2s), SKIP 5건은 기존과 동일, `gofmt -l`·`go vet` 무출력, gate.py secrets clean.
- 이 테스트는 `activeContractValue` 도 단언하므로(financial.read 에서 0, spend.read/analytics.read/`*` 에서 실제값) 다음 사람이 두 플래그를 다시 합치면 실패한다.
