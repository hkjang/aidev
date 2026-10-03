# 수리 요약 (시도 2)

- 비평가의 지적이 맞았다. 임시 probe 테스트로 직접 측정: `Settings{Enabled:true, Provider:matomo, MatomoURL:"https://"+8000×'a'+".corp.example", MatomoSiteID:"1"}` 은 `Validate()` 를 통과하고 `pagePolicy` 가 **24339 바이트** 헤더를 만든다(`MomentoURL`+`MomentoProxy:false` 도 동일 24339, 기본 헤더 273). 즉 "설정 문서가 만들 수 있는 최악은 16KiB 아래" 는 거짓이었다.
- 두 번째 선택지(두 URL 에 상한 추가)는 고르지 않았다 — 이 PR 의 범위는 스니펫 출처이고, 제공자 주소 상한은 별개의 결함이자 새 검증 규칙이므로. 대신 **주석을 측정한 사실로 한정**했다: `internal/tracking/tracking.go` 상수 주석과 `internal/api/tracking_test.go` 의 `carried` 주석 둘 다 "이 두 설정만 15921 바이트로 묶였고 헤더 전체는 아직 묶이지 않았다" 를 명시하며, 반례 설정과 24339 를 그대로 적어 다음 독자가 재현할 수 있게 했다. 같은 거짓이 남아 있던 두 문장(테스트 함수 주석, `both` 블록의 "worst case a settings document can produce")도 함께 고쳤다.
- 코드·상한·테스트 단언은 한 줄도 바꾸지 않았다(주석만). 검증: `gofmt -l`·`go vet` 깨끗, `go test -race ./cmd/... ./internal/...` 전부 ok, 측정 로그 재확인(8180B→592출처→24813B, 결합 15921B).
- 남는 구멍은 닫지 않았다: `MatomoURL`·`MomentoURL` 의 길이 상한 없음(24339B). 이제 주석이 그것을 이름으로 가리키므로 다음 회차의 과제로 적합하다.
