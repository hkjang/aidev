## Data Works v0.9.58

### 주요 변경 사항
- **출시 게이트의 가격 판정과 은퇴 후보의 접근권 판정이 런타임과 다르게 읽히던 문제 수정 (v0.9.58)**: `EvaluatePublishGateV2` 의 비엄격 분기는 `PricingModelConfigured` 를 `PricingModel != "" || QueryCost > 0` 로만 계산해 운영비(`OpsCost`)나 가공비(`DataProcessingCost`)만 입력한 표준 상품이 워크벤치 게이트 카드("가격 및 비용")와 작업 공간 완성도 행에 미완료로 떴고, 같은 비용을 가진 엄격 상품은 `Query|Ops|DataProcessing` 술어로 통과했음. 두 분기가 하나의 헬퍼 `pricingModelConfigured(p, cost)` 를 호출하게 통일했다(`LLMCost` 는 종전대로 세지 않고, 비엄격 분기가 경고를 쌓지 않는 동작과 JSON 필드 이름은 그대로). `EvaluateRetirementCandidate` 는 활성 접근권을 자체 `expiredAt`(trim 없이 `RFC3339Nano` 만 파싱, `Before(now)`)로 판정해 런타임 게이트·action center 의 `store.EntitlementActive`(trim 후 `parseStoredTime`, `After(now)`)와 어긋났고, 공백이 든 레거시 `expires_at` 행은 고객 키가 정상 서빙받는데도 "no active API entitlements" 로 위험 점수가 +20 됐음. 판정을 `store.EntitlementActive` 에 위임하고 호출부가 없어진 `expiredAt` 을 삭제했다(`store` 는 `dataworks` 를 import 하지 않아 순환 없음). 표 테스트 4개 추가 — `TestEvaluatePublishGateV2StrictEvidence`(완료·quality 결과 없음은 경고만·quality 실패는 차단·민감 상품 masking 없음 차단·risk/api/sla/pricing 누락은 경고만), `TestEvaluatePublishGateV2PricingPredicateIsSameForStandardAndStrict`(OpsCost/DataProcessingCost/QueryCost/PricingModel/없음 5사례에서 표준·엄격 상품이 같은 값), `TestEvaluateRetirementCandidateThresholds`(keep/improve/retire 경계 49·50·75, 100 클램프, 사유 중복 제거, watermark 대소문자 무관 +10, 평균 fit 45 경계), `TestRetirementCountsEntitlementLikeRuntimeGate`(공백 행 → UsageCount 1, `ExpiresAt == now` 는 비활성, 빈 만료는 활성 — 각 픽스처가 `store.EntitlementActive` 와 일치함을 함께 단언). 두 수정을 각각 되돌리면 해당 테스트만 실패함을 확인. `docs/OPERATIONS.md` 5절 "만료 예정 계약·권한 확인"에 은퇴 후보 평가도 같은 규칙을 쓴다는 두 줄 추가. `dataworks:v0.9.58` 이미지를 `dataworks-v0.9.58.tar.gz` 단일 오프라인 GitHub Release asset으로 배포.


### 배포 파일
| 파일 | 설명 |
|------|------|
| dataworks-v0.9.58.tar.gz | 오프라인 적재 가능한 Data Works Docker 이미지 (linux/amd64) |

### 빠른 시작
```bash
# 이미지 로드
gunzip -c dataworks-v0.9.58.tar.gz | docker load

# 실행
docker run -d --name dataworks --restart=always \
  -p 8080:8080 \
  -e POSTGRES_DSN='postgres://dataworks:change-me@postgres:5432/dataworks?sslmode=disable' \
  -e BOOTSTRAP_ADMIN='admin@dataworks.local' \
  -e BOOTSTRAP_ADMIN_PASSWORD='change-me' \
  -e ENCRYPTION_KEY='replace-with-64-hex-characters' \
  dataworks:v0.9.58
```
