# 과제서 2026-09-20 — dataworks

- 과제: `EvaluatePublishGateV2`·`EvaluateRetirementCandidate` 가 런타임과 다르게 읽는 값 두 곳을 통일하고 표 테스트로 고정 (가치 3 / 위험 1 / 작업량 M)
- 왜: `internal/dataworks/domain.go` 의 두 도메인 함수는 직접 단위 테스트가 없고(`domain_test.go` 는 V1 게이트·retire 한 사례·unparseable expiry 만), 읽어 보니 같은 값을 다른 경로와 다르게 읽는 자리가 둘 있다. (1) `EvaluatePublishGateV2` 의 비엄격(non-strict) 분기(`domain.go:184`)는 `PricingModelConfigured = p.PricingModel != "" || cost.QueryCost > 0` 인데 엄격 분기(`domain.go:221`)는 `QueryCost || OpsCost || DataProcessingCost` 라서, 운영비만 입력한 표준 상품은 워크벤치 게이트 카드(`web/src/features/publish-gate/publish-gate-card.tsx:33` "가격 및 비용" warning)와 작업 공간 완성도(`web/src/pages/products/product-workspace-page.tsx:237` CompletionRow)에 미완료로 뜨고, 같은 비용을 가진 엄격 상품은 통과로 뜬다. (2) `EvaluateRetirementCandidate`(`domain.go:424`) 의 활성 접근권 판정은 자체 `expiredAt`(`domain.go:791`, trim 없이 `RFC3339Nano` 파싱, `Before(now)`)을 쓰는데 런타임 게이트·action center 는 `store.EntitlementActive`(`internal/store/dataworks_operations.go:541`, trim 후 `parseStoredTime`, `After(now)`)를 쓴다. 쓰기 경로가 trim 하기 전에 저장된 공백 포함 `expires_at` 행은 고객 키가 정상 서빙받는데 은퇴 후보 평가에서는 "no active API entitlements" 로 잡혀 위험 점수가 +20 된다(운영자 지침: "같은 값을 읽는 경로가 여럿이면 모든 경로가 같게 읽어야 한다").
- 수용 기준:
  1) `EvaluatePublishGateV2` 는 엄격/비엄격 모두 같은 술어로 `PricingModelConfigured` 를 계산한다 — 헬퍼 `pricingModelConfigured(p store.DataProduct, cost *store.ProductCost) bool` 하나를 두고 두 분기가 호출. 비엄격 상품 + `ProductCost{OpsCost: 25}` 만 있을 때 `pricing_model_configured == true`(변경 전 false). `LLMCost` 는 넓히지 말 것(현재 엄격 분기도 세지 않음 — 현행 유지).
  2) `EvaluateRetirementCandidate` 는 활성 접근권을 `store.EntitlementActive(ent, now)` 로 판정한다. 호출부가 없어진 `expiredAt` 은 삭제(다른 호출부 없음 — `grep expiredAt( internal/dataworks` 로 재확인). 기존 `TestRetirementIgnoresEntitlementWithUnparseableExpiry` 는 그대로 통과해야 함(`parseStoredTime` 도 `2026-12-31` 을 못 읽으므로 동일).
  3) 새 표 테스트가 증명할 것 — `internal/dataworks/domain_test.go` 에 추가:
     - `TestEvaluatePublishGateV2StrictEvidence`(표): 기존 `TestEvaluatePublishGateBlocksStrictProductUntilEvidenceIsComplete` 의 완전 픽스처(readiness·approvals 3건·pack) 위에 V2 인자를 얹어 (a) quality 통과 1건·risk true·`APISpec` 비어있지 않음·sla non-nil·`PricingModel` 있음·masking true → `Allowed` 이고 `MissingEvidence` 비어 있음, (b) `qualityResults` 빈 슬라이스 → `QualityPassed=false`·`MissingEvidence` 에 `quality_results`·**`Allowed` 는 그대로 true**(경고만), (c) `Passed:false` 결과 1건 → `BlockedReasons` 에 "data quality rule failed: …"·`Allowed=false`, (d) `Sensitivity:"personal_credit"` 에 masking false → `BlockedReasons`·`MissingEvidence` 에 `masking_policy`·`Allowed=false`, (e) risk false·`APISpec` 빈값·sla nil·pricing 없음 → 4개 모두 false 이고 `MissingEvidence` 에 `risk_review`·`api_contract`·`product_sla`·`pricing_model`, 그러나 `Allowed` 는 true(경고만).
     - `TestEvaluatePublishGateV2PricingPredicateIsSameForStandardAndStrict`: 비엄격 상품(`SourceType:"batch"`, `RiskScore:10`, `Sensitivity:"public"`)과 엄격 상품 각각에 `cost=&ProductCost{OpsCost:25}` / `{DataProcessingCost:10}` / `{QueryCost:100}` / `nil`+`PricingModel:"per_call"` / `nil` 을 넣어 `PricingModelConfigured` 가 두 상품에서 같게 나오는지 표로 확인. 비엄격 결과에는 `pricing_model` 이 `MissingEvidence` 에 없고 `StrictGate=false`·`MaskingConfigured=true`·`QualityPassed=true` 인 것도 확인(비엄격 분기가 경고를 쌓지 않는 현행 유지).
     - `TestEvaluateRetirementCandidateThresholds`(표): `RiskScore` 와 신호 조합으로 `keep`(<50)·`improve`(50~74)·`retire`(>=75) 경계, `clamp` 로 100 초과 안 됨, reason 이 `uniqueStrings` 로 중복 제거되고 신호 없으면 "healthy product signals", `LastUsedAt` 이 가장 큰 `UpdatedAt`, watermark `Stale`/`DELAYED`/`failed` 대소문자 무관 +10 씩.
     - `TestRetirementCountsEntitlementLikeRuntimeGate`: `Status:" Active "`, `ExpiresAt:" 2030-01-01T00:00:00Z "`(앞뒤 공백) 행이 `UsageCount=1`(변경 전 0 → 이 테스트가 변경 전 실패해야 함), `Status:"revoked"` 는 0, `ExpiresAt` 이 `now` 와 정확히 같으면 런타임과 같이 비활성(`After(now)` 규칙), `ExpiresAt:""` 는 활성.
- 건드릴 파일:
  - `internal/dataworks/domain.go:163-240 EvaluatePublishGateV2` — 헬퍼 `pricingModelConfigured` 추출, 두 분기에서 호출
  - `internal/dataworks/domain.go:415-478 EvaluateRetirementCandidate` — 424행 판정을 `store.EntitlementActive(ent, now)` 로 교체; `domain.go:791 expiredAt` 삭제
  - `internal/dataworks/domain_test.go` — 위 표 테스트 4개 추가(기존 7개 유지, `hasString` 헬퍼 재사용)
  - `docs/OPERATIONS.md` 5절(접근권/계약 운영 절, 이전 회차가 `EntitlementActive` 통일을 적은 자리) — 은퇴 후보 평가도 런타임 판정을 쓴다는 한 줄. 사용자 가이드 PDF 재생성은 불필요(OPERATIONS 는 PDF 없음 — 미확인이면 `ls docs/*.pdf` 로 확인)
- 검증 명령:
  - `go test ./internal/dataworks/ -run 'PublishGateV2|Retirement' -v`
  - 변경 전 실패 확인: `store.EntitlementActive` 교체와 헬퍼 추출을 각각 되돌려 `TestRetirementCountsEntitlementLikeRuntimeGate`·`TestEvaluatePublishGateV2PricingPredicateIsSameForStandardAndStrict` 가 실제로 실패하는 것을 보고 다시 적용
  - `gofmt -l internal/dataworks/` (출력 없어야 함)
  - `go build ./... && go vet ./... && go test ./...` (약 2분)
  - `go run ./cmd/api-surface-audit` (gap 0 — 라우트 변경은 없으므로 그대로여야 함)
  - web 변경 없음 → 웹 체크 생략 가능(JSON 필드 이름은 바뀌지 않음)
- 위험과 피할 것:
  - `internal/store` 는 `internal/dataworks` 를 import 하지 않으므로 `dataworks → store.EntitlementActive` 호출은 순환 없음(이미 `store` import 중). 반대 방향으로 옮기지 말 것.
  - `EvaluatePublishGate`(V1)·`RequiresStrictPublishGate`·승인 단계 로직·`BuildEvidencePack` 은 손대지 말 것. `PublishGateResult` JSON 필드 이름 변경 금지(`web/src/types/dataworks.ts:208` 이 읽음).
  - `LLMCost` 를 가격 술어에 새로 포함하지 말 것 — 현행 두 분기 어디에도 없어 넓히면 별도 판단이 필요.
  - `internal/proxy/admin_dataworks.go:1651·1736` 호출부는 그대로. 인증·마이그레이션·워크플로·`web/` 는 무관.
  - 테스트는 실제 함수 출력(구조체 값)으로 증명하고 소스 문자열 검사·대역 금지(운영자 지침).
  - `EvaluatePublishGateV2` 비엄격 분기가 경고를 쌓지 않는 현행 동작을 바꾸지 말 것(화면 문구가 바뀜) — 술어만 통일.
- 차선 후보: action center 가 상품이 사라진 고아 Contract Scope·Entitlement 도 집계 (가치 2 / 위험 1 / S) — `internal/proxy/admin_dataworks.go` `handleDataWorksActionCenter`(약 280~310행) 에서 상품 목록과 대조해 `orphaned_access` 카운트 추가 + HTTP 회귀 테스트. 1순위와 겹치지 않음.
