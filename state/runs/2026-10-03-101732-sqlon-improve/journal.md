# 회차 노트 2026-10-03-101732-sqlon-improve — sqlon
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 10:17] base pinned — main@57f99b7
- [러너 10:17] autonomy release — 
- [러너 10:17] scout skipped — 요청된 작업이 있어 정찰을 건너뛴다 (과제는 이미 정해져 있다)

## 구현 노트
- `.github/workflows/ci.yml` 하나를 추가해(+CHANGELOG 5줄) 러너가 이미 돌리는 다섯 명령(go mod download → build → vet → test -count=1 → sh scripts/build.sh)을 push(main)·pull_request 에서 돌게 했다. 커밋 b236046. 프로덕션 소스 변경 0개.
- 검증: `GOTOOLCHAIN=go1.25.0`(CI 가 go.mod 에서 설치할 바로 그 버전)으로 다섯 명령 전부 로컬 통과, 그리고 `git archive HEAD` 로 만든 깨끗한 체크아웃에서도 `go test ./... -count=1` 종료코드 0 · 19 ok · FAIL 0 확인. YAML 은 PyYAML 로 파싱 확인.
- 확신 없는 곳: GitHub 러너에서의 **실제 실행은 미검증**이다(이 세션에 push 권한 없음). actions/checkout@v4·setup-go@v5 의 버전 태그, ubuntu-latest 에서 `sh scripts/build.sh` 의 arm64 크로스빌드(CGO=0 이라 문제없을 것으로 보지만 러너에서 돌려본 적 없음), setup-go 캐시 동작이 미확인 지점이다.
- 일부러 하지 않은 것: `gofmt -l` 게이트(CRLF 때문에 현재 통과하지 않음), integration 태그 테스트(실 DB 컨테이너 필요), Oracle 빌드(OCI 런타임 필요), `-race`·govulncheck(명세의 "지금 통과하는 것만" 범위 밖이며 비차단 관찰이 먼저다). 이유는 ci.yml 머리말 주석에도 남겼다.
- 다음 역할이 조심할 것: CHANGELOG 는 CRLF/LF 혼합이라 Edit 도구로 건드리면 파일 전체 줄바꿈이 뒤집힌다 — 이번에도 바이트 단위 삽입으로 `git diff --numstat` 5/0 을 확인했다. `sh scripts/build.sh` 는 `dist/` 를 만든다(gitignore 됨, 커밋 전 삭제했다).
- [러너 10:22] verify passed — 검증 3개 통과 (auto)
- [러너 10:22] pr created — https://github.com/hkjang/sqlon/pull/16
- [러너 10:22] guard held — .github/workflows/ci.yml 
