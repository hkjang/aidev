# fix-summary — PR #7 (madi) 커밋 1525a82

- 문제: CI 두 단계가 PR 내용과 무관한 외부 드리프트로 실패. `test` 잡 9단계 govulncheck → GO-2026-6348 (google.golang.org/grpc v1.82.1, `Server.StartJobs` 경로에서 도달) exit 3; `offline-image` 잡 `docker build` → Alpine 3.24 저장소가 ca-certificates 20260611-r0·tzdata 2026c-r0 를 제거해 정확 버전 `deploy/runtime-apk.lock` 이 `unable to select packages` 로 실패. 둘 다 로컬에서 같은 메시지로 재현(GitHub jobs API 로 단계별 결론 확인, 실패 로그 본문은 미제공).
- 고침: `go get google.golang.org/grpc@v1.83.1 && go mod tidy`(x/sys·x/text 가 direct 로 이동한 것은 tidy 결과), apk lock 3핀을 현재 저장소 버전(20260909-r0 / 2026d-r0)으로, `node scripts/licenses.mjs` 로 attribution 매니페스트 재생성(go.mod/go.sum 해시를 포함하므로 필수). 워크플로·테스트·e8e0960 내용 무변경.
- 검증(전부 exit 0): `govulncheck ./...` "No vulnerabilities found"; `go build ./... && go vet ./...`; `node scripts/licenses.mjs --check` ok/423; `docker build -t madi:ci .`; `bash scripts/verify-image.sh madi:ci` PASS(69 APKs / 52 origins 유지); `go test ./tests/deployment-contract ./web/ ./cmd/...`.
- 참고: 같은 수정이 PR #8(8451b5b, 이 브랜치 위에 만들어진 별도 브랜치) 로 이미 CI 통과했으므로 두 PR 은 내용이 겹친다 — 중재자가 하나만 병합하면 된다.
