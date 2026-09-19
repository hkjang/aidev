# 수리 요약 — PR #7 (커밋 750a3d8)

- 문제 1(맞음): `internal/meta/settings.go` 에서 MCP SSO 항목을 끼워넣으며 `SetCacheTTL` SettingDef 가 삭제돼 `PUT /api/settings {"cache_ttl_seconds":...}` 가 400 으로 거부되고 콘솔/GET 에서도 사라짐 → 항목을 `SettingDefs` 끝("성능" 그룹)에 그대로 복원.
- 문제 2(맞음): PUT 핸들러가 키마다 즉시 저장하다 뒤늦게 400 을 내 부분 저장이 남음 → `meta.ValidateSetting(key, value)` 를 추가하고 핸들러가 본문 전체를 먼저 검증한 뒤에만 저장하도록 2단계로 분리(검증 실패 시 400, 저장은 0건; 저장 단계의 오류는 500). `ApplySetting` 도 같은 함수를 쓰도록 정리해 동작 변화 없음.
- 검증: 새 테스트 `TestSettingsPutValidatesBeforeStoringAndKeepsCacheTTL`(수정 전 재현 실패 확인 → 수정 후 통과), `go build ./...`, `go vet`, `go test ./... -count=1` 전부 통과.
