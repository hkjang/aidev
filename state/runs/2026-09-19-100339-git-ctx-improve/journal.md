# 회차 노트 2026-09-19-100339-git-ctx-improve — git-ctx
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 10:03] base pinned — main@be3e9ce
- [러너 10:03] autonomy release — 

## 정찰 노트
- 우선 과제(릴리즈 워크플로 실패 수정)를 그대로 과제로 삼음. GitHub·네트워크 접근이 없어 실제 실패 단계·로그는 미확인이라 brief는 "단계별 로컬 재현 → 근본 원인 수정" 절차서로 씀.
- 직접 관찰한 유일한 실패: main@be3e9ce에서 `go test -tags sqlite_fts5 ./...` 전체 실행 시 `FAIL internal/app 102.5s`, 단독 재실행은 ok → 부하 아래서만 깨지는 flaky 테스트. 실패 테스트 이름은 출력 필터 때문에 놓침(미확인).
- 추측으로 적은 것: flaky 후보가 mcpoauth_test.go·작업 큐 대기 테스트라는 점, govulncheck DB 갱신이 원인일 수 있다는 점. Dockerfile 본문·통합 테스트는 안 봄.
- 구현자 주의: 워크플로·verify-version-sync.sh 완화 금지, t.Skip·타임아웃 상향으로 덮지 말 것, 태그 재지정은 사람 작업.
- [러너 10:08] scout done — 릴리즈 워크플로(`.github/workflows/release.yml` build job)가 두 번 같은 이유로 실패한 원인을 찾아 코드·스크립�

## 구현 노트
- 바꾼 것: `go.mod`/`go.sum` 만 (commit 77dc5af). release.yml build job 을 단계 순서대로 로컬 재현한 결과 `Verify reachable dependencies` 만 실패 — `govulncheck ./...` exit 3, 첫 오류 줄 `Vulnerability #1: GO-2026-6348 … google.golang.org/grpc@v1.82.1, Fixed in v1.83.1, internal/search/service.go:4364:25: search.Query calls sync.Once.Do, which eventually calls transport.NewHTTP2Client`. 취약점 DB 갱신만으로 같은 태그가 반복 실패하는 유형. grpc v1.83.2(GO-2026-6348·6443)와 grpc 가 요구하는 x/net v0.58.0(GO-2026-5942)·x/sync·x/sys·x/text 로 올렸고 전부 간접 의존성 패치/마이너, go/toolchain 줄 불변.
- 검증: 수정 전 govulncheck exit 3 → 후 exit 0(verbose 도 0건). build·vet·gofmt·`go test ./...` 32 ok·`-race ./...` 32 ok·node check·web 테스트·`docker build`+package+verify-offline-image 까지 exit 0. 앞선 단계(태그=HEAD, Version 일치, version-sync 두 스크립트)도 전부 통과.
- 확신 없는 곳: 실제 GitHub 실패 로그는 못 봤다(gh 미인증). 실패 단계가 govulncheck 였다는 것은 "로컬에서 유일하게 실패하는 단계" 라는 근거뿐이다. 통합 테스트 단계(PostgreSQL/pgvector/Vault)는 서버가 없어 미확인.
- 정찰이 본 `FAIL internal/app 102s` 는 4회(일반·-race·부하 동반·taskset 4 CPU)에서 재현되지 않았다. 이 패키지는 통과할 때도 매번 ~105초이므로 시간 자체는 신호가 아니다. 실패 테스트 이름 미확인 상태로 보류.
- 일부러 하지 않은 것: 워크플로·스크립트 수정 없음. 의존성 상한 테스트(go.mod 문자열 검사)는 대역 증거라 추가하지 않았다 — 검사는 워크플로가 매번 돌리는 govulncheck 자체다. grpc v1.84.0 이 아니라 v1.83.2 를 고른 것은 취약점 수정 최소 버전이기 때문.
- 다음 역할 주의: govulncheck 는 네트워크가 필요하며 `$(go env GOPATH)/bin` 이 PATH 에 있어야 한다. 릴리즈 세션은 태그 전에 `govulncheck ./...` 를 한 번 더 돌려 보는 것이 싸다(release.sh 에는 이 단계가 없음). dist/·로컬 이미지는 지웠다.
- [러너 10:22] brief accepted — 채택 — 절차서대로 단계를 재현해 govulncheck 단계가 실패 원인임을 특정했고(과제서 5번 가설), 정찰이 유력하다고 본 flak
- [러너 10:23] verify passed — 검증 3개 통과 (auto)

## 비평 노트
- 확인: diff 는 go.mod/go.sum 만. x/net·sync·sys·text 상승분은 grpc v1.83.2 go.mod 요구치와 정확히 일치(범위 이탈 아님), `go mod tidy -diff` 빈 출력, `go mod verify` 통과, vendor 디렉터리 없음.
- 재현: govulncheck@v1.7.0(DB 2026-09-16) 을 main 트리와 HEAD 트리에서 각각 실행 — main 은 GO-2026-6348(grpc, Fixed in v1.83.1) exit 3, HEAD 는 0건 exit 0. build·vet·`go test -tags sqlite_fts5 -count=1 ./...` 전체 통과, internal/app 실패는 이번에도 미재현.
- 못 본 것: GitHub 실제 실패 로그, 통합 테스트 단계(PG/pgvector/Vault), Docker 빌드(구현자가 돌렸다고 함).
- 판정 approve / risk low. 릴리즈 담당은 태그 직전 `govulncheck ./...` 를 한 번 더 돌릴 것 — release.sh 에 이 단계가 없고 DB 갱신만으로 결과가 바뀌는 유형. 릴리즈 노트에 grpc 1.82.1→1.83.2(GO-2026-6348) 를 명시.
- [러너 10:27] review approved — 리뷰 승인 (risk=low)
- [러너 10:27] pr created — https://github.com/hkjang/git-ctx/pull/31
- [러너 10:34] ci passed — 검사 5개 모두 success
- [러너 10:34] merge done — 77dc5af
- [러너 10:51] release published — v0.77.14
- [러너 11:05] assets verified — v0.77.14 자산 2개 (이전 v0.77.13: 2)
