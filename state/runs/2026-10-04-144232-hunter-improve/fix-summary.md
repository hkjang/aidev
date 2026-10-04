# 수리 요약 (11398ed)

- 비평 3건 모두 사실이었다. `internal/app/domain.go:486-491` 이 status=="accepted" 인 모든 findings 쓰기에서 RFC3339+미래를 요구하므로 "findings expires_at 은 쓰기 검증기가 없어 조용히 저장되고 오집계된다"는 서술은 거짓이고, 읽는 네 곳이 전부 status='accepted' 전제라 도달 경로가 없다.
- 거짓 서술을 네 곳(`web/src/resource-form-state.ts:50-67` 주석, `internal/app/testdata/resource-datetime.json` note, `web/tests/resource-form-state.test.mjs:87-90` 주석, Go 테스트 주석)에서 실제 계약으로 바꿨다: 네 필드 모두 400 으로 거절되며 findings expires_at 은 validateResource 가 막는다. 가드의 근거는 "조용한 오집계"가 아니라 "입력한 값이 서버 오류로 사라짐"이다.
- 항상 참이던 `resource_datetime_test.go:95-98` 블록을 삭제하고, validateResource 가 서비스 조회 때문에 DSN 없이 호출 불가라는 사실과 그 검사가 위에서 이미 단언한 validateScope 와 같은 쌍이라는 점을 주석으로 남겼다. 남은 단언은 전부 Hunter 검증기를 실제로 호출한다.
- 프로덕션 로직은 한 줄도 바꾸지 않았다(주석만). 실측: `go test ./internal/app` ok(14 서브테스트 PASS), `npm --prefix web test` 108 pass/0 fail/0 skipped, `tsc --noEmit` 0, `go vet ./internal/app` 0, `verify-pentagi` 312, `go build ./cmd/hunter` OK.
- 미검증: `go test -race ./...`(HUNTER_TEST_DSN 없음), vite/docker build, 실제 브라우저 입력·오류 배너 렌더. 릴리즈 노트에 "조용한 오집계" 문구를 쓰면 안 된다.
