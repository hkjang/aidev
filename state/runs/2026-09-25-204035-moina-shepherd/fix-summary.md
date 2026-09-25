# 수리 결과: 수정 없음 (CI 실패는 이 PR 때문이 아님)

- 실패한 `Image and browser smoke` 단계를 로컬에서 그대로 재현했다 — `make image`(moina:v0.1.37) → postgres:17-alpine 새 DB → `--read-only` 앱 컨테이너 → `scripts/qa-api-smoke.sh` → 공식 `mcr.microsoft.com/playwright:v1.62.1-noble`에서 `npm test --prefix e2e`. **빈 DB로 2회 모두 통과**: 시각 회귀 52/52(최대 차이 dark-desktop-admin-smtp 0.345% < 허용 0.5%), 접근성 36 Axe+9 검증, 브라우저 smoke 27 route, API smoke 통과.
- 이 PR이 이미지 동작에 주는 차이는 `social.go`의 `followTopic` INSERT가 **실패했을 때만** 타는 새 500 분기 하나뿐이다. `api/openapi.yaml`은 Dockerfile에 복사되지도 서빙되지도 않고(`grep openapi Dockerfile` 0건), 새 `_test.go`는 바이너리에 들어가지 않는다. e2e·smoke 스크립트에는 `topic`·`follow`·`openapi` 참조가 0건이라 그 분기를 건드릴 경로 자체가 없다.
- PR의 의도한 동작도 재확인했다: throwaway postgres:17에서 `go test -race -count=1 -run TestPostgreSQLFollowTopic ./internal/httpapi/` → 5개 subtest 전부 PASS, SKIP 0줄.
- 원인을 끝까지 특정하지는 못했다. GitHub 로그·진단 artifact는 인증이 없어 받지 못했고(`/logs` 403), 남은 단서는 annotation "Process completed with exit code 1"과 artifact 163KB뿐이다. 163KB는 실패 PNG 1~2장 규모여서 폰트 차이로 52장이 전부 깨진 경우(수 MB)는 아니며, 러너 환경에서 화면 1개 또는 accessibility/browser-smoke 단계가 한 번 흔들린 쪽에 가깝다.
- 절대 규칙에 따라 테스트·워크플로를 손대지 않고 **커밋 없이** 끝낸다. 권한 있는 사람이 이 job을 re-run 하는 것이 다음 단계로 적절하다.
