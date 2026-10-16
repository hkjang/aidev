# 수리 요약 — jikim PR #64

- 문제: `docs/guides/compatibility.md:136` 의 새 문장이 (1) 비정규 버전 토큰 거절을 "저장소를 호출하기 전에" 라고 썼지만 실제 거절은 저장소 **안**(`internal/store/transit.go:67`, DB 쿼리 `transitKey` 이전)에서 일어나고 — httpapi 단건 decrypt(`openbao.go:768-772`)는 `nil` 검사만 하고 `decryptTransit` 를 부른다 — (2) batch 판정이 빠져 `400` 으로 단정했다. 두 지적 모두 코드 확인 결과 맞았다.
- 고친 방법: 해당 한 문장만 사실로 교체했다. "저장소가 DB에 닿기 전에 거절" + "이 판정은 httpapi 가 아니라 저장소 안에서 일어나므로 단건은 `400`, batch 는 항목별 `error` 라서 성공 항목이 하나라도 있으면 응답은 `200`". 코드·테스트는 손대지 않았다(비평가 판단과 동일).
- 근거: 단건 `400` 은 `baoTransitFailure` 의 `ErrInvalid`→400 매핑, batch `200` 은 `baoTransitBatchStatus`(openbao.go:628-636)와 기존 테스트 `TestOpenBaoTransitDecryptBatchKeepsOrderAndReportsItemErrors`(ErrInvalid 항목 + 성공 항목 → 200) 가 이미 고정한다.
- 검증: `node scripts/verify-docs.mjs` 통과, `go vet ./...` 깨끗, `go test ./... -count=1` 전체 통과(store 0.148s / httpapi 0.624s), `gofmt -l` 없음. PostgreSQL 왕복·E2E·`verify.sh` 전체·프런트는 이번에도 미실행(문서 한 줄 변경이라 영향 없음).
- 커밋: `fee6e17` (브랜치 위 새 커밋, amend·force 없음).
