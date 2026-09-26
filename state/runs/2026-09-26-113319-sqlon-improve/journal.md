# 회차 노트 2026-09-26-113319-sqlon-improve — sqlon
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 11:33] base pinned — main@57f99b7
- [러너 11:33] autonomy release — 
- [러너 11:33] scout skipped — 요청된 작업이 있어 정찰을 건너뛴다 (과제는 이미 정해져 있다)

## 구현 노트
- `.github/workflows/ci.yml` 추가: 러너가 이미 돌리는 `go mod download`→`go build ./...`→`go vet ./...`→`go test ./... -count=1`→`sh scripts/build.sh` 를 push(main)·pull_request 에서 실행한다. `.github` 에 FUNDING.yml 만 있어 PR 이 전부 'CI 검사 없음' 이었다. 커밋 3935e5e. Go 버전은 `setup-go` 의 `go-version-file: go.mod`.
- 확신 없는 곳: **GitHub 러너에서 실제로 돌려 보지 못했다** — 이 세션은 push 권한이 없다. 대신 CI 가 설치할 버전 그대로(`GOTOOLCHAIN=go1.25.0`, 로컬 기본은 1.26.7)와 `git archive HEAD` 로 만든 깨끗한 체크아웃에서 5개 명령 전부 통과를 확인했다. 남은 차이는 러너 환경뿐(캐시 미스, 아키텍처).
- 확신 없는 곳: `push: branches: [main]` 로 좁힌 판단. 명세는 "push 와 pull_request" 라고만 했고, 좁히지 않으면 같은 커밋이 브랜치 push 와 PR 로 두 번 돈다. 같은 저장소 브랜치 PR 은 `pull_request` 로 검사가 붙으니 PR 게이팅에는 영향이 없다고 보지만, 러너가 push 검사를 모든 브랜치에서 요구한다면 이 한 줄을 지우면 된다.
- 일부러 하지 않은 것: `gofmt -l` 게이트 — 지금 87개 파일이 CRLF 로 실패하므로 넣으면 모든 PR 이 막힌다(워크플로 주석에 근거 기록). integration 태그 테스트(DB 컨테이너 필요)·Oracle 빌드(OCI 필요)·`-race`·govulncheck 도 제외. 명세대로 새 테스트를 쓰거나 기존 테스트를 고치지 않았다.
- 다음 역할이 조심할 것: CHANGELOG.md 는 Edit 도구로 손대면 줄바꿈이 전부 뒤집힌다 — 이번에도 4줄 추가가 115/111 diff 가 되어 되돌리고 바이트 단위로 다시 넣었다(`git diff --numstat` 4/0 확인). 워크플로는 보호 경로라 이 PR 은 사람 승인이 필요하다.
- [러너 11:40] verify passed — 검증 3개 통과 (auto)
- [러너 11:40] pr created — https://github.com/hkjang/sqlon/pull/15
- [러너 11:40] guard held — .github/workflows/ci.yml 
