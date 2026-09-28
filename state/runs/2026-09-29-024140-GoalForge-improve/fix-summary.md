# 수리 요약 (시도 1)

- 문제: `requirePushable` 이 공용 `effectFixture`(effects_test.go:57) 안에 있어, push 를 전혀 하지 않는 `TestGuardAllowsRetryWhenNothingWasApplied`·`TestMergeReconciliationUsesTheBranchContents`·`TestUnreachableRemoteLeavesTheEffectUnresolved` 까지 같이 SKIP 됐다. 비평가 지적이 맞다 — 4건 전부 SKIP 을 `-v` 로 재현했다.
- 수정: 픽스처에서 호출을 빼고 실제로 push 하는 `TestPublishReconcilesInsteadOfRepeating` 첫 줄로 옮겼다(`requirePushable(t, project.RepositoryPath, "origin")`). 주석에 "실제로 push 하는 테스트에서만 부르라, 픽스처에 두면 무관한 테스트가 조용히 꺼진다" 를 명시했다.
- 검증: 수정 후 그 3건은 PASS, push 하는 1건만 SKIP. `GIT_CONFIG_COUNT=0` 으로 차단을 걷은 1회성 실행에서 `TestPublishReconcilesInsteadOfRepeating` 은 스킵이 아니라 **PASS** — 프로브 게이트가 살아 있다.
- 전체: `go build ./...`, `go vet ./...`, `gofmt -l` 무출력, `go test ./... -count=1` exit 0(25 패키지 ok). 전 저장소 SKIP 은 push 하는 4건(`TestCLIFullLifecycle`, `TestRestoreVerifiesRecordsAndSettlesOutsideWork`, `TestPublishReconcilesInsteadOfRepeating`, `TestPushBranchPublishesToLocalRemote`)뿐 — 원래 실패하던 집합과 정확히 일치한다.
- 남은 우려: `TestCLIFullLifecycle` 은 라이프사이클 자체가 push 를 포함해 프로브를 앞에 둘 수밖에 없고, push 막힌 환경에선 e2e 신원 커버리지도 함께 사라진다. 실제 GitHub Actions 러너에서는 여전히 실행해 보지 못했다.
