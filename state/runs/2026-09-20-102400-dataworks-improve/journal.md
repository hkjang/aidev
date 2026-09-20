# 회차 노트 2026-09-20-102400-dataworks-improve — dataworks
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 10:24] base pinned — main@e69dac7
- [러너 10:24] autonomy release — 

## 정찰 노트
- 선택 이유: 5회차 연속 보류된 `internal/dataworks` 도메인 테스트 보강을 골랐고, 읽어 보니 두 함수에 실제 불일치(비엄격 pricing 술어가 QueryCost 만 봄 / 은퇴 후보의 `expiredAt` 이 trim 없이 파싱해 `store.EntitlementActive` 와 어긋남)가 있어 "효과 없는 변경" 반려 위험 없이 출력이 바뀌는 수정 + 테스트로 묶었다. 고아 접근권 집계는 상품 목록 조회가 더 얽혀 차선으로.
- 확신 없는 곳: `docs/OPERATIONS.md` 5절 위치와 PDF 유무는 미확인(`ls docs/*.pdf` 로 확인 지시). `handleDataWorksActionCenter` 행 번호(약 280~310)는 grep 결과로 추정. 공백 포함 `expires_at` 레거시 행이 실제 배포 DB 에 있는지는 미확인(쓰기 경로는 1431행에서 trim).
- 구현자 주의: `LLMCost` 를 술어에 새로 넣지 말 것(넓히기 금지), 비엄격 분기의 경고 미생성 동작은 유지, `PublishGateResult` JSON 이름 불변. `store` 가 `dataworks` 를 import 하지 않으므로 순환 없음. 새 테스트 2개는 변경을 되돌리면 실제로 실패하는지 확인할 것.
- [러너 10:26] scout done — `EvaluatePublishGateV2`·`EvaluateRetirementCandidate` 가 런타임과 다르게 읽는 값 두 곳을 통일하고 표 테스트로 고정

## 구현 노트
- 바꾼 것: `internal/dataworks/domain.go` — `pricingModelConfigured(p, cost)` 헬퍼를 두고 `EvaluatePublishGateV2` 엄격/비엄격 분기가 모두 호출(비엄격이 OpsCost·DataProcessingCost 를 안 보던 문제), `EvaluateRetirementCandidate` 의 활성 판정을 `store.EntitlementActive` 로 위임하고 `expiredAt` 삭제. 커밋 ee1e127.
- 검증: 표 테스트 4개 추가, 두 수정을 각각 되돌려 해당 테스트만 실패함을 확인. `go build/vet/test ./...` 전체 통과, api-surface-audit gap 0, gofmt 클린.
- 확신 없는 곳: `TestEvaluatePublishGateV2StrictEvidence`·`TestEvaluateRetirementCandidateThresholds` 는 현행 동작을 고정하는 특성화 테스트라 변경 전에도 통과한다(과제서 의도대로 "회귀 방지"용). `TestRetirementCountsEntitlementLikeRuntimeGate` 의 `ExpiresAt == now` 사례는 옛 `Before(now)` 규칙과 실제로 다른 결과(활성→비활성)를 내며 이것도 런타임과 맞춘 의도적 변화다.
- 일부러 하지 않은 것: `LLMCost` 를 술어에 넣지 않음(현행 두 분기 어디에도 없음), 비엄격 분기에 경고/MissingEvidence 추가하지 않음(화면 문구가 바뀜), V1 게이트·`RequiresStrictPublishGate`·`BuildEvidencePack`·호출부(`admin_dataworks.go`)·web 미변경, OPERATIONS 는 PDF 없어 재생성 없음.
- 다음 역할 주의: 모든 테스트는 순수 함수 단위 테스트라 DB 불필요(`go test ./internal/dataworks/`). `PublishGateResult` JSON 이름은 바뀌지 않아 `web/src/types/dataworks.ts` 와 호환.
- [러너 10:31] brief accepted — 채택 — 과제서의 두 불일치(비엄격 pricing 술어·`expiredAt` vs `EntitlementActive`)가 코드와 정확히 일치했고 수용 기준 1~3 을 �
- [러너 10:31] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인한 것: `git diff e69dac7...HEAD`(3파일, 로컬 `main` 은 stale 이라 pinned base 사용). `domain.go` 를 base 로 되돌려 `go test` — 실제 수정을 지키는 두 테스트(pricing 술어 ops/data-processing 케이스, 은퇴 후보 padded-row·expiry==now)만 실패하는 것을 직접 재현. `store.EntitlementActive`·`parseStoredTime` 을 열어 trim·해석불가→비활성·`After(now)` 규칙이 옛 `expiredAt` 과 어디서 갈리는지 확인(기존 unparseable 테스트 유지). OPERATIONS.md 의 라우트는 `admin_dataworks.go:968` 로 실재, PDF 없음.
- 못 본 것: 실제 배포 DB 에 공백 포함 `expires_at` 행이 있는지(구현자도 미확인) — 영향은 은퇴 점수 -20 뿐이라 승인에 영향 없음. web 화면은 JSON 이름 불변이라 열지 않음.
- 판정 approve, risk low, blocking 없음. 보안·법무 접촉면 없음(순수 도메인 함수).
- 릴리즈 노트: (1) 비엄격 상품도 OpsCost/DataProcessingCost 만으로 `pricing_model_configured=true`, (2) `expires_at == now` 인 접근권은 이제 비활성(런타임 게이트와 동일), (3) 저장된 retirement 후보 점수는 다음 재평가 때 바뀔 수 있음.
- 다음 회차: `TestEvaluateRetirementCandidateThresholds` 의 "deduplicated per asset" 케이스명은 reason 문자열만 중복 제거되고 risk 는 행마다 +10 누적됨을 뜻함 — 코드와 일치하나 이름 정리 후보.
- [러너 10:33] review approved — 리뷰 승인 (risk=low)
- [러너 10:33] pr created — https://github.com/hkjang/dataworks/pull/25
- [러너 10:36] ci passed — 검사 2개 모두 success
- [러너 10:36] merge done — ee1e127
- [러너 10:48] release published — v0.9.58
- [러너 10:48] gh-release created — GitHub Release v0.9.58
- [러너 10:48] manifest ok — dataworks-v0.9.58.tar.gz 
- [러너 10:49] assets uploaded — 1개
- [러너 10:49] assets verified — v0.9.58 자산 1개 (이전 v0.9.57: 1)
